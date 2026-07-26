package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"mcp-gateway/internal/claude"
	"mcp-gateway/internal/config"
	"mcp-gateway/internal/endpoint"
	"mcp-gateway/internal/project"
)

type fakeProjectRegistrar struct {
	directory project.Dir
	port      endpoint.Port
	result    claude.ProjectRegistration
	err       error
}

func (f *fakeProjectRegistrar) Register(_ context.Context, directory project.Dir, port endpoint.Port) (claude.ProjectRegistration, error) {
	f.directory, f.port = directory, port
	return f.result, f.err
}

type fakeClaudeRegistrar struct {
	port   endpoint.Port
	result claude.InstallResult
	err    error
}

func (f *fakeClaudeRegistrar) Install(_ context.Context, port endpoint.Port) (claude.InstallResult, error) {
	f.port = port
	return f.result, f.err
}

func TestRegisterProjectResuelveDirectorioYPuerto(t *testing.T) {
	repository := testRepository(t)
	_, err := repository.Update(context.Background(), func(document *config.Document) error {
		document.Port = endpoint.MustPort(5555)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	registrar := &fakeProjectRegistrar{result: claude.ProjectRegistration{Created: true, Changed: true}}
	runtimeApp := NewRuntime(repository)
	runtimeApp.projectRegistrar = registrar
	result, err := runtimeApp.RegisterProject(context.Background(), RegisterProjectRequest{ProjectDir: filepath.Join(directory, ".")})
	if err != nil || result.ProjectDir != directory || !result.Created || registrar.port.Number() != 5555 {
		t.Fatalf("RegisterProject = %#v, %v, registrar=%#v", result, err, registrar)
	}
	port := endpoint.MustPort(6666)
	_, err = runtimeApp.RegisterProject(context.Background(), RegisterProjectRequest{ProjectDir: directory, Port: &port})
	if err != nil || registrar.port.Number() != 6666 {
		t.Fatalf("puerto CLI = %d, %v", registrar.port.Number(), err)
	}
}

func TestRegisterProjectUsaDirectorioActual(t *testing.T) {
	repository := testRepository(t)
	directory := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(directory); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
	registrar := &fakeProjectRegistrar{result: claude.ProjectRegistration{Updated: true}}
	runtimeApp := NewRuntime(repository)
	runtimeApp.projectRegistrar = registrar
	if _, err := runtimeApp.RegisterProject(context.Background(), RegisterProjectRequest{}); err != nil {
		t.Fatal(err)
	}
	if registrar.directory.Path() != directory || registrar.port.Number() != 3333 {
		t.Fatalf("directorio=%q puerto=%d", registrar.directory.Path(), registrar.port.Number())
	}
}

func TestInstallClaudeResuelvePuertoYPropagaResultado(t *testing.T) {
	repository := testRepository(t)
	registrar := &fakeClaudeRegistrar{result: claude.InstallResult{Installed: true}}
	runtimeApp := NewRuntime(repository)
	runtimeApp.claudeRegistrar = registrar
	port := endpoint.MustPort(4444)
	result, err := runtimeApp.InstallClaude(context.Background(), InstallClaudeRequest{Port: &port})
	if err != nil || !result.Installed || registrar.port.Number() != 4444 {
		t.Fatalf("InstallClaude = %#v, %v, puerto=%d", result, err, registrar.port.Number())
	}
}
