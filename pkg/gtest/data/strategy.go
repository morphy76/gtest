package data

import "errors"

// Strategy defines how records are distributed across VUs and iterations.
type Strategy int

const (
	// Sequential round-robins across records deterministically by VU ID and iteration.
	Sequential Strategy = iota
	// Random selects records uniformly at random with thread safety.
	Random
	// UniquePerVU assigns a deterministic subset/offset of records per Virtual User ID.
	UniquePerVU
	// SharedQueue dispenses each record exactly once across all concurrent VUs until exhausted.
	SharedQueue
)

// ErrDatasetExhausted is returned when a SharedQueue strategy has dispensed all records.
var ErrDatasetExhausted = errors.New("gtest/data: dataset exhausted")

// ContextAccessor provides access to execution context variables needed by dataset strategies.
type ContextAccessor interface {
	VUID() int64
	Iteration() int64
}
