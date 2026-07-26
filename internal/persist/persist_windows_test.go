//go:build windows

package persist

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestWindowsReplaceAndRestrictedDACL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	store := NewStore()
	for _, value := range []string{"nuevo", "reemplazado"} {
		if err := store.Replace(context.Background(), path, SecretFile, func(writer io.Writer) error {
			_, err := io.WriteString(writer, value)
			return err
		}); err != nil {
			t.Fatal(err)
		}
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "reemplazado" {
		t.Fatalf("contenido = %q, %v", data, err)
	}
	descriptor, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	dacl, _, err := descriptor.DACL()
	if err != nil || dacl == nil || dacl.AceCount < 1 {
		t.Fatalf("DACL inválida: %#v, %v", dacl, err)
	}
}

func TestWindowsLockCancellationAndFailedReplace(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "config.yaml")
	lock := filepath.Join(directory, "config.lock")
	if err := os.WriteFile(path, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := NewStore()
	entered := make(chan struct{})
	release := make(chan struct{})
	go func() {
		_ = store.WithLock(context.Background(), lock, func() error {
			close(entered)
			<-release
			return nil
		})
	}()
	<-entered
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if err := store.WithLock(ctx, lock, func() error { return nil }); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("cancelación = %v", err)
	}
	close(release)
	want := errors.New("fallo")
	if err := store.Replace(context.Background(), path, SecretFile, func(io.Writer) error { return want }); !errors.Is(err, want) {
		t.Fatalf("fallo = %v", err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "original" {
		t.Fatalf("original alterado: %q", data)
	}
}
