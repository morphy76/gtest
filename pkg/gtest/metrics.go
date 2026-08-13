package gtest

import "time"

// Tags is an optional set of key-value labels attached to a metric observation.
// Tag keys and values must be non-empty strings. Panics on nil map — use gtest.Tags{} for no tags.
type Tags map[string]string

// MetricsCollector is available inside ScenarioContext.
// All returned metric handles are safe for concurrent use from multiple VU goroutines.
type MetricsCollector interface {
	// Counter returns a monotonically increasing counter identified by name+tags.
	Counter(name string, tags Tags) Counter

	// Gauge returns an instantaneous value handle identified by name+tags.
	Gauge(name string, tags Tags) Gauge

	// Duration returns a latency histogram identified by name+tags.
	// Internally uses per-VU HDR histograms merged at report time.
	Duration(name string, tags Tags) Duration

	// Rate returns a ratio tracker identified by name+tags.
	// Test developers record numerator and denominator together.
	// Threshold stat "rate" computes sum(numerator)/sum(denominator) across all observations.
	Rate(name string, tags Tags) Rate
}

// Counter is a monotonically increasing counter.
type Counter interface {
	Inc()
	Add(delta int64)
}

// Gauge is an instantaneous value handle.
type Gauge interface {
	Set(value float64)
	Add(delta float64)
}

// Duration records latency samples into an HDR histogram.
type Duration interface {
	// Observe records one latency sample.
	Observe(d time.Duration)
}

// Rate tracks a ratio of numerator to denominator.
type Rate interface {
	// Add records numerator events out of denominator total attempts.
	// Both must be >= 0. Denominator == 0 is ignored (no observation recorded).
	Add(numerator, denominator int64)
}
