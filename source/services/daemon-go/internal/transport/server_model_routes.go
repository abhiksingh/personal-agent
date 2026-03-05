package transport

import (
	"net/http"

	openapitypes "personalagent/runtime/internal/transport/openapitypes"
)

func (s *Server) handleModelList(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}
	if s.models == nil {
		writeJSONError(writer, http.StatusNotImplemented, "model service is not configured", correlationID)
		return
	}

	var openapiPayload openapitypes.ModelListRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid model list payload", &openapiPayload) {
		return
	}
	payload := fromOpenAPIModelListRequest(openapiPayload)

	response, err := s.models.ListModels(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, toOpenAPIModelListResponse(response), correlationID)
}

func (s *Server) handleModelDiscover(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}
	if s.models == nil {
		writeJSONError(writer, http.StatusNotImplemented, "model service is not configured", correlationID)
		return
	}

	var openapiPayload openapitypes.ModelDiscoverRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid model discover payload", &openapiPayload) {
		return
	}
	payload := fromOpenAPIModelDiscoverRequest(openapiPayload)

	response, err := s.models.DiscoverModels(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, toOpenAPIModelDiscoverResponse(response), correlationID)
}

func (s *Server) handleModelAdd(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}
	if s.models == nil {
		writeJSONError(writer, http.StatusNotImplemented, "model service is not configured", correlationID)
		return
	}

	var payload ModelCatalogAddRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid model add payload", &payload) {
		return
	}

	record, err := s.models.AddModel(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, record, correlationID)
}

func (s *Server) handleModelRemove(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}
	if s.models == nil {
		writeJSONError(writer, http.StatusNotImplemented, "model service is not configured", correlationID)
		return
	}

	var payload ModelCatalogRemoveRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid model remove payload", &payload) {
		return
	}

	response, err := s.models.RemoveModel(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleModelEnable(writer http.ResponseWriter, request *http.Request) {
	s.handleModelToggle(writer, request, true)
}

func (s *Server) handleModelDisable(writer http.ResponseWriter, request *http.Request) {
	s.handleModelToggle(writer, request, false)
}

func (s *Server) handleModelToggle(writer http.ResponseWriter, request *http.Request, enabled bool) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}
	if s.models == nil {
		writeJSONError(writer, http.StatusNotImplemented, "model service is not configured", correlationID)
		return
	}

	var payload ModelToggleRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid model toggle payload", &payload) {
		return
	}

	var (
		record ModelCatalogEntryRecord
		err    error
	)
	if enabled {
		record, err = s.models.EnableModel(request.Context(), payload)
	} else {
		record, err = s.models.DisableModel(request.Context(), payload)
	}
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, record, correlationID)
}

func (s *Server) handleModelSelect(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}
	if s.models == nil {
		writeJSONError(writer, http.StatusNotImplemented, "model service is not configured", correlationID)
		return
	}

	var payload ModelSelectRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid model select payload", &payload) {
		return
	}

	record, err := s.models.SelectModelRoute(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, record, correlationID)
}

func (s *Server) handleModelPolicy(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}
	if s.models == nil {
		writeJSONError(writer, http.StatusNotImplemented, "model service is not configured", correlationID)
		return
	}

	var payload ModelPolicyRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid model policy payload", &payload) {
		return
	}

	response, err := s.models.GetModelPolicy(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleModelResolve(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}
	if s.models == nil {
		writeJSONError(writer, http.StatusNotImplemented, "model service is not configured", correlationID)
		return
	}

	var payload ModelResolveRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid model resolve payload", &payload) {
		return
	}

	response, err := s.models.ResolveModelRoute(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleModelRouteSimulate(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}
	if s.models == nil {
		writeJSONError(writer, http.StatusNotImplemented, "model service is not configured", correlationID)
		return
	}

	var payload ModelRouteSimulationRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid model route simulation payload", &payload) {
		return
	}

	response, err := s.models.SimulateModelRoute(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleModelRouteExplain(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}
	if s.models == nil {
		writeJSONError(writer, http.StatusNotImplemented, "model service is not configured", correlationID)
		return
	}

	var openapiPayload openapitypes.ModelRouteExplainRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid model route explain payload", &openapiPayload) {
		return
	}
	payload := fromOpenAPIModelRouteExplainRequest(openapiPayload)

	response, err := s.models.ExplainModelRoute(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, toOpenAPIModelRouteExplainResponse(response), correlationID)
}
