# Ramping VUs (Spike Testing) Example

A reference example demonstrating dynamic multi-stage workload pacing using the `ramping_vus` engine.

---

## Concept Overview

Simulating realistic traffic profiles often requires dynamic scaling:
- **`ramping_vus` engine**: Dynamically scales the active Virtual User pool up or down across defined time stages.
- **Workload Stages (`stages`)**: Define consecutive target VU counts and durations (e.g. warm-up ramp, steady state, sudden spike, cool down).
- **Spike & Stress Testing**: Evaluate how target systems handle sudden traffic surges and verify latency recovery when load decreases.
- **Graceful Worker Scaling**: Active VUs adjust dynamically in real time without goroutine leaks or artificial pauses.

---

## Key Files

| File | Description |
|---|---|
| [`main.go`](main.go) | Scenario definition with lifecycle hooks, request execution, and metric collection. Includes an in-process mock server with request counting. |
| [`vuhive.yaml`](vuhive.yaml) | 5-stage ramping configuration (ramp up -> hold -> spike -> hold spike -> ramp down) and SLA thresholds. |

---

## How to Run

From the repository root:

```bash
go run -tags=vuhive_example ./examples/ramping_vus --config ./examples/ramping_vus/vuhive.yaml
```

Or from within the example directory:

```bash
cd examples/ramping_vus
go run -tags=vuhive_example .
```

---

## Configuration Breakdown (`vuhive.yaml`)

```yaml
version: "1.0"
default_scenario: spike_test

scenarios:
  spike_test:
    type: ramping_vus
    stages:
      - target: 5
        duration: 200ms   # Stage 1: Ramp up 0 -> 5 VUs
      - target: 5
        duration: 300ms   # Stage 2: Hold steady at 5 VUs
      - target: 15
        duration: 200ms   # Stage 3: Spike up to 15 VUs
      - target: 15
        duration: 300ms   # Stage 4: Hold peak spike
      - target: 0
        duration: 200ms   # Stage 5: Ramp down to 0 VUs
    ramp_down: 100ms      # Grace period for in-flight iterations
    vu_timeout: 1s

    thresholds:
      - metric: api_response_time
        stat: p95
        operator: "<"
        target: "50ms"
      - metric: api_success_rate
        stat: rate
        operator: ">="
        target: "0.99"
```

---

## Expected Output

```text
================================================================================
                        VUHIVE LOAD TEST SUMMARY
================================================================================
Scenario:     spike_test                      Version: dev
Mode:         ramping_vus                     Commit:  none
Duration:     00:00:01  (ramp-up: 0s | run: 0s | ramp-down: 100ms)
Iterations:   9902 total  |  0 failed (0.00%)  |  0 timeout

BUILT-IN METRICS
────────────────────────────────────────────────────────────────
vuhive.vu.iterations_total      Counter    9902
vuhive.vu.iterations_failed     Counter    0
vuhive.vu.iterations_timeout    Counter    0
vuhive.vu.panics                Counter    0
vuhive.vu.pretest_errors        Counter    0
vuhive.checks.passed            Counter    0
vuhive.checks.failed            Counter    0

CUSTOM METRICS
────────────────────────────────────────────────────────────────
Metric                         Type       Count    Min     Mean    p95     p99     Max    
api_requests_total             Counter    9902    
api_response_time              Duration   9902     190µs   1.013ms 1.743ms 1.861ms 10.943ms
api_success_rate               Rate       (rate: 1)

SLA THRESHOLD EVALUATION
────────────────────────────────────────────────────────────────
  [PASS]  api_response_time       p95 < 50ms      → actual: 1.743ms
  [PASS]  api_success_rate        rate >= 0.99    → actual: 1
────────────────────────────────────────────────────────────────
OVERALL: PASSED                                         (exit 0)
================================================================================
```
