//go:build gtest_example

package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/morphy76/gtest/pkg/gtest"
)

func main() {
	// Start an in-process HTTP test server to handle load requests
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Millisecond) // Simulate lightweight work
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer ts.Close()

	suite := gtest.NewSuite("HTTP Checkout Flow Suite")

	suite.RegisterScenario("http_checkout_flow", gtest.Scenario{
		Setup: func(ctx gtest.ScenarioContext) (map[string]any, error) {
			client := &http.Client{
				Timeout: 2 * time.Second,
			}
			return map[string]any{
				"client":     client,
				"server_url": ts.URL,
			}, nil
		},
		PreTest: func(ctx gtest.ScenarioContext) error {
			ctx.Log().Debug().Msg("preparing checkout iteration")
			return nil
		},
		RunVU: func(ctx gtest.ScenarioContext) error {
			client := ctx.GlobalState("client").(*http.Client)
			serverURL := ctx.GlobalState("server_url").(string)

			checkoutPath := ctx.Param("checkout_path")
			if checkoutPath == "" {
				checkoutPath = "/checkout"
			}

			start := time.Now()
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, serverURL+checkoutPath, nil)
			if err != nil {
				ctx.Metrics().Rate("checkout_success_rate", gtest.Tags{}).Add(0, 1)
				return fmt.Errorf("failed to create request: %w", err)
			}

			resp, err := client.Do(req)
			elapsed := time.Since(start)

			ctx.Metrics().Duration("http_request_duration", gtest.Tags{"path": checkoutPath}).Observe(elapsed)

			if err != nil || resp.StatusCode != http.StatusOK {
				ctx.Metrics().Rate("checkout_success_rate", gtest.Tags{}).Add(0, 1)
				if resp != nil {
					_ = resp.Body.Close()
				}
				return fmt.Errorf("http request failed with code %d: %v", resp.StatusCode, err)
			}

			_ = resp.Body.Close()
			ctx.Metrics().Rate("checkout_success_rate", gtest.Tags{}).Add(1, 1)
			ctx.Metrics().Counter("http_requests_total", gtest.Tags{"status": "200"}).Inc()
			return nil
		},
		AfterTest: func(ctx gtest.ScenarioContext) error {
			ctx.Log().Debug().Msg("completed checkout iteration")
			return nil
		},
		Teardown: func(ctx gtest.ScenarioContext, state map[string]any) error {
			return nil
		},
	})

	if err := suite.Execute(); err != nil {
		fmt.Printf("Execution failed: %v\n", err)
	}
}
