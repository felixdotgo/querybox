package plugin

import "testing"

func TestValidateManifestAllowsExtendedCapabilities(t *testing.T) {
	manifest := Manifest{
		ID:      "postgresql",
		Type:    1,
		Version: "1.0.0",
		Runtime: RuntimeSpec{Kind: RuntimeKindLocal},
		Capabilities: []string{
			"resource.graph",
			"query.execute",
			"schema.inspect",
			"explain-query",
			"mutate-row",
		},
		Limits: Limits{TimeoutSeconds: 30},
	}

	if err := ValidateManifest(manifest, "postgresql"); err != nil {
		t.Fatalf("ValidateManifest returned error for supported capabilities: %v", err)
	}
}

func TestValidateManifestRejectsUnknownCapability(t *testing.T) {
	manifest := Manifest{
		ID:           "postgresql",
		Type:         1,
		Version:      "1.0.0",
		Runtime:      RuntimeSpec{Kind: RuntimeKindLocal},
		Capabilities: []string{"resource.graph", "nope"},
		Limits:       Limits{TimeoutSeconds: 30},
	}

	if err := ValidateManifest(manifest, "postgresql"); err == nil {
		t.Fatal("ValidateManifest unexpectedly accepted unknown capability")
	}
}
