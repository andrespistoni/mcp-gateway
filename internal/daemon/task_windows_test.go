//go:build windows

package daemon

import (
	"strings"
	"testing"

	"mcp-gateway/internal/endpoint"
)

func TestTaskXMLHasNativeCommandAndArguments(t *testing.T) {
	spec, _ := NewSpec(`C:\Program Files\mcp-gateway.exe`, endpoint.MustPort(3333))
	text := string(taskXML(spec))
	for _, want := range []string{"<LogonTrigger>", "<Command>C:\\Program Files\\mcp-gateway.exe</Command>", "<Arguments>serve --port 3333</Arguments>", "<LogonType>InteractiveToken</LogonType>"} {
		if !strings.Contains(text, want) {
			t.Fatalf("XML no contiene %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "cmd.exe") || strings.Contains(text, "powershell") || strings.Contains(text, "127.0.0.1") {
		t.Fatalf("XML inseguro:\n%s", text)
	}
}
