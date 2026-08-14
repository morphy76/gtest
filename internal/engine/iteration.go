package engine

import (
	"context"

	"github.com/morphy76/gtest/internal/log"
	"github.com/morphy76/gtest/internal/metric"
)

// recordIterationResult handles metrics and logging for a completed or interrupted RunVU iteration.
// It differentiates between:
// 1. Scenario completion / cancellation (parent ctx.Err() != nil):
//    - If err == nil: iteration finished cleanly before/at completion -> increment MetricIterationsTotal.
//    - If err != nil: iteration was interrupted mid-flight by scenario shutdown -> do not count as timeout or failure.
// 2. Normal execution (parent ctx.Err() == nil):
//    - If iterCtx.Err() == context.DeadlineExceeded: genuine per-iteration VUTimeout -> count as timeout, failure, total.
//    - If err != nil: application/scenario error -> count as failure, total.
//    - If err == nil: success -> count as total.
func recordIterationResult(
	ctx context.Context,
	iterCtx context.Context,
	err error,
	metrics metric.Collector,
	logger log.Logger,
) {
	if ctx.Err() != nil {
		// Scenario lifecycle context was cancelled or expired.
		if err == nil {
			metrics.Counter(metric.MetricIterationsTotal, metric.Tags{}).Inc()
		} else if logger != nil {
			logger.Debug().Err(err).Msg("RunVU iteration interrupted by scenario completion")
		}
		return
	}

	if iterCtx.Err() == context.DeadlineExceeded {
		metrics.Counter(metric.MetricIterationsTimeout, metric.Tags{}).Inc()
		metrics.Counter(metric.MetricIterationsFailed, metric.Tags{}).Inc()
		metrics.Counter(metric.MetricIterationsTotal, metric.Tags{}).Inc()
		if logger != nil {
			logger.Error().Err(iterCtx.Err()).Msg("RunVU iteration timed out")
		}
	} else if err != nil {
		metrics.Counter(metric.MetricIterationsFailed, metric.Tags{}).Inc()
		metrics.Counter(metric.MetricIterationsTotal, metric.Tags{}).Inc()
		if logger != nil {
			logger.Error().Err(err).Msg("RunVU returned error")
		}
	} else {
		metrics.Counter(metric.MetricIterationsTotal, metric.Tags{}).Inc()
	}
}
