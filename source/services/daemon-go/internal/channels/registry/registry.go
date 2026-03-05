package registry

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"personalagent/runtime/internal/channels/contract"
	shared "personalagent/runtime/internal/shared/contracts"
)

type Registry struct {
	mu       sync.RWMutex
	adapters map[string]contract.Adapter
}

func New() *Registry {
	return &Registry{adapters: map[string]contract.Adapter{}}
}

func (r *Registry) Register(adapter contract.Adapter) error {
	metadata := adapter.Metadata()
	if err := validateMetadata(metadata, shared.AdapterKindChannel); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.adapters[metadata.ID]; exists {
		return fmt.Errorf("channel adapter already registered: %s", metadata.ID)
	}
	r.adapters[metadata.ID] = adapter
	return nil
}

func (r *Registry) Get(adapterID string) (contract.Adapter, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	adapter, ok := r.adapters[adapterID]
	return adapter, ok
}

func (r *Registry) ListMetadata() []contract.Metadata {
	r.mu.RLock()
	defer r.mu.RUnlock()
	metadata := make([]contract.Metadata, 0, len(r.adapters))
	for _, adapter := range r.adapters {
		metadata = append(metadata, adapter.Metadata())
	}
	sort.Slice(metadata, func(i, j int) bool {
		return metadata[i].ID < metadata[j].ID
	})
	return metadata
}

func (r *Registry) ListCapabilityKeys() []string {
	keys := map[string]struct{}{}
	for _, metadata := range r.ListMetadata() {
		for _, capability := range metadata.Capabilities {
			key := strings.TrimSpace(capability.Key)
			if key == "" {
				continue
			}
			keys[key] = struct{}{}
		}
	}

	capabilityKeys := make([]string, 0, len(keys))
	for key := range keys {
		capabilityKeys = append(capabilityKeys, key)
	}
	sort.Strings(capabilityKeys)
	return capabilityKeys
}

func (r *Registry) FindByCapability(capabilityKey string) []contract.Adapter {
	key := strings.TrimSpace(capabilityKey)
	if key == "" {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0)
	for id, adapter := range r.adapters {
		if supportsCapability(adapter.Metadata(), key) {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)

	matches := make([]contract.Adapter, 0, len(ids))
	for _, id := range ids {
		matches = append(matches, r.adapters[id])
	}
	return matches
}

func (r *Registry) SelectByCapability(capabilityKey string, preferredAdapterID string) (contract.Adapter, error) {
	key := strings.TrimSpace(capabilityKey)
	if key == "" {
		return nil, fmt.Errorf("capability key is required")
	}

	preferredID := strings.TrimSpace(preferredAdapterID)
	if preferredID != "" {
		adapter, ok := r.Get(preferredID)
		if !ok {
			return nil, fmt.Errorf("preferred channel adapter not registered: %s", preferredID)
		}
		if !supportsCapability(adapter.Metadata(), key) {
			return nil, fmt.Errorf("preferred channel adapter %s does not support capability %s", preferredID, key)
		}
		return adapter, nil
	}

	matches := r.FindByCapability(key)
	if len(matches) == 0 {
		return nil, fmt.Errorf("no channel adapter supports capability: %s", key)
	}
	return matches[0], nil
}

func supportsCapability(metadata contract.Metadata, capabilityKey string) bool {
	for _, capability := range metadata.Capabilities {
		if strings.TrimSpace(capability.Key) == capabilityKey {
			return true
		}
	}
	return false
}

func validateMetadata(metadata contract.Metadata, expectedKind shared.AdapterKind) error {
	id := strings.TrimSpace(metadata.ID)
	if id == "" {
		return fmt.Errorf("channel adapter metadata.id is required")
	}
	if metadata.Kind != expectedKind {
		return fmt.Errorf("channel adapter %s has invalid kind %s", id, metadata.Kind)
	}
	if len(metadata.Capabilities) == 0 {
		return fmt.Errorf("channel adapter %s must declare at least one capability", id)
	}

	seen := map[string]struct{}{}
	for _, capability := range metadata.Capabilities {
		key := strings.TrimSpace(capability.Key)
		if key == "" {
			return fmt.Errorf("channel adapter %s has capability with empty key", id)
		}
		if _, exists := seen[key]; exists {
			return fmt.Errorf("channel adapter %s declares duplicate capability %s", id, key)
		}
		seen[key] = struct{}{}
	}
	return nil
}
