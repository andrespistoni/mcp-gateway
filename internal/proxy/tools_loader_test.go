package proxy

import (
	"context"
	"testing"

	"mcp-gateway/internal/config"
	"mcp-gateway/internal/testsupport/fakemcp"
)

func TestRuntimeToolsLoaderLimitsAndFailures(t *testing.T) {
	binary := buildFake(t)
	tests := []struct {
		scenario fakemcp.Scenario
		count    int
		failed   bool
	}{
		{scenario: fakemcp.HundredPages, count: 100},
		{scenario: fakemcp.MaxTools, count: MaxTools},
		{scenario: fakemcp.TooManyPages, failed: true},
		{scenario: fakemcp.TooManyTools, failed: true},
		{scenario: fakemcp.CursorCycle, failed: true},
		{scenario: fakemcp.InvalidCursor, failed: true},
		{scenario: fakemcp.InvalidTools, failed: true},
		{scenario: fakemcp.MissingTools, failed: true},
		{scenario: fakemcp.InvalidTool, failed: true},
	}
	for _, test := range tests {
		t.Run(string(test.scenario), func(t *testing.T) {
			configured := downstream("tested", "tested__", binary, test.scenario, true)
			service, err := Start(context.Background(), []config.Downstream{configured})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { shutdownService(t, service) })
			status := service.Statuses()[0]
			if test.failed {
				if status.State != StateUnavailable || len(service.Catalog().Tools()) != 0 {
					t.Fatalf("state=%s tools=%d", status.State, len(service.Catalog().Tools()))
				}
				return
			}
			if len(service.Catalog().Tools()) != test.count {
				t.Fatalf("tools=%d want=%d", len(service.Catalog().Tools()), test.count)
			}
		})
	}
}
