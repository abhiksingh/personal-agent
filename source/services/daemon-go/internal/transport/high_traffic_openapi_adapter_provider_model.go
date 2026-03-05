package transport

import (
	openapitypes "personalagent/runtime/internal/transport/openapitypes"
)

func toOpenAPIProviderSetRequest(request ProviderSetRequest) openapitypes.ProviderSetRequest {
	return openapitypes.ProviderSetRequest{
		WorkspaceId:      stringPointer(request.WorkspaceID),
		Provider:         stringPointer(request.Provider),
		Endpoint:         optionalStringPointer(request.Endpoint),
		ApiKeySecretName: optionalStringPointer(request.APIKeySecretName),
		ClearApiKey:      boolPointer(request.ClearAPIKey),
	}
}

func fromOpenAPIProviderSetRequest(request openapitypes.ProviderSetRequest) ProviderSetRequest {
	return ProviderSetRequest{
		WorkspaceID:      derefString(request.WorkspaceId),
		Provider:         derefString(request.Provider),
		Endpoint:         derefString(request.Endpoint),
		APIKeySecretName: derefString(request.ApiKeySecretName),
		ClearAPIKey:      derefBool(request.ClearApiKey),
	}
}

func toOpenAPIProviderConfigRecord(record ProviderConfigRecord) openapitypes.ProviderConfigRecord {
	return openapitypes.ProviderConfigRecord{
		WorkspaceId:      stringPointer(record.WorkspaceID),
		Provider:         stringPointer(record.Provider),
		Endpoint:         stringPointer(record.Endpoint),
		ApiKeySecretName: optionalStringPointer(record.APIKeySecretName),
		ApiKeyConfigured: boolPointer(record.APIKeyConfigured),
		UpdatedAt:        optionalTimePointer(record.UpdatedAt),
	}
}

func fromOpenAPIProviderConfigRecord(record openapitypes.ProviderConfigRecord) ProviderConfigRecord {
	return ProviderConfigRecord{
		WorkspaceID:      derefString(record.WorkspaceId),
		Provider:         derefString(record.Provider),
		Endpoint:         derefString(record.Endpoint),
		APIKeySecretName: derefString(record.ApiKeySecretName),
		APIKeyConfigured: derefBool(record.ApiKeyConfigured),
		UpdatedAt:        derefTime(record.UpdatedAt),
	}
}

func toOpenAPIProviderListRequest(request ProviderListRequest) openapitypes.ProviderListRequest {
	return openapitypes.ProviderListRequest{
		WorkspaceId: stringPointer(request.WorkspaceID),
	}
}

func fromOpenAPIProviderListRequest(request openapitypes.ProviderListRequest) ProviderListRequest {
	return ProviderListRequest{
		WorkspaceID: derefString(request.WorkspaceId),
	}
}

func toOpenAPIProviderListResponse(response ProviderListResponse) openapitypes.ProviderListResponse {
	providers := make([]openapitypes.ProviderConfigRecord, 0, len(response.Providers))
	for _, provider := range response.Providers {
		providers = append(providers, toOpenAPIProviderConfigRecord(provider))
	}
	return openapitypes.ProviderListResponse{
		WorkspaceId: stringPointer(response.WorkspaceID),
		Providers:   &providers,
	}
}

func fromOpenAPIProviderListResponse(response openapitypes.ProviderListResponse) ProviderListResponse {
	result := ProviderListResponse{
		WorkspaceID: derefString(response.WorkspaceId),
	}
	if response.Providers == nil {
		return result
	}
	result.Providers = make([]ProviderConfigRecord, 0, len(*response.Providers))
	for _, provider := range *response.Providers {
		result.Providers = append(result.Providers, fromOpenAPIProviderConfigRecord(provider))
	}
	return result
}

func toOpenAPIProviderCheckRequest(request ProviderCheckRequest) openapitypes.ProviderCheckRequest {
	return openapitypes.ProviderCheckRequest{
		WorkspaceId: stringPointer(request.WorkspaceID),
		Provider:    optionalStringPointer(request.Provider),
	}
}

func fromOpenAPIProviderCheckRequest(request openapitypes.ProviderCheckRequest) ProviderCheckRequest {
	return ProviderCheckRequest{
		WorkspaceID: derefString(request.WorkspaceId),
		Provider:    derefString(request.Provider),
	}
}

