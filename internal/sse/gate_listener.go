package sse

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	gateHeadTimeout        = 5 * time.Second
	gateRejectWriteTimeout = time.Second
	maxGateParsers         = 128
)

type gateListener struct {
	net.Listener

	accepted chan acceptedGateConn
	done     chan struct{}
	slots    chan struct{}

	mu      sync.Mutex
	pending map[net.Conn]struct{}
	closed  bool
	once    sync.Once
}

type acceptedGateConn struct {
	raw  net.Conn
	conn net.Conn
}

func newGateListener(listener net.Listener) *gateListener {
	gate := &gateListener{
		Listener: listener,
		accepted: make(chan acceptedGateConn, maxGateParsers),
		done:     make(chan struct{}),
		slots:    make(chan struct{}, maxGateParsers),
		pending:  make(map[net.Conn]struct{}),
	}
	go gate.acceptLoop()
	return gate
}

func (l *gateListener) Accept() (net.Conn, error) {
	select {
	case <-l.done:
		return nil, net.ErrClosed
	case accepted := <-l.accepted:
		l.forget(accepted.raw)
		return accepted.conn, nil
	}
}

func (l *gateListener) acceptLoop() {
	for {
		connection, err := l.Listener.Accept()
		if err != nil {
			return
		}
		if !l.track(connection) {
			_ = connection.Close()
			continue
		}
		select {
		case l.slots <- struct{}{}:
			go l.parseConnection(connection)
		default:
			l.reject(connection, http.StatusServiceUnavailable)
		}
	}
}

func (l *gateListener) parseConnection(connection net.Conn) {
	defer func() { <-l.slots }()
	if err := connection.SetReadDeadline(time.Now().Add(gateHeadTimeout)); err != nil {
		l.reject(connection, http.StatusBadRequest)
		return
	}
	head, err := parseRequestHead(bufio.NewReader(connection))
	if err != nil {
		status := http.StatusBadRequest
		if gated, ok := err.(*gateError); ok {
			status = gated.status
		}
		l.reject(connection, status)
		return
	}
	if err := connection.SetReadDeadline(time.Time{}); err != nil {
		l.reject(connection, http.StatusBadRequest)
		return
	}
	replay := append(append([]byte(nil), head.raw...), head.bodyPrefix...)
	accepted := acceptedGateConn{
		raw:  connection,
		conn: &gatedConn{Conn: connection, replay: bytes.NewReader(replay), remaining: head.contentLength - int64(len(head.bodyPrefix))},
	}
	select {
	case <-l.done:
		_ = connection.Close()
	case l.accepted <- accepted:
	default:
		l.reject(connection, http.StatusServiceUnavailable)
	}
}

func (l *gateListener) reject(connection net.Conn, status int) {
	l.forget(connection)
	_ = connection.SetWriteDeadline(time.Now().Add(gateRejectWriteTimeout))
	_, _ = fmt.Fprintf(connection, "HTTP/1.1 %d %s\r\nConnection: close\r\nContent-Length: 0\r\n\r\n", status, http.StatusText(status))
	_ = connection.Close()
}

func (l *gateListener) track(connection net.Conn) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return false
	}
	l.pending[connection] = struct{}{}
	return true
}

func (l *gateListener) forget(connection net.Conn) {
	l.mu.Lock()
	delete(l.pending, connection)
	l.mu.Unlock()
}

func (l *gateListener) Close() error {
	var closeErr error
	l.once.Do(func() {
		l.mu.Lock()
		l.closed = true
		connections := make([]net.Conn, 0, len(l.pending))
		for connection := range l.pending {
			connections = append(connections, connection)
		}
		l.pending = make(map[net.Conn]struct{})
		l.mu.Unlock()
		close(l.done)
		closeErr = l.Listener.Close()
		for _, connection := range connections {
			_ = connection.Close()
		}
	})
	return closeErr
}

// gatedConn exposes exactly one parsed request (head plus declared body) to
// net/http. Pipelined bytes are never made visible to the HTTP server.
type gatedConn struct {
	net.Conn
	replay    *bytes.Reader
	remaining int64
	mu        sync.Mutex
}

func (c *gatedConn) Read(target []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.replay.Len() > 0 {
		return c.replay.Read(target)
	}
	if c.remaining == 0 {
		// Keep the transport open while a streaming handler owns the request.
		// Server keep-alives are disabled, so net/http will close it after the
		// sole response rather than treating later bytes as another request.
		return c.Conn.Read(target)
	}
	if int64(len(target)) > c.remaining {
		target = target[:c.remaining]
	}
	n, err := c.Conn.Read(target)
	c.remaining -= int64(n)
	return n, err
}
