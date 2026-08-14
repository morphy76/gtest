package metric

import (
	"errors"
	"fmt"
	"sort"
	"sync"
)

// MetricType identifies the kind of metric registered under a name.
type MetricType int

const (
	MetricTypeCounter MetricType = iota
	MetricTypeGauge
	MetricTypeDuration
	MetricTypeRate
)

// String returns the human-readable name of the metric type.
func (mt MetricType) String() string {
	switch mt {
	case MetricTypeCounter:
		return "Counter"
	case MetricTypeGauge:
		return "Gauge"
	case MetricTypeDuration:
		return "Duration"
	case MetricTypeRate:
		return "Rate"
	default:
		return "unknown"
	}
}

// ErrTypeCollision is returned when a metric name is already registered with a different type.
type ErrTypeCollision struct {
	Name     string
	Existing MetricType
	New      MetricType
}

func (e *ErrTypeCollision) Error() string {
	return fmt.Sprintf("gtest: metric %q already registered as %s, cannot register as %s",
		e.Name, e.Existing, e.New)
}

// Registry tracks registered metric types and detects type collisions with thread safety.
type Registry interface {
	Register(name string, mt MetricType) error
	MustRegister(name string, mt MetricType)
	RegisterMetric(name string, m Metric) error
	MetricType(name string) (MetricType, bool)
	NamesByType(mt MetricType) []string
	CounterNames() []string
	GaugeNames() []string
	HistogramNames() []string
	RateNames() []string
}

type registry struct {
	mu        sync.RWMutex
	nameTypes map[string]MetricType
}

// NewRegistry creates a new thread-safe metric Registry with built-in metrics registered.
func NewRegistry() Registry {
	r := &registry{
		nameTypes: make(map[string]MetricType),
	}
	r.nameTypes["gtest.checks.passed"] = MetricTypeCounter
	r.nameTypes["gtest.checks.failed"] = MetricTypeCounter
	return r
}

// Register registers a metric name with a given MetricType.
// Returns an *ErrTypeCollision if the name was already registered with a different type.
func (r *registry) Register(name string, mt MetricType) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, ok := r.nameTypes[name]; ok && existing != mt {
		return &ErrTypeCollision{
			Name:     name,
			Existing: existing,
			New:      mt,
		}
	}
	r.nameTypes[name] = mt
	return nil
}

// MustRegister registers a metric name or panics if there is a type collision.
func (r *registry) MustRegister(name string, mt MetricType) {
	if err := r.Register(name, mt); err != nil {
		panic(err.Error())
	}
}

// RegisterMetric registers a metric instance by extracting its type.
func (r *registry) RegisterMetric(name string, m Metric) error {
	if m == nil {
		return errors.New("cannot register nil metric")
	}
	return r.Register(name, m.Type())
}

// MetricType returns the registered type for the given name, or false if unregistered.
func (r *registry) MetricType(name string) (MetricType, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	mt, ok := r.nameTypes[name]
	return mt, ok
}

// NamesByType returns sorted unique names for a given metric type.
func (r *registry) NamesByType(mt MetricType) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var names []string
	for name, t := range r.nameTypes {
		if t == mt {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// CounterNames returns all registered counter names sorted alphabetically.
func (r *registry) CounterNames() []string {
	return r.NamesByType(MetricTypeCounter)
}

// GaugeNames returns all registered gauge names sorted alphabetically.
func (r *registry) GaugeNames() []string {
	return r.NamesByType(MetricTypeGauge)
}

// HistogramNames returns all registered histogram/duration names sorted alphabetically.
func (r *registry) HistogramNames() []string {
	return r.NamesByType(MetricTypeDuration)
}

// RateNames returns all registered rate names sorted alphabetically.
func (r *registry) RateNames() []string {
	return r.NamesByType(MetricTypeRate)
}

// Compile-time check
var _ Registry = (*registry)(nil)
