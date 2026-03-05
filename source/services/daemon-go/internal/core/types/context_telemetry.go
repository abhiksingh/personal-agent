package types

import "time"

type ContextBudgetSample struct {
	SampleID         string
	WorkspaceID      string
	TaskClass        string
	ModelKey         string
	ContextWindow    int
	OutputLimit      int
	DeepAnalysis     bool
	RemainingBudget  int
	RetrievalTarget  int
	RetrievalUsed    int
	PromptTokens     int
	CompletionTokens int
	RecordedAt       time.Time
}

func (s ContextBudgetSample) TotalTokens() int {
	return s.PromptTokens + s.CompletionTokens
}

type ContextBudgetTuningProfile struct {
	WorkspaceID             string
	TaskClass               string
	RetrievalMultiplier     float64
	SampleCount             int
	AvgRetrievalUtilization float64
	AvgPromptUtilization    float64
	UpdatedAt               time.Time
}

type ContextBudgetTuningDecision struct {
	WorkspaceID             string
	TaskClass               string
	PreviousMultiplier      float64
	NewMultiplier           float64
	Changed                 bool
	Reason                  string
	SampleCount             int
	AvgRetrievalUtilization float64
	AvgPromptUtilization    float64
	EvaluatedAt             time.Time
}
