import XCTest
@testable import PersonalAgentUI

@MainActor
final class AppShellStateHighImpactActionTests: XCTestCase {
    func testRequestStartDaemonPresentsConfirmation() {
        let state = AppShellState()

        state.requestStartDaemon()

        XCTAssertEqual(state.pendingHighImpactActionConfirmation?.title, "Start Daemon?")
        XCTAssertEqual(state.pendingHighImpactActionConfirmation?.confirmButtonTitle, "Start Daemon")
        XCTAssertEqual(state.pendingHighImpactActionConfirmation?.isDestructive, false)
    }

    func testConfirmStartDaemonShowsUndoPrompt() {
        let state = AppShellState()
        state.clearLocalDevToken()
        state.requestStartDaemon()

        state.confirmPendingHighImpactAction()

        XCTAssertNil(state.pendingHighImpactActionConfirmation)
        XCTAssertEqual(state.activeUndoActionPrompt?.title, "Daemon start requested")
        XCTAssertEqual(state.activeUndoActionPrompt?.actionTitle, "Undo Start")
    }

    func testCancelPendingHighImpactActionClearsConfirmation() {
        let state = AppShellState()
        state.requestStopDaemon()

        state.cancelPendingHighImpactAction()

        XCTAssertNil(state.pendingHighImpactActionConfirmation)
    }

    func testPerformUndoActionClearsPrompt() {
        let state = AppShellState()
        state.clearLocalDevToken()
        state.requestStopDaemon()
        state.confirmPendingHighImpactAction()
        XCTAssertNotNil(state.activeUndoActionPrompt)

        state.performActiveUndoAction()

        XCTAssertNil(state.activeUndoActionPrompt)
    }

    func testRequestConnectorPermissionSkipsConfirmation() {
        let state = AppShellState()

        state.requestConnectorPermissionWithConfirmation(connectorID: "mail")

        XCTAssertNil(state.pendingHighImpactActionConfirmation)
    }

    func testRequestRevokeDelegationRuleUsesDestructiveConfirmation() {
        let state = AppShellState()

        state.requestRevokeDelegationRule(ruleID: "rule-123")

        XCTAssertEqual(state.pendingHighImpactActionConfirmation?.title, "Revoke Delegation Rule?")
        XCTAssertEqual(state.pendingHighImpactActionConfirmation?.confirmButtonTitle, "Revoke Rule")
        XCTAssertEqual(state.pendingHighImpactActionConfirmation?.isDestructive, true)
    }

    func testRequestSaveChannelDeliveryPolicyUsesConfirmation() {
        let state = AppShellState()
        state.startNewChannelDeliveryPolicyDraft(channelID: "message")
        state.setChannelDeliveryPolicyPrimaryChannel(channelID: "message", primaryChannel: "message")
        state.setChannelDeliveryPolicyFallbackChannelsText(channelID: "message", fallbackChannelsText: "voice")
        state.setChannelDeliveryPolicyEndpointPattern(channelID: "message", endpointPattern: "+1555*")

        state.requestSaveChannelDeliveryPolicy(channelID: "message")

        XCTAssertEqual(state.pendingHighImpactActionConfirmation?.title, "Save Delivery Policy?")
        XCTAssertEqual(state.pendingHighImpactActionConfirmation?.confirmButtonTitle, "Save Policy")
        XCTAssertEqual(state.pendingHighImpactActionConfirmation?.isDestructive, false)
    }

    func testRequestSaveModelRoutePolicyUsesConfirmation() {
        let state = AppShellState()

        state.requestSaveModelRoutePolicy(
            taskClass: "chat",
            providerID: "openai",
            modelKey: "gpt-4.1"
        )

        XCTAssertEqual(state.pendingHighImpactActionConfirmation?.title, "Save Route Policy?")
        XCTAssertEqual(state.pendingHighImpactActionConfirmation?.confirmButtonTitle, "Save Route Policy")
        XCTAssertEqual(state.pendingHighImpactActionConfirmation?.isDestructive, false)
    }

    func testRequestRunRetentionPurgeUsesIrreversibleDestructiveConfirmation() {
        let state = AppShellState()

        state.requestRunRetentionPurge()

        XCTAssertEqual(state.pendingHighImpactActionConfirmation?.title, "Run Retention Purge?")
        XCTAssertEqual(state.pendingHighImpactActionConfirmation?.confirmButtonTitle, "Run Purge")
        XCTAssertEqual(state.pendingHighImpactActionConfirmation?.isDestructive, true)
        XCTAssertEqual(
            state.pendingHighImpactActionConfirmation?.irreversibleNote,
            "Purged records cannot be restored from app history."
        )
    }

    func testRequestRunRetentionCompactMemoryPreviewIsNonDestructive() {
        let state = AppShellState()
        state.retentionCompactionApply = false

        state.requestRunRetentionCompactMemory()

        XCTAssertEqual(state.pendingHighImpactActionConfirmation?.title, "Preview Memory Compaction?")
        XCTAssertEqual(state.pendingHighImpactActionConfirmation?.confirmButtonTitle, "Run Preview")
        XCTAssertEqual(state.pendingHighImpactActionConfirmation?.isDestructive, false)
    }

    func testRequestRunRetentionCompactMemoryApplyIsDestructive() {
        let state = AppShellState()
        state.retentionCompactionApply = true

        state.requestRunRetentionCompactMemory()

        XCTAssertEqual(state.pendingHighImpactActionConfirmation?.title, "Run Memory Compaction?")
        XCTAssertEqual(state.pendingHighImpactActionConfirmation?.confirmButtonTitle, "Run Compaction")
        XCTAssertEqual(state.pendingHighImpactActionConfirmation?.isDestructive, true)
        XCTAssertEqual(
            state.pendingHighImpactActionConfirmation?.irreversibleNote,
            "Compaction writes mutate retained memory entries."
        )
    }
}
