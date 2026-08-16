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

// AC-1.15.1: HandleSummary is invoked after test completion and report generation
func TestHandleSummaryInvokedPostExecution(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "vuhive.yaml")

	yamlContent := `version: "1.0"
default_scenario: summary_test
scenarios:
  summary_test:
    type: constant_vus
    vus: 1
    run_period: 50ms
    vu_timeout: 1s
`
	require.NoError(t, os.WriteFile(configPath, []byte(yamlContent), 0644))

	var handleSummaryCalled bool
	var capturedSummary vuhive.SummaryData

	suite := vuhive.NewSuite("Summary Test Suite")
	suite.RegisterScenario("summary_test", vuhive.Scenario{
		RunVU: func(ctx vuhive.VUContext) error {
			ctx.Metrics().Counter("custom_counter", vuhive.Tags{}).Inc()
			return nil
		},
		HandleSummary: func(ctx vuhive.SummaryContext, summary vuhive.SummaryData) error {
			handleSummaryCalled = true
			capturedSummary = summary
			// Test SummaryContext methods
			_ = ctx.Param("nonexistent")
			ctx.Log().Debug().Msg("summary hook execution")
			return nil
		},
	})

	var stdout bytes.Buffer
	res := suite.ExecuteWithArgs([]string{"--config", configPath}, &stdout)
	require.NoError(t, res.Error)

	assert.True(t, handleSummaryCalled, "HandleSummary must be invoked after test execution")
	assert.Equal(t, "Summary Test Suite", capturedSummary.SuiteName)
	assert.Equal(t, "summary_test", capturedSummary.Scenario)
}

// AC-1.15.2: SummaryData contains complete suite name, scenario name, execution duration, metrics, and SLA threshold results
func TestSummaryDataContents(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "vuhive.yaml")

	yamlContent := `version: "1.0"
default_scenario: metrics_summary_test
scenarios:
  metrics_summary_test:
    type: constant_vus
    vus: 2
    run_period: 80ms
    vu_timeout: 1s
    thresholds:
      - metric: requests
        stat: count
        operator: ">="
        target: "1"
`
	require.NoError(t, os.WriteFile(configPath, []byte(yamlContent), 0644))

	var capturedSummary vuhive.SummaryData

	suite := vuhive.NewSuite("Complete Summary Suite")
	suite.RegisterScenario("metrics_summary_test", vuhive.Scenario{
		RunVU: func(ctx vuhive.VUContext) error {
			ctx.Metrics().Counter("requests", vuhive.Tags{}).Inc()
			ctx.Metrics().Duration("latency", vuhive.Tags{}).Observe(10 * time.Millisecond)
			ctx.Metrics().Rate("success_rate", vuhive.Tags{}).Add(1, 1)
			return nil
		},
		HandleSummary: func(ctx vuhive.SummaryContext, summary vuhive.SummaryData) error {
			capturedSummary = summary
			return nil
		},
	})

	var stdout bytes.Buffer
	res := suite.ExecuteWithArgs([]string{"--config", configPath}, &stdout)
	require.NoError(t, res.Error)

	// Validate metadata
	assert.Equal(t, "Complete Summary Suite", capturedSummary.SuiteName)
	assert.Equal(t, "metrics_summary_test", capturedSummary.Scenario)
	assert.False(t, capturedSummary.StartedAt.IsZero())
	assert.False(t, capturedSummary.EndedAt.IsZero())
	assert.Greater(t, capturedSummary.Duration, time.Duration(0))
	assert.True(t, capturedSummary.Passed)

	// Validate thresholds
	require.Len(t, capturedSummary.Thresholds, 1)
	assert.Equal(t, "requests", capturedSummary.Thresholds[0].Metric)
	assert.Equal(t, "count", capturedSummary.Thresholds[0].Stat)
	assert.True(t, capturedSummary.Thresholds[0].Passed)

	// Validate metrics
	require.NotEmpty(t, capturedSummary.Metrics)
	var foundCounter, foundDuration, foundRate bool
	for _, m := range capturedSummary.Metrics {
		if m.Name == "requests" {
			foundCounter = true
			assert.Equal(t, "counter", m.Type)
			assert.GreaterOrEqual(t, m.Count, int64(1))
		}
		if m.Name == "latency" {
			foundDuration = true
			assert.Equal(t, "duration", m.Type)
			assert.GreaterOrEqual(t, m.Count, int64(1))
		}
		if m.Name == "success_rate" {
			foundRate = true
			assert.Equal(t, "rate", m.Type)
			assert.InDelta(t, 1.0, m.Rate, 1e-9)
		}
	}
	assert.True(t, foundCounter, "counter metric must be present in summary")
	assert.True(t, foundDuration, "duration metric must be present in summary")
	assert.True(t, foundRate, "rate metric must be present in summary")
}

