//go:build gtest_example

package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/morphy76/gtest/pkg/gtest"
)

type userDB struct {
	users map[int]string
}

func main() {
	suite := gtest.NewSuite("gRPC User Service Load Test")

	suite.RegisterScenario("grpc_user_service_flow", gtest.Scenario{
		Setup: func(ctx gtest.ScenarioContext) (map[string]any, error) {
			db := &userDB{
				users: map[int]string{
					1: "Alice",
					2: "Bob",
					3: "Charlie",
				},
			}
			return map[string]any{
				"db": db,
			}, nil
		},
		PreTest: func(ctx gtest.ScenarioContext) error {
			ctx.Log().Debug().Msg("preparing RPC invocation")
			return nil
		},
		RunVU: func(ctx gtest.ScenarioContext) error {
			db := ctx.GlobalState("db").(*userDB)
			serviceName := ctx.Param("service_name")
			method := ctx.Param("method")

			start := time.Now()
			userID := rand.Intn(3) + 1
			userName, found := db.users[userID]
			latency := time.Duration(2+rand.Intn(5)) * time.Millisecond
			time.Sleep(latency)
			elapsed := time.Since(start)

			ctx.Metrics().Duration("grpc_latency", gtest.Tags{
				"service": serviceName,
				"method":  method,
			}).Observe(elapsed)

			if !found {
				ctx.Metrics().Rate("rpc_success_rate", gtest.Tags{}).Add(0, 1)
				return fmt.Errorf("user %d not found", userID)
			}

			_ = userName
			ctx.Metrics().Rate("rpc_success_rate", gtest.Tags{}).Add(1, 1)
			ctx.Metrics().Counter("grpc_calls_total", gtest.Tags{"status": "OK"}).Inc()
			return nil
		},
		AfterTest: func(ctx gtest.ScenarioContext) error {
			ctx.Log().Debug().Msg("completed RPC invocation")
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
