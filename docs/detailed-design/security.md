# Security & Privacy

## Areas covered
- AuthN/AuthZ
- Secrets management (OS keyring + master key)
- Network & runtime isolation for drivers
- Logging and data retention

## Recommendations
- Use AES-256-GCM for credential blobs (already in design).
- Require master-key provisioning via secret manager in server installs.
- Drivers run non‑root with resource caps; add seccomp / namespace isolation post‑MVP.

## Threat model & controls
- Never log plaintext credentials; redact query parameters when configured.
- Audit access to `connections` and master-key operations.

## Compliance notes
- Document retention and deletion policies for `audit_logs` and credentials.
