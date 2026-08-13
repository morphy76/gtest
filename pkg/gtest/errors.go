package gtest

import (
	"github.com/morphy76/gtest/internal/config"
	"github.com/morphy76/gtest/internal/engine"
	"github.com/morphy76/gtest/internal/runner"
)

// ConfigError represents a failure to load or parse the configuration file.
type ConfigError = config.ConfigError

// ValidationError represents a configuration validation failure.
type ValidationError = config.ValidationError

// ScenarioNotFoundError indicates a scenario was not found in the config or not registered.
type ScenarioNotFoundError = runner.ScenarioNotFoundError

// SetupError wraps an error returned by the Setup hook.
type SetupError = engine.SetupError
