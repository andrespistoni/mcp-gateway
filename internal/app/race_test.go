package app

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
)

type concurrentCloseListener struct{ closes atomic.Int32 }

func (l *concurrentCloseListener) Close() error {
	l.closes.Add(1)
	return nil
}

type concurrentCloseProxy struct {
	shutdowns atomic.Int32
	kills     atomic.Int32
	waits     atomic.Int32
}

func (p *concurrentCloseProxy) Shutdown(context.Context) error { p.shutdowns.Add(1); return nil }
func (p *concurrentCloseProxy) Kill()                          { p.kills.Add(1) }
func (p *concurrentCloseProxy) Wait()                          { p.waits.Add(1) }

func TestStartedRuntimeCloseIsRaceSafeAndIdempotent(t *testing.T) {
	listener := &concurrentCloseListener{}
	proxy := &concurrentCloseProxy{}
	runtime := &startedRuntime{listener: listener, proxy: proxy}
	var group sync.WaitGroup
	for range 32 {
		group.Add(1)
		go func() {
			defer group.Done()
			if err := runtime.Close(context.Background()); err != nil {
				t.Error(err)
			}
		}()
	}
	group.Wait()
	if listener.closes.Load() != 1 || proxy.shutdowns.Load() != 1 || proxy.kills.Load() != 0 || proxy.waits.Load() != 0 {
		t.Fatalf("close listener=%d shutdown=%d kill=%d wait=%d", listener.closes.Load(), proxy.shutdowns.Load(), proxy.kills.Load(), proxy.waits.Load())
	}
}
