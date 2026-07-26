package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"mcp-gateway/internal/config"
	"mcp-gateway/internal/discovery"
	"mcp-gateway/internal/persist"
	"mcp-gateway/internal/proc"
	"mcp-gateway/internal/proxy"
)

type staticDiscovery struct{ result discovery.Result }

func (s staticDiscovery) Discover(context.Context) (discovery.Result, error) { return s.result, nil }

func TestDiscoverReadOnlyAndWriteMergeWithoutOverwrite(t *testing.T) {
	repository := testRepository(t)
	_, err := repository.Update(context.Background(), func(document *config.Document) error {
		document.Downstreams = []config.Downstream{{
			Name: "codegraph", Prefix: "custom__", Binary: "user-binary", Args: []string{}, Enabled: true, Env: map[string]string{},
		}}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	candidates := []config.Downstream{
		{Name: "codegraph", Prefix: "codegraph__", Binary: "/fake/codegraph", Args: []string{}, Enabled: true, Env: map[string]string{}},
		{Name: "engram", Prefix: "engram__", Binary: "/fake/engram", Args: []string{}, Enabled: true, Env: map[string]string{}},
	}
	runtimeApp := NewRuntime(repository)
	runtimeApp.discovery = staticDiscovery{result: discovery.Result{Downstreams: candidates}}
	if _, err := runtimeApp.Discover(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(repository.Path())
	result, err := runtimeApp.Discover(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(repository.Path())
	if string(before) == string(after) || result.Items[0].Added || !result.Items[1].Added {
		t.Fatalf("merge = %#v", result.Items)
	}
	snapshot, err := repository.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	downstreams := snapshot.Downstreams()
	if len(downstreams) != 2 || downstreams[0].Binary != "user-binary" {
		t.Fatalf("se sobrescribió configuración: %#v", downstreams)
	}
}

func TestAddSkipValidationPreservesEnabledDefaultAndDisabledFlag(t *testing.T) {
	repository := testRepository(t)
	runtimeApp := NewRuntime(repository)
	for _, request := range []AddRequest{
		{Name: "enabled", Prefix: "enabled__", Binary: "missing-one", Environment: map[string]string{}, SkipValidation: true},
		{Name: "disabled", Prefix: "disabled__", Binary: "missing-two", Environment: map[string]string{}, SkipValidation: true, Disabled: true},
	} {
		if err := runtimeApp.Add(context.Background(), request); err != nil {
			t.Fatal(err)
		}
	}
	snapshot, err := repository.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	downstreams := snapshot.Downstreams()
	if len(downstreams) != 2 || !downstreams[0].Enabled || downstreams[1].Enabled {
		t.Fatalf("enabled = %#v", downstreams)
	}
}

func TestAddValidatesBeforePersisting(t *testing.T) {
	repository := testRepository(t)
	name := "candidate"
	if runtime.GOOS == "windows" {
		name += ".exe"
		t.Setenv("PATHEXT", ".EXE")
	}
	binary := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(binary, []byte("fake"), 0o700); err != nil {
		t.Fatal(err)
	}
	runtimeApp := NewRuntime(repository)
	called := false
	runtimeApp.probe = func(context.Context, proc.ExecSpec) (proxy.ProbeResult, error) {
		called = true
		return proxy.ProbeResult{}, errors.New("handshake inválido")
	}
	request := AddRequest{Name: "candidate", Prefix: "candidate__", Binary: binary, Environment: map[string]string{}}
	if err := runtimeApp.Add(context.Background(), request); err == nil || !called {
		t.Fatalf("Add = called:%v err:%v", called, err)
	}
	if _, err := os.Stat(repository.Path()); !os.IsNotExist(err) {
		t.Fatalf("se persistió antes de validar: %v", err)
	}
	runtimeApp.probe = func(context.Context, proc.ExecSpec) (proxy.ProbeResult, error) {
		return proxy.ProbeResult{}, nil
	}
	if err := runtimeApp.Add(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func testRepository(t *testing.T) *config.Repository {
	t.Helper()
	path := filepath.Join(t.TempDir(), "mcp-downstreams.yaml")
	repository, err := config.NewRepository(path, persist.NewStore())
	if err != nil {
		t.Fatal(err)
	}
	return repository
}
