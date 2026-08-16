package sla

import (
	"cmp"
	"fmt"
	"strconv"
	"time"

	"github.com/morphy76/vuhive/internal/config"
	"github.com/morphy76/vuhive/internal/metric"
)

// MetricReader defines the read-only metric capabilities required by the SLA evaluator.
type MetricReader interface {
	MetricType(name string) (metric.MetricType, bool)
	MergedHistogramSnapshot(name string) metric.HistogramSnapshot
	AggregatedCounterValue(name string) int64
	RateData(name string) (float64, bool)
	LastGaugeValue(name string) float64
}

// Evaluate checks all threshold configurations against the metrics reader.
// It evaluates EVERY threshold in order (no short-circuiting) and returns a slice of results.
func Evaluate(thresholds []config.ThresholdConfig, reader MetricReader) []ThresholdResult {
	results := make([]ThresholdResult, 0, len(thresholds))
	for _, th := range thresholds {
		results = append(results, EvaluateThreshold(th, reader))
	}
	return results
}

// EvaluateThreshold evaluates a single threshold configuration against the metrics reader.
func EvaluateThreshold(th config.ThresholdConfig, reader MetricReader) ThresholdResult {
	res := ThresholdResult{
		Threshold: th,
		Metric:    th.Metric,
		Stat:      th.Stat,
		Operator:  th.Operator,
		Target:    th.Target,
	}

	_, exists := reader.MetricType(th.Metric)
	if !exists {
		res.Passed = false
		res.Actual = "no data"
		res.Reason = "no data"
		return res
	}

	if config.IsDurationStat(th.Stat) {
		snap := reader.MergedHistogramSnapshot(th.Metric)
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
		c := reader.AggregatedCounterValue(th.Metric)
		actualFloat = float64(c)
		res.Actual = strconv.FormatInt(c, 10)
	case "rate":
		rateVal, hasData := reader.RateData(th.Metric)
		if !hasData {
			res.Passed = false
			res.Actual = "no data"
			res.Reason = "no data"
			return res
		}
		actualFloat = rateVal
		res.Actual = strconv.FormatFloat(actualFloat, 'f', -1, 64)
	case "value":
		actualFloat = reader.LastGaugeValue(th.Metric)
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

// compare evaluates whether the actual value satisfies the threshold target based on
// the specified comparison operator op.
//
// Supported operators are:
//   - "<"  : actual is strictly less than target
//   - "<=" : actual is less than or equal to target
//   - ">"  : actual is strictly greater than target
//   - ">=" : actual is greater than or equal to target
//
// For any unrecognized operator string, compare returns false.
func compare[T cmp.Ordered](actual, target T, op string) bool {
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

// compareDuration evaluates whether actual duration satisfies target duration using operator op (<, <=, >, >=).
func compareDuration(actual, target time.Duration, op string) bool {
	return compare(actual, target, op)
}

// compareFloat evaluates whether actual float satisfies target float using operator op (<, <=, >, >=).
func compareFloat(actual, target float64, op string) bool {
	return compare(actual, target, op)
}
