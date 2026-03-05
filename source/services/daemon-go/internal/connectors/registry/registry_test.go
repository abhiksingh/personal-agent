package registry

import (
	"context"
	"testing"

	"personalagent/runtime/internal/connectors/contract"
)

type fakeConnectorAdapter struct {
	metadata contract.Metadata
}

func (a fakeConnectorAdapter) Metadata() contract.Metadata {
	return a.metadata
}

func (a fakeConnectorAdapter) HealthCheck(_ context.Context) error {
	return nil
}

func (a fakeConnectorAdapter) ExecuteStep(_ context.Context, _ contract.ExecutionContext, _ contract.TaskStep) (contract.StepExecutionResult, error) {
	return contract.StepExecutionResult{}, nil
}

func TestRegisterRejectsInvalidConnectorMetadata(t *testing.T) {
	registry := New()

	if err := registry.Register(fakeConnectorAdapter{metadata: contract.Metadata{}}); err == nil {
		t.Fatalf("expected error for missing id")
	}

	if err := registry.Register(fakeConnectorAdapter{metadata: contract.Metadata{
		ID:           "connector-wrong-kind",
		Kind:         "channel",
		Capabilities: []contract.CapabilityDescriptor{{Key: "mail_send"}},
	}}); err == nil {
		t.Fatalf("expected error for wrong kind")
	}

	if err := registry.Register(fakeConnectorAdapter{metadata: contract.Metadata{
		ID:           "connector-empty-cap",
		Kind:         "connector",
		Capabilities: []contract.CapabilityDescriptor{{Key: ""}},
	}}); err == nil {
		t.Fatalf("expected error for empty capability")
	}
}

func TestConnectorRegistryDeterministicSelection(t *testing.T) {
	registry := New()

	beta := fakeConnectorAdapter{metadata: contract.Metadata{
		ID:   "beta-connector",
		Kind: "connector",
		Capabilities: []contract.CapabilityDescriptor{
			{Key: "mail_send"},
			{Key: "mail_draft"},
		},
	}}
	alpha := fakeConnectorAdapter{metadata: contract.Metadata{
		ID:   "alpha-connector",
		Kind: "connector",
		Capabilities: []contract.CapabilityDescriptor{
			{Key: "mail_send"},
		},
	}}

	if err := registry.Register(beta); err != nil {
		t.Fatalf("register beta connector: %v", err)
	}
	if err := registry.Register(alpha); err != nil {
		t.Fatalf("register alpha connector: %v", err)
	}

	selected, err := registry.SelectByCapability("mail_send", "")
	if err != nil {
		t.Fatalf("select by capability: %v", err)
	}
	if selected.Metadata().ID != "alpha-connector" {
		t.Fatalf("expected alpha-connector due deterministic ordering, got %s", selected.Metadata().ID)
	}

	preferred, err := registry.SelectByCapability("mail_send", "beta-connector")
	if err != nil {
		t.Fatalf("select preferred connector: %v", err)
	}
	if preferred.Metadata().ID != "beta-connector" {
		t.Fatalf("expected preferred beta-connector, got %s", preferred.Metadata().ID)
	}
}

func TestConnectorRegistryCapabilityDiscovery(t *testing.T) {
	registry := New()

	if err := registry.Register(fakeConnectorAdapter{metadata: contract.Metadata{
		ID:           "connector-a",
		Kind:         "connector",
		Capabilities: []contract.CapabilityDescriptor{{Key: "calendar_create"}, {Key: "mail_send"}},
	}}); err != nil {
		t.Fatalf("register connector-a: %v", err)
	}
	if err := registry.Register(fakeConnectorAdapter{metadata: contract.Metadata{
		ID:           "connector-b",
		Kind:         "connector",
		Capabilities: []contract.CapabilityDescriptor{{Key: "mail_send"}},
	}}); err != nil {
		t.Fatalf("register connector-b: %v", err)
	}

	keys := registry.ListCapabilityKeys()
	if len(keys) != 2 {
		t.Fatalf("expected 2 capability keys, got %d", len(keys))
	}
	if keys[0] != "calendar_create" || keys[1] != "mail_send" {
		t.Fatalf("unexpected capability key order: %v", keys)
	}

	matches := registry.FindByCapability("mail_send")
	if len(matches) != 2 {
		t.Fatalf("expected 2 matching connectors, got %d", len(matches))
	}
	if matches[0].Metadata().ID != "connector-a" || matches[1].Metadata().ID != "connector-b" {
		t.Fatalf("expected deterministic connector order, got [%s, %s]", matches[0].Metadata().ID, matches[1].Metadata().ID)
	}
}
