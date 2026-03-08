package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/felixdotgo/querybox/pkg/certs"
	"github.com/felixdotgo/querybox/pkg/plugin"
	pluginpb "github.com/felixdotgo/querybox/rpc/contracts/plugin/v1"

	_ "github.com/lib/pq" // postgres driver
)

// postgresqlPlugin implements the protobuf PluginServiceServer interface for a simple PostgreSQL executor.
type postgresqlPlugin struct {
	pluginpb.UnimplementedPluginServiceServer
}

func (m *postgresqlPlugin) Info(ctx context.Context, _ *pluginpb.PluginV1_InfoRequest) (*plugin.InfoResponse, error) {
	return &plugin.InfoResponse{
		Type:        plugin.TypeDriver,
		Name:        "PostgreSQL",
		Version:     "0.1.0",
		Description: "PostgreSQL database driver",
		Url:         "https://www.postgresql.org/",
		Author:      "PostgreSQL Global Development Group",
		Capabilities: []string{"query", "explain-query"},
		Tags:        []string{"sql", "relational"},
		License:     "PostgreSQL",
		IconUrl:     "https://www.postgresql.org/media/img/about/press/elephant.png",
	}, nil
}

func (m *postgresqlPlugin) AuthForms(ctx context.Context, _ *plugin.AuthFormsRequest) (*plugin.AuthFormsResponse, error) {
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

// ensureSSLMode ensures that a DSN string has an explicit sslmode
// directive when the caller asked for TLS disabled.  Two common DSN
// forms exist: keyword/value pairs ("host=... sslmode=...") and URL form
// ("postgres://user@host/db?sslmode=...").  The driver defaults to an
// SSL mode which may not match our expectation; we prefer to explicitly
// set `sslmode=disable` if no value is present.
func ensureSSLMode(dsn string) string {
	// ensure an sslmode parameter exists; default to disable if missing.
	if !strings.Contains(dsn, "sslmode=") {
		u, err := url.Parse(dsn)
		if err == nil && (u.Scheme == "postgres" || u.Scheme == "postgresql") {
			q := u.Query()
			q.Set("sslmode", "disable")
			u.RawQuery = q.Encode()
			dsn = u.String()
		} else if strings.ContainsAny(dsn, " \t") {
			dsn = dsn + " sslmode=disable"
		}
	}

	// if verification mode requested and missing root cert, attach our bundle
	mode := ""
	for _, part := range strings.Fields(dsn) {
		if strings.HasPrefix(part, "sslmode=") {
			mode = strings.TrimPrefix(part, "sslmode=")
			break
		}
	}
	// if not found via whitespace, try URL query
	if mode == "" {
		if u, err := url.Parse(dsn); err == nil {
			mode = u.Query().Get("sslmode")
		}
	}
	if mode == "verify-ca" || mode == "verify-full" {
		if !strings.Contains(dsn, "sslrootcert=") {
			if path, err := certs.RootCertPath(); err == nil {
				if strings.ContainsAny(dsn, " \t") {
					dsn = dsn + " sslrootcert=" + path
				} else if u, err := url.Parse(dsn); err == nil {
					q := u.Query()
					q.Set("sslrootcert", path)
					u.RawQuery = q.Encode()
					dsn = u.String()
				} else {
					dsn = dsn + " sslrootcert=" + path
				}
			}
		}
	}
	return dsn
}

// setSSLMode forces the supplied sslmode into the DSN, overwriting any existing
// value.  It handles both URL and keyword‑style strings.
func setSSLMode(dsn, mode string) string {
    if mode == "" {
        return dsn
    }
    if u, err := url.Parse(dsn); err == nil && (u.Scheme == "postgres" || u.Scheme == "postgresql") {
        q := u.Query()
        q.Del("sslmode")
        q.Set("sslmode", mode)
        u.RawQuery = q.Encode()
        return u.String()
    }
    parts := strings.Fields(dsn)
    var kept []string
    for _, p := range parts {
        if strings.HasPrefix(p, "sslmode=") {
            continue
        }
        kept = append(kept, p)
    }
    if len(kept) > 0 {
        return strings.Join(kept, " ") + " sslmode=" + mode
    }
    return dsn + " sslmode=" + mode
}

// buildConnString constructs a postgres keyword=value connection string from
// the provided connection map.  Extra DSN parameters are appended as
// space-separated key=value pairs as required by lib/pq; URL-encoded (&)
// format is NOT used because it is invalid for the postgres DSN format.
//
// Historically we ignored the "database" field when a raw DSN was present.
// that meant that ConnectionTree created new connections for each database but
// the DSN still pointed at the original database.  The symptom was that all
// databases in the tree showed the same schemas/tables.  This helper now
// overrides the DSN if a database override is supplied.
func buildConnString(connection map[string]string) (string, error) {
	// honour explicit DSN value and still ensure sslmode defaults correctly
	if dsn, ok := connection["dsn"]; ok && dsn != "" {
		// if the caller also supplied a "database" field, it should override
		// whatever database is encoded in the DSN.
		if db, ok2 := connection["database"]; ok2 && db != "" {
			var err error
			dsn, err = overrideDatabaseInDSN(dsn, db)
			if err != nil {
				return "", err
			}
		}
		if tls, ok2 := connection["tls"]; ok2 && tls != "" {
			dsn = setSSLMode(dsn, tls)
		}
		return ensureSSLMode(dsn), nil
	}
	dsn, ok := connection["dsn"]
	if !ok || dsn == "" {
		if blob, ok2 := connection["credential_blob"]; ok2 && blob != "" {
			var payload struct {
				Form   string            `json:"form"`
				Values map[string]string `json:"values"`
			}
			if err := json.Unmarshal([]byte(blob), &payload); err == nil {
				if v, ok := payload.Values["dsn"]; ok && v != "" {
					dsn = ensureSSLMode(v)
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
						// build keyword-style DSN; omit dbname when blank.  Including
						// an empty "dbname=" followed by a space could cause lib/pq to
						// treat the next token (e.g. "sslmode=disable") as the
						// database name, which is what was reported by users.
						parts := []string{
							"host=" + host,
							"port=" + port,
							"user=" + user,
							"password=" + pass,
						}
						if dbname != "" {
							parts = append(parts, "dbname="+dbname)
						}
						parts = append(parts, "sslmode="+sslmode)
						dsn = strings.Join(parts, " ")
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
	// Apply explicit database override that may have been injected by ConnectionTree
	// when scanning non-current databases.  The credential_blob paths above build the
	// DSN from blob fields without knowledge of this override, so we apply it here
	// once, covering both the embedded-DSN and separate-fields blob forms.
	if db, ok := connection["database"]; ok && db != "" && dsn != "" {
		if newDSN, oErr := overrideDatabaseInDSN(dsn, db); oErr == nil {
			dsn = newDSN
		}
	}
	// final normalisation
	dsn = ensureSSLMode(dsn)
	return dsn, nil
}

// overrideDatabaseInDSN returns a copy of the supplied DSN with its database
// name replaced by the provided value.  Both key/value and URL forms are
// handled.  We do not attempt to fully validate the DSN; the operation is
// best-effort so that callers can continue with whatever the driver accepts.
func overrideDatabaseInDSN(dsn, database string) (string, error) {
	// URL style
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		u, err := url.Parse(dsn)
		if err != nil {
			return "", err
		}
		// override path (u.Path includes leading '/').  Some URLs may encode the
		// database in a query parameter instead; set both to be safe.
		u.Path = "/" + database
		q := u.Query()
		q.Set("dbname", database)
		u.RawQuery = q.Encode()
		return u.String(), nil
	}

	// keyword/value style: replace existing dbname= token if present, otherwise
	// append it.
	parts := strings.Fields(dsn)
	var out []string
	replaced := false
	for _, tok := range parts {
		if strings.HasPrefix(tok, "dbname=") {
			out = append(out, "dbname="+database)
			replaced = true
		} else {
			out = append(out, tok)
		}
	}
	if !replaced {
		out = append(out, "dbname="+database)
	}
	return strings.Join(out, " "), nil
}
// openPostgresDB wraps sql.Open so unit tests can replace it with a mock.
var openPostgresDB = func(dsn string) (*sql.DB, error) {
	return sql.Open("postgres", dsn)
}

// getDatabaseFromConn extracts a requested database name from the
// connection metadata.  It checks the explicit "database" field, the
// credential_blob payload, and finally any dbname element in a supplied
// DSN string.  An empty return value indicates no preference.
func getDatabaseFromConn(conn map[string]string) string {
	if db, ok := conn["database"]; ok && db != "" {
		return db
	}
	if blob, ok := conn["credential_blob"]; ok && blob != "" {
		var payload struct {
			Form   string            `json:"form"`
			Values map[string]string `json:"values"`
		}
		if err := json.Unmarshal([]byte(blob), &payload); err == nil {
			if v, ok := payload.Values["database"]; ok && v != "" {
				return v
			}
			if v, ok := payload.Values["dsn"]; ok && v != "" {
				if name := extractDBName(v); name != "" {
					return name
				}
			}
		}
	}
	if dsn, ok := conn["dsn"]; ok && dsn != "" {
		if name := extractDBName(dsn); name != "" {
			return name
		}
	}
	return ""
}

// extractDBName returns the database name found in the provided DSN string.
// Supports both URL and keyword forms.
func extractDBName(dsn string) string {
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		if u, err := url.Parse(dsn); err == nil {
			if p := strings.TrimPrefix(u.Path, "/"); p != "" {
				return p
			}
		}
	}
	for _, part := range strings.Fields(dsn) {
		if strings.HasPrefix(part, "dbname=") {
			return strings.TrimPrefix(part, "dbname=")
		}
	}
	return ""
}

func (m *postgresqlPlugin) DescribeSchema(ctx context.Context, req *plugin.DescribeSchemaRequest) (*plugin.DescribeSchemaResponse, error) {
    dsn, err := buildConnString(req.Connection)
    if err != nil {
        return &plugin.DescribeSchemaResponse{}, nil
    }
    if dsn == "" {
        return &plugin.DescribeSchemaResponse{}, nil
    }
    db, err := openPostgresDB(dsn)
    if err != nil {
        return &plugin.DescribeSchemaResponse{}, nil
    }
    defer db.Close()

    resp := &plugin.DescribeSchemaResponse{}
    // base tables, excluding Postgres system schemas and partition children
    query := `SELECT t.table_schema, t.table_name
FROM information_schema.tables t
WHERE t.table_type='BASE TABLE'
  AND t.table_schema NOT IN ('pg_catalog','information_schema')
  AND NOT EXISTS (
      SELECT 1
      FROM pg_catalog.pg_class c2
      JOIN pg_catalog.pg_namespace n2 ON n2.oid = c2.relnamespace
      JOIN pg_catalog.pg_inherits i ON i.inhrelid = c2.oid
      WHERE n2.nspname = t.table_schema
        AND c2.relname = t.table_name
  )`
    args := []interface{}{}
    if req.Database != "" {
        query += " AND table_catalog = ?"
        args = append(args, req.Database)
    }
    if req.Table != "" {
        query += " AND table_name = ?"
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
        ts := &plugin.TableSchema{Name: schema + "." + tbl}
        // columns
        colQ := `SELECT column_name, data_type, is_nullable, ordinal_position, column_default
                 FROM information_schema.columns
                 WHERE table_schema=$1 AND table_name=$2
                 ORDER BY ordinal_position`
        colRows, err := db.Query(colQ, schema, tbl)
        if err == nil {
            defer colRows.Close()
            for colRows.Next() {
                var name, dtype, isNull string
                var pos int32
                var def sql.NullString
                if err := colRows.Scan(&name, &dtype, &isNull, &pos, &def); err != nil {
                    continue
                }
                cs := &plugin.ColumnSchema{
                    Name:       name,
                    Type:       dtype,
                    Nullable:   strings.EqualFold(isNull, "YES"),
                    Ordinal:    pos,
                }
                if def.Valid {
                    cs.Default = def.String
                }
                ts.Columns = append(ts.Columns, cs)
            }
        }
        // indexes (basic names and uniqueness)
        idxQ := `SELECT indexname, indexdef FROM pg_indexes WHERE schemaname=$1 AND tablename=$2`
        idxRows, err := db.Query(idxQ, schema, tbl)
        if err == nil {
            defer idxRows.Close()
            for idxRows.Next() {
                var name, def string
                if idxRows.Scan(&name, &def) != nil {
                    continue
                }
                idx := &plugin.IndexSchema{Name: name}
                if strings.Contains(def, "UNIQUE") {
                    idx.Unique = true
                }
                ts.Indexes = append(ts.Indexes, idx)
            }
        }
        resp.Tables = append(resp.Tables, ts)
    }
    return resp, nil
}

func (m *postgresqlPlugin) Exec(ctx context.Context, req *plugin.ExecRequest) (*plugin.ExecResponse, error) {
	if req.Options != nil {
		if v, ok := req.Options["explain-query"]; ok && v == "yes" {
			req.Query = "EXPLAIN " + req.Query
		}
	}
	dsn, err := buildConnString(req.Connection)
	if err != nil {
		return &plugin.ExecResponse{Error: fmt.Sprintf("invalid connection: %v", err)}, nil
	}
	if dsn == "" {
		return &plugin.ExecResponse{Error: "missing dsn in connection"}, nil
	}

	// open postgres driver (custom hook for testing)
	db, err := openPostgresDB(dsn)
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

// ConnectionTree returns a server → database → schema → table hierarchy.
// It now enumerates _all_ databases on the server (subject to an explicit
// database filter) rather than just the one to which the connection is
// currently attached.  Behaviour falls back gracefully when listing fails.
func (m *postgresqlPlugin) ConnectionTree(ctx context.Context, req *plugin.ConnectionTreeRequest) (*plugin.ConnectionTreeResponse, error) {
	dsn, err := buildConnString(req.Connection)
	if err != nil || dsn == "" {
		fmt.Fprintf(os.Stderr, "postgresql: ConnectionTree: DSN error: %v dsn=%q\n", err, dsn)
		return &plugin.ConnectionTreeResponse{}, nil
	}

	db, err := openPostgresDB(dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "postgresql: ConnectionTree: open error: %v\n", err)
		return &plugin.ConnectionTreeResponse{}, nil
	}
	defer db.Close()

	// determine the database we are connected to now; used for reuse below
	var currentDB string
	if scanErr := db.QueryRow("SELECT current_database()").Scan(&currentDB); scanErr != nil {
		currentDB = "current"
	}

	// optional filter coming from the connection info
	filterDB := getDatabaseFromConn(req.Connection)

	// retrieve list of databases on the server
	dbNames := []string{}
	rows, err := db.Query(`SELECT datname FROM pg_database WHERE NOT datistemplate AND datallowconn`)
	if err != nil {
		// if we can't list, fall back to the one we know about
		dbNames = []string{currentDB}
	} else {
		defer rows.Close()
		var name string
		for rows.Next() {
			if err := rows.Scan(&name); err == nil {
				dbNames = append(dbNames, name)
			}
		}
		if len(dbNames) == 0 {
			dbNames = []string{currentDB}
		}
	}

	// apply explicit database filter if supplied
	if filterDB != "" {
		found := false
		for _, n := range dbNames {
			if n == filterDB {
				dbNames = []string{n}
				found = true
				break
			}
		}
		if !found {
			dbNames = []string{filterDB} // still show the requested name
		}
	}

	// helper to build schema nodes for a given *sql.DB
	loadSchemas := func(conn *sql.DB) []*plugin.ConnectionTreeNode {
		schemaRows, err := conn.Query(`
SELECT schema_name
FROM information_schema.schemata
WHERE schema_name NOT IN ('pg_catalog','information_schema','pg_toast')
  AND schema_name NOT LIKE 'pg_%'
ORDER BY schema_name`)
		if err != nil {
			return nil
		}
		defer schemaRows.Close()

		var schemaNodes []*plugin.ConnectionTreeNode
		for schemaRows.Next() {
			var schemaName string
			if err := schemaRows.Scan(&schemaName); err != nil {
				continue
			}

			// ── Tables (regular + partitioned) (hide partition children) ─────
			var tableNodes []*plugin.ConnectionTreeNode
			if rows, err := conn.Query(`
SELECT c.relname
FROM pg_catalog.pg_class c
JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = $1
  AND c.relkind IN ('r', 'p')
  -- exclude tables that inherit from another (i.e. partitions)
  AND NOT EXISTS (
      SELECT 1 FROM pg_catalog.pg_inherits i WHERE i.inhrelid = c.oid
  )
ORDER BY c.relname`, schemaName); err == nil {
				for rows.Next() {
					var tbl string
					if err := rows.Scan(&tbl); err == nil {
						tableNodes = append(tableNodes, &plugin.ConnectionTreeNode{
							Key:      schemaName + "." + tbl,
							Label:    tbl,
							NodeType: plugin.ConnectionTreeNodeTypeTable,
							Actions: []*plugin.ConnectionTreeAction{
								{
									Type:   plugin.ConnectionTreeActionSelect,
									Title:  "Select rows",
									Query:  fmt.Sprintf(`SELECT * FROM "%s"."%s" LIMIT 100;`, schemaName, tbl),
									Hidden: true,
									NewTab: true,
								},
								{
									Type:  plugin.ConnectionTreeActionDropTable,
									Title: "Drop table",
									Query: fmt.Sprintf(`DROP TABLE "%s"."%s";`, schemaName, tbl),
								},
							},
						})
					}
				}
				rows.Close()
			}

			// ── Views ────────────────────────────────────────────────────────
// 			var viewNodes []*plugin.ConnectionTreeNode
// 			if rows, err := conn.Query(`
// SELECT c.relname
// FROM pg_catalog.pg_class c
// JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
// WHERE n.nspname = $1
//   AND c.relkind = 'v'
// ORDER BY c.relname`, schemaName); err == nil {
// 				for rows.Next() {
// 					var v string
// 					if err := rows.Scan(&v); err == nil {
// 						viewNodes = append(viewNodes, &plugin.ConnectionTreeNode{
// 							Key:      schemaName + ".v." + v,
// 							Label:    v,
// 							NodeType: plugin.ConnectionTreeNodeTypeView,
// 							Actions: []*plugin.ConnectionTreeAction{
// 								{
// 									Type:   plugin.ConnectionTreeActionSelect,
// 									Title:  "Select rows",
// 									Query:  fmt.Sprintf(`SELECT * FROM "%s"."%s" LIMIT 100;`, schemaName, v),
// 									Hidden: true,
// 									NewTab: true,
// 								},
// 							},
// 						})
// 					}
// 				}
// 				rows.Close()
// 			}

			// ── Materialized Views ───────────────────────────────────────────
// 			var matViewNodes []*plugin.ConnectionTreeNode
// 			if rows, err := conn.Query(`
// SELECT c.relname
// FROM pg_catalog.pg_class c
// JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
// WHERE n.nspname = $1
//   AND c.relkind = 'm'
// ORDER BY c.relname`, schemaName); err == nil {
// 				for rows.Next() {
// 					var v string
// 					if err := rows.Scan(&v); err == nil {
// 						matViewNodes = append(matViewNodes, &plugin.ConnectionTreeNode{
// 							Key:      schemaName + ".mv." + v,
// 							Label:    v,
// 							NodeType: plugin.ConnectionTreeNodeTypeView,
// 							Actions: []*plugin.ConnectionTreeAction{
// 								{
// 									Type:   plugin.ConnectionTreeActionSelect,
// 									Title:  "Select rows",
// 									Query:  fmt.Sprintf(`SELECT * FROM "%s"."%s" LIMIT 100;`, schemaName, v),
// 									Hidden: true,
// 									NewTab: true,
// 								},
// 							},
// 						})
// 					}
// 				}
// 				rows.Close()
// 			}

			// ── Foreign Tables ───────────────────────────────────────────────
// 			var foreignTableNodes []*plugin.ConnectionTreeNode
// 			if rows, err := conn.Query(`
// SELECT c.relname
// FROM pg_catalog.pg_class c
// JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
// WHERE n.nspname = $1
//   AND c.relkind = 'f'
// ORDER BY c.relname`, schemaName); err == nil {
// 				for rows.Next() {
// 					var ft string
// 					if err := rows.Scan(&ft); err == nil {
// 						foreignTableNodes = append(foreignTableNodes, &plugin.ConnectionTreeNode{
// 							Key:      schemaName + ".ft." + ft,
// 							Label:    ft,
// 							NodeType: plugin.ConnectionTreeNodeTypeTable,
// 							Actions: []*plugin.ConnectionTreeAction{
// 								{
// 									Type:   plugin.ConnectionTreeActionSelect,
// 									Title:  "Select rows",
// 									Query:  fmt.Sprintf(`SELECT * FROM "%s"."%s" LIMIT 100;`, schemaName, ft),
// 									Hidden: true,
// 									NewTab: true,
// 								},
// 							},
// 						})
// 					}
// 				}
// 				rows.Close()
// 			}

			// ── Indexes ──────────────────────────────────────────────────────
// 			var indexNodes []*plugin.ConnectionTreeNode
// 			if rows, err := conn.Query(`
// SELECT indexname
// FROM pg_indexes
// WHERE schemaname = $1
// ORDER BY indexname`, schemaName); err == nil {
// 				for rows.Next() {
// 					var idx string
// 					if err := rows.Scan(&idx); err == nil {
// 						indexNodes = append(indexNodes, &plugin.ConnectionTreeNode{
// 							Key:      schemaName + ".idx." + idx,
// 							Label:    idx,
// 							NodeType: plugin.ConnectionTreeNodeTypeGroup,
// 						})
// 					}
// 				}
// 				rows.Close()
// 			}

			// ── Functions ────────────────────────────────────────────────────
// 			var functionNodes []*plugin.ConnectionTreeNode
// 			if rows, err := conn.Query(`
// SELECT p.proname || '(' || pg_catalog.pg_get_function_identity_arguments(p.oid) || ')' AS signature
// FROM pg_catalog.pg_proc p
// JOIN pg_catalog.pg_namespace n ON n.oid = p.pronamespace
// WHERE n.nspname = $1
//   AND p.prokind = 'f'
// ORDER BY p.proname`, schemaName); err == nil {
// 				for rows.Next() {
// 					var sig string
// 					if err := rows.Scan(&sig); err == nil {
// 						functionNodes = append(functionNodes, &plugin.ConnectionTreeNode{
// 							Key:      schemaName + ".fn." + sig,
// 							Label:    sig,
// 							NodeType: plugin.ConnectionTreeNodeTypeGroup,
// 						})
// 					}
// 				}
// 				rows.Close()
// 			}

			// ── Sequences ────────────────────────────────────────────────────
// 			var sequenceNodes []*plugin.ConnectionTreeNode
// 			if rows, err := conn.Query(`
// SELECT sequence_name
// FROM information_schema.sequences
// WHERE sequence_schema = $1
// ORDER BY sequence_name`, schemaName); err == nil {
// 				for rows.Next() {
// 					var seq string
// 					if err := rows.Scan(&seq); err == nil {
// 						sequenceNodes = append(sequenceNodes, &plugin.ConnectionTreeNode{
// 							Key:      schemaName + ".seq." + seq,
// 							Label:    seq,
// 							NodeType: plugin.ConnectionTreeNodeTypeGroup,
// 						})
// 					}
// 				}
// 				rows.Close()
// 			}

			// ── Assemble category group nodes ────────────────────────────────
			categories := []*plugin.ConnectionTreeNode{
				{
					Key:      schemaName + ".Tables",
					Label:    "Tables",
					NodeType: plugin.ConnectionTreeNodeTypeGroup,
					Children: tableNodes,
					Actions: []*plugin.ConnectionTreeAction{
						{
							Type:  plugin.ConnectionTreeActionCreateTable,
							Title: "Create table",
							Query: fmt.Sprintf("CREATE TABLE \"%s\".\"new_table\" (\n    id SERIAL PRIMARY KEY\n);", schemaName),
						},
					},
				},
				// {
				// 	Key:      schemaName + ".Views",
				// 	Label:    "Views",
				// 	NodeType: plugin.ConnectionTreeNodeTypeGroup,
				// 	Children: viewNodes,
				// },
				// {
				// 	Key:      schemaName + ".Materialized Views",
				// 	Label:    "Materialized Views",
				// 	NodeType: plugin.ConnectionTreeNodeTypeGroup,
				// 	Children: matViewNodes,
				// },
				// {
				// 	Key:      schemaName + ".Foreign Tables",
				// 	Label:    "Foreign Tables",
				// 	NodeType: plugin.ConnectionTreeNodeTypeGroup,
				// 	Children: foreignTableNodes,
				// },
				// {
				// 	Key:      schemaName + ".Indexes",
				// 	Label:    "Indexes",
				// 	NodeType: plugin.ConnectionTreeNodeTypeGroup,
				// 	Children: indexNodes,
				// },
				// {
				// 	Key:      schemaName + ".Functions",
				// 	Label:    "Functions",
				// 	NodeType: plugin.ConnectionTreeNodeTypeGroup,
				// 	Children: functionNodes,
				// },
				// {
				// 	Key:      schemaName + ".Sequences",
				// 	Label:    "Sequences",
				// 	NodeType: plugin.ConnectionTreeNodeTypeGroup,
				// 	Children: sequenceNodes,
				// },
			}

			schemaNode := &plugin.ConnectionTreeNode{
				Key:      schemaName,
				Label:    schemaName,
				NodeType: plugin.ConnectionTreeNodeTypeSchema,
				Children: categories,
			}
			schemaNodes = append(schemaNodes, schemaNode)
		}
		return schemaNodes
	}

	var dbNodes []*plugin.ConnectionTreeNode
	for _, dbname := range dbNames {
		var schemas []*plugin.ConnectionTreeNode
		if dbname == currentDB {
			schemas = loadSchemas(db)
		} else {
			connMap := make(map[string]string)
			for k, v := range req.Connection {
				connMap[k] = v
			}
			connMap["database"] = dbname
			if dsn2, err := buildConnString(connMap); err == nil && dsn2 != "" {
				if db2, err2 := openPostgresDB(dsn2); err2 == nil {
					schemas = loadSchemas(db2)
					db2.Close()
				}
			}
		}
		node := &plugin.ConnectionTreeNode{
			Key:      dbname,
			Label:    dbname,
			NodeType: plugin.ConnectionTreeNodeTypeDatabase,
			Children: schemas,
			Actions: []*plugin.ConnectionTreeAction{
				{
					Type:  plugin.ConnectionTreeActionDropDatabase,
					Title: "Drop database",
					Query: fmt.Sprintf(`DROP DATABASE "%s";`, dbname),
				},
			},
		}
		dbNodes = append(dbNodes, node)
	}

	createNode := &plugin.ConnectionTreeNode{
		Key:      "__create_database__",
		Label:    "New database",
		NodeType: plugin.ConnectionTreeNodeTypeAction,
		Actions: []*plugin.ConnectionTreeAction{
			{
				Type:  plugin.ConnectionTreeActionCreateDatabase,
				Title: "Create database",
				Query: `CREATE DATABASE "new_database";`,
				Hidden: true,
			},
		},
	}

	return &plugin.ConnectionTreeResponse{Nodes: append([]*plugin.ConnectionTreeNode{createNode}, dbNodes...)}, nil
}

// formatPingError wraps a ping failure with supplemental hints when the
// underlying error indicates an SSL mis‑match.  It is public for testing.
func formatPingError(err error) string {
	msg := fmt.Sprintf("ping error: %v", err)
	if err != nil && strings.Contains(err.Error(), "SSL is not enabled on the server") {
		msg += " (hint: server has SSL disabled; set sslmode=disable or enable SSL on the server)"
	}
	return msg
}


// TestConnection opens a PostgreSQL connection and pings the server to verify
// the supplied credentials are valid. Nothing is persisted.
// GetCompletionFields returns column names and types for the given table,
// enabling context-aware auto-completion in the editor.
func (m *postgresqlPlugin) GetCompletionFields(ctx context.Context, req *plugin.GetCompletionFieldsRequest) (*plugin.GetCompletionFieldsResponse, error) {
	if req.Collection == "" {
		return &plugin.GetCompletionFieldsResponse{}, nil
	}
	dsn, err := buildConnString(req.Connection)
	if err != nil || dsn == "" {
		return &plugin.GetCompletionFieldsResponse{}, nil
	}
	db, err := openPostgresDB(dsn)
	if err != nil {
		return &plugin.GetCompletionFieldsResponse{}, nil
	}
	defer db.Close()

	// Support both "schema.table" and plain "table" forms.
	schemaName := ""
	tableName := req.Collection
	if parts := strings.SplitN(req.Collection, ".", 2); len(parts) == 2 {
		schemaName = parts[0]
		tableName = parts[1]
	}

	// When no schema is known, ask the live connection for its current search-path
	// schema so we don't hard-code "public" (schemas like dbo, myapp, etc. exist).
	if schemaName == "" {
		_ = db.QueryRowContext(ctx, "SELECT current_schema()").Scan(&schemaName)
	}
	if schemaName == "" {
		schemaName = "public"
	}

	colQ := `SELECT column_name, data_type
			 FROM information_schema.columns
			 WHERE table_schema=$1 AND table_name=$2
			 ORDER BY ordinal_position`
	rows, err := db.QueryContext(ctx, colQ, schemaName, tableName)
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

func (m *postgresqlPlugin) TestConnection(ctx context.Context, req *plugin.TestConnectionRequest) (*plugin.TestConnectionResponse, error) {
	dsn, err := buildConnString(req.Connection)
	if err != nil || dsn == "" {
		msg := "invalid connection parameters"
		if err != nil {
			msg = err.Error()
		}
		return &plugin.TestConnectionResponse{Ok: false, Message: msg}, nil
	}
	db, err := openPostgresDB(dsn)
	if err != nil {
		return &plugin.TestConnectionResponse{Ok: false, Message: fmt.Sprintf("open error: %v", err)}, nil
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		return &plugin.TestConnectionResponse{Ok: false, Message: formatPingError(err)}, nil
	}
	return &plugin.TestConnectionResponse{Ok: true, Message: "Connection successful"}, nil
}

func main() {
	plugin.ServeCLI(&postgresqlPlugin{})
}
