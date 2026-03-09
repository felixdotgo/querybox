package main

import (
	"context"
	"testing"

	"github.com/felixdotgo/querybox/pkg/plugin"
	pluginpb "github.com/felixdotgo/querybox/rpc/contracts/plugin/v1"
)

// Basic sanity tests to ensure the template plugin implementation compiles
// and the new MutateRow method behaves as expected.
func TestTemplatePlugin_MutateRow(t *testing.T) {
    p := &templatePlugin{}
    req := &plugin.MutateRowRequest{
        Connection: map[string]string{"foo": "bar"},
        Operation:  pluginpb.PluginV1_MutateRowRequest_UPDATE,
        Source:     "t1",
        Values:     map[string]string{"a": "1"},
        Filter:     "id=1",
    }
    resp, err := p.MutateRow(context.Background(), req)
    if err != nil {
        t.Fatalf("MutateRow returned error: %v", err)
    }
    if !resp.Success {
        t.Errorf("expected success, got %+v", resp)
    }
}
