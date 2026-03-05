package transport

import (
	openapitypes "personalagent/runtime/internal/transport/openapitypes"
)

func isEmptyChatTurnChannelContext(channel ChatTurnChannelContext) bool {
	return channel.ChannelID == "" && channel.ConnectorID == "" && channel.ThreadID == ""
}

func toOpenAPIChatTurnChannelContext(channel ChatTurnChannelContext) openapitypes.ChatTurnChannelContext {
	return openapitypes.ChatTurnChannelContext{
		ChannelId:   optionalStringPointer(channel.ChannelID),
		ConnectorId: optionalStringPointer(channel.ConnectorID),
		ThreadId:    optionalStringPointer(channel.ThreadID),
	}
}

func fromOpenAPIChatTurnChannelContext(channel *openapitypes.ChatTurnChannelContext) ChatTurnChannelContext {
	if channel == nil {
		return ChatTurnChannelContext{}
	}
	return ChatTurnChannelContext{
		ChannelID:   derefString(channel.ChannelId),
		ConnectorID: derefString(channel.ConnectorId),
		ThreadID:    derefString(channel.ThreadId),
	}
}

func toOpenAPIChatTurnToolCatalogEntry(entry ChatTurnToolCatalogEntry) openapitypes.ChatTurnToolCatalogEntry {
	var capabilityKeys *[]string
	if len(entry.CapabilityKeys) > 0 {
		capabilityKeys = toOpenAPIStringSlicePointer(entry.CapabilityKeys)
	}
	return openapitypes.ChatTurnToolCatalogEntry{
		Name:           optionalStringPointer(entry.Name),
		Description:    optionalStringPointer(entry.Description),
		CapabilityKeys: capabilityKeys,
		InputSchema:    optionalAnyMapPointer(entry.InputSchema),
	}
}

func fromOpenAPIChatTurnToolCatalogEntry(entry openapitypes.ChatTurnToolCatalogEntry) ChatTurnToolCatalogEntry {
	return ChatTurnToolCatalogEntry{
		Name:           derefString(entry.Name),
		Description:    derefString(entry.Description),
		CapabilityKeys: fromOpenAPIStringSlicePointer(entry.CapabilityKeys),
		InputSchema:    fromOpenAPIAnyMap(entry.InputSchema),
	}
}

func toOpenAPIChatTurnItem(item ChatTurnItem) openapitypes.ChatTurnItem {
	metadataMap := item.Metadata.AsMap()
	return openapitypes.ChatTurnItem{
		ItemId:            optionalStringPointer(item.ItemID),
		Type:              stringPointer(item.Type),
		Role:              optionalStringPointer(item.Role),
		Status:            optionalStringPointer(item.Status),
		Content:           optionalStringPointer(item.Content),
		ToolName:          optionalStringPointer(item.ToolName),
		ToolCallId:        optionalStringPointer(item.ToolCallID),
		Arguments:         optionalAnyMapPointer(item.Arguments),
		Output:            optionalAnyMapPointer(item.Output),
		ErrorCode:         optionalStringPointer(item.ErrorCode),
		ErrorMessage:      optionalStringPointer(item.ErrorMessage),
		ApprovalRequestId: optionalStringPointer(item.ApprovalRequestID),
		Metadata:          optionalAnyMapPointer(metadataMap),
	}
}

func fromOpenAPIChatTurnItem(item openapitypes.ChatTurnItem) ChatTurnItem {
	return ChatTurnItem{
		ItemID:            derefString(item.ItemId),
		Type:              derefString(item.Type),
		Role:              derefString(item.Role),
		Status:            derefString(item.Status),
		Content:           derefString(item.Content),
		ToolName:          derefString(item.ToolName),
		ToolCallID:        derefString(item.ToolCallId),
		Arguments:         fromOpenAPIAnyMap(item.Arguments),
		Output:            fromOpenAPIAnyMap(item.Output),
		ErrorCode:         derefString(item.ErrorCode),
		ErrorMessage:      derefString(item.ErrorMessage),
		ApprovalRequestID: derefString(item.ApprovalRequestId),
		Metadata:          ChatTurnItemMetadataFromMap(fromOpenAPIAnyMap(item.Metadata)),
	}
}

func toOpenAPIChatTurnTaskRunCorrelation(correlation ChatTurnTaskRunCorrelation) openapitypes.ChatTurnTaskRunCorrelation {
	return openapitypes.ChatTurnTaskRunCorrelation{
		Available: boolPointer(correlation.Available),
		Source:    optionalStringPointer(correlation.Source),
		TaskId:    optionalStringPointer(correlation.TaskID),
		RunId:     optionalStringPointer(correlation.RunID),
		TaskState: optionalStringPointer(correlation.TaskState),
		RunState:  optionalStringPointer(correlation.RunState),
	}
}

func fromOpenAPIChatTurnTaskRunCorrelation(correlation *openapitypes.ChatTurnTaskRunCorrelation) ChatTurnTaskRunCorrelation {
	if correlation == nil {
		return ChatTurnTaskRunCorrelation{}
	}
	return ChatTurnTaskRunCorrelation{
		Available: derefBool(correlation.Available),
		Source:    derefString(correlation.Source),
		TaskID:    derefString(correlation.TaskId),
		RunID:     derefString(correlation.RunId),
		TaskState: derefString(correlation.TaskState),
		RunState:  derefString(correlation.RunState),
	}
}

