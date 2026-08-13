//go:build gtest_example

package main

import (
	"fmt"

	_ "github.com/morphy76/gtest/internal/runner"
	"github.com/morphy76/gtest/pkg/gtest"
)

func main() {
	suite := gtest.NewSuite("Conversational AI Load Test Suite")

	suite.RegisterScenario("conversation_test_flow", gtest.Scenario{
		Setup:     Setup,
		PreTest:   PreTest,
		RunVU:     RunVU,
		AfterTest: AfterTest,
		Teardown:  Teardown,
	})

	if err := suite.Execute(); err != nil {
		fmt.Printf("Execution failed: %v\n", err)
	}
}
