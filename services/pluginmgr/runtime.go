package pluginmgr

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/felixdotgo/querybox/pkg/plugin"
)

type RuntimeManager struct {
	local PluginHost
}

type LocalPluginHost struct{}

type PluginHost interface {
	Run(info PluginInfo, command string, timeout time.Duration, req []byte) ([]byte, error)
}

func NewRuntimeManager() *RuntimeManager {
	return &RuntimeManager{
		local: &LocalPluginHost{},
	}
}

func (m *RuntimeManager) Run(info PluginInfo, command string, timeout time.Duration, req []byte) ([]byte, error) {
	return m.local.Run(info, command, timeout, req)
}

func (h *LocalPluginHost) Run(info PluginInfo, command string, timeout time.Duration, req []byte) ([]byte, error) {
	entrypoint, err := resolveEntrypoint(info)
	if err != nil {
		return nil, err
	}
	timeout = resolveTimeout(timeout, info.Limits)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	args := []string{command}
	if info.Runtime != nil {
		args = append(args, info.Runtime.Args...)
	}
	cmd := exec.CommandContext(ctx, entrypoint, args...)
	hideWindow(cmd)
	cmd.Env = buildPluginEnv(info.ID)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe error: %w", err)
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe error: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe error: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start error: %w", err)
	}

	if _, err := stdin.Write(req); err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("stdin write error: %w", err)
	}
	if err := stdin.Close(); err != nil {
		return nil, fmt.Errorf("stdin close error: %w", err)
	}

	stdout, _ := io.ReadAll(stdoutPipe)
	stderr, _ := io.ReadAll(stderrPipe)

	if err := cmd.Wait(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("plugin timed out after %s", timeout)
		}
		return nil, fmt.Errorf("plugin exited: %w - stderr: %s", err, string(stderr))
	}

	return stdout, nil
}

func resolveEntrypoint(info PluginInfo) (string, error) {
	entrypoint := info.Path
	if info.Runtime != nil && info.Runtime.Entrypoint != "" {
		if filepath.IsAbs(info.Runtime.Entrypoint) {
			entrypoint = info.Runtime.Entrypoint
		} else {
			entrypoint = filepath.Join(filepath.Dir(info.Path), info.Runtime.Entrypoint)
		}
	}
	if entrypoint == "" {
		return "", fmt.Errorf("plugin %s has no runtime entrypoint", info.ID)
	}
	if !isExecutable(entrypoint) {
		return "", fmt.Errorf("plugin %s is not executable", info.ID)
	}
	return entrypoint, nil
}

func resolveTimeout(defaultTimeout time.Duration, limits *plugin.Limits) time.Duration {
	if limits == nil || limits.TimeoutSeconds <= 0 {
		return defaultTimeout
	}
	limited := time.Duration(limits.TimeoutSeconds) * time.Second
	if limited < defaultTimeout {
		return limited
	}
	return defaultTimeout
}

func buildPluginEnv(name string) []string {
	return append(os.Environ(), "QUERYBOX_PLUGIN_NAME="+name)
}
