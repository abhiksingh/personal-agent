package transport

// WorkflowRouteMetadata provides deterministic route context so clients can render
// provider/model/task-class state without inferring it from other payload fields.
type WorkflowRouteMetadata struct {
	Available       bool   `json:"available"`
	TaskClass       string `json:"task_class"`
	Provider        string `json:"provider,omitempty"`
	ModelKey        string `json:"model_key,omitempty"`
	TaskClassSource string `json:"task_class_source"`
	RouteSource     string `json:"route_source"`
	Notes           string `json:"notes,omitempty"`
}
