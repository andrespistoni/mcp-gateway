//go:build windows

package daemon

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"strings"

	"mcp-gateway/internal/persist"
)

type taskManager struct {
	store  definitionStore
	runner Runner
	temp   func(string, string) (*os.File, error)
}

func newDefaultTaskManager(runner Runner) (*taskManager, error) {
	return newTaskManager(persist.NewStore(), runner), nil
}

func newTaskManager(store definitionStore, runner Runner) *taskManager {
	return &taskManager{store: store, runner: runner, temp: os.CreateTemp}
}

func (m *taskManager) Status(ctx context.Context) (Status, error) {
	result, err := runCommand(ctx, m.runner, "schtasks.exe", "/Query", "/TN", Name, "/XML")
	if err != nil {
		return Status{}, fmt.Errorf("consultar tarea programada: %w", err)
	}
	return Status{Installed: result.ExitCode == 0}, nil
}

func (m *taskManager) Enable(ctx context.Context, spec Spec) error {
	if err := requireSpec(spec); err != nil {
		return err
	}
	status, err := m.Status(ctx)
	if err != nil {
		return err
	}
	if status.Installed {
		if _, err := runCommand(ctx, m.runner, "schtasks.exe", "/End", "/TN", Name); err != nil {
			return fmt.Errorf("detener tarea anterior: %w", err)
		}
	}
	path, err := m.writeTemporaryXML(taskXML(spec))
	if err != nil {
		return err
	}
	defer os.Remove(path)
	if result, err := runCommand(ctx, m.runner, "schtasks.exe", "/Create", "/TN", Name, "/XML", path, "/F"); err != nil || result.ExitCode != 0 {
		if err == nil {
			err = fmt.Errorf("schtasks /Create devolvió %d", result.ExitCode)
		}
		return fmt.Errorf("crear tarea programada: %w", err)
	}
	if _, err := runCommand(ctx, m.runner, "schtasks.exe", "/Run", "/TN", Name); err != nil {
		return fmt.Errorf("iniciar tarea programada: %w", err)
	}
	return nil
}

func (m *taskManager) Disable(ctx context.Context) error {
	status, err := m.Status(ctx)
	if err != nil || !status.Installed {
		return err
	}
	if _, err := runCommand(ctx, m.runner, "schtasks.exe", "/End", "/TN", Name); err != nil {
		return fmt.Errorf("detener tarea programada: %w", err)
	}
	if _, err := runCommand(ctx, m.runner, "schtasks.exe", "/Delete", "/TN", Name, "/F"); err != nil {
		return fmt.Errorf("eliminar tarea programada: %w", err)
	}
	return nil
}

func (m *taskManager) Restart(ctx context.Context) error {
	status, err := m.Status(ctx)
	if err != nil {
		return err
	}
	if !status.Installed {
		return fmt.Errorf("el daemon no está instalado")
	}
	if _, err := runCommand(ctx, m.runner, "schtasks.exe", "/End", "/TN", Name); err != nil {
		return fmt.Errorf("detener tarea programada: %w", err)
	}
	if _, err := runCommand(ctx, m.runner, "schtasks.exe", "/Run", "/TN", Name); err != nil {
		return fmt.Errorf("reiniciar tarea programada: %w", err)
	}
	return nil
}

func (m *taskManager) writeTemporaryXML(data []byte) (string, error) {
	file, err := m.temp("", "mcp-gateway-task-*.xml")
	if err != nil {
		return "", fmt.Errorf("crear XML temporal: %w", err)
	}
	path := file.Name()
	if err := file.Chmod(0o600); err == nil {
		_, err = file.Write(data)
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("escribir XML temporal: %w", err)
	}
	return path, nil
}

func taskXML(spec Spec) []byte {
	var output bytes.Buffer
	encoder := xml.NewEncoder(&output)
	_, _ = output.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?><Task version=\"1.4\" xmlns=\"http://schemas.microsoft.com/windows/2004/02/mit/task\">")
	_, _ = output.WriteString("<Triggers><LogonTrigger><Enabled>true</Enabled></LogonTrigger></Triggers><Principals><Principal id=\"Author\"><LogonType>InteractiveToken</LogonType><RunLevel>LeastPrivilege</RunLevel></Principal></Principals><Settings><MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy><StartWhenAvailable>true</StartWhenAvailable></Settings><Actions Context=\"Author\"><Exec>")
	_ = encoder.EncodeElement(spec.Binary(), xml.StartElement{Name: xml.Name{Local: "Command"}})
	_ = encoder.EncodeElement(strings.Join(spec.Args(), " "), xml.StartElement{Name: xml.Name{Local: "Arguments"}})
	_, _ = output.WriteString("</Exec></Actions></Task>\n")
	return output.Bytes()
}
