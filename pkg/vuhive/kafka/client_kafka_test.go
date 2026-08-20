//go:build kafka

package kafka_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kfake"

	"github.com/morphy76/vuhive/internal/metric"
	"github.com/morphy76/vuhive/pkg/vuhive"
	"github.com/morphy76/vuhive/pkg/vuhive/kafka"
)

type metricsTestAdapter struct {
	collector metric.Collector
}

func (m *metricsTestAdapter) Counter(name string, tags vuhive.Tags) vuhive.Counter {
	return m.collector.Counter(name, metric.Tags(tags))
}

func (m *metricsTestAdapter) Gauge(name string, tags vuhive.Tags) vuhive.Gauge {
	return m.collector.Gauge(name, metric.Tags(tags))
}

func (m *metricsTestAdapter) Duration(name string, tags vuhive.Tags) vuhive.Duration {
	return m.collector.Duration(name, metric.Tags(tags))
}

func (m *metricsTestAdapter) Rate(name string, tags vuhive.Tags) vuhive.Rate {
	return m.collector.Rate(name, metric.Tags(tags))
}

func newTestStore() (*metric.Store, vuhive.MetricsCollector) {
	store := metric.NewStore()
	return store, &metricsTestAdapter{collector: store}
}

func startFakeKafka(t *testing.T, topics ...string) (*kfake.Cluster, []string) {
	t.Helper()
	var opts []kfake.Opt
	opts = append(opts, kfake.AllowAutoTopicCreation())
	if len(topics) > 0 {
		opts = append(opts, kfake.SeedTopics(1, topics...))
	}
	cluster, err := kfake.NewCluster(opts...)
	require.NoError(t, err)
	t.Cleanup(cluster.Close)
	return cluster, cluster.ListenAddrs()
}

func TestKafkaPublisher_Publish_SingleMessage(t *testing.T) {
	_, brokers := startFakeKafka(t, "test-topic")
	store, collector := newTestStore()

	pub, err := kafka.NewPublisherWithCollector(collector,
		kafka.WithBrokers(brokers...),
		kafka.WithTopic("test-topic"),
	)
	require.NoError(t, err)
	defer pub.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	msg := &kafka.Message{
		Key:     []byte("key-1"),
		Value:   []byte("value-1"),
		Headers: map[string][]byte{"trace-id": []byte("12345")},
	}

	err = pub.Publish(ctx, msg)
	require.NoError(t, err)

	// Verify metrics
	assert.Equal(t, int64(1), store.AggregatedCounterValue(vuhive.MetricKafkaPubTotal))
	assert.Equal(t, int64(1), store.MergedHistogramSnapshot(vuhive.MetricKafkaPubDuration).Count)
	assert.Equal(t, 0.0, store.AggregatedRateValue(vuhive.MetricKafkaPubFailed))
	assert.True(t, store.AggregatedCounterValue(vuhive.MetricKafkaPubBytes) > 0)
}

func TestKafkaPublisher_PublishBatch(t *testing.T) {
	_, brokers := startFakeKafka(t, "batch-topic")
	store, collector := newTestStore()

	pub, err := kafka.NewPublisherWithCollector(collector,
		kafka.WithBrokers(brokers...),
		kafka.WithTopic("batch-topic"),
	)
	require.NoError(t, err)
	defer pub.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	msgs := []*kafka.Message{
		{Key: []byte("k1"), Value: []byte("v1")},
		{Key: []byte("k2"), Value: []byte("v2")},
		{Key: []byte("k3"), Value: []byte("v3")},
	}

	err = pub.PublishBatch(ctx, msgs)
	require.NoError(t, err)

	assert.Equal(t, int64(1), store.AggregatedCounterValue(vuhive.MetricKafkaPubTotal))
	assert.Equal(t, int64(1), store.MergedHistogramSnapshot(vuhive.MetricKafkaPubDuration).Count)
	assert.Equal(t, 0.0, store.AggregatedRateValue(vuhive.MetricKafkaPubFailed))
}

