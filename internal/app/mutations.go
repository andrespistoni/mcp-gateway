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
	if err == nil && change.Changed {
		err = r.afterMutation(ctx)
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
	if err == nil && change.Changed {
		err = r.afterMutation(ctx)
	}
	return change.Changed, err
}

// afterMutation runs only after the configuration transaction has committed.
// A lifecycle failure is intentionally returned without attempting to undo the
// confirmed configuration change.
func (r *Runtime) afterMutation(ctx context.Context) error {
	if r.mutationCommitted != nil {
		return r.mutationCommitted(ctx)
	}
	if r.daemonManager == nil {
		return nil
	}
	status, err := r.daemonManager.Status(ctx)
	if err != nil {
		return diagnostics.NewFault(diagnostics.Process, "no se pudo consultar el daemon tras la mutación", err)
	}
	if !status.Installed || !status.Running {
		return nil
	}
	if err := r.daemonManager.Restart(ctx); err != nil {
		return diagnostics.NewFault(diagnostics.Process, "la mutación se guardó, pero no se pudo reiniciar el daemon", err)
	}
	return nil
}
