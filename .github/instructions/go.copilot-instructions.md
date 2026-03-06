---
applyTo: "**/*.go"
description: "Go Overlay - Principal Architect Protocol"
---

# Go Overlay (Principal Architect)

## 1) Scope
- This file provides Go-specific implementation standards.
- Cross-cutting behavior, SDLC workflow, and completion rules are defined in [core.copilot-instructions.md](core.copilot-instructions.md).

## 2) Go Architecture and Design Rules
- Follow existing project structure and package boundaries.
- Keep interfaces consumer-defined; avoid producer-side interface inflation.
- Prefer composition over inheritance-like abstractions.
- Avoid premature abstraction and speculative framework layers.

## 3) Implementation Standards
- Always pass `context.Context` explicitly through call chains.
- Handle errors immediately and wrap with context: `fmt.Errorf("<operation>: %w", err)`.
- Do not use `panic` in normal application flow.
- Avoid mutable global state.
- Keep goroutine ownership explicit.
- Prefer directional channels (`chan<-`, `<-chan`) where possible.

## 4) Quality and Safety Rules
- Validate boundary inputs and fail fast with clear error context.
- Avoid hidden side effects in exported functions.
- Keep domain logic free from infrastructure logging side effects.
- Use standard library first; add third-party dependencies only when justified.

## 5) Testing Rules
- Prefer table-driven tests.
- Assert behavior and outputs, not internal implementation details.
- Mock only at external boundaries.
- Cover success path, failure path, and critical edge cases (`nil`, empty values, timeout/cancel).

## 6) Security and Compliance Additions
- Treat all external input as untrusted.
- Enforce authorization and ownership checks at service/handler boundaries.
- Do not log secrets, tokens, or sensitive payloads.
- Raise compliance unknowns as explicit assumptions instead of guessing.

## 7) Release, Operations, and Completion Routing
- Use core protocol as the single source for release, operations, and completion artifacts.

## 8) Forbidden Go Patterns
- `interface{}` / `any` without clear justification at boundary points.
- Swallowing errors or returning unwrapped opaque errors.
- Shared mutable state across goroutines without synchronization strategy.
- Domain logic coupled directly to transport/database concerns.

## 9) Output Rule
- Do not duplicate completion checklist here; follow core completion contract.
