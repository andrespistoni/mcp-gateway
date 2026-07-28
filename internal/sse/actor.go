package sse

import (
	"net/http"
	"strings"
	"time"

	"mcp-gateway/internal/project"
)

func (s *Server) handleSSE(writer http.ResponseWriter, request *http.Request) {
	directory, err := fromRequestProject(request)
	if err != nil {
		http.Error(writer, `{"error":"projectDir inválido"}`, http.StatusBadRequest)
		return
	}
	id, err := NewSessionID(s.deps.entropy())
	if err != nil {
		http.Error(writer, `{"error":"no se pudo crear sesión"}`, http.StatusInternalServerError)
		return
	}
	session, reserved := s.registry.reserve(id, directory)
	if !reserved {
		writer.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	defer func() { session.close(); s.registry.release(id) }()

	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("X-Accel-Buffering", "no")
	writer.WriteHeader(http.StatusOK)
	controller := http.NewResponseController(writer)
	if !flushEvent(writer, controller, func() error { return writeEndpoint(writer, id) }) {
		return
	}

	timer := time.NewTimer(15 * time.Second)
	defer timer.Stop()
	for {
		select {
		case <-request.Context().Done():
			return
		case <-session.closed:
			return
		case command := <-session.commands:
			if command.complete != nil {
				s.completeCall(writer, controller, session, *command.complete)
			} else {
				s.handleSubmission(writer, controller, session, command)
			}
			timer.Reset(15 * time.Second)
		case <-timer.C:
			if !flushEvent(writer, controller, func() error { return writeHeartbeat(writer) }) {
				return
			}
			timer.Reset(15 * time.Second)
		}
	}
}

func flushEvent(writer http.ResponseWriter, controller *http.ResponseController, write func() error) bool {
	if err := controller.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return false
	}
	defer func() { _ = controller.SetWriteDeadline(time.Time{}) }()
	if err := write(); err != nil {
		return false
	}
	return controller.Flush() == nil
}

func fromRequestProject(request *http.Request) (project.OptionalDir, error) {
	var queryValue *string
	values := rawQueryValues(request.URL.RawQuery, "projectDir")
	if len(values) == 1 {
		queryValue = &values[0]
	} else if len(values) > 1 {
		return project.OptionalDir{}, errInvalidProject
	}
	var headerValue *string
	if values := request.Header.Values("X-Project-Dir"); len(values) == 1 {
		value := values[0]
		headerValue = &value
	} else if len(request.Header.Values("X-Project-Dir")) > 1 {
		return project.OptionalDir{}, errInvalidProject
	}
	return project.FromHTTP(queryValue, headerValue)
}

func rawQueryValues(raw, key string) []string {
	var values []string
	for _, part := range strings.FieldsFunc(raw, func(r rune) bool { return r == '&' || r == ';' }) {
		name, value, found := strings.Cut(part, "=")
		if found && name == key {
			values = append(values, value)
		}
	}
	return values
}
