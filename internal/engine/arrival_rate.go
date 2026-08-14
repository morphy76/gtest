package engine

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/morphy76/gtest/internal/config"
	"github.com/morphy76/gtest/internal/log"
	"github.com/morphy76/gtest/internal/metric"
	"golang.org/x/time/rate"
)

// RunArrivalRate executes the arrival_rate pacing schedule using token bucket dispatch.
func RunArrivalRate(
	ctx context.Context,
	scenario Scenario,
	cfg config.ScenarioConfig,
	scenarioName string,
	globalState map[string]any,
	logger log.Logger,
	metrics metric.Collector,
) {
	// Total context includes ramp_down as a grace period for in-flight workers.
	totalDuration := cfg.RampUp + cfg.RunPeriod + cfg.RampDown
	runCtx, cancel := context.WithTimeout(ctx, totalDuration)
	defer cancel()

	// Dispatch phase ends at ramp_up + run_period.
	dispatchDuration := cfg.RampUp + cfg.RunPeriod
	dispatchCtx, dispatchCancel := context.WithTimeout(ctx, dispatchDuration)
	defer dispatchCancel()

	sem := make(chan struct{}, cfg.MaxVUs)
	var wg sync.WaitGroup
	var vuidSeq int64

	dispatchToken := func() {
		select {
		case sem <- struct{}{}:
			wg.Add(1)
			vuid := atomic.AddInt64(&vuidSeq, 1)
			go runArrivalRateWorker(runCtx, scenario, cfg, scenarioName, vuid, globalState, logger, metrics, sem, &wg)
		default:
			metrics.Counter(metric.MetricPacingDroppedIterations, metric.Tags{}).Inc()
		}
	}

	// 1. Ramp-up phase (if configured)
	if cfg.RampUp > 0 {
		// First token during linear ramp-up occurs at the midpoint of ramp_up
		midpoint := cfg.RampUp / 2
		select {
		case <-dispatchCtx.Done():
			wg.Wait()
			return
		case <-time.After(midpoint):
			dispatchToken()
		}

		// Wait remaining half of ramp_up before steady state
		remainingRamp := cfg.RampUp - midpoint
		select {
		case <-dispatchCtx.Done():
			wg.Wait()
			return
		case <-time.After(remainingRamp):
		}
	}

	// 2. Steady-state phase — dispatch ends at ramp_up + run_period
	limiter := rate.NewLimiter(rate.Limit(cfg.TargetTPS), 1)

	for {
		select {
		case <-dispatchCtx.Done():
			wg.Wait()
			return
		default:
		}

		if err := limiter.Wait(dispatchCtx); err != nil {
			wg.Wait()
			return
		}

		dispatchToken()
	}
}

func runArrivalRateWorker(
	ctx context.Context,
	scenario Scenario,
	cfg config.ScenarioConfig,
	scenarioName string,
	vuid int64,
	globalState map[string]any,
	logger log.Logger,
	metrics metric.Collector,
	sem chan struct{},
	wg *sync.WaitGroup,
) {
	defer func() {
		<-sem
		wg.Done()
	}()

	activeGauge := metrics.Gauge(metric.MetricVUActive, metric.Tags{})
	activeGauge.Add(1)
	defer activeGauge.Add(-1)

	// AfterTest is guaranteed to run after PreTest/RunVU exit.
	defer func() {
		if scenario.AfterTest != nil {
			afterCtx := newScenarioContext(ctx, vuid, 0, cfg, scenarioName, globalState, logger, metrics)
			if err := scenario.AfterTest(afterCtx); err != nil {
				logger.Error().Err(err).Msg("AfterTest hook error")
			}
		}
	}()

	// PreTest hook (if present)
	if scenario.PreTest != nil {
		preCtx := newScenarioContext(ctx, vuid, 0, cfg, scenarioName, globalState, logger, metrics)
		if err := scenario.PreTest(preCtx); err != nil {
			metrics.Counter(metric.MetricVUPretestErrors, metric.Tags{}).Inc()
			logger.Error().Err(err).Msg("PreTest hook failed, skipping RunVU")
			return
		}
	}

	iterCtx, cancel := context.WithTimeout(ctx, cfg.VUTimeout)
	defer cancel()

	sCtx := newScenarioContext(iterCtx, vuid, 0, cfg, scenarioName, globalState, logger, metrics)

	func() {
		defer func() {
			if r := recover(); r != nil {
				metrics.Counter(metric.MetricVUPanics, metric.Tags{}).Inc()
				metrics.Counter(metric.MetricIterationsFailed, metric.Tags{}).Inc()
				metrics.Counter(metric.MetricIterationsTotal, metric.Tags{}).Inc()
				logger.Error().Str("panic", fmt.Sprintf("%v", r)).Msg("RunVU panicked")
			}
		}()

		err := scenario.RunVU(sCtx)

		if iterCtx.Err() == context.DeadlineExceeded {
			metrics.Counter(metric.MetricIterationsTimeout, metric.Tags{}).Inc()
			metrics.Counter(metric.MetricIterationsFailed, metric.Tags{}).Inc()
			metrics.Counter(metric.MetricIterationsTotal, metric.Tags{}).Inc()
			logger.Error().Err(iterCtx.Err()).Msg("RunVU iteration timed out")
		} else if err != nil {
			metrics.Counter(metric.MetricIterationsFailed, metric.Tags{}).Inc()
			metrics.Counter(metric.MetricIterationsTotal, metric.Tags{}).Inc()
			logger.Error().Err(err).Msg("RunVU returned error")
		} else {
			metrics.Counter(metric.MetricIterationsTotal, metric.Tags{}).Inc()
		}
	}()
}
