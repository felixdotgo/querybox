package main

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/felixdotgo/querybox/pkg/certs"
	"github.com/felixdotgo/querybox/pkg/plugin"
	pluginpb "github.com/felixdotgo/querybox/rpc/contracts/plugin/v1"

	"github.com/go-sql-driver/mysql"
)

// mysqlPlugin implements the protobuf PluginServiceServer interface for a simple MySQL executor.
type mysqlPlugin struct {
	pluginpb.UnimplementedPluginServiceServer
}

func (m *mysqlPlugin) Info(ctx context.Context, _ *pluginpb.PluginV1_InfoRequest) (*plugin.InfoResponse, error) {
	return &plugin.InfoResponse{
		Type:        plugin.TypeDriver,
		Name:        "MySQL",
		Version:     "0.1.0",
		Description: "MySQL database driver",
		Url:         "https://www.mysql.com/",
		Author:      "Oracle",
		Capabilities: []string{"query", "explain-query", "mutate-row", "describe-schema"},
		Tags:        []string{"sql", "relational"},
		License:     "GPL-2.0",
		IconUrl:     "https://www.mysql.com/common/logos/logo-mysql-170x115.png",
	}, nil
}

func (m *mysqlPlugin) AuthForms(ctx context.Context, _ *plugin.AuthFormsRequest) (*plugin.AuthFormsResponse, error) {
	// Provide two options: a `basic` property-based form and a `dsn` fallback.
	basic := plugin.AuthForm{
		Key:  "basic",
		Name: "Basic",
		Fields: []*plugin.AuthField{
			{Type: plugin.AuthFieldText, Name: "host", Label: "Host", Required: true, Placeholder: "127.0.0.1", Value: "127.0.0.1"},
			{Type: plugin.AuthFieldNumber, Name: "port", Label: "Port", Placeholder: "3306", Value: "3306"},
			{Type: plugin.AuthFieldText, Name: "user", Label: "User", Value: "root"},
			{Type: plugin.AuthFieldPassword, Name: "password", Label: "Password"},
			{Type: plugin.AuthFieldText, Name: "database", Label: "Database name"},
			// allow users to specify extra params such as tls=skip-verify
			{Type: plugin.AuthFieldSelect, Name: "tls", Label: "TLS mode (e.g. skip-verify)", Options: []string{"skip-verify", "true", "false", "preferred"}, Value: "skip-verify"},
			{Type: plugin.AuthFieldText, Name: "params", Label: "Extra params", Placeholder: "charset=utf8&parseTime=true"},
		},
	}

	dsn := plugin.AuthForm{
		Key:  "dsn",
		Name: "DSN",
		Fields: []*plugin.AuthField{
			{Type: plugin.AuthFieldText, Name: "dsn", Label: "DSN", Placeholder: "user:pass@tcp(host:port)/dbname"},
		},
	}
	return &plugin.AuthFormsResponse{Forms: map[string]*plugin.AuthForm{"basic": &basic, "dsn": &dsn}}, nil
}

// buildDSN constructs a mysql DSN from the provided connection map.  The
// logic mirrors what Exec historically did so both execution and browsing can
// reuse the same rules (dsn value or credential_blob JSON).
func init() {
    // register a TLS config using our embedded root certificates; callers
    // can select it via tls=querybox in the DSN.
    if pool, err := certs.RootCertPool(); err == nil {
        mysql.RegisterTLSConfig("querybox", &tls.Config{RootCAs: pool})
    }
}

