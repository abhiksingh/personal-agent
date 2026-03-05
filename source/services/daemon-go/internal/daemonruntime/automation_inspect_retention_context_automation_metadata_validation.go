package daemonruntime

import (
	"context"
	"encoding/json"
	"strings"

	"personalagent/runtime/internal/core/types"
	"personalagent/runtime/internal/transport"
)

func (s *AutomationInspectRetentionContextService) AutomationCommTriggerMetadata(_ context.Context, _ transport.AutomationCommTriggerMetadataRequest) (transport.AutomationCommTriggerMetadataResponse, error) {
	return transport.AutomationCommTriggerMetadataResponse{
		TriggerType:          "ON_COMM_EVENT",
		RequiredDefaults:     automationCommTriggerRequiredDefaults(),
		IdempotencyKeyFields: []string{"workspace_id", "trigger_id", "source_event_id"},
		FilterDefaults:       automationCommTriggerEmptyFilter(),
		FilterSchema: []transport.AutomationCommTriggerFilterFieldSchema{
			{
				Field:          "channels",
				ValueType:      "string[]",
				MatchSemantics: "case_insensitive_equals_any",
				Description:    "Optional channel include list. Empty means all channels are eligible.",
			},
			{
				Field:          "principal_actor_ids",
				ValueType:      "string[]",
				MatchSemantics: "case_insensitive_equals_subject_principal",
				Description:    "Optional subject-principal allowlist. Values are matched against directive subject principal actor ID.",
			},
			{
				Field:          "sender_allowlist",
				ValueType:      "string[]",
				MatchSemantics: "case_insensitive_equals_any",
				Description:    "Optional sender-address allowlist for inbound event sender identity.",
			},
			{
				Field:          "thread_ids",
				ValueType:      "string[]",
				MatchSemantics: "case_insensitive_equals_any",
				Description:    "Optional conversation/thread include list.",
			},
			{
				Field:          "keywords.contains_any",
				ValueType:      "string[]",
				MatchSemantics: "case_insensitive_body_contains_any_term",
				Description:    "Optional keyword terms where any one term may match body text.",
			},
			{
				Field:          "keywords.contains_all",
				ValueType:      "string[]",
				MatchSemantics: "case_insensitive_body_contains_all_terms",
				Description:    "Optional keyword terms where all terms must match body text.",
			},
			{
				Field:          "keywords.exact_phrases",
				ValueType:      "string[]",
				MatchSemantics: "case_insensitive_body_contains_phrase",
				Description:    "Optional phrase list. At least one phrase must appear when the list is non-empty.",
			},
		},
		Compatibility: transport.AutomationCommTriggerMetadataCompatibility{
			PrincipalFilterBehavior: "principal_actor_ids compare against trigger subject_principal_actor_id",
			KeywordMatchBehavior:    "keyword matching is case-insensitive substring matching",
		},
	}, nil
}

func (s *AutomationInspectRetentionContextService) AutomationCommTriggerValidate(_ context.Context, request transport.AutomationCommTriggerValidateRequest) (transport.AutomationCommTriggerValidateResponse, error) {
	response := transport.AutomationCommTriggerValidateResponse{
		Valid:                true,
		TriggerType:          "ON_COMM_EVENT",
		RequiredDefaults:     automationCommTriggerRequiredDefaults(),
		NormalizedFilter:     automationCommTriggerEmptyFilter(),
		NormalizedFilterJSON: "{}",
		Errors:               []transport.AutomationCommTriggerValidationIssue{},
		Warnings:             []transport.AutomationCommTriggerValidationIssue{},
		Compatibility: transport.AutomationCommTriggerValidationCompatibility{
			Compatible:                  true,
			SubjectActorID:              strings.TrimSpace(request.SubjectActorID),
			SubjectMatchesPrincipalRule: true,
		},
	}

	filter, parseErrors := parseAutomationCommTriggerValidateFilter(request)
	if len(parseErrors) > 0 {
		response.Valid = false
		response.Compatibility.Compatible = false
		response.Errors = append(response.Errors, parseErrors...)
		return response, nil
	}

	normalized := normalizeAutomationCommTriggerFilter(filter)
	response.NormalizedFilter = normalized
	response.NormalizedFilterJSON = marshalAutomationCommTriggerFilterJSON(normalized)

	if automationCommTriggerFilterIsBroad(normalized) {
		response.Warnings = append(response.Warnings, transport.AutomationCommTriggerValidationIssue{
			Code:    "broad_filter_match",
			Field:   "filter",
			Message: "no optional filters are set; trigger may match all inbound non-assistant MESSAGE events",
		})
	}

	subjectActorID := strings.TrimSpace(request.SubjectActorID)
	if subjectActorID != "" && len(normalized.PrincipalActorIDs) > 0 {
		matched := automationCommTriggerContainsCaseInsensitive(normalized.PrincipalActorIDs, subjectActorID)
		response.Compatibility.SubjectMatchesPrincipalRule = matched
		if !matched {
			response.Compatibility.Compatible = false
			response.Warnings = append(response.Warnings, transport.AutomationCommTriggerValidationIssue{
				Code:    "subject_actor_not_in_principal_filter",
				Field:   "principal_actor_ids",
				Message: "subject_actor_id is not included in principal_actor_ids; trigger will not match for this subject actor",
			})
		}
	}

	return response, nil
}

