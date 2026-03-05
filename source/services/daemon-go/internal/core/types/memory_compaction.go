package types

import "time"

type MemoryRecord struct {
	ID            string
	Kind          string
	Status        string
	IsCanonical   bool
	TokenEstimate int
	Content       string
	SourceRef     string
	LastUpdatedAt time.Time
}

type MemoryCompactionConfig struct {
	TokenThreshold int
	StaleAfter     time.Duration
}

type MemorySummary struct {
	SummaryID     string
	SourceIDs     []string
	SourceRefs    []string
	Content       string
	TokenEstimate int
}

type MemoryCompactionResult struct {
	KeptRecords    []MemoryRecord
	Summaries      []MemorySummary
	DroppedIDs     []string
	OriginalTokens int
	FinalTokens    int
}
