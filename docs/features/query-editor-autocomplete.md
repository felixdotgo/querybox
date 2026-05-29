# Query Editor Autocomplete

## What it is

The query editor combines static language suggestions with plugin-provided schema metadata to accelerate query authoring.

## Current behavior

- Built-in keywords and commands are always available.
- Optional `completion-fields` requests enrich suggestions with driver-specific field or column names.
- Suggestions are cached per tab context to avoid excessive plugin calls.

## Key components

- query editor component
- `frontend/src/composables/useTabCompletion.ts`
- plugin optional `completion-fields` support

## How it connects to other modules

- depends on the connection and selected resource context
- uses plugin-exposed metadata through the backend
- improves result-oriented workflows without changing the execution contract

## Limits and roadmap

- Failures degrade to built-in suggestions only.
- Streaming or live metadata enrichment is not part of the current baseline.
