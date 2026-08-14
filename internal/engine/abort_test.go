package engine_test

import (
	"context"
	"errors"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/morphy76/gtest/internal/config"
	"github.com/morphy76/gtest/internal/engine"
	"github.com/morphy76/gtest/internal/log"
	"github.com/morphy76/gtest/internal/metric"
)

// AC-1.16.1 & AC-1.16.3: Threshold configured with abort_on_fail=true evaluates periodically and cancels context immediately on breach
func TestAbortOnFail_TriggersCancellation(t *testing.T) {
	store := metric.NewStore()
	logger := log.New(io.Discard, zerolog.Disabled)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	thresholds := []config.ThresholdConfig{
		{
			Metric:         metric.MetricIterationsFailed,
			Stat:           "count",
			Operator:       "==",
			Target:         "0",
			TargetFloat:    0,
			AbortOnFail:    true,
			DelayAbortEval: 0,
		},
	}

	abortedCh, breachReason := engine.MonitorAbortThresholds(ctx, cancel, time.Now(), thresholds, store, logger)

	// Simulate failure breach
	store.Counter(metric.MetricIterationsFailed, metric.Tags{}).Inc()

	select {
	case <-abortedCh:
		// Success: context was cancelled by abort monitor
		assert.Equal(t, context.Canceled, ctx.Err())
		assert.NotEmpty(t, breachReason(), "breach reason must be populated")
	case <-time.After(1 * time.Second):
		t.Fatal("expected abort monitor to trigger cancellation within timeout")
	}
}

// AC-1.16.2: Breach before delay_abort_eval is ignored (warm-up grace period)
func TestAbortOnFail_RespectsWarmupGracePeriod(t *testing.T) {
	store := metric.NewStore()
	logger := log.New(io.Discard, zerolog.Disabled)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	thresholds := []config.ThresholdConfig{
		{
			Metric:         metric.MetricIterationsFailed,
			Stat:           "count",
			Operator:       "==",
			Target:         "0",
			TargetFloat:    0,
			AbortOnFail:    true,
			DelayAbortEval: 1 * time.Second, // 1s warm-up grace period
		},
	}

	abortedCh, _ := engine.MonitorAbortThresholds(ctx, cancel, time.Now(), thresholds, store, logger)

	// Simulate immediate failure breach during warm-up period
	store.Counter(metric.MetricIterationsFailed, metric.Tags{}).Inc()

	select {
	case <-abortedCh:
		t.Fatal("abort monitor should NOT trigger during warm-up grace period")
	case <-time.After(200 * time.Millisecond):
		// Success: context was not cancelled during warm-up
		assert.NoError(t, ctx.Err())
	}
}

// Issue #70: When a scenario is aborted early by abort_on_fail, active in-flight VUs
// interrupted by context cancellation must not be recorded as timeouts.
func TestAbortOnFail_InFlightVUsNotCountedAsTimeouts(t *testing.T) {
	store := metric.NewStore()
	logger := log.New(io.Discard, zerolog.Disabled)

	var vu1Failed atomic.Bool

	scenario := engine.Scenario{
		RunVU: func(ctx engine.ScenarioContext) error {
			if ctx.VUID() == 1 {
				if vu1Failed.CompareAndSwap(false, true) {
					// Trigger the failure on VU 1 once
					return errors.New("deliberate trigger error")
				}
				<-ctx.Done()
				return ctx.Err()
			}
			// Other VUs are waiting
			select {
			case <-time.After(500 * time.Millisecond):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	}

	cfg := config.ScenarioConfig{
		Type:      config.ScenarioTypeConstantVUs,
		VUs:       5,
		RunPeriod: 1 * time.Second,
		VUTimeout: 2 * time.Second,
		Thresholds: []config.ThresholdConfig{
			{
				Metric:      metric.MetricIterationsFailed,
				Stat:        "count",
				Operator:    "==",
				Target:      "0",
				TargetFloat: 0,
				AbortOnFail: true,
			},
		},
	}

	exec := engine.NewExecutor("test_abort_scenario", scenario, cfg, logger, store)
	err := exec.Execute(context.Background())
	assert.NoError(t, err)
	assert.True(t, exec.Aborted, "executor should be marked as aborted")

	// Only VU 1's actual failure should be counted as failed
	assert.Equal(t, int64(0), store.AggregatedCounterValue(metric.MetricIterationsTimeout), "no timeouts should be recorded on abort")
	assert.Equal(t, int64(1), store.AggregatedCounterValue(metric.MetricIterationsFailed), "only the deliberate failure should be counted")
}

