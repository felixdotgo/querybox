package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/felixdotgo/querybox/pkg/plugin"
	pluginpb "github.com/felixdotgo/querybox/rpc/contracts/plugin/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	defaultRedisHost  = "127.0.0.1"
	defaultRedisPort  = "6379"
	defaultRedisDB    = "0"
	defaultScanCount  = 50
	defaultValueLimit = 100
)

type redisPlugin struct {
	pluginpb.UnimplementedPluginServiceServer
}

type redisConfig struct {
	Address  string
	Username string
	Password string
	Database string
	UseTLS   bool
}

type redisClient struct {
	conn net.Conn
	rw   *bufio.ReadWriter
}

type redisValue struct {
	kind  byte
	text  string
	int64 int64
	array []redisValue
	null  bool
}

var dialRedis = func(ctx context.Context, network, address string, tlsConfig *tls.Config) (net.Conn, error) {
	dialer := &net.Dialer{}
	if tlsConfig != nil {
		return tls.DialWithDialer(dialer, network, address, tlsConfig)
	}
	return dialer.DialContext(ctx, network, address)
}

func (m *redisPlugin) Info(ctx context.Context, _ *pluginpb.PluginV1_InfoRequest) (*plugin.InfoResponse, error) {
	return &plugin.InfoResponse{
		Type:         plugin.TypeDriver,
		Name:         "Redis",
		Version:      "0.2.0",
		Description:  "Redis key inspector plugin",
		Url:          "https://redis.io/",
		Author:       "QueryBox Team",
		Capabilities: []string{"resource.graph", "connection.test", "query.execute"},
		Tags:         []string{"nosql", "key-value", "cache"},
		License:      "MIT",
		IconUrl:      "https://redis.io/images/redis-white.png",
		Metadata: map[string]string{
			"simple_icon": "redis",
		},
	}, nil
}

func (m *redisPlugin) AuthForms(ctx context.Context, _ *plugin.AuthFormsRequest) (*plugin.AuthFormsResponse, error) {
	basic := plugin.AuthForm{
		Key:  "basic",
		Name: "Basic",
		Fields: []*plugin.AuthField{
			{Type: plugin.AuthFieldText, Name: "host", Label: "Host", Required: true, Placeholder: defaultRedisHost, Value: defaultRedisHost},
			{Type: plugin.AuthFieldNumber, Name: "port", Label: "Port", Placeholder: defaultRedisPort, Value: defaultRedisPort},
			{Type: plugin.AuthFieldText, Name: "username", Label: "Username"},
			{Type: plugin.AuthFieldPassword, Name: "password", Label: "Password"},
			{Type: plugin.AuthFieldNumber, Name: "database", Label: "Database", Placeholder: defaultRedisDB, Value: defaultRedisDB},
			{Type: plugin.AuthFieldCheckbox, Name: "tls", Label: "Use TLS"},
		},
	}
	redisURL := plugin.AuthForm{
		Key:  "url",
		Name: "URL",
		Fields: []*plugin.AuthField{
			{Type: plugin.AuthFieldText, Name: "url", Label: "Redis URL", Required: true, Placeholder: "redis://localhost:6379/0"},
		},
	}
	return &plugin.AuthFormsResponse{Forms: map[string]*plugin.AuthForm{"basic": &basic, "url": &redisURL}}, nil
}

func (m *redisPlugin) TestConnection(ctx context.Context, req *plugin.TestConnectionRequest) (*plugin.TestConnectionResponse, error) {
	cfg, err := redisConfigFromConnection(req.Connection)
	if err != nil {
		return &plugin.TestConnectionResponse{Ok: false, Message: err.Error()}, nil
	}
	client, err := newRedisClient(ctx, cfg)
	if err != nil {
		return &plugin.TestConnectionResponse{Ok: false, Message: fmt.Sprintf("redis: connect failed: %v", err)}, nil
	}
	defer client.Close()
	if pong, err := client.simpleCommand(ctx, "PING"); err != nil {
		return &plugin.TestConnectionResponse{Ok: false, Message: fmt.Sprintf("redis: ping failed: %v", err)}, nil
	} else if pong == "" {
		pong = "PONG"
		return &plugin.TestConnectionResponse{Ok: true, Message: pong}, nil
	} else {
		return &plugin.TestConnectionResponse{Ok: true, Message: pong}, nil
	}
}

