package claude

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mcp-gateway/internal/endpoint"
	"mcp-gateway/internal/persist"
	"mcp-gateway/internal/project"
)

func TestProjectRegistrarPreservaJSONAjenoYEsIdempotente(t *testing.T) {
	directory := t.TempDir()
	original := `{"future":{"enabled":true},"mcpServers":{"other":{"type":"stdio","command":"other"}}}`
	if err := os.WriteFile(filepath.Join(directory, ".mcp.json"), []byte(original), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, ".gitignore"), []byte("build/\n.mcp.json\n.mcp.json\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	registrar := NewDefaultProjectRegistrar()
	dir, _ := project.Resolve(directory)
	result, err := registrar.Register(context.Background(), dir, endpoint.MustPort(3333))
	if err != nil || !result.Created || result.Updated || !result.Changed {
		t.Fatalf("Register = %#v, %v", result, err)
	}
	data, err := os.ReadFile(filepath.Join(directory, ".mcp.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 || data[len(data)-1] != '\n' || !json.Valid(data) {
		t.Fatalf("JSON no estable: %q", data)
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	if !jsonSemanticallyEqual(document["future"], json.RawMessage(`{"enabled":true}`)) {
		t.Fatalf("se alteró clave ajena: %s", document["future"])
	}
	var servers map[string]map[string]any
	if err := json.Unmarshal(document["mcpServers"], &servers); err != nil {
		t.Fatal(err)
	}
	wantedURL := endpoint.LocalhostURL(endpoint.MustPort(3333), "/sse", url.Values{"projectDir": []string{directory}})
	if servers["other"]["command"] != "other" || servers[projectServerName]["type"] != "sse" || servers[projectServerName]["url"] != wantedURL {
		t.Fatalf("servers = %#v", servers)
	}
	ignore, _ := os.ReadFile(filepath.Join(directory, ".gitignore"))
	if strings.Count(string(ignore), ".mcp.json") != 1 {
		t.Fatalf("gitignore = %q", ignore)
	}
	beforeJSON := append([]byte(nil), data...)
	beforeIgnore := append([]byte(nil), ignore...)
	result, err = registrar.Register(context.Background(), dir, endpoint.MustPort(3333))
	if err != nil || result.Changed {
		t.Fatalf("registro repetido = %#v, %v", result, err)
	}
	afterJSON, _ := os.ReadFile(filepath.Join(directory, ".mcp.json"))
	afterIgnore, _ := os.ReadFile(filepath.Join(directory, ".gitignore"))
	if string(beforeJSON) != string(afterJSON) || string(beforeIgnore) != string(afterIgnore) {
		t.Fatal("el registro idéntico reescribió archivos")
	}
}

func TestProjectRegistrarCreaArchivosConURLCodificada(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "proyecto con espacio")
	if err := os.Mkdir(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	dir, _ := project.Resolve(directory)
	result, err := NewDefaultProjectRegistrar().Register(context.Background(), dir, endpoint.MustPort(4444))
	if err != nil || !result.Created || !result.Changed {
		t.Fatalf("Register = %#v, %v", result, err)
	}
	data, _ := os.ReadFile(filepath.Join(directory, ".mcp.json"))
	if !strings.Contains(string(data), `http://localhost:4444/sse?projectDir=`) || !strings.Contains(string(data), "+con+espacio") {
		t.Fatalf("URL = %s", data)
	}
	ignore, _ := os.ReadFile(filepath.Join(directory, ".gitignore"))
	if string(ignore) != ".mcp.json\n" {
		t.Fatalf("gitignore = %q", ignore)
	}
}

func TestProjectRegistrarNoSobrescribeJSONMalformado(t *testing.T) {
	directory := t.TempDir()
	jsonPath := filepath.Join(directory, ".mcp.json")
	ignorePath := filepath.Join(directory, ".gitignore")
	malformed := []byte(`{"mcpServers":`)
	if err := os.WriteFile(jsonPath, malformed, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ignorePath, []byte("build/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir, _ := project.Resolve(directory)
	if _, err := NewDefaultProjectRegistrar().Register(context.Background(), dir, endpoint.MustPort(3333)); err == nil {
		t.Fatal("se esperaba error")
	}
	afterJSON, _ := os.ReadFile(jsonPath)
	afterIgnore, _ := os.ReadFile(ignorePath)
	if string(afterJSON) != string(malformed) || string(afterIgnore) != "build/\n" {
		t.Fatal("se modificaron archivos tras JSON malformado")
	}
}

type failingProjectStore struct {
	real   *persist.Store
	calls  int
	failAt map[int]error
}

func (s *failingProjectStore) WithLock(ctx context.Context, path string, fn func() error) error {
	return s.real.WithLock(ctx, path, fn)
}

func (s *failingProjectStore) Replace(ctx context.Context, path string, policy persist.ModePolicy, fn func(io.Writer) error) error {
	s.calls++
	if err := s.failAt[s.calls]; err != nil {
		return err
	}
	return s.real.Replace(ctx, path, policy, fn)
}

func TestProjectRegistrarRestauraPrimerArchivoSiFallaSegundo(t *testing.T) {
	directory := t.TempDir()
	jsonPath := filepath.Join(directory, ".mcp.json")
	original := []byte("{\n  \"mcpServers\": {}\n}\n")
	if err := os.WriteFile(jsonPath, original, 0o600); err != nil {
		t.Fatal(err)
	}
	store := &failingProjectStore{real: persist.NewStore(), failAt: map[int]error{2: errors.New("fallo inyectado")}}
	registrar, _ := NewProjectRegistrar(store)
	dir, _ := project.Resolve(directory)
	if _, err := registrar.Register(context.Background(), dir, endpoint.MustPort(3333)); err == nil || strings.Contains(err.Error(), "parcialmente") {
		t.Fatalf("error de rollback = %v", err)
	}
	after, _ := os.ReadFile(jsonPath)
	if string(after) != string(original) {
		t.Fatalf("no se restauró .mcp.json: %q", after)
	}
	if _, err := os.Stat(filepath.Join(directory, ".gitignore")); !os.IsNotExist(err) {
		t.Fatalf(".gitignore inesperado: %v", err)
	}
}

func TestProjectRegistrarExponeRecuperacionParcial(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, ".mcp.json"), []byte(`{"mcpServers":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	store := &failingProjectStore{real: persist.NewStore(), failAt: map[int]error{
		2: errors.New("fallo de ignore"), 3: errors.New("fallo de recuperación"),
	}}
	registrar, _ := NewProjectRegistrar(store)
	dir, _ := project.Resolve(directory)
	_, err := registrar.Register(context.Background(), dir, endpoint.MustPort(3333))
	var partial *PartialRecoveryError
	if !errors.As(err, &partial) || !strings.Contains(err.Error(), "parcialmente") {
		t.Fatalf("error = %T %v", err, err)
	}
}
