package config_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/morphy76/gtest/internal/config"
	"github.com/morphy76/gtest/pkg/gtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validYAML = `
version: "1.0"
default_scenario: "checkout"

scenarios:
  checkout:
    type: "constant_vus"
    vus: 10
    ramp_up: "5s"
    run_period: "30s"
    ramp_down: "3s"
    vu_timeout: "2s"
    params:
      base_url: "https://api.example.com"
    thresholds:
      - metric: "http_request_duration"
        stat: "p95"
        operator: "<"
        target: "200ms"
      - metric: "success_rate"
        stat: "rate"
        operator: ">"
        target: "0.995"

  payment:
    type: "arrival_rate"
    target_tps: 100
    max_vus: 50
    ramp_up: "10s"
    run_period: "1m"
    ramp_down: "5s"
    vu_timeout: "3s"
    thresholds:
      - metric: "payment_duration"
        stat: "p99"
        operator: "<"
        target: "500ms"
`

// AC-1.2.1: Valid YAML round-trips correctly
func TestValidYAMLRoundTrips(t *testing.T) {
	cfg, err := config.Load(strings.NewReader(validYAML))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "1.0", cfg.Version)
	assert.Equal(t, "checkout", cfg.DefaultScenario)
	assert.Len(t, cfg.Scenarios, 2)

	// Verify checkout scenario.
	checkout, ok := cfg.Scenarios["checkout"]
	require.True(t, ok, "checkout scenario must exist")
	assert.Equal(t, config.ScenarioTypeConstantVUs, checkout.Type)
	assert.Equal(t, 10, checkout.VUs)
	assert.Equal(t, 5*time.Second, checkout.RampUp)
	assert.Equal(t, 30*time.Second, checkout.RunPeriod)
	assert.Equal(t, 3*time.Second, checkout.RampDown)
	assert.Equal(t, 2*time.Second, checkout.VUTimeout)
	assert.Equal(t, "https://api.example.com", checkout.Params["base_url"])
	assert.Len(t, checkout.Thresholds, 2)

	// Verify payment scenario (arrival_rate).
	payment, ok := cfg.Scenarios["payment"]
	require.True(t, ok, "payment scenario must exist")
	assert.Equal(t, config.ScenarioTypeArrivalRate, payment.Type)
	assert.Equal(t, 100, payment.TargetTPS)
	assert.Equal(t, 50, payment.MaxVUs)
	assert.Equal(t, 10*time.Second, payment.RampUp)
	assert.Equal(t, 1*time.Minute, payment.RunPeriod)
	assert.Equal(t, 5*time.Second, payment.RampDown)
	assert.Equal(t, 3*time.Second, payment.VUTimeout)
}

// AC-1.2.2: Missing required field returns ValidationError
func TestMissingRequiredFieldReturnsValidationError(t *testing.T) {
	tests := []struct {
		name  string
		yaml  string
		field string
	}{
		{
			name: "missing version",
			yaml: `
scenarios:
  s1:
    type: "constant_vus"
    vus: 1
    run_period: "10s"
    vu_timeout: "1s"
`,
			field: "version",
		},
		{
			name: "missing scenarios",
			yaml: `
version: "1.0"
`,
			field: "scenarios",
		},
		{
			name: "missing type",
			yaml: `
version: "1.0"
scenarios:
  s1:
    vus: 1
    run_period: "10s"
    vu_timeout: "1s"
`,
			field: "scenarios.s1.type",
		},
		{
			name: "missing run_period",
			yaml: `
version: "1.0"
scenarios:
  s1:
    type: "constant_vus"
    vus: 1
    vu_timeout: "1s"
`,
			field: "scenarios.s1.run_period",
		},
		{
			name: "missing vu_timeout",
			yaml: `
version: "1.0"
scenarios:
  s1:
    type: "constant_vus"
    vus: 1
    run_period: "10s"
`,
			field: "scenarios.s1.vu_timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := config.Load(strings.NewReader(tt.yaml))
			require.Error(t, err)

			var valErr *gtest.ValidationError
			require.True(t, errors.As(err, &valErr), "expected *gtest.ValidationError, got %T: %v", err, err)
			assert.Equal(t, tt.field, valErr.Field)
		})
	}
}

// AC-1.2.3: Unknown scenario in default_scenario returns ValidationError
func TestUnknownDefaultScenarioReturnsValidationError(t *testing.T) {
	yaml := `
version: "1.0"
default_scenario: "nonexistent"

scenarios:
  s1:
    type: "constant_vus"
    vus: 1
    run_period: "10s"
    vu_timeout: "1s"
`
	_, err := config.Load(strings.NewReader(yaml))
	require.Error(t, err)

	var valErr *gtest.ValidationError
	require.True(t, errors.As(err, &valErr), "expected *gtest.ValidationError, got %T", err)
	assert.Equal(t, "default_scenario", valErr.Field)
	assert.Contains(t, valErr.Message, "nonexistent")
}

// AC-1.2.4: arrival_rate without target_tps returns ValidationError
func TestArrivalRateWithoutTargetTPSReturnsValidationError(t *testing.T) {
	yaml := `