func buildDSN(connection map[string]string) (string, error) {
    // Accept either a full DSN under key "dsn" (legacy) or a credential blob
    // JSON (recommended) stored under "credential_blob" containing: {"form":"basic","values": { ... }}
    // Additionally we allow arbitrary extra parameters (including tls) which
    // are appended as query parameters.  This lets callers configure SSL
    // (tls=skip-verify, tls=true, etc) or other driver options.
    dsn, ok := connection["dsn"]
    if !ok || dsn == "" {
        // try credential_blob
        if cred, err := plugin.ParseCredentialBlob(connection); err == nil {
                // if plugin stored a dsn inside values, prefer that
                if v, ok := cred.Values["dsn"]; ok && v != "" {
                    dsn = v
                } else {
                    // build a simple DSN from common keys
                    host := cred.Values["host"]
                    user := cred.Values["user"]
                    pass := cred.Values["password"]
                    port := cred.Values["port"]
                    dbname := cred.Values["database"]
                    if port == "" {
                        port = "3306"
                    }
                    if host != "" {
                        dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", user, pass, host, port, dbname)
                    }
                }
                // append any extra parameters as query string
                if dsn != "" {
                    params := url.Values{}
                    for k, v := range cred.Values {
                        switch k {
                        case "host", "user", "password", "port", "database", "dsn":
                            // already handled above
                            continue
                        }
                        if v != "" {
                            params.Add(k, v)
                        }
                    }
                    // convert generic tls flags to our registered config
                    if t := params.Get("tls"); t == "true" || t == "preferred" {
                        params.Set("tls", "querybox")
                    }
                    if len(params) > 0 {
                        // ensure we always have a reasonable connection timeout so the
                        // plugin can't hang indefinitely (30s context is managed by
                        // caller).  driver accepts values like "5s".
                        if params.Get("timeout") == "" {
                            params.Set("timeout", "5s")
                        }
                        sep := "?"
                        if strings.Contains(dsn, "?") {
                            sep = "&"
                        }
                        dsn = dsn + sep + params.Encode()
                    }
                }
        }
    }
    // If the caller forwarded a specific database (e.g. from the tree-node
    // context), override the DBName in the parsed DSN so the connection opens
    // against the correct database regardless of the saved credential.
    if dbOverride, ok := connection["database"]; ok && dbOverride != "" && dsn != "" {
        if cfg, err2 := mysql.ParseDSN(dsn); err2 == nil {
            cfg.DBName = dbOverride
            if rebuilt := cfg.FormatDSN(); rebuilt != "" {
                dsn = rebuilt
            }
        }
    }
    return dsn, nil
}

// getDatabaseFromConn returns the database name the connection will use, or
// an empty string if none was provided explicitly.  This is used by
// ConnectionTree to decide whether to restrict the returned node list.
func getDatabaseFromConn(connection map[string]string) string {
	dsn, _ := buildDSN(connection)
	if dsn == "" {
		return ""
	}
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return ""
	}
	return cfg.DBName
}

func (m *mysqlPlugin) DescribeSchema(ctx context.Context, req *plugin.DescribeSchemaRequest) (*plugin.DescribeSchemaResponse, error) {
    dsn, err := buildDSN(req.Connection)
    if err != nil {
        return &plugin.DescribeSchemaResponse{}, nil
    }
    db, err := sql.Open("mysql", dsn)
    if err != nil {
        return &plugin.DescribeSchemaResponse{}, nil
    }
    defer db.Close()

    resp := &plugin.DescribeSchemaResponse{}

    // fetch tables matching optional filters
    query := "SELECT TABLE_SCHEMA, TABLE_NAME FROM information_schema.TABLES WHERE TABLE_TYPE='BASE TABLE'"
    args := []interface{}{}
    if req.Database != "" {
        query += " AND TABLE_SCHEMA = ?"
        args = append(args, req.Database)
    }
    if req.Table != "" {
        query += " AND TABLE_NAME = ?"
        args = append(args, req.Table)
    }
    rows, err := db.Query(query, args...)
    if err != nil {
        return resp, nil
    }
    defer rows.Close()

    for rows.Next() {
        var schema, tbl string
        if rows.Scan(&schema, &tbl) != nil {
            continue
        }
        ts := &plugin.TableSchema{Name: fmt.Sprintf("%s.%s", schema, tbl)}
        // columns
        colQ := `SELECT COLUMN_NAME, COLUMN_TYPE, IS_NULLABLE, COLUMN_KEY='PRI', ORDINAL_POSITION, COLUMN_DEFAULT
                   FROM information_schema.COLUMNS
                   WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? ORDER BY ORDINAL_POSITION`
        colRows, err := db.Query(colQ, schema, tbl)
        if err == nil {
            defer colRows.Close()
            for colRows.Next() {
                var name, ctype, isNull sql.NullString
                var pk bool
                var pos int32
                var def sql.NullString
                if err := colRows.Scan(&name, &ctype, &isNull, &pk, &pos, &def); err != nil {
                    continue
                }
                cs := &plugin.ColumnSchema{
                    Name:       name.String,
                    Type:       ctype.String,
                    Nullable:   strings.EqualFold(isNull.String, "YES"),
                    PrimaryKey: pk,
                    Ordinal:    pos,
                }
                if def.Valid {
                    cs.Default = def.String
                }
                ts.Columns = append(ts.Columns, cs)
            }
        }
        // indexes
        idxQ := `SELECT INDEX_NAME, COLUMN_NAME, NON_UNIQUE, SEQ_IN_INDEX, INDEX_COMMENT, INDEX_TYPE
                  FROM information_schema.STATISTICS
                  WHERE TABLE_SCHEMA=? AND TABLE_NAME=? ORDER BY INDEX_NAME, SEQ_IN_INDEX`
        idxRows, err := db.Query(idxQ, schema, tbl)
        if err == nil {
            defer idxRows.Close()
            var current *plugin.IndexSchema
            lastName := ""
            for idxRows.Next() {
                var idxName, colName string
                var nonUnique int
                var seq int
                var comment, idxType string
                if idxRows.Scan(&idxName, &colName, &nonUnique, &seq, &comment, &idxType) != nil {
                    continue
                }
                if idxName != lastName {
                    current = &plugin.IndexSchema{Name: idxName, Unique: nonUnique == 0}
                    if idxName == "PRIMARY" {
                        current.Primary = true
                    }
                    ts.Indexes = append(ts.Indexes, current)
                    lastName = idxName
                }
                if current != nil {
                    current.Columns = append(current.Columns, colName)
                }
            }
        }
        resp.Tables = append(resp.Tables, ts)
    }
    return resp, nil
}

