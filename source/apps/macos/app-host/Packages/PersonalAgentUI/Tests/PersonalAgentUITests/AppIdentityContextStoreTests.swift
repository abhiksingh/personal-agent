import XCTest
@testable import PersonalAgentUI

@MainActor
final class AppIdentityContextStoreTests: XCTestCase {
    func testConfigureInitialWorkspaceContextPrefersEnvironmentWorkspace() {
        let store = AppIdentityContextStore()
        var persistedSelections: [String] = []

        store.configureInitialWorkspaceContext(
            envWorkspaceID: "ws-env",
            storedWorkspaceID: "ws-stored",
            defaultWorkspaceID: "ws-default",
            canonicalWorkspaceID: canonicalWorkspaceID,
            persistWorkspaceSelection: { persistedSelections.append($0) }
        )

        XCTAssertEqual(store.workspaceID, "ws-env")
        XCTAssertTrue(persistedSelections.isEmpty)
    }

    func testConfigureInitialWorkspaceContextUsesStoredWorkspaceWhenEnvironmentMissing() {
        let store = AppIdentityContextStore()
        var persistedSelections: [String] = []

        store.configureInitialWorkspaceContext(
            envWorkspaceID: nil,
            storedWorkspaceID: "ws-stored",
            defaultWorkspaceID: "ws-default",
            canonicalWorkspaceID: canonicalWorkspaceID,
            persistWorkspaceSelection: { persistedSelections.append($0) }
        )

        XCTAssertEqual(store.workspaceID, "ws-stored")
        XCTAssertEqual(persistedSelections, ["ws-stored"])
    }

    func testUpdateWorkspaceContextPassiveResponseDoesNotMutateWorkspace() {
        let store = AppIdentityContextStore()
        store.workspaceID = "ws-initial"
        var densityApplications: [String] = []

        store.updateWorkspaceContext(
            from: "ws-passive",
            intent: .passiveResponse,
            canonicalWorkspaceID: canonicalWorkspaceID,
            applyWorkspaceScopedInformationDensityMode: { densityApplications.append($0) },
            persistWorkspaceSelection: { _ in XCTFail("Should not persist on passive updates") }
        )

        XCTAssertEqual(store.workspaceID, "ws-initial")
        XCTAssertTrue(densityApplications.isEmpty)
    }

    func testUpdateWorkspaceContextIdentitySyncRespectsExplicitPin() {
        let store = AppIdentityContextStore()
        var persistedSelections: [String] = []

        store.updateWorkspaceContext(
            from: "ws-selected",
            intent: .explicitSelection,
            canonicalWorkspaceID: canonicalWorkspaceID,
            applyWorkspaceScopedInformationDensityMode: { _ in },
            persistWorkspaceSelection: { persistedSelections.append($0) }
        )
        store.updateWorkspaceContext(
            from: "ws-directory",
            intent: .identityDirectorySync,
            canonicalWorkspaceID: canonicalWorkspaceID,
            applyWorkspaceScopedInformationDensityMode: { _ in },
            persistWorkspaceSelection: { _ in XCTFail("Identity sync should not persist") }
        )

        XCTAssertEqual(store.workspaceID, "ws-selected")
        XCTAssertEqual(persistedSelections, ["ws-selected"])
    }

    func testPrincipalOptionsForPrincipalSelectionIncludesDefaultAndKnownPrincipals() {
        let store = AppIdentityContextStore()
        store.principalOptions = ["actor.requester"]
        store.selectedPrincipal = "actor.delegate"
        store.identityPrincipalItems = [
            IdentityPrincipalItem(
                id: "actor.identity",
                displayName: "Identity",
                actorType: "user",
                actorStatus: "ACTIVE",
                principalStatus: "ACTIVE",
                isActive: true,
                handles: []
            )
        ]

        let options = store.principalOptionsForPrincipalSelection(including: "actor.extra")

        XCTAssertEqual(options, ["actor.delegate", "actor.extra", "actor.identity", "actor.requester", "default"])
    }

    func testActingAsValidationRejectsUnknownPrincipalWhenDirectoryLoaded() {
        let store = AppIdentityContextStore()
        store.identityPrincipalItems = [
            IdentityPrincipalItem(
                id: "actor.requester",
                displayName: "Requester",
                actorType: "user",
                actorStatus: "ACTIVE",
                principalStatus: "ACTIVE",
                isActive: true,
                handles: []
            )
        ]

        XCTAssertEqual(
            store.actingAsValidationMessage(for: "actor.unknown"),
            "Selected acting-as principal `actor.unknown` is not in the active workspace directory. Refresh Identity Directory in Configuration."
        )
    }

    private func canonicalWorkspaceID(_ raw: String?, _ fallbackToDefault: Bool) -> String? {
        let trimmed = raw?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if trimmed.isEmpty {
            return fallbackToDefault ? "ws1" : nil
        }
        return trimmed == "default" ? "ws1" : trimmed
    }
}
