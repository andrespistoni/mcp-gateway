package sse

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"mcp-gateway/internal/mcp"
)

const (
	maxHTTPBody     = 1 << 20
	bodyReadTimeout = 5 * time.Second
)

func (s *Server) handleMessage(writer http.ResponseWriter, request *http.Request) {
	mediaType, parameters, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" || (parameters["charset"] != "" && !strings.EqualFold(parameters["charset"], "utf-8")) {
		writeHTTPRPCError(writer, http.StatusUnsupportedMediaType, -32600, "Invalid Request")
		return
	}
	query, err := url.ParseQuery(request.URL.RawQuery)
	if err != nil {
		writeHTTPRPCError(writer, http.StatusBadRequest, -32600, "Invalid Request")
		return
	}
	values := query["sessionId"]
	if len(values) != 1 {
		writeHTTPRPCError(writer, http.StatusBadRequest, -32600, "Invalid Request")
		return
	}
	id, err := ParseSessionID(values[0])
	if err != nil {
		writeHTTPRPCError(writer, http.StatusBadRequest, -32600, "Invalid Request")
		return
	}
	if request.ContentLength > maxHTTPBody {
		writeHTTPRPCError(writer, http.StatusRequestEntityTooLarge, -32004, "Resource limit exceeded")
		return
	}
	controller := http.NewResponseController(writer)
	deadlineSet := controller.SetReadDeadline(time.Now().Add(bodyReadTimeout)) == nil
	body, err := io.ReadAll(io.LimitReader(request.Body, maxHTTPBody+1))
	if deadlineSet && err == nil {
		_ = controller.SetReadDeadline(time.Time{})
	}
	if err != nil {
		writeHTTPRPCError(writer, http.StatusBadRequest, -32700, "Parse error")
		return
	}
	if len(body) > maxHTTPBody {
		writeHTTPRPCError(writer, http.StatusRequestEntityTooLarge, -32004, "Resource limit exceeded")
		return
	}
	if !json.Valid(body) || !utf8.Valid(body) {
		writeHTTPRPCError(writer, http.StatusBadRequest, -32700, "Parse error")
		return
	}
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 || trimmed[0] == '[' {
		writeHTTPRPCError(writer, http.StatusBadRequest, -32600, "Invalid Request")
		return
	}
	requestID := parsedRequestID(trimmed)
	if _, err := mcp.ParseEnvelope(trimmed); err != nil {
		writeHTTPRPCErrorID(writer, http.StatusBadRequest, -32600, "Invalid Request", requestID)
		return
	}
	session := s.registry.get(id)
	if session == nil {
		writeHTTPRPCErrorID(writer, http.StatusNotFound, -32001, "Invalid session state", requestID)
		return
	}
	command := submission{envelope: append(json.RawMessage(nil), trimmed...), result: make(chan submissionResult, 1), ctx: request.Context()}
	select {
	case <-request.Context().Done():
		return
	case <-session.closed:
		writeHTTPRPCError(writer, http.StatusNotFound, -32001, "Invalid session state")
		return
	case session.commands <- command:
	}
	select {
	case <-request.Context().Done():
		return
	case result := <-command.result:
		if result.httpError {
			code := result.rpcCode
			message := "Operation cancelled or deadline exceeded"
			if code == 0 {
				code, message = -32004, "Resource limit exceeded"
			}
			writeHTTPRPCError(writer, result.status, code, message)
			return
		}
		if result.status == 0 {
			result.status = http.StatusAccepted
		}
		writer.WriteHeader(result.status)
	}
}

// parsedRequestID returns an ID only when its JSON form is independently
// valid. It is intentionally tolerant of an otherwise invalid envelope so
// pre-admission JSON-RPC errors can retain a caller-provided valid ID.
func parsedRequestID(body []byte) *mcp.RawID {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil || fields == nil {
		return nil
	}
	raw, ok := fields["id"]
	if !ok {
		return nil
	}
	id, err := mcp.ParseID(raw)
	if err != nil {
		return nil
	}
	return &id
}
