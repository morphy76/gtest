package metric_test

import (
	"testing"
	"time"

	"github.com/morphy76/gtest/internal/metric"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAggregator_InterfaceSatisfaction(t *testing.T) {
	reg := metric.NewRegistry()
	coll := metric.NewCollector(reg)
	_ = metric.NewAggregator(coll)
}

func TestAggregator_MergedHistogramSnapshot(t *testing.T) {
	reg := metric.NewRegistry()
	coll := metric.NewCollector(reg)
	agg := metric.NewAggregator(coll)

	// No data
	emptySnap := agg.MergedHistogramSnapshot("nonexistent")
	assert.Equal(t, int64(0), emptySnap.Count)

	// Single histogram
	d1 := coll.Duration("latency", metric.Tags{"tag": "1"})
	for i := 1; i <= 50; i++ {
		d1.Observe(time.Duration(i) * time.Millisecond)
	}

	snap1 := agg.MergedHistogramSnapshot("latency")
	assert.Equal(t, int64(50), snap1.Count)

	// Multiple histograms sharing name across tags
	d2 := coll.Duration("latency", metric.Tags{"tag": "2"})
	for i := 51; i <= 100; i++ {
		d2.Observe(time.Duration(i) * time.Millisecond)
	}

	mergedSnap := agg.MergedHistogramSnapshot("latency")
	assert.Equal(t, int64(100), mergedSnap.Count)
	assertDurationWithinTolerance(t, 1*time.Millisecond, mergedSnap.Min, 100*time.Microsecond, "min")
	assertDurationWithinTolerance(t, 100*time.Millisecond, mergedSnap.Max, 2*time.Millisecond, "max")
	assertDurationWithinTolerance(t, 50*time.Millisecond, mergedSnap.P50, 2*time.Millisecond, "p50")
}

func TestAggregator_AggregatedCounterValue(t *testing.T) {
	reg := metric.NewRegistry()
	coll := metric.NewCollector(reg)
	agg := metric.NewAggregator(coll)

	c1 := coll.Counter("hits", metric.Tags{"host": "a"})
	c2 := coll.Counter("hits", metric.Tags{"host": "b"})
	c1.Add(15)
	c2.Add(35)

	total := agg.AggregatedCounterValue("hits")
	assert.Equal(t, int64(50), total)
	assert.Equal(t, int64(0), agg.AggregatedCounterValue("nonexistent"))
}

func TestAggregator_AggregatedRateValue_And_RateData(t *testing.T) {
	reg := metric.NewRegistry()
	coll := metric.NewCollector(reg)
	agg := metric.NewAggregator(coll)

	// No data
	val, hasData := agg.RateData("unknown_rate")
	assert.False(t, hasData)
	assert.Equal(t, 0.0, val)
	assert.Equal(t, 0.0, agg.AggregatedRateValue("unknown_rate"))

	// With data across tags
	r1 := coll.Rate("error_rate", metric.Tags{"svc": "x"})
	r2 := coll.Rate("error_rate", metric.Tags{"svc": "y"})
	r1.Add(1, 10) // 1/10
	r2.Add(3, 10) // 3/10

	val, hasData = agg.RateData("error_rate")
	assert.True(t, hasData)
	assert.InDelta(t, 0.2, val, 1e-9) // 4/20 = 0.2
	assert.InDelta(t, 0.2, agg.AggregatedRateValue("error_rate"), 1e-9)
}

func TestAggregator_LastGaugeValue(t *testing.T) {
	reg := metric.NewRegistry()
	coll := metric.NewCollector(reg)
	agg := metric.NewAggregator(coll)

	assert.Equal(t, 0.0, agg.LastGaugeValue("nonexistent"))

	g := coll.Gauge("cpu_usage", metric.Tags{"core": "0"})
	g.Set(75.5)

	assert.InDelta(t, 75.5, agg.LastGaugeValue("cpu_usage"), 1e-9)
}

func TestAggregator_CheckSummaries(t *testing.T) {
	reg := metric.NewRegistry()
	coll := metric.NewCollector(reg)
	agg := metric.NewAggregator(coll)

	// No checks
	assert.Nil(t, agg.CheckSummaries())

	coll.Counter(metric.MetricChecksPassed, metric.Tags{"name": "status 200"}).Add(9)
	coll.Counter(metric.MetricChecksFailed, metric.Tags{"name": "status 200"}).Add(1)
	coll.Counter(metric.MetricChecksPassed, metric.Tags{"name": "body valid"}).Add(20)

	summaries := agg.CheckSummaries()
	require.Len(t, summaries, 2)

	// Sorted alphabetically by Name
	assert.Equal(t, "body valid", summaries[0].Name)
	assert.Equal(t, int64(20), summaries[0].Passed)
	assert.Equal(t, int64(0), summaries[0].Failed)
	assert.Equal(t, int64(20), summaries[0].Total)
	assert.InDelta(t, 100.0, summaries[0].PassPct, 1e-9)

	assert.Equal(t, "status 200", summaries[1].Name)
	assert.Equal(t, int64(9), summaries[1].Passed)
	assert.Equal(t, int64(1), summaries[1].Failed)
	assert.Equal(t, int64(10), summaries[1].Total)
	assert.InDelta(t, 90.0, summaries[1].PassPct, 1e-9)
}
