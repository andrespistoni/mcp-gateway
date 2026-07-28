package sse

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"mcp-gateway/internal/endpoint"
)

func TestMessageRejectsPreAdmissionFailuresOverHTTPOnly(t *testing.T) {
	t.Parallel()
	port := endpoint.MustPort(4444)
	server := &Server{port: port, registry: newRegistry()}
	for _, test := range []struct {
		name        string
		contentType string
		path        string
		body        string
		status      int
	}{
		{name: "tipo", contentType: "text/plain", path: "/message?sessionId=x", body: `{}`, status: http.StatusUnsupportedMediaType},
		{name: "id-ausente", contentType: "application/json", path: "/message", body: `{}`, status: http.StatusBadRequest},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, test.path, strings.NewReader(test.body))
			request.Header.Set("Content-Type", test.contentType)
			writer := httptest.NewRecorder()
			server.handleMessage(writer, request)
			if writer.Code != test.status {
				t.Fatalf("status = %d, se esperaba %d", writer.Code, test.status)
			}
			if writer.Header().Get("Content-Type") != "application/json" {
				t.Fatalf("Content-Type = %q", writer.Header().Get("Content-Type"))
			}
		})
	}
}
