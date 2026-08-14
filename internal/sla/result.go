package sla

import "github.com/morphy76/gtest/internal/config"

// ThresholdResult represents the evaluation outcome for a single SLA threshold.
type ThresholdResult struct {
	Threshold config.ThresholdConfig
	Metric    string
	Stat      string
	Operator  string
	Target    string
	Actual    string
	Passed    bool
	Reason    string
}

// AllPassed returns true if every threshold result in the slice passed.
func AllPassed(results []ThresholdResult) bool {
	for _, r := range results {
		if !r.Passed {
			return false
		}
	}
	return true
}
