---
name: delivery-sdlc-execution
description: Execute software tasks end-to-end across SDLC gates. Activate for multi-step implementation with quality, release, and operational handoff expectations.
---

## Procedure
1. **Scope** — define objective, acceptance criteria, constraints
2. **Analyze + Design** — map dependencies/risk, choose approach, document trade-offs
3. **Build** — minimal, focused, reversible changes
4. **Verify** — targeted checks first → broader checks
5. **Release + Operate** — rollout/rollback plan, observability, ownership, docs/changelog

## Decision Rules
- Scope aligned to requested outcome only
- Assumptions explicit when unavoidable
- No completion without validation evidence

## Anti-patterns
- Skipping gates for risky changes
- Shipping without rollback plan or monitoring
- Treating docs/ops as optional for impactful changes

**Output**: Scope → Implementation → Verification → Release/Risk
