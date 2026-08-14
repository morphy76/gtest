package gtest

import (
	"context"
	"time"
)

// ExecutionIdentity provides execution identity attributes (VU ID, iteration index, and scenario name).
type ExecutionIdentity interface {
	VUID() int64
	Iteration() int64
	ScenarioName() string
}

// ConfigProvider provides access to scenario configuration parameters.
type ConfigProvider interface {
	// Param retrieves a string value from the scenario's params map. Returns "" if absent.
	Param(key string) string

	// ParamInt retrieves a params value parsed as int. Returns defaultValue if key is absent.
	// If the value is present but cannot be parsed as an integer, a Warn-level log is emitted
	// and defaultValue is returned.
	ParamInt(key string, defaultValue int) int

	// ParamDuration retrieves a params value parsed as time.Duration. Returns defaultValue if key is absent.
	// If the value is present but cannot be parsed as a duration, a Warn-level log is emitted
	// and defaultValue is returned.
	ParamDuration(key string, defaultValue time.Duration) time.Duration
}

// StateProvider provides read-only access to global scenario state returned by Setup.
// Note: Global state is shallow-copied by the framework. Complex or nested mutable values
// (such as slices, maps, or pointers) must be immutable or protected with explicit synchronization.
type StateProvider interface {
	GlobalState(key string) any
}

// ObservabilityProvider provides access to structured logging and metric collection.
type ObservabilityProvider interface {
	Log() Logger
	Metrics() MetricsCollector
}

// WorkflowController provides workflow execution controls such as delays and inline assertions.
type WorkflowController interface {
	Sleep(d ...time.Duration) error
	Check(name string, fn CheckFunc) bool
}

// ScenarioContext is the scoped execution context passed to every VU hook.
// It embeds context.Context and composes focused capability interfaces.
type ScenarioContext interface {
	context.Context
	ExecutionIdentity
	ConfigProvider
	StateProvider
	ObservabilityProvider
	WorkflowController
}
