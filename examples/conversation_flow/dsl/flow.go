package dsl

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/morphy76/gtest/pkg/gtest"
)

// FlowConfig holds all configurable parameters for a ConversationFlow execution.
type FlowConfig struct {
	DialogModel      string
	Turns            int
	InteractionDelay time.Duration
	SSEEventTimeout  time.Duration
	Messages         []Message
}

// ConversationFlow orchestrates a reactive, event-driven conversation lifecycle.
// It mirrors the JS k6 SSE event-loop pattern: SSE events drive the flow forward
// via dispatch to typed handlers, with per-event timeout guards to prevent VU hangs.
type ConversationFlow struct {
	client  *ConversationClient
	config  FlowConfig
	metrics *ConversationMetrics

	// Runtime state
	session                  *ConversationSession
	currentTurn              int
	customerMessagesReceived int
	botMessagesReceived      int
	conversationSuccess      bool
	flowErr                  error
	turnStart                time.Time
}

// NewConversationFlow creates a ConversationFlow with the given client and configuration.
func NewConversationFlow(client *ConversationClient, config FlowConfig) *ConversationFlow {
	if config.Turns <= 0 {
		config.Turns = 1
	}
	if config.SSEEventTimeout <= 0 {
		config.SSEEventTimeout = 5 * time.Second
	}
	if len(config.Messages) == 0 {
		config.Messages = []Message{
			{ID: "default", Text: "Hello"},
		}
	}
	return &ConversationFlow{
		client:              client,
		config:              config,
		conversationSuccess: true,
	}
}

// Run executes the full conversation lifecycle: open SSE → event dispatch loop → close → verify accounting.
func (f *ConversationFlow) Run(ctx gtest.ScenarioContext) error {
	f.metrics = NewMetrics(ctx.Metrics())
	startTotal := time.Now()

	externalID := fmt.Sprintf("vu-%d-iter-%d", ctx.VUID(), ctx.Iteration())

	f.metrics.RecordSSEConnectionRequested()

	session, err := f.client.OpenConversation(ctx, externalID, f.config.DialogModel, f.config.SSEEventTimeout)
	if err != nil {
		f.metrics.RecordSSEConnectionFailed()
		f.metrics.RecordConversationResult(0, false)
		return fmt.Errorf("OpenConversation failed: %w", err)
	}
	f.session = session
	f.metrics.RecordSSEConnectionSuccessful()
	defer func() {
		session.Close()
		f.metrics.RecordSSEConnectionClosed()
	}()

	f.metrics.RecordDialogCreated()

	// Event dispatch loop
	if err := f.eventLoop(ctx); err != nil {
		f.metrics.RecordConversationResult(0, false)
		return err
	}

	// Verify round-trip accounting
	f.verifyAccounting()

	totalDuration := time.Since(startTotal)
	f.metrics.RecordConversationResult(totalDuration, f.conversationSuccess)

	if !f.conversationSuccess {
		return f.flowErr
	}
	return nil
}

