---
name: domain-onboarding
description: "Bootstrap domain knowledge from existing codebase and docs. Activate when entering a new domain project to rapidly build domain context."
---

## When to Activate
- First time working in a new project/domain
- User asks to "learn the codebase" or "understand the domain"
- Domain-specific terms appear that are unfamiliar
- Setting up project-specific instructions for the first time

## Procedure

### Phase 1: Rapid Scan (broad, shallow)
1. Read project README and any docs/INDEX.md
2. `list_directory` on root, `src/`, `app/`, `lib/`, `internal/` (whichever exist)
3. Identify tech stack from config files (package.json, go.mod, composer.json, etc.)
4. Note directory structure patterns (DDD, MVC, Clean Architecture, etc.)

### Phase 2: Domain Entity Discovery
1. Scan model/entity layer:
   - `grep` for class/struct/type definitions in model/entity directories
   - Note entity names, key fields, relationships
2. Scan database migrations or schema files for data model
3. Scan API routes/controllers for domain operations
4. Scan test file names for business workflow hints

### Phase 3: Business Rule Extraction
1. Look for validation logic (validators, form requests, middleware)
2. Scan for domain constants, enums, status machines
3. Read key service/use-case files for business workflows
4. Note authorization rules and access patterns

### Phase 4: Domain Skill Generation
Generate `.claude/skills/domain-<project-name>.md`:

```
---
name: domain-<project-name>
description: "Domain knowledge for <Project Name>"
---

## Domain Glossary
| Term | Definition |
|------|-----------|
| <term> | <definition> |

## Core Entities & Relationships
| Entity | Key Fields | Relationships |
|--------|-----------|---------------|
| <entity> | <fields> | <relations> |

## Business Rules (invariants)
- <rule>: <description>

## Key Workflows
1. <workflow-name>: <step1> → <step2> → <step3>

## Domain Anti-patterns
- <anti-pattern>: <why it's wrong in this domain>

## Validation Rules
| Field/Entity | Constraint |
|-------------|-----------|
| <field> | <rule> |

## Authorization Model
- <role>: <what they can do>

## External Integrations
| System | Purpose | Interface |
|--------|---------|-----------|
| <system> | <why> | <API/queue/file> |
```

### Phase 5: Review & Refine
> **Terminology Rule**: Domain terms (Ubiquitous Language) must NEVER be translated — not in docs, not in conversation, not in code comments. Translating domain terms breaks the shared understanding between code, documentation, and team communication. This applies to all languages (Vietnamese, etc.). Example: if the domain uses "Invoice", "Shipment", "Ledger", "Claim" — keep them as-is everywhere.

1. Present generated domain skill to user for review
2. User corrects/supplements domain knowledge
3. Finalize and save the domain skill file

## Decision Rules
- Breadth-first: scan wide before going deep
- Prefer code evidence over assumptions
- Mark low-confidence items with "(?)" for user review
- Stop discovery after Phase 2 if user only needs quick context
- Full Phase 1-5 for comprehensive onboarding
- **Domain terms are sacred**: never translate Ubiquitous Language terms — they are the shared vocabulary between code, docs, and people. Translating them (e.g. "Invoice" → "Hóa đơn") creates semantic drift and breaks DDD alignment

## Anti-patterns
- Reading every file in the project (token explosion)
- Guessing domain terms without code evidence
- Generating domain skill without user review
- Ignoring test files (they often encode business rules clearly)
- **Translating domain terms**: converting Ubiquitous Language into another language destroys the shared vocabulary that DDD depends on — domain terms must stay in their original form across code, docs, conversation, and UI labels where applicable

## Output
Domain skill file draft → User review → Finalized `.claude/skills/domain-<name>.md`
