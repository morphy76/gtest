package sla

import "github.com/morphy76/gtest/internal/config"

// ThresholdResult represents the evaluation outcome for a single SLA threshold.
type ThresholdResult struct {
	Threshold config.ThresholdConfig `json:"threshold"`
	Metric    string                 `json:"metric"`
	Stat      string                 `json:"stat"`
	Operator  string                 `json:"operator"`
	Target    string                 `json:"target"`
	Actual    string                 `json:"actual"`
	Passed    bool                   `json:"passed"`
	Reason    string                 `json:"reason,omitempty"`
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
