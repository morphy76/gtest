package dsl

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/morphy76/gtest/pkg/gtest"
)

// ConversationClient manages protocol interactions with the conversational AI service.
type ConversationClient struct {
	BaseURL    string
	Token      string
	Tenant     string
	HTTPClient *http.Client
}

// NewConversationClient creates a DSL client for conversational AI load testing.
func NewConversationClient(baseURL, token, tenant string, httpClient *http.Client) *ConversationClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &ConversationClient{
		BaseURL:    baseURL,
		Token:      token,
		Tenant:     tenant,
		HTTPClient: httpClient,
	}
}

// SSEEvent represents a parsed Server-Sent Event payload.
type SSEEvent struct {
	Event   string `json:"event,omitempty"`
	Message *struct {
		Event string `json:"event"`
		Role  string `json:"role"`
		Text  string `json:"text"`
	} `json:"message,omitempty"`
	Lifecycle *struct {
		Event    string `json:"event"`
		DialogID string `json:"dialog_id"`
	} `json:"lifecycle,omitempty"`
}

// ConversationSession represents an active user conversation state over SSE.
type ConversationSession struct {
	ExternalID string
	DialogID   string
	SSEStream  io.ReadCloser
	events     chan SSEEvent
	errs       chan error
	ctx        context.Context
	cancel     context.CancelFunc
}

// Events returns a read-only channel delivering parsed SSE events from the background reader.
func (s *ConversationSession) Events() <-chan SSEEvent { return s.events }

// Errors returns a read-only channel delivering SSE stream errors from the background reader.
func (s *ConversationSession) Errors() <-chan error { return s.errs }

