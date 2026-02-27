//go:build !windows
// +build !windows

package pluginmgr

import "os/exec"

// hideWindow is a no-op on non-Windows platforms. It exists so that the
// main package can call the function unconditionally without build errors.
func hideWindow(cmd *exec.Cmd) {
    // nothing to do
}
