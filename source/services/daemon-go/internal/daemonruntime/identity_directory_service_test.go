package daemonruntime

import (
	"context"
	"database/sql"
	"reflect"
	"strings"
	"testing"

	"personalagent/runtime/internal/transport"
)

func TestIdentityDirectoryListPrincipalsIncludesHandlesAndActiveContext(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedIdentityDirectoryFixtures(t, container.DB)

	service, err := NewIdentityDirectoryService(container)
	if err != nil {
		t.Fatalf("new identity directory service: %v", err)
	}

	_, err = service.SelectWorkspace(context.Background(), transport.IdentityWorkspaceSelectRequest{
		WorkspaceID:      "ws-alpha",
		PrincipalActorID: "actor.alice",
	})
	if err != nil {
		t.Fatalf("select workspace: %v", err)
	}

	response, err := service.ListPrincipals(context.Background(), transport.IdentityPrincipalsRequest{
		WorkspaceID: "ws-alpha",
	})
	if err != nil {
		t.Fatalf("list principals: %v", err)
	}
	if response.WorkspaceID != "ws-alpha" {
		t.Fatalf("expected workspace ws-alpha, got %s", response.WorkspaceID)
	}
	if !response.ActiveContext.WorkspaceResolved || response.ActiveContext.WorkspaceID != "ws-alpha" {
		t.Fatalf("expected resolved active context for ws-alpha, got %+v", response.ActiveContext)
	}
	if response.ActiveContext.PrincipalActorID != "actor.alice" {
		t.Fatalf("expected active principal actor.alice, got %+v", response.ActiveContext)
	}
	if response.ActiveContext.WorkspaceSource != identitySourceRequest || response.ActiveContext.PrincipalSource != identitySourceSelected {
		t.Fatalf("expected request workspace source and selected principal source, got %+v", response.ActiveContext)
	}
	if len(response.Principals) != 2 {
		t.Fatalf("expected 2 principals, got %d", len(response.Principals))
	}

	var alice *transport.IdentityPrincipalRecord
	for idx := range response.Principals {
		if response.Principals[idx].ActorID == "actor.alice" {
			alice = &response.Principals[idx]
			break
		}
	}
	if alice == nil {
		t.Fatalf("expected actor.alice in principal response: %+v", response.Principals)
	}
	if !alice.IsActive {
		t.Fatalf("expected actor.alice to be active principal")
	}
	if len(alice.Handles) != 2 {
		t.Fatalf("expected actor.alice handle mapping, got %+v", alice.Handles)
	}
}

func TestIdentityDirectorySelectWorkspaceRejectsUnknownWorkspace(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	service, err := NewIdentityDirectoryService(container)
	if err != nil {
		t.Fatalf("new identity directory service: %v", err)
	}

	_, err = service.SelectWorkspace(context.Background(), transport.IdentityWorkspaceSelectRequest{
		WorkspaceID: "ws-missing",
	})
	if err == nil {
		t.Fatalf("expected select workspace to fail for unknown workspace")
	}
}

func TestIdentityDirectorySelectWorkspaceRejectsReservedSystemWorkspace(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	service, err := NewIdentityDirectoryService(container)
	if err != nil {
		t.Fatalf("new identity directory service: %v", err)
	}

	_, err = service.SelectWorkspace(context.Background(), transport.IdentityWorkspaceSelectRequest{
		WorkspaceID: daemonPluginAuditWorkspaceID,
	})
	if err == nil {
		t.Fatalf("expected select workspace to fail for reserved system workspace")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "reserved") {
		t.Fatalf("expected reserved workspace validation error, got %v", err)
	}
}

func TestIdentityDirectoryListWorkspacesReturnsCountsAndSelectedWorkspace(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedIdentityDirectoryFixtures(t, container.DB)

	service, err := NewIdentityDirectoryService(container)
	if err != nil {
		t.Fatalf("new identity directory service: %v", err)
	}
	if _, err := service.SelectWorkspace(context.Background(), transport.IdentityWorkspaceSelectRequest{
		WorkspaceID: "ws-beta",
	}); err != nil {
		t.Fatalf("select workspace: %v", err)
	}

	response, err := service.ListWorkspaces(context.Background(), transport.IdentityWorkspacesRequest{IncludeInactive: true})
	if err != nil {
		t.Fatalf("list workspaces: %v", err)
	}
	if len(response.Workspaces) != 2 {
		t.Fatalf("expected 2 workspace rows, got %d", len(response.Workspaces))
	}
	if response.ActiveContext.WorkspaceID != "ws-beta" || !response.ActiveContext.WorkspaceResolved {
		t.Fatalf("expected ws-beta selected as active context, got %+v", response.ActiveContext)
	}

	alphaRecord := findWorkspaceRecord(response.Workspaces, "ws-alpha")
	if alphaRecord == nil {
		t.Fatalf("expected ws-alpha record in response: %+v", response.Workspaces)
	}
	if alphaRecord.PrincipalCount != 2 || alphaRecord.ActorCount != 2 || alphaRecord.HandleCount != 3 {
		t.Fatalf("unexpected ws-alpha aggregate counts: %+v", alphaRecord)
	}

	betaRecord := findWorkspaceRecord(response.Workspaces, "ws-beta")
	if betaRecord == nil {
		t.Fatalf("expected ws-beta record in response: %+v", response.Workspaces)
	}
	if !betaRecord.IsActive {
		t.Fatalf("expected ws-beta to be marked active workspace")
	}
}

func TestIdentityDirectoryDerivedWorkspaceRemainsStableWhenUpdatedAtRecencyChanges(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedIdentityDirectoryFixtures(t, container.DB)

	service, err := NewIdentityDirectoryService(container)
	if err != nil {
		t.Fatalf("new identity directory service: %v", err)
	}

	initial, err := service.GetActiveContext(context.Background(), transport.IdentityActiveContextRequest{})
	if err != nil {
		t.Fatalf("get active context: %v", err)
	}
	if initial.ActiveContext.WorkspaceID != "ws-alpha" {
		t.Fatalf("expected deterministic derived workspace ws-alpha, got %+v", initial.ActiveContext)
	}
	if initial.ActiveContext.WorkspaceSource != identitySourceDerived {
		t.Fatalf("expected derived workspace source, got %+v", initial.ActiveContext)
	}
	if initial.ActiveContext.SelectionVersion != 0 {
		t.Fatalf("expected derived fallback to keep selection version at 0, got %+v", initial.ActiveContext)
	}
	initialUpdatedAt := initial.ActiveContext.LastUpdatedAt

	if _, err := container.DB.Exec(`
		UPDATE workspaces
		SET updated_at = '2099-01-01T00:00:00Z'
		WHERE id = 'ws-beta'
	`); err != nil {
		t.Fatalf("update ws-beta updated_at: %v", err)
	}

	listing, err := service.ListWorkspaces(context.Background(), transport.IdentityWorkspacesRequest{IncludeInactive: true})
	if err != nil {
		t.Fatalf("list workspaces: %v", err)
	}
	if listing.ActiveContext.WorkspaceID != initial.ActiveContext.WorkspaceID {
		t.Fatalf("expected active workspace to remain %s, got %+v", initial.ActiveContext.WorkspaceID, listing.ActiveContext)
	}
	if listing.ActiveContext.LastUpdatedAt != initialUpdatedAt {
		t.Fatalf("expected stable active-context timestamp %s, got %+v", initialUpdatedAt, listing.ActiveContext)
	}

	after, err := service.GetActiveContext(context.Background(), transport.IdentityActiveContextRequest{})
	if err != nil {
		t.Fatalf("get active context after updated_at change: %v", err)
	}
	if after.ActiveContext.WorkspaceID != initial.ActiveContext.WorkspaceID {
		t.Fatalf("expected active workspace to remain %s, got %+v", initial.ActiveContext.WorkspaceID, after.ActiveContext)
	}
	if after.ActiveContext.WorkspaceSource != identitySourceDerived {
		t.Fatalf("expected active workspace source to remain derived, got %+v", after.ActiveContext)
	}
	if after.ActiveContext.LastUpdatedAt != initialUpdatedAt {
		t.Fatalf("expected stable active-context timestamp %s, got %+v", initialUpdatedAt, after.ActiveContext)
	}
	if after.ActiveContext.SelectionVersion != 0 {
		t.Fatalf("expected derived fallback to keep selection version at 0, got %+v", after.ActiveContext)
	}
}

