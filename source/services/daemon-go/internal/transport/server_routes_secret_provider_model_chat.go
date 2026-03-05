package transport

import "net/http"

func (s *Server) registerSecretProviderModelChatRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v1/secrets/refs", s.handleSecretReferenceUpsert)
	mux.HandleFunc("/v1/secrets/refs/", s.handleSecretReferenceLookup)
	mux.HandleFunc("/v1/providers/set", s.handleProviderSet)
	mux.HandleFunc("/v1/providers/list", s.handleProviderList)
	mux.HandleFunc("/v1/providers/check", s.handleProviderCheck)
	mux.HandleFunc("/v1/models/list", s.handleModelList)
	mux.HandleFunc("/v1/models/discover", s.handleModelDiscover)
	mux.HandleFunc("/v1/models/add", s.handleModelAdd)
	mux.HandleFunc("/v1/models/remove", s.handleModelRemove)
	mux.HandleFunc("/v1/models/enable", s.handleModelEnable)
	mux.HandleFunc("/v1/models/disable", s.handleModelDisable)
	mux.HandleFunc("/v1/models/select", s.handleModelSelect)
	mux.HandleFunc("/v1/models/policy", s.handleModelPolicy)
	mux.HandleFunc("/v1/models/resolve", s.handleModelResolve)
	mux.HandleFunc("/v1/models/route/simulate", s.handleModelRouteSimulate)
	mux.HandleFunc("/v1/models/route/explain", s.handleModelRouteExplain)
	mux.HandleFunc("/v1/chat/turn/explain", s.handleChatTurnExplain)
	mux.HandleFunc("/v1/chat/turn", s.handleChatTurn)
	mux.HandleFunc("/v1/chat/history", s.handleChatTurnHistory)
	mux.HandleFunc("/v1/chat/persona/get", s.handleChatPersonaPolicyGet)
	mux.HandleFunc("/v1/chat/persona/set", s.handleChatPersonaPolicySet)
}