func toOpenAPIProviderCheckItem(item ProviderCheckItem) openapitypes.ProviderCheckItem {
	return openapitypes.ProviderCheckItem{
		Provider:   optionalStringPointer(item.Provider),
		Endpoint:   optionalStringPointer(item.Endpoint),
		Success:    boolPointer(item.Success),
		StatusCode: intPointer(item.StatusCode),
		LatencyMs:  int64Pointer(item.LatencyMS),
		Message:    optionalStringPointer(item.Message),
	}
}

func fromOpenAPIProviderCheckItem(item openapitypes.ProviderCheckItem) ProviderCheckItem {
	return ProviderCheckItem{
		Provider:   derefString(item.Provider),
		Endpoint:   derefString(item.Endpoint),
		Success:    derefBool(item.Success),
		StatusCode: derefInt(item.StatusCode),
		LatencyMS:  derefInt64(item.LatencyMs),
		Message:    derefString(item.Message),
	}
}

func toOpenAPIProviderCheckResponse(response ProviderCheckResponse) openapitypes.ProviderCheckResponse {
	results := make([]openapitypes.ProviderCheckItem, 0, len(response.Results))
	for _, item := range response.Results {
		results = append(results, toOpenAPIProviderCheckItem(item))
	}
	return openapitypes.ProviderCheckResponse{
		WorkspaceId: stringPointer(response.WorkspaceID),
		Success:     boolPointer(response.Success),
		Results:     &results,
	}
}

func fromOpenAPIProviderCheckResponse(response openapitypes.ProviderCheckResponse) ProviderCheckResponse {
	result := ProviderCheckResponse{
		WorkspaceID: derefString(response.WorkspaceId),
		Success:     derefBool(response.Success),
	}
	if response.Results == nil {
		return result
	}
	result.Results = make([]ProviderCheckItem, 0, len(*response.Results))
	for _, item := range *response.Results {
		result.Results = append(result.Results, fromOpenAPIProviderCheckItem(item))
	}
	return result
}

func toOpenAPIModelListRequest(request ModelListRequest) openapitypes.ModelListRequest {
	return openapitypes.ModelListRequest{
		WorkspaceId: stringPointer(request.WorkspaceID),
		Provider:    optionalStringPointer(request.Provider),
	}
}

func fromOpenAPIModelListRequest(request openapitypes.ModelListRequest) ModelListRequest {
	return ModelListRequest{
		WorkspaceID: derefString(request.WorkspaceId),
		Provider:    derefString(request.Provider),
	}
}

func toOpenAPIModelListItem(item ModelListItem) openapitypes.ModelListItem {
	return openapitypes.ModelListItem{
		WorkspaceId:      optionalStringPointer(item.WorkspaceID),
		Provider:         optionalStringPointer(item.Provider),
		ModelKey:         optionalStringPointer(item.ModelKey),
		Enabled:          boolPointer(item.Enabled),
		ProviderReady:    boolPointer(item.ProviderReady),
		ProviderEndpoint: optionalStringPointer(item.ProviderEndpoint),
	}
}

func fromOpenAPIModelListItem(item openapitypes.ModelListItem) ModelListItem {
	return ModelListItem{
		WorkspaceID:      derefString(item.WorkspaceId),
		Provider:         derefString(item.Provider),
		ModelKey:         derefString(item.ModelKey),
		Enabled:          derefBool(item.Enabled),
		ProviderReady:    derefBool(item.ProviderReady),
		ProviderEndpoint: derefString(item.ProviderEndpoint),
	}
}

func toOpenAPIModelListResponse(response ModelListResponse) openapitypes.ModelListResponse {
	models := make([]openapitypes.ModelListItem, 0, len(response.Models))
	for _, item := range response.Models {
		models = append(models, toOpenAPIModelListItem(item))
	}
	return openapitypes.ModelListResponse{
		WorkspaceId: stringPointer(response.WorkspaceID),
		Models:      &models,
	}
}

func fromOpenAPIModelListResponse(response openapitypes.ModelListResponse) ModelListResponse {
	result := ModelListResponse{
		WorkspaceID: derefString(response.WorkspaceId),
	}
	if response.Models == nil {
		return result
	}
	result.Models = make([]ModelListItem, 0, len(*response.Models))
	for _, item := range *response.Models {
		result.Models = append(result.Models, fromOpenAPIModelListItem(item))
	}
	return result
}

