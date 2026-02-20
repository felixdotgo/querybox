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

func (t *templatePlugin) Exec(req plugin.ExecRequest) (plugin.ExecResponse, error) {
	// return a simple key/value map containing the query and connection for demo
	data := map[string]string{"query": req.Query}
	for k, v := range req.Connection {
		data[k] = v
	}
	return plugin.ExecResponse{
		Result: &plugin.ExecResult{
			Payload: &pluginpb.PluginV1_ExecResult_Kv{
				Kv: &plugin.KeyValueResult{
					Data: data,
				},
			},
		},
	}, nil
}

func (t *templatePlugin) AuthForms(plugin.AuthFormsRequest) (plugin.AuthFormsResponse, error) {
	basic := plugin.AuthForm{Key: "basic", Name: "Basic", Fields: []*plugin.AuthField{
		{Type: plugin.AuthField_TEXT, Name: "host", Label: "Host", Required: true, Placeholder: "127.0.0.1"},
		{Type: plugin.AuthField_TEXT, Name: "user", Label: "User"},
		{Type: plugin.AuthField_PASSWORD, Name: "password", Label: "Password"},
	}}
	return plugin.AuthFormsResponse{Forms: map[string]*plugin.AuthForm{"basic": &basic}}, nil
}

func main() {
	plugin.ServeCLI(&templatePlugin{})
}
