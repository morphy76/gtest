package engine_test

import (
	"context"
	"testing"
	"time"

	"github.com/morphy76/vuhive/internal/config"
	"github.com/morphy76/vuhive/internal/engine"
	"github.com/morphy76/vuhive/internal/metric"
)

func BenchmarkEngine_ConstantVUs_NoopIteration(b *testing.B) {
	scenario := engine.Scenario{
		RunVU: func(ctx engine.VUContext) error {
			return nil
		},
	}

	cfg := config.ScenarioConfig{
		Type:      config.ScenarioTypeConstantVUs,
		VUs:       8,
		RunPeriod: 100 * time.Millisecond,
		VUTimeout: 1 * time.Second,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger, metrics := newTestDeps()
		exec := engine.NewExecutor("bench_constant_vus", scenario, cfg, logger, metrics)
		_ = exec.Execute(context.Background())
	}
}

func BenchmarkEngine_ConstantVUs_NoopIteration_NoTimeout(b *testing.B) {
	scenario := engine.Scenario{
		RunVU: func(ctx engine.VUContext) error {
			return nil
		},
	}

	cfg := config.ScenarioConfig{
		Type:      config.ScenarioTypeConstantVUs,
		VUs:       8,
		RunPeriod: 100 * time.Millisecond,
		VUTimeout: 0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger, metrics := newTestDeps()
		exec := engine.NewExecutor("bench_constant_vus_notimeout", scenario, cfg, logger, metrics)
		_ = exec.Execute(context.Background())
	}
}

func BenchmarkEngine_ArrivalRate_NoopIteration(b *testing.B) {
	scenario := engine.Scenario{
		RunVU: func(ctx engine.VUContext) error {
			return nil
		},
	}

	cfg := config.ScenarioConfig{
		Type:      config.ScenarioTypeArrivalRate,
		TargetTPS: 1000,
		MaxVUs:    16,
		RunPeriod: 100 * time.Millisecond,
		VUTimeout: 1 * time.Second,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger, metrics := newTestDeps()
		exec := engine.NewExecutor("bench_arrival_rate", scenario, cfg, logger, metrics)
		_ = exec.Execute(context.Background())
	}
}

func BenchmarkEngine_RampingVUs_NoopIteration(b *testing.B) {
	scenario := engine.Scenario{
		RunVU: func(ctx engine.VUContext) error {
			return nil
		},
	}

	cfg := config.ScenarioConfig{
		Type: config.ScenarioTypeRampingVUs,
		Stages: []config.StageConfig{
			{Target: 8, Duration: 50 * time.Millisecond},
			{Target: 8, Duration: 50 * time.Millisecond},
		},
		VUTimeout: 1 * time.Second,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger, metrics := newTestDeps()
		exec := engine.NewExecutor("bench_ramping_vus", scenario, cfg, logger, metrics)
		_ = exec.Execute(context.Background())
	}
}

func BenchmarkScenarioContext_Check(b *testing.B) {
	metrics := metric.NewStore()
	sCtx := engine.NewScenarioContext(context.Background(), 1, 0, config.ScenarioConfig{}, "test", nil, nil, metrics)

	checkFn := func() string { return "" }

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sCtx.Check("status_is_200", checkFn)
	}
}

func BenchmarkScenarioContext_ParamAccess(b *testing.B) {
	cfg := config.ScenarioConfig{
		Params: map[string]string{
			"url":     "http://localhost:8080",
			"retries": "3",
			"timeout": "500ms",
		},
	}
	sCtx := engine.NewScenarioContext(context.Background(), 1, 0, cfg, "test", nil, nil, nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = sCtx.Param("url")
		_ = sCtx.ParamInt("retries", 1)
		_ = sCtx.ParamDuration("timeout", time.Second)
	}
}
