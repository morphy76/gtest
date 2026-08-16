package engine

import (
	"context"
	"strconv"
	"time"

	"github.com/morphy76/vuhive/internal/config"
	"github.com/morphy76/vuhive/internal/delay"
	"github.com/morphy76/vuhive/internal/log"
	"github.com/morphy76/vuhive/internal/metric"
)

// ExecutionIdentity provides execution identity attributes (VU ID, iteration index, and scenario name).
type ExecutionIdentity interface {
	VUID() int64
	Iteration() int64
	ScenarioName() string
}

// ConfigProvider provides access to scenario configuration parameters.
type ConfigProvider interface {
	// Param retrieves a string value from the scenario's params map. Returns "" if absent.
	Param(key string) string

	// ParamInt retrieves a params value parsed as int. Returns defaultValue if key is absent.
	// If the value is present but cannot be parsed as an integer, a Warn-level log is emitted
	// and defaultValue is returned.
	ParamInt(key string, defaultValue int) int

	// ParamDuration retrieves a params value parsed as time.Duration. Returns defaultValue if key is absent.
	// If the value is present but cannot be parsed as a duration, a Warn-level log is emitted
	// and defaultValue is returned.
	ParamDuration(key string, defaultValue time.Duration) time.Duration
}

// StateProvider provides read-only access to global scenario state returned by Setup.
// Note: Global state is shallow-copied by the framework. Complex or nested mutable values
// (such as slices, maps, or pointers) must be immutable or protected with explicit synchronization.
type StateProvider interface {
	GlobalState(key string) any
}

// ObservabilityProvider provides access to structured logging and metric collection.
type ObservabilityProvider interface {
	Log() log.Logger
	Metrics() metric.Collector
}

// WorkflowController provides workflow execution controls such as delays and inline assertions.
type WorkflowController interface {
	Sleep(d ...time.Duration) error
	Check(name string, fn CheckFunc) bool
}

// SetupContext provides configuration access and structured observability during scenario setup.
type SetupContext interface {
	context.Context
	ConfigProvider
	ObservabilityProvider
}

// VUContext is the scoped execution context passed to active Virtual User hooks (PreTest, RunVU, AfterTest).
type VUContext interface {
	context.Context
	ExecutionIdentity
	ConfigProvider
	StateProvider
	ObservabilityProvider
	WorkflowController
}

// TeardownContext provides configuration, read-only global state, and observability for scenario teardown.
type TeardownContext interface {
	context.Context
	ConfigProvider
	StateProvider
	ObservabilityProvider
}

// SummaryContext provides context cancellation, scenario params, and structured logging for post-run reporting.
type SummaryContext interface {
	context.Context
	ConfigProvider
	ObservabilityProvider
}

// ScenarioContext is the scoped execution context passed to every VU hook.
// It embeds context.Context and composes focused capability interfaces.
type ScenarioContext interface {
	context.Context
	ExecutionIdentity
	ConfigProvider
	StateProvider
	ObservabilityProvider
	WorkflowController
}

type scenarioBinder interface {
	WithScenario(string) any
	WithVU(int) any
	WithIteration(int64) any
}

type checkCounterPair struct {
	passed metric.Counter
	failed metric.Counter
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
	checkCache   map[string]checkCounterPair
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
		checkCache:   make(map[string]checkCounterPair, 4),
	}
}

// newVUScenarioContext constructs a reusable ScenarioContext scoped to a VU goroutine.
func newVUScenarioContext(
	ctx context.Context,
	vuid int64,
	cfg config.ScenarioConfig,
	scenarioName string,
	globalState map[string]any,
	logger log.Logger,
	metrics metric.Collector,
) *scenarioContext {
	boundLogger := logger
	if b, ok := logger.(scenarioBinder); ok {
		if s, ok := b.WithScenario(scenarioName).(scenarioBinder); ok {
			if v, ok := s.WithVU(int(vuid)).(log.Logger); ok {
				boundLogger = v
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
		iteration:    0,
		scenarioName: scenarioName,
		params:       cfg.Params,
		globalState:  globalState,
		logger:       boundLogger,
		metrics:      metrics,
		delayGen:     delayGen,
		checkCache:   make(map[string]checkCounterPair, 4),
	}
}

func (c *scenarioContext) prepareIteration(ctx context.Context, iteration int64) {
	c.Context = ctx
	c.iteration = iteration
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
		if c.logger != nil {
			c.logger.Warn().
				Str("key", key).
				Str("value", v).
				Err(err).
				Int("default", defaultValue).
				Msg("failed to parse param as integer; using default value")
		}
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
		if c.logger != nil {
			c.logger.Warn().
				Str("key", key).
				Str("value", v).
				Err(err).
				Dur("default", defaultValue).
				Msg("failed to parse param as duration; using default value")
		}
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

	if c.metrics == nil {
		if reason != "" && c.logger != nil {
			c.logger.Warn().Str("check", name).Str("reason", reason).Msg("check failed")
		}
		return reason == ""
	}

	pair, ok := c.checkCache[name]
	if !ok {
		pair = checkCounterPair{
			passed: c.metrics.Counter(metric.MetricChecksPassed, metric.Tags{"name": name}),
			failed: c.metrics.Counter(metric.MetricChecksFailed, metric.Tags{"name": name}),
		}
		if c.checkCache == nil {
			c.checkCache = make(map[string]checkCounterPair, 4)
		}
		c.checkCache[name] = pair
	}

	if reason == "" {
		pair.passed.Inc()
		return true
	}

	pair.failed.Inc()
	if c.logger != nil {
		c.logger.Warn().Str("check", name).Str("reason", reason).Msg("check failed")
	}
	return false
}

// Compile-time interface satisfaction checks.
var (
	_ ScenarioContext       = (*scenarioContext)(nil)
	_ SetupContext          = (*scenarioContext)(nil)
	_ VUContext             = (*scenarioContext)(nil)
	_ TeardownContext       = (*scenarioContext)(nil)
	_ SummaryContext        = (*scenarioContext)(nil)
	_ ExecutionIdentity     = (*scenarioContext)(nil)
	_ ConfigProvider        = (*scenarioContext)(nil)
	_ StateProvider         = (*scenarioContext)(nil)
	_ ObservabilityProvider = (*scenarioContext)(nil)
	_ WorkflowController    = (*scenarioContext)(nil)
)


