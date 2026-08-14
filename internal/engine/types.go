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
// It receives the same global state produced by Setup.
// A non-nil error is logged but does not affect the overall pass/fail verdict.
type TeardownHook func(ctx ScenarioContext, state map[string]any) error

// SummaryHook is called after test execution and report generation.
// It receives the complete execution summary data.
type SummaryHook func(ctx context.Context, summary SummaryData) error

// CheckFunc is a function that returns an empty string on pass, or a non-empty error message on failure.
type CheckFunc func() string

// CheckSummary represents aggregated results for a named inline check.
type CheckSummary struct {
	Name    string  `json:"name"`
	Passed  int64   `json:"passed"`
	Failed  int64   `json:"failed"`
	Total   int64   `json:"total"`
	PassPct float64 `json:"pass_pct"`
}

// MetricSummary represents a metric entry in the execution summary.
type MetricSummary struct {
	Name   string            `json:"name"`
	Type   string            `json:"type"`
	Tags   map[string]string `json:"tags,omitempty"`
	Count  int64             `json:"count,omitempty"`
	Value  float64           `json:"value,omitempty"`
	Rate   float64           `json:"rate,omitempty"`
	Min    time.Duration     `json:"min,omitempty"`
	Mean   time.Duration     `json:"mean,omitempty"`
	P50    time.Duration     `json:"p50,omitempty"`
	P90    time.Duration     `json:"p90,omitempty"`
	P95    time.Duration     `json:"p95,omitempty"`
	P99    time.Duration     `json:"p99,omitempty"`
	Max    time.Duration     `json:"max,omitempty"`
}

// ThresholdSummary represents the outcome of a single SLA threshold evaluation.
type ThresholdSummary struct {
	Metric   string `json:"metric"`
	Stat     string `json:"stat"`
	Operator string `json:"operator"`
	Target   string `json:"target"`
	Actual   string `json:"actual"`
	Passed   bool   `json:"passed"`
}

// SummaryData contains the complete structured report information post-execution.
type SummaryData struct {
	SuiteName   string             `json:"suite_name"`
	Scenario    string             `json:"scenario"`
	Version     string             `json:"version"`
	Commit      string             `json:"commit"`
	StartedAt   time.Time          `json:"started_at"`
	EndedAt     time.Time          `json:"ended_at"`
	Duration    time.Duration      `json:"duration"`
	Config      any                `json:"config"`
	Metrics     []MetricSummary    `json:"metrics"`
	Checks      []CheckSummary     `json:"checks,omitempty"`
	Thresholds  []ThresholdSummary `json:"thresholds"`
	Passed      bool               `json:"passed"`
	Aborted     bool               `json:"aborted"`
	AbortReason string             `json:"abort_reason,omitempty"`
}

// Scenario groups all lifecycle hooks for a named test scenario.
type Scenario struct {
	Setup         SetupHook
	PreTest       PreTestHook
	RunVU         VURunnerHook
	AfterTest     AfterTestHook
	Teardown      TeardownHook
	HandleSummary SummaryHook
}

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
	Log() log.Logger
	Metrics() metric.Collector
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
