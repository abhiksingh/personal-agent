package transport

import "net/http"

func (s *Server) registerRoutes(mux *http.ServeMux) {
	s.registerTaskApprovalWorkflowRoutes(mux)
	s.registerMetadataRoutes(mux)
	s.registerLifecycleRoutes(mux)
	s.registerSecretProviderModelChatRoutes(mux)
	s.registerAgentDelegationRoutes(mux)
	s.registerIdentityRoutes(mux)
	s.registerCommRoutes(mux)
	s.registerTwilioChannelRoutes(mux)
	s.registerUIStatusRoutes(mux)
	s.registerCloudflaredConnectorRoutes(mux)
	s.registerAutomationRoutes(mux)
	s.registerInspectRetentionContextRoutes(mux)
}
