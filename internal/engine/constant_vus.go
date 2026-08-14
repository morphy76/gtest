package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/morphy76/gtest/internal/config"
	"github.com/morphy76/gtest/internal/log"
	"github.com/morphy76/gtest/internal/metric"
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

	activeGauge := metrics.Gauge("gtest.vu.active", metric.Tags{})
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
			metrics.Counter("gtest.vu.pretest_errors", metric.Tags{}).Inc()
			logger.Error().Err(err).Msg("PreTest hook failed, skipping RunVU")
			return // skips RunVU, deferred AfterTest still runs
		}
	}

	var iteration int64 = 0
	for {
		// Stop starting new iterations when either:
		// - stopCh fires (run_period ended, ramp_down grace period active), or
		// - ctx cancelled (total duration expired, including ramp_down=0 case).
		select {
		case <-stopCh:
			return
		case <-ctx.Done():
			return
		default:
		}

		iterCtx, cancel := context.WithTimeout(ctx, cfg.VUTimeout)
		sCtx := newScenarioContext(iterCtx, vuid, iteration, cfg, scenarioName, globalState, logger, metrics)

		func() {
			defer cancel()
			defer func() {
				if r := recover(); r != nil {
					metrics.Counter("gtest.vu.panics", metric.Tags{}).Inc()
					metrics.Counter("gtest.vu.iterations_failed", metric.Tags{}).Inc()
					metrics.Counter("gtest.vu.iterations_total", metric.Tags{}).Inc()
					logger.Error().Str("panic", fmt.Sprintf("%v", r)).Msg("RunVU panicked")
				}
			}()

			err := scenario.RunVU(sCtx)

			if iterCtx.Err() == context.DeadlineExceeded {
				metrics.Counter("gtest.vu.iterations_timeout", metric.Tags{}).Inc()
				metrics.Counter("gtest.vu.iterations_failed", metric.Tags{}).Inc()
				metrics.Counter("gtest.vu.iterations_total", metric.Tags{}).Inc()
				logger.Error().Err(iterCtx.Err()).Msg("RunVU iteration timed out")
			} else if err != nil {
				metrics.Counter("gtest.vu.iterations_failed", metric.Tags{}).Inc()
				metrics.Counter("gtest.vu.iterations_total", metric.Tags{}).Inc()
				logger.Error().Err(err).Msg("RunVU returned error")
			} else {
				metrics.Counter("gtest.vu.iterations_total", metric.Tags{}).Inc()
			}
		}()

		iteration++
	}
}



