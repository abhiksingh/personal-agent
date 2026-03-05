package transport

import (
	"encoding/json"
	"fmt"
	"strings"
)

// UIStatusMappedConnector captures typed mapped-connector details for UI-status payloads.
type UIStatusMappedConnector struct {
	ConnectorID string         `json:"connector_id,omitempty"`
	Enabled     *bool          `json:"enabled,omitempty"`
	Priority    *int           `json:"priority,omitempty"`
	Configured  *bool          `json:"configured,omitempty"`
	Status      string         `json:"status,omitempty"`
	Summary     string         `json:"summary,omitempty"`
	Additional  map[string]any `json:"-"`
}

func (m UIStatusMappedConnector) IsZero() bool {
	return m.ConnectorID == "" && m.Enabled == nil && m.Priority == nil && m.Configured == nil && m.Status == "" && m.Summary == "" && len(m.Additional) == 0
}

func (m UIStatusMappedConnector) AsMap() map[string]any {
	result := cloneAnyMapShallow(m.Additional)
	setStringField(result, "connector_id", m.ConnectorID)
	setBoolPointerField(result, "enabled", m.Enabled)
	setIntPointerField(result, "priority", m.Priority)
	setBoolPointerField(result, "configured", m.Configured)
	setStringField(result, "status", m.Status)
	setStringField(result, "summary", m.Summary)
	return result
}

func uiStatusMappedConnectorFromMap(value map[string]any) UIStatusMappedConnector {
	if len(value) == 0 {
		return UIStatusMappedConnector{}
	}
	result := UIStatusMappedConnector{
		ConnectorID: readAnyString(value["connector_id"]),
		Enabled:     readAnyBoolPointer(value["enabled"]),
		Priority:    readAnyIntPointer(value["priority"]),
		Configured:  readAnyBoolPointer(value["configured"]),
		Status:      readAnyString(value["status"]),
		Summary:     readAnyString(value["summary"]),
	}
	result.Additional = removeKnownKeys(value,
		"connector_id",
		"enabled",
		"priority",
		"configured",
		"status",
		"summary",
	)
	return result
}

// UIStatusConfiguration types commonly consumed UI-status configuration fields and
// preserves connector/channel-specific extensibility keys.
type UIStatusConfiguration struct {
	Enabled                *bool                     `json:"enabled,omitempty"`
	Transport              string                    `json:"transport,omitempty"`
	Mode                   string                    `json:"mode,omitempty"`
	Number                 string                    `json:"number,omitempty"`
	Scope                  string                    `json:"scope,omitempty"`
	StatusReason           string                    `json:"status_reason,omitempty"`
	FallbackPolicy         string                    `json:"fallback_policy,omitempty"`
	PrimaryConnectorID     string                    `json:"primary_connector_id,omitempty"`
	MappedConnectorIDs     []string                  `json:"mapped_connector_ids,omitempty"`
	EnabledConnectorIDs    []string                  `json:"enabled_connector_ids,omitempty"`
	MappedConnectors       []UIStatusMappedConnector `json:"mapped_connectors,omitempty"`
	BoundConnector         string                    `json:"bound_connector,omitempty"`
	BoundToChannel         *bool                     `json:"bound_to_channel,omitempty"`
	IngestSourceScope      string                    `json:"ingest_source_scope,omitempty"`
	IngestUpdatedAt        string                    `json:"ingest_updated_at,omitempty"`
	IngestLastError        string                    `json:"ingest_last_error,omitempty"`
	CredentialsConfigured  *bool                     `json:"credentials_configured,omitempty"`
	PermissionState        string                    `json:"permission_state,omitempty"`
	ExecutePathProbeReady  *bool                     `json:"execute_path_probe_ready,omitempty"`
	ExecutePathProbeStatus *int                      `json:"execute_path_probe_status_code,omitempty"`
	ExecutePathProbeError  string                    `json:"execute_path_probe_error,omitempty"`
	CloudflaredAvailable   *bool                     `json:"cloudflared_available,omitempty"`
	CloudflaredBinaryPath  string                    `json:"cloudflared_binary_path,omitempty"`
	CloudflaredDryRun      *bool                     `json:"cloudflared_dry_run,omitempty"`
	CloudflaredExitCode    *int                      `json:"cloudflared_exit_code,omitempty"`
	CloudflaredError       string                    `json:"cloudflared_error,omitempty"`
	LocalIngestBridgeReady *bool                     `json:"local_ingest_bridge_ready,omitempty"`
	Additional             map[string]any            `json:"-"`
}

