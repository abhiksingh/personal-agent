package transport

import (
	openapitypes "personalagent/runtime/internal/transport/openapitypes"
)

func toOpenAPIModelRouteExplainRequest(request ModelRouteExplainRequest) openapitypes.ModelRouteExplainRequest {
	return openapitypes.ModelRouteExplainRequest{
		WorkspaceId:      stringPointer(request.WorkspaceID),
		TaskClass:        optionalStringPointer(request.TaskClass),
		PrincipalActorId: optionalStringPointer(request.PrincipalActorID),
	}
}

func fromOpenAPIModelRouteExplainRequest(request openapitypes.ModelRouteExplainRequest) ModelRouteExplainRequest {
	return ModelRouteExplainRequest{
		WorkspaceID:      derefString(request.WorkspaceId),
		TaskClass:        derefString(request.TaskClass),
		PrincipalActorID: derefString(request.PrincipalActorId),
	}
}

func toOpenAPIModelRouteDecision(decision ModelRouteDecision) openapitypes.ModelRouteDecision {
	return openapitypes.ModelRouteDecision{
		Step:       optionalStringPointer(decision.Step),
		Decision:   optionalStringPointer(decision.Decision),
		ReasonCode: optionalStringPointer(decision.ReasonCode),
		Provider:   optionalStringPointer(decision.Provider),
		ModelKey:   optionalStringPointer(decision.ModelKey),
		Note:       optionalStringPointer(decision.Note),
	}
}

func fromOpenAPIModelRouteDecision(decision openapitypes.ModelRouteDecision) ModelRouteDecision {
	return ModelRouteDecision{
		Step:       derefString(decision.Step),
		Decision:   derefString(decision.Decision),
		ReasonCode: derefString(decision.ReasonCode),
		Provider:   derefString(decision.Provider),
		ModelKey:   derefString(decision.ModelKey),
		Note:       derefString(decision.Note),
	}
}

func toOpenAPIModelRouteFallbackDecision(decision ModelRouteFallbackDecision) openapitypes.ModelRouteFallbackDecision {
	return openapitypes.ModelRouteFallbackDecision{
		Rank:       intPointer(decision.Rank),
		Provider:   optionalStringPointer(decision.Provider),
		ModelKey:   optionalStringPointer(decision.ModelKey),
		Selected:   boolPointer(decision.Selected),
		ReasonCode: optionalStringPointer(decision.ReasonCode),
	}
}

func fromOpenAPIModelRouteFallbackDecision(decision openapitypes.ModelRouteFallbackDecision) ModelRouteFallbackDecision {
	return ModelRouteFallbackDecision{
		Rank:       derefInt(decision.Rank),
		Provider:   derefString(decision.Provider),
		ModelKey:   derefString(decision.ModelKey),
		Selected:   derefBool(decision.Selected),
		ReasonCode: derefString(decision.ReasonCode),
	}
}

func toOpenAPIModelRouteExplainResponse(response ModelRouteExplainResponse) openapitypes.ModelRouteExplainResponse {
	decisions := make([]openapitypes.ModelRouteDecision, 0, len(response.Decisions))
	for _, decision := range response.Decisions {
		decisions = append(decisions, toOpenAPIModelRouteDecision(decision))
	}
	fallbackChain := make([]openapitypes.ModelRouteFallbackDecision, 0, len(response.FallbackChain))
	for _, fallback := range response.FallbackChain {
		fallbackChain = append(fallbackChain, toOpenAPIModelRouteFallbackDecision(fallback))
	}
	return openapitypes.ModelRouteExplainResponse{
		WorkspaceId:      stringPointer(response.WorkspaceID),
		TaskClass:        stringPointer(response.TaskClass),
		PrincipalActorId: optionalStringPointer(response.PrincipalActorID),
		SelectedProvider: stringPointer(response.SelectedProvider),
		SelectedModelKey: stringPointer(response.SelectedModelKey),
		SelectedSource:   stringPointer(response.SelectedSource),
		Summary:          stringPointer(response.Summary),
		Explanations:     toOpenAPIStringSlicePointer(response.Explanations),
		ReasonCodes:      toOpenAPIStringSlicePointer(response.ReasonCodes),
		Decisions:        &decisions,
		FallbackChain:    &fallbackChain,
	}
}

