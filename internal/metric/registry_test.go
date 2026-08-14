package metric_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/morphy76/gtest/internal/metric"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_InterfaceSatisfaction(t *testing.T) {
	_ = metric.NewRegistry()
}

func TestRegistry_RegisterAndMetricType(t *testing.T) {
	reg := metric.NewRegistry()

	err := reg.Register("my_counter", metric.MetricTypeCounter)
	require.NoError(t, err)

	err = reg.Register("my_gauge", metric.MetricTypeGauge)
	require.NoError(t, err)

	err = reg.Register("my_duration", metric.MetricTypeDuration)
	require.NoError(t, err)

	err = reg.Register("my_rate", metric.MetricTypeRate)
	require.NoError(t, err)

	mt, ok := reg.MetricType("my_counter")
	assert.True(t, ok)
	assert.Equal(t, metric.MetricTypeCounter, mt)

	mt, ok = reg.MetricType("my_gauge")
	assert.True(t, ok)
	assert.Equal(t, metric.MetricTypeGauge, mt)

	mt, ok = reg.MetricType("my_duration")
	assert.True(t, ok)
	assert.Equal(t, metric.MetricTypeDuration, mt)

	mt, ok = reg.MetricType("my_rate")
	assert.True(t, ok)
	assert.Equal(t, metric.MetricTypeRate, mt)

	_, ok = reg.MetricType("nonexistent")
	assert.False(t, ok)
}

func TestRegistry_PreRegisteredMetrics(t *testing.T) {
	reg := metric.NewRegistry()

	mt, ok := reg.MetricType(metric.MetricChecksPassed)
	assert.True(t, ok)
	assert.Equal(t, metric.MetricTypeCounter, mt)

	mt, ok = reg.MetricType(metric.MetricChecksFailed)
	assert.True(t, ok)
	assert.Equal(t, metric.MetricTypeCounter, mt)
}

func TestRegistry_TypeCollision_RegisterReturnsError(t *testing.T) {
	reg := metric.NewRegistry()

	err := reg.Register("metric_x", metric.MetricTypeCounter)
	require.NoError(t, err)

	err = reg.Register("metric_x", metric.MetricTypeGauge)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `metric "metric_x" already registered as Counter, cannot register as Gauge`)

	var colErr *metric.ErrTypeCollision
	assert.ErrorAs(t, err, &colErr)
	assert.Equal(t, "metric_x", colErr.Name)
	assert.Equal(t, metric.MetricTypeCounter, colErr.Existing)
	assert.Equal(t, metric.MetricTypeGauge, colErr.New)
}

func TestRegistry_TypeCollision_MustRegisterPanics(t *testing.T) {
	reg := metric.NewRegistry()

	reg.MustRegister("metric_y", metric.MetricTypeCounter)

	assert.PanicsWithValue(t,
		`gtest: metric "metric_y" already registered as Counter, cannot register as Duration`,
		func() {
			reg.MustRegister("metric_y", metric.MetricTypeDuration)
		},
	)
}

func TestRegistry_RegisterSameTypeIsNoOp(t *testing.T) {
	reg := metric.NewRegistry()

	require.NoError(t, reg.Register("counter_a", metric.MetricTypeCounter))
	require.NoError(t, reg.Register("counter_a", metric.MetricTypeCounter))

	assert.NotPanics(t, func() {
		reg.MustRegister("counter_a", metric.MetricTypeCounter)
	})
}

func TestRegistry_NamesByType(t *testing.T) {
	reg := metric.NewRegistry()

	reg.MustRegister("cnt_b", metric.MetricTypeCounter)
	reg.MustRegister("cnt_a", metric.MetricTypeCounter)
	reg.MustRegister("gauge_z", metric.MetricTypeGauge)
	reg.MustRegister("gauge_y", metric.MetricTypeGauge)
	reg.MustRegister("dur_p", metric.MetricTypeDuration)
	reg.MustRegister("rate_r", metric.MetricTypeRate)

	// Built-in metrics: gtest.checks.failed, gtest.checks.passed
	assert.Equal(t, []string{"cnt_a", "cnt_b", metric.MetricChecksFailed, metric.MetricChecksPassed}, reg.CounterNames())
	assert.Equal(t, []string{"gauge_y", "gauge_z"}, reg.GaugeNames())
	assert.Equal(t, []string{"dur_p"}, reg.HistogramNames())
	assert.Equal(t, []string{"rate_r"}, reg.RateNames())

	assert.Equal(t, []string{"gauge_y", "gauge_z"}, reg.NamesByType(metric.MetricTypeGauge))
}

type customMetric struct {
	mType metric.MetricType
}

func (c *customMetric) Type() metric.MetricType {
	return c.mType
}

func TestRegistry_RegisterMetric(t *testing.T) {
	reg := metric.NewRegistry()

	cm := &customMetric{mType: metric.MetricTypeRate}
	err := reg.RegisterMetric("custom_rate", cm)
	require.NoError(t, err)

	mt, ok := reg.MetricType("custom_rate")
	assert.True(t, ok)
	assert.Equal(t, metric.MetricTypeRate, mt)

	// Nil metric returns error
	err = reg.RegisterMetric("nil_metric", nil)
	require.Error(t, err)
}

func TestRegistry_ConcurrentRegistration(t *testing.T) {
	reg := metric.NewRegistry()
	const goroutines = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		idx := i
		go func() {
			defer wg.Done()
			name := fmt.Sprintf("metric_%d", idx)
			reg.MustRegister(name, metric.MetricTypeCounter)
			_, _ = reg.MetricType(name)
			_ = reg.CounterNames()
		}()
	}

	wg.Wait()
	names := reg.CounterNames()
	// goroutines + 2 built-in
	assert.Len(t, names, goroutines+2)
}
