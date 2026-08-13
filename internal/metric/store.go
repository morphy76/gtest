// Package metric provides an in-memory metrics store implementing gtest.MetricsCollector.
// All metric handles are safe for concurrent use from multiple VU goroutines.
package metric

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// metricType identifies the kind of metric registered under a name.
type metricType int

const (
	metricTypeCounter   metricType = iota
	metricTypeGauge
	metricTypeDuration
	metricTypeRate
)

// metricKey uniquely identifies a metric instance by name and sorted tags.
type metricKey struct {
	name    string
	tagsKey string
}

// makeKey produces a metricKey from a name and tags map.
// Tags are sorted by key to ensure deterministic identity.
func makeKey(name string, tags Tags) metricKey {
	return metricKey{name: name, tagsKey: sortedTagsKey(tags)}
}

// sortedTagsKey produces a canonical string representation of tags for identity comparison.
func sortedTagsKey(tags Tags) string {
	if len(tags) == 0 {
		return ""
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

// Store is an in-memory metrics store implementing Collector.
// It is safe for concurrent use from multiple goroutines.
type Store struct {
	mu sync.RWMutex

	// nameTypes tracks the registered metric type per name to detect type collisions.
	nameTypes map[string]metricType

	// counters, gauges, histograms, rates are keyed by metricKey.
	counters   map[metricKey]*counter
	gauges     map[metricKey]*gauge
	histograms map[metricKey]*histogram
	rates      map[metricKey]*rate
}

// NewStore creates a new empty metrics store.
func NewStore() *Store {
	return &Store{
		nameTypes:  make(map[string]metricType),
		counters:   make(map[metricKey]*counter),
		gauges:     make(map[metricKey]*gauge),
		histograms: make(map[metricKey]*histogram),
		rates:      make(map[metricKey]*rate),
	}
}

// Counter returns a monotonically increasing counter identified by name+tags.
// Panics if the same name was previously registered with a different metric type.
func (s *Store) Counter(name string, tags Tags) Counter {
	key := makeKey(name, tags)

	s.mu.RLock()
	if c, ok := s.counters[key]; ok {
		s.mu.RUnlock()
		return c
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check after acquiring write lock.
	if c, ok := s.counters[key]; ok {
		return c
	}

	s.checkTypeCollision(name, metricTypeCounter)

	c := &counter{}
	s.counters[key] = c
	return c
}

// Gauge returns an instantaneous value handle identified by name+tags.
// Panics if the same name was previously registered with a different metric type.
func (s *Store) Gauge(name string, tags Tags) Gauge {
	key := makeKey(name, tags)

	s.mu.RLock()
	if g, ok := s.gauges[key]; ok {
		s.mu.RUnlock()
		return g
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	if g, ok := s.gauges[key]; ok {
		return g
	}

	s.checkTypeCollision(name, metricTypeGauge)

	g := &gauge{}
	s.gauges[key] = g
	return g
}

// Duration returns a latency histogram identified by name+tags.
// Panics if the same name was previously registered with a different metric type.
func (s *Store) Duration(name string, tags Tags) Duration {
	key := makeKey(name, tags)

	s.mu.RLock()
	if h, ok := s.histograms[key]; ok {
		s.mu.RUnlock()
		return h
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	if h, ok := s.histograms[key]; ok {
		return h
	}

	s.checkTypeCollision(name, metricTypeDuration)

	h := newHistogram()
	s.histograms[key] = h
	return h
}

// Rate returns a ratio tracker identified by name+tags.
// Panics if the same name was previously registered with a different metric type.
func (s *Store) Rate(name string, tags Tags) Rate {
	key := makeKey(name, tags)

	s.mu.RLock()
	if r, ok := s.rates[key]; ok {
		s.mu.RUnlock()
		return r
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	if r, ok := s.rates[key]; ok {
		return r
	}

	s.checkTypeCollision(name, metricTypeRate)

	r := &rate{}
	s.rates[key] = r
	return r
}

// checkTypeCollision panics if a metric name is already registered with a different type.
// Must be called under write lock.
func (s *Store) checkTypeCollision(name string, mt metricType) {
	if existing, ok := s.nameTypes[name]; ok && existing != mt {
		panic(fmt.Sprintf("gtest: metric %q already registered as %s, cannot register as %s",
			name, metricTypeName(existing), metricTypeName(mt)))
	}
	s.nameTypes[name] = mt
}

// metricTypeName returns a human-readable name for a metric type.
func metricTypeName(mt metricType) string {
	switch mt {
	case metricTypeCounter:
		return "Counter"
	case metricTypeGauge:
		return "Gauge"
	case metricTypeDuration:
		return "Duration"
	case metricTypeRate:
		return "Rate"
	default:
		return "unknown"
	}
}

// GetCounter returns the internal counter for the given key, or nil if not found.
// Used by SLA evaluation and reporting.
func (s *Store) GetCounter(name string, tags Tags) *counter {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.counters[makeKey(name, tags)]
}

// GetGauge returns the internal gauge for the given key, or nil if not found.
func (s *Store) GetGauge(name string, tags Tags) *gauge {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.gauges[makeKey(name, tags)]
}

// GetHistogram returns the internal histogram for the given key, or nil if not found.
func (s *Store) GetHistogram(name string, tags Tags) *histogram {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.histograms[makeKey(name, tags)]
}

// GetRate returns the internal rate for the given key, or nil if not found.
func (s *Store) GetRate(name string, tags Tags) *rate {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.rates[makeKey(name, tags)]
}

// MetricType returns the registered type for the given name, or false if unregistered.
func (s *Store) MetricType(name string) (metricType, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	mt, ok := s.nameTypes[name]
	return mt, ok
}

// CounterNames returns all registered counter names sorted alphabetically.
func (s *Store) CounterNames() []string {
	return s.namesByType(metricTypeCounter)
}

// GaugeNames returns all registered gauge names sorted alphabetically.
func (s *Store) GaugeNames() []string {
	return s.namesByType(metricTypeGauge)
}

// HistogramNames returns all registered histogram/duration names sorted alphabetically.
func (s *Store) HistogramNames() []string {
	return s.namesByType(metricTypeDuration)
}

// RateNames returns all registered rate names sorted alphabetically.
func (s *Store) RateNames() []string {
	return s.namesByType(metricTypeRate)
}

// namesByType returns sorted unique names for a given metric type.
func (s *Store) namesByType(mt metricType) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	seen := make(map[string]bool)
	for name, t := range s.nameTypes {
		if t == mt {
			seen[name] = true
		}
	}

	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// MergedHistogramSnapshot returns a merged snapshot of all histogram instances
// sharing the given metric name, across all tag combinations.
// Returns a zero snapshot if no histograms exist for the name.
func (s *Store) MergedHistogramSnapshot(name string) HistogramSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var all []*histogram
	for key, h := range s.histograms {
		if key.name == name {
			all = append(all, h)
		}
	}

	if len(all) == 0 {
		return HistogramSnapshot{}
	}

	// Merge all histograms for this name.
	if len(all) == 1 {
		return all[0].Snapshot()
	}

	// Collect shared + per-VU histograms from every tag variant.
	merged := newHistogram()
	for _, h := range all {
		h.mu.Lock()
		if h.shared != nil && h.shared.TotalCount() > 0 {
			merged.histograms = append(merged.histograms, h.shared)
		}
		for _, sub := range h.histograms {
			merged.histograms = append(merged.histograms, sub)
		}
		h.mu.Unlock()
	}
	return merged.Snapshot()
}

// AggregatedCounterValue returns the sum of all counter values sharing the given name
// across all tag combinations.
func (s *Store) AggregatedCounterValue(name string) int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var total int64
	for key, c := range s.counters {
		if key.name == name {
			total += c.Value()
		}
	}
	return total
}

// AggregatedRateValue returns the aggregated rate across all tag combinations for the given name.
// Returns 0 if no rate data exists.
func (s *Store) AggregatedRateValue(name string) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var totalNum, totalDen int64
	for key, r := range s.rates {
		if key.name == name {
			totalNum += r.Numerator()
			totalDen += r.Denominator()
		}
	}

	if totalDen == 0 {
		return 0
	}
	return float64(totalNum) / float64(totalDen)
}

// RateData returns the aggregated rate and a boolean indicating whether any observations exist.
func (s *Store) RateData(name string) (float64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var totalNum, totalDen int64
	for key, r := range s.rates {
		if key.name == name {
			totalNum += r.Numerator()
			totalDen += r.Denominator()
		}
	}

	if totalDen == 0 {
		return 0, false
	}
	return float64(totalNum) / float64(totalDen), true
}

// LastGaugeValue returns the last recorded value for the given gauge name.
// If multiple tag variants exist, returns the value of an arbitrary one.
// Returns 0 if no gauge data exists.
func (s *Store) LastGaugeValue(name string) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for key, g := range s.gauges {
		if key.name == name {
			return g.Value()
		}
	}
	return 0
}

// Compile-time interface satisfaction check.
var _ Collector = (*Store)(nil)
