package sse

import (
	"net/http"

	"mcp-gateway/internal/endpoint"
)

func validRequestOrigin(request *http.Request, port endpoint.Port) bool {
	expectedHost := endpoint.LocalhostAddress(port)
	if request.Host != expectedHost {
		return false
	}
	origin := request.Header.Get("Origin")
	return origin == "" || origin == "http://"+expectedHost
}
