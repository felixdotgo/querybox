package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"net"
	"reflect"
	"strings"
	"testing"

	"github.com/felixdotgo/querybox/pkg/plugin"
)

func TestRedisConfigFromConnectionBlobURL(t *testing.T) {
	conn := map[string]string{
		"credential_blob": plugin.MakeTestBlob(map[string]string{
			"url": "rediss://alice:secret@redis.example.com:6380/2",
		}),
	}
	cfg, err := redisConfigFromConnection(conn)
	if err != nil {
		t.Fatalf("redisConfigFromConnection error: %v", err)
	}
	if cfg.Address != "redis.example.com:6380" {
		t.Fatalf("unexpected address: %s", cfg.Address)
	}
	if cfg.Username != "alice" || cfg.Password != "secret" {
		t.Fatalf("unexpected auth config: %+v", cfg)
	}
	if cfg.Database != "2" || !cfg.UseTLS {
		t.Fatalf("unexpected db/tls config: %+v", cfg)
	}
}

func TestExecGetReturnsKeyValue(t *testing.T) {
	restore := stubRedisDialer(t, func(args []string) string {
		switch strings.ToUpper(args[0]) {
		case "GET":
			if len(args) != 2 || args[1] != "session:1" {
				t.Fatalf("unexpected GET args: %v", args)
			}
			return "$5\r\nhello\r\n"
		default:
			t.Fatalf("unexpected command: %v", args)
			return ""
		}
	})
	defer restore()

	p := &redisPlugin{}
	resp, err := p.Exec(context.Background(), &plugin.ExecRequest{
		Connection: map[string]string{"credential_blob": plugin.MakeTestBlob(map[string]string{"host": "localhost"})},
		Query:      `GET "session:1"`,
	})
	if err != nil {
		t.Fatalf("Exec error: %v", err)
	}
	got := resp.GetResult().GetKv().GetData()
	if got["key"] != "session:1" || got["value"] != "hello" {
		t.Fatalf("unexpected kv payload: %+v", got)
	}
}

func TestResourceGraphGroupsKeysByType(t *testing.T) {
	restore := stubRedisDialer(t, func(args []string) string {
		switch strings.ToUpper(args[0]) {
		case "SCAN":
			return "*2\r\n$1\r\n0\r\n*3\r\n$9\r\nsession-1\r\n$7\r\nuser:42\r\n$6\r\nqueue1\r\n"
		case "TYPE":
			switch args[1] {
			case "session-1":
				return "+string\r\n"
			case "user:42":
				return "+hash\r\n"
			case "queue1":
				return "+list\r\n"
			default:
				t.Fatalf("unexpected TYPE key: %v", args)
			}
		case "TTL":
			return ":-1\r\n"
		default:
			t.Fatalf("unexpected command: %v", args)
		}
		return ""
	})
	defer restore()

	p := &redisPlugin{}
	graph, err := p.ResourceGraph(context.Background(), &plugin.ResourceGraphRequest{
		Connection: map[string]string{"credential_blob": plugin.MakeTestBlob(map[string]string{"host": "localhost"})},
	})
	if err != nil {
		t.Fatalf("ResourceGraph error: %v", err)
	}
	if len(graph.Nodes) != 4 {
		t.Fatalf("expected 4 top-level nodes, got %d", len(graph.Nodes))
	}
	if graph.Nodes[0].Kind != "action" || graph.Nodes[0].Name != "Server info" {
		t.Fatalf("unexpected first node: %+v", graph.Nodes[0])
	}
	groupNames := []string{graph.Nodes[1].Name, graph.Nodes[2].Name, graph.Nodes[3].Name}
	wantGroups := []string{"String keys", "Hash keys", "List keys"}
	if !reflect.DeepEqual(groupNames, wantGroups) {
		t.Fatalf("unexpected group names: got %v want %v", groupNames, wantGroups)
	}
	hashNode := graph.Nodes[2].Children[0]
	if hashNode.Name != "user:42" {
		t.Fatalf("unexpected hash node: %+v", hashNode)
	}
	if hashNode.Actions[0].Query != `HGETALL "user:42"` {
		t.Fatalf("unexpected inspect query: %+v", hashNode.Actions[0])
	}
	if hashNode.Actions[1].Kind != "delete-key" || hashNode.Actions[1].Query != `DEL "user:42"` {
		t.Fatalf("unexpected delete action: %+v", hashNode.Actions[1])
	}
}

