---
name: security-reliability
description: Integrate security and reliability checks into implementation and review. Activate when changes touch trust boundaries, data handling, or operational stability.
---

## Procedure
1. Identify assets, entry points, and trust boundaries
2. Enumerate plausible threats and failure modes
3. Validate controls: input validation, authorization, safe defaults
4. Add safeguards: timeouts, retries, circuit-breaking, idempotency as needed
5. Ensure observability for new risks
6. Verify with targeted security and reliability checks

## Decision Rules
- Least privilege and explicit authorization
- Fail-safe over silent corruption
- Measurable mitigations with monitoring coverage

## Anti-patterns
- Logging secrets or sensitive payloads
- Assuming trusted input without validation
- Ignoring partial failure and retry behavior

**Output**: Risk Surface → Mitigation → Verification → Monitoring Notes
