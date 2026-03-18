# AI Instructions

## Precedence
Platform policies > User request > Core protocol > Language overlay > Project-specific

Conflict → follow higher precedence; state assumptions.

## Communication
- Default: **Vietnamese**; English only on explicit request
- Code artifacts (identifiers, comments, tests, errors): **always English**
- Clarify ambiguities before implementing; ask minimum questions needed
- **Terminology preservation**: never translate technical terms or domain terms (Ubiquitous Language) — keep original English form in all contexts: docs translation, conversation, code review, domain modeling. Translating domain terms breaks shared understanding (DDD) and causes ambiguity. Translate surrounding prose only.

## Skills
Reusable skill guides in `.claude/skills/`:
- `problem-decomposition` — broad/ambiguous tasks, multi-file, multi-phase
- `debugging-root-cause` — bugs, regressions, flaky tests, incidents
- `testing-verification` — verification after non-trivial changes
- `clean-code-refactor` — tech debt, readability, maintainability
- `security-reliability` — trust boundaries, data handling, stability
- `delivery-sdlc-execution` — multi-gate delivery with release + ops handoff
- `failure-escalation` — auto-escalate from retry → re-plan → user when fixes fail repeatedly
- `session-continuity` — checkpoint progress for long tasks; auto-resume across sessions
- `docs-discovery` — fast documentation lookup; INDEX.md-first strategy
- `domain-onboarding` — bootstrap domain knowledge from codebase for new projects
- `code-review-pr` — structured PR/MR review with severity + actionable suggestions
- `agent-orchestration` — multi-agent parallel execution for large independent tasks
- `migration-upgrade` — safe database migrations, dependency upgrades, framework bumps

## Failure Escalation
- Max 2 fix attempts per approach; then switch approach via `debugging-root-cause`
- After 2 failed approaches: re-plan via `problem-decomposition`
- After re-plan still fails: stop and escalate to user with full report
- For bugs: fix autonomously from logs/errors/tests; don't ask for hand-holding — escalate only at Level 2+
- See `failure-escalation` skill for detailed protocol

## Session Continuity
- Multi-slice tasks: auto-checkpoint to `.claude/checkpoints/` after each slice
- At ~85% context budget: finalize, checkpoint, stop with resume instructions
- On resume: read latest checkpoint, verify file state, continue
- See `session-continuity` skill for detailed protocol

## Documentation Discovery
- If project has `docs/` directory: check `docs/INDEX.md` first
- If no index: suggest creating one after first doc search
- For domain questions: search docs before searching code
- See `docs-discovery` skill for detailed protocol

## Task Management
- Before implementing: write checklist to `tasks/todo.md`; check items off as you go
- After any correction: update `tasks/lessons.md` with a rule preventing that mistake; review at session start

## Completion
- Non-trivial changes: pause before done — ask "is there a more elegant way?" and "would a staff engineer approve this?"
- One focused task per subagent; avoid mixing research and implementation in same agent