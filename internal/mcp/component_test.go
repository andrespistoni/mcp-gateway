package mcp

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
)

type failingWriter struct {
	written int
	err     error
	zero    bool
}

func (w *failingWriter) Write(data []byte) (int, error) {
	if w.zero {
		return 0, nil
	}
	if w.err != nil {
		return 0, w.err
	}
	limit := len(data)
	if limit > 3 {
		limit = 3
	}
	w.written += limit
	return limit, nil
}

func TestOutboundEnvelopeConstructorsAndAccessors(t *testing.T) {
	stringID := StringID("request")
	numberID := NumberID(42)
	if string(stringID.Bytes()) != `"request"` || string(numberID.Bytes()) != "42" || stringID.Equal(numberID) {
		t.Fatal("constructores de ID inválidos")
	}
	copyID := stringID.Bytes()
	copyID[0] = 'x'
	if string(stringID.Bytes()) != `"request"` {
		t.Fatal("Bytes expuso memoria interna")
	}
	if _, err := json.Marshal(RawID{}); err == nil {
		t.Fatal("RawID vacío debía fallar")
	}

	request, err := NewRequest(stringID, "tools/call", map[string]any{"name": "demo"})
	if err != nil {
		t.Fatal(err)
	}
	id, ok := request.ID()
	if !ok || !id.Equal(stringID) || request.Kind() != Request || request.Method() != "tools/call" {
		t.Fatalf("request inválido: %#v", request)
	}
	if !strings.Contains(string(request.Params()), `"demo"`) {
		t.Fatalf("params = %s", request.Params())
	}
	fields := request.Fields()
	fields["method"][0] = 'x'
	if request.Method() != "tools/call" {
		t.Fatal("Fields expuso memoria interna")
	}

	notification, err := NewNotification("notifications/initialized", nil)
	if err != nil || notification.Kind() != Notification {
		t.Fatalf("notification = %#v, %v", notification, err)
	}
	if _, hasID := notification.ID(); hasID || notification.Params() != nil {
		t.Fatal("notification no debía contener id o params")
	}

	result, err := NewResult(numberID, map[string]bool{"ok": true})
	if err != nil || result.Kind() != Result || string(result.Result()) != `{"ok":true}` {
		t.Fatalf("result = %#v, %v", result, err)
	}
	resultCopy := result.Result()
	resultCopy[0] = '['
	if string(result.Result()) != `{"ok":true}` {
		t.Fatal("Result expuso memoria interna")
	}

	rpcInput := RPCError{
		Code: -32000, Message: "fallo", Data: json.RawMessage(`{"retry":false}`),
		fields: map[string]json.RawMessage{"future": json.RawMessage(`true`)},
	}
	failure, err := NewError(numberID, rpcInput)
	if err != nil || failure.Kind() != Error {
		t.Fatalf("error envelope = %#v, %v", failure, err)
	}
	rpcError, ok := failure.RPCError()
	if !ok || rpcError.Code != -32000 || rpcError.Message != "fallo" ||
		string(rpcError.Data) != `{"retry":false}` || string(rpcError.Fields()["future"]) != "true" {
		t.Fatalf("RPCError = %#v", rpcError)
	}
	rpcError.Data[0] = '['
	if original, _ := failure.RPCError(); string(original.Data) != `{"retry":false}` {
		t.Fatal("RPCError expuso memoria interna")
	}
	if _, ok := request.RPCError(); ok {
		t.Fatal("request no debía contener RPCError")
	}
}

