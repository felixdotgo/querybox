---
description: "Go Overlay — Claude Instructions. Apply when working on Go files."
---

Extends core protocol (`CLAUDE.md`). Go-specific rules only; shared behavior in core.

## Architecture
- Follow existing project structure and package boundaries
- Consumer-defined interfaces; avoid producer-side inflation
- Prefer composition; avoid premature abstraction and speculative frameworks

## Implementation
- Pass `context.Context` explicitly through all call chains
- Wrap errors with context: `fmt.Errorf("<operation>: %w", err)`
- No `panic` in normal application flow
- No mutable global state
- Goroutine ownership explicit; prefer directional channels (`chan<-`, `<-chan`)

## Testing
- Table-driven tests; assert behavior not implementation
- Mock only at external boundaries
- Cover: success, failure, edge (nil, empty, timeout/cancel)

## Forbidden
- `interface{}` / `any` without clear boundary justification
- Swallowed or unwrapped errors
- Shared mutable state without synchronization
- Domain logic coupled to transport/database
