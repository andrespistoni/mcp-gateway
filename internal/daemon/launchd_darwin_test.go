//go:build darwin

package daemon

import (
	"strings"
	"testing"

	"mcp-gateway/internal/endpoint"
)

func TestLaunchdPlistUsesSeparatedArguments(t *testing.T) {
	spec, _ := NewSpec("/Applications/MCP Gateway/mcp-gateway", endpoint.MustPort(3333))
	text := string(launchdPlist(spec))
	for _, want := range []string{"<key>Label</key><string>mcp-gateway</string>", "<key>ProgramArguments</key><array>", "<string>serve</string>", "<string>--port</string>", "<string>3333</string>", "<key>RunAtLoad</key><true/>"} {
		if !strings.Contains(text, want) {
			t.Fatalf("plist no contiene %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "sh -c") || strings.Contains(text, "127.0.0.1") {
		t.Fatalf("plist inseguro:\n%s", text)
	}
}