// AC-1.15.3: Error returned by HandleSummary is logged as an error but does not mutate the exit code
func TestHandleSummaryErrorDoesNotMutateExitCode(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "vuhive.yaml")

	yamlContent := `version: "1.0"
default_scenario: summary_err_test
scenarios:
  summary_err_test:
    type: constant_vus
    vus: 1
    run_period: 50ms
    vu_timeout: 1s
    thresholds:
      - metric: requests
        stat: count
        operator: ">="
        target: "1"
`
	require.NoError(t, os.WriteFile(configPath, []byte(yamlContent), 0644))

	suite := vuhive.NewSuite("Summary Error Suite")
	suite.RegisterScenario("summary_err_test", vuhive.Scenario{
		RunVU: func(ctx vuhive.VUContext) error {
			ctx.Metrics().Counter("requests", vuhive.Tags{}).Inc()
			return nil
		},
		HandleSummary: func(ctx vuhive.SummaryContext, summary vuhive.SummaryData) error {
			return errors.New("slack webhook timeout")
		},
	})

	var stdout bytes.Buffer
	res := suite.ExecuteWithArgs([]string{"--config", configPath}, &stdout)
	require.NoError(t, res.Error)

	// Since threshold passed, exit code must remain 0 despite HandleSummary error
	assert.Equal(t, 0, res.ExitCode(), "HandleSummary error must not mutate exit code 0 to 1")
}

func TestSummaryDataHelperMethods(t *testing.T) {
	summary := vuhive.SummaryData{
		SuiteName: "Helper Suite",
		Scenario:  "scenario_1",
		Passed:    true,
		Metrics: []vuhive.MetricSummary{
			{Name: "req_count", Type: "counter", Count: 42},
			{Name: "active_vus", Type: "gauge", Value: 5.0},
			{Name: "success_ratio", Type: "rate", Rate: 0.98},
			{Name: "latency", Type: "duration", Count: 100, P95: 150 * time.Millisecond},
		},
		Thresholds: []vuhive.ThresholdSummary{
			{Metric: "latency", Stat: "p95", Operator: "<", Target: "200ms", Actual: "150ms", Passed: true},
		},
	}

	assert.Equal(t, int64(42), summary.Counter("req_count"))
	assert.Equal(t, int64(0), summary.Counter("nonexistent"))

	assert.InDelta(t, 5.0, summary.Gauge("active_vus"), 1e-9)
	assert.InDelta(t, 0.0, summary.Gauge("nonexistent"), 1e-9)

	assert.InDelta(t, 0.98, summary.Rate("success_ratio"), 1e-9)
	assert.InDelta(t, 0.0, summary.Rate("nonexistent"), 1e-9)

	m := summary.Metric("latency")
	require.NotNil(t, m)
	assert.Equal(t, 150*time.Millisecond, m.P95)
	assert.Nil(t, summary.Metric("nonexistent"))

	th := summary.Threshold("latency")
	require.NotNil(t, th)
	assert.True(t, th.Passed)
	assert.Nil(t, summary.Threshold("nonexistent"))

	jsonBytes, err := summary.JSON()
	require.NoError(t, err)
	assert.Contains(t, string(jsonBytes), `"suite_name": "Helper Suite"`)
	assert.Contains(t, string(jsonBytes), `"req_count"`)
}

