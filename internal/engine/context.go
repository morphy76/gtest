package engine

import (
	"context"
	"strconv"
	"time"

	"github.com/morphy76/gtest/internal/config"
	"github.com/morphy76/gtest/internal/delay"
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
	delayGen     delay.DelayGenerator
}

// NewScenarioContext constructs a ScenarioContext.
func NewScenarioContext(
	ctx context.Context,
	vuid int64,
	iteration int64,
	cfg config.ScenarioConfig,
	scenarioName string,
	globalState map[string]any,
	logger log.Logger,
	metrics metric.Collector,
) ScenarioContext {
	return newScenarioContext(ctx, vuid, iteration, cfg, scenarioName, globalState, logger, metrics)
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

	var delayGen delay.DelayGenerator
	delayCfg := cfg.InteractionDelay
	if delayCfg == nil {
		delayCfg = cfg.ThinkTime
	}
	if delayCfg != nil {
		delayGen, _ = delay.NewDelayGenerator(delayCfg)
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
		delayGen:     delayGen,
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

func (c *scenarioContext) Sleep(d ...time.Duration) error {
	var duration time.Duration
	if len(d) > 0 {
		duration = d[0]
	} else if c.delayGen != nil {
		duration = c.delayGen.Next()
	}

	if duration <= 0 {
		select {
		case <-c.Done():
			return c.Err()
		default:
			return nil
		}
	}

	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-c.Done():
		return c.Err()
	case <-timer.C:
		return nil
	}
}

func (c *scenarioContext) Check(name string, fn CheckFunc) bool {
	var reason string
	if fn != nil {
		reason = fn()
	}

	if reason == "" {
		if c.metrics != nil {
			c.metrics.Counter("gtest.checks.passed", metric.Tags{"name": name}).Inc()
		}
		return true
	}

	if c.metrics != nil {
		c.metrics.Counter("gtest.checks.failed", metric.Tags{"name": name}).Inc()
	}
	if c.logger != nil {
		c.logger.Warn().Str("check", name).Str("reason", reason).Msg("check failed")
	}
	return false
}

// Compile-time interface satisfaction check.
var _ ScenarioContext = (*scenarioContext)(nil)


