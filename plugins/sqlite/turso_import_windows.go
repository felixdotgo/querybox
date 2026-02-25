//go:build windows
// +build windows

package main

// stub file for windows; do not import go-libsql since it has no
// windows-compatible sources.  absence of the import means the
// "libsql" driver won't be registered and attempting to use it
// will error earlier in driverDSN.
