package engine

import (
	"context"
	"fmt"

	"github.com/morphy76/gtest/internal/config"
	"github.com/morphy76/gtest/internal/metric"
	"github.com/morphy76/gtest/pkg/gtest"
)

// Executor orchestrates scenario execution: Setup -> VUs -> Teardown.
type Executor struct {
	ScenarioName string
	Scenario     gtest.Scenario
	Config       config.ScenarioConfig
	Logger       gtest.Logger
	Metrics      *metric.Store
}

// NewExecutor creates a new scenario executor.
func NewExecutor(
	scenarioName string,
	scenario gtest.Scenario,
	cfg config.ScenarioConfig,
	logger gtest.Logger,
	metrics *metric.Store,
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
		var err error
		globalState, err = e.Scenario.Setup(ctx)
		if err != nil {
			return &gtest.SetupError{Err: err}
		}
	}

	// 2. VU Pacing Engine phase
	switch e.Config.Type {
	case config.ScenarioTypeConstantVUs:
		RunConstantVUs(ctx, e.Scenario, e.Config, e.ScenarioName, globalState, e.Logger, e.Metrics)
	case config.ScenarioTypeArrivalRate:
		RunArrivalRate(ctx, e.Scenario, e.Config, e.ScenarioName, globalState, e.Logger, e.Metrics)
	default:
		return fmt.Errorf("gtest: unsupported scenario type %q", e.Config.Type)
	}

	// 3. Teardown phase
	if e.Scenario.Teardown != nil {
		if err := e.Scenario.Teardown(ctx, globalState); err != nil {
			e.Logger.Error().Err(err).Msg("Teardown hook error")
		}
	}

	return nil
}
