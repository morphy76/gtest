package report_test

import (
	"testing"
	"time"

	"github.com/morphy76/gtest/internal/report"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSummaryData_QueryMethods(t *testing.T) {
	summary := report.SummaryData{
		SuiteName: "ecommerce_suite",
		Scenario:  "checkout",
		Version:   "1.0.0",
		Commit:    "abc1234",
		StartedAt: time.Now(),
		EndedAt:   time.Now().Add(5 * time.Second),
		Duration:  5 * time.Second,
		Metrics: []report.MetricSummary{
			{
				Name:  "http_requests_total",
				Type:  "counter",
				Count: 150,
			},
			{
				Name:  "success_rate",
				Type:  "rate",
				Rate:  0.98,
			},
			{
				Name:  "active_users",
				Type:  "gauge",
				Value: 42.0,
			},
			{
				Name: "request_duration",
				Type: "duration",
				P95:  120 * time.Millisecond,
			},
		},
		Checks: []report.CheckSummary{
			{
				Name:    "status_200",
				Passed:  148,
				Failed:  2,
				Total:   150,
				PassPct: 98.67,
			},
		},
		Thresholds: []report.ThresholdSummary{
			{
				Metric:   "request_duration",
				Stat:     "p95",
				Operator: "<",
				Target:   "200ms",
				Actual:   "120ms",
				Passed:   true,
			},
		},
		Passed:  true,
		Aborted: false,
	}

	t.Run("Metric lookup", func(t *testing.T) {
		m := summary.Metric("http_requests_total")
		require.NotNil(t, m)
		assert.Equal(t, "http_requests_total", m.Name)
		assert.Equal(t, int64(150), m.Count)

		assert.Nil(t, summary.Metric("nonexistent"))
	})

	t.Run("Counter value lookup", func(t *testing.T) {
		assert.Equal(t, int64(150), summary.Counter("http_requests_total"))
		assert.Equal(t, int64(0), summary.Counter("nonexistent"))
	})

	t.Run("Rate value lookup", func(t *testing.T) {
		assert.InDelta(t, 0.98, summary.Rate("success_rate"), 0.0001)
		assert.Equal(t, float64(0), summary.Rate("nonexistent"))
	})

	t.Run("Gauge value lookup", func(t *testing.T) {
		assert.InDelta(t, 42.0, summary.Gauge("active_users"), 0.0001)
		assert.Equal(t, float64(0), summary.Gauge("nonexistent"))
	})

	t.Run("Threshold lookup", func(t *testing.T) {
		th := summary.Threshold("request_duration")
		require.NotNil(t, th)
		assert.Equal(t, "request_duration", th.Metric)
		assert.True(t, th.Passed)

		assert.Nil(t, summary.Threshold("nonexistent"))
	})
}
