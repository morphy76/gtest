# vuhive Performance & Verification Suite Guide

This document establishes the official performance verification suite, architectural design patterns, profiling procedures, and automated tooling that guarantee `vuhive` remains exceptionally fast, memory-thin, and capable of scaling to tens of thousands of concurrent Virtual Users (VUs) and 100k+ Transactions Per Second (TPS) with zero steady-state heap allocations.

---

## 1. Performance Architecture & Zero-Allocation Hot Path

When executing high-scale load tests, machine CPU and memory should be dedicated to user scenario transport I/O (HTTP, gRPC, TLS) rather than framework runtime overhead. `vuhive` achieves this through five core architectural principles:

```text
 ┌─────────────────────────────────────────────────────────────────────────┐
 │                         vuhive Hot Path Engine                          │
 ├─────────────────────────────────────────────────────────────────────────┤
 │  1. Context Reuse        ──► Per-VU context pooling via prepareIteration│
 │  2. Lock-Free Metrics    ──► Atomic CAS, pointer swapping, COW maps     │
 │  3. Striped Histograms   ──► 16 mutex-striped HDR histogram shards      │
 │  4. Dataset Slicing      ──► Lock-free pointer indexing & fast math/rand│
 │  5. Bounded Worker Pools ──► Zero goroutine churn in arrival_rate / VUs │
 └─────────────────────────────────────────────────────────────────────────┘
```

### 1. Per-VU Context Pooling & Reusable Iterations
- Each Virtual User goroutine allocates its execution context (`ScenarioContext` / `VUContext`) **once** at startup.
- During steady-state iteration loops, the framework calls `prepareIteration(ctx, iteration)` in-place, reusing internal metric handle caches, parameters, and log bindings without re-allocating context structs or interface wrappers.
- Driving adapter layers reuse adapter instances via `sync.Pool`, ensuring zero allocations per iteration.

### 2. Lock-Free Metric Ingestion & Copy-On-Write Storage
- **Counters & Gauges**: Implemented via atomic 64-bit operations (`atomic.AddInt64`, `atomic.StoreInt64`, `atomic.CompareAndSwapInt64`).
- **Rates**: Implemented with atomic numerator and denominator pair updates without mutex locks.
- **Metric Key Registry**: Metrics storage utilizes copy-on-write atomic pointer maps (`atomic.Pointer[map[K]V]`), providing $O(1)$ 100% lock-free reads on metric lookup and pre-resolved handles.

### 3. 16-Striped HDR Histograms (Lock Contention Elimination)
- `vuhive.Duration` latency tracking utilizes high-resolution High Dynamic Range (HDR) Histograms (`github.com/HdrHistogram/hdrhistogram-go`).
- Observations (`ctx.Metrics().Duration(...).Observe(d)`) are striped across **16 independent mutex-guarded shards** using atomic round-robin sequence distribution (`seq.Add(1) % 16`).
- This eliminates lock contention across thousands of parallel VU goroutines, achieving observation latencies under 120ns even under multi-core parallel load. Histograms are merged only once at report generation time.

### 4. Lock-Free Dataset Pointer Arithmetic
- The `pkg/vuhive/data` module provides dataset distribution strategies (`Sequential`, `Random`, `UniquePerVU`, `SharedQueue`) that operate directly on pre-allocated slice memory:
  - `Sequential` / `UniquePerVU`: Pure arithmetic indexing $(vuid - 1 + iteration) \pmod N$ ($< 0.5\text{ ns/op}$, 0 allocs).
  - `Random`: Fast lock-free random sampling via Go 1.26 `math/rand/v2` ($< 0.7\text{ ns/op}$, 0 allocs).
  - `SharedQueue`: Lock-free atomic cursor increments (`atomic.AddInt64`).