// OpenConversation initiates an SSE stream, receives the lifecycle 'created' event, and returns a ConversationSession.
func (c *ConversationClient) OpenConversation(ctx gtest.ScenarioContext, externalID, dialogModel string, timeout time.Duration) (*ConversationSession, error) {
	metrics := NewMetrics(ctx.Metrics())
	start := time.Now()
	sseURL := fmt.Sprintf("%s/api/v1/conversation/%s?with_dialog_model=%s", c.BaseURL, externalID, dialogModel)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sseURL, nil)
	if err != nil {
		metrics.RecordConnectionFailure()
		return nil, fmt.Errorf("failed to build SSE request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("X-Tenant-ID", c.Tenant)
	req.Header.Set("Accept", "text/event-stream")

	// TODO this is SSE, this api blocks until the connection is closed, so we need to use a loop to read the stream.
	// and callbacks to notify the user about events and proceed with the test scenario
	resp, err := c.HTTPClient.Do(req)
	openDuration := time.Since(start)

	if err != nil {
		metrics.RecordConnectionFailure()
		metrics.RecordSSEAvailability(false)
		return nil, fmt.Errorf("failed to open SSE connection: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		metrics.RecordConnectionFailure()
		metrics.RecordSSEAvailability(false)
		return nil, fmt.Errorf("SSE connection failed with HTTP status %d", resp.StatusCode)
	}


	metrics.RecordSSEOpenTime(openDuration)
	metrics.RecordSSEAvailability(true)

	sessCtx, cancel := context.WithCancel(context.Background())
	session := &ConversationSession{
		ExternalID: externalID,
		SSEStream:  resp.Body,
		events:     make(chan SSEEvent, 100),
		errs:       make(chan error, 10),
		ctx:        sessCtx,
		cancel:     cancel,
	}

	// Start background reader loop
	go session.readLoop(resp.Body)

	// Wait for 'created' lifecycle event with SSE timeout
	createdStart := time.Now()
	event, err := session.ReadNextEvent(ctx, timeout)
	createdDuration := time.Since(createdStart)

	if err != nil {
		session.Close()
		return nil, fmt.Errorf("failed to receive created event: %w", err)
	}

	if event == nil || event.Lifecycle == nil || event.Lifecycle.Event != "created" || event.Lifecycle.DialogID == "" {
		session.Close()
		return nil, fmt.Errorf("unexpected initial SSE event: expected lifecycle created, got %v", event)
	}

	session.DialogID = event.Lifecycle.DialogID
	metrics.RecordDialogCreatedTime(createdDuration)
	return session, nil
}

// readLoop runs as a dedicated background goroutine to parse incoming SSE stream frames.
func (s *ConversationSession) readLoop(body io.ReadCloser) {
	scanner := bufio.NewScanner(body)
	var currentEvent string

	for scanner.Scan() {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			currentEvent = ""
			continue
		}

		if strings.HasPrefix(line, ":") {
			// SSE comment / heartbeat
			continue
		}

		if strings.HasPrefix(line, "event:") {
			currentEvent = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}

		if strings.HasPrefix(line, "data:") {
			dataStr := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if dataStr == "" || dataStr == "{}" {
				continue
			}
			var evt SSEEvent
			if err := json.Unmarshal([]byte(dataStr), &evt); err == nil {
				if evt.Event == "" {
					evt.Event = currentEvent
				}
				select {
				case s.events <- evt:
				case <-s.ctx.Done():
					return
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		select {
		case s.errs <- err:
		case <-s.ctx.Done():
		}
	}
}

// AddMessage posts a customer message to an active dialog.
func (c *ConversationClient) AddMessage(ctx gtest.ScenarioContext, session *ConversationSession, messageText string) error {
	if session == nil {
		return fmt.Errorf("cannot add message: session is nil")
	}
	metrics := NewMetrics(ctx.Metrics())
	start := time.Now()
	url := fmt.Sprintf("%s/api/v1/message/%s", c.BaseURL, session.DialogID)

	payload := map[string]string{
		"external_id": session.ExternalID,
		"command":     "addMessage",
		"role":        "CUSTOMER",
		"text":        messageText,
	}
	jsonBytes, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBytes))
	if err != nil {
		metrics.RecordMessageDelivery(0, false)
		return fmt.Errorf("failed to build AddMessage request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("X-Tenant-ID", c.Tenant)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	deliveryTime := time.Since(start)

	if err != nil || (resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated) {
		metrics.RecordMessageDelivery(deliveryTime, false)
		if resp != nil {
			_ = resp.Body.Close()
		}
		return fmt.Errorf("AddMessage failed with status %d: %v", resp.StatusCode, err)
	}

	_ = resp.Body.Close()
	metrics.RecordMessageDelivery(deliveryTime, true)
	return nil
}

// AwaitBotResponse waits for a BOT role message via SSE, enforcing SSE timeout.
func (s *ConversationSession) AwaitBotResponse(ctx gtest.ScenarioContext, timeout time.Duration) (*SSEEvent, error) {
	metrics := NewMetrics(ctx.Metrics())
	start := time.Now()

	for {
		event, err := s.ReadNextEvent(ctx, timeout)
		if err != nil {
			return nil, err
		}

		if event != nil && event.Message != nil {
			switch event.Message.Role {
			case "CUSTOMER":
				metrics.RecordCustomerMessageReceived()
			case "BOT":
				rtt := time.Since(start)
				metrics.RecordBotMessageReceived(rtt)
				return event, nil
			}
		}
	}
}

// ReadNextEvent reads the next event line from the SSE stream channel with configurable timeout handling.
func (s *ConversationSession) ReadNextEvent(ctx gtest.ScenarioContext, timeout time.Duration) (*SSEEvent, error) {
	metrics := NewMetrics(ctx.Metrics())
	select {
	case evt, ok := <-s.events:
		if !ok {
			return nil, io.EOF
		}
		return &evt, nil
	case err := <-s.errs:
		return nil, err
	case <-time.After(timeout):
		metrics.RecordSSETimeout()
		return nil, fmt.Errorf("SSE event timeout after %v", timeout)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Close terminates the SSE stream and stops the background reader goroutine.
func (s *ConversationSession) Close() {
	if s.cancel != nil {
		s.cancel()
	}
	if s.SSEStream != nil {
		_ = s.SSEStream.Close()
	}
}


// CloseConversation sends a DELETE request to close the dialog.
func (c *ConversationClient) CloseConversation(ctx gtest.ScenarioContext, externalID, dialogID string) error {
	url := fmt.Sprintf("%s/api/v1/close/%s/%s", c.BaseURL, externalID, dialogID)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("failed to build close request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("X-Tenant-ID", c.Tenant)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send close request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()


	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("close conversation returned status %d", resp.StatusCode)
	}

	return nil
}
