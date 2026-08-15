# Data Parameterization Example (`pkg/gtest/data`)

A reference example demonstrating how to ingest external datasets (CSV, JSON, JSONL) and feed dynamic records into Virtual User load iterations using thread-safe distribution strategies.

---

## Concept Overview

Dynamic data feeds simulate realistic multi-user workloads by parameterizing credentials, SKUs, queries, and IDs:
- **`pkg/gtest/data` module**: Provides native loaders for CSV (`LoadCSV` / `LoadCSVFile`), JSON arrays (`LoadJSON` / `LoadJSONFile`), and JSON Lines (`LoadJSONL` / `LoadJSONLFile`).
- **Distribution Strategies**:
  - **`data.Sequential`**: Deterministic round-robin per Virtual User and iteration index: `(vu_id - 1 + iteration) % total_records`.
  - **`data.Random`**: Lock-free, thread-safe uniform random record selection across all rows.
  - **`data.SharedQueue`**: Atomic, thread-safe FIFO single-consumption queue. Returns `data.ErrDatasetExhausted` when empty.
  - **`data.UniquePerVU`**: Partitions records deterministically per Virtual User.
- **Iteration Ingestion**: Records are consumed inside `RunVU` via `ds.Next(ctx)`, returning a `map[string]string` record.

---

## Key Files

| File | Description |
|---|---|
| [`main.go`](main.go) | Scenario demonstrating CSV (`Sequential`), JSON (`Random`), and JSONL (`SharedQueue`) loaders, data retrieval in `RunVU`, and mock HTTP endpoints. |
| [`gtest.yaml`](gtest.yaml) | Scenario configuration with success rate and check failure thresholds. |

---

## How to Run

From the repository root:

```bash
go run -tags=gtest_example ./examples/data_parameterization --config ./examples/data_parameterization/gtest.yaml
```

Or from within the example directory:

```bash
cd examples/data_parameterization
go run -tags=gtest_example .
```

---

## Configuration Breakdown (`gtest.yaml`)

```yaml
version: "1.0"
default_scenario: data_parameterization_flow

scenarios:
  data_parameterization_flow:
    type: constant_vus
    vus: 4
    ramp_up: 50ms
    run_period: 300ms
    ramp_down: 50ms
    vu_timeout: 1s

    thresholds:
      - metric: dataset_success_rate
        stat: rate
        operator: ">="
        target: "0.95"
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
Scenario:     data_parameterization_flow      Version: dev
Mode:         constant_vus (4 VUs)            Commit:  none
Duration:     00:00:00  (ramp-up: 50ms | run: 300ms | ramp-down: 50ms)
Iterations:   1382 total  |  0 failed (0.00%)  |  0 timeout

BUILT-IN METRICS
────────────────────────────────────────────────────────────────
gtest.vu.iterations_total      Counter    1382
gtest.vu.iterations_failed     Counter    0
gtest.vu.iterations_timeout    Counter    0
gtest.vu.panics                Counter    0
gtest.vu.pretest_errors        Counter    0
gtest.checks.passed            Counter    2764
gtest.checks.failed            Counter    0

CHECKS
────────────────────────────────────────────────────────────────
Check Name                     Passed     Failed   Pass %  
product request status is 200  1382       0        100.00%
user request status is 200     1382       0        100.00%

CUSTOM METRICS
────────────────────────────────────────────────────────────────
Metric                         Type       Count    Min     Mean    p95     p99     Max    
dataset_success_rate           Rate       (rate: 1)
parameterized_requests_total   Counter    1382    

SLA THRESHOLD EVALUATION
────────────────────────────────────────────────────────────────
  [PASS]  dataset_success_rate    rate >= 0.95    → actual: 1
  [PASS]  gtest.checks.failed     count <= 0      → actual: 0
────────────────────────────────────────────────────────────────
OVERALL: PASSED                                         (exit 0)
================================================================================
```
