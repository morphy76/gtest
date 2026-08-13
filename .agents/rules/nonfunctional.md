---
trigger: always_on
description: General non-functional requirements, including technology stack locks, monorepo Makefile build system, semantic versioning lifecycle, structured logging, and clean SOLID code principles.
---

# General NFR and System Constraints

This rule enforces general software constraints, technology stack lock-in, monorepo build orchestration, idiomatic Go binary versioning, structured logging guidelines, graceful teardown, and clean SOLID code principles for maintainability and readability.

## 1. Approved Technology Stack

The following software versions, frameworks, and client libraries are strictly locked. Any deviation or addition requires formal approval/constitution amendment:

- **Core Language:** Go 1.26
- **HTTP Routing:** Gin (`github.com/gin-gonic/gin`)
- **Structured Logging:** zerolog (`github.com/rs/zerolog`)
- **Testing Libraries:** testify (`github.com/stretchr/testify`) and testcontainers-go
- **Configuration Engine:** Viper (`github.com/spf13/viper`)

## 2. Monorepo Build System & Go Version Management

A single root-level `Makefile` orchestrates all system build and version injection activities. Never create nested component-specific Makefiles:

- **Target PHONY Declaration:** Every single target declared in the `Makefile` MUST be declared as `.PHONY`.
- **Idiomatic Go Version Injection (`ldflags`):**
  - Component versioning MUST be injected into the Go binary at compile time via `ldflags`.
  - Maintain an `internal/version/` package in each component defining package variables:
    ```go
    package version

    var (
        Version   = "dev"
        Commit    = "none"
        BuildTime = "unknown"
    )
    ```
  - `Makefile` must pass compile flags reading from `VERSION.<component>` and git metadata:
    ```makefile
    VERSION ?= $(shell cat VERSION.$(COMPONENT) 2>/dev/null || echo "0.0.0")
    COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    BUILD_TIME ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
    LDFLAGS := -s -w \
      -X 'github.com/tacito-square/internal/version.Version=$(VERSION)' \
      -X 'github.com/tacito-square/internal/version.Commit=$(COMMIT)' \
      -X 'github.com/tacito-square/internal/version.BuildTime=$(BUILD_TIME)'

    build:
    	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/$(COMPONENT) ./cmd/$(COMPONENT)
    ```
- **Runtime Version Exposure:** All binary applications MUST expose version information via startup logs, `--version` CLI flag, and HTTP `/health` or `/version` endpoints.
- **Required Make Targets:**
  - **Compile:** `build`, `build-<component>`
  - **Testing:** `test` (unit tests), `test-integration`, `test-bench`, `test-race`
  - **Fidelity/Quality:** `lint`, `generate`
- **Automatic Target Help Documentation:** The `make help` command must parse and print descriptions from `## ` comments.

## 3. Component Versioning and Lifecycle

Tacito Square components use independent semantic versions:

- **Version File:** Each component must maintain its current SemVer in a flat text file at the root: `VERSION.keeper`, `VERSION.agent`, `VERSION.operator`, `VERSION.bff`.
- **SemVer Format:** Strictly adhere to Semantic Versioning 2.0.0 (`MAJOR.MINOR.PATCH`).
- **Helm System Versioning:** The parent `deploy/helm/tacito-square/Chart.yaml` version field governs the overall **system version**, which operates independently of component releases.
- **Git Tags Structure:**
  - Component Release tag format: `<component>-v<version>` (e.g. `keeper-v0.2.0`, `agent-v0.1.3`).
  - Helm Chart Release tag format: `chart-<chart-name>-v<version>` (e.g. `chart-tacito-square-v0.2.0`).
- **Atomic Bumps:** Perform version bumps as standalone atomic commits (do not mix code adjustments and version bumps in a single commit).

## 4. Structured Logging Strategy (`zerolog`)

All components MUST utilize `github.com/rs/zerolog` for structured JSON logging adhering to the following rules:

- **Context-Aware Logging:** Extract logger instances from `context.Context` (`zerolog.Ctx(ctx)`) to automatically propagate request correlation attributes across function calls.
- **Strictly Structured Fields:** NEVER write unstructured log strings or use `fmt.Sprintf` inside log messages. Always attach typed context fields:
  ```go
  log.Info().Str("user_id", userID).Int("items_count", count).Msg("processed item batch")
  ```
