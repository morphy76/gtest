package config

import "fmt"

// ConfigError is returned when gtest.yaml cannot be found, read, or parsed as valid YAML.
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

// ValidationError is returned when gtest.yaml violates a structural or semantic invariant.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("gtest: validation error for field %q: %s", e.Field, e.Message)
}
