//go:build gtest_example

package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/morphy76/gtest/pkg/gtest"
	"github.com/morphy76/gtest/pkg/gtest/data"
)

const sampleCSV = `username,user_id,role
alice,u101,admin
bob,u102,user
charlie,u103,user
`

func main() {
	// Start an in-process HTTP test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"missing user_id"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{"status":"ok","user_id":"%s"}`, userID)
	}))
	defer ts.Close()

	suite := gtest.NewSuite("Data Parameterization Demo Suite")

	suite.RegisterScenario("data_demo", gtest.Scenario{
		Setup: func(ctx gtest.ScenarioContext) (map[string]any, error) {
			// Load CSV dataset with Sequential round-robin strategy
			ds, err := data.LoadCSV(strings.NewReader(sampleCSV), data.Sequential)
			if err != nil {
				return nil, fmt.Errorf("failed to load CSV dataset: %w", err)
			}

			client := &http.Client{Timeout: 2 * time.Second}
			return map[string]any{
				"dataset":    ds,
				"client":     client,
				"server_url": ts.URL,
			}, nil
		},
		RunVU: func(ctx gtest.ScenarioContext) error {
			ds := ctx.GlobalState("dataset").(*data.DataSet)
			client := ctx.GlobalState("client").(*http.Client)
			serverURL := ctx.GlobalState("server_url").(string)

			// Fetch record parameterized for this VU and iteration
			record, err := ds.Next(ctx)
			if err != nil {
				return fmt.Errorf("failed to get next dataset record: %w", err)
			}

			userID := record["user_id"]
			username := record["username"]

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/api/user?user_id=%s", serverURL, userID), nil)
			if err != nil {
				return fmt.Errorf("failed to build request: %w", err)
			}

			resp, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("request failed for user %s (%s): %w", username, userID, err)
			}
			defer resp.Body.Close()

			// Perform inline check assertions
			ctx.Check("user request status is 200", func() string {
				if resp.StatusCode != http.StatusOK {
					return fmt.Sprintf("user %s (%s) got HTTP %d, expected 200", username, userID, resp.StatusCode)
				}
				return ""
			})

			ctx.Check("record username is non-empty", func() string {
				if username == "" {
					return "parsed dataset record contains empty username"
				}
				return ""
			})

			return nil
		},
	})

	if err := suite.Execute(); err != nil {
		fmt.Printf("Execution failed: %v\n", err)
	}
}
