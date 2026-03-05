package transport

import "net/http"

func (s *Server) registerTaskApprovalWorkflowRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v1/tasks", s.handleSubmitTask)
	mux.HandleFunc("/v1/tasks/list", s.handleTaskRunList)
	mux.HandleFunc("/v1/tasks/cancel", s.handleTaskCancel)
	mux.HandleFunc("/v1/tasks/retry", s.handleTaskRetry)
	mux.HandleFunc("/v1/tasks/requeue", s.handleTaskRequeue)
	mux.HandleFunc("/v1/tasks/", s.handleTaskStatus)
	mux.HandleFunc("/v1/approvals/list", s.handleApprovalInbox)
	mux.HandleFunc("/v1/comm/threads/list", s.handleCommThreadList)
	mux.HandleFunc("/v1/comm/events/list", s.handleCommEventTimeline)
	mux.HandleFunc("/v1/comm/call-sessions/list", s.handleCommCallSessionList)
	mux.HandleFunc("/v1/capabilities/smoke", s.handleCapabilitySmoke)
	mux.HandleFunc("/v1/realtime/ws", s.handleRealtimeWS)
}
