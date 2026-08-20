//go:build !kafka

package kafka

import (
	"context"
	"time"

	"github.com/morphy76/vuhive/pkg/vuhive"
)

type noopPublisher struct {
	cfg clientConfig
}

func (p *noopPublisher) Publish(ctx context.Context, msg *Message) error {
	return ErrKafkaDisabled
}

func (p *noopPublisher) PublishBatch(ctx context.Context, msgs []*Message) error {
	return ErrKafkaDisabled
}

func (p *noopPublisher) Close() error {
	return nil
}

type noopConsumer struct {
	cfg clientConfig
}

func (c *noopConsumer) Consume(ctx context.Context) (*Message, error) {
	return nil, ErrKafkaDisabled
}

func (c *noopConsumer) ConsumeBatch(ctx context.Context, batchSize int, timeout time.Duration) ([]*Message, error) {
	return nil, ErrKafkaDisabled
}

func (c *noopConsumer) Commit(ctx context.Context, msg *Message) error {
	return ErrKafkaDisabled
}

func (c *noopConsumer) Close() error {
	return nil
}

type noopClient struct {
	noopPublisher
	noopConsumer
}

func (c *noopClient) Close() error {
	return nil
}

// NewPublisher creates a new Kafka Publisher. When built without the 'kafka' tag,
// a no-op implementation is returned and calls to Publish will return ErrKafkaDisabled.
func NewPublisher(ctx vuhive.SetupContext, opts ...Option) (Publisher, error) {
	return NewPublisherWithCollector(ctx.Metrics(), opts...)
}

// NewPublisherFromVU creates a per-VU Kafka Publisher. When built without the 'kafka' tag,
// a no-op implementation is returned and calls to Publish will return ErrKafkaDisabled.
func NewPublisherFromVU(ctx vuhive.VUContext, opts ...Option) (Publisher, error) {
	return NewPublisherWithCollector(ctx.Metrics(), opts...)
}

// NewPublisherWithCollector creates a Kafka Publisher using the given metrics collector.
func NewPublisherWithCollector(metrics vuhive.MetricsCollector, opts ...Option) (Publisher, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	return &noopPublisher{cfg: cfg}, nil
}

// NewConsumer creates a new Kafka Consumer. When built without the 'kafka' tag,
// a no-op implementation is returned and calls to Consume will return ErrKafkaDisabled.
func NewConsumer(ctx vuhive.SetupContext, opts ...Option) (Consumer, error) {
	return NewConsumerWithCollector(ctx.Metrics(), opts...)
}

// NewConsumerFromVU creates a per-VU Kafka Consumer. When built without the 'kafka' tag,
// a no-op implementation is returned and calls to Consume will return ErrKafkaDisabled.
func NewConsumerFromVU(ctx vuhive.VUContext, opts ...Option) (Consumer, error) {
	return NewConsumerWithCollector(ctx.Metrics(), opts...)
}

// NewConsumerWithCollector creates a Kafka Consumer using the given metrics collector.
func NewConsumerWithCollector(metrics vuhive.MetricsCollector, opts ...Option) (Consumer, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	return &noopConsumer{cfg: cfg}, nil
}

// NewClient creates a unified Kafka Client (Publisher + Consumer). When built without the 'kafka' tag,
// a no-op implementation is returned and operations will return ErrKafkaDisabled.
func NewClient(ctx vuhive.SetupContext, opts ...Option) (Client, error) {
	return NewClientWithCollector(ctx.Metrics(), opts...)
}

// NewClientFromVU creates a per-VU unified Kafka Client. When built without the 'kafka' tag,
// a no-op implementation is returned and operations will return ErrKafkaDisabled.
func NewClientFromVU(ctx vuhive.VUContext, opts ...Option) (Client, error) {
	return NewClientWithCollector(ctx.Metrics(), opts...)
}

// NewClientWithCollector creates a unified Kafka Client using the given metrics collector.
func NewClientWithCollector(metrics vuhive.MetricsCollector, opts ...Option) (Client, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	return &noopClient{
		noopPublisher: noopPublisher{cfg: cfg},
		noopConsumer:  noopConsumer{cfg: cfg},
	}, nil
}
