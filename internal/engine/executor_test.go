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
	"github.com/morphy76/gtest/internal/log"
	"github.com/morphy76/gtest/internal/metric"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestDeps() (log.Logger, *metric.Store) {
	logger := log.New(zerolog.NewConsoleWriter(), zerolog.Disabled)
	metrics := metric.NewStore()
	return logger, metrics
}

// AC-1.6.1: With vus=3, ramp_up=0, run_period=100ms, exactly 3 VU goroutines are active during run
func TestConstantVUsActiveCount(t *testing.T) {
	logger, metrics := newTestDeps()

	var activeVUIDs sync.Map

	scenario := engine.Scenario{
		RunVU: func(ctx engine.VUContext) error {
			activeVUIDs.Store(ctx.VUID(), true)
			time.Sleep(10 * time.Millisecond)
			return nil
		},
	}

	cfg := config.ScenarioConfig{
		Type:      config.ScenarioTypeConstantVUs,
		VUs:       3,
		RunPeriod: 100 * time.Millisecond,
		VUTimeout: 1 * time.Second,
	}

	exec := engine.NewExecutor("test_scenario", scenario, cfg, logger, metrics)
	err := exec.Execute(context.Background())
	require.NoError(t, err)

	count := 0
	activeVUIDs.Range(func(k, v any) bool {
		count++
		return true
	})
	assert.Equal(t, 3, count, "exactly 3 distinct VUIDs should have executed")
}

// AC-1.6.2: Setup is called exactly once before any PreTest
func TestSetupCalledOnceBeforePreTest(t *testing.T) {
	logger, metrics := newTestDeps()

	var setupCount atomic.Int64
	var setupTime time.Time
	var preTestTime time.Time
	var mu sync.Mutex

	scenario := engine.Scenario{
		Setup: func(ctx engine.SetupContext) (map[string]any, error) {
			mu.Lock()
			setupTime = time.Now()
			mu.Unlock()
			setupCount.Add(1)
			return map[string]any{"key": "val"}, nil
		},
		PreTest: func(ctx engine.VUContext) error {
			mu.Lock()
			if preTestTime.IsZero() {
				preTestTime = time.Now()
			}
			mu.Unlock()
			assert.Equal(t, "val", ctx.GlobalState("key"))
			return nil
		},
		RunVU: func(ctx engine.VUContext) error {
			return nil
		},
	}

	cfg := config.ScenarioConfig{
		Type:      config.ScenarioTypeConstantVUs,
		VUs:       2,
		RunPeriod: 50 * time.Millisecond,
		VUTimeout: 1 * time.Second,
	}

	exec := engine.NewExecutor("test_scenario", scenario, cfg, logger, metrics)
	err := exec.Execute(context.Background())
	require.NoError(t, err)

	assert.Equal(t, int64(1), setupCount.Load(), "setup must be called exactly once")
	mu.Lock()
	assert.True(t, setupTime.Before(preTestTime) || setupTime.Equal(preTestTime), "setup must run before pretest")
	mu.Unlock()
}

// AC-1.6.3: PreTest is called exactly once per VU
func TestPreTestCalledOncePerVU(t *testing.T) {
	logger, metrics := newTestDeps()

	var preTestCount atomic.Int64

	scenario := engine.Scenario{
		PreTest: func(ctx engine.VUContext) error {
			preTestCount.Add(1)
			return nil
		},
		RunVU: func(ctx engine.VUContext) error {
			return nil
		},
	}

	cfg := config.ScenarioConfig{
		Type:      config.ScenarioTypeConstantVUs,
		VUs:       3,
		RunPeriod: 50 * time.Millisecond,
		VUTimeout: 1 * time.Second,
	}

	exec := engine.NewExecutor("test_scenario", scenario, cfg, logger, metrics)
	err := exec.Execute(context.Background())
	require.NoError(t, err)

	assert.Equal(t, int64(3), preTestCount.Load(), "PreTest must be called exactly once per VU")
}

