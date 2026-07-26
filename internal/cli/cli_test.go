package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"mcp-gateway/internal/app"
	"mcp-gateway/internal/config"
	"mcp-gateway/internal/diagnostics"
	"mcp-gateway/internal/persist"
)

type fakeApplication struct {
	items []app.ListItem
	err   error
}

func (f fakeApplication) List(context.Context) ([]app.ListItem, error) {
	return f.items, f.err
}

func TestRunCodesAndChannels(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		application Application
		code        int
		stdout      string
		stderr      string
	}{
		{name: "help", args: []string{"help"}, application: fakeApplication{}, code: 0, stdout: "Uso:"},
		{name: "version", args: []string{"version"}, application: fakeApplication{}, code: 0, stdout: "dev"},
		{name: "uso", args: []string{"desconocido"}, application: fakeApplication{}, code: 2, stderr: "comando desconocido"},
		{name: "puerto", args: []string{"serve", "--port", "80"}, application: fakeApplication{}, code: 2, stderr: "--port inválido"},
		{name: "operacional", args: []string{"list"}, application: fakeApplication{err: diagnostics.NewFault(diagnostics.Configuration, "configuración inválida", errors.New("secreto"))}, code: 1, stderr: "configuración inválida"},
		{name: "pendiente", args: []string{"serve", "--port", "4444"}, application: fakeApplication{}, code: 1, stderr: "todavía no implementado"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Run(context.Background(), test.args, Streams{Out: &stdout, Err: &stderr}, test.application)
			if code != test.code || !strings.Contains(stdout.String(), test.stdout) || !strings.Contains(stderr.String(), test.stderr) {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
			if strings.Contains(stderr.String(), "secreto") {
				t.Fatal("se expuso causa cruda")
			}
		})
	}
}

func TestParserAcceptsEveryPublicCommandAndRejectsBadShapes(t *testing.T) {
	valid := [][]string{
		{"setup", "--port", "3333"}, {"discover", "--write"},
		{"add", "custom", "--prefix", "custom__", "--binary", "custom-mcp", "--arg", "a b", "--arg", "c", "--env", "TOKEN=${TOKEN}", "--inject-project", "projectPath", "--disabled", "--skip-validation"},
		{"remove", "custom"}, {"enable", "custom"}, {"disable", "custom"}, {"list"}, {"doctor", "--verbose"},
		{"serve", "--port=4444"}, {"enable-daemon"}, {"disable-daemon"}, {"restart"},
		{"register-project", "--project-dir", "/tmp", "--port", "5555"}, {"install-claude"}, {"version"}, {"help"},
	}
	for _, args := range valid {
		if _, err := parse(args); err != nil {
			t.Errorf("parse(%v) = %v", args, err)
		}
	}
	invalid := [][]string{
		{}, {"list", "extra"}, {"doctor", "--unknown"}, {"add", "name", "--prefix", "name__"},
		{"add", "bad name", "--prefix", "name__", "--binary", "tool"}, {"remove"}, {"restart", "extra"},
		{"serve", "--port="}, {"add", "name", "--prefix", "name__", "--binary", "tool", "--inject-project="},
		{"register-project", "--project-dir="},
	}
	for _, args := range invalid {
		if _, err := parse(args); err == nil {
			t.Errorf("parse(%v) debía fallar", args)
		}
	}
}

func TestListUsesIsolatedHomeAndPathWithoutEnvironmentValues(t *testing.T) {
	home := t.TempDir()
	bin := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", bin)
	name := "available-tool"
	if runtime.GOOS == "windows" {
		name += ".exe"
		t.Setenv("PATHEXT", ".EXE")
	}
	executable := filepath.Join(bin, name)
	if err := os.WriteFile(executable, []byte("fake"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(executable, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, ".mcp-gateway", "mcp-downstreams.yaml")
	repository, _ := config.NewRepository(path, persist.NewStore())
	_, err := repository.Update(context.Background(), func(document *config.Document) error {
		document.Downstreams = []config.Downstream{
			{Name: "available", Prefix: "available__", Binary: name, Args: []string{}, Enabled: true, Env: map[string]string{}},
			{Name: "fallback", Prefix: "fallback__", Binary: filepath.Join(home, "missing", name), Args: []string{}, Enabled: true, Env: map[string]string{}},
			{Name: "unavailable", Prefix: "unavailable__", Binary: filepath.Join(home, "missing", "absent"), Args: []string{}, Enabled: true, Env: map[string]string{}},
			{Name: "missing-env", Prefix: "missing-env__", Binary: name, Args: []string{}, Enabled: true, Env: map[string]string{"API_TOKEN": "${MCP_GATEWAY_NEVER_SET_TOKEN}"}},
			{Name: "disabled", Prefix: "disabled__", Binary: "disabled", Args: []string{}, Enabled: false, Env: map[string]string{}},
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = os.Unsetenv("MCP_GATEWAY_NEVER_SET_TOKEN")
	runtimeApp := app.NewRuntime(repository)
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"list"}, Streams{Out: &stdout, Err: &stderr}, runtimeApp)
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	output := stdout.String()
	for _, expected := range []string{"available", "unavailable", "disabled", "fallback PATH", executable, "entorno no disponible"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("list no contiene %q:\n%s", expected, output)
		}
	}
	if strings.Contains(output, "MCP_GATEWAY_NEVER_SET_TOKEN") || strings.Contains(output, "API_TOKEN") {
		t.Fatalf("list expuso claves de entorno:\n%s", output)
	}
}
