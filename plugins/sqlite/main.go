package main

import (
	"context"
	"database/sql"
	"fmt"
	"runtime"
	"strings"

	"github.com/felixdotgo/querybox/pkg/plugin"
	pluginpb "github.com/felixdotgo/querybox/rpc/contracts/plugin/v1"

	_ "modernc.org/sqlite"
)

// sqlitePlugin implements the protobuf-generated PluginServiceServer interface.
// embedding the unimplemented struct ensures forward compatibility when new
// methods are added to the service definition.
type sqlitePlugin struct {
	pluginpb.UnimplementedPluginServiceServer
}

func (m *sqlitePlugin) Info(ctx context.Context, _ *pluginpb.PluginV1_InfoRequest) (*plugin.InfoResponse, error) {
	return &plugin.InfoResponse{
		Type:        plugin.TypeDriver,
		Name:        "SQLite",
		Version:     "0.1.0",
		Description: "SQLite database driver",
		Url:         "https://www.sqlite.org/",
		Author:      "SQLite Consortium",
		Capabilities: []string{"query", "explain-query", "mutate-row"},
		Tags:        []string{"sql", "relational"},
		License:     "Public Domain",
		IconUrl:     "https://www.sqlite.org/images/logo-square.jpg",
	}, nil
}

func (m *sqlitePlugin) AuthForms(ctx context.Context, _ *plugin.AuthFormsRequest) (*plugin.AuthFormsResponse, error) {
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
	// if OS is windows, not return turso-cloud form, because libsql driver does not support windows yet.
	if strings.Contains(strings.ToLower(runtime.GOOS), "windows") {
		return &plugin.AuthFormsResponse{Forms: map[string]*plugin.AuthForm{"basic": &basic}}, nil
	}
	return &plugin.AuthFormsResponse{Forms: map[string]*plugin.AuthForm{"basic": &basic, "turso-cloud": &turso}}, nil
}

func parseCredential(connection map[string]string) plugin.CredentialBlob {
	cred, err := plugin.ParseCredentialBlob(connection)
	if err != nil {
		return plugin.CredentialBlob{}
	}
	return cred
}

func tursoURL(c plugin.CredentialBlob) string {
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
func driverDSN(c plugin.CredentialBlob) (driver, dsn string, err error) {
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

func (m *sqlitePlugin) Exec(ctx context.Context, req *plugin.ExecRequest) (*plugin.ExecResponse, error) {
	// honour explain-request flag by prefixing the query; plugins may
	// interpret this differently but most SQL drivers simply prepend
	// "EXPLAIN".
	if req.Options != nil {
		if v, ok := req.Options["explain-query"]; ok && v == "yes" {
			req.Query = "EXPLAIN " + req.Query
		}
	}

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
func (m *sqlitePlugin) ConnectionTree(ctx context.Context, req *plugin.ConnectionTreeRequest) (*plugin.ConnectionTreeResponse, error) {
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
			NodeType: plugin.ConnectionTreeNodeTypeTable,
			Actions: []*plugin.ConnectionTreeAction{
				{Type: plugin.ConnectionTreeActionSelect, Title: "Select rows", Query: fmt.Sprintf(`SELECT * FROM "%s"`, tbl), Hidden: true, NewTab: true},
				{Type: plugin.ConnectionTreeActionDropTable, Title: "Drop table", Query: fmt.Sprintf(`DROP TABLE "%s";`, tbl)},
			},
		})
	}

	// Prepend a leaf node for the create-table action so the user can
	// create a new table without a redundant wrapper server node.
	createNode := &plugin.ConnectionTreeNode{
		Key:      "__create_table__",
		Label:    "New table",
		NodeType: plugin.ConnectionTreeNodeTypeAction,
		Actions: []*plugin.ConnectionTreeAction{
			{
				Type:  plugin.ConnectionTreeActionCreateTable,
				Title: "Create table",
				Query: "CREATE TABLE \"new_table\" (\n    \"id\" INTEGER PRIMARY KEY AUTOINCREMENT\n);",
				Hidden: true, // hide the action from the UI since it doesn't work out-of-the-box and requires user editing
			},
		},
	}

	return &plugin.ConnectionTreeResponse{Nodes: append([]*plugin.ConnectionTreeNode{createNode}, tableNodes...)}, nil
}