func (m *redisPlugin) ResourceGraph(ctx context.Context, req *plugin.ResourceGraphRequest) (*plugin.ResourceGraphResponse, error) {
	connection := map[string]string(nil)
	if req != nil {
		connection = req.Connection
	}
	cfg, err := redisConfigFromConnection(connection)
	if err != nil {
		return nil, fmt.Errorf("redis: invalid connection: %w", err)
	}
	client, err := newRedisClient(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("redis: connect failed: %w", err)
	}
	defer client.Close()

	keys, nextCursor, err := client.scanKeys(ctx, "0", defaultScanCount)
	if err != nil {
		return nil, fmt.Errorf("redis: scan failed: %w", err)
	}
	sort.Strings(keys)

	groups := map[string]*plugin.ResourceNode{}
	groupOrder := []string{"string", "hash", "list", "set", "zset", "stream", "other"}

	for _, key := range keys {
		keyType, err := client.simpleCommand(ctx, "TYPE", key)
		if err != nil {
			return nil, fmt.Errorf("redis: type lookup failed for %q: %w", key, err)
		}
		ttl, err := client.integerCommand(ctx, "TTL", key)
		if err != nil {
			return nil, fmt.Errorf("redis: ttl lookup failed for %q: %w", key, err)
		}

		groupID, groupName := redisGroupForType(keyType)
		group := groups[groupID]
		if group == nil {
			group = &plugin.ResourceNode{
				ID:       groupID,
				Name:     groupName,
				Kind:     "group",
				Path:     groupID,
				Children: []*plugin.ResourceNode{},
			}
			groups[groupID] = group
		}

		metadata := map[string]string{"redis_type": keyType}
		if ttl >= 0 {
			metadata["ttl_seconds"] = strconv.FormatInt(ttl, 10)
		}
		group.Children = append(group.Children, &plugin.ResourceNode{
			ID:       groupID + "/" + key,
			Name:     key,
			Kind:     "key",
			Path:     groupID + "/" + key,
			Metadata: metadata,
			Actions: []*plugin.ResourceAction{
				{
					ID:     "select",
					Kind:   "select",
					Title:  redisInspectTitle(keyType),
					Query:  redisInspectQuery(keyType, key),
					NewTab: true,
				},
				{
					ID:    "delete-key",
					Kind:  "delete-key",
					Title: "Delete key",
					Query: "DEL " + strconv.Quote(key),
				},
			},
		})
	}

	nodes := []*plugin.ResourceNode{
		{
			ID:   "__server_info__",
			Name: "Server info",
			Kind: "action",
			Path: "__server_info__",
			Actions: []*plugin.ResourceAction{
				{ID: "select", Kind: "select", Title: "Server info", Query: "INFO", NewTab: true},
			},
		},
	}

	for _, groupID := range groupOrder {
		group := groups[groupID]
		if group == nil {
			continue
		}
		sort.Slice(group.Children, func(i, j int) bool { return group.Children[i].Name < group.Children[j].Name })
		nodes = append(nodes, group)
	}

	if nextCursor != "0" {
		nodes = append(nodes, &plugin.ResourceNode{
			ID:   "__scan_limited__",
			Name: "More keys available",
			Kind: "resource",
			Path: "__scan_limited__",
			Metadata: map[string]string{
				"scan_cursor": nextCursor,
				"scan_count":  strconv.Itoa(defaultScanCount),
			},
			Actions: []*plugin.ResourceAction{
				{
					ID:     "select",
					Kind:   "select",
					Title:  "Inspect next scan page",
					Query:  fmt.Sprintf("SCAN %s COUNT %d", strconv.Quote(nextCursor), defaultScanCount),
					NewTab: true,
				},
			},
		})
	}

	return &plugin.ResourceGraphResponse{Nodes: nodes}, nil
}

