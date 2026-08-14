package engine_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/morphy76/gtest/internal/config"
	"github.com/morphy76/gtest/internal/engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)


// AC-1.12.6: ctx.Sleep() halts for generated duration and aborts immediately on ctx.Done()
func TestScenarioContextSleep(t *testing.T) {
	logger, metrics := newTestDeps()

	t.Run("explicit duration pauses for expected time", func(t *testing.T) {
		var durations []time.Duration
		var mu sync.Mutex

		scenario := engine.Scenario{
			RunVU: func(ctx engine.ScenarioContext) error {
				start := time.Now()
				err := ctx.Sleep(50 * time.Millisecond)
				if err == nil {
					mu.Lock()
					durations = append(durations, time.Since(start))
					mu.Unlock()
				}
				return err
			},
		}

		cfg := config.ScenarioConfig{
			Type:      config.ScenarioTypeConstantVUs,
			VUs:       1,
			RunPeriod: 100 * time.Millisecond,
			VUTimeout: 1 * time.Second,
		}

		exec := engine.NewExecutor("sleep_explicit", scenario, cfg, logger, metrics)
		err := exec.Execute(context.Background())
		require.NoError(t, err)

		mu.Lock()
		defer mu.Unlock()
		require.NotEmpty(t, durations)
		for _, d := range durations {
			assert.GreaterOrEqual(t, d, 45*time.Millisecond)
		}
	})


	t.Run("aborts immediately when context is cancelled", func(t *testing.T) {
		var sleepErr error
		var duration time.Duration

		ctx, cancel := context.WithCancel(context.Background())

		scenario := engine.Scenario{
			RunVU: func(sc engine.ScenarioContext) error {
				go func() {
					time.Sleep(20 * time.Millisecond)
					cancel()
				}()
				start := time.Now()
				sleepErr = sc.Sleep(1 * time.Second)
				duration = time.Since(start)
				return sleepErr
			},
		}

		cfg := config.ScenarioConfig{
			Type:      config.ScenarioTypeConstantVUs,
			VUs:       1,
			RunPeriod: 2 * time.Second,
			VUTimeout: 5 * time.Second,
		}

		exec := engine.NewExecutor("sleep_cancel", scenario, cfg, logger, metrics)
		_ = exec.Execute(ctx)

		require.Error(t, sleepErr)
		assert.Equal(t, context.Canceled, sleepErr)
		assert.Less(t, duration, 200*time.Millisecond, "sleep should have aborted quickly on context cancellation")
	})

	t.Run("uses configured interaction_delay when called with no arguments", func(t *testing.T) {
		var durations []time.Duration
		var mu sync.Mutex

		scenario := engine.Scenario{
			RunVU: func(ctx engine.ScenarioContext) error {
				start := time.Now()
				err := ctx.Sleep()
				if err == nil {
					mu.Lock()
					durations = append(durations, time.Since(start))
					mu.Unlock()
				}
				return err
			},
		}

		cfg := config.ScenarioConfig{
			Type:      config.ScenarioTypeConstantVUs,
			VUs:       1,
			RunPeriod: 150 * time.Millisecond,
			VUTimeout: 1 * time.Second,
			InteractionDelay: &config.InteractionDelayConfig{
				Type:     "fixed",
				Duration: 60 * time.Millisecond,
			},
		}

		exec := engine.NewExecutor("sleep_configured", scenario, cfg, logger, metrics)
		err := exec.Execute(context.Background())
		require.NoError(t, err)

		mu.Lock()
		defer mu.Unlock()
		require.NotEmpty(t, durations, "must have completed at least one sleep")
		for _, d := range durations {
			assert.GreaterOrEqual(t, d, 55*time.Millisecond)
		}
	})


	t.Run("returns nil immediately when no delay is configured and no arg passed", func(t *testing.T) {
		var sleptDuration time.Duration

		scenario := engine.Scenario{
			RunVU: func(ctx engine.ScenarioContext) error {
				start := time.Now()
				err := ctx.Sleep()
				sleptDuration = time.Since(start)
				return err
			},
		}

		cfg := config.ScenarioConfig{
			Type:      config.ScenarioTypeConstantVUs,
			VUs:       1,
			RunPeriod: 50 * time.Millisecond,
			VUTimeout: 1 * time.Second,
		}

		exec := engine.NewExecutor("sleep_none", scenario, cfg, logger, metrics)
		err := exec.Execute(context.Background())
		require.NoError(t, err)

		assert.Less(t, sleptDuration, 10*time.Millisecond)
	})

	t.Run("uses configured think_time when called with no arguments", func(t *testing.T) {
		var durations []time.Duration
		var mu sync.Mutex

		scenario := engine.Scenario{
			RunVU: func(ctx engine.ScenarioContext) error {
				start := time.Now()
				err := ctx.Sleep()
				if err == nil {
					mu.Lock()
					durations = append(durations, time.Since(start))
					mu.Unlock()
				}
				return err
			},
		}

		cfg := config.ScenarioConfig{
			Type:      config.ScenarioTypeConstantVUs,
			VUs:       1,
			RunPeriod: 150 * time.Millisecond,
			VUTimeout: 1 * time.Second,
			ThinkTime: &config.ThinkTimeConfig{
				Type:     "fixed",
				Duration: 60 * time.Millisecond,
			},
		}

		exec := engine.NewExecutor("sleep_think_time_configured", scenario, cfg, logger, metrics)
		err := exec.Execute(context.Background())
		require.NoError(t, err)

		mu.Lock()
		defer mu.Unlock()
		require.NotEmpty(t, durations, "must have completed at least one sleep")
		for _, d := range durations {
			assert.GreaterOrEqual(t, d, 55*time.Millisecond)
		}
	})

	t.Run("zero or negative explicit duration returns nil immediately", func(t *testing.T) {
		var sleptDuration time.Duration

		scenario := engine.Scenario{
			RunVU: func(ctx engine.ScenarioContext) error {
				start := time.Now()
				err1 := ctx.Sleep(0)
				err2 := ctx.Sleep(-50 * time.Millisecond)
				sleptDuration = time.Since(start)
				if err1 != nil {
					return err1
				}
				return err2
			},
		}

		cfg := config.ScenarioConfig{
			Type:      config.ScenarioTypeConstantVUs,
			VUs:       1,
			RunPeriod: 50 * time.Millisecond,
			VUTimeout: 1 * time.Second,
		}

		exec := engine.NewExecutor("sleep_zero_negative", scenario, cfg, logger, metrics)
		err := exec.Execute(context.Background())
		require.NoError(t, err)

		assert.Less(t, sleptDuration, 10*time.Millisecond)
	})

	t.Run("pre-cancelled context returns ctx.Err() immediately", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		sCtx := engine.NewScenarioContext(ctx, 1, 0, config.ScenarioConfig{}, "test", nil, logger, metrics)
		err := sCtx.Sleep(1 * time.Second)

		require.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})
}



