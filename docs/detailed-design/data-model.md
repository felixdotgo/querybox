# Data model

## Purpose
Describe persistent data structures, constraints, indexes, and migration considerations.

## ER diagram
- Add an ER diagram (source in `diagrams/`)

## Tables (example)
- `connections` — id, name, driver_type, encrypted_credential_blob, owner_id, created_at, updated_at
- `audit_logs` — id, session_id, event_type, payload, created_at

## Indexes & constraints
- Uniqueness, foreign keys, critical indexes for queries used by Core.

## Migration notes
- Include backward-compatibility considerations and offline/online migration steps.

## Example SQL
```sql
CREATE TABLE connections (
  id UUID PRIMARY KEY,
  name TEXT NOT NULL,
  driver_type TEXT NOT NULL,
  encrypted_credential_blob BYTEA NOT NULL,
  owner_id UUID NOT NULL,
  created_at TIMESTAMPTZ DEFAULT now()
);
```

## TODO
- Add detailed column descriptions and retention policy for `audit_logs`.
