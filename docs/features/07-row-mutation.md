# Row Mutation

QueryBox now supports editing and deleting individual records or documents
returned by a query.  This feature is driven by a new `MutateRow` RPC in the
plugin contract which allows the core to forward user-initiated mutations to
driver plugins.

## User experience

- After executing a query, each row (or document / key-value entry) displays
  a pencil (edit) and trash can (delete) icon.
- Clicking **Edit** opens a modal pre-filled with the row's values and an
  editable filter expression.  Users may modify field values and/or alter the
  filter before submitting.
- Clicking **Delete** shows a confirmation dialog with the computed filter; the
  user can edit the filter if desired.
- Upon successful mutation the result tab automatically refreshes.
- If the connected plugin does not implement `MutateRow` (or advertises
  `capabilities: ["mutate-row"]`), the icons are hidden and the UI remains
  read-only.

## Plugin contract

The protobuf definition for the new RPC lives in `contracts/plugin/v1/plugin.proto`:

```proto
rpc MutateRow(MutateRowRequest) returns (MutateRowResponse);

message MutateRowRequest {
  map<string,string> connection = 1;
  OperationType operation = 2;
  string source = 3;
  map<string,string> values = 4;
  string filter = 5;
}

message MutateRowResponse {
  bool success = 1;
  string error = 2;
}
```

`OperationType` is an enum with values `INSERT`, `UPDATE`, and `DELETE`.
Plugins may treat `source`, `values` and `filter` however they choose; these are
opaque strings forwarded from the UI.  The request structure is optional – if a
plugin does not implement the RPC it should either decline the CLI command or
return `success=false` with an error message.  The core also hides the edit/
delete icons when the plugin's `capabilities` list omits `"mutate-row"`.

## Backend implementation

- `pkg/plugin` exposes aliases for the new request/response types and
  extends `ServeCLI` with a `mutate-row` command.
- `services/pluginmgr.Manager` gains a `MutateRow` helper that invokes the
delegate binary with a 30-second timeout, marshals the request to JSON, and
returns the response.
- The Wails bindings were regenerated to expose `MutateRow` to the frontend.

## Plugin author guidance

- Implement the `MutateRow` RPC on your `PluginServiceServer`.  A stub that
  always returns success is permissible; sophisticated drivers should apply
  the requested mutation to the data store and honour the `filter` parameter.
  If the operation cannot be performed, return `success=false` and a
  descriptive `error` string – this is how the host distinguishes unsupported
  or failed mutations from transport errors.
- Populate `capabilities` metadata with `"mutate-row"` if you support the
  feature; the frontend checks this list and hides the edit/delete icons for
  drivers that do not advertise the capability.
- When running as a CLI binary you should also handle the new
  `mutate-row` command in `pkg/plugin.ServeCLI` (the help message has been
  updated accordingly).  The host writes a JSON request to stdin and expects a
  JSON response, so your CLI path must decode/encode appropriately.
## Frontend changes

- New composable `useRowMutation.ts` wraps the generated binding and handles
  credential retrieval.
- `ResultViewer` and its subcomponents render per-row action icons and present
  a lightweight `RowEditorModal` for user input.
- `WorkspacePanel` listens for the `mutated` event and re-executes the query.
- Unit tests added for the backend, plugin examples, and basic UI behavior.

## Future enhancements

- Automatically derive filter expressions using primary-key metadata.
- Offer inline in-table editing rather than a modal.
- Persist mutations in query history or undo stack.
