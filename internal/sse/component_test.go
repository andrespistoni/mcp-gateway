package sse

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"mcp-gateway/internal/endpoint"
	"mcp-gateway/internal/mcp"
	"mcp-gateway/internal/project"
)

type testCaller struct{}

func (testCaller) TryCall(context.Context, mcp.RawID, json.RawMessage, project.OptionalDir) (<-chan mcp.Envelope, func(), bool) {
	return nil, func() {}, false
}

type fakeListener struct {
	address net.Addr
	closed  bool
}

func (l *fakeListener) Accept() (net.Conn, error) { return nil, net.ErrClosed }
func (l *fakeListener) Close() error              { l.closed = true; return nil }
func (l *fakeListener) Addr() net.Addr            { return l.address }

type fakeAddr string

func (a fakeAddr) Network() string { return "fake" }
func (a fakeAddr) String() string  { return string(a) }

type errorReader struct{ err error }

func (r errorReader) Read([]byte) (int, error) { return 0, r.err }

func TestProjectExtractionAndRawQueryValues(t *testing.T) {
	directory := t.TempDir()
	request := httptest.NewRequest(http.MethodGet, "/sse?projectDir="+url.QueryEscape(directory), nil)
	result, err := fromRequestProject(request)
	if err != nil || !result.Present() || result.Path() != directory {
		t.Fatalf("query project = %#v, %v", result, err)
	}

	request = httptest.NewRequest(http.MethodGet, "/sse", nil)
	request.Header.Set("X-Project-Dir", directory)
	result, err = fromRequestProject(request)
	if err != nil || !result.Present() {
		t.Fatalf("header project = %#v, %v", result, err)
	}

	request = httptest.NewRequest(http.MethodGet, "/sse?projectDir=a;projectDir=b", nil)
	if _, err := fromRequestProject(request); err == nil {
		t.Fatal("projectDir duplicado debía fallar")
	}
	request = httptest.NewRequest(http.MethodGet, "/sse", nil)
	request.Header.Add("X-Project-Dir", "a")
	request.Header.Add("X-Project-Dir", "b")
	if _, err := fromRequestProject(request); err == nil {
		t.Fatal("header projectDir duplicado debía fallar")
	}
	if values := rawQueryValues("x=1&projectDir=a;projectDir=b&empty", "projectDir"); len(values) != 2 {
		t.Fatalf("rawQueryValues = %#v", values)
	}
}

func TestEventsErrorsAndDependencies(t *testing.T) {
	var output bytes.Buffer
	if err := writeHeartbeat(&output); err != nil || output.String() != ": heartbeat\n\n" {
		t.Fatalf("heartbeat = %q, %v", output.String(), err)
	}
	if (&protocolError{message: "bad"}).Error() != "bad" {
		t.Fatal("protocolError inválido")
	}
	if (Dependencies{}).entropy() == nil || (Dependencies{Entropy: testEntropy{1}}).entropy() == nil {
		t.Fatal("entropy ausente")
	}

	recorder := httptest.NewRecorder()
	id := mcp.NumberID(7)
	writeHTTPRPCErrorID(recorder, http.StatusBadRequest, -32600, "bad", &id)
	if recorder.Code != http.StatusBadRequest || recorder.Header().Get("Content-Type") != "application/json" ||
		!bytes.Contains(recorder.Body.Bytes(), []byte(`"id":7`)) {
		t.Fatalf("rpc error = %d %q", recorder.Code, recorder.Body.String())
	}
}

func TestServerCompositionGuards(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Bind(ctx, endpoint.MustPort(3333), Dependencies{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Bind cancelado = %v", err)
	}
	if _, err := bindListener(endpoint.MustPort(3333), nil, Dependencies{}); err == nil {
		t.Fatal("listener nil debía fallar")
	}
	invalid := &fakeListener{address: fakeAddr("remote")}
	if _, err := bindListener(endpoint.MustPort(3333), invalid, Dependencies{}); err == nil || !invalid.closed {
		t.Fatalf("listener inválido = %v, closed=%v", err, invalid.closed)
	}

	server := &Server{registry: newRegistry(), deps: Dependencies{Entropy: testEntropy{2}}}
	if err := server.SetCaller(nil); err == nil {
		t.Fatal("caller nil debía fallar")
	}
	if err := server.SetCaller(testCaller{}); err != nil {
		t.Fatal(err)
	}
	if err := server.SetCatalog(nil); err == nil {
		t.Fatal("catalog nil debía fallar")
	}
	if err := server.SetCatalog(testCatalog{}); err != nil {
		t.Fatal(err)
	}
	if err := server.SetCatalog(testCatalog{}); err == nil {
		t.Fatal("catálogo duplicado debía fallar")
	}
	if err := server.SetCaller(testCaller{}); err == nil {
		t.Fatal("caller posterior a ready debía fallar")
	}
	badEntropy := &Server{registry: newRegistry(), deps: Dependencies{Entropy: errorReader{err: io.ErrUnexpectedEOF}}}
	if err := badEntropy.SetCatalog(testCatalog{}); err == nil {
		t.Fatal("catálogo debía fallar con entropy inválida")
	}
	if _, err := NewSessionID(errorReader{err: io.ErrUnexpectedEOF}); err == nil {
		t.Fatal("entropy fallida debía impedir SessionID")
	}
}

func TestServerPortAndGracefulShutdown(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port, err := endpoint.NewPort(listener.Addr().(*net.TCPAddr).Port)
	if err != nil {
		t.Fatal(err)
	}
	server, err := bindListener(port, listener, Dependencies{Entropy: testEntropy{3}})
	if err != nil {
		t.Fatal(err)
	}
	if server.Port() != port {
		t.Fatalf("Port = %v", server.Port())
	}
	done := make(chan error, 1)
	go func() { done <- server.Serve() }()
	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
