//go:build linux || (darwin && arm64)

package main

// Import libsql only on platforms where go-libsql ships a native archive.
import _ "github.com/tursodatabase/go-libsql"
