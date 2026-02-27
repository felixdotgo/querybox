//go:build windows
// +build windows

package pluginmgr

import (
	"os/exec"
	"syscall"
)

// hideWindow sets the necessary syscall attributes to prevent the
// console window from popping up when executing a subprocess on Windows.
// On non-Windows platforms this function is a no-op (see pluginmgr_nonwindows.go).
func hideWindow(cmd *exec.Cmd) {
    if cmd == nil {
        return
    }
    if cmd.SysProcAttr == nil {
        cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
    } else {
        cmd.SysProcAttr.HideWindow = true
    }
}
