package daemon

import (
	"context"
	"fmt"
)

type unavailableManager struct{ err error }

func (m unavailableManager) Status(context.Context) (Status, error) { return Status{}, m.err }
func (m unavailableManager) Enable(context.Context, Spec) error {
	return fmt.Errorf("gestor de daemon no disponible: %w", m.err)
}
func (m unavailableManager) Disable(context.Context) error {
	return fmt.Errorf("gestor de daemon no disponible: %w", m.err)
}
func (m unavailableManager) Restart(context.Context) error {
	return fmt.Errorf("gestor de daemon no disponible: %w", m.err)
}
