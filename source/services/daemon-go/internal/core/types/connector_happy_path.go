package types

type MailHappyPathRequest struct {
	WorkspaceID        string
	RunID              string
	RequestedByActor   string
	SubjectPrincipal   string
	ActingAsActor      string
	CorrelationID      string
	PreferredAdapterID string
}

type ConnectorStepTrace struct {
	StepName      string
	CapabilityKey string
	AdapterID     string
	Summary       string
	Evidence      map[string]string
}

type MailStepTrace = ConnectorStepTrace

type MailHappyPathResult struct {
	DraftTrace MailStepTrace
	SendTrace  MailStepTrace
	ReplyTrace MailStepTrace
}

type CalendarHappyPathRequest struct {
	WorkspaceID        string
	RunID              string
	RequestedByActor   string
	SubjectPrincipal   string
	ActingAsActor      string
	CorrelationID      string
	PreferredAdapterID string
}

type CalendarHappyPathResult struct {
	CreateTrace ConnectorStepTrace
	UpdateTrace ConnectorStepTrace
	CancelTrace ConnectorStepTrace
}

type BrowserHappyPathRequest struct {
	WorkspaceID        string
	RunID              string
	RequestedByActor   string
	SubjectPrincipal   string
	ActingAsActor      string
	CorrelationID      string
	TargetURL          string
	PreferredAdapterID string
}

type BrowserHappyPathResult struct {
	OpenTrace    ConnectorStepTrace
	ExtractTrace ConnectorStepTrace
	CloseTrace   ConnectorStepTrace
}

type FinderHappyPathRequest struct {
	WorkspaceID        string
	RunID              string
	RequestedByActor   string
	SubjectPrincipal   string
	ActingAsActor      string
	CorrelationID      string
	TargetPath         string
	TargetQuery        string
	SearchRootPath     string
	ApprovalPhrase     string
	PreferredAdapterID string
}

type FinderHappyPathResult struct {
	FindTrace    ConnectorStepTrace
	ListTrace    ConnectorStepTrace
	PreviewTrace ConnectorStepTrace
	DeleteTrace  ConnectorStepTrace
	GateDecision ApprovalGateDecision
}
