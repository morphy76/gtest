# Kafka Messaging & Event Streaming Example

Demonstrates the **vuhive Kafka module** (`pkg/vuhive/kafka`), which provides auto-instrumented Kafka Publisher and Consumer clients conditionally compiled using Go build tags (`//go:build kafka`).

## Concept Overview

Testing event-driven architectures requires generating high-throughput message streams and measuring end-to-end publish latency, consumer processing durations, payload throughput (bytes/sec), and message error rates.

The `vuhive/kafka` module eliminates manual instrumentation boilerplate and supports zero-dependency builds for users who don't need Kafka drivers.

### Conditional Compilation

| Build Command | Active Driver | Behavior |
|---|---|---|
| `go build .` | No-Op (`!kafka`) | Zero external Kafka dependencies in binary. Operations return `ErrKafkaDisabled`. |
| `go build -tags kafka .` | Concrete Driver (`kafka`) | Pure-Go high-throughput Kafka driver (`franz-go`) with automatic telemetry. |

## Key Files

| File | Description |
|---|---|
| `main.go` | Scenario using `kafka.NewClient(ctx)` for publishing and consuming |
| `vuhive.yaml` | Load profile configuration with SLA thresholds on Kafka metrics |

## How to Run

```bash
# Run standalone with the in-process mock Kafka broker:
go run -tags "vuhive_example kafka" ./examples/kafka

# Or specify custom config:
go run -tags "vuhive_example kafka" ./examples/kafka --config ./examples/kafka/vuhive.yaml
```

## Automatic Metrics

The Kafka module records the following telemetry metrics for every publish and consume operation:

| Metric | Type | Tags | Description |
|---|---|---|---|
| `vuhive.kafka.pub_duration` | Duration | `topic`, `status` | Publish round-trip latency histogram |
| `vuhive.kafka.pub_total` | Counter | `topic`, `status` | Total messages published |
| `vuhive.kafka.pub_bytes` | Counter | `topic` | Total payload bytes published |
| `vuhive.kafka.pub_failed` | Rate | `topic`, `status` | Ratio of failed publish operations |
| `vuhive.kafka.sub_duration` | Duration | `topic`, `group`, `status` | Message fetch/wait duration |
| `vuhive.kafka.sub_total` | Counter | `topic`, `group`, `status` | Total messages consumed |
| `vuhive.kafka.sub_bytes` | Counter | `topic` | Total payload bytes consumed |
| `vuhive.kafka.sub_failed` | Rate | `topic`, `group`, `status` | Ratio of failed consume operations |

## Configuration & Thresholds

```yaml
thresholds:
  - metric: vuhive.kafka.pub_duration
    stat: p95
    operator: "<"
    target: "50ms"

  - metric: vuhive.kafka.pub_failed
    stat: rate
    operator: "<="
    target: "0.01"
```
