package fakemcp

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"mcp-gateway/internal/mcp"
)

func TestFakeHealthyHandshakeAndPagination(t *testing.T) {
	var input bytes.Buffer
	codec := mcp.NewCodec(nil, &input)
	initialize, _ := mcp.NewRequest(mcp.NumberID(1), "initialize", map[string]any{"protocolVersion": mcp.ProtocolVersion})
	initialized, _ := mcp.NewNotification("notifications/initialized", map[string]any{})
	list, _ := mcp.NewRequest(mcp.NumberID(2), "tools/list", map[string]any{})
	for _, envelope := range []mcp.Envelope{initialize, initialized, list} {
		if err := codec.Write(envelope); err != nil {
			t.Fatal(err)
		}
	}
	var output, failures bytes.Buffer
	if code := Main([]string{"--scenario=healthy"}, &input, &output, &failures); code != 0 {
		t.Fatalf("code=%d stderr=%q", code, failures.String())
	}
	reader := mcp.NewCodec(&output, nil)
	for range 2 {
		if _, err := reader.Read(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestBuildCreatesOnlyRequestedTemporaryBinary(t *testing.T) {
	name := "fake-mcp"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	destination := filepath.Join(t.TempDir(), name)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := Build(ctx, destination); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(destination); err != nil || !info.Mode().IsRegular() {
		t.Fatalf("binario temporal inválido: %v", err)
	}
}
