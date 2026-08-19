package sla_test

import (
	"testing"
	"time"

	"github.com/morphy76/vuhive/internal/config"
	"github.com/morphy76/vuhive/internal/metric"
	"github.com/morphy76/vuhive/internal/sla"
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

type mockMetricReader struct {
	metricTypes map[string]metric.MetricType
	histograms  map[string]metric.HistogramSnapshot
	counters    map[string]int64
	rates       map[string]float64
	rateHasData map[string]bool
	gauges      map[string]float64
}

func (m *mockMetricReader) MetricType(name string) (metric.MetricType, bool) {
	mt, ok := m.metricTypes[name]
	return mt, ok
}

func (m *mockMetricReader) MergedHistogramSnapshot(name string) metric.HistogramSnapshot {
	return m.histograms[name]
}

func (m *mockMetricReader) AggregatedCounterValue(name string) int64 {
	return m.counters[name]
}

func (m *mockMetricReader) RateData(name string) (float64, bool) {
	val, ok := m.rates[name]
	hasData := m.rateHasData[name]
	return val, ok && hasData
}

func (m *mockMetricReader) LastGaugeValue(name string) float64 {
	return m.gauges[name]
}

func TestEvaluateWithMockMetricReader(t *testing.T) {
	mockReader := &mockMetricReader{
		metricTypes: map[string]metric.MetricType{
			"mock_duration": metric.MetricTypeDuration,
			"mock_counter":  metric.MetricTypeCounter,
		},
		histograms: map[string]metric.HistogramSnapshot{
			"mock_duration": {
				Count: 10,
				P95:   80 * time.Millisecond,
			},
		},
		counters: map[string]int64{
			"mock_counter": 150,
		},
	}

	thresholds := []config.ThresholdConfig{
		{
			Metric:         "mock_duration",
			Stat:           "p95",
			Operator:       "<",
			Target:         "100ms",
			TargetDuration: 100 * time.Millisecond,
		},
		{
			Metric:      "mock_counter",
			Stat:        "count",
			Operator:    ">=",
			Target:      "100",
			TargetFloat: 100,
		},
	}

	results := sla.Evaluate(thresholds, mockReader)
	require.Len(t, results, 2)
	assert.True(t, results[0].Passed)
	assert.True(t, results[1].Passed)
	assert.True(t, sla.AllPassed(results))
}

func TestComparisonOperators(t *testing.T) {
	store := metric.NewStore()
	d := store.Duration("latency", metric.Tags{})
	d.Observe(100 * time.Millisecond)

	c := store.Counter("reqs", metric.Tags{})
	c.Add(50)

	testCases := []struct {
		name     string
		th       config.ThresholdConfig
		expected bool
	}{
		{
			name: "duration less than pass",
			th: config.ThresholdConfig{
				Metric: "latency", Stat: "p50", Operator: "<", Target: "200ms", TargetDuration: 200 * time.Millisecond,
			},
			expected: true,
		},
		{
			name: "duration less than or equal pass",
			th: config.ThresholdConfig{
				Metric: "latency", Stat: "p50", Operator: "<=", Target: "200ms", TargetDuration: 200 * time.Millisecond,
			},
			expected: true,
		},
		{
			name: "duration greater than fail",
			th: config.ThresholdConfig{
				Metric: "latency", Stat: "p50", Operator: ">", Target: "200ms", TargetDuration: 200 * time.Millisecond,
			},
			expected: false,
		},
		{
			name: "duration greater than or equal pass",
			th: config.ThresholdConfig{
				Metric: "latency", Stat: "p50", Operator: ">=", Target: "50ms", TargetDuration: 50 * time.Millisecond,
			},
			expected: true,
		},
		{
			name: "duration invalid operator fail",
			th: config.ThresholdConfig{
				Metric: "latency", Stat: "p50", Operator: "==", Target: "100ms", TargetDuration: 100 * time.Millisecond,
			},
			expected: false,
		},
		{
			name: "float less than fail",
			th: config.ThresholdConfig{
				Metric: "reqs", Stat: "count", Operator: "<", Target: "50", TargetFloat: 50,
			},
			expected: false,
		},
		{
			name: "float less than or equal pass",
			th: config.ThresholdConfig{
				Metric: "reqs", Stat: "count", Operator: "<=", Target: "50", TargetFloat: 50,
			},
			expected: true,
		},
		{
			name: "float greater than pass",
			th: config.ThresholdConfig{
				Metric: "reqs", Stat: "count", Operator: ">", Target: "40", TargetFloat: 40,
			},
			expected: true,
		},
		{
			name: "float greater than or equal pass",
			th: config.ThresholdConfig{
				Metric: "reqs", Stat: "count", Operator: ">=", Target: "50", TargetFloat: 50,
			},
			expected: true,
		},
		{
			name: "float invalid operator fail",
			th: config.ThresholdConfig{
				Metric: "reqs", Stat: "count", Operator: "unknown", Target: "50", TargetFloat: 50,
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := sla.EvaluateThreshold(tc.th, store)
			assert.Equal(t, tc.expected, res.Passed)
		})
	}
}

func TestEvaluateThreshold_Counter_NoData_ZeroTarget_Passes(t *testing.T) {
	store := metric.NewStore()

	th := config.ThresholdConfig{
		Metric:      "authentication_errors",
		Stat:        "count",
		Operator:    "<=",
		Target:      "0",
		TargetFloat: 0,
	}

	res := sla.EvaluateThreshold(th, store)
	assert.True(t, res.Passed, "zero-target counter threshold should pass when no data recorded")
	assert.Equal(t, "0", res.Actual)
	assert.Empty(t, res.Reason)
}

func TestEvaluateThreshold_Counter_NoData_PositiveTarget_Fails(t *testing.T) {
	store := metric.NewStore()

	th := config.ThresholdConfig{
		Metric:      "authentication_errors",
		Stat:        "count",
		Operator:    ">",
		Target:      "0",
		TargetFloat: 0,
	}

	res := sla.EvaluateThreshold(th, store)
	assert.False(t, res.Passed, "counter > 0 threshold should fail when count is 0")
	assert.Equal(t, "0", res.Actual)
	assert.NotEmpty(t, res.Reason)
}

func TestEvaluateThreshold_Duration_NoData_FailsByDefault(t *testing.T) {
	store := metric.NewStore()

	th := config.ThresholdConfig{
		Metric:         "http_req_duration",
		Stat:           "p95",
		Operator:       "<",
		Target:         "200ms",
		TargetDuration: 200 * time.Millisecond,
	}

	res := sla.EvaluateThreshold(th, store)
	assert.False(t, res.Passed, "duration threshold with no data should fail by default")
	assert.Equal(t, "no data", res.Actual)
	assert.Equal(t, "no data", res.Reason)
}

func TestEvaluateThreshold_OnNoData_Strategies(t *testing.T) {
	store := metric.NewStore()

	t.Run("on_no_data: zero for missing duration passes < target", func(t *testing.T) {
		th := config.ThresholdConfig{
			Metric:         "unrecorded_latency",
			Stat:           "p95",
			Operator:       "<",
			Target:         "200ms",
			TargetDuration: 200 * time.Millisecond,
			OnNoData:       "zero",
		}
		res := sla.EvaluateThreshold(th, store)
		assert.True(t, res.Passed)
		assert.Equal(t, "0s", res.Actual)
		assert.Empty(t, res.Reason)
	})

	t.Run("on_no_data: zero for missing rate evaluates 0", func(t *testing.T) {
		th := config.ThresholdConfig{
			Metric:      "unrecorded_rate",
			Stat:        "rate",
			Operator:    "<=",
			Target:      "0",
			TargetFloat: 0,
			OnNoData:    "zero",
		}
		res := sla.EvaluateThreshold(th, store)
		assert.True(t, res.Passed)
		assert.Equal(t, "0", res.Actual)
	})

	t.Run("on_no_data: fail for missing counter fails with no data", func(t *testing.T) {
		th := config.ThresholdConfig{
			Metric:      "unrecorded_errors",
			Stat:        "count",
			Operator:    "<=",
			Target:      "0",
			TargetFloat: 0,
			OnNoData:    "fail",
		}
		res := sla.EvaluateThreshold(th, store)
		assert.False(t, res.Passed)
		assert.Equal(t, "no data", res.Actual)
		assert.Equal(t, "no data", res.Reason)
	})

	t.Run("on_no_data: pass for missing duration passes", func(t *testing.T) {
		th := config.ThresholdConfig{
			Metric:         "optional_duration",
			Stat:           "p95",
			Operator:       "<",
			Target:         "100ms",
			TargetDuration: 100 * time.Millisecond,
			OnNoData:       "pass",
		}
		res := sla.EvaluateThreshold(th, store)
		assert.True(t, res.Passed)
		assert.Equal(t, "no data", res.Actual)
		assert.Empty(t, res.Reason)
	})

	t.Run("on_no_data: ignore for missing metric passes", func(t *testing.T) {
		th := config.ThresholdConfig{
			Metric:         "skipped_metric",
			Stat:           "p99",
			Operator:       "<",
			Target:         "50ms",
			TargetDuration: 50 * time.Millisecond,
			OnNoData:       "ignore",
		}
		res := sla.EvaluateThreshold(th, store)
		assert.True(t, res.Passed)
		assert.Equal(t, "no data", res.Actual)
		assert.Empty(t, res.Reason)
	})

	t.Run("on_no_data: skip alias for missing metric passes", func(t *testing.T) {
		th := config.ThresholdConfig{
			Metric:         "skipped_metric",
			Stat:           "p99",
			Operator:       "<",
			Target:         "50ms",
			TargetDuration: 50 * time.Millisecond,
			OnNoData:       "skip",
		}
		res := sla.EvaluateThreshold(th, store)
		assert.True(t, res.Passed)
		assert.Equal(t, "no data", res.Actual)
		assert.Empty(t, res.Reason)
	})
}

func TestEvaluateThreshold_Duration_EmptySnapshot_HonorsOnNoData(t *testing.T) {
	store := metric.NewStore()
	_ = store.Duration("empty_duration", metric.Tags{}) // registered but 0 observations

	thZero := config.ThresholdConfig{
		Metric:         "empty_duration",
		Stat:           "p95",
		Operator:       "<",
		Target:         "200ms",
		TargetDuration: 200 * time.Millisecond,
		OnNoData:       "zero",
	}
	resZero := sla.EvaluateThreshold(thZero, store)
	assert.True(t, resZero.Passed)
	assert.Equal(t, "0s", resZero.Actual)

	thFail := config.ThresholdConfig{
		Metric:         "empty_duration",
		Stat:           "p95",
		Operator:       "<",
		Target:         "200ms",
		TargetDuration: 200 * time.Millisecond,
		OnNoData:       "fail",
	}
	resFail := sla.EvaluateThreshold(thFail, store)
	assert.False(t, resFail.Passed)
	assert.Equal(t, "no data", resFail.Actual)
	assert.Equal(t, "no data", resFail.Reason)

	thPass := config.ThresholdConfig{
		Metric:         "empty_duration",
		Stat:           "p95",
		Operator:       "<",
		Target:         "200ms",
		TargetDuration: 200 * time.Millisecond,
		OnNoData:       "pass",
	}
	resPass := sla.EvaluateThreshold(thPass, store)
	assert.True(t, resPass.Passed)
	assert.Equal(t, "no data", resPass.Actual)
}

func TestEvaluateThreshold_Rate_NoData_HonorsOnNoData(t *testing.T) {
	store := metric.NewStore()
	_ = store.Rate("empty_rate", metric.Tags{}) // registered but 0 observations

	thZero := config.ThresholdConfig{
		Metric:      "empty_rate",
		Stat:        "rate",
		Operator:    "<=",
		Target:      "0",
		TargetFloat: 0,
		OnNoData:    "zero",
	}
	resZero := sla.EvaluateThreshold(thZero, store)
	assert.True(t, resZero.Passed)
	assert.Equal(t, "0", resZero.Actual)

	thPass := config.ThresholdConfig{
		Metric:      "empty_rate",
		Stat:        "rate",
		Operator:    ">",
		Target:      "0.9",
		TargetFloat: 0.9,
		OnNoData:    "pass",
	}
	resPass := sla.EvaluateThreshold(thPass, store)
	assert.True(t, resPass.Passed)
	assert.Equal(t, "no data", resPass.Actual)
}


