import XCTest
@testable import PersonalAgentUI

@MainActor
final class ConfigurationDraftStoreTests: XCTestCase {
    func testDelegationActorOptionsAndSeedKeepDraftValid() {
        let store = ConfigurationDraftStore()
        store.delegationFromActorID = "missing-from"
        store.delegationToActorID = "missing-to"

        let options = store.delegationActorOptions(principalOptions: [" default ", "actor.a"])
        XCTAssertTrue(options.contains("default"))
        XCTAssertTrue(options.contains("actor.a"))
        XCTAssertTrue(options.contains("missing-from"))
        XCTAssertTrue(options.contains("missing-to"))

        store.delegationFromActorID = "stale-from"
        store.delegationToActorID = "stale-to"
        store.seedDelegationDraftIfNeeded(principalOptions: ["default", "actor.a", "actor.b"])

        let seededOptions = store.delegationActorOptions(principalOptions: ["default", "actor.a", "actor.b"])
        XCTAssertTrue(seededOptions.contains(store.delegationFromActorID))
        XCTAssertTrue(seededOptions.contains(store.delegationToActorID))
        XCTAssertNotEqual(store.delegationFromActorID, store.delegationToActorID)
    }

    func testDelegationGrantInputAndDisabledRules() {
        let store = ConfigurationDraftStore()
        store.delegationFromActorID = "default"
        store.delegationToActorID = "default"
        XCTAssertTrue(store.isDelegationGrantDisabled(isGrantInFlight: false, isLoading: false))

        store.delegationToActorID = "actor.a"
        XCTAssertFalse(store.isDelegationGrantDisabled(isGrantInFlight: false, isLoading: false))
        XCTAssertTrue(store.isDelegationGrantDisabled(isGrantInFlight: true, isLoading: false))

        store.delegationScopeType = "ALL"
        store.delegationScopeKey = "ignored"
        store.delegationExpiresAt = "2026-03-05T00:00:00Z"
        let allInput = store.delegationGrantInput()
        XCTAssertNil(allInput.scopeKey)

        store.delegationScopeType = "EXECUTION"
        let scopedInput = store.delegationGrantInput()
        XCTAssertEqual(scopedInput.scopeKey, "ignored")
    }

    func testCapabilityGuidedAndRawScopeSyncLifecycle() {
        let store = ConfigurationDraftStore()
        store.capabilityGrantScopeDraftKey = "region"
        store.capabilityGrantScopeDraftValue = "us"
        store.addCapabilityGrantScopeEntry()

        XCTAssertEqual(store.capabilityGrantScopeEntries.count, 1)
        XCTAssertEqual(store.capabilityGrantDraftScopeJSON, "{\"region\":\"us\"}")

        store.useCapabilityGrantRawScopeOverride = true
        store.capabilityGrantDraftScopeJSON = "{"
        XCTAssertFalse(store.isCapabilityGrantRawScopeJSONValid)
        XCTAssertTrue(store.isCapabilityGrantMutationDisabled(isInFlight: false))

        store.capabilityGrantDraftScopeJSON = "{\"team\":\"ops\"}"
        XCTAssertTrue(store.isCapabilityGrantRawScopeJSONValid)
        XCTAssertTrue(store.applyCapabilityGrantRawScopeToGuided())
        XCTAssertFalse(store.useCapabilityGrantRawScopeOverride)
        XCTAssertEqual(store.capabilityGrantScopeEntries.map(\.key), ["team"])
        XCTAssertEqual(store.capabilityGrantScopeEntries.map(\.value), ["ops"])
    }

    func testCapabilityLoadDraftAndMutationInputHonorRawOverride() {
        let store = ConfigurationDraftStore()
        store.loadCapabilityGrantDraft(
            CapabilityGrantItem(
                id: "grant-1",
                workspaceID: "ws1",
                actorID: "actor.requester.ws1",
                capabilityKey: "mail_send",
                scopeJSON: "{\"channel\":\"mail\"}",
                scopeSummary: "channel=mail",
                status: "revoked",
                createdAtLabel: "now",
                createdAtRaw: "2026-03-05T00:00:00Z",
                expiresAtRaw: "2026-03-06T00:00:00Z",
                expiresAtLabel: "tomorrow",
                sortTimestamp: Date(timeIntervalSince1970: 1)
            )
        )

        XCTAssertEqual(store.capabilityGrantDraftStatus, "REVOKED")
        XCTAssertFalse(store.useCapabilityGrantRawScopeOverride)
        XCTAssertEqual(store.capabilityGrantScopeEntries.count, 1)

        store.useCapabilityGrantRawScopeOverride = true
        store.capabilityGrantDraftScopeJSON = "  {\"override\":\"yes\"}  "
        let input = store.capabilityGrantMutationInput()
        XCTAssertEqual(input.grantID, "grant-1")
        XCTAssertEqual(input.actorID, "actor.requester.ws1")
        XCTAssertEqual(input.scopeJSON, "{\"override\":\"yes\"}")
    }
}
