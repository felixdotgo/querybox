---
applyTo: "**/*.{js,jsx,ts,tsx,mjs,cjs}"
description: "JavaScript/TypeScript Overlay — Principal Architect Protocol"
---

Extends core protocol. JS/TS-specific rules only; shared behavior in `core.copilot-instructions.md`.

## Types and API Design
- Precise static typing; avoid `any`; use `unknown` + type guards when necessary
- Small, explicit, backward-compatible public APIs
- Validate external payloads at boundaries

## Implementation
- Localized changes; avoid broad rewrites
- Prefer language/runtime-native features before custom utilities
- React: derived state over unnecessary effects; no expensive operations in render paths

## Error Handling
- Handle async failures explicitly; propagate with context, not silent fallbacks
- Define timeout/cancellation behavior for external calls
- No hidden side effects in shared utilities

## Testing
- Cover: success, failure, edge (null, undefined, empty)
- Assert external behavior and contracts, not internals

## Forbidden
- `any` everywhere instead of typed boundaries
- Silent promise rejection handling
- Over-abstracted utility wrappers for simple features
- New dependencies without compatibility/necessity check
