package sla_test

import (
	"testing"
	"time"

	"github.com/morphy76/gtest/internal/config"
	"github.com/morphy76/gtest/internal/metric"
	"github.com/morphy76/gtest/internal/sla"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AC-1.4.1: p95 < 200ms passes when actual p95 = 110ms
func TestP95PassesWhenUnderTarget(t *testing.T) {
	store := metric.NewStore()
	d := store.Duration("http_request_duration", metric.Tags{})

	// Record observations to produce ~110ms p95.
	for i := 1; i <= 90; i++ {
		d.Observe(40 * time.Millisecond)
	}
	for i := 91; i <= 95; i++ {
		d.Observe(110 * time.Millisecond)
	}
	for i := 96; i <= 100; i++ {
		d.Observe(150 * time.Millisecond)
	}

	th := config.ThresholdConfig{
		Metric:         "http_request_duration",
		Stat:           "p95",
		Operator:       "<",
		Target:         "200ms",
		TargetDuration: 200 * time.Millisecond,
	}

	res := sla.EvaluateThreshold(th, store)
	assert.True(t, res.Passed, "threshold should pass")
	assert.Contains(t, res.Actual, "ms")
	assert.Empty(t, res.Reason)
}

// AC-1.4.2: p95 < 200ms fails when actual p95 = 250ms
func TestP95FailsWhenOverTarget(t *testing.T) {
	store := metric.NewStore()
	d := store.Duration("http_request_duration", metric.Tags{})

	// Record observations to produce ~250ms p95.
	for i := 1; i <= 90; i++ {
		d.Observe(50 * time.Millisecond)
	}
	for i := 91; i <= 100; i++ {
		d.Observe(250 * time.Millisecond)
	}

	th := config.ThresholdConfig{
		Metric:         "http_request_duration",
		Stat:           "p95",
		Operator:       "<",
		Target:         "200ms",
		TargetDuration: 200 * time.Millisecond,
	}

	res := sla.EvaluateThreshold(th, store)
	assert.False(t, res.Passed, "threshold should fail")
	assert.Contains(t, res.Actual, "ms")
	assert.NotEmpty(t, res.Reason)
}

// AC-1.4.3: rate > 0.995 passes when rate = 0.9992
func TestRatePassesWhenOverTarget(t *testing.T) {
	store := metric.NewStore()
	r := store.Rate("checkout_success_rate", metric.Tags{})

	// Record 9992 successes out of 10000 = 0.9992 rate.
	r.Add(9992, 10000)

	th := config.ThresholdConfig{
		Metric:      "checkout_success_rate",
		Stat:        "rate",
		Operator:    ">",
		Target:      "0.995",
		TargetFloat: 0.995,
	}

	res := sla.EvaluateThreshold(th, store)
	assert.True(t, res.Passed, "rate threshold should pass")
	assert.Equal(t, "0.9992", res.Actual)
	assert.Empty(t, res.Reason)
}

// AC-1.4.4: count >= 100 passes when count = 100
func TestCountPassesWhenEqualTarget(t *testing.T) {
	store := metric.NewStore()
	c := store.Counter("total_orders", metric.Tags{})
	c.Add(100)

	th := config.ThresholdConfig{
		Metric:      "total_orders",
		Stat:        "count",
		Operator:    ">=",
		Target:      "100",
		TargetFloat: 100,
	}

	res := sla.EvaluateThreshold(th, store)
	assert.True(t, res.Passed, "count threshold should pass")
	assert.Equal(t, "100", res.Actual)
	assert.Empty(t, res.Reason)
}

// AC-1.4.5: Metric not found in store returns a failed threshold with "no data" reason
func TestMetricNotFoundReturnsFailedWithNoDataReason(t *testing.T) {
	store := metric.NewStore()

	th := config.ThresholdConfig{
		Metric:         "non_existent_metric",
		Stat:           "p95",
		Operator:       "<",
		Target:         "200ms",
		TargetDuration: 200 * time.Millisecond,
	}

	res := sla.EvaluateThreshold(th, store)
	assert.False(t, res.Passed, "threshold should fail for missing metric")
	assert.Equal(t, "no data", res.Actual)
	assert.Equal(t, "no data", res.Reason)
}

// AC-1.4.6: All thresholds evaluated even if the first one fails (no short-circuit)
func TestAllThresholdsEvaluatedWithoutShortCircuit(t *testing.T) {
	store := metric.NewStore()

	d := store.Duration("http_latency", metric.Tags{})
	d.Observe(300 * time.Millisecond) // p95 will be ~300ms

	c := store.Counter("requests", metric.Tags{})
	c.Add(500)

	thresholds := []config.ThresholdConfig{
		{
			Metric:         "http_latency",
			Stat:           "p95",
			Operator:       "<",
			Target:         "200ms",
			TargetDuration: 200 * time.Millisecond,
		},
		{
			Metric:      "requests",
			Stat:        "count",
			Operator:    ">=",
			Target:      "100",
			TargetFloat: 100,
		},
		{
			Metric:         "missing_metric",
			Stat:           "p90",
			Operator:       "<",
			Target:         "100ms",
			TargetDuration: 100 * time.Millisecond,
		},
	}

	results := sla.Evaluate(thresholds, store)
	require.Len(t, results, 3, "must return results for all 3 thresholds")

	// Threshold 1 (fails)
	assert.False(t, results[0].Passed)
	assert.Contains(t, results[0].Actual, "ms")

	// Threshold 2 (passes)
	assert.True(t, results[1].Passed)
	assert.Equal(t, "500", results[1].Actual)

	// Threshold 3 (fails due to no data)
	assert.False(t, results[2].Passed)
	assert.Equal(t, "no data", results[2].Reason)

	// Verify AllPassed helper
	assert.False(t, sla.AllPassed(results))
}

// Additional test: Gauge threshold evaluation
func TestGaugeThresholdEvaluation(t *testing.T) {
	store := metric.NewStore()
	g := store.Gauge("active_users", metric.Tags{})
	g.Set(50)

	thPass := config.ThresholdConfig{
		Metric:      "active_users",
		Stat:        "value",
		Operator:    "<=",
		Target:      "100",
		TargetFloat: 100,
	}

	resPass := sla.EvaluateThreshold(thPass, store)
	assert.True(t, resPass.Passed)
	assert.Equal(t, "50", resPass.Actual)

	thFail := config.ThresholdConfig{
		Metric:      "active_users",
		Stat:        "value",
		Operator:    ">",
		Target:      "100",
		TargetFloat: 100,
	}

	resFail := sla.EvaluateThreshold(thFail, store)
	assert.False(t, resFail.Passed)
	assert.Equal(t, "50", resFail.Actual)
}

// Additional test: Rate with no data (denominator = 0) returns "no data"
func TestRateNoDataReturnsNoData(t *testing.T) {
	store := metric.NewStore()
	_ = store.Rate("empty_rate", metric.Tags{}) // registered but no Add calls

	th := config.ThresholdConfig{
		Metric:      "empty_rate",
		Stat:        "rate",
		Operator:    ">",
		Target:      "0.5",
		TargetFloat: 0.5,
	}

	res := sla.EvaluateThreshold(th, store)
	assert.False(t, res.Passed)
	assert.Equal(t, "no data", res.Actual)
	assert.Equal(t, "no data", res.Reason)
}