func fromOpenAPIModelRouteExplainResponse(response openapitypes.ModelRouteExplainResponse) ModelRouteExplainResponse {
	result := ModelRouteExplainResponse{
		WorkspaceID:      derefString(response.WorkspaceId),
		TaskClass:        derefString(response.TaskClass),
		PrincipalActorID: derefString(response.PrincipalActorId),
		SelectedProvider: derefString(response.SelectedProvider),
		SelectedModelKey: derefString(response.SelectedModelKey),
		SelectedSource:   derefString(response.SelectedSource),
		Summary:          derefString(response.Summary),
		Explanations:     fromOpenAPIStringSlicePointer(response.Explanations),
		ReasonCodes:      fromOpenAPIStringSlicePointer(response.ReasonCodes),
	}
	if response.Decisions != nil {
		result.Decisions = make([]ModelRouteDecision, 0, len(*response.Decisions))
		for _, decision := range *response.Decisions {
			result.Decisions = append(result.Decisions, fromOpenAPIModelRouteDecision(decision))
		}
	}
	if response.FallbackChain != nil {
		result.FallbackChain = make([]ModelRouteFallbackDecision, 0, len(*response.FallbackChain))
		for _, decision := range *response.FallbackChain {
			result.FallbackChain = append(result.FallbackChain, fromOpenAPIModelRouteFallbackDecision(decision))
		}
	}
	return result
}

func toOpenAPIWorkflowRouteMetadata(metadata WorkflowRouteMetadata) openapitypes.WorkflowRouteMetadata {
	return openapitypes.WorkflowRouteMetadata{
		Available:       boolPointer(metadata.Available),
		TaskClass:       optionalStringPointer(metadata.TaskClass),
		TaskClassSource: optionalStringPointer(metadata.TaskClassSource),
		Provider:        optionalStringPointer(metadata.Provider),
		ModelKey:        optionalStringPointer(metadata.ModelKey),
		RouteSource:     optionalStringPointer(metadata.RouteSource),
		Notes:           optionalStringPointer(metadata.Notes),
	}
}

func fromOpenAPIWorkflowRouteMetadata(metadata *openapitypes.WorkflowRouteMetadata) WorkflowRouteMetadata {
	if metadata == nil {
		return WorkflowRouteMetadata{}
	}
	return WorkflowRouteMetadata{
		Available:       derefBool(metadata.Available),
		TaskClass:       derefString(metadata.TaskClass),
		TaskClassSource: derefString(metadata.TaskClassSource),
		Provider:        derefString(metadata.Provider),
		ModelKey:        derefString(metadata.ModelKey),
		RouteSource:     derefString(metadata.RouteSource),
		Notes:           derefString(metadata.Notes),
	}
}

func toOpenAPITaskRunActionAvailability(actions TaskRunActionAvailability) openapitypes.TaskRunActionAvailability {
	return openapitypes.TaskRunActionAvailability{
		CanCancel:  actions.CanCancel,
		CanRetry:   actions.CanRetry,
		CanRequeue: actions.CanRequeue,
	}
}

func fromOpenAPITaskRunActionAvailability(actions *openapitypes.TaskRunActionAvailability) TaskRunActionAvailability {
	if actions == nil {
		return TaskRunActionAvailability{}
	}
	return TaskRunActionAvailability{
		CanCancel:  actions.CanCancel,
		CanRetry:   actions.CanRetry,
		CanRequeue: actions.CanRequeue,
	}
}

func toOpenAPIApprovalInboxRequest(request ApprovalInboxRequest) openapitypes.ApprovalInboxRequest {
	return openapitypes.ApprovalInboxRequest{
		WorkspaceId:  stringPointer(request.WorkspaceID),
		IncludeFinal: boolPointer(request.IncludeFinal),
		Limit:        intPointer(request.Limit),
		State:        optionalStringPointer(request.State),
	}
}

func fromOpenAPIApprovalInboxRequest(request openapitypes.ApprovalInboxRequest) ApprovalInboxRequest {
	return ApprovalInboxRequest{
		WorkspaceID:  derefString(request.WorkspaceId),
		IncludeFinal: derefBool(request.IncludeFinal),
		Limit:        derefInt(request.Limit),
		State:        derefString(request.State),
	}
}

