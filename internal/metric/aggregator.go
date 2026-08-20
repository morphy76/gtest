package metric

import (
	"sort"
	"strings"
	"time"
)

// CheckSummary holds aggregated metrics for a named check.
type CheckSummary struct {
	Name    string
	Passed  int64
	Failed  int64
	Total   int64
	PassPct float64
}

// GroupSummary holds aggregated latency metrics for a named transaction group.
type GroupSummary struct {
	Name   string
	Count  int64
	Min    time.Duration
	Mean   time.Duration
	P50    time.Duration
	P90    time.Duration
	P95    time.Duration
	P99    time.Duration
	Max    time.Duration
}

// MetricProvider provides read-only traversal over recorded metrics.
type MetricProvider interface {
	ForEachCounter(fn func(key metricKey, c *counter))
	ForEachGauge(fn func(key metricKey, g *gauge))
	ForEachHistogram(fn func(key metricKey, h *histogram))
	ForEachRate(fn func(key metricKey, r *rate))
}

// Aggregator computes summary statistics across recorded metrics.
type Aggregator interface {
	MergedHistogramSnapshot(name string) HistogramSnapshot
	AggregatedCounterValue(name string) int64
	AggregatedRateValue(name string) float64
	RateData(name string) (float64, bool)
	LastGaugeValue(name string) float64
	CheckSummaries() []CheckSummary
	GroupSummaries() []GroupSummary
}

type aggregator struct {
	provider MetricProvider
}

// NewAggregator creates a new Aggregator using the given MetricProvider.
func NewAggregator(provider MetricProvider) Aggregator {
	return &aggregator{
		provider: provider,
	}
}

// MergedHistogramSnapshot returns a merged snapshot of all histogram instances
// sharing the given metric name, across all tag combinations.
// Returns a zero snapshot if no histograms exist for the name.
func (a *aggregator) MergedHistogramSnapshot(name string) HistogramSnapshot {
	var all []*histogram
	a.provider.ForEachHistogram(func(key metricKey, h *histogram) {
		if key.name == name {
			all = append(all, h)
		}
	})

	if len(all) == 0 {
		return HistogramSnapshot{}
	}

	if len(all) == 1 {
		return all[0].Snapshot()
	}

	// Collect shards + per-VU histograms from every tag variant.
	merged := newHistogram()
	for _, h := range all {
		h.mu.Lock()
		for i := 0; i < histShards; i++ {
			shard := &h.shards[i]
			shard.mu.Lock()
			if shard.hist != nil && shard.hist.TotalCount() > 0 {
				merged.histograms = append(merged.histograms, shard.hist)
			}
			shard.mu.Unlock()
		}
		merged.histograms = append(merged.histograms, h.histograms...)
		h.mu.Unlock()
	}
	return merged.Snapshot()
}

// AggregatedCounterValue returns the sum of all counter values sharing the given name
// across all tag combinations.
func (a *aggregator) AggregatedCounterValue(name string) int64 {
	var total int64
	a.provider.ForEachCounter(func(key metricKey, c *counter) {
		if key.name == name {
			total += c.Value()
		}
	})
	return total
}

// AggregatedRateValue returns the aggregated rate across all tag combinations for the given name.
// Returns 0 if no rate data exists.
func (a *aggregator) AggregatedRateValue(name string) float64 {
	val, _ := a.RateData(name)
	return val
}

// RateData returns the aggregated rate and a boolean indicating whether any observations exist.
func (a *aggregator) RateData(name string) (float64, bool) {
	var totalNum, totalDen int64
	a.provider.ForEachRate(func(key metricKey, r *rate) {
		if key.name == name {
			totalNum += r.Numerator()
			totalDen += r.Denominator()
		}
	})

	if totalDen == 0 {
		return 0, false
	}
	return float64(totalNum) / float64(totalDen), true
}

// LastGaugeValue returns the last recorded value for the given gauge name.
// If multiple tag variants exist, returns the value of an arbitrary one.
// Returns 0 if no gauge data exists.
func (a *aggregator) LastGaugeValue(name string) float64 {
	var val float64
	var found bool
	a.provider.ForEachGauge(func(key metricKey, g *gauge) {
		if !found && key.name == name {
			val = g.Value()
			found = true
		}
	})
	return val
}

// CheckSummaries returns aggregated pass/fail statistics for all named checks sorted by name.
func (a *aggregator) CheckSummaries() []CheckSummary {
	checkMap := make(map[string]*CheckSummary)

	a.provider.ForEachCounter(func(key metricKey, c *counter) {
		if key.name != MetricChecksPassed && key.name != MetricChecksFailed {
			return
		}
		checkName := strings.TrimPrefix(key.tagsKey, "name=")
		if checkName == "" {
			return
		}

		entry, ok := checkMap[checkName]
		if !ok {
			entry = &CheckSummary{Name: checkName}
			checkMap[checkName] = entry
		}

		if key.name == MetricChecksPassed {
			entry.Passed += c.Value()
		} else {
			entry.Failed += c.Value()
		}
	})

	if len(checkMap) == 0 {
		return nil
	}

	names := make([]string, 0, len(checkMap))
	for name := range checkMap {
		names = append(names, name)
	}
	sort.Strings(names)

	summaries := make([]CheckSummary, 0, len(names))
	for _, name := range names {
		entry := checkMap[name]
		entry.Total = entry.Passed + entry.Failed
		if entry.Total > 0 {
			entry.PassPct = (float64(entry.Passed) / float64(entry.Total)) * 100.0
		}
		summaries = append(summaries, *entry)
	}

	return summaries
}

// GroupSummaries returns aggregated latency statistics for all transaction groups sorted alphabetically by group name.
func (a *aggregator) GroupSummaries() []GroupSummary {
	var groupNames []string
	seen := make(map[string]struct{})

	a.provider.ForEachHistogram(func(key metricKey, h *histogram) {
		if strings.HasPrefix(key.name, MetricGroupPrefix) && strings.HasSuffix(key.name, MetricGroupSuffix) {
			groupPath := strings.TrimSuffix(strings.TrimPrefix(key.name, MetricGroupPrefix), MetricGroupSuffix)
			if groupPath != "" {
				if _, ok := seen[groupPath]; !ok {
					seen[groupPath] = struct{}{}
					groupNames = append(groupNames, groupPath)
				}
			}
		}
	})

	if len(groupNames) == 0 {
		return nil
	}

	sort.Strings(groupNames)

	summaries := make([]GroupSummary, 0, len(groupNames))
	for _, name := range groupNames {
		metricName := MetricGroupPrefix + name + MetricGroupSuffix
		snap := a.MergedHistogramSnapshot(metricName)
		summaries = append(summaries, GroupSummary{
			Name:  name,
			Count: snap.Count,
			Min:   snap.Min,
			Mean:  snap.Mean,
			P50:   snap.P50,
			P90:   snap.P90,
			P95:   snap.P95,
			P99:   snap.P99,
			Max:   snap.Max,
		})
	}

	return summaries
}

// Compile-time check
var _ Aggregator = (*aggregator)(nil)

