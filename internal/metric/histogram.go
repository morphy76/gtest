package metric

import (
	"sync"
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

// histogram implements gtest.Duration using HDR histograms merged at report time.
// The default path uses a single shared histogram protected by a mutex.
// For zero-contention writes, callers can use CreateVUHistogram() per goroutine
// and AddHistogram() at cleanup time.
type histogram struct {
	mu         sync.Mutex
	shared     *hdrhistogram.Histogram // primary write target for Observe()
	histograms []*hdrhistogram.Histogram
}

// newHistogram creates a new histogram container with a shared write histogram.
func newHistogram() *histogram {
	return &histogram{
		shared: hdrhistogram.New(histMinValue, histMaxValue, histSignificantFigures),
	}
}

// Observe records one latency sample. Durations are stored in microseconds.
// Values below 1µs are clamped to 1µs. Values above 60s are clamped to 60s.
func (h *histogram) Observe(d time.Duration) {
	us := d.Microseconds()
	if us < histMinValue {
		us = histMinValue
	}
	if us > histMaxValue {
		us = histMaxValue
	}

	h.mu.Lock()
	_ = h.shared.RecordValue(us)
	h.mu.Unlock()
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

// Snapshot merges all recorded histograms (shared + per-VU) and extracts percentile statistics.
// Returns a zero snapshot if no data has been recorded.
func (h *histogram) Snapshot() HistogramSnapshot {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Merge the shared histogram and any per-VU histograms.
	merged := hdrhistogram.New(histMinValue, histMaxValue, histSignificantFigures)

	if h.shared != nil && h.shared.TotalCount() > 0 {
		merged.Merge(h.shared)
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
