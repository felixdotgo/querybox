package pluginmgr

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/felixdotgo/querybox/pkg/driverid"
	"github.com/felixdotgo/querybox/pkg/plugin"
	pluginpb "github.com/felixdotgo/querybox/rpc/contracts/plugin/v1"
	"github.com/felixdotgo/querybox/services"
	"google.golang.org/protobuf/encoding/protojson"
)

// PluginExecutor abstracts the subprocess execution of plugin commands.
// It is satisfied by *Manager and can be replaced with a test double in
// unit tests that should not spawn real processes.
type PluginExecutor interface {
	RunCommand(name, command string, timeout time.Duration, req []byte) ([]byte, error)
}

// Verify Manager satisfies PluginExecutor at compile time.
var _ PluginExecutor = (*Manager)(nil)

// RunCommand is the public implementation of PluginExecutor. It delegates to
// the internal runPluginCommand with a fixed caller label.
func (m *Manager) RunCommand(name, command string, timeout time.Duration, req []byte) ([]byte, error) {
	return m.runPluginCommand("RunCommand", name, command, timeout, req)
}

// runPluginCommand resolves the named plugin, spawns its binary with the given
// sub-command, writes reqBytes to stdin, and returns the raw stdout output.
// It handles plugin lookup, executable validation, pipe management, timeout
// detection, and error logging.  Callers are responsible for marshaling the
// request and unmarshaling the response.
//
// The `caller` parameter is a label used in log/error messages (e.g.
// "ExecPlugin", "GetConnectionTree") so that each call site produces
// recognisable diagnostics.
//
// Serialization contract: requests are serialized with encoding/json because
// all request structs are plain Go types (no proto enums or oneofs). Responses
// are parsed with protojson because plugins marshal proto messages with
// protojson. Do NOT change request types to generated proto messages without
// also switching request serialization to protojson.Marshal -- encoding/json
// would emit numeric enum values and Go field names instead of proto names,
// causing parse errors on the plugin side.
func (m *Manager) runPluginCommand(caller, name, command string, timeout time.Duration, reqBytes []byte) ([]byte, error) {
	name = driverid.Normalize(name)
	m.mu.Lock()
	info, ok := m.plugins[name]
	m.mu.Unlock()
	if !ok {
		m.emitLog(services.LogLevelError, fmt.Sprintf("%s: plugin '%s' not found", caller, name))
		return nil, fmt.Errorf("%s: plugin %s not found", caller, name)
	}
	full := info.Path
	if !isExecutable(full) {
		m.emitLog(services.LogLevelError, fmt.Sprintf("%s: plugin '%s' is not executable", caller, name))
		return nil, fmt.Errorf("%s: plugin %s is not executable", caller, name)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, full, command)
	hideWindow(cmd)
	cmd.Env = append(os.Environ(), "QUERYBOX_PLUGIN_NAME="+name)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		m.emitLog(services.LogLevelError, fmt.Sprintf("%s: stdin pipe error for plugin '%s': %v", caller, name, err))
		return nil, fmt.Errorf("%s: stdin pipe error: %w", caller, err)
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		m.emitLog(services.LogLevelError, fmt.Sprintf("%s: stdout pipe error for plugin '%s': %v", caller, name, err))
		return nil, fmt.Errorf("%s: stdout pipe error: %w", caller, err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		m.emitLog(services.LogLevelError, fmt.Sprintf("%s: stderr pipe error for plugin '%s': %v", caller, name, err))
		return nil, fmt.Errorf("%s: stderr pipe error: %w", caller, err)
	}

	if err := cmd.Start(); err != nil {
		m.emitLog(services.LogLevelError, fmt.Sprintf("%s: failed to start plugin '%s': %v", caller, name, err))
		return nil, fmt.Errorf("%s: start error: %w", caller, err)
	}

	if _, werr := stdin.Write(reqBytes); werr != nil {
		m.emitLog(services.LogLevelError, fmt.Sprintf("%s: stdin write error for plugin '%s': %v", caller, name, werr))
	}
	if cerr := stdin.Close(); cerr != nil {
		m.emitLog(services.LogLevelError, fmt.Sprintf("%s: stdin close error for plugin '%s': %v", caller, name, cerr))
	}

	outB, _ := io.ReadAll(stdoutPipe)
	errB, _ := io.ReadAll(stderrPipe)

	if err := cmd.Wait(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			m.emitLog(services.LogLevelError, fmt.Sprintf("%s: plugin '%s' timed out after %s", caller, name, timeout))
			return nil, fmt.Errorf("%s: plugin timed out after %s", caller, timeout)
		}
		m.emitLog(services.LogLevelError, fmt.Sprintf("%s: plugin '%s' exited with error: %v", caller, name, err))
		return nil, fmt.Errorf("%s: plugin exited: %w - stderr: %s", caller, err, string(errB))
	}

	return outB, nil
}

