package daemonruntime

import (
	"context"
	"fmt"
	"strings"

	"personalagent/runtime/internal/transport"
)

func (s *UIStatusService) UpsertChannelConfig(ctx context.Context, request transport.ChannelConfigUpsertRequest) (transport.ChannelConfigUpsertResponse, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	channelID, err := normalizeLogicalChannelID(request.ChannelID, false)
	if err != nil {
		return transport.ChannelConfigUpsertResponse{}, err
	}

	config, updatedAt, err := s.upsertUIConfig(ctx, workspace, uiChannelConfigPrefix+channelID, request.Configuration.AsMap(), request.Merge)
	if err != nil {
		return transport.ChannelConfigUpsertResponse{}, err
	}
	normalized, twilioUpdatedAt, canonicalErr := s.upsertCanonicalTwilioChannelConfig(ctx, workspace, channelID, config)
	if canonicalErr != nil {
		return transport.ChannelConfigUpsertResponse{}, canonicalErr
	}
	if len(normalized) > 0 {
		config = normalized
	}
	if strings.TrimSpace(twilioUpdatedAt) != "" {
		updatedAt = twilioUpdatedAt
	}
	return transport.ChannelConfigUpsertResponse{
		WorkspaceID:   workspace,
		ChannelID:     channelID,
		Configuration: transport.UIStatusConfigurationFromMap(config),
		UpdatedAt:     updatedAt,
	}, nil
}

func (s *UIStatusService) UpsertConnectorConfig(ctx context.Context, request transport.ConnectorConfigUpsertRequest) (transport.ConnectorConfigUpsertResponse, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	connectorID := strings.ToLower(strings.TrimSpace(request.ConnectorID))
	if connectorID == "" {
		return transport.ConnectorConfigUpsertResponse{}, fmt.Errorf("connector_id is required")
	}

	config, updatedAt, err := s.upsertUIConfig(ctx, workspace, uiConnectorConfigPrefix+connectorID, request.Configuration.AsMap(), request.Merge)
	if err != nil {
		return transport.ConnectorConfigUpsertResponse{}, err
	}
	if connectorID == "twilio" {
		normalized, twilioUpdatedAt, canonicalErr := s.upsertCanonicalTwilioConnectorConfig(ctx, workspace, config)
		if canonicalErr != nil {
			return transport.ConnectorConfigUpsertResponse{}, canonicalErr
		}
		if len(normalized) > 0 {
			config = normalized
		}
		if strings.TrimSpace(twilioUpdatedAt) != "" {
			updatedAt = twilioUpdatedAt
		}
	}
	return transport.ConnectorConfigUpsertResponse{
		WorkspaceID:   workspace,
		ConnectorID:   connectorID,
		Configuration: transport.UIStatusConfigurationFromMap(config),
		UpdatedAt:     updatedAt,
	}, nil
}
