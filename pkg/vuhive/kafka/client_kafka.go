//go:build kafka

package kafka

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
	saslplain "github.com/twmb/franz-go/pkg/sasl/plain"
	saslscram "github.com/twmb/franz-go/pkg/sasl/scram"

	"github.com/morphy76/vuhive/pkg/vuhive"
)

type kafkaPublisher struct {
	client  *kgo.Client
	cfg     clientConfig
	metrics vuhive.MetricsCollector
}

func (p *kafkaPublisher) Publish(ctx context.Context, msg *Message) error {
	if msg == nil {
		return ErrNilMessage
	}

	topic := msg.Topic
	if topic == "" {
		topic = p.cfg.topic
	}
	if topic == "" {
		return ErrTopicRequired
	}

	rec := &kgo.Record{
		Topic: topic,
		Key:   msg.Key,
		Value: msg.Value,
	}
	bytesCount := len(msg.Key) + len(msg.Value)
	for k, v := range msg.Headers {
		rec.Headers = append(rec.Headers, kgo.RecordHeader{Key: k, Value: v})
		bytesCount += len(k) + len(v)
	}

	start := time.Now()
	res := p.client.ProduceSync(ctx, rec)
	duration := time.Since(start)

	err := res.FirstErr()
	recordPubMetrics(p.metrics, p.cfg.metricPrefix, topic, duration, bytesCount, err)

	if err != nil {
		return fmt.Errorf("vuhive/kafka: publish failed: %w", err)
	}
	return nil
}

func (p *kafkaPublisher) PublishBatch(ctx context.Context, msgs []*Message) error {
	if len(msgs) == 0 {
		return nil
	}

	records := make([]*kgo.Record, 0, len(msgs))
	totalBytes := 0
	primaryTopic := p.cfg.topic

	for _, msg := range msgs {
		if msg == nil {
			return ErrNilMessage
		}
		topic := msg.Topic
		if topic == "" {
			topic = p.cfg.topic
		}
		if topic == "" {
			return ErrTopicRequired
		}
		if primaryTopic == "" {
			primaryTopic = topic
		}

		rec := &kgo.Record{
			Topic: topic,
			Key:   msg.Key,
			Value: msg.Value,
		}
		totalBytes += len(msg.Key) + len(msg.Value)
		for k, v := range msg.Headers {
			rec.Headers = append(rec.Headers, kgo.RecordHeader{Key: k, Value: v})
			totalBytes += len(k) + len(v)
		}
		records = append(records, rec)
	}

	start := time.Now()
	res := p.client.ProduceSync(ctx, records...)
	duration := time.Since(start)

	err := res.FirstErr()
	recordPubMetrics(p.metrics, p.cfg.metricPrefix, primaryTopic, duration, totalBytes, err)

	if err != nil {
		return fmt.Errorf("vuhive/kafka: batch publish failed: %w", err)
	}
	return nil
}

func (p *kafkaPublisher) Close() error {
	p.client.Close()
	return nil
}

type kafkaConsumer struct {
	client  *kgo.Client
	cfg     clientConfig
	metrics vuhive.MetricsCollector
}

func (c *kafkaConsumer) Consume(ctx context.Context) (*Message, error) {
	start := time.Now()
	fetches := c.client.PollRecords(ctx, 1)
	duration := time.Since(start)

	if errs := fetches.Errors(); len(errs) > 0 {
		firstErr := errs[0].Err
		recordSubMetrics(c.metrics, c.cfg.metricPrefix, c.cfg.topic, c.cfg.groupID, duration, 0, firstErr)
		return nil, fmt.Errorf("vuhive/kafka: consume failed: %w", firstErr)
	}

	records := fetches.Records()
	if len(records) == 0 {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, nil
	}

	r := records[0]
	bytesCount := len(r.Key) + len(r.Value)
	headers := make(map[string][]byte, len(r.Headers))
	for _, h := range r.Headers {
		headers[h.Key] = h.Value
		bytesCount += len(h.Key) + len(h.Value)
	}

	recordSubMetrics(c.metrics, c.cfg.metricPrefix, r.Topic, c.cfg.groupID, duration, bytesCount, nil)

	return &Message{
		Topic:     r.Topic,
		Key:       r.Key,
		Value:     r.Value,
		Headers:   headers,
		Partition: r.Partition,
		Offset:    r.Offset,
		Timestamp: r.Timestamp,
	}, nil
}

