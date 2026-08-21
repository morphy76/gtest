//go:build vuhive_example

package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"github.com/morphy76/vuhive/pkg/vuhive"
	vuhivehttp "github.com/morphy76/vuhive/pkg/vuhive/http"
)

// --- Mock Backend (Test Infrastructure) ---
// In production load tests, this would be replaced by your actual target system URL
// configured via ctx.Param("base_url") in vuhive.yaml.

type checkoutResponse struct {
	Status  string `json:"status"`
	OrderID string `json:"order_id"`
}

func startMockCheckoutServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Millisecond) // Simulate backend processing
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","order_id":"ord-1001"}`))
	}))
}

// --- Load Test Scenario ---

func main() {
	// 1. Start mock backend
	ts := startMockCheckoutServer()
	defer ts.Close()

	// 2. Initialize vuhive suite
	suite := vuhive.NewSuite("HTTP Module Demo Suite")

	// 3. Register scenario using the built-in HTTP module
	suite.RegisterScenario("http_module_demo", vuhive.Scenario{
		// Setup: create a SHARED instrumented HTTP client from declarative config
		//
		// Declarative HTTP client settings (timeout, headers, pool) are defined in vuhive.yaml
		// under scenarios.<scenario>.http and automatically bound via NewClientFromConfig(ctx).
		// Programmatic options can override settings (e.g. dynamic mock server URL).
		Setup: func(ctx vuhive.SetupContext) (map[string]any, error) {
			client := vuhivehttp.NewClientFromConfig(ctx,
				vuhivehttp.WithBaseURL(ts.URL),
			)
			return map[string]any{
				"client": client,
			}, nil
		},

		// RunVU: execute HTTP requests with automatic instrumentation
		RunVU: func(ctx vuhive.VUContext) error {
			// Step 1: Retrieve the shared instrumented client from global state
			client := ctx.GlobalState("client").(*vuhivehttp.Client)

			checkoutPath := ctx.Param("checkout_path")
			if checkoutPath == "" {
				checkoutPath = "/checkout"
			}

			// Step 2: Execute request — relative path resolved against client's BaseURL, metrics recorded automatically
			resp, err := client.Get(ctx, checkoutPath)
			if err != nil {
				return fmt.Errorf("checkout request failed: %w", err)
			}

			// Step 3: Validate response using inline checks
			ctx.Check("status_200", func() string {
				if resp.StatusCode != http.StatusOK {
					return fmt.Sprintf("expected 200, got %d", resp.StatusCode)
				}
				return ""
			})

			// Step 4: Parse JSON response body
			var result checkoutResponse
			if err := resp.JSON(&result); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}

			ctx.Check("order_created", func() string {
				if result.OrderID == "" {
					return "missing order_id in response"
				}
				return ""
			})

			return nil
		},

		Teardown: func(ctx vuhive.TeardownContext, _ map[string]any) error {
			ctx.Log().Info().Msg("HTTP module demo completed")
			return nil
		},
	})

	// 4. Execute and exit
	res := suite.Execute()
	if res.Error != nil {
		fmt.Fprintf(os.Stderr, "Execution error: %v\n", res.Error)
	}
	os.Exit(res.ExitCode())
}
