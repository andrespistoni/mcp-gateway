package config

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"testing"

	"mcp-gateway/internal/persist"
)

type failingStore struct {
	base *persist.Store
	err  error
}

func (s failingStore) EnsurePrivateDirectory(path string) error {
	return s.base.EnsurePrivateDirectory(path)
}

func (s failingStore) WithLock(ctx context.Context, path string, fn func() error) error {
	return s.base.WithLock(ctx, path, fn)
}

func (s failingStore) Replace(context.Context, string, persist.ModePolicy, func(io.Writer) error) error {
	return s.err
}

func TestDefaultPathUsesCanonicalHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", home)
	}
	path, err := DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".mcp-gateway", "mcp-downstreams.yaml")
	if path != want {
		t.Fatalf("DefaultPath = %q, want %q", path, want)
	}
}

func TestRepositoryUpdateLoadAndNoop(t *testing.T) {
	path := filepath.Join(t.TempDir(), "home", ".mcp-gateway", "mcp-downstreams.yaml")
	repository, err := NewRepository(path, persist.NewStore())
	if err != nil {
		t.Fatal(err)
	}
	change, err := repository.Update(context.Background(), func(document *Document) error {
		document.Downstreams = append(document.Downstreams, Downstream{
			Name: "one", Prefix: "one__", Binary: "one", Args: []string{}, Enabled: true, Env: map[string]string{},
		})
		return nil
	})
	if err != nil || !change.Changed {
		t.Fatalf("Update = %#v, %v", change, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("modo config = %o", info.Mode().Perm())
	}
	snapshot, err := repository.Load(context.Background())
	if err != nil || snapshot.Downstreams()[0].Name != "one" {
		t.Fatalf("Load = %#v, %v", snapshot, err)
	}
	change, err = repository.Update(context.Background(), func(*Document) error { return nil })
	if err != nil || change.Changed {
		t.Fatalf("noop = %#v, %v", change, err)
	}
}

func TestRepositoryFailureKeepsOriginal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mcp-downstreams.yaml")
	original := []byte("version: 1\ndownstreams: []\n")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	repository, _ := NewRepository(path, persist.NewStore())
	want := errors.New("callback fallida")
	if _, err := repository.Update(context.Background(), func(document *Document) error {
		document.Version = 9
		return want
	}); !errors.Is(err, want) {
		t.Fatalf("error = %v", err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != string(original) {
		t.Fatalf("archivo alterado: %q", data)
	}
	if err := os.WriteFile(path, []byte("version: ["), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.Update(context.Background(), func(*Document) error { return nil }); err == nil {
		t.Fatal("YAML previo inválido debía fallar")
	}
	data, _ = os.ReadFile(path)
	if string(data) != "version: [" {
		t.Fatalf("YAML inválido alterado: %q", data)
	}
}

func TestRepositoryPersistenceFailureKeepsOriginal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mcp-downstreams.yaml")
	original := []byte("version: 1\ndownstreams: []\n")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	want := errors.New("replace fallido")
	repository, err := NewRepository(path, failingStore{base: persist.NewStore(), err: want})
	if err != nil {
		t.Fatal(err)
	}
	_, err = repository.Update(context.Background(), func(document *Document) error {
		document.Downstreams = append(document.Downstreams, Downstream{
			Name: "one", Prefix: "one__", Binary: "one", Args: []string{}, Enabled: true, Env: map[string]string{},
		})
		return nil
	})
	if !errors.Is(err, want) {
		t.Fatalf("error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != string(original) {
		t.Fatalf("original = %q, %v", data, err)
	}
}

func TestRepositorySerializesConcurrentUpdatesWithoutLoss(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mcp-downstreams.yaml")
	repository, _ := NewRepository(path, persist.NewStore())
	if _, err := repository.Update(context.Background(), func(*Document) error { return nil }); err != nil {
		t.Fatal(err)
	}
	const writers = 8
	var group sync.WaitGroup
	errorsFound := make(chan error, writers)
	for i := 0; i < writers; i++ {
		i := i
		group.Add(1)
		go func() {
			defer group.Done()
			_, err := repository.Update(context.Background(), func(document *Document) error {
				name := "item" + strconv.Itoa(i)
				document.Downstreams = append(document.Downstreams, Downstream{
					Name: name, Prefix: name + "__", Binary: name, Args: []string{}, Enabled: true, Env: map[string]string{},
				})
				return nil
			})
			errorsFound <- err
		}()
	}
	group.Wait()
	close(errorsFound)
	for err := range errorsFound {
		if err != nil {
			t.Fatal(err)
		}
	}
	snapshot, err := repository.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := len(snapshot.Downstreams()); got != writers {
		t.Fatalf("downstreams = %d, se esperaban %d", got, writers)
	}
}
