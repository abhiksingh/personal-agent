package transport

import (
	"net/http"
)

func (s *Server) handleIdentityWorkspaces(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.identityDirectory == nil {
		writeJSONError(writer, http.StatusNotImplemented, "identity directory service is not configured", correlationID)
		return
	}

	var payload IdentityWorkspacesRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid identity workspaces payload", &payload) {
		return
	}

	response, err := s.identityDirectory.ListWorkspaces(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleIdentityPrincipals(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.identityDirectory == nil {
		writeJSONError(writer, http.StatusNotImplemented, "identity directory service is not configured", correlationID)
		return
	}

	var payload IdentityPrincipalsRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid identity principals payload", &payload) {
		return
	}

	response, err := s.identityDirectory.ListPrincipals(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleIdentityContext(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.identityDirectory == nil {
		writeJSONError(writer, http.StatusNotImplemented, "identity directory service is not configured", correlationID)
		return
	}

	var payload IdentityActiveContextRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid identity context payload", &payload) {
		return
	}

	response, err := s.identityDirectory.GetActiveContext(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleIdentitySelectWorkspace(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.identityDirectory == nil {
		writeJSONError(writer, http.StatusNotImplemented, "identity directory service is not configured", correlationID)
		return
	}

	var payload IdentityWorkspaceSelectRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid identity workspace select payload", &payload) {
		return
	}

	response, err := s.identityDirectory.SelectWorkspace(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleIdentityBootstrap(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.identityDirectory == nil {
		writeJSONError(writer, http.StatusNotImplemented, "identity directory service is not configured", correlationID)
		return
	}

	var payload IdentityBootstrapRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid identity bootstrap payload", &payload) {
		return
	}

	response, err := s.identityDirectory.Bootstrap(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleIdentityDevices(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.identityDirectory == nil {
		writeJSONError(writer, http.StatusNotImplemented, "identity directory service is not configured", correlationID)
		return
	}

	var payload IdentityDeviceListRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid identity devices payload", &payload) {
		return
	}

	response, err := s.identityDirectory.ListDevices(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleIdentitySessions(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.identityDirectory == nil {
		writeJSONError(writer, http.StatusNotImplemented, "identity directory service is not configured", correlationID)
		return
	}

	var payload IdentitySessionListRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid identity sessions payload", &payload) {
		return
	}

	response, err := s.identityDirectory.ListSessions(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleIdentitySessionRevoke(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.identityDirectory == nil {
		writeJSONError(writer, http.StatusNotImplemented, "identity directory service is not configured", correlationID)
		return
	}

	var payload IdentitySessionRevokeRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid identity sessions revoke payload", &payload) {
		return
	}

	response, err := s.identityDirectory.RevokeSession(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}
