package sse

import (
	"bufio"
	"bytes"
	"net/http"
	"strings"
	"testing"
)

// Accepted heads must retain the HTTP interpretation that net/http receives
// after the gate replays the original bytes.
func FuzzGateDifferential(f *testing.F) {
	f.Add("GET /sse HTTP/1.1\r\nHost: localhost:4444\r\n\r\n")
	f.Add("POST /message?sessionId=x HTTP/1.1\r\nHost: localhost:4444\r\nContent-Length: 2\r\n\r\n{}")
	f.Fuzz(func(t *testing.T, input string) {
		reader := bufio.NewReader(strings.NewReader(input))
		head, err := parseRequestHead(reader)
		if err != nil {
			return
		}
		request, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(head.raw)))
		if err != nil {
			t.Fatalf("el gate aceptó un head que net/http rechazó: %q: %v", input, err)
		}
		if request.Host == "" || request.Method == "" || request.RequestURI == "" {
			t.Fatalf("interpretación incompleta: %#v", request)
		}
	})
}
