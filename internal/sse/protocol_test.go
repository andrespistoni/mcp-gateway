package sse

import (
	"encoding/json"
	"testing"

	"mcp-gateway/internal/mcp"
)

func TestInitializeNegotiatesSupportedProtocolVersion(t *testing.T) {
	t.Parallel()
	request, err := mcp.NewRequest(mcp.NumberID(1), "initialize", map[string]any{
		"protocolVersion": "2025-06-18",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "Claude Code", "version": "2.1.220"},
	})
	if err != nil {
		t.Fatal(err)
	}

	response := (&Server{}).dispatch(mcp.NumberID(1), request)
	if response.Kind() != mcp.Result {
		t.Fatalf("initialize devolvió %v, se esperaba result", response.Kind())
	}
	var result mcp.InitializeResult
	if err := json.Unmarshal(response.Result(), &result); err != nil {
		t.Fatal(err)
	}
	if result.ProtocolVersion != mcp.ProtocolVersion {
		t.Fatalf("protocolVersion = %q, se esperaba %q", result.ProtocolVersion, mcp.ProtocolVersion)
	}
}

func TestInitializeRejectsMissingProtocolVersion(t *testing.T) {
	t.Parallel()
	request, err := mcp.NewRequest(mcp.NumberID(1), "initialize", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}

	response := (&Server{}).dispatch(mcp.NumberID(1), request)
	rpcError, ok := response.RPCError()
	if response.Kind() != mcp.Error || !ok || rpcError.Code != -32602 {
		t.Fatalf("initialize sin versión = %#v", response)
	}
}
