package transport

import (
	"context"
	"fmt"
	"net/http"
	openapitypes "personalagent/runtime/internal/transport/openapitypes"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

func (s *Server) handleSubmitTask(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}

	var openapiPayload openapitypes.TaskSubmitRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid submit task payload", &openapiPayload) {
		return
	}

	scopeType, scopeKey := controlRateLimitScopeFromActorOrWorkspace(
		firstNonEmptyTrimmed(openapiPayload.SubjectPrincipalActorId, openapiPayload.RequestedByActorId),
		openapiPayload.WorkspaceId,
	)
	if !s.enforceControlRateLimit(writer, request, correlationID, controlRateLimitKeyTaskSubmit, scopeType, scopeKey) {
		return
	}

	payload := fromOpenAPITaskSubmitRequest(openapiPayload)
	response, err := s.backend.SubmitTask(request.Context(), payload, correlationID)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusAccepted, toOpenAPITaskSubmitResponse(response), correlationID)
}

func (s *Server) handleTaskStatus(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodGet)
	if !ok {
		return
	}

	taskID := strings.TrimSpace(strings.TrimPrefix(request.URL.Path, "/v1/tasks/"))
	if taskID == "" {
		writeJSONError(writer, http.StatusBadRequest, "task id is required", correlationID)
		return
	}

	response, err := s.backend.TaskStatus(request.Context(), taskID, correlationID)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusInternalServerError, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleTaskCancel(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}

	var payload TaskCancelRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid task cancel payload", &payload) {
		return
	}
	scopeType, scopeKey := controlRateLimitScopeFromActorOrWorkspace("", payload.WorkspaceID)
	if !s.enforceControlRateLimit(writer, request, correlationID, controlRateLimitKeyTaskCancel, scopeType, scopeKey) {
		return
	}

	response, err := s.backend.CancelTask(request.Context(), payload, correlationID)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusInternalServerError, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleTaskRetry(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}

	var payload TaskRetryRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid task retry payload", &payload) {
		return
	}
	scopeType, scopeKey := controlRateLimitScopeFromActorOrWorkspace("", payload.WorkspaceID)
	if !s.enforceControlRateLimit(writer, request, correlationID, controlRateLimitKeyTaskRetry, scopeType, scopeKey) {
		return
	}

	response, err := s.backend.RetryTask(request.Context(), payload, correlationID)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusInternalServerError, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleTaskRequeue(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}

	var payload TaskRequeueRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid task requeue payload", &payload) {
		return
	}
	scopeType, scopeKey := controlRateLimitScopeFromActorOrWorkspace("", payload.WorkspaceID)
	if !s.enforceControlRateLimit(writer, request, correlationID, controlRateLimitKeyTaskRequeue, scopeType, scopeKey) {
		return
	}

	response, err := s.backend.RequeueTask(request.Context(), payload, correlationID)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusInternalServerError, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleCapabilitySmoke(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodGet)
	if !ok {
		return
	}

	response, err := s.backend.CapabilitySmoke(request.Context(), correlationID)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusInternalServerError, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleRealtimeWS(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodGet)
	if !ok {
		return
	}
	if !s.realtimeOrigins.allowsRequest(request) {
		writeJSONError(writer, http.StatusForbidden, "realtime websocket origin is not allowed", correlationID)
		return
	}
	reserved, reserveFailure := s.reserveRealtimeSession()
	if !reserved {
		activeConnections, activeSubscriptions := s.realtimeSessionCounts()
		writeJSONErrorWithDetails(writer, http.StatusTooManyRequests, "realtime websocket capacity exceeded", correlationID, map[string]any{
			"category":             "realtime_capacity",
			"limit_type":           reserveFailure.limitType,
			"configured_limit":     reserveFailure.limit,
			"active_at_rejection":  reserveFailure.active,
			"active_connections":   activeConnections,
			"active_subscriptions": activeSubscriptions,
			"path":                 "/v1/realtime/ws",
		})
		return
	}
	defer s.releaseRealtimeSession()

	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return s.realtimeOrigins.allowsRequest(r) }}
	conn, err := upgrader.Upgrade(writer, request, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	conn.SetReadLimit(s.config.RealtimeReadLimitBytes)
	_ = conn.SetReadDeadline(time.Now().Add(s.config.RealtimePongTimeout))
	conn.SetPongHandler(func(_ string) error {
		return conn.SetReadDeadline(time.Now().Add(s.config.RealtimePongTimeout))
	})

	subID, subscription := s.broker.Subscribe(64)
	defer s.broker.Unsubscribe(subID)

	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		for {
			var signal ClientSignal
			if readErr := conn.ReadJSON(&signal); readErr != nil {
				return
			}
			signalCorrelationID := firstNonEmpty(signal.CorrelationID, correlationID)
			_ = s.broker.Publish(RealtimeEventEnvelope{
				EventID:       mustRandomID(),
				EventType:     "client_signal",
				OccurredAt:    time.Now().UTC(),
				CorrelationID: signalCorrelationID,
				Payload: RealtimeEventPayload{
					SignalType: signal.SignalType,
					TaskID:     signal.TaskID,
					RunID:      signal.RunID,
					Reason:     signal.Reason,
				},
			})
			_ = s.broker.Publish(s.evaluateClientSignal(request.Context(), signal, signalCorrelationID))
		}
	}()
	pingTicker := time.NewTicker(s.config.RealtimePingInterval)
	defer pingTicker.Stop()

	for {
		select {
		case <-request.Context().Done():
			return
		case <-readDone:
			return
		case <-pingTicker.C:
			if err := s.realtimeWriteControl(conn, websocket.PingMessage, nil); err != nil {
				return
			}
		case event, ok := <-subscription:
			if !ok {
				return
			}
			if event.OccurredAt.IsZero() {
				event.OccurredAt = time.Now().UTC()
			}
			if err := s.realtimeWriteJSON(conn, event); err != nil {
				return
			}
		}
	}
}

