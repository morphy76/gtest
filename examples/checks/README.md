# Inline Assertions (Checks) Example

A focused reference example demonstrating how to validate functional conditions during load test iterations using `ctx.Check()`.

---

## Concept Overview

In load testing, validating API correctness without prematurely aborting virtual users is essential:
- **`ctx.Check(name, fn)`** allows you to perform inline assertions (status codes, headers, response JSON values) in `RunVU`.
- **Non-Fatal**: If a check fails, the iteration continues normally, recording the failure for statistical analysis.
- **Contract**: The check function returns `""` (empty string) on pass, or a descriptive reason string on failure.
- **Auto-Instrumentation**: `gtest` automatically tracks `gtest.checks.passed` and `gtest.checks.failed` counters tagged with the check name.
- **Dedicated Reporting**: Check pass rates and failure counts are displayed in a formatted `CHECKS` summary table.
- **SLA Threshold Integration**: You can define SLA quality gates on check metrics (e.g. `gtest.checks.failed count <= 0`).

---

## Key Files

| File | Description |
|---|---|
| [`main.go`](main.go) | Scenario demonstrating HTTP status code, Content-Type header, and JSON payload checks. Includes an in-process mock API server. |
| [`gtest.yaml`](gtest.yaml) | Configuration with SLA thresholds asserting zero check failures (`gtest.checks.failed <= 0`). |

---

## How to Run

From the repository root:

```bash
go run -tags=gtest_example ./examples/checks --config ./examples/checks/gtest.yaml
```

Or from within the example directory:

```bash
cd examples/checks
go run -tags=gtest_example .
```

---

## Configuration Breakdown (`gtest.yaml`)

```yaml
version: "1.0"
default_scenario: checks_demo

scenarios:
  checks_demo:
    type: constant_vus
    vus: 4
    ramp_up: 50ms
    run_period: 300ms
    ramp_down: 50ms
    vu_timeout: 1s

    thresholds:
      # Enforce zero check failures across all iterations
      - metric: gtest.checks.failed
        stat: count
        operator: "<="
        target: "0"

      # Ensure at least 10 checks executed and passed
      - metric: gtest.checks.passed
        stat: count
        operator: ">="
        target: "10"
```

---

## Expected Output

```text
================================================================================
                        GTEST LOAD TEST SUMMARY
================================================================================
Scenario:     checks_demo                     Version: dev
Mode:         constant_vus (4 VUs)            Commit:  none
Duration:     00:00:00  (ramp-up: 50ms | run: 300ms | ramp-down: 50ms)
Iterations:   13234 total  |  0 failed (0.00%)  |  0 timeout

BUILT-IN METRICS
────────────────────────────────────────────────────────────────
gtest.vu.iterations_total      Counter    13234
gtest.vu.iterations_failed     Counter    0
gtest.vu.iterations_timeout    Counter    0
gtest.vu.panics                Counter    0
gtest.vu.pretest_errors        Counter    0
gtest.checks.passed            Counter    39702
gtest.checks.failed            Counter    0

CHECKS
────────────────────────────────────────────────────────────────
Check Name                     Passed     Failed   Pass %  
content-type is json           13234      0        100.00%
response status is success     13234      0        100.00%
status code is 200             13234      0        100.00%

SLA THRESHOLD EVALUATION
────────────────────────────────────────────────────────────────
  [PASS]  gtest.checks.failed     count <= 0      → actual: 0
  [PASS]  gtest.checks.passed     count >= 10     → actual: 39702
────────────────────────────────────────────────────────────────
OVERALL: PASSED                                         (exit 0)
================================================================================
```
