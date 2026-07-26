package claude

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"mcp-gateway/internal/diagnostics"
	"mcp-gateway/internal/endpoint"
	"mcp-gateway/internal/proc"
)

type runnerCall struct {
	path     string
	args     []string
	limit    int
	deadline time.Duration
}

type fakeRunner struct {
	results []CommandResult
	errors  []error
	calls   []runnerCall
	block   bool
}

func (r *fakeRunner) Run(ctx context.Context, executable proc.ResolvedExecutable, args []string, limit int) (CommandResult, error) {
	deadline, _ := ctx.Deadline()
	r.calls = append(r.calls, runnerCall{
		path: executable.Path(), args: append([]string(nil), args...), limit: limit, deadline: time.Until(deadline),
	})
	if r.block {
		<-ctx.Done()
		return CommandResult{}, ctx.Err()
	}
	index := len(r.calls) - 1
	var result CommandResult
	var err error
	if index < len(r.results) {
		result = r.results[index]
	}
	if index < len(r.errors) {
		err = r.errors[index]
	}
	return result, err
}

func installFakeClaude(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	name := "claude"
	if runtime.GOOS == "windows" {
		name += ".exe"
		t.Setenv("PATHEXT", ".EXE")
	}
	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, []byte("fake que no debe ejecutarse"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", directory)
	return path
}

func TestCLIRegistrarAusenteEjecutaGetYAddSeguro(t *testing.T) {
	executable := installFakeClaude(t)
	runner := &fakeRunner{results: []CommandResult{
		{ExitCode: 1, Output: []byte("No MCP server found with name: mcp-gateway\n")},
		{ExitCode: 0, Output: []byte("Added")},
	}}
	registrar, _ := NewCLIRegistrar(runner)
	result, err := registrar.Install(context.Background(), endpoint.MustPort(3333))
	if err != nil || !result.Installed || len(runner.calls) != 2 {
		t.Fatalf("Install = %#v, %v, calls=%#v", result, err, runner.calls)
	}
	if runner.calls[0].path != executable || strings.Join(runner.calls[0].args, " ") != "mcp get mcp-gateway" {
		t.Fatalf("get = %#v", runner.calls[0])
	}
	wantAdd := "mcp add --transport sse --scope user mcp-gateway http://localhost:3333/sse"
	if strings.Join(runner.calls[1].args, " ") != wantAdd {
		t.Fatalf("add = %q", strings.Join(runner.calls[1].args, " "))
	}
	for _, call := range runner.calls {
		if call.limit != maxCLIOutput || call.deadline <= 0 || call.deadline > claudeTimeout {
			t.Fatalf("límites = %#v", call)
		}
	}
}

func TestCLIRegistrarRegistroIdenticoEsExito(t *testing.T) {
	installFakeClaude(t)
	runner := &fakeRunner{results: []CommandResult{{Output: []byte(
		"mcp-gateway:\n  Scope: User config\n  Status: Connected\n  Type: sse\n  URL: http://localhost:4444/sse\n",
	)}}}
	registrar, _ := NewCLIRegistrar(runner)
	result, err := registrar.Install(context.Background(), endpoint.MustPort(4444))
	if err != nil || result.Installed || len(runner.calls) != 1 {
		t.Fatalf("Install = %#v, %v, calls=%d", result, err, len(runner.calls))
	}
}

func TestCLIRegistrarFallaCerradoSinSobrescribir(t *testing.T) {
	installFakeClaude(t)
	outputs := []CommandResult{
		{Output: []byte("mcp-gateway:\n Scope: User config\n Type: stdio\n URL: http://localhost:3333/sse\n")},
		{ExitCode: 1, Output: []byte("fallo interno")},
		{Output: []byte("Scope: User config\nType: sse\nURL: http://localhost:3333/sse\n"), Truncated: true},
	}
	for _, output := range outputs {
		runner := &fakeRunner{results: []CommandResult{output}}
		registrar, _ := NewCLIRegistrar(runner)
		if _, err := registrar.Install(context.Background(), endpoint.MustPort(3333)); err == nil || len(runner.calls) != 1 {
			t.Fatalf("output=%q err=%v calls=%d", output.Output, err, len(runner.calls))
		}
	}
}

func TestCLIRegistrarClaudeAusenteYTimeout(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	registrar, _ := NewCLIRegistrar(&fakeRunner{})
	if _, err := registrar.Install(context.Background(), endpoint.MustPort(3333)); err == nil || diagnostics.KindOf(err) != diagnostics.Unavailable {
		t.Fatalf("ausente = %v", err)
	}
	installFakeClaude(t)
	runner := &fakeRunner{block: true}
	registrar, _ = NewCLIRegistrar(runner)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	if _, err := registrar.Install(ctx, endpoint.MustPort(3333)); !errors.Is(err, context.DeadlineExceeded) || diagnostics.KindOf(err) != diagnostics.Timeout {
		t.Fatalf("timeout = %v", err)
	}
}

func TestCapturaCLISeLimitaYContinuaDrenando(t *testing.T) {
	capture := newOutputCapture(5)
	stdout := capture.writer(&capture.stdout)
	stderr := capture.writer(&capture.stderr)
	if count, err := stdout.Write([]byte("1234")); err != nil || count != 4 {
		t.Fatalf("stdout = %d, %v", count, err)
	}
	if count, err := stderr.Write([]byte("567890")); err != nil || count != 6 {
		t.Fatalf("stderr = %d, %v", count, err)
	}
	result := capture.result()
	if len(result.Output) != 5 || string(result.Output) != "12345" || !result.Truncated {
		t.Fatalf("captura = %#v", result)
	}
}
