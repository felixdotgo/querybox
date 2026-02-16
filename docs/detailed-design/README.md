# Detailed Design

This folder contains the *detailed design* documents for QueryBox — component designs, data models, operational runbooks, and diagrams intended for implementers and reviewers.

How to use
- Keep narrative decisions and diagrams here.
- Use `kebab-case` for filenames and `README.md` at folder roots.
- Store diagram sources (`.mmd`, `.puml`) in `diagrams/` so they can be exported.

Important files
- `architecture.md` — system architecture and high-level flows
- `data-model.md` — ER diagrams and schema details
- `components/` — per-component design docs (Core, Drivers)
- `ops/runbook.md` — operational procedures and incident playbooks

Templates
- Follow the templates in `components/` for new component docs and add examples to `flows/` for sequence diagrams.

Versioning
- For major redesigns create `v2/` under this folder and keep old versions immutable.
