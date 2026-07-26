package mcp

import (
	"bytes"
	"strings"
	"testing"
)

type fragmentedReader struct{ data []byte }

func (r *fragmentedReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, nil
	}
	p[0] = r.data[0]
	r.data = r.data[1:]
	return 1, nil
}

func TestCodecAcceptsLFCRLFAndWritesLF(t *testing.T) {
	input := []byte("{\"jsonrpc\":\"2.0\",\"method\":\"a\"}\r\n{\"jsonrpc\":\"2.0\",\"method\":\"b\"}\n")
	var output bytes.Buffer
	codec := NewCodec(&fragmentedReader{data: input}, &output)
	for _, method := range []string{"a", "b"} {
		envelope, err := codec.Read()
		if err != nil || envelope.Method() != method {
			t.Fatalf("Read = %q, %v", envelope.Method(), err)
		}
		if err := codec.Write(envelope); err != nil {
			t.Fatal(err)
		}
	}
	if strings.Contains(output.String(), "\r") || strings.Count(output.String(), "\n") != 2 {
		t.Fatalf("framing de salida = %q", output.String())
	}
}

func TestCodecLimitsAndInvalidInput(t *testing.T) {
	padding := MaxMessageSize - len(`{"jsonrpc":"2.0","method":"x","padding":""}`)
	exact := `{"jsonrpc":"2.0","method":"x","padding":"` + strings.Repeat("a", padding) + `"}`
	if len(exact) != MaxMessageSize {
		t.Fatalf("fixture = %d", len(exact))
	}
	if _, err := NewCodec(strings.NewReader(exact+"\n"), nil).Read(); err != nil {
		t.Fatalf("límite exacto rechazado: %v", err)
	}
	invalid := []string{
		exact + "x\n", "\n", "[]\n", "{\"jsonrpc\":\"2.0\",\"method\":\"x\"} trailing\n",
		"{\"jsonrpc\":\"2.0\",\"method\":\"x\"}", string([]byte{0xff, '\n'}), "",
	}
	for _, input := range invalid {
		if _, err := NewCodec(strings.NewReader(input), nil).Read(); err == nil {
			t.Errorf("se aceptó entrada inválida de %d bytes", len(input))
		}
	}
}
