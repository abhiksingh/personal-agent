package securestore

import (
	"fmt"
	"sync"
)

type MemoryBackend struct {
	mu     sync.RWMutex
	values map[string]string
}

func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{values: map[string]string{}}
}

func (b *MemoryBackend) Set(service string, account string, value string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.values[composeKey(service, account)] = value
	return nil
}

func (b *MemoryBackend) Get(service string, account string) (string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	value, ok := b.values[composeKey(service, account)]
	if !ok {
		return "", ErrSecretNotFound
	}
	return value, nil
}

func (b *MemoryBackend) Delete(service string, account string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	key := composeKey(service, account)
	if _, ok := b.values[key]; !ok {
		return ErrSecretNotFound
	}
	delete(b.values, key)
	return nil
}

func composeKey(service string, account string) string {
	return fmt.Sprintf("%s|%s", service, account)
}
