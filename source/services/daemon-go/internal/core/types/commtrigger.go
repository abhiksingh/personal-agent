package types

type CommEventRecord struct {
	EventID          string
	WorkspaceID      string
	ThreadID         string
	Channel          string
	EventType        string
	Direction        string
	AssistantEmitted bool
	BodyText         string
	SenderAddress    string
}

type OnCommEventTrigger struct {
	TriggerID             string
	WorkspaceID           string
	DirectiveID           string
	SubjectPrincipalActor string
	FilterJSON            string
	DirectiveTitle        string
	DirectiveInstruction  string
}

type KeywordFilter struct {
	ContainsAny  []string `json:"contains_any"`
	ContainsAll  []string `json:"contains_all"`
	ExactPhrases []string `json:"exact_phrases"`
}

type CommEventTriggerFilter struct {
	Channels          []string      `json:"channels"`
	PrincipalActorIDs []string      `json:"principal_actor_ids"`
	SenderAllowlist   []string      `json:"sender_allowlist"`
	ThreadIDs         []string      `json:"thread_ids"`
	Keywords          KeywordFilter `json:"keywords"`
}

type CommTriggerEvaluationResult struct {
	Processed int
	Matched   int
	Created   int
	Skipped   int
	Failed    int
}
