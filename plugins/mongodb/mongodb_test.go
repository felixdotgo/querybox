package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// helper: build a credential_blob with form="basic" and the given values.
func makeBasicBlob(vals map[string]string) map[string]string {
	payload := map[string]interface{}{"form": "basic", "values": vals}
	b, _ := json.Marshal(payload)
	return map[string]string{"credential_blob": string(b)}
}

// helper: build a credential_blob with form="uri" and the given URI.
func makeURIBlob(uri string) map[string]string {
	payload := map[string]interface{}{"form": "uri", "values": map[string]string{"uri": uri}}
	b, _ := json.Marshal(payload)
	return map[string]string{"credential_blob": string(b)}
}

func TestBuildURI(t *testing.T) {
	tests := []struct {
		name    string
		conn    map[string]string
		want    string // substring expected in URI
		wantErr bool
	}{
		{
			name: "direct uri key",
			conn: map[string]string{"uri": "mongodb://localhost:27017"},
			want: "mongodb://localhost:27017",
		},
		{
			name: "uri in credential_blob",
			conn: makeURIBlob("mongodb://admin:s3cr3t@db.example.com:27017/prod"),
			want: "mongodb://admin:s3cr3t@db.example.com:27017/prod",
		},
		{
			name: "basic host and port only",
			conn: makeBasicBlob(map[string]string{"host": "192.168.1.1", "port": "27017"}),
			want: "mongodb://192.168.1.1:27017",
		},
		{
			name: "basic with user and database",
			conn: makeBasicBlob(map[string]string{
				"host":     "mongo.local",
				"port":     "27017",
				"user":     "alice",
				"password": "pass123",
				"database": "myapp",
			}),
			want: "mongodb://alice:pass123@mongo.local:27017/myapp",
		},
		{
			name: "plain map fallback (no blob)",
			conn: map[string]string{"host": "127.0.0.1", "port": "27017"},
			want: "mongodb://127.0.0.1:27017",
		},
		{
			name: "tls=true adds tls param",
			conn: makeBasicBlob(map[string]string{
				"host": "localhost",
				"port": "27017",
				"tls":  "true",
			}),
			want: "tls=true",
		},
		{
			name: "invalid blob returns error",
			conn: map[string]string{"credential_blob": "not-json"},
			want: "", wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uri, _, err := buildURI(tt.conn)
			if (err != nil) != tt.wantErr {
				t.Fatalf("buildURI() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.want != "" && !strings.Contains(uri, tt.want) {
				t.Errorf("buildURI() = %q, want substring %q", uri, tt.want)
			}
		})
	}
}

func TestGetDatabaseName(t *testing.T) {
	tests := []struct {
		name   string
		conn   map[string]string
		wantDB string
	}{
		{"empty conn", map[string]string{}, ""},
		{"direct key", map[string]string{"database": "mydb"}, "mydb"},
		{"from blob", makeBasicBlob(map[string]string{"database": "appdb"}), "appdb"},
		{"blob without database", makeBasicBlob(map[string]string{"host": "localhost"}), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getDatabaseName(tt.conn)
			if got != tt.wantDB {
				t.Errorf("getDatabaseName() = %q, want %q", got, tt.wantDB)
			}
		})
	}
}

func TestSplitTopLevelArgs(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{
			input: `{"a": 1}`,
			want:  []string{`{"a": 1}`},
		},
		{
			input: `{"a": 1}, {"b": 2}`,
			want:  []string{`{"a": 1}`, `{"b": 2}`},
		},
		{
			input: `{"a": {"c": 1}}, {"b": 2}`,
			want:  []string{`{"a": {"c": 1}}`, `{"b": 2}`},
		},
		{
			input: `[{"$match": {}}], {}`,
			want:  []string{`[{"$match": {}}]`, `{}`},
		},
		{
			input: `"field", {"x": 1}`,
			want:  []string{`"field"`, `{"x": 1}`},
		},
		{
			input: `{}`,
			want:  []string{`{}`},
		},
		{
			input: ``,
			want:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := splitTopLevelArgs(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("splitTopLevelArgs(%q) = %v (len %d), want %v (len %d)",
					tt.input, got, len(got), tt.want, len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("arg[%d]: got %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestParseMQLCommand(t *testing.T) {
	tests := []struct {
		query      string
		wantTarget string
		wantOp     string
		wantArgs   string
		wantOk     bool
	}{
		{
			query:      `db.users.find({})`,
			wantTarget: "users", wantOp: "find", wantArgs: "{}", wantOk: true,
		},
		{
			query:      `db.users.findOne({"_id": 1})`,
			wantTarget: "users", wantOp: "findOne", wantArgs: `{"_id": 1}`, wantOk: true,
		},
		{
			query:      `db.orders.updateOne({"_id": 1}, {"$set": {"status": "done"}})`,
			wantTarget: "orders", wantOp: "updateOne",
			wantArgs: `{"_id": 1}, {"$set": {"status": "done"}}`, wantOk: true,
		},
		{
			query:      `db.logs.aggregate([{"$match": {}}, {"$count": "total"}])`,
			wantTarget: "logs", wantOp: "aggregate",
			wantArgs: `[{"$match": {}}, {"$count": "total"}]`, wantOk: true,
		},
		{
			query:      `db.dropDatabase()`,
			wantTarget: "", wantOp: "dropDatabase", wantArgs: "", wantOk: true,
		},
		{
			query:      `db.createCollection("events")`,
			wantTarget: "", wantOp: "createCollection", wantArgs: `"events"`, wantOk: true,
		},
		{
			query:   `{"ping": 1}`,
			wantOk:  false, // raw command, not shell syntax
		},
		{
			query:   `SELECT * FROM users`,
			wantOk:  false, // SQL, not MQL
		},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			target, op, args, ok := parseMQLCommand(tt.query)
			if ok != tt.wantOk {
				t.Fatalf("parseMQLCommand(%q): ok=%v, want %v", tt.query, ok, tt.wantOk)
			}
			if !tt.wantOk {
				return
			}
			if target != tt.wantTarget {
				t.Errorf("target: got %q, want %q", target, tt.wantTarget)
			}
			if op != tt.wantOp {
				t.Errorf("op: got %q, want %q", op, tt.wantOp)
			}
			if args != tt.wantArgs {
				t.Errorf("args: got %q, want %q", args, tt.wantArgs)
			}
		})
	}
}

func TestParseBSONDoc(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"empty string", "", false},
		{"empty braces", "{}", false},
		{"simple object", `{"name": "Alice", "age": 30}`, false},
		{"nested", `{"a": {"b": {"c": 1}}}`, false},
		{"unquoted key – invalid JSON", `{name: "Alice"}`, true},
		{"array instead of doc", `[1, 2, 3]`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseBSONDoc(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseBSONDoc(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestParseBSONArray(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantLen int
		wantErr bool
	}{
		{"empty string", "", 0, false},
		{"empty array", "[]", 0, false},
		{"single stage", `[{"$match": {"status": "A"}}]`, 1, false},
		{"two stages", `[{"$match": {}}, {"$limit": 10}]`, 2, false},
		{"invalid", `{not an array}`, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			arr, err := parseBSONArray(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseBSONArray(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if err == nil && len(arr) != tt.wantLen {
				t.Errorf("parseBSONArray(%q) len = %d, want %d", tt.input, len(arr), tt.wantLen)
			}
		})
	}
}
