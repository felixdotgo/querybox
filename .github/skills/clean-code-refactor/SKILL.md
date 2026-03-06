---
name: clean-code-refactor
description: Guide for safe, incremental refactoring that improves readability and maintainability without changing intended behavior. Use this for technical debt reduction and code quality improvements.
---

# Clean Code Refactor Skill

Use this skill to perform behavior-preserving refactors with clear intent and low risk.

## When to Activate
- Code is hard to read, duplicate, overly coupled, or brittle.
- User requests cleanup, simplification, or maintainability improvement.
- New feature work exposes structural debt.

## Required Inputs
- Current implementation and pain points.
- Existing tests and expected behavior contracts.
- Scope boundaries and non-goals.

## Method Foundation
- Behavior-Preserving Refactoring (Fowler): small, semantics-preserving transformations.
- Boy Scout Rule: leave code cleaner than found, in bounded scope.
- Working Agreements: single-change intent, reversible steps, contract stability.

## Scientific Basis
- Complexity and defect risk correlate with larger, mixed-purpose changes; incremental refactors reduce change risk.
- Fast feedback loops (run-near tests after each step) improve defect detection latency.
- Cognitive load research supports simpler control flow and clearer naming for maintainability.

## Procedure
1. Define invariants (what must not change).
2. Identify highest-value refactor target.
3. Apply one focused refactor step at a time.
4. Keep public contracts stable unless explicitly approved.
5. Re-run targeted tests after each logical step.
6. Document trade-offs and remaining debt.

## Refactor Priorities
- Clarify naming and control flow.
- Reduce duplication with proven abstractions.
- Isolate side effects from domain logic.
- Simplify interfaces and boundaries.

## Decision Rules
- Prefer clarity over cleverness.
- Prefer incremental, reversible changes.
- Stop when readability and maintainability goals are met.

## Anti-Patterns
- Large rewrites with mixed concerns.
- Changing behavior under “refactor” label.
- Introducing abstractions without repeated use cases.

## Output Template (Minimal by Default)
- Report only sections with concrete evidence for this task.
- Default order: Goal -> Changes -> Verification -> Residual Risk.
- Do not restate SDLC/completion checklists already covered by core protocol.
