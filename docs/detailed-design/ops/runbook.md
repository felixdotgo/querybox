# Runbook — Core service

## Daily checks
- Alert queue size, error-rate, and Prometheus health metrics.

## Deploy
1. Bump image tag and run canary.
2. Monitor telemetry for 5-15 minutes.
3. Promote on success, roll back on error.

## Common incidents
- Driver spawn failures: check resource limits and kernel logs.
- Master-key missing: fail fast and notify ops; restore from secret manager.

## Rollback
- Revert deployment; notify stakeholders; run health checks.
