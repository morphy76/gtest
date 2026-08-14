# gtest — High-Performance Go Load Testing Framework

`gtest` is a developer-centric, high-performance load testing library and execution framework built for Go 1.26+. It separates load profile configuration from scenario execution code, allowing developers to define test scenarios in Go with rich lifecycle hooks while managing concurrency, pacing profiles, and SLA assertions declaratively via YAML.

> **New to gtest?** See the [Developer Guide](docs/GUIDE.md) for a step-by-step adoption walkthrough.

---

## Key Features

- **Hexagonal Architecture with DDD Boundaries**: Clean separation of core domain models (`pkg/gtest`), configuration engines, pacing engines, metrics storage, and reporting CLI adapters.
- **Triple Pacing Engines**:
  - **`constant_vus`**: Closed-system model maintaining a fixed number of Concurrent Virtual Users with linear ramp-up and ramp-down spacing.
  - **`arrival_rate`**: Open-system token bucket rate-limiting engine (`golang.org/x/time/rate`) targeting precise Transactions Per Second (TPS) with a bounded worker pool (`max_vus`).
  - **`ramping_vus`**: Dynamic multi-stage pacing engine allowing stage-based VU target ramps, holds, and spikes over time.
- **Lock-Free In-Memory Metrics Engine**: Atomic counters, CAS gauges, atomic rate tracking, and per-VU HDR Histograms (`github.com/HdrHistogram/hdrhistogram-go`) providing zero-contention, high-resolution percentile calculations (`p50`, `p90`, `p95`, `p99`, `mean`, `min`, `max`).
- **Structured Logging**: Zerolog (`github.com/rs/zerolog`) integration with automatic VU ID, Scenario, and Iteration correlation context.
- **Data Parameterization Module (`pkg/gtest/data`)**: CSV, JSON, and JSON Lines dataset loaders (`LoadCSV`, `LoadJSON`, `LoadJSONL`) supporting thread-safe distribution strategies (`Sequential`, `Random`, `UniquePerVU`, `SharedQueue`).
- **SLA Threshold Evaluator & Graceful Abort**: Declarative quality gates evaluated post-execution, with optional real-time early termination (`abort_on_fail: true`, `delay_abort_eval: 5s`) to stop runaway failures instantly. Returns exit code `0` on success or `1` on SLA breach/abort.
- **Deterministic Reporting**: Terminal summary and JSON reports (§10 schema) with alphabetically sorted metrics.

---

## Installation

```bash
go get github.com/morphy76/gtest
```

---

## Core Facilities & Lifecycle Hooks

Test suites are created using `gtest.NewSuite("Suite Name")`. Each scenario registers up to 6 lifecycle hooks:

```go
type Scenario struct {
    Setup          func(ctx SetupContext) (map[string]any, error)
    PreTest        func(ctx VUContext) error
    RunVU          func(ctx VUContext) error
    AfterTest      func(ctx VUContext) error
    Teardown       func(ctx TeardownContext, state map[string]any) error
    HandleSummary  func(ctx SummaryContext, summary SummaryData) error
}
```

### Lifecycle Hook Sequence

```text
       ┌────────────────────────────┐
       │   Setup(ctx SetupContext)  │  (Runs once per scenario before VUs spawn)
       └─────────────┬──────────────┘
                     │  returns globalState map[string]any
                     ▼
┌──────────────────────────────────────────────┐
│ For each VU Iteration:                       │
│                                              │
│   ┌────────────────────────┐                 │
│   │   PreTest(ctx VUContext│                 │
│   └───────────┬────────────┘                 │
│               │ (if err != nil, skips RunVU) │
│               ▼                              │
│   ┌────────────────────────┐                 │
│   │   RunVU(ctx VUContext) │                 │
│   └───────────┬────────────┘                 │
│               │                              │
│               ▼                              │
│   ┌────────────────────────┐                 │
│   │  AfterTest(ctx VUContxt│ (defer guarantee│
│   └────────────────────────┘  runs always)   │
└──────────────────┬───────────────────────────┘
                   │
                   ▼
       ┌──────────────────────────────────────┐
       │ Teardown(ctx TeardownContext, state) │  (Runs once after all VUs exit)
       └───────────┬──────────────────────────┘
                   │
                   ▼
       ┌──────────────────────────────────────┐
       │ HandleSummary(ctx SummaryCtx, summ.) │  (Runs post-report with full execution summary)
       └──────────────────────────────────────┘
```


---

## Context Hierarchy & Capabilities