func (c *kafkaConsumer) ConsumeBatch(ctx context.Context, batchSize int, timeout time.Duration) ([]*Message, error) {
	if batchSize <= 0 {
		batchSize = 100
	}

	consumeCtx := ctx
	var cancel context.CancelFunc
	if timeout > 0 {
		consumeCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	start := time.Now()
	result := make([]*Message, 0, batchSize)
	totalBytes := 0
	primaryTopic := c.cfg.topic

	for len(result) < batchSize {
		needed := batchSize - len(result)
		fetches := c.client.PollRecords(consumeCtx, needed)
		if errs := fetches.Errors(); len(errs) > 0 {
			firstErr := errs[0].Err
			if len(result) > 0 {
				break
			}
			duration := time.Since(start)
			recordSubMetrics(c.metrics, c.cfg.metricPrefix, c.cfg.topic, c.cfg.groupID, duration, 0, firstErr)
			return nil, fmt.Errorf("vuhive/kafka: batch consume failed: %w", firstErr)
		}

		records := fetches.Records()
		if len(records) == 0 {
			if consumeCtx.Err() != nil {
				break
			}
			continue
		}

		for _, r := range records {
			if primaryTopic == "" {
				primaryTopic = r.Topic
			}
			bytesCount := len(r.Key) + len(r.Value)
			headers := make(map[string][]byte, len(r.Headers))
			for _, h := range r.Headers {
				headers[h.Key] = h.Value
				bytesCount += len(h.Key) + len(h.Value)
			}
			totalBytes += bytesCount

			result = append(result, &Message{
				Topic:     r.Topic,
				Key:       r.Key,
				Value:     r.Value,
				Headers:   headers,
				Partition: r.Partition,
				Offset:    r.Offset,
				Timestamp: r.Timestamp,
			})
			if len(result) >= batchSize {
				break
			}
		}
	}

	duration := time.Since(start)
	recordSubMetrics(c.metrics, c.cfg.metricPrefix, primaryTopic, c.cfg.groupID, duration, totalBytes, nil)
	return result, nil
}

func (c *kafkaConsumer) Commit(ctx context.Context, msg *Message) error {
	if msg == nil {
		return nil
	}
	r := &kgo.Record{
		Topic:     msg.Topic,
		Partition: msg.Partition,
		Offset:    msg.Offset,
	}
	return c.client.CommitRecords(ctx, r)
}

func (c *kafkaConsumer) Close() error {
	c.client.Close()
	return nil
}

type kafkaClient struct {
	publisher *kafkaPublisher
	consumer  *kafkaConsumer
	client    *kgo.Client
}

func (c *kafkaClient) Publish(ctx context.Context, msg *Message) error {
	return c.publisher.Publish(ctx, msg)
}

func (c *kafkaClient) PublishBatch(ctx context.Context, msgs []*Message) error {
	return c.publisher.PublishBatch(ctx, msgs)
}

func (c *kafkaClient) Consume(ctx context.Context) (*Message, error) {
	return c.consumer.Consume(ctx)
}

func (c *kafkaClient) ConsumeBatch(ctx context.Context, batchSize int, timeout time.Duration) ([]*Message, error) {
	return c.consumer.ConsumeBatch(ctx, batchSize, timeout)
}

func (c *kafkaClient) Commit(ctx context.Context, msg *Message) error {
	return c.consumer.Commit(ctx, msg)
}

func (c *kafkaClient) Close() error {
	c.client.Close()
	return nil
}

func buildKgoOptions(cfg clientConfig) []kgo.Opt {
	var kopts []kgo.Opt

	kopts = append(kopts, kgo.AllowAutoTopicCreation())

	if len(cfg.brokers) > 0 {
		kopts = append(kopts, kgo.SeedBrokers(cfg.brokers...))
	}
	if cfg.clientID != "" {
		kopts = append(kopts, kgo.ClientID(cfg.clientID))
	}
	if cfg.groupID != "" {
		kopts = append(kopts, kgo.ConsumerGroup(cfg.groupID))
	}
	if cfg.topic != "" {
		kopts = append(kopts, kgo.DefaultProduceTopic(cfg.topic))
	}
	if len(cfg.topics) > 0 {
		kopts = append(kopts, kgo.ConsumeTopics(cfg.topics...))
	}

	// Acks
	switch cfg.acks {
	case 0:
		kopts = append(kopts, kgo.RequiredAcks(kgo.NoAck()), kgo.DisableIdempotentWrite())
	case 1:
		kopts = append(kopts, kgo.RequiredAcks(kgo.LeaderAck()), kgo.DisableIdempotentWrite())
	default:
		kopts = append(kopts, kgo.RequiredAcks(kgo.AllISRAcks()))
	}

	// Batching
	if cfg.batchSize > 0 {
		kopts = append(kopts, kgo.MaxBufferedRecords(cfg.batchSize))
	}
	if cfg.batchTimeout > 0 {
		kopts = append(kopts, kgo.ProducerLinger(cfg.batchTimeout))
	}

	// Compression
	switch cfg.compression {
	case "gzip":
		kopts = append(kopts, kgo.ProducerBatchCompression(kgo.GzipCompression()))
	case "snappy":
		kopts = append(kopts, kgo.ProducerBatchCompression(kgo.SnappyCompression()))
	case "lz4":
		kopts = append(kopts, kgo.ProducerBatchCompression(kgo.Lz4Compression()))
	case "zstd":
		kopts = append(kopts, kgo.ProducerBatchCompression(kgo.ZstdCompression()))
	}

	// SASL
	if cfg.saslMechanism != "" && cfg.saslUser != "" {
		switch cfg.saslMechanism {
		case SASLPlain:
			kopts = append(kopts, kgo.SASL(saslplain.Auth{
				User: cfg.saslUser,
				Pass: cfg.saslPassword,
			}.AsMechanism()))
		case SASLScramSHA256:
			kopts = append(kopts, kgo.SASL(saslscram.Auth{
				User: cfg.saslUser,
				Pass: cfg.saslPassword,
			}.AsSha256Mechanism()))
		case SASLScramSHA512:
			kopts = append(kopts, kgo.SASL(saslscram.Auth{
				User: cfg.saslUser,
				Pass: cfg.saslPassword,
			}.AsSha512Mechanism()))
		}
	}

	// TLS
	if cfg.tlsConfig != nil {
		kopts = append(kopts, kgo.DialTLSConfig(cfg.tlsConfig))
	} else if cfg.tlsInsecureSkipVerify {
		kopts = append(kopts, kgo.DialTLSConfig(&tls.Config{
			InsecureSkipVerify: true, //nolint:gosec // user explicitly requested
		}))
	}

	// Offset Reset
	if cfg.startOffset == -2 {
		kopts = append(kopts, kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()))
	} else if cfg.startOffset == -1 {
		kopts = append(kopts, kgo.ConsumeResetOffset(kgo.NewOffset().AtEnd()))
	}

	return kopts
}

