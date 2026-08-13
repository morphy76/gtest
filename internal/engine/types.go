package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/morphy76/gtest/internal/log"
	"github.com/morphy76/gtest/internal/metric"
)

// SetupHook is called once before any VU is spawned.
// It returns a global state map shared (read-only) with all VUs via ScenarioContext.GlobalState().
type SetupHook func(ctx ScenarioContext) (state map[string]any, err error)

// PreTestHook is called once per VU goroutine before its iteration loop begins.
type PreTestHook func(ctx ScenarioContext) error

// VURunnerHook is called repeatedly in a loop for each VU during the run_period.
type VURunnerHook func(ctx ScenarioContext) error

// AfterTestHook is called once per VU after the run loop ends (or after PreTest failure).
type AfterTestHook func(ctx ScenarioContext) error

// TeardownHook is called once after all VU goroutines have exited.
type TeardownHook func(ctx ScenarioContext, state map[string]any) error

// Scenario groups all lifecycle hooks for a named test scenario.
type Scenario struct {
	Setup     SetupHook
	PreTest   PreTestHook
	RunVU     VURunnerHook
	AfterTest AfterTestHook
	Teardown  TeardownHook
}

// ScenarioContext is the scoped execution context passed to every VU hook.
type ScenarioContext interface {
	context.Context
	VUID() int64
	Iteration() int64
	ScenarioName() string
	Param(key string) string
	ParamInt(key string, defaultValue int) int
	ParamDuration(key string, defaultValue time.Duration) time.Duration
	GlobalState(key string) any
	Log() log.Logger
	Metrics() metric.Collector
}

// SetupError wraps an error returned by the Setup hook.
type SetupError struct {
	Err error
}

func (e *SetupError) Error() string {
	return fmt.Sprintf("gtest: setup hook failed: %s", e.Err)
}

func (e *SetupError) Unwrap() error {
	return e.Err
}
