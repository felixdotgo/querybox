package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"unicode/utf8"

	pluginpb "github.com/felixdotgo/querybox/rpc/contracts/plugin/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

// Reuse proto-derived types (aliases) so plugin authors import a single
// stable package (`github.com/felixdotgo/querybox/rpc/contracts/plugin/v1`) if they prefer. These aliases keep
// the existing `pkg/plugin` API stable while aligning it with the .proto.
type InfoResponse = pluginpb.PluginV1_InfoResponse

type ExecRequest = pluginpb.PluginV1_ExecRequest

// ExecResponse now contains a typed ExecResult which can represent SQL rows,
// document lists, or key/value maps. Plugins should return one of those
// payloads rather than a flat string.
type ExecResponse = pluginpb.PluginV1_ExecResponse

// result-specific helpers.  Exported for plugin authors and tests.
// FormatSQLValue translates a value returned by `database/sql` Row.Scan
// into a human-readable string suitable for presenting in the host UI. The
// built-in drivers often return []byte for text columns, so we convert those
// to strings rather than letting fmt.Sprintf render them as numeric byte
// slices. A nil value becomes the empty string.
func FormatSQLValue(v interface{}) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case []byte:
		// Drivers commonly return []byte for text columns. Convert to
		// string when the bytes represent valid UTF-8; otherwise encode as a
		// hex string so the frontend can still display something sensible and
		// avoid embedding potentially invalid/unprintable data in the JSON
		// payload.
		if utf8.Valid(t) {
			return string(t)
		}
		// show binary data as hex prefixed with 0x (similar to SQL conventions)
		return fmt.Sprintf("0x%x", t)
	default:
		// Fallback to the generic formatter used previously.
		return fmt.Sprintf("%v", v)
	}
}

type ExecResult = pluginpb.PluginV1_ExecResult

type SqlResult = pluginpb.PluginV1_SqlResult

type Column = pluginpb.PluginV1_Column

type Row = pluginpb.PluginV1_Row

type DocumentResult = pluginpb.PluginV1_DocumentResult

type KeyValueResult = pluginpb.PluginV1_KeyValueResult

// DriverType reuse from protobuf enum
type DriverType = pluginpb.PluginV1_Type

// Aliases for the new AuthForms protobuf messages
type AuthField = pluginpb.PluginV1_AuthField
type AuthForm = pluginpb.PluginV1_AuthForm
type AuthFormsRequest = pluginpb.PluginV1_AuthFormsRequest
type AuthFormsResponse = pluginpb.PluginV1_AuthFormsResponse

// Connection‑tree aliases
// these correspond to the `ConnectionTree` RPC introduced for browsing a
// connection.  Each driver may return its own structure; the core simply
// renders the nodes and forwards any action queries back to the plugin.

type ConnectionTreeRequest = pluginpb.PluginV1_ConnectionTreeRequest
type ConnectionTreeResponse = pluginpb.PluginV1_ConnectionTreeResponse
type ConnectionTreeNode = pluginpb.PluginV1_ConnectionTreeNode
type ConnectionTreeAction = pluginpb.PluginV1_ConnectionTreeAction

// TestConnectionRequest / TestConnectionResponse are type aliases for the
// proto-package types defined in rpc/contracts/plugin/v1.  When protoc
// regenerates plugin.pb.go these will resolve to the fully-registered proto
// structs; until then they resolve to the hand-maintained plain structs in
// plugin_test_connection.go in the same package.
type TestConnectionRequest = pluginpb.PluginV1_TestConnectionRequest
type TestConnectionResponse = pluginpb.PluginV1_TestConnectionResponse

