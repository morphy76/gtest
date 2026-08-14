package delay_test

import (
	"math"
	"testing"
	"time"

	"github.com/morphy76/gtest/internal/config"
	"github.com/morphy76/gtest/internal/delay"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AC-1.12.1: Fixed delay returns constant duration
func TestFixedDelay(t *testing.T) {
	d := 350 * time.Millisecond
	gen := delay.FixedDelay(d)
	require.NotNil(t, gen)

	for i := 0; i < 50; i++ {
		assert.Equal(t, d, gen.Next())
	}
}

// AC-1.12.2: Range delay returns values uniformly distributed within [min, max]
func TestRangeDelay(t *testing.T) {
	minD := 100 * time.Millisecond
	maxD := 500 * time.Millisecond
	gen := delay.RangeDelay(minD, maxD)
	require.NotNil(t, gen)

	const samples = 5000
	var sum float64
	for i := 0; i < samples; i++ {
		val := gen.Next()
		assert.GreaterOrEqual(t, val, minD, "sample below min")
		assert.LessOrEqual(t, val, maxD, "sample above max")
		sum += float64(val)
	}

	// Uniform distribution mean should be (min + max) / 2
	expectedMean := float64(minD+maxD) / 2.0
	actualMean := sum / samples
	// Allow 5% tolerance for 5000 samples
	assert.InDelta(t, expectedMean, actualMean, expectedMean*0.05)
}

// AC-1.12.3: Expo delay generates exponential distribution with sample mean converging to target mean
func TestExpoDelay(t *testing.T) {
	meanD := 200 * time.Millisecond
	gen := delay.ExpoDelay(meanD)
	require.NotNil(t, gen)

	const samples = 10000
	var sum float64
	for i := 0; i < samples; i++ {
		val := gen.Next()
		assert.GreaterOrEqual(t, val, time.Duration(0), "exponential duration must not be negative")
		sum += float64(val)
	}

	actualMean := sum / samples
	// Allow 5% tolerance on sample mean
	assert.InDelta(t, float64(meanD), actualMean, float64(meanD)*0.05)
}

// AC-1.12.4: Gaussian delay generates normally distributed durations with specified mean and std_dev
func TestGaussianDelay(t *testing.T) {
	meanD := 500 * time.Millisecond
	stdDevD := 50 * time.Millisecond
	gen := delay.GaussianDelay(meanD, stdDevD)
	require.NotNil(t, gen)

	const samples = 10000
	var sum float64
	var sumSqDiff float64
	vals := make([]float64, samples)

	for i := 0; i < samples; i++ {
		val := gen.Next()
		assert.GreaterOrEqual(t, val, time.Duration(0), "duration must be non-negative")
		vals[i] = float64(val)
		sum += vals[i]
	}

	sampleMean := sum / samples
	for _, v := range vals {
		diff := v - sampleMean
		sumSqDiff += diff * diff
	}
	sampleStdDev := math.Sqrt(sumSqDiff / samples)

	// Verify mean within 5%
	assert.InDelta(t, float64(meanD), sampleMean, float64(meanD)*0.05)
	// Verify std dev within 10%
	assert.InDelta(t, float64(stdDevD), sampleStdDev, float64(stdDevD)*0.10)
}

// AC-1.12.5: All delay strategies respect min/max clamping when configured
func TestClamping(t *testing.T) {
	t.Run("expo with min and max clamping", func(t *testing.T) {
		meanD := 100 * time.Millisecond
		minD := 50 * time.Millisecond
		maxD := 200 * time.Millisecond
		gen := delay.ExpoDelay(meanD, minD, maxD)
		require.NotNil(t, gen)

		for i := 0; i < 2000; i++ {
			val := gen.Next()
			assert.GreaterOrEqual(t, val, minD)
			assert.LessOrEqual(t, val, maxD)
		}
	})

	t.Run("gaussian with min and max clamping", func(t *testing.T) {
		meanD := 500 * time.Millisecond
		stdDevD := 200 * time.Millisecond
		minD := 400 * time.Millisecond
		maxD := 600 * time.Millisecond
		gen := delay.GaussianDelay(meanD, stdDevD, minD, maxD)
		require.NotNil(t, gen)

		for i := 0; i < 2000; i++ {
			val := gen.Next()
			assert.GreaterOrEqual(t, val, minD)
			assert.LessOrEqual(t, val, maxD)
		}
	})
}

func TestNewDelayGeneratorFactory(t *testing.T) {
	t.Run("nil config returns nil", func(t *testing.T) {
		gen, err := delay.NewDelayGenerator(nil)
		require.NoError(t, err)
		assert.Nil(t, gen)
	})

	t.Run("fixed config", func(t *testing.T) {
		cfg := &config.InteractionDelayConfig{
			Type:     "fixed",
			Duration: 250 * time.Millisecond,
		}
		gen, err := delay.NewDelayGenerator(cfg)
		require.NoError(t, err)
		require.NotNil(t, gen)
		assert.Equal(t, 250*time.Millisecond, gen.Next())
	})

	t.Run("range config", func(t *testing.T) {
		cfg := &config.InteractionDelayConfig{
			Type: "range",
			Min:  100 * time.Millisecond,
			Max:  200 * time.Millisecond,
		}
		gen, err := delay.NewDelayGenerator(cfg)
		require.NoError(t, err)
		require.NotNil(t, gen)
		val := gen.Next()
		assert.GreaterOrEqual(t, val, 100*time.Millisecond)
		assert.LessOrEqual(t, val, 200*time.Millisecond)
	})

	t.Run("expo config", func(t *testing.T) {
		cfg := &config.InteractionDelayConfig{
			Type: "expo",
			Mean: 300 * time.Millisecond,
			Min:  50 * time.Millisecond,
			Max:  600 * time.Millisecond,
		}
		gen, err := delay.NewDelayGenerator(cfg)
		require.NoError(t, err)
		require.NotNil(t, gen)
		val := gen.Next()
		assert.GreaterOrEqual(t, val, 50*time.Millisecond)
		assert.LessOrEqual(t, val, 600*time.Millisecond)
	})

	t.Run("gaussian config", func(t *testing.T) {
		cfg := &config.InteractionDelayConfig{
			Type:   "gaussian",
			Mean:   400 * time.Millisecond,
			StdDev: 80 * time.Millisecond,
			Min:    200 * time.Millisecond,
			Max:    600 * time.Millisecond,
		}
		gen, err := delay.NewDelayGenerator(cfg)
		require.NoError(t, err)
		require.NotNil(t, gen)
		val := gen.Next()
		assert.GreaterOrEqual(t, val, 200*time.Millisecond)
		assert.LessOrEqual(t, val, 600*time.Millisecond)
	})

	t.Run("unknown type returns error", func(t *testing.T) {
		cfg := &config.InteractionDelayConfig{
			Type: "invalid_type",
		}
		_, err := delay.NewDelayGenerator(cfg)
		require.Error(t, err)
	})
}