func TestOutboundEnvelopeRejectsInvalidValues(t *testing.T) {
	invalidID := RawID{}
	if _, err := NewRequest(invalidID, "ping", nil); err == nil {
		t.Fatal("request aceptó ID inválido")
	}
	if _, err := NewRequest(NumberID(1), "", nil); err == nil {
		t.Fatal("request aceptó method vacío")
	}
	if _, err := NewNotification("", nil); err == nil {
		t.Fatal("notification aceptó method vacío")
	}
	unsupported := func() {}
	if _, err := NewRequest(NumberID(1), "x", unsupported); err == nil {
		t.Fatal("request aceptó params no serializables")
	}
	if _, err := NewResult(NumberID(1), unsupported); err == nil {
		t.Fatal("result aceptó valor no serializable")
	}
	if _, err := (Envelope{}).MarshalJSON(); err == nil {
		t.Fatal("envelope vacío debía fallar")
	}
}

func TestEnvelopeRejectsRemainingInvalidShapes(t *testing.T) {
	invalid := [][]byte{
		{0xff},
		[]byte(`{"jsonrpc":"2.0","method":"x"} {}`),
		[]byte(`{"jsonrpc":"2.0","method":1}`),
		[]byte(`{"jsonrpc":"2.0","method":"x","result":{}}`),
		[]byte(`{"jsonrpc":"2.0","method":"x","error":{}}`),
		[]byte(`{"jsonrpc":"2.0","result":{}}`),
		[]byte(`{"jsonrpc":"2.0","id":1,"error":[]}`),
		[]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":"x","message":"m"}}`),
		[]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":1.5,"message":"m"}}`),
		[]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":1,"message":7}}`),
	}
	for _, input := range invalid {
		if _, err := ParseEnvelope(input); err == nil {
			t.Errorf("se aceptó %q", input)
		}
	}
	for _, input := range []json.RawMessage{nil, json.RawMessage(`x`), json.RawMessage(`null`)} {
		if _, err := ParseID(input); err == nil {
			t.Errorf("se aceptó ID %q", input)
		}
	}
}

func TestCodecWriterAndReaderFailures(t *testing.T) {
	envelope, err := NewNotification("ping", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := NewCodec(nil, nil).Write(envelope); err == nil {
		t.Fatal("writer ausente debía fallar")
	}
	sentinel := errors.New("write failed")
	if err := NewCodec(nil, &failingWriter{err: sentinel}).Write(envelope); !errors.Is(err, sentinel) {
		t.Fatalf("error de writer = %v", err)
	}
	if err := NewCodec(nil, &failingWriter{zero: true}).Write(envelope); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("short write = %v", err)
	}
	fragmented := &failingWriter{}
	if err := NewCodec(nil, fragmented).Write(envelope); err != nil || fragmented.written == 0 {
		t.Fatalf("escritura fragmentada = %d, %v", fragmented.written, err)
	}

	overLimit := strings.Repeat("x", MaxMessageSize+3) + "\n"
	if _, err := NewCodec(strings.NewReader(overLimit), nil).Read(); err == nil {
		t.Fatal("línea sobredimensionada debía fallar")
	}
}

func TestToolValidationAndObjectDecoding(t *testing.T) {
	for _, raw := range []string{`null`, `[]`, `{}`, `{"name":1}`, `{"name":""}`} {
		if _, err := ParseTool(json.RawMessage(raw)); err == nil {
			t.Errorf("ParseTool aceptó %s", raw)
		}
	}
	tool, err := ParseTool(json.RawMessage(`{"name":"demo","inputSchema":{"type":"object"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tool.WithName(""); err == nil {
		t.Fatal("WithName aceptó nombre vacío")
	}
	encoded, err := json.Marshal(tool)
	if err != nil || !bytes.Contains(encoded, []byte(`"demo"`)) {
		t.Fatalf("tool JSON = %s, %v", encoded, err)
	}

	object, err := DecodeObject(json.RawMessage(`{"a":1}`))
	if err != nil || string(object["a"]) != "1" {
		t.Fatalf("DecodeObject = %#v, %v", object, err)
	}
	for _, raw := range []string{`null`, `[]`, `{invalid}`} {
		if _, err := DecodeObject(json.RawMessage(raw)); err == nil {
			t.Errorf("DecodeObject aceptó %s", raw)
		}
	}
}
