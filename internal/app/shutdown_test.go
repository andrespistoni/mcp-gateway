package app

import (
	"context"
	"sync"
	"testing"
)

type shutdownServerStub struct {
	mu       sync.Mutex
	quiesced bool
	closed   bool
}

func (s *shutdownServerStub) Quiesce()                       { s.mu.Lock(); s.quiesced = true; s.mu.Unlock() }
func (s *shutdownServerStub) Shutdown(context.Context) error { return nil }
func (s *shutdownServerStub) Close() error                   { s.mu.Lock(); s.closed = true; s.mu.Unlock(); return nil }

type shutdownProxyStub struct {
	mu       sync.Mutex
	shutdown bool
	killed   bool
}

func (p *shutdownProxyStub) Shutdown(context.Context) error {
	p.mu.Lock()
	p.shutdown = true
	p.mu.Unlock()
	return nil
}
func (p *shutdownProxyStub) Kill() { p.mu.Lock(); p.killed = true; p.mu.Unlock() }
func (p *shutdownProxyStub) Wait() {}

func TestShutdownRuntimeQuiescesAndJoinsBothOwners(t *testing.T) {
	server := &shutdownServerStub{}
	proxy := &shutdownProxyStub{}
	if err := shutdownRuntime(server, proxy); err != nil {
		t.Fatal(err)
	}
	server.mu.Lock()
	quiesced := server.quiesced
	server.mu.Unlock()
	proxy.mu.Lock()
	shutdown := proxy.shutdown
	proxy.mu.Unlock()
	if !quiesced || !shutdown {
		t.Fatalf("shutdown did not coordinate owners: quiesced=%v proxy=%v", quiesced, shutdown)
	}
}
