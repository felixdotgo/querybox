package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/felixdotgo/querybox/pkg/certs"
	"github.com/felixdotgo/querybox/pkg/plugin"
	pluginpb "github.com/felixdotgo/querybox/rpc/contracts/plugin/v1"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/protobuf/types/known/structpb"
)

// mongoPlugin implements the protobuf PluginServiceServer interface for MongoDB.
type mongoPlugin struct {
	pluginpb.UnimplementedPluginServiceServer
}

func (m *mongoPlugin) Info(ctx context.Context, _ *pluginpb.PluginV1_InfoRequest) (*plugin.InfoResponse, error) {
	return &plugin.InfoResponse{
		Type:         plugin.TypeDriver,
		Name:         "MongoDB",
		Version:      "0.1.0",
		Description:  "MongoDB document database driver",
		Url:          "https://www.mongodb.com/",
		Author:       "MongoDB Inc.",
		Capabilities: []string{"query"},
		Tags:         []string{"nosql", "document"},
		License:      "Apache-2.0",
		IconUrl:      "https://www.mongodb.com/assets/images/global/favicon.ico",
		Contact:      "https://www.mongodb.com/community/forums/",
	}, nil
}

func (m *mongoPlugin) AuthForms(ctx context.Context, _ *plugin.AuthFormsRequest) (*plugin.AuthFormsResponse, error) {
	basic := plugin.AuthForm{
		Key:  "basic",
		Name: "Basic",
		Fields: []*plugin.AuthField{
			{Type: plugin.AuthFieldText, Name: "host", Label: "Host", Required: true, Placeholder: "127.0.0.1", Value: "127.0.0.1"},
			{Type: plugin.AuthFieldNumber, Name: "port", Label: "Port", Placeholder: "27017", Value: "27017"},
			{Type: plugin.AuthFieldText, Name: "user", Label: "Username"},
			{Type: plugin.AuthFieldPassword, Name: "password", Label: "Password"},
			{Type: plugin.AuthFieldText, Name: "database", Label: "Database", Placeholder: "mydb"},
			{Type: plugin.AuthFieldText, Name: "auth_source", Label: "Auth Source", Placeholder: "admin", Value: "admin"},
			{Type: plugin.AuthFieldSelect, Name: "tls", Label: "TLS", Options: []string{"false", "true"}, Value: "false"},
		},
	}
	uri := plugin.AuthForm{
		Key:  "uri",
		Name: "URI",
		Fields: []*plugin.AuthField{
			{Type: plugin.AuthFieldText, Name: "uri", Label: "MongoDB URI", Required: true, Placeholder: "mongodb://user:pass@localhost:27017/mydb"},
		},
	}
	return &plugin.AuthFormsResponse{Forms: map[string]*plugin.AuthForm{"basic": &basic, "uri": &uri}}, nil
}

// credentialPayload is the JSON shape stored in connection["credential_blob"].
type credentialPayload struct {
	Form   string            `json:"form"`
	Values map[string]string `json:"values"`
}

// buildURI constructs a MongoDB connection URI from the connection map.
// Returns the URI string, the explicitly configured database name, and any error.
func buildURI(connection map[string]string) (string, string, error) {
	// Direct URI key takes precedence.
	if u, ok := connection["uri"]; ok && u != "" {
		return u, "", nil
	}

	if blob, ok := connection["credential_blob"]; ok && blob != "" {
		var payload credentialPayload
		if err := json.Unmarshal([]byte(blob), &payload); err != nil {
			return "", "", fmt.Errorf("invalid credential blob: %w", err)
		}
		if u, ok := payload.Values["uri"]; ok && u != "" {
			return u, "", nil
		}
		return buildURIFromValues(payload.Values)
	}

	return buildURIFromValues(connection)
}

