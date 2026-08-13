package gtest

import (
	"context"
	"time"
)

// ScenarioContext is the scoped execution context passed to every VU hook.
// It embeds context.Context so it can be passed directly to stdlib calls
// (http.NewRequestWithContext, etc.).
type ScenarioContext interface {
	context.Context

	// VUID returns the 1-based unique identifier for this Virtual User goroutine.
	VUID() int64

	// Iteration returns the 0-based iteration count for the current VU.
	// Always 0 inside PreTest and AfterTest hooks.
	Iteration() int64

	// ScenarioName returns the active scenario name as declared in gtest.yaml.
	ScenarioName() string

	// Param retrieves a string value from the scenario's params map.
	// Returns "" if key is absent.
	Param(key string) string

	// ParamInt retrieves a params value parsed as int.
	// Returns defaultValue if key is absent or value cannot be parsed.
	ParamInt(key string, defaultValue int) int

	// ParamDuration retrieves a params value parsed as time.Duration.
	// Returns defaultValue if key is absent or value cannot be parsed.
	ParamDuration(key string, defaultValue time.Duration) time.Duration

	// GlobalState retrieves a value from the map returned by the Setup hook.
	// The map is read-only and shared across all VUs; callers must not mutate it.
	// Returns nil if key is absent or Setup was not provided.
	GlobalState(key string) any

	// Log returns the VU-scoped logger pre-enriched with scenario, vu_id, iteration fields.
	Log() Logger

	// Metrics returns the shared metrics collector.
	Metrics() MetricsCollector
}
