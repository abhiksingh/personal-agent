package daemonruntime

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	shared "personalagent/runtime/internal/shared/contracts"
	"personalagent/runtime/internal/transport"
)

func normalizeLogicalChannelID(raw string, allowEmpty bool) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" {
		if allowEmpty {
			return "", nil
		}
		return "", fmt.Errorf("channel_id is required")
	}
	switch normalized {
	case "app":
		return "app", nil
	case "message":
		return "message", nil
	case "voice":
		return "voice", nil
	default:
		return "", fmt.Errorf("unsupported logical channel %q (allowed: app|message|voice)", raw)
	}
}

func normalizeChannelMappingConnectorID(raw string) string {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	return normalized
}

func normalizeChannelConnectorFallbackPolicy(raw string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" {
		return channelConnectorFallbackPolicyPriorityOrder, nil
	}
	if normalized != channelConnectorFallbackPolicyPriorityOrder {
		return "", fmt.Errorf("unsupported fallback_policy %q (allowed: %s)", raw, channelConnectorFallbackPolicyPriorityOrder)
	}
	return normalized, nil
}

func channelConnectorBindingRecordID(workspaceID string, channelID string, connectorID string) string {
	connectorToken := strings.NewReplacer(".", "_", "-", "_").Replace(strings.ToLower(strings.TrimSpace(connectorID)))
	if connectorToken == "" {
		connectorToken = "connector"
	}
	return "ccb." + workspaceID + "." + channelID + "." + connectorToken
}

func loadChannelConnectorBindingRows(
	ctx context.Context,
	tx *sql.Tx,
	workspaceID string,
	channelID string,
) ([]channelConnectorBindingRecord, error) {
	workspace := normalizeWorkspaceID(workspaceID)
	query := `
		SELECT
			id,
			workspace_id,
			channel_id,
			connector_id,
			enabled,
			priority,
			created_at,
			updated_at
		FROM channel_connector_bindings
		WHERE workspace_id = ?
	`
	args := []any{workspace}
	if strings.TrimSpace(channelID) != "" {
		query += " AND channel_id = ?"
		args = append(args, channelID)
	}
	query += " ORDER BY channel_id ASC, priority ASC, connector_id ASC"

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		if strings.Contains(strings.ToLower(strings.TrimSpace(err.Error())), "no such table: channel_connector_bindings") {
			return []channelConnectorBindingRecord{}, nil
		}
		return nil, fmt.Errorf("list channel connector bindings: %w", err)
	}

	result := []channelConnectorBindingRecord{}
	for rows.Next() {
		var (
			record    channelConnectorBindingRecord
			enabledDB int
		)
		if err := rows.Scan(
			&record.ID,
			&record.WorkspaceID,
			&record.ChannelID,
			&record.ConnectorID,
			&enabledDB,
			&record.Priority,
			&record.CreatedAt,
			&record.UpdatedAt,
		); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan channel connector binding: %w", err)
		}
		record.ChannelID = strings.ToLower(strings.TrimSpace(record.ChannelID))
		record.ConnectorID = normalizeChannelMappingConnectorID(record.ConnectorID)
		record.Enabled = enabledDB == 1
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate channel connector bindings: %w", err)
	}
	if closeErr := rows.Close(); closeErr != nil {
		return nil, fmt.Errorf("close channel connector binding rows: %w", closeErr)
	}
	return result, nil
}

