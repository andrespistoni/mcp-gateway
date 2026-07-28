package sse

import (
	"bufio"
	"strings"
	"testing"
)

func TestParseRequestHeadBoundaries(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name   string
		target int
		status int
	}{
		{name: "target-exacto", target: maxRequestTarget, status: 0},
		{name: "target-excesivo", target: maxRequestTarget + 1, status: 414},
	} {
		t.Run(test.name, func(t *testing.T) {
			head := "GET /" + strings.Repeat("x", test.target-1) + " HTTP/1.1\r\nHost: localhost:4444\r\n\r\n"
			_, err := parseRequestHead(bufio.NewReader(strings.NewReader(head)))
			if test.status == 0 && err != nil {
				t.Fatalf("parseRequestHead() error = %v", err)
			}
			if test.status != 0 {
				gate, ok := err.(*gateError)
				if !ok || gate.status != test.status {
					t.Fatalf("error = %#v, se esperaba status %d", err, test.status)
				}
			}
		})
	}
}

func TestParseRequestHeadRejectsAmbiguousFraming(t *testing.T) {
	t.Parallel()
	for _, head := range []string{
		"GET / HTTP/1.1\nHost: localhost:4444\n\n",
		"GET / HTTP/1.1\r\nHost: localhost:4444\r\nTransfer-Encoding: chunked\r\n\r\n",
		"GET / HTTP/1.1\r\nHost: localhost:4444\r\nContent-Length: 1\r\nContent-Length: 1\r\n\r\n",
		"GET / HTTP/1.1\r\nHost: localhost:4444\r\nBad Header: x\r\n\r\n",
	} {
		if _, err := parseRequestHead(bufio.NewReader(strings.NewReader(head))); err == nil {
			t.Fatalf("se aceptó head ambiguo: %q", head)
		}
	}
}
