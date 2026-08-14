package config_test

import (
	"errors"
	"testing"
	"time"

	"github.com/morphy76/gtest/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsDurationStat(t *testing.T) {
	durationStats := []string{"p50", "p90", "p95", "p99", "mean", "max"}
	for _, stat := range durationStats {
		t.Run(stat+" is duration stat", func(t *testing.T) {
			assert.True(t, config.IsDurationStat(stat))
		})
	}

	nonDurationStats := []string{"count", "rate", "value", "unknown", ""}
	for _, stat := range nonDurationStats {
		t.Run(stat+" is not duration stat", func(t *testing.T) {
			assert.False(t, config.IsDurationStat(stat))
		})
	}
}

func TestValidate_CustomDelayValidatorRegistry(t *testing.T) {
	// Register a custom delay strategy validator.
	config.RegisterDelayValidator("custom_step", func(prefix string, delay *config.ThinkTimeConfig) error {
		if delay.Duration <= 0 {
			return &config.ValidationError{
				Field:   prefix + ".duration",
				Message: "must be > 0 for custom_step delay",
			}
		}
		return nil
	})

	t.Run("custom delay valid", func(t *testing.T) {
		cfg := &config.Config{
			Version: "1.0",
			Scenarios: map[string]config.ScenarioConfig{
				"s1": {
					Type:      config.ScenarioTypeConstantVUs,
					VUs:       1,
					RunPeriod: 10 * time.Second,
					VUTimeout: 1 * time.Second,
					InteractionDelay: &config.InteractionDelayConfig{
						Type:     "custom_step",
						Duration: 100 * time.Millisecond,
					},
				},
			},
		}
		err := config.Validate(cfg)
		require.NoError(t, err)
	})

	t.Run("custom delay invalid", func(t *testing.T) {
		cfg := &config.Config{
			Version: "1.0",
			Scenarios: map[string]config.ScenarioConfig{
				"s1": {
					Type:      config.ScenarioTypeConstantVUs,
					VUs:       1,
					RunPeriod: 10 * time.Second,
					VUTimeout: 1 * time.Second,
					InteractionDelay: &config.InteractionDelayConfig{
						Type:     "custom_step",
						Duration: 0,
					},
				},
			},
		}
		err := config.Validate(cfg)
		require.Error(t, err)
		var valErr *config.ValidationError
		require.True(t, errors.As(err, &valErr))
		assert.Equal(t, "scenarios.s1.interaction_delay.duration", valErr.Field)
		assert.Contains(t, valErr.Message, "must be > 0 for custom_step delay")
	})
}

func TestValidate_DelayStrategyEdgeCases(t *testing.T) {
	baseCfg := func(delay *config.ThinkTimeConfig) *config.Config {
		return &config.Config{
			Version: "1.0",
			Scenarios: map[string]config.ScenarioConfig{
				"s1": {
					Type:             config.ScenarioTypeConstantVUs,
					VUs:              1,
					RunPeriod:        10 * time.Second,
					VUTimeout:        1 * time.Second,
					InteractionDelay: delay,
				},
			},
		}
	}

	tests := []struct {
		name      string
		delay     *config.ThinkTimeConfig
		wantField string
	}{
		{
			name: "range delay min negative",
			delay: &config.ThinkTimeConfig{
				Type: "range",
				Min:  -1 * time.Second,
				Max:  1 * time.Second,
			},
			wantField: "scenarios.s1.interaction_delay.min",
		},
		{
			name: "range delay max zero",
			delay: &config.ThinkTimeConfig{
				Type: "range",
				Min:  0,
				Max:  0,
			},
			wantField: "scenarios.s1.interaction_delay.max",
		},
		{
			name: "expo delay min negative",
			delay: &config.ThinkTimeConfig{
				Type: "expo",
				Mean: 500 * time.Millisecond,
				Min:  -1 * time.Millisecond,
			},
			wantField: "scenarios.s1.interaction_delay.min",
		},
		{
			name: "expo delay max negative",
			delay: &config.ThinkTimeConfig{
				Type: "expo",
				Mean: 500 * time.Millisecond,
				Max:  -1 * time.Millisecond,
			},
			wantField: "scenarios.s1.interaction_delay.max",
		},
		{
			name: "expo delay max less than min",
			delay: &config.ThinkTimeConfig{
				Type: "expo",
				Mean: 500 * time.Millisecond,
				Min:  500 * time.Millisecond,
				Max:  100 * time.Millisecond,
			},
			wantField: "scenarios.s1.interaction_delay.max",
		},
		{
			name: "gaussian delay mean zero",
			delay: &config.ThinkTimeConfig{
				Type:   "gaussian",
				Mean:   0,
				StdDev: 10 * time.Millisecond,
			},
			wantField: "scenarios.s1.interaction_delay.mean",
		},
		{
			name: "gaussian delay stddev zero",
			delay: &config.ThinkTimeConfig{
				Type:   "gaussian",
				Mean:   100 * time.Millisecond,
				StdDev: 0,
			},
			wantField: "scenarios.s1.interaction_delay.std_dev",
		},
		{
			name: "gaussian delay min negative",
			delay: &config.ThinkTimeConfig{
				Type:   "gaussian",
				Mean:   100 * time.Millisecond,
				StdDev: 10 * time.Millisecond,
				Min:    -10 * time.Millisecond,
			},
			wantField: "scenarios.s1.interaction_delay.min",
		},
		{
			name: "gaussian delay max negative",
			delay: &config.ThinkTimeConfig{
				Type:   "gaussian",
				Mean:   100 * time.Millisecond,
				StdDev: 10 * time.Millisecond,
				Max:    -10 * time.Millisecond,
			},
			wantField: "scenarios.s1.interaction_delay.max",
		},
		{
			name: "gaussian delay max less than min",
			delay: &config.ThinkTimeConfig{
				Type:   "gaussian",
				Mean:   100 * time.Millisecond,
				StdDev: 10 * time.Millisecond,
				Min:    200 * time.Millisecond,
				Max:    50 * time.Millisecond,
			},
			wantField: "scenarios.s1.interaction_delay.max",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := baseCfg(tt.delay)
			err := config.Validate(cfg)
			require.Error(t, err)
			var valErr *config.ValidationError
			require.True(t, errors.As(err, &valErr))
			assert.Equal(t, tt.wantField, valErr.Field)
		})
	}
}

