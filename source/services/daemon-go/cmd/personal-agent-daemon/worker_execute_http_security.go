package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	workerExecuteReadHeaderTimeout = 2 * time.Second
	workerExecuteReadTimeout       = 10 * time.Second
	workerExecuteWriteTimeout      = 2 * time.Minute
	workerExecuteIdleTimeout       = 30 * time.Second
	workerExecuteMaxBodyBytes      = int64(1 << 20)
)

func newWorkerExecuteHTTPServer(handler http.Handler) *http.Server {
	return &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: workerExecuteReadHeaderTimeout,
		ReadTimeout:       workerExecuteReadTimeout,
		WriteTimeout:      workerExecuteWriteTimeout,
		IdleTimeout:       workerExecuteIdleTimeout,
	}
}

func decodeWorkerExecuteJSONPayload(writer http.ResponseWriter, request *http.Request, target any, payloadLabel string) (int, error) {
	if target == nil {
		return http.StatusInternalServerError, fmt.Errorf("decode target is required")
	}
	label := strings.TrimSpace(payloadLabel)
	if label == "" {
		label = "execute"
	}

	request.Body = http.MaxBytesReader(writer, request.Body, workerExecuteMaxBodyBytes)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		return classifyWorkerExecuteDecodeError(err, label)
	}

	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err != nil {
			return classifyWorkerExecuteDecodeError(err, label)
		}
		return http.StatusBadRequest, fmt.Errorf("invalid %s payload: trailing content is not allowed", label)
	}

	return http.StatusOK, nil
}

func classifyWorkerExecuteDecodeError(err error, payloadLabel string) (int, error) {
	var maxBodyErr *http.MaxBytesError
	if errors.As(err, &maxBodyErr) {
		return http.StatusRequestEntityTooLarge, fmt.Errorf("%s payload exceeds %d bytes", payloadLabel, workerExecuteMaxBodyBytes)
	}
	return http.StatusBadRequest, fmt.Errorf("invalid %s payload: %w", payloadLabel, err)
}