func automationCommTriggerRequiredDefaults() transport.AutomationCommTriggerRequiredDefaults {
	return transport.AutomationCommTriggerRequiredDefaults{
		EventType:        "MESSAGE",
		Direction:        "INBOUND",
		AssistantEmitted: false,
	}
}

func automationCommTriggerEmptyFilter() transport.AutomationCommTriggerFilter {
	return transport.AutomationCommTriggerFilter{
		Channels:          []string{},
		PrincipalActorIDs: []string{},
		SenderAllowlist:   []string{},
		ThreadIDs:         []string{},
		Keywords: transport.AutomationCommTriggerKeywordFilter{
			ContainsAny:  []string{},
			ContainsAll:  []string{},
			ExactPhrases: []string{},
		},
	}
}

func parseAutomationCommTriggerValidateFilter(
	request transport.AutomationCommTriggerValidateRequest,
) (transport.AutomationCommTriggerFilter, []transport.AutomationCommTriggerValidationIssue) {
	filter := automationCommTriggerEmptyFilter()
	errors := make([]transport.AutomationCommTriggerValidationIssue, 0)

	if request.Filter != nil {
		filter = *request.Filter
		return filter, errors
	}
	return filter, errors
}

func automationCommTriggerFilterFromCore(filter types.CommEventTriggerFilter) transport.AutomationCommTriggerFilter {
	return transport.AutomationCommTriggerFilter{
		Channels:          append([]string(nil), filter.Channels...),
		PrincipalActorIDs: append([]string(nil), filter.PrincipalActorIDs...),
		SenderAllowlist:   append([]string(nil), filter.SenderAllowlist...),
		ThreadIDs:         append([]string(nil), filter.ThreadIDs...),
		Keywords: transport.AutomationCommTriggerKeywordFilter{
			ContainsAny:  append([]string(nil), filter.Keywords.ContainsAny...),
			ContainsAll:  append([]string(nil), filter.Keywords.ContainsAll...),
			ExactPhrases: append([]string(nil), filter.Keywords.ExactPhrases...),
		},
	}
}

func automationCommTriggerFilterToCore(filter transport.AutomationCommTriggerFilter) types.CommEventTriggerFilter {
	return types.CommEventTriggerFilter{
		Channels:          append([]string(nil), filter.Channels...),
		PrincipalActorIDs: append([]string(nil), filter.PrincipalActorIDs...),
		SenderAllowlist:   append([]string(nil), filter.SenderAllowlist...),
		ThreadIDs:         append([]string(nil), filter.ThreadIDs...),
		Keywords: types.KeywordFilter{
			ContainsAny:  append([]string(nil), filter.Keywords.ContainsAny...),
			ContainsAll:  append([]string(nil), filter.Keywords.ContainsAll...),
			ExactPhrases: append([]string(nil), filter.Keywords.ExactPhrases...),
		},
	}
}

func normalizeAutomationCommTriggerFilter(input transport.AutomationCommTriggerFilter) transport.AutomationCommTriggerFilter {
	return transport.AutomationCommTriggerFilter{
		Channels:          normalizeAutomationCommTriggerFilterValues(input.Channels),
		PrincipalActorIDs: normalizeAutomationCommTriggerFilterValues(input.PrincipalActorIDs),
		SenderAllowlist:   normalizeAutomationCommTriggerFilterValues(input.SenderAllowlist),
		ThreadIDs:         normalizeAutomationCommTriggerFilterValues(input.ThreadIDs),
		Keywords: transport.AutomationCommTriggerKeywordFilter{
			ContainsAny:  normalizeAutomationCommTriggerFilterValues(input.Keywords.ContainsAny),
			ContainsAll:  normalizeAutomationCommTriggerFilterValues(input.Keywords.ContainsAll),
			ExactPhrases: normalizeAutomationCommTriggerFilterValues(input.Keywords.ExactPhrases),
		},
	}
}

func normalizeAutomationCommTriggerFilterValues(values []string) []string {
	normalized := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		candidate := strings.ToLower(strings.TrimSpace(value))
		if candidate == "" {
			continue
		}
		if _, exists := seen[candidate]; exists {
			continue
		}
		seen[candidate] = struct{}{}
		normalized = append(normalized, candidate)
	}
	return normalized
}

func automationCommTriggerContainsCaseInsensitive(values []string, candidate string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(candidate)) {
			return true
		}
	}
	return false
}

func automationCommTriggerFilterIsBroad(filter transport.AutomationCommTriggerFilter) bool {
	return len(filter.Channels) == 0 &&
		len(filter.PrincipalActorIDs) == 0 &&
		len(filter.SenderAllowlist) == 0 &&
		len(filter.ThreadIDs) == 0 &&
		len(filter.Keywords.ContainsAny) == 0 &&
		len(filter.Keywords.ContainsAll) == 0 &&
		len(filter.Keywords.ExactPhrases) == 0
}

func marshalAutomationCommTriggerFilterJSON(filter transport.AutomationCommTriggerFilter) string {
	coreFilter := automationCommTriggerFilterToCore(filter)
	encoded, err := json.Marshal(coreFilter)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}
