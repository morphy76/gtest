# gtest Examples Reference Suite

Welcome to the `gtest` reference suite! This directory contains standalone, self-contained, compilable examples demonstrating every core capability, pacing engine, protocol adapter, and lifecycle hook of the `gtest` load testing framework.

Each example is paired with an in-process mock server (or simulation harness), a companion `gtest.yaml` configuration, and a dedicated `README.md` explaining the architectural concept, configuration schema, and expected report outputs.

---

## Quickstart: Running Any Example

All examples can be executed immediately without external network dependencies:

```bash
# Run any example by passing its config file:
go run -tags=gtest_example ./examples/http_checkout --config ./examples/http_checkout/gtest.yaml

# Or navigate to the directory and run directly:
cd examples/http_checkout
go run -tags=gtest_example .
```

Verify compilation across all examples in the repository:

```bash
make test-examples
```

---

## Structured Learning Progression Path

We recommend exploring the examples in the following progression:

### Level 1: Fundamentals
Start here to understand test suite registration, lifecycle hooks, and basic assertion workflows.

| Example Directory | Pacing Engine | What You Will Learn |
|---|---|---|
| [`http_checkout/`](http_checkout/) | `constant_vus` | **Getting Started**: Initializing HTTP clients in `SetupContext`, passing state via `GlobalState`, reading scenario parameters with `ctx.Param()`, and recording HDR duration histograms and success rates. |
| [`checks/`](checks/) | `constant_vus` | **Inline Assertions**: Validating HTTP status codes, headers, and JSON body fields with `ctx.Check()` without prematurely stopping VU iterations. Auto-instrumented check counters and report tables. |

---

### Level 2: Intermediate Workflows
Explore user behavior modeling, dataset ingestion, and dynamic multi-stage workload scaling.

| Example Directory | Pacing Engine | What You Will Learn |
|---|---|---|
| [`think_time/`](think_time/) | `constant_vus` | **Pacing & Delays**: Modeling human pauses in multi-step user journeys using declarative `interaction_delay` (`ctx.Sleep()`) and custom mathematical distributions (`gtest.ExpoDelay`). |
| [`data_parameterization/`](data_parameterization/) | `constant_vus` | **Dynamic Data Feeds**: Loading CSV, JSON arrays, and JSON Lines datasets with `pkg/gtest/data` using `Sequential`, `Random`, and `SharedQueue` distribution strategies. |
| [`ramping_vus/`](ramping_vus/) | `ramping_vus` | **Multi-Stage Spike Testing**: Configuring stage-based ramp-ups, holds, and sudden traffic spikes to measure system resilience and recovery. |

---

### Level 3: Advanced Protocols & Lifecycle Hooks
Master production-grade quality gates, custom summary export, real-time streaming, and high-throughput RPC engines.

| Example Directory | Pacing Engine | What You Will Learn |
|---|---|---|
| [`sla_thresholds/`](sla_thresholds/) | `constant_vus` | **Quality Gates & Early Stop**: Defining multi-metric SLA thresholds (p95, p99, rate, check counts) and configuring real-time early test termination with `abort_on_fail: true`. |
| [`handle_summary/`](handle_summary/) | `constant_vus` | **Post-Run Hooks**: Programmatically consuming `gtest.SummaryData` in `HandleSummary` to dispatch webhook alerts (Slack/Teams/Datadog) and export custom JSON artifacts. |
| [`conversation_flow/`](conversation_flow/) | `constant_vus` | **Stateful Streaming & DSLs**: Real-time Server-Sent Events (SSE) streaming load tests, multi-turn state machines, and event-driven reactive DSL clients. |
| [`grpc_user_service/`](grpc_user_service/) | `arrival_rate` | **High-Throughput RPC & Open Systems**: Open-system token-bucket pacing (`target_tps`) with bounded worker pools (`max_vus`) for gRPC/RPC protocols. |

---

## Feature Matrix

