package gtest

import "context"

// SetupHook is called once before any VU is spawned.
// It returns a global state map shared (read-only) with all VUs via ScenarioContext.GlobalState().
// A non-nil error aborts the test run immediately.
type SetupHook func(ctx context.Context) (state map[string]any, err error)

// PreTestHook is called once per VU goroutine before its iteration loop begins.
// A non-nil error skips RunVU but still guarantees AfterTestHook execution for that VU.
type PreTestHook func(ctx ScenarioContext) error

// VURunnerHook is called repeatedly in a loop for each VU during the run_period.
// Each call receives a fresh child context with the vu_timeout deadline applied.
// A non-nil error or panic is caught, logged, and counted; the loop continues.
type VURunnerHook func(ctx ScenarioContext) error

// AfterTestHook is called once per VU after the run loop ends (or after PreTest failure).
// It runs in a deferred call, so it executes even if RunVU panicked.
// A non-nil error is logged but does not affect the overall pass/fail verdict.
type AfterTestHook func(ctx ScenarioContext) error

// TeardownHook is called once after all VU goroutines have exited.
// It receives the same global state produced by Setup.
// A non-nil error is logged but does not affect the overall pass/fail verdict.
type TeardownHook func(ctx context.Context, state map[string]any) error

// Scenario groups all lifecycle hooks for a named test scenario.
// Only RunVU is required. All other hooks are optional and may be nil.
type Scenario struct {
	Setup     SetupHook     // optional
	PreTest   PreTestHook   // optional
	RunVU     VURunnerHook  // required
	AfterTest AfterTestHook // optional
	Teardown  TeardownHook  // optional
}
