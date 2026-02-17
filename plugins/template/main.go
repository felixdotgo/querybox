package main

import (
	"fmt"

	"github.com/felixdotgo/querybox/pkg/plugin"
)

// templatePlugin implements the pkg/plugin.Plugin interface for examples.
type templatePlugin struct{}

func (t *templatePlugin) Info() (plugin.InfoResponse, error) {
	return plugin.InfoResponse{Name: "template", Version: "0.1.0", Description: "Template plugin (on-demand)"}, nil
}

func (t *templatePlugin) Exec(req plugin.ExecRequest) (plugin.ExecResponse, error) {
	// echo the SQL and connection keys for demonstration
	return plugin.ExecResponse{Result: fmt.Sprintf("executed sql: %s | connKeys=%v", req.Sql, req.Connection)}, nil
}

func main() {
	plugin.ServeCLI(&templatePlugin{})
}
