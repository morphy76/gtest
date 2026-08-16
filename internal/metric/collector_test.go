package metric_test

import (
	"sync"
	"testing"
	"time"

	"github.com/morphy76/vuhive/internal/metric"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollector_InterfaceSatisfaction(t *testing.T) {
	var _ metric.Collector = metric.NewCollector(metric.NewRegistry())
}

func TestCollector_IngestionHandles(t *testing.T) {
	reg := metric.NewRegistry()
	coll := metric.NewCollector(reg)

	// Counter
	c := coll.Counter("http_reqs", metric.Tags{"method": "GET"})
	require.NotNil(t, c)
	c.Inc()
	c.Add(4)

	gc := coll.GetCounter("http_reqs", metric.Tags{"method": "GET"})
	require.NotNil(t, gc)
	assert.Equal(t, int64(5), gc.Value())

	// Gauge
	g := coll.Gauge("active_users", metric.Tags{"role": "admin"})
	require.NotNil(t, g)
	g.Set(10.0)
	g.Add(2.5)

	gg := coll.GetGauge("active_users", metric.Tags{"role": "admin"})
	require.NotNil(t, gg)
	assert.InDelta(t, 12.5, gg.Value(), 1e-9)

	// Duration
	d := coll.Duration("resp_time", metric.Tags{"path": "/api"})
	require.NotNil(t, d)
	d.Observe(15 * time.Millisecond)

	gd := coll.GetHistogram("resp_time", metric.Tags{"path": "/api"})
	require.NotNil(t, gd)
	assert.Equal(t, int64(1), gd.Snapshot().Count)

	// Rate
	r := coll.Rate("success_ratio", metric.Tags{"service": "auth"})
	require.NotNil(t, r)
	r.Add(9, 10)

	gr := coll.GetRate("success_ratio", metric.Tags{"service": "auth"})
	require.NotNil(t, gr)
	assert.InDelta(t, 0.9, gr.Value(), 1e-9)
}

func TestCollector_DoubleCheckedLocking_SameInstance(t *testing.T) {
	reg := metric.NewRegistry()
	coll := metric.NewCollector(reg)

	tags := metric.Tags{"region": "us-east-1"}
	c1 := coll.Counter("requests", tags)
	c2 := coll.Counter("requests", tags)

	assert.Same(t, c1, c2, "Double-checked locking must return identical instance for same name and tags")

	c1.Add(10)
	assert.Equal(t, int64(10), coll.GetCounter("requests", tags).Value())
}

func TestCollector_TypeCollisionEnforcedViaRegistry(t *testing.T) {
	reg := metric.NewRegistry()
	coll := metric.NewCollector(reg)

	coll.Counter("latency", metric.Tags{})

	assert.Panics(t, func() {
		coll.Duration("latency", metric.Tags{})
	})
}

func TestCollector_ConcurrentIngestion(t *testing.T) {
	reg := metric.NewRegistry()
	coll := metric.NewCollector(reg)

	const goroutines = 50
	const incs = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < incs; j++ {
				c := coll.Counter("concurrent_counter", metric.Tags{"env": "test"})
				c.Inc()
			}
		}()
	}

	wg.Wait()
	gc := coll.GetCounter("concurrent_counter", metric.Tags{"env": "test"})
	require.NotNil(t, gc)
	assert.Equal(t, int64(goroutines*incs), gc.Value())
}

func TestMetric_InterfaceImplementations(t *testing.T) {
	reg := metric.NewRegistry()
	coll := metric.NewCollector(reg)

	_ = coll.Counter("c", metric.Tags{})
	_ = coll.Gauge("g", metric.Tags{})
	_ = coll.Duration("d", metric.Tags{})
	_ = coll.Rate("r", metric.Tags{})

	c := coll.GetCounter("c", metric.Tags{})
	g := coll.GetGauge("g", metric.Tags{})
	h := coll.GetHistogram("d", metric.Tags{})
	r := coll.GetRate("r", metric.Tags{})

	var mc metric.Metric = c
	var mg metric.Metric = g
	var mh metric.Metric = h
	var mr metric.Metric = r

	assert.Equal(t, metric.MetricTypeCounter, mc.Type())
	assert.Equal(t, metric.MetricTypeGauge, mg.Type())
	assert.Equal(t, metric.MetricTypeDuration, mh.Type())
	assert.Equal(t, metric.MetricTypeRate, mr.Type())
}