func (m *redisPlugin) Exec(ctx context.Context, req *plugin.ExecRequest) (*plugin.ExecResponse, error) {
	cfg, err := redisConfigFromConnection(req.Connection)
	if err != nil {
		return &plugin.ExecResponse{Error: fmt.Sprintf("redis: invalid connection: %v", err)}, nil
	}
	args, err := splitRedisCommand(req.Query)
	if err != nil {
		return &plugin.ExecResponse{Error: fmt.Sprintf("redis: invalid command: %v", err)}, nil
	}
	if len(args) == 0 {
		return &plugin.ExecResponse{Error: "redis: empty command"}, nil
	}

	client, err := newRedisClient(ctx, cfg)
	if err != nil {
		return &plugin.ExecResponse{Error: fmt.Sprintf("redis: connect failed: %v", err)}, nil
	}
	defer client.Close()

	cmd := strings.ToUpper(args[0])
	switch cmd {
	case "PING":
		status, err := client.simpleCommand(ctx, args...)
		if err != nil {
			return &plugin.ExecResponse{Error: fmt.Sprintf("redis: ping failed: %v", err)}, nil
		}
		if status == "" {
			status = "PONG"
		}
		return kvResponse(map[string]string{"status": status}), nil
	case "INFO":
		data, err := client.infoCommand(ctx, args...)
		if err != nil {
			return &plugin.ExecResponse{Error: fmt.Sprintf("redis: info failed: %v", err)}, nil
		}
		return kvResponse(data), nil
	case "TYPE":
		if len(args) < 2 {
			return &plugin.ExecResponse{Error: "redis: TYPE requires a key"}, nil
		}
		value, err := client.simpleCommand(ctx, args...)
		if err != nil {
			return &plugin.ExecResponse{Error: fmt.Sprintf("redis: type failed: %v", err)}, nil
		}
		return kvResponse(map[string]string{"key": args[1], "type": value}), nil
	case "TTL":
		if len(args) < 2 {
			return &plugin.ExecResponse{Error: "redis: TTL requires a key"}, nil
		}
		ttl, err := client.integerCommand(ctx, args...)
		if err != nil {
			return &plugin.ExecResponse{Error: fmt.Sprintf("redis: ttl failed: %v", err)}, nil
		}
		return kvResponse(map[string]string{"key": args[1], "ttl_seconds": strconv.FormatInt(ttl, 10)}), nil
	case "GET":
		if len(args) < 2 {
			return &plugin.ExecResponse{Error: "redis: GET requires a key"}, nil
		}
		value, ok, err := client.getCommand(ctx, args...)
		if err != nil {
			return &plugin.ExecResponse{Error: fmt.Sprintf("redis: get failed: %v", err)}, nil
		}
		if !ok {
			return kvResponse(map[string]string{"key": args[1], "exists": "false"}), nil
		}
		return kvResponse(map[string]string{"key": args[1], "value": value}), nil
	case "SCAN":
		cursor := "0"
		count := defaultScanCount
		if len(args) >= 2 {
			cursor = args[1]
		}
		for i := 2; i < len(args); i += 2 {
			if strings.EqualFold(args[i], "COUNT") {
				if i+1 >= len(args) {
					return &plugin.ExecResponse{Error: "redis: SCAN COUNT requires a value"}, nil
				}
				parsed, err := strconv.Atoi(args[i+1])
				if err != nil || parsed <= 0 {
					return &plugin.ExecResponse{Error: "redis: SCAN COUNT must be a positive integer"}, nil
				}
				count = parsed
				continue
			}
			return &plugin.ExecResponse{Error: fmt.Sprintf("redis: unsupported SCAN option %q", args[i])}, nil
		}
		keys, nextCursor, err := client.scanKeys(ctx, cursor, count)
		if err != nil {
			return &plugin.ExecResponse{Error: fmt.Sprintf("redis: scan failed: %v", err)}, nil
		}
		sort.Strings(keys)
		docs := make([]map[string]any, 0, len(keys)+1)
		for _, key := range keys {
			docs = append(docs, map[string]any{"key": key})
		}
		if nextCursor != "0" {
			docs = append(docs, map[string]any{"next_cursor": nextCursor, "count": float64(count)})
		}
		return documentResponse(docs)
	case "DEL":
		deleted, err := client.integerCommand(ctx, args...)
		if err != nil {
			return &plugin.ExecResponse{Error: fmt.Sprintf("redis: delete failed: %v", err)}, nil
		}
		return kvResponse(map[string]string{"deleted": strconv.FormatInt(deleted, 10)}), nil
	case "HGETALL":
		data, err := client.hashCommand(ctx, args...)
		if err != nil {
			return &plugin.ExecResponse{Error: fmt.Sprintf("redis: hgetall failed: %v", err)}, nil
		}
		return kvResponse(data), nil
	case "LRANGE":
		items, err := client.arrayStringsCommand(ctx, args...)
		if err != nil {
			return &plugin.ExecResponse{Error: fmt.Sprintf("redis: lrange failed: %v", err)}, nil
		}
		docs := make([]map[string]any, 0, len(items))
		for i, item := range items {
			docs = append(docs, map[string]any{"index": float64(i), "value": item})
		}
		return documentResponse(docs)
	case "SMEMBERS":
		items, err := client.arrayStringsCommand(ctx, args...)
		if err != nil {
			return &plugin.ExecResponse{Error: fmt.Sprintf("redis: smembers failed: %v", err)}, nil
		}
		sort.Strings(items)
		docs := make([]map[string]any, 0, len(items))
		for _, item := range items {
			docs = append(docs, map[string]any{"member": item})
		}
		return documentResponse(docs)
	case "ZRANGE":
		items, err := client.arrayStringsCommand(ctx, args...)
		if err != nil {
			return &plugin.ExecResponse{Error: fmt.Sprintf("redis: zrange failed: %v", err)}, nil
		}
		docs := make([]map[string]any, 0, len(items)/2)
		for i := 0; i < len(items); i += 2 {
			doc := map[string]any{"member": items[i]}
			if i+1 < len(items) {
				doc["score"] = items[i+1]
			}
			docs = append(docs, doc)
		}
		return documentResponse(docs)
	case "XRANGE":
		docs, err := client.streamEntriesCommand(ctx, args...)
		if err != nil {
			return &plugin.ExecResponse{Error: fmt.Sprintf("redis: xrange failed: %v", err)}, nil
		}
		return documentResponse(docs)
	default:
		return &plugin.ExecResponse{Error: fmt.Sprintf("redis: unsupported command %q", cmd)}, nil
	}
}

