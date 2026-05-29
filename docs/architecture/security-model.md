# Security Model

## Current baseline

- Credentials are kept out of `data/connections.db`.
- Plugins receive credentials through stdin, not environment variables or command arguments.
- Process isolation is coarse: plugins run with the same OS privileges as the host app.
- Timeout and manifest limits provide bounded execution, but not strong sandboxing.

## Threat model summary

| Threat | Current mitigation | Status |
|--------|--------------------|--------|
| Secrets on disk | OS keyring primary storage, reference-only connection DB | strong baseline |
| Plugin crash or hang | per-request subprocess model plus timeouts | good baseline |
| Malicious plugin access | manifest validation and bounded execution only | incomplete |
| Resource exhaustion | timeout and output limits | partial |
| Cross-user secret access | OS keyring isolation | platform-dependent |

## Roadmap

- surfaced permission declarations in UX
- stronger host controls for env, workdir, and output ceilings
- sandbox level model for trust tiers
- possible future code signing and audit trails

## Related docs

- [Plugin capabilities reference](../reference/plugin-capabilities.md)
- [Runtime evolution roadmap](runtime-evolution-roadmap.md)
