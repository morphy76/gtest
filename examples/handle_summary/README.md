# Execution Summary Hook (`HandleSummary`) Example

A reference example demonstrating how to consume test execution results programmatically via the `HandleSummary` lifecycle hook and dispatch webhook alerts.

---

## Concept Overview

While `vuhive` automatically formats terminal summaries and JSON report files, many workflows require programmatic processing:
- **`HandleSummary` lifecycle hook**: Executes **once** after all VUs exit and report generation completes.
- **`vuhive.SummaryData` model**: Provides full metadata (`SuiteName`, `Scenario`, `Duration`, `Passed`), metric aggregates (`Counter`, `Rate`, `Metric`), and evaluated `Thresholds`.
- **Use Cases**:
  - Sending notifications to Slack, Discord, MS Teams, or PagerDuty.
  - Exporting custom telemetry to Datadog, Prometheus, or CloudWatch.
  - Generating custom Markdown, HTML, or CSV summary artifacts for CI/CD archives.
  - Triggering downstream deployment or verification pipelines.
- **Safe Execution**: Any error returned by `HandleSummary` is logged to standard error but does not mutate the process exit code.

---

## Key Files

| File | Description |
|---|---|
| [`main.go`](main.go) | Scenario definition with `RunVU` and `HandleSummary` hook inspecting `SummaryData` and posting to an in-process mock webhook endpoint. |
| [`vuhive.yaml`](vuhive.yaml) | Scenario configuration and SLA threshold rules. |

---

## How to Run

From the repository root:

```bash
go run -tags=vuhive_example ./examples/handle_summary --config ./examples/handle_summary/vuhive.yaml
```

Or from within the example directory:

```bash
cd examples/handle_summary
go run -tags=vuhive_example .
```

---

## Configuration Breakdown (`vuhive.yaml`)

```yaml
version: "1.0"
default_scenario: summary_hook_demo

scenarios:
  summary_hook_demo:
    type: constant_vus
    vus: 3
    ramp_up: 50ms
    run_period: 300ms
    ramp_down: 50ms
    vu_timeout: 1s

    thresholds:
      - metric: task_latency
        stat: p95
        operator: "<"
        target: "100ms"
      - metric: task_success_rate
        stat: rate
        operator: ">="
        target: "0.95"
```

---

## Expected Output

```text
================================================================================
                        VUHIVE LOAD TEST SUMMARY
================================================================================
Scenario:     summary_hook_demo               Version: dev
Mode:         constant_vus (3 VUs)            Commit:  none
Duration:     00:00:00  (ramp-up: 50ms | run: 300ms | ramp-down: 50ms)
Iterations:   2821 total  |  0 failed (0.00%)  |  0 timeout

BUILT-IN METRICS
────────────────────────────────────────────────────────────────
vuhive.vu.iterations_total      Counter    2821
vuhive.vu.iterations_failed     Counter    0
vuhive.vu.iterations_timeout    Counter    0
vuhive.vu.panics                Counter    0
vuhive.vu.pretest_errors        Counter    0
vuhive.checks.passed            Counter    0
vuhive.checks.failed            Counter    0

CUSTOM METRICS
────────────────────────────────────────────────────────────────
Metric                         Type       Count    Min     Mean    p95     p99     Max    
task_latency                   Duration   2821     257µs   344µs   535µs   757µs   1.862ms
task_success_rate              Rate       (rate: 1)
tasks_completed_total          Counter    2821    

SLA THRESHOLD EVALUATION
────────────────────────────────────────────────────────────────
  [PASS]  task_latency            p95 < 100ms     → actual: 535µs
  [PASS]  task_success_rate       rate >= 0.95    → actual: 1
────────────────────────────────────────────────────────────────
OVERALL: PASSED                                         (exit 0)
================================================================================

--- [HandleSummary Hook Invoked] ---
Suite:       Execution Summary Hook Demo Suite
Scenario:    summary_hook_demo
Duration:    350.262417ms
SLA Verdict: Passed=true
Total Tasks: 2821 | Success Rate: 100.00%
Latency p95: 535µs | Max: 1.862ms
Threshold [PASS]: task_latency p95 100ms (actual: 535µs)
Threshold [PASS]: task_success_rate rate 0.95 (actual: 1)
Successfully delivered notification payload to webhook.
------------------------------------
```
