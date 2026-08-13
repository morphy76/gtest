package metric_test

import (
	"sync"
	"testing"
	"time"

	"github.com/morphy76/gtest/internal/metric"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AC-1.3.1: Counter.Inc is atomic — concurrent increments from 100 goroutines produce exact total
func TestCounterConcurrentInc(t *testing.T) {
	store := metric.NewStore()
	c := store.Counter("requests", metric.Tags{})

	const goroutines = 100
	const incsPerGoroutine = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			for range incsPerGoroutine {
				c.Inc()
			}
		}()
	}

	wg.Wait()

	ic := store.GetCounter("requests", metric.Tags{})
	require.NotNil(t, ic)
	assert.Equal(t, int64(goroutines*incsPerGoroutine), ic.Value())
}

// AC-1.3.2: Counter with identical name+tags returns the same instance
func TestCounterSameInstance(t *testing.T) {
	store := metric.NewStore()
	tags := metric.Tags{"env": "staging"}

	c1 := store.Counter("requests", tags)
	c2 := store.Counter("requests", tags)

	c1.Inc()
	c2.Inc()

	ic := store.GetCounter("requests", tags)
	require.NotNil(t, ic)
	assert.Equal(t, int64(2), ic.Value(), "both handles should write to the same counter")
}

// AC-1.3.3: Counter and Duration with same name panic (type collision)
func TestTypeCollisionPanics(t *testing.T) {
	store := metric.NewStore()
	store.Counter("latency", metric.Tags{})

	assert.Panics(t, func() {
		store.Duration("latency", metric.Tags{})
	}, "registering Duration after Counter with same name must panic")
}

// AC-1.3.4: Duration.Observe stores values retrievable as p50/p95/p99 within HDR precision
func TestDurationObservePercentiles(t *testing.T) {
	store := metric.NewStore()
	d := store.Duration("http_latency", metric.Tags{})

	// Record 100 values from 1ms to 100ms.
	for i := 1; i <= 100; i++ {
		d.Observe(time.Duration(i) * time.Millisecond)
	}

	snap := store.MergedHistogramSnapshot("http_latency")
	assert.Equal(t, int64(100), snap.Count)

	// HDR histograms have precision within 3 significant digits.
	// p50 should be ~50ms, p95 ~95ms, p99 ~99ms.
	assertDurationWithinTolerance(t, 50*time.Millisecond, snap.P50, 2*time.Millisecond, "p50")
	assertDurationWithinTolerance(t, 95*time.Millisecond, snap.P95, 2*time.Millisecond, "p95")
	assertDurationWithinTolerance(t, 99*time.Millisecond, snap.P99, 2*time.Millisecond, "p99")
}

// AC-1.3.5: Rate.Add(1,1) + Rate.Add(0,1) produces rate = 0.5
func TestRateComputation(t *testing.T) {
	store := metric.NewStore()
	r := store.Rate("success_rate", metric.Tags{})

	r.Add(1, 1) // 1 success out of 1
	r.Add(0, 1) // 0 success out of 1

	ir := store.GetRate("success_rate", metric.Tags{})
	require.NotNil(t, ir)
	assert.InDelta(t, 0.5, ir.Value(), 1e-9)
}

// AC-1.3.6: Rate.Add(x, 0) is a no-op (denominator 0 is ignored)
func TestRateDenominatorZeroIsNoOp(t *testing.T) {
	store := metric.NewStore()
	r := store.Rate("error_rate", metric.Tags{})

	r.Add(5, 0) // Should be ignored.
	r.Add(1, 2) // 1 out of 2.

	ir := store.GetRate("error_rate", metric.Tags{})
	require.NotNil(t, ir)
	assert.InDelta(t, 0.5, ir.Value(), 1e-9)
	assert.Equal(t, int64(1), ir.Numerator())
	assert.Equal(t, int64(2), ir.Denominator())
}

// AC-1.3.7: Gauge.Set(5.0) + Gauge.Add(-2.0) produces 3.0
func TestGaugeSetAndAdd(t *testing.T) {
	store := metric.NewStore()
	g := store.Gauge("active_vus", metric.Tags{})

	g.Set(5.0)
	g.Add(-2.0)

	ig := store.GetGauge("active_vus", metric.Tags{})
	require.NotNil(t, ig)
	assert.InDelta(t, 3.0, ig.Value(), 1e-9)
}

// AC-1.3.8: Tags {"a":"1"} and {"a":"2"} produce separate Counter instances for same name
func TestDifferentTagsProduceSeparateInstances(t *testing.T) {
	store := metric.NewStore()
	c1 := store.Counter("requests", metric.Tags{"a": "1"})
	c2 := store.Counter("requests", metric.Tags{"a": "2"})

	c1.Add(10)
	c2.Add(20)

	ic1 := store.GetCounter("requests", metric.Tags{"a": "1"})
	ic2 := store.GetCounter("requests", metric.Tags{"a": "2"})

	require.NotNil(t, ic1)
	require.NotNil(t, ic2)
	assert.Equal(t, int64(10), ic1.Value())
	assert.Equal(t, int64(20), ic2.Value())

	// Aggregated counter value should be the sum.
	assert.Equal(t, int64(30), store.AggregatedCounterValue("requests"))
}

// Additional: Counter.Add with concurrent goroutines produces exact total
func TestCounterConcurrentAdd(t *testing.T) {
	store := metric.NewStore()
	c := store.Counter("bytes_sent", metric.Tags{})

	const goroutines = 50
	const addsPerGoroutine = 500

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			for range addsPerGoroutine {
				c.Add(3)
			}
		}()
	}

	wg.Wait()

	ic := store.GetCounter("bytes_sent", metric.Tags{})
	require.NotNil(t, ic)
	assert.Equal(t, int64(goroutines*addsPerGoroutine*3), ic.Value())
}

