---
name: debugging-root-cause
description: Isolate root causes with fast, evidence-based debugging. Activate for bugs, regressions, flaky behavior, or production incidents.
---

## Procedure
1. Reproduce with smallest possible case
2. Capture evidence (error output, stack trace, inputs, environment)
3. Localize failure boundary (module, function, dependency, data path)
4. Test hypotheses one-by-one — one variable at a time
5. Identify root cause and contributing factors
6. Implement minimal fix at defect source
7. Verify with targeted tests, then broader regression checks

## Decision Rules
- Facts from repro/logs over assumptions
- Fix root mechanism, not downstream symptoms
- No success without regression verification

## Anti-patterns
- Patching without reproducible evidence
- Mixing multiple speculative fixes in one step
- Declaring success without regression verification

**Output**: Symptom → Root Cause → Fix → Verification → Residual Risk
