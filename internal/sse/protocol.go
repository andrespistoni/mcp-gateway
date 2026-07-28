package sse

import (
	"encoding/json"
	"net/http"

	"mcp-gateway/internal/mcp"
	"mcp-gateway/internal/version"
)

func (s *Server) handleSubmission(writer http.ResponseWriter, controller *http.ResponseController, session *session, command submission) {
	if command.ctx != nil && command.ctx.Err() != nil {
		return
	}
	envelope, err := mcp.ParseEnvelope(command.envelope)
	if err != nil { // Already validated in POST; keep the actor defensive.
		command.reply(submissionResult{status: http.StatusBadRequest})
		return
	}
	if envelope.Kind() == mcp.Notification {
		if envelope.Method() == "notifications/initialized" {
			session.mu.Lock()
			valid := session.state == stateWaitingInitialized
			if valid {
				session.state = stateReady
			}
			session.mu.Unlock()
			if !valid {
				command.reply(submissionResult{status: http.StatusConflict, httpError: true, rpcCode: -32001})
				return
			}
		}
		command.reply(submissionResult{status: http.StatusAccepted})
		return
	}
	if envelope.Kind() != mcp.Request {
		command.reply(submissionResult{status: http.StatusBadRequest})
		return
	}
	id, _ := envelope.ID()
	var response mcp.Envelope
	if !s.allowed(session, envelope.Method()) {
		response, _ = mcp.NewError(id, mcp.RPCError{Code: -32001, Message: "Invalid session state"})
	} else {
		if envelope.Method() == "tools/call" {
			s.beginCall(session, command, id, envelope)
			return
		}
		response = s.dispatch(id, envelope)
		if envelope.Method() == "initialize" && response.Kind() == mcp.Result {
			// The state changes only after the response is fully delivered.
			session.mu.Lock()
			session.state = stateWaitingInitialized
			session.mu.Unlock()
		}
	}
	payload, err := response.MarshalJSON()
	if err != nil || !flushEvent(writer, controller, func() error { return writeMessage(writer, payload) }) {
		session.close()
		command.reply(submissionResult{status: http.StatusGatewayTimeout, httpError: true, rpcCode: -32003})
		return
	}
	command.reply(submissionResult{status: http.StatusAccepted})
}

func (s *Server) allowed(session *session, method string) bool {
	session.mu.Lock()
	defer session.mu.Unlock()
	switch session.state {
	case stateUninitialized:
		return method == "initialize"
	case stateWaitingInitialized:
		return method == "ping"
	case stateReady:
		// A ready session admits requests so dispatch can distinguish unsupported
		// methods (-32601) from an invalid session state (-32001).
		return true
	default:
		return false
	}
}

func (s *Server) dispatch(id mcp.RawID, envelope mcp.Envelope) mcp.Envelope {
	switch envelope.Method() {
	case "initialize":
		var params struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		if err := json.Unmarshal(envelope.Params(), &params); err != nil || params.ProtocolVersion == "" {
			response, _ := mcp.NewError(id, mcp.RPCError{Code: -32602, Message: "Invalid params"})
			return response
		}
		response, _ := mcp.NewResult(id, map[string]any{
			"protocolVersion": mcp.ProtocolVersion,
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "mcp-gateway", "version": version.Current().Release},
		})
		return response
	case "ping":
		response, _ := mcp.NewResult(id, map[string]any{})
		return response
	case "tools/list":
		return s.listTools(id, envelope.Params())
	default:
		response, _ := mcp.NewError(id, mcp.RPCError{Code: -32601, Message: "Method not found"})
		return response
	}
}
