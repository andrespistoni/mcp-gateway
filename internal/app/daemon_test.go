package app

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"mcp-gateway/internal/config"
	"mcp-gateway/internal/daemon"
	"mcp-gateway/internal/endpoint"
)

type missingDaemonConfig struct{}

func (missingDaemonConfig) Load(context.Context) (config.Snapshot, error) {
	return config.Snapshot{}, os.ErrNotExist
}

type fakeDaemonManager struct {
	status     daemon.Status
	spec       daemon.Spec
	calls      []string
	err        error
	restartErr error
}

func (f *fakeDaemonManager) Status(context.Context) (daemon.Status, error) { return f.status, f.err }
func (f *fakeDaemonManager) Enable(_ context.Context, spec daemon.Spec) error {
	f.calls = append(f.calls, "enable")
	f.spec = spec
	return f.err
}
func (f *fakeDaemonManager) Disable(context.Context) error {
	f.calls = append(f.calls, "disable")
	return f.err
}
func (f *fakeDaemonManager) Restart(context.Context) error {
	f.calls = append(f.calls, "restart")
	if f.restartErr != nil {
		return f.restartErr
	}
	return f.err
}

func TestDaemonLifecycleUsesCurrentAbsoluteBinaryAndResolvedPort(t *testing.T) {
	manager := &fakeDaemonManager{status: daemon.Status{Installed: true, Running: true}}
	runtime := &Runtime{config: missingDaemonConfig{}, daemonManager: manager}
	port := endpoint.MustPort(4444)
	if err := runtime.EnableDaemon(context.Background(), EnableDaemonRequest{Port: &port}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(manager.spec.Binary(), string(os.PathSeparator)) || manager.spec.Port().Number() != 4444 || strings.Join(manager.spec.Args(), " ") != "serve --port 4444" {
		t.Fatalf("spec = binary %q args %q", manager.spec.Binary(), manager.spec.Args())
	}
	if err := runtime.DisableDaemon(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := runtime.Restart(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(manager.calls, ","); got != "enable,disable,restart" {
		t.Fatalf("calls = %q", got)
	}
	status, err := runtime.DaemonStatus(context.Background())
	if err != nil || !status.Installed || !status.Running {
		t.Fatalf("status = %#v, %v", status, err)
	}
}

func TestDaemonLifecycleWrapsManagerFailure(t *testing.T) {
	runtime := &Runtime{config: missingDaemonConfig{}, daemonManager: &fakeDaemonManager{err: errors.New("private detail")}}
	if err := runtime.DisableDaemon(context.Background()); err == nil {
		t.Fatal("DisableDaemon tuvo éxito")
	}
}
