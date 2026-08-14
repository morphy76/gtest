package metric_test

import (
	"testing"

	"github.com/morphy76/gtest/internal/metric"
	"github.com/stretchr/testify/assert"
)

func TestBuiltInMetricConstants(t *testing.T) {
	assert.Equal(t, "gtest.", metric.MetricPrefix)
	assert.Equal(t, "gtest.vu.active", metric.MetricVUActive)
	assert.Equal(t, "gtest.vu.panics", metric.MetricVUPanics)
	assert.Equal(t, "gtest.vu.iterations_total", metric.MetricIterationsTotal)
	assert.Equal(t, "gtest.vu.iterations_failed", metric.MetricIterationsFailed)
	assert.Equal(t, "gtest.vu.iterations_timeout", metric.MetricIterationsTimeout)
	assert.Equal(t, "gtest.vu.iteration_duration", metric.MetricIterationDuration)
	assert.Equal(t, "gtest.vu.pretest_errors", metric.MetricVUPretestErrors)
	assert.Equal(t, "gtest.pacing.dropped_iterations", metric.MetricPacingDroppedIterations)
	assert.Equal(t, "gtest.checks.passed", metric.MetricChecksPassed)
	assert.Equal(t, "gtest.checks.failed", metric.MetricChecksFailed)
}