func TestIdentityDirectoryResolveContextPrefersCanonicalDefaultWorkspaceWhenPresent(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)

	statements := []string{
		`INSERT INTO workspaces(id, name, status, created_at, updated_at) VALUES ('daemon', 'daemon', 'ACTIVE', '2026-02-25T00:00:00Z', '2026-02-25T00:00:00Z')`,
		`INSERT INTO workspaces(id, name, status, created_at, updated_at) VALUES ('ws1', 'ws1', 'ACTIVE', '2026-02-25T01:00:00Z', '2026-02-25T01:00:00Z')`,
	}
	for _, statement := range statements {
		if _, err := container.DB.Exec(statement); err != nil {
			t.Fatalf("seed canonical workspace preference fixtures: %v\nstatement: %s", err, statement)
		}
	}

	service, err := NewIdentityDirectoryService(container)
	if err != nil {
		t.Fatalf("new identity directory service: %v", err)
	}

	response, err := service.GetActiveContext(context.Background(), transport.IdentityActiveContextRequest{})
	if err != nil {
		t.Fatalf("get active context: %v", err)
	}
	if response.ActiveContext.WorkspaceID != "ws1" {
		t.Fatalf("expected canonical default workspace ws1 to be preferred, got %+v", response.ActiveContext)
	}
	if response.ActiveContext.WorkspaceSource != identitySourceDefault {
		t.Fatalf("expected canonical default workspace source=%s, got %+v", identitySourceDefault, response.ActiveContext)
	}
	if response.ActiveContext.MutationReason != identityMutationReasonDefault {
		t.Fatalf("expected canonical default mutation reason=%s, got %+v", identityMutationReasonDefault, response.ActiveContext)
	}
}

func TestIdentityDirectoryResolveContextPreservesExplicitDefaultWorkspaceID(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	if _, err := container.DB.Exec(`
		INSERT INTO workspaces(id, name, status, created_at, updated_at)
		VALUES ('default', 'default', 'ACTIVE', '2026-02-25T00:00:00Z', '2026-02-25T00:00:00Z')
	`); err != nil {
		t.Fatalf("seed legacy default workspace fixture: %v", err)
	}

	service, err := NewIdentityDirectoryService(container)
	if err != nil {
		t.Fatalf("new identity directory service: %v", err)
	}

	response, err := service.GetActiveContext(context.Background(), transport.IdentityActiveContextRequest{})
	if err != nil {
		t.Fatalf("get active context: %v", err)
	}
	if response.ActiveContext.WorkspaceID != "default" {
		t.Fatalf("expected explicit workspace id default to be preserved, got %+v", response.ActiveContext)
	}
	if !response.ActiveContext.WorkspaceResolved {
		t.Fatalf("expected active context workspace to resolve, got %+v", response.ActiveContext)
	}
}

func TestIdentityDirectoryResolveContextSkipsReservedSystemWorkspaceFallback(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)

	statements := []string{
		`INSERT INTO workspaces(id, name, status, created_at, updated_at) VALUES ('daemon', 'daemon', 'ACTIVE', '2026-02-25T00:00:00Z', '2026-02-25T00:00:00Z')`,
		`INSERT INTO workspaces(id, name, status, created_at, updated_at) VALUES ('ws-alpha', 'Workspace Alpha', 'ACTIVE', '2026-02-25T01:00:00Z', '2026-02-25T01:00:00Z')`,
	}
	for _, statement := range statements {
		if _, err := container.DB.Exec(statement); err != nil {
			t.Fatalf("seed reserved workspace fallback fixtures: %v\nstatement: %s", err, statement)
		}
	}

	service, err := NewIdentityDirectoryService(container)
	if err != nil {
		t.Fatalf("new identity directory service: %v", err)
	}

	response, err := service.GetActiveContext(context.Background(), transport.IdentityActiveContextRequest{})
	if err != nil {
		t.Fatalf("get active context: %v", err)
	}
	if response.ActiveContext.WorkspaceID != "ws-alpha" {
		t.Fatalf("expected reserved daemon workspace to be skipped in fallback selection, got %+v", response.ActiveContext)
	}
}

func TestIdentityDirectoryListWorkspacesExcludesReservedSystemWorkspace(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)

	statements := []string{
		`INSERT INTO workspaces(id, name, status, created_at, updated_at) VALUES ('daemon', 'daemon', 'ACTIVE', '2026-02-25T00:00:00Z', '2026-02-25T00:00:00Z')`,
		`INSERT INTO workspaces(id, name, status, created_at, updated_at) VALUES ('ws-alpha', 'Workspace Alpha', 'ACTIVE', '2026-02-25T01:00:00Z', '2026-02-25T01:00:00Z')`,
	}
	for _, statement := range statements {
		if _, err := container.DB.Exec(statement); err != nil {
			t.Fatalf("seed reserved workspace list fixtures: %v\nstatement: %s", err, statement)
		}
	}

	service, err := NewIdentityDirectoryService(container)
	if err != nil {
		t.Fatalf("new identity directory service: %v", err)
	}

	response, err := service.ListWorkspaces(context.Background(), transport.IdentityWorkspacesRequest{IncludeInactive: true})
	if err != nil {
		t.Fatalf("list workspaces: %v", err)
	}
	if len(response.Workspaces) != 1 {
		t.Fatalf("expected reserved daemon workspace to be excluded from list, got %+v", response.Workspaces)
	}
	if response.Workspaces[0].WorkspaceID != "ws-alpha" {
		t.Fatalf("expected ws-alpha in listed workspaces, got %+v", response.Workspaces[0])
	}
}

