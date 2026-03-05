package transport

import "net/http"

func (s *Server) registerAgentDelegationRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v1/agent/run", s.handleAgentRun)
	mux.HandleFunc("/v1/agent/approve", s.handleAgentApprove)
	mux.HandleFunc("/v1/delegation/grant", s.handleDelegationGrant)
	mux.HandleFunc("/v1/delegation/list", s.handleDelegationList)
	mux.HandleFunc("/v1/delegation/revoke", s.handleDelegationRevoke)
	mux.HandleFunc("/v1/delegation/check", s.handleDelegationCheck)
	mux.HandleFunc("/v1/delegation/capability-grants/upsert", s.handleCapabilityGrantUpsert)
	mux.HandleFunc("/v1/delegation/capability-grants/list", s.handleCapabilityGrantList)
}

func (s *Server) registerIdentityRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v1/identity/workspaces", s.handleIdentityWorkspaces)
	mux.HandleFunc("/v1/identity/principals", s.handleIdentityPrincipals)
	mux.HandleFunc("/v1/identity/context", s.handleIdentityContext)
	mux.HandleFunc("/v1/identity/context/select-workspace", s.handleIdentitySelectWorkspace)
	mux.HandleFunc("/v1/identity/bootstrap", s.handleIdentityBootstrap)
	mux.HandleFunc("/v1/identity/devices/list", s.handleIdentityDevices)
	mux.HandleFunc("/v1/identity/sessions/list", s.handleIdentitySessions)
	mux.HandleFunc("/v1/identity/sessions/revoke", s.handleIdentitySessionRevoke)
}
