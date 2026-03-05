package transport

import (
	"net/http"

	openapitypes "personalagent/runtime/internal/transport/openapitypes"
)

func (s *Server) handleProviderSet(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}
	if s.providers == nil {
		writeJSONError(writer, http.StatusNotImplemented, "provider service is not configured", correlationID)
		return
	}

	var openapiPayload openapitypes.ProviderSetRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid provider set payload", &openapiPayload) {
		return
	}
	payload := fromOpenAPIProviderSetRequest(openapiPayload)

	record, err := s.providers.SetProvider(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, toOpenAPIProviderConfigRecord(record), correlationID)
}

func (s *Server) handleProviderList(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}
	if s.providers == nil {
		writeJSONError(writer, http.StatusNotImplemented, "provider service is not configured", correlationID)
		return
	}

	var openapiPayload openapitypes.ProviderListRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid provider list payload", &openapiPayload) {
		return
	}
	payload := fromOpenAPIProviderListRequest(openapiPayload)

	response, err := s.providers.ListProviders(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, toOpenAPIProviderListResponse(response), correlationID)
}

func (s *Server) handleProviderCheck(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}
	if s.providers == nil {
		writeJSONError(writer, http.StatusNotImplemented, "provider service is not configured", correlationID)
		return
	}

	var openapiPayload openapitypes.ProviderCheckRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid provider check payload", &openapiPayload) {
		return
	}
	payload := fromOpenAPIProviderCheckRequest(openapiPayload)

	response, err := s.providers.CheckProviders(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, toOpenAPIProviderCheckResponse(response), correlationID)
}
