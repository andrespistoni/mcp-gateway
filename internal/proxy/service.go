package proxy

import (
	"context"
	"fmt"
	"sync"
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