func toOpenAPIApprovalInboxItem(item ApprovalInboxItem) openapitypes.ApprovalInboxItem {
	return openapitypes.ApprovalInboxItem{
		ApprovalRequestId:       optionalStringPointer(item.ApprovalRequestID),
		WorkspaceId:             optionalStringPointer(item.WorkspaceID),
		State:                   optionalStringPointer(item.State),
		Decision:                optionalStringPointer(item.Decision),
		RequestedPhrase:         optionalStringPointer(item.RequestedPhrase),
		RiskLevel:               optionalStringPointer(item.RiskLevel),
		RiskRationale:           optionalStringPointer(item.RiskRationale),
		RequestedAt:             optionalStringPointer(item.RequestedAt),
		DecidedAt:               optionalStringPointer(item.DecidedAt),
		DecisionByActorId:       optionalStringPointer(item.DecisionByActorID),
		DecisionRationale:       optionalStringPointer(item.DecisionRationale),
		TaskId:                  optionalStringPointer(item.TaskID),
		TaskTitle:               optionalStringPointer(item.TaskTitle),
		TaskState:               optionalStringPointer(item.TaskState),
		RunId:                   optionalStringPointer(item.RunID),
		RunState:                optionalStringPointer(item.RunState),
		StepId:                  optionalStringPointer(item.StepID),
		StepName:                optionalStringPointer(item.StepName),
		StepCapabilityKey:       optionalStringPointer(item.StepCapabilityKey),
		RequestedByActorId:      optionalStringPointer(item.RequestedByActorID),
		SubjectPrincipalActorId: optionalStringPointer(item.SubjectPrincipalActorID),
		ActingAsActorId:         optionalStringPointer(item.ActingAsActorID),
		Route: func() *openapitypes.WorkflowRouteMetadata {
			route := toOpenAPIWorkflowRouteMetadata(item.Route)
			return &route
		}(),
	}
}

func fromOpenAPIApprovalInboxItem(item openapitypes.ApprovalInboxItem) ApprovalInboxItem {
	return ApprovalInboxItem{
		ApprovalRequestID:       derefString(item.ApprovalRequestId),
		WorkspaceID:             derefString(item.WorkspaceId),
		State:                   derefString(item.State),
		Decision:                derefString(item.Decision),
		RequestedPhrase:         derefString(item.RequestedPhrase),
		RiskLevel:               derefString(item.RiskLevel),
		RiskRationale:           derefString(item.RiskRationale),
		RequestedAt:             derefString(item.RequestedAt),
		DecidedAt:               derefString(item.DecidedAt),
		DecisionByActorID:       derefString(item.DecisionByActorId),
		DecisionRationale:       derefString(item.DecisionRationale),
		TaskID:                  derefString(item.TaskId),
		TaskTitle:               derefString(item.TaskTitle),
		TaskState:               derefString(item.TaskState),
		RunID:                   derefString(item.RunId),
		RunState:                derefString(item.RunState),
		StepID:                  derefString(item.StepId),
		StepName:                derefString(item.StepName),
		StepCapabilityKey:       derefString(item.StepCapabilityKey),
		RequestedByActorID:      derefString(item.RequestedByActorId),
		SubjectPrincipalActorID: derefString(item.SubjectPrincipalActorId),
		ActingAsActorID:         derefString(item.ActingAsActorId),
		Route:                   fromOpenAPIWorkflowRouteMetadata(item.Route),
	}
}

func toOpenAPIApprovalInboxResponse(response ApprovalInboxResponse) openapitypes.ApprovalInboxResponse {
	items := make([]openapitypes.ApprovalInboxItem, 0, len(response.Approvals))
	for _, item := range response.Approvals {
		items = append(items, toOpenAPIApprovalInboxItem(item))
	}
	return openapitypes.ApprovalInboxResponse{
		WorkspaceId: stringPointer(response.WorkspaceID),
		Approvals:   &items,
	}
}

func fromOpenAPIApprovalInboxResponse(response openapitypes.ApprovalInboxResponse) ApprovalInboxResponse {
	result := ApprovalInboxResponse{
		WorkspaceID: derefString(response.WorkspaceId),
	}
	if response.Approvals == nil {
		return result
	}
	result.Approvals = make([]ApprovalInboxItem, 0, len(*response.Approvals))
	for _, item := range *response.Approvals {
		result.Approvals = append(result.Approvals, fromOpenAPIApprovalInboxItem(item))
	}
	return result
}

func toOpenAPITaskRunListRequest(request TaskRunListRequest) openapitypes.TaskRunListRequest {
	return openapitypes.TaskRunListRequest{
		WorkspaceId: stringPointer(request.WorkspaceID),
		State:       optionalStringPointer(request.State),
		Limit:       intPointer(request.Limit),
	}
}