version: "1.0"

scenarios:
  s1:
    type: "arrival_rate"
    max_vus: 10
    run_period: "10s"
    vu_timeout: "1s"
`
	_, err := config.Load(strings.NewReader(yaml))
	require.Error(t, err)

	var valErr *gtest.ValidationError
	require.True(t, errors.As(err, &valErr), "expected *gtest.ValidationError, got %T", err)
	assert.Equal(t, "scenarios.s1.target_tps", valErr.Field)
}

// AC-1.2.5: constant_vus without vus returns ValidationError
func TestConstantVUsWithoutVUsReturnsValidationError(t *testing.T) {
	yaml := `
version: "1.0"

scenarios:
  s1:
    type: "constant_vus"
    run_period: "10s"
    vu_timeout: "1s"
`
	_, err := config.Load(strings.NewReader(yaml))
	require.Error(t, err)

	var valErr *gtest.ValidationError
	require.True(t, errors.As(err, &valErr), "expected *gtest.ValidationError, got %T", err)
	assert.Equal(t, "scenarios.s1.vus", valErr.Field)
}

// AC-1.2.6: threshold with invalid operator returns ValidationError
func TestThresholdWithInvalidOperatorReturnsValidationError(t *testing.T) {
	yaml := `
version: "1.0"

scenarios:
  s1:
    type: "constant_vus"
    vus: 1
    run_period: "10s"
    vu_timeout: "1s"
    thresholds:
      - metric: "latency"
        stat: "p95"
        operator: "=="
        target: "200ms"
`
	_, err := config.Load(strings.NewReader(yaml))
	require.Error(t, err)

	var valErr *gtest.ValidationError
	require.True(t, errors.As(err, &valErr), "expected *gtest.ValidationError, got %T", err)
	assert.Contains(t, valErr.Field, "operator")
}

// AC-1.2.7: threshold target "200ms" parses correctly for Duration stats
func TestThresholdDurationTargetParsesCorrectly(t *testing.T) {
	yaml := `
version: "1.0"

scenarios:
  s1:
    type: "constant_vus"
    vus: 1
    run_period: "10s"
    vu_timeout: "1s"
    thresholds:
      - metric: "latency"
        stat: "p95"
        operator: "<"
        target: "200ms"
`
	cfg, err := config.Load(strings.NewReader(yaml))
	require.NoError(t, err)

	th := cfg.Scenarios["s1"].Thresholds[0]
	assert.Equal(t, 200*time.Millisecond, th.TargetDuration)
	assert.Equal(t, float64(0), th.TargetFloat)
}

// AC-1.2.8: threshold target "0.005" parses correctly for rate stat
func TestThresholdRateTargetParsesCorrectly(t *testing.T) {
	yaml := `
version: "1.0"

scenarios:
  s1:
    type: "constant_vus"
    vus: 1
    run_period: "10s"
    vu_timeout: "1s"
    thresholds:
      - metric: "error_rate"
        stat: "rate"
        operator: "<"
        target: "0.005"
`
	cfg, err := config.Load(strings.NewReader(yaml))
	require.NoError(t, err)

	th := cfg.Scenarios["s1"].Thresholds[0]
	assert.InDelta(t, 0.005, th.TargetFloat, 1e-9)
	assert.Equal(t, time.Duration(0), th.TargetDuration)
}

// AC-1.2.9: threshold target "200ms" for "rate" stat returns ValidationError
func TestThresholdDurationTargetForRateStatReturnsValidationError(t *testing.T) {
	yaml := `
version: "1.0"

scenarios:
  s1:
    type: "constant_vus"
    vus: 1
    run_period: "10s"
    vu_timeout: "1s"
    thresholds:
      - metric: "error_rate"
        stat: "rate"
        operator: "<"
        target: "200ms"
`
	_, err := config.Load(strings.NewReader(yaml))
	require.Error(t, err)

	var valErr *gtest.ValidationError
	require.True(t, errors.As(err, &valErr), "expected *gtest.ValidationError, got %T", err)
	assert.Contains(t, valErr.Field, "target")
}

// Additional: arrival_rate without max_vus returns ValidationError
func TestArrivalRateWithoutMaxVUsReturnsValidationError(t *testing.T) {
	yaml := `
version: "1.0"

scenarios:
  s1:
    type: "arrival_rate"
    target_tps: 10
    run_period: "10s"
    vu_timeout: "1s"
`
	_, err := config.Load(strings.NewReader(yaml))
	require.Error(t, err)

	var valErr *gtest.ValidationError
	require.True(t, errors.As(err, &valErr), "expected *gtest.ValidationError, got %T", err)
	assert.Equal(t, "scenarios.s1.max_vus", valErr.Field)
}

// Additional: invalid YAML returns ConfigError
func TestInvalidYAMLReturnsConfigError(t *testing.T) {
	_, err := config.Load(strings.NewReader("{{invalid yaml"))
	require.Error(t, err)

	var cfgErr *gtest.ConfigError
	require.True(t, errors.As(err, &cfgErr), "expected *gtest.ConfigError, got %T", err)
}

// Additional: wrong version returns ValidationError
func TestWrongVersionReturnsValidationError(t *testing.T) {
	yaml := `