// ExecPlugin runs the named plugin with the provided connection info, query
// and optional options map.  Under the hood the manager spawns the binary,
// writes a protobuf-JSON `PluginV1_ExecRequest` to stdin, and reads a
// `PluginV1_ExecResponse` from stdout.  The `plugin` package exposes convenient
// aliases but the contract is defined in `contracts/plugin/v1/plugin.proto`.
// Callers receive the structured `plugin.ExecResponse` (alias for the proto
// type) or an error.  Historically this returned a raw string; callers may need
// to examine the `Result` field to access rows, documents, or key/value data.
func (m *Manager) ExecPlugin(name string, connection map[string]string, query string, options map[string]string) (*plugin.ExecResponse, error) {
	// Truncate long queries in log output to keep messages readable
	logQuery := query
	if len(logQuery) > 80 {
		logQuery = logQuery[:80] + "..."
	}
	if len(options) > 0 {
		m.emitLog(services.LogLevelInfo, fmt.Sprintf("ExecPlugin: executing (driver: %s, query: %q, options: %v)", name, logQuery, options))
	} else {
		m.emitLog(services.LogLevelInfo, fmt.Sprintf("ExecPlugin: executing (driver: %s, query: %q)", name, logQuery))
	}

	// build request envelope; include options map if supplied
	req := execRequest{Connection: connection, Query: query, Options: options}
	b, err := json.Marshal(&req)
	if err != nil {
		return nil, fmt.Errorf("ExecPlugin: marshal request: %w", err)
	}

	outB, err := m.runPluginCommand("ExecPlugin", name, "exec", defaultPluginTimeout, b)
	if err != nil {
		return nil, err
	}

	// if the plugin didn't emit JSON we still want to return something useful
	// so wrap the raw output in a simple key/value result.  Older clients may
	// still just render the string.
	resp := &plugin.ExecResponse{}
	if len(outB) == 0 {
		return resp, nil
	}
	// protobuf structs are better parsed with protojson which correctly
	// handles oneof fields and enum names.  Older plugins that used
	// `encoding/json` to marshal a proto struct would emit a top-level
	// "Payload" field instead of the expected variant-specific name
	// (e.g. "sql", "kv").  When that happens protojson.Unmarshal complains
	// about an unknown field; we attempt to repair the JSON so the response
	// can still be interpreted.
	if err := protojson.Unmarshal(outB, resp); err != nil {
		// attempt to correct common mis-formatting
		if strings.Contains(err.Error(), "unknown field \"Payload\"") {
			var raw map[string]interface{}
			if jerr := json.Unmarshal(outB, &raw); jerr == nil {
				if r, ok := raw["result"].(map[string]interface{}); ok {
					if payload, ok2 := r["Payload"].(map[string]interface{}); ok2 {
						// move inner keys (should be one of sql/document/kv) up
						for k, v := range payload {
							// older JSON produced by encoding/json used Go struct field names (Sql, Kv, Document).
							// lowercase them so protojson will match the proto name.
							r[strings.ToLower(k)] = v
						}
						delete(r, "Payload")
						if fixed, merr := json.Marshal(raw); merr == nil {
							if perr := protojson.Unmarshal(fixed, resp); perr == nil {
								return resp, nil
							}
						}
					}
				}
			}
		}
		m.emitLog(services.LogLevelError, fmt.Sprintf("ExecPlugin: JSON unmarshal failed for plugin '%s': %v", name, err))
		// fallback to embedding the raw output in a KV map under "_".
		return &plugin.ExecResponse{
			Result: &pluginpb.PluginV1_ExecResult{
				Payload: &pluginpb.PluginV1_ExecResult_Kv{
					Kv: &pluginpb.PluginV1_KeyValueResult{
						Data: map[string]string{"_": string(outB)},
					},
				},
			},
		}, nil
	}
	if resp.Error != "" {
		m.emitLog(services.LogLevelError, fmt.Sprintf("ExecPlugin: plugin '%s' returned error: %s", name, resp.Error))
		return resp, fmt.Errorf("ExecPlugin: plugin error: %s", resp.Error)
	}
	m.emitLog(services.LogLevelInfo, fmt.Sprintf("ExecPlugin: (driver: %s) completed successfully", name))
	return resp, nil
}

