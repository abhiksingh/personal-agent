package transport

import "net/http"

func (s *Server) registerUIStatusRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v1/channels/status", s.handleChannelStatus)
	mux.HandleFunc("/v1/connectors/status", s.handleConnectorStatus)
	mux.HandleFunc("/v1/channels/diagnostics", s.handleChannelDiagnostics)
	mux.HandleFunc("/v1/connectors/diagnostics", s.handleConnectorDiagnostics)
	mux.HandleFunc("/v1/channels/config/upsert", s.handleChannelConfigUpsert)
	mux.HandleFunc("/v1/connectors/config/upsert", s.handleConnectorConfigUpsert)
	mux.HandleFunc("/v1/channels/test", s.handleChannelTestOperation)
	mux.HandleFunc("/v1/connectors/test", s.handleConnectorTestOperation)
	mux.HandleFunc("/v1/connectors/permission/request", s.handleConnectorPermissionRequest)
}