func TestValidate_ScenarioEdgeCases(t *testing.T) {
	t.Run("negative ramp_up", func(t *testing.T) {
		cfg := &config.Config{
			Version: "1.0",
			Scenarios: map[string]config.ScenarioConfig{
				"s1": {
					Type:      config.ScenarioTypeConstantVUs,
					VUs:       1,
					RunPeriod: 10 * time.Second,
					VUTimeout: 1 * time.Second,
					RampUp:    -1 * time.Second,
				},
			},
		}
		err := config.Validate(cfg)
		require.Error(t, err)
		var valErr *config.ValidationError
		require.True(t, errors.As(err, &valErr))
		assert.Equal(t, "scenarios.s1.ramp_up", valErr.Field)
	})

	t.Run("negative ramp_down", func(t *testing.T) {
		cfg := &config.Config{
			Version: "1.0",
			Scenarios: map[string]config.ScenarioConfig{
				"s1": {
					Type:      config.ScenarioTypeConstantVUs,
					VUs:       1,
					RunPeriod: 10 * time.Second,
					VUTimeout: 1 * time.Second,
					RampDown:  -1 * time.Second,
				},
			},
		}
		err := config.Validate(cfg)
		require.Error(t, err)
		var valErr *config.ValidationError
		require.True(t, errors.As(err, &valErr))
		assert.Equal(t, "scenarios.s1.ramp_down", valErr.Field)
	})

	t.Run("empty param key", func(t *testing.T) {
		cfg := &config.Config{
			Version: "1.0",
			Scenarios: map[string]config.ScenarioConfig{
				"s1": {
					Type:      config.ScenarioTypeConstantVUs,
					VUs:       1,
					RunPeriod: 10 * time.Second,
					VUTimeout: 1 * time.Second,
					Params:    map[string]string{"": "val"},
				},
			},
		}
		err := config.Validate(cfg)
		require.Error(t, err)
		var valErr *config.ValidationError
		require.True(t, errors.As(err, &valErr))
		assert.Equal(t, "scenarios.s1.params", valErr.Field)
	})

	t.Run("empty param value", func(t *testing.T) {
		cfg := &config.Config{
			Version: "1.0",
			Scenarios: map[string]config.ScenarioConfig{
				"s1": {
					Type:      config.ScenarioTypeConstantVUs,
					VUs:       1,
					RunPeriod: 10 * time.Second,
					VUTimeout: 1 * time.Second,
					Params:    map[string]string{"key": ""},
				},
			},
		}
		err := config.Validate(cfg)
		require.Error(t, err)
		var valErr *config.ValidationError
		require.True(t, errors.As(err, &valErr))
		assert.Equal(t, "scenarios.s1.params.key", valErr.Field)
	})

	t.Run("threshold metric empty", func(t *testing.T) {
		cfg := &config.Config{
			Version: "1.0",
			Scenarios: map[string]config.ScenarioConfig{
				"s1": {
					Type:      config.ScenarioTypeConstantVUs,
					VUs:       1,
					RunPeriod: 10 * time.Second,
					VUTimeout: 1 * time.Second,
					Thresholds: []config.ThresholdConfig{
						{Metric: "", Stat: "p95", Operator: "<", Target: "100ms"},
					},
				},
			},
		}
		err := config.Validate(cfg)
		require.Error(t, err)
		var valErr *config.ValidationError
		require.True(t, errors.As(err, &valErr))
		assert.Equal(t, "scenarios.s1.thresholds[0].metric", valErr.Field)
	})

	t.Run("threshold target empty", func(t *testing.T) {
		cfg := &config.Config{
			Version: "1.0",
			Scenarios: map[string]config.ScenarioConfig{
				"s1": {
					Type:      config.ScenarioTypeConstantVUs,
					VUs:       1,
					RunPeriod: 10 * time.Second,
					VUTimeout: 1 * time.Second,
					Thresholds: []config.ThresholdConfig{
						{Metric: "m", Stat: "p95", Operator: "<", Target: ""},
					},
				},
			},
		}
		err := config.Validate(cfg)
		require.Error(t, err)
		var valErr *config.ValidationError
		require.True(t, errors.As(err, &valErr))
		assert.Equal(t, "scenarios.s1.thresholds[0].target", valErr.Field)
	})

	t.Run("threshold delay_abort_eval negative", func(t *testing.T) {
		cfg := &config.Config{
			Version: "1.0",
			Scenarios: map[string]config.ScenarioConfig{
				"s1": {
					Type:      config.ScenarioTypeConstantVUs,
					VUs:       1,
					RunPeriod: 10 * time.Second,
					VUTimeout: 1 * time.Second,
					Thresholds: []config.ThresholdConfig{
						{Metric: "m", Stat: "p95", Operator: "<", Target: "100ms", DelayAbortEval: -1 * time.Second},
					},
				},
			},
		}
		err := config.Validate(cfg)
		require.Error(t, err)
		var valErr *config.ValidationError
		require.True(t, errors.As(err, &valErr))
		assert.Equal(t, "scenarios.s1.thresholds[0].delay_abort_eval", valErr.Field)
	})
}
