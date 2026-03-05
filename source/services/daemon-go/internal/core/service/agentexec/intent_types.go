package agentexec

import (
	"context"
	"regexp"
)

const (
	WorkflowMail     = "mail"
	WorkflowCalendar = "calendar"
	WorkflowMessages = "messages"
	WorkflowBrowser  = "browser"
	WorkflowFinder   = "finder"
)

const (
	mailOperationDraft           = "draft"
	mailOperationSend            = "send"
	mailOperationReply           = "reply"
	mailOperationSummarizeUnread = "summarize_unread"

	calendarOperationCreate = "create"
	calendarOperationUpdate = "update"
	calendarOperationCancel = "cancel"

	finderOperationFind    = "find"
	finderOperationList    = "list"
	finderOperationPreview = "preview"
	finderOperationDelete  = "delete"
)

const (
	slotTargetURL        = "target_url"
	slotTargetPath       = "target_path"
	slotFinderQuery      = "finder_query"
	slotMessageChannel   = "message_channel"
	slotMessageRecipient = "message_recipient"
	slotMessageBody      = "message_body"
	slotCalendarEventID  = "calendar_event_id"
	slotCalendarTitle    = "calendar_title"
)

var (
	emailAddressPattern         = regexp.MustCompile(`(?i)\b[A-Z0-9._%+\-]+@[A-Z0-9.\-]+\.[A-Z]{2,}\b`)
	phoneNumberPattern          = regexp.MustCompile(`\+?[0-9][0-9\-\s\(\)]{6,}[0-9]`)
	recipientHintRegex          = regexp.MustCompile(`(?i)\b(?:to|for)\s+([A-Za-z0-9_.+\-@]+)`)
	calendarEventIDHintPattern  = regexp.MustCompile(`(?i)\b(?:event[_\s-]*id)\s*[:=]?\s*([A-Za-z0-9][A-Za-z0-9._:-]{2,127})\b`)
	calendarEventIDTokenPattern = regexp.MustCompile(`(?i)\b(?:event|evt)-[A-Za-z0-9][A-Za-z0-9._:-]{1,127}\b`)
	finderQueryWordPattern      = regexp.MustCompile(`[A-Za-z0-9][A-Za-z0-9._-]*`)
	finderStopWords             = map[string]struct{}{
		"a": {}, "an": {}, "and": {}, "at": {}, "by": {}, "delete": {}, "directory": {}, "file": {}, "files": {},
		"find": {}, "finder": {}, "folder": {}, "for": {}, "in": {}, "inspect": {}, "list": {}, "locate": {}, "my": {},
		"now": {}, "of": {}, "on": {}, "open": {}, "path": {}, "please": {}, "preview": {}, "read": {}, "remove": {},
		"search": {}, "show": {}, "the": {}, "this": {}, "to": {}, "trash": {}, "up": {},
	}
)

type NativeAction struct {
	Connector string          `json:"connector"`
	Operation string          `json:"operation"`
	Mail      *MailAction     `json:"mail,omitempty"`
	Calendar  *CalendarAction `json:"calendar,omitempty"`
	Messages  *MessagesAction `json:"messages,omitempty"`
	Browser   *BrowserAction  `json:"browser,omitempty"`
	Finder    *FinderAction   `json:"finder,omitempty"`
}

type MailAction struct {
	Operation string `json:"operation"`
	Recipient string `json:"recipient,omitempty"`
	Subject   string `json:"subject,omitempty"`
	Body      string `json:"body,omitempty"`
	Limit     int    `json:"limit,omitempty"`
}

type CalendarAction struct {
	Operation string `json:"operation"`
	EventID   string `json:"event_id,omitempty"`
	Title     string `json:"title,omitempty"`
	Notes     string `json:"notes,omitempty"`
}

type MessagesAction struct {
	Operation string `json:"operation"`
	Channel   string `json:"channel,omitempty"`
	Recipient string `json:"recipient,omitempty"`
	Body      string `json:"body,omitempty"`
}

type BrowserAction struct {
	Operation string `json:"operation"`
	TargetURL string `json:"target_url,omitempty"`
	Query     string `json:"query,omitempty"`
}

type FinderAction struct {
	Operation  string `json:"operation"`
	TargetPath string `json:"target_path,omitempty"`
	Query      string `json:"query,omitempty"`
	RootPath   string `json:"root_path,omitempty"`
}

type Intent struct {
	Workflow            string
	TargetURL           string
	TargetPath          string
	RawRequest          string
	Action              NativeAction
	MissingSlots        []string
	ClarificationPrompt string
}

func (i Intent) RequiresClarification() bool {
	return len(i.MissingSlots) > 0
}

type IntentInterpreter interface {
	Interpret(ctx context.Context, workspaceID string, request string) (Intent, error)
}

type ModelIntentExtractor interface {
	ExtractIntent(ctx context.Context, workspaceID string, request string) (ModelIntentCandidate, error)
}

type ModelIntentCandidate struct {
	Workflow         string
	TargetURL        string
	TargetPath       string
	TargetQuery      string
	MessageChannel   string
	MessageRecipient string
	MessageBody      string
	Confidence       float64
	Rationale        string
}
