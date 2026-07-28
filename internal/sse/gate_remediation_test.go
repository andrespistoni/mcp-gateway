package sse

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

func TestGatePartialHeadsDoNotBlockLaterConnections(t *testing.T) {
	_, port := startMatrixServer(t)
	address := fmt.Sprintf("127.0.0.1:%d", port.Number())
	host := fmt.Sprintf("Host: localhost:%d\r\n", port.Number())

	for _, test := range []struct {
		name    string
		partial string
	}{
		{name: "request-line-without-CRLF", partial: "GET / HTTP/1.1\r"},
		{name: "header-without-CRLF", partial: "GET / HTTP/1.1\r\n" + host + "X-Slow: value\r"},
	} {
		t.Run(test.name, func(t *testing.T) {
			connection, err := net.Dial("tcp", address)
			if err != nil {
				t.Fatal(err)
			}
			defer connection.Close()
			if _, err := connection.Write([]byte(test.partial)); err != nil {
				t.Fatal(err)
			}

			assertDynamicTCPStatus(t, address, "GET /later HTTP/1.1\r\n"+host+"\r\n", 404)

			if err := connection.SetReadDeadline(time.Now().Add(gateHeadTimeout + time.Second)); err != nil {
				t.Fatal(err)
			}
			line, err := bufio.NewReader(connection).ReadString('\n')
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(line, " 400 ") {
				t.Fatalf("respuesta parcial = %q, want 400", line)
			}
		})
	}
}

func TestGateRejectsOverLimitLinesWithoutLF(t *testing.T) {
	_, port := startMatrixServer(t)
	address := fmt.Sprintf("127.0.0.1:%d", port.Number())
	host := fmt.Sprintf("Host: localhost:%d\r\n", port.Number())

	assertDynamicTCPStatus(t, address, "GET /"+strings.Repeat("x", maxRequestTarget)+" HTTP/1.1", 414)
	assertDynamicTCPStatus(t, address, "GET / HTTP/1.1\r\n"+host+"X-Fill: "+strings.Repeat("x", maxHeaderBytes), 431)
}

func TestSlowPOSTBodyTimesOutWithoutClosingSSE(t *testing.T) {
	_, port := startMatrixServer(t)
	address := fmt.Sprintf("127.0.0.1:%d", port.Number())
	host := fmt.Sprintf("Host: localhost:%d\r\n", port.Number())

	stream, reader, endpointPath := openMatrixSession(t, fmt.Sprintf("http://localhost:%d", port.Number()))
	defer stream.Close()

	connection, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	bodyLength := 64
	head := fmt.Sprintf("POST /message?sessionId=%s HTTP/1.1\r\n%sContent-Type: application/json\r\nContent-Length: %d\r\n\r\n{", validMatrixSessionID(), host, bodyLength)
	if _, err := connection.Write([]byte(head)); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		time.Sleep(time.Second)
		if _, err := connection.Write([]byte(" ")); err != nil {
			t.Fatal(err)
		}
	}
	if err := connection.SetReadDeadline(time.Now().Add(bodyReadTimeout)); err != nil {
		t.Fatal(err)
	}
	line, err := bufio.NewReader(connection).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(line, " 400 ") {
		t.Fatalf("respuesta body lento = %q, want 400", line)
	}

	// The body deadline is per POST connection. A pre-existing SSE stream
	// remains usable after that interval.
	time.Sleep(time.Second)
	response := matrixPOST(t, fmt.Sprintf("http://localhost:%d%s", port.Number(), endpointPath), "application/json", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}`))
	defer response.Body.Close()
	if response.StatusCode != 202 {
		t.Fatalf("initialize after body timeout = %d, want 202", response.StatusCode)
	}
	assertMatrixSSEErrorOrResult(t, reader, "1", 0)

	assertDynamicTCPStatus(t, address, "GET /after-timeout HTTP/1.1\r\n"+host+"\r\n", 404)
}

func assertDynamicTCPStatus(t *testing.T, address, request string, status int) {
	t.Helper()
	connection, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	if err := connection.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := connection.Write([]byte(request)); err != nil {
		t.Fatal(err)
	}
	line, err := bufio.NewReader(connection).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(line, fmt.Sprintf(" %d ", status)) {
		t.Fatalf("respuesta = %q, want status %d", line, status)
	}
}
