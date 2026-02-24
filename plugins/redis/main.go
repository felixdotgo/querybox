package main

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/felixdotgo/querybox/pkg/plugin"
	pluginpb "github.com/felixdotgo/querybox/rpc/contracts/plugin/v1"
	"github.com/redis/go-redis/v9"
)

// redisPlugin implements the plugin.Plugin interface for Redis.
type redisPlugin struct{}

func (r *redisPlugin) Info() (plugin.InfoResponse, error) {
	return plugin.InfoResponse{
		Type:        plugin.TypeDriver,
		Name:        "Redis",
		Version:     "0.1.0",
		Description: "Redis key-value store driver",
	}, nil
}

func (r *redisPlugin) AuthForms(*plugin.AuthFormsRequest) (*plugin.AuthFormsResponse, error) {
	basic := plugin.AuthForm{
		Key:  "basic",
		Name: "Basic",
		Fields: []*plugin.AuthField{
			{Type: plugin.AuthFieldText, Name: "host", Label: "Host", Required: true, Placeholder: "127.0.0.1", Value: "127.0.0.1"},
			{Type: plugin.AuthFieldNumber, Name: "port", Label: "Port", Placeholder: "6379", Value: "6379"},
			{Type: plugin.AuthFieldPassword, Name: "password", Label: "Password"},
			{Type: plugin.AuthFieldNumber, Name: "db", Label: "Database index", Placeholder: "0", Value: "0"},
			{Type: plugin.AuthFieldSelect, Name: "tls", Label: "TLS", Options: []string{"false", "true"}, Value: "false"},
		},
	}
	url := plugin.AuthForm{
		Key:  "url",
		Name: "URL",
		Fields: []*plugin.AuthField{
			{Type: plugin.AuthFieldText, Name: "url", Label: "Redis URL", Required: true, Placeholder: "redis://:password@localhost:6379/0"},
		},
	}
	return &plugin.AuthFormsResponse{Forms: map[string]*plugin.AuthForm{"basic": &basic, "url": &url}}, nil
}

// buildClient constructs a go-redis client from the connection map.
// Supports both a raw URL and basic host/port/password/db fields via
// credential_blob JSON (form: "basic" or "url").
func buildClient(connection map[string]string) (*redis.Client, error) {
	// Direct URL key (legacy path).
	if u, ok := connection["url"]; ok && u != "" {
		opts, err := redis.ParseURL(u)
		if err != nil {
			return nil, fmt.Errorf("invalid Redis URL: %w", err)
		}
		return redis.NewClient(opts), nil
	}

	// credential_blob JSON path.
	blob, ok := connection["credential_blob"]
	if !ok || blob == "" {
		return nil, fmt.Errorf("missing connection parameters")
	}

	var payload struct {
		Form   string            `json:"form"`
		Values map[string]string `json:"values"`
	}
	if err := json.Unmarshal([]byte(blob), &payload); err != nil {
		return nil, fmt.Errorf("invalid credential blob: %w", err)
	}

	if u := payload.Values["url"]; u != "" {
		opts, err := redis.ParseURL(u)
		if err != nil {
			return nil, fmt.Errorf("invalid Redis URL: %w", err)
		}
		return redis.NewClient(opts), nil
	}

	host := payload.Values["host"]
	if host == "" {
		host = "127.0.0.1"
	}
	port := payload.Values["port"]
	if port == "" {
		port = "6379"
	}
	dbIndex := 0
	if dbStr := payload.Values["db"]; dbStr != "" {
		if n, err := strconv.Atoi(dbStr); err == nil {
			dbIndex = n
		}
	}

	opts := &redis.Options{
		Addr:     fmt.Sprintf("%s:%s", host, port),
		Password: payload.Values["password"],
		DB:       dbIndex,
	}
	return redis.NewClient(opts), nil
}

// buildClientForDB is identical to buildClient but forces the connection to use
// the specified logical database index.  This is used to handle the SELECT
// action without relying on the raw SELECT command via a pooled connection.
func buildClientForDB(connection map[string]string, dbIdx int) (*redis.Client, error) {
	client, err := buildClient(connection)
	if err != nil {
		return nil, err
	}
	// Reconstruct the options with the desired DB index.
	opts := client.Options()
	client.Close()
	opts.DB = dbIdx
	return redis.NewClient(opts), nil
}

// parseCommand splits a Redis command string into the command name and its
// arguments.  Quoted tokens are preserved as single arguments so callers can
// include values with spaces (e.g. SET key "hello world").
var argSplitter = regexp.MustCompile(`"[^"]*"|'[^']*'|\S+`)

func parseCommand(query string) []interface{} {
	tokens := argSplitter.FindAllString(strings.TrimSpace(query), -1)
	args := make([]interface{}, len(tokens))
	for i, t := range tokens {
		// Strip surrounding quotes.
		if (strings.HasPrefix(t, `"`) && strings.HasSuffix(t, `"`)) ||
			(strings.HasPrefix(t, `'`) && strings.HasSuffix(t, `'`)) {
			t = t[1 : len(t)-1]
		}
		args[i] = t
	}
	return args
}