func (c UIStatusConfiguration) IsZero() bool {
	return c.Enabled == nil &&
		c.Transport == "" &&
		c.Mode == "" &&
		c.Number == "" &&
		c.Scope == "" &&
		c.StatusReason == "" &&
		c.FallbackPolicy == "" &&
		c.PrimaryConnectorID == "" &&
		len(c.MappedConnectorIDs) == 0 &&
		len(c.EnabledConnectorIDs) == 0 &&
		len(c.MappedConnectors) == 0 &&
		c.BoundConnector == "" &&
		c.BoundToChannel == nil &&
		c.IngestSourceScope == "" &&
		c.IngestUpdatedAt == "" &&
		c.IngestLastError == "" &&
		c.CredentialsConfigured == nil &&
		c.PermissionState == "" &&
		c.ExecutePathProbeReady == nil &&
		c.ExecutePathProbeStatus == nil &&
		c.ExecutePathProbeError == "" &&
		c.CloudflaredAvailable == nil &&
		c.CloudflaredBinaryPath == "" &&
		c.CloudflaredDryRun == nil &&
		c.CloudflaredExitCode == nil &&
		c.CloudflaredError == "" &&
		c.LocalIngestBridgeReady == nil &&
		len(c.Additional) == 0
}

func (c UIStatusConfiguration) AsMap() map[string]any {
	result := cloneAnyMapShallow(c.Additional)
	setBoolPointerField(result, "enabled", c.Enabled)
	setStringField(result, "transport", c.Transport)
	setStringField(result, "mode", c.Mode)
	setStringField(result, "number", c.Number)
	setStringField(result, "scope", c.Scope)
	setStringField(result, "status_reason", c.StatusReason)
	setStringField(result, "fallback_policy", c.FallbackPolicy)
	setStringField(result, "primary_connector_id", c.PrimaryConnectorID)
	setStringSliceField(result, "mapped_connector_ids", c.MappedConnectorIDs)
	setStringSliceField(result, "enabled_connector_ids", c.EnabledConnectorIDs)
	if len(c.MappedConnectors) > 0 {
		mapped := make([]map[string]any, 0, len(c.MappedConnectors))
		for _, item := range c.MappedConnectors {
			mapped = append(mapped, item.AsMap())
		}
		result["mapped_connectors"] = mapped
	}
	setStringField(result, "bound_connector", c.BoundConnector)
	setBoolPointerField(result, "bound_to_channel", c.BoundToChannel)
	setStringField(result, "ingest_source_scope", c.IngestSourceScope)
	setStringField(result, "ingest_updated_at", c.IngestUpdatedAt)
	setStringField(result, "ingest_last_error", c.IngestLastError)
	setBoolPointerField(result, "credentials_configured", c.CredentialsConfigured)
	setStringField(result, "permission_state", c.PermissionState)
	setBoolPointerField(result, "execute_path_probe_ready", c.ExecutePathProbeReady)
	setIntPointerField(result, "execute_path_probe_status_code", c.ExecutePathProbeStatus)
	setStringField(result, "execute_path_probe_error", c.ExecutePathProbeError)
	setBoolPointerField(result, "cloudflared_available", c.CloudflaredAvailable)
	setStringField(result, "cloudflared_binary_path", c.CloudflaredBinaryPath)
	setBoolPointerField(result, "cloudflared_dry_run", c.CloudflaredDryRun)
	setIntPointerField(result, "cloudflared_exit_code", c.CloudflaredExitCode)
	setStringField(result, "cloudflared_error", c.CloudflaredError)
	setBoolPointerField(result, "local_ingest_bridge_ready", c.LocalIngestBridgeReady)
	return result
}

