package main

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/felixdotgo/querybox/pkg/plugin"
	pluginpb "github.com/felixdotgo/querybox/rpc/contracts/plugin/v1"

	_ "modernc.org/sqlite"
)

// sqlitePlugin implements the plugin.Plugin interface for a simple SQLite executor.
type sqlitePlugin struct{}

func (m *sqlitePlugin) Info() (plugin.InfoResponse, error) {
	return plugin.InfoResponse{
		Type:        plugin.TypeDriver,
		Name:        "SQLite",
		Version:     "0.1.0",
		Description: "SQLite database driver",
	}, nil
}

func (m *sqlitePlugin) AuthForms(*plugin.AuthFormsRequest) (*plugin.AuthFormsResponse, error) {
	// only support a single form for now
	basic := plugin.AuthForm{
		Key:  "basic",
		Name: "Basic",
		Fields: []*plugin.AuthField{
			{Type: plugin.AuthFieldText, Name: "file", Label: "Database file path", Required: true, Placeholder: "/path/to/database.db"},
		},
	}
	return &plugin.AuthFormsResponse{Forms: map[string]*plugin.AuthForm{"basic": &basic}}, nil
}

func filePath(connection map[string]string) string {
	if blob, ok := connection["credential_blob"]; ok && blob != "" {
		var payload struct {
			Form   string            `json:"form"`
			Values map[string]string `json:"values"`
		}
		if err := json.Unmarshal([]byte(blob), &payload); err == nil {
			return payload.Values["file"]
		}
	}
	return ""
}

func (m *sqlitePlugin) Exec(req *plugin.ExecRequest) (*plugin.ExecResponse, error) {
	path := filePath(req.Connection)
	if path == "" {
		return &plugin.ExecResponse{Error: "missing file path in connection"}, nil
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return &plugin.ExecResponse{Error: fmt.Sprintf("open error: %v", err)}, nil
	}
	defer db.Close()

	rows, err := db.Query(req.Query)
	if err != nil {
		return &plugin.ExecResponse{Error: fmt.Sprintf("query error: %v", err)}, nil
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return &plugin.ExecResponse{Error: fmt.Sprintf("cols error: %v", err)}, nil
	}

	colMeta := make([]*plugin.Column, len(cols))
	for i, c := range cols {
		colMeta[i] = &plugin.Column{Name: c}
	}

	var rowResults []*plugin.Row
	for rows.Next() {
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return &plugin.ExecResponse{Error: fmt.Sprintf("scan error: %v", err)}, nil
		}
		strs := make([]string, len(cols))
		for i, v := range vals {
			strs[i] = plugin.FormatSQLValue(v)
		}
		rowResults = append(rowResults, &plugin.Row{Values: strs})
	}

	return &plugin.ExecResponse{
		Result: &plugin.ExecResult{
			Payload: &pluginpb.PluginV1_ExecResult_Sql{
				Sql: &plugin.SqlResult{
					Columns: colMeta,
					Rows:    rowResults,
				},
			},
		},
	}, nil
}

// ConnectionTree returns a list of tables in the SQLite database.
func (m *sqlitePlugin) ConnectionTree(req *plugin.ConnectionTreeRequest) (*plugin.ConnectionTreeResponse, error) {
	path := filePath(req.Connection)
	if path == "" {
		return &plugin.ConnectionTreeResponse{}, nil
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return &plugin.ConnectionTreeResponse{}, nil
	}
	defer db.Close()

	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
	if err != nil {
		return &plugin.ConnectionTreeResponse{}, nil
	}
	defer rows.Close()

	var nodes []*plugin.ConnectionTreeNode
	for rows.Next() {
		var tbl string
		if err := rows.Scan(&tbl); err != nil {
			continue
		}
		nodes = append(nodes, &plugin.ConnectionTreeNode{
			Key:      tbl,
			Label:    tbl,
			NodeType: "table",
			Actions: []*plugin.ConnectionTreeAction{
				{Type: plugin.ConnectionTreeActionSelect, Title: tbl, Query: fmt.Sprintf(`SELECT * FROM "%s" LIMIT 100;`, tbl)},
			},
		})
	}

	return &plugin.ConnectionTreeResponse{Nodes: nodes}, nil
}

// TestConnection opens a SQLite file and pings the handle to verify the file
// path is accessible. Nothing is persisted (SQLite creates the file on open,
// but the caller's path must exist for a database-backed connection).
func (m *sqlitePlugin) TestConnection(req *plugin.TestConnectionRequest) (*plugin.TestConnectionResponse, error) {
	path := filePath(req.Connection)
	if path == "" {
		return &plugin.TestConnectionResponse{Ok: false, Message: "missing file path in connection"}, nil
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return &plugin.TestConnectionResponse{Ok: false, Message: fmt.Sprintf("open error: %v", err)}, nil
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		return &plugin.TestConnectionResponse{Ok: false, Message: fmt.Sprintf("ping error: %v", err)}, nil
	}
	return &plugin.TestConnectionResponse{Ok: true, Message: "Connection successful"}, nil
}

func main() {
	plugin.ServeCLI(&sqlitePlugin{})
}
