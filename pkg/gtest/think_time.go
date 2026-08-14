package gtest

import "time"

// ThinkTimeConfig holds configuration for inter-iteration think time delays.
type ThinkTimeConfig struct {
	Type     string        `mapstructure:"type"`
	Duration time.Duration `mapstructure:"duration"`
	Min      time.Duration `mapstructure:"min"`
	Max      time.Duration `mapstructure:"max"`
	Mean     time.Duration `mapstructure:"mean"`
	StdDev   time.Duration `mapstructure:"std_dev"`
}
