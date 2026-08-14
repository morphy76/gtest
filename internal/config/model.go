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
	Version string `mapstructure:"version"`

	// DefaultScenario is the name of the scenario to run when --scenario is not provided.
	// Must match a key in Scenarios if present.
	DefaultScenario string `mapstructure:"default_scenario"`

	// Scenarios maps scenario names to their configurations.
	Scenarios map[string]ScenarioConfig `mapstructure:"scenarios"`
}

// ScenarioConfig holds the configuration for a single load test scenario.
type ScenarioConfig struct {
	// Type is the execution model: "constant_vus" or "arrival_rate".
	Type ScenarioType `mapstructure:"type"`

	// VUs is the number of virtual user goroutines (required for constant_vus).
	VUs int `mapstructure:"vus"`

	// TargetTPS is the target transactions per second (required for arrival_rate).
	TargetTPS int `mapstructure:"target_tps"`

	// MaxVUs is the hard cap on concurrent goroutines (required for arrival_rate).
	MaxVUs int `mapstructure:"max_vus"`

	// RampUp is the duration to linearly ramp up to the target level.
	RampUp time.Duration `mapstructure:"ramp_up"`

	// RunPeriod is the steady-state execution duration (required).
	RunPeriod time.Duration `mapstructure:"run_period"`

	// RampDown is the duration for graceful ramp-down.
	RampDown time.Duration `mapstructure:"ramp_down"`

	// VUTimeout is the per-iteration context timeout (required).
	VUTimeout time.Duration `mapstructure:"vu_timeout"`

	// Params is an arbitrary key-value map available to test code via ScenarioContext.Param().
	Params map[string]string `mapstructure:"params"`

	// InteractionDelay defines think time strategy between actions (via ctx.Sleep).
	InteractionDelay *InteractionDelayConfig `mapstructure:"interaction_delay"`

	// ThinkTime defines inter-iteration pacing delay automatically executed by the engine loop.
	ThinkTime *ThinkTimeConfig `mapstructure:"think_time"`

	// Thresholds defines SLA assertions evaluated after the test run.
	Thresholds []ThresholdConfig `mapstructure:"thresholds"`
}

// ThinkTimeConfig holds configuration for inter-iteration think time delays.
type ThinkTimeConfig struct {
	// Type is the strategy type: "fixed", "range", "expo", "gaussian".
	Type string `mapstructure:"type"`

	// Duration is the static pause duration (used for "fixed").
	Duration time.Duration `mapstructure:"duration"`

	// Min is the minimum duration (used for "range", optional clamp for "expo" and "gaussian").
	Min time.Duration `mapstructure:"min"`

	// Max is the maximum duration (used for "range", optional clamp for "expo" and "gaussian").
	Max time.Duration `mapstructure:"max"`

	// Mean is the average duration (used for "expo" and "gaussian").
	Mean time.Duration `mapstructure:"mean"`

	// StdDev is the standard deviation (used for "gaussian").
	StdDev time.Duration `mapstructure:"std_dev"`
}

// InteractionDelayConfig holds configuration for in-iteration think time delays.
type InteractionDelayConfig = ThinkTimeConfig



// ThresholdConfig defines a single SLA threshold assertion.
type ThresholdConfig struct {
	// Metric is the exact metric name as recorded by the test developer.
	Metric string `mapstructure:"metric"`

	// Stat is the statistic to evaluate (p50, p90, p95, p99, mean, max, count, rate, value).
	Stat string `mapstructure:"stat"`

	// Operator is the comparison operator: <, <=, >, >=.
	Operator string `mapstructure:"operator"`

	// Target is the threshold value as a raw string.
	// Parsed as time.Duration for duration stats (p50, p90, p95, p99, mean, max),
	// or as float64 for non-duration stats (count, rate, value).
	Target string `mapstructure:"target"`

	// AbortOnFail triggers early test termination if breached during execution.
	AbortOnFail bool `mapstructure:"abort_on_fail"`

	// DelayAbortEval is a warm-up grace period before abort evaluation begins.
	DelayAbortEval time.Duration `mapstructure:"delay_abort_eval"`

	// TargetDuration is the parsed target when Stat is a duration stat.
	// Populated by validation; zero value if not applicable.
	TargetDuration time.Duration `mapstructure:"-"`

	// TargetFloat is the parsed target when Stat is a non-duration stat.
	// Populated by validation; zero value if not applicable.
	TargetFloat float64 `mapstructure:"-"`
}

// durationStats is the set of stat values that require a time.Duration target.
var durationStats = map[string]bool{
	"p50":  true,
	"p90":  true,
	"p95":  true,
	"p99":  true,
	"mean": true,
	"max":  true,
}

// IsDurationStat reports whether the stat requires a time.Duration target.
func IsDurationStat(stat string) bool {
	return durationStats[stat]
}

// validStats is the complete set of supported stat values.
var validStats = map[string]bool{
	"p50":   true,
	"p90":   true,
	"p95":   true,
	"p99":   true,
	"mean":  true,
	"max":   true,
	"count": true,
	"rate":  true,
	"value": true,
}

// validOperators is the set of supported comparison operators.
var validOperators = map[string]bool{
	"<":  true,
	"<=": true,
	">":  true,
	">=": true,
}
