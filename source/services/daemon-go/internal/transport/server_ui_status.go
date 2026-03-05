package transport

import (
	"net/http"
)

func (s *Server) handleChannelConnectorMappingList(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.uiStatus == nil {
		writeJSONError(writer, http.StatusNotImplemented, "ui status service is not configured", correlationID)
		return
	}

	var payload ChannelConnectorMappingListRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid channel connector mapping list payload", &payload) {
		return
	}

	response, err := s.uiStatus.ListChannelConnectorMappings(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleChannelConnectorMappingUpsert(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.uiStatus == nil {
		writeJSONError(writer, http.StatusNotImplemented, "ui status service is not configured", correlationID)
		return
	}

	var payload ChannelConnectorMappingUpsertRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid channel connector mapping upsert payload", &payload) {
		return
	}

	response, err := s.uiStatus.UpsertChannelConnectorMapping(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleChannelStatus(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.uiStatus == nil {
		writeJSONError(writer, http.StatusNotImplemented, "ui status service is not configured", correlationID)
		return
	}

	var payload ChannelStatusRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid channel status payload", &payload) {
		return
	}

	response, err := s.uiStatus.ListChannelStatus(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, normalizeChannelStatusResponseDescriptors(response), correlationID)
}

func (s *Server) handleConnectorStatus(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.uiStatus == nil {
		writeJSONError(writer, http.StatusNotImplemented, "ui status service is not configured", correlationID)
		return
	}

	var payload ConnectorStatusRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid connector status payload", &payload) {
		return
	}

	response, err := s.uiStatus.ListConnectorStatus(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, normalizeConnectorStatusResponseDescriptors(response), correlationID)
}

func (s *Server) handleChannelDiagnostics(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.uiStatus == nil {
		writeJSONError(writer, http.StatusNotImplemented, "ui status service is not configured", correlationID)
		return
	}

	var payload ChannelDiagnosticsRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid channel diagnostics payload", &payload) {
		return
	}

	response, err := s.uiStatus.ListChannelDiagnostics(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleConnectorDiagnostics(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.uiStatus == nil {
		writeJSONError(writer, http.StatusNotImplemented, "ui status service is not configured", correlationID)
		return
	}

	var payload ConnectorDiagnosticsRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid connector diagnostics payload", &payload) {
		return
	}

	response, err := s.uiStatus.ListConnectorDiagnostics(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleConnectorPermissionRequest(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.uiStatus == nil {
		writeJSONError(writer, http.StatusNotImplemented, "ui status service is not configured", correlationID)
		return
	}

	var payload ConnectorPermissionRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid connector permission request payload", &payload) {
		return
	}

	response, err := s.uiStatus.RequestConnectorPermission(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleChannelConfigUpsert(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.uiStatus == nil {
		writeJSONError(writer, http.StatusNotImplemented, "ui status service is not configured", correlationID)
		return
	}

	var payload ChannelConfigUpsertRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid channel config upsert payload", &payload) {
		return
	}

	response, err := s.uiStatus.UpsertChannelConfig(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleConnectorConfigUpsert(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.uiStatus == nil {
		writeJSONError(writer, http.StatusNotImplemented, "ui status service is not configured", correlationID)
		return
	}

	var payload ConnectorConfigUpsertRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid connector config upsert payload", &payload) {
		return
	}

	response, err := s.uiStatus.UpsertConnectorConfig(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleChannelTestOperation(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.uiStatus == nil {
		writeJSONError(writer, http.StatusNotImplemented, "ui status service is not configured", correlationID)
		return
	}

	var payload ChannelTestOperationRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid channel test payload", &payload) {
		return
	}

	response, err := s.uiStatus.TestChannelOperation(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleConnectorTestOperation(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.uiStatus == nil {
		writeJSONError(writer, http.StatusNotImplemented, "ui status service is not configured", correlationID)
		return
	}

	var payload ConnectorTestOperationRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid connector test payload", &payload) {
		return
	}

	response, err := s.uiStatus.TestConnectorOperation(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func normalizeChannelStatusResponseDescriptors(response ChannelStatusResponse) ChannelStatusResponse {
	for index := range response.Channels {
		response.Channels[index].ConfigFieldDescriptors = normalizeConfigFieldDescriptorDefaults(
			response.Channels[index].ConfigFieldDescriptors,
		)
	}
	return response
}

func normalizeConnectorStatusResponseDescriptors(response ConnectorStatusResponse) ConnectorStatusResponse {
	for index := range response.Connectors {
		response.Connectors[index].ConfigFieldDescriptors = normalizeConfigFieldDescriptorDefaults(
			response.Connectors[index].ConfigFieldDescriptors,
		)
	}
	return response
}

func normalizeConfigFieldDescriptorDefaults(descriptors []ConfigFieldDescriptor) []ConfigFieldDescriptor {
	if len(descriptors) == 0 {
		return descriptors
	}
	normalized := make([]ConfigFieldDescriptor, len(descriptors))
	copy(normalized, descriptors)
	for index := range normalized {
		if normalized[index].EnumOptions == nil {
			normalized[index].EnumOptions = []string{}
		}
	}
	return normalized
}
