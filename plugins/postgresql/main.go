package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/felixdotgo/querybox/pkg/plugin"
	pluginpb "github.com/felixdotgo/querybox/rpc/contracts/plugin/v1"

	_ "github.com/lib/pq" // postgres driver
)

// postgresqlPlugin implements the plugin.Plugin interface for a simple PostgreSQL executor.
type postgresqlPlugin struct{}

func (m *postgresqlPlugin) Info() (plugin.InfoResponse, error) {
	return plugin.InfoResponse{
		Type:        plugin.TypeDriver,
		Name:        "PostgreSQL",
		Version:     "0.1.0",
		Description: "PostgreSQL database driver",
	}, nil
}

func (m *postgresqlPlugin) AuthForms(*plugin.AuthFormsRequest) (*plugin.AuthFormsResponse, error) {
	// Provide two options: a `basic` property-based form and a `dsn` fallback.
	basic := plugin.AuthForm{
		Key: "basic",
		Name: "Basic",
		Fields: []*plugin.AuthField{
			{Type: plugin.AuthFieldText, Name: "host", Label: "Host", Required: true, Placeholder: "127.0.0.1"},
			{Type: plugin.AuthFieldNumber, Name: "port", Label: "Port", Placeholder: "5432"},
			{Type: plugin.AuthFieldText, Name: "user", Label: "User"},
			{Type: plugin.AuthFieldPassword, Name: "password", Label: "Password"},
			{Type: plugin.AuthFieldText, Name: "database", Label: "Database name"},
			// allow tls and extra params similar to mysql
			{Type: plugin.AuthFieldText, Name: "tls", Label: "TLS mode (e.g. disable/require)"},
			{Type: plugin.AuthFieldText, Name: "params", Label: "Extra params", Placeholder: "connect_timeout=5&application_name=myapp"},
		},
	}

	return &plugin.AuthFormsResponse{Forms: map[string]*plugin.AuthForm{"basic": &basic}}, nil
}

// buildConnString constructs a postgres connection string from the provided
// connection map.  Reuses the same rules used by Exec and the connection tree
// logic.
func buildConnString(connection map[string]string) (string, error) {
	dsn, ok := connection["dsn"]
	if !ok || dsn == "" {
		if blob, ok2 := connection["credential_blob"]; ok2 && blob != "" {
			var payload struct {
				Form   string            `json:"form"`
				Values map[string]string `json:"values"`
			}
			if err := json.Unmarshal([]byte(blob), &payload); err == nil {
				if v, ok := payload.Values["dsn"]; ok && v != "" {
					dsn = v
				} else {
					host := payload.Values["host"]
					user := payload.Values["user"]
					pass := payload.Values["password"]
					port := payload.Values["port"]
					dbname := payload.Values["database"]
					if port == "" {
						port = "5432"
					}
					if host != "" {
						dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, pass, dbname)
					}
				}
				// append extra params analogous to mysql buildDSN
				if dsn != "" {
					params := url.Values{}
					for k, v := range payload.Values {
						switch k {
						case "host", "user", "password", "port", "database", "dsn":
							continue
						}
						if v != "" {
							params.Add(k, v)
						}
					}
					if params.Get("connect_timeout") == "" {
						params.Set("connect_timeout", "5")
					}
					if len(params) > 0 {
						dsn = dsn + " " + params.Encode()
					}
				}
			}
		}
	}
	return dsn, nil
}

func (m *postgresqlPlugin) Exec(req *plugin.ExecRequest) (*plugin.ExecResponse, error) {
	dsn, err := buildConnString(req.Connection)
	if err != nil {
		return &plugin.ExecResponse{Error: fmt.Sprintf("invalid connection: %v", err)}, nil
	}
	if dsn == "" {
		return &plugin.ExecResponse{Error: "missing dsn in connection"}, nil
	}

	// open postgres driver
	db, err := sql.Open("postgres", dsn)
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
					Rows: rowResults,
				},
			},
		},
	}, nil
}

// ConnectionTree returns a simple list of non-template databases.  The host
// can display the names and optionally invoke the provided action if the
// user requests it.  Errors or missing connection details result in an empty
// response so the core treats the plugin as having no tree support.
func (m *postgresqlPlugin) ConnectionTree(req *plugin.ConnectionTreeRequest) (*plugin.ConnectionTreeResponse, error) {
	dsn, err := buildConnString(req.Connection)
	if err != nil || dsn == "" {
		return &plugin.ConnectionTreeResponse{}, nil
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return &plugin.ConnectionTreeResponse{}, nil
	}
	defer db.Close()

	rows, err := db.Query("SELECT datname FROM pg_database WHERE datistemplate = false")
	if err != nil {
		return &plugin.ConnectionTreeResponse{}, nil
	}
	defer rows.Close()

	var nodes []*plugin.ConnectionTreeNode
	for rows.Next() {
		var dbname string
		if err := rows.Scan(&dbname); err != nil {
			continue
		}
		// fetch tables for this db; note the DSN may already target a specific
		// database so querying across databases may not work, but this is
		// illustrative.
		tables := []*plugin.ConnectionTreeNode{}
		tblRows, err := db.Query(`
SELECT tablename
FROM pg_catalog.pg_tables
WHERE schemaname NOT IN ('pg_catalog','information_schema')
`)
		if err == nil {
			for tblRows.Next() {
				var tbl string
				if tblRows.Scan(&tbl) == nil {
					tables = append(tables, &plugin.ConnectionTreeNode{
					Key:   dbname + "." + tbl,
					Label: tbl,
					Actions: []*plugin.ConnectionTreeAction{
						{Type: plugin.ConnectionTreeActionSelect, Title: "Select", Query: fmt.Sprintf("SELECT * FROM \"%s\".%s LIMIT 100;", dbname, tbl)},
					},
				})
				}
			}
			tblRows.Close()
		}
		nodes = append(nodes, &plugin.ConnectionTreeNode{
			Key:   dbname,
			Label: dbname,
			Children: tables,
			Actions: []*plugin.ConnectionTreeAction{
				{Type: plugin.ConnectionTreeActionSelect, Title: "Query", Query: "SELECT 1"},
			},
		})
	}

	return &plugin.ConnectionTreeResponse{Nodes: nodes}, nil
}

func main() {
	plugin.ServeCLI(&postgresqlPlugin{})
}
