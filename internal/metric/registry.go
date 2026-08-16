package metric

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
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
	return fmt.Sprintf("vuhive: metric %q already registered as %s, cannot register as %s",
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
	mu      sync.Mutex
	readMap atomic.Pointer[map[string]MetricType]
}

// NewRegistry creates a new thread-safe metric Registry with built-in metrics registered.
func NewRegistry() Registry {
	r := &registry{}
	initial := map[string]MetricType{
		MetricChecksPassed: MetricTypeCounter,
		MetricChecksFailed: MetricTypeCounter,
	}
	r.readMap.Store(&initial)
	return r
}

// Register registers a metric name with a given MetricType.
// Returns an *ErrTypeCollision if the name was already registered with a different type.
func (r *registry) Register(name string, mt MetricType) error {
	m := r.readMap.Load()
	if m != nil {
		if existing, ok := (*m)[name]; ok {
			if existing != mt {
				return &ErrTypeCollision{
					Name:     name,
					Existing: existing,
					New:      mt,
				}
			}
			return nil
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	m = r.readMap.Load()
	if m != nil {
		if existing, ok := (*m)[name]; ok {
			if existing != mt {
				return &ErrTypeCollision{
					Name:     name,
					Existing: existing,
					New:      mt,
				}
			}
			return nil
		}
	}

	var newMap map[string]MetricType
	if m != nil {
		newMap = make(map[string]MetricType, len(*m)+1)
		for k, v := range *m {
			newMap[k] = v
		}
	} else {
		newMap = make(map[string]MetricType, 1)
	}
	newMap[name] = mt
	r.readMap.Store(&newMap)
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
// 100% lock-free via atomic pointer load.
func (r *registry) MetricType(name string) (MetricType, bool) {
	m := r.readMap.Load()
	if m == nil {
		return 0, false
	}
	mt, ok := (*m)[name]
	return mt, ok
}

// NamesByType returns sorted unique names for a given metric type.
func (r *registry) NamesByType(mt MetricType) []string {
	m := r.readMap.Load()
	if m == nil {
		return nil
	}

	var names []string
	for name, t := range *m {
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
