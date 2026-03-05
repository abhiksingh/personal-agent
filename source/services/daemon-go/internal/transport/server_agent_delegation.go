package transport

import (
	"net/http"
	"strings"
)

func (s *Server) handleAgentRun(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}

	var payload AgentRunRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid agent run payload", &payload) {
		return
	}
	scopeType, scopeKey := controlRateLimitScopeFromActorOrWorkspace(
		firstNonEmptyTrimmed(payload.ActingAsActorID, payload.SubjectActorID, payload.RequestedByActorID),
		payload.WorkspaceID,
	)
	if !s.enforceControlRateLimit(writer, request, correlationID, controlRateLimitKeyAgentRun, scopeType, scopeKey) {
		return
	}
	if s.agent == nil {
		writeJSONError(writer, http.StatusNotImplemented, "agent service is not configured", correlationID)
		return
	}
	if strings.TrimSpace(payload.CorrelationID) == "" {
		payload.CorrelationID = correlationID
	}

	response, err := s.agent.RunAgent(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleAgentApprove(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}

	var payload AgentApproveRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid agent approve payload", &payload) {
		return
	}
	scopeType, scopeKey := controlRateLimitScopeFromActorOrWorkspace(payload.DecisionByActorID, payload.WorkspaceID)
	if !s.enforceControlRateLimit(writer, request, correlationID, controlRateLimitKeyAgentApprove, scopeType, scopeKey) {
		return
	}
	if s.agent == nil {
		writeJSONError(writer, http.StatusNotImplemented, "agent service is not configured", correlationID)
		return
	}
	if strings.TrimSpace(payload.CorrelationID) == "" {
		payload.CorrelationID = correlationID
	}

	response, err := s.agent.ApproveAgent(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleDelegationGrant(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}
	if s.delegation == nil {
		writeJSONError(writer, http.StatusNotImplemented, "delegation service is not configured", correlationID)
		return
	}

	var payload DelegationGrantRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid delegation grant payload", &payload) {
		return
	}

	response, err := s.delegation.GrantDelegation(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleDelegationList(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}
	if s.delegation == nil {
		writeJSONError(writer, http.StatusNotImplemented, "delegation service is not configured", correlationID)
		return
	}

	var payload DelegationListRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid delegation list payload", &payload) {
		return
	}

	response, err := s.delegation.ListDelegations(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleDelegationRevoke(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}
	if s.delegation == nil {
		writeJSONError(writer, http.StatusNotImplemented, "delegation service is not configured", correlationID)
		return
	}

	var payload DelegationRevokeRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid delegation revoke payload", &payload) {
		return
	}

	response, err := s.delegation.RevokeDelegation(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleDelegationCheck(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}
	if s.delegation == nil {
		writeJSONError(writer, http.StatusNotImplemented, "delegation service is not configured", correlationID)
		return
	}

	var payload DelegationCheckRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid delegation check payload", &payload) {
		return
	}

	response, err := s.delegation.CheckDelegation(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}
