# Groups (Transaction Boundaries) Example

A focused reference example demonstrating how to organize Virtual User (`RunVU`) logic into named transaction boundaries with automatic duration tracking and per-step latency SLA thresholds using `ctx.Group()`.

---

## Concept Overview

When testing multi-step user journeys (such as login → browse → cart → payment), measuring end-to-end iteration duration alone is insufficient to isolate performance bottlenecks.

`vuhive` provides **Groups (Transaction Boundaries)** inspired by k6 `group()` and Gatling `exec().group()`:

- **`ctx.Group(name, fn)`**: Executes the function `fn` inside a named transaction boundary.
- **Auto-Instrumentation**: Automatically records execution latency to an HDR duration histogram named `vuhive.group.<name>.duration`.
- **Nested Boundaries**: Groups can be arbitrarily nested. Child group metric names are concatenated using the `::` delimiter (e.g. `vuhive.group.03_Checkout::Submit_Payment.duration`).
- **Error Propagation & Safe Metrics**: If `fn` returns an error or panics, the elapsed duration up to that point is still recorded before propagating the error or recovering the panic.
- **Dedicated Reporting**: Group latencies are displayed in a formatted `GROUPS` table in the console report and as a structured `groups` array in JSON reports.
- **Granular SLA Thresholds**: Quality gates in `vuhive.yaml` can directly target per-step and nested group latencies (e.g. `vuhive.group.01_Login.duration p95 < 200ms`).

---

## Key Files

| File | Description |
|---|---|
| [`main.go`](main.go) | Multi-step e-commerce flow with sequential steps (`01_Login`, `02_Browse_Catalog`) and nested steps (`03_Checkout` containing `Add_To_Cart` and `Submit_Payment`). Includes an in-process mock HTTP API. |
| [`vuhive.yaml`](vuhive.yaml) | Configuration with per-group SLA thresholds verifying p95 latencies for each step. |

---

## How to Run

From the repository root:

```bash
go run -tags=vuhive_example ./examples/groups --config ./examples/groups/vuhive.yaml
```

Or from within the example directory:

```bash
cd examples/groups
go run -tags=vuhive_example .
```

---

## Configuration Breakdown (`vuhive.yaml`)

```yaml
version: "1.0"
default_scenario: ecommerce_flow

scenarios:
  ecommerce_flow:
    type: constant_vus
    vus: 2
    ramp_up: 50ms
    run_period: 300ms
    ramp_down: 50ms
    vu_timeout: 2s

    thresholds:
      # Step-level latency SLA quality gates
      - metric: "vuhive.group.01_Login.duration"
        stat: p95
        operator: "<"
        target: "200ms"

      - metric: "vuhive.group.02_Browse_Catalog.duration"
        stat: p95
        operator: "<"
        target: "200ms"

      - metric: "vuhive.group.03_Checkout.duration"
        stat: p95
        operator: "<"
        target: "500ms"

      - metric: "vuhive.group.03_Checkout::Submit_Payment.duration"
        stat: p95
        operator: "<"
        target: "250ms"
```

---

## Expected Output

```text
================================================================================
                        VUHIVE LOAD TEST SUMMARY
================================================================================
Scenario:     ecommerce_flow                  Version: dev
Mode:         constant_vus (2 VUs)            Commit:  none
Duration:     00:00:00  (ramp-up: 50ms | run: 300ms | ramp-down: 50ms)
Iterations:   30 total  |  0 failed (0.00%)  |  0 timeout

BUILT-IN METRICS
────────────────────────────────────────────────────────────────
vuhive.vu.iterations_total      Counter    30
vuhive.vu.iterations_failed     Counter    0
vuhive.vu.iterations_timeout    Counter    0
vuhive.vu.panics                Counter    0
vuhive.vu.pretest_errors        Counter    0
vuhive.checks.passed            Counter    120
vuhive.checks.failed            Counter    0

CHECKS
────────────────────────────────────────────────────────────────
Check Name                     Passed     Failed   Pass %  
browse status 200              30         0        100.00%
cart status 200                30         0        100.00%
login status 200               30         0        100.00%
payment status 200             30         0        100.00%

GROUPS
────────────────────────────────────────────────────────────────
Group Name                     Type       Count    Min     Mean    p95     p99     Max    
01_Login                       Duration   30       3.1ms   3.6ms   4.8ms   5.1ms   5.2ms  
02_Browse_Catalog              Duration   30       5.2ms   5.8ms   6.9ms   7.2ms   7.4ms  
03_Checkout                    Duration   30       14.2ms  15.1ms  17.4ms  18.0ms  18.2ms 
03_Checkout::Add_To_Cart       Duration   30       4.1ms   4.7ms   5.6ms   6.0ms   6.1ms  
03_Checkout::Submit_Payment    Duration   30       8.2ms   8.9ms   10.2ms  10.8ms  11.0ms 

SLA THRESHOLD EVALUATION
────────────────────────────────────────────────────────────────
  [PASS]  vuhive.group.01_Login.duration p95 < 200ms     → actual: 4.8ms
  [PASS]  vuhive.group.02_Browse_Catalog.duration p95 < 200ms     → actual: 6.9ms
  [PASS]  vuhive.group.03_Checkout.duration p95 < 500ms     → actual: 17.4ms
  [PASS]  vuhive.group.03_Checkout::Submit_Payment.duration p95 < 250ms     → actual: 10.2ms
  [PASS]  vuhive.checks.failed           count <= 0      → actual: 0
────────────────────────────────────────────────────────────────
OVERALL: PASSED                                         (exit 0)
================================================================================
```