| Capability / API | `http_checkout` | `checks` | `think_time` | `data_parameterization` | `ramping_vus` | `sla_thresholds` | `handle_summary` | `conversation_flow` | `grpc_user_service` |
|---|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| **Pacing: `constant_vus`** | :white_check_mark: | :white_check_mark: | :white_check_mark: | :white_check_mark: | | :white_check_mark: | :white_check_mark: | :white_check_mark: | |
| **Pacing: `ramping_vus`** | | | | | :white_check_mark: | | | | |
| **Pacing: `arrival_rate`** | | | | | | | | | :white_check_mark: |
| **Hook: `SetupContext`** | :white_check_mark: | :white_check_mark: | :white_check_mark: | :white_check_mark: | :white_check_mark: | :white_check_mark: | :white_check_mark: | :white_check_mark: | :white_check_mark: |
| **Hook: `PreTest` / `AfterTest`** | :white_check_mark: | | :white_check_mark: | :white_check_mark: | :white_check_mark: | :white_check_mark: | | :white_check_mark: | :white_check_mark: |
| **Hook: `TeardownContext`** | :white_check_mark: | | | | | | | :white_check_mark: | :white_check_mark: |
| **Hook: `HandleSummary`** | | | | | | | :white_check_mark: | | |
| **Inline Checks (`ctx.Check`)** | | :white_check_mark: | | :white_check_mark: | | :white_check_mark: | | | |
| **Thinking Time (`ctx.Sleep`)** | | | :white_check_mark: | | | | | :white_check_mark: | |
| **Data Feeds (`pkg/gtest/data`)** | | | | :white_check_mark: | | | | :white_check_mark: | |
| **HDR Duration Histograms** | :white_check_mark: | | :white_check_mark: | | :white_check_mark: | :white_check_mark: | :white_check_mark: | :white_check_mark: | :white_check_mark: |
| **Counters, Rates, Gauges** | :white_check_mark: | | :white_check_mark: | :white_check_mark: | :white_check_mark: | :white_check_mark: | :white_check_mark: | :white_check_mark: | :white_check_mark: |
| **SLA Quality Gates** | :white_check_mark: | :white_check_mark: | :white_check_mark: | :white_check_mark: | :white_check_mark: | :white_check_mark: | :white_check_mark: | :white_check_mark: | :white_check_mark: |
| **Early Stop (`abort_on_fail`)** | | | | | | :white_check_mark: | | | |
| **Streaming / Protocol DSL** | | | | | | | | :white_check_mark: | :white_check_mark: |

---

## Directory Index

- [`examples/http_checkout/`](http_checkout/) — [Documentation](http_checkout/README.md) | [Source Code](http_checkout/main.go) | [Config](http_checkout/gtest.yaml)
- [`examples/checks/`](checks/) — [Documentation](checks/README.md) | [Source Code](checks/main.go) | [Config](checks/gtest.yaml)
- [`examples/think_time/`](think_time/) — [Documentation](think_time/README.md) | [Source Code](think_time/main.go) | [Config](think_time/gtest.yaml)
- [`examples/data_parameterization/`](data_parameterization/) — [Documentation](data_parameterization/README.md) | [Source Code](data_parameterization/main.go) | [Config](data_parameterization/gtest.yaml)
- [`examples/ramping_vus/`](ramping_vus/) — [Documentation](ramping_vus/README.md) | [Source Code](ramping_vus/main.go) | [Config](ramping_vus/gtest.yaml)
- [`examples/sla_thresholds/`](sla_thresholds/) — [Documentation](sla_thresholds/README.md) | [Source Code](sla_thresholds/main.go) | [Config](sla_thresholds/gtest.yaml)
- [`examples/handle_summary/`](handle_summary/) — [Documentation](handle_summary/README.md) | [Source Code](handle_summary/main.go) | [Config](handle_summary/gtest.yaml)
- [`examples/conversation_flow/`](conversation_flow/) — [Documentation](conversation_flow/README.md) | [Source Code](conversation_flow/scenario.go) | [Config](conversation_flow/gtest.yaml)
- [`examples/grpc_user_service/`](grpc_user_service/) — [Documentation](grpc_user_service/README.md) | [Source Code](grpc_user_service/main.go) | [Config](grpc_user_service/gtest.yaml)