func TestIdentityDirectoryActiveWorkspaceStableAcrossConfigStatusAndCommQueries(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedIdentityDirectoryFixtures(t, container.DB)

	identityService, err := NewIdentityDirectoryService(container)
	if err != nil {
		t.Fatalf("new identity directory service: %v", err)
	}
	uiStatusService, err := NewUIStatusService(container)
	if err != nil {
		t.Fatalf("new ui status service: %v", err)
	}
	agentService, err := NewAgentDelegationService(container)
	if err != nil {
		t.Fatalf("new agent delegation service: %v", err)
	}

	initial, err := identityService.GetActiveContext(context.Background(), transport.IdentityActiveContextRequest{})
	if err != nil {
		t.Fatalf("get initial active context: %v", err)
	}
	activeWorkspace := initial.ActiveContext.WorkspaceID
	if activeWorkspace != "ws-alpha" {
		t.Fatalf("expected deterministic active workspace ws-alpha, got %+v", initial.ActiveContext)
	}
	if initial.ActiveContext.SelectionVersion != 0 {
		t.Fatalf("expected no selected context version for derived fallback, got %+v", initial.ActiveContext)
	}
	initialUpdatedAt := initial.ActiveContext.LastUpdatedAt

	if _, err := uiStatusService.UpsertChannelConfig(context.Background(), transport.ChannelConfigUpsertRequest{
		WorkspaceID:   "ws-beta",
		ChannelID:     "app",
		Merge:         true,
		Configuration: transport.UIStatusConfigurationFromMap(map[string]any{"enabled": true}),
	}); err != nil {
		t.Fatalf("upsert channel config in ws-beta: %v", err)
	}

	if _, err := uiStatusService.ListChannelStatus(context.Background(), transport.ChannelStatusRequest{
		WorkspaceID: activeWorkspace,
	}); err != nil {
		t.Fatalf("list channel status for active workspace: %v", err)
	}

	if _, err := agentService.ListCommThreads(context.Background(), transport.CommThreadListRequest{
		WorkspaceID: activeWorkspace,
		Limit:       10,
	}); err != nil {
		t.Fatalf("list comm threads for active workspace: %v", err)
	}

	after, err := identityService.ListWorkspaces(context.Background(), transport.IdentityWorkspacesRequest{IncludeInactive: true})
	if err != nil {
		t.Fatalf("list workspaces after query sequence: %v", err)
	}
	if after.ActiveContext.WorkspaceID != activeWorkspace {
		t.Fatalf("expected active workspace to remain %s, got %+v", activeWorkspace, after.ActiveContext)
	}
	if after.ActiveContext.LastUpdatedAt != initialUpdatedAt {
		t.Fatalf("expected active-context timestamp to remain %s, got %+v", initialUpdatedAt, after.ActiveContext)
	}
}

func TestIdentityDirectoryExplicitSelectionRemainsStickyUntilChanged(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedIdentityDirectoryFixtures(t, container.DB)

	identityService, err := NewIdentityDirectoryService(container)
	if err != nil {
		t.Fatalf("new identity directory service: %v", err)
	}
	uiStatusService, err := NewUIStatusService(container)
	if err != nil {
		t.Fatalf("new ui status service: %v", err)
	}
	agentService, err := NewAgentDelegationService(container)
	if err != nil {
		t.Fatalf("new agent delegation service: %v", err)
	}

	selected, err := identityService.SelectWorkspace(context.Background(), transport.IdentityWorkspaceSelectRequest{
		WorkspaceID:      "ws-beta",
		PrincipalActorID: "actor.beta",
	})
	if err != nil {
		t.Fatalf("select ws-beta: %v", err)
	}
	selectedUpdatedAt := selected.ActiveContext.LastUpdatedAt
	if selected.ActiveContext.WorkspaceID != "ws-beta" || selected.ActiveContext.PrincipalActorID != "actor.beta" {
		t.Fatalf("unexpected selected context payload: %+v", selected.ActiveContext)
	}

	if _, err := uiStatusService.UpsertChannelConfig(context.Background(), transport.ChannelConfigUpsertRequest{
		WorkspaceID:   "ws-alpha",
		ChannelID:     "app",
		Merge:         true,
		Configuration: transport.UIStatusConfigurationFromMap(map[string]any{"enabled": true}),
	}); err != nil {
		t.Fatalf("upsert ws-alpha channel config: %v", err)
	}

	if _, err := uiStatusService.ListChannelStatus(context.Background(), transport.ChannelStatusRequest{
		WorkspaceID: "ws-alpha",
	}); err != nil {
		t.Fatalf("list ws-alpha channel status: %v", err)
	}
	if _, err := agentService.ListCommThreads(context.Background(), transport.CommThreadListRequest{
		WorkspaceID: "ws-alpha",
		Limit:       10,
	}); err != nil {
		t.Fatalf("list ws-alpha comm threads: %v", err)
	}

	sticky, err := identityService.GetActiveContext(context.Background(), transport.IdentityActiveContextRequest{})
	if err != nil {
		t.Fatalf("get sticky active context: %v", err)
	}
	if sticky.ActiveContext.WorkspaceID != "ws-beta" || sticky.ActiveContext.PrincipalActorID != "actor.beta" {
		t.Fatalf("expected active context to remain ws-beta/actor.beta, got %+v", sticky.ActiveContext)
	}
	if sticky.ActiveContext.WorkspaceSource != identitySourceSelected || sticky.ActiveContext.PrincipalSource != identitySourceSelected {
		t.Fatalf("expected selected context sources to remain selected, got %+v", sticky.ActiveContext)
	}
	if sticky.ActiveContext.LastUpdatedAt != selectedUpdatedAt {
		t.Fatalf("expected stable selected-context timestamp %s, got %+v", selectedUpdatedAt, sticky.ActiveContext)
	}

	if _, err := identityService.SelectWorkspace(context.Background(), transport.IdentityWorkspaceSelectRequest{
		WorkspaceID:      "ws-alpha",
		PrincipalActorID: "actor.alice",
	}); err != nil {
		t.Fatalf("select ws-alpha: %v", err)
	}
	afterReselect, err := identityService.GetActiveContext(context.Background(), transport.IdentityActiveContextRequest{})
	if err != nil {
		t.Fatalf("get active context after explicit reselection: %v", err)
	}
	if afterReselect.ActiveContext.WorkspaceID != "ws-alpha" || afterReselect.ActiveContext.PrincipalActorID != "actor.alice" {
		t.Fatalf("expected explicit reselection to update context, got %+v", afterReselect.ActiveContext)
	}
	if afterReselect.ActiveContext.WorkspaceSource != identitySourceSelected || afterReselect.ActiveContext.PrincipalSource != identitySourceSelected {
		t.Fatalf("expected selected context sources after reselection, got %+v", afterReselect.ActiveContext)
	}
}

