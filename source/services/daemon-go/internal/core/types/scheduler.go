package types

import "time"

type ScheduleTrigger struct {
	TriggerID             string
	WorkspaceID           string
	DirectiveID           string
	SubjectPrincipalActor string
	FilterJSON            string
	DirectiveTitle        string
	DirectiveInstruction  string
}

type ScheduleConfig struct {
	IntervalSeconds int `json:"interval_seconds"`
}

type TriggerFireReservation struct {
	FireID        string
	WorkspaceID   string
	TriggerID     string
	SourceEventID string
	FiredAt       time.Time
}

type ScheduleEvaluationResult struct {
	Processed int
	Created   int
	Skipped   int
	Failed    int
}
