package vuhive_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/morphy76/vuhive/pkg/vuhive"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AC-1.8.1: --version flag prints version string and returns ExecutionResult (no os.Exit)
func TestExecuteVersionFlagReturnsNil(t *testing.T) {
	suite := vuhive.NewSuite("Test Suite")
	var stdout bytes.Buffer

	res := suite.ExecuteWithArgs([]string{"--version"}, &stdout)
	require.NoError(t, res.Error, "--version should return nil error")
	assert.True(t, res.Passed)
	assert.False(t, res.Aborted)
	assert.Equal(t, 0, res.ExitCode())
	assert.Contains(t, stdout.String(), "vuhive version")
}

// AC-1.8.2: --config pointing to nonexistent file returns *vuhive.ConfigError
func TestExecuteNonexistentConfigReturnsConfigError(t *testing.T) {
	suite := vuhive.NewSuite("Test Suite")
	var stdout bytes.Buffer

	res := suite.ExecuteWithArgs([]string{"--config", "nonexistent_vuhive.yaml"}, &stdout)
	require.Error(t, res.Error)
	assert.Equal(t, 1, res.ExitCode())

	var cfgErr *vuhive.ConfigError
	require.True(t, errors.As(res.Error, &cfgErr), "must return *vuhive.ConfigError, got %T", res.Error)
	assert.Equal(t, "nonexistent_vuhive.yaml", cfgErr.Path)
}

// AC-1.8.3: --scenario not in config returns *vuhive.ScenarioNotFoundError
func TestExecuteScenarioNotInConfigReturnsScenarioNotFoundError(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "vuhive.yaml")

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

	suite := vuhive.NewSuite("Test Suite")
	suite.RegisterScenario("scenario_a", vuhive.Scenario{
		RunVU: func(ctx vuhive.VUContext) error { return nil },
	})
	suite.RegisterScenario("unconfigured_scenario", vuhive.Scenario{
		RunVU: func(ctx vuhive.VUContext) error { return nil },
	})

	var stdout bytes.Buffer
	res := suite.ExecuteWithArgs([]string{"--config", configPath, "--scenario", "unconfigured_scenario"}, &stdout)
	require.Error(t, res.Error)
	assert.Equal(t, 1, res.ExitCode())

	var notFoundErr *vuhive.ScenarioNotFoundError
	require.True(t, errors.As(res.Error, &notFoundErr), "must return *vuhive.ScenarioNotFoundError")
	assert.Equal(t, "unconfigured_scenario", notFoundErr.Name)
}

// AC-1.8.4: Scenario registered but not in config returns *vuhive.ScenarioNotFoundError
func TestExecuteScenarioRegisteredNotInConfigReturnsScenarioNotFoundError(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "vuhive.yaml")

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

	suite := vuhive.NewSuite("Test Suite")
	suite.RegisterScenario("scenario_a", vuhive.Scenario{
		RunVU: func(ctx vuhive.VUContext) error { return nil },
	})

	var stdout bytes.Buffer
	res := suite.ExecuteWithArgs([]string{"--config", configPath, "--scenario", "unregistered_scenario"}, &stdout)
	require.Error(t, res.Error)
	assert.Equal(t, 1, res.ExitCode())

	var notFoundErr *vuhive.ScenarioNotFoundError
	require.True(t, errors.As(res.Error, &notFoundErr), "must return *vuhive.ScenarioNotFoundError")
	assert.Equal(t, "unregistered_scenario", notFoundErr.Name)
}

// AC-1.8.5: All thresholds pass → report printed, ExitCode() is 0
func TestExecuteAllThresholdsPassExitsZero(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "vuhive.yaml")

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

	suite := vuhive.NewSuite("Test Suite")
	suite.RegisterScenario("scenario_a", vuhive.Scenario{
		RunVU: func(ctx vuhive.VUContext) error {
			ctx.Metrics().Counter("http_requests", vuhive.Tags{}).Inc()
			return nil
		},
	})

	var stdout bytes.Buffer
	res := suite.ExecuteWithArgs([]string{"--config", configPath}, &stdout)
	require.NoError(t, res.Error)
	assert.True(t, res.Passed)
	assert.False(t, res.Aborted)
	assert.Equal(t, 0, res.ExitCode(), "must return ExitCode() == 0 when all thresholds pass")
	assert.Contains(t, stdout.String(), "VUHIVE LOAD TEST SUMMARY")
	assert.Contains(t, stdout.String(), "[PASS]")
	assert.Contains(t, stdout.String(), "OVERALL: PASSED")
}

// AC-1.8.6: Any threshold fails → report printed with [FAIL] row, ExitCode() is 1
func TestExecuteThresholdBreachedExitsOne(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "vuhive.yaml")

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

	suite := vuhive.NewSuite("Test Suite")
	suite.RegisterScenario("scenario_a", vuhive.Scenario{
		RunVU: func(ctx vuhive.VUContext) error {
			ctx.Metrics().Counter("http_requests", vuhive.Tags{}).Inc()
			time.Sleep(10 * time.Millisecond)
			return nil
		},
	})

	var stdout bytes.Buffer
	res := suite.ExecuteWithArgs([]string{"--config", configPath}, &stdout)
	require.NoError(t, res.Error)
	assert.False(t, res.Passed)
	assert.Equal(t, 1, res.ExitCode(), "must return ExitCode() == 1 when a threshold breaches")
	assert.Contains(t, stdout.String(), "VUHIVE LOAD TEST SUMMARY")
	assert.Contains(t, stdout.String(), "[FAIL]")
	assert.Contains(t, stdout.String(), "OVERALL: FAILED")
}

func TestExecuteJSONReportOutFlag(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "vuhive.yaml")
	jsonReportPath := filepath.Join(tempDir, "report.json")

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

	suite := vuhive.NewSuite("JSON Export Test Suite")
	suite.RegisterScenario("scenario_a", vuhive.Scenario{
		RunVU: func(ctx vuhive.VUContext) error {
			ctx.Metrics().Counter("http_requests", vuhive.Tags{}).Inc()
			return nil
		},
	})

	var stdout bytes.Buffer
	res := suite.ExecuteWithArgs([]string{"--config", configPath, "--json-report-out", jsonReportPath}, &stdout)
	require.NoError(t, res.Error)
	assert.True(t, res.Passed)
	assert.Equal(t, 0, res.ExitCode())

	assert.Contains(t, stdout.String(), "VUHIVE LOAD TEST SUMMARY", "console summary should be printed to stdout")

	jsonBytes, err := os.ReadFile(jsonReportPath)
	require.NoError(t, err, "json report file must exist")
	assert.Contains(t, string(jsonBytes), `"suite_name": "JSON Export Test Suite"`)
	assert.Contains(t, string(jsonBytes), `"scenario": "scenario_a"`)
	assert.Contains(t, string(jsonBytes), `"name": "http_requests"`)
}


