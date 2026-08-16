package metric

import (
	"sync"
	"sync/atomic"
	"time"

	hdrhistogram "github.com/HdrHistogram/hdrhistogram-go"
)

const (
	// histMinValue is the minimum recordable value in microseconds (1µs).
	histMinValue = 1
	// histMaxValue is the maximum recordable value in microseconds (60s).
	histMaxValue = 60_000_000
	// histSignificantFigures is the number of significant digits of precision.
	histSignificantFigures = 3
)

const histShards = 16

type histShard struct {
	mu   sync.Mutex
	hist *hdrhistogram.Histogram
}

// histogram implements vuhive.Duration using sharded HDR histograms merged at report time.
// Observations are striped across 16 independent mutex-guarded shards to eliminate lock contention.
type histogram struct {
	shards     [histShards]histShard
	seq        atomic.Uint32
	mu         sync.Mutex
	histograms []*hdrhistogram.Histogram
}

// newHistogram creates a new histogram container with sharded write targets.
func newHistogram() *histogram {
	h := &histogram{}
	for i := 0; i < histShards; i++ {
		h.shards[i].hist = hdrhistogram.New(histMinValue, histMaxValue, histSignificantFigures)
	}
	return h
}

// Type returns the metric type for histogram/duration.
func (h *histogram) Type() MetricType {
	return MetricTypeDuration
}

// Observe records one latency sample. Durations are stored in microseconds.
// Values below 1µs are clamped to 1µs. Values above 60s are clamped to 60s.
// Observations are striped across independent shards for zero lock contention.
func (h *histogram) Observe(d time.Duration) {
	us := d.Microseconds()
	if us < histMinValue {
		us = histMinValue
	}
	if us > histMaxValue {
		us = histMaxValue
	}

	shardIdx := h.seq.Add(1) % histShards
	shard := &h.shards[shardIdx]
	shard.mu.Lock()
	_ = shard.hist.RecordValue(us)
	shard.mu.Unlock()
}

// CreateVUHistogram returns a new HDR histogram instance that a VU can use directly.
// The returned histogram is NOT thread-safe — each VU should own its own instance.
// Call AddHistogram to merge it back at report time.
func (h *histogram) CreateVUHistogram() *hdrhistogram.Histogram {
	return hdrhistogram.New(histMinValue, histMaxValue, histSignificantFigures)
}

// AddHistogram merges a VU's histogram into the collection for report-time aggregation.
func (h *histogram) AddHistogram(hist *hdrhistogram.Histogram) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.histograms = append(h.histograms, hist)
}

// HistogramSnapshot holds the merged statistics from all VU histograms.
type HistogramSnapshot struct {
	Count int64
	Min   time.Duration
	Max   time.Duration
	Mean  time.Duration
	P50   time.Duration
	P90   time.Duration
	P95   time.Duration
	P99   time.Duration
}

// Snapshot merges all recorded histograms (shards + per-VU) and extracts percentile statistics.
// Returns a zero snapshot if no data has been recorded.
func (h *histogram) Snapshot() HistogramSnapshot {
	h.mu.Lock()
	defer h.mu.Unlock()

	merged := hdrhistogram.New(histMinValue, histMaxValue, histSignificantFigures)

	for i := 0; i < histShards; i++ {
		shard := &h.shards[i]
		shard.mu.Lock()
		if shard.hist != nil && shard.hist.TotalCount() > 0 {
			merged.Merge(shard.hist)
		}
		shard.mu.Unlock()
	}

	for _, hist := range h.histograms {
		merged.Merge(hist)
	}

	if merged.TotalCount() == 0 {
		return HistogramSnapshot{}
	}

	return HistogramSnapshot{
		Count: merged.TotalCount(),
		Min:   time.Duration(merged.Min()) * time.Microsecond,
		Max:   time.Duration(merged.Max()) * time.Microsecond,
		Mean:  time.Duration(merged.Mean()) * time.Microsecond,
		P50:   time.Duration(merged.ValueAtPercentile(50)) * time.Microsecond,
		P90:   time.Duration(merged.ValueAtPercentile(90)) * time.Microsecond,
		P95:   time.Duration(merged.ValueAtPercentile(95)) * time.Microsecond,
		P99:   time.Duration(merged.ValueAtPercentile(99)) * time.Microsecond,
	}
}

// Compile-time checks.
var (
	_ Duration = (*histogram)(nil)
	_ Metric   = (*histogram)(nil)
)
