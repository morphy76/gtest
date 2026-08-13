package gtest

import (
	"fmt"

	"github.com/morphy76/gtest/internal/config"
)

// ConfigError represents a failure to load or parse the configuration file.
type ConfigError = config.ConfigError

// ValidationError represents a configuration validation failure.
type ValidationError = config.ValidationError

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

// SetupError wraps an error returned by the Setup hook.
type SetupError struct {
	Err error
}

func (e *SetupError) Error() string {
	return fmt.Sprintf("gtest: setup hook failed: %s", e.Err)
}

func (e *SetupError) Unwrap() error {
	return e.Err
}
