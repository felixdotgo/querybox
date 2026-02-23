package main

import (
	"github.com/felixdotgo/querybox/pkg/plugin"
	pluginpb "github.com/felixdotgo/querybox/rpc/contracts/plugin/v1"
)

// templatePlugin implements the pkg/plugin.Plugin interface for examples.
type templatePlugin struct{}

func (t *templatePlugin) Info() (plugin.InfoResponse, error) {
	return plugin.InfoResponse{Name: "template", Version: "0.1.0", Description: "Template plugin (on-demand)"}, nil
}

func (t *templatePlugin) Exec(req *plugin.ExecRequest) (*plugin.ExecResponse, error) {
	// return a simple key/value map containing the query and connection for demo
	data := map[string]string{"query": req.Query}
	for k, v := range req.Connection {
		data[k] = v
	}
	return &plugin.ExecResponse{
		Result: &plugin.ExecResult{
			Payload: &pluginpb.PluginV1_ExecResult_Kv{
				Kv: &plugin.KeyValueResult{
					Data: data,
				},
			},
		},
	}, nil
}

func (t *templatePlugin) AuthForms(*plugin.AuthFormsRequest) (*plugin.AuthFormsResponse, error) {
	basic := plugin.AuthForm{Key: "basic", Name: "Basic", Fields: []*plugin.AuthField{
		{Type: plugin.AuthFieldText, Name: "host", Label: "Host", Required: true, Placeholder: "127.0.0.1"},
		{Type: plugin.AuthFieldText, Name: "user", Label: "User"},
		{Type: plugin.AuthFieldPassword, Name: "password", Label: "Password"},
	}}
	return &plugin.AuthFormsResponse{Forms: map[string]*plugin.AuthForm{"basic": &basic}}, nil
}

// ConnectionTree returns a trivial tree for demonstration purposes.  In a
// real plugin the structure would be derived from the connection (e.g. list of
// databases/tables).
func (t *templatePlugin) ConnectionTree(req *plugin.ConnectionTreeRequest) (*plugin.ConnectionTreeResponse, error) {
	return &plugin.ConnectionTreeResponse{
		Nodes: []*plugin.ConnectionTreeNode{
			{
				Key:   "dummy",
				Label: "Dummy node",
				Actions: []*plugin.ConnectionTreeAction{
					{Type: plugin.ConnectionTreeActionSelect, Title: "Echo query", Query: "SELECT 1"},
				},
			},
		},
	}, nil
}

// ConnectionTreeAction simply echoes back the action's query for demo purposes.
// In a real plugin this would execute the query and return results or perform some other side effect.
func (t *templatePlugin) ConnectionTreeAction(req *plugin.ConnectionTreeAction) (*plugin.ExecResponse, error) {
	// simply echo back the action's query for demo purposes
	data := map[string]string{"action_query": req.Query}
	return &plugin.ExecResponse{
		Result: &plugin.ExecResult{
			Payload: &pluginpb.PluginV1_ExecResult_Kv{
				Kv: &plugin.KeyValueResult{
					Data: data,
				},
			},
		},
	}, nil
}

// TestConnection always succeeds for the template plugin. Real plugins should
// open the data store and verify credentials.
func (t *templatePlugin) TestConnection(req *plugin.TestConnectionRequest) (*plugin.TestConnectionResponse, error) {
	return &plugin.TestConnectionResponse{Ok: true, Message: "Connection successful (template stub)"}, nil
}

func main() {
	plugin.ServeCLI(&templatePlugin{})
}
