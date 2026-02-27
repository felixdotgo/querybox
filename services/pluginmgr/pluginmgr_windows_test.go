//go:build windows
// +build windows

package pluginmgr

import (
	"os/exec"
	"testing"
)

func TestHideWindowSetsFlag(t *testing.T) {
    cmd := exec.Command("echo", "hello")
    if cmd.SysProcAttr != nil && cmd.SysProcAttr.HideWindow {
        // already hidden for some reason, still valid
        return
    }
    hideWindow(cmd)
    if cmd.SysProcAttr == nil {
        t.Fatal("SysProcAttr should be non-nil after hideWindow")
    }
    if !cmd.SysProcAttr.HideWindow {
        t.Errorf("expected HideWindow=true, got %+v", cmd.SysProcAttr)
    }
}
