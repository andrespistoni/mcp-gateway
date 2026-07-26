package config

import "testing"

func FuzzDecode(f *testing.F) {
	f.Add([]byte("version: 1\ndownstreams: []\n"))
	f.Add([]byte("version: 1\nunknown: true\n"))
	f.Add([]byte("version: 1\nversion: 1\n"))
	f.Fuzz(func(t *testing.T, data []byte) {
		snapshot, err := Decode(data)
		if err != nil {
			return
		}
		document := snapshot.Document()
		if err := Validate(&document); err != nil {
			t.Fatalf("Decode aceptó un documento inválido: %v", err)
		}
	})
}
