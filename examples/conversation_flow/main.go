//go:build gtest_example

package main

import (
	"fmt"
	"os"

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

	res := suite.Execute()
	if res.Error != nil {
		fmt.Fprintf(os.Stderr, "Execution error: %v\n", res.Error)
	}
	os.Exit(res.ExitCode())
}

