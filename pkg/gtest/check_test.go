package gtest_test

import (
	"context"
	"io"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/morphy76/gtest/internal/config"
	"github.com/morphy76/gtest/internal/engine"
	"github.com/morphy76/gtest/internal/log"
	"github.com/morphy76/gtest/internal/metric"
	"github.com/morphy76/gtest/pkg/gtest"
)

func TestCheck_Passing(t *testing.T) {
	store := metric.NewStore()
	logger := log.New(io.Discard, zerolog.Disabled)
	ctx := engine.NewScenarioContext(
		context.Background(),
		1,
		1,
		config.ScenarioConfig{},
		"test_scenario",
		nil,
		logger,
		store,
	)

	var checkFn gtest.CheckFunc = func() string {
		return ""
	}

	passed := ctx.Check("status is 200", engine.CheckFunc(checkFn))

	assert.True(t, passed, "passing check should return true")
	assert.Equal(t, int64(1), store.AggregatedCounterValue(gtest.MetricChecksPassed))
	assert.Equal(t, int64(0), store.AggregatedCounterValue(gtest.MetricChecksFailed))
}

func TestCheck_Failing(t *testing.T) {
	store := metric.NewStore()
	logger := log.New(io.Discard, zerolog.Disabled)
	ctx := engine.NewScenarioContext(
		context.Background(),
		1,
		1,
		config.ScenarioConfig{},
		"test_scenario",
		nil,
		logger,
		store,
	)

	passed := ctx.Check("status is 200", func() string {
		return "expected 200 OK, got 500 Internal Server Error"
	})

	assert.False(t, passed, "failing check should return false")
	assert.Equal(t, int64(0), store.AggregatedCounterValue(gtest.MetricChecksPassed))
	assert.Equal(t, int64(1), store.AggregatedCounterValue(gtest.MetricChecksFailed))
}

func TestCheck_MultipleIndependent(t *testing.T) {
	store := metric.NewStore()
	logger := log.New(io.Discard, zerolog.Disabled)
	ctx := engine.NewScenarioContext(
		context.Background(),
		1,
		1,
		config.ScenarioConfig{},
		"test_scenario",
		nil,
		logger,
		store,
	)

	res1 := ctx.Check("check_1", func() string { return "" })
	res2 := ctx.Check("check_2", func() string { return "fail reason" })
	res3 := ctx.Check("check_3", func() string { return "" })

	assert.True(t, res1)
	assert.False(t, res2)
	assert.True(t, res3)

	assert.Equal(t, int64(2), store.AggregatedCounterValue(gtest.MetricChecksPassed))
	assert.Equal(t, int64(1), store.AggregatedCounterValue(gtest.MetricChecksFailed))
}
