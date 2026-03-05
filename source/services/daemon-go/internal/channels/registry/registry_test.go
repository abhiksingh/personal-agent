package registry

import (
	"context"
	"testing"

	"personalagent/runtime/internal/channels/contract"
)

type fakeChannelAdapter struct {
	metadata contract.Metadata
}

func (a fakeChannelAdapter) Metadata() contract.Metadata {
	return a.metadata
}

func (a fakeChannelAdapter) HealthCheck(_ context.Context) error {
	return nil
}

func (a fakeChannelAdapter) ExecuteStep(_ context.Context, _ contract.ExecutionContext, _ contract.TaskStep) (contract.StepExecutionResult, error) {
	return contract.StepExecutionResult{}, nil
}

func TestRegisterRejectsInvalidChannelMetadata(t *testing.T) {
	registry := New()

	if err := registry.Register(fakeChannelAdapter{metadata: contract.Metadata{}}); err == nil {
		t.Fatalf("expected error for missing id")
	}

	if err := registry.Register(fakeChannelAdapter{metadata: contract.Metadata{
		ID:           "channel-wrong-kind",
		Kind:         "connector",
		Capabilities: []contract.CapabilityDescriptor{{Key: "send_message"}},
	}}); err == nil {
		t.Fatalf("expected error for wrong kind")
	}

	if err := registry.Register(fakeChannelAdapter{metadata: contract.Metadata{
		ID:           "channel-empty-cap",
		Kind:         "channel",
		Capabilities: []contract.CapabilityDescriptor{{Key: ""}},
	}}); err == nil {
		t.Fatalf("expected error for empty capability")
	}
}

func TestChannelRegistryDeterministicSelection(t *testing.T) {
	registry := New()

	beta := fakeChannelAdapter{metadata: contract.Metadata{
		ID:   "beta-adapter",
		Kind: "channel",
		Capabilities: []contract.CapabilityDescriptor{
			{Key: "send_message"},
			{Key: "voice_ack"},
		},
	}}
	alpha := fakeChannelAdapter{metadata: contract.Metadata{
		ID:   "alpha-adapter",
		Kind: "channel",
		Capabilities: []contract.CapabilityDescriptor{
			{Key: "send_message"},
		},
	}}

	if err := registry.Register(beta); err != nil {
		t.Fatalf("register beta adapter: %v", err)
	}
	if err := registry.Register(alpha); err != nil {
		t.Fatalf("register alpha adapter: %v", err)
	}

	selected, err := registry.SelectByCapability("send_message", "")
	if err != nil {
		t.Fatalf("select by capability: %v", err)
	}
	if selected.Metadata().ID != "alpha-adapter" {
		t.Fatalf("expected alpha-adapter due deterministic ordering, got %s", selected.Metadata().ID)
	}

	preferred, err := registry.SelectByCapability("send_message", "beta-adapter")
	if err != nil {
		t.Fatalf("select preferred adapter: %v", err)
	}
	if preferred.Metadata().ID != "beta-adapter" {
		t.Fatalf("expected preferred beta-adapter, got %s", preferred.Metadata().ID)
	}
}

func TestChannelRegistryCapabilityDiscovery(t *testing.T) {
	registry := New()

	if err := registry.Register(fakeChannelAdapter{metadata: contract.Metadata{
		ID:           "adapter-a",
		Kind:         "channel",
		Capabilities: []contract.CapabilityDescriptor{{Key: "send_message"}, {Key: "voice_ack"}},
	}}); err != nil {
		t.Fatalf("register adapter-a: %v", err)
	}
	if err := registry.Register(fakeChannelAdapter{metadata: contract.Metadata{
		ID:           "adapter-b",
		Kind:         "channel",
		Capabilities: []contract.CapabilityDescriptor{{Key: "send_message"}},
	}}); err != nil {
		t.Fatalf("register adapter-b: %v", err)
	}

	keys := registry.ListCapabilityKeys()
	if len(keys) != 2 {
		t.Fatalf("expected 2 capability keys, got %d", len(keys))
	}
	if keys[0] != "send_message" || keys[1] != "voice_ack" {
		t.Fatalf("unexpected capability key order: %v", keys)
	}

	matches := registry.FindByCapability("send_message")
	if len(matches) != 2 {
		t.Fatalf("expected 2 matching adapters, got %d", len(matches))
	}
	if matches[0].Metadata().ID != "adapter-a" || matches[1].Metadata().ID != "adapter-b" {
		t.Fatalf("expected deterministic adapter order, got [%s, %s]", matches[0].Metadata().ID, matches[1].Metadata().ID)
	}
}
