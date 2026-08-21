---
trigger: always_on
description: Hexagonal design pattern, DDD layering boundaries, and Go concurrency/reactive programming paradigms.
---

# Code Architecture Guidelines

This rule enforces structural alignment with Hexagonal (Ports & Adapters) architecture, Domain-Driven Design (DDD) boundaries, and Go reactive concurrency standards.

## 1. Hexagonal Architecture with DDD

All components must adhere to a strict clean-architecture separation:

### A. Layering and Import Constraints
- **Domain Layer (`domain/`):** Contains core business aggregates, entities, value objects, domain errors, state machines, events, and stateless domain services.
  - **CRITICAL CONSTRAINT:** The domain layer MUST NOT import any packages from the `application` layer or the `adapters` layer. It must remain pure business logic.
- **Application Layer (`application/`):** Orchestrates domain objects to execute specific use cases.
  - **CRITICAL CONSTRAINT:** The application layer MUST NOT import any packages from the `adapters` layer. It has zero knowledge of database technologies, NATS messaging, or external HTTP clients.
- **Adapters Layer (`adapters/`):** Implements infrastructure bindings.
  - Adapters may import both domain and application layers, but domain and application layers must never import concrete adapters.

### B. Ports and Package Structure
Structure every component context under `internal/<component>/` using the following exact directory hierarchy:
```text
internal/<component>/
├── domain/                  # Core Business Domain Layer (pure)
│   ├── model/               # Aggregates, Entities, Value Objects, Domain Errors, State Machines
│   ├── event/               # Domain Events
│   └── service/             # Domain Services (cross-aggregate logic)
├── application/             # Use Case Orchestration Layer
│   ├── ports/
│   │   ├── inbound/         # driving interfaces (consumed by HTTP handlers, NATS subscribers)
│   │   └── outbound/        # driven interfaces (repositories, Redis caches, pub-sub adapters)
│   └── service/             # use case implementations (depends ONLY on outbound port interfaces)
└── adapters/                # Infrastructure & I/O Adapters Layer
    ├── inbound/             # Driving Adapters (Gin HTTP handlers, NATS subscribers)
    └── outbound/            # Driven Adapters (pgx databases, OpenAI, Redis, Qdrant, NATS clients)
```

### C. Dependency Injection & Interface Verification
- Always define dependencies as interfaces inside `application/ports/`.
- Concrete adapters in `adapters/` must implement these interfaces.
- Concrete structs implementing interfaces MUST include static, compile-time type assertions (`var _ Interface = (*Concrete)(nil)`).
- Application service constructors must accept outbound or inbound port *interfaces*, never concrete adapters.

### D. Data Mapping & DTO Boundaries
- **Inbound DTOs:** Driving adapter request/response structures (e.g., Gin JSON binding structs, NATS message payloads) MUST reside strictly in `adapters/inbound/`.
- **Outbound DTOs:** Driven adapter persistence/IO models (e.g., pgx database table structs, Redis hash models, Qdrant vectors) MUST reside strictly in `adapters/outbound/`.
- **Domain Purity:** Application services and domain entities operate EXCLUSIVELY on domain models defined in `domain/model/`.
- **Explicit Conversion:** Inbound and outbound adapters are responsible for mapping between external DTOs and domain models. Never pass framework tags (e.g., `json:"..."`, `db:"..."`, `binding:"..."`) into the domain or application layers.

### E. Error Handling & Layer Translation
- **Domain Errors:** Define core business errors (e.g., `ErrNotFound`, `ErrInvalidState`, `ErrConflict`) inside `domain/model/`.
- **Outbound Error Translation:** Driven adapters MUST map driver/infrastructure errors (e.g., `pgx.ErrNoRows`, `redis.Nil`) into corresponding domain errors before returning them to application services.
- **Inbound Error Translation:** Driving adapters (e.g., HTTP handlers) MUST inspect domain errors and convert them into protocol-specific responses (e.g., Gin `c.JSON(http.StatusNotFound, ...)`). Never expose raw database or internal adapter errors directly to HTTP clients.

### F. Pragmatism vs. Over-Engineering
- **CQRS / Read Paths:** Simple, read-only queries or listing endpoints DO NOT require heavy domain aggregate reconstruction or multi-stage mappers. Application services may query read-optimized port interfaces returning lightweight read models.
- **Proportional Abstraction:** Do not construct full DDD aggregates for simple single-entity CRUD operations. Enforce strict DDD aggregates and domain services only where rich business invariants, state transitions, or multi-entity rules exist.

## 2. Reactive Programming & Concurrency

Prioritize highly concurrent, reactive workflows over traditional sequential/imperative code while ensuring strict resource limits:

- **Go Concurrency Primitives:** Leverage Go Goroutines, Channels, and `select` blocks to build responsive, non-blocking pipelines.
- **Asynchronous Execution:** Heavy computation, external API network calls, and complex I/O pipelines must not block execution synchronously. Wrap them in goroutines and coordinate via channels.
- **Event-Driven Coupling:** Favor decoupling components by emitting events to asynchronous channels.
- **Context Management:** Standard Go `context.Context` must be actively propagated across all concurrent and asynchronous goroutine boundaries. Always manage context cancellations and timeouts to prevent goroutine leaks.
- **Resource Limits & Worker Pools:** NEVER spawn unbounded goroutines (`go func()`) for incoming request loops or unbound queues. Use worker pools, bounded channels, or semaphores (e.g., `golang.org/x/sync/errgroup`) to throttle concurrent execution.
- **Graceful Teardown:** All concurrent background loops MUST check `ctx.Done()` or stop channels and utilize `sync.WaitGroup` or `errgroup.Group` to guarantee clean shutdown without dropping in-flight tasks.
