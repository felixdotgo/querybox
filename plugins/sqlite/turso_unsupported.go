//go:build !(linux || (darwin && arm64))

package main

func tursoCloudSupported() bool {
	return false
}
