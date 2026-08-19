package engine_test

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/morphy76/vuhive/internal/config"
	"github.com/morphy76/vuhive/internal/engine"
	"github.com/morphy76/vuhive/internal/log"
	"github.com/morphy76/vuhive/internal/metric"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestAlloc_ScenarioContext_Check_Passing(t *testing.T) {
	metrics := metric.NewStore()
	logger := log.New(io.Discard, zerolog.Disabled)
	sCtx := engine.NewScenarioContext(context.Background(), 1, 0, config.ScenarioConfig{}, "test_scenario", nil, logger, metrics)

	// Warm-up check cache
	sCtx.Check("status_200", func() string { return "" })

	allocs := testing.AllocsPerRun(1000, func() {
		_ = sCtx.Check("status_200", func() string { return "" })
	})

	assert.Equal(t, float64(0), allocs, "steady-state passing Check must produce 0 heap allocations")
}

func TestAlloc_ScenarioContext_Check_Failing_NoLogger(t *testing.T) {
	metrics := metric.NewStore()
	sCtx := engine.NewScenarioContext(context.Background(), 1, 0, config.ScenarioConfig{}, "test_scenario", nil, nil, metrics)

	// Warm-up check cache
	sCtx.Check("status_200", func() string { return "expected 200, got 500" })

	allocs := testing.AllocsPerRun(1000, func() {
		_ = sCtx.Check("status_200", func() string { return "expected 200, got 500" })
	})

	assert.Equal(t, float64(0), allocs, "steady-state failing Check with nil logger must produce 0 heap allocations")
}

func TestAlloc_ScenarioContext_ParamAccess(t *testing.T) {
	cfg := config.ScenarioConfig{
		Params: map[string]string{
			"url":     "http://localhost:8080/api",
			"retries": "5",
			"timeout": "250ms",
		},
	}
	sCtx := engine.NewScenarioContext(context.Background(), 1, 0, cfg, "test_scenario", nil, nil, nil)

	allocs := testing.AllocsPerRun(1000, func() {
		_ = sCtx.Param("url")
		_ = sCtx.ParamInt("retries", 1)
		_ = sCtx.ParamDuration("timeout", time.Second)
	})

	assert.Equal(t, float64(0), allocs, "typed parameter access must produce 0 heap allocations")
}
