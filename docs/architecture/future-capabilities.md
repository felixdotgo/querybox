# Future Capabilities

## Future / not implemented

These directions remain useful design targets but are not part of the current shipped baseline.

## Candidate capability areas

- `stream.read` for logs, queues, and live views
- remote runtime targets and self-hosted agent execution
- explicit workspace persistence for investigation context
- stronger sandbox levels and trust policy
- future integrations such as Kafka, S3/MinIO, Kubernetes, Prometheus, and logs

## Product caution

QueryBox should expand only where the operational-workspace model stays clear:

- inspect
- query
- debug
- operate

The product should avoid becoming a generic infrastructure control plane or accumulating integrations that dilute the core workflow quality.
