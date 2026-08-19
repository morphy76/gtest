package config

import (
	"fmt"
	"reflect"
	"time"

	"github.com/go-viper/mapstructure/v2"
)

// configDTO is the top-level configuration DTO loaded from YAML via mapstructure.
type configDTO struct {
	Version         string                       `mapstructure:"version"`
	DefaultScenario string                       `mapstructure:"default_scenario"`
	Scenarios       map[string]scenarioConfigDTO `mapstructure:"scenarios"`
}

// scenarioConfigDTO is the DTO for a single scenario configuration.
type scenarioConfigDTO struct {
	Type             ScenarioType         `mapstructure:"type"`
	VUs              int                  `mapstructure:"vus"`
	TargetTPS        int                  `mapstructure:"target_tps"`
	MaxVUs           int                  `mapstructure:"max_vus"`
	Stages           []stageConfigDTO     `mapstructure:"stages"`
	RampUp           time.Duration        `mapstructure:"ramp_up"`
	RunPeriod        time.Duration        `mapstructure:"run_period"`
	RampDown         time.Duration        `mapstructure:"ramp_down"`
	VUTimeout        time.Duration        `mapstructure:"vu_timeout"`
	Params           map[string]string    `mapstructure:"params"`
	InteractionDelay *thinkTimeConfigDTO  `mapstructure:"interaction_delay"`
	ThinkTime        *thinkTimeConfigDTO  `mapstructure:"think_time"`
	Thresholds       []thresholdConfigDTO `mapstructure:"thresholds"`
}

// stageConfigDTO is the DTO for a single stage in ramping_vus scenarios.
type stageConfigDTO struct {
	Target   int           `mapstructure:"target"`
	Duration time.Duration `mapstructure:"duration"`
}

// thinkTimeConfigDTO is the DTO for think time and interaction delay configurations.
type thinkTimeConfigDTO struct {
	Type     string        `mapstructure:"type"`
	Duration time.Duration `mapstructure:"duration"`
	Min      time.Duration `mapstructure:"min"`
	Max      time.Duration `mapstructure:"max"`
	Mean     time.Duration `mapstructure:"mean"`
	StdDev   time.Duration `mapstructure:"std_dev"`
}

// thresholdConfigDTO is the DTO for SLA threshold assertions.
type thresholdConfigDTO struct {
	Metric         string        `mapstructure:"metric"`
	Stat           string        `mapstructure:"stat"`
	Operator       string        `mapstructure:"operator"`
	Target         string        `mapstructure:"target"`
	OnNoData       string        `mapstructure:"on_no_data"`
	AbortOnFail    bool          `mapstructure:"abort_on_fail"`
	DelayAbortEval time.Duration `mapstructure:"delay_abort_eval"`
}

// toModel converts a configDTO to a pure domain Config model.
func (d *configDTO) toModel() *Config {
	if d == nil {
		return nil
	}
	cfg := &Config{
		Version:         d.Version,
		DefaultScenario: d.DefaultScenario,
	}
	if d.Scenarios != nil {
		cfg.Scenarios = make(map[string]ScenarioConfig, len(d.Scenarios))
		for k, sc := range d.Scenarios {
			cfg.Scenarios[k] = sc.toModel()
		}
	}
	return cfg
}

// toModel converts a stageConfigDTO to a pure domain StageConfig model.
func (d *stageConfigDTO) toModel() StageConfig {
	return StageConfig{
		Target:   d.Target,
		Duration: d.Duration,
	}
}

// toModel converts a scenarioConfigDTO to a pure domain ScenarioConfig model.
func (d *scenarioConfigDTO) toModel() ScenarioConfig {
	sc := ScenarioConfig{
		Type:      d.Type,
		VUs:       d.VUs,
		TargetTPS: d.TargetTPS,
		MaxVUs:    d.MaxVUs,
		RampUp:    d.RampUp,
		RunPeriod: d.RunPeriod,
		RampDown:  d.RampDown,
		VUTimeout: d.VUTimeout,
	}
	if d.Stages != nil {
		sc.Stages = make([]StageConfig, len(d.Stages))
		for i, st := range d.Stages {
			sc.Stages[i] = st.toModel()
		}
	}
	if d.Params != nil {
		sc.Params = make(map[string]string, len(d.Params))
		for k, v := range d.Params {
			sc.Params[k] = v
		}
	}
	if d.InteractionDelay != nil {
		sc.InteractionDelay = d.InteractionDelay.toModel()
	}
	if d.ThinkTime != nil {
		sc.ThinkTime = d.ThinkTime.toModel()
	}
	if d.Thresholds != nil {
		sc.Thresholds = make([]ThresholdConfig, len(d.Thresholds))
		for i, th := range d.Thresholds {
			sc.Thresholds[i] = th.toModel()
		}
	}
	return sc
}

// toModel converts a thinkTimeConfigDTO to a pure domain ThinkTimeConfig model.
func (d *thinkTimeConfigDTO) toModel() *ThinkTimeConfig {
	if d == nil {
		return nil
	}
	return &ThinkTimeConfig{
		Type:     d.Type,
		Duration: d.Duration,
		Min:      d.Min,
		Max:      d.Max,
		Mean:     d.Mean,
		StdDev:   d.StdDev,
	}
}

// toModel converts a thresholdConfigDTO to a pure domain ThresholdConfig model.
func (d *thresholdConfigDTO) toModel() ThresholdConfig {
	return ThresholdConfig{
		Metric:         d.Metric,
		Stat:           d.Stat,
		Operator:       d.Operator,
		Target:         d.Target,
		OnNoData:       d.OnNoData,
		AbortOnFail:    d.AbortOnFail,
		DelayAbortEval: d.DelayAbortEval,
	}
}

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
