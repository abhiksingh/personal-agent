package transport

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestInMemoryControlBackendCapabilitySmokeUsesCanonicalDefaults(t *testing.T) {
	backend := NewInMemoryControlBackend(NewEventBroker())
	response, err := backend.CapabilitySmoke(context.Background(), "corr-smoke-defaults")
	if err != nil {
		t.Fatalf("capability smoke: %v", err)
	}
	if !reflect.DeepEqual(response.Channels, DefaultCapabilitySmokeChannels()) {
		t.Fatalf("expected canonical channels %+v, got %+v", DefaultCapabilitySmokeChannels(), response.Channels)
	}
	if !reflect.DeepEqual(response.Connectors, DefaultCapabilitySmokeConnectors()) {
		t.Fatalf("expected canonical connectors %+v, got %+v", DefaultCapabilitySmokeConnectors(), response.Connectors)
	}
}

func TestCapabilitySmokeDefaultsReturnDefensiveCopies(t *testing.T) {
	channels := DefaultCapabilitySmokeChannels()
	connectors := DefaultCapabilitySmokeConnectors()
	channels[0] = "mutated"
	connectors[0] = "mutated"

	if got := DefaultCapabilitySmokeChannels()[0]; got == "mutated" {
		t.Fatalf("expected channels defaults to return defensive copy")
	}
	if got := DefaultCapabilitySmokeConnectors()[0]; got == "mutated" {
		t.Fatalf("expected connectors defaults to return defensive copy")
	}
}

func TestInMemoryControlBackendTaskControlErrorsExposeTypedMapping(t *testing.T) {
	backend := NewInMemoryControlBackend(nil)

	_, err := backend.CancelTask(context.Background(), TaskCancelRequest{}, "corr-cancel-missing")
	assertTaskControlDomainError(t, err, http.StatusBadRequest, "missing_required_field", taskControlErrorCategoryValidation)

	_, err = backend.TaskStatus(context.Background(), "task-missing", "corr-status-missing")
	assertTaskControlDomainError(t, err, http.StatusNotFound, "resource_not_found", taskControlErrorCategoryLookup)

	submitResponse, err := backend.SubmitTask(context.Background(), SubmitTaskRequest{
		WorkspaceID:             "ws1",
		RequestedByActorID:      "actor.requester",
		SubjectPrincipalActorID: "actor.subject",
		Title:                   "Domain error test task",
	}, "corr-submit")
	if err != nil {
		t.Fatalf("submit task: %v", err)
	}
	if strings.TrimSpace(submitResponse.RunID) == "" {
		t.Fatalf("expected run id in submit response")
	}

	_, err = backend.RetryTask(context.Background(), TaskRetryRequest{
		RunID: submitResponse.RunID,
	}, "corr-retry-conflict")
	assertTaskControlDomainError(t, err, http.StatusConflict, "resource_conflict", taskControlErrorCategoryStateConflict)
}

func assertTaskControlDomainError(t *testing.T, err error, statusCode int, code string, category string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error")
	}
	var domainErr TransportDomainError
	if !errors.As(err, &domainErr) {
		t.Fatalf("expected TransportDomainError, got %T", err)
	}
	if got := domainErr.TransportStatusCode(); got != statusCode {
		t.Fatalf("expected status %d, got %d", statusCode, got)
	}
	if got := strings.TrimSpace(domainErr.TransportErrorCode()); got != code {
		t.Fatalf("expected code %q, got %q", code, got)
	}
	details, ok := domainErr.TransportErrorDetails().(map[string]any)
	if !ok {
		t.Fatalf("expected error details map, got %T", domainErr.TransportErrorDetails())
	}
	if got := strings.TrimSpace(fmt.Sprint(details["category"])); got != category {
		t.Fatalf("expected details.category %q, got %q", category, got)
	}
}
