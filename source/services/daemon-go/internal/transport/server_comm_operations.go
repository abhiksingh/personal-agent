package transport

import "net/http"

func (s *Server) handleCommSend(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}
	if s.comm == nil {
		writeJSONError(writer, http.StatusNotImplemented, "comm service is not configured", correlationID)
		return
	}

	var payload CommSendRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid comm send payload", &payload) {
		return
	}

	response, err := s.comm.SendComm(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleCommAttempts(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}
	if s.comm == nil {
		writeJSONError(writer, http.StatusNotImplemented, "comm service is not configured", correlationID)
		return
	}

	var payload CommAttemptsRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid comm attempts payload", &payload) {
		return
	}

	response, err := s.comm.ListCommAttempts(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleCommPolicySet(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}
	if s.comm == nil {
		writeJSONError(writer, http.StatusNotImplemented, "comm service is not configured", correlationID)
		return
	}

	var payload CommPolicySetRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid comm policy set payload", &payload) {
		return
	}

	response, err := s.comm.SetCommPolicy(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleCommPolicyList(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}
	if s.comm == nil {
		writeJSONError(writer, http.StatusNotImplemented, "comm service is not configured", correlationID)
		return
	}

	var payload CommPolicyListRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid comm policy list payload", &payload) {
		return
	}

	response, err := s.comm.ListCommPolicies(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleCommMessagesIngest(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}
	if s.comm == nil {
		writeJSONError(writer, http.StatusNotImplemented, "comm service is not configured", correlationID)
		return
	}

	var payload MessagesIngestRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid comm messages ingest payload", &payload) {
		return
	}

	response, err := s.comm.IngestMessages(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleCommMailIngest(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}
	if s.comm == nil {
		writeJSONError(writer, http.StatusNotImplemented, "comm service is not configured", correlationID)
		return
	}

	var payload MailRuleIngestRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid comm mail ingest payload", &payload) {
		return
	}

	response, err := s.comm.IngestMailRuleEvent(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleCommCalendarIngest(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}
	if s.comm == nil {
		writeJSONError(writer, http.StatusNotImplemented, "comm service is not configured", correlationID)
		return
	}

	var payload CalendarChangeIngestRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid comm calendar ingest payload", &payload) {
		return
	}

	response, err := s.comm.IngestCalendarChange(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleCommBrowserIngest(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}
	if s.comm == nil {
		writeJSONError(writer, http.StatusNotImplemented, "comm service is not configured", correlationID)
		return
	}

	var payload BrowserEventIngestRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid comm browser ingest payload", &payload) {
		return
	}

	response, err := s.comm.IngestBrowserEvent(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}
