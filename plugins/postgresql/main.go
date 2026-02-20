package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

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

func (m *postgresqlPlugin) AuthForms(plugin.AuthFormsRequest) (plugin.AuthFormsResponse, error) {
	// Provide two options: a `basic` property-based form and a `dsn` fallback.
	basic := plugin.AuthForm{
		Key: "basic",
		Name: "Basic",
		Fields: []*plugin.AuthField{
			{Type: plugin.AuthField_TEXT, Name: "host", Label: "Host", Required: true, Placeholder: "127.0.0.1"},
			{Type: plugin.AuthField_NUMBER, Name: "port", Label: "Port", Placeholder: "5432"},
			{Type: plugin.AuthField_TEXT, Name: "user", Label: "User"},
			{Type: plugin.AuthField_PASSWORD, Name: "password", Label: "Password"},
			{Type: plugin.AuthField_TEXT, Name: "database", Label: "Database name"},
		},
	}

	return plugin.AuthFormsResponse{Forms: map[string]*plugin.AuthForm{"basic": &basic}}, nil
}

func (m *postgresqlPlugin) Exec(req plugin.ExecRequest) (plugin.ExecResponse, error) {
	// Accept either a full DSN under key "dsn" (legacy) or a credential blob
	// JSON (recommended) stored under "credential_blob" containing: {"form":"basic","values": { ... }}
	dsn, ok := req.Connection["dsn"]
	if !ok || dsn == "" {
		// try credential_blob
		if blob, ok2 := req.Connection["credential_blob"]; ok2 && blob != "" {
			var payload struct {
				Form   string            `json:"form"`
				Values map[string]string `json:"values"`
			}
			if err := json.Unmarshal([]byte(blob), &payload); err == nil {
				// if plugin stored a dsn inside values, prefer that
				if v, ok := payload.Values["dsn"]; ok && v != "" {
					dsn = v
				} else {
					// build a simple postgres connection string
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
			}
		}
	}

	if dsn == "" {
		return plugin.ExecResponse{Error: "missing dsn in connection"}, nil
	}

	// open postgres driver
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return plugin.ExecResponse{Error: fmt.Sprintf("open error: %v", err)}, nil
	}
	defer db.Close()

	rows, err := db.Query(req.Query)
	if err != nil {
		return plugin.ExecResponse{Error: fmt.Sprintf("query error: %v", err)}, nil
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return plugin.ExecResponse{Error: fmt.Sprintf("cols error: %v", err)}, nil
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
			return plugin.ExecResponse{Error: fmt.Sprintf("scan error: %v", err)}, nil
		}
		strs := make([]string, len(cols))
		for i, v := range vals {
			if v == nil {
				strs[i] = ""
			} else {
				strs[i] = fmt.Sprintf("%v", v)
			}
		}
		rowResults = append(rowResults, &plugin.Row{Values: strs})
	}

	return plugin.ExecResponse{
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

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: postgres info | exec | authforms")
		os.Exit(2)
	}

	// Allow running via pkg/plugin.ServeCLI as well but keep a fallback CLI that
	// decodes stdin and calls the implementation for direct builds.
	impl := &postgresqlPlugin{}
	switch args[0] {
	case "info":
		info, _ := impl.Info()
		b, _ := json.Marshal(info)
		os.Stdout.Write(b)
	case "exec":
		var req plugin.ExecRequest
		in, _ := io.ReadAll(os.Stdin)
		_ = json.Unmarshal(in, &req)
		res, _ := impl.Exec(req)
		b, _ := json.Marshal(res)
		os.Stdout.Write(b)
	case "authforms":
		res, _ := impl.AuthForms(plugin.AuthFormsRequest{})
		b, _ := json.Marshal(res)
		os.Stdout.Write(b)
	default:
		fmt.Fprintln(os.Stderr, "Usage: postgres info | exec | authforms")
		os.Exit(2)
	}
}
