package sse

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"mcp-gateway/internal/endpoint"
	"mcp-gateway/internal/mcp"
)

type testCatalog struct {
	tools    []mcp.Tool
	identity [32]byte
}

func (c testCatalog) Tools() []mcp.Tool  { return append([]mcp.Tool(nil), c.tools...) }
func (c testCatalog) Identity() [32]byte { return c.identity }

func TestSSEInitializePingAndToolsListOnDynamicPort(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port, err := endpoint.NewPort(listener.Addr().(*net.TCPAddr).Port)
	if err != nil {
		t.Fatal(err)
	}
	tool, err := mcp.ParseTool(json.RawMessage(`{"name":"example","description":"prueba"}`))
	if err != nil {
		t.Fatal(err)
	}
	server, err := bindListener(port, listener, Dependencies{Entropy: testEntropy{7}})
	if err != nil {
		t.Fatal(err)
	}
	if err := server.SetCatalog(testCatalog{tools: []mcp.Tool{tool}}); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- server.Serve() }()
	t.Cleanup(func() {
		_ = server.Close()
		select {
		case err := <-done:
			if err != nil {
				t.Error(err)
			}
		case <-time.After(time.Second):
			t.Error("Serve no terminó")
		}
	})

	url := fmt.Sprintf("http://localhost:%d/sse", port.Number())
	response, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || response.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("respuesta SSE = %d, %q", response.StatusCode, response.Header.Get("Content-Type"))
	}
	reader := bufio.NewReader(response.Body)
	line, err := reader.ReadString('\n')
	if err != nil || line != "event: endpoint\n" {
		t.Fatalf("primer evento = %q, %v", line, err)
	}
	data, err := reader.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	endpointPath := strings.TrimSpace(strings.TrimPrefix(data, "data: "))
	_, _ = reader.ReadString('\n')

	post := func(payload string) {
		request, err := http.NewRequestWithContext(context.Background(), http.MethodPost, fmt.Sprintf("http://localhost:%d%s", port.Number(), endpointPath), strings.NewReader(payload))
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Content-Type", "application/json; charset=utf-8")
		result, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		defer result.Body.Close()
		if result.StatusCode != http.StatusAccepted {
			body, _ := io.ReadAll(result.Body)
			t.Fatalf("POST status = %d, body=%s", result.StatusCode, body)
		}
	}
	post(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}`)
	assertMessage(t, reader, "initialize")
	post(`{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	post(`{"jsonrpc":"2.0","id":2,"method":"ping"}`)
	assertMessage(t, reader, "ping")
	post(`{"jsonrpc":"2.0","id":3,"method":"tools/list"}`)
	assertMessage(t, reader, "tools/list")
}

func assertMessage(t *testing.T, reader *bufio.Reader, name string) {
	t.Helper()
	line, err := reader.ReadString('\n')
	if err != nil || line != "event: message\n" {
		t.Fatalf("%s: evento = %q, %v", name, line, err)
	}
	data, err := reader.ReadString('\n')
	if err != nil || !strings.HasPrefix(data, "data: {\"") {
		t.Fatalf("%s: data = %q, %v", name, data, err)
	}
	_, _ = reader.ReadString('\n')
}