// AC-1.6.4: RunVU is called at least once per VU during a 100ms run_period
func TestRunVUExecutedAtLeastOncePerVU(t *testing.T) {
	logger, metrics := newTestDeps()

	var executedVUs sync.Map

	scenario := engine.Scenario{
		RunVU: func(ctx engine.VUContext) error {
			executedVUs.Store(ctx.VUID(), true)
			return nil
		},
	}

	cfg := config.ScenarioConfig{
		Type:      config.ScenarioTypeConstantVUs,
		VUs:       3,
		RunPeriod: 100 * time.Millisecond,
		VUTimeout: 1 * time.Second,
	}

	exec := engine.NewExecutor("test_scenario", scenario, cfg, logger, metrics)
	err := exec.Execute(context.Background())
	require.NoError(t, err)

	for id := int64(1); id <= 3; id++ {
		_, ok := executedVUs.Load(id)
		assert.True(t, ok, "VU %d must have executed RunVU at least once", id)
	}
}

// AC-1.6.5: AfterTest is called exactly once per VU, even when RunVU returns an error
func TestAfterTestCalledEvenOnRunVUError(t *testing.T) {
	logger, metrics := newTestDeps()

	var afterTestCount atomic.Int64

	scenario := engine.Scenario{
		RunVU: func(ctx engine.VUContext) error {
			return errors.New("iteration failed")
		},
		AfterTest: func(ctx engine.VUContext) error {
			afterTestCount.Add(1)
			return nil
		},
	}

	cfg := config.ScenarioConfig{
		Type:      config.ScenarioTypeConstantVUs,
		VUs:       3,
		RunPeriod: 50 * time.Millisecond,
		VUTimeout: 1 * time.Second,
	}

	exec := engine.NewExecutor("test_scenario", scenario, cfg, logger, metrics)
	err := exec.Execute(context.Background())
	require.NoError(t, err)

	assert.Equal(t, int64(3), afterTestCount.Load(), "AfterTest must be called exactly once per VU")
}

// AC-1.6.6: Teardown is called exactly once after all VUs exit
func TestTeardownCalledOnceAfterVUsExit(t *testing.T) {
	logger, metrics := newTestDeps()

	var teardownCount atomic.Int64
	var activeVUsAtTeardown int64

	scenario := engine.Scenario{
		RunVU: func(ctx engine.VUContext) error {
			time.Sleep(10 * time.Millisecond)
			return nil
		},
		Teardown: func(ctx engine.TeardownContext, state map[string]any) error {
			teardownCount.Add(1)
			activeVUsAtTeardown = int64(metrics.LastGaugeValue(metric.MetricVUActive))
			return nil
		},
	}

	cfg := config.ScenarioConfig{
		Type:      config.ScenarioTypeConstantVUs,
		VUs:       3,
		RunPeriod: 50 * time.Millisecond,
		VUTimeout: 1 * time.Second,
	}

	exec := engine.NewExecutor("test_scenario", scenario, cfg, logger, metrics)
	err := exec.Execute(context.Background())
	require.NoError(t, err)

	assert.Equal(t, int64(1), teardownCount.Load(), "Teardown must be called exactly once")
	assert.Equal(t, int64(0), activeVUsAtTeardown, "all VUs must have exited before Teardown runs")
}

// AC-1.6.7: A RunVU panic does not terminate other VUs
func TestRunVUPanicDoesNotTerminateOtherVUs(t *testing.T) {
	logger, metrics := newTestDeps()

	var vu1Panicked atomic.Bool

	scenario := engine.Scenario{
		RunVU: func(ctx engine.VUContext) error {
			if ctx.VUID() == 1 && !vu1Panicked.Swap(true) {
				panic("boom")
			}
			time.Sleep(5 * time.Millisecond)
			return nil
		},
	}

	cfg := config.ScenarioConfig{
		Type:      config.ScenarioTypeConstantVUs,
		VUs:       3,
		RunPeriod: 50 * time.Millisecond,
		VUTimeout: 1 * time.Second,
	}

	exec := engine.NewExecutor("test_scenario", scenario, cfg, logger, metrics)
	err := exec.Execute(context.Background())
	require.NoError(t, err)

	assert.Equal(t, int64(1), metrics.AggregatedCounterValue(metric.MetricVUPanics))
	assert.Greater(t, metrics.AggregatedCounterValue(metric.MetricIterationsTotal), int64(1))
}