- **Public Function Logging Pattern (Start/Done):**
  - **On Enter (Debug):** Every exported/public function or service entry point MUST emit a `Debug` log level event upon invocation, logging relevant input identifiers ("starting <operation>").
  - **On Exit (Info/Error):** Upon completion, the function MUST emit an `Info` log level event ("completed <operation>") including execution duration (`time.Since(start)`). If an error occurs, emit an `Error` level log event with error context.
  - **Example Pattern:**
    ```go
    func (s *OrderService) CreateOrder(ctx context.Context, cmd CreateOrderCommand) (*Order, error) {
        start := time.Now()
        log := zerolog.Ctx(ctx).With().Str("op", "CreateOrder").Str("customer_id", cmd.CustomerID).Logger()
        log.Debug().Msg("starting order creation")

        order, err := s.domainService.Create(ctx, cmd)
        if err != nil {
            log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to create order")
            return nil, err
        }

        log.Info().Str("order_id", order.ID).Dur("duration_ms", time.Since(start)).Msg("completed order creation")
        return order, nil
    }
    ```
- **Log Levels:**
  - `Debug`: Detailed operational tracking, step-by-step function entrance logs, internal state changes.
  - `Info`: Function completion, high-level business events, application lifecycle milestones.
  - `Warn`: Non-fatal degradation, fallback execution paths, retry attempts.
  - `Error`: Failures requiring attention, unhandled errors, downstream dependency failures.
- **Sanitization & Security:** NEVER log authorization tokens, API keys, passwords, or raw PII payloads.

## 5. Graceful Teardown & OS Signals

All long-running components (HTTP servers, queue consumers, background daemons) MUST implement graceful shutdown:

- Listen for `os.Interrupt`, `syscall.SIGINT`, and `syscall.SIGTERM`.
- Stop accepting new incoming HTTP/RPC requests.
- Provide a configurable shutdown timeout context (e.g., 10 seconds) to flush in-flight operations before terminating.

## 6. Clean Code & SOLID Principles for Maintainability and Readability

All code must adhere to clean code practices and SOLID design principles to ensure long-term maintainability, testability, and readability across the monorepo:

### A. SOLID Design Principles (Go-Idiomatic)
- **Single Responsibility Principle (SRP):** Every package, struct, and function must have one clearly defined responsibility. Avoid monolithic "God" structs or HTTP handlers mixed with complex business calculations.
- **Open/Closed Principle (OCP):** Design systems to be open for extension but closed for modification. Use Go interface composition, functional options, or strategy patterns to add behavior without mutating existing tested logic.
- **Liskov Substitution Principle (LSP):** Implementation structs satisfying an interface must fulfill the complete behavioral contract expected by consumers. Implementations should never introduce unexpected side-effects, panic, or bypass documented interface semantics.
- **Interface Segregation Principle (ISP):** Keep interfaces small, focused, and client-defined ("accept interfaces, return structs"). Prefer small, single-purpose interfaces (e.g., 1–3 methods) scoped to consumer requirements in `application/ports/` rather than large, bloated interfaces.
- **Dependency Inversion Principle (DIP):** High-level application services must depend exclusively on abstractions (port interfaces), never on low-level concrete adapters (e.g., direct database clients or external SDKs).

### B. Readability & Function Granularity
- **Small, Granular Functions:** Functions should do one thing well. Keep function length short and focused on a single level of abstraction.
- **Left-Aligned Happy Path (Guard Clauses):** Handle errors and edge cases early using guard clauses and early returns. Keep the primary success logic unindented along the left margin to maximize visual clarity.
- **Explicit Domain Naming:** Choose precise, self-documenting names reflecting domain context. Avoid generic naming conventions (`data`, `mgr`, `process`, `doStuff`, `info`) and cryptic abbreviations.

### C. Maintainability & Code Quality
- **Explicit Error Handling:** Never suppress, swallow, or ignore errors using blank identifiers (`_`). Explicitly handle, wrap, or translate errors with domain context.
- **Avoid Global State & Side-Effects:** Do not use package-level global variables or implicit side-effects in `init()` functions. State must be explicitly constructed and passed via dependency injection.
- **Pragmatic DRY (Don't Repeat Yourself):** Eliminate duplication of core domain logic and complex calculations. However, avoid premature abstraction or tight coupling for simple, structural code similarities across distinct boundary layers.