// DescribeSchema returns column/index metadata for one or more tables.
func (m *sqlitePlugin) DescribeSchema(ctx context.Context, req *plugin.DescribeSchemaRequest) (*plugin.DescribeSchemaResponse, error) {
    c := parseCredential(req.Connection)
    driver, dsn, err := driverDSN(c)
    if err != nil {
        return &plugin.DescribeSchemaResponse{}, nil
    }
    db, err := sql.Open(driver, dsn)
    if err != nil {
        return &plugin.DescribeSchemaResponse{}, nil
    }
    defer db.Close()

    var tables []string
    if req.Table != "" {
        tables = []string{req.Table}
    } else {
        rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
        if err == nil {
            defer rows.Close()
            for rows.Next() {
                var tbl string
                if rows.Scan(&tbl) == nil {
                    tables = append(tables, tbl)
                }
            }
        }
    }

    resp := &plugin.DescribeSchemaResponse{}
    for _, tbl := range tables {
        ts := &plugin.TableSchema{Name: tbl}
        // columns
        colRows, err := db.Query(fmt.Sprintf("PRAGMA table_info('%s')", tbl))
        if err == nil {
            defer colRows.Close()
            for colRows.Next() {
                var cid int
                var name, ctype string
                var notnull, pk int
                var dflt sql.NullString
                if err := colRows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
                    continue
                }
                cs := &plugin.ColumnSchema{
                    Name:       name,
                    Type:       ctype,
                    Nullable:   notnull == 0,
                    PrimaryKey: pk != 0,
                    Ordinal:    int32(cid),
                }
                if dflt.Valid {
                    cs.Default = dflt.String
                }
                ts.Columns = append(ts.Columns, cs)
            }
        }
        // indexes
        idxRows, err := db.Query(fmt.Sprintf("PRAGMA index_list('%s')", tbl))
        if err == nil {
            defer idxRows.Close()
            for idxRows.Next() {
                var seq int
                var name string
                var unique int
                var origin string
                var partial int
                if err := idxRows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
                    continue
                }
                idx := &plugin.IndexSchema{Name: name, Unique: unique != 0, Primary: origin == "pk"}
                // fetch columns for this index
                infoRows, err := db.Query(fmt.Sprintf("PRAGMA index_info('%s')", name))
                if err == nil {
                    defer infoRows.Close()
                    for infoRows.Next() {
                        var seqno, cid int
                        var colname string
                        if infoRows.Scan(&seqno, &cid, &colname) == nil {
                            idx.Columns = append(idx.Columns, colname)
                        }
                    }
                }
                ts.Indexes = append(ts.Indexes, idx)
            }
        }
        resp.Tables = append(resp.Tables, ts)
    }
    return resp, nil
}

// TestConnection verifies the connection is reachable without persisting any state.
// GetCompletionFields returns column names and types for the given table,
// enabling context-aware auto-completion in the editor.
func (m *sqlitePlugin) GetCompletionFields(ctx context.Context, req *plugin.GetCompletionFieldsRequest) (*plugin.GetCompletionFieldsResponse, error) {
	if req.Collection == "" {
		return &plugin.GetCompletionFieldsResponse{}, nil
	}
	c := parseCredential(req.Connection)
	driver, dsn, err := driverDSN(c)
	if err != nil {
		return &plugin.GetCompletionFieldsResponse{}, nil
	}
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return &plugin.GetCompletionFieldsResponse{}, nil
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info('%s')", req.Collection))
	if err != nil {
		return &plugin.GetCompletionFieldsResponse{}, nil
	}
	defer rows.Close()

	resp := &plugin.GetCompletionFieldsResponse{}
	for rows.Next() {
		var cid, notnull, pk int
		var name, colType string
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &colType, &notnull, &dflt, &pk); err != nil {
			continue
		}
		resp.Fields = append(resp.Fields, &plugin.FieldInfo{Name: name, Type: colType})
	}
	return resp, nil
}

// MutateRow implements the optional mutation RPC for sqlite.  This stub
// simply returns success and does not modify the database.  Real drivers
// could open a connection and execute an INSERT/UPDATE/DELETE derived
// from the parameters.
func (m *sqlitePlugin) MutateRow(ctx context.Context, req *plugin.MutateRowRequest) (*plugin.MutateRowResponse, error) {
	return &plugin.MutateRowResponse{Success: true}, nil
}

func (m *sqlitePlugin) TestConnection(ctx context.Context, req *plugin.TestConnectionRequest) (*plugin.TestConnectionResponse, error) {
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
