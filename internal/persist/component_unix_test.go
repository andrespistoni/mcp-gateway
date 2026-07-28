//go:build !windows

package persist

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestEnsurePrivateDirectoryAndPreserveMode(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "private")
	store := NewStore()
	if err := store.EnsurePrivateDirectory(directory); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(directory)
	if err != nil || info.Mode().Perm() != 0o700 {
		t.Fatalf("directory mode = %v, %v", info.Mode().Perm(), err)
	}

	path := filepath.Join(directory, "config")
	if err := os.WriteFile(path, []byte("old"), 0o640); err != nil {
		t.Fatal(err)
	}
	policy := ModePolicy{Mode: 0o600, PreserveExisting: true}
	if err := store.Replace(context.Background(), path, policy, func(writer io.Writer) error {
		_, err := io.WriteString(writer, "new")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	info, err = os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o640 {
		t.Fatalf("preserved mode = %v, %v", info.Mode().Perm(), err)
	}
}

func TestStoreCancellationAndInvalidDestinations(t *testing.T) {
	store := NewStore()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := store.WithLock(ctx, filepath.Join(t.TempDir(), "lock"), func() error { return nil }); !errors.Is(err, context.Canceled) {
		t.Fatalf("WithLock cancellation = %v", err)
	}
	if err := store.Replace(ctx, filepath.Join(t.TempDir(), "file"), ModePolicy{}, func(io.Writer) error { return nil }); !errors.Is(err, context.Canceled) {
		t.Fatalf("Replace cancellation = %v", err)
	}

	root := t.TempDir()
	missingParent := filepath.Join(root, "missing", "file")
	if err := store.Replace(context.Background(), missingParent, ModePolicy{}, func(io.Writer) error { return nil }); err == nil {
		t.Fatal("Replace debía fallar sin directorio padre")
	}
	directoryTarget := filepath.Join(root, "target")
	if err := os.Mkdir(directoryTarget, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := store.Replace(context.Background(), directoryTarget, ModePolicy{PreserveExisting: true}, func(io.Writer) error { return nil }); err == nil {
		t.Fatal("Replace debía rechazar un directorio destino")
	}
}
