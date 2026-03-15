package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	driver "github.com/arangodb/go-driver"
	driverHttp "github.com/arangodb/go-driver/http"
	"github.com/felixdotgo/querybox/pkg/certs"
	"github.com/felixdotgo/querybox/pkg/plugin"
	pluginpb "github.com/felixdotgo/querybox/rpc/contracts/plugin/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// arangoPlugin implements the protobuf PluginServiceServer interface for ArangoDB.
// embedding the unimplemented server ensures compatibility with future additions.
type arangoPlugin struct {
	pluginpb.UnimplementedPluginServiceServer
}

func (a *arangoPlugin) Info(ctx context.Context, _ *pluginpb.PluginV1_InfoRequest) (*plugin.InfoResponse, error) {
	return &plugin.InfoResponse{
		Type:        plugin.TypeDriver,
		Name:        "ArangoDB",
		Version:     "0.1.0",
		Description: "ArangoDB multi-model database driver",
		Url:         "https://www.arangodb.com/",
		Author:      "ArangoDB GmbH",
		Capabilities: []string{"query"},
		Tags:        []string{"nosql", "multi-model"},
		License:     "Apache-2.0",
		IconUrl:     "https://www.arangodb.com/wp-content/uploads/2019/03/arangodb-logo.png",
	}, nil
}

func (a *arangoPlugin) DescribeSchema(ctx context.Context, req *plugin.DescribeSchemaRequest) (*plugin.DescribeSchemaResponse, error) {
    // ArangoDB is schemaless; return an empty descriptor.
    return &plugin.DescribeSchemaResponse{}, nil
}

func (a *arangoPlugin) AuthForms(ctx context.Context, _ *plugin.AuthFormsRequest) (*plugin.AuthFormsResponse, error) {
	basic := plugin.AuthForm{
		Key:  "basic",
		Name: "Basic",
		Fields: []*plugin.AuthField{
			{Type: plugin.AuthFieldText, Name: "host", Label: "Host", Required: true, Placeholder: "127.0.0.1", Value: "127.0.0.1"},
			{Type: plugin.AuthFieldNumber, Name: "port", Label: "Port", Placeholder: "8529", Value: "8529"},
			{Type: plugin.AuthFieldText, Name: "user", Label: "User", Value: "root"},
			{Type: plugin.AuthFieldPassword, Name: "password", Label: "Password"},
			{Type: plugin.AuthFieldText, Name: "database", Label: "Database", Value: "_system"},
			{Type: plugin.AuthFieldSelect, Name: "tls", Label: "TLS", Options: []string{"false", "true"}, Value: "false"},
		},
	}
	return &plugin.AuthFormsResponse{Forms: map[string]*plugin.AuthForm{"basic": &basic}}, nil
}

// connParams holds the parsed connection parameters extracted from
// the connection map supplied by the host.
type connParams struct {
	host     string
	port     string
	user     string
	password string
	database string
	tls      bool
}

// parseConnParams extracts connection parameters from the host-supplied map.
// It understands both the legacy flat map and the recommended credential_blob
// JSON payload.
func parseConnParams(connection map[string]string) (connParams, error) {
	p := connParams{
		host:     "127.0.0.1",
		port:     "8529",
		user:     "root",
		database: "_system",
	}

	blob := connection["credential_blob"]
	if blob == "" {
		// Try direct keys as fallback (legacy).
		if h := connection["host"]; h != "" {
			p.host = h
		}
		if port := connection["port"]; port != "" {
			p.port = port
		}
		p.user = connection["user"]
		p.password = connection["password"]
		if db := connection["database"]; db != "" {
			p.database = db
		}
		return p, nil
	}

	cred, err := plugin.ParseCredentialBlob(connection)
	if err != nil {
		return p, fmt.Errorf("invalid credential blob: %w", err)
	}

	if h := cred.Values["host"]; h != "" {
		p.host = h
	}
	if port := cred.Values["port"]; port != "" {
		p.port = port
	}
	if u := cred.Values["user"]; u != "" {
		p.user = u
	}
	p.password = cred.Values["password"]
	if db := cred.Values["database"]; db != "" {
		p.database = db
	}
	p.tls = cred.Values["tls"] == "true"
	return p, nil
}

