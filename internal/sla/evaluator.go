package sla

import (
	"fmt"
	"strconv"
	"time"

	"github.com/morphy76/gtest/internal/config"
	"github.com/morphy76/gtest/internal/metric"
)

// Evaluate checks all threshold configurations against the metrics store snapshot.
// It evaluates EVERY threshold in order (no short-circuiting) and returns a slice of results.
func Evaluate(thresholds []config.ThresholdConfig, store *metric.Store) []ThresholdResult {
	results := make([]ThresholdResult, 0, len(thresholds))
	for _, th := range thresholds {
		results = append(results, EvaluateThreshold(th, store))
	}
	return results
}

// EvaluateThreshold evaluates a single threshold configuration against the metrics store.
func EvaluateThreshold(th config.ThresholdConfig, store *metric.Store) ThresholdResult {
	res := ThresholdResult{
		Threshold: th,
		Metric:    th.Metric,
		Stat:      th.Stat,
		Operator:  th.Operator,
		Target:    th.Target,
	}

	_, exists := store.MetricType(th.Metric)
	if !exists {
		res.Passed = false
		res.Actual = "no data"
		res.Reason = "no data"
		return res
	}

	if config.IsDurationStat(th.Stat) {
		snap := store.MergedHistogramSnapshot(th.Metric)
		if snap.Count == 0 {
			res.Passed = false
			res.Actual = "no data"
			res.Reason = "no data"
			return res
		}

		var actualDuration time.Duration
		switch th.Stat {
		case "p50":
			actualDuration = snap.P50
		case "p90":
			actualDuration = snap.P90
		case "p95":
			actualDuration = snap.P95
		case "p99":
			actualDuration = snap.P99
		case "mean":
			actualDuration = snap.Mean
		case "max":
			actualDuration = snap.Max
		}

		res.Actual = actualDuration.String()
		res.Passed = compareDuration(actualDuration, th.TargetDuration, th.Operator)
		if !res.Passed {
			res.Reason = fmt.Sprintf("actual %s (%s) does not satisfy %s %s",
				th.Stat, actualDuration.String(), th.Operator, th.Target)
		}
		return res
	}

	var actualFloat float64
	switch th.Stat {
	case "count":
		c := store.AggregatedCounterValue(th.Metric)
		actualFloat = float64(c)
		res.Actual = strconv.FormatInt(c, 10)
	case "rate":
		rateVal, hasData := store.RateData(th.Metric)
		if !hasData {
			res.Passed = false
			res.Actual = "no data"
			res.Reason = "no data"
			return res
		}
		actualFloat = rateVal
		res.Actual = strconv.FormatFloat(actualFloat, 'f', -1, 64)
	case "value":
		actualFloat = store.LastGaugeValue(th.Metric)
		res.Actual = strconv.FormatFloat(actualFloat, 'f', -1, 64)
	default:
		res.Passed = false
		res.Actual = "unknown stat"
		res.Reason = fmt.Sprintf("unsupported stat: %s", th.Stat)
		return res
	}

	res.Passed = compareFloat(actualFloat, th.TargetFloat, th.Operator)
	if !res.Passed {
		res.Reason = fmt.Sprintf("actual %s (%s) does not satisfy %s %s",
			th.Stat, res.Actual, th.Operator, th.Target)
	}

	return res
}

func compareDuration(actual, target time.Duration, op string) bool {
	switch op {
	case "<":
		return actual < target
	case "<=":
		return actual <= target
	case ">":
		return actual > target
	case ">=":
		return actual >= target
	default:
		return false
	}
}

func compareFloat(actual, target float64, op string) bool {
	switch op {
	case "<":
		return actual < target
	case "<=":
		return actual <= target
	case ">":
		return actual > target
	case ">=":
		return actual >= target
	default:
		return false
	}
}
