package transport

import "context"

type UIStatusService interface {
	ListChannelConnectorMappings(ctx context.Context, request ChannelConnectorMappingListRequest) (ChannelConnectorMappingListResponse, error)
	UpsertChannelConnectorMapping(ctx context.Context, request ChannelConnectorMappingUpsertRequest) (ChannelConnectorMappingUpsertResponse, error)
	ListChannelStatus(ctx context.Context, request ChannelStatusRequest) (ChannelStatusResponse, error)
	ListConnectorStatus(ctx context.Context, request ConnectorStatusRequest) (ConnectorStatusResponse, error)
	ListChannelDiagnostics(ctx context.Context, request ChannelDiagnosticsRequest) (ChannelDiagnosticsResponse, error)
	ListConnectorDiagnostics(ctx context.Context, request ConnectorDiagnosticsRequest) (ConnectorDiagnosticsResponse, error)
	RequestConnectorPermission(ctx context.Context, request ConnectorPermissionRequest) (ConnectorPermissionResponse, error)
	UpsertChannelConfig(ctx context.Context, request ChannelConfigUpsertRequest) (ChannelConfigUpsertResponse, error)
	UpsertConnectorConfig(ctx context.Context, request ConnectorConfigUpsertRequest) (ConnectorConfigUpsertResponse, error)
	TestChannelOperation(ctx context.Context, request ChannelTestOperationRequest) (ChannelTestOperationResponse, error)
	TestConnectorOperation(ctx context.Context, request ConnectorTestOperationRequest) (ConnectorTestOperationResponse, error)
}
