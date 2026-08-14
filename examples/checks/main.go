//go:build gtest_example

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/morphy76/gtest/pkg/gtest"
)

type apiResponse struct {
	Status  string `json:"status"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func main() {
	// Start an in-process HTTP server simulating an API backend
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"success","code":200,"message":"operation completed"}`))
	}))
	defer ts.Close()

	suite := gtest.NewSuite("Checks Demo Suite")

	suite.RegisterScenario("checks_demo", gtest.Scenario{
		Setup: func(ctx gtest.ScenarioContext) (map[string]any, error) {
			client := &http.Client{Timeout: 2 * time.Second}
			return map[string]any{
				"client":     client,
				"server_url": ts.URL,
			}, nil
		},
		RunVU: func(ctx gtest.ScenarioContext) error {
			client := ctx.GlobalState("client").(*http.Client)
			serverURL := ctx.GlobalState("server_url").(string)

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, serverURL+"/api/resource", nil)
			if err != nil {
				return fmt.Errorf("failed to create request: %w", err)
			}

			resp, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("http request failed: %w", err)
			}
			defer resp.Body.Close()

			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to read response body: %w", err)
			}

			// Check 1: HTTP Status Code is 200
			ctx.Check("status code is 200", func() string {
				if resp.StatusCode != http.StatusOK {
					return fmt.Sprintf("expected HTTP 200, got %d", resp.StatusCode)
				}
				return ""
			})

			// Check 2: Content-Type header is JSON
			ctx.Check("content-type is json", func() string {
				ct := resp.Header.Get("Content-Type")
				if !strings.Contains(ct, "application/json") {
					return fmt.Sprintf("expected application/json, got %q", ct)
				}
				return ""
			})

			// Check 3: JSON response payload status field is "success"
			var res apiResponse
			if err := json.Unmarshal(bodyBytes, &res); err != nil {
				ctx.Check("response body is valid json", func() string {
					return fmt.Sprintf("invalid json payload: %v", err)
				})
			} else {
				ctx.Check("response status is success", func() string {
					if res.Status != "success" {
						return fmt.Sprintf("expected status 'success', got %q", res.Status)
					}
					return ""
				})
			}

			return nil
		},
	})

	if err := suite.Execute(); err != nil {
		fmt.Printf("Execution failed: %v\n", err)
	}
}
