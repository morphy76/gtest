package metric_test

import (
	"testing"
	"time"

	"github.com/morphy76/vuhive/internal/metric"
	"github.com/stretchr/testify/assert"
)

func TestAlloc_Counter_Inc(t *testing.T) {
	coll := metric.NewCollector(nil)
	counter := coll.Counter("test_counter", metric.Tags{})

	allocs := testing.AllocsPerRun(1000, func() {
		counter.Inc()
	})

	assert.Equal(t, float64(0), allocs, "Counter.Inc must produce 0 heap allocations")
}

func TestAlloc_Counter_Add(t *testing.T) {
	coll := metric.NewCollector(nil)
	counter := coll.Counter("test_counter", metric.Tags{})

	allocs := testing.AllocsPerRun(1000, func() {
		counter.Add(5)
	})

	assert.Equal(t, float64(0), allocs, "Counter.Add must produce 0 heap allocations")
}

func TestAlloc_Gauge_Set(t *testing.T) {
	coll := metric.NewCollector(nil)
	gauge := coll.Gauge("test_gauge", metric.Tags{})

	allocs := testing.AllocsPerRun(1000, func() {
		gauge.Set(42.5)
	})

	assert.Equal(t, float64(0), allocs, "Gauge.Set must produce 0 heap allocations")
}

func TestAlloc_Gauge_Add(t *testing.T) {
	coll := metric.NewCollector(nil)
	gauge := coll.Gauge("test_gauge", metric.Tags{})

	allocs := testing.AllocsPerRun(1000, func() {
		gauge.Add(1.0)
	})

	assert.Equal(t, float64(0), allocs, "Gauge.Add must produce 0 heap allocations")
}

func TestAlloc_Duration_Observe(t *testing.T) {
	coll := metric.NewCollector(nil)
	duration := coll.Duration("test_duration", metric.Tags{})

	allocs := testing.AllocsPerRun(1000, func() {
		duration.Observe(10 * time.Millisecond)
	})

	assert.Equal(t, float64(0), allocs, "Duration.Observe must produce 0 heap allocations")
}

func TestAlloc_Rate_Add(t *testing.T) {
	coll := metric.NewCollector(nil)
	rate := coll.Rate("test_rate", metric.Tags{})

	allocs := testing.AllocsPerRun(1000, func() {
		rate.Add(1, 1)
	})

	assert.Equal(t, float64(0), allocs, "Rate.Add must produce 0 heap allocations")
}

func TestAlloc_Collector_Lookup_EmptyTags(t *testing.T) {
	coll := metric.NewCollector(nil)
	emptyTags := metric.Tags{}
	_ = coll.Counter("test_counter", emptyTags)

	allocs := testing.AllocsPerRun(1000, func() {
		_ = coll.Counter("test_counter", emptyTags)
	})

	assert.Equal(t, float64(0), allocs, "Collector.Counter lookup with empty tags must produce 0 heap allocations")
}
