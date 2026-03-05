package daemonruntime

import (
	"context"
	"fmt"
	"strings"
	"time"

	"personalagent/runtime/internal/transport"
	"personalagent/runtime/internal/workspaceid"
)

func (s *IdentityDirectoryService) ListWorkspaces(ctx context.Context, request transport.IdentityWorkspacesRequest) (transport.IdentityWorkspacesResponse, error) {
	rows, err := s.loadWorkspaces(ctx, request.IncludeInactive)
	if err != nil {
		return transport.IdentityWorkspacesResponse{}, err
	}

	resolvedContext, err := s.resolveContext(ctx, "")
	if err != nil {
		return transport.IdentityWorkspacesResponse{}, err
	}

	records := make([]transport.IdentityWorkspaceRecord, 0, len(rows))
	for _, row := range rows {
		if isReservedSystemWorkspaceID(row.WorkspaceID) {
			continue
		}
		rowCanonicalID := normalizeWorkspaceID(row.WorkspaceID)
		records = append(records, transport.IdentityWorkspaceRecord{
			WorkspaceID:    rowCanonicalID,
			Name:           row.Name,
			Status:         row.Status,
			PrincipalCount: row.PrincipalCount,
			ActorCount:     row.ActorCount,
			HandleCount:    row.HandleCount,
			UpdatedAt:      row.UpdatedAt,
			IsActive:       rowCanonicalID == resolvedContext.WorkspaceID && resolvedContext.WorkspaceResolved,
		})
	}

	return transport.IdentityWorkspacesResponse{
		ActiveContext: resolvedContext,
		Workspaces:    records,
	}, nil
}

func (s *IdentityDirectoryService) ListPrincipals(ctx context.Context, request transport.IdentityPrincipalsRequest) (transport.IdentityPrincipalsResponse, error) {
	workspaceID := normalizeWorkspaceID(request.WorkspaceID)
	if isReservedSystemWorkspaceID(workspaceID) {
		return transport.IdentityPrincipalsResponse{}, fmt.Errorf("workspace %q is reserved for daemon runtime operations", workspaceID)
	}
	exists, err := s.workspaceExists(ctx, workspaceID)
	if err != nil {
		return transport.IdentityPrincipalsResponse{}, err
	}
	if !exists {
		return transport.IdentityPrincipalsResponse{}, fmt.Errorf("workspace %q not found", workspaceID)
	}

	contextRecord, err := s.resolveContext(ctx, workspaceID)
	if err != nil {
		return transport.IdentityPrincipalsResponse{}, err
	}
	principals, err := s.loadPrincipals(ctx, workspaceID, request.IncludeInactive)
	if err != nil {
		return transport.IdentityPrincipalsResponse{}, err
	}
	handlesByActor, err := s.loadActorHandles(ctx, workspaceID)
	if err != nil {
		return transport.IdentityPrincipalsResponse{}, err
	}

	records := make([]transport.IdentityPrincipalRecord, 0, len(principals))
	for _, principal := range principals {
		records = append(records, transport.IdentityPrincipalRecord{
			ActorID:         principal.ActorID,
			DisplayName:     principal.DisplayName,
			ActorType:       principal.ActorType,
			ActorStatus:     principal.ActorStatus,
			PrincipalStatus: principal.PrincipalStatus,
			Handles:         handlesByActor[principal.ActorID],
			IsActive:        contextRecord.PrincipalActorID != "" && principal.ActorID == contextRecord.PrincipalActorID,
		})
	}

	return transport.IdentityPrincipalsResponse{
		WorkspaceID:   workspaceID,
		ActiveContext: contextRecord,
		Principals:    records,
	}, nil
}

func (s *IdentityDirectoryService) GetActiveContext(ctx context.Context, request transport.IdentityActiveContextRequest) (transport.IdentityActiveContextResponse, error) {
	preferredWorkspace := strings.TrimSpace(request.WorkspaceID)
	if preferredWorkspace != "" {
		preferredWorkspace = normalizeWorkspaceID(preferredWorkspace)
		if isReservedSystemWorkspaceID(preferredWorkspace) {
			return transport.IdentityActiveContextResponse{}, fmt.Errorf("workspace %q is reserved for daemon runtime operations", preferredWorkspace)
		}
		exists, err := s.workspaceExists(ctx, preferredWorkspace)
		if err != nil {
			return transport.IdentityActiveContextResponse{}, err
		}
		if !exists {
			return transport.IdentityActiveContextResponse{}, fmt.Errorf("workspace %q not found", preferredWorkspace)
		}
	}

	record, err := s.resolveContext(ctx, preferredWorkspace)
	if err != nil {
		return transport.IdentityActiveContextResponse{}, err
	}
	return transport.IdentityActiveContextResponse{
		ActiveContext: record,
	}, nil
}