func TestKafkaPublisher_ValidationErrors(t *testing.T) {
	_, brokers := startFakeKafka(t)
	_, collector := newTestStore()

	pub, err := kafka.NewPublisherWithCollector(collector,
		kafka.WithBrokers(brokers...),
	)
	require.NoError(t, err)
	defer pub.Close()

	ctx := context.Background()

	// Nil message
	err = pub.Publish(ctx, nil)
	assert.ErrorIs(t, err, kafka.ErrNilMessage)

	// Missing topic
	err = pub.Publish(ctx, &kafka.Message{Value: []byte("test")})
	assert.ErrorIs(t, err, kafka.ErrTopicRequired)
}

func TestKafkaClient_EndToEnd_ProduceAndConsume(t *testing.T) {
	_, brokers := startFakeKafka(t, "e2e-topic")
	store, collector := newTestStore()

	client, err := kafka.NewClientWithCollector(collector,
		kafka.WithBrokers(brokers...),
		kafka.WithTopic("e2e-topic"),
		kafka.WithGroupID("e2e-group"),
		kafka.WithStartOffset(-2), // start from oldest
	)
	require.NoError(t, err)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Publish a message
	sendMsg := &kafka.Message{
		Key:     []byte("order-1"),
		Value:   []byte(`{"item":"book","qty":2}`),
		Headers: map[string][]byte{"source": []byte("test")},
	}
	err = client.Publish(ctx, sendMsg)
	require.NoError(t, err)

	// 2. Consume the message
	recvMsg, err := client.Consume(ctx)
	require.NoError(t, err)
	require.NotNil(t, recvMsg)

	assert.Equal(t, "e2e-topic", recvMsg.Topic)
	assert.Equal(t, []byte("order-1"), recvMsg.Key)
	assert.Equal(t, []byte(`{"item":"book","qty":2}`), recvMsg.Value)
	assert.Equal(t, []byte("test"), recvMsg.Headers["source"])

	// 3. Commit message offset
	err = client.Commit(ctx, recvMsg)
	require.NoError(t, err)

	// 4. Verify metrics
	assert.Equal(t, int64(1), store.AggregatedCounterValue(vuhive.MetricKafkaPubTotal))
	assert.Equal(t, int64(1), store.AggregatedCounterValue(vuhive.MetricKafkaSubTotal))
	assert.Equal(t, int64(1), store.MergedHistogramSnapshot(vuhive.MetricKafkaSubDuration).Count)
	assert.Equal(t, 0.0, store.AggregatedRateValue(vuhive.MetricKafkaSubFailed))
}

func TestKafkaConsumer_ConsumeBatch(t *testing.T) {
	_, brokers := startFakeKafka(t, "batch-sub-topic")
	store, collector := newTestStore()

	client, err := kafka.NewClientWithCollector(collector,
		kafka.WithBrokers(brokers...),
		kafka.WithTopic("batch-sub-topic"),
		kafka.WithGroupID("batch-group"),
		kafka.WithStartOffset(-2),
	)
	require.NoError(t, err)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Publish 3 messages
	for i := 0; i < 3; i++ {
		err = client.Publish(ctx, &kafka.Message{
			Key:   []byte("key"),
			Value: []byte("val"),
		})
		require.NoError(t, err)
	}

	// Consume batch
	msgs, err := client.ConsumeBatch(ctx, 3, 5*time.Second)
	require.NoError(t, err)
	assert.Len(t, msgs, 3)

	assert.Equal(t, int64(3), store.AggregatedCounterValue(vuhive.MetricKafkaPubTotal))
	assert.Equal(t, int64(1), store.AggregatedCounterValue(vuhive.MetricKafkaSubTotal))
}

func TestKafkaOptions_CustomPrefix(t *testing.T) {
	_, brokers := startFakeKafka(t, "prefix-topic")
	store, collector := newTestStore()

	pub, err := kafka.NewPublisherWithCollector(collector,
		kafka.WithBrokers(brokers...),
		kafka.WithTopic("prefix-topic"),
		kafka.WithCustomMetricPrefix("custom.kafka."),
	)
	require.NoError(t, err)
	defer pub.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = pub.Publish(ctx, &kafka.Message{Value: []byte("hello")})
	require.NoError(t, err)

	assert.Equal(t, int64(1), store.AggregatedCounterValue("custom.kafka.pub_total"))
}
