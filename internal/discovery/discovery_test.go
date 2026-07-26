package discovery

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"mcp-gateway/internal/proc"
)

type recordingProber struct {
	paths []string
	fail  map[string]error
}

func (p *recordingProber) Probe(_ context.Context, spec proc.ExecSpec) error {
	path := spec.Executable().Path()
	p.paths = append(p.paths, path)
	return p.fail[path]
}

func TestRecipesAreExact(t *testing.T) {
	recipes := Recipes()
	if len(recipes) != 3 || recipes[0].Name != "codegraph" || recipes[0].Prefix != "codegraph__" ||
		!reflect.DeepEqual(recipes[0].Args, []string{"serve", "--mcp"}) || recipes[1].Prefix != "cbm__" ||
		!reflect.DeepEqual(recipes[2].Args, []string{"mcp", "--tools=agent"}) {
		t.Fatalf("recetas inesperadas: %#v", recipes)
	}
}

func TestDiscoveryOrdersExplicitBeforePATHAndContinuesAfterFailure(t *testing.T) {
	home := t.TempDir()
	pathDir := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", pathDir)
	explicit := filepath.Join(home, ".local", "bin", "codegraph")
	if err := os.MkdirAll(filepath.Dir(explicit), 0o700); err != nil {
		t.Fatal(err)
	}
	writeExecutable(t, explicit)
	pathCandidate := filepath.Join(pathDir, "codegraph")
	writeExecutable(t, pathCandidate)
	prober := &recordingProber{fail: map[string]error{explicit: errors.New("fallo")}}
	service := New(prober)
	service.recipes = service.recipes[:1]
	result, err := service.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Downstreams) != 1 || result.Downstreams[0].Binary != pathCandidate {
		t.Fatalf("resultado = %#v", result.Downstreams)
	}
	if !reflect.DeepEqual(prober.paths, []string{explicit, pathCandidate}) {
		t.Fatalf("orden = %#v", prober.paths)
	}
}

func TestDiscoveryNoInstallationsIsSuccess(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("PATH", t.TempDir())
	result, err := New(&recordingProber{}).Discover(context.Background())
	if err != nil || len(result.Downstreams) != 0 || len(result.Attempts) != 6 {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func writeExecutable(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("fake"), 0o700); err != nil {
		t.Fatal(err)
	}
}