func canonicalizeChannelConnectorBindings(rows []channelConnectorBindingRecord) []channelConnectorBindingRecord {
	if len(rows) == 0 {
		return []channelConnectorBindingRecord{}
	}

	normalized := make([]channelConnectorBindingRecord, 0, len(rows))
	seenByChannel := map[string]map[string]struct{}{}
	for _, row := range rows {
		channelID := strings.ToLower(strings.TrimSpace(row.ChannelID))
		connectorID := normalizeChannelMappingConnectorID(row.ConnectorID)
		if channelID == "" || connectorID == "" {
			continue
		}
		channelSeen, exists := seenByChannel[channelID]
		if !exists {
			channelSeen = map[string]struct{}{}
			seenByChannel[channelID] = channelSeen
		}
		if _, exists := channelSeen[connectorID]; exists {
			continue
		}
		channelSeen[connectorID] = struct{}{}

		row.ChannelID = channelID
		row.ConnectorID = connectorID
		if row.Priority <= 0 {
			row.Priority = len(channelSeen)
		}
		normalized = append(normalized, row)
	}

	sort.Slice(normalized, func(i, j int) bool {
		if normalized[i].ChannelID == normalized[j].ChannelID {
			if normalized[i].Priority == normalized[j].Priority {
				return normalized[i].ConnectorID < normalized[j].ConnectorID
			}
			return normalized[i].Priority < normalized[j].Priority
		}
		return normalized[i].ChannelID < normalized[j].ChannelID
	})

	channelPriority := map[string]int{}
	for index := range normalized {
		channel := normalized[index].ChannelID
		channelPriority[channel]++
		normalized[index].Priority = channelPriority[channel]
	}
	return normalized
}

func reorderChannelConnectorBindings(
	rows []channelConnectorBindingRecord,
	targetIndex int,
	targetPriority int,
) ([]channelConnectorBindingRecord, int, error) {
	if len(rows) == 0 {
		return nil, 0, fmt.Errorf("channel connector binding list is empty")
	}
	if targetIndex < 0 || targetIndex >= len(rows) {
		return nil, 0, fmt.Errorf("target connector binding is missing")
	}

	reordered := make([]channelConnectorBindingRecord, 0, len(rows))
	reordered = append(reordered, rows...)

	target := reordered[targetIndex]
	reordered = append(reordered[:targetIndex], reordered[targetIndex+1:]...)

	if targetPriority <= 0 {
		targetPriority = 1
	}
	if targetPriority > len(reordered)+1 {
		targetPriority = len(reordered) + 1
	}
	insertAt := targetPriority - 1
	reordered = append(reordered, channelConnectorBindingRecord{})
	copy(reordered[insertAt+1:], reordered[insertAt:])
	reordered[insertAt] = target
	return reordered, targetPriority, nil
}

