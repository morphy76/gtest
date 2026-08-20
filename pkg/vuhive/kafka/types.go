package kafka

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrKafkaDisabled is returned when Kafka operations are invoked on a binary built without the 'kafka' tag.
	ErrKafkaDisabled = errors.New("vuhive: kafka module disabled (recompile with '-tags kafka')")

	// ErrTopicRequired is returned when an operation requires a topic but none was configured.
	ErrTopicRequired = errors.New("vuhive: kafka topic is required")

	// ErrBrokersRequired is returned when a client is created without specifying any broker addresses.
	ErrBrokersRequired = errors.New("vuhive: at least one kafka broker address is required")

	// ErrNilMessage is returned when attempting to publish a nil message.
	ErrNilMessage = errors.New("vuhive: message cannot be nil")
)

const defaultMetricPrefix = "vuhive.kafka."

// Built-in metric name suffixes for Kafka telemetry.
const (
	MetricSuffixPubDuration = "pub_duration"
	MetricSuffixPubTotal    = "pub_total"
	MetricSuffixPubBytes    = "pub_bytes"
	MetricSuffixPubFailed   = "pub_failed"

	MetricSuffixSubDuration = "sub_duration"
	MetricSuffixSubTotal    = "sub_total"
	MetricSuffixSubBytes    = "sub_bytes"
	MetricSuffixSubFailed   = "sub_failed"
)

// Message represents a Kafka record for publishing or consumption.
type Message struct {
	// Topic is the Kafka topic name.
	Topic string

	// Key is the optional partition routing key.
	Key []byte

	// Value is the message payload.
	Value []byte

	// Headers contains optional key-value metadata headers.
	Headers map[string][]byte

	// Partition is the partition index (populated on consume).
	Partition int32

	// Offset is the record offset in the partition (populated on consume).
	Offset int64

	// Timestamp is the message timestamp.
	Timestamp time.Time
}

// Publisher provides high-throughput message publishing capabilities.
// Publisher implementations are safe for concurrent use across multiple VU goroutines.
type Publisher interface {
	// Publish writes a single message to Kafka and records latency and throughput metrics.
	Publish(ctx context.Context, msg *Message) error

	// PublishBatch writes a batch of messages to Kafka atomically or sequentially depending on configuration.
	PublishBatch(ctx context.Context, msgs []*Message) error

	// Close flushes buffered messages and closes underlying broker connections.
	Close() error
}

// Consumer provides message consumption and offset management capabilities.
// Consumer implementations are safe for concurrent use across multiple VU goroutines.
type Consumer interface {
	// Consume reads a single message from Kafka and records fetch duration and byte metrics.
	Consume(ctx context.Context) (*Message, error)

	// ConsumeBatch reads up to batchSize messages within the specified timeout duration.
	ConsumeBatch(ctx context.Context, batchSize int, timeout time.Duration) ([]*Message, error)

	// Commit commits the offset of the given message.
	Commit(ctx context.Context, msg *Message) error

	// Close closes the underlying consumer and releases group coordinator connections.
	Close() error
}

// Client combines Publisher and Consumer capabilities into a unified handle.
type Client interface {
	Publisher
	Consumer
}
