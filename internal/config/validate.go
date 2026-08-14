package config

import (
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/go-viper/mapstructure/v2"
)

// durationDecodeHook returns a mapstructure decode hook that converts string values
// to time.Duration for fields typed as time.Duration.
func durationDecodeHook() mapstructure.DecodeHookFuncType {
	return func(from reflect.Type, to reflect.Type, data any) (any, error) {
		if from.Kind() != reflect.String {
			return data, nil
		}
		if to != reflect.TypeOf(time.Duration(0)) {
			return data, nil
		}

		s, ok := data.(string)
		if !ok {
			return data, nil
		}

		// Empty string means zero duration (for optional fields with default "0s").
		if s == "" {
			return time.Duration(0), nil
		}

		d, err := time.ParseDuration(s)
		if err != nil {
			return nil, fmt.Errorf("cannot parse duration %q: %w", s, err)
		}
		return d, nil
	}
}

// Validate checks all semantic invariants on a parsed Config.
// Returns a *gtest.ValidationError on the first violation found.
func Validate(cfg *Config) error {
	// Version must be "1.0".
	if cfg.Version != "1.0" {
		return &ValidationError{
			Field:   "version",
			Message: fmt.Sprintf("must be %q, got %q", "1.0", cfg.Version),
		}
	}

	// Must have at least one scenario.
	if len(cfg.Scenarios) == 0 {
		return &ValidationError{
			Field:   "scenarios",
			Message: "at least one scenario must be defined",
		}
	}

	// default_scenario must match a key in scenarios if present.
	if cfg.DefaultScenario != "" {
		if _, ok := cfg.Scenarios[cfg.DefaultScenario]; !ok {
			return &ValidationError{
				Field:   "default_scenario",
				Message: fmt.Sprintf("references unknown scenario %q", cfg.DefaultScenario),
			}
		}
	}

	// Validate each scenario.
	for name, sc := range cfg.Scenarios {
		if err := validateScenario(name, &sc); err != nil {
			return err
		}
		// Write back the validated scenario (threshold parsed values).
		cfg.Scenarios[name] = sc
	}

	return nil
}

// validateScenario checks a single scenario configuration.
func validateScenario(name string, sc *ScenarioConfig) error {
	prefix := fmt.Sprintf("scenarios.%s", name)

	// Type is required and must be one of the known types.
	switch sc.Type {
	case ScenarioTypeConstantVUs:
		if sc.VUs <= 0 {
			return &ValidationError{
				Field:   prefix + ".vus",
				Message: "must be > 0 for constant_vus scenario type",
			}
		}
	case ScenarioTypeArrivalRate:
		if sc.TargetTPS <= 0 {
			return &ValidationError{
				Field:   prefix + ".target_tps",
				Message: "must be > 0 for arrival_rate scenario type",
			}
		}
		if sc.MaxVUs <= 0 {
			return &ValidationError{
				Field:   prefix + ".max_vus",
				Message: "must be > 0 for arrival_rate scenario type",
			}
		}
	default:
		return &ValidationError{
			Field:   prefix + ".type",
			Message: fmt.Sprintf("must be %q or %q, got %q", ScenarioTypeConstantVUs, ScenarioTypeArrivalRate, sc.Type),
		}
	}

	// run_period is required and must be > 0.
	if sc.RunPeriod <= 0 {
		return &ValidationError{
			Field:   prefix + ".run_period",
			Message: "must be > 0",
		}
	}

	// vu_timeout is required and must be > 0.
	if sc.VUTimeout <= 0 {
		return &ValidationError{
			Field:   prefix + ".vu_timeout",
			Message: "must be > 0",
		}
	}

	// ramp_up must be >= 0 (it defaults to 0).
	if sc.RampUp < 0 {
		return &ValidationError{
			Field:   prefix + ".ramp_up",
			Message: "must be >= 0",
		}
	}

	// ramp_down must be >= 0 (it defaults to 0).
	if sc.RampDown < 0 {
		return &ValidationError{
			Field:   prefix + ".ramp_down",
			Message: "must be >= 0",
		}
	}

	// Validate params: keys and values must be non-empty strings.
	for k, v := range sc.Params {
		if k == "" {
			return &ValidationError{
				Field:   prefix + ".params",
				Message: "param keys must be non-empty strings",
			}
		}
		if v == "" {
			return &ValidationError{
				Field:   prefix + ".params." + k,
				Message: "param values must be non-empty strings",
			}
		}
	}

	// Validate interaction_delay if specified.
	if sc.InteractionDelay != nil {
		if err := validateInteractionDelay(prefix, sc.InteractionDelay); err != nil {
			return err
		}
	}

	// Validate think_time if specified.
	if sc.ThinkTime != nil {
		if err := validateThinkTime(prefix, sc.ThinkTime); err != nil {
			return err
		}
	}

	// Validate thresholds.
	for i := range sc.Thresholds {
		if err := validateThreshold(prefix, i, &sc.Thresholds[i]); err != nil {
			return err
		}
	}

	return nil
}

