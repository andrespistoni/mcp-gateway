//go:build !windows

package persist

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestReplaceCreatesAndReplacesWithSecretMode(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "config.yaml")
	store := NewStore()
	write := func(value string) error {
		return store.Replace(context.Background(), path, SecretFile, func(writer io.Writer) error {
			_, err := io.WriteString(writer, value)
			return err
		})
	}
	if err := write("primero"); err != nil {
		t.Fatal(err)
	}
	if err := write("segundo"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "segundo" {
		t.Fatalf("contenido = %q", data)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("modo = %o", got)
	}
}

func TestReplaceFailureKeepsOriginalAndCleansTemporary(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "config.yaml")
	if err := os.WriteFile(path, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	want := errors.New("fallo inyectado")
	err := NewStore().Replace(context.Background(), path, SecretFile, func(writer io.Writer) error {
		_, _ = io.WriteString(writer, "parcial")
		return want
	})
	if !errors.Is(err, want) {
		t.Fatalf("error = %v", err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "original" {
		t.Fatalf("original alterado: %q", data)
	}
	matches, err := filepath.Glob(filepath.Join(directory, ".mcp-gateway-*"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("temporales restantes: %v, %v", matches, err)
	}
}

func TestWithLockSerializesAndHonorsCancellation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.lock")
	store := NewStore()
	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- store.WithLock(context.Background(), path, func() error {
			close(entered)
			<-release
			return nil
		})
	}()
	<-entered
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	if err := store.WithLock(ctx, path, func() error { return nil }); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("lock cancelado = %v", err)
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}

	var order []int
	var mu sync.Mutex
	firstEntered := make(chan struct{})
	firstRelease := make(chan struct{})
	var group sync.WaitGroup
	group.Add(2)
	go func() {
		defer group.Done()
		_ = store.WithLock(context.Background(), path, func() error {
			mu.Lock()
			order = append(order, 1)
			mu.Unlock()
			close(firstEntered)
			<-firstRelease
			return nil
		})
	}()
	<-firstEntered
	go func() {
		defer group.Done()
		_ = store.WithLock(context.Background(), path, func() error {
			mu.Lock()
			order = append(order, 2)
			mu.Unlock()
			return nil
		})
	}()
	time.Sleep(20 * time.Millisecond)
	mu.Lock()
	if len(order) != 1 {
		t.Fatalf("el segundo lock entró antes de tiempo: %v", order)
	}
	mu.Unlock()
	close(firstRelease)
	group.Wait()
	if len(order) != 2 || order[0] != 1 || order[1] != 2 {
		t.Fatalf("orden = %v", order)
	}
}
