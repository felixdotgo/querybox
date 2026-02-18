package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/felixdotgo/querybox/pkg/plugin"

	_ "github.com/go-sql-driver/mysql"
)

// mysqlPlugin implements the plugin.Plugin interface for a simple MySQL executor.
type mysqlPlugin struct{}

func (m *mysqlPlugin) Info() (plugin.InfoResponse, error) {
	return plugin.InfoResponse{
		Type:        plugin.TypeDriver,
		Name:        "mysql",
		Version:     "0.1.0",
		Description: "MySQL plugin (exec only)",
	}, nil
}

func (m *mysqlPlugin) Exec(req plugin.ExecRequest) (plugin.ExecResponse, error) {
	// Expect connection map to contain a DSN under key "dsn" for simplicity.
	dsn, ok := req.Connection["dsn"]
	if !ok || dsn == "" {
		return plugin.ExecResponse{Error: "missing dsn in connection"}, nil
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return plugin.ExecResponse{Error: fmt.Sprintf("open error: %v", err)}, nil
	}
	defer db.Close()

	rows, err := db.Query(req.Sql)
	if err != nil {
		return plugin.ExecResponse{Error: fmt.Sprintf("query error: %v", err)}, nil
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return plugin.ExecResponse{Error: fmt.Sprintf("cols error: %v", err)}, nil
	}

	results := make([]map[string]interface{}, 0)
	for rows.Next() {
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return plugin.ExecResponse{Error: fmt.Sprintf("scan error: %v", err)}, nil
		}
		row := map[string]interface{}{}
		for i, c := range cols {
			row[c] = vals[i]
		}
		results = append(results, row)
	}

	b, _ := json.Marshal(results)
	return plugin.ExecResponse{Result: string(b)}, nil
}

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: mysql info | exec")
		os.Exit(2)
	}

	// Allow running via pkg/plugin.ServeCLI as well but keep a fallback CLI that
	// decodes stdin and calls the implementation for direct builds.
	impl := &mysqlPlugin{}
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
	default:
		fmt.Fprintln(os.Stderr, "Usage: mysql info | exec")
		os.Exit(2)
	}
}
