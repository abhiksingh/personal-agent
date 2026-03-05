package transport

import (
	"net/http"
)

func (s *Server) handleAutomationCreate(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	var payload AutomationCreateRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid automation create payload", &payload) {
		return
	}
	scopeType, scopeKey := controlRateLimitScopeFromActorOrWorkspace(payload.SubjectActorID, payload.WorkspaceID)
	if !s.enforceControlRateLimit(writer, request, correlationID, controlRateLimitKeyAutomationCreate, scopeType, scopeKey) {
		return
	}
	if s.automation == nil {
		writeJSONError(writer, http.StatusNotImplemented, "automation service is not configured", correlationID)
		return
	}

	response, err := s.automation.CreateAutomation(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleAutomationList(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.automation == nil {
		writeJSONError(writer, http.StatusNotImplemented, "automation service is not configured", correlationID)
		return
	}

	var payload AutomationListRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid automation list payload", &payload) {
		return
	}

	response, err := s.automation.ListAutomation(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleAutomationFireHistory(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.automation == nil {
		writeJSONError(writer, http.StatusNotImplemented, "automation service is not configured", correlationID)
		return
	}

	var payload AutomationFireHistoryRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid automation fire-history payload", &payload) {
		return
	}

	response, err := s.automation.ListAutomationFireHistory(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleAutomationUpdate(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	var payload AutomationUpdateRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid automation update payload", &payload) {
		return
	}
	scopeType, scopeKey := controlRateLimitScopeFromActorOrWorkspace(payload.SubjectActorID, payload.WorkspaceID)
	if !s.enforceControlRateLimit(writer, request, correlationID, controlRateLimitKeyAutomationUpdate, scopeType, scopeKey) {
		return
	}
	if s.automation == nil {
		writeJSONError(writer, http.StatusNotImplemented, "automation service is not configured", correlationID)
		return
	}

	response, err := s.automation.UpdateAutomation(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleAutomationDelete(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	var payload AutomationDeleteRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid automation delete payload", &payload) {
		return
	}
	scopeType, scopeKey := controlRateLimitScopeFromActorOrWorkspace("", payload.WorkspaceID)
	if !s.enforceControlRateLimit(writer, request, correlationID, controlRateLimitKeyAutomationDelete, scopeType, scopeKey) {
		return
	}
	if s.automation == nil {
		writeJSONError(writer, http.StatusNotImplemented, "automation service is not configured", correlationID)
		return
	}

	response, err := s.automation.DeleteAutomation(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleAutomationRunSchedule(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	var payload AutomationRunScheduleRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid automation run schedule payload", &payload) {
		return
	}
	if !s.enforceControlRateLimit(writer, request, correlationID, controlRateLimitKeyAutomationRunSchedule, "", "") {
		return
	}
	if s.automation == nil {
		writeJSONError(writer, http.StatusNotImplemented, "automation service is not configured", correlationID)
		return
	}

	response, err := s.automation.RunAutomationSchedule(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleAutomationRunCommEvent(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	var payload AutomationRunCommEventRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid automation run comm-event payload", &payload) {
		return
	}
	scopeType, scopeKey := controlRateLimitScopeFromActorOrWorkspace("", payload.WorkspaceID)
	if !s.enforceControlRateLimit(writer, request, correlationID, controlRateLimitKeyAutomationRunCommEvent, scopeType, scopeKey) {
		return
	}
	if s.automation == nil {
		writeJSONError(writer, http.StatusNotImplemented, "automation service is not configured", correlationID)
		return
	}

	response, err := s.automation.RunAutomationCommEvent(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleAutomationCommTriggerMetadata(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.automation == nil {
		writeJSONError(writer, http.StatusNotImplemented, "automation service is not configured", correlationID)
		return
	}

	var payload AutomationCommTriggerMetadataRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid automation comm-trigger metadata payload", &payload) {
		return
	}

	response, err := s.automation.AutomationCommTriggerMetadata(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleAutomationCommTriggerValidate(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.automation == nil {
		writeJSONError(writer, http.StatusNotImplemented, "automation service is not configured", correlationID)
		return
	}

	var payload AutomationCommTriggerValidateRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid automation comm-trigger validate payload", &payload) {
		return
	}

	response, err := s.automation.AutomationCommTriggerValidate(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}
