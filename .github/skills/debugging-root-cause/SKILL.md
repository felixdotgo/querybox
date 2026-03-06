---
name: debugging-root-cause
description: Guide for isolating root causes of defects with fast, evidence-based debugging. Use this when fixing bugs, regressions, flaky behavior, or production incidents.
---

# Debugging Root Cause Skill

Use this skill to drive bug investigation from symptom to verified root cause before implementing fixes.

## When to Activate
- User reports a bug, regression, crash, or inconsistent behavior.
- Tests fail and failure reason is unclear.
- Errors occur in production-like environments.

## Required Inputs
- Repro steps and observed behavior.
- Expected behavior.
- Recent changes, logs, and failing tests (if available).

## Method Foundation
- Scientific Method: observe, hypothesize, experiment, conclude.
- Hypothesis-Driven Debugging: isolate one variable per experiment.
- 5 Whys / Causal Chain: distinguish proximate symptom from root mechanism.

## Scientific Basis
- Controlled experiments with one-variable changes improve causal attribution.
- Reliable reproduction is required for valid diagnosis and prevents confirmation bias.
- Post-fix regression checks reduce recurrence by validating adjacent behaviors.

## Procedure
1. Reproduce reliably with the smallest possible case.
2. Capture evidence (error output, stack trace, inputs, environment).
3. Localize failure boundary (module, function, dependency, data path).
4. Build and test hypotheses one by one.
5. Identify the root cause and contributing factors.
6. Implement minimal fix at source of defect.
7. Verify with targeted tests, then broader regression checks.

## Decision Rules
- Prefer facts from repro/logs over assumptions.
- Change one variable at a time during investigation.
- Fix the root mechanism, not only downstream symptoms.

## Anti-Patterns
- Patching without reproducible evidence.
- Mixing multiple speculative fixes in one step.
- Declaring success without regression verification.

## Output Template (Minimal by Default)
- Report only sections with direct evidence from repro/logs/tests.
- Default order: Symptom -> Root Cause -> Fix -> Verification -> Residual Risk.
- Omit empty sections; do not pad with generic text.
