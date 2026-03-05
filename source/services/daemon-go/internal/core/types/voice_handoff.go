package types

type ExecutionOrigin string

const (
	ExecutionOriginApp   ExecutionOrigin = "app"
	ExecutionOriginCLI   ExecutionOrigin = "cli"
	ExecutionOriginVoice ExecutionOrigin = "voice"
)

type VoiceHandoffInput struct {
	Origin                 ExecutionOrigin
	DestructiveAction      bool
	InAppApprovalConfirmed bool
}

type VoiceHandoffState string

const (
	VoiceHandoffStateAwaitingApproval VoiceHandoffState = "awaiting_approval"
	VoiceHandoffStateRunning          VoiceHandoffState = "running"
)

type VoiceHandoffDecision struct {
	AllowExecution       bool
	RequireInAppApproval bool
	NextState            VoiceHandoffState
	Reason               string
}