// buildURIFromValues constructs a MongoDB URI from a flat key/value map.
func buildURIFromValues(values map[string]string) (string, string, error) {
	host := values["host"]
	if host == "" {
		host = "127.0.0.1"
	}
	port := values["port"]
	if port == "" {
		port = "27017"
	}
	user := values["user"]
	pass := values["password"]
	dbname := values["database"]
	authSource := values["auth_source"]
	if authSource == "" {
		authSource = "admin"
	}
	tlsMode := values["tls"]

	u := url.URL{
		Scheme: "mongodb",
		Host:   fmt.Sprintf("%s:%s", host, port),
	}
	if user != "" {
		u.User = url.UserPassword(user, pass)
	}
	if dbname != "" {
		u.Path = "/" + dbname
	}
	q := url.Values{}
	if user != "" {
		q.Set("authSource", authSource)
	}
	if tlsMode == "true" {
		q.Set("tls", "true")
	}
	if len(q) > 0 {
		u.RawQuery = q.Encode()
	}
	return u.String(), dbname, nil
}

// getDatabaseName returns the database name from the connection map, if specified.
func getDatabaseName(connection map[string]string) string {
	if blob, ok := connection["credential_blob"]; ok && blob != "" {
		var payload credentialPayload
		if json.Unmarshal([]byte(blob), &payload) == nil {
			if d := payload.Values["database"]; d != "" {
				return d
			}
		}
	}
	return connection["database"]
}

// connectMongo builds a *mongo.Client from the connection map.
// The caller is responsible for calling client.Disconnect.
func connectMongo(ctx context.Context, connection map[string]string) (*mongo.Client, string, error) {
	uri, dbname, err := buildURI(connection)
	if err != nil {
		return nil, "", err
	}
	if uri == "" {
		return nil, "", fmt.Errorf("missing connection parameters")
	}

	opts := options.Client().ApplyURI(uri)

	// Attach embedded root CA pool when TLS is requested.
	if strings.Contains(uri, "tls=true") || strings.Contains(uri, "ssl=true") {
		if pool, e := certs.RootCertPool(); e == nil {
			opts.SetTLSConfig(&tls.Config{RootCAs: pool})
		}
	}

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, "", fmt.Errorf("connect error: %w", err)
	}
	return client, dbname, nil
}

