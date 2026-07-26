package config

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"

	"mcp-gateway/internal/diagnostics"
	"mcp-gateway/internal/persist"

	"gopkg.in/yaml.v3"
)

type store interface {
	EnsurePrivateDirectory(string) error
	WithLock(context.Context, string, func() error) error
	Replace(context.Context, string, persist.ModePolicy, func(io.Writer) error) error
}

type Repository struct {
	path  string
	store store
}

type Change struct {
	Changed  bool
	Snapshot Snapshot
}

func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".mcp-gateway", "mcp-downstreams.yaml"), nil
}

func NewRepository(path string, storage store) (*Repository, error) {
	if storage == nil {
		return nil, fmt.Errorf("persist store es obligatorio")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	return &Repository{path: filepath.Clean(absolute), store: storage}, nil
}

func NewDefaultRepository() (*Repository, error) {
	path, err := DefaultPath()
	if err != nil {
		return nil, err
	}
	return NewRepository(path, persist.NewStore())
}

func (r *Repository) Path() string {
	return r.path
}

func (r *Repository) Load(ctx context.Context) (Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}
	data, err := os.ReadFile(r.path)
	if err != nil {
		return Snapshot{}, diagnostics.NewFault(diagnostics.Configuration, "no se pudo leer la configuración", err)
	}
	snapshot, err := Decode(data)
	if err != nil {
		return Snapshot{}, diagnostics.NewFault(diagnostics.Configuration, "la configuración no es válida", err)
	}
	return snapshot, nil
}

func (r *Repository) Update(ctx context.Context, callback func(*Document) error) (Change, error) {
	if callback == nil {
		return Change{}, diagnostics.NewFault(diagnostics.Validation, "la actualización de configuración es inválida", nil)
	}
	if err := r.store.EnsurePrivateDirectory(filepath.Dir(r.path)); err != nil {
		return Change{}, diagnostics.NewFault(diagnostics.Persistence, "no se pudo preparar el directorio de configuración", err)
	}
	var change Change
	err := r.store.WithLock(ctx, r.path+".lock", func() error {
		document := NewDocument()
		data, readErr := os.ReadFile(r.path)
		if readErr == nil {
			snapshot, decodeErr := Decode(data)
			if decodeErr != nil {
				return diagnostics.NewFault(diagnostics.Configuration, "la configuración existente no es válida", decodeErr)
			}
			document = snapshot.Document()
		} else if !os.IsNotExist(readErr) {
			return diagnostics.NewFault(diagnostics.Configuration, "no se pudo leer la configuración existente", readErr)
		}
		before := document.Clone()
		working := document.Clone()
		if err := callback(&working); err != nil {
			return err
		}
		if err := Validate(&working); err != nil {
			return diagnostics.NewFault(diagnostics.Validation, "la configuración actualizada no es válida", err)
		}
		change.Snapshot = newSnapshot(working)
		if readErr == nil && reflect.DeepEqual(before, working) {
			return nil
		}
		encoded, err := marshalDocument(working)
		if err != nil {
			return diagnostics.NewFault(diagnostics.Configuration, "no se pudo serializar la configuración", err)
		}
		if err := r.store.Replace(ctx, r.path, persist.SecretFile, func(writer io.Writer) error {
			_, err := io.Copy(writer, bytes.NewReader(encoded))
			return err
		}); err != nil {
			return diagnostics.NewFault(diagnostics.Persistence, "no se pudo persistir la configuración", err)
		}
		change.Changed = true
		return nil
	})
	if err != nil {
		return Change{}, err
	}
	return change, nil
}

func marshalDocument(document Document) ([]byte, error) {
	return yaml.Marshal(document)
}
