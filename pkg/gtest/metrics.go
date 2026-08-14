package gtest

import "time"

// Tags is an optional set of key-value labels attached to a metric observation.
type Tags map[string]string

// Counter is a monotonically increasing counter.
type Counter interface {
	// Inc increments the counter by 1.
	Inc()

	// Add increments the counter by the specified delta.
	Add(delta int64)
}

// Gauge is an instantaneous value handle.
type Gauge interface {
	// Set sets the gauge to the specified float64 value.
	Set(value float64)

	// Add adjusts the gauge value by adding delta.
	Add(delta float64)
}

// Duration records latency samples into an HDR histogram.
type Duration interface {
	// Observe records a latency duration observation.
	Observe(d time.Duration)
}

// Rate tracks a ratio of numerator to denominator.
type Rate interface {
	// Add increments the numerator and denominator accumulators.
	Add(numerator, denominator int64)
}

// MetricsCollector is available inside ScenarioContext.
// All returned metric handles are safe for concurrent use from multiple VU goroutines.
type MetricsCollector interface {
	// Counter returns a thread-safe handle to a named counter metric with the given tags.
	Counter(name string, tags Tags) Counter

	// Gauge returns a thread-safe handle to a named gauge metric with the given tags.
	Gauge(name string, tags Tags) Gauge

	// Duration returns a thread-safe handle to a named duration histogram metric with the given tags.
	Duration(name string, tags Tags) Duration

	// Rate returns a thread-safe handle to a named rate metric with the given tags.
	Rate(name string, tags Tags) Rate
}

// Built-in framework metric names.
const (
	// MetricPrefix is the prefix reserved for all built-in framework telemetry metrics.
	MetricPrefix = "gtest."

	// MetricVUActive tracks the number of currently active VU goroutines.
	MetricVUActive = "gtest.vu.active"

	// MetricVUPanics tracks the total number of recovered panics in RunVU.
	MetricVUPanics = "gtest.vu.panics"

	// MetricIterationsTotal tracks the total number of completed VU iterations.
	MetricIterationsTotal = "gtest.vu.iterations_total"

	// MetricIterationsFailed tracks the total number of failed VU iterations (errors or panics).
	MetricIterationsFailed = "gtest.vu.iterations_failed"

	// MetricIterationsTimeout tracks the total number of VU iterations that exceeded vu_timeout.
	MetricIterationsTimeout = "gtest.vu.iterations_timeout"

	// MetricIterationDuration records latency of VU iterations.
	MetricIterationDuration = "gtest.vu.iteration_duration"

	// MetricVUPretestErrors tracks the number of PreTest hook errors.
	MetricVUPretestErrors = "gtest.vu.pretest_errors"

	// MetricPacingDroppedIterations tracks the number of arrival-rate iterations dropped due to pool saturation.
	MetricPacingDroppedIterations = "gtest.pacing.dropped_iterations"

	// MetricChecksPassed tracks the count of passed inline checks.
	MetricChecksPassed = "gtest.checks.passed"

	// MetricChecksFailed tracks the count of failed inline checks.
	MetricChecksFailed = "gtest.checks.failed"
)