Adhering to the **Interface Segregation Principle (ISP)**, gtest provides role-specific context interfaces (`SetupContext`, `VUContext`, `TeardownContext`, `SummaryContext`) composing granular capability interfaces. `ScenarioContext` is preserved as an alias to `VUContext` for backward compatibility.

| Method | Capability Interface | Description |
|--------|----------------------|-------------|
| `ctx.VUID()` | `ExecutionIdentity` | Returns the 1-based Virtual User ID (`int64`). |
| `ctx.Iteration()` | `ExecutionIdentity` | Returns the 0-based iteration index (`int64`). |
| `ctx.ScenarioName()` | `ExecutionIdentity` | Returns the scenario string identifier. |
| `ctx.Param(key)` | `ConfigProvider` | Returns scenario param string from YAML config. |
| `ctx.ParamInt(key, default)` | `ConfigProvider` | Parses scenario param as integer (logs warning and returns default on parse failure). |
| `ctx.ParamDuration(key, default)` | `ConfigProvider` | Parses scenario param as `time.Duration` (e.g. `200ms`, logs warning and returns default on parse failure). |
| `ctx.GlobalState(key)` | `StateProvider` | Accesses values returned by the `Setup` hook (shallow-copied, read-only). |
| `ctx.Log()` | `ObservabilityProvider` | Structured `Logger` instance bound with VU ID and iteration context. |
| `ctx.Metrics()` | `ObservabilityProvider` | `MetricsCollector` for recording custom counters, gauges, durations, and rates. |
| `ctx.Sleep(d ...time.Duration)` | `WorkflowController` | Pauses for explicit duration or configured `interaction_delay` strategy (respects `ctx.Done()`). |
| `ctx.Check(name, fn)` | `WorkflowController` | Evaluates inline pass/fail assertion (`CheckFunc`) without stopping VU iteration execution. |


---

## Inline Assertions (Checks)

Inline assertions allow developers to validate real-time pass/fail conditions inside `RunVU` without terminating the iteration:

```go
ctx.Check("status code is 200", func() string {
    if resp.StatusCode != http.StatusOK {
        return fmt.Sprintf("expected 200, got %d", resp.StatusCode)
    }
    return "" // empty string indicates check passed
})
```

- **Pass/Fail Contract**: Return `""` (empty string) for pass (`true`); return non-empty failure reason string for fail (`false`).
- **Auto-Instrumentation**: Automatically increments built-in counters `gtest.checks.passed` and `gtest.checks.failed` tagged with `name`.
- **Reporting & Thresholds**: Per-check pass/fail counts and percentages are displayed in console and JSON reports. SLA thresholds can target check metrics (e.g. `gtest.checks.failed count == 0`).

## Recording Metrics

`gtest` provides four metric types accessed via `ctx.Metrics()`:

```go
// 1. Duration (HDR Histogram)
ctx.Metrics().Duration("http_request_duration", gtest.Tags{"endpoint": "/checkout"}).Observe(120 * time.Millisecond)

// 2. Counter
ctx.Metrics().Counter("http_requests_total", gtest.Tags{"status": "200"}).Inc()
ctx.Metrics().Counter("bytes_transferred", gtest.Tags{}).Add(1024)

// 3. Gauge
ctx.Metrics().Gauge("active_connections", gtest.Tags{}).Set(42)

// 4. Rate (Explicit Numerator/Denominator)
ctx.Metrics().Rate("checkout_success_rate", gtest.Tags{}).Add(1, 1) // 1 success out of 1 trial
```

