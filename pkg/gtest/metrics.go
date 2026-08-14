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

// Built-in framework metric names.
const (
	// MetricPrefix is the prefix reserved for all built-in framework telemetry metrics.
	MetricPrefix = metric.MetricPrefix

	// MetricVUActive tracks the number of currently active VU goroutines.
	MetricVUActive = metric.MetricVUActive

	// MetricVUPanics tracks the total number of recovered panics in RunVU.
	MetricVUPanics = metric.MetricVUPanics

	// MetricIterationsTotal tracks the total number of completed VU iterations.
	MetricIterationsTotal = metric.MetricIterationsTotal

	// MetricIterationsFailed tracks the total number of failed VU iterations (errors or panics).
	MetricIterationsFailed = metric.MetricIterationsFailed

	// MetricIterationsTimeout tracks the total number of VU iterations that exceeded vu_timeout.
	MetricIterationsTimeout = metric.MetricIterationsTimeout

	// MetricIterationDuration records latency of VU iterations.
	MetricIterationDuration = metric.MetricIterationDuration

	// MetricVUPretestErrors tracks the number of PreTest hook errors.
	MetricVUPretestErrors = metric.MetricVUPretestErrors

	// MetricPacingDroppedIterations tracks the number of arrival-rate iterations dropped due to pool saturation.
	MetricPacingDroppedIterations = metric.MetricPacingDroppedIterations

	// MetricChecksPassed tracks the count of passed inline checks.
	MetricChecksPassed = metric.MetricChecksPassed

	// MetricChecksFailed tracks the count of failed inline checks.
	MetricChecksFailed = metric.MetricChecksFailed
)
