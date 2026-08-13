package gtest

import "github.com/morphy76/gtest/internal/metric"

// Tags is an optional set of key-value labels attached to a metric observation.
type Tags = metric.Tags

// MetricsCollector is available inside ScenarioContext.
// All returned metric handles are safe for concurrent use from multiple VU goroutines.
type MetricsCollector = metric.Collector

// Counter is a monotonically increasing counter.
type Counter = metric.Counter

// Gauge is an instantaneous value handle.
type Gauge = metric.Gauge

// Duration records latency samples into an HDR histogram.
type Duration = metric.Duration

// Rate tracks a ratio of numerator to denominator.
type Rate = metric.Rate