// formatResult converts a raw redis.Do response into an ExecResult payload.
// The mapping is:
//   - nil            → KeyValueResult{"result": "(nil)"}
//   - string / int64 → KeyValueResult{"result": value}
//   - []interface{}  → even-count slices whose index-0 element is a string are
//     treated as alternating field/value pairs (HGETALL/HMGET) and rendered as
//     a KeyValueResult map.  Odd-length or non-string-keyed slices fall back to
//     a SqlResult with a single "value" column.
func formatResult(val interface{}) *plugin.ExecResult {
	switch v := val.(type) {
	case nil:
		return kvSingleResult("(nil)")

	case string:
		return kvSingleResult(v)

	case int64:
		return kvSingleResult(strconv.FormatInt(v, 10))

	case []interface{}:
		// Treat as hash pairs when the slice has an even, non-zero length and
		// the first element is a non-empty string (field name).
		if len(v) > 0 && len(v)%2 == 0 {
			_, firstIsStr := v[0].(string)
			if firstIsStr {
				data := make(map[string]string, len(v)/2)
				for i := 0; i+1 < len(v); i += 2 {
					data[fmt.Sprintf("%v", v[i])] = fmt.Sprintf("%v", v[i+1])
				}
				return &plugin.ExecResult{
					Payload: &pluginpb.PluginV1_ExecResult_Kv{
						Kv: &plugin.KeyValueResult{Data: data},
					},
				}
			}
		}
		// Generic list - single "value" column.
		cols := []*plugin.Column{{Name: "value"}}
		var rows []*plugin.Row
		for _, item := range v {
			rows = append(rows, &plugin.Row{Values: []string{fmt.Sprintf("%v", item)}})
		}
		return &plugin.ExecResult{
			Payload: &pluginpb.PluginV1_ExecResult_Sql{
				Sql: &plugin.SqlResult{Columns: cols, Rows: rows},
			},
		}

	default:
		return kvSingleResult(fmt.Sprintf("%v", v))
	}
}

// kvSingleResult wraps a single scalar value in a KeyValueResult under the
// "result" key, which is the natural representation for Redis scalar commands
// such as GET, SET, INCR, EXPIRE, etc.
func kvSingleResult(value string) *plugin.ExecResult {
	return &plugin.ExecResult{
		Payload: &pluginpb.PluginV1_ExecResult_Kv{
			Kv: &plugin.KeyValueResult{Data: map[string]string{"result": value}},
		},
	}
}

func (r *redisPlugin) Exec(req *plugin.ExecRequest) (*plugin.ExecResponse, error) {
	client, err := buildClient(req.Connection)
	if err != nil {
		return &plugin.ExecResponse{Error: fmt.Sprintf("connection error: %v", err)}, nil
	}
	defer client.Close()

	args := parseCommand(req.Query)
	if len(args) == 0 {
		return &plugin.ExecResponse{Error: "empty command"}, nil
	}

	ctx := context.Background()

	// SELECT is a connection-state command that go-redis cannot execute via Do
	// on a pooled client.  Handle it by reconnecting to the requested DB and
	// returning its DBSIZE so the user sees something meaningful.
	if strings.EqualFold(fmt.Sprintf("%v", args[0]), "select") {
		dbIdx := 0
		if len(args) > 1 {
			if n, err := strconv.Atoi(fmt.Sprintf("%v", args[1])); err == nil {
				dbIdx = n
			}
		}
		// Build a new client scoped to the requested DB.
		connCopy := make(map[string]string, len(req.Connection))
		for k, v := range req.Connection {
			connCopy[k] = v
		}
		client.Close()
		dbClient, dbErr := buildClientForDB(connCopy, dbIdx)
		if dbErr != nil {
			return &plugin.ExecResponse{Error: fmt.Sprintf("select error: %v", dbErr)}, nil
		}
		defer dbClient.Close()
		size, dbErr := dbClient.DBSize(ctx).Result()
		if dbErr != nil {
			return &plugin.ExecResponse{Error: fmt.Sprintf("select error: %v", dbErr)}, nil
		}
		return &plugin.ExecResponse{
			Result: &plugin.ExecResult{
				Payload: &pluginpb.PluginV1_ExecResult_Kv{
					Kv: &plugin.KeyValueResult{Data: map[string]string{
						"db":   fmt.Sprintf("db%d", dbIdx),
						"keys": strconv.FormatInt(size, 10),
					}},
				},
			},
		}, nil
	}

	cmd := client.Do(ctx, args...)
	val, err := cmd.Result()
	if err != nil && err != redis.Nil {
		return &plugin.ExecResponse{Error: fmt.Sprintf("command error: %v", err)}, nil
	}
	if err == redis.Nil {
		val = nil
	}

	return &plugin.ExecResponse{Result: formatResult(val)}, nil
}