func redisConfigFromConnection(connection map[string]string) (redisConfig, error) {
	if cred, err := plugin.ParseCredentialBlob(connection); err == nil {
		return redisConfigFromValues(cred.Values)
	}
	return redisConfigFromValues(connection)
}

func redisConfigFromValues(values map[string]string) (redisConfig, error) {
	if values == nil {
		return redisConfig{}, fmt.Errorf("missing connection parameters")
	}
	if rawURL := firstNonEmpty(values["url"], values["dsn"]); rawURL != "" {
		return redisConfigFromURL(rawURL)
	}
	host := firstNonEmpty(values["host"], defaultRedisHost)
	port := firstNonEmpty(values["port"], defaultRedisPort)
	address := net.JoinHostPort(host, port)
	return redisConfig{
		Address:  address,
		Username: values["username"],
		Password: values["password"],
		Database: firstNonEmpty(values["database"], defaultRedisDB),
		UseTLS:   parseBoolish(values["tls"]),
	}, nil
}

func redisConfigFromURL(rawURL string) (redisConfig, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return redisConfig{}, fmt.Errorf("invalid redis url: %w", err)
	}
	if u.Scheme != "redis" && u.Scheme != "rediss" {
		return redisConfig{}, fmt.Errorf("unsupported redis url scheme %q", u.Scheme)
	}
	address := u.Host
	if address == "" {
		address = net.JoinHostPort(defaultRedisHost, defaultRedisPort)
	}
	if _, _, err := net.SplitHostPort(address); err != nil {
		address = net.JoinHostPort(address, defaultRedisPort)
	}
	password, _ := u.User.Password()
	database := strings.TrimPrefix(u.Path, "/")
	if database == "" {
		database = defaultRedisDB
	}
	return redisConfig{
		Address:  address,
		Username: u.User.Username(),
		Password: password,
		Database: database,
		UseTLS:   u.Scheme == "rediss" || parseBoolish(u.Query().Get("tls")),
	}, nil
}

