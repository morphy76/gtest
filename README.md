# gtest — High-Performance Go Load Testing Framework

`gtest` is a developer-centric, high-performance load testing library and execution framework built for Go 1.26+. It separates load profile configuration from scenario execution code, allowing developers to define test scenarios in Go with rich lifecycle hooks while managing concurrency, pacing profiles, and SLA assertions declaratively via YAML.

> **New to gtest?** See the [Developer Guide](docs/GUIDE.md) for a step-by-step adoption walkthrough.

---

## Key Features

- **Hexagonal Architecture with DDD Boundaries**: Clean separation of core domain models (`pkg/gtest`), configuration engines, pacing engines, metrics storage, and reporting CLI adapters.
- **Dual Pacing Engines**:
  - **`constant_vus`**: Closed-system model maintaining a fixed number of Concurrent Virtual Users with linear ramp-up and ramp-down spacing.
  - **`arrival_rate`**: Open-system token bucket rate-limiting engine (`golang.org/x/time/rate`) targeting precise Transactions Per Second (TPS) with a bounded worker pool (`max_vus`).
- **Lock-Free In-Memory Metrics Engine**: Atomic counters, CAS gauges, atomic rate tracking, and per-VU HDR Histograms (`github.com/HdrHistogram/hdrhistogram-go`) providing zero-contention, high-resolution percentile calculations (`p50`, `p90`, `p95`, `p99`, `mean`, `min`, `max`).
- **Structured Logging**: Zerolog (`github.com/rs/zerolog`) integration with automatic VU ID, Scenario, and Iteration correlation context.
- **SLA Threshold Evaluator**: Declarative quality gates evaluated without short-circuiting after execution. Returns exit code `0` on success or exit code `1` on SLA breach.
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
    Setup          func(ctx ScenarioContext) (map[string]any, error)
    PreTest        func(ctx ScenarioContext) error
    RunVU          func(ctx ScenarioContext) error
    AfterTest      func(ctx ScenarioContext) error
    Teardown       func(ctx ScenarioContext, state map[string]any) error
    HandleSummary  func(ctx context.Context, summary SummaryData) error
}
```

### Lifecycle Hook Sequence

```text
       ┌────────────────────────┐
       │   Setup(ctx)           │  (Runs once per scenario before VUs spawn)
       └───────────┬────────────┘
                   │  returns globalState map[string]any
                   ▼
┌──────────────────────────────────────────────┐
│ For each VU Iteration:                       │
│                                              │
│   ┌────────────────────────┐                 │
│   │   PreTest(sCtx)        │                 │
│   └───────────┬────────────┘                 │
│               │ (if err != nil, skips RunVU) │
│               ▼                              │
│   ┌────────────────────────┐                 │
│   │   RunVU(sCtx)          │                 │
│   └───────────┬────────────┘                 │
│               │                              │
│               ▼                              │
│   ┌────────────────────────┐                 │
│   │   AfterTest(sCtx)      │ (defer guarantee│
│   └────────────────────────┘  runs always)   │
└──────────────────┬───────────────────────────┘
                   │
                   ▼
       ┌────────────────────────┐
       │  Teardown(ctx, state)  │  (Runs once after all VUs exit)
       └───────────┬────────────┘
                   │
                   ▼
       ┌────────────────────────┐
       │  HandleSummary(summary)│  (Runs post-report with full execution summary)
       └────────────────────────┘
```


---

## ScenarioContext API

Inside `PreTest`, `RunVU`, and `AfterTest`, developers interact with `ScenarioContext`:

| Method | Description |
|--------|-------------|
| `ctx.VUID()` | Returns the 1-based Virtual User ID (`int64`). |
| `ctx.Iteration()` | Returns the 0-based iteration index (`int64`). |
| `ctx.ScenarioName()` | Returns the scenario string identifier. |
| `ctx.Param(key)` | Returns scenario param string from YAML config. |
| `ctx.ParamInt(key, default)` | Parses scenario param as integer. |
| `ctx.ParamDuration(key, default)` | Parses scenario param as `time.Duration` (e.g. `200ms`). |
| `ctx.GlobalState(key)` | Accesses values returned by the `Setup` hook. |
| `ctx.Log()` | Structured `Logger` instance bound with VU ID and iteration context. |
| `ctx.Metrics()` | `MetricsCollector` for recording custom counters, gauges, durations, and rates. |
| `ctx.Sleep(d ...time.Duration)` | Pauses for explicit duration or configured `interaction_delay` strategy (respects `ctx.Done()`). |
| `ctx.Check(name, fn)` | Evaluates inline pass/fail assertion (`CheckFunc`) without stopping VU iteration execution. |


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
        operator: "=="
        target: "0"
```

### Supported Pacing Modes

1. **`constant_vus`**:
   - `vus`: Number of concurrent VUs (`int > 0`).
   - `ramp_up`: Staggered linear VU spawn duration (`time.Duration`).
   - `run_period`: Steady-state load duration (`time.Duration`).
   - `ramp_down`: Graceful exit duration (`time.Duration`).

2. **`arrival_rate`**:
   - `target_tps`: Desired transactions/iterations per second (`int > 0`).
   - `max_vus`: Maximum size of the worker pool (`int > 0`). If the pool saturates, unhandled tokens increment `gtest.pacing.dropped_iterations`.
   - `ramp_up`: Linear rate ramp-up duration (`time.Duration`).
   - `run_period`: Steady-state arrival duration (`time.Duration`).

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
RunVU: func(ctx gtest.ScenarioContext) error {
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




## Writing a Load Test (Code Example)

```go
package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/morphy76/gtest/pkg/gtest"
)

func main() {
	suite := gtest.NewSuite("E-Commerce Load Test Suite")

	suite.RegisterScenario("http_checkout_flow", gtest.Scenario{
		Setup: func(ctx gtest.ScenarioContext) (map[string]any, error) {
			client := &http.Client{Timeout: 5 * time.Second}
			return map[string]any{"client": client}, nil
		},
		PreTest: func(ctx gtest.ScenarioContext) error {
			ctx.Log().Debug().Msg("preparing iteration")
			return nil
		},
		RunVU: func(ctx gtest.ScenarioContext) error {
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
		AfterTest: func(ctx gtest.ScenarioContext) error {
			ctx.Log().Debug().Msg("iteration finished")
			return nil
		},
		Teardown: func(ctx gtest.ScenarioContext, state map[string]any) error {
			return nil
		},
		HandleSummary: func(ctx context.Context, summary gtest.SummaryData) error {
			fmt.Printf("Summary Hook: %s completed in %v, passed=%v\n", summary.Scenario, summary.Duration, summary.Passed)
			return nil
		},
	})

	if err := suite.Execute(); err != nil {
		fmt.Printf("Fatal error: %v\n", err)
	}
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
