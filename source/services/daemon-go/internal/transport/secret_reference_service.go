package transport

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"personalagent/runtime/internal/workspaceid"
)

var ErrSecretReferenceNotFound = errors.New("secret reference not found")

type SecretReferenceService interface {
	UpsertSecretReference(ctx context.Context, request SecretReferenceUpsertRequest) (SecretReferenceRecord, error)
	GetSecretReference(ctx context.Context, workspaceID string, name string) (SecretReferenceRecord, error)
	DeleteSecretReference(ctx context.Context, workspaceID string, name string) (SecretReferenceRecord, error)
}

type InMemorySecretReferenceService struct {
	mu         sync.RWMutex
	references map[string]SecretReferenceRecord
}

func NewInMemorySecretReferenceService() *InMemorySecretReferenceService {
	return &InMemorySecretReferenceService{references: map[string]SecretReferenceRecord{}}
}

func (s *InMemorySecretReferenceService) UpsertSecretReference(_ context.Context, request SecretReferenceUpsertRequest) (SecretReferenceRecord, error) {
	workspace := normalizeSecretWorkspace(request.WorkspaceID)
	name := strings.TrimSpace(request.Name)
	if name == "" {
		return SecretReferenceRecord{}, fmt.Errorf("secret name is required")
	}
	service := strings.TrimSpace(request.Service)
	account := strings.TrimSpace(request.Account)
	if service == "" || account == "" {
		return SecretReferenceRecord{}, fmt.Errorf("secret service/account are required")
	}

	record := SecretReferenceRecord{
		WorkspaceID: workspace,
		Name:        name,
		Backend:     strings.TrimSpace(request.Backend),
		Service:     service,
		Account:     account,
	}

	s.mu.Lock()
	s.references[secretReferenceMapKey(workspace, name)] = record
	s.mu.Unlock()
	return record, nil
}

func (s *InMemorySecretReferenceService) GetSecretReference(_ context.Context, workspaceID string, name string) (SecretReferenceRecord, error) {
	workspace := normalizeSecretWorkspace(workspaceID)
	secretName := strings.TrimSpace(name)
	if secretName == "" {
		return SecretReferenceRecord{}, fmt.Errorf("secret name is required")
	}

	s.mu.RLock()
	record, ok := s.references[secretReferenceMapKey(workspace, secretName)]
	s.mu.RUnlock()
	if !ok {
		return SecretReferenceRecord{}, ErrSecretReferenceNotFound
	}
	return record, nil
}

func (s *InMemorySecretReferenceService) DeleteSecretReference(_ context.Context, workspaceID string, name string) (SecretReferenceRecord, error) {
	workspace := normalizeSecretWorkspace(workspaceID)
	secretName := strings.TrimSpace(name)
	if secretName == "" {
		return SecretReferenceRecord{}, fmt.Errorf("secret name is required")
	}

	key := secretReferenceMapKey(workspace, secretName)
	s.mu.Lock()
	record, ok := s.references[key]
	if ok {
		delete(s.references, key)
	}
	s.mu.Unlock()
	if !ok {
		return SecretReferenceRecord{}, ErrSecretReferenceNotFound
	}
	return record, nil
}

func normalizeSecretWorkspace(workspaceID string) string {
	return workspaceid.Normalize(workspaceID)
}

func secretReferenceMapKey(workspaceID string, name string) string {
	return workspaceID + "|" + name
}
