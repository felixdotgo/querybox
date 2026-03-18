---
description: "Project-Specific Space Template — Fill placeholders for each project. Copy to project root or .claude/ directory."
---

> Copy this file to your project's `CLAUDE.md` or `.claude/instructions/project.md` and replace all `{{PLACEHOLDER}}` values.

## Project Identity
- **Name**: {{PROJECT_NAME}}
- **Domain**: {{BUSINESS_DOMAIN}}
- **Stack**: {{PRIMARY_STACK}}
- **Environments**: {{DEV | STAGING | PROD}}
- **Criticality**: {{LOW | MEDIUM | HIGH | CRITICAL}}

## Product and Delivery Goals
- **Business Goal**: {{WHAT_THIS_PROJECT_ACHIEVES}}
- **User Outcomes**: {{WHO_BENEFITS_AND_HOW}}
- **Non-Goals**: {{EXPLICITLY_OUT_OF_SCOPE}}
- **Definition of Done**: {{WHAT_DONE_LOOKS_LIKE}}

## Architecture, Security, and Constraints
- **Architectural Style**: {{MONOLITH | MICROSERVICES | SERVERLESS | etc.}}
- **Hard Constraints**: {{PERFORMANCE | COMPLIANCE | BUDGET | etc.}}
- **Performance Targets**: {{LATENCY | THROUGHPUT | AVAILABILITY}}
- **Data Residency/Compliance**: {{GDPR | HIPAA | PCI | NONE}}
- **Threat Model**: {{KEY_ATTACK_VECTORS}}
- **AuthN/AuthZ**: {{MECHANISM_AND_SCOPE}}
- **Secrets Policy**: {{HOW_SECRETS_ARE_MANAGED}}

## Coding Rules
- **Naming Rules**: {{PROJECT_SPECIFIC_NAMING}}
- **Module Boundaries**: {{WHAT_CAN_DEPEND_ON_WHAT}}
- **Allowed Dependencies**: {{APPROVED_LIBRARIES}}
- **Forbidden Patterns**: {{WHAT_IS_BANNED_AND_WHY}}

## Programming Principles Priority
1. {{TOP_PRIORITY — e.g., correctness, security, readability}}
2. {{SECOND_PRIORITY}}
3. {{THIRD_PRIORITY}}

## SDLC Overrides
- **Branching**: {{STRATEGY — e.g., trunk-based, feature branches}}
- **Change size**: {{MAX_PR_SIZE_POLICY}}
- **Mandatory checks**: {{REQUIRED_BEFORE_MERGE}}
- **Test depth by risk**: {{LOW=unit | MED=integration | HIGH=e2e}}
- **Rollout strategy**: {{BLUE_GREEN | CANARY | ROLLING | ALL_AT_ONCE}}
- **Rollback criteria**: {{WHAT_TRIGGERS_ROLLBACK}}
- **Observability**: {{REQUIRED_LOGS_METRICS_TRACES}}
- **Runbook ownership**: {{TEAM_OR_PERSON}}

## Repository Conventions
- **Directory ownership**: {{WHO_OWNS_WHAT}}
- **API contracts**: {{WHERE_SPECS_LIVE}}
- **Migration policy**: {{HOW_DB_MIGRATIONS_ARE_HANDLED}}
- **Feature flags**: {{SYSTEM_AND_POLICY}}

## Agent Behavior Overrides
- **Response language**: {{LANGUAGE}}
- **Verbosity**: {{TERSE | NORMAL | DETAILED}}
- **Clarification threshold**: {{ASK_ALWAYS | ASK_WHEN_AMBIGUOUS | ASSUME_AND_PROCEED}}