func TestIdentityDirectoryImessageVisibilityRemainsStableAcrossSectionRefresh(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedIdentityDirectoryFixtures(t, container.DB)
	seedIdentityCommVisibilityFixtures(t, container.DB)

	identityService, err := NewIdentityDirectoryService(container)
	if err != nil {
		t.Fatalf("new identity directory service: %v", err)
	}
	uiStatusService, err := NewUIStatusService(container)
	if err != nil {
		t.Fatalf("new ui status service: %v", err)
	}
	agentService, err := NewAgentDelegationService(container)
	if err != nil {
		t.Fatalf("new agent delegation service: %v", err)
	}

	if _, err := identityService.SelectWorkspace(context.Background(), transport.IdentityWorkspaceSelectRequest{
		WorkspaceID:      "ws-alpha",
		PrincipalActorID: "actor.alice",
	}); err != nil {
		t.Fatalf("select ws-alpha: %v", err)
	}

	baselineThreads, err := agentService.ListCommThreads(context.Background(), transport.CommThreadListRequest{
		WorkspaceID: "ws-alpha",
		Channel:     "imessage",
		Limit:       25,
	})
	if err != nil {
		t.Fatalf("baseline list ws-alpha iMessage threads: %v", err)
	}
	if len(baselineThreads.Items) == 0 {
		t.Fatalf("expected at least one ws-alpha iMessage thread for baseline")
	}
	baselineThreadIDs := commThreadIDs(baselineThreads.Items)

	baselineEvents, err := agentService.ListCommEvents(context.Background(), transport.CommEventTimelineRequest{
		WorkspaceID: "ws-alpha",
		Channel:     "imessage",
		Limit:       25,
	})
	if err != nil {
		t.Fatalf("baseline list ws-alpha iMessage events: %v", err)
	}
	if len(baselineEvents.Items) == 0 {
		t.Fatalf("expected at least one ws-alpha iMessage event for baseline")
	}
	baselineEventIDs := commEventIDs(baselineEvents.Items)

	for cycle := 0; cycle < 3; cycle++ {
		if _, err := uiStatusService.ListChannelStatus(context.Background(), transport.ChannelStatusRequest{
			WorkspaceID: "ws-alpha",
		}); err != nil {
			t.Fatalf("cycle %d list ws-alpha channel status: %v", cycle, err)
		}
		if _, err := uiStatusService.ListChannelStatus(context.Background(), transport.ChannelStatusRequest{
			WorkspaceID: "ws-beta",
		}); err != nil {
			t.Fatalf("cycle %d list ws-beta channel status: %v", cycle, err)
		}

		activeContext, err := identityService.GetActiveContext(context.Background(), transport.IdentityActiveContextRequest{})
		if err != nil {
			t.Fatalf("cycle %d get active context: %v", cycle, err)
		}
		if activeContext.ActiveContext.WorkspaceID != "ws-alpha" {
			t.Fatalf("cycle %d expected selected workspace ws-alpha, got %+v", cycle, activeContext.ActiveContext)
		}

		threadsAfter, err := agentService.ListCommThreads(context.Background(), transport.CommThreadListRequest{
			WorkspaceID: activeContext.ActiveContext.WorkspaceID,
			Channel:     "imessage",
			Limit:       25,
		})
		if err != nil {
			t.Fatalf("cycle %d list iMessage threads: %v", cycle, err)
		}
		eventsAfter, err := agentService.ListCommEvents(context.Background(), transport.CommEventTimelineRequest{
			WorkspaceID: activeContext.ActiveContext.WorkspaceID,
			Channel:     "imessage",
			Limit:       25,
		})
		if err != nil {
			t.Fatalf("cycle %d list iMessage events: %v", cycle, err)
		}

		if !reflect.DeepEqual(commThreadIDs(threadsAfter.Items), baselineThreadIDs) {
			t.Fatalf(
				"cycle %d expected stable iMessage thread visibility in ws-alpha; baseline=%v current=%v",
				cycle,
				baselineThreadIDs,
				commThreadIDs(threadsAfter.Items),
			)
		}
		if !reflect.DeepEqual(commEventIDs(eventsAfter.Items), baselineEventIDs) {
			t.Fatalf(
				"cycle %d expected stable iMessage event visibility in ws-alpha; baseline=%v current=%v",
				cycle,
				baselineEventIDs,
				commEventIDs(eventsAfter.Items),
			)
		}
	}
}

func TestIdentityDirectorySelectionLastWriteWinsWithMutationMetadata(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedIdentityDirectoryFixtures(t, container.DB)

	service, err := NewIdentityDirectoryService(container)
	if err != nil {
		t.Fatalf("new identity directory service: %v", err)
	}

	first, err := service.SelectWorkspace(context.Background(), transport.IdentityWorkspaceSelectRequest{
		WorkspaceID:      "ws-alpha",
		PrincipalActorID: "actor.alice",
		Source:           "cli",
	})
	if err != nil {
		t.Fatalf("select ws-alpha from cli: %v", err)
	}
	if first.ActiveContext.SelectionVersion <= 0 {
		t.Fatalf("expected positive selection version on first select, got %+v", first.ActiveContext)
	}
	if first.ActiveContext.MutationSource != "cli" || first.ActiveContext.MutationReason != identityMutationReasonExplicitSelect {
		t.Fatalf("unexpected mutation metadata for first select: %+v", first.ActiveContext)
	}

	second, err := service.SelectWorkspace(context.Background(), transport.IdentityWorkspaceSelectRequest{
		WorkspaceID:      "ws-beta",
		PrincipalActorID: "actor.beta",
		Source:           "app",
	})
	if err != nil {
		t.Fatalf("select ws-beta from app: %v", err)
	}
	if second.ActiveContext.SelectionVersion <= first.ActiveContext.SelectionVersion {
		t.Fatalf("expected monotonic selection version, got first=%d second=%d", first.ActiveContext.SelectionVersion, second.ActiveContext.SelectionVersion)
	}
	if second.ActiveContext.MutationSource != "app" || second.ActiveContext.MutationReason != identityMutationReasonExplicitSelect {
		t.Fatalf("unexpected mutation metadata for second select: %+v", second.ActiveContext)
	}

	active, err := service.GetActiveContext(context.Background(), transport.IdentityActiveContextRequest{})
	if err != nil {
		t.Fatalf("get active context: %v", err)
	}
	if active.ActiveContext.WorkspaceID != "ws-beta" || active.ActiveContext.PrincipalActorID != "actor.beta" {
		t.Fatalf("expected last-write selection ws-beta/actor.beta, got %+v", active.ActiveContext)
	}
	if active.ActiveContext.MutationSource != "app" || active.ActiveContext.MutationReason != identityMutationReasonExplicitSelect {
		t.Fatalf("expected active mutation metadata to reflect app write, got %+v", active.ActiveContext)
	}
	if active.ActiveContext.SelectionVersion != second.ActiveContext.SelectionVersion {
		t.Fatalf("expected active selection version %d, got %d", second.ActiveContext.SelectionVersion, active.ActiveContext.SelectionVersion)
	}
}

func TestIdentityDirectoryRequestOverrideDoesNotMutateSelectedContext(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedIdentityDirectoryFixtures(t, container.DB)

	service, err := NewIdentityDirectoryService(container)
	if err != nil {
		t.Fatalf("new identity directory service: %v", err)
	}

	selected, err := service.SelectWorkspace(context.Background(), transport.IdentityWorkspaceSelectRequest{
		WorkspaceID:      "ws-beta",
		PrincipalActorID: "actor.beta",
		Source:           "app",
	})
	if err != nil {
		t.Fatalf("select ws-beta from app: %v", err)
	}

	override, err := service.GetActiveContext(context.Background(), transport.IdentityActiveContextRequest{
		WorkspaceID: "ws-alpha",
	})
	if err != nil {
		t.Fatalf("get active context with request override: %v", err)
	}
	if override.ActiveContext.WorkspaceID != "ws-alpha" || override.ActiveContext.WorkspaceSource != identitySourceRequest {
		t.Fatalf("expected request override workspace metadata, got %+v", override.ActiveContext)
	}
	if override.ActiveContext.MutationReason != identityMutationReasonRequest {
		t.Fatalf("expected request override mutation reason, got %+v", override.ActiveContext)
	}
	if override.ActiveContext.SelectionVersion != selected.ActiveContext.SelectionVersion {
		t.Fatalf("expected request override to preserve selection version %d, got %d", selected.ActiveContext.SelectionVersion, override.ActiveContext.SelectionVersion)
	}

	after, err := service.GetActiveContext(context.Background(), transport.IdentityActiveContextRequest{})
	if err != nil {
		t.Fatalf("get active context after request override: %v", err)
	}
	if after.ActiveContext.WorkspaceID != "ws-beta" || after.ActiveContext.PrincipalActorID != "actor.beta" {
		t.Fatalf("expected selected workspace/principal to remain ws-beta/actor.beta, got %+v", after.ActiveContext)
	}
	if after.ActiveContext.SelectionVersion != selected.ActiveContext.SelectionVersion {
		t.Fatalf("expected selected version to remain %d after request override, got %d", selected.ActiveContext.SelectionVersion, after.ActiveContext.SelectionVersion)
	}
}

