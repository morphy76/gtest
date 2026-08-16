package metric_test

import (
	"testing"

	"github.com/morphy76/vuhive/internal/metric"
	"github.com/stretchr/testify/assert"
)

func TestBuiltInMetricConstants(t *testing.T) {
	assert.Equal(t, "vuhive.", metric.MetricPrefix)
	assert.Equal(t, "vuhive.vu.active", metric.MetricVUActive)
	assert.Equal(t, "vuhive.vu.panics", metric.MetricVUPanics)
	assert.Equal(t, "vuhive.vu.iterations_total", metric.MetricIterationsTotal)
	assert.Equal(t, "vuhive.vu.iterations_failed", metric.MetricIterationsFailed)
	assert.Equal(t, "vuhive.vu.iterations_timeout", metric.MetricIterationsTimeout)
	assert.Equal(t, "vuhive.vu.iteration_duration", metric.MetricIterationDuration)
	assert.Equal(t, "vuhive.vu.pretest_errors", metric.MetricVUPretestErrors)
	assert.Equal(t, "vuhive.pacing.dropped_iterations", metric.MetricPacingDroppedIterations)
	assert.Equal(t, "vuhive.checks.passed", metric.MetricChecksPassed)
	assert.Equal(t, "vuhive.checks.failed", metric.MetricChecksFailed)
}
