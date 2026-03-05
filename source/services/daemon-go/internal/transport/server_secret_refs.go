package transport

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

func (s *Server) handleSecretReferenceUpsert(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}
	if s.secretRefs == nil {
		writeJSONError(writer, http.StatusNotImplemented, "secret reference service is not configured", correlationID)
		return
	}

	var payload SecretReferenceUpsertRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid secret reference payload", &payload) {
		return
	}

	record, err := s.secretRefs.UpsertSecretReference(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}

	writeJSON(writer, http.StatusOK, SecretReferenceResponse{
		Reference:     record,
		CorrelationID: correlationID,
	}, correlationID)
}

func (s *Server) handleSecretReferenceLookup(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.secretRefs == nil {
		writeJSONError(writer, http.StatusNotImplemented, "secret reference service is not configured", correlationID)
		return
	}

	workspaceID, name, parseErr := parseSecretReferencePath(request.URL.Path)
	if parseErr != nil {
		writeJSONError(writer, http.StatusBadRequest, parseErr.Error(), correlationID)
		return
	}

	switch request.Method {
	case http.MethodGet:
		record, err := s.secretRefs.GetSecretReference(request.Context(), workspaceID, name)
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, ErrSecretReferenceNotFound) {
				status = http.StatusNotFound
			}
			writeJSONErrorFromError(writer, status, err, correlationID)
			return
		}
		writeJSON(writer, http.StatusOK, SecretReferenceResponse{
			Reference:     record,
			CorrelationID: correlationID,
		}, correlationID)
	case http.MethodDelete:
		record, err := s.secretRefs.DeleteSecretReference(request.Context(), workspaceID, name)
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, ErrSecretReferenceNotFound) {
				status = http.StatusNotFound
			}
			writeJSONErrorFromError(writer, status, err, correlationID)
			return
		}
		writeJSON(writer, http.StatusOK, SecretReferenceDeleteResponse{
			Reference:     record,
			Deleted:       true,
			CorrelationID: correlationID,
		}, correlationID)
	default:
		writeMethodNotAllowed(writer, http.MethodGet+", "+http.MethodDelete)
	}
}

func parseSecretReferencePath(path string) (string, string, error) {
	trimmed := strings.TrimSpace(strings.TrimPrefix(path, "/v1/secrets/refs/"))
	if trimmed == "" || trimmed == path {
		return "", "", fmt.Errorf("workspace_id and secret name are required")
	}
	parts := strings.Split(trimmed, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("secret reference path must be /v1/secrets/refs/{workspace_id}/{name}")
	}

	workspaceID, err := url.PathUnescape(parts[0])
	if err != nil {
		return "", "", fmt.Errorf("invalid workspace_id path segment")
	}
	name, err := url.PathUnescape(parts[1])
	if err != nil {
		return "", "", fmt.Errorf("invalid secret name path segment")
	}

	if strings.TrimSpace(workspaceID) == "" || strings.TrimSpace(name) == "" {
		return "", "", fmt.Errorf("workspace_id and secret name are required")
	}
	return workspaceID, name, nil
}
