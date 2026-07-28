package sse

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"mcp-gateway/internal/endpoint"
	"mcp-gateway/internal/mcp"
	"mcp-gateway/internal/project"
)

type callStub struct {
	result   chan mcp.Envelope
	admitted bool
}

func (s callStub) TryCall(context.Context, mcp.RawID, json.RawMessage, project.OptionalDir) (<-chan mcp.Envelope, func(), bool) {
	return s.result, func() {}, s.admitted
}

func TestBeginCallRejectsSeventeenthPendingBeforeProxy(t *testing.T) {
	t.Parallel()
	session := newSession(SessionID{}, project.OptionalDir{})
	for i := 0; i < maxPendingCalls; i++ {
		session.pending[uint64(i+1)] = pendingCall{cancel: func() {}}
	}
	command := submission{ctx: context.Background(), result: make(chan submissionResult, 1)}
	envelope, _ := mcp.NewRequest(mcp.NumberID(1), "tools/call", map[string]any{"name": "x"})
	(&Server{deps: Dependencies{Caller: callStub{admitted: true}}}).beginCall(session, command, mcp.NumberID(1), envelope)
	select {
	case result := <-command.result:
		if result.status != 429 || !result.httpError || result.rpcCode != 0 {
			t.Fatalf("unexpected result: %+v", result)
		}
	case <-time.After(time.Second):
		t.Fatal("pending limit did not reply")
	}
}

func TestSessionCloseCancelsAndRepliesToPending(t *testing.T) {
	t.Parallel()
	session := newSession(SessionID{}, project.OptionalDir{})
	cancelled := make(chan struct{})
	command := submission{result: make(chan submissionResult, 1)}
	session.pending[1] = pendingCall{command: command, cancel: func() { close(cancelled) }}
	session.close()
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("pending call was not cancelled")
	}
	select {
	case result := <-command.result:
		if result.status != 504 || result.rpcCode != -32003 {
			t.Fatalf("unexpected result: %+v", result)
		}
	case <-time.After(time.Second):
		t.Fatal("pending post was not released")
	}
}

func TestToolsCallDeliversResultBeforeHTTPAccepted(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	portNumber := listener.Addr().(*net.TCPAddr).Port
	port, err := endpoint.NewPort(portNumber)
	if err != nil {
		t.Fatal(err)
	}
	tool, _ := mcp.ParseTool(json.RawMessage(`{"name":"p__echo"}`))
	server, err := bindListener(port, listener, Dependencies{Entropy: testEntropy{9}})
	if err != nil {
		t.Fatal(err)
	}
	response := make(chan mcp.Envelope, 1)
	serverCaller := callStub{result: response, admitted: true}
	if err := server.SetCaller(serverCaller); err != nil {
		t.Fatal(err)
	}
	if err := server.SetCatalog(testCatalog{tools: []mcp.Tool{tool}}); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- server.Serve() }()
	t.Cleanup(func() {
		_ = server.Close()
		select {
		case err := <-done:
			if err != nil {
				t.Error(err)
			}
		case <-time.After(time.Second):
			t.Error("Serve no terminó")
		}
	})

	stream, err := http.Get(fmt.Sprintf("http://localhost:%d/sse", portNumber))
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Body.Close()
	reader := bufio.NewReader(stream.Body)
	if line, _ := reader.ReadString('\n'); line != "event: endpoint\n" {
		t.Fatalf("endpoint event = %q", line)
	}
	data, err := reader.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	endpointPath := strings.TrimSpace(strings.TrimPrefix(data, "data: "))
	_, _ = reader.ReadString('\n')
	post := func(payload string) *http.Response {
		request, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://localhost:%d%s", portNumber, endpointPath), strings.NewReader(payload))
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Content-Type", "application/json")
		result, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	initialize := post(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}`)
	if initialize.StatusCode != http.StatusAccepted {
		t.Fatalf("initialize status = %d", initialize.StatusCode)
	}
	_ = initialize.Body.Close()
	assertMessage(t, reader, "initialize")
	initialized := post(`{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	if initialized.StatusCode != http.StatusAccepted {
		t.Fatalf("initialized status = %d", initialized.StatusCode)
	}
	_ = initialized.Body.Close()

	id := mcp.NumberID(3)
	callResult, _ := mcp.NewResult(id, map[string]any{"content": []any{}})
	response <- callResult
	called := post(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"p__echo","arguments":{}}}`)
	defer called.Body.Close()
	if called.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(called.Body)
		t.Fatalf("tools/call status = %d, body=%s", called.StatusCode, body)
	}
	assertMessage(t, reader, "tools/call")
}
