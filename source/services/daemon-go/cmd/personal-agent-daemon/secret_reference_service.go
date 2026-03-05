package main

import (
	"context"
	"errors"

	"personalagent/runtime/internal/daemonruntime"
	"personalagent/runtime/internal/securestore"
	"personalagent/runtime/internal/transport"
)

type daemonSecretReferenceService struct {
	container *daemonruntime.ServiceContainer
}

func (s *daemonSecretReferenceService) UpsertSecretReference(ctx context.Context, request transport.SecretReferenceUpsertRequest) (transport.SecretReferenceRecord, error) {
	record, err := s.container.RegisterSecretReference(ctx, securestore.SecretReference{
		WorkspaceID: request.WorkspaceID,
		Name:        request.Name,
		Backend:     request.Backend,
		Service:     request.Service,
		Account:     request.Account,
	})
	if err != nil {
		return transport.SecretReferenceRecord{}, err
	}

	return transport.SecretReferenceRecord{
		WorkspaceID: record.WorkspaceID,
		Name:        record.Name,
		Backend:     record.Backend,
		Service:     record.Service,
		Account:     record.Account,
	}, nil
}

func (s *daemonSecretReferenceService) GetSecretReference(ctx context.Context, workspaceID string, name string) (transport.SecretReferenceRecord, error) {
	record, err := s.container.GetSecretReference(ctx, workspaceID, name)
	if err != nil {
		if errors.Is(err, daemonruntime.ErrSecretReferenceNotFound) {
			return transport.SecretReferenceRecord{}, transport.ErrSecretReferenceNotFound
		}
		return transport.SecretReferenceRecord{}, err
	}

	return transport.SecretReferenceRecord{
		WorkspaceID: record.WorkspaceID,
		Name:        record.Name,
		Backend:     record.Backend,
		Service:     record.Service,
		Account:     record.Account,
	}, nil
}

func (s *daemonSecretReferenceService) DeleteSecretReference(ctx context.Context, workspaceID string, name string) (transport.SecretReferenceRecord, error) {
	record, err := s.container.DeleteSecretReference(ctx, workspaceID, name)
	if err != nil {
		if errors.Is(err, daemonruntime.ErrSecretReferenceNotFound) {
			return transport.SecretReferenceRecord{}, transport.ErrSecretReferenceNotFound
		}
		return transport.SecretReferenceRecord{}, err
	}

	return transport.SecretReferenceRecord{
		WorkspaceID: record.WorkspaceID,
		Name:        record.Name,
		Backend:     record.Backend,
		Service:     record.Service,
		Account:     record.Account,
	}, nil
}
