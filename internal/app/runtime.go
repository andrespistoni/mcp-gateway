package app

import (
	"context"
	"sync"

	"mcp-gateway/internal/config"
	"mcp-gateway/internal/endpoint"
)

type startupListener interface {
	Close() error
}

type startupProxy interface {
	Shutdown(context.Context) error
	Kill()
	Wait()
}

type proxyFactory func(context.Context, []config.Downstream) (startupProxy, error)

type listenerFactory func(context.Context, endpoint.Port) (startupListener, error)
type startupComposer func(startupListener, startupProxy) error

type startedRuntime struct {
	listener startupListener
	proxy    startupProxy
	once     sync.Once
	err      error
}

func (s *startedRuntime) Close(ctx context.Context) error {
	s.once.Do(func() {
		if err := s.proxy.Shutdown(ctx); err != nil {
			s.proxy.Kill()
			s.proxy.Wait()
			s.err = err
		}
		if err := s.listener.Close(); s.err == nil {
			s.err = err
		}
	})
	return s.err
}
