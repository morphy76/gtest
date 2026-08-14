package gtest_test

import (
	"testing"

	"github.com/morphy76/gtest/pkg/gtest"
	"github.com/stretchr/testify/assert"
)

func TestPublicMetricConstants(t *testing.T) {
	assert.Equal(t, "gtest.", gtest.MetricPrefix)
	assert.Equal(t, "gtest.vu.active", gtest.MetricVUActive)
	assert.Equal(t, "gtest.vu.panics", gtest.MetricVUPanics)
	assert.Equal(t, "gtest.vu.iterations_total", gtest.MetricIterationsTotal)
	assert.Equal(t, "gtest.vu.iterations_failed", gtest.MetricIterationsFailed)
	assert.Equal(t, "gtest.vu.iterations_timeout", gtest.MetricIterationsTimeout)
	assert.Equal(t, "gtest.vu.iteration_duration", gtest.MetricIterationDuration)
	assert.Equal(t, "gtest.vu.pretest_errors", gtest.MetricVUPretestErrors)
	assert.Equal(t, "gtest.pacing.dropped_iterations", gtest.MetricPacingDroppedIterations)
	assert.Equal(t, "gtest.checks.passed", gtest.MetricChecksPassed)
	assert.Equal(t, "gtest.checks.failed", gtest.MetricChecksFailed)
}
