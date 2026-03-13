package plugin

import (
	"encoding/json"
	"fmt"
)

// CredentialBlob represents the JSON structure stored under the
// "credential_blob" key in the connection map.  All plugins that accept
// form-based authentication share this shape.
type CredentialBlob struct {
	Form   string            `json:"form"`
	Values map[string]string `json:"values"`
}

// ParseCredentialBlob extracts and decodes the "credential_blob" entry from a
// connection map.  Returns an error if the key is missing/empty or the JSON is
// malformed.
func ParseCredentialBlob(connection map[string]string) (CredentialBlob, error) {
	blob, ok := connection["credential_blob"]
	if !ok || blob == "" {
		return CredentialBlob{}, fmt.Errorf("missing credential_blob in connection parameters")
	}
	var cb CredentialBlob
	if err := json.Unmarshal([]byte(blob), &cb); err != nil {
		return CredentialBlob{}, fmt.Errorf("invalid credential_blob JSON: %w", err)
	}
	return cb, nil
}
