package engine

import (
	"context"
	"strconv"
	"time"

	"github.com/morphy76/gtest/internal/config"
	"github.com/morphy76/gtest/internal/log"
	"github.com/morphy76/gtest/internal/metric"
)

type scenarioBinder interface {
	WithScenario(string) any
	WithVU(int) any
	WithIteration(int64) any
}

type scenarioContext struct {
	context.Context
	vuid         int64
	iteration    int64
	scenarioName string
	params       map[string]string
	globalState  map[string]any
	logger       log.Logger
	metrics      metric.Collector
}

func newScenarioContext(
	ctx context.Context,
	vuid int64,
	iteration int64,
	cfg config.ScenarioConfig,
	scenarioName string,
	globalState map[string]any,
	logger log.Logger,
	metrics metric.Collector,
) *scenarioContext {
	boundLogger := logger
	if b, ok := logger.(scenarioBinder); ok {
		if s, ok := b.WithScenario(scenarioName).(scenarioBinder); ok {
			if v, ok := s.WithVU(int(vuid)).(scenarioBinder); ok {
				if i, ok := v.WithIteration(iteration).(log.Logger); ok {
					boundLogger = i
				}
			}
		}
	}

	return &scenarioContext{
		Context:      ctx,
		vuid:         vuid,
		iteration:    iteration,
		scenarioName: scenarioName,
		params:       cfg.Params,
		globalState:  globalState,
		logger:       boundLogger,
		metrics:      metrics,
	}
}

func (c *scenarioContext) VUID() int64 {
	return c.vuid
}

func (c *scenarioContext) Iteration() int64 {
	return c.iteration
}

func (c *scenarioContext) ScenarioName() string {
	return c.scenarioName
}

func (c *scenarioContext) Param(key string) string {
	if c.params == nil {
		return ""
	}
	return c.params[key]
}

func (c *scenarioContext) ParamInt(key string, defaultValue int) int {
	v := c.Param(key)
	if v == "" {
		return defaultValue
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return defaultValue
	}
	return i
}

func (c *scenarioContext) ParamDuration(key string, defaultValue time.Duration) time.Duration {
	v := c.Param(key)
	if v == "" {
		return defaultValue
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return defaultValue
	}
	return d
}

func (c *scenarioContext) GlobalState(key string) any {
	if c.globalState == nil {
		return nil
	}
	return c.globalState[key]
}

func (c *scenarioContext) Log() log.Logger {
	return c.logger
}

func (c *scenarioContext) Metrics() metric.Collector {
	return c.metrics
}

// Compile-time interface satisfaction check.
var _ ScenarioContext = (*scenarioContext)(nil)
