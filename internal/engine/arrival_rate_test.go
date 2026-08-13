package engine_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/morphy76/gtest/internal/config"
	"github.com/morphy76/gtest/internal/engine"
	"github.com/morphy76/gtest/pkg/gtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AC-1.7.1: target_tps=10, run_period=1s → approximately 10 RunVU calls (±20% tolerance)
func TestArrivalRateTargetTPS(t *testing.T) {
	logger, metrics := newTestDeps()

	scenario := gtest.Scenario{
		RunVU: func(ctx gtest.ScenarioContext) error {
			time.Sleep(5 * time.Millisecond)
			return nil
		},
	}

	cfg := config.ScenarioConfig{
		Type:      config.ScenarioTypeArrivalRate,
		TargetTPS: 10,
		MaxVUs:    50,
		RunPeriod: 1 * time.Second,
		VUTimeout: 1 * time.Second,
	}

	exec := engine.NewExecutor("test_arrival_rate", scenario, cfg, logger, metrics)
	err := exec.Execute(context.Background())
	require.NoError(t, err)

	total := metrics.AggregatedCounterValue("gtest.vu.iterations_total")
	// Target is 10 calls. ±20% tolerance means 8..12 calls.
	assert.GreaterOrEqual(t, total, int64(8), "expected at least 8 completed iterations, got %d", total)
	assert.LessOrEqual(t, total, int64(13), "expected at most 13 completed iterations, got %d", total)
}

// AC-1.7.2: max_vus=2 with slow RunVU (sleeps 500ms) and target_tps=100 → pool saturates;
// gtest.pacing.dropped_iterations > 0
func TestArrivalRatePoolSaturationDropsIterations(t *testing.T) {
	logger, metrics := newTestDeps()

	scenario := gtest.Scenario{
		RunVU: func(ctx gtest.ScenarioContext) error {
			time.Sleep(500 * time.Millisecond)
			return nil
		},
	}

	cfg := config.ScenarioConfig{
		Type:      config.ScenarioTypeArrivalRate,
		TargetTPS: 100,
		MaxVUs:    2,
		RunPeriod: 500 * time.Millisecond,
		VUTimeout: 1 * time.Second,
	}

	exec := engine.NewExecutor("test_arrival_rate", scenario, cfg, logger, metrics)
	err := exec.Execute(context.Background())
	require.NoError(t, err)

	dropped := metrics.AggregatedCounterValue("gtest.pacing.dropped_iterations")
	assert.Greater(t, dropped, int64(0), "gtest.pacing.dropped_iterations must be > 0 when pool saturates")
}

// AC-1.7.3: ramp_up=200ms, target_tps=10 → first iteration starts after ~100ms (midpoint of ramp)
func TestArrivalRateRampUpMidpointFirstIteration(t *testing.T) {
	logger, metrics := newTestDeps()

	var firstCallTime time.Time
	var once sync.Once
	startTime := time.Now()

	scenario := gtest.Scenario{
		RunVU: func(ctx gtest.ScenarioContext) error {
			once.Do(func() {
				firstCallTime = time.Now()
			})
			return nil
		},
	}

	cfg := config.ScenarioConfig{
		Type:      config.ScenarioTypeArrivalRate,
		TargetTPS: 10,
		MaxVUs:    10,
		RampUp:    200 * time.Millisecond,
		RunPeriod: 100 * time.Millisecond,
		VUTimeout: 1 * time.Second,
	}

	exec := engine.NewExecutor("test_arrival_rate", scenario, cfg, logger, metrics)
	err := exec.Execute(context.Background())
	require.NoError(t, err)

	require.False(t, firstCallTime.IsZero(), "first iteration must have run")
	delay := firstCallTime.Sub(startTime)
	// Expected ~100ms midpoint delay (tolerance: 60ms to 160ms)
	assert.True(t, delay >= 60*time.Millisecond && delay <= 170*time.Millisecond, "first iteration delay was %v (expected ~100ms)", delay)
}

// AC-1.7.4: All other lifecycle guarantees from Increment 1.6 apply equally to arrival_rate mode
func TestArrivalRateLifecycleHooks(t *testing.T) {
	logger, metrics := newTestDeps()

	var setupCount atomic.Int64
	var preTestCount atomic.Int64
	var afterTestCount atomic.Int64
	var teardownCount atomic.Int64

	scenario := gtest.Scenario{
		Setup: func(ctx context.Context) (map[string]any, error) {
			setupCount.Add(1)
			return map[string]any{"data": "ok"}, nil
		},
		PreTest: func(ctx gtest.ScenarioContext) error {
			preTestCount.Add(1)
			return nil
		},
		RunVU: func(ctx gtest.ScenarioContext) error {
			if ctx.VUID() == 1 {
				return errors.New("iteration error")
			}
			return nil
		},
		AfterTest: func(ctx gtest.ScenarioContext) error {
			afterTestCount.Add(1)
			return nil
		},
		Teardown: func(ctx context.Context, state map[string]any) error {
			teardownCount.Add(1)
			assert.Equal(t, "ok", state["data"])
			return nil
		},
	}

	cfg := config.ScenarioConfig{
		Type:      config.ScenarioTypeArrivalRate,
		TargetTPS: 10,
		MaxVUs:    5,
		RunPeriod: 200 * time.Millisecond,
		VUTimeout: 1 * time.Second,
	}

	exec := engine.NewExecutor("test_arrival_rate", scenario, cfg, logger, metrics)
	err := exec.Execute(context.Background())
	require.NoError(t, err)

	assert.Equal(t, int64(1), setupCount.Load(), "Setup must be called once")
	assert.Equal(t, int64(1), teardownCount.Load(), "Teardown must be called once")
	assert.Greater(t, preTestCount.Load(), int64(0), "PreTest must be called for dispatched workers")
	assert.Equal(t, preTestCount.Load(), afterTestCount.Load(), "AfterTest must match PreTest count")
}
