package gtest_test

import (
	"context"
	"testing"
	"time"

	"github.com/morphy76/gtest/pkg/gtest"
	"github.com/morphy76/gtest/pkg/gtest/data"
	"github.com/stretchr/testify/assert"
)

// Compile-time checks for interface satisfaction
var (
	_ context.Context           = (gtest.ScenarioContext)(nil)
	_ gtest.ExecutionIdentity   = (gtest.ScenarioContext)(nil)
	_ gtest.ConfigProvider      = (gtest.ScenarioContext)(nil)
	_ gtest.StateProvider       = (gtest.ScenarioContext)(nil)
	_ gtest.ObservabilityProvider = (gtest.ScenarioContext)(nil)
	_ gtest.WorkflowController  = (gtest.ScenarioContext)(nil)
	_ data.ContextAccessor      = (gtest.ExecutionIdentity)(nil)
	_ data.ContextAccessor      = (gtest.ScenarioContext)(nil)
)

// Mock implementations verifying that clients only need to implement the small interface they consume

type mockIdentity struct{}

func (mockIdentity) VUID() int64           { return 42 }
func (mockIdentity) Iteration() int64      { return 7 }
func (mockIdentity) ScenarioName() string  { return "mock_scenario" }

type mockConfig struct{}

func (mockConfig) Param(key string) string                                    { return "val_" + key }
func (mockConfig) ParamInt(key string, defaultValue int) int                  { return defaultValue + 1 }
func (mockConfig) ParamDuration(key string, defaultValue time.Duration) time.Duration {
	return defaultValue + time.Second
}

type mockState struct{}

func (mockState) GlobalState(key string) any {
	if key == "token" {
		return "secret"
	}
	return nil
}

type mockWorkflow struct {
	slept  bool
	checkResult bool
}

func (m *mockWorkflow) Sleep(d ...time.Duration) error {
	m.slept = true
	return nil
}

func (m *mockWorkflow) Check(name string, fn gtest.CheckFunc) bool {
	if fn != nil && fn() == "" {
		m.checkResult = true
		return true
	}
	m.checkResult = false
	return false
}

func TestInterfaceSegregation_ExecutionIdentity(t *testing.T) {
	var identity gtest.ExecutionIdentity = mockIdentity{}
	assert.Equal(t, int64(42), identity.VUID())
	assert.Equal(t, int64(7), identity.Iteration())
	assert.Equal(t, "mock_scenario", identity.ScenarioName())

	// Satisfies data.ContextAccessor
	var accessor data.ContextAccessor = identity
	assert.Equal(t, int64(42), accessor.VUID())
	assert.Equal(t, int64(7), accessor.Iteration())
}

func TestInterfaceSegregation_ConfigProvider(t *testing.T) {
	var cfg gtest.ConfigProvider = mockConfig{}
	assert.Equal(t, "val_host", cfg.Param("host"))
	assert.Equal(t, 11, cfg.ParamInt("port", 10))
	assert.Equal(t, 2*time.Second, cfg.ParamDuration("timeout", time.Second))
}

func TestInterfaceSegregation_StateProvider(t *testing.T) {
	var state gtest.StateProvider = mockState{}
	assert.Equal(t, "secret", state.GlobalState("token"))
	assert.Nil(t, state.GlobalState("unknown"))
}

func TestInterfaceSegregation_WorkflowController(t *testing.T) {
	wf := &mockWorkflow{}
	var controller gtest.WorkflowController = wf

	err := controller.Sleep(10 * time.Millisecond)
	assert.NoError(t, err)
	assert.True(t, wf.slept)

	passed := controller.Check("test_pass", func() string { return "" })
	assert.True(t, passed)
	assert.True(t, wf.checkResult)

	failed := controller.Check("test_fail", func() string { return "error" })
	assert.False(t, failed)
	assert.False(t, wf.checkResult)
}
