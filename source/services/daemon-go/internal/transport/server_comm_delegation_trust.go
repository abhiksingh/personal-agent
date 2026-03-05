package transport

import (
	"net/http"
)

func (s *Server) handleCapabilityGrantUpsert(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.delegation == nil {
		writeJSONError(writer, http.StatusNotImplemented, "delegation service is not configured", correlationID)
		return
	}

	var payload CapabilityGrantUpsertRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid capability grant upsert payload", &payload) {
		return
	}

	response, err := s.delegation.UpsertCapabilityGrant(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleCapabilityGrantList(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.delegation == nil {
		writeJSONError(writer, http.StatusNotImplemented, "delegation service is not configured", correlationID)
		return
	}

	var payload CapabilityGrantListRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid capability grant list payload", &payload) {
		return
	}

	response, err := s.delegation.ListCapabilityGrants(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleCommWebhookReceiptList(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.comm == nil {
		writeJSONError(writer, http.StatusNotImplemented, "comm service is not configured", correlationID)
		return
	}

	var payload CommWebhookReceiptListRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid comm webhook-receipt list payload", &payload) {
		return
	}

	response, err := s.comm.ListCommWebhookReceipts(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleCommIngestReceiptList(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.comm == nil {
		writeJSONError(writer, http.StatusNotImplemented, "comm service is not configured", correlationID)
		return
	}

	var payload CommIngestReceiptListRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid comm ingest-receipt list payload", &payload) {
		return
	}

	response, err := s.comm.ListCommIngestReceipts(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}
