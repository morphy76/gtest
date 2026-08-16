//go:build vuhive_example

package main

import (
	"fmt"
	"os"

	"github.com/morphy76/vuhive/pkg/vuhive"
)

func main() {
	// 1. Initialize vuhive suite
	suite := vuhive.NewSuite("Conversational AI Load Test Suite")

	// 2. Register conversational flow scenario
	suite.RegisterScenario("conversation_test_flow", vuhive.Scenario{
		Setup:     Setup,
		PreTest:   PreTest,
		RunVU:     RunVU,
		AfterTest: AfterTest,
		Teardown:  Teardown,
	})

	// 3. Execute suite and terminate with exit code
	res := suite.Execute()
	if res.Error != nil {
		fmt.Fprintf(os.Stderr, "Execution error: %v\n", res.Error)
	}
	os.Exit(res.ExitCode())
}
