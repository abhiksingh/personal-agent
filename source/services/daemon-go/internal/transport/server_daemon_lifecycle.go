package transport

import (
	"net/http"
)

func (s *Server) handleDaemonLifecycleStatus(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writeMethodNotAllowed(writer, http.MethodGet)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.lifecycle == nil {
		writeJSONError(writer, http.StatusNotImplemented, "daemon lifecycle service is not configured", correlationID)
		return
	}

	response, err := s.lifecycle.DaemonLifecycleStatus(request.Context())
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleDaemonLifecycleControl(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if !s.enforceControlRateLimit(writer, request, correlationID, controlRateLimitKeyDaemonLifecycleControl, "", "") {
		return
	}
	if s.lifecycle == nil {
		writeJSONError(writer, http.StatusNotImplemented, "daemon lifecycle service is not configured", correlationID)
		return
	}

	var payload DaemonLifecycleControlRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid daemon lifecycle control payload", &payload) {
		return
	}

	response, err := s.lifecycle.DaemonLifecycleControl(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleDaemonPluginLifecycleHistory(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.lifecycle == nil {
		writeJSONError(writer, http.StatusNotImplemented, "daemon lifecycle service is not configured", correlationID)
		return
	}

	var payload DaemonPluginLifecycleHistoryRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid daemon plugin lifecycle history payload", &payload) {
		return
	}

	response, err := s.lifecycle.DaemonPluginLifecycleHistory(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}
