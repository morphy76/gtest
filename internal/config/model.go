// Package config provides YAML configuration loading, parsing, and validation
// for gtest load test scenarios.
package config

import "time"

// ScenarioType defines the execution model for a scenario.
type ScenarioType string

const (
	// ScenarioTypeConstantVUs runs a fixed number of virtual user goroutines.
	ScenarioTypeConstantVUs ScenarioType = "constant_vus"

	// ScenarioTypeArrivalRate dispatches iterations at a target TPS using a token bucket.
	ScenarioTypeArrivalRate ScenarioType = "arrival_rate"
)

// Config is the top-level configuration loaded from gtest.yaml.
type Config struct {
	// Version must be "1.0".
	Version string

	// DefaultScenario is the name of the scenario to run when --scenario is not provided.
	// Must match a key in Scenarios if present.
	DefaultScenario string

	// Scenarios maps scenario names to their configurations.
	Scenarios map[string]ScenarioConfig
}

// ScenarioConfig holds the configuration for a single load test scenario.
type ScenarioConfig struct {
	// Type is the execution model: "constant_vus" or "arrival_rate".
	Type ScenarioType

	// VUs is the number of virtual user goroutines (required for constant_vus).
	VUs int

	// TargetTPS is the target transactions per second (required for arrival_rate).
	TargetTPS int

	// MaxVUs is the hard cap on concurrent goroutines (required for arrival_rate).
	MaxVUs int

	// RampUp is the duration to linearly ramp up to the target level.
	RampUp time.Duration

	// RunPeriod is the steady-state execution duration (required).
	RunPeriod time.Duration

	// RampDown is the duration for graceful ramp-down.
	RampDown time.Duration

	// VUTimeout is the per-iteration context timeout (required).
	VUTimeout time.Duration

	// Params is an arbitrary key-value map available to test code via ScenarioContext.Param().
	Params map[string]string

	// InteractionDelay defines think time strategy between actions (via ctx.Sleep).
	InteractionDelay *InteractionDelayConfig

	// ThinkTime defines inter-iteration pacing delay automatically executed by the engine loop.
	ThinkTime *ThinkTimeConfig

	// Thresholds defines SLA assertions evaluated after the test run.
	Thresholds []ThresholdConfig
}

// ThinkTimeConfig holds configuration for inter-iteration think time delays.
type ThinkTimeConfig struct {
	// Type is the strategy type: "fixed", "range", "expo", "gaussian".
	Type string

	// Duration is the static pause duration (used for "fixed").
	Duration time.Duration

	// Min is the minimum duration (used for "range", optional clamp for "expo" and "gaussian").
	Min time.Duration

	// Max is the maximum duration (used for "range", optional clamp for "expo" and "gaussian").
	Max time.Duration

	// Mean is the average duration (used for "expo" and "gaussian").
	Mean time.Duration

	// StdDev is the standard deviation (used for "gaussian").
	StdDev time.Duration
}

// InteractionDelayConfig holds configuration for in-iteration think time delays.
type InteractionDelayConfig = ThinkTimeConfig

// ThresholdConfig defines a single SLA threshold assertion.
type ThresholdConfig struct {
	// Metric is the exact metric name as recorded by the test developer.
	Metric string

	// Stat is the statistic to evaluate (p50, p90, p95, p99, mean, max, count, rate, value).
	Stat string

	// Operator is the comparison operator: <, <=, >, >=.
	Operator string

	// Target is the threshold value as a raw string.
	// Parsed as time.Duration for duration stats (p50, p90, p95, p99, mean, max),
	// or as float64 for non-duration stats (count, rate, value).
	Target string

	// AbortOnFail triggers early test termination if breached during execution.
	AbortOnFail bool

	// DelayAbortEval is a warm-up grace period before abort evaluation begins.
	DelayAbortEval time.Duration

	// TargetDuration is the parsed target when Stat is a duration stat.
	// Populated by validation; zero value if not applicable.
	TargetDuration time.Duration

	// TargetFloat is the parsed target when Stat is a non-duration stat.
	// Populated by validation; zero value if not applicable.
	TargetFloat float64
}

