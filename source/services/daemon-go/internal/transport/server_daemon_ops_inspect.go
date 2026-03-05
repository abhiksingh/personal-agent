package transport

import (
	"net/http"
)

func (s *Server) handleInspectRun(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.inspect == nil {
		writeJSONError(writer, http.StatusNotImplemented, "inspect service is not configured", correlationID)
		return
	}

	var payload InspectRunRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid inspect run payload", &payload) {
		return
	}

	response, err := s.inspect.InspectRun(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleInspectTranscript(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.inspect == nil {
		writeJSONError(writer, http.StatusNotImplemented, "inspect service is not configured", correlationID)
		return
	}

	var payload InspectTranscriptRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid inspect transcript payload", &payload) {
		return
	}

	response, err := s.inspect.InspectTranscript(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleInspectMemory(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.inspect == nil {
		writeJSONError(writer, http.StatusNotImplemented, "inspect service is not configured", correlationID)
		return
	}

	var payload InspectMemoryRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid inspect memory payload", &payload) {
		return
	}

	response, err := s.inspect.InspectMemory(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleInspectLogsQuery(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.inspect == nil {
		writeJSONError(writer, http.StatusNotImplemented, "inspect service is not configured", correlationID)
		return
	}

	var payload InspectLogQueryRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid inspect logs query payload", &payload) {
		return
	}

	response, err := s.inspect.QueryInspectLogs(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleInspectLogsStream(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.inspect == nil {
		writeJSONError(writer, http.StatusNotImplemented, "inspect service is not configured", correlationID)
		return
	}

	var payload InspectLogStreamRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid inspect logs stream payload", &payload) {
		return
	}

	response, err := s.inspect.StreamInspectLogs(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}