func (s *Server) evaluateClientSignal(ctx context.Context, signal ClientSignal, correlationID string) RealtimeEventEnvelope {
	now := time.Now().UTC()
	normalizedSignalType := strings.ToLower(strings.TrimSpace(signal.SignalType))
	normalizedTaskID := strings.TrimSpace(signal.TaskID)
	normalizedRunID := strings.TrimSpace(signal.RunID)
	normalizedReason := strings.TrimSpace(signal.Reason)

	payload := RealtimeEventPayload{
		SignalType: normalizedSignalType,
		TaskID:     normalizedTaskID,
		RunID:      normalizedRunID,
	}
	ackReason := ""
	accepted := false

	switch normalizedSignalType {
	case "cancel":
		if normalizedTaskID == "" && normalizedRunID == "" {
			ackReason = "cancel signal requires task_id or run_id"
			break
		}
		cancelResponse, err := s.backend.CancelTask(ctx, TaskCancelRequest{
			TaskID: normalizedTaskID,
			RunID:  normalizedRunID,
			Reason: normalizedReason,
		}, correlationID)
		if err != nil {
			ackReason = strings.TrimSpace(err.Error())
			break
		}
		accepted = true
		payload.WorkspaceID = strings.TrimSpace(cancelResponse.WorkspaceID)
		payload.TaskState = strings.TrimSpace(cancelResponse.TaskState)
		payload.RunState = strings.TrimSpace(cancelResponse.RunState)
		payload.Cancelled = boolPointer(cancelResponse.Cancelled)
		payload.AlreadyTerminal = boolPointer(cancelResponse.AlreadyTerminal)
		switch {
		case cancelResponse.Cancelled:
			ackReason = "cancelled"
		case cancelResponse.AlreadyTerminal:
			ackReason = "already_terminal"
		default:
			ackReason = "cancel request processed"
		}
	default:
		ackReason = fmt.Sprintf("unsupported signal_type %q", strings.TrimSpace(signal.SignalType))
	}

	payload.Accepted = boolPointer(accepted)
	payload.Reason = ackReason
	return RealtimeEventEnvelope{
		EventID:       mustRandomID(),
		EventType:     "client_signal_ack",
		OccurredAt:    now,
		CorrelationID: strings.TrimSpace(correlationID),
		Payload:       payload,
	}
}
