package gtest

import (
	"encoding/json"
	"time"
)

// CheckSummary represents aggregated results for a named inline check.
type CheckSummary struct {
	Name    string  `json:"name"`
	Passed  int64   `json:"passed"`
	Failed  int64   `json:"failed"`
	Total   int64   `json:"total"`
	PassPct float64 `json:"pass_pct"`
}

// MetricSummary represents a metric entry in the execution summary.
type MetricSummary struct {
	Name   string            `json:"name"`
	Type   string            `json:"type"`
	Tags   map[string]string `json:"tags,omitempty"`
	Count  int64             `json:"count,omitempty"`
	Value  float64           `json:"value,omitempty"`
	Rate   float64           `json:"rate,omitempty"`
	Min    time.Duration     `json:"min,omitempty"`
	Mean   time.Duration     `json:"mean,omitempty"`
	P50    time.Duration     `json:"p50,omitempty"`
	P90    time.Duration     `json:"p90,omitempty"`
	P95    time.Duration     `json:"p95,omitempty"`
	P99    time.Duration     `json:"p99,omitempty"`
	Max    time.Duration     `json:"max,omitempty"`
}

// ThresholdSummary represents the outcome of a single SLA threshold evaluation.
type ThresholdSummary struct {
	Metric   string `json:"metric"`
	Stat     string `json:"stat"`
	Operator string `json:"operator"`
	Target   string `json:"target"`
	Actual   string `json:"actual"`
	Passed   bool   `json:"passed"`
}

// SummaryData contains the complete structured report information post-execution.
type SummaryData struct {
	SuiteName   string             `json:"suite_name"`
	Scenario    string             `json:"scenario"`
	Version     string             `json:"version"`
	Commit      string             `json:"commit"`
	StartedAt   time.Time          `json:"started_at"`
	EndedAt     time.Time          `json:"ended_at"`
	Duration    time.Duration      `json:"duration"`
	Config      any                `json:"config"`
	Metrics     []MetricSummary    `json:"metrics"`
	Checks      []CheckSummary     `json:"checks,omitempty"`
	Thresholds  []ThresholdSummary `json:"thresholds"`
	Passed      bool               `json:"passed"`
	Aborted     bool               `json:"aborted"`
	AbortReason string             `json:"abort_reason,omitempty"`
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
