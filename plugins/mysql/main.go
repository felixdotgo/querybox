package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/felixdotgo/querybox/pkg/plugin"
	pluginpb "github.com/felixdotgo/querybox/rpc/contracts/plugin/v1"

	_ "github.com/go-sql-driver/mysql"
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
		Capabilities: []string{"transactions", "replication"},
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
			{Type: plugin.AuthFieldSelect, Name: "tls", Label: "TLS mode (e.g. skip-verify)", Options: []string{"skip-verify", "true", "false"}, Value: "skip-verify"},
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
func buildDSN(connection map[string]string) (string, error) {
	// Accept either a full DSN under key "dsn" (legacy) or a credential blob
	// JSON (recommended) stored under "credential_blob" containing: {"form":"basic","values": { ... }}
	// Additionally we allow arbitrary extra parameters (including tls) which
	// are appended as query parameters.  This lets callers configure SSL
	// (tls=skip-verify, tls=true, etc) or other driver options.
	dsn, ok := connection["dsn"]
	if !ok || dsn == "" {
		// try credential_blob
		if blob, ok2 := connection["credential_blob"]; ok2 && blob != "" {
			var payload struct {
				Form   string            `json:"form"`
				Values map[string]string `json:"values"`
			}
			if err := json.Unmarshal([]byte(blob), &payload); err == nil {
				// if plugin stored a dsn inside values, prefer that
				if v, ok := payload.Values["dsn"]; ok && v != "" {
					dsn = v
				} else {
					// build a simple DSN from common keys
					host := payload.Values["host"]
					user := payload.Values["user"]
					pass := payload.Values["password"]
					port := payload.Values["port"]
					dbname := payload.Values["database"]
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
					for k, v := range payload.Values {
						switch k {
						case "host", "user", "password", "port", "database", "dsn":
							// already handled above
							continue
						}
						if v != "" {
							params.Add(k, v)
						}
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
	}
	return dsn, nil
}

func (m *mysqlPlugin) Exec(ctx context.Context, req *plugin.ExecRequest) (*plugin.ExecResponse, error) {
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
							{Type: plugin.ConnectionTreeActionSelect, Title: "Select rows", Query: fmt.Sprintf("SELECT * FROM `%s`.`%s` LIMIT 100;", dbname, tbl), Hidden: true, NewTab: true},
							{Type: plugin.ConnectionTreeActionDropTable, Title: "Drop table", Query: fmt.Sprintf("DROP TABLE `%s`.`%s`;", dbname, tbl)},
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
				{Type: plugin.ConnectionTreeActionCreateTable, Title: "Create table", Query: fmt.Sprintf("CREATE TABLE `%s`.`new_table` (\n  `id` INT NOT NULL AUTO_INCREMENT,\n  PRIMARY KEY (`id`)\n);", dbname)},
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

func main() {
	plugin.ServeCLI(&mysqlPlugin{})
}
