package transport

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	openapitypes "personalagent/runtime/internal/transport/openapitypes"
)

type chatTurnHistoryService interface {
	QueryChatTurnHistory(ctx context.Context, request ChatTurnHistoryRequest) (ChatTurnHistoryResponse, error)
}

type chatPersonaPolicyService interface {
	GetChatPersonaPolicy(ctx context.Context, request ChatPersonaPolicyRequest) (ChatPersonaPolicyResponse, error)
	UpsertChatPersonaPolicy(ctx context.Context, request ChatPersonaPolicyUpsertRequest) (ChatPersonaPolicyResponse, error)
}

type chatTurnExplainService interface {
	ExplainChatTurn(ctx context.Context, request ChatTurnExplainRequest) (ChatTurnExplainResponse, error)
}

func (s *Server) handleChatTurnHistory(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}
	historyService, ok := s.chat.(chatTurnHistoryService)
	if !ok {
		writeJSONError(writer, http.StatusNotImplemented, "chat history service is not configured", correlationID)
		return
	}

	var payload ChatTurnHistoryRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid chat history payload", &payload) {
		return
	}
	response, err := historyService.QueryChatTurnHistory(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleChatTurnExplain(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}
	explainService, ok := s.chat.(chatTurnExplainService)
	if !ok {
		writeJSONError(writer, http.StatusNotImplemented, "chat explain service is not configured", correlationID)
		return
	}

	var payload ChatTurnExplainRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid chat turn explain payload", &payload) {
		return
	}
	response, err := explainService.ExplainChatTurn(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleChatPersonaPolicyGet(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}
	personaService, ok := s.chat.(chatPersonaPolicyService)
	if !ok {
		writeJSONError(writer, http.StatusNotImplemented, "chat persona policy service is not configured", correlationID)
		return
	}

	var payload ChatPersonaPolicyRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid chat persona get payload", &payload) {
		return
	}
	response, err := personaService.GetChatPersonaPolicy(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleChatPersonaPolicySet(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}
	personaService, ok := s.chat.(chatPersonaPolicyService)
	if !ok {
		writeJSONError(writer, http.StatusNotImplemented, "chat persona policy service is not configured", correlationID)
		return
	}

	var payload ChatPersonaPolicyUpsertRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid chat persona set payload", &payload) {
		return
	}
	scopeType, scopeKey := controlRateLimitScopeFromActorOrWorkspace(payload.PrincipalActorID, payload.WorkspaceID)
	if !s.enforceControlRateLimit(writer, request, correlationID, controlRateLimitKeyChatPersonaSet, scopeType, scopeKey) {
		return
	}
	response, err := personaService.UpsertChatPersonaPolicy(request.Context(), payload)
	if err != nil {
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}
	writeJSON(writer, http.StatusOK, response, correlationID)
}

func (s *Server) handleChatTurn(writer http.ResponseWriter, request *http.Request) {
	correlationID, ok := s.requireAuthorizedMethod(writer, request, http.MethodPost)
	if !ok {
		return
	}

	var openapiPayload openapitypes.ChatTurnRequest
	if !s.decodeRequestBody(writer, request, correlationID, "invalid chat turn payload", &openapiPayload) {
		return
	}
	payload := fromOpenAPIChatTurnRequest(openapiPayload)
	scopeType, scopeKey := controlRateLimitScopeFromActorOrWorkspace(
		firstNonEmptyTrimmed(payload.ActingAsActorID, payload.SubjectActorID, payload.RequestedByActorID),
		payload.WorkspaceID,
	)
	if !s.enforceControlRateLimit(writer, request, correlationID, controlRateLimitKeyChatTurn, scopeType, scopeKey) {
		return
	}
	if s.chat == nil {
		writeJSONError(writer, http.StatusNotImplemented, "chat service is not configured", correlationID)
		return
	}

	response, err := s.chat.ChatTurn(request.Context(), payload, correlationID, func(delta string) {
		if s.broker == nil || delta == "" {
			return
		}
		now := time.Now().UTC()
		_ = s.broker.Publish(newChatRealtimeEventEnvelope("turn_item_delta", correlationID, now, RealtimeEventPayload{
			ItemType: "assistant_message",
			Delta:    delta,
		}))
	})
	if err != nil {
		s.publishChatErrorEvent(correlationID, err)
		writeJSONErrorFromError(writer, http.StatusBadRequest, err, correlationID)
		return
	}

	if strings.TrimSpace(response.CorrelationID) == "" {
		response.CorrelationID = correlationID
	}
	if strings.TrimSpace(response.TaskRunCorrelation.Source) == "" {
		response.TaskRunCorrelation = ChatTurnTaskRunCorrelation{
			Available: false,
			Source:    "none",
		}
	}
	ensureChatTurnContractVersions(&response)
	s.publishTurnItemLifecycleEvents(correlationID, response.Items)
	s.publishChatCompletionEvent(correlationID, response)
	writeJSON(writer, http.StatusOK, toOpenAPIChatTurnResponse(response), correlationID)
}