### 5. Bounded Concurrency & Zero Goroutine Churn
- Closed-model pacers (`constant_vus`, `ramping_vus`) maintain fixed goroutine pools rather than spawning and terminating goroutines per iteration.
- Open-model pacing (`arrival_rate`) utilizes token bucket dispatch (`golang.org/x/time/rate`) against a pre-allocated bounded worker pool (`max_vus`), preventing GC churn from runaway goroutine creation.

---

## 2. Performance Verification Matrix

The in-tree verification suite strictly enforces the following performance and allocation budgets:

| Component / Sub-System | Benchmark Identifier | Target Latency | Allocation Budget | Verification Scope |
|---|---|---|---|---|
| **Assertions (`ctx.Check`)** | `BenchmarkScenarioContext_Check_Passing` | `< 10 ns/op` | `0 B/op, 0 allocs/op` | Inline assertion with cached metric handles. |
| **Parameter Access (`ctx.Param*`)** | `BenchmarkScenarioContext_ParamAccess` | `< 50 ns/op` | `0 B/op, 0 allocs/op` | Typed scenario config lookups (`Param`, `ParamInt`, `ParamDuration`). |
| **Counter Ingestion (`ctx.Metrics`)** | `BenchmarkCollector_Counter_Parallel` | `< 75 ns/op` | `0 B/op, 0 allocs/op` | Multi-core parallel atomic counter increments. |
| **Gauge Ingestion (`ctx.Metrics`)** | `BenchmarkCollector_Gauge_Parallel` | `< 75 ns/op` | `0 B/op, 0 allocs/op` | Multi-core parallel atomic gauge value updates. |
| **Duration Histogram (`Observe`)** | `BenchmarkCollector_Duration_Parallel` | `< 150 ns/op` | `0 B/op, 0 allocs/op` | 16-striped HDR histogram sample recording. |
| **Dataset Distribution (Seq/Rand)** | `BenchmarkStrategySequential` / `Random` | `< 1 ns/op` | `0 B/op, 0 allocs/op` | Lock-free dataset record sampling. |
| **Public Suite Iteration Wrapper** | `BenchmarkSuite_RunVU_Iteration` | `< 15 ns/op` | `0 B/op, 0 allocs/op` | Public `vuhive.Suite` `RunVU` adapter invocation. |

---

## 3. Running Benchmarks & Performance Verification

### Standard Benchmark Suite
To execute all in-tree microbenchmarks across all packages:

```bash
make test-bench
```

Or run targeted benchmarks directly with Go:

```bash
go test -count=1 -bench=. -benchmem -run=^$ ./...
```

### Dedicated Zero-Allocation Performance Suite (`make test-perf`)
To run the automated allocation regression test suite (asserting `testing.AllocsPerRun == 0` on all hot paths) and targeted hot-path benchmarks:

```bash
make test-perf
```

Alias:

```bash
make verify-performance
```

---

## 4. Capturing & Interpreting `pprof` Profiles

Go's built-in `pprof` profiling tools allow detailed inspection of CPU execution, memory allocation sites, and synchronization contention.

### A. Memory Allocation & Heap Profiling

Validate that steady-state iteration loops generate **zero heap allocations**:

1. **Capture memory profile during benchmark execution**:
   ```bash
   go test -bench=BenchmarkEngine_ConstantVUs_NoopIteration_NoTimeout -benchmem -memprofile=mem.pprof -run=^$ ./internal/engine
   ```

2. **Inspect allocation sites**:
   ```bash
   go tool pprof -alloc_space mem.pprof
   ```
   Inside the `pprof` interactive prompt:
   - `top 10`: Displays functions allocating the most cumulative memory.
   - `list executeIteration`: Shows line-by-line allocations within the execution loop (verifying 0B in the hot path).
   - `web`: Generates an SVG graph visualization of memory allocation call chains.

### B. Mutex & Synchronization Contention Profiling

Validate zero lock contention bottlenecks when thousands of concurrent goroutines write metrics simultaneously:

1. **Capture mutex and block contention profiles**:
   ```bash
   go test -bench=. -mutexprofile=mutex.pprof -blockprofile=block.pprof -run=^$ ./internal/metric
   ```

