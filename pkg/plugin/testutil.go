package plugin

import "encoding/json"

// MakeTestBlob returns a JSON-encoded credential blob string from the given
// key-value pairs.  It is intended only for use in tests.
func MakeTestBlob(vals map[string]string) string {
	payload := struct {
		Form   string            `json:"form"`
		Values map[string]string `json:"values"`
	}{Form: "basic", Values: vals}
	b, _ := json.Marshal(payload)
	return string(b)
}
