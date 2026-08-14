package engine_test

import (
	"context"
	"io"
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
