package gtest_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/morphy76/gtest/pkg/gtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Verify that all core domain interfaces and structs can be constructed and interacted with
// directly via pkg/gtest without any dependency on internal package aliases.

func TestPublicDomainTypesDirectInstantiation(t *testing.T) {
	// Scenario and Hook types
	var setupCalled, preTestCalled, runVUCalled, afterTestCalled, teardownCalled, handleSummaryCalled bool

	sc := gtest.Scenario{
		Setup: func(ctx gtest.ScenarioContext) (map[string]any, error) {
			setupCalled = true
			return map[string]any{"token": "xyz"}, nil
		},
		PreTest: func(ctx gtest.ScenarioContext) error {
			preTestCalled = true
			return nil
		},
		RunVU: func(ctx gtest.ScenarioContext) error {
			runVUCalled = true
			return nil
		},
		AfterTest: func(ctx gtest.ScenarioContext) error {
			afterTestCalled = true
			return nil
		},
		Teardown: func(ctx gtest.ScenarioContext, state map[string]any) error {
			teardownCalled = true
			assert.Equal(t, "xyz", state["token"])
			return nil
		},
		HandleSummary: func(ctx context.Context, summary gtest.SummaryData) error {
			handleSummaryCalled = true
			assert.Equal(t, "Public Domain Suite", summary.SuiteName)
			return nil
		},
	}

	require.NotNil(t, sc.Setup)
	require.NotNil(t, sc.PreTest)
	require.NotNil(t, sc.RunVU)
	require.NotNil(t, sc.AfterTest)
	require.NotNil(t, sc.Teardown)
	require.NotNil(t, sc.HandleSummary)

	// Execute hooks directly to verify type compatibility
	_, err := sc.Setup(nil)
	assert.NoError(t, err)
	assert.True(t, setupCalled)

	assert.NoError(t, sc.PreTest(nil))
	assert.True(t, preTestCalled)

	assert.NoError(t, sc.RunVU(nil))
	assert.True(t, runVUCalled)

	assert.NoError(t, sc.AfterTest(nil))
	assert.True(t, afterTestCalled)

	assert.NoError(t, sc.Teardown(nil, map[string]any{"token": "xyz"}))
	assert.True(t, teardownCalled)

	assert.NoError(t, sc.HandleSummary(context.Background(), gtest.SummaryData{SuiteName: "Public Domain Suite"}))
	assert.True(t, handleSummaryCalled)
}

func TestPublicErrorTypes(t *testing.T) {
	t.Run("ConfigError", func(t *testing.T) {
		baseErr := errors.New("file not found")
		cfgErr := &gtest.ConfigError{Path: "gtest.yaml", Err: baseErr}
		assert.Equal(t, "gtest: configuration error in gtest.yaml: file not found", cfgErr.Error())
		assert.Equal(t, baseErr, cfgErr.Unwrap())

		cfgErrNoPath := &gtest.ConfigError{Err: baseErr}
		assert.Equal(t, "gtest: configuration error: file not found", cfgErrNoPath.Error())
	})

	t.Run("ValidationError", func(t *testing.T) {
		valErr := &gtest.ValidationError{Field: "vus", Message: "must be greater than 0"}
		assert.Equal(t, `gtest: validation error for field "vus": must be greater than 0`, valErr.Error())
	})

	t.Run("ScenarioNotFoundError", func(t *testing.T) {
		snfErr := &gtest.ScenarioNotFoundError{Name: "checkout", Message: "not defined"}
		assert.Equal(t, `gtest: scenario "checkout" not found: not defined`, snfErr.Error())

		snfErrNoName := &gtest.ScenarioNotFoundError{Message: "no scenario specified"}
		assert.Equal(t, "gtest: scenario not found: no scenario specified", snfErrNoName.Error())
	})

	t.Run("SetupError", func(t *testing.T) {
		baseErr := errors.New("db connection failed")
		setupErr := &gtest.SetupError{Err: baseErr}
		assert.Equal(t, "gtest: setup hook failed: db connection failed", setupErr.Error())
		assert.Equal(t, baseErr, setupErr.Unwrap())
	})
}