func UIStatusConfigurationFromMap(value map[string]any) UIStatusConfiguration {
	if len(value) == 0 {
		return UIStatusConfiguration{}
	}
	result := UIStatusConfiguration{
		Enabled:                readAnyBoolPointer(value["enabled"]),
		Transport:              readAnyString(value["transport"]),
		Mode:                   readAnyString(value["mode"]),
		Number:                 readAnyString(value["number"]),
		Scope:                  readAnyString(value["scope"]),
		StatusReason:           readAnyString(value["status_reason"]),
		FallbackPolicy:         readAnyString(value["fallback_policy"]),
		PrimaryConnectorID:     readAnyString(value["primary_connector_id"]),
		MappedConnectorIDs:     readAnyStringSlice(value["mapped_connector_ids"]),
		EnabledConnectorIDs:    readAnyStringSlice(value["enabled_connector_ids"]),
		BoundConnector:         readAnyString(value["bound_connector"]),
		BoundToChannel:         readAnyBoolPointer(value["bound_to_channel"]),
		IngestSourceScope:      readAnyString(value["ingest_source_scope"]),
		IngestUpdatedAt:        readAnyString(value["ingest_updated_at"]),
		IngestLastError:        readAnyString(value["ingest_last_error"]),
		CredentialsConfigured:  readAnyBoolPointer(value["credentials_configured"]),
		PermissionState:        readAnyString(value["permission_state"]),
		ExecutePathProbeReady:  readAnyBoolPointer(value["execute_path_probe_ready"]),
		ExecutePathProbeStatus: readAnyIntPointer(value["execute_path_probe_status_code"]),
		ExecutePathProbeError:  readAnyString(value["execute_path_probe_error"]),
		CloudflaredAvailable:   readAnyBoolPointer(value["cloudflared_available"]),
		CloudflaredBinaryPath:  readAnyString(value["cloudflared_binary_path"]),
		CloudflaredDryRun:      readAnyBoolPointer(value["cloudflared_dry_run"]),
		CloudflaredExitCode:    readAnyIntPointer(value["cloudflared_exit_code"]),
		CloudflaredError:       readAnyString(value["cloudflared_error"]),
		LocalIngestBridgeReady: readAnyBoolPointer(value["local_ingest_bridge_ready"]),
	}
	if rawMapped, ok := value["mapped_connectors"].([]any); ok {
		result.MappedConnectors = make([]UIStatusMappedConnector, 0, len(rawMapped))
		for _, candidate := range rawMapped {
			decoded := readAnyMap(candidate)
			if len(decoded) == 0 {
				continue
			}
			result.MappedConnectors = append(result.MappedConnectors, uiStatusMappedConnectorFromMap(decoded))
		}
	}
	result.Additional = removeKnownKeys(value,
		"enabled",
		"transport",
		"mode",
		"number",
		"scope",
		"status_reason",
		"fallback_policy",
		"primary_connector_id",
		"mapped_connector_ids",
		"enabled_connector_ids",
		"mapped_connectors",
		"bound_connector",
		"bound_to_channel",
		"ingest_source_scope",
		"ingest_updated_at",
		"ingest_last_error",
		"credentials_configured",
		"permission_state",
		"execute_path_probe_ready",
		"execute_path_probe_status_code",
		"execute_path_probe_error",
		"cloudflared_available",
		"cloudflared_binary_path",
		"cloudflared_dry_run",
		"cloudflared_exit_code",
		"cloudflared_error",
		"local_ingest_bridge_ready",
	)
	return result
}

func (c UIStatusConfiguration) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.AsMap())
}

func (c *UIStatusConfiguration) UnmarshalJSON(data []byte) error {
	if c == nil {
		return fmt.Errorf("nil UIStatusConfiguration")
	}
	if len(strings.TrimSpace(string(data))) == 0 || string(data) == "null" {
		*c = UIStatusConfiguration{}
		return nil
	}
	decoded := map[string]any{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*c = UIStatusConfigurationFromMap(decoded)
	return nil
}
