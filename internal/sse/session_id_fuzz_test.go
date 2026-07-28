package sse

import "testing"

func TestSessionIDCanonical(t *testing.T) {
	t.Parallel()
	id, err := NewSessionID(testEntropy{1})
	if err != nil {
		t.Fatal(err)
	}
	if len(id.Encode()) != 43 {
		t.Fatalf("longitud = %d", len(id.Encode()))
	}
	parsed, err := ParseSessionID(id.Encode())
	if err != nil || parsed != id {
		t.Fatalf("ParseSessionID() = %v, %v", parsed, err)
	}
}

func FuzzSessionID(f *testing.F) {
	f.Add("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
	f.Fuzz(func(t *testing.T, value string) { _, _ = ParseSessionID(value) })
}

type testEntropy struct{ value byte }

func (r testEntropy) Read(data []byte) (int, error) {
	for index := range data {
		data[index] = r.value
	}
	return len(data), nil
}
