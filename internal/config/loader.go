package config

import (
	"fmt"
	"io"

	"github.com/spf13/viper"
)

// Load reads YAML configuration from the given reader and returns a validated Config.
// It returns a *ConfigError for parse failures and a *ValidationError for semantic validation failures.
func Load(r io.Reader) (*Config, error) {
	v := viper.New()
	v.SetConfigType("yaml")

	if err := v.ReadConfig(r); err != nil {
		return nil, &ConfigError{Err: fmt.Errorf("failed to parse YAML: %w", err)}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg, viper.DecodeHook(durationDecodeHook())); err != nil {
		return nil, &ConfigError{Err: fmt.Errorf("failed to unmarshal config: %w", err)}
	}

	if err := Validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// LoadFromFile reads YAML configuration from the given file path and returns a validated Config.
func LoadFromFile(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)

	if err := v.ReadInConfig(); err != nil {
		return nil, &ConfigError{Path: path, Err: fmt.Errorf("failed to read config file: %w", err)}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg, viper.DecodeHook(durationDecodeHook())); err != nil {
		return nil, &ConfigError{Path: path, Err: fmt.Errorf("failed to unmarshal config: %w", err)}
	}

	if err := Validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
