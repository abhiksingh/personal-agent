package transport

import (
	"net/http"
)

func (s *Server) handleTwilioIngestSMS(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.twilio == nil {
		writeJSONError(writer, http.StatusNotImplemented, "twilio channel service is not configured", correlationID)
		return
	}

	var payload TwilioIngestSMSRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid twilio ingest sms payload", &payload) {
		return
	}

	response, err := s.twilio.IngestTwilioSMS(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleTwilioIngestVoice(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writeMethodNotAllowed(writer, http.MethodPost)
		return
	}
	correlationID, ok := s.authorize(writer, request)
	if !ok {
		return
	}
	if s.twilio == nil {
		writeJSONError(writer, http.StatusNotImplemented, "twilio channel service is not configured", correlationID)
		return
	}

	var payload TwilioIngestVoiceRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid twilio ingest voice payload", &payload) {
		return
	}

	response, err := s.twilio.IngestTwilioVoice(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}
