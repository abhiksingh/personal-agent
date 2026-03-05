import XCTest
@testable import PersonalAgentUI

@MainActor
final class AppShellStateIdentityTests: XCTestCase {
    private let workspaceDefaultsKey = "personalagent.ui.workspace_id"

    func testIdentityWorkspaceOptionsIncludesCurrentWorkspace() {
        let state = AppShellState()

        XCTAssertTrue(state.identityWorkspaceOptions.contains(state.workspaceLabel))
    }

    func testRefreshIdentityDirectoryWithoutTokenSetsDeterministicStatus() async {
        let state = AppShellState()
        state.clearLocalDevToken()

        state.refreshIdentityDirectory()
        try? await Task.sleep(for: .milliseconds(50))

        XCTAssertEqual(state.identityStatusMessage, "Set Assistant Access Token to query principal context.")
        XCTAssertEqual(state.principalStatusMessage, "Set Assistant Access Token to query principal context.")
        XCTAssertTrue(state.identityWorkspaceItems.isEmpty)
        XCTAssertTrue(state.identityPrincipalItems.isEmpty)
    }

    func testSelectIdentityWorkspaceWithoutTokenSetsDeterministicStatus() async {
        let state = AppShellState()
        state.clearLocalDevToken()

        state.selectIdentityWorkspace("ws-alt")
        try? await Task.sleep(for: .milliseconds(50))

        XCTAssertEqual(state.identityStatusMessage, "Set Assistant Access Token to switch workspace context.")
        XCTAssertEqual(state.principalStatusMessage, "Set Assistant Access Token to switch workspace context.")
    }

    func testRefreshIdentityDeviceInventoryWithoutTokenSetsDeterministicStatus() async {
        let state = AppShellState()
        state.identityDeviceItems = [
            IdentityDeviceItem(
                id: "device-1",
                workspaceID: "ws1",
                userID: "user.primary",
                deviceType: "desktop",
                platform: "macos",
                label: "Mac",
                lastSeenAtLabel: "n/a",
                createdAtLabel: "n/a",
                sessionTotal: 1,
                sessionActiveCount: 1,
                sessionExpiredCount: 0,
                sessionRevokedCount: 0,
                sessionLatestStartedAtLabel: nil,
                sortTimestamp: .distantPast
            )
        ]
        state.clearLocalDevToken()

        state.refreshIdentityDeviceInventory()
        try? await Task.sleep(for: .milliseconds(50))

        XCTAssertEqual(state.identityDeviceInventoryStatusMessage, "Set Assistant Access Token to query identity devices.")
        XCTAssertTrue(state.identityDeviceItems.isEmpty)
        XCTAssertFalse(state.isIdentityDeviceInventoryLoading)
    }

    func testRefreshIdentitySessionInventoryWithoutTokenSetsDeterministicStatus() async {
        let state = AppShellState()
        state.identitySessionItems = [
            IdentitySessionItem(
                id: "session-1",
                workspaceID: "ws1",
                deviceID: "device-1",
                userID: "user.primary",
                deviceType: "desktop",
                platform: "macos",
                deviceLabel: "Mac",
                deviceLastSeenAtLabel: nil,
                startedAtLabel: "n/a",
                expiresAtLabel: "n/a",
                revokedAtLabel: nil,
                sessionHealth: "active",
                sortTimestamp: .distantPast
            )
        ]
        state.clearLocalDevToken()

        state.refreshIdentitySessionInventory()
        try? await Task.sleep(for: .milliseconds(50))

        XCTAssertEqual(state.identitySessionInventoryStatusMessage, "Set Assistant Access Token to query identity sessions.")
        XCTAssertTrue(state.identitySessionItems.isEmpty)
        XCTAssertFalse(state.isIdentitySessionInventoryLoading)
    }

    func testRevokeIdentitySessionWithoutTokenShowsDeterministicStatus() async {
        let state = AppShellState()
        state.clearLocalDevToken()

        state.revokeIdentitySession(sessionID: "session-123")
        try? await Task.sleep(for: .milliseconds(50))

        XCTAssertEqual(
            state.identitySessionActionStatusByID["session-123"],
            "Set Assistant Access Token before revoking identity sessions."
        )
        XCTAssertEqual(
            state.identitySessionInventoryStatusMessage,
            "Set Assistant Access Token before revoking identity sessions."
        )
    }

