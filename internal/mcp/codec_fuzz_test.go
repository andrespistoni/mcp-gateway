package mcp

import (
	"bytes"
	"testing"
)

func FuzzCodec(f *testing.F) {
	f.Add([]byte(`{"jsonrpc":"2.0","method":"ping"}`), false)
	f.Add([]byte(`[]`), true)
	f.Fuzz(func(t *testing.T, payload []byte, crlf bool) {
		if len(payload) > MaxMessageSize+1 {
			return
		}
		delimiter := []byte{'\n'}
		if crlf {
			delimiter = []byte{'\r', '\n'}
		}
		input := append(append([]byte(nil), payload...), delimiter...)
		envelope, err := NewCodec(bytes.NewReader(input), nil).Read()
		if err != nil {
			return
		}
		var output bytes.Buffer
		if err := NewCodec(nil, &output).Write(envelope); err != nil {
			t.Fatal(err)
		}
		if output.Len() > MaxMessageSize+1 || output.Bytes()[output.Len()-1] != '\n' {
			t.Fatal("writer produjo framing inválido")
		}
	})
}
