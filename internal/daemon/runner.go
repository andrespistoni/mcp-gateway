package daemon

import (
	"bytes"
	"context"
	"io"
	"os/exec"
	"sync"
)

const (
	runnerTimeout = 5_000_000_000 // five seconds, without a duration literal in callers
	maxOutput     = 64 * 1024
)

type CommandResult struct {
	ExitCode  int
	Output    []byte
	Truncated bool
}

// Runner executes a native manager tool as argv. Implementations must not
// invoke a shell or inherit stdin.
type Runner interface {
	Run(context.Context, string, []string, int) (CommandResult, error)
}

type directRunner struct{}

func (directRunner) Run(ctx context.Context, binary string, args []string, limit int) (CommandResult, error) {
	command := exec.CommandContext(ctx, binary, args...)
	command.Stdin = nil
	stdout, err := command.StdoutPipe()
	if err != nil {
		return CommandResult{}, err
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		return CommandResult{}, err
	}
	if err := command.Start(); err != nil {
		return CommandResult{}, err
	}
	capture := &outputCapture{remaining: limit}
	var drains sync.WaitGroup
	drains.Add(2)
	go func() { defer drains.Done(); _, _ = io.Copy(capture, stdout) }()
	go func() { defer drains.Done(); _, _ = io.Copy(capture, stderr) }()
	drains.Wait()
	waitErr := command.Wait()
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
	buffer    bytes.Buffer
}

func (c *outputCapture) Write(data []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	accepted := len(data)
	if accepted > c.remaining {
		accepted = c.remaining
		c.truncated = true
	}
	if accepted > 0 {
		_, _ = c.buffer.Write(data[:accepted])
		c.remaining -= accepted
	}
	return len(data), nil
}

func (c *outputCapture) result() CommandResult {
	c.mu.Lock()
	defer c.mu.Unlock()
	return CommandResult{Output: append([]byte(nil), c.buffer.Bytes()...), Truncated: c.truncated}
}

func runCommand(ctx context.Context, runner Runner, binary string, args ...string) (CommandResult, error) {
	commandCtx, cancel := context.WithTimeout(ctx, runnerTimeout)
	defer cancel()
	return runner.Run(commandCtx, binary, append([]string(nil), args...), maxOutput)
}
