package agentexec

import (
	"context"
	"fmt"
	"strings"
)

type DeterministicIntentInterpreter struct{}

func (DeterministicIntentInterpreter) Interpret(_ context.Context, _ string, request string) (Intent, error) {
	return InterpretIntent(request)
}

func NewDeterministicIntentInterpreter() IntentInterpreter {
	return DeterministicIntentInterpreter{}
}

type ModelAssistedIntentInterpreter struct {
	Extractor     ModelIntentExtractor
	MinConfidence float64
}

func NewModelAssistedIntentInterpreter(extractor ModelIntentExtractor, minConfidence float64) IntentInterpreter {
	return ModelAssistedIntentInterpreter{
		Extractor:     extractor,
		MinConfidence: minConfidence,
	}
}

func (i ModelAssistedIntentInterpreter) Interpret(ctx context.Context, workspaceID string, request string) (Intent, error) {
	trimmedRequest := strings.TrimSpace(request)
	if trimmedRequest == "" {
		return Intent{}, fmt.Errorf("request is required")
	}

	deterministicIntent, deterministicErr := InterpretIntent(trimmedRequest)
	extractor := i.Extractor
	if extractor != nil {
		candidate, candidateErr := extractor.ExtractIntent(ctx, strings.TrimSpace(workspaceID), trimmedRequest)
		if candidateErr == nil {
			minConfidence := i.MinConfidence
			if minConfidence <= 0 {
				minConfidence = 0.6
			}
			modelIntent, normalizeErr := normalizeModelIntentCandidate(candidate, trimmedRequest, minConfidence)
			if normalizeErr == nil {
				if modelIntent.RequiresClarification() && deterministicErr == nil && !deterministicIntent.RequiresClarification() {
					return deterministicIntent, nil
				}
				return modelIntent, nil
			}
		}
	}

	if deterministicErr == nil {
		return deterministicIntent, nil
	}
	return Intent{}, deterministicErr
}
