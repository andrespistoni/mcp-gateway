package sse

import (
	"encoding/json"
	"net/http"

	"mcp-gateway/internal/mcp"
)

type rpcErrorWire struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Error   struct {
		Code    int64  `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

var errInvalidProject = &protocolError{message: "projectDir inválido"}

type protocolError struct{ message string }

func (e *protocolError) Error() string { return e.message }

func writeHTTPRPCError(writer http.ResponseWriter, status int, code int64, message string) {
	writeHTTPRPCErrorID(writer, status, code, message, nil)
}

// writeHTTPRPCErrorID preserves a validated request ID for errors rejected
// before SSE admission. Callers must pass only an ID already accepted by the
// JSON-RPC validator; all malformed or unavailable IDs remain null.
func writeHTTPRPCErrorID(writer http.ResponseWriter, status int, code int64, message string, id *mcp.RawID) {
	payload := rpcErrorWire{JSONRPC: "2.0", ID: nil}
	if id != nil {
		payload.ID = id.Bytes()
	}
	payload.Error.Code, payload.Error.Message = code, message
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(payload)
}
