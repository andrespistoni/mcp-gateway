package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mcp-gateway/internal/diagnostics"
	"mcp-gateway/internal/endpoint"
	"mcp-gateway/internal/persist"
	"mcp-gateway/internal/project"
)

const projectServerName = "mcp-gateway"

type projectStore interface {
	WithLock(context.Context, string, func() error) error
	Replace(context.Context, string, persist.ModePolicy, func(io.Writer) error) error
}

type ProjectRegistrar struct {
	store projectStore
}

type ProjectRegistration struct {
	Created bool
	Updated bool
	Changed bool
}

type PartialRecoveryError struct {
	UpdateErr   error
	RecoveryErr error
}

func (e *PartialRecoveryError) Error() string {
	return "el registro quedó parcialmente actualizado y la recuperación falló"
}

func (e *PartialRecoveryError) Unwrap() []error {
	return []error{e.UpdateErr, e.RecoveryErr}
}

func NewProjectRegistrar(store projectStore) (*ProjectRegistrar, error) {
	if store == nil {
		return nil, fmt.Errorf("persist store es obligatorio")
	}
	return &ProjectRegistrar{store: store}, nil
}

func NewDefaultProjectRegistrar() *ProjectRegistrar {
	registrar, _ := NewProjectRegistrar(persist.NewStore())
	return registrar
}

func (r *ProjectRegistrar) Register(ctx context.Context, directory project.Dir, port endpoint.Port) (ProjectRegistration, error) {
	if directory.Path() == "" {
		return ProjectRegistration{}, diagnostics.NewFault(diagnostics.Validation, "projectDir no es válido", nil)
	}
	jsonPath := filepath.Join(directory.Path(), ".mcp.json")
	ignorePath := filepath.Join(directory.Path(), ".gitignore")
	lockPath := filepath.Join(directory.Path(), ".mcp-gateway-register.lock")
	var result ProjectRegistration
	err := r.store.WithLock(ctx, lockPath, func() error {
		originalJSON, jsonExists, err := readProjectFile(jsonPath)
		if err != nil {
			return diagnostics.NewFault(diagnostics.Persistence, "no se pudo leer .mcp.json", err)
		}
		originalIgnore, ignoreExists, err := readProjectFile(ignorePath)
		if err != nil {
			return diagnostics.NewFault(diagnostics.Persistence, "no se pudo leer .gitignore", err)
		}

		gatewayURL := endpoint.LocalhostURL(port, "/sse", url.Values{"projectDir": []string{directory.Path()}})
		updatedJSON, existed, err := mergeProjectJSON(originalJSON, jsonExists, gatewayURL)
		if err != nil {
			return diagnostics.NewFault(diagnostics.Configuration, ".mcp.json existente no es válido", err)
		}
		updatedIgnore := mergeGitignore(originalIgnore)
		jsonChanged := !jsonExists || !bytes.Equal(originalJSON, updatedJSON)
		ignoreChanged := !ignoreExists || !bytes.Equal(originalIgnore, updatedIgnore)
		result = ProjectRegistration{Created: !existed, Updated: existed, Changed: jsonChanged || ignoreChanged}

		if jsonChanged {
			if err := replaceBytes(ctx, r.store, jsonPath, updatedJSON); err != nil {
				return diagnostics.NewFault(diagnostics.Persistence, "no se pudo actualizar .mcp.json", err)
			}
		}
		if !ignoreChanged {
			return nil
		}
		if err := replaceBytes(ctx, r.store, ignorePath, updatedIgnore); err == nil {
			return nil
		} else if !jsonChanged {
			return diagnostics.NewFault(diagnostics.Persistence, "no se pudo actualizar .gitignore", err)
		} else {
			updateErr := err
			recoveryCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			recoveryErr := restoreProjectJSON(recoveryCtx, r.store, jsonPath, originalJSON, jsonExists)
			if recoveryErr != nil {
				partial := &PartialRecoveryError{UpdateErr: updateErr, RecoveryErr: recoveryErr}
				return diagnostics.NewFault(diagnostics.Persistence, "el registro quedó parcialmente actualizado; falló la recuperación", partial)
			}
			return diagnostics.NewFault(diagnostics.Persistence, "no se pudo actualizar .gitignore; se restauró .mcp.json", updateErr)
		}
	})
	if err != nil {
		return ProjectRegistration{}, err
	}
	return result, nil
}