// keyQuery returns the appropriate Redis read command for a given key type so
// that selecting a key from the connection tree always returns a meaningful
// key-value result.
func keyQuery(key, keyType string) string {
	switch keyType {
	case "hash":
		return fmt.Sprintf("HGETALL %s", key)
	case "list":
		return fmt.Sprintf("LRANGE %s 0 -1", key)
	case "set":
		return fmt.Sprintf("SMEMBERS %s", key)
	case "zset":
		return fmt.Sprintf("ZRANGE %s 0 -1 WITHSCORES", key)
	default: // string and unknown types
		return fmt.Sprintf("GET %s", key)
	}
}

// parseKeyspaceInfo extracts a db-index → key-count map from the output of
// "INFO keyspace".  Lines have the form "db0:keys=42,expires=0,avg_ttl=0".
func parseKeyspaceInfo(info string) map[int]string {
	result := make(map[int]string)
	for _, line := range strings.Split(info, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "db") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		dbIdx, err := strconv.Atoi(strings.TrimPrefix(parts[0], "db"))
		if err != nil {
			continue
		}
		for _, kv := range strings.Split(parts[1], ",") {
			if strings.HasPrefix(kv, "keys=") {
				result[dbIdx] = strings.TrimPrefix(kv, "keys=")
				break
			}
		}
	}
	return result
}

// ConnectionTree always lists all 16 logical Redis databases (db0–db15) so
// the user can see and select any database regardless of whether it is
// populated.  Databases that contain keys show a SCAN-based preview of the
// first 50 keys as children.  Key nodes carry a type-appropriate read action
// so the result is always rendered as a key-value payload.
func (r *redisPlugin) ConnectionTree(req *plugin.ConnectionTreeRequest) (*plugin.ConnectionTreeResponse, error) {
	client, err := buildClient(req.Connection)
	if err != nil {
		return &plugin.ConnectionTreeResponse{}, nil
	}
	defer client.Close()

	ctx := context.Background()

	// Retrieve keyspace info for key counts; errors are non-fatal.
	infoStr, _ := client.Info(ctx, "keyspace").Result()
	keyCounts := parseKeyspaceInfo(infoStr)

	var nodes []*plugin.ConnectionTreeNode

	// Redis supports 16 logical databases by default (configurable via
	// databases directive in redis.conf, but 16 is the standard default).
	const totalDatabases = 16
	for dbIdx := 0; dbIdx < totalDatabases; dbIdx++ {
		dbLabel := fmt.Sprintf("db%d", dbIdx)
		if count, ok := keyCounts[dbIdx]; ok {
			dbLabel = fmt.Sprintf("db%d (%s keys)", dbIdx, count)
		}

		// Use a dedicated connection scoped to this logical database.
		dbClient := client.Conn()
		_ = dbClient.Do(ctx, "SELECT", dbIdx)

		var keyNodes []*plugin.ConnectionTreeNode
		if _, populated := keyCounts[dbIdx]; populated {
			keys, _, scanErr := dbClient.Scan(ctx, 0, "*", 50).Result()
			if scanErr == nil {
				for _, k := range keys {
					kType, _ := dbClient.Type(ctx, k).Result()
					keyNodes = append(keyNodes, &plugin.ConnectionTreeNode{
						Key:      fmt.Sprintf("db%d:%s", dbIdx, k),
						Label:    fmt.Sprintf("%s (%s)", k, kType),
						NodeType: plugin.ConnectionTreeNodeTypeKey,
						Actions: []*plugin.ConnectionTreeAction{
							{
								Type:   plugin.ConnectionTreeActionSelect,
								Title:  k,
								Query:  keyQuery(k, kType),
								NewTab: true,
							},
						},
					})
				}
			}
		}
		dbClient.Close()

		nodes = append(nodes, &plugin.ConnectionTreeNode{
			Key:      fmt.Sprintf("db%d", dbIdx),
			Label:    dbLabel,
			NodeType: plugin.ConnectionTreeNodeTypeDatabase,
			Children: keyNodes,
			Actions: []*plugin.ConnectionTreeAction{
				{Type: plugin.ConnectionTreeActionSelect, Title: "Select DB", Query: fmt.Sprintf("SCAN %d MATCH * COUNT 100", dbIdx), NewTab: true},
				// Redis logical databases cannot be created or dropped; FLUSHDB
				// removes all keys from the selected database (use with care).
				{Type: plugin.ConnectionTreeActionDropDatabase, Title: "Flush DB (delete all keys)", Query: "FLUSHDB"},
			},
		})
	}

	return &plugin.ConnectionTreeResponse{Nodes: nodes}, nil
}

// TestConnection pings the Redis server to verify the supplied credentials.
func (r *redisPlugin) TestConnection(req *plugin.TestConnectionRequest) (*plugin.TestConnectionResponse, error) {
	client, err := buildClient(req.Connection)
	if err != nil {
		return &plugin.TestConnectionResponse{Ok: false, Message: err.Error()}, nil
	}
	defer client.Close()

	if err := client.Ping(context.Background()).Err(); err != nil {
		return &plugin.TestConnectionResponse{Ok: false, Message: fmt.Sprintf("ping error: %v", err)}, nil
	}
	return &plugin.TestConnectionResponse{Ok: true, Message: "Connection successful"}, nil
}

func main() {
	plugin.ServeCLI(&redisPlugin{})
}
