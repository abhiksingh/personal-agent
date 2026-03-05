package daemonruntime

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"personalagent/runtime/internal/transport"
)

func (s *UIStatusService) ListChannelConnectorMappings(
	ctx context.Context,
	request transport.ChannelConnectorMappingListRequest,
) (transport.ChannelConnectorMappingListResponse, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	channelID, err := normalizeLogicalChannelID(request.ChannelID, true)
	if err != nil {
		return transport.ChannelConnectorMappingListResponse{}, err
	}

	tx, err := s.container.DB.BeginTx(ctx, nil)
	if err != nil {
		return transport.ChannelConnectorMappingListResponse{}, fmt.Errorf("begin channel connector mapping list transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	now := time.Now().UTC().Format(time.RFC3339Nano)
	if err := ensureWorkspace(ctx, tx, workspace, now); err != nil {
		return transport.ChannelConnectorMappingListResponse{}, err
	}

	rows, err := loadChannelConnectorBindingRows(ctx, tx, workspace, channelID)
	if err != nil {
		return transport.ChannelConnectorMappingListResponse{}, err
	}
	records := canonicalizeChannelConnectorBindings(rows)
	if len(records) > 0 {
		if err := replaceChannelConnectorBindingRows(ctx, tx, workspace, channelID, records, now); err != nil {
			return transport.ChannelConnectorMappingListResponse{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return transport.ChannelConnectorMappingListResponse{}, fmt.Errorf("commit channel connector mapping list transaction: %w", err)
	}

	capabilityIndex := s.channelConnectorMappingCapabilities()
	bindings := make([]transport.ChannelConnectorMappingRecord, 0, len(records))
	for _, row := range records {
		connectorID := normalizeChannelMappingConnectorID(row.ConnectorID)
		if connectorID == "" {
			continue
		}
		bindings = append(bindings, transport.ChannelConnectorMappingRecord{
			ChannelID:    row.ChannelID,
			ConnectorID:  connectorID,
			Enabled:      row.Enabled,
			Priority:     row.Priority,
			Capabilities: append([]string{}, capabilityIndex[connectorID]...),
			CreatedAt:    row.CreatedAt,
			UpdatedAt:    row.UpdatedAt,
		})
	}
	sort.Slice(bindings, func(i, j int) bool {
		if bindings[i].ChannelID == bindings[j].ChannelID {
			if bindings[i].Priority == bindings[j].Priority {
				return bindings[i].ConnectorID < bindings[j].ConnectorID
			}
			return bindings[i].Priority < bindings[j].Priority
		}
		return bindings[i].ChannelID < bindings[j].ChannelID
	})

	return transport.ChannelConnectorMappingListResponse{
		WorkspaceID:    workspace,
		ChannelID:      channelID,
		FallbackPolicy: channelConnectorFallbackPolicyPriorityOrder,
		Bindings:       bindings,
	}, nil
}

func (s *UIStatusService) UpsertChannelConnectorMapping(
	ctx context.Context,
	request transport.ChannelConnectorMappingUpsertRequest,
) (transport.ChannelConnectorMappingUpsertResponse, error) {
	workspace := normalizeWorkspaceID(request.WorkspaceID)
	channelID, err := normalizeLogicalChannelID(request.ChannelID, false)
	if err != nil {
		return transport.ChannelConnectorMappingUpsertResponse{}, err
	}
	connectorID := normalizeChannelMappingConnectorID(request.ConnectorID)
	if connectorID == "" {
		return transport.ChannelConnectorMappingUpsertResponse{}, fmt.Errorf("connector_id is required")
	}
	fallbackPolicy, err := normalizeChannelConnectorFallbackPolicy(request.FallbackPolicy)
	if err != nil {
		return transport.ChannelConnectorMappingUpsertResponse{}, err
	}

	capabilityIndex := s.channelConnectorMappingCapabilities()
	if err := validateChannelConnectorMappingCapabilities(channelID, connectorID, capabilityIndex); err != nil {
		return transport.ChannelConnectorMappingUpsertResponse{}, err
	}

	tx, err := s.container.DB.BeginTx(ctx, nil)
	if err != nil {
		return transport.ChannelConnectorMappingUpsertResponse{}, fmt.Errorf("begin channel connector mapping upsert transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	now := time.Now().UTC().Format(time.RFC3339Nano)
	if err := ensureWorkspace(ctx, tx, workspace, now); err != nil {
		return transport.ChannelConnectorMappingUpsertResponse{}, err
	}

	rows, err := loadChannelConnectorBindingRows(ctx, tx, workspace, channelID)
	if err != nil {
		return transport.ChannelConnectorMappingUpsertResponse{}, err
	}
	records := canonicalizeChannelConnectorBindings(rows)

	targetPriority := request.Priority
	targetIndex := -1
	for index := range records {
		if normalizeChannelMappingConnectorID(records[index].ConnectorID) == connectorID {
			targetIndex = index
			break
		}
	}
	if targetIndex >= 0 {
		records[targetIndex].Enabled = request.Enabled
		records[targetIndex].UpdatedAt = now
		if targetPriority <= 0 {
			targetPriority = records[targetIndex].Priority
		}
	} else {
		if targetPriority <= 0 {
			targetPriority = len(records) + 1
		}
		records = append(records, channelConnectorBindingRecord{
			ID:          channelConnectorBindingRecordID(workspace, channelID, connectorID),
			WorkspaceID: workspace,
			ChannelID:   channelID,
			ConnectorID: connectorID,
			Enabled:     request.Enabled,
			Priority:    targetPriority,
			CreatedAt:   now,
			UpdatedAt:   now,
		})
		targetIndex = len(records) - 1
	}

	reordered, resolvedPriority, reorderErr := reorderChannelConnectorBindings(records, targetIndex, targetPriority)
	if reorderErr != nil {
		return transport.ChannelConnectorMappingUpsertResponse{}, reorderErr
	}
	for index := range reordered {
		reordered[index].WorkspaceID = workspace
		reordered[index].ChannelID = channelID
		reordered[index].ConnectorID = normalizeChannelMappingConnectorID(reordered[index].ConnectorID)
		reordered[index].Priority = index + 1
		if strings.TrimSpace(reordered[index].ID) == "" {
			reordered[index].ID = channelConnectorBindingRecordID(workspace, channelID, reordered[index].ConnectorID)
		}
		if strings.TrimSpace(reordered[index].CreatedAt) == "" {
			reordered[index].CreatedAt = now
		}
		reordered[index].UpdatedAt = now
	}

	if err := replaceChannelConnectorBindingRows(ctx, tx, workspace, channelID, reordered, now); err != nil {
		return transport.ChannelConnectorMappingUpsertResponse{}, err
	}
	if err := tx.Commit(); err != nil {
		return transport.ChannelConnectorMappingUpsertResponse{}, fmt.Errorf("commit channel connector mapping upsert transaction: %w", err)
	}

	bindings := make([]transport.ChannelConnectorMappingRecord, 0, len(reordered))
	targetEnabled := request.Enabled
	for _, row := range reordered {
		normalizedConnectorID := normalizeChannelMappingConnectorID(row.ConnectorID)
		if normalizedConnectorID == "" {
			continue
		}
		if normalizedConnectorID == connectorID {
			targetEnabled = row.Enabled
		}
		bindings = append(bindings, transport.ChannelConnectorMappingRecord{
			ChannelID:    row.ChannelID,
			ConnectorID:  normalizedConnectorID,
			Enabled:      row.Enabled,
			Priority:     row.Priority,
			Capabilities: append([]string{}, capabilityIndex[normalizedConnectorID]...),
			CreatedAt:    row.CreatedAt,
			UpdatedAt:    row.UpdatedAt,
		})
	}

	return transport.ChannelConnectorMappingUpsertResponse{
		WorkspaceID:    workspace,
		ChannelID:      channelID,
		ConnectorID:    connectorID,
		Enabled:        targetEnabled,
		Priority:       resolvedPriority,
		FallbackPolicy: fallbackPolicy,
		UpdatedAt:      now,
		Bindings:       bindings,
	}, nil
}
