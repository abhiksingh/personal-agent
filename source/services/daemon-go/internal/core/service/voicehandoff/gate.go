package voicehandoff

import "personalagent/runtime/internal/core/types"

type Gate struct{}

func NewGate() *Gate {
	return &Gate{}
}

func (g *Gate) Evaluate(input types.VoiceHandoffInput) types.VoiceHandoffDecision {
	if input.Origin != types.ExecutionOriginVoice {
		return types.VoiceHandoffDecision{
			AllowExecution:       true,
			RequireInAppApproval: false,
			NextState:            types.VoiceHandoffStateRunning,
			Reason:               "non-voice origin",
		}
	}

	if !input.DestructiveAction {
		return types.VoiceHandoffDecision{
			AllowExecution:       true,
			RequireInAppApproval: false,
			NextState:            types.VoiceHandoffStateRunning,
			Reason:               "voice non-destructive action",
		}
	}

	if !input.InAppApprovalConfirmed {
		return types.VoiceHandoffDecision{
			AllowExecution:       false,
			RequireInAppApproval: true,
			NextState:            types.VoiceHandoffStateAwaitingApproval,
			Reason:               "voice destructive action requires in-app approval handoff",
		}
	}

	return types.VoiceHandoffDecision{
		AllowExecution:       true,
		RequireInAppApproval: false,
		NextState:            types.VoiceHandoffStateRunning,
		Reason:               "in-app approval confirmed",
	}
}
