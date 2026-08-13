package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/morphy76/gtest/internal/config"
	"github.com/morphy76/gtest/pkg/gtest"
)

// RunConstantVUs executes the constant_vus pacing schedule.
func RunConstantVUs(
	ctx context.Context,
	scenario gtest.Scenario,
	cfg config.ScenarioConfig,
	scenarioName string,
	globalState map[string]any,
	logger gtest.Logger,
	metrics gtest.MetricsCollector,
) {
	totalDuration := cfg.RampUp + cfg.RunPeriod
	runCtx, cancel := context.WithTimeout(ctx, totalDuration)
	defer cancel()

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
		go runVUGoroutine(runCtx, scenario, cfg, scenarioName, vuid, globalState, logger, metrics, &wg)
	}

	wg.Wait()
}

func runVUGoroutine(
	ctx context.Context,
	scenario gtest.Scenario,
	cfg config.ScenarioConfig,
	scenarioName string,
	vuid int64,
	globalState map[string]any,
	logger gtest.Logger,
	metrics gtest.MetricsCollector,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	activeGauge := metrics.Gauge("gtest.vu.active", gtest.Tags{})
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
			metrics.Counter("gtest.vu.pretest_errors", gtest.Tags{}).Inc()
			logger.Error().Err(err).Msg("PreTest hook failed, skipping RunVU")
			return // skips RunVU, deferred AfterTest still runs
		}
	}

	var iteration int64 = 0
	for {
		select {
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
					metrics.Counter("gtest.vu.panics", gtest.Tags{}).Inc()
					metrics.Counter("gtest.vu.iterations_failed", gtest.Tags{}).Inc()
					metrics.Counter("gtest.vu.iterations_total", gtest.Tags{}).Inc()
					logger.Error().Str("panic", fmt.Sprintf("%v", r)).Msg("RunVU panicked")
				}
			}()

			err := scenario.RunVU(sCtx)

			if iterCtx.Err() == context.DeadlineExceeded {
				metrics.Counter("gtest.vu.iterations_timeout", gtest.Tags{}).Inc()
				metrics.Counter("gtest.vu.iterations_failed", gtest.Tags{}).Inc()
				metrics.Counter("gtest.vu.iterations_total", gtest.Tags{}).Inc()
				logger.Error().Err(iterCtx.Err()).Msg("RunVU iteration timed out")
			} else if err != nil {
				metrics.Counter("gtest.vu.iterations_failed", gtest.Tags{}).Inc()
				metrics.Counter("gtest.vu.iterations_total", gtest.Tags{}).Inc()
				logger.Error().Err(err).Msg("RunVU returned error")
			} else {
				metrics.Counter("gtest.vu.iterations_total", gtest.Tags{}).Inc()
			}
		}()

		iteration++
	}
}
