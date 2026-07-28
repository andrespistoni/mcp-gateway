package sse

import (
	"encoding/json"

	"mcp-gateway/internal/mcp"
)

func (s *Server) listTools(id mcp.RawID, params json.RawMessage) mcp.Envelope {
	s.mu.RLock()
	catalog, codec, ready := s.deps.Catalog, s.cursor, s.ready
	s.mu.RUnlock()
	if !ready || catalog == nil {
		response, _ := mcp.NewError(id, mcp.RPCError{Code: -32603, Message: "Internal error"})
		return response
	}
	var request struct {
		Cursor string `json:"cursor"`
	}
	if len(params) > 0 && json.Unmarshal(params, &request) != nil {
		response, _ := mcp.NewError(id, mcp.RPCError{Code: -32602, Message: "Invalid params"})
		return response
	}
	tools := catalog.Tools()
	start := 0
	if request.Cursor != "" {
		var err error
		start, err = codec.decode(request.Cursor, len(tools))
		if err != nil {
			response, _ := mcp.NewError(id, mcp.RPCError{Code: -32602, Message: "Invalid params"})
			return response
		}
	}
	end := start + 100
	if end > len(tools) {
		end = len(tools)
	}
	result := map[string]any{"tools": tools[start:end]}
	if end < len(tools) {
		result["nextCursor"] = codec.encode(end)
	}
	response, _ := mcp.NewResult(id, result)
	return response
}
