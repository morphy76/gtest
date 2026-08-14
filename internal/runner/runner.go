package runner

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/morphy76/gtest/internal/engine"
	"github.com/morphy76/gtest/internal/log"
	"github.com/morphy76/gtest/internal/metric"
	"github.com/morphy76/gtest/internal/sla"
	"github.com/rs/zerolog"
)

// Result represents the outcome of running a load test suite.
type Result struct {
	Passed      bool
	Aborted     bool
	AbortReason string
	Error       error
}

// RunSuite executes the suite CLI workflow.
func RunSuite(s ScenarioRegistry, args []string, stdout io.Writer) Result {
	if stdout == nil {
		stdout = io.Discard
	}

	resolved, err := resolveScenario(s, args, stdout)
	if err != nil {
		return Result{Error: err}
	}

	if resolved.ShowVersion {
		return Result{Passed: true}
	}

	// Setup logger
	logLevel, parseErr := zerolog.ParseLevel(resolved.Flags.LogLevel)
	if parseErr != nil {
		logLevel = zerolog.InfoLevel
	}
	logger := log.NewWithFormat(stdout, logLevel, resolved.Flags.LogFormat)
	metricsStore := metric.NewStore()

	startedAt := time.Now()
	executor := engine.NewExecutor(resolved.TargetScenario, resolved.Scenario, resolved.ScenarioCfg, logger, metricsStore)

	execErr := executor.Execute(context.Background())
	endedAt := time.Now()

	if execErr != nil {
		var setupErr *engine.SetupError
		if errors.As(execErr, &setupErr) {
			return Result{Error: setupErr}
		}
		return Result{Error: execErr}
	}

	// Evaluate SLA thresholds
	thresholdResults := sla.Evaluate(resolved.ScenarioCfg.Thresholds, metricsStore)
	allPassed := sla.AllPassed(thresholdResults)
	if executor.Aborted {
		allPassed = false
	}

	reportExecution(context.Background(), ReportParams{
		SuiteName:        s.Name(),
		ScenarioName:     resolved.TargetScenario,
		Scenario:         resolved.Scenario,
		ScenarioCfg:      resolved.ScenarioCfg,
		Flags:            resolved.Flags,
		MetricsStore:     metricsStore,
		ThresholdResults: thresholdResults,
		AllPassed:        allPassed,
		Aborted:          executor.Aborted,
		AbortReason:      executor.AbortReason,
		StartedAt:        startedAt,
		EndedAt:          endedAt,
		Stdout:           stdout,
		Logger:           logger.Zerolog(),
	})

	return Result{
		Passed:      allPassed,
		Aborted:     executor.Aborted,
		AbortReason: executor.AbortReason,
		Error:       nil,
	}
}
