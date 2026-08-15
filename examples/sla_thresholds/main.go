//go:build gtest_example

package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"github.com/morphy76/gtest/pkg/gtest"
)

// startMockBackendServer starts an in-process HTTP mock server simulating order and payment endpoints with jitter.
func startMockBackendServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Simulate realistic backend processing latency with jitter (5ms - 25ms)
		jitter := time.Duration(5+rand.Intn(20)) * time.Millisecond
		time.Sleep(jitter)

		switch r.URL.Path {
		case "/api/orders":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"success","order_id":"ord-12345"}`))

		case "/api/payments":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"authorized","tx_id":"tx-98765"}`))

		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"route not found"}`))
		}
	}))
}

func main() {
	// 1. Launch in-process backend mock server
	ts := startMockBackendServer()
	defer ts.Close()

	// 2. Initialize gtest suite
	suite := gtest.NewSuite("SLA Thresholds & Error Handling Suite")

	// 3. Register scenario with multi-step operations and quality gates
	suite.RegisterScenario("sla_quality_gates", gtest.Scenario{
		Setup: func(ctx gtest.SetupContext) (map[string]any, error) {
			client := &http.Client{Timeout: 2 * time.Second}
			return map[string]any{
				"client":     client,
				"server_url": ts.URL,
			}, nil
		},

		PreTest: func(ctx gtest.VUContext) error {
			ctx.Log().Debug().Int64("vu", ctx.VUID()).Msg("initiating SLA evaluation iteration")

			// Pedagogical Note:
			// Pre-initialize counters with Add(0) in PreTest so the metric is registered in the
			// metric store with value 0 before any errors occur, ensuring accurate threshold evaluation.
			ctx.Metrics().Counter("api_errors_total", gtest.Tags{}).Add(0)
			return nil
		},

		RunVU: func(ctx gtest.VUContext) error {
			client := ctx.GlobalState("client").(*http.Client)
			serverURL := ctx.GlobalState("server_url").(string)

			// Update active concurrent operations gauge metric
			ctx.Metrics().Gauge("concurrent_operations", gtest.Tags{}).Set(float64(ctx.VUID()))

			// Step 1: Place Order
			startOrder := time.Now()
			reqOrder, err := http.NewRequestWithContext(ctx, http.MethodPost, serverURL+"/api/orders", nil)
			if err != nil {
				ctx.Metrics().Rate("order_success_rate", gtest.Tags{}).Add(0, 1)
				return fmt.Errorf("failed to build order request: %w", err)
			}

			respOrder, err := client.Do(reqOrder)
			orderDuration := time.Since(startOrder)
			ctx.Metrics().Duration("order_placement_latency", gtest.Tags{"endpoint": "/api/orders"}).Observe(orderDuration)

			if err != nil || respOrder.StatusCode != http.StatusOK {
				ctx.Metrics().Rate("order_success_rate", gtest.Tags{}).Add(0, 1)
				ctx.Metrics().Counter("api_errors_total", gtest.Tags{"type": "http_failure"}).Inc()
				if respOrder != nil {
					_ = respOrder.Body.Close()
				}
				return fmt.Errorf("order placement failed: %v", err)
			}
			_ = respOrder.Body.Close()
			ctx.Metrics().Rate("order_success_rate", gtest.Tags{}).Add(1, 1)
			ctx.Metrics().Counter("api_requests_total", gtest.Tags{"endpoint": "/api/orders"}).Inc()

			// Check 1: Inline assertion on order response
			ctx.Check("order status is 200", func() string {
				if respOrder.StatusCode != http.StatusOK {
					return fmt.Sprintf("expected HTTP 200, got %d", respOrder.StatusCode)
				}
				return ""
			})

			// Step 2: Authorize Payment
			startPayment := time.Now()
			reqPay, err := http.NewRequestWithContext(ctx, http.MethodPost, serverURL+"/api/payments", nil)
			if err != nil {
				ctx.Metrics().Rate("payment_success_rate", gtest.Tags{}).Add(0, 1)
				return fmt.Errorf("failed to build payment request: %w", err)
			}

			respPay, err := client.Do(reqPay)
			paymentDuration := time.Since(startPayment)
			ctx.Metrics().Duration("payment_auth_latency", gtest.Tags{"endpoint": "/api/payments"}).Observe(paymentDuration)

			if err != nil || respPay.StatusCode != http.StatusOK {
				ctx.Metrics().Rate("payment_success_rate", gtest.Tags{}).Add(0, 1)
				ctx.Metrics().Counter("api_errors_total", gtest.Tags{"type": "payment_failure"}).Inc()
				if respPay != nil {
					_ = respPay.Body.Close()
				}
				return fmt.Errorf("payment auth failed: %v", err)
			}
			_ = respPay.Body.Close()
			ctx.Metrics().Rate("payment_success_rate", gtest.Tags{}).Add(1, 1)
			ctx.Metrics().Counter("api_requests_total", gtest.Tags{"endpoint": "/api/payments"}).Inc()

			// Check 2: Inline assertion on payment response
			ctx.Check("payment status is 200", func() string {
				if respPay.StatusCode != http.StatusOK {
					return fmt.Sprintf("expected HTTP 200, got %d", respPay.StatusCode)
				}
				return ""
			})

			return nil
		},

		AfterTest: func(ctx gtest.VUContext) error {
			ctx.Log().Debug().Int64("vu", ctx.VUID()).Msg("completed SLA evaluation iteration")
			return nil
		},
	})

	// 4. Run the suite and exit with pass/fail exit code
	res := suite.Execute()
	if res.Error != nil {
		fmt.Fprintf(os.Stderr, "Execution error: %v\n", res.Error)
	}
	os.Exit(res.ExitCode())
}