func TestIdentityDirectoryPassiveReadsDoNotAdvanceSelectionVersion(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedIdentityDirectoryFixtures(t, container.DB)

	service, err := NewIdentityDirectoryService(container)
	if err != nil {
		t.Fatalf("new identity directory service: %v", err)
	}

	firstContext, err := service.GetActiveContext(context.Background(), transport.IdentityActiveContextRequest{})
	if err != nil {
		t.Fatalf("first get active context: %v", err)
	}
	if firstContext.ActiveContext.SelectionVersion != 0 {
		t.Fatalf("expected first passive read to keep selection version 0, got %+v", firstContext.ActiveContext)
	}

	workspaces, err := service.ListWorkspaces(context.Background(), transport.IdentityWorkspacesRequest{IncludeInactive: true})
	if err != nil {
		t.Fatalf("list workspaces: %v", err)
	}
	if workspaces.ActiveContext.SelectionVersion != 0 {
		t.Fatalf("expected list-workspaces passive read to keep selection version 0, got %+v", workspaces.ActiveContext)
	}

	secondContext, err := service.GetActiveContext(context.Background(), transport.IdentityActiveContextRequest{})
	if err != nil {
		t.Fatalf("second get active context: %v", err)
	}
	if secondContext.ActiveContext.SelectionVersion != 0 {
		t.Fatalf("expected second passive read to keep selection version 0, got %+v", secondContext.ActiveContext)
	}
}

func TestIdentityDirectoryFallbackDoesNotOverwriteExplicitSelection(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedIdentityDirectoryFixtures(t, container.DB)

	service, err := NewIdentityDirectoryService(container)
	if err != nil {
		t.Fatalf("new identity directory service: %v", err)
	}

	selected, err := service.SelectWorkspace(context.Background(), transport.IdentityWorkspaceSelectRequest{
		WorkspaceID:      "ws-beta",
		PrincipalActorID: "actor.beta",
		Source:           "app",
	})
	if err != nil {
		t.Fatalf("select ws-beta: %v", err)
	}
	if selected.ActiveContext.SelectionVersion <= 0 {
		t.Fatalf("expected positive selection version after explicit select, got %+v", selected.ActiveContext)
	}

	if _, err := container.DB.Exec(`DELETE FROM workspaces WHERE id = 'ws-beta'`); err != nil {
		t.Fatalf("delete ws-beta workspace: %v", err)
	}

	fallback, err := service.GetActiveContext(context.Background(), transport.IdentityActiveContextRequest{})
	if err != nil {
		t.Fatalf("get active context after deleting selected workspace: %v", err)
	}
	if fallback.ActiveContext.WorkspaceID != "ws-alpha" || fallback.ActiveContext.WorkspaceSource != identitySourceDerived {
		t.Fatalf("expected derived fallback workspace ws-alpha while selected workspace is unavailable, got %+v", fallback.ActiveContext)
	}
	if fallback.ActiveContext.MutationReason != identityMutationReasonDerived {
		t.Fatalf("expected derived mutation reason during fallback, got %+v", fallback.ActiveContext)
	}
	if fallback.ActiveContext.SelectionVersion != selected.ActiveContext.SelectionVersion {
		t.Fatalf("expected fallback read to preserve selection version %d, got %d", selected.ActiveContext.SelectionVersion, fallback.ActiveContext.SelectionVersion)
	}

	if _, err := container.DB.Exec(`
		INSERT INTO workspaces(id, name, status, created_at, updated_at)
		VALUES ('ws-beta', 'Workspace Beta', 'ACTIVE', '2026-02-25T00:00:00Z', '2026-02-25T00:00:00Z')
	`); err != nil {
		t.Fatalf("reinsert ws-beta workspace: %v", err)
	}

	restored, err := service.GetActiveContext(context.Background(), transport.IdentityActiveContextRequest{})
	if err != nil {
		t.Fatalf("get active context after restoring selected workspace: %v", err)
	}
	if restored.ActiveContext.WorkspaceID != "ws-beta" || restored.ActiveContext.PrincipalActorID != "actor.beta" {
		t.Fatalf("expected explicit selected context to recover after workspace restore, got %+v", restored.ActiveContext)
	}
	if restored.ActiveContext.WorkspaceSource != identitySourceSelected || restored.ActiveContext.PrincipalSource != identitySourceSelected {
		t.Fatalf("expected restored selected context sources, got %+v", restored.ActiveContext)
	}
	if restored.ActiveContext.MutationSource != "app" || restored.ActiveContext.MutationReason != identityMutationReasonExplicitSelect {
		t.Fatalf("expected restored context mutation metadata from explicit select, got %+v", restored.ActiveContext)
	}
	if restored.ActiveContext.SelectionVersion != selected.ActiveContext.SelectionVersion {
		t.Fatalf("expected restored context to keep selection version %d, got %d", selected.ActiveContext.SelectionVersion, restored.ActiveContext.SelectionVersion)
	}
}

