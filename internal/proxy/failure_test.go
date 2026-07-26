package proxy

import (
	"context"
	"testing"

	"mcp-gateway/internal/config"
	"mcp-gateway/internal/testsupport/fakemcp"
)

func TestBadDownstreamDoesNotAffectHealthyRuntime(t *testing.T) {
	binary := buildFake(t)
	service, err := Start(context.Background(), []config.Downstream{
		downstream("invalid", "invalid__", binary, fakemcp.Batch, true),
		downstream("healthy", "healthy__", binary, fakemcp.RuntimeHealthy, true),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { shutdownService(t, service) })
	statuses := service.Statuses()
	if statuses[0].State != StateUnavailable || statuses[1].State != StateAvailable {
		t.Fatalf("estados = %#v", statuses)
	}
	if names := toolNames(service.Catalog()); len(names) != 1 || names[0] != "healthy__echo" {
		t.Fatalf("catálogo = %v", names)
	}
}
