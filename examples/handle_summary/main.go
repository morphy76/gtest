//go:build gtest_example

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"github.com/morphy76/gtest/pkg/gtest"
)


func main() {
	// Start an in-process mock server serving both the target business API and a webhook endpoint
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/task":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"completed","task_id":"task-42"}`))

		case "/webhook/alerts":
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"bad payload"}`))
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"notification_received"}`))

		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"not found"}`))
		}
	}))
	defer ts.Close()

	suite := gtest.NewSuite("Execution Summary Hook Demo Suite")

	suite.RegisterScenario("summary_hook_demo", gtest.Scenario{
		Setup: func(ctx gtest.SetupContext) (map[string]any, error) {
			client := &http.Client{Timeout: 2 * time.Second}
			return map[string]any{
				"client":      client,
				"server_url":  ts.URL,
				"webhook_url": ts.URL + "/webhook/alerts",
			}, nil
		},
		RunVU: func(ctx gtest.VUContext) error {
			client := ctx.GlobalState("client").(*http.Client)
			serverURL := ctx.GlobalState("server_url").(string)

			start := time.Now()
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, serverURL+"/api/task", nil)
			if err != nil {
				ctx.Metrics().Rate("task_success_rate", gtest.Tags{}).Add(0, 1)
				return fmt.Errorf("failed to create task request: %w", err)
			}

			resp, err := client.Do(req)
			latency := time.Since(start)
			ctx.Metrics().Duration("task_latency", gtest.Tags{}).Observe(latency)

			if err != nil || resp.StatusCode != http.StatusOK {
				ctx.Metrics().Rate("task_success_rate", gtest.Tags{}).Add(0, 1)
				if resp != nil {
					_ = resp.Body.Close()
				}
				return fmt.Errorf("task request failed: %v", err)
			}
			_ = resp.Body.Close()

			ctx.Metrics().Rate("task_success_rate", gtest.Tags{}).Add(1, 1)
			ctx.Metrics().Counter("tasks_completed_total", gtest.Tags{}).Inc()
			return nil
		},
		HandleSummary: func(ctx gtest.SummaryContext, summary gtest.SummaryData) error {
			// HandleSummary executes ONCE post-test run after all reports have been generated.
			// It receives the full SummaryData model with metadata, metrics, and threshold outcomes.

			fmt.Println("\n--- [HandleSummary Hook Invoked] ---")
			fmt.Printf("Suite:       %s\n", summary.SuiteName)
			fmt.Printf("Scenario:    %s\n", summary.Scenario)
			fmt.Printf("Duration:    %v\n", summary.Duration)
			fmt.Printf("SLA Verdict: Passed=%v\n", summary.Passed)

			// 1. Inspect metric aggregates
			totalTasks := summary.Counter("tasks_completed_total")
			successRate := summary.Rate("task_success_rate")
			latencyMetric := summary.Metric("task_latency")

			fmt.Printf("Total Tasks: %d | Success Rate: %.2f%%\n", totalTasks, successRate*100)
			if latencyMetric != nil {
				fmt.Printf("Latency p95: %v | Max: %v\n", latencyMetric.P95, latencyMetric.Max)
			}

			// 2. Evaluate SLA threshold outcomes programmatically
			for _, th := range summary.Thresholds {
				status := "PASS"
				if !th.Passed {
					status = "FAIL"
				}
				fmt.Printf("Threshold [%s]: %s %s %s (actual: %s)\n",
					status, th.Metric, th.Stat, th.Target, th.Actual)
			}

			// 3. Post structured results to a webhook endpoint
			notification := map[string]any{
				"suite":            summary.SuiteName,
				"scenario":         summary.Scenario,
				"passed":           summary.Passed,
				"duration_seconds": summary.Duration.Seconds(),
				"total_tasks":      totalTasks,
				"success_rate":     successRate,
			}

			bodyBytes, _ := json.Marshal(notification)
			webhookReq, err := http.NewRequestWithContext(ctx, http.MethodPost, ts.URL+"/webhook/alerts", bytes.NewReader(bodyBytes))
			if err != nil {
				return fmt.Errorf("failed to prepare webhook request: %w", err)
			}
			webhookReq.Header.Set("Content-Type", "application/json")

			webhookClient := &http.Client{Timeout: 3 * time.Second}
			resp, err := webhookClient.Do(webhookReq)
			if err != nil {
				return fmt.Errorf("failed to dispatch webhook notification: %w", err)
			}
			_ = resp.Body.Close()

			fmt.Println("Successfully delivered notification payload to webhook.")
			fmt.Println("------------------------------------")
			return nil
		},
	})

	res := suite.Execute()
	if res.Error != nil {
		fmt.Fprintf(os.Stderr, "Execution error: %v\n", res.Error)
	}
	os.Exit(res.ExitCode())
}
