package gtest

import (
	"time"

	"github.com/morphy76/gtest/internal/config"
	"github.com/morphy76/gtest/internal/delay"
)

// DelayStrategy defines the algorithm used for generating interaction delays.
type DelayStrategy = delay.DelayStrategy

const (
	// DelayFixed generates a static pause duration.
	DelayFixed = delay.DelayFixed

	// DelayRange generates a uniform random distribution U(min, max).
	DelayRange = delay.DelayRange

	// DelayExpo generates an exponential distribution with specified mean.
	DelayExpo = delay.DelayExpo

	// DelayGaussian generates a normal distribution N(mean, std_dev).
	DelayGaussian = delay.DelayGaussian
)

// DelayGenerator generates successive delay durations.
type DelayGenerator = delay.DelayGenerator

// InteractionDelayConfig holds configuration for think time delays.
type InteractionDelayConfig = config.InteractionDelayConfig

// FixedDelay returns a generator that always yields the constant duration d.
func FixedDelay(d time.Duration) DelayGenerator {
	return delay.FixedDelay(d)
}

// RangeDelay returns a generator yielding uniform random durations within [min, max].
func RangeDelay(min, max time.Duration) DelayGenerator {
	return delay.RangeDelay(min, max)
}

// ExpoDelay returns a generator yielding exponentially distributed durations with the given mean
// and optional min/max clamping.
// Clamp args: clamp[0] = min, clamp[1] = max.
func ExpoDelay(mean time.Duration, clamp ...time.Duration) DelayGenerator {
	return delay.ExpoDelay(mean, clamp...)
}

// GaussianDelay returns a generator yielding normally distributed durations with the given mean,
// stdDev, and optional min/max clamping.
// Clamp args: clamp[0] = min, clamp[1] = max.
func GaussianDelay(mean, stdDev time.Duration, clamp ...time.Duration) DelayGenerator {
	return delay.GaussianDelay(mean, stdDev, clamp...)
}

// NewDelayGenerator constructs a DelayGenerator from an InteractionDelayConfig.
// Returns nil if cfg is nil.
func NewDelayGenerator(cfg *InteractionDelayConfig) (DelayGenerator, error) {
	return delay.NewDelayGenerator(cfg)
}