// buildClient creates an ArangoDB client from the supplied connection params.
func buildClient(p connParams) (driver.Client, error) {
	scheme := "http"
	if p.tls {
		scheme = "https"
	}
	endpoint := fmt.Sprintf("%s://%s:%s", scheme, p.host, p.port)

	var transport driver.Connection
	var err error

	// Use a custom http.Transport.  If TLS is requested we populate
	// TLSClientConfig.RootCAs with our embedded root bundle so that the
	// connection will verify the server certificate.  In development we
	// still allow the caller to disable verification by setting
	// "tls" to false; production users should supply a valid cert.
	var httpTrans = &http.Transport{}
	if p.tls {
		if pool, err2 := certs.RootCertPool(); err2 == nil {
			httpTrans.TLSClientConfig = &tls.Config{RootCAs: pool}
		}
	}
	transport, err = driverHttp.NewConnection(driverHttp.ConnectionConfig{
		Endpoints: []string{endpoint},
		Transport: httpTrans,
	})
	if err != nil {
		return nil, fmt.Errorf("create connection: %w", err)
	}

	c, err := driver.NewClient(driver.ClientConfig{
		Connection:     transport,
		Authentication: driver.BasicAuthentication(p.user, p.password),
	})
	if err != nil {
		return nil, fmt.Errorf("create client: %w", err)
	}
	return c, nil
}

// valueToStruct converts any AQL result value into a *structpb.Struct suitable
// for inclusion in a DocumentResult payload.  AQL can return objects, scalars,
// or arrays, so we normalise each case:
//   - map   → direct conversion via structpb.NewStruct
//   - other → wrapped as {"value": <v>} so the frontend always receives an
//     object-shaped payload
func valueToStruct(v interface{}) (*structpb.Struct, error) {
	// Re-encode through JSON to get a fully normalised Go value that
	// structpb.NewStruct can handle (e.g. float64 instead of int).
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var decoded interface{}
	if err := json.Unmarshal(b, &decoded); err != nil {
		return nil, err
	}
	if m, ok := decoded.(map[string]interface{}); ok {
		return structpb.NewStruct(m)
	}
	// Scalar or array: wrap so the payload is always an object.
	return structpb.NewStruct(map[string]interface{}{"value": decoded})
}

// ddlPattern matches simple DDL meta-commands that ArangoDB AQL does not
// natively support.  Exec intercepts these before sending to the AQL engine.
//
//	CREATE DATABASE <name>
//	DROP   DATABASE <name>
//	CREATE COLLECTION <db>.<name>
//	DROP   COLLECTION <db>.<name>
//
// For COLLECTION operations the name field uses a <db>.<collection> format so
// the target database is unambiguous regardless of the connection default.
var ddlPattern = regexp.MustCompile(`(?i)^\s*(CREATE|DROP)\s+(DATABASE|COLLECTION)\s+(\S+)\s*;?\s*$`)

