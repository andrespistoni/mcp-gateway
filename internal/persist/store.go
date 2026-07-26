package persist

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

type ModePolicy struct {
	Mode             fs.FileMode
	PreserveExisting bool
	RestrictToUser   bool
}

var SecretFile = ModePolicy{Mode: 0o600, RestrictToUser: true}

type Store struct{}

func NewStore() *Store {
	return &Store{}
}

func (s *Store) EnsurePrivateDirectory(path string) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(path, 0o700); err != nil {
		return err
	}
	return restrictDirectoryToUser(path)
}

func (s *Store) WithLock(ctx context.Context, lockPath string, fn func() error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	release, err := acquireLock(ctx, lockPath)
	if err != nil {
		return fmt.Errorf("adquirir lock: %w", err)
	}
	defer release()
	return fn()
}

func (s *Store) Replace(ctx context.Context, path string, policy ModePolicy, writeFn func(io.Writer) error) (err error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	directory := filepath.Dir(path)
	mode := policy.Mode
	if mode == 0 {
		mode = 0o600
	}
	if policy.PreserveExisting {
		if info, statErr := os.Stat(path); statErr == nil {
			mode = info.Mode().Perm()
		} else if !os.IsNotExist(statErr) {
			return fmt.Errorf("consultar destino: %w", statErr)
		}
	}

	temporary, err := os.CreateTemp(directory, ".mcp-gateway-*")
	if err != nil {
		return fmt.Errorf("crear temporal: %w", err)
	}
	temporaryPath := temporary.Name()
	committed := false
	defer func() {
		if !committed {
			_ = temporary.Close()
			_ = os.Remove(temporaryPath)
		}
	}()

	if err := temporary.Chmod(mode); err != nil {
		return fmt.Errorf("aplicar permisos al temporal: %w", err)
	}
	if policy.RestrictToUser {
		if err := restrictFileToUser(temporaryPath); err != nil {
			return fmt.Errorf("restringir temporal al usuario: %w", err)
		}
	}
	if err := writeFn(temporary); err != nil {
		return fmt.Errorf("escribir temporal: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("sincronizar temporal: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("cerrar temporal: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := replacePath(temporaryPath, path); err != nil {
		return fmt.Errorf("reemplazar destino: %w", err)
	}
	committed = true
	if err := syncDirectory(directory); err != nil {
		return fmt.Errorf("sincronizar directorio: %w", err)
	}
	return nil
}
