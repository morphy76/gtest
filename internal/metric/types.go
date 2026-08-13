package metric

import "time"

// Tags is a set of key-value labels attached to a metric observation.
type Tags map[string]string

// Collector is the internal interface for recording metrics.
type Collector interface {
	Counter(name string, tags Tags) Counter
	Gauge(name string, tags Tags) Gauge
	Duration(name string, tags Tags) Duration
	Rate(name string, tags Tags) Rate
}

// Counter is a monotonically increasing counter.
type Counter interface {
	Inc()
	Add(delta int64)
}

// Gauge is an instantaneous value handle.
type Gauge interface {
	Set(value float64)
	Add(delta float64)
}

// Duration records latency samples into an HDR histogram.
type Duration interface {
	Observe(d time.Duration)
}

// Rate tracks a ratio of numerator to denominator.
type Rate interface {
	Add(numerator, denominator int64)
}
