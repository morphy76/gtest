package vuhive_test

import (
	"context"
	"testing"

	"github.com/morphy76/vuhive/pkg/vuhive"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AC-1.1.1: NewSuite returns a non-nil *Suite
func TestNewSuiteReturnsNonNil(t *testing.T) {
	suite := vuhive.NewSuite("test")
	assert.NotNil(t, suite)
}

// AC-1.1.2: RegisterScenario with a nil RunVU panics
func TestRegisterScenarioWithNilRunVUPanics(t *testing.T) {
	suite := vuhive.NewSuite("test")
	assert.Panics(t, func() {
		suite.RegisterScenario("bad", vuhive.Scenario{RunVU: nil})
	})
}

// AC-1.1.3: RegisterScenario with an empty name panics
func TestRegisterScenarioWithEmptyNamePanics(t *testing.T) {
	suite := vuhive.NewSuite("test")
	assert.Panics(t, func() {
		suite.RegisterScenario("", vuhive.Scenario{
			RunVU: func(ctx vuhive.VUContext) error { return nil },
		})
	})
}

// AC-1.1.4: ScenarioContext is embeddable as context.Context (compile-time check)
func TestScenarioContextEmbeddsContextContext(t *testing.T) {
	var _ context.Context = (vuhive.ScenarioContext)(nil)
	var _ context.Context = (vuhive.SetupContext)(nil)
	var _ context.Context = (vuhive.VUContext)(nil)
	var _ context.Context = (vuhive.TeardownContext)(nil)
	var _ context.Context = (vuhive.SummaryContext)(nil)
	require.True(t, true, "compile-time check passed: context interfaces embed context.Context")
}

// Additional: RegisterScenario with valid input does not panic
func TestRegisterScenarioWithValidInput(t *testing.T) {
	suite := vuhive.NewSuite("test")
	assert.NotPanics(t, func() {
		suite.RegisterScenario("valid", vuhive.Scenario{
			RunVU: func(ctx vuhive.VUContext) error { return nil },
		})
	})
}

// Additional: RegisterScenario with all hooks does not panic
func TestRegisterScenarioWithAllHooks(t *testing.T) {
	suite := vuhive.NewSuite("test")
	assert.NotPanics(t, func() {
		suite.RegisterScenario("full", vuhive.Scenario{
			Setup: func(ctx vuhive.SetupContext) (map[string]any, error) {
				return nil, nil
			},
			PreTest: func(ctx vuhive.VUContext) error {
				return nil
			},
			RunVU: func(ctx vuhive.VUContext) error {
				return nil
			},
			AfterTest: func(ctx vuhive.VUContext) error {
				return nil
			},
			Teardown: func(ctx vuhive.TeardownContext, state map[string]any) error {
				return nil
			},
			HandleSummary: func(ctx vuhive.SummaryContext, summary vuhive.SummaryData) error {
				return nil
			},
		})
	})
}

// Additional: Multiple scenarios can be registered with different names
func TestRegisterMultipleScenarios(t *testing.T) {
	suite := vuhive.NewSuite("test")
	runner := func(ctx vuhive.VUContext) error { return nil }

	assert.NotPanics(t, func() {
		suite.RegisterScenario("scenario_a", vuhive.Scenario{RunVU: runner})
		suite.RegisterScenario("scenario_b", vuhive.Scenario{RunVU: runner})
	})
}

// Issue #39: RegisterScenario panics if called after Execute
func TestRegisterScenarioPanicsAfterExecute(t *testing.T) {
	suite := vuhive.NewSuite("test")
	suite.RegisterScenario("scenario_a", vuhive.Scenario{
		RunVU: func(ctx vuhive.VUContext) error { return nil },
	})

	_ = suite.ExecuteWithArgs([]string{"--version"}, nil)

	assert.PanicsWithValue(t, "vuhive: cannot call RegisterScenario after Execute", func() {
		suite.RegisterScenario("scenario_b", vuhive.Scenario{
			RunVU: func(ctx vuhive.VUContext) error { return nil },
		})
	})
}

// Issue #39: ExecutionResult.ExitCode() returns 0 on success and 1 on failure/abort/error
func TestExecutionResultExitCode(t *testing.T) {
	// Clean success
	resSuccess := vuhive.ExecutionResult{Passed: true}
	assert.Equal(t, 0, resSuccess.ExitCode())

	// Failed threshold (Passed == false)
	resFailed := vuhive.ExecutionResult{Passed: false}
	assert.Equal(t, 1, resFailed.ExitCode())

	// Aborted (Aborted == true)
	resAborted := vuhive.ExecutionResult{Passed: false, Aborted: true, AbortReason: "error rate exceeded"}
	assert.Equal(t, 1, resAborted.ExitCode())

	// Error (Error != nil)
	resError := vuhive.ExecutionResult{Passed: true, Error: assert.AnError}
	assert.Equal(t, 1, resError.ExitCode())
}

