//go:build darwin

package daemon

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"mcp-gateway/internal/persist"
)

const launchdFileName = "mcp-gateway.plist"

type launchdManager struct {
	path   string
	domain string
	store  definitionStore
	runner Runner
}

func newDefaultLaunchdManager(runner Runner) (*launchdManager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return newLaunchdManager(filepath.Join(home, "Library", "LaunchAgents", launchdFileName), "gui/"+strconv.Itoa(os.Getuid()), persist.NewStore(), runner)
}

func newLaunchdManager(path, domain string, store definitionStore, runner Runner) (*launchdManager, error) {
	if path == "" || !filepath.IsAbs(path) || domain == "" || store == nil || runner == nil {
		return nil, fmt.Errorf("composición launchd incompleta")
	}
	return &launchdManager{path: filepath.Clean(path), domain: domain, store: store, runner: runner}, nil
}

func (m *launchdManager) Status(ctx context.Context) (Status, error) {
	if err := ctx.Err(); err != nil {
		return Status{}, err
	}
	_, exists, err := readDefinition(m.path)
	if err != nil || !exists {
		return Status{}, err
	}
	result, err := runCommand(ctx, m.runner, "launchctl", "print", m.domain+"/"+Name)
	if err != nil {
		return Status{}, fmt.Errorf("consultar LaunchAgent: %w", err)
	}
	return Status{Installed: true, Running: result.ExitCode == 0}, nil
}

func (m *launchdManager) Enable(ctx context.Context, spec Spec) error {
	if err := requireSpec(spec); err != nil {
		return err
	}
	previous, existed, err := readDefinition(m.path)
	if err != nil {
		return err
	}
	if existed {
		if _, err := runCommand(ctx, m.runner, "launchctl", "bootout", m.domain, m.path); err != nil {
			return fmt.Errorf("detener LaunchAgent anterior: %w", err)
		}
	}
	if err := m.store.EnsurePrivateDirectory(filepath.Dir(m.path)); err != nil {
		return fmt.Errorf("preparar directorio LaunchAgents: %w", err)
	}
	if err := replaceDefinition(ctx, m.store, m.path, launchdPlist(spec)); err != nil {
		return fmt.Errorf("persistir LaunchAgent: %w", err)
	}
	if _, err := runCommand(ctx, m.runner, "launchctl", "bootstrap", m.domain, m.path); err != nil {
		return m.restore(ctx, previous, existed, fmt.Errorf("registrar LaunchAgent: %w", err))
	}
	if _, err := runCommand(ctx, m.runner, "launchctl", "kickstart", "-k", m.domain+"/"+Name); err != nil {
		return m.restore(ctx, previous, existed, fmt.Errorf("iniciar LaunchAgent: %w", err))
	}
	return nil
}

func (m *launchdManager) Disable(ctx context.Context) error {
	_, exists, err := readDefinition(m.path)
	if err != nil || !exists {
		return err
	}
	if _, err := runCommand(ctx, m.runner, "launchctl", "bootout", m.domain, m.path); err != nil {
		return fmt.Errorf("deshabilitar LaunchAgent: %w", err)
	}
	if err := os.Remove(m.path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("eliminar LaunchAgent: %w", err)
	}
	return nil
}

func (m *launchdManager) Restart(ctx context.Context) error {
	if _, exists, err := readDefinition(m.path); err != nil {
		return err
	} else if !exists {
		return fmt.Errorf("el daemon no está instalado")
	}
	if _, err := runCommand(ctx, m.runner, "launchctl", "kickstart", "-k", m.domain+"/"+Name); err != nil {
		return fmt.Errorf("reiniciar LaunchAgent: %w", err)
	}
	return nil
}

func (m *launchdManager) restore(ctx context.Context, previous []byte, existed bool, operation error) error {
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
		return fmt.Errorf("%w; no se pudo restaurar el LaunchAgent previo: %v", operation, restoreErr)
	}
	return operation
}

func launchdPlist(spec Spec) []byte {
	var output bytes.Buffer
	encoder := xml.NewEncoder(&output)
	_, _ = output.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<!DOCTYPE plist PUBLIC \"-//Apple//DTD PLIST 1.0//EN\" \"http://www.apple.com/DTDs/PropertyList-1.0.dtd\">\n<plist version=\"1.0\"><dict>\n")
	writePlistKeyString(encoder, "Label", Name)
	_, _ = output.WriteString("<key>ProgramArguments</key><array>")
	for _, argument := range append([]string{spec.Binary()}, spec.Args()...) {
		_ = encoder.EncodeElement(argument, xml.StartElement{Name: xml.Name{Local: "string"}})
	}
	_, _ = output.WriteString("</array><key>RunAtLoad</key><true/>\n</dict></plist>\n")
	return output.Bytes()
}

func writePlistKeyString(encoder *xml.Encoder, key, value string) {
	_ = encoder.EncodeElement(key, xml.StartElement{Name: xml.Name{Local: "key"}})
	_ = encoder.EncodeElement(value, xml.StartElement{Name: xml.Name{Local: "string"}})
}
