package cli

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"mcp-gateway/internal/app"
	"mcp-gateway/internal/discovery"
	"mcp-gateway/internal/endpoint"
)

type serveRecorder struct {
	port *endpoint.Port
	err  error
}

func (s *serveRecorder) Serve(_ context.Context, port *endpoint.Port) error {
	s.port = port
	return s.err
}

func TestDiscoverOutputStatesAndAttempts(t *testing.T) {
	application := &workflowApplication{discovery: app.DiscoveryResult{
		Attempts: []discovery.Attempt{
			{Recipe: "ignored", Candidate: "ok"},
			{Recipe: "recipe", Candidate: "missing", Failure: "no disponible"},
		},
		Items: []app.DiscoveryItem{
			{Name: "added", Path: "/a", Added: true},
			{Name: "kept", Path: "/b"},
		},
	}}
	var output bytes.Buffer
	if err := writeDiscover(context.Background(), &output, application, true); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"recipe\tmissing\tno disponible",
		"added\tañadido\t/a",
		"kept\tconservado\t/b",
	} {
		if !bytes.Contains(output.Bytes(), []byte(expected)) {
			t.Fatalf("output no contiene %q: %q", expected, output.String())
		}
	}

	output.Reset()
	if err := writeDiscover(context.Background(), &output, application, false); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(output.Bytes(), []byte("added\tencontrado\t/a")) {
		t.Fatalf("read-only output = %q", output.String())
	}
}

func TestServeDispatchAndErrors(t *testing.T) {
	port := endpoint.MustPort(4444)
	recorder := &serveRecorder{}
	if err := runServe(context.Background(), recorder, parsedCommand{port: &port}); err != nil || recorder.port != &port {
		t.Fatalf("runServe = %v, %#v", err, recorder.port)
	}
	recorder.err = errors.New("serve failed")
	if err := runServe(context.Background(), recorder, parsedCommand{}); !errors.Is(err, recorder.err) {
		t.Fatalf("runServe error = %v", err)
	}
}
