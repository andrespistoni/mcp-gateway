package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"mcp-gateway/internal/mcp"
	"mcp-gateway/internal/project"
)

type State string

const (
	StateDisabled    State = "disabled"
	StateStarting    State = "starting"
	StateAvailable   State = "available"
	StateUnavailable State = "unavailable"
	StateStopped     State = "stopped"
)

var ErrDownstreamUnavailable = fmt.Errorf("downstream no disponible")

const (
	maxQueuedCalls = 32
	callTimeout    = 60 * time.Second
)

type DownstreamStatus struct {
	Name       string
	State      State
	Binary     string
	Fallback   bool
	Diagnostic string
	Stderr     StderrSnapshot
}

type runtimeDownstream struct {
	mu      sync.RWMutex
	status  DownstreamStatus
	process *managedProcess
}

type Service struct {
	downstreams []*runtimeDownstream
	byName      map[string]*runtimeDownstream
	catalog     CatalogSnapshot
	closeOnce   sync.Once
}

func (d *runtimeDownstream) setState(state State, diagnostic string) {
	d.mu.Lock()
	d.status.State = state
	d.status.Diagnostic = diagnostic
	d.mu.Unlock()
}

func (s *Service) Catalog() CatalogSnapshot { return s.catalog }

func (s *Service) Statuses() []DownstreamStatus {
	statuses := make([]DownstreamStatus, len(s.downstreams))
	for index, downstream := range s.downstreams {
		downstream.mu.RLock()
		statuses[index] = downstream.status
		if downstream.process != nil {
			statuses[index].Stderr = downstream.process.stderr.snapshot()
		}
		downstream.mu.RUnlock()
	}
	return statuses
}

func (s *Service) RequireAvailable(name string) error {
	downstream, exists := s.byName[name]
	if !exists {
		return ErrDownstreamUnavailable
	}
	downstream.mu.RLock()
	defer downstream.mu.RUnlock()
	if downstream.status.State != StateAvailable {
		return ErrDownstreamUnavailable
	}
	return nil
}

// TryCall reserves a downstream queue slot without blocking. A false admitted
// value means the caller must reject the HTTP request before admitting it.
// All other failures are represented by an immediately available JSON-RPC
// response so they retain the normal SSE delivery contract.
func (s *Service) TryCall(ctx context.Context, upstreamID mcp.RawID, params json.RawMessage, directory project.OptionalDir) (<-chan mcp.Envelope, func(), bool) {
	result := make(chan mcp.Envelope, 1)
	tool, route, forwarded, err := s.routeCall(params, directory)
	if err != nil {
		result <- rpcFailure(upstreamID, -32602, "Invalid params")
		return result, func() {}, true
	}
	downstream := s.byName[route.Downstream]
	if downstream == nil {
		result <- rpcFailure(upstreamID, -32002, "Downstream unavailable")
		return result, func() {}, true
	}
	downstream.mu.RLock()
	available := downstream.status.State == StateAvailable
	process := downstream.process
	downstream.mu.RUnlock()
	if !available || process == nil {
		result <- rpcFailure(upstreamID, -32002, "Downstream unavailable")
		return result, func() {}, true
	}
	select {
	case <-process.credits:
	default:
		return nil, nil, false
	}
	callCtx, cancel := context.WithTimeout(ctx, callTimeout)
	request := &call{ctx: callCtx, cancel: cancel, upstreamID: upstreamID, tool: tool, params: forwarded, result: result, finished: make(chan struct{})}
	select {
	case process.calls <- request:
	case <-process.done:
		cancel()
		process.credits <- struct{}{}
		result <- rpcFailure(upstreamID, -32002, "Downstream unavailable")
		return result, func() {}, true
	case <-ctx.Done():
		cancel()
		process.credits <- struct{}{}
		result <- rpcFailure(upstreamID, -32003, "Operation cancelled or deadline exceeded")
		return result, func() {}, true
	}
	go func() {
		select {
		case <-request.ctx.Done():
			process.cancel(request)
		case <-request.finished:
		}
	}()
	return result, func() { cancel(); process.cancel(request) }, true
}

func (s *Service) Shutdown(ctx context.Context) error {
	s.closeOnce.Do(func() {
		for _, downstream := range s.downstreams {
			downstream.mu.RLock()
			process := downstream.process
			downstream.mu.RUnlock()
			if process != nil {
				go process.stopAndWait(ctx)
			}
		}
	})
	for _, downstream := range s.downstreams {
		downstream.mu.RLock()
		process := downstream.process
		downstream.mu.RUnlock()
		if process == nil {
			continue
		}
		select {
		case <-process.done:
			downstream.setState(StateStopped, "")
		case <-ctx.Done():
			s.Kill()
			s.Wait()
			return ctx.Err()
		}
	}
	return nil
}

func (s *Service) Kill() {
	for _, downstream := range s.downstreams {
		downstream.mu.RLock()
		process := downstream.process
		downstream.mu.RUnlock()
		if process != nil {
			process.kill()
		}
	}
}

func (s *Service) Wait() {
	for _, downstream := range s.downstreams {
		downstream.mu.RLock()
		process := downstream.process
		downstream.mu.RUnlock()
		if process != nil {
			<-process.done
		}
	}
}
