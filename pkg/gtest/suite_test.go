package gtest_test

import (
	"context"
	"testing"

	"github.com/morphy76/gtest/pkg/gtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AC-1.1.1: NewSuite returns a non-nil *Suite
func TestNewSuiteReturnsNonNil(t *testing.T) {
	suite := gtest.NewSuite("test")
	assert.NotNil(t, suite)
}

// AC-1.1.2: RegisterScenario with a nil RunVU panics
func TestRegisterScenarioWithNilRunVUPanics(t *testing.T) {
	suite := gtest.NewSuite("test")
	assert.Panics(t, func() {
		suite.RegisterScenario("bad", gtest.Scenario{RunVU: nil})
	})
}

// AC-1.1.3: RegisterScenario with an empty name panics
func TestRegisterScenarioWithEmptyNamePanics(t *testing.T) {
	suite := gtest.NewSuite("test")
	assert.Panics(t, func() {
		suite.RegisterScenario("", gtest.Scenario{
			RunVU: func(ctx gtest.VUContext) error { return nil },
		})
	})
}

// AC-1.1.4: ScenarioContext is embeddable as context.Context (compile-time check)
func TestScenarioContextEmbeddsContextContext(t *testing.T) {
	var _ context.Context = (gtest.ScenarioContext)(nil)
	var _ context.Context = (gtest.SetupContext)(nil)
	var _ context.Context = (gtest.VUContext)(nil)
	var _ context.Context = (gtest.TeardownContext)(nil)
	var _ context.Context = (gtest.SummaryContext)(nil)
	require.True(t, true, "compile-time check passed: context interfaces embed context.Context")
}

// Additional: RegisterScenario with valid input does not panic
func TestRegisterScenarioWithValidInput(t *testing.T) {
	suite := gtest.NewSuite("test")
	assert.NotPanics(t, func() {
		suite.RegisterScenario("valid", gtest.Scenario{
			RunVU: func(ctx gtest.VUContext) error { return nil },
		})
	})
}

// Additional: RegisterScenario with all hooks does not panic
func TestRegisterScenarioWithAllHooks(t *testing.T) {
	suite := gtest.NewSuite("test")
	assert.NotPanics(t, func() {
		suite.RegisterScenario("full", gtest.Scenario{
			Setup: func(ctx gtest.SetupContext) (map[string]any, error) {
				return nil, nil
			},
			PreTest: func(ctx gtest.VUContext) error {
				return nil
			},
			RunVU: func(ctx gtest.VUContext) error {
				return nil
			},
			AfterTest: func(ctx gtest.VUContext) error {
				return nil
			},
			Teardown: func(ctx gtest.TeardownContext, state map[string]any) error {
				return nil
			},
			HandleSummary: func(ctx gtest.SummaryContext, summary gtest.SummaryData) error {
				return nil
			},
		})
	})
}

// Additional: Multiple scenarios can be registered with different names
func TestRegisterMultipleScenarios(t *testing.T) {
	suite := gtest.NewSuite("test")
	runner := func(ctx gtest.VUContext) error { return nil }

	assert.NotPanics(t, func() {
		suite.RegisterScenario("scenario_a", gtest.Scenario{RunVU: runner})
		suite.RegisterScenario("scenario_b", gtest.Scenario{RunVU: runner})
	})
}

// Issue #39: RegisterScenario panics if called after Execute
func TestRegisterScenarioPanicsAfterExecute(t *testing.T) {
	suite := gtest.NewSuite("test")
	suite.RegisterScenario("scenario_a", gtest.Scenario{
		RunVU: func(ctx gtest.VUContext) error { return nil },
	})

	_ = suite.ExecuteWithArgs([]string{"--version"}, nil)

	assert.PanicsWithValue(t, "gtest: cannot call RegisterScenario after Execute", func() {
		suite.RegisterScenario("scenario_b", gtest.Scenario{
			RunVU: func(ctx gtest.VUContext) error { return nil },
		})
	})
}

// Issue #39: ExecutionResult.ExitCode() returns 0 on success and 1 on failure/abort/error
func TestExecutionResultExitCode(t *testing.T) {
	// Clean success
	resSuccess := gtest.ExecutionResult{Passed: true}
	assert.Equal(t, 0, resSuccess.ExitCode())

	// Failed threshold (Passed == false)
	resFailed := gtest.ExecutionResult{Passed: false}
	assert.Equal(t, 1, resFailed.ExitCode())

	// Aborted (Aborted == true)
	resAborted := gtest.ExecutionResult{Passed: false, Aborted: true, AbortReason: "error rate exceeded"}
	assert.Equal(t, 1, resAborted.ExitCode())

	// Error (Error != nil)
	resError := gtest.ExecutionResult{Passed: true, Error: assert.AnError}
	assert.Equal(t, 1, resError.ExitCode())
}

