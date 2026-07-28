package sse

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"testing"
)

func TestGateLimitBoundariesOverDynamicTCP(t *testing.T) {
	_, port := startMatrixServer(t)
	address := fmt.Sprintf("127.0.0.1:%d", port.Number())
	host := fmt.Sprintf("Host: localhost:%d\r\n", port.Number())

	for _, test := range []struct {
		name    string
		request string
		status  int
	}{
		{"target-at-limit", "GET /" + strings.Repeat("a", maxRequestTarget-1) + " HTTP/1.1\r\n" + host + "\r\n", 404},
		{"target-over-limit", "GET /" + strings.Repeat("a", maxRequestTarget) + " HTTP/1.1\r\n" + host + "\r\n", 414},
		{"headers-at-limit", "GET / HTTP/1.1\r\n" + matrixHeaders(host, maxHeaderBytes) + "\r\n", 404},
		{"headers-over-limit", "GET / HTTP/1.1\r\n" + matrixHeaders(host, maxHeaderBytes+1) + "\r\n", 431},
	} {
		t.Run(test.name, func(t *testing.T) {
			connection, err := net.Dial("tcp", address)
			if err != nil {
				t.Fatal(err)
			}
			defer connection.Close()
			if _, err := connection.Write([]byte(test.request)); err != nil {
				t.Fatal(err)
			}
			response, err := bufio.NewReader(connection).ReadString('\n')
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(response, fmt.Sprintf(" %d ", test.status)) {
				t.Fatalf("response = %q, want status %d", response, test.status)
			}
		})
	}
}

func matrixHeaders(host string, total int) string {
	const prefix = "X-Fill: "
	const suffix = "\r\n"
	fill := total - len(host) - len(prefix) - len(suffix)
	return host + prefix + strings.Repeat("x", fill) + suffix
}