func TestPublicSummaryDataMethods(t *testing.T) {
	summary := gtest.SummaryData{
		SuiteName: "E2E Performance",
		Scenario:  "smoke",
		Version:   "1.0.0",
		Commit:    "abc1234",
		StartedAt: time.Now().Add(-10 * time.Second),
		EndedAt:   time.Now(),
		Duration:  10 * time.Second,
		Config:    map[string]any{"vus": 5},
		Metrics: []gtest.MetricSummary{
			{Name: "req_total", Type: "counter", Count: 100},
			{Name: "active_vus", Type: "gauge", Value: 5.0},
			{Name: "success_rate", Type: "rate", Rate: 0.99},
			{Name: "latency", Type: "duration", Count: 100, P50: 10 * time.Millisecond, P95: 50 * time.Millisecond, P99: 100 * time.Millisecond},
		},
		Checks: []gtest.CheckSummary{
			{Name: "status_200", Passed: 99, Failed: 1, Total: 100, PassPct: 99.0},
		},
		Thresholds: []gtest.ThresholdSummary{
			{Metric: "latency", Stat: "p95", Operator: "<=", Target: "100ms", Actual: "50ms", Passed: true},
		},
		Passed:  true,
		Aborted: false,
	}

	assert.Equal(t, int64(100), summary.Counter("req_total"))
	assert.Equal(t, int64(0), summary.Counter("nonexistent"))

	assert.InDelta(t, 5.0, summary.Gauge("active_vus"), 1e-9)
	assert.InDelta(t, 0.0, summary.Gauge("nonexistent"), 1e-9)

	assert.InDelta(t, 0.99, summary.Rate("success_rate"), 1e-9)
	assert.InDelta(t, 0.0, summary.Rate("nonexistent"), 1e-9)

	latMetric := summary.Metric("latency")
	require.NotNil(t, latMetric)
	assert.Equal(t, 50*time.Millisecond, latMetric.P95)
	assert.Nil(t, summary.Metric("nonexistent"))

	th := summary.Threshold("latency")
	require.NotNil(t, th)
	assert.True(t, th.Passed)
	assert.Nil(t, summary.Threshold("nonexistent"))

	jsonBytes, err := summary.JSON()
	require.NoError(t, err)
	assert.Contains(t, string(jsonBytes), `"suite_name": "E2E Performance"`)
	assert.Contains(t, string(jsonBytes), `"status_200"`)
}

func TestPublicDelayGenerators(t *testing.T) {
	// Test DelayStrategy constants
	assert.Equal(t, gtest.DelayStrategy("fixed"), gtest.DelayFixed)
	assert.Equal(t, gtest.DelayStrategy("range"), gtest.DelayRange)
	assert.Equal(t, gtest.DelayStrategy("expo"), gtest.DelayExpo)
	assert.Equal(t, gtest.DelayStrategy("gaussian"), gtest.DelayGaussian)

	// FixedDelay
	fGen := gtest.FixedDelay(50 * time.Millisecond)
	assert.Equal(t, 50*time.Millisecond, fGen.Next())

	// RangeDelay
	rGen := gtest.RangeDelay(10*time.Millisecond, 20*time.Millisecond)
	d := rGen.Next()
	assert.GreaterOrEqual(t, d, 10*time.Millisecond)
	assert.LessOrEqual(t, d, 20*time.Millisecond)

	// ExpoDelay
	eGen := gtest.ExpoDelay(100*time.Millisecond, 10*time.Millisecond, 500*time.Millisecond)
	ed := eGen.Next()
	assert.GreaterOrEqual(t, ed, 10*time.Millisecond)
	assert.LessOrEqual(t, ed, 500*time.Millisecond)

	// GaussianDelay
	gGen := gtest.GaussianDelay(100*time.Millisecond, 20*time.Millisecond, 10*time.Millisecond, 300*time.Millisecond)
	gd := gGen.Next()
	assert.GreaterOrEqual(t, gd, 10*time.Millisecond)
	assert.LessOrEqual(t, gd, 300*time.Millisecond)

	// NewDelayGenerator
	cfg := &gtest.InteractionDelayConfig{
		Type:     "fixed",
		Duration: 75 * time.Millisecond,
	}
	cfgGen, err := gtest.NewDelayGenerator(cfg)
	require.NoError(t, err)
	require.NotNil(t, cfgGen)
	assert.Equal(t, 75*time.Millisecond, cfgGen.Next())

	nilGen, err := gtest.NewDelayGenerator(nil)
	require.NoError(t, err)
	assert.Nil(t, nilGen)

	invalidCfg := &gtest.InteractionDelayConfig{
		Type: "unknown_strategy",
	}
	_, err = gtest.NewDelayGenerator(invalidCfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown delay type")
}

func TestCheckFunc(t *testing.T) {
	var passing gtest.CheckFunc = func() string { return "" }
	var failing gtest.CheckFunc = func() string { return "expected 200, got 500" }

	assert.Empty(t, passing())
	assert.Equal(t, "expected 200, got 500", failing())
}