func newRedisClient(ctx context.Context, cfg redisConfig) (*redisClient, error) {
	var tlsConfig *tls.Config
	if cfg.UseTLS {
		host, _, err := net.SplitHostPort(cfg.Address)
		if err != nil {
			host = cfg.Address
		}
		tlsConfig = &tls.Config{MinVersion: tls.VersionTLS12, ServerName: host}
	}
	conn, err := dialRedis(ctx, "tcp", cfg.Address, tlsConfig)
	if err != nil {
		return nil, err
	}
	client := &redisClient{
		conn: conn,
		rw:   bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn)),
	}
	if cfg.Password != "" || cfg.Username != "" {
		authArgs := []string{"AUTH"}
		if cfg.Username != "" {
			authArgs = append(authArgs, cfg.Username, cfg.Password)
		} else {
			authArgs = append(authArgs, cfg.Password)
		}
		if _, err := client.simpleCommand(ctx, authArgs...); err != nil {
			client.Close()
			return nil, err
		}
	}
	if cfg.Database != "" && cfg.Database != defaultRedisDB {
		if _, err := client.simpleCommand(ctx, "SELECT", cfg.Database); err != nil {
			client.Close()
			return nil, err
		}
	}
	return client, nil
}

func (c *redisClient) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *redisClient) command(ctx context.Context, args ...string) (redisValue, error) {
	if deadline, ok := ctx.Deadline(); ok {
		_ = c.conn.SetDeadline(deadline)
	} else {
		_ = c.conn.SetDeadline(time.Now().Add(5 * time.Second))
	}
	if err := writeRESPArray(c.rw.Writer, args); err != nil {
		return redisValue{}, err
	}
	if err := c.rw.Flush(); err != nil {
		return redisValue{}, err
	}
	return readRESP(c.rw.Reader)
}

func (c *redisClient) simpleCommand(ctx context.Context, args ...string) (string, error) {
	value, err := c.command(ctx, args...)
	if err != nil {
		return "", err
	}
	return value.text, nil
}

func (c *redisClient) integerCommand(ctx context.Context, args ...string) (int64, error) {
	value, err := c.command(ctx, args...)
	if err != nil {
		return 0, err
	}
	return value.int64, nil
}

func (c *redisClient) getCommand(ctx context.Context, args ...string) (string, bool, error) {
	value, err := c.command(ctx, args...)
	if err != nil {
		return "", false, err
	}
	if value.null {
		return "", false, nil
	}
	return value.text, true, nil
}

func (c *redisClient) hashCommand(ctx context.Context, args ...string) (map[string]string, error) {
	value, err := c.command(ctx, args...)
	if err != nil {
		return nil, err
	}
	parts, err := arrayAsStrings(value)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for i := 0; i < len(parts); i += 2 {
		if i+1 >= len(parts) {
			break
		}
		out[parts[i]] = parts[i+1]
	}
	return out, nil
}

func (c *redisClient) arrayStringsCommand(ctx context.Context, args ...string) ([]string, error) {
	value, err := c.command(ctx, args...)
	if err != nil {
		return nil, err
	}
	return arrayAsStrings(value)
}

