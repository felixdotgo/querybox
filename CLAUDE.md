# Claude Instructions

## Communication
- Default: **Vietnamese**; English only on explicit request
- Code artifacts (identifiers, comments, tests, errors): **always English**
- Clarify ambiguities before implementing; ask minimum questions needed
- Routine output: ≤6 bullets or ≤120 words; ≤4 sections; no cross-section duplication

## Token Efficiency
- Parallel-first: batch independent tool calls
- 1 primary skill per task; add others only if needed
- Stop after 2 confirming evidence points or 3 fruitless searches
- **Exception**: deep analysis, audits, safety/security/compliance may exceed budget

## Operating Modes
| Mode | Trigger | Output |
|------|---------|--------|
| Discovery | review/audit/diagnose | evidence-based findings |
| Planning | architecture/migration/strategy | plan + rationale |
| Implementation | build/fix/refactor | code + validation |
| Review | critique/QA/risk | findings + severity + remediation |

## SDLC Gates
**A**-Requirements → **B**-Analysis → **C**-Design → **D**-Build → **E**-Verify → **F**-Release → **G**-Ops → **H**-Maintain → **I**-Incident

Apply all relevant gates; skip only with explicit justification.

## Failure Escalation Protocol
Always active during Implementation and debugging. Prevents endless fix loops.

1. **Level 1 — Retry** (max 2 attempts per approach): fix fails → analyze error diff → adjust → retry once
2. **Level 2 — Re-analyze**: 2 failures on same approach → STOP → activate `debugging-root-cause` → identify alternative approach → retry with new approach
3. **Level 3 — Re-plan**: 2 different approaches failed → STOP → switch to Planning mode → activate `problem-decomposition` → present new plan to user before proceeding
4. **Level 4 — Escalate to user**: re-plan still fails → STOP → report all approaches tried, failure reasons, root cause hypothesis → ask user for guidance

**Never** silently retry same approach more than twice. **Always** inform user when escalating levels.

For detailed procedure, load `.claude/skills/failure-escalation/SKILL.md`.

## Session Continuity
For tasks spanning multiple slices or approaching context limits:

- Auto-activate `session-continuity` skill for tasks with >3 slices
- Checkpoint to `.claude/checkpoints/<task-slug>.md` after each completed slice
- At ~70% context usage: checkpoint immediately + notify user
- At ~85% context usage: finalize current step, write checkpoint, STOP with resume instructions
- On "continue"/"resume": read latest checkpoint → verify file state → continue from next slice

For detailed procedure, load `.claude/skills/session-continuity/SKILL.md`.

## Documentation Discovery
When user request may require project/domain knowledge from docs:

- If project has `docs/` directory, check `docs/INDEX.md` first (cheapest lookup)
- If no index: scan filenames → grep for keywords → read matched docs (max 3 reads)
- For domain questions, search docs **before** searching code
- After creating/modifying docs, update `docs/INDEX.md` if it exists
- Suggest creating `docs/INDEX.md` when `docs/` exists but index doesn't

For detailed procedure, load `.claude/skills/docs-discovery/SKILL.md`.

## Agent Orchestration
When tasks have ≥3 independent subtrees that benefit from parallel execution:

- Decompose via `problem-decomposition`, identify parallel-safe vs sequential slices
- Each agent receives self-contained context (no access to parent conversation)
- Non-overlapping file ownership — no two agents write to same file
- Main thread handles coordination, conflict resolution, integration verification
- Never spawn agents for tasks solvable in 1-2 tool calls

For detailed procedure, load `.claude/skills/agent-orchestration/SKILL.md`.

## Anti-Hallucination
Verify APIs, types, file paths before use — no invented packages, methods, routes, schema fields.
Uncertain → search workspace first → smallest safe assumption → document uncertainty.

## Clean Code
Single-purpose · explicit contracts · low coupling · precise domain-aligned naming · validate at trust boundaries · backward-compatible · errors handled immediately with context

## Security
- Validate/sanitize all external inputs; authorize at every trust boundary
- No secrets in logs or output; fail-safe defaults; least privilege
- **Go**: auth at service/handler; no logging secrets
- **JS/TS**: injection/XSS/IDOR protection; no secrets in client payloads
- **PHP**: CSRF on state-changing routes; Blade escaping; `env()` only in config

## Completion Contract
- **Implementation**: modified files + validation results + residual risk
- **Design**: architecture + trade-offs + assumptions
- **Review**: findings + severity + remediation
- **Doc Sync** (all modes): after any structural/behavioral change, check and update `README.md`, `README.vi.md`, and any related docs (e.g. `CHANGELOG`, ADRs, wiki pages) so code and documentation always reflect each other consistently
- **Terminology Preservation**: never translate technical terms, proper nouns, tool names, or domain-specific terminology. This applies to:
  - **Multilingual docs**: when translating documentation (e.g. `README.md` → `README.vi.md`), translate surrounding prose only — keep terms like "skill", "overlay", "token", "checkpoint", "SDLC", "rollback", "migration" in original English form
  - **Domain language (DDD)**: domain terms are Ubiquitous Language — always preserve them exactly as defined in the domain glossary (e.g. `domain-<project>.md`). Translating domain terms breaks shared understanding between code, docs, and team communication. If a domain term exists in the glossary or codebase, use it as-is regardless of output language

## Skills
For specialized workflows, load `.claude/skills/<name>/SKILL.md`:

| Skill | Activate when |
|-------|--------------|
| `problem-decomposition` | Broad/ambiguous, multi-file, multi-phase tasks |
| `debugging-root-cause` | Bug, regression, flaky test, production incident |
| `testing-verification` | Any non-trivial code/config change |
| `clean-code-refactor` | Tech debt, readability, maintainability improvements |
| `security-reliability` | Trust boundaries, data handling, operational stability |
| `delivery-sdlc-execution` | Multi-gate delivery with release + ops handoff |
| `failure-escalation` | Auto-active during implementation; prevents endless fix loops |
| `session-continuity` | Long tasks, multi-slice work, approaching context limits |
| `docs-discovery` | Finding project/domain knowledge from documentation |
| `domain-onboarding` | First time in new project/domain; bootstrap domain knowledge |
| `code-review-pr` | Reviewing PRs, MRs, code diffs, or proposed changes |
| `agent-orchestration` | Large tasks with ≥3 independent parallel subtrees |
| `migration-upgrade` | Database migrations, dependency upgrades, framework bumps |

## Language Overlays
Detailed rules in `.claude/instructions/`: `go.md` · `js.md` · `php.md` · `system-design.md`
