package gtest

import (
	"fmt"
	"io"
	"os"
	"sync"
)

// SuiteExecutorFunc is the signature for the suite CLI execution engine.
type SuiteExecutorFunc func(s *Suite, args []string, stdout io.Writer, exitFunc func(int)) error

var defaultExecutor SuiteExecutorFunc

// SetExecutor registers the execution engine for Suite.Execute.
func SetExecutor(exec SuiteExecutorFunc) {
	defaultExecutor = exec
}

// Suite is the root object that test developers interact with.
type Suite struct {
	name      string
	scenarios map[string]Scenario
	mu        sync.Mutex
}

// NewSuite creates an empty suite with the given display name.
// The name appears in terminal reports only.
func NewSuite(name string) *Suite {
	return &Suite{
		name:      name,
		scenarios: make(map[string]Scenario),
	}
}

// Name returns the suite display name.
func (s *Suite) Name() string {
	return s.name
}

// GetScenario retrieves a registered scenario by name.
func (s *Suite) GetScenario(name string) (Scenario, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sc, ok := s.scenarios[name]
	return sc, ok
}

// RegisterScenario associates a named Scenario with the suite.
// The name must exactly match a scenario key in gtest.yaml.
// Panics if name is empty or if RunVU is nil.
// Calling RegisterScenario after Execute has been called is undefined behavior.
func (s *Suite) RegisterScenario(name string, scenario Scenario) {
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
//  1. Parses CLI flags (see §6 for the full flag inventory).
//  2. Loads and validates gtest.yaml via Viper.
//  3. Resolves the target scenario (--scenario flag or default_scenario).
//  4. Executes the scenario lifecycle (Setup → ramp-up → run → ramp-down → Teardown).
//  5. Evaluates SLA thresholds.
//  6. Prints the terminal summary report.
//  7. Returns an error if the scenario was not found, config was invalid,
//     or Setup returned an error. Does NOT return an error for SLA threshold failures
//     (those are expressed via os.Exit(1)).
//
// Execute calls os.Exit(1) directly if any SLA threshold is breached.
// Execute calls os.Exit(0) on clean completion.
// It returns a non-nil error only for fatal pre-execution failures (config, registration).
func (s *Suite) Execute() error {
	return s.ExecuteWithArgs(os.Args[1:], os.Stdout, os.Exit)
}

// ExecuteWithArgs allows executing the suite with custom args, stdout, and exitFunc for testing.
func (s *Suite) ExecuteWithArgs(args []string, stdout io.Writer, exitFunc func(int)) error {
	if defaultExecutor == nil {
		return fmt.Errorf("gtest: suite executor is not registered")
	}
	return defaultExecutor(s, args, stdout, exitFunc)
}
