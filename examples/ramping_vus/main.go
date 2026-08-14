//go:build gtest_example

package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"time"

	"github.com/morphy76/gtest/pkg/gtest"
)

func main() {
	var requestCount int64

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&requestCount, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","timestamp":"` + time.Now().UTC().Format(time.RFC3339) + `"}`))
	}))
	defer ts.Close()

	suite := gtest.NewSuite("Ramping VUs Spike Test Demo Suite")

	suite.RegisterScenario("spike_test", gtest.Scenario{
		Setup: func(ctx gtest.SetupContext) (map[string]any, error) {
			client := &http.Client{Timeout: 3 * time.Second}
			return map[string]any{
				"client":     client,
				"server_url": ts.URL,
			}, nil
		},
		PreTest: func(ctx gtest.VUContext) error {
			ctx.Log().Debug().Int64("vu_id", ctx.VUID()).Msg("initializing virtual user for ramping test")
			return nil
		},
		RunVU: func(ctx gtest.VUContext) error {
			client := ctx.GlobalState("client").(*http.Client)
			serverURL := ctx.GlobalState("server_url").(string)

			start := time.Now()
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, serverURL, nil)
			if err != nil {
				return fmt.Errorf("failed to create request: %w", err)
			}

			resp, err := client.Do(req)
			if err != nil {
				ctx.Metrics().Rate("api_success_rate", gtest.Tags{}).Add(0, 1)
				return fmt.Errorf("http request failed: %w", err)
			}
			_ = resp.Body.Close()

			ctx.Metrics().Duration("api_response_time", gtest.Tags{}).Observe(time.Since(start))
			ctx.Metrics().Rate("api_success_rate", gtest.Tags{}).Add(1, 1)
			ctx.Metrics().Counter("api_requests_total", gtest.Tags{}).Inc()

			return nil
		},
		AfterTest: func(ctx gtest.VUContext) error {
			ctx.Log().Debug().Int64("vu_id", ctx.VUID()).Msg("virtual user ramping lifecycle finished")
			return nil
		},
	})

	res := suite.Execute()
	if res.Error != nil {
		fmt.Fprintf(os.Stderr, "Execution error: %v\n", res.Error)
	}
	os.Exit(res.ExitCode())
}