func replaceChannelConnectorBindingRows(
	ctx context.Context,
	tx *sql.Tx,
	workspaceID string,
	channelFilter string,
	rows []channelConnectorBindingRecord,
	now string,
) error {
	query := `DELETE FROM channel_connector_bindings WHERE workspace_id = ?`
	args := []any{workspaceID}
	if strings.TrimSpace(channelFilter) != "" {
		query += " AND channel_id = ?"
		args = append(args, channelFilter)
	}
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("delete existing channel connector bindings: %w", err)
	}

	for index := range rows {
		record := rows[index]
		if strings.TrimSpace(record.ID) == "" {
			record.ID = channelConnectorBindingRecordID(workspaceID, record.ChannelID, record.ConnectorID)
		}
		if strings.TrimSpace(record.CreatedAt) == "" {
			record.CreatedAt = now
		}
		if strings.TrimSpace(record.UpdatedAt) == "" {
			record.UpdatedAt = now
		}
		enabledDB := 0
		if record.Enabled {
			enabledDB = 1
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO channel_connector_bindings(
				id,
				workspace_id,
				channel_id,
				connector_id,
				enabled,
				priority,
				created_at,
				updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`,
			record.ID,
			workspaceID,
			record.ChannelID,
			record.ConnectorID,
			enabledDB,
			record.Priority,
			record.CreatedAt,
			record.UpdatedAt,
		); err != nil {
			return fmt.Errorf("insert channel connector binding %s/%s: %w", record.ChannelID, record.ConnectorID, err)
		}
	}
	return nil
}

func validateChannelConnectorMappingCapabilities(
	channelID string,
	connectorID string,
	capabilityIndex map[string][]string,
) error {
	normalizedChannel := strings.ToLower(strings.TrimSpace(channelID))
	normalizedConnector := normalizeChannelMappingConnectorID(connectorID)
	if normalizedChannel == "" || normalizedConnector == "" {
		return fmt.Errorf("channel_id and connector_id are required")
	}

	capabilities := append([]string{}, capabilityIndex[normalizedConnector]...)
	if len(capabilities) == 0 {
		capabilities = append(capabilities, channelConnectorMappingDefaultCapabilities[normalizedConnector]...)
	}

	requiredByConnector := channelConnectorMappingCapabilityRequirements[normalizedChannel]
	if len(requiredByConnector) > 0 {
		requiredCapabilities, explicit := requiredByConnector[normalizedConnector]
		if explicit {
			if !capabilitiesContainAny(capabilities, requiredCapabilities) {
				return fmt.Errorf(
					"connector %q does not advertise required capabilities for channel %q (need one of: %s)",
					normalizedConnector,
					normalizedChannel,
					strings.Join(requiredCapabilities, ", "),
				)
			}
			return nil
		}
	}

	if len(capabilities) == 0 {
		return fmt.Errorf(
			"connector %q is not mapped to logical channel %q and has no advertised capabilities for compatibility validation",
			normalizedConnector,
			normalizedChannel,
		)
	}
	for _, capability := range capabilities {
		if capabilitySupportsLogicalChannel(normalizedChannel, capability) {
			return nil
		}
	}
	return fmt.Errorf("connector %q does not support logical channel %q", normalizedConnector, normalizedChannel)
}

func capabilitiesContainAny(capabilities []string, required []string) bool {
	if len(capabilities) == 0 || len(required) == 0 {
		return false
	}
	set := map[string]struct{}{}
	for _, capability := range capabilities {
		key := strings.ToLower(strings.TrimSpace(capability))
		if key == "" {
			continue
		}
		set[key] = struct{}{}
	}
	for _, capability := range required {
		key := strings.ToLower(strings.TrimSpace(capability))
		if key == "" {
			continue
		}
		if _, exists := set[key]; exists {
			return true
		}
	}
	return false
}

func capabilitySupportsLogicalChannel(channelID string, capability string) bool {
	normalizedChannel := strings.ToLower(strings.TrimSpace(channelID))
	normalizedCapability := strings.ToLower(strings.TrimSpace(capability))
	if normalizedCapability == "" {
		return false
	}
	switch normalizedChannel {
	case "app":
		return strings.Contains(normalizedCapability, "channel.app_chat.") || strings.Contains(normalizedCapability, "channel.app.")
	case "message":
		return strings.Contains(normalizedCapability, "channel.messages.") ||
			strings.Contains(normalizedCapability, ".sms.") ||
			strings.HasPrefix(normalizedCapability, "channel.sms.")
	case "voice":
		return strings.Contains(normalizedCapability, ".voice.") ||
			strings.HasPrefix(normalizedCapability, "channel.voice.")
	default:
		return false
	}
}

func (s *UIStatusService) channelConnectorMappingCapabilities() map[string][]string {
	merged := map[string]map[string]struct{}{}
	for connectorID, capabilities := range channelConnectorMappingDefaultCapabilities {
		normalizedConnectorID := normalizeChannelMappingConnectorID(connectorID)
		if normalizedConnectorID == "" {
			continue
		}
		set := merged[normalizedConnectorID]
		if set == nil {
			set = map[string]struct{}{}
			merged[normalizedConnectorID] = set
		}
		for _, capability := range capabilities {
			key := strings.TrimSpace(capability)
			if key == "" {
				continue
			}
			set[key] = struct{}{}
		}
	}

	if s != nil && s.container != nil && s.container.ConnectorRegistry != nil {
		for _, metadata := range s.container.ConnectorRegistry.ListMetadata() {
			connectorID := normalizeChannelMappingConnectorID(connectorIDFromPluginID(metadata.ID))
			if connectorID == "" {
				continue
			}
			set := merged[connectorID]
			if set == nil {
				set = map[string]struct{}{}
				merged[connectorID] = set
			}
			for _, capability := range metadata.Capabilities {
				key := strings.TrimSpace(capability.Key)
				if key == "" {
					continue
				}
				set[key] = struct{}{}
			}
		}
	}

	for pluginID, worker := range listWorkerStatusByPluginID(s.container) {
		connectorID := connectorIDFromMappingPlugin(pluginID, worker)
		if connectorID == "" {
			continue
		}
		set := merged[connectorID]
		if set == nil {
			set = map[string]struct{}{}
			merged[connectorID] = set
		}
		for _, capability := range capabilityKeys(worker.Metadata.Capabilities) {
			if strings.TrimSpace(capability) == "" {
				continue
			}
			set[capability] = struct{}{}
		}
	}

	result := map[string][]string{}
	for connectorID, set := range merged {
		keys := make([]string, 0, len(set))
		for capability := range set {
			keys = append(keys, capability)
		}
		sort.Strings(keys)
		result[connectorID] = keys
	}
	return result
}

func connectorIDFromMappingPlugin(pluginID string, worker PluginWorkerStatus) string {
	switch strings.TrimSpace(pluginID) {
	case appChatWorkerPluginID:
		return "builtin.app"
	case messagesWorkerPluginID:
		return "imessage"
	case twilioWorkerPluginID:
		return "twilio"
	}
	if worker.Kind != shared.AdapterKindConnector {
		return ""
	}
	return normalizeChannelMappingConnectorID(connectorIDFromPluginID(pluginID))
}

func workerStateNeedsRepair(worker *transport.PluginWorkerStatusCard) bool {
	if worker == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(worker.State)) {
	case "failed", "stopped":
		return true
	default:
		return false
	}
}

func workerStateStarting(worker *transport.PluginWorkerStatusCard) bool {
	if worker == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(worker.State)) {
	case "registered", "starting", "restarting":
		return true
	default:
		return false
	}
}

func connectorSupportsPermissionPrompt(connectorID string) bool {
	switch strings.ToLower(strings.TrimSpace(connectorID)) {
	case "imessage", "mail", "calendar", "browser", "finder":
		return true
	default:
		return false
	}
}

func channelStatusReason(card transport.ChannelStatusCard) string {
	if len(card.Configuration) == 0 {
		return ""
	}
	raw, ok := card.Configuration["status_reason"]
	if !ok {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(value))
}

func connectorStatusReason(card transport.ConnectorStatusCard) string {
	if len(card.Configuration) == 0 {
		return ""
	}
	raw, ok := card.Configuration["status_reason"]
	if !ok {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(value))
}

func connectorPermissionReason(connectorID string, enabled bool) string {
	if enabled {
		return ""
	}
	return "Permission request flow is not implemented for connector " + strings.TrimSpace(connectorID) + "."
}

func connectorSystemSettingsTarget(connectorID string) string {
	switch strings.ToLower(strings.TrimSpace(connectorID)) {
	case "imessage":
		return "ui://system-settings/privacy/full-disk-access"
	case "calendar", "browser", "mail", "finder":
		return "ui://system-settings/privacy/automation"
	default:
		return "ui://system-settings/privacy"
	}
}

func connectorPermissionSystemSettingsTarget(connectorID string) string {
	switch strings.ToLower(strings.TrimSpace(connectorID)) {
	case "imessage":
		return "ui://system-settings/privacy/automation"
	default:
		return connectorSystemSettingsTarget(connectorID)
	}
}

func capabilityKeys(capabilities []shared.CapabilityDescriptor) []string {
	keys := make([]string, 0, len(capabilities))
	for _, capability := range capabilities {
		key := strings.TrimSpace(capability.Key)
		if key == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func connectorIDFromPluginID(pluginID string) string {
	trimmed := strings.TrimSpace(pluginID)
	if trimmed == "" {
		return "connector"
	}
	trimmed = strings.TrimSuffix(trimmed, ".daemon")
	if strings.Contains(trimmed, ".") {
		parts := strings.Split(trimmed, ".")
		trimmed = parts[len(parts)-1]
	}
	trimmed = strings.TrimSpace(trimmed)
	if trimmed == "" {
		return "connector"
	}
	return trimmed
}

func humanizeConnectorID(connectorID string) string {
	trimmed := strings.TrimSpace(connectorID)
	if trimmed == "" {
		return "Connector"
	}
	return strings.ToUpper(trimmed[:1]) + trimmed[1:] + " Connector"
}

func formatTimeCard(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func truncateDiagnosticsText(value string, max int) string {
	trimmed := strings.TrimSpace(value)
	if max <= 0 || len(trimmed) <= max {
		return trimmed
	}
	return strings.TrimSpace(trimmed[:max]) + "..."
}

func uiFirstNonEmpty(primary string, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return primary
	}
	return fallback
}
