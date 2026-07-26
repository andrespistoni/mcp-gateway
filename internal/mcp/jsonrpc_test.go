package mcp

import (
	"encoding/json"
	"testing"
)

func TestEnvelopeKindsIDsAndPreservation(t *testing.T) {
	tests := []struct {
		input string
		kind  EnvelopeKind
	}{
		{`{"jsonrpc":"2.0","id":"01","method":"ping","extra":{"x":1}}`, Request},
		{`{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}`, Notification},
		{`{"jsonrpc":"2.0","id":1.0,"result":{"ok":true},"extra":7}`, Result},
		{`{"jsonrpc":"2.0","id":2,"error":{"code":-1,"message":"fallo","data":{"x":1},"future":true}}`, Error},
	}
	for _, test := range tests {
		envelope, err := ParseEnvelope([]byte(test.input))
		if err != nil || envelope.Kind() != test.kind {
			t.Fatalf("ParseEnvelope(%s) = %v, %v", test.input, envelope.Kind(), err)
		}
		encoded, err := json.Marshal(envelope)
		if err != nil {
			t.Fatal(err)
		}
		reparsed, err := ParseEnvelope(encoded)
		if err != nil || reparsed.Kind() != test.kind {
			t.Fatalf("roundtrip = %s, %v", encoded, err)
		}
	}
}

func TestEnvelopeRejectsInvalidShapesAndIDs(t *testing.T) {
	invalid := []string{
		`[]`, `{}`, `{"jsonrpc":"1.0","method":"x"}`, `{"jsonrpc":"2.0","method":""}`,
		`{"jsonrpc":"2.0","method":"x","id":null}`, `{"jsonrpc":"2.0","method":"x","id":true}`,
		`{"jsonrpc":"2.0","method":"x","id":{}}`, `{"jsonrpc":"2.0","method":"x","id":[]}`,
		`{"jsonrpc":"2.0","id":1}`, `{"jsonrpc":"2.0","id":1,"result":{},"error":{}}`,
	}
	for _, input := range invalid {
		if _, err := ParseEnvelope([]byte(input)); err == nil {
			t.Errorf("se aceptó %s", input)
		}
	}
}

func TestToolChangesOnlyName(t *testing.T) {
	tool, err := ParseTool(json.RawMessage(`{"name":"old","description":"d","inputSchema":{"type":"object"},"future":[1]}`))
	if err != nil {
		t.Fatal(err)
	}
	prefixed, err := tool.WithName("p__old")
	if err != nil {
		t.Fatal(err)
	}
	fields := prefixed.Fields()
	if prefixed.Name() != "p__old" || string(fields["future"]) != `[1]` || string(fields["description"]) != `"d"` {
		t.Fatalf("tool no preservada: %#v", fields)
	}
}