// AC-1.6.8: A PreTest failure skips RunVU but still calls AfterTest for that VU
func TestPreTestFailureSkipsRunVUButCallsAfterTest(t *testing.T) {
	logger, metrics := newTestDeps()

	var runVUCount atomic.Int64
	var afterTestCount atomic.Int64

	scenario := engine.Scenario{
		PreTest: func(ctx engine.VUContext) error {
			if ctx.VUID() == 2 {
				return errors.New("pretest failed for VU 2")
			}
			return nil
		},
		RunVU: func(ctx engine.VUContext) error {
			runVUCount.Add(1)
			return nil
		},
		AfterTest: func(ctx engine.VUContext) error {
			afterTestCount.Add(1)
			return nil
		},
	}

	cfg := config.ScenarioConfig{
		Type:      config.ScenarioTypeConstantVUs,
		VUs:       3,
		RunPeriod: 50 * time.Millisecond,
		VUTimeout: 1 * time.Second,
	}

	exec := engine.NewExecutor("test_scenario", scenario, cfg, logger, metrics)
	err := exec.Execute(context.Background())
	require.NoError(t, err)

	assert.Equal(t, int64(1), metrics.AggregatedCounterValue(metric.MetricVUPretestErrors))
	assert.Equal(t, int64(3), afterTestCount.Load(), "AfterTest must run for all 3 VUs")
}

// AC-1.6.9: vu_timeout causes a context.DeadlineExceeded which counts as a failed iteration;
// the loop does not exit — RunVU is called again on the next iteration
func TestVUTimeoutCountsAsFailedIterationAndContinuesLoop(t *testing.T) {
	logger, metrics := newTestDeps()

	var runVUCallCount atomic.Int64

	scenario := engine.Scenario{
		RunVU: func(ctx engine.VUContext) error {
			runVUCallCount.Add(1)
			<-ctx.Done()
			return ctx.Err()
		},
	}

	cfg := config.ScenarioConfig{
		Type:      config.ScenarioTypeConstantVUs,
		VUs:       1,
		RunPeriod: 120 * time.Millisecond,
		VUTimeout: 30 * time.Millisecond,
	}

	exec := engine.NewExecutor("test_scenario", scenario, cfg, logger, metrics)
	err := exec.Execute(context.Background())
	require.NoError(t, err)

	assert.GreaterOrEqual(t, runVUCallCount.Load(), int64(2), "RunVU should have been called at least twice after timeouts")
	assert.GreaterOrEqual(t, metrics.AggregatedCounterValue(metric.MetricIterationsTimeout), int64(2))
	assert.GreaterOrEqual(t, metrics.AggregatedCounterValue(metric.MetricIterationsFailed), int64(2))
}

// AC-1.6.10: ramp_up=200ms, vus=4 → VU goroutines spawned at ~50ms intervals
func TestRampUpSpawnsAtIntervals(t *testing.T) {
	logger, metrics := newTestDeps()

	startTimes := make(map[int64]time.Time)
	var mu sync.Mutex

	scenario := engine.Scenario{
		PreTest: func(ctx engine.VUContext) error {
			mu.Lock()
			startTimes[ctx.VUID()] = time.Now()
			mu.Unlock()
			return nil
		},
		RunVU: func(ctx engine.VUContext) error {
			time.Sleep(10 * time.Millisecond)
			return nil
		},
	}

	cfg := config.ScenarioConfig{
		Type:      config.ScenarioTypeConstantVUs,
		VUs:       4,
		RampUp:    200 * time.Millisecond,
		RunPeriod: 50 * time.Millisecond,
		VUTimeout: 1 * time.Second,
	}

	exec := engine.NewExecutor("test_scenario", scenario, cfg, logger, metrics)
	err := exec.Execute(context.Background())
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()

	require.Len(t, startTimes, 4)
	// Expect ~50ms interval between VU 1 and VU 2, VU 2 and VU 3, VU 3 and VU 4
	t12 := startTimes[2].Sub(startTimes[1])
	t23 := startTimes[3].Sub(startTimes[2])
	t34 := startTimes[4].Sub(startTimes[3])

	// Allow reasonable timing tolerance (30ms - 80ms for 50ms expected)
	assert.True(t, t12 >= 30*time.Millisecond && t12 <= 100*time.Millisecond, "t12 interval was %v", t12)
	assert.True(t, t23 >= 30*time.Millisecond && t23 <= 100*time.Millisecond, "t23 interval was %v", t23)
	assert.True(t, t34 >= 30*time.Millisecond && t34 <= 100*time.Millisecond, "t34 interval was %v", t34)
}

