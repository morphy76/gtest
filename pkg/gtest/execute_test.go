package gtest_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/morphy76/gtest/internal/runner"
	"github.com/morphy76/gtest/pkg/gtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AC-1.8.1: --version flag prints version string and returns nil (no os.Exit)
func TestExecuteVersionFlagReturnsNil(t *testing.T) {
	suite := gtest.NewSuite("Test Suite")
	var stdout bytes.Buffer
	var exitCalled bool

	exitFunc := func(code int) {
		exitCalled = true
	}

	err := suite.ExecuteWithArgs([]string{"--version"}, &stdout, exitFunc)
	require.NoError(t, err, "--version should return nil error")
	assert.False(t, exitCalled, "exitFunc must not be called for --version")
	assert.Contains(t, stdout.String(), "gtest version")
}

// AC-1.8.2: --config pointing to nonexistent file returns *gtest.ConfigError
func TestExecuteNonexistentConfigReturnsConfigError(t *testing.T) {
	suite := gtest.NewSuite("Test Suite")
	var stdout bytes.Buffer

	err := suite.ExecuteWithArgs([]string{"--config", "nonexistent_gtest.yaml"}, &stdout, nil)
	require.Error(t, err)

	var cfgErr *gtest.ConfigError
	require.True(t, errors.As(err, &cfgErr), "must return *gtest.ConfigError, got %T", err)
	assert.Equal(t, "nonexistent_gtest.yaml", cfgErr.Path)
}

// AC-1.8.3: --scenario not in config returns *gtest.ScenarioNotFoundError
func TestExecuteScenarioNotInConfigReturnsScenarioNotFoundError(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "gtest.yaml")

	yamlContent := `version: "1.0"
default_scenario: scenario_a
scenarios:
  scenario_a:
    type: constant_vus
    vus: 1
    run_period: 50ms
    vu_timeout: 1s
`
	require.NoError(t, os.WriteFile(configPath, []byte(yamlContent), 0644))

	suite := gtest.NewSuite("Test Suite")
	suite.RegisterScenario("scenario_a", gtest.Scenario{
		RunVU: func(ctx gtest.ScenarioContext) error { return nil },
	})
	suite.RegisterScenario("unconfigured_scenario", gtest.Scenario{
		RunVU: func(ctx gtest.ScenarioContext) error { return nil },
	})

	var stdout bytes.Buffer
	err := suite.ExecuteWithArgs([]string{"--config", configPath, "--scenario", "unconfigured_scenario"}, &stdout, nil)
	require.Error(t, err)

	var notFoundErr *gtest.ScenarioNotFoundError
	require.True(t, errors.As(err, &notFoundErr), "must return *gtest.ScenarioNotFoundError")
	assert.Equal(t, "unconfigured_scenario", notFoundErr.Name)
}

// AC-1.8.4: Scenario registered but not in config returns *gtest.ScenarioNotFoundError
func TestExecuteScenarioRegisteredNotInConfigReturnsScenarioNotFoundError(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "gtest.yaml")

	yamlContent := `version: "1.0"
default_scenario: scenario_a
scenarios:
  scenario_a:
    type: constant_vus
    vus: 1
    run_period: 50ms
    vu_timeout: 1s
`
	require.NoError(t, os.WriteFile(configPath, []byte(yamlContent), 0644))

	suite := gtest.NewSuite("Test Suite")
	suite.RegisterScenario("scenario_a", gtest.Scenario{
		RunVU: func(ctx gtest.ScenarioContext) error { return nil },
	})

	var stdout bytes.Buffer
	err := suite.ExecuteWithArgs([]string{"--config", configPath, "--scenario", "unregistered_scenario"}, &stdout, nil)
	require.Error(t, err)

	var notFoundErr *gtest.ScenarioNotFoundError
	require.True(t, errors.As(err, &notFoundErr), "must return *gtest.ScenarioNotFoundError")
	assert.Equal(t, "unregistered_scenario", notFoundErr.Name)
}

// AC-1.8.5: All thresholds pass → report printed, os.Exit(0) called
func TestExecuteAllThresholdsPassExitsZero(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "gtest.yaml")

	yamlContent := `version: "1.0"
default_scenario: scenario_a
scenarios:
  scenario_a:
    type: constant_vus
    vus: 1
    run_period: 50ms
    vu_timeout: 1s
    thresholds:
      - metric: http_requests
        stat: count
        operator: ">="
        target: "1"
`
	require.NoError(t, os.WriteFile(configPath, []byte(yamlContent), 0644))

	suite := gtest.NewSuite("Test Suite")
	suite.RegisterScenario("scenario_a", gtest.Scenario{
		RunVU: func(ctx gtest.ScenarioContext) error {
			ctx.Metrics().Counter("http_requests", gtest.Tags{}).Inc()
			return nil
		},
	})

	var stdout bytes.Buffer
	var exitCode int = -1

	exitFunc := func(code int) {
		exitCode = code
	}

	err := suite.ExecuteWithArgs([]string{"--config", configPath}, &stdout, exitFunc)
	require.NoError(t, err)
	assert.Equal(t, 0, exitCode, "must call exitFunc(0) when all thresholds pass")
	assert.Contains(t, stdout.String(), "GTEST LOAD TEST SUMMARY")
	assert.Contains(t, stdout.String(), "[PASS]")
	assert.Contains(t, stdout.String(), "OVERALL: PASSED")
}

// AC-1.8.6: Any threshold fails → report printed with [FAIL] row, os.Exit(1) called
func TestExecuteThresholdBreachedExitsOne(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "gtest.yaml")

	yamlContent := `version: "1.0"
default_scenario: scenario_a
scenarios:
  scenario_a:
    type: constant_vus
    vus: 1
    run_period: 50ms
    vu_timeout: 1s
    thresholds:
      - metric: http_requests
        stat: count
        operator: ">="
        target: "1000"
`
	require.NoError(t, os.WriteFile(configPath, []byte(yamlContent), 0644))

	suite := gtest.NewSuite("Test Suite")
	suite.RegisterScenario("scenario_a", gtest.Scenario{
		RunVU: func(ctx gtest.ScenarioContext) error {
			ctx.Metrics().Counter("http_requests", gtest.Tags{}).Inc()
			time.Sleep(10 * time.Millisecond)
			return nil
		},
	})

	var stdout bytes.Buffer
	var exitCode int = -1

	exitFunc := func(code int) {
		exitCode = code
	}

	err := suite.ExecuteWithArgs([]string{"--config", configPath}, &stdout, exitFunc)
	require.NoError(t, err)
	assert.Equal(t, 1, exitCode, "must call exitFunc(1) when a threshold breaches")
	assert.Contains(t, stdout.String(), "GTEST LOAD TEST SUMMARY")
	assert.Contains(t, stdout.String(), "[FAIL]")
	assert.Contains(t, stdout.String(), "OVERALL: FAILED")
}
