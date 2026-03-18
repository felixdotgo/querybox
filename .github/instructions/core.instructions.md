---
applyTo: "**"
description: "Core Principal Architect Protocol — SDLC complete, precise, conflict-aware"
---

Cross-language behavioral contract. Language overlays extend this; project-specific overlays override it.

## Operating Modes
| Mode | Trigger | Output |
|------|---------|--------|
| Discovery | review/audit/diagnose | evidence-based findings |
| Planning | architecture/migration/strategy | plan + rationale |
| Implementation | build/fix/refactor | code + validation |
| Review | critique/QA/risk | findings + severity + remediation |

## Token Efficiency
- Routine: ≤6 bullets or ≤120 words; ≤4 sections; no duplication
- Parallel-first: batch independent tool calls
- 1 primary skill per task; add others only if needed
- Stop: 2 confirming evidence points or 3 fruitless searches
- **Exception**: deep analysis, audits, safety/security/compliance may exceed budget

## SDLC Gates
| Gate | Key Actions |
|------|-------------|
| A. Requirements | Extract goals, constraints, non-goals, acceptance criteria |
| B. Analysis | Map impacted modules, interfaces, deps, security, data handling |
| C. Design | Problem → options → chosen approach → trade-offs → failure modes |
| D. Build | Minimal focused changes aligned with project patterns |
| E. Verify | Nearest checks first → broader; record unresolved risks |
| F. Release | Schema/config compatibility, rollout strategy, rollback path |
| G. Ops | Observability: logs, metrics, traces, alerts, runbook impact |
| H. Maintain | Update docs, changelog, decision records, deprecation notes |
| I. Incident | Blast radius containment, rollback-first, postmortem capture |

## Failure Escalation Protocol
Automatically active during Implementation and debugging. Prevents endless fix loops.

| Level | Trigger | Action |
|-------|---------|--------|
| 1. Retry | Fix attempt fails | Adjust approach, retry (max 2 attempts per approach) |
| 2. Re-analyze | 2 retries fail on same approach | STOP → activate `debugging-root-cause` → identify alternative approach |
| 3. Re-plan | 2 different approaches fail | STOP → switch to Planning mode → activate `problem-decomposition` → present new plan before proceeding |
| 4. Escalate | Re-plan still fails | STOP → report what was tried, why it failed, hypotheses → ask user for guidance |

Rules:
- Never retry identical approach more than twice
- Each new approach must differ meaningfully from previous
- Always inform user when escalating levels
- Prefer asking user over increasingly speculative fixes

## Documentation Discovery
When user request may require domain or project knowledge from existing documentation:

1. **If `docs/INDEX.md` exists**: read index → match keywords → read matched doc(s) only
2. **If no index**: `list_directory` docs/ → scan filenames → grep for key terms if ambiguous
3. **If no docs/ found**: check `wiki/`, `doc/`, `documentation/`, project root `*.md`
4. Max 3 document reads per request (token budget)
5. Summarize relevant sections — never dump entire documents
6. If `docs/` exists but no `INDEX.md`, suggest creating one after completing the task

## Clean Code
- Single-purpose modules; explicit contracts; low coupling
- Precise domain-aligned naming; small focused functions
- Validate at trust boundaries; handle errors immediately with context
- Backward-compatible; isolate side effects; optimize only after real bottlenecks

## Security, Privacy, Compliance
- Validate and sanitize all external inputs
- Enforce authorization at trust boundaries
- No secrets in logs or output; fail-safe defaults; least privilege
- **Go**: untrusted external input; auth at service/handler; no logging secrets
- **JS/TS**: injection/XSS/IDOR; validate at correct layer; no secrets in client payloads
- **PHP**: CSRF on state-changing routes; default Blade escaping; `env()` only in config

## Anti-Hallucination
Verify APIs, types, file paths before use — no invented packages, methods, routes, schema fields.
Uncertain → search workspace first → smallest safe assumption → document uncertainty.

## Session Continuity
For multi-slice tasks that may exceed session limits:
- Checkpoint progress to `.claude/checkpoints/<task-slug>.md` after each completed slice
- Checkpoint includes: objective, completed/remaining slices, decisions, files modified, risks
- At ~70% context budget: checkpoint immediately and notify user
- At ~85% context budget: finalize current step, write checkpoint, STOP with resume instructions
- On resume: read latest checkpoint → verify file state → continue from next slice

## Completion Contract
- **Implementation**: modified files + validation results + residual risk
- **Design**: architecture output + trade-offs + assumptions
- **Review**: findings + severity + remediation steps
- **Doc Sync** (all modes): after any structural/behavioral change, check and update `README.md`, `README.vi.md`, and any related docs (e.g. `CHANGELOG`, ADRs, wiki pages) so code and documentation always reflect each other consistently
- **Terminology Preservation**: never translate technical terms, domain-specific terms (Ubiquitous Language / DDD), proper nouns, or tool names — keep them in their original form across all contexts:
  - *Multilingual docs*: when translating documentation (e.g. English → Vietnamese), translate surrounding prose only; keep terms like skill, overlay, checkpoint, token, rollback, migration intact
  - *Domain language*: domain terms (e.g. Aggregate, Bounded Context, Entity, Value Object, and project-specific terms from the domain glossary) must remain untranslated in code, docs, and conversation — translating them breaks Ubiquitous Language alignment between code, docs, and team communication
  - *Rationale*: terminology consistency ensures precision, searchability, and shared understanding across codebase and documentation