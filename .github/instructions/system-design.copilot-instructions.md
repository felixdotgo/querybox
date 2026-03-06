---
applyTo: "**/*.md"
description: "System Design Overlay - Principal Architect Protocol"
---

# System Design Overlay (Principal Architect)

## 1) Scope
This file applies to:
- system design documents
- architecture decision records (ADR)
- technical design specs and RFCs
- architecture review outputs

Allowed output formats:
- Markdown
- Mermaid
- PlantUML

Disallowed output:
- application source code
- pseudo-code that behaves like implementation detail

Cross-cutting behavior and SDLC governance are defined in [core.copilot-instructions.md](core.copilot-instructions.md).

## 2) Domain Language Rules
- Reuse existing domain terms from source documents whenever available.
- Keep naming consistent through the document.
- If new terminology is required, define it once and use it consistently.

## 3) Structure Rules
- If an existing document structure exists, follow it.
- If no structure exists, use this default:
  1. Problem Statement
  2. Context and Constraints
  3. Functional Requirements
  4. Non-Functional Requirements
  5. Architecture Overview
  6. Components and Responsibilities
  7. Data Model and Data Lifecycle
  8. Key Flows
  9. Failure Modes and Recovery
  10. Security and Compliance
  11. Capacity and Scaling
  12. Delivery Plan and Milestones
  13. Rollout and Rollback Strategy
  14. Observability and Operations
  15. Trade-offs and Alternatives
  16. Open Questions and Assumptions

## 4) Decision Quality Rules
For each major decision, state:
- why this decision exists
- what is gained
- what is traded off
- why it fits current constraints

Avoid absolute claims without assumptions and rationale.

## 5) Diagram Rules
- One diagram per concern.
- Diagram names and component labels must match domain language.
- Keep diagrams functional, not decorative.

## 6) Anti-Hallucination Rules
- Do not invent business rules, scale numbers, or infrastructure constraints.
- If data is missing, record as assumption or open question.
- Do not claim guarantees without verification path.

## 7) SDLC Handoff Requirements (Mandatory)
Every design artifact should include design-specific handoff details:
- implementation readiness criteria
- verification strategy framing (functional, reliability, security)
- architecture-level rollout and rollback intent
- operations intent (SLI/SLO direction, logs, metrics, traces, alerts)
- ownership and post-release review points

Use core protocol as the canonical source for generic completion checklist and token-efficiency limits.

## 8) Completion Criteria
A system design task is complete only when:
- structure is consistent
- assumptions/open questions are explicit
- trade-offs are documented
- at least one architecture diagram is provided when architecture is in scope
- delivery and operational handoff sections are included
