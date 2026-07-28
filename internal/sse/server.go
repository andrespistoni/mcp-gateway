package sse

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"mcp-gateway/internal/endpoint"
)

type Server struct {
	listener net.Listener
	port     endpoint.Port
	registry *registry
	deps     Dependencies
	http     *http.Server
	mu       sync.RWMutex
	cursor   cursorCodec
	ready    bool
}

func Bind(ctx context.Context, port endpoint.Port, deps Dependencies) (*Server, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	addresses, err := net.LookupIP("localhost")
	if err != nil || len(addresses) == 0 {
		return nil, fmt.Errorf("no se pudo resolver localhost: %w", err)
	}
	for _, address := range addresses {
		if !address.IsLoopback() {
			return nil, fmt.Errorf("localhost no resuelve exclusivamente a loopback")
		}
	}
	listener, err := net.Listen("tcp", endpoint.LocalhostAddress(port))
	if err != nil {
		return nil, fmt.Errorf("escuchar localhost: %w", err)
	}
	return bindListener(port, listener, deps)
}

// bindListener is deliberately package-private: tests can preacquire a
// loopback listener without ever touching the contractual default port.
func bindListener(port endpoint.Port, listener net.Listener, deps Dependencies) (*Server, error) {
	if listener == nil {
		return nil, fmt.Errorf("listener ausente")
	}
	address, ok := listener.Addr().(*net.TCPAddr)
	if !ok || !address.IP.IsLoopback() {
		_ = listener.Close()
		return nil, fmt.Errorf("el listener no es loopback")
	}
	codec, err := newCursorCodec([32]byte{}, deps.entropy())
	if err != nil {
		_ = listener.Close()
		return nil, err
	}
	server := &Server{listener: newGateListener(listener), port: port, registry: newRegistry(), deps: deps, cursor: codec}
	server.http = &http.Server{
		Handler:           server.router(),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       5 * time.Second,
		MaxHeaderBytes:    maxHeaderBytes + 1024,
	}
	return server, nil
}

func (s *Server) SetCatalog(catalog Catalog) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ready {
		return fmt.Errorf("el catálogo ya está configurado")
	}
	if catalog == nil {
		return fmt.Errorf("catálogo ausente")
	}
	codec, err := newCursorCodec(catalog.Identity(), s.deps.entropy())
	if err != nil {
		return err
	}
	s.deps.Catalog = catalog
	s.cursor = codec
	s.ready = true
	return nil
}

// SetCaller wires the consumer-owned call interface before the server is made
// ready. It deliberately does not expose proxy implementation types.
func (s *Server) SetCaller(caller Caller) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ready {
		return fmt.Errorf("el caller ya no puede cambiarse")
	}
	if caller == nil {
		return fmt.Errorf("caller ausente")
	}
	s.deps.Caller = caller
	return nil
}

func (s *Server) Serve() error {
	s.http.SetKeepAlivesEnabled(false)
	err := s.http.Serve(s.listener)
	if errors.Is(err, http.ErrServerClosed) || errors.Is(err, net.ErrClosed) {
		return nil
	}
	return err
}

func (s *Server) Port() endpoint.Port { return s.port }

func (s *Server) Quiesce() { s.registry.quiesce() }

func (s *Server) CloseSessions() {
	s.registry.mu.RLock()
	sessions := make([]*session, 0, len(s.registry.sessions))
	for _, session := range s.registry.sessions {
		sessions = append(sessions, session)
	}
	s.registry.mu.RUnlock()
	for _, session := range sessions {
		session.close()
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.Quiesce()
	s.CloseSessions()
	return s.http.Shutdown(ctx)
}

func (s *Server) Close() error {
	s.Quiesce()
	s.CloseSessions()
	return s.http.Close()
}
