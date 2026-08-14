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
	Param(key string) string
	ParamInt(key string, defaultValue int) int
	ParamDuration(key string, defaultValue time.Duration) time.Duration
}

// StateProvider provides read-only access to global scenario state returned by Setup.
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