func (m *mysqlPlugin) Exec(ctx context.Context, req *plugin.ExecRequest) (*plugin.ExecResponse, error) {
	if req.Options != nil {
		if v, ok := req.Options["explain-query"]; ok && v == "yes" {
			req.Query = "EXPLAIN " + req.Query
		}
	}
	dsn, err := buildDSN(req.Connection)
	if err != nil {
		return &plugin.ExecResponse{Error: fmt.Sprintf("invalid connection: %v", err)}, nil
	}
	if dsn == "" {
		return &plugin.ExecResponse{Error: "missing dsn in connection"}, nil
	}

	db, err := sql.Open("mysql", dsn)
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

	// prepare column metadata (type info currently unavailable, leave empty)
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

// ConnectionTree returns a server root node, a list of databases, and their
// tables for browsing.  Each level exposes DDL actions so the user can create
// or drop databases and tables directly from the connection tree.  If the
// connection is invalid or the query fails an empty tree is returned.
func (m *mysqlPlugin) ConnectionTree(ctx context.Context, req *plugin.ConnectionTreeRequest) (*plugin.ConnectionTreeResponse, error) {
	dsn, err := buildDSN(req.Connection)
	if err != nil || dsn == "" {
		return &plugin.ConnectionTreeResponse{}, nil
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return &plugin.ConnectionTreeResponse{}, nil
	}
	defer db.Close()

	// if the user supplied a database explicitly we only show that one
	filterDB := getDatabaseFromConn(req.Connection)

	rows, err := db.Query("SHOW DATABASES")
	if err != nil {
		return &plugin.ConnectionTreeResponse{}, nil
	}
	defer rows.Close()

	var dbNodes []*plugin.ConnectionTreeNode
	for rows.Next() {
		var dbname string
		if err := rows.Scan(&dbname); err != nil {
			continue
		}
		if filterDB != "" && dbname != filterDB {
			continue
		}
		// For each database expose a child list of tables.  Clicking a table
		// pre-fills a SELECT query; the DDL actions allow create/drop.
		tables := []*plugin.ConnectionTreeNode{}
		tblRows, err := db.Query(fmt.Sprintf("SHOW TABLES FROM `%s`", dbname))
		if err == nil {
			for tblRows.Next() {
				var tbl string
				if tblRows.Scan(&tbl) == nil {
					tables = append(tables, &plugin.ConnectionTreeNode{
						Key:      dbname + "." + tbl,
						Label:    tbl,
						NodeType: plugin.ConnectionTreeNodeTypeTable,
						Actions: []*plugin.ConnectionTreeAction{
						{Type: plugin.ConnectionTreeActionSelect, Title: "Select rows", Query: fmt.Sprintf("SELECT * FROM `%s` LIMIT 100;", tbl), Hidden: true, NewTab: true},
						{Type: plugin.ConnectionTreeActionDropTable, Title: "Drop table", Query: fmt.Sprintf("DROP TABLE `%s`;", tbl)},
						},
					})
				}
			}
			tblRows.Close()
		}
		dbNodes = append(dbNodes, &plugin.ConnectionTreeNode{
			Key:      dbname,
			Label:    dbname,
			NodeType: plugin.ConnectionTreeNodeTypeDatabase,
			Children: tables,
			Actions: []*plugin.ConnectionTreeAction{
				{Type: plugin.ConnectionTreeActionCreateTable, Title: "Create table", Query: "CREATE TABLE `new_table` (\n  `id` INT NOT NULL AUTO_INCREMENT,\n  PRIMARY KEY (`id`)\n);"},
				{Type: plugin.ConnectionTreeActionDropDatabase, Title: "Drop database", Query: fmt.Sprintf("DROP DATABASE `%s`;", dbname)},
			},
		})
	}

	// Prepend a leaf node for the create-database action so the user can
	// create a new database without a redundant wrapper server node.
	createNode := &plugin.ConnectionTreeNode{
		Key:      "__create_database__",
		Label:    "New database",
		NodeType: plugin.ConnectionTreeNodeTypeAction,
		Actions: []*plugin.ConnectionTreeAction{
			{Type: plugin.ConnectionTreeActionCreateDatabase, Title: "Create database", Query: "CREATE DATABASE `new_database`;", Hidden: true},
		},
	}

	return &plugin.ConnectionTreeResponse{Nodes: append([]*plugin.ConnectionTreeNode{createNode}, dbNodes...)}, nil
}

// TestConnection opens a MySQL connection and pings the server to verify the
// supplied credentials are valid. Nothing is persisted.
// GetCompletionFields returns column names and types for the given table,
// enabling context-aware auto-completion in the editor.
func (m *mysqlPlugin) GetCompletionFields(ctx context.Context, req *plugin.GetCompletionFieldsRequest) (*plugin.GetCompletionFieldsResponse, error) {
	if req.Collection == "" {
		return &plugin.GetCompletionFieldsResponse{}, nil
	}
	dsn, err := buildDSN(req.Connection)
	if err != nil || dsn == "" {
		return &plugin.GetCompletionFieldsResponse{}, nil
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return &plugin.GetCompletionFieldsResponse{}, nil
	}
	defer db.Close()

	// Support both "schema.table" and plain "table" forms.
	tableName := req.Collection
	schemaName := req.Database
	if parts := strings.SplitN(req.Collection, ".", 2); len(parts) == 2 {
		schemaName = parts[0]
		tableName = parts[1]
	}

	// When no schema is known, ask the live connection for the current database.
	// This handles the common case where the DSN contains the DB but req.Database
	// was not forwarded by the caller.
	if schemaName == "" {
		_ = db.QueryRowContext(ctx, "SELECT DATABASE()").Scan(&schemaName)
	}

	var rows *sql.Rows
	if schemaName != "" {
		colQ := `SELECT COLUMN_NAME, COLUMN_TYPE
			 FROM information_schema.COLUMNS
			 WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
			 ORDER BY ORDINAL_POSITION`
		rows, err = db.QueryContext(ctx, colQ, schemaName, tableName)
	} else {
		// Last resort: search all schemas for this table name.
		colQ := `SELECT COLUMN_NAME, COLUMN_TYPE
			 FROM information_schema.COLUMNS
			 WHERE TABLE_NAME = ?
			 ORDER BY TABLE_SCHEMA, ORDINAL_POSITION`
		rows, err = db.QueryContext(ctx, colQ, tableName)
	}
	if err != nil {
		return &plugin.GetCompletionFieldsResponse{}, nil
	}
	defer rows.Close()

	resp := &plugin.GetCompletionFieldsResponse{}
	for rows.Next() {
		var name, colType string
		if rows.Scan(&name, &colType) != nil {
			continue
		}
		resp.Fields = append(resp.Fields, &plugin.FieldInfo{Name: name, Type: colType})
	}
	return resp, nil
}

func (m *mysqlPlugin) TestConnection(ctx context.Context, req *plugin.TestConnectionRequest) (*plugin.TestConnectionResponse, error) {
	dsn, err := buildDSN(req.Connection)
	if err != nil || dsn == "" {
		msg := "invalid connection parameters"
		if err != nil {
			msg = err.Error()
		}
		return &plugin.TestConnectionResponse{Ok: false, Message: msg}, nil
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return &plugin.TestConnectionResponse{Ok: false, Message: fmt.Sprintf("open error: %v", err)}, nil
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		return &plugin.TestConnectionResponse{Ok: false, Message: fmt.Sprintf("ping error: %v", err)}, nil
	}
	return &plugin.TestConnectionResponse{Ok: true, Message: "Connection successful"}, nil
}

// escapeBacktick doubles any backtick characters in s so it can be safely
// embedded between MySQL backtick identifier delimiters.
func escapeBacktick(s string) string {
	return strings.ReplaceAll(s, "`", "``")
}

// quoteSource wraps a table reference in backticks, handling the optional
// "database.table" form produced by DescribeSchema (e.g. "employees.users"
// becomes `employees`.`users`).
func quoteSource(source string) string {
	parts := strings.SplitN(source, ".", 2)
	if len(parts) == 2 {
		return fmt.Sprintf("`%s`.`%s`", escapeBacktick(parts[0]), escapeBacktick(parts[1]))
	}
	return fmt.Sprintf("`%s`", escapeBacktick(source))
}

// MutateRow executes an UPDATE or DELETE against the MySQL database identified
// by req.Connection.  req.Source must be the unquoted table name and
// req.Filter must be a non-empty SQL WHERE expression; both are supplied by
// the frontend modal.  Column values are passed as query parameters so they
// cannot alter the statement structure; source and filter are backtick-quoted
// and forwarded verbatim, which is appropriate for a developer-facing tool
// where the user controls those fields directly.
func (m *mysqlPlugin) MutateRow(ctx context.Context, req *plugin.MutateRowRequest) (*plugin.MutateRowResponse, error) {
	if req.Source == "" {
		return &plugin.MutateRowResponse{Success: false, Error: "source (table name) is required"}, nil
	}
	if req.Filter == "" {
		return &plugin.MutateRowResponse{Success: false, Error: "filter (WHERE clause) is required"}, nil
	}

	dsn, err := buildDSN(req.Connection)
	if err != nil || dsn == "" {
		return &plugin.MutateRowResponse{Success: false, Error: "invalid connection"}, nil
	}

	// Defense-in-depth: if the DSN has no default database selected but the
	// source is a qualified "db.table" reference, derive the database from
	// the source so the connection targets the correct schema.  This covers
	// the case where the frontend omits the database key in the connection map
	// (e.g. when the credential was saved without a default database).
	if cfg, parseErr := mysql.ParseDSN(dsn); parseErr == nil && cfg.DBName == "" {
		if parts := strings.SplitN(req.Source, ".", 2); len(parts) == 2 && parts[0] != "" {
			cfg.DBName = parts[0]
			if rebuilt := cfg.FormatDSN(); rebuilt != "" {
				dsn = rebuilt
			}
		}
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return &plugin.MutateRowResponse{Success: false, Error: fmt.Sprintf("open error: %v", err)}, nil
	}
	defer db.Close()

	var query string
	var args []interface{}

	switch req.Operation {
	case pluginpb.PluginV1_MutateRowRequest_UPDATE:
		if len(req.Values) == 0 {
			return &plugin.MutateRowResponse{Success: false, Error: "values are required for UPDATE"}, nil
		}
		// Collect column names in sorted order so the SET clause is deterministic.
		keys := make([]string, 0, len(req.Values))
		for k := range req.Values {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		setParts := make([]string, 0, len(keys))
		for _, k := range keys {
			setParts = append(setParts, fmt.Sprintf("`%s`=?", escapeBacktick(k)))
			args = append(args, req.Values[k])
		}
		query = fmt.Sprintf("UPDATE %s SET %s WHERE %s",
			quoteSource(req.Source), strings.Join(setParts, ", "), req.Filter)
	case pluginpb.PluginV1_MutateRowRequest_DELETE:
		query = fmt.Sprintf("DELETE FROM %s WHERE %s", quoteSource(req.Source), req.Filter)
	default:
		return &plugin.MutateRowResponse{Success: false, Error: "operation not supported"}, nil
	}

	if _, err := db.ExecContext(ctx, query, args...); err != nil {
		return &plugin.MutateRowResponse{Success: false, Error: err.Error()}, nil
	}
	return &plugin.MutateRowResponse{Success: true}, nil
}

func main() {
	plugin.ServeCLI(&mysqlPlugin{})
}
