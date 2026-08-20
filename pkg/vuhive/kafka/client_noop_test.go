//go:build !kafka

package kafka_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/morphy76/vuhive/internal/metric"
	"github.com/morphy76/vuhive/pkg/vuhive"
	"github.com/morphy76/vuhive/pkg/vuhive/kafka"
)

type metricsAdapter struct {
	collector metric.Collector
}

func (m *metricsAdapter) Counter(name string, tags vuhive.Tags) vuhive.Counter {
	return m.collector.Counter(name, metric.Tags(tags))
}

func (m *metricsAdapter) Gauge(name string, tags vuhive.Tags) vuhive.Gauge {
	return m.collector.Gauge(name, metric.Tags(tags))
}

func (m *metricsAdapter) Duration(name string, tags vuhive.Tags) vuhive.Duration {
	return m.collector.Duration(name, metric.Tags(tags))
}

func (m *metricsAdapter) Rate(name string, tags vuhive.Tags) vuhive.Rate {
	return m.collector.Rate(name, metric.Tags(tags))
}

func newTestCollector() vuhive.MetricsCollector {
	return &metricsAdapter{collector: metric.NewStore()}
}

func TestNoopPublisher_ReturnsErrKafkaDisabled(t *testing.T) {
	metrics := newTestCollector()
	pub, err := kafka.NewPublisherWithCollector(metrics, kafka.WithBrokers("localhost:9092"), kafka.WithTopic("orders"))
	require.NoError(t, err)
	require.NotNil(t, pub)

	ctx := context.Background()
	msg := &kafka.Message{Topic: "orders", Key: []byte("k1"), Value: []byte("v1")}

	err = pub.Publish(ctx, msg)
	assert.ErrorIs(t, err, kafka.ErrKafkaDisabled)

	err = pub.PublishBatch(ctx, []*kafka.Message{msg})
	assert.ErrorIs(t, err, kafka.ErrKafkaDisabled)

	err = pub.Close()
	assert.NoError(t, err)
}

func TestNoopConsumer_ReturnsErrKafkaDisabled(t *testing.T) {
	metrics := newTestCollector()
	sub, err := kafka.NewConsumerWithCollector(metrics, kafka.WithBrokers("localhost:9092"), kafka.WithGroupID("group1"), kafka.WithTopics("orders"))
	require.NoError(t, err)
	require.NotNil(t, sub)

	ctx := context.Background()
	msg, err := sub.Consume(ctx)
	assert.Nil(t, msg)
	assert.ErrorIs(t, err, kafka.ErrKafkaDisabled)

	msgs, err := sub.ConsumeBatch(ctx, 10, 100*time.Millisecond)
	assert.Nil(t, msgs)
	assert.ErrorIs(t, err, kafka.ErrKafkaDisabled)

	err = sub.Commit(ctx, &kafka.Message{})
	assert.ErrorIs(t, err, kafka.ErrKafkaDisabled)

	err = sub.Close()
	assert.NoError(t, err)
}

func TestNoopClient_ReturnsErrKafkaDisabled(t *testing.T) {
	metrics := newTestCollector()
	client, err := kafka.NewClientWithCollector(metrics, kafka.WithBrokers("localhost:9092"), kafka.WithTopic("orders"))
	require.NoError(t, err)
	require.NotNil(t, client)

	ctx := context.Background()
	err = client.Publish(ctx, &kafka.Message{Topic: "orders", Value: []byte("val")})
	assert.ErrorIs(t, err, kafka.ErrKafkaDisabled)

	msg, err := client.Consume(ctx)
	assert.Nil(t, msg)
	assert.ErrorIs(t, err, kafka.ErrKafkaDisabled)

	err = client.Close()
	assert.NoError(t, err)
}