// NewPublisher creates a new Kafka Publisher backed by franz-go.
func NewPublisher(ctx vuhive.SetupContext, opts ...Option) (Publisher, error) {
	return NewPublisherWithCollector(ctx.Metrics(), opts...)
}

// NewPublisherFromVU creates a per-VU Kafka Publisher.
func NewPublisherFromVU(ctx vuhive.VUContext, opts ...Option) (Publisher, error) {
	return NewPublisherWithCollector(ctx.Metrics(), opts...)
}

// NewPublisherWithCollector creates a Kafka Publisher using the given metrics collector.
func NewPublisherWithCollector(metrics vuhive.MetricsCollector, opts ...Option) (Publisher, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	kopts := buildKgoOptions(cfg)
	cl, err := kgo.NewClient(kopts...)
	if err != nil {
		return nil, fmt.Errorf("vuhive/kafka: failed to initialize kafka client: %w", err)
	}

	return &kafkaPublisher{
		client:  cl,
		cfg:     cfg,
		metrics: metrics,
	}, nil
}

// NewConsumer creates a new Kafka Consumer backed by franz-go.
func NewConsumer(ctx vuhive.SetupContext, opts ...Option) (Consumer, error) {
	return NewConsumerWithCollector(ctx.Metrics(), opts...)
}

// NewConsumerFromVU creates a per-VU Kafka Consumer.
func NewConsumerFromVU(ctx vuhive.VUContext, opts ...Option) (Consumer, error) {
	return NewConsumerWithCollector(ctx.Metrics(), opts...)
}

// NewConsumerWithCollector creates a Kafka Consumer using the given metrics collector.
func NewConsumerWithCollector(metrics vuhive.MetricsCollector, opts ...Option) (Consumer, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	kopts := buildKgoOptions(cfg)
	cl, err := kgo.NewClient(kopts...)
	if err != nil {
		return nil, fmt.Errorf("vuhive/kafka: failed to initialize kafka consumer: %w", err)
	}

	return &kafkaConsumer{
		client:  cl,
		cfg:     cfg,
		metrics: metrics,
	}, nil
}

// NewClient creates a unified Kafka Client (Publisher + Consumer).
func NewClient(ctx vuhive.SetupContext, opts ...Option) (Client, error) {
	return NewClientWithCollector(ctx.Metrics(), opts...)
}

// NewClientFromVU creates a per-VU unified Kafka Client.
func NewClientFromVU(ctx vuhive.VUContext, opts ...Option) (Client, error) {
	return NewClientWithCollector(ctx.Metrics(), opts...)
}

// NewClientWithCollector creates a unified Kafka Client using the given metrics collector.
func NewClientWithCollector(metrics vuhive.MetricsCollector, opts ...Option) (Client, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	kopts := buildKgoOptions(cfg)
	cl, err := kgo.NewClient(kopts...)
	if err != nil {
		return nil, fmt.Errorf("vuhive/kafka: failed to initialize kafka client: %w", err)
	}

	pub := &kafkaPublisher{client: cl, cfg: cfg, metrics: metrics}
	sub := &kafkaConsumer{client: cl, cfg: cfg, metrics: metrics}

	return &kafkaClient{
		publisher: pub,
		consumer:  sub,
		client:    cl,
	}, nil
}
