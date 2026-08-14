// Package metric provides an in-memory metrics store implementing gtest.MetricsCollector.
// All metric handles are safe for concurrent use from multiple VU goroutines.
package metric

// Store is an in-memory metrics store composing Registry, Collector, and Aggregator.
// It is safe for concurrent use from multiple goroutines.
type Store struct {
	Registry
	*collector
	Aggregator
}

// NewStore creates a new empty metrics store composing Registry, Collector, and Aggregator.
func NewStore() *Store {
	reg := NewRegistry()
	coll := NewCollector(reg)
	agg := NewAggregator(coll)

	return &Store{
		Registry:   reg,
		collector:  coll,
		Aggregator: agg,
	}
}

// Compile-time interface satisfaction checks.
var (
	_ Collector  = (*Store)(nil)
	_ Registry   = (*Store)(nil)
	_ Aggregator = (*Store)(nil)
	_ Reader     = (*Store)(nil)
)
