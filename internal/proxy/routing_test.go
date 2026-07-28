package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"mcp-gateway/internal/config"
	"mcp-gateway/internal/mcp"
	"mcp-gateway/internal/project"
	"mcp-gateway/internal/testsupport/fakemcp"
)

func TestRouteCallRestoresExactNameAndInjectsOnlyWhenAbsent(t *testing.T) {
	t.Parallel()
	tool, err := mcp.ParseTool(json.RawMessage(`{"name":"alpha__echo","future":{"kept":true}}`))
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := buildCatalog([]catalogEntry{{config: config.Downstream{Name: "one", Prefix: "p__", InjectProject: true, ProjectArgument: "project"}, tools: []mcp.Tool{tool}}}, bytes.NewReader(make([]byte, 32)))
	if err != nil {
		t.Fatal(err)
	}
	directory, err := project.Resolve(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	service := &Service{catalog: snapshot}
	_, route, forwarded, err := service.routeCall(json.RawMessage(`{"name":"p__alpha__echo","arguments":{"x":1},"future":true}`), project.Some(directory))
	if err != nil {
		t.Fatal(err)
	}
	if route.OriginalName != "alpha__echo" {
		t.Fatalf("original name = %q", route.OriginalName)
	}
	fields, err := mcp.DecodeObject(forwarded)
	if err != nil {
		t.Fatal(err)
	}
	var name string
	if err := json.Unmarshal(fields["name"], &name); err != nil || name != "alpha__echo" {
		t.Fatalf("forwarded name = %q, err = %v", name, err)
	}
	arguments, err := mcp.DecodeObject(fields["arguments"])
	if err != nil {
		t.Fatal(err)
	}
	var injected string
	if err := json.Unmarshal(arguments["project"], &injected); err != nil || injected != directory.Path() {
		t.Fatalf("injected project = %q, err = %v", injected, err)
	}

	_, _, forwarded, err = service.routeCall(json.RawMessage(`{"name":"p__alpha__echo","arguments":{"project":null}}`), project.Some(directory))
	if err != nil {
		t.Fatal(err)
	}
	fields, _ = mcp.DecodeObject(forwarded)
	arguments, _ = mcp.DecodeObject(fields["arguments"])
	if string(arguments["project"]) != "null" {
		t.Fatalf("caller value was overwritten: %s", arguments["project"])
	}
}

func TestTryCallSerializesAndRestoresUpstreamID(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "fake-mcp")
	if err := fakemcp.Build(context.Background(), binary); err != nil {
		t.Fatal(err)
	}
	service, err := Start(context.Background(), []config.Downstream{{
		Name: "one", Prefix: "p__", Binary: binary, Args: []string{"--scenario=runtime-healthy"}, Enabled: true, Env: map[string]string{},
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = service.Shutdown(context.Background()) }()
	id, _ := mcp.ParseID(json.RawMessage(`"same-id"`))
	params := json.RawMessage(`{"name":"p__echo","arguments":{"value":1}}`)
	first, cancelFirst, admitted := service.TryCall(context.Background(), id, params, project.OptionalDir{})
	if !admitted {
		t.Fatal("first call was not admitted")
	}
	defer cancelFirst()
	second, cancelSecond, admitted := service.TryCall(context.Background(), id, params, project.OptionalDir{})
	if !admitted {
		t.Fatal("second call was not admitted")
	}
	defer cancelSecond()
	for _, result := range []<-chan mcp.Envelope{first, second} {
		select {
		case response := <-result:
			responseID, _ := response.ID()
			if !responseID.Equal(id) || response.Kind() != mcp.Result {
				t.Fatalf("response did not restore upstream ID: %#v", response)
			}
			var payload struct {
				Echo json.RawMessage `json:"echo"`
			}
			if err := json.Unmarshal(response.Result(), &payload); err != nil {
				t.Fatal(err)
			}
			fields, err := mcp.DecodeObject(payload.Echo)
			if err != nil {
				t.Fatal(err)
			}
			var name string
			if err := json.Unmarshal(fields["name"], &name); err != nil || name != "echo" {
				t.Fatalf("downstream name = %q, err = %v", name, err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("call did not complete")
		}
	}
}

func TestRouteCallDoesNotInjectNonObjectArguments(t *testing.T) {
	t.Parallel()
	tool, _ := mcp.ParseTool(json.RawMessage(`{"name":"echo"}`))
	snapshot, err := buildCatalog([]catalogEntry{{config: config.Downstream{Name: "one", Prefix: "p__", InjectProject: true, ProjectArgument: "project"}, tools: []mcp.Tool{tool}}}, bytes.NewReader(make([]byte, 32)))
	if err != nil {
		t.Fatal(err)
	}
	directory, _ := project.Resolve(t.TempDir())
	_, _, forwarded, err := (&Service{catalog: snapshot}).routeCall(json.RawMessage(`{"name":"p__echo","arguments":[1]}`), project.Some(directory))
	if err != nil {
		t.Fatal(err)
	}
	fields, _ := mcp.DecodeObject(forwarded)
	if string(fields["arguments"]) != "[1]" {
		t.Fatalf("arguments changed: %s", fields["arguments"])
	}
}
