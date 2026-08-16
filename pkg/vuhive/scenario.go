package vuhive

// SetupHook is called once before any VU is spawned.
// It returns a global state map shared (read-only) with all VUs via VUContext.GlobalState().
// A non-nil error aborts the test run immediately.
type SetupHook func(ctx SetupContext) (state map[string]any, err error)

// PreTestHook is called once per VU goroutine before its iteration loop begins.
// A non-nil error skips RunVU but still guarantees AfterTestHook execution for that VU.
type PreTestHook func(ctx VUContext) error

// VURunnerHook is called repeatedly in a loop for each VU during the run_period.
// Each call receives a fresh child context with the vu_timeout deadline applied.
// A non-nil error or panic is caught, logged, and counted; the loop continues.
type VURunnerHook func(ctx VUContext) error

// AfterTestHook is called once per VU after the run loop ends (or after PreTest failure).
// It runs in a deferred call, so it executes even if RunVU panicked.
// A non-nil error is logged but does not affect the overall pass/fail verdict.
type AfterTestHook func(ctx VUContext) error

// TeardownHook is called once after all VU goroutines have exited.
// It receives the same global state produced by Setup.
// A non-nil error is logged but does not affect the overall pass/fail verdict.
type TeardownHook func(ctx TeardownContext, state map[string]any) error

// SummaryHook is called after test execution and report generation.
// It receives the execution context and complete execution summary data.
type SummaryHook func(ctx SummaryContext, summary SummaryData) error

// Scenario groups all lifecycle hooks for a named test scenario.
// Only RunVU is required. All other hooks are optional and may be nil.
type Scenario struct {
	// Setup is an optional hook executed once sequentially before any VU is spawned.
	// It can return a global state map shared across all VUs.
	Setup SetupHook

	// PreTest is an optional hook executed once per VU goroutine before starting iterations.
	PreTest PreTestHook

	// RunVU is the mandatory per-iteration hook executed repeatedly by each VU during the run period.
	RunVU VURunnerHook

	// AfterTest is an optional hook executed once per VU in a deferred call after iterations complete.
	AfterTest AfterTestHook

	// Teardown is an optional hook executed once sequentially after all VUs have terminated.
	Teardown TeardownHook

	// HandleSummary is an optional hook executed post-test with structured execution summary data.
	HandleSummary SummaryHook
}
