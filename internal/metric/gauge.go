package metric

import (
	"math"
	"sync/atomic"
)

// gauge implements gtest.Gauge using atomic operations on float64 bits for lock-free thread safety.
type gauge struct {
	bits atomic.Uint64
}

// Set stores the given value atomically.
func (g *gauge) Set(value float64) {
	g.bits.Store(math.Float64bits(value))
}

// Add atomically adds delta to the current gauge value using a CAS loop.
func (g *gauge) Add(delta float64) {
	for {
		old := g.bits.Load()
		newVal := math.Float64frombits(old) + delta
		if g.bits.CompareAndSwap(old, math.Float64bits(newVal)) {
			return
		}
	}
}

// Value returns the current gauge value.
func (g *gauge) Value() float64 {
	return math.Float64frombits(g.bits.Load())
}