Built-in framework metrics (`gtest.MetricIterationsTotal`, `gtest.MetricChecksPassed`, etc.) are pre-registered and exported as constants in package `gtest`. See [Developer Guide](docs/GUIDE.md#built-in-metrics-auto-recorded-by-the-framework) for the full inventory.

---

## Configuration (`gtest.yaml`)

Configuration is managed declaratively in `gtest.yaml`.

### Example `gtest.yaml`

```yaml
version: "1.0"
default_scenario: http_checkout_flow

scenarios:
  http_checkout_flow:
    type: constant_vus
    vus: 10
    ramp_up: 5s
    run_period: 30s
    ramp_down: 5s
    vu_timeout: 2s
    params:
      base_url: "https://api.example.com"
      timeout_ms: "500"
    thresholds:
      - metric: http_request_duration
        stat: p95
        operator: "<"
        target: "200ms"
      - metric: checkout_success_rate
        stat: rate
        operator: ">="
        target: "0.99"
      - metric: http_requests_total
        stat: count
        operator: ">="
        target: "500"

  user_registration_api:
    type: arrival_rate
    target_tps: 50
    max_vus: 20
    ramp_up: 10s
    run_period: 1m
    vu_timeout: 1s
    thresholds:
      - metric: gtest.pacing.dropped_iterations
        stat: count
        operator: "<="
        target: "0"
```

### Supported Pacing Modes

1. **`constant_vus`**:
   - `vus`: Number of concurrent VUs (`int > 0`).
   - `ramp_up`: Staggered linear VU spawn duration (`time.Duration`).
   - `run_period`: Steady-state load duration (`time.Duration`).
   - `ramp_down`: Graceful exit duration for in-flight iterations (`time.Duration`, default `0s`).

2. **`arrival_rate`**:
   - `target_tps`: Desired transactions/iterations per second (`int > 0`).
   - `max_vus`: Maximum size of the worker pool (`int > 0`). If the pool saturates, unhandled tokens increment `gtest.pacing.dropped_iterations`.
   - `ramp_up`: Linear rate ramp-up duration (`time.Duration`).
   - `run_period`: Steady-state arrival duration (`time.Duration`).
   - `ramp_down`: Graceful exit duration for in-flight workers (`time.Duration`, default `0s`).

3. **`ramping_vus`**:
   - `stages`: List of stage definitions (`target: int`, `duration: time.Duration`).
   - `ramp_down`: Graceful exit duration for remaining workers (`time.Duration`, default `0s`).
   - `vu_timeout`: Per-iteration timeout (`time.Duration`).
   - *Details and patterns in [Developer Guide](docs/GUIDE.md#ramping_vus-multi-stage-pacing).*

---

## Thinking Time & Interaction Delay Strategies

Simulate realistic human reading and decision pauses between user actions, conversation turns, or multi-step requests. Thinking time is **explicitly invoked by the test developer** using `ctx.Sleep()` and configured declaratively in `gtest.yaml` (or generated programmatically).

### Supported Delay Strategies

| Strategy | YAML Type | Key Parameters | Description |
|---|---|---|---|
| **Fixed** | `fixed` | `duration: 500ms` | Static deterministic pause. |
| **Range** | `range` | `min: 200ms`, `max: 1s` | Uniform random distribution $U(\text{min}, \text{max})$. |
| **Exponential** | `expo` | `mean: 500ms`, `min`, `max` (optional) | Exponential distribution (Poisson arrival modeling) $D = -\text{mean} \cdot \ln(U)$ with optional clamping. |
| **Gaussian** | `gaussian` | `mean: 500ms`, `std_dev: 100ms`, `min`, `max` (optional) | Normal distribution $N(\mu, \sigma)$ with non-negative guarantee and optional clamping. |

### Configuration in `gtest.yaml`

```yaml
scenarios:
  user_checkout:
    type: constant_vus
    vus: 10
    run_period: 1m
    vu_timeout: 5s
    # Thinking time strategy used when calling ctx.Sleep() without arguments:
    interaction_delay:
      type: range
      min: 200ms
      max: 1s
```

### Usage in Code

```go
RunVU: func(ctx gtest.VUContext) error {
    // Step 1: Browse catalog / receive message
    // ...

    // Explicitly execute thinking time using scenario-configured strategy (respects ctx.Done())
    if err := ctx.Sleep(); err != nil {
        return err // aborted due to context cancellation
    }

    // Or pause for an explicit duration
    if err := ctx.Sleep(250 * time.Millisecond); err != nil {
        return err
    }

    // Step 2: Next action (e.g. add to cart / next customer message)
    return nil
}
```

Programmatic generators are also available: `gtest.FixedDelay(d)`, `gtest.RangeDelay(min, max)`, `gtest.ExpoDelay(mean, min, max)`, `gtest.GaussianDelay(mean, stdDev, min, max)`.

---

## Data Parameterization (`pkg/gtest/data`)

Feed external datasets into your load tests using the built-in `pkg/gtest/data` module supporting CSV, JSON, and JSON Lines formats with thread-safe row selection strategies:

### Supported Dataset Formats & Loaders

| Format | Loader Function | Description |
|---|---|---|
| **CSV** | `data.LoadCSV(reader, strategy)` / `data.LoadCSVFile(path, strategy)` | Reads CSV with headers into string key-value maps (`Record`). |
| **JSON** | `data.LoadJSON(reader, strategy)` / `data.LoadJSONFile(path, strategy)` | Parses a JSON array of objects into key-value records. |
| **JSONL** | `data.LoadJSONL(reader, strategy)` / `data.LoadJSONLFile(path, strategy)` | Parses newline-delimited JSON objects line-by-line. |

### Distribution Strategies

| Strategy | Enum Constant | Behavior |
|---|---|---|
| **Sequential** | `data.Sequential` | Deterministic round-robin across rows based on VU ID and iteration index: `(vuid - 1 + iteration) % N` (requires context). |
| **Random** | `data.Random` | Lock-free, thread-safe uniform random selection across all rows. |
| **UniquePerVU** | `data.UniquePerVU` | Partitions rows to avoid VU overlap where possible (requires context). |
| **SharedQueue** | `data.SharedQueue` | Thread-safe atomic single-consumption queue. Returns `data.ErrDatasetExhausted` when depleted. |

### Example Usage

```go
Setup: func(ctx gtest.SetupContext) (map[string]any, error) {
    ds, err := data.LoadCSVFile("data/users.csv", data.Sequential)
    if err != nil {
        return nil, err
    }
    return map[string]any{"users": ds}, nil
},
RunVU: func(ctx gtest.VUContext) error {
    ds := ctx.GlobalState("users").(*data.DataSet)
    user, err := ds.Next(ctx)
    if err != nil {
        return err
    }
    // Access string values by column/key name
    userID := user["user_id"]
    username := user["username"]
    // ...
    return nil
},
```

---





## Writing a Load Test (Code Example)

```go
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/morphy76/gtest/pkg/gtest"
)

func main() {
	suite := gtest.NewSuite("E-Commerce Load Test Suite")

	suite.RegisterScenario("http_checkout_flow", gtest.Scenario{
		Setup: func(ctx gtest.SetupContext) (map[string]any, error) {
			client := &http.Client{Timeout: 5 * time.Second}
			return map[string]any{"client": client}, nil
		},
		PreTest: func(ctx gtest.VUContext) error {
			ctx.Log().Debug().Msg("preparing iteration")
			return nil
		},
		RunVU: func(ctx gtest.VUContext) error {
			baseURL := ctx.Param("base_url")
			client := ctx.GlobalState("client").(*http.Client)

			start := time.Now()
			req, _ := http.NewRequestWithContext(ctx, "GET", baseURL+"/health", nil)
			resp, err := client.Do(req)
			elapsed := time.Since(start)

			ctx.Metrics().Duration("http_request_duration", gtest.Tags{}).Observe(elapsed)

			if err != nil || resp.StatusCode != http.StatusOK {
				ctx.Metrics().Rate("checkout_success_rate", gtest.Tags{}).Add(0, 1)
				return fmt.Errorf("request failed: %v", err)
			}
			_ = resp.Body.Close()

			ctx.Metrics().Rate("checkout_success_rate", gtest.Tags{}).Add(1, 1)
			ctx.Metrics().Counter("http_requests_total", gtest.Tags{}).Inc()
			return nil

		},
		AfterTest: func(ctx gtest.VUContext) error {
			ctx.Log().Debug().Msg("iteration finished")
			return nil
		},
		Teardown: func(ctx gtest.TeardownContext, state map[string]any) error {
			return nil
		},
		HandleSummary: func(ctx gtest.SummaryContext, summary gtest.SummaryData) error {
			fmt.Printf("Summary Hook: %s completed in %v, passed=%v\n", summary.Scenario, summary.Duration, summary.Passed)
			return nil
		},
	})

	res := suite.Execute()
	if res.Error != nil {
		fmt.Fprintf(os.Stderr, "Execution error: %v\n", res.Error)
	}
	os.Exit(res.ExitCode())
}
```

---

## Execution Summary Hook (`HandleSummary`)

`HandleSummary` enables developers to receive the complete execution summary programmatically after the test run and terminal/JSON report generation. This is ideal for posting results to Slack, Datadog, webhooks, or generating custom artifacts.

```go
HandleSummary: func(ctx context.Context, summary gtest.SummaryData) error {
    // Check SLA verdict
    if !summary.Passed {
        sendSlackAlert(fmt.Sprintf("SLA breached for %s!", summary.Scenario))
    }

    // Access metrics & thresholds
    reqCount := summary.Counter("http_requests_total")
    latencyMetric := summary.Metric("http_request_duration")
    fmt.Printf("Processed %d requests, p95 latency: %v\n", reqCount, latencyMetric.P95)

    // Export JSON summary directly
    jsonBytes, _ := summary.JSON()
    os.WriteFile("summary-custom.json", jsonBytes, 0644)
    return nil
}
```

> **Note:** Any error returned by `HandleSummary` is logged to the output stream but does not modify the final exit code.

---


## CLI Options & Execution

Run your test binary with command-line flags:

```bash
go run main.go [flags]
```

### Available CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | `gtest.yaml` | Path to the YAML configuration file. |
| `--scenario` | (default in config) | Specific scenario name to execute. |
| `--log-level` | `info` | Log verbosity: `debug`, `info`, `warn`, `error`. |
| `--log-format` | `pretty` | Log output format: `pretty` or `json`. |
| `--report-format` | `console` | Report format: `console` or `json`. |
| `--report-out` | stdout | File path to write the primary summary report. |
| `--json-report-out` | (none) | File path to write the JSON report document (§10.2 schema). |
| `--version` | `false` | Prints library version info and returns `0`. |

### Exit Code Contract

- **Exit `0`**: Scenario completed and **all SLA thresholds passed**.
- **Exit `1`**: One or more SLA thresholds **failed**, or a pre-execution error occurred.

---

## Output Report Sample

```text
================================================================================
                        GTEST LOAD TEST SUMMARY
================================================================================
Scenario:     http_checkout_flow              Version: 0.1.0
Mode:         constant_vus (10 VUs)           Commit:  dev
Duration:     00:00:40  (ramp-up: 5s | run: 30s | ramp-down: 5s)
Iterations:   1,450 total  |  0 failed (0.00%)  |  0 timeout

BUILT-IN METRICS
────────────────────────────────────────────────────────────────
gtest.vu.iterations_total      Counter    1450
gtest.vu.iterations_failed     Counter    0
gtest.vu.iterations_timeout    Counter    0
gtest.vu.panics                Counter    0
gtest.vu.pretest_errors        Counter    0

CUSTOM METRICS
────────────────────────────────────────────────────────────────
Metric                         Type       Count    Min     Mean    p95     p99     Max
checkout_success_rate          Rate       1450     (rate: 1)
http_request_duration          Duration   1450     12ms    45ms    110ms   230ms   850ms
http_requests_total            Counter    1450

SLA THRESHOLD EVALUATION
────────────────────────────────────────────────────────────────
  [PASS]  http_request_duration   p95 < 200ms     → actual: 110ms
  [PASS]  checkout_success_rate   rate >= 0.99    → actual: 1
  [PASS]  http_requests_total     count >= 500    → actual: 1450
────────────────────────────────────────────────────────────────
OVERALL: PASSED                                          (exit 0)
================================================================================
```

---

## Examples & Reference Implementations

The `examples/` directory contains self-contained, compilable load test suites demonstrating all framework capabilities:

| Example Directory | Scenario Type | Features Demonstrated |
|---|---|---|
| [`examples/http_checkout/`](examples/http_checkout/) | `constant_vus` | REST API load test, custom duration/counter/rate metrics, linear ramp-up/down. |
| [`examples/grpc_user_service/`](examples/grpc_user_service/) | `arrival_rate` | High-throughput RPC simulation, token bucket TPS pacing, bounded worker pool. |
| [`examples/ramping_vus/`](examples/ramping_vus/) | `ramping_vus` | Multi-stage spike test with dynamic VU scaling and recovery observation. |
| [`examples/conversation_flow/`](examples/conversation_flow/) | `constant_vus` | Real-time SSE streaming conversational AI load test, multi-turn state machine, DSL client. |
| [`examples/think_time/`](examples/think_time/) | `constant_vus` | Multi-step user journey, declarative `interaction_delay` (`range`), `ctx.Sleep()`, programmatic `ExpoDelay`. |
| [`examples/checks/`](examples/checks/) | `constant_vus` | Inline assertions (`ctx.Check`) for HTTP status, headers, JSON body validation, check metrics. |
| [`examples/data_parameterization/`](examples/data_parameterization/) | `constant_vus` | CSV (`Sequential`), JSON (`Random`), and JSONL (`SharedQueue`) dataset parameterization. |
| [`examples/sla_thresholds/`](examples/sla_thresholds/) | `constant_vus` | Quality gate SLA thresholds across metrics, percentile operators, and early stop with `abort_on_fail`. |
| [`examples/handle_summary/`](examples/handle_summary/) | `constant_vus` | Post-test execution hook (`HandleSummary`), summary metric inspection, webhook notification dispatch. |

### Running Examples

All examples use in-process mock servers and can be executed immediately:

```bash
# Run any example directly from its folder:
cd examples/think_time && go run -tags=gtest_example .

# Verify compilation across all example binaries:
make test-examples
```

