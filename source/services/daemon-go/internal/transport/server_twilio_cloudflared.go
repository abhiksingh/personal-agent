package transport

import (
	"net/http"
	"strings"
)

func (s *Server) handleTwilioSet(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}
	if s.twilio == nil {
		writeJSONError(writer, http.StatusNotImplemented, "twilio channel service is not configured", correlationID)
		return
	}

	var payload TwilioSetRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid twilio set payload", &payload) {
		return
	}

	response, err := s.twilio.SetTwilioChannel(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleTwilioGet(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}
	if s.twilio == nil {
		writeJSONError(writer, http.StatusNotImplemented, "twilio channel service is not configured", correlationID)
		return
	}

	var payload TwilioGetRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid twilio get payload", &payload) {
		return
	}

	response, err := s.twilio.GetTwilioChannel(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleTwilioCheck(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}
	if s.twilio == nil {
		writeJSONError(writer, http.StatusNotImplemented, "twilio channel service is not configured", correlationID)
		return
	}

	var payload TwilioCheckRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid twilio check payload", &payload) {
		return
	}

	response, err := s.twilio.CheckTwilioChannel(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleTwilioSMSChatTurn(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}
	if s.twilio == nil {
		writeJSONError(writer, http.StatusNotImplemented, "twilio channel service is not configured", correlationID)
		return
	}

	var payload TwilioSMSChatTurnRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid twilio sms-chat payload", &payload) {
		return
	}

	response, err := s.twilio.ExecuteTwilioSMSChatTurn(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleTwilioStartCall(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}
	if s.twilio == nil {
		writeJSONError(writer, http.StatusNotImplemented, "twilio channel service is not configured", correlationID)
		return
	}

	var payload TwilioStartCallRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid twilio start-call payload", &payload) {
		return
	}

	response, err := s.twilio.StartTwilioCall(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleTwilioCallStatus(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}
	if s.twilio == nil {
		writeJSONError(writer, http.StatusNotImplemented, "twilio channel service is not configured", correlationID)
		return
	}

	var payload TwilioCallStatusRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid twilio call-status payload", &payload) {
		return
	}

	response, err := s.twilio.ListTwilioCallStatus(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleTwilioTranscript(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}
	if s.twilio == nil {
		writeJSONError(writer, http.StatusNotImplemented, "twilio channel service is not configured", correlationID)
		return
	}

	var payload TwilioTranscriptRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid twilio transcript payload", &payload) {
		return
	}

	response, err := s.twilio.ListTwilioTranscript(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleTwilioWebhookServe(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}
	if s.twilio == nil {
		writeJSONError(writer, http.StatusNotImplemented, "twilio channel service is not configured", correlationID)
		return
	}

	var payload TwilioWebhookServeRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid twilio webhook serve payload", &payload) {
		return
	}
	if s.rejectTwilioWebhookBypassInProd(writer, correlationID, payload.SignatureMode) {
		return
	}

	response, err := s.twilio.ServeTwilioWebhook(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleTwilioWebhookReplay(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}
	if s.twilio == nil {
		writeJSONError(writer, http.StatusNotImplemented, "twilio channel service is not configured", correlationID)
		return
	}

	var payload TwilioWebhookReplayRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid twilio webhook replay payload", &payload) {
		return
	}
	if s.rejectTwilioWebhookBypassInProd(writer, correlationID, payload.SignatureMode) {
		return
	}

	response, err := s.twilio.ReplayTwilioWebhook(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) rejectTwilioWebhookBypassInProd(writer http.ResponseWriter, correlationID string, signatureMode string) bool {
	runtimeProfile, err := normalizeTransportRuntimeProfile(s.config.RuntimeProfile)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return true
	}
	if runtimeProfile != transportRuntimeProfileProd {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(signatureMode), "bypass") {
		writeJSONError(writer, http.StatusBadRequest, "--runtime-profile=prod does not allow --signature-mode=bypass", correlationID)
		return true
	}
	return false
}

func (s *Server) handleCloudflaredVersion(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}
	if s.cloudflared == nil {
		writeJSONError(writer, http.StatusNotImplemented, "cloudflared connector service is not configured", correlationID)
		return
	}

	var payload CloudflaredVersionRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid cloudflared version payload", &payload) {
		return
	}

	response, err := s.cloudflared.CloudflaredVersion(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleCloudflaredExec(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}
	if s.cloudflared == nil {
		writeJSONError(writer, http.StatusNotImplemented, "cloudflared connector service is not configured", correlationID)
		return
	}

	var payload CloudflaredExecRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid cloudflared exec payload", &payload) {
		return
	}

	response, err := s.cloudflared.CloudflaredExec(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}