// execDDL handles the four recognised DDL meta-commands.  It returns (result,
// handled, error).  Callers should only use result when handled is true.
func (a *arangoPlugin) execDDL(ctx context.Context, client driver.Client, p connParams, query string) (*plugin.ExecResponse, bool) {
	m := ddlPattern.FindStringSubmatch(query)
	if m == nil {
		return nil, false
	}
	op, kind, name := strings.ToUpper(m[1]), strings.ToUpper(m[2]), m[3]

	kvResult := func(msg string) *plugin.ExecResponse {
		return &plugin.ExecResponse{
			Result: &plugin.ExecResult{
				Payload: &pluginpb.PluginV1_ExecResult_Kv{
					Kv: &plugin.KeyValueResult{Data: map[string]string{"result": msg}},
				},
			},
		}
	}
	errResult := func(msg string) *plugin.ExecResponse {
		return &plugin.ExecResponse{Error: msg}
	}

	switch {
	case op == "CREATE" && kind == "DATABASE":
		if _, err := client.CreateDatabase(ctx, name, nil); err != nil {
			return errResult(fmt.Sprintf("create database %q: %v", name, err)), true
		}
		return kvResult(fmt.Sprintf("Database %q created.", name)), true

	case op == "DROP" && kind == "DATABASE":
		db, err := client.Database(ctx, name)
		if err != nil {
			return errResult(fmt.Sprintf("open database %q: %v", name, err)), true
		}
		if err := db.Remove(ctx); err != nil {
			return errResult(fmt.Sprintf("drop database %q: %v", name, err)), true
		}
		return kvResult(fmt.Sprintf("Database %q dropped.", name)), true

	case op == "CREATE" && kind == "COLLECTION":
		// name is encoded as "<db>.<collection>" so the target database is
		// explicit.  Fall back to the connection default when the dot is absent.
		dbName, collName := splitDBColl(name, p.database)
		db, err := client.Database(ctx, dbName)
		if err != nil {
			return errResult(fmt.Sprintf("open database %q: %v", dbName, err)), true
		}
		if _, err := db.CreateCollection(ctx, collName, nil); err != nil {
			return errResult(fmt.Sprintf("create collection %q: %v", collName, err)), true
		}
		return kvResult(fmt.Sprintf("Collection %q created in database %q.", collName, dbName)), true

	case op == "DROP" && kind == "COLLECTION":
		// name is encoded as "<db>.<collection>" so the target database is
		// explicit.  Fall back to the connection default when the dot is absent.
		dbName, collName := splitDBColl(name, p.database)
		db, err := client.Database(ctx, dbName)
		if err != nil {
			return errResult(fmt.Sprintf("open database %q: %v", dbName, err)), true
		}
		coll, err := db.Collection(ctx, collName)
		if err != nil {
			return errResult(fmt.Sprintf("open collection %q: %v", collName, err)), true
		}
		if err := coll.Remove(ctx); err != nil {
			return errResult(fmt.Sprintf("drop collection %q: %v", collName, err)), true
		}
		return kvResult(fmt.Sprintf("Collection %q dropped from database %q.", collName, dbName)), true
	}

	return nil, false
}

// splitDBColl splits a "<db>.<collection>" token into (db, collection).
// When there is no dot, the caller-supplied default db is returned.
func splitDBColl(name, defaultDB string) (string, string) {
	if idx := strings.IndexByte(name, '.'); idx > 0 && idx < len(name)-1 {
		return name[:idx], name[idx+1:]
	}
	return defaultDB, name
}

// splitDBFromQuery looks for a simple qualified collection reference
// at the start of an AQL FOR statement (e.g. "FOR x IN db.coll …") and, if
// present, returns the database name along with a rewritten query that has the
// qualification removed.  This allows the host to show a fully qualified
// query in the tree while still executing against the correct database
// without requiring the connection itself to change.
//
// Only the first occurrence is rewritten; more complex AQL (multiple collections,
// LET expressions, subqueries, etc.) is left untouched.  The heuristic is
// intentionally simple since we only need to satisfy the connection‑tree
// templates and user‑supplied queries like "FOR d IN mydb.coll RETURN d".
var qualifiedCollRE = regexp.MustCompile(`(?i)\bIN\s*([A-Za-z0-9_-]+)\.([A-Za-z0-9_-]+)`) // allow zero or more spaces after IN

func splitDBFromQuery(query string) (dbName, rewritten string) {
	m := qualifiedCollRE.FindStringSubmatch(query)
	if m == nil {
		return "", query
	}
	// replace only the first occurrence so the rest of the query stays intact
	rewritten = qualifiedCollRE.ReplaceAllString(query, "IN $2")
	return m[1], rewritten
}