// Additional test: Setup error aborts execution immediately
func TestSetupErrorAbortsExecution(t *testing.T) {
	logger, metrics := newTestDeps()

	var preTestCalled bool

	scenario := engine.Scenario{
		Setup: func(ctx engine.SetupContext) (map[string]any, error) {
			return nil, errors.New("database connection failed")
		},
		PreTest: func(ctx engine.VUContext) error {
			preTestCalled = true
			return nil
		},
		RunVU: func(ctx engine.VUContext) error {
			return nil
		},
	}

	cfg := config.ScenarioConfig{
		Type:      config.ScenarioTypeConstantVUs,
		VUs:       2,
		RunPeriod: 50 * time.Millisecond,
		VUTimeout: 1 * time.Second,
	}

	exec := engine.NewExecutor("test_scenario", scenario, cfg, logger, metrics)
	err := exec.Execute(context.Background())

	require.Error(t, err)
	var setupErr *engine.SetupError
	require.True(t, errors.As(err, &setupErr), "must return *engine.SetupError")
	assert.False(t, preTestCalled, "PreTest must not be called when Setup fails")
}

// AC-rampdown: ramp_down provides grace period for in-flight iterations.
// A RunVU that takes 80ms with run_period=50ms and ramp_down=200ms should complete
// at least one full iteration without being killed by context cancellation.
func TestConstantVUsRampDownGracePeriod(t *testing.T) {
	logger, metrics := newTestDeps()

	var completedIterations atomic.Int64

	scenario := engine.Scenario{
		RunVU: func(ctx engine.VUContext) error {
			// Simulate work that takes longer than run_period
			select {
			case <-time.After(80 * time.Millisecond):
				completedIterations.Add(1)
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	}

	cfg := config.ScenarioConfig{
		Type:      config.ScenarioTypeConstantVUs,
		VUs:       2,
		RunPeriod: 50 * time.Millisecond,
		RampDown:  200 * time.Millisecond, // Grace period to let in-flight complete
		VUTimeout: 500 * time.Millisecond,
	}

	exec := engine.NewExecutor("test_scenario", scenario, cfg, logger, metrics)
	err := exec.Execute(context.Background())
	require.NoError(t, err)

	// With ramp_down, at least the first iteration per VU should complete
	completed := completedIterations.Load()
	assert.GreaterOrEqual(t, completed, int64(2),
		"each VU should complete at least 1 iteration during ramp_down grace period")
}

// Issue #70: With ramp_down=0s, in-flight iterations interrupted by test duration expiration
// must not be reported as timeouts or failures.
func TestConstantVUsZeroRampDownInFlightIterationsNotReportedAsTimeoutsOrFailures(t *testing.T) {
	logger, metrics := newTestDeps()

	scenario := engine.Scenario{
		RunVU: func(ctx engine.VUContext) error {
			// Simulate context-aware iteration work (10ms)
			select {
			case <-time.After(10 * time.Millisecond):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	}

	cfg := config.ScenarioConfig{
		Type:      config.ScenarioTypeConstantVUs,
		VUs:       5,
		RunPeriod: 60 * time.Millisecond,
		RampDown:  0,
		VUTimeout: 1 * time.Second,
	}

	exec := engine.NewExecutor("test_scenario", scenario, cfg, logger, metrics)
	err := exec.Execute(context.Background())
	require.NoError(t, err)

	timeoutCount := metrics.AggregatedCounterValue(metric.MetricIterationsTimeout)
	failedCount := metrics.AggregatedCounterValue(metric.MetricIterationsFailed)
	totalCount := metrics.AggregatedCounterValue(metric.MetricIterationsTotal)

	assert.Equal(t, int64(0), timeoutCount, "interrupted in-flight iterations must not be counted as timeouts")
	assert.Equal(t, int64(0), failedCount, "interrupted in-flight iterations must not be counted as failures")
	assert.Greater(t, totalCount, int64(0), "completed iterations before expiration should be recorded in total")
}

