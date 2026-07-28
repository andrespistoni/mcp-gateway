//go:build linux

package daemon

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"mcp-gateway/internal/persist"
)

const systemdUnitName = "mcp-gateway.service"

type systemdManager struct {
	path   string
	store  definitionStore
	runner Runner
}

func newDefaultSystemdManager(runner Runner) (*systemdManager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		base = filepath.Join(home, ".config")
	}
	return newSystemdManager(filepath.Join(base, "systemd", "user", systemdUnitName), persist.NewStore(), runner)
}

func newSystemdManager(path string, store definitionStore, runner Runner) (*systemdManager, error) {
	if path == "" || !filepath.IsAbs(path) || store == nil || runner == nil {
		return nil, fmt.Errorf("composición systemd incompleta")
	}
	return &systemdManager{path: filepath.Clean(path), store: store, runner: runner}, nil
}

func (m *systemdManager) Status(ctx context.Context) (Status, error) {
	if err := ctx.Err(); err != nil {
		return Status{}, err
	}
	if _, err := os.Stat(m.path); err != nil {
		if os.IsNotExist(err) {
			return Status{}, nil
		}
		return Status{}, fmt.Errorf("consultar unidad systemd: %w", err)
	}
	result, err := runCommand(ctx, m.runner, "systemctl", "--user", "is-active", "--quiet", Name)
	if err != nil {
		return Status{}, fmt.Errorf("consultar servicio systemd: %w", err)
	}
	return Status{Installed: true, Running: result.ExitCode == 0}, nil
}

func (m *systemdManager) Enable(ctx context.Context, spec Spec) error {
	if err := requireSpec(spec); err != nil {
		return err
	}
	previous, existed, err := readDefinition(m.path)
	if err != nil {
		return err
	}
	if existed {
		if _, err := runCommand(ctx, m.runner, "systemctl", "--user", "stop", Name); err != nil {
			return fmt.Errorf("detener unidad anterior: %w", err)
		}
	}
	if err := m.store.EnsurePrivateDirectory(filepath.Dir(m.path)); err != nil {
		return fmt.Errorf("preparar directorio systemd: %w", err)
	}
	if err := replaceDefinition(ctx, m.store, m.path, []byte(systemdUnit(spec))); err != nil {
		return fmt.Errorf("persistir unidad systemd: %w", err)
	}
	if _, err := runCommand(ctx, m.runner, "systemctl", "--user", "daemon-reload"); err != nil {
		return m.restore(ctx, previous, existed, fmt.Errorf("recargar systemd: %w", err))
	}
	if result, err := runCommand(ctx, m.runner, "systemctl", "--user", "enable", "--now", Name); err != nil || result.ExitCode != 0 {
		if err == nil {
			err = fmt.Errorf("systemctl enable devolvió %d", result.ExitCode)
		}
		return m.restore(ctx, previous, existed, fmt.Errorf("habilitar unidad systemd: %w", err))
	}
	return nil
}

func (m *systemdManager) Disable(ctx context.Context) error {
	_, existed, err := readDefinition(m.path)
	if err != nil {
		return err
	}
	if !existed {
		return nil
	}
	if _, err := runCommand(ctx, m.runner, "systemctl", "--user", "disable", "--now", Name); err != nil {
		return fmt.Errorf("deshabilitar unidad systemd: %w", err)
	}
	if err := os.Remove(m.path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("eliminar unidad systemd: %w", err)
	}
	if _, err := runCommand(ctx, m.runner, "systemctl", "--user", "daemon-reload"); err != nil {
		return fmt.Errorf("recargar systemd: %w", err)
	}
	return nil
}

func (m *systemdManager) Restart(ctx context.Context) error {
	if _, exists, err := readDefinition(m.path); err != nil {
		return err
	} else if !exists {
		return fmt.Errorf("el daemon no está instalado")
	}
	if _, err := runCommand(ctx, m.runner, "systemctl", "--user", "restart", Name); err != nil {
		return fmt.Errorf("reiniciar unidad systemd: %w", err)
	}
	return nil
}

func (m *systemdManager) restore(ctx context.Context, previous []byte, existed bool, operation error) error {
	var restoreErr error
	if existed {
		restoreErr = replaceDefinition(ctx, m.store, m.path, previous)
	} else {
		restoreErr = os.Remove(m.path)
		if os.IsNotExist(restoreErr) {
			restoreErr = nil
		}
	}
	if restoreErr != nil {
		return fmt.Errorf("%w; no se pudo restaurar la unidad previa: %v", operation, restoreErr)
	}
	_, _ = runCommand(ctx, m.runner, "systemctl", "--user", "daemon-reload")
	return operation
}

func systemdUnit(spec Spec) string {
	return "[Unit]\nDescription=MCP Gateway\n\n[Service]\nType=simple\nExecStart=" + systemdEscape(spec.Binary()) + " serve --port " + spec.Port().Decimal() + "\nRestart=on-failure\n\n[Install]\nWantedBy=default.target\n"
}

// systemd accepts double-quoted C-style words. strconv.Quote preserves argv
// boundaries without delegating parsing to a shell.
func systemdEscape(value string) string {
	return strconv.Quote(strings.ReplaceAll(value, "\n", ""))
}
