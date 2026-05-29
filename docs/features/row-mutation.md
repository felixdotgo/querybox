# Row Mutation

## What it is

Row mutation lets a plugin-backed result surface edit and delete operations for returned records or documents.

## Current behavior

- The UI shows edit and delete affordances only when the plugin advertises `mutate-row`.
- The backend forwards mutation requests to the plugin over the `mutate-row` command.
- Successful mutation flows refresh the visible result.

## Key components

- `pkg/plugin`
- `services/pluginmgr`
- `frontend/src/composables/useRowMutation.ts`
- result viewers and row editor UI

## How it connects to other modules

- extends the plugin execution model beyond read-only queries
- depends on capability metadata from manifests
- reuses connection credentials and result context

## Limits and roadmap

- Mutation semantics are still plugin-defined and intentionally lightweight.
- Better key inference and richer inline editing remain future enhancements.
