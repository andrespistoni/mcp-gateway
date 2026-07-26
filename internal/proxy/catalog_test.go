package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"mcp-gateway/internal/config"
	"mcp-gateway/internal/mcp"
)

func TestCatalogPreservesOrderFieldsAndDefensiveCopies(t *testing.T) {
	first := testTool(t, "first", 1)
	second := testTool(t, "second", 2)
	snapshot, err := buildCatalog([]catalogEntry{
		{config: config.Downstream{Name: "one", Prefix: "one__", InjectProject: true, ProjectArgument: "projectPath"}, tools: []mcp.Tool{first}},
		{config: config.Downstream{Name: "two", Prefix: "two__"}, tools: []mcp.Tool{second}},
	}, bytes.NewReader(bytes.Repeat([]byte{1}, 32)))
	if err != nil {
		t.Fatal(err)
	}
	tools := snapshot.Tools()
	if len(tools) != 2 || tools[0].Name() != "one__first" || tools[1].Name() != "two__second" {
		t.Fatalf("orden inesperado: %#v", tools)
	}
	fields := tools[0].Fields()
	if string(fields["future"]) != `{"order":1}` || string(fields["description"]) != `"fake"` {
		t.Fatalf("campos no preservados: %s", mustJSON(t, tools[0]))
	}
	fields["description"] = json.RawMessage(`"mutated"`)
	if string(snapshot.Tools()[0].Fields()["description"]) != `"fake"` {
		t.Fatal("el catálogo fue mutado mediante una copia")
	}
	route, ok := snapshot.Route("one__first")
	if !ok || route.Downstream != "one" || route.OriginalName != "first" || !route.InjectProject {
		t.Fatalf("route = %#v", route)
	}
}

func TestCatalogCollisionEmptyAndLimits(t *testing.T) {
	empty, err := buildCatalog(nil, bytes.NewReader(make([]byte, 32)))
	if err != nil || len(empty.Tools()) != 0 {
		t.Fatalf("catálogo vacío: %v, %d", err, len(empty.Tools()))
	}
	collision := []catalogEntry{
		{config: config.Downstream{Name: "short", Prefix: "a__"}, tools: []mcp.Tool{testTool(t, "b__echo", 1)}},
		{config: config.Downstream{Name: "long", Prefix: "a__b__"}, tools: []mcp.Tool{testTool(t, "echo", 2)}},
	}
	if _, err := buildCatalog(collision, bytes.NewReader(make([]byte, 32))); err == nil {
		t.Fatal("la colisión global debía fallar")
	}
	tools := make([]mcp.Tool, MaxCatalogTools)
	for index := range tools {
		tools[index] = testTool(t, fmt.Sprintf("tool-%d", index), index)
	}
	entry := catalogEntry{config: config.Downstream{Name: "max", Prefix: "max__"}, tools: tools}
	if snapshot, err := buildCatalog([]catalogEntry{entry}, bytes.NewReader(make([]byte, 32))); err != nil || len(snapshot.Tools()) != MaxCatalogTools {
		t.Fatalf("límite inclusivo: tools=%d err=%v", len(snapshot.Tools()), err)
	}
	entry.tools = append(entry.tools, testTool(t, "overflow", MaxCatalogTools))
	if _, err := buildCatalog([]catalogEntry{entry}, bytes.NewReader(make([]byte, 32))); err == nil {
		t.Fatal("10001 tools debía fallar")
	}
}

func testTool(t *testing.T, name string, order int) mcp.Tool {
	t.Helper()
	raw, _ := json.Marshal(map[string]any{"name": name, "description": "fake", "future": map[string]any{"order": order}})
	tool, err := mcp.ParseTool(raw)
	if err != nil {
		t.Fatal(err)
	}
	return tool
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
