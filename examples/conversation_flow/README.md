# Conversational AI Load Test (SSE Streaming) Example

An advanced real-world reference example demonstrating how to load test multi-turn Conversational AI platforms communicating via Server-Sent Events (SSE) and HTTP REST APIs.

---

## Concept Overview

Testing conversational AI and LLM streaming backends requires managing stateful, multi-turn dialogues:
- **Server-Sent Events (SSE)**: Opening long-lived unidirectional event streams (`/api/v1/conversation/:id`) and listening for asynchronous lifecycle and token/message events.
- **REST Message Ingestion**: Dispatching user prompts (`/api/v1/message/:dialog_id`) while receiving bot responses over the existing SSE stream.
- **Multi-Turn State Machine**: Coordinating customer prompt -> bot response -> thinking time -> next prompt across configurable turns.
- **Thinking Time Between Turns**: Simulating human reading and typing delays between dialogue turns with `ctx.Sleep()`.
- **Per-Event Timeouts**: Preventing VU hangs on missing SSE events using bounded channel selects.
- **Rich Telemetry**: Tracking stream open duration (`sse_open_time`), bot answer latency (`answer_received_time`), dialogue lifecycle counters, and round-trip verification.

---

## Key Files & Architecture

```text
examples/conversation_flow/
├── main.go            ← Test suite initialization & execution
├── scenario.go        ← Setup, PreTest, RunVU, Teardown, and in-process mock SSE server
├── vuhive.yaml         ← Load profile, model params, thinking time, and SLA gates
└── dsl/               ← Domain-Specific Language package
    ├── client.go      ← SSE connection manager & HTTP client
    ├── flow.go        ← Reactive event dispatch loop & turn orchestrator
    ├── messages.go    ← CSV dataset loader and prompt model
    └── metrics.go     ← Pre-configured telemetry helpers
```

---

## How to Run

From the repository root:

```bash
go run -tags=vuhive_example ./examples/conversation_flow --config ./examples/conversation_flow/vuhive.yaml
```

Or from within the example directory:

```bash
cd examples/conversation_flow
go run -tags=vuhive_example .
```

---

## Configuration Breakdown (`vuhive.yaml`)

```yaml
version: "1.0"
default_scenario: conversation_test_flow

scenarios:
  conversation_test_flow:
    type: constant_vus
    vus: 3
    ramp_up: 100ms
    run_period: 500ms
    ramp_down: 100ms
    vu_timeout: 5s

    # In-iteration thinking time between bot reply and next customer message
    interaction_delay:
      type: range
      min: 50ms
      max: 150ms

    params:
      base_url: "mock"        # "mock" for in-process server or external URL
      token: "pat_token"
      tenant: "tenant_1"
      dialog_model: "gpt-4o"
      turns: "2"              # Number of conversation turns per iteration
      sse_event_timeout: "5s" # Max wait for each expected SSE bot reply

    thresholds:
      - metric: sse_open_time
        stat: p95
        operator: "<"
        target: "200ms"
      - metric: answer_received_time
        stat: p95
        operator: "<"
        target: "500ms"
      - metric: conversation_success_rate
        stat: rate
        operator: ">="
        target: "0.9"
```

---

## Expected Output

```text
================================================================================
                        VUHIVE LOAD TEST SUMMARY
================================================================================
Scenario:     conversation_test_flow          Version: dev
Mode:         constant_vus (3 VUs)            Commit:  none
Duration:     00:00:00  (ramp-up: 100ms | run: 500ms | ramp-down: 100ms)
Iterations:   16 total  |  0 failed (0.00%)  |  0 timeout

BUILT-IN METRICS
────────────────────────────────────────────────────────────────
vuhive.vu.iterations_total      Counter    16
vuhive.vu.iterations_failed     Counter    0
vuhive.vu.iterations_timeout    Counter    0
vuhive.vu.panics                Counter    0
vuhive.vu.pretest_errors        Counter    0
vuhive.checks.passed            Counter    0
vuhive.checks.failed            Counter    0

CUSTOM METRICS
────────────────────────────────────────────────────────────────
Metric                         Type       Count    Min     Mean    p95     p99     Max    
answer_received_time           Duration   32       10.592ms 11.595ms 12.063ms 12.311ms 12.311ms
bot_messages_received          Counter    32      
conversation_duration          Duration   16       77.888ms 117.128ms 152.575ms 168.447ms 168.447ms
conversation_success_rate      Rate       (rate: 1)
customer_messages_received     Counter    32      
dialog_created_event_time      Duration   16       16µs    50µs    94µs    109µs   109µs  
message_delivery_time          Duration   32       307µs   756µs   1.134ms 1.521ms 1.521ms
message_success_rate           Rate       (rate: 1)
messages_sent                  Counter    32      
number_of_closed_dialogs       Counter    16      
number_of_closed_sse_connections Counter    16      
number_of_created_dialogs      Counter    16      
number_of_requested_sse_connections Counter    16      
number_of_successful_sse_connections Counter    16      
sse_channel_availability       Rate       (rate: 1)
sse_open_time                  Duration   16       256µs   426µs   511µs   1.351ms 1.351ms

SLA THRESHOLD EVALUATION
────────────────────────────────────────────────────────────────
  [PASS]  sse_open_time           p95 < 200ms     → actual: 511µs
  [PASS]  answer_received_time    p95 < 500ms     → actual: 12.063ms
  [PASS]  conversation_success_rate rate >= 0.9     → actual: 1
────────────────────────────────────────────────────────────────
OVERALL: PASSED                                         (exit 0)
================================================================================
```
