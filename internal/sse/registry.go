package sse

import (
	"sync"

	"mcp-gateway/internal/project"
)

const maxSessions = 128

type registry struct {
	mu       sync.RWMutex
	sessions map[SessionID]*session
	quiesced bool
}

func newRegistry() *registry { return &registry{sessions: make(map[SessionID]*session)} }

func (r *registry) reserve(id SessionID, directory project.OptionalDir) (*session, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.quiesced || len(r.sessions) >= maxSessions {
		return nil, false
	}
	s := newSession(id, directory)
	r.sessions[id] = s
	return s, true
}

func (r *registry) get(id SessionID) *session {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.sessions[id]
}

func (r *registry) release(id SessionID) {
	r.mu.Lock()
	delete(r.sessions, id)
	r.mu.Unlock()
}

func (r *registry) quiesce() {
	r.mu.Lock()
	r.quiesced = true
	r.mu.Unlock()
}
