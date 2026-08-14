package metric

import (
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

// metricKey uniquely identifies a metric instance by name and sorted tags.
type metricKey struct {
	name    string
	tagsKey string
}

// makeKey produces a metricKey from a name and tags map.
// Tags are sorted by key to ensure deterministic identity.
func makeKey(name string, tags Tags) metricKey {
	if len(tags) == 0 {
		return metricKey{name: name, tagsKey: ""}
	}
	if len(tags) == 1 {
		for k, v := range tags {
			return metricKey{name: name, tagsKey: k + "=" + v}
		}
	}
	return metricKey{name: name, tagsKey: sortedTagsKey(tags)}
}

// sortedTagsKey produces a canonical string representation of tags for identity comparison.
func sortedTagsKey(tags Tags) string {
	if len(tags) == 0 {
		return ""
	}
	if len(tags) == 1 {
		for k, v := range tags {
			return k + "=" + v
		}
	}
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(tags[k])
	}
	return b.String()
}

// syncStorage is a thread-safe map leveraging copy-on-write atomic pointers for 100% lock-free reads.
type syncStorage[K comparable, V any] struct {
	writeMu sync.Mutex
	readMap atomic.Pointer[map[K]V]
}

func newSyncStorage[K comparable, V any]() *syncStorage[K, V] {
	s := &syncStorage[K, V]{}
	empty := make(map[K]V)
	s.readMap.Store(&empty)
	return s
}

// GetOrCreate retrieves an existing item or creates a new one using factory.
// Reads are 100% lock-free via atomic pointer load.
func (s *syncStorage[K, V]) GetOrCreate(key K, beforeCreate func(), factory func() V) V {
	m := s.readMap.Load()
	if m != nil {
		if val, ok := (*m)[key]; ok {
			return val
		}
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	m = s.readMap.Load()
	if m != nil {
		if val, ok := (*m)[key]; ok {
			return val
		}
	}

	if beforeCreate != nil {
		beforeCreate()
	}

	val := factory()
	var newMap map[K]V
	if m != nil {
		newMap = make(map[K]V, len(*m)+1)
		for k, v := range *m {
			newMap[k] = v
		}
	} else {
		newMap = make(map[K]V, 1)
	}
	newMap[key] = val
	s.readMap.Store(&newMap)
	return val
}

// Get returns the item for key, or zero value and false if not found.
func (s *syncStorage[K, V]) Get(key K) (V, bool) {
	m := s.readMap.Load()
	if m == nil {
		var zero V
		return zero, false
	}
	val, ok := (*m)[key]
	return val, ok
}

// ForEach iterates over all items in the storage in a lock-free snapshot.
func (s *syncStorage[K, V]) ForEach(fn func(key K, val V)) {
	m := s.readMap.Load()
	if m == nil {
		return
	}
	for k, v := range *m {
		fn(k, v)
	}
}

// collector handles metric ingestion and per-handle thread-safe storage.
type collector struct {
	registry   Registry
	counters   *syncStorage[metricKey, *counter]
	gauges     *syncStorage[metricKey, *gauge]
	histograms *syncStorage[metricKey, *histogram]
	rates      *syncStorage[metricKey, *rate]
}

// NewCollector creates a new Collector backed by the specified Registry.
// If registry is nil, a default registry is used.
func NewCollector(registry Registry) *collector {
	if registry == nil {
		registry = NewRegistry()
	}
	return &collector{
		registry:   registry,
		counters:   newSyncStorage[metricKey, *counter](),
		gauges:     newSyncStorage[metricKey, *gauge](),
		histograms: newSyncStorage[metricKey, *histogram](),
		rates:      newSyncStorage[metricKey, *rate](),
	}
}

// Counter returns a monotonically increasing counter identified by name+tags.
// Panics if the same name was previously registered with a different metric type.
func (c *collector) Counter(name string, tags Tags) Counter {
	return c.counters.GetOrCreate(
		makeKey(name, tags),
		func() { c.registry.MustRegister(name, MetricTypeCounter) },
		func() *counter { return &counter{} },
	)
}

// Gauge returns an instantaneous value handle identified by name+tags.
// Panics if the same name was previously registered with a different metric type.
func (c *collector) Gauge(name string, tags Tags) Gauge {
	return c.gauges.GetOrCreate(
		makeKey(name, tags),
		func() { c.registry.MustRegister(name, MetricTypeGauge) },
		func() *gauge { return &gauge{} },
	)
}

// Duration returns a latency histogram identified by name+tags.
// Panics if the same name was previously registered with a different metric type.
func (c *collector) Duration(name string, tags Tags) Duration {
	return c.histograms.GetOrCreate(
		makeKey(name, tags),
		func() { c.registry.MustRegister(name, MetricTypeDuration) },
		func() *histogram { return newHistogram() },
	)
}

// Rate returns a ratio tracker identified by name+tags.
// Panics if the same name was previously registered with a different metric type.
func (c *collector) Rate(name string, tags Tags) Rate {
	return c.rates.GetOrCreate(
		makeKey(name, tags),
		func() { c.registry.MustRegister(name, MetricTypeRate) },
		func() *rate { return &rate{} },
	)
}

// GetCounter returns the internal counter for the given key, or nil if not found.
func (c *collector) GetCounter(name string, tags Tags) *counter {
	val, _ := c.counters.Get(makeKey(name, tags))
	return val
}

// GetGauge returns the internal gauge for the given key, or nil if not found.
func (c *collector) GetGauge(name string, tags Tags) *gauge {
	val, _ := c.gauges.Get(makeKey(name, tags))
	return val
}

// GetHistogram returns the internal histogram for the given key, or nil if not found.
func (c *collector) GetHistogram(name string, tags Tags) *histogram {
	val, _ := c.histograms.Get(makeKey(name, tags))
	return val
}

// GetRate returns the internal rate for the given key, or nil if not found.
func (c *collector) GetRate(name string, tags Tags) *rate {
	val, _ := c.rates.Get(makeKey(name, tags))
	return val
}

// ForEachCounter iterates over all recorded counters.
func (c *collector) ForEachCounter(fn func(key metricKey, c *counter)) {
	c.counters.ForEach(fn)
}

// ForEachGauge iterates over all recorded gauges.
func (c *collector) ForEachGauge(fn func(key metricKey, g *gauge)) {
	c.gauges.ForEach(fn)
}

// ForEachHistogram iterates over all recorded histograms.
func (c *collector) ForEachHistogram(fn func(key metricKey, h *histogram)) {
	c.histograms.ForEach(fn)
}

// ForEachRate iterates over all recorded rates.
func (c *collector) ForEachRate(fn func(key metricKey, r *rate)) {
	c.rates.ForEach(fn)
}

// Compile-time checks.
var _ Collector = (*collector)(nil)
var _ MetricProvider = (*collector)(nil)