func fromOpenAPITaskRunListRequest(request openapitypes.TaskRunListRequest) TaskRunListRequest {
	return TaskRunListRequest{
		WorkspaceID: derefString(request.WorkspaceId),
		State:       derefString(request.State),
		Limit:       derefInt(request.Limit),
	}
}

func toOpenAPITaskRunListItem(item TaskRunListItem) openapitypes.TaskRunListItem {
	return openapitypes.TaskRunListItem{
		TaskId:                  optionalStringPointer(item.TaskID),
		RunId:                   optionalStringPointer(item.RunID),
		WorkspaceId:             optionalStringPointer(item.WorkspaceID),
		Title:                   optionalStringPointer(item.Title),
		TaskState:               optionalStringPointer(item.TaskState),
		RunState:                optionalStringPointer(item.RunState),
		Priority:                intPointer(item.Priority),
		RequestedByActorId:      optionalStringPointer(item.RequestedByActorID),
		SubjectPrincipalActorId: optionalStringPointer(item.SubjectPrincipalActorID),
		ActingAsActorId:         optionalStringPointer(item.ActingAsActorID),
		LastError:               optionalStringPointer(item.LastError),
		TaskCreatedAt:           optionalStringPointer(item.TaskCreatedAt),
		TaskUpdatedAt:           optionalStringPointer(item.TaskUpdatedAt),
		RunCreatedAt:            optionalStringPointer(item.RunCreatedAt),
		RunUpdatedAt:            optionalStringPointer(item.RunUpdatedAt),
		StartedAt:               optionalStringPointer(item.StartedAt),
		FinishedAt:              optionalStringPointer(item.FinishedAt),
		Actions: func() *openapitypes.TaskRunActionAvailability {
			actions := toOpenAPITaskRunActionAvailability(item.Actions)
			return &actions
		}(),
		Route: func() *openapitypes.WorkflowRouteMetadata {
			route := toOpenAPIWorkflowRouteMetadata(item.Route)
			return &route
		}(),
	}
}

func fromOpenAPITaskRunListItem(item openapitypes.TaskRunListItem) TaskRunListItem {
	return TaskRunListItem{
		TaskID:                  derefString(item.TaskId),
		RunID:                   derefString(item.RunId),
		WorkspaceID:             derefString(item.WorkspaceId),
		Title:                   derefString(item.Title),
		TaskState:               derefString(item.TaskState),
		RunState:                derefString(item.RunState),
		Priority:                derefInt(item.Priority),
		RequestedByActorID:      derefString(item.RequestedByActorId),
		SubjectPrincipalActorID: derefString(item.SubjectPrincipalActorId),
		ActingAsActorID:         derefString(item.ActingAsActorId),
		LastError:               derefString(item.LastError),
		TaskCreatedAt:           derefString(item.TaskCreatedAt),
		TaskUpdatedAt:           derefString(item.TaskUpdatedAt),
		RunCreatedAt:            derefString(item.RunCreatedAt),
		RunUpdatedAt:            derefString(item.RunUpdatedAt),
		StartedAt:               derefString(item.StartedAt),
		FinishedAt:              derefString(item.FinishedAt),
		Actions:                 fromOpenAPITaskRunActionAvailability(item.Actions),
		Route:                   fromOpenAPIWorkflowRouteMetadata(item.Route),
	}
}

func toOpenAPITaskRunListResponse(response TaskRunListResponse) openapitypes.TaskRunListResponse {
	items := make([]openapitypes.TaskRunListItem, 0, len(response.Items))
	for _, item := range response.Items {
		items = append(items, toOpenAPITaskRunListItem(item))
	}
	return openapitypes.TaskRunListResponse{
		WorkspaceId: stringPointer(response.WorkspaceID),
		Items:       &items,
	}
}

func fromOpenAPITaskRunListResponse(response openapitypes.TaskRunListResponse) TaskRunListResponse {
	result := TaskRunListResponse{
		WorkspaceID: derefString(response.WorkspaceId),
	}
	if response.Items == nil {
		return result
	}
	result.Items = make([]TaskRunListItem, 0, len(*response.Items))
	for _, item := range *response.Items {
		result.Items = append(result.Items, fromOpenAPITaskRunListItem(item))
	}
	return result
}
