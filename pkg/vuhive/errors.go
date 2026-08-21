package vuhive

import "fmt"

// ConfigError represents a failure to load or parse the configuration file.
type ConfigError struct {
	// Path is the filesystem path to the configuration file that caused the error.
	Path string
	// Err is the underlying parsing or I/O error.
	Err error
}

func (e *ConfigError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("vuhive: configuration error in %s: %v", e.Path, e.Err)
	}
	return fmt.Sprintf("vuhive: configuration error: %v", e.Err)
}

func (e *ConfigError) Unwrap() error {
	return e.Err
}

// ValidationError represents a configuration validation failure.
type ValidationError struct {
	// Field is the configuration field key that failed validation.
	Field string
	// Message provides a human-readable explanation of why validation failed.
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("vuhive: validation error for field %q: %s", e.Field, e.Message)
}

// ScenarioNotFoundError indicates a scenario was not found in the config or not registered.
type ScenarioNotFoundError struct {
	// Name is the name of the scenario that could not be found.
	Name string
	// Message describes the specific scenario resolution failure.
	Message string
}

func (e *ScenarioNotFoundError) Error() string {
	if e.Name != "" {
		return fmt.Sprintf("vuhive: scenario %q not found: %s", e.Name, e.Message)
	}
	return fmt.Sprintf("vuhive: scenario not found: %s", e.Message)
}

// SetupError wraps an error returned by the Setup hook.
type SetupError struct {
	// Err is the underlying error returned by the scenario's Setup hook.
	Err error
}

func (e *SetupError) Error() string {
	return fmt.Sprintf("vuhive: setup hook failed: %s", e.Err)
}

func (e *SetupError) Unwrap() error {
	return e.Err
}

// Compile-time interface satisfaction checks.
var (
	_ error = (*ConfigError)(nil)
	_ error = (*ValidationError)(nil)
	_ error = (*ScenarioNotFoundError)(nil)
	_ error = (*SetupError)(nil)
)