// validateInteractionDelay checks the interaction delay configuration.
func validateInteractionDelay(prefix string, delay *InteractionDelayConfig) error {
	return validateDelayConfig(fmt.Sprintf("%s.interaction_delay", prefix), delay)
}

// validateThinkTime checks the inter-iteration think time configuration.
func validateThinkTime(prefix string, tt *ThinkTimeConfig) error {
	return validateDelayConfig(fmt.Sprintf("%s.think_time", prefix), tt)
}

// validateDelayConfig checks a delay configuration against strategy bounds.
func validateDelayConfig(delayPrefix string, delay *ThinkTimeConfig) error {
	switch delay.Type {
	case "fixed":

		if delay.Duration <= 0 {
			return &ValidationError{
				Field:   delayPrefix + ".duration",
				Message: "must be > 0 for fixed delay",
			}
		}
	case "range":
		if delay.Min < 0 {
			return &ValidationError{
				Field:   delayPrefix + ".min",
				Message: "must be >= 0 for range delay",
			}
		}
		if delay.Max <= 0 {
			return &ValidationError{
				Field:   delayPrefix + ".max",
				Message: "must be > 0 for range delay",
			}
		}
		if delay.Max < delay.Min {
			return &ValidationError{
				Field:   delayPrefix + ".max",
				Message: "must be >= min for range delay",
			}
		}
	case "expo":
		if delay.Mean <= 0 {
			return &ValidationError{
				Field:   delayPrefix + ".mean",
				Message: "must be > 0 for expo delay",
			}
		}
		if delay.Min < 0 {
			return &ValidationError{
				Field:   delayPrefix + ".min",
				Message: "must be >= 0",
			}
		}
		if delay.Max < 0 {
			return &ValidationError{
				Field:   delayPrefix + ".max",
				Message: "must be >= 0",
			}
		}
		if delay.Min > 0 && delay.Max > 0 && delay.Max < delay.Min {
			return &ValidationError{
				Field:   delayPrefix + ".max",
				Message: "must be >= min for expo delay",
			}
		}
	case "gaussian":
		if delay.Mean <= 0 {
			return &ValidationError{
				Field:   delayPrefix + ".mean",
				Message: "must be > 0 for gaussian delay",
			}
		}
		if delay.StdDev <= 0 {
			return &ValidationError{
				Field:   delayPrefix + ".std_dev",
				Message: "must be > 0 for gaussian delay",
			}
		}
		if delay.Min < 0 {
			return &ValidationError{
				Field:   delayPrefix + ".min",
				Message: "must be >= 0",
			}
		}
		if delay.Max < 0 {
			return &ValidationError{
				Field:   delayPrefix + ".max",
				Message: "must be >= 0",
			}
		}
		if delay.Min > 0 && delay.Max > 0 && delay.Max < delay.Min {
			return &ValidationError{
				Field:   delayPrefix + ".max",
				Message: "must be >= min for gaussian delay",
			}
		}
	default:
		return &ValidationError{
			Field:   delayPrefix + ".type",
			Message: fmt.Sprintf("must be one of {fixed, range, expo, gaussian}, got %q", delay.Type),
		}
	}

	return nil
}


// validateThreshold checks a single threshold configuration and parses its target value.
func validateThreshold(prefix string, idx int, th *ThresholdConfig) error {
	thPrefix := fmt.Sprintf("%s.thresholds[%d]", prefix, idx)

	// metric is required.
	if th.Metric == "" {
		return &ValidationError{
			Field:   thPrefix + ".metric",
			Message: "must not be empty",
		}
	}

	// stat must be one of the known values.
	if !validStats[th.Stat] {
		return &ValidationError{
			Field:   thPrefix + ".stat",
			Message: fmt.Sprintf("must be one of {p50, p90, p95, p99, mean, max, count, rate, value}, got %q", th.Stat),
		}
	}

	// operator must be one of the known values.
	if !validOperators[th.Operator] {
		return &ValidationError{
			Field:   thPrefix + ".operator",
			Message: fmt.Sprintf("must be one of {<, <=, >, >=}, got %q", th.Operator),
		}
	}

	// target is required.
	if th.Target == "" {
		return &ValidationError{
			Field:   thPrefix + ".target",
			Message: "must not be empty",
		}
	}

	// Parse target based on stat type.
	if IsDurationStat(th.Stat) {
		d, err := time.ParseDuration(th.Target)
		if err != nil {
			return &ValidationError{
				Field:   thPrefix + ".target",
				Message: fmt.Sprintf("cannot parse %q as duration for stat %q: %s", th.Target, th.Stat, err),
			}
		}
		th.TargetDuration = d
	} else {
		f, err := strconv.ParseFloat(th.Target, 64)
		if err != nil {
			return &ValidationError{
				Field:   thPrefix + ".target",
				Message: fmt.Sprintf("cannot parse %q as float64 for stat %q: %s", th.Target, th.Stat, err),
			}
		}
		th.TargetFloat = f
	}

	return nil
}
