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

const (
	TypeDriver DriverType = pluginpb.PluginV1_DRIVER
)

// Plugin describes the minimal contract a plugin should implement. Keeping
// this small and explicit makes it easy to implement either an in-process
// plugin or an on-demand executable that uses ServeCLI below.
type Plugin interface {
	Info() (InfoResponse, error)
	Exec(ExecRequest) (ExecResponse, error)
}

// ServeCLI runs a Plugin implementation as a small CLI shim that supports
// two commands used by the host: `info` and `exec`.
//
// - `plugin info` prints InfoResponse as JSON to stdout
// - `plugin exec` reads ExecRequest JSON from stdin and writes ExecResponse JSON to stdout
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
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: <plugin> info | exec (request on stdin as JSON)")
}
