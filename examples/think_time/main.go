//go:build gtest_example

package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"github.com/morphy76/gtest/pkg/gtest"
)


func main() {
	// Start an in-process mock HTTP server simulating an e-commerce platform
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/catalog":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"products":[{"id":"p1","name":"Wireless Headphones","price":99.99}]}`))
		case "/cart/add":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"item_added","cart_id":"cart-101"}`))
		case "/checkout":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"confirmed","order_id":"ord-999"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"not found"}`))
		}
	}))
	defer ts.Close()

	suite := gtest.NewSuite("Thinking Time & User Delay Demo Suite")

	suite.RegisterScenario("user_journey_with_think_time", gtest.Scenario{
		Setup: func(ctx gtest.ScenarioContext) (map[string]any, error) {
			client := &http.Client{Timeout: 3 * time.Second}
			// Initialize a custom exponential delay generator for intra-step decision modeling
			expoGen := gtest.ExpoDelay(25*time.Millisecond, 10*time.Millisecond, 50*time.Millisecond)

			return map[string]any{
				"client":     client,
				"server_url": ts.URL,
				"expo_delay": expoGen,
			}, nil
		},
		PreTest: func(ctx gtest.ScenarioContext) error {
			ctx.Log().Debug().Msg("initiating user journey iteration")
			return nil
		},
		RunVU: func(ctx gtest.ScenarioContext) error {
			client := ctx.GlobalState("client").(*http.Client)
			serverURL := ctx.GlobalState("server_url").(string)
			expoGen := ctx.GlobalState("expo_delay").(gtest.DelayGenerator)

			// Step 1: Browse catalog
			startCatalog := time.Now()
			req1, err := http.NewRequestWithContext(ctx, http.MethodGet, serverURL+"/catalog", nil)
			if err != nil {
				return fmt.Errorf("failed to create catalog request: %w", err)
			}
			resp1, err := client.Do(req1)
			if err != nil {
				ctx.Metrics().Rate("user_flow_success_rate", gtest.Tags{}).Add(0, 1)
				return fmt.Errorf("catalog request failed: %w", err)
			}
			_ = resp1.Body.Close()
			ctx.Metrics().Duration("catalog_view_duration", gtest.Tags{}).Observe(time.Since(startCatalog))

			// Pause 1: Configured interaction delay from gtest.yaml (respects ctx.Done())
			// Calling ctx.Sleep() with no arguments uses the scenario's configured interaction_delay strategy
			thinkStart1 := time.Now()
			if err := ctx.Sleep(); err != nil {
				return fmt.Errorf("think time aborted: %w", err)
			}
			ctx.Metrics().Duration("think_time_catalog", gtest.Tags{}).Observe(time.Since(thinkStart1))

			// Step 2: Add item to cart
			startCart := time.Now()
			req2, err := http.NewRequestWithContext(ctx, http.MethodPost, serverURL+"/cart/add", nil)
			if err != nil {
				return fmt.Errorf("failed to build cart request: %w", err)
			}
			resp2, err := client.Do(req2)
			if err != nil {
				ctx.Metrics().Rate("user_flow_success_rate", gtest.Tags{}).Add(0, 1)
				return fmt.Errorf("add-to-cart request failed: %w", err)
			}
			_ = resp2.Body.Close()
			ctx.Metrics().Duration("add_to_cart_duration", gtest.Tags{}).Observe(time.Since(startCart))

			// Pause 2: Programmatic pause using custom exponential delay generator
			customPause := expoGen.Next()
			if err := ctx.Sleep(customPause); err != nil {
				return fmt.Errorf("custom pause aborted: %w", err)
			}

			// Step 3: Checkout order
			startCheckout := time.Now()
			req3, err := http.NewRequestWithContext(ctx, http.MethodPost, serverURL+"/checkout", nil)
			if err != nil {
				return fmt.Errorf("failed to build checkout request: %w", err)
			}
			resp3, err := client.Do(req3)
			if err != nil {
				ctx.Metrics().Rate("user_flow_success_rate", gtest.Tags{}).Add(0, 1)
				return fmt.Errorf("checkout request failed: %w", err)
			}
			_ = resp3.Body.Close()
			ctx.Metrics().Duration("checkout_duration", gtest.Tags{}).Observe(time.Since(startCheckout))

			// Flow completed successfully
			ctx.Metrics().Rate("user_flow_success_rate", gtest.Tags{}).Add(1, 1)
			ctx.Metrics().Counter("user_journeys_completed_total", gtest.Tags{}).Inc()

			return nil
		},
		AfterTest: func(ctx gtest.ScenarioContext) error {
			ctx.Log().Debug().Msg("completed user journey iteration")
			return nil
		},
	})

	res := suite.Execute()
	if res.Error != nil {
		fmt.Fprintf(os.Stderr, "Execution error: %v\n", res.Error)
	}
	os.Exit(res.ExitCode())
}
