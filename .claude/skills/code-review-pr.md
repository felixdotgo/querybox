---
name: code-review-pr
description: "Structured code review for pull requests and merge requests. Activate when reviewing changes, diffs, or proposed code."
---

## When to Activate
- User asks to review a PR, MR, or code diff
- User asks to critique or QA code changes
- Operating in Review mode on implementation output

## Procedure

### 1. Understand Scope
1. Read PR description / user's stated intent
2. Identify all changed files and change type (new, modified, deleted)
3. Determine risk tier: low (docs, config) / medium (business logic) / high (auth, data, infra)

### 2. Review Pass — Per File
For each changed file, check:
- **Correctness**: Does it do what it claims? Edge cases handled?
- **Security**: Input validation? Auth checks? Secret exposure?
- **Naming**: Domain-aligned? Consistent with codebase conventions?
- **Contracts**: Public API changes? Breaking changes? Backward compatibility?
- **Error handling**: All error paths covered? Context in error messages?
- **Tests**: Are changes tested? Test quality adequate for risk tier?
- **Performance**: Obvious N+1? Unbounded operations? Missing pagination?

### 3. Cross-cutting Concerns
- Consistency across changed files (naming, patterns, error handling)
- Migration/schema compatibility
- Documentation updated if behavior changed
- Feature flags if partial rollout needed

### 4. Report Findings
For each finding, provide:
- **File + line**: exact location
- **Severity**: 🔴 blocker / 🟡 warning / 🟢 suggestion / 💭 question
- **Category**: correctness / security / style / performance / maintainability
- **Description**: what's wrong and why it matters
- **Suggestion**: concrete fix (code snippet when helpful)

### 5. Summary
- Overall assessment: approve / request changes / needs discussion
- Key risks and blocking issues
- Positive observations (what's done well)

## Severity Guide
| Level | Meaning | Action |
|-------|---------|--------|
| 🔴 Blocker | Bug, security flaw, data loss risk | Must fix before merge |
| 🟡 Warning | Code smell, missing edge case, weak test | Should fix, discuss if disagree |
| 🟢 Suggestion | Style, readability, minor improvement | Nice to have |
| 💭 Question | Unclear intent, needs explanation | Clarify before deciding |

## Decision Rules
- Start with highest-risk files (auth, data, payments)
- Correctness and security before style
- Suggest concrete fixes, not just "this is wrong"
- Praise good patterns — review is not just about finding faults
- If too many findings (>15), group by theme and prioritize top 5

## Anti-patterns
- Nitpicking style in critical security review
- Reviewing only the diff without understanding the surrounding context
- Vague feedback like "this could be better" without actionable suggestion
- Blocking on subjective style preferences
- Missing the forest for the trees (micro-issues but missing architectural problem)

## Output
Scope summary → Per-file findings (severity + category + suggestion) → Cross-cutting concerns → Overall assessment