func (c *redisClient) infoCommand(ctx context.Context, args ...string) (map[string]string, error) {
	value, ok, err := c.getCommand(ctx, args...)
	if err != nil {
		return nil, err
	}
	if !ok {
		return map[string]string{}, nil
	}
	out := map[string]string{}
	for _, line := range strings.Split(value, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, ":")
		if ok {
			out[key] = val
		}
	}
	return out, nil
}

func (c *redisClient) streamEntriesCommand(ctx context.Context, args ...string) ([]map[string]any, error) {
	value, err := c.command(ctx, args...)
	if err != nil {
		return nil, err
	}
	if value.kind != '*' {
		return nil, fmt.Errorf("expected array response")
	}
	out := make([]map[string]any, 0, len(value.array))
	for _, entry := range value.array {
		if entry.kind != '*' || len(entry.array) < 2 {
			continue
		}
		id := entry.array[0].text
		fields, err := arrayAsStrings(entry.array[1])
		if err != nil {
			return nil, err
		}
		fieldMap := map[string]any{}
		for i := 0; i < len(fields); i += 2 {
			if i+1 >= len(fields) {
				break
			}
			fieldMap[fields[i]] = fields[i+1]
		}
		out = append(out, map[string]any{
			"id":     id,
			"fields": fieldMap,
		})
	}
	return out, nil
}

func (c *redisClient) scanKeys(ctx context.Context, cursor string, count int) ([]string, string, error) {
	value, err := c.command(ctx, "SCAN", cursor, "COUNT", strconv.Itoa(count))
	if err != nil {
		return nil, "", err
	}
	if value.kind != '*' || len(value.array) != 2 {
		return nil, "", fmt.Errorf("unexpected SCAN response")
	}
	nextCursor := value.array[0].text
	keys, err := arrayAsStrings(value.array[1])
	if err != nil {
		return nil, "", err
	}
	return keys, nextCursor, nil
}

func splitRedisCommand(input string) ([]string, error) {
	var out []string
	var cur strings.Builder
	var quote rune
	escaped := false

	flush := func() {
		if cur.Len() == 0 {
			return
		}
		out = append(out, cur.String())
		cur.Reset()
	}

	for _, r := range strings.TrimSpace(input) {
		if escaped {
			cur.WriteRune(r)
			escaped = false
			continue
		}
		switch {
		case r == '\\':
			escaped = true
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '"' || r == '\'':
			quote = r
		case r == ' ' || r == '\t' || r == '\n' || r == '\r':
			flush()
		default:
			cur.WriteRune(r)
		}
	}
	if escaped {
		return nil, fmt.Errorf("unfinished escape sequence")
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quoted string")
	}
	flush()
	return out, nil
}

func writeRESPArray(w *bufio.Writer, args []string) error {
	if _, err := fmt.Fprintf(w, "*%d\r\n", len(args)); err != nil {
		return err
	}
	for _, arg := range args {
		if _, err := fmt.Fprintf(w, "$%d\r\n%s\r\n", len(arg), arg); err != nil {
			return err
		}
	}
	return nil
}