func (a *arangoPlugin) Exec(ctx context.Context, req *plugin.ExecRequest) (*plugin.ExecResponse, error) {
	p, err := parseConnParams(req.Connection)
	if err != nil {
		return &plugin.ExecResponse{Error: fmt.Sprintf("connection error: %v", err)}, nil
	}

	client, err := buildClient(p)
	if err != nil {
		return &plugin.ExecResponse{Error: fmt.Sprintf("client error: %v", err)}, nil
	}

	// Intercept DDL meta-commands (CREATE/DROP DATABASE|COLLECTION) before
	// passing the query to the AQL engine, which does not support DDL.
	if res, handled := a.execDDL(ctx, client, p, req.Query); handled {
		return res, nil
	}

	// adjust the target database if the user qualified the collection name
	dbName := p.database
	queryText := req.Query
	if d, q := splitDBFromQuery(queryText); d != "" {
		// only treat the prefix as a database if we can successfully open it.
		// otherwise the user is probably querying a collection whose name
		// contains a dot (which is legal) and we must not rewrite it.
		if _, err := client.Database(ctx, d); err == nil {
			dbName = d
			queryText = q
		}
	}

	db, err := client.Database(ctx, dbName)
	if err != nil {
		return &plugin.ExecResponse{Error: fmt.Sprintf("open database %q: %v", dbName, err)}, nil
	}

	cursor, err := db.Query(ctx, queryText, nil)
	if err != nil {
		return &plugin.ExecResponse{Error: fmt.Sprintf("query error: %v", err)}, nil
	}
	defer cursor.Close()

	var documents []*structpb.Struct
	for cursor.HasMore() {
		// Read into interface{} so scalars, arrays, and objects are all handled
		// gracefully; map values are converted directly, everything else is
		// wrapped under a "value" key.
		var raw interface{}
		if _, err := cursor.ReadDocument(ctx, &raw); err != nil {
			return &plugin.ExecResponse{Error: fmt.Sprintf("read error: %v", err)}, nil
		}
		s, err := valueToStruct(raw)
		if err != nil {
			s, _ = structpb.NewStruct(map[string]interface{}{"_raw": fmt.Sprintf("%v", raw)})
		}
		documents = append(documents, s)
	}

	return &plugin.ExecResponse{
		Result: &plugin.ExecResult{
			Payload: &pluginpb.PluginV1_ExecResult_Document{
				Document: &plugin.DocumentResult{Documents: documents},
			},
		},
	}, nil
}

// ConnectionTree returns a server → database → collection hierarchy.
// DDL actions are exposed at the server (create database), database (drop
// database, create collection) and collection (drop collection) levels.
// The query templates use the DDL meta-commands intercepted by Exec.
// explicitDatabase checks the raw connection map to see if the user
// explicitly supplied a database name.  parseConnParams always populates a
// default value ("_system"), so we cannot rely on its result alone for this
// policy decision.
func explicitDatabase(conn map[string]string) string {
	if db, ok := conn["database"]; ok && db != "" {
		return db
	}
	if blob, ok := conn["credential_blob"]; ok && blob != "" {
		if cred, err := plugin.ParseCredentialBlob(conn); err == nil {
			if db := cred.Values["database"]; db != "" {
				return db
			}
		}
	}
	return ""
}

