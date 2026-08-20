// Package kafka provides auto-instrumented Kafka Publisher and Consumer clients
// for load and stress testing event-driven architectures with vuhive.
//
// # Conditional Compilation
//
// To prevent binary bloat and external dependency trees in standard builds, the concrete
// Kafka client is conditionally compiled using the "kafka" build tag:
//
//   - Default builds (go build . / go test ./...): Uses a lightweight, zero-dependency no-op
//     driver. Client constructors succeed safely, but invocations of Publish or Consume return
//     ErrKafkaDisabled to inform developers to recompile with the tag.
//   - Kafka builds (go build -tags kafka .): Compiles the full high-throughput driver powered
//     by a pure-Go Kafka library (github.com/segmentio/kafka-go).
//
// # Automatic Telemetry Metrics
//
// Operations automatically emit metrics into the scenario's metric collector:
//   - vuhive.kafka.pub_duration (Duration): Publish round-trip latency histogram
//   - vuhive.kafka.pub_total (Counter): Total messages published
//   - vuhive.kafka.pub_bytes (Counter): Total payload bytes published
//   - vuhive.kafka.pub_failed (Rate): Ratio of failed publish attempts
//   - vuhive.kafka.sub_duration (Duration): Message fetch/wait duration
//   - vuhive.kafka.sub_total (Counter): Total messages consumed
//   - vuhive.kafka.sub_bytes (Counter): Total payload bytes consumed
//   - vuhive.kafka.sub_failed (Rate): Ratio of failed consume attempts
//
// # Basic Usage
//
//	// In Setup: initialize shared publisher and consumer
//	pub, err := kafka.NewPublisher(ctx,
//	    kafka.WithBrokers("localhost:9092"),
//	    kafka.WithTopic("orders"),
//	)
//
//	// In RunVU: publish messages — metrics are recorded automatically
//	err := pub.Publish(ctx, &kafka.Message{
//	    Key:   []byte("user-123"),
//	    Value: []byte(`{"order_id":"ord-1001"}`),
//	})
package kafka
