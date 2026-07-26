package claude

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"mcp-gateway/internal/diagnostics"
	"mcp-gateway/internal/endpoint"
	"mcp-gateway/internal/proc"
)

const (
	claudeTimeout = 5 * time.Second
	maxCLIOutput  = 64 * 1024
)

type CommandResult struct {
	Output    []byte
	ExitCode  int
	Truncated bool
}

type Runner interface {
	Run(context.Context, proc.ResolvedExecutable, []string, int) (CommandResult, error)
}

type CLIRegistrar struct {
	runner Runner
}

type InstallResult struct {
	Installed bool
}

func NewCLIRegistrar(runner Runner) (*CLIRegistrar, error) {
	if runner == nil {
		return nil, fmt.Errorf("runner de Claude es obligatorio")
	}
	return &CLIRegistrar{runner: runner}, nil
}

func NewDefaultCLIRegistrar() *CLIRegistrar {
	registrar, _ := NewCLIRegistrar(directRunner{})
	return registrar
}

func (r *CLIRegistrar) Install(ctx context.Context, port endpoint.Port) (InstallResult, error) {
	executable, err := proc.ResolveExecutable("claude")
	if err != nil {
		return InstallResult{}, diagnostics.NewFault(diagnostics.Unavailable, "la CLI de Claude no está disponible", err)
	}
	gatewayURL := endpoint.LocalhostURL(port, "/sse", nil)
	getCtx, cancel := context.WithTimeout(ctx, claudeTimeout)
	get, err := r.runner.Run(getCtx, executable, []string{"mcp", "get", projectServerName}, maxCLIOutput)
	cancel()
	if err != nil {
		return InstallResult{}, commandFault("no se pudo consultar el registro de Claude", err)
	}
	if get.Truncated {
		return InstallResult{}, diagnostics.NewFault(diagnostics.Protocol, "la respuesta de Claude es ambigua o demasiado grande", nil)
	}
	if get.ExitCode == 0 {
		if !matchingClaudeRegistration(get.Output, gatewayURL) {
			return InstallResult{}, diagnostics.NewFault(diagnostics.Conflict, "existe un registro de Claude incompatible o ambiguo", nil)
		}
		return InstallResult{Installed: false}, nil
	}
	if !missingClaudeRegistration(get.Output) {
		return InstallResult{}, diagnostics.NewFault(diagnostics.Process, "no se pudo demostrar que el registro de Claude esté ausente", nil)
	}

	addCtx, cancel := context.WithTimeout(ctx, claudeTimeout)
	add, err := r.runner.Run(addCtx, executable, []string{
		"mcp", "add", "--transport", "sse", "--scope", "user", projectServerName, gatewayURL,
	}, maxCLIOutput)
	cancel()
	if err != nil {
		return InstallResult{}, commandFault("no se pudo registrar el gateway en Claude", err)
	}
	if add.ExitCode != 0 || add.Truncated {
		return InstallResult{}, diagnostics.NewFault(diagnostics.Process, "Claude rechazó el registro del gateway", nil)
	}
	return InstallResult{Installed: true}, nil
}

func commandFault(message string, err error) error {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return diagnostics.NewFault(diagnostics.Timeout, message, err)
	}
	return diagnostics.NewFault(diagnostics.Process, message, err)
}

func missingClaudeRegistration(output []byte) bool {
	line := strings.ToLower(strings.TrimSpace(string(output)))
	return line == "no mcp server found with name: mcp-gateway" ||
		line == `no mcp server found with name "mcp-gateway"`
}

func matchingClaudeRegistration(output []byte, gatewayURL string) bool {
	values := make(map[string]string)
	foundName := false
	for _, rawLine := range strings.Split(string(output), "\n") {
		line := strings.TrimSpace(rawLine)
		if strings.EqualFold(line, projectServerName+":") {
			if foundName {
				return false
			}
			foundName = true
			continue
		}
		if line == "" {
			continue
		}
		key, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		if key != "scope" && key != "type" && key != "url" {
			continue
		}
		if _, duplicate := values[key]; duplicate {
			return false
		}
		values[key] = strings.TrimSpace(value)
	}
	scope := strings.ToLower(values["scope"])
	return foundName && (scope == "user" || scope == "user config") &&
		strings.EqualFold(values["type"], "sse") && values["url"] == gatewayURL
}

type directRunner struct{}

func (directRunner) Run(ctx context.Context, executable proc.ResolvedExecutable, args []string, limit int) (CommandResult, error) {
	if err := ctx.Err(); err != nil {
		return CommandResult{}, err
	}
	spec, err := proc.NewExecSpec(executable, args, nil)
	if err != nil {
		return CommandResult{}, err
	}
	tree, err := proc.Start(spec)
	if err != nil {
		return CommandResult{}, err
	}
	stdin := tree.Stdin()
	stdout := tree.Stdout()
	stderr := tree.Stderr()
	_ = stdin.Close()
	capture := newOutputCapture(limit)
	var drains sync.WaitGroup
	drains.Add(2)
	go func() {
		defer drains.Done()
		defer stdout.Close()
		_, _ = io.Copy(capture.writer(&capture.stdout), stdout)
	}()
	go func() {
		defer drains.Done()
		defer stderr.Close()
		_, _ = io.Copy(capture.writer(&capture.stderr), stderr)
	}()
	waited := make(chan error, 1)
	go func() { waited <- tree.Wait() }()
	var waitErr error
	select {
	case waitErr = <-waited:
	case <-ctx.Done():
		_ = tree.Kill()
		<-waited
		drains.Wait()
		return CommandResult{}, ctx.Err()
	}
	drains.Wait()
	result := capture.result()
	if waitErr != nil {
		result.ExitCode = 1
	}
	return result, nil
}

type outputCapture struct {
	mu        sync.Mutex
	remaining int
	truncated bool
	stdout    bytes.Buffer
	stderr    bytes.Buffer
}

func newOutputCapture(limit int) *outputCapture {
	return &outputCapture{remaining: limit}
}

func (c *outputCapture) writer(destination *bytes.Buffer) io.Writer {
	return writerFunc(func(data []byte) (int, error) {
		c.mu.Lock()
		defer c.mu.Unlock()
		accepted := len(data)
		if accepted > c.remaining {
			accepted = c.remaining
			c.truncated = true
		}
		if accepted > 0 {
			_, _ = destination.Write(data[:accepted])
			c.remaining -= accepted
		}
		return len(data), nil
	})
}

func (c *outputCapture) result() CommandResult {
	c.mu.Lock()
	defer c.mu.Unlock()
	output := append([]byte(nil), c.stdout.Bytes()...)
	if c.stderr.Len() > 0 {
		output = append(output, c.stderr.Bytes()...)
	}
	return CommandResult{Output: output, Truncated: c.truncated}
}

type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(data []byte) (int, error) {
	return f(data)
}
