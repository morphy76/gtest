package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/morphy76/vuhive/internal/config"
	"github.com/morphy76/vuhive/internal/log"
	"github.com/morphy76/vuhive/internal/metric"
)



// RunConstantVUs executes the constant_vus pacing schedule.
func RunConstantVUs(
	ctx context.Context,
	scenario Scenario,
	cfg config.ScenarioConfig,
	scenarioName string,
	globalState map[string]any,
	logger log.Logger,
	metrics metric.Collector,
) {
	// Total context includes ramp_down as a grace period for in-flight iterations.
	totalDuration := cfg.RampUp + cfg.RunPeriod + cfg.RampDown
	runCtx, cancel := context.WithTimeout(ctx, totalDuration)
	defer cancel()

	// stopNewIterations fires at ramp_up + run_period — VUs stop starting new iterations
	// but in-flight iterations continue until ramp_down expires or they complete.
	stopTimer := time.NewTimer(cfg.RampUp + cfg.RunPeriod)
	defer stopTimer.Stop()
	stopCh := make(chan struct{})
	go func() {
		select {
		case <-stopTimer.C:
			close(stopCh)
		case <-runCtx.Done():
		}
	}()

	var wg sync.WaitGroup
	var interval time.Duration
	if cfg.RampUp > 0 && cfg.VUs > 0 {
		interval = cfg.RampUp / time.Duration(cfg.VUs)
	}

	for i := 1; i <= cfg.VUs; i++ {
		if i > 1 && interval > 0 {
			select {
			case <-runCtx.Done():
				return
			case <-time.After(interval):
			}
		}

		wg.Add(1)
		vuid := int64(i)
		go runVUGoroutine(runCtx, stopCh, scenario, cfg, scenarioName, vuid, globalState, logger, metrics, &wg)
	}

	wg.Wait()
}

func runVUGoroutine(
	ctx context.Context,
	stopCh <-chan struct{},
	scenario Scenario,
	cfg config.ScenarioConfig,
	scenarioName string,
	vuid int64,
	globalState map[string]any,
	logger log.Logger,
	metrics metric.Collector,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	activeGauge := metrics.Gauge(metric.MetricVUActive, metric.Tags{})
	activeGauge.Add(1)
	defer activeGauge.Add(-1)

	sCtx := newVUScenarioContext(ctx, vuid, cfg, scenarioName, globalState, logger, metrics)

	// AfterTest is guaranteed to run after PreTest/RunVU exit.
	defer func() {
		if scenario.AfterTest != nil {
			sCtx.prepareIteration(ctx, 0)
			if err := scenario.AfterTest(sCtx); err != nil && logger != nil {
				logger.Error().Err(err).Msg("AfterTest hook error")
			}
		}
	}()

	// PreTest hook (if present)
	if scenario.PreTest != nil {
		sCtx.prepareIteration(ctx, 0)
		if err := scenario.PreTest(sCtx); err != nil {
			metrics.Counter(metric.MetricVUPretestErrors, metric.Tags{}).Inc()
			if logger != nil {
				logger.Error().Err(err).Msg("PreTest hook failed, skipping RunVU")
			}
			return // skips RunVU, deferred AfterTest still runs
		}
	}

	im := newIterationMetrics(metrics)
	hasTimeout := cfg.VUTimeout > 0
	var iteration int64
	for {
		select {
		case <-stopCh:
			return
		case <-ctx.Done():
			return
		default:
		}

		if hasTimeout {
			iterCtx, cancel := context.WithTimeout(ctx, cfg.VUTimeout)
			sCtx.prepareIteration(iterCtx, iteration)
			executeIteration(ctx, iterCtx, sCtx, scenario.RunVU, im, logger)
			cancel()
		} else {
			sCtx.prepareIteration(ctx, iteration)
			executeIteration(ctx, ctx, sCtx, scenario.RunVU, im, logger)
		}

		iteration++
	}
}

func executeIteration(
	ctx context.Context,
	iterCtx context.Context,
	sCtx VUContext,
	runVU func(VUContext) error,
	im iterationMetrics,
	logger log.Logger,
) {
	defer func() {
		if r := recover(); r != nil {
			if im.panics != nil {
				im.panics.Inc()
			}
			if im.failed != nil {
				im.failed.Inc()
			}
			if im.total != nil {
				im.total.Inc()
			}
			if logger != nil {
				logger.Error().Str("panic", fmt.Sprintf("%v", r)).Msg("RunVU panicked")
			}
		}
	}()

	err := runVU(sCtx)
	recordIterationResultFast(ctx, iterCtx, err, im, logger)
}