    func testActingAsOptionsIncludeDefaultAndExplicitCandidate() {
        let state = AppShellState()
        state.principalOptions = ["actor.requester"]
        state.selectedPrincipal = "actor.requester"

        let options = state.actingAsOptions(including: "actor.delegate")

        XCTAssertTrue(options.contains("default"))
        XCTAssertTrue(options.contains("actor.requester"))
        XCTAssertTrue(options.contains("actor.delegate"))
    }

    func testPrincipalIdentityDisplayValuePrefersDisplayNameAndRetainsRawID() {
        let state = AppShellState()
        state.identityPrincipalItems = [makePrincipal(id: "actor.requester", displayName: "Requester")]

        let value = state.principalIdentityDisplayValue(for: "actor.requester")

        XCTAssertEqual(value.displayText, "Requester")
        XCTAssertEqual(value.rawID, "actor.requester")
    }

    func testPrincipalIdentityDisplayValueUsesDeterministicFallbackForUnknownPrincipal() {
        let state = AppShellState()
        let value = state.principalIdentityDisplayValue(for: "actor.missing")

        XCTAssertEqual(value.displayText, "Unrecognized Principal")
        XCTAssertEqual(value.rawID, "actor.missing")
    }

    func testWorkspaceIdentityDisplayValueUsesNameWhenAvailable() {
        let state = AppShellState()
        state.identityWorkspaceItems = [
            IdentityWorkspaceItem(
                id: "ws1",
                name: "Default Context",
                status: "ACTIVE",
                principalCount: 1,
                actorCount: 1,
                handleCount: 0,
                updatedAtLabel: "n/a",
                isActive: true
            )
        ]

        let value = state.workspaceIdentityDisplayValue(for: "ws1")

        XCTAssertEqual(value.displayText, "Default Context")
        XCTAssertEqual(value.rawID, "ws1")
    }

    func testWorkspaceIdentityDisplayValueUsesDeterministicFallbackForUnknownWorkspace() {
        let state = AppShellState()
        let value = state.workspaceIdentityDisplayValue(for: "ws-unknown")

        XCTAssertEqual(value.displayText, "Unrecognized Workspace")
        XCTAssertEqual(value.rawID, "ws-unknown")
    }

    func testActingAsValidationAllowsKnownIdentityPrincipal() {
        let state = AppShellState()
        state.identityPrincipalItems = [makePrincipal(id: "actor.requester", displayName: "Requester")]

        XCTAssertNil(state.actingAsValidationMessage(for: "default"))
        XCTAssertNil(state.actingAsValidationMessage(for: "actor.requester"))
    }

    func testActingAsValidationRejectsUnknownIdentityPrincipalWhenDirectoryLoaded() {
        let state = AppShellState()
        state.identityPrincipalItems = [makePrincipal(id: "actor.requester", displayName: "Requester")]

        XCTAssertEqual(
            state.actingAsValidationMessage(for: "actor.unknown"),
            "Selected acting-as principal `actor.unknown` is not in the active workspace directory. Refresh Identity Directory in Configuration."
        )
    }

    func testPassiveWorkspaceContextUpdatesDoNotOverrideIdentitySyncedWorkspace() {
        let state = AppShellState()

        state._test_applyWorkspaceContextIdentitySync("ws-identity")
        XCTAssertEqual(state.workspaceID, "ws-identity")

        state._test_applyWorkspaceContextPassive("ws-passive")
        XCTAssertEqual(state.workspaceID, "ws-identity")
    }

    func testExplicitWorkspaceSelectionPinsContextUntilExplicitlyChangedAgain() {
        let state = AppShellState()

        state._test_applyWorkspaceContextIdentitySync("ws-initial")
        XCTAssertEqual(state.workspaceID, "ws-initial")

        state._test_applyWorkspaceContextExplicitSelection("ws-selected")
        XCTAssertEqual(state.workspaceID, "ws-selected")

        state._test_applyWorkspaceContextPassive("ws-passive")
        XCTAssertEqual(state.workspaceID, "ws-selected")

        state._test_applyWorkspaceContextIdentitySync("ws-other")
        XCTAssertEqual(state.workspaceID, "ws-selected")

        state._test_applyWorkspaceContextExplicitSelection("ws-final")
        XCTAssertEqual(state.workspaceID, "ws-final")
    }

