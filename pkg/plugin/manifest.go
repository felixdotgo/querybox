package plugin

import (
	"fmt"
	"strings"
)

const (
	ManifestFileSuffix = ".manifest.json"

	RuntimeKindLocal = "local"
)

var SupportedCapabilitiesV1 = map[string]struct{}{
	"connection.test":    {},
	"explain-query":      {},
	"mutate-row":         {},
	"mutate-row::delete": {},
	"mutate-row::edit":   {},
	"query.execute":      {},
	"resource.graph":     {},
	"schema.inspect":     {},
	"stream.read":        {},
}

type Manifest struct {
	ID           string            `json:"id"`
	Type         int               `json:"type"`
	Name         string            `json:"name,omitempty"`
	Description  string            `json:"description,omitempty"`
	Version      string            `json:"version"`
	Runtime      RuntimeSpec       `json:"runtime"`
	Capabilities []string          `json:"capabilities"`
	Permissions  []PermissionDecl  `json:"permissions,omitempty"`
	Limits       Limits            `json:"limits"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type RuntimeSpec struct {
	Kind       string   `json:"kind"`
	Entrypoint string   `json:"entrypoint,omitempty"`
	Args       []string `json:"args,omitempty"`
}

type PermissionDecl struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

type Limits struct {
	TimeoutSeconds int      `json:"timeout_seconds,omitempty"`
	MaxOutputBytes int      `json:"max_output_bytes,omitempty"`
	WorkingDir     string   `json:"working_dir,omitempty"`
	EnvAllowlist   []string `json:"env_allowlist,omitempty"`
}

func ValidateManifest(manifest Manifest, expectedID string) error {
	id := strings.TrimSpace(manifest.ID)
	if id == "" {
		return fmt.Errorf("manifest id is required")
	}
	if expectedID != "" && id != expectedID {
		return fmt.Errorf("manifest id %q does not match plugin binary %q", id, expectedID)
	}
	if strings.TrimSpace(manifest.Version) == "" {
		return fmt.Errorf("manifest version is required")
	}
	if manifest.Type <= 0 {
		return fmt.Errorf("manifest type is required")
	}
	if strings.TrimSpace(manifest.Runtime.Kind) == "" {
		return fmt.Errorf("manifest runtime.kind is required")
	}
	if manifest.Runtime.Kind != RuntimeKindLocal {
		return fmt.Errorf("unsupported runtime kind %q", manifest.Runtime.Kind)
	}
	for _, capability := range manifest.Capabilities {
		if _, ok := SupportedCapabilitiesV1[capability]; !ok {
			return fmt.Errorf("unsupported capability %q", capability)
		}
	}
	for _, permission := range manifest.Permissions {
		if strings.TrimSpace(permission.Name) == "" {
			return fmt.Errorf("permission name is required")
		}
	}
	if manifest.Limits.TimeoutSeconds < 0 {
		return fmt.Errorf("limits.timeout_seconds must be >= 0")
	}
	if manifest.Limits.MaxOutputBytes < 0 {
		return fmt.Errorf("limits.max_output_bytes must be >= 0")
	}
	return nil
}