func TestResourceGraphScanMarkerExposesInspectAction(t *testing.T) {
	restore := stubRedisDialer(t, func(args []string) string {
		switch strings.ToUpper(args[0]) {
		case "SCAN":
			return "*2\r\n$2\r\n12\r\n*1\r\n$9\r\nsession-1\r\n"
		case "TYPE":
			return "+string\r\n"
		case "TTL":
			return ":-1\r\n"
		default:
			t.Fatalf("unexpected command: %v", args)
		}
		return ""
	})
	defer restore()

	p := &redisPlugin{}
	graph, err := p.ResourceGraph(context.Background(), &plugin.ResourceGraphRequest{
		Connection: map[string]string{"credential_blob": plugin.MakeTestBlob(map[string]string{"host": "localhost"})},
	})
	if err != nil {
		t.Fatalf("ResourceGraph error: %v", err)
	}
	marker := graph.Nodes[len(graph.Nodes)-1]
	if marker.ID != "__scan_limited__" {
		t.Fatalf("expected scan marker, got %+v", marker)
	}
	if marker.Metadata["scan_cursor"] != "12" {
		t.Fatalf("unexpected scan cursor metadata: %+v", marker.Metadata)
	}
	if len(marker.Actions) != 1 || marker.Actions[0].Query != `SCAN "12" COUNT 50` || !marker.Actions[0].NewTab {
		t.Fatalf("unexpected marker action: %+v", marker.Actions)
	}
}

func TestExecScanReturnsDocumentRows(t *testing.T) {
	restore := stubRedisDialer(t, func(args []string) string {
		if !reflect.DeepEqual(args, []string{"SCAN", "12", "COUNT", "2"}) {
			t.Fatalf("unexpected SCAN args: %v", args)
		}
		return "*2\r\n$1\r\n0\r\n*2\r\n$1\r\nb\r\n$1\r\na\r\n"
	})
	defer restore()

	p := &redisPlugin{}
	resp, err := p.Exec(context.Background(), &plugin.ExecRequest{
		Connection: map[string]string{"credential_blob": plugin.MakeTestBlob(map[string]string{"host": "localhost"})},
		Query:      `SCAN "12" COUNT 2`,
	})
	if err != nil {
		t.Fatalf("Exec error: %v", err)
	}
	docs := resp.GetResult().GetDocument().GetDocuments()
	if len(docs) != 2 {
		t.Fatalf("expected 2 docs, got %d", len(docs))
	}
	if docs[0].Fields["key"].GetStringValue() != "a" || docs[1].Fields["key"].GetStringValue() != "b" {
		t.Fatalf("unexpected scan docs: %+v", docs)
	}
}

func stubRedisDialer(t *testing.T, handler func(args []string) string) func() {
	t.Helper()
	orig := dialRedis
	dialRedis = func(ctx context.Context, network, address string, tlsConfig *tls.Config) (net.Conn, error) {
		client, server := net.Pipe()
		go func() {
			defer server.Close()
			reader := bufio.NewReader(server)
			writer := bufio.NewWriter(server)
			for {
				value, err := readRESP(reader)
				if err != nil {
					return
				}
				args, err := arrayAsStrings(value)
				if err != nil {
					t.Errorf("arrayAsStrings error: %v", err)
					return
				}
				reply := handler(args)
				if _, err := writer.WriteString(reply); err != nil {
					return
				}
				if err := writer.Flush(); err != nil {
					return
				}
			}
		}()
		return client, nil
	}
	return func() { dialRedis = orig }
}
