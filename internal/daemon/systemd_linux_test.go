//go:build linux

package daemon

import (
	"context"
	"os"
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
	calls []recordedCommand
	fail  map[string]error
}

func (f *fakeRunner) Run(_ context.Context, binary string, args []string, _ int) (CommandResult, error) {
	f.calls = append(f.calls, recordedCommand{binary: binary, args: append([]string(nil), args...)})
	if err := f.fail[strings.Join(args, " ")]; err != nil {
		return CommandResult{}, err
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
