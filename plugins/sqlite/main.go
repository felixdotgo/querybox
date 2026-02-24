package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/felixdotgo/querybox/pkg/plugin"
	pluginpb "github.com/felixdotgo/querybox/rpc/contracts/plugin/v1"

	_ "github.com/tursodatabase/go-libsql"
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
	// Basic: a file path
	basic := plugin.AuthForm{
		Key:  "basic",
		Name: "Basic",
		Fields: []*plugin.AuthField{
			{Type: plugin.AuthFieldFilePath, Name: "file", Label: "Database file path", Required: true, Placeholder: "/path/to/database.db"},
		},
	}

	// turso-cloud: a remote database
	turso := plugin.AuthForm{
		Key:  "turso-cloud",
		Name: "Turso Cloud",
		Fields: []*plugin.AuthField{
			{Type: plugin.AuthFieldText, Name: "database_url", Label: "Database URL", Required: true, Placeholder: "libsql://example.aws-region.turso.io"},
			{Type: plugin.AuthFieldPassword, Name: "token", Label: "Auth Token", Required: true, Placeholder: "your-turso-auth-token"},
		},
	}
	return &plugin.AuthFormsResponse{Forms: map[string]*plugin.AuthForm{"basic": &basic, "turso-cloud": &turso}}, nil
}

type credential struct {
	Form   string            `json:"form"`
	Values map[string]string `json:"values"`
}

func parseCredential(connection map[string]string) credential {
	if blob, ok := connection["credential_blob"]; ok && blob != "" {
		var c credential
		if err := json.Unmarshal([]byte(blob), &c); err == nil {
			return c
		}
	}
	return credential{}
}

func tursoURL(c credential) string {
	url := c.Values["database_url"]
	if url == "" {
		return ""
	}
	if token := c.Values["token"]; token != "" {
		url += "?authToken=" + token
	}
	return url
}

// driverDSN resolves the SQL driver name and DSN from the credential form.
func driverDSN(c credential) (driver, dsn string, err error) {
	if c.Form == "turso-cloud" {
		dsn = tursoURL(c)
		if dsn == "" {
			return "", "", fmt.Errorf("missing database_url in connection")
		}
		return "libsql", dsn, nil
	}
	dsn = c.Values["file"]
	if dsn == "" {
		return "", "", fmt.Errorf("missing file path in connection")
	}
	return "sqlite", dsn, nil
}

func (m *sqlitePlugin) Exec(req *plugin.ExecRequest) (*plugin.ExecResponse, error) {
	c := parseCredential(req.Connection)

	driver, dsn, err := driverDSN(c)
	if err != nil {
		return &plugin.ExecResponse{Error: err.Error()}, nil
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		return &plugin.ExecResponse{Error: fmt.Sprintf("open error: %v", err)}, nil
	}
	defer db.Close()

	// Use Exec for non-SELECT statements (DDL, DML) so they succeed even when
	// they return no rows.  db.Query on a DROP/CREATE would drain silently on
	// some drivers and return a confusing empty-result instead of an error.
	trimmed := strings.TrimSpace(strings.ToUpper(req.Query))
	if !strings.HasPrefix(trimmed, "SELECT") && !strings.HasPrefix(trimmed, "WITH") && !strings.HasPrefix(trimmed, "PRAGMA") {
		if _, execErr := db.Exec(req.Query); execErr != nil {
			return &plugin.ExecResponse{Error: fmt.Sprintf("exec error: %v", execErr)}, nil
		}
		return &plugin.ExecResponse{
			Result: &plugin.ExecResult{
				Payload: &pluginpb.PluginV1_ExecResult_Sql{
					Sql: &plugin.SqlResult{},
				},
			},
		}, nil
	}

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
	c := parseCredential(req.Connection)

	driver, dsn, err := driverDSN(c)
	if err != nil {
		return &plugin.ConnectionTreeResponse{}, nil
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		return &plugin.ConnectionTreeResponse{}, nil
	}
	defer db.Close()

	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
	if err != nil {
		return &plugin.ConnectionTreeResponse{}, nil
	}
	defer rows.Close()

	var tableNodes []*plugin.ConnectionTreeNode
	for rows.Next() {
		var tbl string
		if err := rows.Scan(&tbl); err != nil {
			continue
		}
		tableNodes = append(tableNodes, &plugin.ConnectionTreeNode{
			Key:      tbl,
			Label:    tbl,
			NodeType: "table",
			Actions: []*plugin.ConnectionTreeAction{
				{Type: plugin.ConnectionTreeActionSelect, Title: "Select rows", Query: fmt.Sprintf(`SELECT * FROM "%s" LIMIT 100;`, tbl)},
				{Type: plugin.ConnectionTreeActionDropTable, Title: "Drop table", Query: fmt.Sprintf(`DROP TABLE "%s";`, tbl)},
			},
		})
	}

	// Wrap tables under a root server node that exposes the create-table action.
	serverNode := &plugin.ConnectionTreeNode{
		Key:      "__server__",
		Label:    "Tables",
		NodeType: "server",
		Children: tableNodes,
		Actions: []*plugin.ConnectionTreeAction{
			{
				Type:  plugin.ConnectionTreeActionCreateTable,
				Title: "Create table",
				Query: "CREATE TABLE \"new_table\" (\n    \"id\" INTEGER PRIMARY KEY AUTOINCREMENT\n);",
			},
		},
	}

	return &plugin.ConnectionTreeResponse{Nodes: []*plugin.ConnectionTreeNode{serverNode}}, nil
}

// TestConnection verifies the connection is reachable without persisting any state.
func (m *sqlitePlugin) TestConnection(req *plugin.TestConnectionRequest) (*plugin.TestConnectionResponse, error) {
	c := parseCredential(req.Connection)

	driver, dsn, err := driverDSN(c)
	if err != nil {
		return &plugin.TestConnectionResponse{Ok: false, Message: err.Error()}, nil
	}

	db, err := sql.Open(driver, dsn)
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
