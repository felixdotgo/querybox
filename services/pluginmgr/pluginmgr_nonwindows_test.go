//go:build !windows
// +build !windows

package pluginmgr

import (
	"os/exec"
	"testing"
)

func TestHideWindowNoop(t *testing.T) {
    cmd := exec.Command("echo", "hi")
    hideWindow(cmd)
    if cmd.SysProcAttr != nil {
        t.Errorf("expected SysProcAttr to remain nil on non-Windows, got %+v", cmd.SysProcAttr)
    }
}
