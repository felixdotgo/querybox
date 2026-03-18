---
name: migration-upgrade
description: "Safe migration and upgrade execution. Activate for database migrations, dependency upgrades, framework version bumps, or data transformations."
---

## When to Activate
- Database schema migration (add/alter/drop tables, columns, indexes)
- Dependency or framework version upgrade
- Data transformation or backfill
- API version migration (breaking changes)
- Infrastructure or config migration

## Procedure

### 1. Impact Assessment
1. Identify what's changing and what depends on it
2. Map blast radius: affected services, consumers, data, configs
3. Check backward compatibility requirements
4. Estimate downtime risk (zero-downtime vs maintenance window)
5. Identify rollback complexity (easy/hard/impossible)

### 2. Migration Strategy
Choose approach based on risk:
- **Expand-Contract** (preferred): add new → migrate → remove old
- **Blue-Green**: parallel environments, switch traffic
- **Rolling**: incremental rollout with canary verification
- **Big Bang**: single cutover (last resort, requires maintenance window)

### 3. Pre-migration Checklist
- [ ] Backup strategy defined and tested
- [ ] Rollback script or procedure ready
- [ ] Breaking changes documented
- [ ] Dependent services notified (if applicable)
- [ ] Migration tested in non-production environment
- [ ] Performance impact estimated (large table migrations, index builds)

### 4. Execute Migration
1. Apply migration in smallest safe increments
2. Verify each increment before proceeding
3. For database: use non-locking operations where possible
4. For dependencies: update one major dependency at a time
5. Run test suite after each increment

### 5. Post-migration Verification
1. Verify data integrity (row counts, checksums, spot checks)
2. Run full test suite
3. Check application health metrics
4. Verify rollback procedure still works
5. Clean up old artifacts (old columns, deprecated code, feature flags)

## Database Migration Rules
- Always write both `up` and `down` migrations
- Separate schema changes from data migrations
- Use non-blocking DDL when available (e.g., `CREATE INDEX CONCURRENTLY`)
- Never drop columns in the same release that stops writing to them
- Add new nullable columns or columns with defaults — never add required columns to existing tables without backfill strategy

## Dependency Upgrade Rules
- Read changelog/release notes for breaking changes
- Update lock file, not just manifest
- One major version bump at a time
- Run full test suite after each upgrade
- Check for deprecated API usage

## Decision Rules
- Reversible over irreversible migrations
- Smallest safe increment over large batch changes
- Expand-contract over big bang when possible
- Test rollback before executing migration
- If rollback is impossible, extra verification before proceeding

## Anti-patterns
- Migrating without a rollback plan
- Combining schema changes with application logic changes in one deploy
- Dropping columns/tables without verifying no code references remain
- Skipping non-production testing for "simple" migrations
- Ignoring migration performance on production-sized datasets

## Output
Impact assessment → Strategy chosen → Pre-checks → Execution steps → Verification → Rollback plan