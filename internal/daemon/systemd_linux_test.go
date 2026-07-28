//go:build linux

package daemon

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"mcp-gateway/internal/endpoint"
	"mcp-gateway/internal/persist"
)

type recordedCommand struct {
	binary string
	args   []string
}

type fakeRunner struct {
	calls   []recordedCommand
	fail    map[string]error
	results map[string]CommandResult
}

func (f *fakeRunner) Run(_ context.Context, binary string, args []string, _ int) (CommandResult, error) {
	f.calls = append(f.calls, recordedCommand{binary: binary, args: append([]string(nil), args...)})
	key := strings.Join(args, " ")
	if err := f.fail[key]; err != nil {
		return CommandResult{}, err
	}
	if result, ok := f.results[key]; ok {
		return result, nil
	}
	return CommandResult{}, nil
}

func TestSystemdEnableDisableRestartUsesOnlyNativeArgv(t *testing.T) {
	path := filepath.Join(t.TempDir(), "systemd", "user", systemdUnitName)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{}
	manager, err := newSystemdManager(path, persist.NewStore(), runner)
	if err != nil {
		t.Fatal(err)
	}
	spec, _ := NewSpec("/tmp/gateway with space", endpoint.MustPort(3333))
	if err := manager.Enable(context.Background(), spec); err != nil {
		t.Fatal(err)
	}
	unit, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(unit)
	for _, want := range []string{"ExecStart=\"/tmp/gateway with space\" serve --port 3333", "WantedBy=default.target"} {
		if !strings.Contains(text, want) {
			t.Fatalf("unidad no contiene %q:\n%s", want, text)
		}
	}
	for _, forbidden := range []string{"127.0.0.1", "TOKEN", "SECRET", "projectDir", "sh -c"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("unidad contiene %q:\n%s", forbidden, text)
		}
	}
	wantCalls := []string{
		"--user stop mcp-gateway", "--user daemon-reload", "--user enable --now mcp-gateway",
	}
	if len(runner.calls) != len(wantCalls) {
		t.Fatalf("calls = %#v", runner.calls)
	}
	for index, want := range wantCalls {
		if got := strings.Join(runner.calls[index].args, " "); got != want || runner.calls[index].binary != "systemctl" {
			t.Fatalf("call[%d] = %#v, want %q", index, runner.calls[index], want)
		}
	}
	if err := manager.Restart(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := manager.Disable(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("unidad sigue presente: %v", err)
	}
	if err := manager.Disable(context.Background()); err != nil {
		t.Fatalf("disable idempotente: %v", err)
	}
}

func TestSystemdRestartRequiresDefinition(t *testing.T) {
	manager, err := newSystemdManager(filepath.Join(t.TempDir(), systemdUnitName), persist.NewStore(), &fakeRunner{})
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Restart(context.Background()); err == nil {
		t.Fatal("Restart tuvo éxito sin definición")
	}
}

func TestSystemdStatusAndCancellation(t *testing.T) {
	path := filepath.Join(t.TempDir(), systemdUnitName)
	runner := &fakeRunner{results: map[string]CommandResult{
		"--user is-active --quiet mcp-gateway": {ExitCode: 3},
	}}
	manager, err := newSystemdManager(path, persist.NewStore(), runner)
	if err != nil {
		t.Fatal(err)
	}
	status, err := manager.Status(context.Background())
	if err != nil || status.Installed {
		t.Fatalf("missing status = %#v, %v", status, err)
	}
	if err := os.WriteFile(path, []byte("unit"), 0o600); err != nil {
		t.Fatal(err)
	}
	status, err = manager.Status(context.Background())
	if err != nil || !status.Installed || status.Running {
		t.Fatalf("inactive status = %#v, %v", status, err)
	}
	runner.results["--user is-active --quiet mcp-gateway"] = CommandResult{ExitCode: 0}
	status, err = manager.Status(context.Background())
	if err != nil || !status.Running {
		t.Fatalf("active status = %#v, %v", status, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := manager.Status(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled status = %v", err)
	}
}

func TestSystemdEnableRestoresPreviousDefinition(t *testing.T) {
	path := filepath.Join(t.TempDir(), "systemd", "user", systemdUnitName)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("previous"), 0o600); err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{results: map[string]CommandResult{
		"--user enable --now mcp-gateway": {ExitCode: 1},
	}}
	manager, err := newSystemdManager(path, persist.NewStore(), runner)
	if err != nil {
		t.Fatal(err)
	}
	spec, _ := NewSpec("/tmp/gateway", endpoint.MustPort(3333))
	if err := manager.Enable(context.Background(), spec); err == nil {
		t.Fatal("Enable debía propagar el fallo de systemctl")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "previous" {
		t.Fatalf("definition restored = %q, %v", data, err)
	}
}

func TestSystemdFailurePathsAndConstructors(t *testing.T) {
	if _, err := newSystemdManager("relative", persist.NewStore(), &fakeRunner{}); err == nil {
		t.Fatal("constructor aceptó ruta relativa")
	}
	if _, err := newSystemdManager(filepath.Join(t.TempDir(), "unit"), nil, &fakeRunner{}); err == nil {
		t.Fatal("constructor aceptó store nil")
	}
	if _, err := newSystemdManager(filepath.Join(t.TempDir(), "unit"), persist.NewStore(), nil); err == nil {
		t.Fatal("constructor aceptó runner nil")
	}

	path := filepath.Join(t.TempDir(), "systemd", "user", systemdUnitName)
	runner := &fakeRunner{fail: map[string]error{"--user daemon-reload": errors.New("reload")}}
	manager, err := newSystemdManager(path, persist.NewStore(), runner)
	if err != nil {
		t.Fatal(err)
	}
	spec, _ := NewSpec("/tmp/gateway", endpoint.MustPort(3333))
	if err := manager.Enable(context.Background(), spec); err == nil {
		t.Fatal("Enable debía fallar en daemon-reload")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("definition nueva no fue retirada: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("unit"), 0o600); err != nil {
		t.Fatal(err)
	}
	runner.fail = map[string]error{"--user restart mcp-gateway": errors.New("restart")}
	if err := manager.Restart(context.Background()); err == nil {
		t.Fatal("Restart debía fallar")
	}
	runner.fail = map[string]error{"--user disable --now mcp-gateway": errors.New("disable")}
	if err := manager.Disable(context.Background()); err == nil {
		t.Fatal("Disable debía fallar")
	}
}

func TestDefaultManagerAndDirectRunner(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	manager, err := newDefaultSystemdManager(&fakeRunner{})
	if err != nil || !filepath.IsAbs(manager.path) {
		t.Fatalf("default manager = %#v, %v", manager, err)
	}
	if manager := NewDefaultManager(); manager == nil {
		t.Fatal("NewDefaultManager devolvió nil")
	}

	printf, err := exec.LookPath("printf")
	if err != nil {
		t.Skip("printf no disponible")
	}
	result, err := (directRunner{}).Run(context.Background(), printf, []string{"abcdef"}, 3)
	if err != nil || string(result.Output) != "abc" || !result.Truncated || result.ExitCode != 0 {
		t.Fatalf("direct runner = %#v, %v", result, err)
	}
	if _, err := (directRunner{}).Run(context.Background(), filepath.Join(t.TempDir(), "missing"), nil, 8); err == nil {
		t.Fatal("direct runner aceptó binario ausente")
	}
}
