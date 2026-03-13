# Feature: Query Editor Auto‑completion

## Overview

The query workspace includes a rich editor that helps users write queries faster and
with fewer errors. Auto‑completion combines three sources:

1. **Built‑in language constructs.** Keywords (SELECT, WHERE, etc.), CLI commands
   (`\copy`, `\exec`, etc.), and common functions are always available.
2. **Tab‑specific schema metadata.** When a connection is active the editor calls
   the backend, which in turn probes the associated plugin for field/column names
   using the optional `completion-fields` RPC. The result is cached per tab
   to avoid excessive plugin invocations.
3. **Smart ranking.** The composable `useTabCompletion` ranks suggestions based
   on the current query context, recently used tables, and the selected node in
   the object browser tree.

Suggestions appear in a dropdown as the user types; pressing `Tab` or `Enter`
accepts the highlighted item. The feature is enabled by default but can be
disabled via the settings panel ("Editor → Auto‑completion").

## API / Contract

### Frontend

- `PluginManager.GetCompletionFields(pluginName, connParams, database, collection)`
  — returns `{fields:[{name:string,type?:string}]}`. The call times out after 5s
  and is automatically retried when the tab's connection or database changes.
- The editor component (`QueryEditor.vue`) merges the plugin result with its
  static keyword list and hands the combined array to the underlying Monaco
  / Naive UI `AutoComplete` component.

### Plugin

- Implement the optional `GetCompletionFields` RPC as defined in
  `contracts/plugin/v1/plugin.proto`.
- Response may be empty; the host treats an empty result as "no metadata" and
  simply shows built‑in suggestions.
- Schemaless drivers can sample recently executed documents or leverage an
  attributes API. Example comment from `plugins/arangodb/main.go`:
  ```go
  // ATTRIBUTES() so the editor autocomplete can offer field suggestions.
  ```
- No state should be persisted; the call is fire‑and‑forget and has a short
  timeout. Plugins that do not implement this command should return `{}` or
  simply exit with code 0.

## Implementation Notes

- The completions are cached in `useTabCompletion.js` keyed by connection ID +
  database + collection. Cache entries expire after 3 minutes of inactivity.
- On the frontend, keystroke handling is debounced to 150 ms to avoid flooding
  the backend with requests during fast typing.
- The `QueryEditor` component enables quick suggestions (`quickSuggestions: true`)
  so the dropdown appears without needing to press Ctrl‑Space.

## Edge Cases

- **Plugin times out or crashes.** The call fails silently; the editor continues
  to offer only built‑in suggestions and retries on the next context change.
- **Connection is unauthenticated.** Completion requests are not sent; the
  dropdown still shows static keywords.
- **Large schemaless collections.** Plugins should limit sampling to a small
  subset (e.g. last 100 documents) to stay within the 5s timeout.

---
