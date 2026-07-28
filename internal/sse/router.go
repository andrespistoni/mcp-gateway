package sse

import (
	"net/http"
)

func (s *Server) router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /sse", s.handleSSE)
	mux.HandleFunc("POST /message", s.handleMessage)
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if !validRequestOrigin(request, s.port) {
			writer.WriteHeader(http.StatusForbidden)
			return
		}
		mux.ServeHTTP(writer, request)
	})
}