// GetConnectionTree asks the named plugin for its connection tree.  The
// request contains only the connection map; the plugin defines node structure
// and actions.  A timeout guards misbehaving plugins.
func (m *Manager) GetConnectionTree(name string, connection map[string]string) (*plugin.ConnectionTreeResponse, error) {
	m.emitLog(services.LogLevelInfo, fmt.Sprintf("GetConnectionTree: fetching tree (driver: %s)", name))

	req := plugin.ConnectionTreeRequest{Connection: connection}
	b, err := json.Marshal(&req)
	if err != nil {
		return nil, fmt.Errorf("GetConnectionTree: marshal request: %w", err)
	}

	outB, err := m.runPluginCommand("GetConnectionTree", name, "connection-tree", defaultPluginTimeout, b)
	if err != nil {
		return nil, err
	}

	resp := &plugin.ConnectionTreeResponse{}
	if len(outB) == 0 {
		m.emitLog(services.LogLevelInfo, fmt.Sprintf("GetConnectionTree: (driver: %s) returned empty tree", name))
		return resp, nil
	}
	if err := protojson.Unmarshal(outB, resp); err != nil {
		m.emitLog(services.LogLevelError, fmt.Sprintf("GetConnectionTree: invalid tree JSON from '%s': %v", name, err))
		return nil, fmt.Errorf("GetConnectionTree: invalid tree json: %w", err)
	}
	m.emitLog(services.LogLevelInfo, fmt.Sprintf("GetConnectionTree: (driver: %s) returned %d node(s)", name, len(resp.Nodes)))
	return resp, nil
}

// ExecTreeAction is a convenience wrapper for executing the query payload
// attached to a tree node action.  It simply forwards to ExecPlugin and
// propagates any provided options map (for example "explain-query").
func (m *Manager) ExecTreeAction(name string, connection map[string]string, actionQuery string, options map[string]string) (*plugin.ExecResponse, error) {
	return m.ExecPlugin(name, connection, actionQuery, options)
}

// MutateRow forwards a single-row mutation request to the specified plugin.
// The semantics of `source`, `values` and `filter` are driver-defined; the
// core does not interpret them.  The operation type (insert/update/delete)
// is described by the OperationType enum.  A 30-second timeout guards
// against misbehaving plugins.
func (m *Manager) MutateRow(name string, connection map[string]string, operation plugin.OperationType, source string, values map[string]string, filter string) (*plugin.MutateRowResponse, error) {
	m.emitLog(services.LogLevelInfo, fmt.Sprintf("MutateRow: (driver: %s) op=%v source=%q filter=%q", name, operation, source, filter))

	req := mutateRowRequest{Connection: connection, Operation: operation, Source: source, Values: values, Filter: filter}
	b, err := json.Marshal(&req)
	if err != nil {
		return nil, fmt.Errorf("MutateRow: marshal request: %w", err)
	}

	outB, err := m.runPluginCommand("MutateRow", name, "mutate-row", defaultPluginTimeout, b)
	if err != nil {
		return nil, err
	}

	resp := &plugin.MutateRowResponse{}
	if len(outB) == 0 {
		m.emitLog(services.LogLevelInfo, fmt.Sprintf("MutateRow: (driver: %s) returned empty response", name))
		return resp, nil
	}
	if err := json.Unmarshal(outB, resp); err != nil {
		m.emitLog(services.LogLevelError, fmt.Sprintf("MutateRow: invalid JSON from '%s': %v", name, err))
		return nil, fmt.Errorf("MutateRow: invalid json: %w", err)
	}
	return resp, nil
}

// DescribeSchema asks the named plugin to provide schema metadata for the
// given connection.  The optional database/table arguments may be empty;
// plugins are free to ignore them.  A 30-second timeout prevents hangs.
func (m *Manager) DescribeSchema(name string, connection map[string]string, database, table string) (*plugin.DescribeSchemaResponse, error) {
	m.emitLog(services.LogLevelInfo, fmt.Sprintf("DescribeSchema: fetching schema (driver: %s)", name))

	req := plugin.DescribeSchemaRequest{Connection: connection, Database: database, Table: table}
	b, err := json.Marshal(&req)
	if err != nil {
		return nil, fmt.Errorf("DescribeSchema: marshal request: %w", err)
	}

	outB, err := m.runPluginCommand("DescribeSchema", name, "describe-schema", defaultPluginTimeout, b)
	if err != nil {
		return nil, err
	}

	resp := &plugin.DescribeSchemaResponse{}
	if len(outB) == 0 {
		m.emitLog(services.LogLevelInfo, fmt.Sprintf("DescribeSchema: (driver: %s) returned empty schema", name))
		return resp, nil
	}
	if err := protojson.Unmarshal(outB, resp); err != nil {
		m.emitLog(services.LogLevelError, fmt.Sprintf("DescribeSchema: invalid JSON from '%s': %v", name, err))
		return nil, fmt.Errorf("DescribeSchema: invalid json: %w", err)
	}
	m.emitLog(services.LogLevelInfo, fmt.Sprintf("DescribeSchema: (driver: %s) returned %d tables", name, len(resp.Tables)))
	return resp, nil
}

