# gtest Developer Guide

A step-by-step guide for load test developers adopting the `gtest` framework.

---

## Table of Contents

1. [Getting Started](#1-getting-started)
2. [Project Structure](#2-project-structure)
3. [Writing Your First Scenario](#3-writing-your-first-scenario)
4. [Configuration (gtest.yaml)](#4-configuration-gtestyaml)
5. [ScenarioContext API Reference](#5-scenariocontext-api-reference)
6. [Lifecycle Hooks Deep Dive](#6-lifecycle-hooks-deep-dive)
7. [Recording Metrics](#7-recording-metrics)
8. [SLA Thresholds (Quality Gates)](#8-sla-thresholds-quality-gates)
9. [Pacing Modes](#9-pacing-modes)
10. [CLI Flags & Execution](#10-cli-flags--execution)
11. [Patterns & Recipes](#11-patterns--recipes)
12. [Troubleshooting](#12-troubleshooting)
13. [Reference Implementations (examples/)](#13-reference-implementations-examples)

---

## 1. Getting Started

### Prerequisites

- Go 1.26+
- A target system to load test (HTTP, gRPC, SSE, WebSocket, etc.)

### Install

```bash
mkdir my-load-test && cd my-load-test
go mod init myorg/my-load-test
go get github.com/morphy76/gtest
```

### Minimal example

Create `main.go`:

```go
//go:build gtest_example

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/morphy76/gtest/pkg/gtest"
)

func main() {
	suite := gtest.NewSuite("My First Load Test")

	suite.RegisterScenario("hello", gtest.Scenario{
		RunVU: func(ctx gtest.ScenarioContext) error {
			start := time.Now()
			time.Sleep(10 * time.Millisecond) // replace with real work
			ctx.Metrics().Duration("response_time", gtest.Tags{}).Observe(time.Since(start))
			ctx.Metrics().Counter("requests", gtest.Tags{}).Inc()
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

Create `gtest.yaml`:

```yaml
version: "1.0"
default_scenario: hello

scenarios:
  hello:
    type: constant_vus
    vus: 5
    run_period: 10s
    vu_timeout: 2s
```

Run:

```bash
go run main.go
```

---

## 2. Project Structure

gtest is a **library** — you write a Go `main` package that imports it, compile it, and run the binary. There is no separate CLI tool.

### Recommended layout

```text
my-load-test/
├── go.mod
├── main.go              ← registers scenarios, calls suite.Execute()
├── gtest.yaml           ← load profile configuration
├── dsl/                 ← optional: domain-specific helpers (HTTP client, SSE, CSV loader)
│   ├── client.go
│   ├── metrics.go
│   └── flow.go
└── data/
    └── prompts.csv      ← optional: test data files
```

### Key principle

| Concern | Where |
|---------|-------|
| **What** to test (business logic) | `main.go` / `dsl/` (Go code) |
| **How much** load (concurrency, duration) | `gtest.yaml` (YAML config) |
| **Pass/fail criteria** (SLAs) | `gtest.yaml` → `thresholds` block |

---

## 3. Writing Your First Scenario

A scenario is a struct with up to 6 lifecycle hooks. Only `RunVU` is required:

```go
suite.RegisterScenario("checkout_flow", gtest.Scenario{
    // (1) Setup — runs ONCE before any VU spawns
    Setup: func(ctx gtest.ScenarioContext) (map[string]any, error) {
        client := &http.Client{Timeout: 5 * time.Second}
        return map[string]any{"client": client}, nil
    },

    // (2) PreTest — runs ONCE per VU before its iteration loop
    PreTest: func(ctx gtest.ScenarioContext) error {
        ctx.Log().Debug().Int64("vu", ctx.VUID()).Msg("VU starting")
        return nil
    },

    // (3) RunVU — called repeatedly in a loop per VU during run_period
    RunVU: func(ctx gtest.ScenarioContext) error {
        baseURL := ctx.Param("base_url")
        client := ctx.GlobalState("client").(*http.Client)

        start := time.Now()
        req, _ := http.NewRequestWithContext(ctx, "GET", baseURL+"/api/health", nil)
        resp, err := client.Do(req)
        elapsed := time.Since(start)

        ctx.Metrics().Duration("http_latency", gtest.Tags{"endpoint": "/health"}).Observe(elapsed)

        if err != nil {
            ctx.Metrics().Counter("errors", gtest.Tags{}).Inc()
            return err
        }
        defer func() {
            _ = resp.Body.Close()
        }()


        ctx.Metrics().Counter("requests", gtest.Tags{"status": fmt.Sprint(resp.StatusCode)}).Inc()
        return nil
    },

    // (4) AfterTest — runs ONCE per VU after loop ends (defer-guaranteed)
    AfterTest: func(ctx gtest.ScenarioContext) error {
        ctx.Log().Info().Int64("vu", ctx.VUID()).Msg("VU finished")
        return nil
    },

    // (5) Teardown — runs ONCE after ALL VUs exit
    Teardown: func(ctx gtest.ScenarioContext, state map[string]any) error {
        ctx.Log().Info().Msg("cleaning up shared resources")
        return nil
    },

    // (6) HandleSummary — runs ONCE after report generation with structured results
    HandleSummary: func(ctx context.Context, summary gtest.SummaryData) error {
        ctx := context.Background()
        _ = ctx
        fmt.Printf("Scenario %s ended in %v, passed=%v\n", summary.Scenario, summary.Duration, summary.Passed)
        return nil
    },
})
```


---

## 4. Configuration (gtest.yaml)

All load profile settings are declarative YAML. Your Go code reads them via `ctx.Param()`.

### Complete field reference

```yaml
version: "1.0"                  # required, must be "1.0"
default_scenario: my_scenario   # optional, fallback when --scenario not passed

scenarios:
  my_scenario:
    type: constant_vus           # or "arrival_rate"

    # --- constant_vus fields ---
    vus: 10                      # number of concurrent VUs (required for constant_vus)
    ramp_up: 5s                  # linear stagger VU spawn (optional, default 0)
    run_period: 30s              # steady-state duration (required)
    ramp_down: 5s                # grace period for in-flight iterations (optional, default 0)
    vu_timeout: 2s               # per-iteration context deadline (required)

    # --- arrival_rate fields ---
    # target_tps: 100            # target transactions/sec (required for arrival_rate)
    # max_vus: 50                # worker pool cap (required for arrival_rate)

    # --- params (available via ctx.Param) ---
    params:
      base_url: "https://api.example.com"
      timeout_ms: "500"
      messages_file: "data/prompts.csv"

    # --- interaction_delay / think_time (thinking time strategy invoked via ctx.Sleep()) ---
    interaction_delay:
      type: range                # "fixed", "range", "expo", or "gaussian"
      min: 200ms
      max: 1s


    # --- SLA thresholds ---
    thresholds:

      - metric: http_latency
        stat: p95
        operator: "<"
        target: "200ms"
      - metric: requests
        stat: count
        operator: ">="
        target: "1000"
```

### Duration format

All duration fields use Go's `time.ParseDuration` format: `50ms`, `1s`, `5m`, `1h30m`.

---

## 5. ScenarioContext API Reference

`ScenarioContext` embeds `context.Context` and composes 5 focused capability interfaces adhering to the **Interface Segregation Principle (ISP)**:

| Method | Interface | Return | Description |
|--------|-----------|--------|-------------|
| `VUID()` | `ExecutionIdentity` | `int64` | 1-based virtual user ID |
| `Iteration()` | `ExecutionIdentity` | `int64` | 0-based iteration count (0 in PreTest/AfterTest) |
| `ScenarioName()` | `ExecutionIdentity` | `string` | Active scenario name from YAML |
| `Param(key)` | `ConfigProvider` | `string` | Read scenario `params` by key; `""` if absent |
| `ParamInt(key, default)` | `ConfigProvider` | `int` | Parse param as int; returns default on failure |
| `ParamDuration(key, default)` | `ConfigProvider` | `time.Duration` | Parse param as duration; returns default on failure |
| `GlobalState(key)` | `StateProvider` | `any` | Read value from Setup's returned map (read-only) |
| `Log()` | `ObservabilityProvider` | `Logger` | Zerolog logger pre-enriched with scenario/VU/iteration |
| `Metrics()` | `ObservabilityProvider` | `MetricsCollector` | Record custom counters, gauges, durations, rates |
| `Sleep(d ...time.Duration)` | `WorkflowController` | `error` | Pause for explicit duration or scenario `interaction_delay` strategy (respects `ctx.Done()`) |
| `Check(name, fn)` | `WorkflowController` | `bool` | Evaluate inline pass/fail assertion (`CheckFunc`) without stopping VU iteration |

### Interface Segregation & Modular Helpers

Because `ScenarioContext` is decomposed into discrete interfaces, helper functions and custom components can accept only the specific capability they need instead of depending on the entire fat context:

```go
// Accepts only execution identity (e.g. for deterministic data partition or logging correlation)
func ProcessUserBatch(id gtest.ExecutionIdentity, data []string) string {
    idx := (id.VUID() - 1 + id.Iteration()) % int64(len(data))
    return data[idx]
}

// Accepts only configuration parameters
func BuildClientURL(cfg gtest.ConfigProvider) string {
    return fmt.Sprintf("%s/api/v1", cfg.Param("base_url"))
}

// Accepts only workflow controls (sleep / check)
func PerformHealthCheck(wf gtest.WorkflowController, client *http.Client, url string) bool {
    return wf.Check("endpoint reachable", func() string {
        resp, err := client.Get(url)
        if err != nil || resp.StatusCode != http.StatusOK {
            return "health check failed"
        }
        return ""
    })
}
```

This also makes unit testing helper functions trivial, as test authors only need to stub 1-3 methods rather than the full `ScenarioContext`.

### Using as context.Context

Because `ScenarioContext` embeds `context.Context`, you can pass it directly to stdlib and third-party calls:

```go
req, _ := http.NewRequestWithContext(ctx, "POST", url, body)
resp, err := client.Do(req)
```

The context carries the `vu_timeout` deadline. When it expires, `ctx.Err() == context.DeadlineExceeded`.

---

## 6. Lifecycle Hooks Deep Dive

### Execution order

```text
Setup(ctx) ─────────────────────── runs once
  │
  ├── VU 1: PreTest → RunVU loop → AfterTest (defer)
  ├── VU 2: PreTest → RunVU loop → AfterTest (defer)
  └── VU N: PreTest → RunVU loop → AfterTest (defer)
  │
Teardown(ctx, state) ──────────── runs once
  │
HandleSummary(summary) ────────── runs once (post-report generation)
```

### Key guarantees

| Hook | Cardinality | Error behavior |
|------|-------------|----------------|
| **Setup** | Once | Non-nil error **aborts** entire test (returns `*SetupError`) |
| **PreTest** | Once per VU | Non-nil error **skips** RunVU; AfterTest still runs |
| **RunVU** | Loop per VU | Errors and panics are **caught, logged, counted**; loop continues |
| **AfterTest** | Once per VU | Runs via `defer` — **guaranteed** even after RunVU panic |
| **Teardown** | Once | Error is **logged** but does not affect pass/fail verdict |
| **HandleSummary** | Once | Error is **logged** but does not affect exit code or SLA verdict |


### Panic recovery

If `RunVU` panics, the framework:
1. Catches the panic via `recover()`
2. Increments `gtest.vu.panics` and `gtest.vu.iterations_failed`
3. Logs the panic value
4. **Continues the iteration loop** (the VU is not killed)

### GlobalState contract

- Setup returns `map[string]any` → the framework makes a **shallow copy**
- All VUs share the **same copied map** (read-only)
- **Do not mutate** the map from VU code — use `sync.Map` or channels for shared mutable state

---

## 7. Recording Metrics

### Four metric types

```go
// Duration — HDR histogram for latency percentiles
ctx.Metrics().Duration("api_latency", gtest.Tags{"route": "/checkout"}).Observe(elapsed)

// Counter — monotonically increasing integer
ctx.Metrics().Counter("api_requests", gtest.Tags{"status": "200"}).Inc()
ctx.Metrics().Counter("bytes_sent", gtest.Tags{}).Add(4096)

// Gauge — instantaneous snapshot value
ctx.Metrics().Gauge("active_connections", gtest.Tags{}).Set(42)
ctx.Metrics().Gauge("active_connections", gtest.Tags{}).Add(-1)

// Rate — explicit numerator/denominator ratio
ctx.Metrics().Rate("success_rate", gtest.Tags{}).Add(1, 1) // 1 success out of 1 attempt
ctx.Metrics().Rate("success_rate", gtest.Tags{}).Add(0, 1) // 0 successes out of 1 attempt
```

### Tags

Tags let you slice metrics by dimension. Same metric name + different tags = separate time series, merged at report time for SLA evaluation.

```go
ctx.Metrics().Duration("api_latency", gtest.Tags{"endpoint": "/login"}).Observe(d1)
ctx.Metrics().Duration("api_latency", gtest.Tags{"endpoint": "/checkout"}).Observe(d2)
```

The SLA threshold `api_latency.p95 < 200ms` evaluates the **merged** histogram across all tag variants.

### Type collision protection

A metric name is locked to its first-registered type. Attempting to use the same name as a different type panics:

```go
ctx.Metrics().Counter("foo", gtest.Tags{}).Inc()
ctx.Metrics().Duration("foo", gtest.Tags{}).Observe(d) // PANIC: "foo" already registered as Counter
```

### Built-in metrics (auto-recorded by the framework)

All built-in metrics are exported as typed constants in `pkg/gtest` (e.g. `gtest.MetricIterationsTotal`) and `internal/metric`:

| Constant (`pkg/gtest`) | String Identifier | Type | Description |
|------------------------|-------------------|------|-------------|
| `gtest.MetricIterationsTotal` | `gtest.vu.iterations_total` | Counter | Total iterations executed |
| `gtest.MetricIterationsFailed` | `gtest.vu.iterations_failed` | Counter | Iterations that errored or panicked |
| `gtest.MetricIterationsTimeout` | `gtest.vu.iterations_timeout` | Counter | Iterations exceeding `vu_timeout` |
| `gtest.MetricVUPanics` | `gtest.vu.panics` | Counter | RunVU panics recovered |
| `gtest.MetricVUPretestErrors` | `gtest.vu.pretest_errors` | Counter | PreTest hook failures |
| `gtest.MetricVUActive` | `gtest.vu.active` | Gauge | Currently active VU goroutines |
| `gtest.MetricIterationDuration` | `gtest.vu.iteration_duration` | Duration | Completed VU iteration duration |
| `gtest.MetricPacingDroppedIterations` | `gtest.pacing.dropped_iterations` | Counter | Arrival-rate tokens dropped due to pool saturation |
| `gtest.MetricChecksPassed` | `gtest.checks.passed` | Counter | Total inline checks that passed |
| `gtest.MetricChecksFailed` | `gtest.checks.failed` | Counter | Total inline checks that failed |

> **In-Flight Iterations on Shutdown:** `gtest.vu.iterations_timeout` tracks only genuine per-iteration timeouts where `RunVU` exceeded `vu_timeout` during active execution. In-flight iterations that are interrupted when the overall scenario completes (`run_period` / `ramp_down` expiration or early abort) are cleanly cancelled and discarded without being counted as timeouts or failures.

---

## 8. SLA Thresholds (Quality Gates)

Thresholds are declarative pass/fail assertions evaluated after the test run.

### Available stats

| Stat | Applicable to | Description |
|------|---------------|-------------|
| `p50`, `p90`, `p95`, `p99` | Duration | Percentile latency (target is `time.Duration` string) |
| `mean`, `max` | Duration | Mean/max latency |
| `count` | Counter | Total count (target is `float64`) |
| `rate` | Rate | `sum(numerator)/sum(denominator)` (target is `float64`) |
| `value` | Gauge | Last observed value (target is `float64`) |

### Operators

`<`, `<=`, `>`, `>=`

### Example

```yaml
thresholds:
  - metric: api_latency
    stat: p95
    operator: "<"
    target: "200ms"

  - metric: success_rate
    stat: rate
    operator: ">="
    target: "0.99"

  - metric: api_requests
    stat: count
    operator: ">="
    target: "1000"
```

### 8.1 Early Stop / Graceful Abort (`abort_on_fail`)

Configure `abort_on_fail: true` on any threshold entry to immediately terminate test execution if that threshold is breached during execution, preventing wasted test execution or runaway system degradation.

Optionally specify `delay_abort_eval` to establish a warm-up grace period before early termination monitoring begins:

```yaml
thresholds:
  - metric: gtest.vu.iterations_failed
    stat: count
    operator: "=="
    target: "0"
    abort_on_fail: true
    delay_abort_eval: 5s   # Ignore failures during first 5 seconds of ramp-up

  - metric: http_request_duration
    stat: p95
    operator: "<"
    target: "500ms"
    abort_on_fail: true
```

When an early stop is triggered:
1. All active VU contexts are cancelled immediately (`ctx.Done()`).
2. The terminal summary report displays `OVERALL: ABORTED (exit 1)` along with the triggering threshold reason.
3. The process exits with code `1`.

### Exit codes

- **Exit 0**: All thresholds pass → `OVERALL: PASSED`
- **Exit 1**: Any threshold fails → `OVERALL: FAILED`

This makes gtest suitable for CI/CD pipelines where exit code drives the build status.

---

## 9. Pacing Modes

### constant_vus (Closed System)

Maintains a **fixed pool** of VU goroutines. Each VU runs RunVU in a loop until `run_period` expires.

```text
Time ────────────────────────────────────────►
     │← ramp_up →│← run_period ────────→│← ramp_down →│

VU 1 ·············▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓░░░░░░░
VU 2 ··············▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓░░░░░░░
VU 3 ···············▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓░░░░░░░

· = waiting during ramp_up stagger
▓ = actively running iterations
░ = ramp_down grace period (in-flight iterations complete)
```

**Best for**: simulating a fixed number of concurrent users.

> **Graceful Completion:** When `run_period` ends, VUs stop starting new iterations. In-flight iterations are allowed up to `ramp_down` duration to finish. Any iterations still in-flight after `ramp_down` expires (or immediately if `ramp_down: 0s`) are interrupted and discarded without incrementing timeout or failure metrics.

### arrival_rate (Open System)

Dispatches iterations at a target TPS using a token bucket. A bounded worker pool (`max_vus`) prevents goroutine explosion.

```yaml
type: arrival_rate
target_tps: 100       # 100 iterations/second
max_vus: 50           # max concurrent workers
ramp_up: 10s
run_period: 1m
ramp_down: 5s         # optional grace period for in-flight workers
vu_timeout: 2s
```

If all `max_vus` workers are busy when a token arrives, the iteration is **dropped** and `gtest.pacing.dropped_iterations` is incremented. In-flight workers interrupted when the scenario finishes are discarded without false timeout or error reporting.

**Best for**: testing at a specific throughput regardless of response time.

---

## 10. CLI Flags & Execution

```bash
go run main.go --config gtest.yaml --scenario my_scenario
```

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | `gtest.yaml` | Path to config file |
| `--scenario` | *(from config)* | Scenario name to run |
| `--log-level` | `info` | `debug`, `info`, `warn`, `error` |
| `--log-format` | `pretty` | `pretty` (human-readable) or `json` |
| `--report-format` | `console` | `console` or `json` |
| `--report-out` | *(stdout)* | Write report to file |
| `--json-report-out` | *(disabled)* | Write additional JSON report to file |
| `--version` | `false` | Print version and exit |

---

## 11. Patterns & Recipes

### 11.1 HTTP Load Test with Tags

```go
RunVU: func(ctx gtest.ScenarioContext) error {
    endpoints := []string{"/api/users", "/api/orders", "/api/products"}
    for _, ep := range endpoints {
        start := time.Now()
        req, _ := http.NewRequestWithContext(ctx, "GET", baseURL+ep, nil)
        resp, err := client.Do(req)
        elapsed := time.Since(start)

        tags := gtest.Tags{"endpoint": ep}
        ctx.Metrics().Duration("http_latency", tags).Observe(elapsed)

        if err != nil || resp.StatusCode >= 400 {
            ctx.Metrics().Counter("http_errors", tags).Inc()
            ctx.Metrics().Rate("success_rate", gtest.Tags{}).Add(0, 1)
        } else {
            _ = resp.Body.Close()
            ctx.Metrics().Rate("success_rate", gtest.Tags{}).Add(1, 1)
        }

    }
    return nil
},
```

### 11.2 Data Parameterization (`pkg/gtest/data`)

Use the dedicated `github.com/morphy76/gtest/pkg/gtest/data` package to load CSV, JSON, or JSON Lines datasets with thread-safe record distribution strategies:

- **Loaders**: `data.LoadCSV`, `data.LoadJSON`, `data.LoadJSONL` (or `data.LoadCSVFile`, etc.)
- **Strategies**:
  - `data.Sequential`: Round-robins across records deterministically by VU ID and iteration (requires non-nil context; returns `data.ErrNilContext` if missing).
  - `data.Random`: Lock-free, thread-safe uniform random record selection powered by `math/rand/v2`.
  - `data.UniquePerVU`: Distributes records deterministically per Virtual User ID (requires non-nil context; returns `data.ErrNilContext` if missing).
  - `data.SharedQueue`: Dispenses each record exactly once across concurrent VUs until exhausted (`data.ErrDatasetExhausted`).

```go
Setup: func(ctx gtest.ScenarioContext) (map[string]any, error) {
    ds, err := data.LoadCSVFile("testdata/users.csv", data.Sequential)
    if err != nil {
        return nil, err
    }
    return map[string]any{"dataset": ds}, nil
},

RunVU: func(ctx gtest.ScenarioContext) error {
    ds := ctx.GlobalState("dataset").(*data.DataSet)
    record, err := ds.Next(ctx)
    if err != nil {
        return err
    }

    username := record["username"]
    userID := record["user_id"]
    // ... make request with dataset fields ...
    return nil
},
```

### 11.3 Thinking Time & Interaction Delay

Thinking time simulates human reading, processing, or decision delays between user actions, conversation turns, or multi-step requests. Thinking time is **explicitly invoked by the test developer** using `ctx.Sleep()`, which actively respects `ctx.Done()` for immediate cancellation during ramp-down or teardown.

#### Option A: Declarative via `gtest.yaml`

Configure an `interaction_delay` strategy in `gtest.yaml`:

```yaml
scenarios:
  conversation_flow:
    type: constant_vus
    vus: 10
    run_period: 2m
    vu_timeout: 5s
    # In-iteration thinking time strategy invoked via ctx.Sleep():
    interaction_delay:
      type: range        # "fixed" | "range" | "expo" | "gaussian"
      min: 200ms
      max: 1s
```

Then call `ctx.Sleep()` at the exact points in your test logic where a user pause occurs (e.g. after receiving a bot response before sending the next message, only if turns remain):

```go
RunVU: func(ctx gtest.ScenarioContext) error {
    // Action 1: Receive bot response / browse product
    // ...

    // Thinking time before next user interaction (uses configured YAML strategy):
    if err := ctx.Sleep(); err != nil {
        return err // Cancelled due to test termination
    }

    // Action 2: Send next customer message / checkout
    return nil
},
```

#### Option B: Explicit Duration Sleep

Pass an explicit duration to override scenario defaults:

```go
RunVU: func(ctx gtest.ScenarioContext) error {
    if err := ctx.Sleep(300 * time.Millisecond); err != nil {
        return err
    }
    return nil
},
```

#### Option C: Custom Mathematical Delay Generators

`gtest` provides 4 mathematical distribution generators:
- `gtest.FixedDelay(d)` — constant duration
- `gtest.RangeDelay(min, max)` — uniform random $U(\text{min}, \text{max})$
- `gtest.ExpoDelay(mean, min, max)` — exponential distribution (Poisson arrival) with optional clamping
- `gtest.GaussianDelay(mean, stdDev, min, max)` — normal distribution $N(\mu, \sigma)$ with non-negative guarantee and optional clamping

```go
thinkTimer := gtest.ExpoDelay(500*time.Millisecond, 100*time.Millisecond, 2*time.Second)

RunVU: func(ctx gtest.ScenarioContext) error {
    if err := ctx.Sleep(thinkTimer.Next()); err != nil {
        return err
    }
    return nil
},
```



### 11.4 Multi-Step User Journey

```go
RunVU: func(ctx gtest.ScenarioContext) error {
    // Step 1: Login
    token, err := login(ctx, baseURL, username, password)
    if err != nil {
        return fmt.Errorf("login failed: %w", err)
    }

    // Step 2: Browse catalog
    products, err := browseCatalog(ctx, baseURL, token)
    if err != nil {
        return fmt.Errorf("browse failed: %w", err)
    }

    // Step 3: Add to cart
    if err := addToCart(ctx, baseURL, token, products[0].ID); err != nil {
        return fmt.Errorf("add to cart failed: %w", err)
    }

    // Step 4: Checkout
    if err := checkout(ctx, baseURL, token); err != nil {
        return fmt.Errorf("checkout failed: %w", err)
    }

    return nil
},
```

### 11.5 Conditional Metric Recording with Tags

```go
if resp.StatusCode == http.StatusOK {
    ctx.Metrics().Counter("http_2xx", gtest.Tags{}).Inc()
} else if resp.StatusCode >= 500 {
    ctx.Metrics().Counter("http_5xx", gtest.Tags{}).Inc()
}
```

### 11.6 Execution Summary Hook (Slack Webhooks & Custom Metrics)

Use `HandleSummary` to receive the full structured summary after report generation for notifications, metrics export, or CI artifact generation:

```go
HandleSummary: func(ctx context.Context, summary gtest.SummaryData) error {
    // 1. Check overall SLA pass/fail status
    if !summary.Passed {
        postSlackAlert(fmt.Sprintf("❌ Load test SLA breached for %s (%s)", summary.Scenario, summary.SuiteName))
    }

    // 2. Read specific metric aggregates
    totalRequests := summary.Counter("http_requests_total")
    successRate := summary.Rate("success_rate")
    latency := summary.Metric("http_request_duration")

    if latency != nil {
        fmt.Printf("Summary: %d reqs, success rate: %.2f%%, p95 latency: %v\n",
            totalRequests, successRate*100, latency.P95)
    }

    // 3. Export custom JSON document
    jsonBytes, err := summary.JSON()
    if err == nil {
        _ = os.WriteFile("ci-summary.json", jsonBytes, 0644)
    }

    return nil
},
```

### 11.7 Inline Assertions (Checks)

```go
RunVU: func(ctx gtest.ScenarioContext) error {
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    // Assert HTTP status 200 without aborting iteration on check fail
    ctx.Check("status is 200", func() string {
        if resp.StatusCode != 200 {
            return fmt.Sprintf("expected 200, got %d", resp.StatusCode)
        }
        return ""
    })

    return nil
}
```

---


## 12. Troubleshooting

### "gtest: RegisterScenario called with nil RunVU"

**Cause**: `Scenario.RunVU` is nil.  
**Fix**: Every scenario must have a non-nil `RunVU` function.

### VU iterations all timing out

**Cause**: `vu_timeout` is too short for your workload.  
**Fix**: Increase `vu_timeout` in `gtest.yaml`. For SSE/WebSocket scenarios, use a longer timeout or implement per-event timeouts in your RunVU logic.

### "metric X already registered as Counter"

**Cause**: You used the same metric name for two different types.  
**Fix**: Each metric name is locked to its first-registered type. Use distinct names.

### No iterations completing (0 total)

**Cause**: `run_period` is shorter than one RunVU iteration.  
**Fix**: Increase `run_period` or add `ramp_down` to give in-flight iterations a grace period.

### Arrival rate: many dropped iterations

**Cause**: `max_vus` is too low for `target_tps` given your iteration latency.  
**Formula**: `max_vus ≥ target_tps × avg_iteration_latency_seconds`  
**Fix**: Increase `max_vus` or optimize your RunVU to reduce latency.

---

## 13. Reference Implementations (`examples/`)

The repository includes a comprehensive set of compilable, self-contained example suites under `examples/`. Each example is paired with an in-process mock server (`httptest.Server` or simulated domain model) and a companion `gtest.yaml` configuration:

| Example | Directory | Core Concepts |
|---|---|---|
| **HTTP REST Checkout** | [`examples/http_checkout/`](../examples/http_checkout/) | Standard REST API load test, HDR duration histogram, success rate, constant VU concurrency. |
| **gRPC RPC Service** | [`examples/grpc_user_service/`](../examples/grpc_user_service/) | Open-system arrival rate pacing (`arrival_rate`), target TPS, bounded worker pool (`max_vus`). |
| **Conversational AI Flow** | [`examples/conversation_flow/`](../examples/conversation_flow/) | Real-time Server-Sent Events (SSE) streaming, multi-turn state machine, DSL client architecture. |
| **Thinking Time & Delays** | [`examples/think_time/`](../examples/think_time/) | Multi-step journey, declarative `interaction_delay` (`range`), `ctx.Sleep()`, programmatic `ExpoDelay`. |
| **Inline Checks (Assertions)** | [`examples/checks/`](../examples/checks/) | Inline assertions (`ctx.Check`), non-aborting validations, auto-instrumented check counters and report tables. |
| **Data Parameterization** | [`examples/data_parameterization/`](../examples/data_parameterization/) | `pkg/gtest/data` dataset ingestion (CSV, JSON, JSONL) with `Sequential`, `Random`, and `SharedQueue` strategies. |
| **SLA Thresholds & Quality Gates** | [`examples/sla_thresholds/`](../examples/sla_thresholds/) | Multi-metric thresholds, percentile gates, rate assertions, and early test termination with `abort_on_fail`. |
| **Execution Summary Hook** | [`examples/handle_summary/`](../examples/handle_summary/) | `HandleSummary` lifecycle hook, summary data inspection, programmatic webhook dispatch and artifact generation. |

### Running Examples

Run any example directly by navigating to its directory:

```bash
# Run the think_time example:
cd examples/think_time
go run -tags=gtest_example .

# Run the checks example:
cd examples/checks
go run -tags=gtest_example .

# Run the data parameterization example:
cd examples/data_parameterization
go run -tags=gtest_example .

# Run the SLA thresholds example:
cd examples/sla_thresholds
go run -tags=gtest_example .

# Run the summary hook example:
cd examples/handle_summary
go run -tags=gtest_example .
```

Verify that all examples compile cleanly across the entire workspace:

```bash
make test-examples
```

