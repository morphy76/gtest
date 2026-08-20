package kafka

import (
	"crypto/tls"
	"time"
)

// Option configures a Kafka Publisher, Consumer, or Client.
type Option func(*clientConfig)

// SASLMechanism defines supported SASL authentication mechanisms.
type SASLMechanism string

const (
	// SASLPlain represents PLAIN username/password authentication.
	SASLPlain SASLMechanism = "plain"

	// SASLScramSHA256 represents SCRAM-SHA-256 authentication.
	SASLScramSHA256 SASLMechanism = "scram-sha-256"

	// SASLScramSHA512 represents SCRAM-SHA-512 authentication.
	SASLScramSHA512 SASLMechanism = "scram-sha-512"
)

// clientConfig holds the resolved configuration for a Kafka client handle.
type clientConfig struct {
	brokers               []string
	clientID              string
	groupID               string
	topic                 string
	topics                []string
	compression           string
	saslMechanism         SASLMechanism
	saslUser              string
	saslPassword          string
	tlsConfig             *tls.Config
	tlsInsecureSkipVerify bool
	timeout               time.Duration
	metricPrefix          string
	acks                  int
	batchSize             int
	batchTimeout          time.Duration
	minBytes              int
	maxBytes              int
	startOffset           int64
}

func defaultConfig() clientConfig {
	return clientConfig{
		brokers:      []string{"127.0.0.1:9092"},
		metricPrefix: defaultMetricPrefix,
		timeout:      10 * time.Second,
		acks:         -1, // all ISR acks by default
		batchSize:    100,
		batchTimeout: 10 * time.Millisecond,
		minBytes:     1,
		maxBytes:     10e6, // 10MB
		startOffset:  -1,   // newest by default
	}
}

// WithBrokers sets the list of Kafka broker seed addresses (e.g. "localhost:9092").
func WithBrokers(brokers ...string) Option {
	return func(c *clientConfig) {
		if len(brokers) > 0 {
			c.brokers = brokers
		}
	}
}

// WithClientID sets the client identifier string sent with Kafka protocol requests.
func WithClientID(id string) Option {
	return func(c *clientConfig) {
		c.clientID = id
	}
}

// WithGroupID sets the consumer group ID for consumer group partition assignment and offset management.
func WithGroupID(groupID string) Option {
	return func(c *clientConfig) {
		c.groupID = groupID
	}
}

// WithTopic sets the default topic for publishing or single-topic consumption.
func WithTopic(topic string) Option {
	return func(c *clientConfig) {
		c.topic = topic
		if len(c.topics) == 0 && topic != "" {
			c.topics = []string{topic}
		}
	}
}

// WithTopics sets the list of topics for multi-topic consumer subscriptions.
func WithTopics(topics ...string) Option {
	return func(c *clientConfig) {
		c.topics = topics
		if len(topics) > 0 && c.topic == "" {
			c.topic = topics[0]
		}
	}
}

// WithCompression configures the message compression codec ("gzip", "snappy", "lz4", "zstd", "none").
func WithCompression(codec string) Option {
	return func(c *clientConfig) {
		c.compression = codec
	}
}

// WithSASL configures SASL authentication with the given mechanism ("plain", "scram-sha-256", "scram-sha-512").
func WithSASL(mechanism SASLMechanism, user, pass string) Option {
	return func(c *clientConfig) {
		c.saslMechanism = mechanism
		c.saslUser = user
		c.saslPassword = pass
	}
}

// WithSASLPlain configures plain SASL username/password authentication.
func WithSASLPlain(user, pass string) Option {
	return WithSASL(SASLPlain, user, pass)
}

// WithSASLScram256 configures SCRAM-SHA-256 SASL authentication.
func WithSASLScram256(user, pass string) Option {
	return WithSASL(SASLScramSHA256, user, pass)
}

// WithSASLScram512 configures SCRAM-SHA-512 SASL authentication.
func WithSASLScram512(user, pass string) Option {
	return WithSASL(SASLScramSHA512, user, pass)
}

// WithTLS sets custom TLS configuration for encrypted broker transport.
func WithTLS(cfg *tls.Config) Option {
	return func(c *clientConfig) {
		c.tlsConfig = cfg
	}
}

// WithTLSInsecureSkipVerify enables TLS transport while skipping server certificate verification.
func WithTLSInsecureSkipVerify() Option {
	return func(c *clientConfig) {
		c.tlsInsecureSkipVerify = true
	}
}

// WithTimeout sets the network I/O timeout for read and write operations.
func WithTimeout(timeout time.Duration) Option {
	return func(c *clientConfig) {
		c.timeout = timeout
	}
}

// WithCustomMetricPrefix overrides the default metric name prefix ("vuhive.kafka.").
func WithCustomMetricPrefix(prefix string) Option {
	return func(c *clientConfig) {
		c.metricPrefix = prefix
	}
}

// WithAcks sets the required acknowledgment level for publisher writes (0=None, 1=Leader, -1=All).
func WithAcks(acks int) Option {
	return func(c *clientConfig) {
		c.acks = acks
	}
}

// WithMaxBatchSize sets the maximum number of messages buffered before flushing a batch to Kafka.
func WithMaxBatchSize(n int) Option {
	return func(c *clientConfig) {
		c.batchSize = n
	}
}

// WithBatchTimeout sets the maximum time to wait for a batch to fill before writing to Kafka.
func WithBatchTimeout(d time.Duration) Option {
	return func(c *clientConfig) {
		c.batchTimeout = d
	}
}

// WithMinBytes sets the minimum batch bytes required for the consumer to return data.
func WithMinBytes(n int) Option {
	return func(c *clientConfig) {
		c.minBytes = n
	}
}

// WithMaxBytes sets the maximum batch bytes retrieved per consumer fetch request.
func WithMaxBytes(n int) Option {
	return func(c *clientConfig) {
		c.maxBytes = n
	}
}

// WithStartOffset sets the initial partition offset when no committed offset is found (-2=oldest, -1=newest).
func WithStartOffset(offset int64) Option {
	return func(c *clientConfig) {
		c.startOffset = offset
	}
}
