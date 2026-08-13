---
trigger: always_on
description: Mandatory strict Test-Driven Development (TDD) discipline emphasizing Red-Green-Refactor cycle principles.
---

# Strict Test-Driven Development (TDD) Discipline

This rule mandates strict adherence to Test-Driven Development (TDD) across all feature development, bug fixes, and refactoring efforts.

## 1. Core TDD Principles

All code must be developed using a rigorous **Red-Green-Refactor** cycle. Writing implementation code prior to having a failing test is strictly prohibited.

## 2. The Red-Green-Refactor Cycle

### Phase 1: Red (Design API & Write a Failing Test First)
- **Rule:** Before writing any implementation code, write a concise, focused test that specifies what the unit under development should do and how callers will interact with it.
- **Software Design Decisions:** The Red phase is fundamentally a software design exercise—it is not merely about expected outputs, but about declaring how the unit is intended to be used, including interface boundaries, method signatures, dependency contracts, and error conditions from the consumer's perspective.
- **Requirement:** Execute the test and verify that it fails for the expected reason (e.g., missing symbol, failing assertion, or unhandled case). Never move forward without confirming test failure.

### Phase 2: Green (Fastest Path to Pass)
- **Rule:** Implement the simplest, fastest, and most straightforward code necessary to make the failing test pass.
- **Requirement:** Focus exclusively on making the test pass with minimal effort. Do NOT attempt premature optimization, architectural abstractions, or code polish during this phase.

### Phase 3: Refactor (Optimization, Maintainability, & Architecture)
- **Rule:** With all tests passing (green), clean up and optimize the implementation while preserving functional correctness.
- **Focus Areas:**
  - **Optimizations:** Improve runtime performance, algorithm efficiency, memory consumption, and resource utilization.
  - **Maintainability & Clean Code:** Remove code duplication, enhance readability, simplify complex expressions, and improve naming.
  - **Architectural Integrity:** Ensure strict adherence to Hexagonal Architecture, DDD boundaries, and Go clean code standards.
  - **Regression Verification:** Continuously re-run the test suite after every refactoring step to ensure zero behavior breakage.

## 3. Strict Execution Guidelines

- **No Code Without Tests:** Production and business logic code must never be introduced without a failing test driving its implementation.
- **Micro-Iterations:** Keep iteration cycles small and granular (write one test case -> reach green fast -> refactor -> repeat).
- **Test Integrity:** Never disable, alter, or remove failing tests just to pass the build.
