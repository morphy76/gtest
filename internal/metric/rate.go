package metric

import (
	"sync/atomic"
)

// rate implements vuhive.Rate using two atomic accumulators for lock-free thread safety.
type rate struct {
	numerator   atomic.Int64
	denominator atomic.Int64
}

// Type returns the metric type for rate.
func (r *rate) Type() MetricType {
	return MetricTypeRate
}

// Add records numerator events out of denominator total attempts.
// Both must be >= 0. Denominator == 0 is ignored (no observation recorded).
func (r *rate) Add(numerator, denominator int64) {
	if denominator <= 0 {
		return
	}
	if numerator < 0 {
		numerator = 0
	}
	r.numerator.Add(numerator)
	r.denominator.Add(denominator)
}

// Value returns the computed rate as sum(numerator)/sum(denominator).
// Returns 0 if no observations have been recorded (denominator == 0).
func (r *rate) Value() float64 {
	d := r.denominator.Load()
	if d == 0 {
		return 0
	}
	return float64(r.numerator.Load()) / float64(d)
}

// Numerator returns the accumulated numerator value.
func (r *rate) Numerator() int64 {
	return r.numerator.Load()
}

// Denominator returns the accumulated denominator value.
func (r *rate) Denominator() int64 {
	return r.denominator.Load()
}

// Compile-time checks.
var (
	_ Rate   = (*rate)(nil)
	_ Metric = (*rate)(nil)
)