func toOpenAPIChatTurnRequest(request ChatTurnRequest) openapitypes.ChatTurnRequest {
	items := make([]openapitypes.ChatTurnItem, 0, len(request.Items))
	for _, item := range request.Items {
		items = append(items, toOpenAPIChatTurnItem(item))
	}

	var channel *openapitypes.ChatTurnChannelContext
	if !isEmptyChatTurnChannelContext(request.Channel) {
		converted := toOpenAPIChatTurnChannelContext(request.Channel)
		channel = &converted
	}

	var toolCatalog *[]openapitypes.ChatTurnToolCatalogEntry
	if len(request.ToolCatalog) > 0 {
		converted := make([]openapitypes.ChatTurnToolCatalogEntry, 0, len(request.ToolCatalog))
		for _, entry := range request.ToolCatalog {
			converted = append(converted, toOpenAPIChatTurnToolCatalogEntry(entry))
		}
		toolCatalog = &converted
	}

	return openapitypes.ChatTurnRequest{
		WorkspaceId:        stringPointer(request.WorkspaceID),
		TaskClass:          optionalStringPointer(request.TaskClass),
		RequestedByActorId: optionalStringPointer(request.RequestedByActorID),
		SubjectActorId:     optionalStringPointer(request.SubjectActorID),
		ActingAsActorId:    optionalStringPointer(request.ActingAsActorID),
		Provider:           optionalStringPointer(request.ProviderOverride),
		Model:              optionalStringPointer(request.ModelOverride),
		SystemPrompt:       optionalStringPointer(request.SystemPrompt),
		Channel:            channel,
		ToolCatalog:        toolCatalog,
		Items:              &items,
	}
}

func fromOpenAPIChatTurnRequest(request openapitypes.ChatTurnRequest) ChatTurnRequest {
	result := ChatTurnRequest{
		WorkspaceID:        derefString(request.WorkspaceId),
		TaskClass:          derefString(request.TaskClass),
		RequestedByActorID: derefString(request.RequestedByActorId),
		SubjectActorID:     derefString(request.SubjectActorId),
		ActingAsActorID:    derefString(request.ActingAsActorId),
		ProviderOverride:   derefString(request.Provider),
		ModelOverride:      derefString(request.Model),
		SystemPrompt:       derefString(request.SystemPrompt),
		Channel:            fromOpenAPIChatTurnChannelContext(request.Channel),
	}
	if request.ToolCatalog != nil {
		result.ToolCatalog = make([]ChatTurnToolCatalogEntry, 0, len(*request.ToolCatalog))
		for _, entry := range *request.ToolCatalog {
			result.ToolCatalog = append(result.ToolCatalog, fromOpenAPIChatTurnToolCatalogEntry(entry))
		}
	}
	if request.Items != nil {
		result.Items = make([]ChatTurnItem, 0, len(*request.Items))
		for _, item := range *request.Items {
			result.Items = append(result.Items, fromOpenAPIChatTurnItem(item))
		}
	}
	return result
}

func toOpenAPIChatTurnResponse(response ChatTurnResponse) openapitypes.ChatTurnResponse {
	items := make([]openapitypes.ChatTurnItem, 0, len(response.Items))
	for _, item := range response.Items {
		items = append(items, toOpenAPIChatTurnItem(item))
	}
	convertedCorrelation := toOpenAPIChatTurnTaskRunCorrelation(response.TaskRunCorrelation)

	var channel *openapitypes.ChatTurnChannelContext
	if !isEmptyChatTurnChannelContext(response.Channel) {
		converted := toOpenAPIChatTurnChannelContext(response.Channel)
		channel = &converted
	}

	return openapitypes.ChatTurnResponse{
		WorkspaceId:                  stringPointer(response.WorkspaceID),
		TaskClass:                    stringPointer(response.TaskClass),
		Provider:                     stringPointer(response.Provider),
		ModelKey:                     stringPointer(response.ModelKey),
		CorrelationId:                stringPointer(response.CorrelationID),
		ContractVersion:              optionalStringPointer(response.ContractVersion),
		TurnItemSchemaVersion:        optionalStringPointer(response.TurnItemSchemaVersion),
		RealtimeEventContractVersion: optionalStringPointer(response.RealtimeEventContractVersion),
		Channel:                      channel,
		Items:                        &items,
		TaskRunCorrelation:           &convertedCorrelation,
	}
}

func fromOpenAPIChatTurnResponse(response openapitypes.ChatTurnResponse) ChatTurnResponse {
	result := ChatTurnResponse{
		WorkspaceID:                  derefString(response.WorkspaceId),
		TaskClass:                    derefString(response.TaskClass),
		Provider:                     derefString(response.Provider),
		ModelKey:                     derefString(response.ModelKey),
		CorrelationID:                derefString(response.CorrelationId),
		ContractVersion:              derefString(response.ContractVersion),
		TurnItemSchemaVersion:        derefString(response.TurnItemSchemaVersion),
		RealtimeEventContractVersion: derefString(response.RealtimeEventContractVersion),
		Channel:                      fromOpenAPIChatTurnChannelContext(response.Channel),
		TaskRunCorrelation:           fromOpenAPIChatTurnTaskRunCorrelation(response.TaskRunCorrelation),
	}
	if response.Items == nil {
		return result
	}
	result.Items = make([]ChatTurnItem, 0, len(*response.Items))
	for _, item := range *response.Items {
		result.Items = append(result.Items, fromOpenAPIChatTurnItem(item))
	}
	return result
}