func readProjectFile(path string) ([]byte, bool, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if !info.Mode().IsRegular() {
		return nil, false, fmt.Errorf("el destino no es un archivo regular")
	}
	data, err := os.ReadFile(path)
	return data, true, err
}

func mergeProjectJSON(data []byte, exists bool, gatewayURL string) ([]byte, bool, error) {
	root := make(map[string]json.RawMessage)
	if exists {
		if err := validateJSON(data); err != nil {
			return nil, false, err
		}
		if err := json.Unmarshal(data, &root); err != nil || root == nil {
			return nil, false, fmt.Errorf("la raíz debe ser un objeto JSON")
		}
	}
	servers := make(map[string]json.RawMessage)
	if raw, ok := root["mcpServers"]; ok {
		if err := json.Unmarshal(raw, &servers); err != nil || servers == nil {
			return nil, false, fmt.Errorf("mcpServers debe ser un objeto JSON")
		}
	}
	_, existed := servers[projectServerName]
	entry, err := json.Marshal(map[string]string{"type": "sse", "url": gatewayURL})
	if err != nil {
		return nil, false, err
	}
	servers[projectServerName] = entry
	serversRaw, err := json.Marshal(servers)
	if err != nil {
		return nil, false, err
	}
	root["mcpServers"] = serversRaw
	encoded, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, false, err
	}
	return append(encoded, '\n'), existed, nil
}

func validateJSON(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := validateJSONValue(decoder); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return fmt.Errorf("JSON contiene valores adicionales")
	}
	return nil
}

func validateJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("clave JSON inválida")
			}
			if _, duplicate := seen[key]; duplicate {
				return fmt.Errorf("clave JSON duplicada")
			}
			seen[key] = struct{}{}
			if err := validateJSONValue(decoder); err != nil {
				return err
			}
		}
	case '[':
		for decoder.More() {
			if err := validateJSONValue(decoder); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("delimitador JSON inválido")
	}
	_, err = decoder.Token()
	return err
}

func mergeGitignore(data []byte) []byte {
	segments := strings.SplitAfter(string(data), "\n")
	result := make([]string, 0, len(segments)+1)
	found := false
	changed := false
	newline := "\n"
	if bytes.Contains(data, []byte("\r\n")) {
		newline = "\r\n"
	}
	for _, segment := range segments {
		if segment == "" {
			continue
		}
		line := strings.TrimSuffix(strings.TrimSuffix(segment, "\n"), "\r")
		if line == ".mcp.json" {
			if found {
				changed = true
				continue
			}
			found = true
		}
		result = append(result, segment)
	}
	if found && !changed {
		return append([]byte(nil), data...)
	}
	if !found {
		if len(result) > 0 && !strings.HasSuffix(result[len(result)-1], "\n") {
			result[len(result)-1] += newline
		}
		result = append(result, ".mcp.json"+newline)
	}
	return []byte(strings.Join(result, ""))
}

func replaceBytes(ctx context.Context, store projectStore, path string, data []byte) error {
	policy := persist.ModePolicy{Mode: 0o644, PreserveExisting: true}
	return store.Replace(ctx, path, policy, func(writer io.Writer) error {
		_, err := writer.Write(data)
		return err
	})
}

func restoreProjectJSON(ctx context.Context, store projectStore, path string, data []byte, existed bool) error {
	if existed {
		return replaceBytes(ctx, store, path, data)
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
