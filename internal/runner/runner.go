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

// ScenarioNotFoundError indicates a scenario was not found in the config or not registered.
type ScenarioNotFoundError struct {
	Name    string
	Message string
}

func (e *ScenarioNotFoundError) Error() string {
	if e.Name != "" {
		return fmt.Sprintf("gtest: scenario %q not found: %s", e.Name, e.Message)
	}
	return fmt.Sprintf("gtest: scenario not found: %s", e.Message)
}

// ScenarioRegistry provides access to named scenarios.
type ScenarioRegistry interface {
	Name() string
	GetScenario(name string) (engine.Scenario, bool)
}

// RunSuite executes the suite CLI workflow.
func RunSuite(s ScenarioRegistry, args []string, stdout io.Writer) Result {
	resolved, err := resolveScenario(s, args, stdout)
	if err != nil {
		return err
	}

	if resolved.ShowVersion {
		return nil
	}
  
	if stdout == nil {
		stdout = io.Discard
	}
	flags, err := cli.ParseFlags(args, stdout)
	if err != nil {
		return Result{Error: &config.ConfigError{Err: err}}
	}

	if flags.ShowVersion {
		if _, err := fmt.Fprintf(stdout, "gtest version %s (commit: %s, build_time: %s)\n",
			version.Version, version.Commit, version.BuildTime); err != nil {
			return Result{Error: err}
		}
		return Result{Passed: true}
	}

	cfg, err := config.LoadFromFile(flags.ConfigPath)
	if err != nil {
		var valErr *config.ValidationError
		if errors.As(err, &valErr) {
			return Result{Error: valErr}
		}
		var cfgErr *config.ConfigError
		if errors.As(err, &cfgErr) {
			return Result{Error: cfgErr}
		}
		return Result{Error: &config.ConfigError{Path: flags.ConfigPath, Err: err}}
	}

	targetScenario := flags.ScenarioName
	if targetScenario == "" {
		targetScenario = cfg.DefaultScenario
	}
	if targetScenario == "" {
		return Result{
			Error: &ScenarioNotFoundError{
				Name:    "",
				Message: "no scenario specified via --scenario flag or default_scenario in config",
			},
		}
	}

	scenarioCfg, inConfig := cfg.Scenarios[targetScenario]
	scenario, registered := s.GetScenario(targetScenario)

	if !registered {
		return Result{
			Error: &ScenarioNotFoundError{
				Name:    targetScenario,
				Message: fmt.Sprintf("scenario %q is not registered in Suite", targetScenario),
			},
		}
	}
	if !inConfig {
		return Result{
			Error: &ScenarioNotFoundError{
				Name:    targetScenario,
				Message: fmt.Sprintf("scenario %q is registered in code but not defined in config file %q", targetScenario, flags.ConfigPath),
			},
		}
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
