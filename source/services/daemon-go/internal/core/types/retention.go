package types

import "time"

type RetentionPolicy struct {
	TraceDays      int
	TranscriptDays int
	MemoryDays     int
}

func DefaultRetentionPolicy() RetentionPolicy {
	return RetentionPolicy{
		TraceDays:      7,
		TranscriptDays: 7,
		MemoryDays:     7,
	}
}

type RetentionCutoffs struct {
	TraceBefore      time.Time
	TranscriptBefore time.Time
	MemoryBefore     time.Time
}

type RetentionPurgeConsistencyMode string

const (
	RetentionPurgeConsistencyModePartialSuccess RetentionPurgeConsistencyMode = "partial_success"
)

type RetentionPurgeStatus string

const (
	RetentionPurgeStatusCompleted      RetentionPurgeStatus = "completed"
	RetentionPurgeStatusPartialFailure RetentionPurgeStatus = "partial_failure"
)

type RetentionPurgeFailureStage string

const (
	RetentionPurgeFailureStageTrace      RetentionPurgeFailureStage = "trace"
	RetentionPurgeFailureStageTranscript RetentionPurgeFailureStage = "transcript"
	RetentionPurgeFailureStageMemory     RetentionPurgeFailureStage = "memory"
)

type RetentionPurgeFailureCode string

const (
	RetentionPurgeFailureCodeStatementFailed RetentionPurgeFailureCode = "statement_failed"
)

type RetentionPurgeFailure struct {
	Stage   RetentionPurgeFailureStage
	Code    RetentionPurgeFailureCode
	Details string
}

type RetentionPurgeResult struct {
	TracesDeleted      int64
	TranscriptsDeleted int64
	MemoryDeleted      int64
	ConsistencyMode    RetentionPurgeConsistencyMode
	Status             RetentionPurgeStatus
	Failure            *RetentionPurgeFailure
}
