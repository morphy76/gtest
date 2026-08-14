package dsl

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Test helpers for SSE mock servers ---

type sseTestChannel struct {
	mu      sync.Mutex
	w       http.ResponseWriter
	flusher http.Flusher
}

func sendSSEEvent(ch *sseTestChannel, payload any) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	data, _ := json.Marshal(payload)
	fmt.Fprintf(ch.w, "data: %s\n\n", data)
	ch.flusher.Flush()
}

func splitPath(path string) []string {
	var parts []string
	for _, p := range splitBySlash(path) {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

func splitBySlash(s string) []string {
	result := []string{""}
	for _, c := range s {
		if c == '/' {
			result = append(result, "")
		} else {
			result[len(result)-1] += string(c)
		}
	}
	return result
}

// fullProtocolMockServer creates a mock that:
// - On GET /api/v1/conversation/:id → Opens SSE, sends 'created', keeps alive
// - On POST /api/v1/message/:dialog_id → Echoes customer message + sends bot reply over SSE
// - On DELETE /api/v1/close/:external_id/:dialog_id → Returns 200
func fullProtocolMockServer(t *testing.T) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	channels := make(map[string]*sseTestChannel)

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		parts := splitPath(path)

		if r.Method == http.MethodGet && len(parts) >= 4 && parts[2] == "conversation" {
			externalID := parts[3]
			dialogID := "dlg-" + externalID

			flusher, ok := w.(http.Flusher)
			require.True(t, ok)

			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")

			ch := &sseTestChannel{w: w, flusher: flusher}

			mu.Lock()
			channels[dialogID] = ch
			mu.Unlock()

			// Send lifecycle 'created' event
			sendSSEEvent(ch, map[string]any{
				"lifecycle": map[string]string{
					"event":     "created",
					"dialog_id": dialogID,
				},
			})

			<-r.Context().Done()

			mu.Lock()
			delete(channels, dialogID)
			mu.Unlock()
			return
		}

		if r.Method == http.MethodPost && len(parts) >= 4 && parts[2] == "message" {
			dialogID := parts[3]

			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)

			mu.Lock()
			ch, exists := channels[dialogID]
			mu.Unlock()

			if exists {
				// Echo customer message
				sendSSEEvent(ch, map[string]any{
					"message": map[string]string{
						"event": "messageAdded",
						"role":  "CUSTOMER",
						"text":  body["text"],
					},
				})
				// Bot reply
				sendSSEEvent(ch, map[string]any{
					"message": map[string]string{
						"event": "messageAdded",
						"role":  "BOT",
						"text":  "Response to: " + body["text"],
					},
				})
			}

			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method == http.MethodDelete && len(parts) >= 4 && parts[2] == "close" {
			w.WriteHeader(http.StatusOK)
			return
		}

		http.NotFound(w, r)
	}))
}

// --- Flow Tests ---

func TestConversationFlow_HappyPath_TwoTurns(t *testing.T) {
	server := fullProtocolMockServer(t)
	defer server.Close()

	ctx := newMockScenarioContext()
	client := NewConversationClient(server.URL, "token", "tenant", nil)
	messages := []Message{
		{ID: "1", Text: "Hello", Category: "general", ExpectedTokens: 15},
		{ID: "2", Text: "Help me", Category: "support", ExpectedTokens: 20},
	}

	flow := NewConversationFlow(client, FlowConfig{
		DialogModel:     "gpt-4o",
		Turns:           2,
		SSEEventTimeout: 3 * time.Second,
		Messages:        messages,
	})

	err := flow.Run(ctx)

	require.NoError(t, err)
}

func TestConversationFlow_AbortedLifecycle(t *testing.T) {
	// Server that sends 'created' then 'aborted' — does NOT auto-reply to POST
	abortServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		parts := splitPath(path)

		if r.Method == http.MethodGet && len(parts) >= 4 && parts[2] == "conversation" {
			externalID := parts[3]
			dialogID := "dlg-" + externalID

			flusher := w.(http.Flusher)
			w.Header().Set("Content-Type", "text/event-stream")

			ch := &sseTestChannel{w: w, flusher: flusher}

			sendSSEEvent(ch, map[string]any{
				"lifecycle": map[string]string{
					"event":     "created",
					"dialog_id": dialogID,
				},
			})

			// Wait briefly, then send aborted
			time.Sleep(50 * time.Millisecond)
			sendSSEEvent(ch, map[string]any{
				"lifecycle": map[string]string{
					"event":     "aborted",
					"dialog_id": dialogID,
				},
			})

			<-r.Context().Done()
			return
		}

		if r.Method == http.MethodPost {
			// Accept message but no SSE reply — the abort will arrive instead
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusOK)
			return
		}

		http.NotFound(w, r)
	}))
	defer abortServer.Close()

	ctx := newMockScenarioContext()
	client := NewConversationClient(abortServer.URL, "token", "tenant", nil)

	flow := NewConversationFlow(client, FlowConfig{
		DialogModel:     "gpt-4o",
		Turns:           2,
		SSEEventTimeout: 3 * time.Second,
		Messages: []Message{
			{ID: "1", Text: "Hello"},
		},
	})

	err := flow.Run(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "aborted")
}

