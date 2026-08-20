# HTTP Module Example

Demonstrates the **vuhive HTTP module** (`pkg/vuhive/http`), which provides an instrumented HTTP client that automatically records latency, request counts, and failure rates — eliminating boilerplate metric recording from `RunVU`.

## Concept Overview

In a typical vuhive load test, developers manually time HTTP requests, record duration histograms, increment counters, and track failure rates. The HTTP module eliminates this boilerplate by wrapping `net/http.Client` with automatic instrumentation.

**Before (manual instrumentation):**
```go
start := time.Now()
resp, err := client.Do(req)
elapsed := time.Since(start)
ctx.Metrics().Duration("http_request_duration", tags).Observe(elapsed)
ctx.Metrics().Counter("http_requests_total", tags).Inc()
if err != nil || resp.StatusCode >= 400 {
    ctx.Metrics().Rate("checkout_success_rate", tags).Add(0, 1)
}
```

**After (HTTP module):**
```go
resp, err := client.Get(ctx, serverURL+"/checkout")
// All metrics recorded automatically!
```

## Key Files

| File | Description |
|---|---|
| `main.go` | Scenario using the instrumented HTTP client |
| `vuhive.yaml` | Configuration with SLA thresholds on built-in HTTP metrics |

## How to Run

```bash
go run -tags vuhive_example ./examples/http_module
```

## Automatic Metrics

The HTTP module records the following metrics for every request:

| Metric | Type | Description |
|---|---|---|
| `vuhive.http.req_duration` | Duration | Total request latency (HDR histogram) |
| `vuhive.http.req_failed` | Rate | Failed request ratio (non-2xx or transport error) |
| `vuhive.http.reqs` | Counter | Total request count |

### Opt-in Phase Metrics

Enable with `vuhivehttp.WithDetailedTiming()`:

| Metric | Type | Description |
|---|---|---|
| `vuhive.http.req_connecting` | Duration | TCP connection establishment time |
| `vuhive.http.req_tls_handshaking` | Duration | TLS handshake time |
| `vuhive.http.req_sending` | Duration | Request write time |
| `vuhive.http.req_receiving` | Duration | Response read time |

## Configuration Breakdown

```yaml
thresholds:
  - metric: vuhive.http.req_duration    # Built-in HTTP module metric
    stat: p95
    operator: "<"
    target: "200ms"                     # 95th percentile must be under 200ms
  - metric: vuhive.http.req_failed
    stat: rate
    operator: "<="
    target: "0.01"                      # Less than 1% failure rate
```

## Expected Output

```text
SCENARIO EXECUTION SUMMARY
════════════════════════════════════════════════════════════════
Suite:      HTTP Module Demo Suite
Scenario:   http_module_demo
Duration:   14.0s (ramp_up: 2s, run: 10s, ramp_down: 2s)
VUs:        10 (constant_vus)

CHECKS
────────────────────────────────────────────────────────────────
  ✓ status_200         100.00% (XXX/XXX)
  ✓ order_created      100.00% (XXX/XXX)

CUSTOM METRICS
────────────────────────────────────────────────────────────────
  vuhive.http.req_duration .......... p50=Xms   p95=Xms   p99=Xms
  vuhive.http.req_failed ............ 0.00%
  vuhive.http.reqs .................. XXX

THRESHOLDS
────────────────────────────────────────────────────────────────
  [PASS]  vuhive.http.req_duration  p95 < 200ms
  [PASS]  vuhive.http.req_failed   rate <= 0.01
```
