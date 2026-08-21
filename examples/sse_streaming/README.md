# Server-Sent Events (SSE) Streaming Example

Demonstrates real-time **Server-Sent Events (SSE)** streaming load testing using the built-in HTTP client in `pkg/vuhive/http`.

## Concept Overview

Modern architectures increasingly rely on persistent HTTP streams (`text/event-stream`) for unidirectional real-time data push:
- **AI/LLM Token Streaming**: Measuring Time-To-First-Token (TTFT) and token generation throughput (e.g. OpenAI, Anthropic, Ollama, vLLM).
- **Financial Market Feeds**: Continuous tick-by-tick quotes and order book updates.
- **Live Notifications & Telemetry**: Real-time push alerts and IoT status reporting.

Standard HTTP request-response load testing tools attempt to buffer entire responses in memory (`io.ReadAll`) and enforce short execution timeouts, causing continuous streams to hang or fail. 

`vuhive` solves this with first-class streaming handles (`*vuhivehttp.SSEStream`) and dedicated streaming telemetry:

```go
// In RunVU:
client := vuhivehttp.Default(ctx)

// Open continuous stream (Accept: text/event-stream is added automatically)
stream, err := client.StreamSSE(ctx, "/v1/events")
if err != nil {
    return err
}
defer stream.Close()

// Stream events iteratively without unbounded memory buffering
for stream.Next() {
    event := stream.Event()
    ctx.Check("valid_event", event.Event == "token" || event.Event == "message")
}

return stream.Err()
```

## Key Files

| File | Description |
|---|---|
| `main.go` | Scenario using `vuhivehttp.Default(ctx)` and `client.DoStream(ctx, req)` against a simulated LLM token server |
| `vuhive.yaml` | Declarative HTTP client config and SSE streaming SLA quality gates |

## How to Run

```bash
go run -tags vuhive_example ./examples/sse_streaming
```

## Dedicated SSE Metrics

The HTTP module automatically tracks real-time streaming telemetry:

| Metric | Type | Description |
|---|---|---|
| `vuhive.http.sse.connections_total` | Counter | Total number of SSE stream connection attempts |
| `vuhive.http.sse.connect_duration` | Duration | Latency to establish connection and receive HTTP headers |
| `vuhive.http.sse.events_total` | Counter | Total number of received SSE events (tagged by `event_type`) |
| `vuhive.http.sse.event_latency` | Duration | Inter-arrival latency between successive events (TTFE / token latency) |
| `vuhive.http.sse.stream_duration` | Duration | Total active lifespan of streaming sessions |
| `vuhive.http.sse.errors_total` | Counter | Stream disconnections, read errors, or framing errors |

## Configuration Breakdown

```yaml
http:
  base_url: "http://localhost:8080"
  timeout: 30s
  headers:
    Accept: "text/event-stream"
    User-Agent: "vuhive/1.0"
  pool:
    max_idle_conns: 100
    max_idle_conns_per_host: 20
    idle_conn_timeout: 90s

thresholds:
  - metric: vuhive.http.sse.connect_duration
    stat: p95
    operator: "<"
    target: "200ms"
  - metric: vuhive.http.sse.events_total
    stat: count
    operator: ">="
    target: "50"
  - metric: vuhive.http.sse.errors_total
    stat: count
    operator: "<="
    target: "0"
```

## Expected Output

```text
SCENARIO EXECUTION SUMMARY
════════════════════════════════════════════════════════════════
Suite:      SSE Streaming Demo Suite
Scenario:   sse_streaming_demo
Duration:   14.0s (ramp_up: 2s, run: 10s, ramp_down: 2s)
VUs:        5 (constant_vus)

CHECKS
────────────────────────────────────────────────────────────────
  ✓ stream_status_200    100.00% (XXX/XXX)
  ✓ received_tokens      100.00% (XXX/XXX)

CUSTOM METRICS
────────────────────────────────────────────────────────────────
  vuhive.http.sse.connect_duration ... p50=Xms   p95=Xms   p99=Xms
  vuhive.http.sse.connections_total .. XXX
  vuhive.http.sse.event_latency ...... p50=Xms   p95=Xms   p99=Xms
  vuhive.http.sse.events_total ....... XXX
  vuhive.http.sse.stream_duration .... p50=Xms   p95=Xms   p99=Xms

THRESHOLDS
────────────────────────────────────────────────────────────────
  [PASS]  vuhive.http.sse.connect_duration  p95 < 200ms
  [PASS]  vuhive.http.sse.events_total     count >= 50
  [PASS]  vuhive.http.sse.errors_total     count <= 0
```