func TestConversationFlow_PerEventTimeout(t *testing.T) {
	// Server that sends 'created' but never sends bot response
	silentServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		parts := splitPath(path)

		if r.Method == http.MethodGet && len(parts) >= 4 && parts[2] == "conversation" {
			externalID := parts[3]
			dialogID := "dlg-" + externalID

			flusher := w.(http.Flusher)
			w.Header().Set("Content-Type", "text/event-stream")

			sendSSEEvent(&sseTestChannel{w: w, flusher: flusher}, map[string]any{
				"lifecycle": map[string]string{
					"event":     "created",
					"dialog_id": dialogID,
				},
			})

			<-r.Context().Done()
			return
		}

		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusOK)
			return
		}

		http.NotFound(w, r)
	}))
	defer silentServer.Close()

	ctx := newMockScenarioContext()
	client := NewConversationClient(silentServer.URL, "token", "tenant", nil)

	flow := NewConversationFlow(client, FlowConfig{
		DialogModel:     "gpt-4o",
		Turns:           1,
		SSEEventTimeout: 150 * time.Millisecond,
		Messages: []Message{
			{ID: "1", Text: "Hello"},
		},
	})

	start := time.Now()
	err := flow.Run(ctx)
	elapsed := time.Since(start)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
	assert.Less(t, elapsed, 2*time.Second, "flow should not hang on missing SSE events")
}

func TestConversationFlow_MessageSendFailure(t *testing.T) {
	// Server that accepts SSE but rejects POST messages
	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		parts := splitPath(path)

		if r.Method == http.MethodGet && len(parts) >= 4 && parts[2] == "conversation" {
			externalID := parts[3]
			dialogID := "dlg-" + externalID

			flusher := w.(http.Flusher)
			w.Header().Set("Content-Type", "text/event-stream")

			sendSSEEvent(&sseTestChannel{w: w, flusher: flusher}, map[string]any{
				"lifecycle": map[string]string{
					"event":     "created",
					"dialog_id": dialogID,
				},
			})

			<-r.Context().Done()
			return
		}

		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusOK)
			return
		}

		http.NotFound(w, r)
	}))
	defer failServer.Close()

	ctx := newMockScenarioContext()
	client := NewConversationClient(failServer.URL, "token", "tenant", nil)

	flow := NewConversationFlow(client, FlowConfig{
		DialogModel:     "gpt-4o",
		Turns:           1,
		SSEEventTimeout: 1 * time.Second,
		Messages: []Message{
			{ID: "1", Text: "Hello"},
		},
	})

	err := flow.Run(ctx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "AddMessage")
}

func TestConversationFlow_ThinkingTimeOnlyBeforeNextMessage(t *testing.T) {
	server := fullProtocolMockServer(t)
	defer server.Close()

	t.Run("three turns executes thinking time twice", func(t *testing.T) {
		ctx := newMockScenarioContext()
		client := NewConversationClient(server.URL, "token", "tenant", nil)
		messages := []Message{
			{ID: "1", Text: "Hello"},
			{ID: "2", Text: "Question 2"},
			{ID: "3", Text: "Goodbye"},
		}

		flow := NewConversationFlow(client, FlowConfig{
			DialogModel:      "gpt-4o",
			Turns:            3,
			InteractionDelay: 50 * time.Millisecond,
			SSEEventTimeout:  3 * time.Second,
			Messages:         messages,
		})

		err := flow.Run(ctx)
		require.NoError(t, err)

		// For 3 turns, think time should be executed after turn 1 and turn 2, but NOT after turn 3 (exhausted)
		ctx.mu.Lock()
		defer ctx.mu.Unlock()
		assert.Len(t, ctx.sleepCalls, 2, "must sleep exactly 2 times for a 3-turn flow")
		for _, d := range ctx.sleepCalls {
			assert.Equal(t, 50*time.Millisecond, d)
		}
	})

	t.Run("one turn does not execute thinking time after completion", func(t *testing.T) {
		ctx := newMockScenarioContext()
		client := NewConversationClient(server.URL, "token", "tenant", nil)
		messages := []Message{
			{ID: "1", Text: "Single turn question"},
		}

		flow := NewConversationFlow(client, FlowConfig{
			DialogModel:      "gpt-4o",
			Turns:            1,
			InteractionDelay: 50 * time.Millisecond,
			SSEEventTimeout:  3 * time.Second,
			Messages:         messages,
		})

		err := flow.Run(ctx)
		require.NoError(t, err)

		// For 1 turn, turns are exhausted after the first bot reply -> 0 sleep calls
		ctx.mu.Lock()
		defer ctx.mu.Unlock()
		assert.Empty(t, ctx.sleepCalls, "must not sleep when turns are exhausted")
	})
}

