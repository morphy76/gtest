package vuhive

import "github.com/morphy76/vuhive/internal/runner"

// Export for internal tests in vuhive_test.
func SuiteAdapterForTest(s *Suite) runner.ScenarioRegistry {
	return &runnerSuiteAdapter{suite: s}
}
