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
			RunVU: func(ctx gtest.ScenarioContext) error { return nil },
		})
	})
}

// AC-1.1.4: ScenarioContext is embeddable as context.Context (compile-time check)
func TestScenarioContextEmbeddsContextContext(t *testing.T) {
	var _ context.Context = (gtest.ScenarioContext)(nil)
	require.True(t, true, "compile-time check passed: ScenarioContext embeds context.Context")
}

// Additional: RegisterScenario with valid input does not panic
func TestRegisterScenarioWithValidInput(t *testing.T) {
	suite := gtest.NewSuite("test")
	assert.NotPanics(t, func() {
		suite.RegisterScenario("valid", gtest.Scenario{
			RunVU: func(ctx gtest.ScenarioContext) error { return nil },
		})
	})
}

// Additional: RegisterScenario with all hooks does not panic
func TestRegisterScenarioWithAllHooks(t *testing.T) {
	suite := gtest.NewSuite("test")
	assert.NotPanics(t, func() {
		suite.RegisterScenario("full", gtest.Scenario{
			Setup: func(ctx gtest.ScenarioContext) (map[string]any, error) {
				return nil, nil
			},
			PreTest: func(ctx gtest.ScenarioContext) error {
				return nil
			},
			RunVU: func(ctx gtest.ScenarioContext) error {
				return nil
			},
			AfterTest: func(ctx gtest.ScenarioContext) error {
				return nil
			},
			Teardown: func(ctx gtest.ScenarioContext, state map[string]any) error {
				return nil
			},
		})
	})
}

// Additional: Multiple scenarios can be registered with different names
func TestRegisterMultipleScenarios(t *testing.T) {
	suite := gtest.NewSuite("test")
	runner := func(ctx gtest.ScenarioContext) error { return nil }

	assert.NotPanics(t, func() {
		suite.RegisterScenario("scenario_a", gtest.Scenario{RunVU: runner})
		suite.RegisterScenario("scenario_b", gtest.Scenario{RunVU: runner})
	})
}
