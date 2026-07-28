package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"mcp-gateway/internal/daemon"
	"mcp-gateway/internal/diagnostics"
	"mcp-gateway/internal/endpoint"
)

type EnableDaemonRequest struct {
	Port *endpoint.Port
}

type DaemonStatus struct {
	Installed bool
	Running   bool
}

func (r *Runtime) EnableDaemon(ctx context.Context, request EnableDaemonRequest) error {
	manager, err := r.requireDaemon()
	if err != nil {
		return err
	}
	port, err := r.resolvePort(ctx, request.Port)
	if err != nil {
		return err
	}
	binary, err := os.Executable()
	if err != nil {
		return diagnostics.NewFault(diagnostics.Process, "no se pudo localizar el binario del gateway", err)
	}
	binary, err = filepath.Abs(binary)
	if err != nil {
		return diagnostics.NewFault(diagnostics.Process, "no se pudo normalizar el binario del gateway", err)
	}
	spec, err := daemon.NewSpec(binary, port)
	if err != nil {
		return diagnostics.NewFault(diagnostics.Validation, "la definición del daemon no es válida", err)
	}
	if err := manager.Enable(ctx, spec); err != nil {
		return diagnostics.NewFault(diagnostics.Process, "no se pudo habilitar el daemon", err)
	}
	return nil
}

func (r *Runtime) DisableDaemon(ctx context.Context) error {
	manager, err := r.requireDaemon()
	if err != nil {
		return err
	}
	if err := manager.Disable(ctx); err != nil {
		return diagnostics.NewFault(diagnostics.Process, "no se pudo deshabilitar el daemon", err)
	}
	return nil
}

func (r *Runtime) Restart(ctx context.Context) error {
	manager, err := r.requireDaemon()
	if err != nil {
		return err
	}
	if err := manager.Restart(ctx); err != nil {
		return diagnostics.NewFault(diagnostics.Process, "no se pudo reiniciar el daemon", err)
	}
	return nil
}

func (r *Runtime) DaemonStatus(ctx context.Context) (DaemonStatus, error) {
	manager, err := r.requireDaemon()
	if err != nil {
		return DaemonStatus{}, err
	}
	status, err := manager.Status(ctx)
	if err != nil {
		return DaemonStatus{}, diagnostics.NewFault(diagnostics.Process, "no se pudo consultar el daemon", err)
	}
	return DaemonStatus{Installed: status.Installed, Running: status.Running}, nil
}

func (r *Runtime) requireDaemon() (daemon.Manager, error) {
	if r.daemonManager == nil {
		return nil, diagnostics.NewFault(diagnostics.Process, "el gestor de daemon no está disponible", fmt.Errorf("manager ausente"))
	}
	return r.daemonManager, nil
}
