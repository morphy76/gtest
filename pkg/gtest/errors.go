package gtest

import "fmt"

// ConfigError represents a failure to load or parse the configuration file.
type ConfigError struct {
	Path string
	Err  error
}

func (e *ConfigError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("gtest: configuration error in %s: %v", e.Path, e.Err)
	}
	return fmt.Sprintf("gtest: configuration error: %v", e.Err)
}

func (e *ConfigError) Unwrap() error {
	return e.Err
}

// ValidationError represents a configuration validation failure.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("gtest: validation error for field %q: %s", e.Field, e.Message)
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