// Additional: Gauge concurrent Set/Add does not race
func TestGaugeConcurrentAccess(t *testing.T) {
	store := metric.NewStore()
	g := store.Gauge("temperature", metric.Tags{})

	const goroutines = 50

	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	for range goroutines {
		go func() {
			defer wg.Done()
			g.Set(42.0)
		}()
		go func() {
			defer wg.Done()
			g.Add(1.0)
		}()
	}

	wg.Wait()
	// We just verify no race detector complaints — the final value is non-deterministic.
	ig := store.GetGauge("temperature", metric.Tags{})
	require.NotNil(t, ig)
	_ = ig.Value()
}

// Additional: Rate concurrent Add does not race
func TestRateConcurrentAdd(t *testing.T) {
	store := metric.NewStore()
	r := store.Rate("success", metric.Tags{})

	const goroutines = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			r.Add(1, 1)
		}()
	}

	wg.Wait()

	ir := store.GetRate("success", metric.Tags{})
	require.NotNil(t, ir)
	assert.InDelta(t, 1.0, ir.Value(), 1e-9)
	assert.Equal(t, int64(goroutines), ir.Numerator())
	assert.Equal(t, int64(goroutines), ir.Denominator())
}

// Additional: type collision in reverse (Duration then Counter)
func TestTypeCollisionDurationThenCounter(t *testing.T) {
	store := metric.NewStore()
	store.Duration("foo", metric.Tags{})

	assert.Panics(t, func() {
		store.Counter("foo", metric.Tags{})
	})
}

// Additional: type collision Gauge vs Rate
func TestTypeCollisionGaugeVsRate(t *testing.T) {
	store := metric.NewStore()
	store.Gauge("bar", metric.Tags{})

	assert.Panics(t, func() {
		store.Rate("bar", metric.Tags{})
	})
}

// Additional: same name same type different tags does NOT panic
func TestSameNameSameTypeDifferentTagsNoPanic(t *testing.T) {
	store := metric.NewStore()

	assert.NotPanics(t, func() {
		store.Counter("requests", metric.Tags{"env": "staging"})
		store.Counter("requests", metric.Tags{"env": "prod"})
	})
}

// Additional: tag order does not matter for identity
func TestTagOrderDoesNotMatter(t *testing.T) {
	store := metric.NewStore()
	c1 := store.Counter("requests", metric.Tags{"a": "1", "b": "2"})
	c2 := store.Counter("requests", metric.Tags{"b": "2", "a": "1"})

	c1.Inc()
	c2.Inc()

	ic := store.GetCounter("requests", metric.Tags{"a": "1", "b": "2"})
	require.NotNil(t, ic)
	assert.Equal(t, int64(2), ic.Value(), "tag order should not affect identity")
}

// Additional: empty tags and nil-equivalent tags are the same
func TestEmptyTagsIdentity(t *testing.T) {
	store := metric.NewStore()
	c1 := store.Counter("requests", metric.Tags{})
	c2 := store.Counter("requests", metric.Tags{})

	c1.Inc()
	c2.Inc()

	ic := store.GetCounter("requests", metric.Tags{})
	require.NotNil(t, ic)
	assert.Equal(t, int64(2), ic.Value())
}

// Additional: MergedHistogramSnapshot across different tags
func TestMergedHistogramAcrossTags(t *testing.T) {
	store := metric.NewStore()
	d1 := store.Duration("latency", metric.Tags{"endpoint": "/a"})
	d2 := store.Duration("latency", metric.Tags{"endpoint": "/b"})

	for i := 1; i <= 50; i++ {
		d1.Observe(time.Duration(i) * time.Millisecond)
	}
	for i := 51; i <= 100; i++ {
		d2.Observe(time.Duration(i) * time.Millisecond)
	}

	snap := store.MergedHistogramSnapshot("latency")
	assert.Equal(t, int64(100), snap.Count)
	assertDurationWithinTolerance(t, 50*time.Millisecond, snap.P50, 2*time.Millisecond, "merged p50")
}

// Additional: AggregatedRateValue across tags
func TestAggregatedRateAcrossTags(t *testing.T) {
	store := metric.NewStore()
	r1 := store.Rate("success", metric.Tags{"svc": "a"})
	r2 := store.Rate("success", metric.Tags{"svc": "b"})

	r1.Add(8, 10) // 80%
	r2.Add(9, 10) // 90%

	// Aggregated: 17/20 = 0.85
	assert.InDelta(t, 0.85, store.AggregatedRateValue("success"), 1e-9)
}

// Additional: Rate with no observations returns 0
func TestRateNoObservationsReturnsZero(t *testing.T) {
	store := metric.NewStore()
	_ = store.Rate("unused", metric.Tags{})

	ir := store.GetRate("unused", metric.Tags{})
	require.NotNil(t, ir)
	assert.InDelta(t, 0.0, ir.Value(), 1e-9)
}

// Additional: compile-time check that Store implements metric.Collector
func TestStoreImplementsMetricsCollector(t *testing.T) {
	var _ metric.Collector = (*metric.Store)(nil)
}

// helper
func assertDurationWithinTolerance(t *testing.T, expected, actual, tolerance time.Duration, label string) {
	t.Helper()
	diff := expected - actual
	if diff < 0 {
		diff = -diff
	}
	assert.LessOrEqual(t, diff, tolerance, "%s: expected ~%v, got %v (tolerance %v)", label, expected, actual, tolerance)
}
