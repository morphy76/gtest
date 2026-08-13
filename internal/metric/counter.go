package metric

import (
	"sync/atomic"
)

// counter implements gtest.Counter using atomic operations for lock-free thread safety.
type counter struct {
	val atomic.Int64
}

// Inc increments the counter by 1.
func (c *counter) Inc() {
	c.val.Add(1)
}

// Add increments the counter by delta.
func (c *counter) Add(delta int64) {
	c.val.Add(delta)
}

// Value returns the current counter value.
func (c *counter) Value() int64 {
	return c.val.Load()
}
