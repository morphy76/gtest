package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/morphy76/gtest/internal/config"
	"github.com/morphy76/gtest/internal/log"
	"github.com/morphy76/gtest/internal/metric"
	"github.com/morphy76/gtest/internal/sla"
)

// MetricsStore defines the metric recording and reading capabilities required by the scenario executor.
type MetricsStore interface {
	metric.Collector
	sla.MetricReader
}

// Executor orchestrates scenario execution: Setup -> VUs -> Teardown.
type Executor struct {
	ScenarioName string
	Scenario     Scenario
	Config       config.ScenarioConfig
	Logger       log.Logger
	Metrics      MetricsStore
	Aborted      bool
	AbortReason  string
}

// NewExecutor creates a new scenario executor.
func NewExecutor(
	scenarioName string,
	scenario Scenario,
	cfg config.ScenarioConfig,
	logger log.Logger,
	metrics MetricsStore,
) *Executor {
	return &Executor{
		ScenarioName: scenarioName,
		Scenario:     scenario,
		Config:       cfg,
		Logger:       logger,
		Metrics:      metrics,
	}
}

// Execute runs the complete scenario lifecycle.
func (e *Executor) Execute(ctx context.Context) error {
	var globalState map[string]any

	// 1. Setup phase
	if e.Scenario.Setup != nil {
		setupCtx := newScenarioContext(ctx, 0, 0, e.Config, e.ScenarioName, nil, e.Logger, e.Metrics)
		var err error
		globalState, err = e.Scenario.Setup(setupCtx)
		if err != nil {
			return &SetupError{Err: err}
		}
		// Shallow copy the state map so VUs cannot mutate the original (spec §4.3).
		if globalState != nil {
			safeCopy := make(map[string]any, len(globalState))
			for k, v := range globalState {
				safeCopy[k] = v
			}
			globalState = safeCopy
		}
	}

	// 2. VU Pacing Engine phase with abort monitor
	pacingCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	startTime := time.Now()
	abortedCh, getReason := MonitorAbortThresholds(pacingCtx, cancel, startTime, e.Config.Thresholds, e.Metrics, e.Logger)

	switch e.Config.Type {
	case config.ScenarioTypeConstantVUs:
		RunConstantVUs(pacingCtx, e.Scenario, e.Config, e.ScenarioName, globalState, e.Logger, e.Metrics)
	case config.ScenarioTypeArrivalRate:
		RunArrivalRate(pacingCtx, e.Scenario, e.Config, e.ScenarioName, globalState, e.Logger, e.Metrics)
	default:
		return fmt.Errorf("gtest: unsupported scenario type %q", e.Config.Type)
	}

	select {
	case <-abortedCh:
		e.Aborted = true
		e.AbortReason = getReason()
	default:
		if reason := getReason(); reason != "" {
			e.Aborted = true
			e.AbortReason = reason
		}
	}

	// 3. Teardown phase
	if e.Scenario.Teardown != nil {
		teardownCtx := newScenarioContext(ctx, 0, 0, e.Config, e.ScenarioName, globalState, e.Logger, e.Metrics)
		if err := e.Scenario.Teardown(teardownCtx, globalState); err != nil {
			e.Logger.Error().Err(err).Msg("Teardown hook error")
		}
	}

	return nil
}
