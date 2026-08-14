package gtest

import "time"

// ThinkTimeConfig holds configuration for inter-iteration think time delays.
type ThinkTimeConfig struct {
	// Type specifies the delay distribution strategy ("fixed", "range", "expo", "gaussian").
	Type string

	// Duration specifies the constant pause duration when Type is "fixed".
	Duration time.Duration

	// Min specifies the lower bound clamp for "range", "expo", or "gaussian" delays.
	Min time.Duration

	// Max specifies the upper bound clamp for "range", "expo", or "gaussian" delays.
	Max time.Duration

	// Mean specifies the expected average duration for "expo" or "gaussian" delays.
	Mean time.Duration

	// StdDev specifies the standard deviation for "gaussian" delays.
	StdDev time.Duration
}

