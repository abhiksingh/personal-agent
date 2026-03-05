package securestore

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"personalagent/runtime/internal/workspaceid"
)

var (
	namePattern       = regexp.MustCompile(`^[a-zA-Z0-9_.-]{1,128}$`)
	ErrSecretNotFound = errors.New("secret not found")
)

type Backend interface {
	Set(service string, account string, value string) error
	Get(service string, account string) (string, error)
	Delete(service string, account string) error
}

type SecretReference struct {
	WorkspaceID string `json:"workspace_id"`
	Name        string `json:"name"`
	Backend     string `json:"backend"`
	Service     string `json:"service"`
	Account     string `json:"account"`
}

type Manager struct {
	namespace string
	backendID string
	backend   Backend
}

func NewManager(namespace string, backendID string, backend Backend) (*Manager, error) {
	if strings.TrimSpace(namespace) == "" {
		return nil, fmt.Errorf("namespace is required")
	}
	if strings.TrimSpace(backendID) == "" {
		return nil, fmt.Errorf("backend id is required")
	}
	if backend == nil {
		return nil, fmt.Errorf("backend is required")
	}

	return &Manager{
		namespace: strings.TrimSpace(namespace),
		backendID: strings.TrimSpace(backendID),
		backend:   backend,
	}, nil
}

func (m *Manager) Put(workspaceID string, name string, value string) (SecretReference, error) {
	if strings.TrimSpace(value) == "" {
		return SecretReference{}, fmt.Errorf("secret value is required")
	}
	ref, err := m.reference(workspaceID, name)
	if err != nil {
		return SecretReference{}, err
	}
	if err := m.backend.Set(ref.Service, ref.Account, value); err != nil {
		return SecretReference{}, err
	}
	return ref, nil
}

func (m *Manager) Get(workspaceID string, name string) (SecretReference, string, error) {
	ref, err := m.reference(workspaceID, name)
	if err != nil {
		return SecretReference{}, "", err
	}
	value, err := m.backend.Get(ref.Service, ref.Account)
	if err != nil {
		if errors.Is(err, ErrSecretNotFound) {
			return SecretReference{}, "", ErrSecretNotFound
		}
		return SecretReference{}, "", err
	}
	return ref, value, nil
}

func (m *Manager) Delete(workspaceID string, name string) (SecretReference, error) {
	ref, err := m.reference(workspaceID, name)
	if err != nil {
		return SecretReference{}, err
	}
	if err := m.backend.Delete(ref.Service, ref.Account); err != nil {
		if errors.Is(err, ErrSecretNotFound) {
			return SecretReference{}, ErrSecretNotFound
		}
		return SecretReference{}, err
	}
	return ref, nil
}

func (m *Manager) reference(workspaceID string, name string) (SecretReference, error) {
	normalizedWorkspace := normalizeWorkspace(workspaceID)
	normalizedName := strings.TrimSpace(name)
	if !namePattern.MatchString(normalizedName) {
		return SecretReference{}, fmt.Errorf("invalid secret name %q", name)
	}

	service := fmt.Sprintf("%s.%s", m.namespace, normalizedWorkspace)
	return SecretReference{
		WorkspaceID: normalizedWorkspace,
		Name:        normalizedName,
		Backend:     m.backendID,
		Service:     service,
		Account:     normalizedName,
	}, nil
}

func normalizeWorkspace(workspaceID string) string {
	return workspaceid.Normalize(workspaceID)
}
