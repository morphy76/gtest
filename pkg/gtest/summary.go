package gtest

import (
	"github.com/morphy76/gtest/internal/engine"
)

// MetricSummary represents a metric entry in the execution summary.
type MetricSummary = engine.MetricSummary

// ThresholdSummary represents the outcome of a single SLA threshold evaluation.
type ThresholdSummary = engine.ThresholdSummary

// SummaryData contains the complete structured report information post-execution.
type SummaryData = engine.SummaryData
