package daemonruntime

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"personalagent/runtime/internal/transport"
)

const modelToolSchemaRegistryVersionV1 = "model_tool_schema_registry.v1"

const (
	toolSchemaErrorMissingRequiredArgument = "tool_schema_missing_required_argument"
	toolSchemaErrorInvalidArgumentType     = "tool_schema_invalid_argument_type"
	toolSchemaErrorInvalidArgumentValue    = "tool_schema_invalid_argument_value"
	toolSchemaErrorUnknownArgument         = "tool_schema_unknown_argument"
)

type modelToolSchemaRegistry struct {
	Version string
	Entries []modelToolSchemaRegistryEntry
	index   map[string]int
}

type modelToolSchemaRegistryEntry struct {
	Tool           modelToolDefinition
	CapabilityKey  string
	InputSchema    map[string]any
	Policy         ToolPolicyResult
	PlannerVisible bool
}

type modelToolArgumentValidationError struct {
	Code        string
	ToolName    string
	Argument    string
	Expected    string
	Description string
}

func (e modelToolArgumentValidationError) Error() string {
	return strings.TrimSpace(e.Description)
}

func (s *UnifiedTurnService) buildToolSchemaRegistry(
	ctx context.Context,
	request transport.ChatTurnRequest,
	workspace string,
	tools []modelToolDefinition,
) (modelToolSchemaRegistry, error) {
	entries := make([]modelToolSchemaRegistryEntry, 0, len(tools))
	for _, tool := range tools {
		capabilityKey := normalizePolicyKey(firstCapabilityKey(tool))
		metadata, metadataFound := toolCapabilityPolicyCatalog[capabilityKey]
		if !metadataFound {
			metadata = toolCapabilityPolicyMetadata{
				CapabilityKey: capabilityKeyOrUnknown(capabilityKey),
				RiskClass:     toolRiskClassUnknown,
				ApprovalMode:  toolApprovalModeNever,
			}
		}

		policyResult := buildToolPolicyResult(ToolPolicyRequest{
			WorkspaceID:        workspace,
			RequestedByActorID: strings.TrimSpace(request.RequestedByActorID),
			ActingAsActorID:    strings.TrimSpace(request.ActingAsActorID),
			ChannelID:          normalizeTurnChannelID(request.Channel.ChannelID),
			ToolName:           strings.TrimSpace(tool.Name),
			CapabilityKey:      capabilityKey,
		}, ToolPolicyDecisionAllow, metadata, "policy_engine_unavailable", "tool policy evaluation unavailable", "fallback")

		if s.policy != nil {
			evaluated, err := s.policy.Evaluate(ctx, ToolPolicyRequest{
				WorkspaceID:        workspace,
				RequestedByActorID: strings.TrimSpace(request.RequestedByActorID),
				ActingAsActorID:    strings.TrimSpace(request.ActingAsActorID),
				ChannelID:          normalizeTurnChannelID(request.Channel.ChannelID),
				ToolName:           strings.TrimSpace(tool.Name),
				CapabilityKey:      capabilityKey,
			})
			if err != nil {
				return modelToolSchemaRegistry{}, fmt.Errorf("evaluate tool policy for %s: %w", strings.TrimSpace(tool.Name), err)
			}
			policyResult = evaluated
		}

		entries = append(entries, modelToolSchemaRegistryEntry{
			Tool:           tool,
			CapabilityKey:  capabilityKeyOrUnknown(capabilityKey),
			InputSchema:    modelToolInputSchema(tool),
			Policy:         policyResult,
			PlannerVisible: policyResult.Decision != ToolPolicyDecisionDeny,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(strings.TrimSpace(entries[i].Tool.Name)) < strings.ToLower(strings.TrimSpace(entries[j].Tool.Name))
	})

	index := make(map[string]int, len(entries))
	for i, entry := range entries {
		index[normalizePolicyKey(entry.Tool.Name)] = i
	}
	return modelToolSchemaRegistry{
		Version: modelToolSchemaRegistryVersionV1,
		Entries: entries,
		index:   index,
	}, nil
}

func (r modelToolSchemaRegistry) plannerTools() []modelToolDefinition {
	tools := make([]modelToolDefinition, 0, len(r.Entries))
	for _, entry := range r.Entries {
		if !entry.PlannerVisible {
			continue
		}
		tools = append(tools, entry.Tool)
	}
	return tools
}

func (r modelToolSchemaRegistry) plannerToolCatalogEntries() []transport.ChatTurnToolCatalogEntry {
	return r.toolCatalogEntries(true)
}

func (r modelToolSchemaRegistry) allToolCatalogEntries() []transport.ChatTurnToolCatalogEntry {
	return r.toolCatalogEntries(false)
}

func (r modelToolSchemaRegistry) toolCatalogEntries(plannerOnly bool) []transport.ChatTurnToolCatalogEntry {
	entries := make([]transport.ChatTurnToolCatalogEntry, 0, len(r.Entries))
	for _, entry := range r.Entries {
		if plannerOnly && !entry.PlannerVisible {
			continue
		}
		capabilityKeys := append([]string(nil), entry.Tool.CapabilityKeys...)
		sort.Strings(capabilityKeys)
		entries = append(entries, transport.ChatTurnToolCatalogEntry{
			Name:           strings.TrimSpace(entry.Tool.Name),
			Description:    strings.TrimSpace(entry.Tool.Description),
			CapabilityKeys: capabilityKeys,
			InputSchema:    cloneAnyMap(entry.InputSchema),
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(strings.TrimSpace(entries[i].Name)) < strings.ToLower(strings.TrimSpace(entries[j].Name))
	})
	return entries
}

func (r modelToolSchemaRegistry) findTool(name string) (modelToolSchemaRegistryEntry, bool) {
	if len(r.Entries) == 0 {
		return modelToolSchemaRegistryEntry{}, false
	}
	index, ok := r.index[normalizePolicyKey(name)]
	if !ok || index < 0 || index >= len(r.Entries) {
		return modelToolSchemaRegistryEntry{}, false
	}
	return r.Entries[index], true
}

func validateToolArgumentsAgainstRegistry(entry modelToolSchemaRegistryEntry, arguments map[string]any) error {
	tool := entry.Tool
	if arguments == nil {
		arguments = map[string]any{}
	}

	argumentNames := make([]string, 0, len(tool.Arguments))
	for argumentName := range tool.Arguments {
		argumentNames = append(argumentNames, argumentName)
	}
	sort.Strings(argumentNames)

	for _, argumentName := range argumentNames {
		spec := tool.Arguments[argumentName]
		value, exists := arguments[argumentName]
		if spec.Required && (!exists || strings.TrimSpace(fmt.Sprintf("%v", value)) == "") {
			return modelToolArgumentValidationError{
				Code:        toolSchemaErrorMissingRequiredArgument,
				ToolName:    strings.TrimSpace(tool.Name),
				Argument:    argumentName,
				Expected:    strings.TrimSpace(spec.Type),
				Description: fmt.Sprintf("tool %s missing required argument %s", strings.TrimSpace(tool.Name), argumentName),
			}
		}
		if !exists {
			continue
		}

		switch strings.ToLower(strings.TrimSpace(spec.Type)) {
		case "string":
			stringValue, ok := value.(string)
			if !ok {
				return modelToolArgumentValidationError{
					Code:        toolSchemaErrorInvalidArgumentType,
					ToolName:    strings.TrimSpace(tool.Name),
					Argument:    argumentName,
					Expected:    "string",
					Description: fmt.Sprintf("tool %s argument %s must be string", strings.TrimSpace(tool.Name), argumentName),
				}
			}
			if len(spec.EnumOptions) > 0 {
				normalizedValue := strings.ToLower(strings.TrimSpace(stringValue))
				matched := false
				for _, option := range spec.EnumOptions {
					if normalizedValue == strings.ToLower(strings.TrimSpace(option)) {
						matched = true
						break
					}
				}
				if !matched {
					return modelToolArgumentValidationError{
						Code:        toolSchemaErrorInvalidArgumentValue,
						ToolName:    strings.TrimSpace(tool.Name),
						Argument:    argumentName,
						Expected:    "one_of:" + strings.Join(spec.EnumOptions, ","),
						Description: fmt.Sprintf("tool %s argument %s must be one of %s", strings.TrimSpace(tool.Name), argumentName, strings.Join(spec.EnumOptions, ",")),
					}
				}
			}
		case "integer":
			switch typed := value.(type) {
			case int, int32, int64:
			case float64:
				if typed != float64(int(typed)) {
					return modelToolArgumentValidationError{
						Code:        toolSchemaErrorInvalidArgumentType,
						ToolName:    strings.TrimSpace(tool.Name),
						Argument:    argumentName,
						Expected:    "integer",
						Description: fmt.Sprintf("tool %s argument %s must be integer", strings.TrimSpace(tool.Name), argumentName),
					}
				}
			default:
				return modelToolArgumentValidationError{
					Code:        toolSchemaErrorInvalidArgumentType,
					ToolName:    strings.TrimSpace(tool.Name),
					Argument:    argumentName,
					Expected:    "integer",
					Description: fmt.Sprintf("tool %s argument %s must be integer", strings.TrimSpace(tool.Name), argumentName),
				}
			}
		}
	}

	unsupportedKeys := make([]string, 0)
	for argumentName := range arguments {
		if _, ok := tool.Arguments[argumentName]; !ok {
			unsupportedKeys = append(unsupportedKeys, argumentName)
		}
	}
	sort.Strings(unsupportedKeys)
	if len(unsupportedKeys) > 0 {
		unsupported := strings.TrimSpace(unsupportedKeys[0])
		return modelToolArgumentValidationError{
			Code:        toolSchemaErrorUnknownArgument,
			ToolName:    strings.TrimSpace(tool.Name),
			Argument:    unsupported,
			Expected:    "known_schema_field",
			Description: fmt.Sprintf("tool %s received unsupported argument %s", strings.TrimSpace(tool.Name), unsupported),
		}
	}
	return nil
}
