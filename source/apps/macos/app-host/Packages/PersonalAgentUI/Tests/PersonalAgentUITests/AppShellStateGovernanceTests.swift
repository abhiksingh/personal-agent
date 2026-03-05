import XCTest
@testable import PersonalAgentUI

@MainActor
final class AppShellStateGovernanceTests: XCTestCase {
    func testRefreshCapabilityGrantInventoryWithoutTokenSetsDeterministicStatus() async {
        let state = AppShellState()
        state.capabilityGrantItems = [
            CapabilityGrantItem(
                id: "grant-1",
                workspaceID: "ws1",
                actorID: "actor.requester",
                capabilityKey: "messages_send_sms",
                scopeJSON: "{\"channel\":\"sms\"}",
                scopeSummary: "{\"channel\":\"sms\"}",
                status: "ACTIVE",
                createdAtLabel: "now",
                createdAtRaw: "2026-02-25T00:00:00Z",
                expiresAtRaw: nil,
                expiresAtLabel: nil,
                sortTimestamp: Date()
            )
        ]
        state.clearLocalDevToken()

        state.refreshCapabilityGrantInventory()
        try? await Task.sleep(for: .milliseconds(50))

        XCTAssertTrue(state.capabilityGrantItems.isEmpty)
        XCTAssertEqual(state.capabilityGrantStatusMessage, "Set Assistant Access Token to query capability grants.")
        XCTAssertFalse(state.isCapabilityGrantInventoryLoading)
    }

    func testUpsertCapabilityGrantInvalidScopeJSONFailsBeforeDaemonCall() async {
        let state = AppShellState()

        state.upsertCapabilityGrant(
            CapabilityGrantMutationInput(
                grantID: nil,
                actorID: "actor.requester",
                capabilityKey: "messages_send_sms",
                scopeJSON: "{invalid",
                status: "ACTIVE",
                expiresAt: nil
            )
        )
        try? await Task.sleep(for: .milliseconds(50))

        XCTAssertEqual(state.capabilityGrantMutationStatusMessage, "Scope JSON must be valid JSON.")
        XCTAssertFalse(state.isCapabilityGrantMutationInFlight)
    }

    func testRevokeCapabilityGrantWithoutTokenShowsDeterministicStatus() async {
        let state = AppShellState()
        state.clearLocalDevToken()

        state.revokeCapabilityGrant(grantID: "grant-123")
        try? await Task.sleep(for: .milliseconds(50))

        XCTAssertEqual(
            state.capabilityGrantActionStatusByID["grant-123"],
            "Set Assistant Access Token before revoking capability grants."
        )
        XCTAssertEqual(
            state.capabilityGrantMutationStatusMessage,
            "Set Assistant Access Token before revoking capability grants."
        )
    }

    func testRefreshWebhookTrustReceiptsWithoutTokenSetsDeterministicStatus() async {
        let state = AppShellState()
        state.clearLocalDevToken()

        state.refreshWebhookTrustReceipts()
        try? await Task.sleep(for: .milliseconds(50))

        XCTAssertTrue(state.webhookReceiptItems.isEmpty)
        XCTAssertEqual(state.webhookReceiptsStatusMessage, "Set Assistant Access Token to query webhook trust receipts.")
        XCTAssertFalse(state.isWebhookReceiptsLoading)
    }

    func testRefreshIngestTrustReceiptsWithoutTokenSetsDeterministicStatus() async {
        let state = AppShellState()
        state.clearLocalDevToken()

        state.refreshIngestTrustReceipts()
        try? await Task.sleep(for: .milliseconds(50))

        XCTAssertTrue(state.ingestReceiptItems.isEmpty)
        XCTAssertEqual(state.ingestReceiptsStatusMessage, "Set Assistant Access Token to query ingest trust receipts.")
        XCTAssertFalse(state.isIngestReceiptsLoading)
    }

    func testOpenInspectForTrustReceiptAuditLinkSeedsInspectSearchAndNavigates() {
        let state = AppShellState()
        state.selectedSection = .configuration

        state.openInspectForTrustReceiptAuditLink(
            TrustReceiptAuditLinkItem(
                id: "audit-123",
                eventType: "comm_ingest_rejected",
                createdAtLabel: "now"
            )
        )

        XCTAssertEqual(state.selectedSection, .inspect)
        XCTAssertEqual(state.inspectSearchSeed, "audit-123")
        XCTAssertEqual(state.inspectStatusMessage, "Opened Inspect for receipt audit audit-123.")
    }
}

