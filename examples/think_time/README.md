# Thinking Time & Interaction Delays Example

A reference example demonstrating how to model realistic human reading, browsing, and decision pauses between actions using `ctx.Sleep()`.

---

## Concept Overview

Realistic load tests simulate human delays (think time) between user journey actions:
- **Declarative Think Time (`interaction_delay`)**: Configured in `gtest.yaml` and invoked cleanly via `ctx.Sleep()` with no arguments.
- **Programmatic Generators**: Statistical distribution generators (`gtest.FixedDelay`, `gtest.RangeDelay`, `gtest.ExpoDelay`, `gtest.GaussianDelay`) initialized in `Setup` and invoked via `ctx.Sleep(d)`.
- **Responsive Cancellation**: `ctx.Sleep()` actively monitors `ctx.Done()`. If the scenario duration expires or an early abort occurs, sleeping VUs wake up immediately without hanging or delaying teardown.
- **Multi-Step Workflows**: Measuring individual step latencies (e.g. `catalog_view_duration`, `add_to_cart_duration`, `checkout_duration`) alongside overall flow success rates.

---

## Key Files

| File | Description |
|---|---|
| [`main.go`](main.go) | 3-step e-commerce journey (`catalog` -> `cart/add` -> `checkout`) utilizing declarative `ctx.Sleep()` and programmatic `gtest.ExpoDelay()`. Includes an in-process mock server. |
| [`gtest.yaml`](gtest.yaml) | Configuration specifying `interaction_delay` range strategy (`20ms` to `60ms`) and per-step latency thresholds. |

---

## How to Run

From the repository root:

```bash
go run -tags=gtest_example ./examples/think_time --config ./examples/think_time/gtest.yaml
```

Or from within the example directory:

```bash
cd examples/think_time
go run -tags=gtest_example .
```

---

## Configuration Breakdown (`gtest.yaml`)

```yaml
version: "1.0"
default_scenario: user_journey_with_think_time

scenarios:
  user_journey_with_think_time:
    type: constant_vus
    vus: 3
    ramp_up: 50ms
    run_period: 300ms
    ramp_down: 200ms
    vu_timeout: 2s

    # Declarative thinking time used by ctx.Sleep()
    interaction_delay:
      type: range        # Uniform random distribution U(min, max)
      min: 20ms
      max: 60ms

    thresholds:
      - metric: catalog_view_duration
        stat: p95
        operator: "<"
        target: "100ms"
      - metric: add_to_cart_duration
        stat: p95
        operator: "<"
        target: "100ms"
      - metric: checkout_duration
        stat: p95
        operator: "<"
        target: "100ms"
      - metric: user_flow_success_rate
        stat: rate
        operator: ">="
        target: "0.95"
```

---

## Expected Output

```text
================================================================================
                        GTEST LOAD TEST SUMMARY
================================================================================
Scenario:     user_journey_with_think_time    Version: dev
Mode:         constant_vus (3 VUs)            Commit:  none
Duration:     00:00:00  (ramp-up: 50ms | run: 300ms | ramp-down: 200ms)
Iterations:   16 total  |  0 failed (0.00%)  |  0 timeout

BUILT-IN METRICS
────────────────────────────────────────────────────────────────
gtest.vu.iterations_total      Counter    16
gtest.vu.iterations_failed     Counter    0
gtest.vu.iterations_timeout    Counter    0
gtest.vu.panics                Counter    0
gtest.vu.pretest_errors        Counter    0
gtest.checks.passed            Counter    0
gtest.checks.failed            Counter    0

CUSTOM METRICS
────────────────────────────────────────────────────────────────
Metric                         Type       Count    Min     Mean    p95     p99     Max    
add_to_cart_duration           Duration   16       677µs   1.083ms 1.509ms 1.643ms 1.643ms
catalog_view_duration          Duration   16       434µs   739µs   1.047ms 1.713ms 1.713ms
checkout_duration              Duration   16       450µs   954µs   1.296ms 1.589ms 1.589ms
think_time_catalog             Duration   16       21.552ms 37.362ms 52.991ms 54.623ms 54.623ms
user_flow_success_rate         Rate       (rate: 1)
user_journeys_completed_total  Counter    16      

SLA THRESHOLD EVALUATION
────────────────────────────────────────────────────────────────
  [PASS]  catalog_view_duration   p95 < 100ms     → actual: 1.047ms
  [PASS]  add_to_cart_duration    p95 < 100ms     → actual: 1.509ms
  [PASS]  checkout_duration       p95 < 100ms     → actual: 1.296ms
  [PASS]  user_flow_success_rate  rate >= 0.95    → actual: 1
────────────────────────────────────────────────────────────────
OVERALL: PASSED                                         (exit 0)
================================================================================
```
