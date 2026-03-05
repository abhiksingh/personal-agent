import XCTest
@testable import PersonalAgentUI

@MainActor
final class AppShellStateDelegationTests: XCTestCase {
    func testRefreshDelegationRulesWithoutTokenSetsDeterministicStatus() async {
        let state = AppShellState()
        state.delegationRules = [
            DelegationRuleItem(
                id: "rule-existing",
                workspaceID: "ws1",
                fromActorID: "actor.from",
                toActorID: "actor.to",
                scopeType: "EXECUTION",
                scopeKey: nil,
                status: "ACTIVE",
                createdAtLabel: "now",
                expiresAtLabel: nil
            )
        ]
        state.clearLocalDevToken()

        state.refreshDelegationRules()
        try? await Task.sleep(for: .milliseconds(50))

        XCTAssertTrue(state.delegationRules.isEmpty)
        XCTAssertEqual(state.delegationStatusMessage, "Set Assistant Access Token to query delegation rules.")
        XCTAssertFalse(state.isDelegationLoading)
    }

    func testCreateDelegationRuleRejectsSelfDelegationBeforeDaemonCall() async {
        let state = AppShellState()

        state.createDelegationRule(
            DelegationGrantInput(
                fromActorID: "actor.requester",
                toActorID: "actor.requester",
                scopeType: "EXECUTION",
                scopeKey: nil,
                expiresAt: nil
            )
        )
        try? await Task.sleep(for: .milliseconds(50))

        XCTAssertEqual(state.delegationStatusMessage, "Delegation denied: self delegation is not allowed.")
        XCTAssertFalse(state.isDelegationGrantInFlight)
    }

    func testCreateDelegationRuleRejectsScopeKeyForAllScopeBeforeDaemonCall() async {
        let state = AppShellState()

        state.createDelegationRule(
            DelegationGrantInput(
                fromActorID: "actor.requester",
                toActorID: "actor.delegate",
                scopeType: "ALL",
                scopeKey: "chat.send",
                expiresAt: nil
            )
        )
        try? await Task.sleep(for: .milliseconds(50))

        XCTAssertEqual(state.delegationStatusMessage, "scope_key is not allowed when scope_type=ALL.")
        XCTAssertFalse(state.isDelegationGrantInFlight)
    }

    func testRevokeDelegationRuleWithoutTokenShowsDeterministicStatus() async {
        let state = AppShellState()
        state.clearLocalDevToken()

        state.revokeDelegationRule(ruleID: "rule-123")
        try? await Task.sleep(for: .milliseconds(50))

        XCTAssertEqual(
            state.delegationActionStatusByRuleID["rule-123"],
            "Set Assistant Access Token before revoking delegation rules."
        )
        XCTAssertEqual(state.delegationStatusMessage, "Set Assistant Access Token before revoking delegation rules.")
    }
}
