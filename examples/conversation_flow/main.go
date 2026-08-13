//go:build gtest_example

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	"github.com/morphy76/gtest/examples/conversation_flow/dsl"
	"github.com/morphy76/gtest/pkg/gtest"
)

type sseChannel struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

func main() {
	// Active SSE channels per dialogID
	var mu sync.Mutex
	channels := make(map[string]*sseChannel)

	// Mock server replicating the conversational AI SSE protocol
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if strings.HasPrefix(path, "/api/v1/conversation/") {
			// GET /api/v1/conversation/:external_id -> Open SSE stream & send created event
			parts := strings.Split(path, "/")
			externalID := parts[len(parts)-1]
			dialogID := "dlg-" + externalID

			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")

			flusher, ok := w.(http.Flusher)
			if !ok {
				http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
				return
			}

			mu.Lock()
			ch := &sseChannel{w: w, flusher: flusher}
			channels[dialogID] = ch
			mu.Unlock()

			// Send lifecycle created event
			createdEvent := map[string]any{
				"lifecycle": map[string]string{
					"event":     "created",
					"dialog_id": dialogID,
				},
			}
			data, _ := json.Marshal(createdEvent)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()

			// Keep connection open until context cancelled
			<-r.Context().Done()

			mu.Lock()
			delete(channels, dialogID)
			mu.Unlock()
			return
		}

		if strings.HasPrefix(path, "/api/v1/message/") {
			// POST /api/v1/message/:dialog_id -> Add customer message & emit bot response via SSE
			parts := strings.Split(path, "/")
			dialogID := parts[len(parts)-1]

			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)

			w.WriteHeader(http.StatusOK)

			// Emit bot response event asynchronously over the open SSE connection
			go func() {
				time.Sleep(10 * time.Millisecond) // Simulated LLM inference delay

				mu.Lock()
				ch, exists := channels[dialogID]
				mu.Unlock()

				if exists {
					botEvent := map[string]any{
						"message": map[string]string{
							"event": "messageAdded",
							"role":  "BOT",
							"text":  "Response to: " + body["text"],
						},
					}
					bData, _ := json.Marshal(botEvent)
					fmt.Fprintf(ch.w, "data: %s\n\n", bData)
					ch.flusher.Flush()
				}
			}()
			return
		}

		if strings.HasPrefix(path, "/api/v1/close/") {
			// DELETE /api/v1/close/:external_id/:dialog_id -> Close dialog
			w.WriteHeader(http.StatusOK)
			return
		}

		http.NotFound(w, r)
	}))
	defer server.Close()

	suite := gtest.NewSuite("Conversational AI Load Test Suite")

	suite.RegisterScenario("conversation_test_flow", gtest.Scenario{
		Setup: func(ctx context.Context) (map[string]any, error) {
			return map[string]any{
				"server_url": server.URL,
			}, nil
		},
		PreTest: func(ctx gtest.ScenarioContext) error {
			ctx.Log().Debug().Msg("initiating conversation session")
			return nil
		},
		RunVU: func(ctx gtest.ScenarioContext) error {
			serverURL := ctx.GlobalState("server_url").(string)
			client := dsl.NewConversationClient(serverURL, "pat_token", "tenant_1", nil)

			externalID := fmt.Sprintf("vu-%d-iter-%d", ctx.VUID(), ctx.Iteration())
			dialogModel := ctx.Param("dialog_model")
			if dialogModel == "" {
				dialogModel = "gpt-4o"
			}

			startTotal := time.Now()

			// 1. Open Conversation SSE Stream and await 'created' event
			session, err := client.OpenConversation(ctx, externalID, dialogModel, 5*time.Second)
			if err != nil {
				ctx.Metrics().Rate("conversation_success_rate", gtest.Tags{}).Add(0, 1)
				return fmt.Errorf("OpenConversation failed: %w", err)
			}
			defer session.Close()

			// 2. Perform multi-turn conversation exchanges
			turns := ctx.ParamInt("turns", 2)
			userPrompts := []string{
				"Hello, what services do you offer?",
				"Can you help me with pricing details?",
				"Thank you, goodbye!",
			}

			for i := 0; i < turns; i++ {
				prompt := userPrompts[i%len(userPrompts)]

				// Post customer message
				if err := client.AddMessage(ctx, session, prompt); err != nil {
					ctx.Metrics().Rate("conversation_success_rate", gtest.Tags{}).Add(0, 1)
					return fmt.Errorf("AddMessage turn %d failed: %w", i+1, err)
				}

				// Wait for bot response with SSE timeout (3 seconds)
				_, err := session.AwaitBotResponse(ctx, 3*time.Second)
				if err != nil {
					ctx.Metrics().Rate("conversation_success_rate", gtest.Tags{}).Add(0, 1)
					return fmt.Errorf("AwaitBotResponse turn %d failed: %w", i+1, err)
				}
			}

			// 3. Close Conversation
			if err := client.CloseConversation(ctx, externalID, session.DialogID); err != nil {
				ctx.Metrics().Rate("conversation_success_rate", gtest.Tags{}).Add(0, 1)
				return fmt.Errorf("CloseConversation failed: %w", err)
			}

			totalDuration := time.Since(startTotal)
			ctx.Metrics().Duration("conversation_duration", gtest.Tags{}).Observe(totalDuration)
			ctx.Metrics().Rate("conversation_success_rate", gtest.Tags{}).Add(1, 1)

			return nil
		},
		AfterTest: func(ctx gtest.ScenarioContext) error {
			ctx.Log().Debug().Msg("completed conversation session")
			return nil
		},
		Teardown: func(ctx context.Context, state map[string]any) error {
			return nil
		},
	})

	if err := suite.Execute(); err != nil {
		fmt.Printf("Execution failed: %v\n", err)
	}
}
