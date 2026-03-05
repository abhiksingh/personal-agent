package transport

// DaemonCapabilityRouteGroup describes one supported daemon API route-group/prefix.
type DaemonCapabilityRouteGroup struct {
	ID          string `json:"id"`
	Prefix      string `json:"prefix"`
	Description string `json:"description,omitempty"`
}

// DaemonCapabilitiesResponse provides machine-readable daemon discovery metadata.
type DaemonCapabilitiesResponse struct {
	APIVersion             string                        `json:"api_version"`
	RouteGroups            []DaemonCapabilityRouteGroup  `json:"route_groups"`
	RealtimeEventTypes     []string                      `json:"realtime_event_types"`
	RealtimeBackpressure   EventBrokerBackpressurePolicy `json:"realtime_backpressure"`
	RealtimeDiagnostics    EventBrokerDiagnostics        `json:"realtime_diagnostics"`
	ClientSignalTypes      []string                      `json:"client_signal_types"`
	ProtocolModes          []string                      `json:"protocol_modes"`
	TransportListenerModes []string                      `json:"transport_listener_modes"`
	CorrelationID          string                        `json:"correlation_id"`
}
