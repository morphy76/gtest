# SLA Thresholds & Quality Gates Example

A reference example demonstrating how to configure declarative Service Level Agreement (SLA) quality gates, evaluate latency percentiles, and trigger real-time early test termination with `abort_on_fail`.

---

## Concept Overview

Load tests must enforce measurable performance and reliability criteria:
- **Declarative Quality Gates (`thresholds`)**: Specify pass/fail assertions over metric statistics (`p50`, `p90`, `p95`, `p99`, `mean`, `max`, `count`, `rate`, `value`).
- **Operators**: `<`, `<=`, `>`, `>=`.
- **Early Stop / Graceful Abort (`abort_on_fail`)**: When configured on a threshold, `gtest` continuously evaluates the condition in real-time during execution. If breached, the test is immediately aborted (`OVERALL: ABORTED (exit 1)`), preventing wasted test duration or cascading backend failures.
- **Grace Period (`delay_abort_eval`)**: Delays early abort evaluation to allow initial warm-up and ramp-up phases to stabilize before enforcing strict gates.
- **CI/CD Integration**: The process returns exit code `0` if all thresholds pass, or `1` if any threshold fails or is aborted.

---

## Key Files

| File | Description |
|---|---|
| [`main.go`](main.go) | Scenario with multi-stage API calls (`/api/orders` -> `/api/payments`), duration metrics with endpoint tagging, gauge metrics, and error counters. Includes an in-process mock server with latency jitter. |
| [`gtest.yaml`](gtest.yaml) | Configuration with latency percentile gates (p95, p99), rate thresholds, check assertions, and `abort_on_fail: true`. |

---

## How to Run

From the repository root:

```bash
go run -tags=gtest_example ./examples/sla_thresholds --config ./examples/sla_thresholds/gtest.yaml
```

Or from within the example directory:

```bash
cd examples/sla_thresholds
go run -tags=gtest_example .
```

---

## Configuration Breakdown (`gtest.yaml`)

```yaml
version: "1.0"
default_scenario: sla_quality_gates

scenarios:
  sla_quality_gates:
    type: constant_vus
    vus: 5
    ramp_up: 50ms
    run_period: 350ms
    ramp_down: 100ms
    vu_timeout: 1s

    thresholds:
      # Latency SLA quality gates
      - metric: order_placement_latency
        stat: p95
        operator: "<"
        target: "100ms"
      - metric: order_placement_latency
        stat: p99
        operator: "<"
        target: "200ms"
      - metric: payment_auth_latency
        stat: p95
        operator: "<"
        target: "100ms"

      # Success rate quality gates
      - metric: order_success_rate
        stat: rate
        operator: ">="
        target: "0.95"
      - metric: payment_success_rate
        stat: rate
        operator: ">="
        target: "0.95"

      # Error quality gate with real-time early abort
      - metric: api_errors_total
        stat: count
        operator: "<="
        target: "0"
        abort_on_fail: true       # Stop test instantly if errors occur
        delay_abort_eval: 100ms   # Allow 100ms warm-up before monitoring
      - metric: gtest.checks.failed
        stat: count
        operator: "<="
        target: "0"
```

---

## Expected Output

```text
================================================================================
                        GTEST LOAD TEST SUMMARY
================================================================================
Scenario:     sla_quality_gates               Version: dev
Mode:         constant_vus (5 VUs)            Commit:  none
Duration:     00:00:00  (ramp-up: 50ms | run: 350ms | ramp-down: 100ms)
Iterations:   62 total  |  0 failed (0.00%)  |  0 timeout

BUILT-IN METRICS
────────────────────────────────────────────────────────────────
gtest.vu.iterations_total      Counter    62
gtest.vu.iterations_failed     Counter    0
gtest.vu.iterations_timeout    Counter    0
gtest.vu.panics                Counter    0
gtest.vu.pretest_errors        Counter    0
gtest.checks.passed            Counter    124
gtest.checks.failed            Counter    0

CHECKS
────────────────────────────────────────────────────────────────
Check Name                     Passed     Failed   Pass %  
order status is 200            62         0        100.00%
payment status is 200          62         0        100.00%

CUSTOM METRICS
────────────────────────────────────────────────────────────────
Metric                         Type       Count    Min     Mean    p95     p99     Max    
api_errors_total               Counter    0       
api_requests_total             Counter    124     
concurrent_operations          Gauge      (value: 5)
order_placement_latency        Duration   62       6.132ms 15.951ms 24.895ms 25.839ms 26.495ms
order_success_rate             Rate       (rate: 1)
payment_auth_latency           Duration   62       5.54ms  15.736ms 25.135ms 25.535ms 26.111ms
payment_success_rate           Rate       (rate: 1)

SLA THRESHOLD EVALUATION
────────────────────────────────────────────────────────────────
  [PASS]  order_placement_latency p95 < 100ms     → actual: 24.895ms
  [PASS]  order_placement_latency p99 < 200ms     → actual: 25.839ms
  [PASS]  payment_auth_latency    p95 < 100ms     → actual: 25.135ms
  [PASS]  order_success_rate      rate >= 0.95    → actual: 1
  [PASS]  payment_success_rate    rate >= 0.95    → actual: 1
  [PASS]  api_errors_total        count <= 0      → actual: 0
  [PASS]  gtest.checks.failed     count <= 0      → actual: 0
────────────────────────────────────────────────────────────────
OVERALL: PASSED                                         (exit 0)
================================================================================
```
