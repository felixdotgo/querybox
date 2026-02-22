package plugin

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	pluginpb "github.com/felixdotgo/querybox/rpc/contracts/plugin/v1"
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

const (
	TypeDriver DriverType = pluginpb.PluginV1_DRIVER

	AuthField_TEXT     = pluginpb.PluginV1_AuthField_TEXT
	AuthField_NUMBER   = pluginpb.PluginV1_AuthField_NUMBER
	AuthField_PASSWORD = pluginpb.PluginV1_AuthField_PASSWORD
	AuthField_SELECT   = pluginpb.PluginV1_AuthField_SELECT
	AuthField_CHECKBOX = pluginpb.PluginV1_AuthField_CHECKBOX

	// common action types for ConnectionTree nodes.  Plugins should use
	// these constants rather than hardcoding strings to avoid typos and to
	// document the set of recognised actions.
	ConnectionTreeAction_SELECT   = "select"
	ConnectionTreeAction_DESCRIBE = "describe"
)

// Plugin describes the minimal contract a plugin should implement. Keeping
// this small and explicit makes it easy to implement either an in-process
// plugin or an on-demand executable that uses ServeCLI below.
type Plugin interface {
	// Info returns metadata about the plugin that the host can display.
	Info() (InfoResponse, error)

	// Exec executes a request from the host and returns a response.
	// The host will pass a query and a map of connection/authentication parameters (e.g. host, user, password)
	// that the plugin can use to connect to a database or service and execute the query. The plugin is responsible
	// for defining the expected connection parameters and handling the execution logic.
	Exec(*ExecRequest) (*ExecResponse, error)

	// AuthForms returns available authentication forms the plugin supports.
	// Optional for existing plugins — implementations may return an empty map.
	AuthForms(*AuthFormsRequest) (*AuthFormsResponse, error)

	// ConnectionTree returns a driver-specific hierarchy of nodes and actions for
 	// a given connection.  Drivers that do not support browsing can return an
 	// empty Response or an error; the core will treat that as “no tree”.
 	ConnectionTree(*ConnectionTreeRequest) (*ConnectionTreeResponse, error)
}

// ServeCLI runs a Plugin implementation as a small CLI shim that supports
// three commands used by the host: `info`, `exec` and `authforms`.
//
// - `plugin info` prints InfoResponse as JSON to stdout
// - `plugin exec` reads ExecRequest JSON from stdin and writes ExecResponse JSON to stdout
// - `plugin authforms` prints AuthFormsResponse as JSON to stdout
func ServeCLI(p Plugin) {
	args := os.Args[1:]
	if len(args) == 0 {
		usage()
		os.Exit(2)
	}

	switch args[0] {
	case "info":
		info, err := p.Info()
		if err != nil {
			fmt.Fprintf(os.Stderr, "plugin: info error: %v\n", err)
			os.Exit(1)
		}
		b, _ := json.Marshal(info)
		_, _ = os.Stdout.Write(b)
	case "exec":
		in, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "plugin: failed to read stdin: %v\n", err)
			os.Exit(1)
		}
		var req ExecRequest
		if err := json.Unmarshal(in, &req); err != nil {
			fmt.Fprintf(os.Stderr, "plugin: invalid request json: %v\n", err)
			os.Exit(1)
		}
		res, _ := p.Exec(&req)
		b, _ := json.Marshal(res)
		_, _ = os.Stdout.Write(b)
	case "authforms":
		// no stdin input expected; plugins should return available forms
		res, err := p.AuthForms(&AuthFormsRequest{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "plugin: authforms error: %v\n", err)
			os.Exit(1)
		}
		b, _ := json.Marshal(res)
		_, _ = os.Stdout.Write(b)
	case "connection-tree", "tree":
		in, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "plugin: failed to read stdin: %v\n", err)
			os.Exit(1)
		}
		var req ConnectionTreeRequest
		if err := json.Unmarshal(in, &req); err != nil {
			fmt.Fprintf(os.Stderr, "plugin: invalid tree request json: %v\n", err)
			os.Exit(1)
		}
		res, _ := p.ConnectionTree(&req)
		b, _ := json.Marshal(res)
		_, _ = os.Stdout.Write(b)
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: <plugin> info | exec | authforms (request on stdin as JSON)")
}
