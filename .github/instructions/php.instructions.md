---
applyTo: "**/*.php"
description: "PHP/Laravel Overlay — Principal Architect Protocol"
---

Extends core protocol. PHP/Laravel-specific rules only; shared behavior in `core.copilot-instructions.md`.

## Architecture
- Follow Laravel conventions; thin controllers, logic in services/actions
- Form Request for validation in non-trivial endpoints
- API Resources for stable response contracts

## Implementation
- Strict type hints for parameters and return values
- Eloquent/Query Builder with parameter binding; no raw SQL unless justified
- `$fillable` or `$guarded` against mass assignment
- Authorization via Policies/Gates

## Error Handling
- Appropriate HTTP status codes; transactions for multi-step writes
- Catch exceptions only where recovery/translation needed
- No `dd`, `dump` in production paths

## Testing
- Feature tests: endpoint behavior and authorization
- Unit tests: isolated business logic
- Cover: happy path, validation failures, authorization denial

## Forbidden
- Business logic in Blade views
- Missing authorization on protected operations
- Unparameterized SQL with user input
- `env()` outside configuration files
