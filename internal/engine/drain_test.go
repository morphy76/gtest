package engine_test

import (
	"bytes"
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/morphy76/vuhive/internal/config"
	"github.com/morphy76/vuhive/internal/engine"
	"github.com/morphy76/vuhive/internal/log"
	"github.com/morphy76/vuhive/internal/metric"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AC-Drain-1: In-flight VUs that started during run_period/ramp_down finish cleanly during drain period.
func TestEngine_ConstantVUs_DrainPeriod_AllowsInFlightVUsToFinish(t *testing.T) {
	logger, metrics := newTestDeps()

	var completedIterations atomic.Int64

	scenario := engine.Scenario{
		RunVU: func(ctx engine.VUContext) error {
			// Simulate work that takes 70ms (longer than run_period of 40ms)
			select {
			case <-time.After(70 * time.Millisecond):
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
		RunPeriod: 40 * time.Millisecond,
		RampDown:  10 * time.Millisecond,
		Drain:     200 * time.Millisecond,
		VUTimeout: 500 * time.Millisecond,
	}

	exec := engine.NewExecutor("test_drain_constant", scenario, cfg, logger, metrics)
	err := exec.Execute(context.Background())
	require.NoError(t, err)

	completed := completedIterations.Load()
	assert.GreaterOrEqual(t, completed, int64(2), "each VU should complete its in-flight iteration during drain")
	assert.Equal(t, int64(0), metrics.AggregatedCounterValue(metric.MetricIterationsTimeout))
	assert.Equal(t, int64(0), metrics.AggregatedCounterValue(metric.MetricIterationsFailed))
	assert.GreaterOrEqual(t, metrics.AggregatedCounterValue(metric.MetricIterationsTotal), int64(2))
}

// AC-Drain-2: When all VUs complete before drain timeout, engine exits immediately (no unnecessary wait).
func TestEngine_DrainPeriod_ExitsEarlyWhenAllVUsFinish(t *testing.T) {
	logger, metrics := newTestDeps()

	scenario := engine.Scenario{
		RunVU: func(ctx engine.VUContext) error {
			time.Sleep(10 * time.Millisecond)
			return nil
		},
	}

	cfg := config.ScenarioConfig{
		Type:      config.ScenarioTypeConstantVUs,
		VUs:       2,
		RunPeriod: 50 * time.Millisecond,
		RampDown:  0,
		Drain:     5 * time.Second, // Long drain timeout that should NOT be waited out
		VUTimeout: 1 * time.Second,
	}

	start := time.Now()
	exec := engine.NewExecutor("test_fast_exit", scenario, cfg, logger, metrics)
	err := exec.Execute(context.Background())
	require.NoError(t, err)
	elapsed := time.Since(start)

	assert.Less(t, elapsed, 1*time.Second, "drain phase must exit early when all VUs finish, elapsed was %v", elapsed)
}

type safeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *safeBuffer) Write(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *safeBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

// AC-Drain-3: Hung VUs exceeding drain deadline are cleanly cancelled and diagnostic warning logged.
func TestEngine_DrainPeriod_CancelsHungVUsOnTimeout(t *testing.T) {
	var logBuf safeBuffer
	logger := log.New(&logBuf, zerolog.DebugLevel)
	metrics := metric.NewStore()

	var startedIterations atomic.Int64
	var completedIterations atomic.Int64

	scenario := engine.Scenario{
		RunVU: func(ctx engine.VUContext) error {
			startedIterations.Add(1)
			// Hung worker: simulates operation exceeding drain timeout
			select {
			case <-time.After(10 * time.Second):
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
		RunPeriod: 30 * time.Millisecond,
		RampDown:  0,
		Drain:     60 * time.Millisecond,
		VUTimeout: 10 * time.Second,
	}

	start := time.Now()
	exec := engine.NewExecutor("test_hung_drain", scenario, cfg, logger, metrics)
	err := exec.Execute(context.Background())
	require.NoError(t, err)
	elapsed := time.Since(start)

	assert.Less(t, elapsed, 500*time.Millisecond, "must cancel hung VUs on drain timeout, elapsed was %v", elapsed)
	assert.Equal(t, int64(0), completedIterations.Load(), "hung iterations should not have completed")
	assert.Equal(t, int64(0), metrics.AggregatedCounterValue(metric.MetricIterationsTimeout), "interrupted iterations are discarded, not timeouts")
	assert.Equal(t, int64(0), metrics.AggregatedCounterValue(metric.MetricIterationsFailed), "interrupted iterations are discarded, not failures")
	assert.Equal(t, float64(0), metrics.LastGaugeValue(metric.MetricVUActive))

	logOutput := logBuf.String()
	assert.Contains(t, logOutput, "draining scenario in-flight workers")
	assert.Contains(t, logOutput, "drain phase timed out with active workers remaining")
}

// AC-Drain-4: ArrivalRate pacing enters drain and allows in-flight worker pool iterations to finish.
func TestEngine_ArrivalRate_DrainPeriod_AllowsInFlightWorkersToFinish(t *testing.T) {
	logger, metrics := newTestDeps()

	var completedIterations atomic.Int64

	scenario := engine.Scenario{
		RunVU: func(ctx engine.VUContext) error {
			select {
			case <-time.After(60 * time.Millisecond):
				completedIterations.Add(1)
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	}

	cfg := config.ScenarioConfig{
		Type:      config.ScenarioTypeArrivalRate,
		TargetTPS: 50,
		MaxVUs:    5,
		RunPeriod: 60 * time.Millisecond,
		RampDown:  0,
		Drain:     200 * time.Millisecond,
		VUTimeout: 500 * time.Millisecond,
	}

	exec := engine.NewExecutor("test_drain_arrival", scenario, cfg, logger, metrics)
	err := exec.Execute(context.Background())
	require.NoError(t, err)

	assert.Greater(t, completedIterations.Load(), int64(0))
	assert.Equal(t, int64(0), metrics.AggregatedCounterValue(metric.MetricIterationsTimeout))
	assert.Equal(t, int64(0), metrics.AggregatedCounterValue(metric.MetricIterationsFailed))
	assert.Equal(t, float64(0), metrics.LastGaugeValue(metric.MetricVUActive))
}

// AC-Drain-5: RampingVUs pacing enters drain and allows in-flight workers to finish.
func TestEngine_RampingVUs_DrainPeriod_AllowsInFlightWorkersToFinish(t *testing.T) {
	logger, metrics := newTestDeps()

	var completedIterations atomic.Int64

	scenario := engine.Scenario{
		RunVU: func(ctx engine.VUContext) error {
			select {
			case <-time.After(60 * time.Millisecond):
				completedIterations.Add(1)
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	}

	cfg := config.ScenarioConfig{
		Type:      config.ScenarioTypeRampingVUs,
		VUTimeout: 500 * time.Millisecond,
		Stages: []config.StageConfig{
			{Target: 2, Duration: 60 * time.Millisecond},
		},
		RampDown: 0,
		Drain:    200 * time.Millisecond,
	}

	exec := engine.NewExecutor("test_drain_ramping", scenario, cfg, logger, metrics)
	err := exec.Execute(context.Background())
	require.NoError(t, err)

	assert.Greater(t, completedIterations.Load(), int64(0))
	assert.Equal(t, int64(0), metrics.AggregatedCounterValue(metric.MetricIterationsTimeout))
	assert.Equal(t, int64(0), metrics.AggregatedCounterValue(metric.MetricIterationsFailed))
	assert.Equal(t, float64(0), metrics.LastGaugeValue(metric.MetricVUActive))
}

