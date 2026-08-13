package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/morphy76/gtest/internal/cli"
	"github.com/morphy76/gtest/internal/config"
	"github.com/morphy76/gtest/internal/engine"
	"github.com/morphy76/gtest/internal/log"
	"github.com/morphy76/gtest/internal/metric"
	"github.com/morphy76/gtest/internal/report"
	"github.com/morphy76/gtest/internal/sla"
	"github.com/morphy76/gtest/internal/version"
	"github.com/morphy76/gtest/pkg/gtest"
	"github.com/rs/zerolog"
)

func init() {
	gtest.SetExecutor(RunSuite)
}

// RunSuite executes the suite CLI workflow.
func RunSuite(s *gtest.Suite, args []string, stdout io.Writer, exitFunc func(int)) error {
	flags, err := cli.ParseFlags(args, stdout)
	if err != nil {
		return &gtest.ConfigError{Err: err}
	}

	if flags.ShowVersion {
		fmt.Fprintf(stdout, "gtest version %s (commit: %s, build_time: %s)\n",
			version.Version, version.Commit, version.BuildTime)
		return nil
	}

	cfg, err := config.LoadFromFile(flags.ConfigPath)
	if err != nil {
		var valErr *gtest.ValidationError
		if errors.As(err, &valErr) {
			return valErr
		}
		var cfgErr *gtest.ConfigError
		if errors.As(err, &cfgErr) {
			return cfgErr
		}
		return &gtest.ConfigError{Path: flags.ConfigPath, Err: err}
	}

	targetScenario := flags.ScenarioName
	if targetScenario == "" {
		targetScenario = cfg.DefaultScenario
	}
	if targetScenario == "" {
		return &gtest.ScenarioNotFoundError{
			Name:    "",
			Message: "no scenario specified via --scenario flag or default_scenario in config",
		}
	}

	scenarioCfg, inConfig := cfg.Scenarios[targetScenario]
	scenario, registered := s.GetScenario(targetScenario)

	if !registered {
		return &gtest.ScenarioNotFoundError{
			Name:    targetScenario,
			Message: fmt.Sprintf("scenario %q is not registered in Suite", targetScenario),
		}
	}
	if !inConfig {
		return &gtest.ScenarioNotFoundError{
			Name:    targetScenario,
			Message: fmt.Sprintf("scenario %q is registered in code but not defined in config file %q", targetScenario, flags.ConfigPath),
		}
	}

	// Setup logger
	logLevel, parseErr := zerolog.ParseLevel(flags.LogLevel)
	if parseErr != nil {
		logLevel = zerolog.InfoLevel
	}
	logger := log.New(stdout, logLevel)
	metricsStore := metric.NewStore()

	startedAt := time.Now()
	executor := engine.NewExecutor(targetScenario, scenario, scenarioCfg, logger, metricsStore)

	execErr := executor.Execute(context.Background())
	endedAt := time.Now()

	if execErr != nil {
		var setupErr *gtest.SetupError
		if errors.As(execErr, &setupErr) {
			return setupErr
		}
		return execErr
	}

	// Evaluate SLA thresholds
	thresholdResults := sla.Evaluate(scenarioCfg.Thresholds, metricsStore)
	allPassed := sla.AllPassed(thresholdResults)

	reportData := report.ReportData{
		SuiteName:  s.Name(),
		Scenario:   targetScenario,
		Version:    version.Version,
		Commit:     version.Commit,
		StartedAt:  startedAt,
		EndedAt:    endedAt,
		Config:     scenarioCfg,
		Metrics:    metricsStore,
		Thresholds: thresholdResults,
		Passed:     allPassed,
	}

	if err := report.WriteReport(stdout, flags.ReportFormat, flags.ReportOut, reportData); err != nil {
		logger.Error().Err(err).Msg("failed to write report")
	}

	if exitFunc != nil {
		if allPassed {
			exitFunc(0)
		} else {
			exitFunc(1)
		}
	}

	return nil
}
