package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	driver "github.com/arangodb/go-driver"
	driverHttp "github.com/arangodb/go-driver/http"
	"github.com/felixdotgo/querybox/pkg/plugin"
	pluginpb "github.com/felixdotgo/querybox/rpc/contracts/plugin/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// arangoPlugin implements the plugin.Plugin interface for ArangoDB.
type arangoPlugin struct{}

func (a *arangoPlugin) Info() (plugin.InfoResponse, error) {
	return plugin.InfoResponse{
		Type:        plugin.TypeDriver,
		Name:        "ArangoDB",
		Version:     "0.1.0",
		Description: "ArangoDB multi-model database driver",
	}, nil
}

func (a *arangoPlugin) AuthForms(*plugin.AuthFormsRequest) (*plugin.AuthFormsResponse, error) {
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

	blob, ok := connection["credential_blob"]
	if !ok || blob == "" {
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

	var payload struct {
		Form   string            `json:"form"`
		Values map[string]string `json:"values"`
	}
	if err := json.Unmarshal([]byte(blob), &payload); err != nil {
		return p, fmt.Errorf("invalid credential blob: %w", err)
	}

	if h := payload.Values["host"]; h != "" {
		p.host = h
	}
	if port := payload.Values["port"]; port != "" {
		p.port = port
	}
	if u := payload.Values["user"]; u != "" {
		p.user = u
	}
	p.password = payload.Values["password"]
	if db := payload.Values["database"]; db != "" {
		p.database = db
	}
	p.tls = payload.Values["tls"] == "true"
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

	// Use a custom http.Transport to allow skipping TLS verification in dev
	// environments; production users should supply a valid certificate instead.
	transport, err = driverHttp.NewConnection(driverHttp.ConnectionConfig{
		Endpoints: []string{endpoint},
		Transport: &http.Transport{},
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

func (a *arangoPlugin) Exec(req *plugin.ExecRequest) (*plugin.ExecResponse, error) {
	p, err := parseConnParams(req.Connection)
	if err != nil {
		return &plugin.ExecResponse{Error: fmt.Sprintf("connection error: %v", err)}, nil
	}

	client, err := buildClient(p)
	if err != nil {
		return &plugin.ExecResponse{Error: fmt.Sprintf("client error: %v", err)}, nil
	}

	ctx := context.Background()
	db, err := client.Database(ctx, p.database)
	if err != nil {
		return &plugin.ExecResponse{Error: fmt.Sprintf("open database %q: %v", p.database, err)}, nil
	}

	cursor, err := db.Query(ctx, req.Query, nil)
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

// ConnectionTree returns a two-level hierarchy: databases → collections.
func (a *arangoPlugin) ConnectionTree(req *plugin.ConnectionTreeRequest) (*plugin.ConnectionTreeResponse, error) {
	p, err := parseConnParams(req.Connection)
	if err != nil {
		return &plugin.ConnectionTreeResponse{}, nil
	}

	client, err := buildClient(p)
	if err != nil {
		return &plugin.ConnectionTreeResponse{}, nil
	}

	ctx := context.Background()

	// List all accessible databases.
	databases, err := client.AccessibleDatabases(ctx)
	if err != nil {
		// Fallback: only show the configured database.
		return a.singleDatabaseTree(ctx, client, p.database), nil
	}

	var nodes []*plugin.ConnectionTreeNode
	for _, db := range databases {
		dbName := db.Name()
		collNodes := a.collectionNodes(ctx, db, dbName)
		nodes = append(nodes, &plugin.ConnectionTreeNode{
			Key:      dbName,
			Label:    dbName,
			NodeType: "database",
			Children: collNodes,
		})
	}

	return &plugin.ConnectionTreeResponse{Nodes: nodes}, nil
}

// singleDatabaseTree builds a tree for a single named database when the user
// lacks permissions to list all accessible databases.
func (a *arangoPlugin) singleDatabaseTree(ctx context.Context, client driver.Client, dbName string) *plugin.ConnectionTreeResponse {
	db, err := client.Database(ctx, dbName)
	if err != nil {
		return &plugin.ConnectionTreeResponse{}
	}
	return &plugin.ConnectionTreeResponse{
		Nodes: []*plugin.ConnectionTreeNode{
			{
				Key:      dbName,
				Label:    dbName,
				NodeType: "database",
				Children: a.collectionNodes(ctx, db, dbName),
			},
		},
	}
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
		nodes = append(nodes, &plugin.ConnectionTreeNode{
			Key:      dbName + "." + name,
			Label:    name,
			NodeType: "collection",
			Actions: []*plugin.ConnectionTreeAction{
				{
					Type:  plugin.ConnectionTreeActionSelect,
					Title: name,
					Query: fmt.Sprintf("FOR doc IN %s LIMIT 100 RETURN doc", name),
				},
			},
		})
	}
	return nodes
}

// TestConnection verifies the ArangoDB connection by checking server version.
func (a *arangoPlugin) TestConnection(req *plugin.TestConnectionRequest) (*plugin.TestConnectionResponse, error) {
	p, err := parseConnParams(req.Connection)
	if err != nil {
		return &plugin.TestConnectionResponse{Ok: false, Message: err.Error()}, nil
	}

	client, err := buildClient(p)
	if err != nil {
		return &plugin.TestConnectionResponse{Ok: false, Message: fmt.Sprintf("client error: %v", err)}, nil
	}

	ctx := context.Background()
	v, err := client.Version(ctx)
	if err != nil {
		return &plugin.TestConnectionResponse{Ok: false, Message: fmt.Sprintf("version check error: %v", err)}, nil
	}
	return &plugin.TestConnectionResponse{
		Ok:      true,
		Message: fmt.Sprintf("Connection successful (ArangoDB %s)", v.Version),
	}, nil
}

func main() {
	plugin.ServeCLI(&arangoPlugin{})
}
