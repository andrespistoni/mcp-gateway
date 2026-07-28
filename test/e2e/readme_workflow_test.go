package e2e_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestREADMEWorkflow(t *testing.T) {
	readme, err := os.ReadFile(filepath.Join(repositoryRoot(t), "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		"<!-- flujo-e2e-verificado: inicio -->",
		"mcp-gateway setup --port \"$PUERTO\"",
		"mcp-gateway discover --write",
		"mcp-gateway register-project --project-dir \"$PROJECT_DIR\" --port \"$PUERTO\"",
		"mcp-gateway add <nombre>",
	} {
		if !strings.Contains(string(readme), required) {
			t.Fatalf("README no contiene %q", required)
		}
	}
	runGatewayWorkflow(t)
}