func toOpenAPIModelDiscoverRequest(request ModelDiscoverRequest) openapitypes.ModelDiscoverRequest {
	return openapitypes.ModelDiscoverRequest{
		WorkspaceId: stringPointer(request.WorkspaceID),
		Provider:    optionalStringPointer(request.Provider),
	}
}

func fromOpenAPIModelDiscoverRequest(request openapitypes.ModelDiscoverRequest) ModelDiscoverRequest {
	return ModelDiscoverRequest{
		WorkspaceID: derefString(request.WorkspaceId),
		Provider:    derefString(request.Provider),
	}
}

func toOpenAPIModelDiscoverItem(item ModelDiscoverItem) openapitypes.ModelDiscoverItem {
	return openapitypes.ModelDiscoverItem{
		Provider:    optionalStringPointer(item.Provider),
		ModelKey:    optionalStringPointer(item.ModelKey),
		DisplayName: optionalStringPointer(item.DisplayName),
		Source:      optionalStringPointer(item.Source),
		InCatalog:   boolPointer(item.InCatalog),
		Enabled:     boolPointer(item.Enabled),
	}
}

func fromOpenAPIModelDiscoverItem(item openapitypes.ModelDiscoverItem) ModelDiscoverItem {
	return ModelDiscoverItem{
		Provider:    derefString(item.Provider),
		ModelKey:    derefString(item.ModelKey),
		DisplayName: derefString(item.DisplayName),
		Source:      derefString(item.Source),
		InCatalog:   derefBool(item.InCatalog),
		Enabled:     derefBool(item.Enabled),
	}
}

func toOpenAPIModelDiscoverProviderResult(result ModelDiscoverProviderResult) openapitypes.ModelDiscoverProviderResult {
	models := make([]openapitypes.ModelDiscoverItem, 0, len(result.Models))
	for _, item := range result.Models {
		models = append(models, toOpenAPIModelDiscoverItem(item))
	}
	return openapitypes.ModelDiscoverProviderResult{
		Provider:         optionalStringPointer(result.Provider),
		ProviderReady:    boolPointer(result.ProviderReady),
		ProviderEndpoint: optionalStringPointer(result.ProviderEndpoint),
		Success:          boolPointer(result.Success),
		Message:          optionalStringPointer(result.Message),
		Models:           &models,
	}
}

func fromOpenAPIModelDiscoverProviderResult(result openapitypes.ModelDiscoverProviderResult) ModelDiscoverProviderResult {
	converted := ModelDiscoverProviderResult{
		Provider:         derefString(result.Provider),
		ProviderReady:    derefBool(result.ProviderReady),
		ProviderEndpoint: derefString(result.ProviderEndpoint),
		Success:          derefBool(result.Success),
		Message:          derefString(result.Message),
	}
	if result.Models == nil {
		return converted
	}
	converted.Models = make([]ModelDiscoverItem, 0, len(*result.Models))
	for _, item := range *result.Models {
		converted.Models = append(converted.Models, fromOpenAPIModelDiscoverItem(item))
	}
	return converted
}

func toOpenAPIModelDiscoverResponse(response ModelDiscoverResponse) openapitypes.ModelDiscoverResponse {
	results := make([]openapitypes.ModelDiscoverProviderResult, 0, len(response.Results))
	for _, item := range response.Results {
		results = append(results, toOpenAPIModelDiscoverProviderResult(item))
	}
	return openapitypes.ModelDiscoverResponse{
		WorkspaceId: stringPointer(response.WorkspaceID),
		Results:     &results,
	}
}

func fromOpenAPIModelDiscoverResponse(response openapitypes.ModelDiscoverResponse) ModelDiscoverResponse {
	result := ModelDiscoverResponse{
		WorkspaceID: derefString(response.WorkspaceId),
	}
	if response.Results == nil {
		return result
	}
	result.Results = make([]ModelDiscoverProviderResult, 0, len(*response.Results))
	for _, item := range *response.Results {
		result.Results = append(result.Results, fromOpenAPIModelDiscoverProviderResult(item))
	}
	return result
}