// TestConnection invokes the named plugin's `test-connection` command to verify
// that the supplied connection parameters are valid. The plugin is expected to
// open and ping the underlying data store without persisting anything.
// Old plugins that do not implement the command will exit non-zero; in that
// case TestConnection returns an error rather than a failed response so the
// caller can distinguish "unsupported" from "tested and failed".
func (m *Manager) TestConnection(name string, connection map[string]string) (*plugin.TestConnectionResponse, error) {
	m.emitLog(services.LogLevelInfo, fmt.Sprintf("TestConnection: testing (driver: %s)", name))

	req := plugin.TestConnectionRequest{Connection: connection}
	b, err := json.Marshal(&req)
	if err != nil {
		return nil, fmt.Errorf("TestConnection: marshal request: %w", err)
	}

	outB, err := m.runPluginCommand("TestConnection", name, "test-connection", fastPluginTimeout, b)
	if err != nil {
		return nil, err
	}

	var resp plugin.TestConnectionResponse
	if len(outB) == 0 {
		return &resp, nil
	}
	if err := json.Unmarshal(outB, &resp); err != nil {
		return nil, fmt.Errorf("TestConnection: invalid response json: %w", err)
	}

	if resp.Ok {
		m.emitLog(services.LogLevelInfo, fmt.Sprintf("TestConnection: (driver: %s) success: %s", name, resp.Message))
	} else {
		m.emitLog(services.LogLevelWarn, fmt.Sprintf("TestConnection: (driver: %s) failed: %s", name, resp.Message))
	}
	return &resp, nil
}

// GetPluginAuthForms probes the plugin executable for supported authentication
// forms by invoking `plugin authforms` and decoding the JSON response. If the
// plugin doesn't implement the command or returns no forms an empty map is
// returned.
func (m *Manager) GetPluginAuthForms(name string) (map[string]*plugin.AuthForm, error) {
	// Use runPluginCommand for consistent subprocess handling (env vars,
	// logging, timeout, hideWindow). authforms takes no stdin input.
	out, err := m.runPluginCommand("GetPluginAuthForms", name, "authforms", fastPluginTimeout, nil)
	if err != nil {
		// treat as not implemented gracefully
		return nil, nil
	}
	if len(out) == 0 {
		return nil, nil
	}
	var resp plugin.AuthFormsResponse
	if err := protojson.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("GetPluginAuthForms: invalid authforms json: %w", err)
	}
	ret := make(map[string]*plugin.AuthForm)
	for k, v := range resp.Forms {
		if v == nil {
			continue
		}
		ret[k] = v
	}
	return ret, nil
}

// GetCompletionFields asks the named plugin for discoverable field names for a
// specific database/collection.  The call is used by the editor auto-completion
// feature.  Plugins that don't implement the CompletionFieldsProvider interface
// return an empty response.  A 15-second timeout guards misbehaving plugins.
func (m *Manager) GetCompletionFields(name string, connection map[string]string, database, collection string) (*plugin.GetCompletionFieldsResponse, error) {
	m.emitLog(services.LogLevelInfo, fmt.Sprintf("GetCompletionFields: fetching fields (driver: %s, collection: %s)", name, collection))

	req := plugin.GetCompletionFieldsRequest{Connection: connection, Database: database, Collection: collection}
	b, err := json.Marshal(&req)
	if err != nil {
		return &plugin.GetCompletionFieldsResponse{}, nil
	}

	outB, err := m.runPluginCommand("GetCompletionFields", name, "completion-fields", fastPluginTimeout, b)
	if err != nil {
		// Non-zero exit is expected for older plugins that don't implement this
		// command -- return empty response rather than an error so callers don't
		// have to handle the unsupported case specially.
		return &plugin.GetCompletionFieldsResponse{}, nil
	}

	resp := &plugin.GetCompletionFieldsResponse{}
	if len(outB) == 0 {
		return resp, nil
	}
	if err := protojson.Unmarshal(outB, resp); err != nil {
		m.emitLog(services.LogLevelError, fmt.Sprintf("GetCompletionFields: invalid JSON from '%s': %v", name, err))
		return &plugin.GetCompletionFieldsResponse{}, nil
	}
	m.emitLog(services.LogLevelInfo, fmt.Sprintf("GetCompletionFields: (driver: %s) returned %d fields", name, len(resp.Fields)))
	return resp, nil
}