func (s *IdentityDirectoryService) SelectWorkspace(ctx context.Context, request transport.IdentityWorkspaceSelectRequest) (transport.IdentityActiveContextResponse, error) {
	workspaceID := normalizeWorkspaceID(request.WorkspaceID)
	if isReservedSystemWorkspaceID(workspaceID) {
		return transport.IdentityActiveContextResponse{}, fmt.Errorf("workspace %q is reserved for daemon runtime operations", workspaceID)
	}
	exists, err := s.workspaceExists(ctx, workspaceID)
	if err != nil {
		return transport.IdentityActiveContextResponse{}, err
	}
	if !exists {
		return transport.IdentityActiveContextResponse{}, fmt.Errorf("workspace %q not found", workspaceID)
	}

	principalActorID := strings.TrimSpace(request.PrincipalActorID)
	principalSource := identitySourceDerived
	if principalActorID != "" {
		valid, err := s.principalExists(ctx, workspaceID, principalActorID)
		if err != nil {
			return transport.IdentityActiveContextResponse{}, err
		}
		if !valid {
			return transport.IdentityActiveContextResponse{}, fmt.Errorf("principal actor %q is not registered in workspace %q", principalActorID, workspaceID)
		}
		principalSource = identitySourceSelected
	} else {
		firstPrincipal, ok, err := s.firstPrincipalActorID(ctx, workspaceID)
		if err != nil {
			return transport.IdentityActiveContextResponse{}, err
		}
		if ok {
			principalActorID = firstPrincipal
		} else {
			principalSource = identitySourceDefault
		}
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	source := normalizeIdentityMutationSource(request.Source)
	selectedState := s.setSelectedContext(identitySelectionState{
		WorkspaceID:      workspaceID,
		PrincipalActorID: principalActorID,
		WorkspaceSource:  identitySourceSelected,
		PrincipalSource:  principalSource,
		UpdatedAt:        now,
		MutationSource:   source,
		MutationReason:   identityMutationReasonExplicitSelect,
	})

	return transport.IdentityActiveContextResponse{
		ActiveContext: transport.IdentityActiveContext{
			WorkspaceID:       workspaceID,
			PrincipalActorID:  principalActorID,
			WorkspaceSource:   identitySourceSelected,
			PrincipalSource:   principalSource,
			LastUpdatedAt:     selectedState.UpdatedAt,
			WorkspaceResolved: true,
			MutationSource:    selectedState.MutationSource,
			MutationReason:    selectedState.MutationReason,
			SelectionVersion:  selectedState.Version,
		},
	}, nil
}

func (s *IdentityDirectoryService) resolveContext(ctx context.Context, preferredWorkspace string) (transport.IdentityActiveContext, error) {
	s.mu.RLock()
	selected := s.selected
	s.mu.RUnlock()

	workspaceID := strings.TrimSpace(preferredWorkspace)
	workspaceSource := identitySourceDefault
	lastUpdatedAt := strings.TrimSpace(selected.UpdatedAt)
	mutationSource := normalizeIdentityMutationSource(selected.MutationSource)
	mutationReason := normalizeIdentityMutationReason(selected.MutationReason)
	selectionVersion := selected.Version
	selectedWorkspaceID := ""
	usesSelectedWorkspace := false
	if trimmedSelectedWorkspace := strings.TrimSpace(selected.WorkspaceID); trimmedSelectedWorkspace != "" {
		normalizedSelectedWorkspace := normalizeWorkspaceID(trimmedSelectedWorkspace)
		if !isReservedSystemWorkspaceID(normalizedSelectedWorkspace) {
			selectedWorkspaceID = normalizedSelectedWorkspace
		}
	}

	if workspaceID != "" {
		workspaceID = normalizeWorkspaceID(workspaceID)
		if isReservedSystemWorkspaceID(workspaceID) {
			workspaceID = ""
			workspaceSource = identitySourceDerived
		} else {
			workspaceSource = identitySourceRequest
			mutationReason = identityMutationReasonRequest
		}
	} else if selectedWorkspaceID != "" {
		workspaceID = selectedWorkspaceID
		workspaceSource = normalizeIdentityWorkspaceSource(selected.WorkspaceSource)
	}

	resolved := false
	if workspaceID != "" {
		exists, err := s.workspaceExists(ctx, workspaceID)
		if err != nil {
			return transport.IdentityActiveContext{}, err
		}
		if exists {
			resolved = true
			usesSelectedWorkspace = strings.TrimSpace(preferredWorkspace) == "" && selectedWorkspaceID != "" && workspaceID == selectedWorkspaceID
		} else {
			workspaceID = ""
			workspaceSource = identitySourceDerived
		}
	}
	if !resolved {
		// Prefer canonical default workspace (ws1) when present so daemon-owned
		// background runtimes stay aligned with client defaults after restarts.
		canonicalDefaultWorkspace := workspaceid.CanonicalDefault
		if exists, err := s.workspaceExists(ctx, canonicalDefaultWorkspace); err != nil {
			return transport.IdentityActiveContext{}, err
		} else if exists {
			workspaceID = canonicalDefaultWorkspace
			workspaceSource = identitySourceDefault
			resolved = true
		} else {
			firstWorkspace, ok, err := s.firstWorkspaceID(ctx)
			if err != nil {
				return transport.IdentityActiveContext{}, err
			}
			if ok {
				workspaceID = firstWorkspace
				workspaceSource = identitySourceDerived
				resolved = true
			} else if workspaceID == "" {
				workspaceID = normalizeWorkspaceID("")
				workspaceSource = identitySourceDefault
			}
		}
	}

	principalActorID := ""
	principalSource := identitySourceDefault
	if workspaceID != "" && strings.TrimSpace(selected.PrincipalActorID) != "" && workspaceID == normalizeWorkspaceID(selected.WorkspaceID) {
		valid, err := s.principalExists(ctx, workspaceID, selected.PrincipalActorID)
		if err != nil {
			return transport.IdentityActiveContext{}, err
		}
		if valid {
			principalActorID = selected.PrincipalActorID
			principalSource = normalizeIdentityPrincipalSource(selected.PrincipalSource)
		}
	}
	if principalActorID == "" && workspaceID != "" {
		firstPrincipal, ok, err := s.firstPrincipalActorID(ctx, workspaceID)
		if err != nil {
			return transport.IdentityActiveContext{}, err
		}
		if ok {
			principalActorID = firstPrincipal
			principalSource = identitySourceDerived
		}
	}

	if strings.TrimSpace(preferredWorkspace) != "" {
		mutationSource = firstNonEmpty(mutationSource, identityMutationSourceDaemon)
		mutationReason = identityMutationReasonRequest
	} else if usesSelectedWorkspace {
		mutationSource = firstNonEmpty(mutationSource, identityMutationSourceDaemon)
		mutationReason = firstNonEmpty(mutationReason, identityMutationReasonExplicitSelect)
	} else {
		mutationSource = identityMutationSourceDaemon
		switch workspaceSource {
		case identitySourceDefault:
			mutationReason = identityMutationReasonDefault
		default:
			mutationReason = identityMutationReasonDerived
		}
	}
	if selectionVersion <= 0 {
		lastUpdatedAt = ""
	} else if strings.TrimSpace(lastUpdatedAt) == "" {
		lastUpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}

	return transport.IdentityActiveContext{
		WorkspaceID:       workspaceID,
		PrincipalActorID:  principalActorID,
		WorkspaceSource:   workspaceSource,
		PrincipalSource:   principalSource,
		LastUpdatedAt:     lastUpdatedAt,
		WorkspaceResolved: resolved,
		MutationSource:    mutationSource,
		MutationReason:    mutationReason,
		SelectionVersion:  selectionVersion,
	}, nil
}

func normalizeIdentityWorkspaceSource(raw string) string {
	switch strings.TrimSpace(raw) {
	case identitySourceDerived:
		return identitySourceDerived
	case identitySourceSelected:
		return identitySourceSelected
	default:
		return identitySourceSelected
	}
}

func normalizeIdentityPrincipalSource(raw string) string {
	switch strings.TrimSpace(raw) {
	case identitySourceDerived:
		return identitySourceDerived
	case identitySourceSelected:
		return identitySourceSelected
	case identitySourceDefault:
		return identitySourceDefault
	default:
		return identitySourceDerived
	}
}

func normalizeIdentityMutationSource(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case identityMutationSourceCLI:
		return identityMutationSourceCLI
	case identityMutationSourceApp:
		return identityMutationSourceApp
	case identityMutationSourceDaemon:
		return identityMutationSourceDaemon
	default:
		return identityMutationSourceDaemon
	}
}

func normalizeIdentityMutationReason(raw string) string {
	switch strings.TrimSpace(raw) {
	case identityMutationReasonExplicitSelect:
		return identityMutationReasonExplicitSelect
	case identityMutationReasonRequest:
		return identityMutationReasonRequest
	case identityMutationReasonDerived:
		return identityMutationReasonDerived
	case identityMutationReasonDefault:
		return identityMutationReasonDefault
	default:
		return ""
	}
}

func (s *IdentityDirectoryService) setSelectedContext(next identitySelectionState) identitySelectionState {
	s.mu.Lock()
	defer s.mu.Unlock()

	if strings.TrimSpace(next.UpdatedAt) == "" {
		next.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	next.MutationSource = normalizeIdentityMutationSource(next.MutationSource)
	next.MutationReason = firstNonEmpty(normalizeIdentityMutationReason(next.MutationReason), identityMutationReasonDefault)
	next.Version = s.selected.Version + 1
	if next.Version <= 0 {
		next.Version = 1
	}
	s.selected = next
	return s.selected
}