const (
	TypeDriver DriverType = pluginpb.PluginV1_DRIVER

	AuthFieldText     = pluginpb.PluginV1_AuthField_TEXT
	AuthFieldNumber   = pluginpb.PluginV1_AuthField_NUMBER
	AuthFieldPassword = pluginpb.PluginV1_AuthField_PASSWORD
	AuthFieldSelect   = pluginpb.PluginV1_AuthField_SELECT
	AuthFieldCheckbox = pluginpb.PluginV1_AuthField_CHECKBOX
	AuthFieldFilePath = pluginpb.PluginV1_AuthField_FILE_PATH

	// common action types for ConnectionTree nodes.  Plugins should use
	// these constants rather than hardcoding strings to avoid typos and to
	// document the set of recognised actions.
	ConnectionTreeActionSelect   = "select"
	ConnectionTreeActionDescribe = "describe"

	// DDL action types – rendered as context-menu items on database/table nodes.
	ConnectionTreeActionCreateDatabase = "create-database"
	ConnectionTreeActionDropDatabase   = "drop-database"
	ConnectionTreeActionCreateTable    = "create-table"
	ConnectionTreeActionDropTable      = "drop-table"

	// Common node types for ConnectionTree.  The core uses these to determine
	ConnectionTreeNodeTypeDatabase   = pluginpb.PluginV1_NODE_TYPE_DATABASE
	ConnectionTreeNodeTypeTable      = pluginpb.PluginV1_NODE_TYPE_TABLE
	ConnectionTreeNodeTypeColumn     = pluginpb.PluginV1_NODE_TYPE_COLUMN
	ConnectionTreeNodeTypeSchema     = pluginpb.PluginV1_NODE_TYPE_SCHEMA
	ConnectionTreeNodeTypeView       = pluginpb.PluginV1_NODE_TYPE_VIEW
	ConnectionTreeNodeTypeAction     = pluginpb.PluginV1_NODE_TYPE_ACTION
	ConnectionTreeNodeTypeCollection = pluginpb.PluginV1_NODE_TYPE_COLLECTION
	ConnectionTreeNodeTypeKey        = pluginpb.PluginV1_NODE_TYPE_KEY
)

// Historically this package exported a custom `Plugin` interface, but the
// protobuf contract now defines the canonical server API.  Plugins should
// implement the generated `pluginpb.PluginServiceServer` interface directly.
// We keep a handful of lightweight type aliases for convenience, but the
// local interface has been removed to keep this package lean.

// ServeCLI runs a protobuf-based service implementation over stdin/stdout.
// Plugins written in any language can implement the service; the helper simply
// invokes the corresponding RPC-style methods on the provided server object.
//
// Supported commands are identical to the previous Go-only helper but now use
// protojson for marshalling so communication is language-agnostic.
func ServeCLI(s pluginpb.PluginServiceServer) {
	args := os.Args[1:]
	if len(args) == 0 {
		usage()
		os.Exit(2)
	}

	switch args[0] {
	case "info":
		info, err := s.Info(context.Background(), &pluginpb.PluginV1_InfoRequest{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "plugin: info error: %v\n", err)
			os.Exit(1)
		}
		b, _ := protojson.Marshal(info)
		_, _ = os.Stdout.Write(b)
	case "exec":
		in, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "plugin: failed to read stdin: %v\n", err)
			os.Exit(1)
		}
		var req pluginpb.PluginV1_ExecRequest
		if err := json.Unmarshal(in, &req); err != nil {
			fmt.Fprintf(os.Stderr, "plugin: invalid request json: %v\n", err)
			os.Exit(1)
		}
		res, err := s.Exec(context.Background(), &req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "plugin: exec error: %v\n", err)
			os.Exit(1)
		}
		b, _ := protojson.Marshal(res)
		_, _ = os.Stdout.Write(b)
	case "authforms":
		res, err := s.AuthForms(context.Background(), &pluginpb.PluginV1_AuthFormsRequest{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "plugin: authforms error: %v\n", err)
			os.Exit(1)
		}
		b, _ := protojson.Marshal(res)
		_, _ = os.Stdout.Write(b)
	case "connection-tree", "tree":
		in, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "plugin: failed to read stdin: %v\n", err)
			os.Exit(1)
		}
		var req pluginpb.PluginV1_ConnectionTreeRequest
		if err := json.Unmarshal(in, &req); err != nil {
			fmt.Fprintf(os.Stderr, "plugin: invalid tree request json: %v\n", err)
			os.Exit(1)
		}
		res, err := s.ConnectionTree(context.Background(), &req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "plugin: connection-tree error: %v\n", err)
			os.Exit(1)
		}
		b, _ := protojson.Marshal(res)
		_, _ = os.Stdout.Write(b)
	case "test-connection":
		in, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "plugin: failed to read stdin: %v\n", err)
			os.Exit(1)
		}
		var req pluginpb.PluginV1_TestConnectionRequest
		if err := json.Unmarshal(in, &req); err != nil {
			fmt.Fprintf(os.Stderr, "plugin: invalid test-connection request json: %v\n", err)
			os.Exit(1)
		}
		res, err := s.TestConnection(context.Background(), &req)
		if err != nil {
			res = &pluginpb.PluginV1_TestConnectionResponse{Ok: false, Message: err.Error()}
		}
		b, _ := protojson.Marshal(res)
		_, _ = os.Stdout.Write(b)
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: <plugin> info | exec | authforms | connection-tree | test-connection (request on stdin as JSON)")
}