version: "2.0"

scenarios:
  s1:
    type: "constant_vus"
    vus: 1
    run_period: "10s"
    vu_timeout: "1s"
`
	_, err := config.Load(strings.NewReader(yaml))
	require.Error(t, err)

	var valErr *gtest.ValidationError
	require.True(t, errors.As(err, &valErr), "expected *gtest.ValidationError, got %T", err)
	assert.Equal(t, "version", valErr.Field)
}

// Additional: threshold with invalid stat returns ValidationError
func TestThresholdWithInvalidStatReturnsValidationError(t *testing.T) {
	yaml := `
version: "1.0"

scenarios:
  s1:
    type: "constant_vus"
    vus: 1
    run_period: "10s"
    vu_timeout: "1s"
    thresholds:
      - metric: "latency"
        stat: "p100"
        operator: "<"
        target: "200ms"
`
	_, err := config.Load(strings.NewReader(yaml))
	require.Error(t, err)

	var valErr *gtest.ValidationError
	require.True(t, errors.As(err, &valErr), "expected *gtest.ValidationError, got %T", err)
	assert.Contains(t, valErr.Field, "stat")
}

// Additional: all valid duration stats are accepted
func TestAllDurationStatsAccepted(t *testing.T) {
	for _, stat := range []string{"p50", "p90", "p95", "p99", "mean", "max"} {
		t.Run(stat, func(t *testing.T) {
			assert.True(t, config.IsDurationStat(stat))

			yaml := `
version: "1.0"

scenarios:
  s1:
    type: "constant_vus"
    vus: 1
    run_period: "10s"
    vu_timeout: "1s"
    thresholds:
      - metric: "latency"
        stat: "` + stat + `"
        operator: "<"
        target: "100ms"
`
			cfg, err := config.Load(strings.NewReader(yaml))
			require.NoError(t, err)
			assert.Equal(t, 100*time.Millisecond, cfg.Scenarios["s1"].Thresholds[0].TargetDuration)
		})
	}
}

// Additional: all valid non-duration stats are accepted
func TestAllNonDurationStatsAccepted(t *testing.T) {
	for _, stat := range []string{"count", "rate", "value"} {
		t.Run(stat, func(t *testing.T) {
			assert.False(t, config.IsDurationStat(stat))

			yaml := `
version: "1.0"

scenarios:
  s1:
    type: "constant_vus"
    vus: 1
    run_period: "10s"
    vu_timeout: "1s"
    thresholds:
      - metric: "some_metric"
        stat: "` + stat + `"
        operator: ">="
        target: "42.5"
`
			cfg, err := config.Load(strings.NewReader(yaml))
			require.NoError(t, err)
			assert.InDelta(t, 42.5, cfg.Scenarios["s1"].Thresholds[0].TargetFloat, 1e-9)
		})
	}
}

// Additional: all valid operators are accepted
func TestAllValidOperatorsAccepted(t *testing.T) {
	for _, op := range []string{"<", "<=", ">", ">="} {
		t.Run(op, func(t *testing.T) {
			yaml := `
version: "1.0"

scenarios:
  s1:
    type: "constant_vus"
    vus: 1
    run_period: "10s"
    vu_timeout: "1s"
    thresholds:
      - metric: "latency"
        stat: "p95"
        operator: "` + op + `"
        target: "200ms"
`
			_, err := config.Load(strings.NewReader(yaml))
			require.NoError(t, err)
		})
	}
}

// Additional: optional fields default to zero values
func TestOptionalFieldsDefaultToZero(t *testing.T) {
	yaml := `
version: "1.0"

scenarios:
  s1:
    type: "constant_vus"
    vus: 5
    run_period: "10s"
    vu_timeout: "1s"
`
	cfg, err := config.Load(strings.NewReader(yaml))
	require.NoError(t, err)

	s := cfg.Scenarios["s1"]
	assert.Equal(t, time.Duration(0), s.RampUp)
	assert.Equal(t, time.Duration(0), s.RampDown)
	assert.Empty(t, s.Params)
	assert.Empty(t, s.Thresholds)
	assert.Empty(t, cfg.DefaultScenario)
}

// Additional: LoadFromFile with nonexistent file returns ConfigError
func TestLoadFromFileNotFoundReturnsConfigError(t *testing.T) {
	_, err := config.LoadFromFile("/nonexistent/path/gtest.yaml")
	require.Error(t, err)

	var cfgErr *gtest.ConfigError
	require.True(t, errors.As(err, &cfgErr), "expected *gtest.ConfigError, got %T", err)
	assert.Equal(t, "/nonexistent/path/gtest.yaml", cfgErr.Path)
}
