//go:build !windows

package proc

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
)

type Tree struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
	pgid   int
	done   chan struct{}
	mu     sync.Mutex
	err    error
}

func Start(spec ExecSpec) (*Tree, error) {
	path := spec.Executable().Path()
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("no se pudo revalidar el ejecutable: %w", err)
	}
	if err := validateExecutable(path, info); err != nil {
		return nil, fmt.Errorf("el ejecutable dejó de ser válido: %w", err)
	}
	cmd := exec.Command(path, spec.Args()...)
	cmd.Env = spec.Environment()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	childIn, parentIn, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	parentOut, childOut, err := os.Pipe()
	if err != nil {
		closeFiles(childIn, parentIn)
		return nil, err
	}
	parentErr, childErr, err := os.Pipe()
	if err != nil {
		closeFiles(childIn, parentIn, parentOut, childOut)
		return nil, err
	}
	cmd.Stdin = childIn
	cmd.Stdout = childOut
	cmd.Stderr = childErr
	if err := cmd.Start(); err != nil {
		closeFiles(childIn, parentIn, parentOut, childOut, parentErr, childErr)
		return nil, err
	}
	closeFiles(childIn, childOut, childErr)
	tree := &Tree{cmd: cmd, stdin: parentIn, stdout: parentOut, stderr: parentErr, pgid: cmd.Process.Pid, done: make(chan struct{})}
	go tree.wait()
	return tree, nil
}

func closeFiles(files ...*os.File) {
	for _, file := range files {
		if file != nil {
			_ = file.Close()
		}
	}
}

func (t *Tree) Stdin() io.WriteCloser { return t.stdin }
func (t *Tree) Stdout() io.ReadCloser { return t.stdout }
func (t *Tree) Stderr() io.ReadCloser { return t.stderr }

func (t *Tree) wait() {
	err := t.cmd.Wait()
	// El líder puede salir antes que sus descendientes; el grupo sigue siendo
	// propiedad de este handle y se fuerza su cierre antes de publicar Wait.
	_ = syscall.Kill(-t.pgid, syscall.SIGKILL)
	t.mu.Lock()
	t.err = err
	t.mu.Unlock()
	close(t.done)
}

func (t *Tree) Wait() error {
	<-t.done
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.err
}

func (t *Tree) Terminate() error {
	return t.signal(syscall.SIGTERM)
}

func (t *Tree) Kill() error {
	return t.signal(syscall.SIGKILL)
}

func (t *Tree) signal(signal syscall.Signal) error {
	select {
	case <-t.done:
		return nil
	default:
	}
	if err := syscall.Kill(-t.pgid, signal); err != nil && err != syscall.ESRCH {
		return err
	}
	return nil
}
