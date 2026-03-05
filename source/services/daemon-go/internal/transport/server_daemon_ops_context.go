package transport

import (
	"net/http"
)

func (s *Server) handleContextSamples(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.contextOps == nil {
		writeJSONError(writer, http.StatusNotImplemented, "context service is not configured", correlationID)
		return
	}

	var payload ContextSamplesRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid context samples payload", &payload) {
		return
	}

	response, err := s.contextOps.ListContextSamples(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleContextTune(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.contextOps == nil {
		writeJSONError(writer, http.StatusNotImplemented, "context service is not configured", correlationID)
		return
	}

	var payload ContextTuneRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid context tune payload", &payload) {
		return
	}

	response, err := s.contextOps.TuneContext(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleContextMemoryInventory(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.contextOps == nil {
		writeJSONError(writer, http.StatusNotImplemented, "context service is not configured", correlationID)
		return
	}

	var payload ContextMemoryInventoryRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid context memory inventory payload", &payload) {
		return
	}

	response, err := s.contextOps.QueryContextMemoryInventory(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleContextMemoryCandidates(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.contextOps == nil {
		writeJSONError(writer, http.StatusNotImplemented, "context service is not configured", correlationID)
		return
	}

	var payload ContextMemoryCandidatesRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid context memory candidates payload", &payload) {
		return
	}

	response, err := s.contextOps.QueryContextMemoryCandidates(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleContextRetrievalDocuments(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.contextOps == nil {
		writeJSONError(writer, http.StatusNotImplemented, "context service is not configured", correlationID)
		return
	}

	var payload ContextRetrievalDocumentsRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid context retrieval documents payload", &payload) {
		return
	}

	response, err := s.contextOps.QueryContextRetrievalDocuments(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleContextRetrievalChunks(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.contextOps == nil {
		writeJSONError(writer, http.StatusNotImplemented, "context service is not configured", correlationID)
		return
	}

	var payload ContextRetrievalChunksRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid context retrieval chunks payload", &payload) {
		return
	}

	response, err := s.contextOps.QueryContextRetrievalChunks(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}
