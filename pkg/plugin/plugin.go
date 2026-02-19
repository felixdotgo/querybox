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
type ExecResponse = pluginpb.PluginV1_ExecResponse
type DriverType = pluginpb.PluginV1_Type

// Aliases for the new AuthForms protobuf messages
type AuthField = pluginpb.PluginV1_AuthField
type AuthForm = pluginpb.PluginV1_AuthForm
type AuthFormsRequest = pluginpb.PluginV1_AuthFormsRequest
type AuthFormsResponse = pluginpb.PluginV1_AuthFormsResponse

const (
	TypeDriver DriverType = pluginpb.PluginV1_DRIVER

	AuthField_TEXT     = pluginpb.PluginV1_AuthField_TEXT
	AuthField_NUMBER   = pluginpb.PluginV1_AuthField_NUMBER
	AuthField_PASSWORD = pluginpb.PluginV1_AuthField_PASSWORD
	AuthField_SELECT   = pluginpb.PluginV1_AuthField_SELECT
	AuthField_CHECKBOX = pluginpb.PluginV1_AuthField_CHECKBOX
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
	Exec(ExecRequest) (ExecResponse, error)

	// AuthForms returns available authentication forms the plugin supports.
	// Optional for existing plugins — implementations may return an empty map.
	AuthForms(AuthFormsRequest) (AuthFormsResponse, error)
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
		res, _ := p.Exec(req)
		b, _ := json.Marshal(res)
		_, _ = os.Stdout.Write(b)
	case "authforms":
		// no stdin input expected; plugins should return available forms
		res, err := p.AuthForms(AuthFormsRequest{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "plugin: authforms error: %v\n", err)
			os.Exit(1)
		}
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