    func testExplicitWorkspaceSelectionPersistsWorkspaceForNextSession() {
        let defaults = appShellStateTestUserDefaults()
        let priorWorkspace = defaults.object(forKey: workspaceDefaultsKey)
        defer {
            if let priorWorkspace {
                defaults.set(priorWorkspace, forKey: workspaceDefaultsKey)
            } else {
                defaults.removeObject(forKey: workspaceDefaultsKey)
            }
        }

        defaults.removeObject(forKey: workspaceDefaultsKey)

        let state = AppShellState()
        state._test_applyWorkspaceContextExplicitSelectionPersisted("ws-persisted")

        XCTAssertEqual(state._test_storedWorkspaceSelection(), "ws-persisted")

        let restored = AppShellState()
        XCTAssertEqual(restored.workspaceID, "ws-persisted")
    }

    func testStoredExplicitDefaultWorkspaceSelectionIsPreserved() {
        let defaults = appShellStateTestUserDefaults()
        let priorWorkspace = defaults.object(forKey: workspaceDefaultsKey)
        defer {
            if let priorWorkspace {
                defaults.set(priorWorkspace, forKey: workspaceDefaultsKey)
            } else {
                defaults.removeObject(forKey: workspaceDefaultsKey)
            }
        }

        defaults.set("default", forKey: workspaceDefaultsKey)

        let state = AppShellState()
        XCTAssertEqual(state.workspaceID, "default")
        XCTAssertEqual(defaults.string(forKey: workspaceDefaultsKey), "default")

        let restored = AppShellState()
        XCTAssertEqual(restored.workspaceID, "default")
    }

    func testIdentityWorkspaceDirectoryDeduplicatesDuplicateWorkspaceIDs() throws {
        let state = AppShellState()
        let workspaceResponse = try decodeJSON(
            DaemonIdentityWorkspacesResponse.self,
            from: """
            {
              "workspaces": [
                {
                  "workspace_id": "ws1",
                  "name": "ws1 older",
                  "status": "ACTIVE",
                  "principal_count": 0,
                  "actor_count": 0,
                  "handle_count": 0,
                  "updated_at": "2026-02-27T11:52:00Z",
                  "is_active": false
                },
                {
                  "workspace_id": "ws1",
                  "name": "ws1 latest",
                  "status": "ACTIVE",
                  "principal_count": 0,
                  "actor_count": 0,
                  "handle_count": 0,
                  "updated_at": "2026-02-27T11:53:00Z",
                  "is_active": true
                }
              ]
            }
            """
        )
        let principalsResponse = try decodeJSON(
            DaemonIdentityPrincipalsResponse.self,
            from: """
            {
              "workspace_id": "ws1",
              "principals": []
            }
            """
        )

        state._test_mapIdentityDirectoryRecords(
            workspaceResponse: workspaceResponse,
            principalsResponse: principalsResponse
        )

        XCTAssertEqual(state.identityWorkspaceItems.count, 1)
        XCTAssertEqual(state.identityWorkspaceItems.first?.id, "ws1")
        XCTAssertEqual(state.identityWorkspaceItems.first?.name, "ws1 latest")
        XCTAssertTrue(state.identityWorkspaceItems.first?.isActive ?? false)
    }

    private func makePrincipal(id: String, displayName: String) -> IdentityPrincipalItem {
        IdentityPrincipalItem(
            id: id,
            displayName: displayName,
            actorType: "human",
            actorStatus: "ACTIVE",
            principalStatus: "ACTIVE",
            isActive: true,
            handles: []
        )
    }

    private func decodeJSON<T: Decodable>(_ type: T.Type, from json: String) throws -> T {
        try JSONDecoder().decode(T.self, from: Data(json.utf8))
    }
}
