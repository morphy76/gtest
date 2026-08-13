package dsl

import (
	"bufio"
	"bytes"
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

// ConversationSession represents an active user conversation state.
type ConversationSession struct {
	ExternalID string
	DialogID   string
	SSEStream  io.ReadCloser
	Scanner    *bufio.Scanner
}

// OpenConversation initiates an SSE stream, receives the lifecycle 'created' event, and returns a ConversationSession.
func (c *ConversationClient) OpenConversation(ctx gtest.ScenarioContext, externalID, dialogModel string, timeout time.Duration) (*ConversationSession, error) {
	start := time.Now()
	sseURL := fmt.Sprintf("%s/api/v1/conversation/%s?with_dialog_model=%s", c.BaseURL, externalID, dialogModel)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sseURL, nil)
	if err != nil {
		ctx.Metrics().Counter("connection_failures", gtest.Tags{}).Inc()
		return nil, fmt.Errorf("failed to build SSE request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("X-Tenant-ID", c.Tenant)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.HTTPClient.Do(req)
	openDuration := time.Since(start)

	if err != nil {
		ctx.Metrics().Counter("connection_failures", gtest.Tags{}).Inc()
		ctx.Metrics().Rate("sse_channel_availability", gtest.Tags{}).Add(0, 1)
		return nil, fmt.Errorf("failed to open SSE connection: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		ctx.Metrics().Counter("connection_failures", gtest.Tags{}).Inc()
		ctx.Metrics().Rate("sse_channel_availability", gtest.Tags{}).Add(0, 1)
		return nil, fmt.Errorf("SSE connection failed with HTTP status %d", resp.StatusCode)
	}

	ctx.Metrics().Duration("sse_open_time", gtest.Tags{}).Observe(openDuration)
	ctx.Metrics().Rate("sse_channel_availability", gtest.Tags{}).Add(1, 1)

	session := &ConversationSession{
		ExternalID: externalID,
		SSEStream:  resp.Body,
		Scanner:    bufio.NewScanner(resp.Body),
	}

	// Wait for 'created' lifecycle event with SSE timeout
	createdStart := time.Now()
	event, err := session.ReadNextEvent(ctx, timeout)
	createdDuration := time.Since(createdStart)

	if err != nil {
		session.Close()
		return nil, fmt.Errorf("failed to receive created event: %w", err)
	}

	if event.Lifecycle == nil || event.Lifecycle.Event != "created" || event.Lifecycle.DialogID == "" {
		session.Close()
		return nil, fmt.Errorf("unexpected initial SSE event: expected lifecycle created, got %v", event)
	}

	session.DialogID = event.Lifecycle.DialogID
	ctx.Metrics().Duration("dialog_created_event_time", gtest.Tags{}).Observe(createdDuration)
	return session, nil
}

// AddMessage posts a customer message to an active dialog.
func (c *ConversationClient) AddMessage(ctx gtest.ScenarioContext, session *ConversationSession, messageText string) error {
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
		ctx.Metrics().Rate("message_success_rate", gtest.Tags{}).Add(0, 1)
		return fmt.Errorf("failed to build AddMessage request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("X-Tenant-ID", c.Tenant)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	deliveryTime := time.Since(start)

	ctx.Metrics().Duration("message_delivery_time", gtest.Tags{}).Observe(deliveryTime)
	ctx.Metrics().Counter("messages_sent", gtest.Tags{}).Inc()

	if err != nil || (resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated) {
		ctx.Metrics().Rate("message_success_rate", gtest.Tags{}).Add(0, 1)
		if resp != nil {
			resp.Body.Close()
		}
		return fmt.Errorf("AddMessage failed with status %d: %v", resp.StatusCode, err)
	}

	resp.Body.Close()
	ctx.Metrics().Rate("message_success_rate", gtest.Tags{}).Add(1, 1)
	return nil
}

// AwaitBotResponse waits for a BOT role message via SSE, enforcing SSE timeout.
func (s *ConversationSession) AwaitBotResponse(ctx gtest.ScenarioContext, timeout time.Duration) (*SSEEvent, error) {
	start := time.Now()

	for {
		event, err := s.ReadNextEvent(ctx, timeout)
		if err != nil {
			return nil, err
		}

		if event.Message != nil {
			if event.Message.Role == "CUSTOMER" {
				ctx.Metrics().Counter("customer_messages_received", gtest.Tags{}).Inc()
			} else if event.Message.Role == "BOT" {
				rtt := time.Since(start)
				ctx.Metrics().Duration("answer_received_time", gtest.Tags{}).Observe(rtt)
				ctx.Metrics().Counter("bot_messages_received", gtest.Tags{}).Inc()
				return event, nil
			}
		}
	}
}

// ReadNextEvent reads the next event line from the SSE stream with configurable timeout handling.
func (s *ConversationSession) ReadNextEvent(ctx gtest.ScenarioContext, timeout time.Duration) (*SSEEvent, error) {
	type result struct {
		event *SSEEvent
		err   error
	}

	ch := make(chan result, 1)

	go func() {
		for s.Scanner.Scan() {
			line := strings.TrimSpace(s.Scanner.Text())
			if strings.HasPrefix(line, "data:") {
				dataStr := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
				if dataStr == "" || dataStr == "{}" {
					continue
				}
				var event SSEEvent
				if err := json.Unmarshal([]byte(dataStr), &event); err != nil {
					ch <- result{err: fmt.Errorf("failed to unmarshal SSE event: %w", err)}
					return
				}
				ch <- result{event: &event}
				return
			}
		}
		if err := s.Scanner.Err(); err != nil {
			ch <- result{err: fmt.Errorf("SSE scanner error: %w", err)}
			return
		}
		ch <- result{err: io.EOF}
	}()

	select {
	case res := <-ch:
		return res.event, res.err
	case <-time.After(timeout):
		ctx.Metrics().Counter("sse_timeout_errors", gtest.Tags{}).Inc()
		return nil, fmt.Errorf("SSE event timeout after %v", timeout)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Close closes the SSE stream.
func (s *ConversationSession) Close() {
	if s.SSEStream != nil {
		s.SSEStream.Close()
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("close conversation returned status %d", resp.StatusCode)
	}

	return nil
}
