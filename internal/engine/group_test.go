package engine_test

import (
	"context"
	"errors"
	"io"
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

func TestScenarioContext_Group_NilFn(t *testing.T) {
	metrics := metric.NewStore()
	logger := log.New(io.Discard, zerolog.Disabled)
	ctx := engine.NewScenarioContext(context.Background(), 1, 0, config.ScenarioConfig{}, "test_sc", nil, logger, metrics)

	err := ctx.Group("noop", nil)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), metrics.MergedHistogramSnapshot("vuhive.group.noop.duration").Count)
}

func TestScenarioContext_Group_Single(t *testing.T) {
	metrics := metric.NewStore()
	logger := log.New(io.Discard, zerolog.Disabled)
	ctx := engine.NewScenarioContext(context.Background(), 1, 0, config.ScenarioConfig{}, "test_sc", nil, logger, metrics)

	var executed bool
	err := ctx.Group("01_Login", func(grpCtx engine.VUContext) error {
		executed = true
		time.Sleep(10 * time.Millisecond)
		return nil
	})

	require.NoError(t, err)
	assert.True(t, executed)

	snap := metrics.MergedHistogramSnapshot("vuhive.group.01_Login.duration")
	assert.Equal(t, int64(1), snap.Count)
	assert.GreaterOrEqual(t, snap.Min, 9*time.Millisecond)
}

func TestScenarioContext_Group_Nested(t *testing.T) {
	metrics := metric.NewStore()
	logger := log.New(io.Discard, zerolog.Disabled)
	ctx := engine.NewScenarioContext(context.Background(), 1, 0, config.ScenarioConfig{}, "test_sc", nil, logger, metrics)

	err := ctx.Group("03_Checkout", func(c1 engine.VUContext) error {
		time.Sleep(5 * time.Millisecond)

		err := c1.Group("Add_To_Cart", func(c2 engine.VUContext) error {
			time.Sleep(10 * time.Millisecond)
			return nil
		})
		if err != nil {
			return err
		}

		err = c1.Group("Submit_Payment", func(c2 engine.VUContext) error {
			time.Sleep(15 * time.Millisecond)
			return nil
		})
		return err
	})

	require.NoError(t, err)

	parentSnap := metrics.MergedHistogramSnapshot("vuhive.group.03_Checkout.duration")
	assert.Equal(t, int64(1), parentSnap.Count)
	assert.GreaterOrEqual(t, parentSnap.Min, 25*time.Millisecond)

	cartSnap := metrics.MergedHistogramSnapshot("vuhive.group.03_Checkout::Add_To_Cart.duration")
	assert.Equal(t, int64(1), cartSnap.Count)
	assert.GreaterOrEqual(t, cartSnap.Min, 9*time.Millisecond)

	paySnap := metrics.MergedHistogramSnapshot("vuhive.group.03_Checkout::Submit_Payment.duration")
	assert.Equal(t, int64(1), paySnap.Count)
	assert.GreaterOrEqual(t, paySnap.Min, 14*time.Millisecond)
}

func TestScenarioContext_Group_ErrorPropagation(t *testing.T) {
	metrics := metric.NewStore()
	logger := log.New(io.Discard, zerolog.Disabled)
	ctx := engine.NewScenarioContext(context.Background(), 1, 0, config.ScenarioConfig{}, "test_sc", nil, logger, metrics)

	expectedErr := errors.New("auth failure")
	err := ctx.Group("auth_step", func(grpCtx engine.VUContext) error {
		time.Sleep(10 * time.Millisecond)
		return expectedErr
	})

	assert.ErrorIs(t, err, expectedErr)

	// Even on error, duration must be recorded
	snap := metrics.MergedHistogramSnapshot("vuhive.group.auth_step.duration")
	assert.Equal(t, int64(1), snap.Count)
	assert.GreaterOrEqual(t, snap.Min, 9*time.Millisecond)
}

func TestScenarioContext_Group_PanicRecordsDuration(t *testing.T) {
	metrics := metric.NewStore()
	logger := log.New(io.Discard, zerolog.Disabled)
	ctx := engine.NewScenarioContext(context.Background(), 1, 0, config.ScenarioConfig{}, "test_sc", nil, logger, metrics)

	assert.Panics(t, func() {
		_ = ctx.Group("panic_step", func(grpCtx engine.VUContext) error {
			time.Sleep(10 * time.Millisecond)
			panic("unexpected nil pointer")
		})
	})

	// Panic should still have recorded duration before bubbling up
	snap := metrics.MergedHistogramSnapshot("vuhive.group.panic_step.duration")
	assert.Equal(t, int64(1), snap.Count)
	assert.GreaterOrEqual(t, snap.Min, 9*time.Millisecond)
}
