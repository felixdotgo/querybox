//go:build !windows
// +build !windows

package main

// import libsql driver on non-windows platforms so the "libsql"
// driver name is registered.  the package doesn't build on
// windows, hence the build constraint.
import _ "github.com/tursodatabase/go-libsql"
