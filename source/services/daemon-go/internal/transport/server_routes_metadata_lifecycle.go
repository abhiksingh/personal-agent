package transport

import "net/http"

func (s *Server) registerMetadataRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v1/meta/capabilities", s.handleMetaCapabilities)
}

func (s *Server) registerLifecycleRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v1/daemon/lifecycle/status", s.handleDaemonLifecycleStatus)
	mux.HandleFunc("/v1/daemon/lifecycle/control", s.handleDaemonLifecycleControl)
	mux.HandleFunc("/v1/daemon/lifecycle/plugins/history", s.handleDaemonPluginLifecycleHistory)
}
