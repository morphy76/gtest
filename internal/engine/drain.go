package engine

import (
	"context"
	"sync"
	"time"

	"github.com/morphy76/vuhive/internal/log"
	"github.com/morphy76/vuhive/internal/metric"
)

func getActiveVUs(metrics metric.Collector) int64 {
	if metrics == nil {
		return 0
	}
	if agg, ok := metrics.(interface{ LastGaugeValue(string) float64 }); ok {
		return int64(agg.LastGaugeValue(metric.MetricVUActive))
	}
	return 0
}

// drainWorkers coordinates the drain execution phase:
// 1. Logs structured lifecycle event: "draining scenario in-flight workers".
// 2. Fast / Early Exit: Returns immediately if all active workers finish before drainTimeout.
// 3. Safety Timeout: If workers exceed drainTimeout, cancels cancelCtx to terminate them,
//    waits for workers to exit, and logs a diagnostic warning.
func drainWorkers(
	runCtx context.Context,
	cancelCtx context.CancelFunc,
	wg *sync.WaitGroup,
	drainTimeout time.Duration,
	logger log.Logger,
	metrics metric.Collector,
) {
	active := getActiveVUs(metrics)

	if logger != nil {
		logger.Info().
			Str("phase", "drain").
			Int64("active_vus", active).
			Dur("drain_timeout", drainTimeout).
			Msg("draining scenario in-flight workers")
	}

	waitCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitCh)
	}()

	if drainTimeout > 0 {
		drainTimer := time.NewTimer(drainTimeout)
		defer drainTimer.Stop()

		select {
		case <-waitCh:
			// Fast / early exit: all active workers finished before drain timeout
			return
		case <-drainTimer.C:
			activeBeforeCancel := getActiveVUs(metrics)
			if cancelCtx != nil {
				cancelCtx()
			}
			<-waitCh
			if logger != nil {
				logger.Warn().
					Str("phase", "drain").
					Int64("active_vus", activeBeforeCancel).
					Dur("drain_timeout", drainTimeout).
					Msg("drain phase timed out with active workers remaining")
			}
		case <-runCtx.Done():
			activeBeforeCancel := getActiveVUs(metrics)
			if cancelCtx != nil {
				cancelCtx()
			}
			<-waitCh
			if logger != nil {
				logger.Warn().
					Str("phase", "drain").
					Int64("active_vus", activeBeforeCancel).
					Dur("drain_timeout", drainTimeout).
					Msg("drain phase timed out with active workers remaining")
			}
		}
	} else {
		select {
		case <-waitCh:
			return
		default:
			activeBeforeCancel := getActiveVUs(metrics)
			if cancelCtx != nil {
				cancelCtx()
			}
			<-waitCh
			if logger != nil && activeBeforeCancel > 0 {
				logger.Warn().
					Str("phase", "drain").
					Int64("active_vus", activeBeforeCancel).
					Dur("drain_timeout", drainTimeout).
					Msg("drain phase timed out with active workers remaining")
			}
		}
	}
}


