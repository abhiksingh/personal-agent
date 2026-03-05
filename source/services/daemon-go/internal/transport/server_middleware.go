package transport

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func (s *Server) requireAuthorizedMethod(writer http.ResponseWriter, request *http.Request, method string) (string, bool) {
	if request.Method != method {
		writeMethodNotAllowed(writer, method)
		return "", false
	}
	return s.authorize(writer, request)
}

func decodeJSONBodyStrict(reader io.Reader, target any) error {
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != nil && err != io.EOF {
		return err
	}
	return nil
}

func (s *Server) decodeRequestBody(writer http.ResponseWriter, request *http.Request, correlationID string, invalidMessage string, target any) bool {
	limitedReader := request.Body
	if s.config.RequestBodyBytesLimit > 0 {
		limitedReader = http.MaxBytesReader(writer, request.Body, s.config.RequestBodyBytesLimit)
	}

	if err := decodeJSONBodyStrict(limitedReader, target); err != nil {
		if isRequestBodyTooLarge(err) {
			writeJSONError(writer, http.StatusRequestEntityTooLarge, fmt.Sprintf("request body exceeds limit of %d bytes", s.config.RequestBodyBytesLimit), correlationID)
			return false
		}
		message := strings.TrimSpace(invalidMessage)
		if message == "" {
			message = "invalid request payload"
		}
		writeJSONError(writer, http.StatusBadRequest, message, correlationID)
		return false
	}
	return true
}

func isRequestBodyTooLarge(err error) bool {
	if err == nil {
		return false
	}
	var maxBytesError *http.MaxBytesError
	if errors.As(err, &maxBytesError) {
		return true
	}
	return strings.Contains(strings.ToLower(strings.TrimSpace(err.Error())), "request body too large")
}
