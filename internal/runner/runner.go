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
	"github.com/rs/zerolog"
)

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
func RunSuite(s ScenarioRegistry, args []string, stdout io.Writer, exitFunc func(int)) error {
	flags, err := cli.ParseFlags(args, stdout)
	if err != nil {
		return &config.ConfigError{Err: err}
	}

	if flags.ShowVersion {
		if _, err := fmt.Fprintf(stdout, "gtest version %s (commit: %s, build_time: %s)\n",
			version.Version, version.Commit, version.BuildTime); err != nil {
			return err
		}
		return nil
	}


	cfg, err := config.LoadFromFile(flags.ConfigPath)
	if err != nil {
		var valErr *config.ValidationError
		if errors.As(err, &valErr) {
			return valErr
		}
		var cfgErr *config.ConfigError
		if errors.As(err, &cfgErr) {
			return cfgErr
		}
		return &config.ConfigError{Path: flags.ConfigPath, Err: err}
	}

	targetScenario := flags.ScenarioName
	if targetScenario == "" {
		targetScenario = cfg.DefaultScenario
	}
	if targetScenario == "" {
		return &ScenarioNotFoundError{
			Name:    "",
			Message: "no scenario specified via --scenario flag or default_scenario in config",
		}
	}

	scenarioCfg, inConfig := cfg.Scenarios[targetScenario]
	scenario, registered := s.GetScenario(targetScenario)

	if !registered {
		return &ScenarioNotFoundError{
			Name:    targetScenario,
			Message: fmt.Sprintf("scenario %q is not registered in Suite", targetScenario),
		}
	}
	if !inConfig {
		return &ScenarioNotFoundError{
			Name:    targetScenario,
			Message: fmt.Sprintf("scenario %q is registered in code but not defined in config file %q", targetScenario, flags.ConfigPath),
		}
	}

	// Setup logger
	logLevel, parseErr := zerolog.ParseLevel(flags.LogLevel)
	if parseErr != nil {
		logLevel = zerolog.InfoLevel
	}
	logger := log.NewWithFormat(stdout, logLevel, flags.LogFormat)
	metricsStore := metric.NewStore()

	startedAt := time.Now()
	executor := engine.NewExecutor(targetScenario, scenario, scenarioCfg, logger, metricsStore)

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

	if flags.JSONReportOut != "" {
		if err := report.WriteReport(stdout, "json", flags.JSONReportOut, reportData); err != nil {
			logger.Error().Err(err).Msg("failed to write JSON report")
		}
	}

	if scenario.HandleSummary != nil {
		summaryData := buildSummaryData(
			s.Name(),
			targetScenario,
			version.Version,
			version.Commit,
			startedAt,
			endedAt,
			scenarioCfg,
			metricsStore,
			thresholdResults,
			allPassed,
		)
		if err := scenario.HandleSummary(context.Background(), summaryData); err != nil {
			logger.Error().Err(err).Msg("HandleSummary hook error")
		}
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

func buildSummaryData(
	suiteName string,
	scenarioName string,
	versionStr string,
	commitStr string,
	startedAt time.Time,
	endedAt time.Time,
	cfg config.ScenarioConfig,
	metricsStore *metric.Store,
	thresholdResults []sla.ThresholdResult,
	allPassed bool,
) engine.SummaryData {
	duration := endedAt.Sub(startedAt)
	if duration < 0 {
		duration = 0
	}

	var thresholds []engine.ThresholdSummary
	for _, th := range thresholdResults {
		thresholds = append(thresholds, engine.ThresholdSummary{
			Metric:   th.Metric,
			Stat:     th.Stat,
			Operator: th.Operator,
			Target:   th.Target,
			Actual:   th.Actual,
			Passed:   th.Passed,
		})
	}

	type namedEntry struct {
		name string
		item engine.MetricSummary
	}
	var entries []namedEntry

	// Histograms
	for _, name := range metricsStore.HistogramNames() {
		snap := metricsStore.MergedHistogramSnapshot(name)
		entries = append(entries, namedEntry{
			name: name,
			item: engine.MetricSummary{
				Name:  name,
				Type:  "duration",
				Count: snap.Count,
				Min:   snap.Min,
				Mean:  snap.Mean,
				P50:   snap.P50,
				P90:   snap.P90,
				P95:   snap.P95,
				P99:   snap.P99,
				Max:   snap.Max,
			},
		})
	}

	// Counters
	for _, name := range metricsStore.CounterNames() {
		val := metricsStore.AggregatedCounterValue(name)
		entries = append(entries, namedEntry{
			name: name,
			item: engine.MetricSummary{
				Name:  name,
				Type:  "counter",
				Count: val,
			},
		})
	}

	// Gauges
	for _, name := range metricsStore.GaugeNames() {
		val := metricsStore.LastGaugeValue(name)
		entries = append(entries, namedEntry{
			name: name,
			item: engine.MetricSummary{
				Name:  name,
				Type:  "gauge",
				Value: val,
			},
		})
	}

	// Rates
	for _, name := range metricsStore.RateNames() {
		val := metricsStore.AggregatedRateValue(name)
		entries = append(entries, namedEntry{
			name: name,
			item: engine.MetricSummary{
				Name: name,
				Type: "rate",
				Rate: val,
			},
		})
	}

	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].name > entries[j].name {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	var metrics []engine.MetricSummary
	for _, e := range entries {
		metrics = append(metrics, e.item)
	}

	return engine.SummaryData{
		SuiteName:  suiteName,
		Scenario:   scenarioName,
		Version:    versionStr,
		Commit:     commitStr,
		StartedAt:  startedAt,
		EndedAt:    endedAt,
		Duration:   duration,
		Config:     cfg,
		Metrics:    metrics,
		Thresholds: thresholds,
		Passed:     allPassed,
	}
}

