package vuhive_test

import (
	"testing"
	"time"

	"github.com/morphy76/vuhive/pkg/vuhive"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublicDelayAPI(t *testing.T) {
	t.Run("FixedDelay", func(t *testing.T) {
		gen := vuhive.FixedDelay(100 * time.Millisecond)
		require.NotNil(t, gen)
		assert.Equal(t, 100*time.Millisecond, gen.Next())
	})

	t.Run("RangeDelay", func(t *testing.T) {
		gen := vuhive.RangeDelay(50*time.Millisecond, 150*time.Millisecond)
		require.NotNil(t, gen)
		val := gen.Next()
		assert.GreaterOrEqual(t, val, 50*time.Millisecond)
		assert.LessOrEqual(t, val, 150*time.Millisecond)
	})

	t.Run("ExpoDelay", func(t *testing.T) {
		gen := vuhive.ExpoDelay(200*time.Millisecond, 50*time.Millisecond, 500*time.Millisecond)
		require.NotNil(t, gen)
		val := gen.Next()
		assert.GreaterOrEqual(t, val, 50*time.Millisecond)
		assert.LessOrEqual(t, val, 500*time.Millisecond)
	})

	t.Run("GaussianDelay", func(t *testing.T) {
		gen := vuhive.GaussianDelay(300*time.Millisecond, 50*time.Millisecond, 100*time.Millisecond, 500*time.Millisecond)
		require.NotNil(t, gen)
		val := gen.Next()
		assert.GreaterOrEqual(t, val, 100*time.Millisecond)
		assert.LessOrEqual(t, val, 500*time.Millisecond)
	})

	t.Run("NewDelayGenerator from Config", func(t *testing.T) {
		cfg := &vuhive.InteractionDelayConfig{
			Type:     "fixed",
			Duration: 200 * time.Millisecond,
		}
		gen, err := vuhive.NewDelayGenerator(cfg)
		require.NoError(t, err)
		require.NotNil(t, gen)
		assert.Equal(t, 200*time.Millisecond, gen.Next())
	})
}
