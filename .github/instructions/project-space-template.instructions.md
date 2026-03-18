---
applyTo: "{{PROJECT_FILE_GLOB}}"
description: "Project-Specific Custom Space Template - Fill placeholders for each project"
---

# Project Custom Space Template

## 1) Project Identity
- Project Name: {{PROJECT_NAME}}
- Domain: {{DOMAIN_NAME}}
- Primary Runtime/Stack: {{STACK}}
- Deployment Environment: {{ENVIRONMENTS}}
- Criticality Level: {{CRITICALITY_LEVEL}}

## 2) Product and Delivery Goals
- Business Goal: {{BUSINESS_GOAL}}
- User Outcomes: {{USER_OUTCOMES}}
- Non-Goals: {{NON_GOALS}}
- Definition of Done: {{DEFINITION_OF_DONE}}

## 3) Architecture, Security, and Constraints
- Architectural Style: {{ARCH_STYLE}}
- Hard Constraints: {{HARD_CONSTRAINTS}}
- Performance Targets: {{PERFORMANCE_TARGETS}}
- Availability Targets: {{AVAILABILITY_TARGETS}}
- Data Residency/Compliance: {{DATA_COMPLIANCE}}
- Threat model baseline: {{THREAT_MODEL_BASELINE}}
- AuthN/AuthZ requirements: {{AUTH_REQUIREMENTS}}
- Secrets policy: {{SECRETS_POLICY}}

## 4) Coding Rules
- Naming Rules: {{NAMING_RULES}}
- Module Boundaries: {{MODULE_BOUNDARIES}}
- Allowed Dependencies: {{ALLOWED_DEPENDENCIES}}
- Forbidden Patterns: {{FORBIDDEN_PATTERNS}}

## 5) Programming Principles Priority
1. {{PRINCIPLE_1}}
2. {{PRINCIPLE_2}}
3. {{PRINCIPLE_3}}

## 6) SDLC Overrides (Project-Specific)

### Implementation
- Branching strategy: {{BRANCHING_STRATEGY}}
- Change size policy: {{CHANGE_SIZE_POLICY}}

### Verification
- Mandatory checks: {{MANDATORY_CHECKS}}
- Test depth by risk tier: {{TEST_POLICY_BY_RISK}}

### Release
- Rollout strategy: {{ROLLOUT_STRATEGY}}
- Rollback criteria: {{ROLLBACK_CRITERIA}}

### Operations
- Required observability: {{OBSERVABILITY_REQUIREMENTS}}
- Runbook ownership: {{RUNBOOK_OWNERSHIP}}

## 7) Repository Conventions
- Directory ownership map: {{OWNERSHIP_MAP}}
- Public API contract locations: {{API_CONTRACT_LOCATIONS}}
- Migration policy: {{MIGRATION_POLICY}}
- Feature flag policy: {{FEATURE_FLAG_POLICY}}

## 8) Agent Behavior Overrides
- Response language override: {{RESPONSE_LANGUAGE_OVERRIDE}}
- Explanation verbosity: {{VERBOSITY_LEVEL}}
- Clarification threshold: {{CLARIFICATION_THRESHOLD}}
