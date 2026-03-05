package transport

import (
	"net/http"

	openapitypes "personalagent/runtime/internal/transport/openapitypes"
)

func (s *Server) handleApprovalInbox(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.workflowQueries == nil {
		writeJSONError(writer, http.StatusNotImplemented, "workflow query service is not configured", correlationID)
		return
	}

	var openapiPayload openapitypes.ApprovalInboxRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid approval inbox payload", &openapiPayload) {
		return
	}
	payload := fromOpenAPIApprovalInboxRequest(openapiPayload)

	response, err := s.workflowQueries.ListApprovalInbox(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, toOpenAPIApprovalInboxResponse(response), correlationID)
}

func (s *Server) handleTaskRunList(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.workflowQueries == nil {
		writeJSONError(writer, http.StatusNotImplemented, "workflow query service is not configured", correlationID)
		return
	}

	var openapiPayload openapitypes.TaskRunListRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid task run list payload", &openapiPayload) {
		return
	}
	payload := fromOpenAPITaskRunListRequest(openapiPayload)

	response, err := s.workflowQueries.ListTaskRuns(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, toOpenAPITaskRunListResponse(response), correlationID)
}

func (s *Server) handleCommThreadList(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.workflowQueries == nil {
		writeJSONError(writer, http.StatusNotImplemented, "workflow query service is not configured", correlationID)
		return
	}

	var payload CommThreadListRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid comm thread list payload", &payload) {
		return
	}

	response, err := s.workflowQueries.ListCommThreads(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleCommEventTimeline(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.workflowQueries == nil {
		writeJSONError(writer, http.StatusNotImplemented, "workflow query service is not configured", correlationID)
		return
	}

	var payload CommEventTimelineRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid comm event timeline payload", &payload) {
		return
	}

	response, err := s.workflowQueries.ListCommEvents(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleCommCallSessionList(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.workflowQueries == nil {
		writeJSONError(writer, http.StatusNotImplemented, "workflow query service is not configured", correlationID)
		return
	}

	var payload CommCallSessionListRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid comm call-session list payload", &payload) {
		return
	}

	response, err := s.workflowQueries.ListCommCallSessions(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}