// eventLoop consumes SSE events from the session and dispatches to handlers.
// Each iteration resets a per-event timeout to prevent VU hangs on lost SSE answers.
func (f *ConversationFlow) eventLoop(ctx gtest.ScenarioContext) error {
	// Post first customer message after created event was already handled in OpenConversation
	if err := f.sendNextMessage(ctx); err != nil {
		return err
	}

	for {
		select {
		case evt, ok := <-f.session.Events():
			if !ok {
				// Channel closed — stream ended
				if f.botMessagesReceived >= f.config.Turns {
					return nil
				}
				return fmt.Errorf("SSE stream closed before all turns completed (got %d/%d bot responses)", f.botMessagesReceived, f.config.Turns)
			}

			action, err := f.dispatch(ctx, &evt)
			if err != nil {
				return err
			}
			if action == flowActionDone {
				return nil
			}

		case err := <-f.session.Errors():
			f.conversationSuccess = false
			f.metrics.RecordSSEErrorCategory("sse_connection_errors")
			return fmt.Errorf("SSE stream error: %w", err)

		case <-time.After(f.config.SSEEventTimeout):
			f.conversationSuccess = false
			f.metrics.RecordSSETimeout()
			return fmt.Errorf("SSE event timeout after %v waiting for expected response (turn %d/%d)", f.config.SSEEventTimeout, f.currentTurn, f.config.Turns)

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

type flowAction int

const (
	flowActionContinue flowAction = iota
	flowActionDone
)

// dispatch routes an SSE event to the appropriate handler.
func (f *ConversationFlow) dispatch(ctx gtest.ScenarioContext, evt *SSEEvent) (flowAction, error) {
	if evt.Lifecycle != nil {
		return f.handleLifecycle(ctx, evt)
	}
	if evt.Message != nil {
		return f.handleMessage(ctx, evt)
	}
	// Unknown event structure — skip
	return flowActionContinue, nil
}

// handleLifecycle processes SSE lifecycle events (aborted, closed).
// Note: 'created' is already handled by OpenConversation.
func (f *ConversationFlow) handleLifecycle(ctx gtest.ScenarioContext, evt *SSEEvent) (flowAction, error) {
	switch evt.Lifecycle.Event {
	case "aborted":
		f.conversationSuccess = false
		f.metrics.RecordSSEErrorCategory("sse_protocol_errors")
		return flowActionDone, fmt.Errorf("conversation aborted via SSE lifecycle event")

	case "closed":
		f.metrics.RecordDialogClosed()
		return flowActionDone, nil

	default:
		// created or other — ignore in event loop (created was handled at connection time)
		return flowActionContinue, nil
	}
}

// handleMessage processes SSE message events (BOT or CUSTOMER role).
func (f *ConversationFlow) handleMessage(ctx gtest.ScenarioContext, evt *SSEEvent) (flowAction, error) {
	switch evt.Message.Role {
	case "CUSTOMER":
		f.customerMessagesReceived++
		f.metrics.RecordCustomerMessageReceived()
		return flowActionContinue, nil

	case "BOT":
		f.botMessagesReceived++
		rtt := time.Since(f.turnStart)
		f.metrics.RecordBotMessageReceived(rtt)

		if f.currentTurn >= f.config.Turns {
			// All turns completed (exhausted) — close conversation immediately without thinking time
			if err := f.closeConversation(ctx); err != nil {
				return flowActionDone, err
			}
			return flowActionDone, nil
		}

		// Thinking time: executed after receiving the bot message, before the next customer message
		// only if turns are not exhausted.
		if f.config.InteractionDelay > 0 {
			if err := ctx.Sleep(f.config.InteractionDelay); err != nil {
				return flowActionDone, err
			}
		} else {
			if err := ctx.Sleep(); err != nil {
				return flowActionDone, err
			}
		}

		// Send next customer message
		if err := f.sendNextMessage(ctx); err != nil {
			return flowActionDone, err
		}
		return flowActionContinue, nil


	default:
		return flowActionContinue, nil
	}
}

// sendNextMessage selects a message from the configured pool and posts it.
func (f *ConversationFlow) sendNextMessage(ctx gtest.ScenarioContext) error {
	f.currentTurn++
	msg := f.config.Messages[rand.Intn(len(f.config.Messages))]
	f.turnStart = time.Now()

	if err := f.client.AddMessage(ctx, f.session, msg.Text); err != nil {
		f.conversationSuccess = false
		return fmt.Errorf("AddMessage turn %d failed: %w", f.currentTurn, err)
	}
	return nil
}

// closeConversation sends the DELETE request to close the dialog.
func (f *ConversationFlow) closeConversation(ctx gtest.ScenarioContext) error {
	if err := f.client.CloseConversation(ctx, f.session.ExternalID, f.session.DialogID); err != nil {
		f.conversationSuccess = false
		return fmt.Errorf("CloseConversation failed: %w", err)
	}
	f.metrics.RecordDialogClosed()
	return nil
}

// verifyAccounting checks that the number of received messages matches expectations.
func (f *ConversationFlow) verifyAccounting() {
	customerDisc := f.customerMessagesReceived - f.currentTurn
	botDisc := f.botMessagesReceived - f.config.Turns

	if customerDisc != 0 || botDisc != 0 {
		f.metrics.RecordOpenRoundTrips(customerDisc, botDisc)
		f.conversationSuccess = false
		f.flowErr = fmt.Errorf("round-trip mismatch: customer messages received=%d (expected %d), bot messages received=%d (expected %d)",
			f.customerMessagesReceived, f.currentTurn, f.botMessagesReceived, f.config.Turns)
	}
}