func TestIdentityDirectoryListDevicesAndSessionsWithHealthMetadata(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedIdentityDirectoryFixtures(t, container.DB)
	seedIdentityDeviceSessionFixtures(t, container.DB)

	service, err := NewIdentityDirectoryService(container)
	if err != nil {
		t.Fatalf("new identity directory service: %v", err)
	}

	devicesPage1, err := service.ListDevices(context.Background(), transport.IdentityDeviceListRequest{
		WorkspaceID: "ws-alpha",
		Limit:       1,
	})
	if err != nil {
		t.Fatalf("list devices page 1: %v", err)
	}
	if len(devicesPage1.Items) != 1 || !devicesPage1.HasMore {
		t.Fatalf("expected paged devices response with has_more, got %+v", devicesPage1)
	}
	if devicesPage1.NextCursorCreatedAt == "" || devicesPage1.NextCursorID == "" {
		t.Fatalf("expected device cursor metadata, got %+v", devicesPage1)
	}
	if devicesPage1.Items[0].DeviceID != "device-beta" || devicesPage1.Items[0].SessionActiveCount != 1 {
		t.Fatalf("unexpected first device row: %+v", devicesPage1.Items[0])
	}

	devicesPage2, err := service.ListDevices(context.Background(), transport.IdentityDeviceListRequest{
		WorkspaceID:     "ws-alpha",
		CursorCreatedAt: devicesPage1.NextCursorCreatedAt,
		CursorID:        devicesPage1.NextCursorID,
		Limit:           5,
	})
	if err != nil {
		t.Fatalf("list devices page 2: %v", err)
	}
	if len(devicesPage2.Items) == 0 {
		t.Fatalf("expected second devices page to contain rows")
	}
	var deviceAlpha *transport.IdentityDeviceRecord
	for idx := range devicesPage2.Items {
		if devicesPage2.Items[idx].DeviceID == "device-alpha" {
			deviceAlpha = &devicesPage2.Items[idx]
			break
		}
	}
	if deviceAlpha == nil {
		t.Fatalf("expected device-alpha in paged response: %+v", devicesPage2.Items)
	}
	if deviceAlpha.SessionTotal != 3 || deviceAlpha.SessionActiveCount != 1 || deviceAlpha.SessionExpiredCount != 1 || deviceAlpha.SessionRevokedCount != 1 {
		t.Fatalf("unexpected session-health aggregate counts for device-alpha: %+v", deviceAlpha)
	}

	sessionsPage1, err := service.ListSessions(context.Background(), transport.IdentitySessionListRequest{
		WorkspaceID: "ws-alpha",
		Limit:       1,
	})
	if err != nil {
		t.Fatalf("list sessions page 1: %v", err)
	}
	if len(sessionsPage1.Items) != 1 || !sessionsPage1.HasMore {
		t.Fatalf("expected paged sessions response with has_more, got %+v", sessionsPage1)
	}
	if sessionsPage1.NextCursorStartedAt == "" || sessionsPage1.NextCursorID == "" {
		t.Fatalf("expected session cursor metadata, got %+v", sessionsPage1)
	}
	if sessionsPage1.Items[0].SessionID != "session-beta-active" || sessionsPage1.Items[0].SessionHealth != identitySessionHealthActive {
		t.Fatalf("unexpected first session row: %+v", sessionsPage1.Items[0])
	}

	revokedOnly, err := service.ListSessions(context.Background(), transport.IdentitySessionListRequest{
		WorkspaceID:   "ws-alpha",
		SessionHealth: "revoked",
		Limit:         10,
	})
	if err != nil {
		t.Fatalf("list revoked sessions: %v", err)
	}
	if len(revokedOnly.Items) != 1 || revokedOnly.Items[0].SessionID != "session-alpha-revoked" || revokedOnly.Items[0].SessionHealth != identitySessionHealthRevoked {
		t.Fatalf("unexpected revoked session payload: %+v", revokedOnly)
	}
	if revokedOnly.Items[0].DeviceLastSeenAt == "" {
		t.Fatalf("expected device_last_seen_at metadata on revoked session row: %+v", revokedOnly.Items[0])
	}
}

func TestIdentityDirectoryRevokeSessionIsIdempotent(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedIdentityDirectoryFixtures(t, container.DB)
	seedIdentityDeviceSessionFixtures(t, container.DB)

	service, err := NewIdentityDirectoryService(container)
	if err != nil {
		t.Fatalf("new identity directory service: %v", err)
	}

	first, err := service.RevokeSession(context.Background(), transport.IdentitySessionRevokeRequest{
		WorkspaceID: "ws-alpha",
		SessionID:   "session-alpha-active",
	})
	if err != nil {
		t.Fatalf("first revoke session: %v", err)
	}
	if first.Idempotent || first.SessionHealth != identitySessionHealthRevoked || first.RevokedAt == "" {
		t.Fatalf("unexpected first revoke response: %+v", first)
	}

	second, err := service.RevokeSession(context.Background(), transport.IdentitySessionRevokeRequest{
		WorkspaceID: "ws-alpha",
		SessionID:   "session-alpha-active",
	})
	if err != nil {
		t.Fatalf("second revoke session: %v", err)
	}
	if !second.Idempotent || second.RevokedAt != first.RevokedAt || second.SessionHealth != identitySessionHealthRevoked {
		t.Fatalf("unexpected second revoke response: %+v", second)
	}

	revokedOnly, err := service.ListSessions(context.Background(), transport.IdentitySessionListRequest{
		WorkspaceID:   "ws-alpha",
		SessionHealth: "revoked",
		Limit:         20,
	})
	if err != nil {
		t.Fatalf("list revoked sessions after revoke: %v", err)
	}
	found := false
	for _, item := range revokedOnly.Items {
		if item.SessionID == "session-alpha-active" {
			found = true
			if item.SessionHealth != identitySessionHealthRevoked {
				t.Fatalf("expected revoked session health for session-alpha-active, got %+v", item)
			}
			break
		}
	}
	if !found {
		t.Fatalf("expected revoked sessions to include session-alpha-active: %+v", revokedOnly.Items)
	}
}

func TestIdentityDirectorySessionValidationErrors(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedIdentityDirectoryFixtures(t, container.DB)
	seedIdentityDeviceSessionFixtures(t, container.DB)

	service, err := NewIdentityDirectoryService(container)
	if err != nil {
		t.Fatalf("new identity directory service: %v", err)
	}

	_, err = service.ListSessions(context.Background(), transport.IdentitySessionListRequest{
		WorkspaceID:   "ws-alpha",
		SessionHealth: "unknown",
	})
	if err == nil || err.Error() == "" {
		t.Fatalf("expected invalid session_health validation error")
	}

	_, err = service.ListDevices(context.Background(), transport.IdentityDeviceListRequest{
		WorkspaceID:     "ws-alpha",
		CursorCreatedAt: "2026-02-25T00:00:00Z",
	})
	if err == nil || err.Error() == "" {
		t.Fatalf("expected cursor validation error for list devices")
	}

	_, err = service.RevokeSession(context.Background(), transport.IdentitySessionRevokeRequest{
		WorkspaceID: "ws-alpha",
	})
	if err == nil || err.Error() == "" {
		t.Fatalf("expected session_id required validation error")
	}
}

