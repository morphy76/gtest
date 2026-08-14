package metric

// Built-in telemetry metric name constants.
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
