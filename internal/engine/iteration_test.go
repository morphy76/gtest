package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/morphy76/vuhive/internal/log"
	"github.com/morphy76/vuhive/internal/metric"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func newTestIterationDeps() (log.Logger, *metric.Store) {
	logger := log.New(zerolog.NewConsoleWriter(), zerolog.Disabled)
	metrics := metric.NewStore()
	return logger, metrics
}

func TestRecordIterationResult_ScenarioInterruptedWithError(t *testing.T) {
	logger, metrics := newTestIterationDeps()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel scenario context

	iterCtx, iterCancel := context.WithTimeout(ctx, 1*time.Second)
	defer iterCancel()

	recordIterationResult(ctx, iterCtx, errors.New("context canceled"), metrics, logger)

	assert.Equal(t, int64(0), metrics.AggregatedCounterValue(metric.MetricIterationsTimeout))
	assert.Equal(t, int64(0), metrics.AggregatedCounterValue(metric.MetricIterationsFailed))
	assert.Equal(t, int64(0), metrics.AggregatedCounterValue(metric.MetricIterationsTotal))
}

func TestRecordIterationResult_ScenarioCancelledWithNilError(t *testing.T) {
	logger, metrics := newTestIterationDeps()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Scenario context completed/cancelled right as RunVU returned nil

	iterCtx, iterCancel := context.WithTimeout(ctx, 1*time.Second)
	defer iterCancel()

	recordIterationResult(ctx, iterCtx, nil, metrics, logger)

	assert.Equal(t, int64(0), metrics.AggregatedCounterValue(metric.MetricIterationsTimeout))
	assert.Equal(t, int64(0), metrics.AggregatedCounterValue(metric.MetricIterationsFailed))
	assert.Equal(t, int64(1), metrics.AggregatedCounterValue(metric.MetricIterationsTotal))
}

func TestRecordIterationResult_GenuineVUTimeout(t *testing.T) {
	logger, metrics := newTestIterationDeps()

	ctx := context.Background() // Scenario context is active

	iterCtx, iterCancel := context.WithDeadline(ctx, time.Now().Add(-1*time.Millisecond))
	defer iterCancel()

	recordIterationResult(ctx, iterCtx, context.DeadlineExceeded, metrics, logger)

	assert.Equal(t, int64(1), metrics.AggregatedCounterValue(metric.MetricIterationsTimeout))
	assert.Equal(t, int64(1), metrics.AggregatedCounterValue(metric.MetricIterationsFailed))
	assert.Equal(t, int64(1), metrics.AggregatedCounterValue(metric.MetricIterationsTotal))
}

func TestRecordIterationResult_ApplicationError(t *testing.T) {
	logger, metrics := newTestIterationDeps()

	ctx := context.Background()
	iterCtx, iterCancel := context.WithTimeout(ctx, 1*time.Second)
	defer iterCancel()

	recordIterationResult(ctx, iterCtx, errors.New("500 Internal Server Error"), metrics, logger)

	assert.Equal(t, int64(0), metrics.AggregatedCounterValue(metric.MetricIterationsTimeout))
	assert.Equal(t, int64(1), metrics.AggregatedCounterValue(metric.MetricIterationsFailed))
	assert.Equal(t, int64(1), metrics.AggregatedCounterValue(metric.MetricIterationsTotal))
}

func TestRecordIterationResult_Success(t *testing.T) {
	logger, metrics := newTestIterationDeps()

	ctx := context.Background()
	iterCtx, iterCancel := context.WithTimeout(ctx, 1*time.Second)
	defer iterCancel()

	recordIterationResult(ctx, iterCtx, nil, metrics, logger)

	assert.Equal(t, int64(0), metrics.AggregatedCounterValue(metric.MetricIterationsTimeout))
	assert.Equal(t, int64(0), metrics.AggregatedCounterValue(metric.MetricIterationsFailed))
	assert.Equal(t, int64(1), metrics.AggregatedCounterValue(metric.MetricIterationsTotal))
}
