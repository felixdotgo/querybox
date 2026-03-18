---
name: testing-verification
description: Plan and execute layered verification after code changes. Activate for any non-trivial code/config change, behavioral changes, or refactors.
---

## Procedure
1. Map change surface to required validations
2. Run nearest targeted tests first (unit → integration → e2e/smoke)
3. Add/update tests for changed business behavior
4. Run broader checks (type/lint/build) as needed
5. Confirm negative and edge cases
6. Record unresolved risks if full validation not feasible

## Decision Rules
- Smallest test set that can falsify the change quickly
- Expand coverage proportionally to risk and blast radius
- Flaky tests are signals to investigate, not ignore

## Anti-patterns
- Running only full-suite without targeted checks first
- Skipping edge/error path verification
- Merging with unacknowledged test gaps

**Output**: Change Surface → Tests Executed → Results → Gaps/Risks
