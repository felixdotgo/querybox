package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"

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
			{Type: plugin.AuthFieldText, Name: "host", Label: "Host", Required: true, Placeholder: "127.0.0.1", Value: "localhost"},
			{Type: plugin.AuthFieldNumber, Name: "port", Label: "Port", Placeholder: "5432", Value: "5432"},
			{Type: plugin.AuthFieldText, Name: "user", Label: "User", Value: "postgres"},
			{Type: plugin.AuthFieldPassword, Name: "password", Label: "Password"},
			{Type: plugin.AuthFieldText, Name: "database", Label: "Database name"},
			// allow tls and extra params similar to mysql
			{Type: plugin.AuthFieldSelect, Name: "tls", Label: "TLS mode (e.g. disable/require)", Options: []string{"disable", "require", "verify-ca", "verify-full"}, Value: "disable"},
			{Type: plugin.AuthFieldText, Name: "params", Label: "Extra params", Placeholder: "connect_timeout=5&application_name=myapp"},
		},
	}

	return &plugin.AuthFormsResponse{Forms: map[string]*plugin.AuthForm{"basic": &basic}}, nil
}

// buildConnString constructs a postgres keyword=value connection string from
// the provided connection map.  Extra DSN parameters are appended as
// space-separated key=value pairs as required by lib/pq; URL-encoded (&)
// format is NOT used because it is invalid for the postgres DSN format.
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
					// The "tls" form field carries a postgres sslmode value
					// (disable / require / verify-ca / verify-full).
					sslmode := payload.Values["tls"]
					if port == "" {
						port = "5432"
					}
					if sslmode == "" {
						sslmode = "disable"
					}
					if host != "" {
						dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
							host, port, user, pass, dbname, sslmode)
					}
				}
				// Append extra postgres DSN params as space-separated key=value
				// pairs.  The "tls", "params", and core credential fields are
				// excluded here because they are handled above or parsed below.
				if dsn != "" {
					skip := map[string]bool{
						"host": true, "user": true, "password": true,
						"port": true, "database": true, "dsn": true,
						"tls": true, "params": true,
					}
					var extra []string
					for k, v := range payload.Values {
						if skip[k] || v == "" {
							continue
						}
						extra = append(extra, fmt.Sprintf("%s=%s", k, v))
					}
					// The "params" field lets users supply additional DSN
					// key=value pairs separated by spaces or "&".
					if raw := payload.Values["params"]; raw != "" {
						for _, part := range strings.FieldsFunc(raw, func(r rune) bool {
							return r == '&' || r == ' '
						}) {
							if kv := strings.SplitN(part, "=", 2); len(kv) == 2 && kv[1] != "" {
								extra = append(extra, fmt.Sprintf("%s=%s", kv[0], kv[1]))
							}
						}
					}
					// Ensure a sensible default connect timeout when the caller
					// has not specified one explicitly.
					hasTimeout := strings.Contains(dsn, "connect_timeout")
					for _, e := range extra {
						if strings.HasPrefix(e, "connect_timeout=") {
							hasTimeout = true
						}
					}
					if !hasTimeout {
						extra = append(extra, "connect_timeout=5")
					}
					if len(extra) > 0 {
						dsn = dsn + " " + strings.Join(extra, " ")
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

// ConnectionTree returns a schema → table hierarchy for the connected database.
// A PostgreSQL connection is scoped to a single database, so we list schemas
// (excluding system schemas) and their tables instead of trying to enumerate
// all server databases.  Errors or missing connection details result in an
// empty response so the core treats the plugin as having no tree support.
func (m *postgresqlPlugin) ConnectionTree(req *plugin.ConnectionTreeRequest) (*plugin.ConnectionTreeResponse, error) {
	dsn, err := buildConnString(req.Connection)
	if err != nil || dsn == "" {
		fmt.Fprintf(os.Stderr, "postgresql: ConnectionTree: DSN error: %v dsn=%q\n", err, dsn)
		return &plugin.ConnectionTreeResponse{}, nil
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "postgresql: ConnectionTree: open error: %v\n", err)
		return &plugin.ConnectionTreeResponse{}, nil
	}
	defer db.Close()

	// List all non-system schemas in the connected database.
	schemaRows, err := db.Query(`
SELECT schema_name
FROM information_schema.schemata
WHERE schema_name NOT IN ('pg_catalog','information_schema','pg_toast')
  AND schema_name NOT LIKE 'pg_%'
ORDER BY schema_name`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "postgresql: ConnectionTree: query schemas error: %v\n", err)
		return &plugin.ConnectionTreeResponse{}, nil
	}
	defer schemaRows.Close()

	var nodes []*plugin.ConnectionTreeNode
	for schemaRows.Next() {
		var schemaName string
		if err := schemaRows.Scan(&schemaName); err != nil {
			continue
		}

		// List base tables within this schema.
		tables := []*plugin.ConnectionTreeNode{}
		tblRows, err := db.Query(`
SELECT table_name
FROM information_schema.tables
WHERE table_schema = $1
  AND table_type = 'BASE TABLE'
ORDER BY table_name`, schemaName)
		if err == nil {
			for tblRows.Next() {
				var tbl string
				if tblRows.Scan(&tbl) == nil {
					tables = append(tables, &plugin.ConnectionTreeNode{
						Key:      schemaName + "." + tbl,
						Label:    tbl,
						NodeType: "table",
						Actions: []*plugin.ConnectionTreeAction{
							{
								Type:  plugin.ConnectionTreeActionSelect,
								Title: fmt.Sprintf("%s.%s", schemaName, tbl),
								Query: fmt.Sprintf(`SELECT * FROM "%s"."%s" LIMIT 100;`, schemaName, tbl),
							},
						},
					})
				}
			}
			tblRows.Close()
		}

		nodes = append(nodes, &plugin.ConnectionTreeNode{
			Key:      schemaName,
			Label:    schemaName,
			NodeType: "schema",
			Children: tables,
		})
	}

	return &plugin.ConnectionTreeResponse{Nodes: nodes}, nil
}

// TestConnection opens a PostgreSQL connection and pings the server to verify
// the supplied credentials are valid. Nothing is persisted.
func (m *postgresqlPlugin) TestConnection(req *plugin.TestConnectionRequest) (*plugin.TestConnectionResponse, error) {
	dsn, err := buildConnString(req.Connection)
	if err != nil || dsn == "" {
		msg := "invalid connection parameters"
		if err != nil {
			msg = err.Error()
		}
		return &plugin.TestConnectionResponse{Ok: false, Message: msg}, nil
	}
	db, err := sql.Open("postgres", dsn)
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
	plugin.ServeCLI(&postgresqlPlugin{})
}
