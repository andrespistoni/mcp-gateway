package app

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"testing"

	"mcp-gateway/internal/config"
	"mcp-gateway/internal/endpoint"
	"mcp-gateway/internal/mcp"
	"mcp-gateway/internal/proxy"
	"mcp-gateway/internal/testsupport/fakemcp"
)

func TestRuntimeIntegrationStartsMixedDownstreamsAndBuildsCatalog(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "fake-mcp")
	if err := fakemcp.Build(context.Background(), binary); err != nil {
		t.Fatal(err)
	}
	snapshot, err := config.Decode([]byte(fmt.Sprintf(`version: 1
port: 4444
downstreams:
  - name: healthy
    prefix: healthy__
    binary: %s
    args: [--scenario=runtime-healthy]
  - name: disabled
    prefix: disabled__
    binary: %s
    enabled: false
`, strconv.Quote(binary), strconv.Quote(binary))))
	if err != nil {
		t.Fatal(err)
	}
	runtime := NewRuntime(startupConfig{snapshot: snapshot})
	events := []string{}
	resources, port, err := runtime.startGateway(context.Background(), nil, func(_ context.Context, port endpoint.Port) (startupListener, error) {
		if port.Number() != 4444 {
			t.Fatalf("puerto de integración = %d", port.Number())
		}
		return &recordingListener{events: &events}, nil
	}, func(_ startupListener, service startupProxy) error {
		concrete, ok := service.(*proxy.Service)
		if !ok {
			return fmt.Errorf("proxy = %T", service)
		}
		tools := concrete.Catalog().Tools()
		if len(tools) != 1 || tools[0].Name() != "healthy__echo" {
			return fmt.Errorf("catálogo = %v", toolNamesForIntegration(tools))
		}
		statuses := concrete.Statuses()
		if len(statuses) != 2 || statuses[0].State != proxy.StateAvailable || statuses[1].State != proxy.StateDisabled {
			return fmt.Errorf("estados = %#v", statuses)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if port.Number() != 4444 {
		t.Fatalf("puerto resuelto = %d", port.Number())
	}
	if err := resources.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(events) != "[listener-close]" {
		t.Fatalf("cierre del listener = %v", events)
	}
}

func toolNamesForIntegration(tools []mcp.Tool) []string {
	names := make([]string, len(tools))
	for index, tool := range tools {
		names[index] = tool.Name()
	}
	return names
}
