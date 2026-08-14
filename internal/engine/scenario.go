package engine

import (
	"fmt"

	"github.com/morphy76/gtest/internal/report"
)

// SetupHook is called once before any VU is spawned.
// It returns a global state map shared (read-only) with all VUs via VUContext.GlobalState().
type SetupHook func(ctx SetupContext) (state map[string]any, err error)

// PreTestHook is called once per VU goroutine before its iteration loop begins.
type PreTestHook func(ctx VUContext) error

// VURunnerHook is called repeatedly in a loop for each VU during the run_period.
type VURunnerHook func(ctx VUContext) error

// AfterTestHook is called once per VU after the run loop ends (or after PreTest failure).
type AfterTestHook func(ctx VUContext) error

// TeardownHook is called once after all VU goroutines have exited.
// It receives the same global state produced by Setup.
// A non-nil error is logged but does not affect the overall pass/fail verdict.
type TeardownHook func(ctx TeardownContext, state map[string]any) error

// SummaryHook is called after test execution and report generation.
// It receives the execution context and complete execution summary data.
type SummaryHook func(ctx SummaryContext, summary report.SummaryData) error

// CheckFunc is a function that returns an empty string on pass, or a non-empty error message on failure.
type CheckFunc func() string

// Scenario groups all lifecycle hooks for a named test scenario.
type Scenario struct {
	Setup         SetupHook
	PreTest       PreTestHook
	RunVU         VURunnerHook
	AfterTest     AfterTestHook
	Teardown      TeardownHook
	HandleSummary SummaryHook
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