func TestIdentityDirectoryBootstrapProvisioningIsIdempotent(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	service, err := NewIdentityDirectoryService(container)
	if err != nil {
		t.Fatalf("new identity directory service: %v", err)
	}

	request := transport.IdentityBootstrapRequest{
		WorkspaceID:          "ws-bootstrap",
		WorkspaceName:        "Bootstrap Workspace",
		PrincipalActorID:     "actor.bootstrap",
		PrincipalDisplayName: "Bootstrap User",
		PrincipalActorType:   "human",
		PrincipalStatus:      "ACTIVE",
		Handle: &transport.IdentityBootstrapHandle{
			Channel:     "message",
			HandleValue: "+15550001111",
			IsPrimary:   true,
		},
		Source: "cli",
	}

	first, err := service.Bootstrap(context.Background(), request)
	if err != nil {
		t.Fatalf("first identity bootstrap: %v", err)
	}
	if !first.WorkspaceCreated || !first.PrincipalCreated || !first.PrincipalLinked || !first.HandleCreated {
		t.Fatalf("expected bootstrap create flags on first call, got %+v", first)
	}
	if first.Idempotent {
		t.Fatalf("expected first bootstrap call to be non-idempotent")
	}
	if first.ActiveContext.WorkspaceID != "ws-bootstrap" || first.ActiveContext.PrincipalActorID != "actor.bootstrap" {
		t.Fatalf("expected selected active context from bootstrap response, got %+v", first.ActiveContext)
	}
	if strings.TrimSpace(first.AuditLogID) == "" {
		t.Fatalf("expected bootstrap response to include audit_log_id")
	}

	second, err := service.Bootstrap(context.Background(), request)
	if err != nil {
		t.Fatalf("second identity bootstrap: %v", err)
	}
	if second.WorkspaceCreated || second.PrincipalCreated || second.PrincipalLinked || second.HandleCreated || second.HandleUpdated {
		t.Fatalf("expected second bootstrap call to avoid duplicate creates, got %+v", second)
	}
	if !second.Idempotent {
		t.Fatalf("expected second bootstrap call to be idempotent")
	}

	assertCount := func(query string, expected int) {
		t.Helper()
		var count int
		if err := container.DB.QueryRow(query).Scan(&count); err != nil {
			t.Fatalf("query count failed: %v\nquery: %s", err, query)
		}
		if count != expected {
			t.Fatalf("unexpected count=%d expected=%d for query: %s", count, expected, query)
		}
	}
	assertCount(`SELECT COUNT(*) FROM workspaces WHERE id = 'ws-bootstrap'`, 1)
	assertCount(`SELECT COUNT(*) FROM actors WHERE id = 'actor.bootstrap' AND workspace_id = 'ws-bootstrap'`, 1)
	assertCount(`SELECT COUNT(*) FROM workspace_principals WHERE workspace_id = 'ws-bootstrap' AND actor_id = 'actor.bootstrap'`, 1)
	assertCount(`SELECT COUNT(*) FROM actor_handles WHERE workspace_id = 'ws-bootstrap' AND actor_id = 'actor.bootstrap' AND channel = 'message' AND handle_value = '+15550001111'`, 1)
	assertCount(`SELECT COUNT(*) FROM audit_log_entries WHERE workspace_id = 'ws-bootstrap' AND event_type = 'identity_bootstrap_upsert'`, 2)
}

func TestIdentityDirectoryBootstrapRejectsHandleOwnershipConflict(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	service, err := NewIdentityDirectoryService(container)
	if err != nil {
		t.Fatalf("new identity directory service: %v", err)
	}

	initial := transport.IdentityBootstrapRequest{
		WorkspaceID:      "ws-bootstrap-conflict",
		PrincipalActorID: "actor.one",
		PrincipalStatus:  "ACTIVE",
		Handle: &transport.IdentityBootstrapHandle{
			Channel:     "message",
			HandleValue: "+15550002222",
			IsPrimary:   true,
		},
	}
	if _, err := service.Bootstrap(context.Background(), initial); err != nil {
		t.Fatalf("seed first bootstrap principal: %v", err)
	}

	_, err = service.Bootstrap(context.Background(), transport.IdentityBootstrapRequest{
		WorkspaceID:      "ws-bootstrap-conflict",
		PrincipalActorID: "actor.two",
		PrincipalStatus:  "ACTIVE",
		Handle: &transport.IdentityBootstrapHandle{
			Channel:     "message",
			HandleValue: "+15550002222",
			IsPrimary:   true,
		},
	})
	if err == nil {
		t.Fatalf("expected handle ownership conflict for second bootstrap principal")
	}
	if !strings.Contains(err.Error(), "already assigned to actor") {
		t.Fatalf("expected ownership conflict error, got %v", err)
	}

	var actorTwoCount int
	if err := container.DB.QueryRow(`SELECT COUNT(*) FROM actors WHERE id = 'actor.two'`).Scan(&actorTwoCount); err != nil {
		t.Fatalf("count actor.two rows: %v", err)
	}
	if actorTwoCount != 0 {
		t.Fatalf("expected transaction rollback to avoid creating actor.two on conflict, got %d rows", actorTwoCount)
	}
}

func seedIdentityDirectoryFixtures(t *testing.T, db *sql.DB) {
	t.Helper()
	statements := []string{
		`INSERT INTO workspaces(id, name, status, created_at, updated_at) VALUES ('ws-alpha', 'Workspace Alpha', 'ACTIVE', '2026-02-25T00:00:00Z', '2026-02-25T00:00:00Z')`,
		`INSERT INTO workspaces(id, name, status, created_at, updated_at) VALUES ('ws-beta', 'Workspace Beta', 'ACTIVE', '2026-02-25T00:00:00Z', '2026-02-25T00:00:00Z')`,
		`INSERT INTO actors(id, workspace_id, actor_type, display_name, status, created_at, updated_at) VALUES ('actor.alice', 'ws-alpha', 'human', 'Alice', 'ACTIVE', '2026-02-25T00:00:00Z', '2026-02-25T00:00:00Z')`,
		`INSERT INTO actors(id, workspace_id, actor_type, display_name, status, created_at, updated_at) VALUES ('actor.bob', 'ws-alpha', 'human', 'Bob', 'ACTIVE', '2026-02-25T00:00:00Z', '2026-02-25T00:00:00Z')`,
		`INSERT INTO actors(id, workspace_id, actor_type, display_name, status, created_at, updated_at) VALUES ('actor.beta', 'ws-beta', 'human', 'Beta User', 'ACTIVE', '2026-02-25T00:00:00Z', '2026-02-25T00:00:00Z')`,
		`INSERT INTO workspace_principals(id, workspace_id, actor_id, status, created_at, updated_at) VALUES ('wp-alice', 'ws-alpha', 'actor.alice', 'ACTIVE', '2026-02-25T00:00:00Z', '2026-02-25T00:00:00Z')`,
		`INSERT INTO workspace_principals(id, workspace_id, actor_id, status, created_at, updated_at) VALUES ('wp-bob', 'ws-alpha', 'actor.bob', 'ACTIVE', '2026-02-25T00:00:00Z', '2026-02-25T00:00:00Z')`,
		`INSERT INTO workspace_principals(id, workspace_id, actor_id, status, created_at, updated_at) VALUES ('wp-beta', 'ws-beta', 'actor.beta', 'ACTIVE', '2026-02-25T00:00:00Z', '2026-02-25T00:00:00Z')`,
		`INSERT INTO actor_handles(id, workspace_id, actor_id, channel, handle_value, is_primary, created_at, updated_at) VALUES ('ah-alice-imessage', 'ws-alpha', 'actor.alice', 'imessage', '+15550000001', 1, '2026-02-25T00:00:00Z', '2026-02-25T00:00:00Z')`,
		`INSERT INTO actor_handles(id, workspace_id, actor_id, channel, handle_value, is_primary, created_at, updated_at) VALUES ('ah-alice-email', 'ws-alpha', 'actor.alice', 'mail', 'alice@example.com', 0, '2026-02-25T00:00:00Z', '2026-02-25T00:00:00Z')`,
		`INSERT INTO actor_handles(id, workspace_id, actor_id, channel, handle_value, is_primary, created_at, updated_at) VALUES ('ah-bob-imessage', 'ws-alpha', 'actor.bob', 'imessage', '+15550000002', 1, '2026-02-25T00:00:00Z', '2026-02-25T00:00:00Z')`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("seed identity fixtures: %v\nstatement: %s", err, statement)
		}
	}
}

