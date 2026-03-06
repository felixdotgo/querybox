---
name: problem-decomposition
description: Guide for breaking complex software tasks into execution-ready slices. Use this when requests are broad, ambiguous, or span multiple files/phases.
---

# Problem Decomposition Skill

Scope: Cross-language software engineering tasks.

## Intent
Use this skill to convert broad, ambiguous, or multi-phase engineering requests into an execution-ready breakdown that an AI coding agent can implement safely and incrementally.

## When to Activate
Activate this skill when one or more conditions are true:
- The request spans multiple files, components, or lifecycle stages.
- Requirements include both implementation and quality gates (tests, release, operations).
- Constraints are present (security, compliance, backward compatibility, timeline, scope limits).
- The task is underspecified and assumptions must be made explicitly.

Do not activate for trivial one-step edits unless the user explicitly asks for decomposition.

## Required Inputs
- User goal and expected outcome.
- Current repository context and relevant files.
- Constraints (technical, product, compliance, timeline).
- Validation expectations (tests/checks/build/lint).

## Method Foundation
- Top-Down Functional Decomposition: split goal into independently verifiable slices.
- Divide-and-Conquer Sequencing: order by dependency and risk.
- Goal-Question-Metric (GQM): derive acceptance checks from objective.

## Scientific Basis
- Task chunking reduces cognitive overload and planning errors in complex work.
- Explicit dependency mapping reduces integration surprises and rework.
- Per-slice verification improves early defect discovery versus end-only validation.

## Outputs
Produce:
1. A short objective statement.
2. A bounded scope definition (in-scope / out-of-scope).
3. A dependency and impact map.
4. Ordered implementation slices with acceptance criteria.
5. Verification plan mapped to changed areas.
6. Explicit assumptions, risks, and rollback note.

## Procedure

### 1) Normalize the Request
- Rewrite the task into one clear objective sentence.
- Extract hard constraints and non-goals.
- Convert vague wording into testable acceptance criteria.

### 2) Map Impact Surface
- Identify touched modules, interfaces, data contracts, and docs.
- Mark coupling risks and external dependencies.
- Highlight potential regressions and compatibility concerns.

### 3) Slice by Value and Risk
- Break work into thin vertical slices that each produce user-visible progress.
- Sequence slices by dependency order and risk reduction.
- Keep each slice independently verifiable.

### 4) Define Execution Contract per Slice
For each slice, include:
- change target (files/components)
- expected behavior change
- acceptance checks
- failure/rollback note if applicable

### 5) Attach Verification Strategy
- Start with narrow checks closest to the change.
- Expand to broader checks as confidence grows.
- Record any untestable area as explicit residual risk.

### 6) Finalize Assumptions and Open Questions
- State only assumptions required to proceed.
- Separate assumptions from facts.
- Flag open questions that materially affect architecture/security/data integrity.

## Decision Rules
- Prefer smallest viable slice over broad refactor.
- Prefer explicit contracts over inferred behavior.
- Prefer reversible changes for risky areas.
- Prefer user-value-first ordering when trade-offs are equal.

## Failure Handling
If decomposition is blocked:
1. Identify exact missing artifact (file, schema, requirement, environment).
2. Propose a safe default assumption.
3. Continue with constrained scope and label uncertainty clearly.

## Quality Bar
A decomposition is acceptable only if:
- Every slice has a concrete outcome and validation method.
- Dependencies and ordering are explicit.
- Scope boundaries prevent unrelated changes.
- Risks and assumptions are visible and minimal.

## Anti-Patterns
- Giant single-step plans with no checkpoints.
- Repeating the user request without transformation.
- Hidden assumptions about APIs, schemas, or environments.
- Validation deferred to the end without per-slice checks.

## Lightweight Output Template (Minimal by Default)
- Include only: Objective, Ordered Slices, Verification Plan, Assumptions/Risks.
- Add Scope and Impacted Areas only when they materially change decisions.
- Keep each section concise and avoid repeating unchanged constraints.
