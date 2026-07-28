package sse

import "testing"

func FuzzCursor(f *testing.F) {
	codec, err := newCursorCodec([32]byte{1}, testEntropy{3})
	if err != nil {
		f.Fatal(err)
	}
	f.Add(codec.encode(100))
	f.Add("invalid")
	f.Fuzz(func(t *testing.T, value string) {
		_, _ = codec.decode(value, 101)
	})
}
