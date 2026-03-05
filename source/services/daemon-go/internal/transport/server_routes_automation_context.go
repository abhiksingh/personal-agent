package transport

import "net/http"

func (s *Server) registerAutomationRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v1/automation/create", s.handleAutomationCreate)
	mux.HandleFunc("/v1/automation/list", s.handleAutomationList)
	mux.HandleFunc("/v1/automation/fire-history", s.handleAutomationFireHistory)
	mux.HandleFunc("/v1/automation/update", s.handleAutomationUpdate)
	mux.HandleFunc("/v1/automation/delete", s.handleAutomationDelete)
	mux.HandleFunc("/v1/automation/run/schedule", s.handleAutomationRunSchedule)
	mux.HandleFunc("/v1/automation/run/comm-event", s.handleAutomationRunCommEvent)
	mux.HandleFunc("/v1/automation/comm-trigger/metadata", s.handleAutomationCommTriggerMetadata)
	mux.HandleFunc("/v1/automation/comm-trigger/validate", s.handleAutomationCommTriggerValidate)
}

func (s *Server) registerInspectRetentionContextRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v1/inspect/run", s.handleInspectRun)
	mux.HandleFunc("/v1/inspect/transcript", s.handleInspectTranscript)
	mux.HandleFunc("/v1/inspect/memory", s.handleInspectMemory)
	mux.HandleFunc("/v1/inspect/logs/query", s.handleInspectLogsQuery)
	mux.HandleFunc("/v1/inspect/logs/stream", s.handleInspectLogsStream)
	mux.HandleFunc("/v1/retention/purge", s.handleRetentionPurge)
	mux.HandleFunc("/v1/retention/compact-memory", s.handleRetentionCompactMemory)
	mux.HandleFunc("/v1/context/samples", s.handleContextSamples)
	mux.HandleFunc("/v1/context/tune", s.handleContextTune)
	mux.HandleFunc("/v1/context/memory/inventory", s.handleContextMemoryInventory)
	mux.HandleFunc("/v1/context/memory/compaction-candidates", s.handleContextMemoryCandidates)
	mux.HandleFunc("/v1/context/retrieval/documents", s.handleContextRetrievalDocuments)
	mux.HandleFunc("/v1/context/retrieval/chunks", s.handleContextRetrievalChunks)
}
