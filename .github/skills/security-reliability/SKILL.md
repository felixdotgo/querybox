---
name: security-reliability
description: Guide for integrating security and reliability checks into implementation and review. Use this when changes touch trust boundaries, data handling, or operational stability.
---

# Security Reliability Skill

Use this skill to reduce security and runtime risks through structured threat and failure analysis.

## When to Activate
- External input handling changes.
- AuthN/AuthZ, secrets, or sensitive data are involved.
- High-impact operational paths are modified.

## Required Inputs
- Data flow and trust boundaries.
- Access control rules and threat assumptions.
- Availability/reliability expectations.

## Method Foundation
- STRIDE-inspired Threat Modeling for security risk enumeration.
- Trust Boundary Analysis for input/control segregation.
- FMEA-style Failure Mode Analysis for reliability controls.

## Scientific Basis
- Structured threat modeling increases coverage versus ad-hoc brainstorming.
- Defense-in-depth and least privilege reduce exploit impact and blast radius.
- Designing for partial failure (timeouts, retries, idempotency) improves resilience.

## Procedure
1. Identify assets, entry points, and trust boundaries.
2. Enumerate plausible threats and failure modes.
3. Validate controls: input validation, authorization, safe defaults.
4. Add safeguards: timeouts, retries, circuit-breaking, idempotency as needed.
5. Ensure observability: logs/metrics/traces/alerts for new risks.
6. Verify with targeted security and reliability checks.

## Decision Rules
- Prefer least privilege and explicit authorization.
- Prefer fail-safe behavior over silent corruption.
- Prefer measurable mitigations with monitoring coverage.

## Anti-Patterns
- Logging secrets or sensitive payloads.
- Assuming trusted input without validation.
- Ignoring partial failure and retry behavior.

## Output Template (Minimal by Default)
- Report only risk items that were analyzed or mitigated.
- Default order: Risk Surface -> Mitigation -> Verification -> Monitoring Notes.
- Avoid generic threat lists when no evidence or change exists.
