---
trigger: always_on
description: Go conventions, static compile-time interface implementation assertions, and type safety standards.
---

# Go Language Guidelines & Static Interface Verification

This rule enforces idiomatic Go conventions, strict compile-time interface implementation checks, and type safety standards across the codebase.

## 1. Static Compile-Time Interface Assertions

In Go, interface satisfaction is implicit. When concrete structs are intended to implement an interface (e.g., adapters implementing inbound/outbound ports, pacing engines, metric collectors/registries/aggregators, loggers, context wrappers), breaking changes to method signatures or missed methods might only be caught when instantiated or assigned at call sites.

To guarantee interface compatibility and receive instant compiler feedback at declaration time:

### A. Mandatory Type Assertion Check Syntax
Every concrete struct that is designed to satisfy an interface **MUST** include an explicit compile-time type assertion using the blank identifier (`_`):

- **Pointer Receivers:**
  ```go
  var _ Interface = (*ConcreteType)(nil)
  ```
- **Value Receivers:**
  ```go
  var _ Interface = ConcreteType{}
  ```

### B. Multiple Interfaces
If a concrete struct implements multiple interfaces, declare compile-time checks for all relevant interfaces in a grouped `var` block:
```go
var (
    _ InterfaceA = (*ConcreteType)(nil)
    _ InterfaceB = (*ConcreteType)(nil)
)
```

### C. Placement Guidelines
- Place compile-time checks at the bottom of the source file where the concrete struct and its methods are declared, or immediately beneath the struct and constructor definition.
- If an adapter or concrete type implements an interface defined in a separate package, include the check in the package where the concrete type is defined.

## 2. General Go Type Safety & Clean Interfaces

- **Accept Interfaces, Return Structs:** Functions and constructors should accept interface parameters to remain decoupled and testable, while returning concrete structs when feasible.
- **Minimal Interfaces:** Keep interface definitions small, focused, and scoped strictly to consumer requirements (ISP).