func readRESP(r *bufio.Reader) (redisValue, error) {
	prefix, err := r.ReadByte()
	if err != nil {
		return redisValue{}, err
	}
	switch prefix {
	case '+':
		line, err := readLine(r)
		if err != nil {
			return redisValue{}, err
		}
		return redisValue{kind: prefix, text: line}, nil
	case '-':
		line, err := readLine(r)
		if err != nil {
			return redisValue{}, err
		}
		return redisValue{}, fmt.Errorf("%s", line)
	case ':':
		line, err := readLine(r)
		if err != nil {
			return redisValue{}, err
		}
		n, err := strconv.ParseInt(line, 10, 64)
		if err != nil {
			return redisValue{}, err
		}
		return redisValue{kind: prefix, int64: n}, nil
	case '$':
		line, err := readLine(r)
		if err != nil {
			return redisValue{}, err
		}
		n, err := strconv.Atoi(line)
		if err != nil {
			return redisValue{}, err
		}
		if n == -1 {
			return redisValue{kind: prefix, null: true}, nil
		}
		buf := make([]byte, n+2)
		if _, err := r.Read(buf); err != nil {
			return redisValue{}, err
		}
		return redisValue{kind: prefix, text: string(buf[:n])}, nil
	case '*':
		line, err := readLine(r)
		if err != nil {
			return redisValue{}, err
		}
		n, err := strconv.Atoi(line)
		if err != nil {
			return redisValue{}, err
		}
		if n == -1 {
			return redisValue{kind: prefix, null: true}, nil
		}
		out := make([]redisValue, 0, n)
		for i := 0; i < n; i++ {
			item, err := readRESP(r)
			if err != nil {
				return redisValue{}, err
			}
			out = append(out, item)
		}
		return redisValue{kind: prefix, array: out}, nil
	default:
		return redisValue{}, fmt.Errorf("unsupported RESP prefix %q", prefix)
	}
}

func readLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r"), nil
}

func arrayAsStrings(value redisValue) ([]string, error) {
	if value.kind != '*' {
		return nil, fmt.Errorf("expected array response")
	}
	out := make([]string, 0, len(value.array))
	for _, item := range value.array {
		if item.null {
			out = append(out, "")
			continue
		}
		switch item.kind {
		case '+', '$':
			out = append(out, item.text)
		case ':':
			out = append(out, strconv.FormatInt(item.int64, 10))
		default:
			return nil, fmt.Errorf("unsupported array item type %q", item.kind)
		}
	}
	return out, nil
}

func redisGroupForType(keyType string) (string, string) {
	switch keyType {
	case "string":
		return "string", "String keys"
	case "hash":
		return "hash", "Hash keys"
	case "list":
		return "list", "List keys"
	case "set":
		return "set", "Set keys"
	case "zset":
		return "zset", "Sorted set keys"
	case "stream":
		return "stream", "Stream keys"
	default:
		return "other", "Other keys"
	}
}

func redisInspectTitle(keyType string) string {
	switch keyType {
	case "hash":
		return "Inspect hash"
	case "list":
		return "Inspect list"
	case "set":
		return "Inspect set"
	case "zset":
		return "Inspect sorted set"
	case "stream":
		return "Inspect stream"
	default:
		return "Inspect value"
	}
}

func redisInspectQuery(keyType, key string) string {
	quoted := strconv.Quote(key)
	switch keyType {
	case "hash":
		return "HGETALL " + quoted
	case "list":
		return fmt.Sprintf("LRANGE %s 0 %d", quoted, defaultValueLimit-1)
	case "set":
		return "SMEMBERS " + quoted
	case "zset":
		return fmt.Sprintf("ZRANGE %s 0 %d WITHSCORES", quoted, defaultValueLimit-1)
	case "stream":
		return fmt.Sprintf("XRANGE %s - + COUNT %d", quoted, defaultValueLimit)
	default:
		return "GET " + quoted
	}
}

func parseBoolish(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func kvResponse(data map[string]string) *plugin.ExecResponse {
	return &plugin.ExecResponse{
		Result: &plugin.ExecResult{
			Payload: &pluginpb.PluginV1_ExecResult_Kv{
				Kv: &plugin.KeyValueResult{Data: data},
			},
		},
	}
}

func documentResponse(items []map[string]any) (*plugin.ExecResponse, error) {
	docs := make([]*structpb.Struct, 0, len(items))
	for _, item := range items {
		doc, err := structpb.NewStruct(item)
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	return &plugin.ExecResponse{
		Result: &plugin.ExecResult{
			Payload: &pluginpb.PluginV1_ExecResult_Document{
				Document: &plugin.DocumentResult{Documents: docs},
			},
		},
	}, nil
}

func main() {
	plugin.ServeCLI(&redisPlugin{})
}
