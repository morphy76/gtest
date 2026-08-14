package gtest

import (
	"io"
	"os"
	"sync"
	"sync/atomic"

	"github.com/morphy76/gtest/internal/engine"
	"github.com/morphy76/gtest/internal/runner"
)

// ExecutionResult represents the final outcome of running a test suite.
type ExecutionResult struct {
	Passed      bool
	Aborted     bool
	AbortReason string
	Error       error
}

// ExitCode returns 0 if the execution was successful and all SLA thresholds passed,
// or 1 if an error occurred, SLA thresholds breached, or execution was aborted.
func (r ExecutionResult) ExitCode() int {
	if r.Error != nil || !r.Passed || r.Aborted {
		return 1
	}
	return 0
}

// Suite is the root object that test developers interact with.
type Suite struct {
	name      string
	scenarios map[string]engine.Scenario
	mu        sync.Mutex
	executed  atomic.Bool
}

// NewSuite creates an empty suite with the given display name.
// The name appears in terminal reports only.
func NewSuite(name string) *Suite {
	return &Suite{
		name:      name,
		scenarios: make(map[string]engine.Scenario),
	}
}

// Name returns the suite display name.
func (s *Suite) Name() string {
	return s.name
}

// GetScenario retrieves a registered scenario by name.
func (s *Suite) GetScenario(name string) (engine.Scenario, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sc, ok := s.scenarios[name]
	return sc, ok
}

// RegisterScenario associates a named Scenario with the suite.
// The name must exactly match a scenario key in gtest.yaml.
// Panics if name is empty or if RunVU is nil.
// Panics if called after Execute or ExecuteWithArgs has been called.
func (s *Suite) RegisterScenario(name string, scenario Scenario) {
	if s.executed.Load() {
		panic("gtest: cannot call RegisterScenario after Execute")
	}
	if name == "" {
		panic("gtest: RegisterScenario called with empty name")
	}
	if scenario.RunVU == nil {
		panic("gtest: RegisterScenario called with nil RunVU for scenario " + name)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.scenarios[name] = scenario
}

// Execute is the CLI entry point. It:
//  1. Parses CLI flags.
//  2. Loads and validates gtest.yaml via Viper.
//  3. Resolves the target scenario (--scenario flag or default_scenario).
//  4. Executes the scenario lifecycle (Setup → ramp-up → run → ramp-down → Teardown).
//  5. Evaluates SLA thresholds.
//  6. Prints the terminal summary report and executes HandleSummary if configured.
//  7. Returns an ExecutionResult containing the execution outcome and does NOT
//     terminate the host process via os.Exit.
func (s *Suite) Execute() ExecutionResult {
	return s.ExecuteWithArgs(os.Args[1:], os.Stdout)
}

// ExecuteWithArgs executes the suite with custom CLI arguments and output writer.
func (s *Suite) ExecuteWithArgs(args []string, stdout io.Writer) ExecutionResult {
	s.executed.Store(true)
	res := runner.RunSuite(s, args, stdout)
	return ExecutionResult{
		Passed:      res.Passed,
		Aborted:     res.Aborted,
		AbortReason: res.AbortReason,
		Error:       res.Error,
	}
}