func (s *Server) publishChatCompletionEvent(correlationID string, response ChatTurnResponse) {
	if s == nil || s.broker == nil {
		return
	}
	assistantEmpty := true
	toolCalls := 0
	approvalItems := 0
	for _, item := range response.Items {
		switch strings.ToLower(strings.TrimSpace(item.Type)) {
		case "assistant_message":
			if strings.TrimSpace(item.Content) != "" {
				assistantEmpty = false
			}
		case "tool_call":
			toolCalls++
		case "approval_request":
			approvalItems++
		}
	}
	payload := RealtimeEventPayload{
		TaskClass:      strings.TrimSpace(response.TaskClass),
		Provider:       strings.TrimSpace(response.Provider),
		ModelKey:       strings.TrimSpace(response.ModelKey),
		AssistantEmpty: boolPointer(assistantEmpty),
		ItemCount:      intPointer(len(response.Items)),
		ToolCallCount:  intPointer(toolCalls),
		ApprovalCount:  intPointer(approvalItems),
	}
	if response.TaskRunCorrelation.Available {
		payload.TaskID = strings.TrimSpace(response.TaskRunCorrelation.TaskID)
		payload.RunID = strings.TrimSpace(response.TaskRunCorrelation.RunID)
		payload.TaskState = strings.TrimSpace(response.TaskRunCorrelation.TaskState)
		payload.RunState = strings.TrimSpace(response.TaskRunCorrelation.RunState)
	}

	_ = s.broker.Publish(newChatRealtimeEventEnvelope("chat_completed", correlationID, time.Now().UTC(), payload))
}

func (s *Server) publishChatErrorEvent(correlationID string, err error) {
	if s == nil || s.broker == nil || err == nil {
		return
	}
	_ = s.broker.Publish(newChatRealtimeEventEnvelope("chat_error", correlationID, time.Now().UTC(), RealtimeEventPayload{
		Message: strings.TrimSpace(err.Error()),
	}))
}

func (s *Server) publishTurnItemLifecycleEvents(correlationID string, items []ChatTurnItem) {
	if s == nil || s.broker == nil || len(items) == 0 {
		return
	}
	for index, item := range items {
		itemType := strings.ToLower(strings.TrimSpace(item.Type))
		itemID := strings.TrimSpace(item.ItemID)
		if itemID == "" {
			itemID = fmt.Sprintf("item-%d", index+1)
		}
		now := time.Now().UTC()
		basePayload := RealtimeEventPayload{
			ItemID:    itemID,
			ItemIndex: intPointer(index),
			ItemType:  itemType,
			Status:    strings.ToLower(strings.TrimSpace(item.Status)),
		}
		_ = s.broker.Publish(newChatRealtimeEventEnvelope("turn_item_started", correlationID, now, basePayload))

		if itemType == "tool_call" {
			_ = s.broker.Publish(newChatRealtimeEventEnvelope("tool_call_started", correlationID, now, RealtimeEventPayload{
				ItemID:     itemID,
				ItemIndex:  intPointer(index),
				ToolName:   strings.TrimSpace(item.ToolName),
				ToolCallID: strings.TrimSpace(item.ToolCallID),
				Arguments:  item.Arguments,
				Metadata:   item.Metadata,
			}))
		}
		if itemType == "tool_result" {
			_ = s.broker.Publish(newChatRealtimeEventEnvelope("tool_call_output", correlationID, now, RealtimeEventPayload{
				ItemID:     itemID,
				ItemIndex:  intPointer(index),
				ToolName:   strings.TrimSpace(item.ToolName),
				ToolCallID: strings.TrimSpace(item.ToolCallID),
				Status:     strings.ToLower(strings.TrimSpace(item.Status)),
				Output:     item.Output,
				ErrorCode:  strings.TrimSpace(item.ErrorCode),
				Error:      strings.TrimSpace(item.ErrorMessage),
				Metadata:   item.Metadata,
			}))
			_ = s.broker.Publish(newChatRealtimeEventEnvelope("tool_call_completed", correlationID, now, RealtimeEventPayload{
				ItemID:     itemID,
				ItemIndex:  intPointer(index),
				ToolName:   strings.TrimSpace(item.ToolName),
				ToolCallID: strings.TrimSpace(item.ToolCallID),
				Status:     strings.ToLower(strings.TrimSpace(item.Status)),
				Metadata:   item.Metadata,
			}))
		}

		_ = s.broker.Publish(newChatRealtimeEventEnvelope("turn_item_completed", correlationID, now, basePayload))
	}
}

func ensureChatTurnContractVersions(response *ChatTurnResponse) {
	if response == nil {
		return
	}
	if strings.TrimSpace(response.ContractVersion) == "" {
		response.ContractVersion = ChatTurnContractVersionV2
	}
	if strings.TrimSpace(response.TurnItemSchemaVersion) == "" {
		response.TurnItemSchemaVersion = ChatTurnItemSchemaVersionV1
	}
	if strings.TrimSpace(response.RealtimeEventContractVersion) == "" {
		response.RealtimeEventContractVersion = ChatRealtimeLifecycleContractVersionV2
	}
}

func newChatRealtimeEventEnvelope(eventType string, correlationID string, occurredAt time.Time, payload RealtimeEventPayload) RealtimeEventEnvelope {
	return RealtimeEventEnvelope{
		EventID:                mustRandomID(),
		EventType:              strings.TrimSpace(eventType),
		OccurredAt:             occurredAt.UTC(),
		CorrelationID:          strings.TrimSpace(correlationID),
		ContractVersion:        ChatRealtimeLifecycleContractVersionV2,
		LifecycleSchemaVersion: ChatTurnItemSchemaVersionV1,
		Payload:                payload,
	}
}
