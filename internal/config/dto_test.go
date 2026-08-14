package config

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDomainModels_HaveNoStructTags(t *testing.T) {
	typesToCheck := []struct {
		name string
		val  any
	}{
		{"Config", Config{}},
		{"ScenarioConfig", ScenarioConfig{}},
		{"ThinkTimeConfig", ThinkTimeConfig{}},
		{"ThresholdConfig", ThresholdConfig{}},
	}

	for _, tc := range typesToCheck {
		t.Run(tc.name, func(t *testing.T) {
			rt := reflect.TypeOf(tc.val)
			for i := 0; i < rt.NumField(); i++ {
				f := rt.Field(i)
				tag := f.Tag
				assert.Empty(t, string(tag), "field %s on struct %s must not have struct tags, found %q", f.Name, tc.name, string(tag))
			}
		})
	}
}

func TestDTO_ToModel(t *testing.T) {
	dto := configDTO{
		Version:         "1.0",
		DefaultScenario: "test_scenario",
		Scenarios: map[string]scenarioConfigDTO{
			"test_scenario": {
				Type:      ScenarioTypeConstantVUs,
				VUs:       10,
				TargetTPS: 100,
				MaxVUs:    20,
				RampUp:    5 * time.Second,
				RunPeriod: 30 * time.Second,
				RampDown:  5 * time.Second,
				VUTimeout: 2 * time.Second,
				Params: map[string]string{
					"key1": "val1",
				},
				InteractionDelay: &thinkTimeConfigDTO{
					Type:     "fixed",
					Duration: 200 * time.Millisecond,
					Min:      100 * time.Millisecond,
					Max:      300 * time.Millisecond,
					Mean:     200 * time.Millisecond,
					StdDev:   50 * time.Millisecond,
				},
				ThinkTime: &thinkTimeConfigDTO{
					Type:     "range",
					Duration: 500 * time.Millisecond,
					Min:      200 * time.Millisecond,
					Max:      1 * time.Second,
					Mean:     500 * time.Millisecond,
					StdDev:   100 * time.Millisecond,
				},
				Thresholds: []thresholdConfigDTO{
					{
						Metric:         "http_req_duration",
						Stat:           "p95",
						Operator:       "<",
						Target:         "200ms",
						AbortOnFail:    true,
						DelayAbortEval: 5 * time.Second,
					},
				},
			},
		},
	}

	cfg := dto.toModel()
	require.NotNil(t, cfg)
	assert.Equal(t, "1.0", cfg.Version)
	assert.Equal(t, "test_scenario", cfg.DefaultScenario)
	require.Len(t, cfg.Scenarios, 1)

	sc, ok := cfg.Scenarios["test_scenario"]
	require.True(t, ok)
	assert.Equal(t, ScenarioTypeConstantVUs, sc.Type)
	assert.Equal(t, 10, sc.VUs)
	assert.Equal(t, 100, sc.TargetTPS)
	assert.Equal(t, 20, sc.MaxVUs)
	assert.Equal(t, 5*time.Second, sc.RampUp)
	assert.Equal(t, 30*time.Second, sc.RunPeriod)
	assert.Equal(t, 5*time.Second, sc.RampDown)
	assert.Equal(t, 2*time.Second, sc.VUTimeout)
	assert.Equal(t, map[string]string{"key1": "val1"}, sc.Params)

	require.NotNil(t, sc.InteractionDelay)
	assert.Equal(t, "fixed", sc.InteractionDelay.Type)
	assert.Equal(t, 200*time.Millisecond, sc.InteractionDelay.Duration)
	assert.Equal(t, 100*time.Millisecond, sc.InteractionDelay.Min)
	assert.Equal(t, 300*time.Millisecond, sc.InteractionDelay.Max)
	assert.Equal(t, 200*time.Millisecond, sc.InteractionDelay.Mean)
	assert.Equal(t, 50*time.Millisecond, sc.InteractionDelay.StdDev)

	require.NotNil(t, sc.ThinkTime)
	assert.Equal(t, "range", sc.ThinkTime.Type)
	assert.Equal(t, 500*time.Millisecond, sc.ThinkTime.Duration)
	assert.Equal(t, 200*time.Millisecond, sc.ThinkTime.Min)
	assert.Equal(t, 1*time.Second, sc.ThinkTime.Max)

	require.Len(t, sc.Thresholds, 1)
	th := sc.Thresholds[0]
	assert.Equal(t, "http_req_duration", th.Metric)
	assert.Equal(t, "p95", th.Stat)
	assert.Equal(t, "<", th.Operator)
	assert.Equal(t, "200ms", th.Target)
	assert.True(t, th.AbortOnFail)
	assert.Equal(t, 5*time.Second, th.DelayAbortEval)
}

func TestDTO_ToModel_NilFields(t *testing.T) {
	dto := configDTO{
		Scenarios: map[string]scenarioConfigDTO{
			"minimal": {
				Type: ScenarioTypeConstantVUs,
			},
		},
	}

	cfg := dto.toModel()
	require.NotNil(t, cfg)
	sc := cfg.Scenarios["minimal"]
	assert.Nil(t, sc.InteractionDelay)
	assert.Nil(t, sc.ThinkTime)
	assert.Nil(t, sc.Thresholds)
	assert.Nil(t, sc.Params)
}
