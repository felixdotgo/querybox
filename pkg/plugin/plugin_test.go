package plugin_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/felixdotgo/querybox/pkg/plugin"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestFormatSQLValue(t *testing.T) {
    tests := []struct {
        name string
        input interface{}
        want string
    }{
        {"nil", nil, ""},
        {"string", "foo", "foo"},
        {"int", 42, "42"},
        {"bool", true, "true"},
        {"float", 3.14, "3.14"},
        {"bytes", []byte("hello"), "hello"},
        {"ascii bytes", []byte{0x41, 0x42, 0x43}, "ABC"},
        {"non-utf8 bytes", []byte{0xff, 0xfe}, "0xfffe"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := plugin.FormatSQLValue(tt.input)
            if got != tt.want {
                t.Errorf("FormatSQLValue(%v) = %q; want %q", tt.input, got, tt.want)
            }
        })
    }
}
// TestServeCLI_DescribeSchema builds a small plugin binary using the
// package helper and exercises the "describe-schema" command.  This
// guards against regressions when ServeCLI is modified.
func TestServeCLI_DescribeSchema(t *testing.T) {
    // create a temporary directory for source and executable
    dir := t.TempDir()
    src := filepath.Join(dir, "main.go")
    bin := filepath.Join(dir, "testplugin")
    if runtime.GOOS == "windows" {
        bin += ".exe"
    }

    // source implements a minimal PluginServiceServer that returns a fixed
    // schema.  We use the same package imports as the real plugins so the
    // compiled binary is representative of production.
    const program = `package main

import (
    "context"

    "github.com/felixdotgo/querybox/pkg/plugin"
    pluginpb "github.com/felixdotgo/querybox/rpc/contracts/plugin/v1"
)

type server struct {
    pluginpb.UnimplementedPluginServiceServer
}

func (s *server) DescribeSchema(ctx context.Context, req *pluginpb.PluginV1_DescribeSchemaRequest) (*pluginpb.PluginV1_DescribeSchemaResponse, error) {
    // ignore req contents; return a single table entry
    return &pluginpb.PluginV1_DescribeSchemaResponse{
        Tables: []*pluginpb.PluginV1_TableSchema{{Name: "t1"}},
    }, nil
}

func (s *server) Info(ctx context.Context, _ *pluginpb.PluginV1_InfoRequest) (*plugin.InfoResponse, error) {
    return &plugin.InfoResponse{Type: plugin.TypeDriver}, nil
}

func main() {
    plugin.ServeCLI(&server{})
}
`

    if err := os.WriteFile(src, []byte(program), 0o644); err != nil {
        t.Fatalf("write source: %v", err)
    }

    // build the test plugin
    cmd := exec.Command("go", "build", "-o", bin, src)
    if out, err := cmd.CombinedOutput(); err != nil {
        t.Fatalf("go build failed: %v\n%s", err, string(out))
    }

    // prepare input JSON for describe-schema
    req := plugin.DescribeSchemaRequest{Connection: map[string]string{"foo": "bar"}}
    in, _ := json.Marshal(&req)

    cmd = exec.Command(bin, "describe-schema")
    cmd.Stdin = bytes.NewReader(in)
    out, err := cmd.CombinedOutput()
    if err != nil {
        t.Fatalf("plugin exited with error: %v\nstderr+stdout:\n%s", err, string(out))
    }

    var resp plugin.DescribeSchemaResponse
    if err := protojson.Unmarshal(out, &resp); err != nil {
        t.Fatalf("unmarshal response: %v", err)
    }
    if len(resp.Tables) != 1 || resp.Tables[0].Name != "t1" {
        t.Errorf("unexpected schema response: %+v", resp)
    }
}