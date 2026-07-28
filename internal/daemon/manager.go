// Package daemon owns user-service definitions and their native lifecycle.
package daemon

import (
	"context"
	"fmt"
	"io"
	"os"

	"mcp-gateway/internal/persist"
)

const Name = "mcp-gateway"

// Status describes only the managed definition; it never inspects arbitrary
// services owned by the user.
type Status struct {
	Installed bool
	Running   bool
}

// Manager is the narrow application-facing lifecycle contract.
type Manager interface {
	Status(context.Context) (Status, error)
	Enable(context.Context, Spec) error
	Disable(context.Context) error
	Restart(context.Context) error
}

type definitionStore interface {
	EnsurePrivateDirectory(string) error
	Replace(context.Context, string, persist.ModePolicy, func(io.Writer) error) error
}

func requireSpec(spec Spec) error {
	if err := spec.Valid(); err != nil {
		return fmt.Errorf("definición del daemon inválida: %w", err)
	}
	return nil
}

func readDefinition(path string) ([]byte, bool, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		return data, true, nil
	}
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	return nil, false, fmt.Errorf("leer definición del daemon: %w", err)
}

func replaceDefinition(ctx context.Context, store definitionStore, path string, data []byte) error {
	return store.Replace(ctx, path, persist.ModePolicy{Mode: 0o600, PreserveExisting: true, RestrictToUser: true}, func(writer io.Writer) error {
		_, err := writer.Write(data)
		return err
	})
}
