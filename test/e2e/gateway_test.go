package e2e_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestGatewayWorkflowAislado(t *testing.T) {
	runGatewayWorkflow(t)
}

func runGatewayWorkflow(t *testing.T) {
	t.Helper()
	environment := newIsolatedEnvironment(t)
	port := dynamicPort(t)

	// setup usa exclusivamente el gestor falso del PATH temporal. discover no
	// encuentra recetas en ese PATH y no ejecuta discovery real.
	setup := environment.run(t, "setup", "--port", strconv.Itoa(port))
	if !strings.Contains(setup, "Setup completado") {
		t.Fatalf("salida setup = %q", setup)
	}
	_ = environment.run(t, "discover", "--write")
	environment.writeRuntimeConfiguration(t, port)

	registered := environment.run(t, "register-project", "--project-dir", environment.project, "--port", strconv.Itoa(port))
	if !strings.Contains(registered, "Proyecto registrado") {
		t.Fatalf("salida register-project = %q", registered)
	}

	environment.startServe(t, port)
	response, reader := openSSE(t, port)
	t.Cleanup(func() { _ = response.Body.Close() })
	endpointPath := string(readSSEEvent(t, reader, "endpoint"))
	if !strings.HasPrefix(endpointPath, "/message?sessionId=") {
		t.Fatalf("endpoint SSE = %q", endpointPath)
	}

	postJSONRPC(t, port, endpointPath, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}`)
	requireResultID(t, readSSEEvent(t, reader, "message"), 1)
	postJSONRPC(t, port, endpointPath, `{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	postJSONRPC(t, port, endpointPath, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	tools := requireResultID(t, readSSEEvent(t, reader, "message"), 2)
	if !strings.Contains(string(tools["result"]), `"alpha__echo"`) || !strings.Contains(string(tools["result"]), `"beta__echo"`) {
		t.Fatalf("tools/list no conserva ambos downstreams: %s", tools["result"])
	}

	for _, call := range []struct {
		id   int
		tool string
	}{{id: 3, tool: "alpha__echo"}, {id: 4, tool: "beta__echo"}} {
		payload, err := json.Marshal(map[string]any{
			"jsonrpc": "2.0", "id": call.id, "method": "tools/call",
			"params": map[string]any{"name": call.tool, "arguments": map[string]any{"saludo": "hola"}},
		})
		if err != nil {
			t.Fatal(err)
		}
		postJSONRPC(t, port, endpointPath, string(payload))
		result := requireResultID(t, readSSEEvent(t, reader, "message"), call.id)
		if !strings.Contains(string(result["result"]), `"name":"echo"`) || !strings.Contains(string(result["result"]), `"futureResult":true`) {
			t.Fatalf("tools/call %s = %s", call.tool, result["result"])
		}
	}

	projectConfig, err := os.ReadFile(filepath.Join(environment.project, ".mcp.json"))
	if err != nil || !strings.Contains(string(projectConfig), "http://localhost:"+strconv.Itoa(port)+"/sse") {
		t.Fatalf("registro de proyecto = %q, %v", projectConfig, err)
	}
}
