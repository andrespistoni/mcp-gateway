package sse

import (
	"context"
	"encoding/json"
	"sync"

	"mcp-gateway/internal/mcp"
	"mcp-gateway/internal/project"
)

type sessionState uint8

const (
	stateUninitialized sessionState = iota
	stateWaitingInitialized
	stateReady
	stateClosed
)

type session struct {
	id        SessionID
	project   project.OptionalDir
	mu        sync.Mutex
	state     sessionState
	commands  chan submission
	closed    chan struct{}
	closeOnce sync.Once
	pending   map[uint64]pendingCall
	nextToken uint64
}

func newSession(id SessionID, directory project.OptionalDir) *session {
	return &session{id: id, project: directory, commands: make(chan submission, 16), closed: make(chan struct{}), pending: make(map[uint64]pendingCall)}
}

func (s *session) close() {
	s.closeOnce.Do(func() {
		s.mu.Lock()
		s.state = stateClosed
		pending := make([]pendingCall, 0, len(s.pending))
		for _, operation := range s.pending {
			pending = append(pending, operation)
		}
		s.pending = make(map[uint64]pendingCall)
		s.mu.Unlock()
		close(s.closed)
		for _, operation := range pending {
			operation.cancel()
			operation.command.reply(submissionResult{status: 504, httpError: true, rpcCode: -32003})
		}
	})
}

type submission struct {
	envelope json.RawMessage
	result   chan submissionResult
	ctx      context.Context
	complete *callCompletion
}

type submissionResult struct {
	status    int
	httpError bool
	rpcCode   int64
}

func (s submission) reply(result submissionResult) {
	select {
	case s.result <- result:
	default:
	}
}

type pendingCall struct {
	command submission
	cancel  func()
}

type callCompletion struct {
	token    uint64
	response mcp.Envelope
}
