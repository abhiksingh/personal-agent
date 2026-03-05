package context

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"personalagent/runtime/internal/core/types"
)

type MemoryCompactor struct {
	now func() time.Time
}

func NewMemoryCompactor(nowFn func() time.Time) *MemoryCompactor {
	now := nowFn
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &MemoryCompactor{now: now}
}

func (c *MemoryCompactor) Compact(records []types.MemoryRecord, cfg types.MemoryCompactionConfig) types.MemoryCompactionResult {
	active := make([]types.MemoryRecord, 0, len(records))
	for _, record := range records {
		if strings.EqualFold(record.Status, "ACTIVE") {
			active = append(active, record)
		}
	}

	result := types.MemoryCompactionResult{OriginalTokens: totalTokens(active)}
	if cfg.TokenThreshold <= 0 || result.OriginalTokens <= cfg.TokenThreshold {
		result.KeptRecords = active
		result.FinalTokens = result.OriginalTokens
		return result
	}

	kept := []types.MemoryRecord{}
	staleConversational := []types.MemoryRecord{}
	now := c.now()
	for _, record := range active {
		if shouldAlwaysKeep(record) {
			kept = append(kept, record)
			continue
		}
		if cfg.StaleAfter > 0 && now.Sub(record.LastUpdatedAt) >= cfg.StaleAfter {
			staleConversational = append(staleConversational, record)
			continue
		}
		kept = append(kept, record)
	}

	if len(staleConversational) > 0 {
		summary := summarize(staleConversational)
		result.Summaries = append(result.Summaries, summary)
		for _, record := range staleConversational {
			result.DroppedIDs = append(result.DroppedIDs, record.ID)
		}
	}

	result.KeptRecords = kept
	result.FinalTokens = totalTokens(kept)
	for _, summary := range result.Summaries {
		result.FinalTokens += summary.TokenEstimate
	}
	return result
}

func shouldAlwaysKeep(record types.MemoryRecord) bool {
	if record.IsCanonical {
		return true
	}
	kind := strings.ToLower(strings.TrimSpace(record.Kind))
	return kind == "fact" || kind == "rule"
}

func summarize(records []types.MemoryRecord) types.MemorySummary {
	sort.Slice(records, func(i, j int) bool {
		return records[i].LastUpdatedAt.Before(records[j].LastUpdatedAt)
	})

	sourceIDs := make([]string, 0, len(records))
	sourceRefs := make([]string, 0, len(records))
	for _, record := range records {
		sourceIDs = append(sourceIDs, record.ID)
		if strings.TrimSpace(record.SourceRef) != "" {
			sourceRefs = append(sourceRefs, record.SourceRef)
		}
	}

	staleTokens := totalTokens(records)
	summaryTokens := staleTokens / 4
	if summaryTokens < 64 {
		summaryTokens = 64
	}

	content := fmt.Sprintf("Summarized %d stale conversational memories.", len(records))
	return types.MemorySummary{
		SummaryID:     fmt.Sprintf("summary-%d", len(records)),
		SourceIDs:     sourceIDs,
		SourceRefs:    sourceRefs,
		Content:       content,
		TokenEstimate: summaryTokens,
	}
}

func totalTokens(records []types.MemoryRecord) int {
	total := 0
	for _, record := range records {
		if record.TokenEstimate > 0 {
			total += record.TokenEstimate
		}
	}
	return total
}
