package dsl

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/morphy76/gtest/pkg/gtest"
)

type mockScenarioContext struct {
	context.Context
	params map[string]string
}

func newMockScenarioContext() *mockScenarioContext {
	return &mockScenarioContext{
		Context: context.Background(),
		params:  make(map[string]string),
	}
}

func (m *mockScenarioContext) VUID() int64                             { return 1 }
func (m *mockScenarioContext) Iteration() int64                        { return 0 }
func (m *mockScenarioContext) ScenarioName() string                    { return "test_scenario" }
func (m *mockScenarioContext) Param(key string) string                 { return m.params[key] }
func (m *mockScenarioContext) ParamInt(key string, def int) int        { return def }
func (m *mockScenarioContext) ParamDuration(key string, def time.Duration) time.Duration { return def }
func (m *mockScenarioContext) GlobalState(key string) any              { return nil }
func (m *mockScenarioContext) Log() gtest.Logger                        { return noopLogger{} }
func (m *mockScenarioContext) Metrics() gtest.MetricsCollector         { return noopMetrics{} }

type noopLogger struct{}

func (l noopLogger) Debug() gtest.LogEvent { return noopLogEvent{} }
func (l noopLogger) Info() gtest.LogEvent  { return noopLogEvent{} }
func (l noopLogger) Warn() gtest.LogEvent  { return noopLogEvent{} }
func (l noopLogger) Error() gtest.LogEvent { return noopLogEvent{} }

type noopLogEvent struct{}

func (e noopLogEvent) Str(k, v string) gtest.LogEvent       { return e }
func (e noopLogEvent) Int(k string, v int) gtest.LogEvent   { return e }
func (e noopLogEvent) Int64(k string, v int64) gtest.LogEvent { return e }
func (e noopLogEvent) Float64(k string, v float64) gtest.LogEvent { return e }
func (e noopLogEvent) Bool(k string, v bool) gtest.LogEvent { return e }
func (e noopLogEvent) Dur(k string, v time.Duration) gtest.LogEvent { return e }
func (e noopLogEvent) Err(err error) gtest.LogEvent         { return e }
func (e noopLogEvent) Msg(msg string)                       {}

type noopMetrics struct{}

func (m noopMetrics) Counter(name string, tags gtest.Tags) gtest.Counter { return noopCounter{} }
func (m noopMetrics) Rate(name string, tags gtest.Tags) gtest.Rate       { return noopRate{} }
func (m noopMetrics) Duration(name string, tags gtest.Tags) gtest.Duration { return noopDuration{} }
func (m noopMetrics) Gauge(name string, tags gtest.Tags) gtest.Gauge    { return noopGauge{} }

type noopCounter struct{}

func (c noopCounter) Inc()           {}
func (c noopCounter) Add(delta int64) {}

type noopRate struct{}

func (r noopRate) Add(success, total int64) {}

type noopDuration struct{}

func (d noopDuration) Observe(val time.Duration) {}

type noopGauge struct{}

func (g noopGauge) Set(val float64)   {}
func (g noopGauge) Add(delta float64) {}

func TestOpenConversationAndAwaitBotResponse(t *testing.T) {
	// Setup test SSE server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/conversation/vu-1" {
			assert.Equal(t, "Bearer test_token", r.Header.Get("Authorization"))
			assert.Equal(t, "test_tenant", r.Header.Get("X-Tenant-ID"))

			w.Header().Set("Content-Type", "text/event-stream")
			flusher, ok := w.(http.Flusher)
			require.True(t, ok)

			// Send lifecycle created event
			createdEvent := map[string]any{
				"lifecycle": map[string]string{
					"event":     "created",
					"dialog_id": "dlg-vu-1",
				},
			}
			b, _ := json.Marshal(createdEvent)
			fmt.Fprintf(w, "data: %s\n\n", b)
			flusher.Flush()

			// Wait a bit and send bot message event
			time.Sleep(50 * time.Millisecond)
			botEvent := map[string]any{
				"message": map[string]string{
					"event": "messageAdded",
					"role":  "BOT",
					"text":  "Hello back!",
				},
			}
			b2, _ := json.Marshal(botEvent)
			fmt.Fprintf(w, "data: %s\n\n", b2)
			flusher.Flush()

			// Keep open until request canceled
			<-r.Context().Done()
			return
		}

		if r.URL.Path == "/api/v1/message/dlg-vu-1" {
			assert.Equal(t, http.MethodPost, r.Method)
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.URL.Path == "/api/v1/close/vu-1/dlg-vu-1" {
			assert.Equal(t, http.MethodDelete, r.Method)
			w.WriteHeader(http.StatusOK)
			return
		}
	}))
	defer server.Close()

	ctx := newMockScenarioContext()
	client := NewConversationClient(server.URL, "test_token", "test_tenant", nil)

	// 1. Open conversation
	session, err := client.OpenConversation(ctx, "vu-1", "gpt-4o", 2*time.Second)
	require.NoError(t, err)
	require.NotNil(t, session)
	assert.Equal(t, "dlg-vu-1", session.DialogID)
	defer session.Close()

	// 2. Add message
	err = client.AddMessage(ctx, session, "Hello")
	require.NoError(t, err)

	// 3. Await bot response
	event, err := session.AwaitBotResponse(ctx, 2*time.Second)
	require.NoError(t, err)
	require.NotNil(t, event)
	require.NotNil(t, event.Message)
	assert.Equal(t, "BOT", event.Message.Role)
	assert.Equal(t, "Hello back!", event.Message.Text)

	// 4. Close conversation
	err = client.CloseConversation(ctx, "vu-1", session.DialogID)
	require.NoError(t, err)
}

func TestAwaitBotResponse_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)

		// Only send created event, no bot response
		createdEvent := map[string]any{
			"lifecycle": map[string]string{
				"event":     "created",
				"dialog_id": "dlg-timeout",
			},
		}
		b, _ := json.Marshal(createdEvent)
		fmt.Fprintf(w, "data: %s\n\n", b)
		flusher.Flush()

		<-r.Context().Done()
	}))
	defer server.Close()

	ctx := newMockScenarioContext()
	client := NewConversationClient(server.URL, "test_token", "test_tenant", nil)

	session, err := client.OpenConversation(ctx, "vu-timeout", "gpt-4o", 1*time.Second)
	require.NoError(t, err)
	defer session.Close()

	// Wait for bot response when none arrives -> expect timeout error
	_, err = session.AwaitBotResponse(ctx, 100*time.Millisecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}
