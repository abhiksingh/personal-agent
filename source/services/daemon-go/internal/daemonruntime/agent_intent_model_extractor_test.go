package daemonruntime

import "testing"

func TestParseIntentExtractionPayloadParsesJSONCandidate(t *testing.T) {
	candidate, err := parseIntentExtractionPayload(`{"workflow":"browser","target_url":"https://example.com","target_path":"","target_query":"","message_channel":"","message_recipient":"","message_body":"","confidence":0.82,"rationale":"contains URL"}`)
	if err != nil {
		t.Fatalf("parse intent extraction payload: %v", err)
	}
	if candidate.Workflow != "browser" {
		t.Fatalf("expected browser workflow, got %s", candidate.Workflow)
	}
	if candidate.TargetURL != "https://example.com" {
		t.Fatalf("expected target_url, got %s", candidate.TargetURL)
	}
	if candidate.Confidence != 0.82 {
		t.Fatalf("expected confidence 0.82, got %v", candidate.Confidence)
	}
}

func TestParseIntentExtractionPayloadExtractsJSONFromMarkdownFence(t *testing.T) {
	candidate, err := parseIntentExtractionPayload("```json\n{\"workflow\":\"mail\",\"target_url\":\"\",\"target_path\":\"\",\"target_query\":\"\",\"message_channel\":\"\",\"message_recipient\":\"\",\"message_body\":\"\",\"confidence\":\"0.77\",\"rationale\":\"email request\"}\n```")
	if err != nil {
		t.Fatalf("parse fenced payload: %v", err)
	}
	if candidate.Workflow != "mail" {
		t.Fatalf("expected mail workflow, got %s", candidate.Workflow)
	}
	if candidate.Confidence != 0.77 {
		t.Fatalf("expected parsed confidence 0.77, got %v", candidate.Confidence)
	}
}

func TestParseIntentExtractionPayloadParsesMessagesFields(t *testing.T) {
	candidate, err := parseIntentExtractionPayload(`{"workflow":"messages","target_url":"","target_path":"","target_query":"","message_channel":"imessage","message_recipient":"+15550001111","message_body":"hi there","confidence":0.91,"rationale":"explicit text request"}`)
	if err != nil {
		t.Fatalf("parse messages payload: %v", err)
	}
	if candidate.Workflow != "messages" {
		t.Fatalf("expected messages workflow, got %s", candidate.Workflow)
	}
	if candidate.MessageChannel != "imessage" {
		t.Fatalf("expected message_channel imessage, got %s", candidate.MessageChannel)
	}
	if candidate.MessageRecipient != "+15550001111" {
		t.Fatalf("expected message_recipient, got %s", candidate.MessageRecipient)
	}
	if candidate.MessageBody != "hi there" {
		t.Fatalf("expected message_body, got %s", candidate.MessageBody)
	}
}

func TestParseIntentExtractionPayloadParsesFinderQueryField(t *testing.T) {
	candidate, err := parseIntentExtractionPayload(`{"workflow":"finder","target_url":"","target_path":"","target_query":"budget report","message_channel":"","message_recipient":"","message_body":"","confidence":0.88,"rationale":"finder query"}`)
	if err != nil {
		t.Fatalf("parse finder payload: %v", err)
	}
	if candidate.Workflow != "finder" {
		t.Fatalf("expected finder workflow, got %s", candidate.Workflow)
	}
	if candidate.TargetQuery != "budget report" {
		t.Fatalf("expected target_query budget report, got %q", candidate.TargetQuery)
	}
}

func TestParseIntentExtractionPayloadRejectsMissingJSONObject(t *testing.T) {
	_, err := parseIntentExtractionPayload("model reply without json")
	if err == nil {
		t.Fatalf("expected missing JSON object error")
	}
}
