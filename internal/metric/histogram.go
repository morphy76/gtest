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

// histogram implements gtest.Duration using per-VU HDR histograms merged at report time.
// Each call to Observe records into a thread-local-style histogram to avoid contention.
type histogram struct {
	mu         sync.Mutex
	histograms []*hdrhistogram.Histogram
}

// newHistogram creates a new histogram container.
func newHistogram() *histogram {
	return &histogram{}
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
	defer h.mu.Unlock()

	// Append a new single-value histogram for each observation.
	// For high-throughput use, the store provides per-VU histograms via CreateVUHistogram.
	hist := hdrhistogram.New(histMinValue, histMaxValue, histSignificantFigures)
	if err := hist.RecordValue(us); err == nil {
		h.histograms = append(h.histograms, hist)
	}
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

// Snapshot merges all recorded histograms and extracts percentile statistics.
// Returns a zero snapshot if no data has been recorded.
func (h *histogram) Snapshot() HistogramSnapshot {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.histograms) == 0 {
		return HistogramSnapshot{}
	}

	merged := hdrhistogram.New(histMinValue, histMaxValue, histSignificantFigures)
	for _, hist := range h.histograms {
		merged.Merge(hist)
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
