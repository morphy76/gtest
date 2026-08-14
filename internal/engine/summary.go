package engine

import (
	"encoding/json"
)

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
