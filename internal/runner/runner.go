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

// RunSuite coordinates scenario resolution, execution, metric aggregation, report dispatch, and exit handling.
func RunSuite(s ScenarioRegistry, args []string, stdout io.Writer, exitFunc func(int)) error {
	resolved, err := resolveScenario(s, args, stdout)
	if err != nil {
		return err
	}

	if resolved.ShowVersion {
		return nil
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
			return setupErr
		}
		return execErr
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

	if exitFunc != nil {
		if allPassed {
			exitFunc(0)
		} else {
			exitFunc(1)
		}
	}

	return nil
}
