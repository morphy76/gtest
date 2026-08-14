package gtest

import (
	"encoding/json"
	"time"
)

// CheckSummary represents aggregated results for a named inline check.
type CheckSummary struct {
	// Name is the unique identifier of the inline check.
	Name string `json:"name"`

	// Passed is the number of times this check evaluated to true.
	Passed int64 `json:"passed"`

	// Failed is the number of times this check evaluated to false.
	Failed int64 `json:"failed"`

	// Total is the total number of evaluations for this check.
	Total int64 `json:"total"`

	// PassPct is the percentage of executions that passed (0.0 to 100.0).
	PassPct float64 `json:"pass_pct"`
}

// MetricSummary represents a metric entry in the execution summary.
type MetricSummary struct {
	// Name is the identifier of the metric.
	Name string `json:"name"`

	// Type is the metric category ("counter", "gauge", "duration", "rate").
	Type string `json:"type"`

	// Tags contains dimensional labels attached to the metric.
	Tags map[string]string `json:"tags,omitempty"`

	// Count is the total observation count for counters and histograms.
	Count int64 `json:"count,omitempty"`

	// Value is the most recent instantaneous value for gauges.
	Value float64 `json:"value,omitempty"`

	// Rate is the computed ratio (numerator / denominator) for rates.
	Rate float64 `json:"rate,omitempty"`

	// Min is the minimum observed latency for duration metrics.
	Min time.Duration `json:"min,omitempty"`

	// Mean is the arithmetic mean latency for duration metrics.
	Mean time.Duration `json:"mean,omitempty"`

	// P50 is the 50th percentile (median) latency for duration metrics.
	P50 time.Duration `json:"p50,omitempty"`

	// P90 is the 90th percentile latency for duration metrics.
	P90 time.Duration `json:"p90,omitempty"`

	// P95 is the 95th percentile latency for duration metrics.
	P95 time.Duration `json:"p95,omitempty"`

	// P99 is the 99th percentile latency for duration metrics.
	P99 time.Duration `json:"p99,omitempty"`

	// Max is the maximum observed latency for duration metrics.
	Max time.Duration `json:"max,omitempty"`
}

// ThresholdSummary represents the outcome of a single SLA threshold evaluation.
type ThresholdSummary struct {
	// Metric is the name of the target metric.
	Metric string `json:"metric"`

	// Stat is the statistic evaluated ("p50", "p90", "p95", "p99", "mean", "max", "count", "rate", "value").
	Stat string `json:"stat"`

	// Operator is the comparison operator ("<", "<=", ">", ">=").
	Operator string `json:"operator"`

	// Target is the threshold target value expression string.
	Target string `json:"target"`

	// Actual is the observed actual value formatted as a string.
	Actual string `json:"actual"`

	// Passed indicates whether the observed value satisfied the SLA condition.
	Passed bool `json:"passed"`
}

// SummaryData contains the complete structured report information post-execution.
type SummaryData struct {
	// SuiteName is the display name of the executing test suite.
	SuiteName string `json:"suite_name"`

	// Scenario is the name of the executed scenario.
	Scenario string `json:"scenario"`

	// Version is the gtest framework version.
	Version string `json:"version"`

	// Commit is the git commit SHA at build time.
	Commit string `json:"commit"`

	// StartedAt is the UTC timestamp when the scenario began execution.
	StartedAt time.Time `json:"started_at"`

	// EndedAt is the UTC timestamp when the scenario finished execution.
	EndedAt time.Time `json:"ended_at"`

	// Duration is the total execution wall-clock time.
	Duration time.Duration `json:"duration"`

	// Config is the parsed scenario configuration object.
	Config any `json:"config"`

	// Metrics is the list of summarized telemetry metrics.
	Metrics []MetricSummary `json:"metrics"`

	// Checks is the list of inline check assertion summaries.
	Checks []CheckSummary `json:"checks,omitempty"`

	// Thresholds is the list of evaluated SLA threshold results.
	Thresholds []ThresholdSummary `json:"thresholds"`

	// Passed indicates whether the scenario and all SLA thresholds passed.
	Passed bool `json:"passed"`

	// Aborted indicates whether execution was aborted early.
	Aborted bool `json:"aborted"`

	// AbortReason describes why early abortion occurred.
	AbortReason string `json:"abort_reason,omitempty"`
}

// JSON marshals SummaryData to pretty-printed JSON bytes.
func (s SummaryData) JSON() ([]byte, error) {
	return json.MarshalIndent(s, "", "  ")
}

// Metric returns the MetricSummary for the given metric name, or nil if not found.
func (s SummaryData) Metric(name string) *MetricSummary {
	for i := range s.Metrics {
		if s.Metrics[i].Name == name {
			return &s.Metrics[i]
		}
	}
	return nil
}

// Counter returns the count value for the named counter metric, or 0 if not found.
func (s SummaryData) Counter(name string) int64 {
	m := s.Metric(name)
	if m == nil {
		return 0
	}
	return m.Count
}

// Rate returns the rate value for the named rate metric, or 0 if not found.
func (s SummaryData) Rate(name string) float64 {
	m := s.Metric(name)
	if m == nil {
		return 0
	}
	return m.Rate
}

// Gauge returns the value for the named gauge metric, or 0 if not found.
func (s SummaryData) Gauge(name string) float64 {
	m := s.Metric(name)
	if m == nil {
		return 0
	}
	return m.Value
}

// Threshold returns the ThresholdSummary for the given metric name, or nil if not found.
func (s SummaryData) Threshold(metric string) *ThresholdSummary {
	for i := range s.Thresholds {
		if s.Thresholds[i].Metric == metric {
			return &s.Thresholds[i]
		}
	}
	return nil
}
