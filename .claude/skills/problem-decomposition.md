---
name: problem-decomposition
description: Break complex tasks into execution-ready slices. Activate for broad/ambiguous requests spanning multiple files, components, or lifecycle stages.
---

## Procedure
1. **Normalize** — rewrite to one clear objective; extract hard constraints; make acceptance criteria testable
2. **Map surface** — identify touched modules, interfaces, data contracts, docs; mark coupling risks
3. **Slice** — thin vertical slices with user-visible progress; sequence by dependency + risk; each independently verifiable
4. **Contract per slice** — change target, expected behavior change, acceptance checks, rollback notes
5. **Verification** — narrow checks first → broader; record untestable areas as residual risk
6. **Assumptions** — state only required assumptions; flag blockers affecting architecture/security/data integrity

## Decision Rules
- Smallest viable slice over broad refactor
- Explicit contracts over inferred behavior
- Reversible changes for risky areas
- User-value-first when trade-offs are equal

## Anti-patterns
- Giant single-step plans without checkpoints
- Repeating request without transformation
- Hidden assumptions about APIs, schemas, environments
- Validation deferred to end without per-slice checks

**Output**: Objective → Ordered Slices → Verification Plan → Assumptions/Risks
