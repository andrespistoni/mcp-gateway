package sse

import (
	"bufio"
	"strings"
	"testing"
)

func FuzzGateParser(f *testing.F) {
	f.Add("GET / HTTP/1.1\r\nHost: localhost:4444\r\n\r\n")
	f.Add("POST /message HTTP/1.1\r\nHost: localhost:4444\r\nContent-Length: 0\r\n\r\n")
	f.Fuzz(func(t *testing.T, head string) {
		_, _ = parseRequestHead(bufio.NewReader(strings.NewReader(head)))
	})
}
