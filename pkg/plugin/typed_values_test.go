package plugin_test

import (
	"testing"

	"github.com/felixdotgo/querybox/pkg/plugin"
)

func TestNewValueMapAndSQLArgFromValue(t *testing.T) {
	values, err := plugin.NewValueMap(map[string]interface{}{
		"active": true,
		"age":    float64(25),
		"meta":   map[string]interface{}{"role": "admin"},
		"name":   nil,
		"title":  "Alice",
	})
	if err != nil {
		t.Fatalf("NewValueMap: %v", err)
	}

	cases := []struct {
		key  string
		want interface{}
	}{
		{"active", true},
		{"age", float64(25)},
		{"meta", `{"role":"admin"}`},
		{"name", nil},
		{"title", "Alice"},
	}
	for _, tc := range cases {
		got, err := plugin.SQLArgFromValue(values[tc.key])
		if err != nil {
			t.Fatalf("SQLArgFromValue(%s): %v", tc.key, err)
		}
		if got != tc.want {
			t.Errorf("SQLArgFromValue(%s) = %#v, want %#v", tc.key, got, tc.want)
		}
	}
	if !plugin.IsNullValue(values["name"]) {
		t.Errorf("expected nil source to become a protobuf null value")
	}
}