// bsonDocToStruct converts a bson.D document to a *structpb.Struct.
// It round-trips through relaxed extended JSON to handle ObjectID and other
// BSON-specific types safely.
func bsonDocToStruct(doc bson.D) (*structpb.Struct, error) {
	raw, err := bson.MarshalExtJSON(doc, false, false)
	if err != nil {
		return nil, fmt.Errorf("marshal ext-json: %w", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("unmarshal to map: %w", err)
	}
	return structpb.NewStruct(m)
}

// parseBSONDoc parses a JSON / relaxed extended JSON string into a bson.D.
func parseBSONDoc(s string) (bson.D, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "{}" {
		return bson.D{}, nil
	}
	var doc bson.D
	if err := bson.UnmarshalExtJSON([]byte(s), false, &doc); err != nil {
		return nil, fmt.Errorf("invalid JSON document: %w", err)
	}
	return doc, nil
}

// parseBSONArray parses a JSON array string into a bson.A.
func parseBSONArray(s string) (bson.A, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "[]" {
		return bson.A{}, nil
	}
	var arr bson.A
	if err := bson.UnmarshalExtJSON([]byte(s), false, &arr); err != nil {
		return nil, fmt.Errorf("invalid JSON array: %w", err)
	}
	return arr, nil
}

// splitTopLevelArgs splits a string by commas that are not nested inside
// brackets or string literals. This allows parsing multi-argument function
// calls such as `{filter}, {update}` or `[pipeline], {}`.
func splitTopLevelArgs(s string) []string {
	var args []string
	depth := 0
	inStr := false
	strChar := rune(0)
	escape := false
	start := 0

	for i, r := range s {
		if escape {
			escape = false
			continue
		}
		if r == '\\' && inStr {
			escape = true
			continue
		}
		if !inStr && (r == '"' || r == '\'') {
			inStr = true
			strChar = r
			continue
		}
		if inStr && r == strChar {
			inStr = false
			continue
		}
		if inStr {
			continue
		}
		switch r {
		case '{', '[', '(':
			depth++
		case '}', ']', ')':
			depth--
		case ',':
			if depth == 0 {
				args = append(args, strings.TrimSpace(s[start:i]))
				start = i + 1
			}
		}
	}
	if tail := strings.TrimSpace(s[start:]); tail != "" {
		args = append(args, tail)
	}
	return args
}

// parseMQLCommand parses a MongoDB shell-style query such as:
//
//	db.collection.find({...})
//	db.createCollection("name")
//
// It returns the target (collection name for collection ops, empty for db-level
// ops), the operation name, the raw argument string, and an ok flag.
func parseMQLCommand(query string) (target, op, argsStr string, ok bool) {
	query = strings.TrimSpace(query)
	if !strings.HasPrefix(query, "db.") {
		return
	}
	rest := query[3:] // strip "db."

	// Find the first opening parenthesis – everything before it is "target.op".
	parenIdx := strings.IndexByte(rest, '(')
	if parenIdx < 0 {
		return
	}
	funcPart := rest[:parenIdx]

	lastDot := strings.LastIndex(funcPart, ".")
	if lastDot < 0 {
		// No dot → top-level db operation, e.g. db.dropDatabase()
		target = ""
		op = strings.TrimSpace(funcPart)
	} else {
		target = strings.TrimSpace(funcPart[:lastDot])
		op = strings.TrimSpace(funcPart[lastDot+1:])
	}

	// Extract the content inside the outermost parentheses (balanced).
	inner := rest[parenIdx+1:]
	depth := 1
	strInner := false
	strInnerChar := rune(0)
	escInner := false

	for i, r := range inner {
		if escInner {
			escInner = false
			continue
		}
		if r == '\\' && strInner {
			escInner = true
			continue
		}
		if !strInner && (r == '"' || r == '\'') {
			strInner = true
			strInnerChar = r
			continue
		}
		if strInner && r == strInnerChar {
			strInner = false
			continue
		}
		if strInner {
			continue
		}
		switch r {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
			if depth == 0 {
				argsStr = strings.TrimSpace(inner[:i])
				ok = true
				return
			}
		}
	}
	return
}

// kvResponse wraps a string map into a KeyValueResult ExecResponse.
func kvResponse(data map[string]string) *plugin.ExecResponse {
	return &plugin.ExecResponse{
		Result: &plugin.ExecResult{
			Payload: &pluginpb.PluginV1_ExecResult_Kv{
				Kv: &plugin.KeyValueResult{Data: data},
			},
		},
	}
}

// cursorToDocumentResponse drains a cursor and returns a DocumentResult response.
func cursorToDocumentResponse(ctx context.Context, cursor *mongo.Cursor) (*plugin.ExecResponse, error) {
	var docs []*structpb.Struct
	for cursor.Next(ctx) {
		var doc bson.D
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		s, err := bsonDocToStruct(doc)
		if err != nil {
			continue
		}
		docs = append(docs, s)
	}
	if err := cursor.Err(); err != nil {
		return &plugin.ExecResponse{Error: fmt.Sprintf("cursor error: %v", err)}, nil
	}
	if docs == nil {
		docs = []*structpb.Struct{}
	}
	return &plugin.ExecResponse{
		Result: &plugin.ExecResult{
			Payload: &pluginpb.PluginV1_ExecResult_Document{
				Document: &plugin.DocumentResult{Documents: docs},
			},
		},
	}, nil
}

// execMQL executes a MongoDB shell-style query or a raw JSON command against db.
func execMQL(ctx context.Context, db *mongo.Database, query string) (*plugin.ExecResponse, error) {
	query = strings.TrimSpace(query)

	target, op, argsStr, ok := parseMQLCommand(query)
	if ok {
		args := splitTopLevelArgs(argsStr)

		// Handle db-level operations (target is empty).
		if target == "" {
			switch op {
			case "dropDatabase":
				if err := db.Drop(ctx); err != nil {
					return &plugin.ExecResponse{Error: fmt.Sprintf("dropDatabase error: %v", err)}, nil
				}
				return kvResponse(map[string]string{"result": "ok", "dropped": db.Name()}), nil

			case "createCollection":
				if len(args) == 0 {
					return &plugin.ExecResponse{Error: "createCollection requires a collection name"}, nil
				}
				collName := strings.Trim(args[0], `"' `)
				if err := db.CreateCollection(ctx, collName); err != nil {
					return &plugin.ExecResponse{Error: fmt.Sprintf("createCollection error: %v", err)}, nil
				}
				return kvResponse(map[string]string{"result": "ok", "created": collName}), nil

			case "listCollections", "getCollectionNames":
				names, err := db.ListCollectionNames(ctx, bson.D{})
				if err != nil {
					return &plugin.ExecResponse{Error: fmt.Sprintf("listCollections error: %v", err)}, nil
				}
				result := make(map[string]string, len(names))
				for i, n := range names {
					result[fmt.Sprintf("%d", i)] = n
				}
				return kvResponse(result), nil
			}
		}

		// Handle collection-level operations.
		coll := db.Collection(target)
		switch op {
		case "find", "findOne":
			filter := bson.D{}
			if len(args) > 0 && args[0] != "" {
				var err error
				filter, err = parseBSONDoc(args[0])
				if err != nil {
					return &plugin.ExecResponse{Error: fmt.Sprintf("filter parse error: %v", err)}, nil
				}
			}
			findOpts := options.Find()
			if op == "findOne" {
				findOpts.SetLimit(1)
			}
			cursor, err := coll.Find(ctx, filter, findOpts)
			if err != nil {
				return &plugin.ExecResponse{Error: fmt.Sprintf("find error: %v", err)}, nil
			}
			defer cursor.Close(ctx)
			return cursorToDocumentResponse(ctx, cursor)

		case "insertOne":
			if len(args) == 0 || args[0] == "" {
				return &plugin.ExecResponse{Error: "insertOne requires a document argument"}, nil
			}
			doc, err := parseBSONDoc(args[0])
			if err != nil {
				return &plugin.ExecResponse{Error: fmt.Sprintf("document parse error: %v", err)}, nil
			}
			res, err := coll.InsertOne(ctx, doc)
			if err != nil {
				return &plugin.ExecResponse{Error: fmt.Sprintf("insertOne error: %v", err)}, nil
			}
			return kvResponse(map[string]string{"insertedId": fmt.Sprintf("%v", res.InsertedID)}), nil

		case "insertMany":
			if len(args) == 0 || args[0] == "" {
				return &plugin.ExecResponse{Error: "insertMany requires a documents array argument"}, nil
			}
			arr, err := parseBSONArray(args[0])
			if err != nil {
				return &plugin.ExecResponse{Error: fmt.Sprintf("documents parse error: %v", err)}, nil
			}
			docs := make([]interface{}, len(arr))
			for i, v := range arr {
				docs[i] = v
			}
			res, err := coll.InsertMany(ctx, docs)
			if err != nil {
				return &plugin.ExecResponse{Error: fmt.Sprintf("insertMany error: %v", err)}, nil
			}
			ids := make([]string, len(res.InsertedIDs))
			for i, id := range res.InsertedIDs {
				ids[i] = fmt.Sprintf("%v", id)
			}
			return kvResponse(map[string]string{
				"insertedCount": fmt.Sprintf("%d", len(ids)),
				"insertedIds":   strings.Join(ids, ", "),
			}), nil

		case "updateOne", "updateMany", "replaceOne":
			if len(args) < 2 {
				return &plugin.ExecResponse{Error: fmt.Sprintf("%s requires filter and update arguments", op)}, nil
			}
			filter, err := parseBSONDoc(args[0])
			if err != nil {
				return &plugin.ExecResponse{Error: fmt.Sprintf("filter parse error: %v", err)}, nil
			}
			update, err := parseBSONDoc(args[1])
			if err != nil {
				return &plugin.ExecResponse{Error: fmt.Sprintf("update parse error: %v", err)}, nil
			}
			var matched, modified int64
			switch op {
			case "updateOne":
				res, err := coll.UpdateOne(ctx, filter, update)
				if err != nil {
					return &plugin.ExecResponse{Error: fmt.Sprintf("updateOne error: %v", err)}, nil
				}
				matched, modified = res.MatchedCount, res.ModifiedCount
			case "updateMany":
				res, err := coll.UpdateMany(ctx, filter, update)
				if err != nil {
					return &plugin.ExecResponse{Error: fmt.Sprintf("updateMany error: %v", err)}, nil
				}
				matched, modified = res.MatchedCount, res.ModifiedCount
			case "replaceOne":
				res, err := coll.ReplaceOne(ctx, filter, update)
				if err != nil {
					return &plugin.ExecResponse{Error: fmt.Sprintf("replaceOne error: %v", err)}, nil
				}
				matched, modified = res.MatchedCount, res.ModifiedCount
			}
			return kvResponse(map[string]string{
				"matchedCount":  fmt.Sprintf("%d", matched),
				"modifiedCount": fmt.Sprintf("%d", modified),
			}), nil

		case "deleteOne", "deleteMany":
			filter := bson.D{}
			if len(args) > 0 && args[0] != "" {
				var err error
				filter, err = parseBSONDoc(args[0])
				if err != nil {
					return &plugin.ExecResponse{Error: fmt.Sprintf("filter parse error: %v", err)}, nil
				}
			}
			var deleted int64
			if op == "deleteOne" {
				res, err := coll.DeleteOne(ctx, filter)
				if err != nil {
					return &plugin.ExecResponse{Error: fmt.Sprintf("deleteOne error: %v", err)}, nil
				}
				deleted = res.DeletedCount
			} else {
				res, err := coll.DeleteMany(ctx, filter)
				if err != nil {
					return &plugin.ExecResponse{Error: fmt.Sprintf("deleteMany error: %v", err)}, nil
				}
				deleted = res.DeletedCount
			}
			return kvResponse(map[string]string{"deletedCount": fmt.Sprintf("%d", deleted)}), nil

		case "aggregate":
			if len(args) == 0 || args[0] == "" {
				return &plugin.ExecResponse{Error: "aggregate requires a pipeline array argument"}, nil
			}
			pipeline, err := parseBSONArray(args[0])
			if err != nil {
				return &plugin.ExecResponse{Error: fmt.Sprintf("pipeline parse error: %v", err)}, nil
			}
			cursor, err := coll.Aggregate(ctx, pipeline)
			if err != nil {
				return &plugin.ExecResponse{Error: fmt.Sprintf("aggregate error: %v", err)}, nil
			}
			defer cursor.Close(ctx)
			return cursorToDocumentResponse(ctx, cursor)

		case "countDocuments":
			filter := bson.D{}
			if len(args) > 0 && args[0] != "" {
				var err error
				filter, err = parseBSONDoc(args[0])
				if err != nil {
					return &plugin.ExecResponse{Error: fmt.Sprintf("filter parse error: %v", err)}, nil
				}
			}
			count, err := coll.CountDocuments(ctx, filter)
			if err != nil {
				return &plugin.ExecResponse{Error: fmt.Sprintf("countDocuments error: %v", err)}, nil
			}
			return kvResponse(map[string]string{"count": fmt.Sprintf("%d", count)}), nil

		case "estimatedDocumentCount":
			count, err := coll.EstimatedDocumentCount(ctx)
			if err != nil {
				return &plugin.ExecResponse{Error: fmt.Sprintf("estimatedDocumentCount error: %v", err)}, nil
			}
			return kvResponse(map[string]string{"count": fmt.Sprintf("%d", count)}), nil

		case "drop":
			if err := coll.Drop(ctx); err != nil {
				return &plugin.ExecResponse{Error: fmt.Sprintf("drop error: %v", err)}, nil
			}
			return kvResponse(map[string]string{"result": "ok", "dropped": target}), nil

		case "createIndex":
			if len(args) == 0 || args[0] == "" {
				return &plugin.ExecResponse{Error: "createIndex requires an index keys document"}, nil
			}
			keys, err := parseBSONDoc(args[0])
			if err != nil {
				return &plugin.ExecResponse{Error: fmt.Sprintf("keys parse error: %v", err)}, nil
			}
			indexModel := mongo.IndexModel{Keys: keys}
			name, err := coll.Indexes().CreateOne(ctx, indexModel)
			if err != nil {
				return &plugin.ExecResponse{Error: fmt.Sprintf("createIndex error: %v", err)}, nil
			}
			return kvResponse(map[string]string{"name": name}), nil

		case "distinct":
			if len(args) == 0 || args[0] == "" {
				return &plugin.ExecResponse{Error: "distinct requires a field name"}, nil
			}
			field := strings.Trim(args[0], `"' `)
			filter := bson.D{}
			if len(args) > 1 && args[1] != "" {
				var err error
				filter, err = parseBSONDoc(args[1])
				if err != nil {
					return &plugin.ExecResponse{Error: fmt.Sprintf("filter parse error: %v", err)}, nil
				}
			}
			values, err := coll.Distinct(ctx, field, filter)
			if err != nil {
				return &plugin.ExecResponse{Error: fmt.Sprintf("distinct error: %v", err)}, nil
			}
			strs := make([]string, len(values))
			for i, v := range values {
				strs[i] = fmt.Sprintf("%v", v)
			}
			return kvResponse(map[string]string{
				"values": strings.Join(strs, ", "),
				"count":  fmt.Sprintf("%d", len(values)),
			}), nil
		}

		return &plugin.ExecResponse{Error: fmt.Sprintf("unknown operation: %s", op)}, nil
	}

	// Fall back to a raw JSON command document.
	if strings.HasPrefix(query, "{") {
		var cmd bson.D
		if err := bson.UnmarshalExtJSON([]byte(query), false, &cmd); err != nil {
			return &plugin.ExecResponse{Error: fmt.Sprintf("invalid command JSON: %v", err)}, nil
		}
		result := db.RunCommand(ctx, cmd)
		if result.Err() != nil {
			return &plugin.ExecResponse{Error: fmt.Sprintf("command error: %v", result.Err())}, nil
		}
		var raw bson.D
		if err := result.Decode(&raw); err != nil {
			return &plugin.ExecResponse{Error: fmt.Sprintf("decode error: %v", err)}, nil
		}
		s, err := bsonDocToStruct(raw)
		if err != nil {
			return &plugin.ExecResponse{Error: fmt.Sprintf("format error: %v", err)}, nil
		}
		return &plugin.ExecResponse{
			Result: &plugin.ExecResult{
				Payload: &pluginpb.PluginV1_ExecResult_Document{
					Document: &plugin.DocumentResult{Documents: []*structpb.Struct{s}},
				},
			},
		}, nil
	}

	return &plugin.ExecResponse{
		Error: "unsupported query format\n" +
			"Examples:\n" +
			"  db.users.find({})\n" +
			"  db.users.insertOne({\"name\": \"Alice\"})\n" +
			"  db.users.updateOne({\"name\": \"Alice\"}, {\"$set\": {\"age\": 30}})\n" +
			"  db.users.deleteOne({\"name\": \"Alice\"})\n" +
			"  db.users.aggregate([{\"$group\": {\"_id\": \"$status\"}}])\n" +
			"  {\"ping\": 1}",
	}, nil
}

func (m *mongoPlugin) Exec(ctx context.Context, req *plugin.ExecRequest) (*plugin.ExecResponse, error) {
	client, dbname, err := connectMongo(ctx, req.Connection)
	if err != nil {
		return &plugin.ExecResponse{Error: fmt.Sprintf("connection error: %v", err)}, nil
	}
	defer client.Disconnect(ctx)

	if dbname == "" {
		dbname = getDatabaseName(req.Connection)
	}

	return execMQL(ctx, client.Database(dbname), req.Query)
}

func (m *mongoPlugin) ConnectionTree(ctx context.Context, req *plugin.ConnectionTreeRequest) (*plugin.ConnectionTreeResponse, error) {
	client, _, err := connectMongo(ctx, req.Connection)
	if err != nil {
		return &plugin.ConnectionTreeResponse{}, nil
	}
	defer client.Disconnect(ctx)

	dbResult, err := client.ListDatabases(ctx, bson.D{})
	if err != nil {
		return &plugin.ConnectionTreeResponse{}, nil
	}

	var dbNodes []*plugin.ConnectionTreeNode
	for _, dbInfo := range dbResult.Databases {
		dbName := dbInfo.Name
		db := client.Database(dbName)

		collNames, _ := db.ListCollectionNames(ctx, bson.D{})

		var collNodes []*plugin.ConnectionTreeNode
		for _, coll := range collNames {
			collNodes = append(collNodes, &plugin.ConnectionTreeNode{
				Key:      dbName + "." + coll,
				Label:    coll,
				NodeType: plugin.ConnectionTreeNodeTypeCollection,
				Actions: []*plugin.ConnectionTreeAction{
					{
						Type:   plugin.ConnectionTreeActionSelect,
						Title:  "Find documents",
						Query:  fmt.Sprintf("db.%s.find({})", coll),
						Hidden: true,
						NewTab: true,
					},
					{
						Type:  plugin.ConnectionTreeActionDropTable,
						Title: "Drop collection",
						Query: fmt.Sprintf("db.%s.drop()", coll),
					},
				},
			})
		}

		dbNodes = append(dbNodes, &plugin.ConnectionTreeNode{
			Key:      dbName,
			Label:    dbName,
			NodeType: plugin.ConnectionTreeNodeTypeDatabase,
			Children: collNodes,
			Actions: []*plugin.ConnectionTreeAction{
				{
					Type:  plugin.ConnectionTreeActionCreateTable,
					Title: "Create collection",
					Query: `db.createCollection("new_collection")`,
				},
				{
					Type:  plugin.ConnectionTreeActionDropDatabase,
					Title: "Drop database",
					Query: "db.dropDatabase()",
				},
			},
		})
	}

	return &plugin.ConnectionTreeResponse{Nodes: dbNodes}, nil
}

// TestConnection pings MongoDB to verify the supplied credentials are valid.
func (m *mongoPlugin) TestConnection(ctx context.Context, req *plugin.TestConnectionRequest) (*plugin.TestConnectionResponse, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	client, _, err := connectMongo(timeoutCtx, req.Connection)
	if err != nil {
		return &plugin.TestConnectionResponse{Ok: false, Message: fmt.Sprintf("connection error: %v", err)}, nil
	}
	defer client.Disconnect(timeoutCtx)

	if err := client.Ping(timeoutCtx, nil); err != nil {
		return &plugin.TestConnectionResponse{Ok: false, Message: fmt.Sprintf("ping error: %v", err)}, nil
	}
	return &plugin.TestConnectionResponse{Ok: true, Message: "Connection successful"}, nil
}

func main() {
	plugin.ServeCLI(&mongoPlugin{})
}
