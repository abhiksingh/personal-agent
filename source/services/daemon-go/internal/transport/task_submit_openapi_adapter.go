package transport

import openapitypes "personalagent/runtime/internal/transport/openapitypes"

func toOpenAPITaskSubmitRequest(request SubmitTaskRequest) openapitypes.TaskSubmitRequest {
	result := openapitypes.TaskSubmitRequest{
		WorkspaceId:             request.WorkspaceID,
		RequestedByActorId:      request.RequestedByActorID,
		SubjectPrincipalActorId: request.SubjectPrincipalActorID,
		Title:                   request.Title,
	}
	if request.Description != "" {
		description := request.Description
		result.Description = &description
	}
	if request.TaskClass != "" {
		taskClass := request.TaskClass
		result.TaskClass = &taskClass
	}
	return result
}

func fromOpenAPITaskSubmitRequest(request openapitypes.TaskSubmitRequest) SubmitTaskRequest {
	result := SubmitTaskRequest{
		WorkspaceID:             request.WorkspaceId,
		RequestedByActorID:      request.RequestedByActorId,
		SubjectPrincipalActorID: request.SubjectPrincipalActorId,
		Title:                   request.Title,
	}
	if request.Description != nil {
		result.Description = *request.Description
	}
	if request.TaskClass != nil {
		result.TaskClass = *request.TaskClass
	}
	return result
}

func toOpenAPITaskSubmitResponse(response SubmitTaskResponse) openapitypes.TaskSubmitResponse {
	return openapitypes.TaskSubmitResponse{
		TaskId:        response.TaskID,
		RunId:         response.RunID,
		State:         response.State,
		CorrelationId: response.CorrelationID,
	}
}

func fromOpenAPITaskSubmitResponse(response openapitypes.TaskSubmitResponse) SubmitTaskResponse {
	return SubmitTaskResponse{
		TaskID:        response.TaskId,
		RunID:         response.RunId,
		State:         response.State,
		CorrelationID: response.CorrelationId,
	}
}
