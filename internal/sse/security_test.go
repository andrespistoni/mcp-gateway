package sse

import (
	"net/http/httptest"
	"testing"

	"mcp-gateway/internal/endpoint"
)

func TestValidRequestOriginRequiresExactLocalhost(t *testing.T) {
	t.Parallel()
	port := endpoint.MustPort(4444)
	request := httptest.NewRequest("GET", "http://localhost:4444/sse", nil)
	request.Host = "localhost:4444"
	if !validRequestOrigin(request, port) {
		t.Fatal("se rechazó Host válido sin Origin")
	}
	request.Header.Set("Origin", "http://localhost:4444")
	if !validRequestOrigin(request, port) {
		t.Fatal("se rechazó Origin válido")
	}
	request.Header.Set("Origin", "http://127.0.0.1:4444")
	if validRequestOrigin(request, port) {
		t.Fatal("se aceptó Origin IP")
	}
}
