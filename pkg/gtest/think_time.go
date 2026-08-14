package gtest

import "time"

// ThinkTimeConfig holds configuration for inter-iteration think time delays.
type ThinkTimeConfig struct {
	// Type specifies the delay distribution strategy ("fixed", "range", "expo", "gaussian").
	Type string `mapstructure:"type"`

	// Duration specifies the constant pause duration when Type is "fixed".
	Duration time.Duration `mapstructure:"duration"`

	// Min specifies the lower bound clamp for "range", "expo", or "gaussian" delays.
	Min time.Duration `mapstructure:"min"`

	// Max specifies the upper bound clamp for "range", "expo", or "gaussian" delays.
	Max time.Duration `mapstructure:"max"`

	// Mean specifies the expected average duration for "expo" or "gaussian" delays.
	Mean time.Duration `mapstructure:"mean"`

	// StdDev specifies the standard deviation for "gaussian" delays.
	StdDev time.Duration `mapstructure:"std_dev"`
}
