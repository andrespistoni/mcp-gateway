package sse

import (
	"context"
	"net/http"

	"mcp-gateway/internal/mcp"
)

const maxPendingCalls = 16

func (s *Server) beginCall(session *session, command submission, id mcp.RawID, envelope mcp.Envelope) {
	if command.ctx == nil {
		command.ctx = context.Background()
	}
	if s.deps.Caller == nil {
		response, _ := mcp.NewError(id, mcp.RPCError{Code: -32002, Message: "Downstream unavailable"})
		s.deliverCallResponse(session, command, response)
		return
	}
	session.mu.Lock()
	if session.state == stateClosed {
		session.mu.Unlock()
		command.reply(submissionResult{status: http.StatusNotFound, httpError: true, rpcCode: -32001})
		return
	}
	if len(session.pending) >= maxPendingCalls {
		session.mu.Unlock()
		command.reply(submissionResult{status: http.StatusTooManyRequests, httpError: true})
		return
	}
	session.mu.Unlock()

	result, cancel, admitted := s.deps.Caller.TryCall(context.Background(), id, envelope.Params(), session.project)
	if !admitted {
		command.reply(submissionResult{status: http.StatusTooManyRequests, httpError: true})
		return
	}
	session.mu.Lock()
	if session.state == stateClosed {
		session.mu.Unlock()
		cancel()
		command.reply(submissionResult{status: http.StatusGatewayTimeout, httpError: true, rpcCode: -32003})
		return
	}
	session.nextToken++
	token := session.nextToken
	session.pending[token] = pendingCall{command: command, cancel: cancel}
	session.mu.Unlock()

	go func() {
		var response mcp.Envelope
		select {
		case response = <-result:
		case <-command.ctx.Done():
			cancel()
			response, _ = mcp.NewError(id, mcp.RPCError{Code: -32003, Message: "Operation cancelled or deadline exceeded"})
		case <-session.closed:
			return
		}
		completion := submission{complete: &callCompletion{token: token, response: response}}
		select {
		case session.commands <- completion:
		case <-session.closed:
		}
	}()
}

func (s *Server) completeCall(writer http.ResponseWriter, controller *http.ResponseController, session *session, completion callCompletion) {
	session.mu.Lock()
	pending, found := session.pending[completion.token]
	if found {
		delete(session.pending, completion.token)
	}
	session.mu.Unlock()
	if !found {
		return
	}
	s.deliverCallResponseWithWriter(writer, controller, session, pending.command, completion.response)
}

func (s *Server) deliverCallResponse(session *session, command submission, response mcp.Envelope) {
	// This helper is only used for the unavailable-before-queue case. It is
	// queued as a completion so the session actor remains the sole SSE writer.
	session.mu.Lock()
	session.nextToken++
	token := session.nextToken
	session.pending[token] = pendingCall{command: command, cancel: func() {}}
	session.mu.Unlock()
	select {
	case session.commands <- submission{complete: &callCompletion{token: token, response: response}}:
	case <-session.closed:
	}
}

func (s *Server) deliverCallResponseWithWriter(writer http.ResponseWriter, controller *http.ResponseController, session *session, command submission, response mcp.Envelope) {
	payload, err := response.MarshalJSON()
	if err != nil || !flushEvent(writer, controller, func() error { return writeMessage(writer, payload) }) {
		session.close()
		command.reply(submissionResult{status: http.StatusGatewayTimeout, httpError: true, rpcCode: -32003})
		return
	}
	command.reply(submissionResult{status: http.StatusAccepted})
}
