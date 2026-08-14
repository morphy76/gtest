package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/morphy76/gtest/internal/config"
	"github.com/morphy76/gtest/internal/log"
	"github.com/morphy76/gtest/internal/metric"
	"github.com/morphy76/gtest/internal/sla"
)

// MonitorAbortThresholds starts a background goroutine that periodically checks
// thresholds configured with abort_on_fail=true. If a breach occurs after delay_abort_eval,
// cancel() is invoked and the aborted channel is closed.
func MonitorAbortThresholds(
	ctx context.Context,
	cancel context.CancelFunc,
	startTime time.Time,
	thresholds []config.ThresholdConfig,
	store *metric.Store,
	logger log.Logger,
) (abortedCh <-chan struct{}, getReason func() string) {
	ch := make(chan struct{})
	var (
		reasonMu sync.RWMutex
		reason   string
	)

	// Filter thresholds that have AbortOnFail = true
	var abortThresholds []config.ThresholdConfig
	for _, th := range thresholds {
		if th.AbortOnFail {
			abortThresholds = append(abortThresholds, th)
		}
	}

	if len(abortThresholds) == 0 {
		return ch, func() string { return "" }
	}

	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				elapsed := time.Since(startTime)
				for _, th := range abortThresholds {
					if elapsed < th.DelayAbortEval {
						continue // Ignore breaches during warm-up grace period
					}

					res := sla.EvaluateThreshold(th, store)
					if !res.Passed {
						reasonStr := fmt.Sprintf("threshold breach on metric %q (%s %s %s, actual: %s)",
							th.Metric, th.Stat, th.Operator, th.Target, res.Actual)

						reasonMu.Lock()
						reason = reasonStr
						reasonMu.Unlock()

						if logger != nil {
							logger.Error().Str("metric", th.Metric).Str("actual", res.Actual).Msg("abort_on_fail triggered: terminating scenario execution early")
						}

						cancel()
						close(ch)
						return
					}
				}
			}
		}
	}()

	return ch, func() string {
		reasonMu.RLock()
		defer reasonMu.RUnlock()
		return reason
	}
}