func seedIdentityCommVisibilityFixtures(t *testing.T, db *sql.DB) {
	t.Helper()
	statements := []string{
		`INSERT INTO comm_threads(id, workspace_id, channel, external_ref, title, created_at, updated_at)
		 VALUES ('thread-alpha-imessage-1', 'ws-alpha', 'imessage', 'alpha-chat-1', 'Alpha iMessage Thread', '2026-02-25T00:00:00Z', '2026-02-25T00:00:04Z')`,
		`INSERT INTO comm_threads(id, workspace_id, channel, external_ref, title, created_at, updated_at)
		 VALUES ('thread-alpha-mail-1', 'ws-alpha', 'mail', 'alpha-mail-1', 'Alpha Mail Thread', '2026-02-25T00:00:00Z', '2026-02-25T00:00:02Z')`,
		`INSERT INTO comm_threads(id, workspace_id, channel, external_ref, title, created_at, updated_at)
		 VALUES ('thread-beta-mail-1', 'ws-beta', 'mail', 'beta-mail-1', 'Beta Mail Thread', '2026-02-25T00:00:00Z', '2026-02-25T00:00:03Z')`,
		`INSERT INTO comm_events(id, workspace_id, thread_id, event_type, direction, assistant_emitted, occurred_at, body_text, created_at)
		 VALUES ('event-alpha-imessage-1', 'ws-alpha', 'thread-alpha-imessage-1', 'MESSAGE', 'INBOUND', 0, '2026-02-25T00:00:03Z', 'hello from alpha imessage', '2026-02-25T00:00:03Z')`,
		`INSERT INTO comm_events(id, workspace_id, thread_id, event_type, direction, assistant_emitted, occurred_at, body_text, created_at)
		 VALUES ('event-alpha-imessage-2', 'ws-alpha', 'thread-alpha-imessage-1', 'MESSAGE', 'OUTBOUND', 1, '2026-02-25T00:00:04Z', 'assistant reply for alpha imessage', '2026-02-25T00:00:04Z')`,
		`INSERT INTO comm_events(id, workspace_id, thread_id, event_type, direction, assistant_emitted, occurred_at, body_text, created_at)
		 VALUES ('event-alpha-mail-1', 'ws-alpha', 'thread-alpha-mail-1', 'MESSAGE', 'INBOUND', 0, '2026-02-25T00:00:02Z', 'alpha mail token', '2026-02-25T00:00:02Z')`,
		`INSERT INTO comm_events(id, workspace_id, thread_id, event_type, direction, assistant_emitted, occurred_at, body_text, created_at)
		 VALUES ('event-beta-mail-1', 'ws-beta', 'thread-beta-mail-1', 'MESSAGE', 'INBOUND', 0, '2026-02-25T00:00:03Z', 'beta mail token', '2026-02-25T00:00:03Z')`,
		`INSERT INTO comm_event_addresses(id, event_id, address_role, address_value, display_name, position, created_at)
		 VALUES ('addr-alpha-imessage-1', 'event-alpha-imessage-1', 'FROM', '+15550000001', 'Alice', 0, '2026-02-25T00:00:03Z')`,
		`INSERT INTO comm_event_addresses(id, event_id, address_role, address_value, display_name, position, created_at)
		 VALUES ('addr-alpha-imessage-2', 'event-alpha-imessage-1', 'TO', '+15550009999', 'Agent', 1, '2026-02-25T00:00:03Z')`,
		`INSERT INTO comm_event_addresses(id, event_id, address_role, address_value, display_name, position, created_at)
		 VALUES ('addr-alpha-imessage-3', 'event-alpha-imessage-2', 'FROM', '+15550009999', 'Agent', 0, '2026-02-25T00:00:04Z')`,
		`INSERT INTO comm_event_addresses(id, event_id, address_role, address_value, display_name, position, created_at)
		 VALUES ('addr-alpha-imessage-4', 'event-alpha-imessage-2', 'TO', '+15550000001', 'Alice', 1, '2026-02-25T00:00:04Z')`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("seed identity comm visibility fixtures: %v\nstatement: %s", err, statement)
		}
	}
}

func seedIdentityDeviceSessionFixtures(t *testing.T, db *sql.DB) {
	t.Helper()
	statements := []string{
		`INSERT INTO users(id, email, display_name, status, created_at, updated_at) VALUES ('user.alpha', 'alpha@example.com', 'User Alpha', 'ACTIVE', '2026-02-25T00:00:00Z', '2026-02-25T00:00:00Z')`,
		`INSERT INTO users(id, email, display_name, status, created_at, updated_at) VALUES ('user.beta', 'beta@example.com', 'User Beta', 'ACTIVE', '2026-02-25T00:00:00Z', '2026-02-25T00:00:00Z')`,
		`INSERT INTO user_devices(id, workspace_id, user_id, device_type, platform, label, last_seen_at, created_at) VALUES ('device-alpha', 'ws-alpha', 'user.alpha', 'phone', 'ios', 'Alpha iPhone', '2026-02-25T00:10:00Z', '2026-02-25T00:00:00Z')`,
		`INSERT INTO user_devices(id, workspace_id, user_id, device_type, platform, label, last_seen_at, created_at) VALUES ('device-beta', 'ws-alpha', 'user.beta', 'desktop', 'macos', 'Beta Mac', '2026-02-25T00:20:00Z', '2026-02-25T00:05:00Z')`,
		`INSERT INTO device_sessions(id, workspace_id, device_id, session_token_hash, started_at, expires_at, revoked_at) VALUES ('session-alpha-active', 'ws-alpha', 'device-alpha', 'hash-active', '2026-02-25T00:01:00Z', '2099-01-01T00:00:00Z', NULL)`,
		`INSERT INTO device_sessions(id, workspace_id, device_id, session_token_hash, started_at, expires_at, revoked_at) VALUES ('session-alpha-expired', 'ws-alpha', 'device-alpha', 'hash-expired', '2026-02-24T00:01:00Z', '2026-02-24T01:00:00Z', NULL)`,
		`INSERT INTO device_sessions(id, workspace_id, device_id, session_token_hash, started_at, expires_at, revoked_at) VALUES ('session-alpha-revoked', 'ws-alpha', 'device-alpha', 'hash-revoked', '2026-02-23T00:01:00Z', '2099-01-01T00:00:00Z', '2026-02-23T02:00:00Z')`,
		`INSERT INTO device_sessions(id, workspace_id, device_id, session_token_hash, started_at, expires_at, revoked_at) VALUES ('session-beta-active', 'ws-alpha', 'device-beta', 'hash-beta-active', '2026-02-25T00:06:00Z', '2099-01-01T00:00:00Z', NULL)`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("seed identity device/session fixtures: %v\nstatement: %s", err, statement)
		}
	}
}

func findWorkspaceRecord(records []transport.IdentityWorkspaceRecord, workspaceID string) *transport.IdentityWorkspaceRecord {
	for idx := range records {
		if records[idx].WorkspaceID == workspaceID {
			return &records[idx]
		}
	}
	return nil
}

func commThreadIDs(items []transport.CommThreadListItem) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ThreadID)
	}
	return ids
}

func commEventIDs(items []transport.CommEventTimelineItem) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.EventID)
	}
	return ids
}
