package transport

import (
	"net/http"
)

func (s *Server) handleRetentionPurge(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.retention == nil {
		writeJSONError(writer, http.StatusNotImplemented, "retention service is not configured", correlationID)
		return
	}

	var payload RetentionPurgeRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid retention purge payload", &payload) {
		return
	}

	response, err := s.retention.PurgeRetention(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleRetentionCompactMemory(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.retention == nil {
		writeJSONError(writer, http.StatusNotImplemented, "retention service is not configured", correlationID)
		return
	}

	var payload RetentionCompactMemoryRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid retention compact-memory payload", &payload) {
		return
	}

	response, err := s.retention.CompactRetentionMemory(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}
