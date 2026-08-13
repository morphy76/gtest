package dsl

import (
	"time"

	"github.com/morphy76/gtest/pkg/gtest"
)

// ConversationMetrics encapsulates domain-specific metric recording for conversational AI load tests.
type ConversationMetrics struct {
	raw gtest.MetricsCollector
}

// NewMetrics wraps a gtest.MetricsCollector into a semantically meaningful domain collector.
func NewMetrics(raw gtest.MetricsCollector) *ConversationMetrics {
	return &ConversationMetrics{raw: raw}
}

// RecordConnectionFailure logs an SSE connection failure.
func (m *ConversationMetrics) RecordConnectionFailure() {
	m.raw.Counter("connection_failures", gtest.Tags{}).Inc()
}

// RecordSSEAvailability tracks the SSE channel availability rate (1 for success, 0 for failure).
func (m *ConversationMetrics) RecordSSEAvailability(success bool) {
	if success {
		m.raw.Rate("sse_channel_availability", gtest.Tags{}).Add(1, 1)
	} else {
		m.raw.Rate("sse_channel_availability", gtest.Tags{}).Add(0, 1)
	}
}

// RecordSSEOpenTime records the latency duration to establish an SSE stream.
func (m *ConversationMetrics) RecordSSEOpenTime(duration time.Duration) {
	m.raw.Duration("sse_open_time", gtest.Tags{}).Observe(duration)
}

// RecordDialogCreatedTime records the duration until the initial 'created' lifecycle event is received.
func (m *ConversationMetrics) RecordDialogCreatedTime(duration time.Duration) {
	m.raw.Duration("dialog_created_event_time", gtest.Tags{}).Observe(duration)
}

// RecordSSETimeout increments the SSE event timeout error counter.
func (m *ConversationMetrics) RecordSSETimeout() {
	m.raw.Counter("sse_timeout_errors", gtest.Tags{}).Inc()
}

// RecordMessageDelivery records POST message delivery latency and success/failure status.
func (m *ConversationMetrics) RecordMessageDelivery(duration time.Duration, success bool) {
	m.raw.Duration("message_delivery_time", gtest.Tags{}).Observe(duration)
	m.raw.Counter("messages_sent", gtest.Tags{}).Inc()
	if success {
		m.raw.Rate("message_success_rate", gtest.Tags{}).Add(1, 1)
	} else {
		m.raw.Rate("message_success_rate", gtest.Tags{}).Add(0, 1)
	}
}

// RecordCustomerMessageReceived increments the count of customer echo messages received over SSE.
func (m *ConversationMetrics) RecordCustomerMessageReceived() {
	m.raw.Counter("customer_messages_received", gtest.Tags{}).Inc()
}

// RecordBotMessageReceived records round-trip latency and count for bot response messages.
func (m *ConversationMetrics) RecordBotMessageReceived(rtt time.Duration) {
	m.raw.Duration("answer_received_time", gtest.Tags{}).Observe(rtt)
	m.raw.Counter("bot_messages_received", gtest.Tags{}).Inc()
}

// RecordConversationResult records overall conversation flow duration and success/failure rate.
func (m *ConversationMetrics) RecordConversationResult(duration time.Duration, success bool) {
	if success {
		m.raw.Duration("conversation_duration", gtest.Tags{}).Observe(duration)
		m.raw.Rate("conversation_success_rate", gtest.Tags{}).Add(1, 1)
	} else {
		m.raw.Rate("conversation_success_rate", gtest.Tags{}).Add(0, 1)
	}
}

// RecordSSEConnectionRequested increments the count of SSE connection attempts.
func (m *ConversationMetrics) RecordSSEConnectionRequested() {
	m.raw.Counter("number_of_requested_sse_connections", gtest.Tags{}).Inc()
}

// RecordSSEConnectionSuccessful increments the count of successfully established SSE connections.
func (m *ConversationMetrics) RecordSSEConnectionSuccessful() {
	m.raw.Counter("number_of_successful_sse_connections", gtest.Tags{}).Inc()
}

// RecordSSEConnectionFailed increments the count of failed SSE connection attempts.
func (m *ConversationMetrics) RecordSSEConnectionFailed() {
	m.raw.Counter("number_of_failed_sse_connections", gtest.Tags{}).Inc()
}

// RecordSSEConnectionClosed increments the count of cleanly closed SSE connections.
func (m *ConversationMetrics) RecordSSEConnectionClosed() {
	m.raw.Counter("number_of_closed_sse_connections", gtest.Tags{}).Inc()
}

// RecordDialogCreated increments the count of dialogs successfully created via SSE lifecycle events.
func (m *ConversationMetrics) RecordDialogCreated() {
	m.raw.Counter("number_of_created_dialogs", gtest.Tags{}).Inc()
}

// RecordDialogClosed increments the count of dialogs explicitly closed by the client.
func (m *ConversationMetrics) RecordDialogClosed() {
	m.raw.Counter("number_of_closed_dialogs", gtest.Tags{}).Inc()
}

// RecordOpenRoundTrips records discrepancies between expected and received message counts.
// A non-zero value signals dropped or duplicated messages.
func (m *ConversationMetrics) RecordOpenRoundTrips(customerDiscrepancy, botDiscrepancy int) {
	if customerDiscrepancy != 0 {
		m.raw.Counter("customer_open_round_trips", gtest.Tags{}).Add(int64(abs(customerDiscrepancy)))
	}
	if botDiscrepancy != 0 {
		m.raw.Counter("bot_response_round_trips", gtest.Tags{}).Add(int64(abs(botDiscrepancy)))
	}
}

// RecordSSEErrorCategory records a categorized SSE error (e.g., "sse_protocol", "sse_authentication", "sse_parsing").
func (m *ConversationMetrics) RecordSSEErrorCategory(category string) {
	m.raw.Counter(category, gtest.Tags{}).Inc()
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
