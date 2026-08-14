package gtest

import (
	"context"
	"time"
)

// ExecutionIdentity provides execution identity attributes (VU ID, iteration index, and scenario name).
type ExecutionIdentity interface {
	// VUID returns the unique 1-indexed identifier of the calling Virtual User.
	VUID() int64

	// Iteration returns the current zero-indexed iteration number of the VU.
	Iteration() int64

	// ScenarioName returns the name of the executing scenario.
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
	// GlobalState returns a value from the scenario's global state returned by Setup.
	// Returns nil if key is not found in the global state map.
	GlobalState(key string) any
}

// ObservabilityProvider provides access to structured logging and metric collection.
type ObservabilityProvider interface {
	// Log returns the structured logger scoped to the current VU execution.
	Log() Logger

	// Metrics returns the thread-safe metrics collector handle for recording telemetry.
	Metrics() MetricsCollector
}

// WorkflowController provides workflow execution controls such as delays and inline assertions.
type WorkflowController interface {
	// Sleep pauses execution for the configured think time or an explicitly provided duration.
	// When called with no arguments (Sleep()), it pauses for the think_time duration generated
	// by the scenario's configured think time strategy.
	// When called with an explicit duration (Sleep(d)), it pauses for that exact duration.
	// Respects context cancellation and returns ctx.Err() if cancelled during sleep.
	Sleep(d ...time.Duration) error

	// Check evaluates an inline assertion function fn and records the result under the given name.
	// Increments gtest.checks.passed if fn returns an empty string, or gtest.checks.failed if fn
	// returns a failure reason. Returns true if the check passed.
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
