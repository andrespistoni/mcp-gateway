package app

import (
	"context"

	"mcp-gateway/internal/config"
	"mcp-gateway/internal/diagnostics"
)

func (r *Runtime) Remove(ctx context.Context, name string) (bool, error) {
	repository, ok := r.config.(configUpdater)
	if !ok {
		return false, diagnostics.NewFault(diagnostics.Configuration, "el repositorio no admite mutaciones", nil)
	}
	change, err := repository.Update(ctx, func(document *config.Document) error {
		for i, downstream := range document.Downstreams {
			if downstream.Name == name {
				document.Downstreams = append(document.Downstreams[:i], document.Downstreams[i+1:]...)
				return nil
			}
		}
		return diagnostics.NewFault(diagnostics.Validation, "el downstream no existe", nil)
	})
	if err == nil && change.Changed && r.mutationCommitted != nil {
		err = r.mutationCommitted(ctx)
	}
	return change.Changed, err
}

func (r *Runtime) SetEnabled(ctx context.Context, name string, enabled bool) (bool, error) {
	repository, ok := r.config.(configUpdater)
	if !ok {
		return false, diagnostics.NewFault(diagnostics.Configuration, "el repositorio no admite mutaciones", nil)
	}
	change, err := repository.Update(ctx, func(document *config.Document) error {
		for i := range document.Downstreams {
			if document.Downstreams[i].Name == name {
				document.Downstreams[i].Enabled = enabled
				return nil
			}
		}
		return diagnostics.NewFault(diagnostics.Validation, "el downstream no existe", nil)
	})
	if err == nil && change.Changed && r.mutationCommitted != nil {
		err = r.mutationCommitted(ctx)
	}
	return change.Changed, err
}
