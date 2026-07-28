package daemon

import (
	"path/filepath"
	"strings"
	"testing"

	"mcp-gateway/internal/endpoint"
)

func TestSpecRequiresAbsoluteBinaryAndBuildsLogicalArgv(t *testing.T) {
	if _, err := NewSpec("mcp-gateway", endpoint.MustPort(3333)); err == nil {
		t.Fatal("NewSpec aceptó ruta relativa")
	}
	spec, err := NewSpec(filepath.Join(t.TempDir(), "mcp-gateway"), endpoint.MustPort(4444))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(spec.Args(), " "); got != "serve --port 4444" {
		t.Fatalf("Args = %q", got)
	}
}

func TestOutputCaptureLimitsAndContinues(t *testing.T) {
	capture := &outputCapture{remaining: 3}
	if _, err := capture.Write([]byte("abcdef")); err != nil {
		t.Fatal(err)
	}
	result := capture.result()
	if string(result.Output) != "abc" || !result.Truncated {
		t.Fatalf("capture = %#v", result)
	}
}
