// Package delay implements user interaction delay (think time) strategies.
package delay

import (
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/morphy76/vuhive/internal/config"
)

// DelayStrategy defines the algorithm used for generating interaction delays.
type DelayStrategy string

const (
	// DelayFixed generates a static pause duration.
	DelayFixed DelayStrategy = "fixed"

	// DelayRange generates a uniform random distribution U(min, max).
	DelayRange DelayStrategy = "range"

	// DelayExpo generates an exponential distribution with specified mean.
	DelayExpo DelayStrategy = "expo"

	// DelayGaussian generates a normal distribution N(mean, std_dev).
	DelayGaussian DelayStrategy = "gaussian"
)

// DelayGenerator generates successive delay durations.
type DelayGenerator interface {
	Next() time.Duration
}

// --- Fixed Delay ---

type fixedDelay struct {
	duration time.Duration
}

func (f *fixedDelay) Next() time.Duration {
	return f.duration
}

// FixedDelay returns a generator that always yields the constant duration d.
func FixedDelay(d time.Duration) DelayGenerator {
	if d < 0 {
		d = 0
	}
	return &fixedDelay{duration: d}
}

// --- Range Delay ---

type rangeDelay struct {
	min time.Duration
	max time.Duration
}

func (r *rangeDelay) Next() time.Duration {
	if r.max <= r.min {
		return r.min
	}
	delta := int64(r.max - r.min)
	offset := rand.Int64N(delta + 1)
	return r.min + time.Duration(offset)
}

// RangeDelay returns a generator yielding uniform random durations within [min, max].
func RangeDelay(min, max time.Duration) DelayGenerator {
	if min < 0 {
		min = 0
	}
	if max < min {
		min, max = max, min
	}
	return &rangeDelay{min: min, max: max}
}

// --- Expo Delay ---

type expoDelay struct {
	mean time.Duration
	min  time.Duration
	max  time.Duration
}

func (e *expoDelay) Next() time.Duration {
	// D = -mean * ln(U), equivalent to rand.ExpFloat64() * mean
	val := rand.ExpFloat64() * float64(e.mean)
	d := time.Duration(val)
	if e.min > 0 && d < e.min {
		d = e.min
	}
	if e.max > 0 && d > e.max {
		d = e.max
	}
	return d
}

// ExpoDelay returns a generator yielding exponentially distributed durations with the given mean
// and optional min/max clamping.
// Clamp args: clamp[0] = min, clamp[1] = max.
func ExpoDelay(mean time.Duration, clamp ...time.Duration) DelayGenerator {
	if mean < 0 {
		mean = 0
	}
	var min, max time.Duration
	if len(clamp) > 0 {
		min = clamp[0]
	}
	if len(clamp) > 1 {
		max = clamp[1]
	}
	return &expoDelay{mean: mean, min: min, max: max}
}

// --- Gaussian Delay ---

type gaussianDelay struct {
	mean   time.Duration
	stdDev time.Duration
	min    time.Duration
	max    time.Duration
}

func (g *gaussianDelay) Next() time.Duration {
	// D = mean + std_dev * NormFloat64()
	val := float64(g.mean) + float64(g.stdDev)*rand.NormFloat64()
	if val < 0 {
		val = 0
	}
	d := time.Duration(val)
	if g.min > 0 && d < g.min {
		d = g.min
	}
	if g.max > 0 && d > g.max {
		d = g.max
	}
	return d
}

// GaussianDelay returns a generator yielding normally distributed durations with the given mean,
// stdDev, and optional min/max clamping.
// Clamp args: clamp[0] = min, clamp[1] = max.
func GaussianDelay(mean, stdDev time.Duration, clamp ...time.Duration) DelayGenerator {
	if mean < 0 {
		mean = 0
	}
	if stdDev < 0 {
		stdDev = 0
	}
	var min, max time.Duration
	if len(clamp) > 0 {
		min = clamp[0]
	}
	if len(clamp) > 1 {
		max = clamp[1]
	}
	return &gaussianDelay{mean: mean, stdDev: stdDev, min: min, max: max}
}

// --- Factory ---

// NewDelayGenerator constructs a DelayGenerator from an InteractionDelayConfig.
// Returns nil if cfg is nil.
func NewDelayGenerator(cfg *config.InteractionDelayConfig) (DelayGenerator, error) {
	if cfg == nil {
		return nil, nil
	}

	switch cfg.Type {
	case string(DelayFixed):
		return FixedDelay(cfg.Duration), nil
	case string(DelayRange):
		return RangeDelay(cfg.Min, cfg.Max), nil
	case string(DelayExpo):
		return ExpoDelay(cfg.Mean, cfg.Min, cfg.Max), nil
	case string(DelayGaussian):
		return GaussianDelay(cfg.Mean, cfg.StdDev, cfg.Min, cfg.Max), nil
	default:
		return nil, fmt.Errorf("unknown delay type: %q", cfg.Type)
	}
}