func (a *arangoPlugin) ConnectionTree(ctx context.Context, req *plugin.ConnectionTreeRequest) (*plugin.ConnectionTreeResponse, error) {
	p, err := parseConnParams(req.Connection)
	if err != nil {
		return &plugin.ConnectionTreeResponse{}, nil
	}

	client, err := buildClient(p)
	if err != nil {
		return &plugin.ConnectionTreeResponse{}, nil
	}

	// if the user requested a particular database explicitly, just build a
	// tree for that database rather than enumerating all accessible ones.
	if db := explicitDatabase(req.Connection); db != "" {
		return a.singleDatabaseTree(ctx, client, db), nil
	}

	// List all accessible databases.
	databases, err := client.AccessibleDatabases(ctx)
	if err != nil {
		// Fallback: only show the configured database.
		return a.singleDatabaseTree(ctx, client, p.database), nil
	}

	var dbNodes []*plugin.ConnectionTreeNode
	for _, db := range databases {
		dbName := db.Name()
		collNodes := a.collectionNodes(ctx, db, dbName)
		dbNodes = append(dbNodes, &plugin.ConnectionTreeNode{
			Key:      dbName,
			Label:    dbName,
			NodeType: plugin.ConnectionTreeNodeTypeDatabase,
			Children: collNodes,
			Actions: []*plugin.ConnectionTreeAction{
				{Type: plugin.ConnectionTreeActionCreateTable, Title: "Create collection", Query: fmt.Sprintf("CREATE COLLECTION %s.new_collection", dbName)},
				{Type: plugin.ConnectionTreeActionDropDatabase, Title: "Drop database", Query: fmt.Sprintf("DROP DATABASE %s", dbName)},
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
			{Type: plugin.ConnectionTreeActionCreateDatabase, Title: "Create database", Query: "CREATE DATABASE new_database", Hidden: true},
		},
	}

	return &plugin.ConnectionTreeResponse{Nodes: append([]*plugin.ConnectionTreeNode{createNode}, dbNodes...)}, nil
}

// singleDatabaseTree builds a tree for a single named database when the user
// lacks permissions to list all accessible databases.
func (a *arangoPlugin) singleDatabaseTree(ctx context.Context, client driver.Client, dbName string) *plugin.ConnectionTreeResponse {
	db, err := client.Database(ctx, dbName)
	if err != nil {
		return &plugin.ConnectionTreeResponse{}
	}
	dbNode := &plugin.ConnectionTreeNode{
		Key:      dbName,
		Label:    dbName,
		NodeType: plugin.ConnectionTreeNodeTypeDatabase,
		Children: a.collectionNodes(ctx, db, dbName),
		Actions: []*plugin.ConnectionTreeAction{
			{Type: plugin.ConnectionTreeActionCreateTable, Title: "Create collection", Query: "CREATE COLLECTION new_collection"},
			{Type: plugin.ConnectionTreeActionDropDatabase, Title: "Drop database", Query: fmt.Sprintf("DROP DATABASE %s", dbName)},
		},
	}
	createNode := &plugin.ConnectionTreeNode{
		Key:      "__create_database__",
		Label:    "New database",
		NodeType: plugin.ConnectionTreeNodeTypeAction,
		Actions: []*plugin.ConnectionTreeAction{
			{Type: plugin.ConnectionTreeActionCreateDatabase, Title: "Create database", Query: "CREATE DATABASE new_database", Hidden: true},
		},
	}
	return &plugin.ConnectionTreeResponse{Nodes: []*plugin.ConnectionTreeNode{createNode, dbNode}}
}

// collectionNodes returns tree nodes for user collections inside db.
func (a *arangoPlugin) collectionNodes(ctx context.Context, db driver.Database, dbName string) []*plugin.ConnectionTreeNode {
	colls, err := db.Collections(ctx)
	if err != nil {
		return nil
	}

	var nodes []*plugin.ConnectionTreeNode
	for _, coll := range colls {
		name := coll.Name()
		// Skip ArangoDB internal collections (prefixed with "_").
		if strings.HasPrefix(name, "_") {
			continue
		}
		// when the user clicks "Select documents" we want to make it obvious
		// which database the collection lives in.  The Exec path will strip the
		// qualification and switch to the correct database before running the
		// query.
		qualified := fmt.Sprintf("%s.%s", dbName, name)
		nodes = append(nodes, &plugin.ConnectionTreeNode{
			Key:      qualified,
			Label:    name,
			NodeType: plugin.ConnectionTreeNodeTypeCollection,
			Actions: []*plugin.ConnectionTreeAction{
				{
					Type:   plugin.ConnectionTreeActionSelect,
					Title:  "Select documents",
					Query:  fmt.Sprintf("FOR doc IN %s LIMIT 100 RETURN doc", qualified),
					Hidden: true,
					NewTab: true,
				},
				{
					Type:  plugin.ConnectionTreeActionDropTable,
					Title: "Drop collection",
					Query: fmt.Sprintf("DROP COLLECTION %s.%s", dbName, name),
				},
			},
		})
	}
	return nodes
}

// GetCompletionFields samples field names from a collection using AQL
// ATTRIBUTES() so the editor autocomplete can offer field suggestions.
// This satisfies the plugin.CompletionFieldsProvider interface.
func (a *arangoPlugin) GetCompletionFields(ctx context.Context, req *pluginpb.PluginV1_GetCompletionFieldsRequest) (*pluginpb.PluginV1_GetCompletionFieldsResponse, error) {
	p, err := parseConnParams(req.Connection)
	if err != nil {
		return &pluginpb.PluginV1_GetCompletionFieldsResponse{}, nil
	}
	client, err := buildClient(p)
	if err != nil {
		return &pluginpb.PluginV1_GetCompletionFieldsResponse{}, nil
	}

	dbName := req.Database
	if dbName == "" {
		dbName = p.database
	}
	collName := req.Collection
	if collName == "" {
		return &pluginpb.PluginV1_GetCompletionFieldsResponse{}, nil
	}

	db, err := client.Database(ctx, dbName)
	if err != nil {
		return &pluginpb.PluginV1_GetCompletionFieldsResponse{}, nil
	}

	// ATTRIBUTES(doc, true) returns all attribute names including nested ones.
	// We flatten the result into top-level names only for simplicity.
	aql := fmt.Sprintf(
		"FOR doc IN `%s` LIMIT 100 FOR attr IN ATTRIBUTES(doc, true) RETURN DISTINCT attr",
		collName,
	)
	cursor, err := db.Query(ctx, aql, nil)
	if err != nil {
		return &pluginpb.PluginV1_GetCompletionFieldsResponse{}, nil
	}
	defer cursor.Close()

	fieldSet := make(map[string]struct{})
	for cursor.HasMore() {
		var name string
		if _, err := cursor.ReadDocument(ctx, &name); err != nil {
			continue
		}
		if name != "" {
			fieldSet[name] = struct{}{}
		}
	}

	fields := make([]*pluginpb.PluginV1_FieldInfo, 0, len(fieldSet))
	for name := range fieldSet {
		fields = append(fields, &pluginpb.PluginV1_FieldInfo{Name: name})
	}

	return &pluginpb.PluginV1_GetCompletionFieldsResponse{Fields: fields}, nil
}

// TestConnection verifies the ArangoDB connection.
// It first attempts a server-level version check.  If that is denied (which
// can happen for non-root users whose credentials are scoped to a specific
// database), it falls back to opening the configured database and calling
// Info, which validates the credentials and confirms database accessibility.
func (a *arangoPlugin) TestConnection(ctx context.Context, req *plugin.TestConnectionRequest) (*plugin.TestConnectionResponse, error) {
	p, err := parseConnParams(req.Connection)
	if err != nil {
		return &plugin.TestConnectionResponse{Ok: false, Message: err.Error()}, nil
	}

	client, err := buildClient(p)
	if err != nil {
		return &plugin.TestConnectionResponse{Ok: false, Message: fmt.Sprintf("client error: %v", err)}, nil
	}

	// Prefer the version endpoint — it requires no database context and gives
	// a rich confirmation message.  Non-root users may not have access to it,
	// so on any error we fall through to a database-scoped check instead of
	// failing immediately.
	if v, vErr := client.Version(ctx); vErr == nil {
		return &plugin.TestConnectionResponse{
			Ok:      true,
			Message: fmt.Sprintf("Connection successful (ArangoDB %s)", v.Version),
		}, nil
	}

	// Version check unavailable — verify connectivity via database access.
	db, err := client.Database(ctx, p.database)
	if err != nil {
		return &plugin.TestConnectionResponse{Ok: false, Message: fmt.Sprintf("connection error: %v", err)}, nil
	}
	info, err := db.Info(ctx)
	if err != nil {
		return &plugin.TestConnectionResponse{Ok: false, Message: fmt.Sprintf("database access error: %v", err)}, nil
	}
	return &plugin.TestConnectionResponse{
		Ok:      true,
		Message: fmt.Sprintf("Connection successful (database: %s)", info.Name),
	}, nil
}

// MutateRow stub for ArangoDB driver; always reports not supported.
func (a *arangoPlugin) MutateRow(ctx context.Context, req *plugin.MutateRowRequest) (*plugin.MutateRowResponse, error) {
	return &plugin.MutateRowResponse{Error: "not supported"}, nil
}

func main() {
	plugin.ServeCLI(&arangoPlugin{})
}
