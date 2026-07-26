package mcp

import "testing"

func FuzzEnvelope(f *testing.F) {
	for _, seed := range []string{
		`{"jsonrpc":"2.0","id":1,"method":"ping"}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":"a","result":{}}`,
		`[]`,
	} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, input []byte) {
		envelope, err := ParseEnvelope(input)
		if err != nil {
			return
		}
		encoded, err := envelope.MarshalJSON()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ParseEnvelope(encoded); err != nil {
			t.Fatalf("envelope aceptado no reparsea: %v", err)
		}
	})
}
