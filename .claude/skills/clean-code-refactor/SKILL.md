---
name: clean-code-refactor
description: Safe, incremental refactoring without changing intended behavior. Activate for technical debt reduction, readability, or maintainability improvements.
---

## Procedure
1. Define invariants — what must not change
2. Identify highest-value refactor target
3. Apply one focused step — keep public contracts stable
4. Re-run targeted tests after each logical step
5. Document trade-offs and remaining debt

## Decision Rules
- Clarity over cleverness
- Incremental and reversible over large rewrites
- Stop when readability/maintainability goals are met

## Anti-patterns
- Large rewrites mixing unrelated concerns
- Changing behavior under "refactor" label
- Introducing abstractions without repeated use cases

**Output**: Goal → Changes → Verification → Residual Risk
