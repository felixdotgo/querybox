// Package driverid provides helpers for working with plugin/driver identifiers.
package driverid

import (
	"path/filepath"
	"strings"
)

// Normalize strips any filesystem extension (e.g. ".exe") from a plugin
// identifier. This ensures the same driver name is used across platforms
// and avoids persisting stale Windows-specific suffixes.
func Normalize(dt string) string {
	if dt == "" {
		return dt
	}
	return strings.TrimSuffix(dt, filepath.Ext(dt))
}