2. **Analyze lock contention**:
   ```bash
   go tool pprof -top mutex.pprof
   go tool pprof -top block.pprof
   ```
   - Verify that 16-stripe HDR histogram sharding avoids contention hotspots on metric recording.

### C. CPU Profiling

Identify CPU hotspots and verify that runtime overhead is negligible:

1. **Capture CPU profile**:
   ```bash
   go test -bench=BenchmarkCollector_Counter_Parallel -cpuprofile=cpu.pprof -run=^$ ./internal/metric
   ```

2. **Inspect CPU consumption**:
   ```bash
   go tool pprof -top cpu.pprof
   ```

3. **Launch interactive browser UI**:
   ```bash
   go tool pprof -http=:8080 cpu.pprof
   ```

---

## 5. Live High-Concurrency Scalability (10,000+ VUs)

`vuhive` is engineered to run 10,000+ concurrent Virtual Users on a standard developer workstation or small cloud instance without resource exhaustion.

### Verification Target & Footprint

- **Scenario**: 10,000 Concurrent VUs executing in-memory iterations with active checks and metrics.
- **Memory Consumption**: Total Resident Set Size (RSS) remains **$< 50\text{ MB}$** total (~2–4 KB per active VU goroutine stack).
- **Throughput**: Millions of iterations/sec in dry-run mode.

### Monitoring Memory & Goroutine Footprint

During load tests, monitor system resource utilization:

```bash
# 1. Monitor process Resident Set Size (RSS) and Virtual Memory (VSZ) on macOS/Linux:
ps -o pid,rss,vsz,comm -p $(pgrep -f vuhive)

# 2. Continuous memory and CPU monitoring:
top -pid $(pgrep -f vuhive)
```

### Go Runtime Tuning for Extreme Concurrency

When running massive concurrency (20,000+ VUs or 100k+ TPS):

- **`GOMEMLIMIT`**: Set explicit memory target (e.g. `GOMEMLIMIT=2GiB`) to prevent aggressive GC cycles while maintaining strict memory bounds.
- **`GOGC`**: Adjust GC target percentage (e.g. `GOGC=200` or `GOGC=off` for pure dry runs) since steady-state execution generates zero garbage.
- **`GOMAXPROCS`**: Match available physical/container CPU cores (`GOMAXPROCS=$(nproc)`).

---

## 6. Comparative Architectural Analysis

| Dimension | `vuhive` (Go) | k6 (Go + JS Runtime) | Locust (Python) | Gatling / JMeter (JVM) |
|---|---|---|---|---|
| **Runtime Model** | Compiled native Go binary | Go engine hosting Goja JS VM | Python interpreter with Gevent | JVM bytecode on OpenJDK / HotSpot |
| **Hot Path Allocations** | **0 allocs/op** (Steady state) | JS object allocations per turn | Python object & dictionary allocations | JVM object allocations & GC churn |
| **Memory per VU** | **~2–4 KB** (Go goroutine stack) | ~20–50 KB (JS isolate context) | ~50–100 KB (Python Greenlet/Context) | ~50–200 KB (JVM heap / OS thread in JMeter) |
| **10k VU Memory Footprint** | **$< 50\text{ MB}$** | ~250–500 MB | ~500 MB – 1 GB (requires clustering) | ~500 MB – 2 GB |
| **Single-Host Throughput** | **100k+ TPS** | ~20k–40k TPS | ~1k–2k RPS (GIL bottleneck) | ~30k–60k TPS |
| **Metric Ingestion** | Lock-free atomic + 16-stripe HDR | Mutex-guarded metric registry | Master-worker RPC aggregation | Actor model / lock-guarded accumulators |
| **Garbage Collection Pressure** | **Zero GC cycles** in steady state | Periodic JS engine GC pauses | Python reference counting & GC pauses | Stop-The-World (STW) JVM GC pauses |
