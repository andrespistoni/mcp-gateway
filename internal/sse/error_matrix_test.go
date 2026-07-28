package sse

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"mcp-gateway/internal/endpoint"
)

type httpRPCError struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Error   struct {
		Code    int64  `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func TestHTTPJSONRPCPreAdmissionErrorMatrix(t *testing.T) {
	server, port := startMatrixServer(t)
	base := fmt.Sprintf("http://localhost:%d", port.Number())

	for _, test := range []struct {
		name        string
		path        string
		contentType string
		body        string
		status      int
		code        int64
		id          string
	}{
		{"content-type", "/message?sessionId=" + validMatrixSessionID(), "text/plain", `{}`, http.StatusUnsupportedMediaType, -32600, "null"},
		{"session-id-malformed", "/message?sessionId=no", "application/json", `{}`, http.StatusBadRequest, -32600, "null"},
		{"invalid-envelope-keeps-id", "/message?sessionId=" + validMatrixSessionID(), "application/json", `{"jsonrpc":"1.0","id":17,"method":"ping"}`, http.StatusBadRequest, -32600, "17"},
		{"unknown-session-keeps-id", "/message?sessionId=" + validMatrixSessionID(), "application/json", `{"jsonrpc":"2.0","id":"caller-id","method":"ping"}`, http.StatusNotFound, -32001, `"caller-id"`},
		{"batch", "/message?sessionId=" + validMatrixSessionID(), "application/json", `[{"jsonrpc":"2.0","id":1,"method":"ping"}]`, http.StatusBadRequest, -32600, "null"},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := matrixPOST(t, base+test.path, test.contentType, strings.NewReader(test.body))
			defer response.Body.Close()
			assertHTTPRPCError(t, response, test.status, test.code, test.id)
		})
	}

	validBody := matrixBodyAtLimit(maxHTTPBody)
	response := matrixPOST(t, base+"/message?sessionId="+validMatrixSessionID(), "application/json", strings.NewReader(validBody))
	defer response.Body.Close()
	assertHTTPRPCError(t, response, http.StatusNotFound, -32001, "1")

	overstated := matrixBodyAtLimit(maxHTTPBody + 1)
	response = matrixPOST(t, base+"/message?sessionId="+validMatrixSessionID(), "application/json", strings.NewReader(overstated))
	defer response.Body.Close()
	assertHTTPRPCError(t, response, http.StatusRequestEntityTooLarge, -32004, "null")

	_ = server
}

func TestHTTPJSONRPCAdmittedAndNotificationMatrix(t *testing.T) {
	_, port := startMatrixServer(t)
	base := fmt.Sprintf("http://localhost:%d", port.Number())
	stream, reader, endpointPath := openMatrixSession(t, base)
	defer stream.Close()

	response := matrixPOST(t, base+endpointPath, "application/json", strings.NewReader(`{"jsonrpc":"2.0","method":"notifications/initialized"}`))
	defer response.Body.Close()
	assertHTTPRPCError(t, response, http.StatusConflict, -32001, "null")

	response = matrixPOST(t, base+endpointPath, "application/json", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}`))
	defer response.Body.Close()
	if response.StatusCode != http.StatusAccepted {
		t.Fatalf("initialize status = %d", response.StatusCode)
	}
	assertMatrixSSEErrorOrResult(t, reader, "1", 0)

	response = matrixPOST(t, base+endpointPath, "application/json", strings.NewReader(`{"jsonrpc":"2.0","method":"notifications/initialized"}`))
	defer response.Body.Close()
	if response.StatusCode != http.StatusAccepted || response.ContentLength > 0 {
		t.Fatalf("initialized response = status %d, content-length %d", response.StatusCode, response.ContentLength)
	}

	response = matrixPOST(t, base+endpointPath, "application/json", strings.NewReader(`{"jsonrpc":"2.0","id":2,"method":"not-supported"}`))
	defer response.Body.Close()
	if response.StatusCode != http.StatusAccepted {
		t.Fatalf("unsupported status = %d", response.StatusCode)
	}
	assertMatrixSSEErrorOrResult(t, reader, "2", -32601)
}

func startMatrixServer(t *testing.T) (*Server, endpoint.Port) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port, err := endpoint.NewPort(listener.Addr().(*net.TCPAddr).Port)
	if err != nil {
		t.Fatal(err)
	}
	server, err := bindListener(port, listener, Dependencies{Entropy: testEntropy{value: 31}})
	if err != nil {
		t.Fatal(err)
	}
	if err := server.SetCatalog(testCatalog{}); err != nil {
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
	return server, port
}

func validMatrixSessionID() string {
	id, _ := NewSessionID(testEntropy{value: 32})
	return id.Encode()
}

func matrixPOST(t *testing.T, address, contentType string, body io.Reader) *http.Response {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, address, body)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", contentType)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func assertHTTPRPCError(t *testing.T, response *http.Response, status int, code int64, id string) {
	t.Helper()
	if response.StatusCode != status {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("status = %d, want %d, body=%s", response.StatusCode, status, body)
	}
	if response.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("Content-Type = %q", response.Header.Get("Content-Type"))
	}
	var payload httpRPCError
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.JSONRPC != "2.0" || payload.Error.Code != code || string(payload.ID) != id {
		t.Fatalf("error = %#v, id=%s; want code=%d id=%s", payload, payload.ID, code, id)
	}
}

func matrixBodyAtLimit(size int) string {
	const prefix = `{"jsonrpc":"2.0","id":1,"method":"ping","params":"`
	const suffix = `"}`
	return prefix + strings.Repeat("x", size-len(prefix)-len(suffix)) + suffix
}

func openMatrixSession(t *testing.T, base string) (io.ReadCloser, *bufio.Reader, string) {
	t.Helper()
	response, err := http.Get(base + "/sse")
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		response.Body.Close()
		t.Fatalf("SSE status = %d", response.StatusCode)
	}
	reader := bufio.NewReader(response.Body)
	line, err := reader.ReadString('\n')
	if err != nil || line != "event: endpoint\n" {
		response.Body.Close()
		t.Fatalf("endpoint event = %q, %v", line, err)
	}
	line, err = reader.ReadString('\n')
	if err != nil {
		response.Body.Close()
		t.Fatal(err)
	}
	endpointPath := strings.TrimSpace(strings.TrimPrefix(line, "data: "))
	if _, err := reader.ReadString('\n'); err != nil {
		response.Body.Close()
		t.Fatal(err)
	}
	return response.Body, reader, endpointPath
}

func assertMatrixSSEErrorOrResult(t *testing.T, reader *bufio.Reader, id string, code int64) {
	t.Helper()
	if line, err := reader.ReadString('\n'); err != nil || line != "event: message\n" {
		t.Fatalf("message event = %q, %v", line, err)
	}
	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		ID    json.RawMessage `json:"id"`
		Error *struct {
			Code int64 `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(bytes.TrimSpace([]byte(strings.TrimPrefix(line, "data: "))), &payload); err != nil {
		t.Fatal(err)
	}
	if string(payload.ID) != id {
		t.Fatalf("SSE id = %s, want %s", payload.ID, id)
	}
	if code == 0 && payload.Error != nil {
		t.Fatalf("respuesta inesperada de error: %+v", payload.Error)
	}
	if code != 0 && (payload.Error == nil || payload.Error.Code != code) {
		t.Fatalf("error SSE = %+v, want %d", payload.Error, code)
	}
	if _, err := reader.ReadString('\n'); err != nil {
		t.Fatal(err)
	}
}
