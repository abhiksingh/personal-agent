import XCTest
@testable import PersonalAgentUI

@MainActor
final class AppShellStatePersonaPolicyTests: XCTestCase {
    func testRefreshChatPersonaPolicyWithoutTokenSetsDeterministicStatus() async {
        let state = AppShellState()
        state.clearLocalDevToken()
        state.chatPersonaStylePromptDraft = "stale"
        state.chatPersonaGuardrailsDraft = "stale"

        state.refreshChatPersonaPolicy()
        try? await Task.sleep(for: .milliseconds(50))

        XCTAssertEqual(state.chatPersonaPolicyStatusMessage, "Set Assistant Access Token to load chat persona policy.")
        XCTAssertFalse(state.chatPersonaHasLoadedPolicy)
        XCTAssertNil(state.chatPersonaPolicyItem)
        XCTAssertEqual(state.chatPersonaStylePromptDraft, "")
        XCTAssertEqual(state.chatPersonaGuardrailsDraft, "")
    }

    func testSaveChatPersonaPolicyWithoutTokenShowsDeterministicStatus() async {
        let state = AppShellState()
        state.clearLocalDevToken()
        state.chatPersonaStylePromptDraft = "Be concise."

        state.saveChatPersonaPolicy()
        try? await Task.sleep(for: .milliseconds(50))

        XCTAssertEqual(state.chatPersonaPolicyStatusMessage, "Set Assistant Access Token before saving persona policy.")
        XCTAssertFalse(state.isChatPersonaPolicySaveInFlight)
    }

    func testResetChatPersonaPolicyDraftRevertsToLoadedPolicy() {
        let state = AppShellState()
        state.chatPersonaPolicyItem = ChatPersonaPolicyItem(
            workspaceID: "ws1",
            principalActorID: "default",
            channelID: "app",
            stylePrompt: "Loaded style.",
            guardrails: ["One", "Two"],
            source: "persisted",
            updatedAtRaw: "2026-02-28T12:30:00Z",
            updatedAtLabel: "Feb 28, 2026 at 12:30 PM"
        )
        state.chatPersonaStylePromptDraft = "Unsaved style."
        state.chatPersonaGuardrailsDraft = "Unsaved guardrail"

        state.resetChatPersonaPolicyDraft()

        XCTAssertEqual(state.chatPersonaStylePromptDraft, "Loaded style.")
        XCTAssertEqual(state.chatPersonaGuardrailsDraft, "One\nTwo")
        XCTAssertEqual(state.chatPersonaPolicyStatusMessage, "Reverted unsaved persona policy changes.")
    }

    func testTestChatPersonaPolicyInChatNavigatesWhenStylePromptPresent() {
        let state = AppShellState()
        state.selectedSection = .configuration
        state.chatPersonaScopeType = .principalChannel
        state.chatPersonaScopePrincipalActorID = "default"
        state.chatPersonaScopeChannelID = "message"
        state.chatPersonaStylePromptDraft = "Use concise status updates."

        state.testChatPersonaPolicyInChat()

        XCTAssertEqual(state.selectedSection, .chat)
        XCTAssertEqual(
            state.chatStatusMessage,
            "Persona test scope active (Principal Default Principal • Message Channel • app • app.default). Send a message to validate tone and guardrails."
        )
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.chips.last, "Response Profile: app.default")
    }

    func testChatPersonaSaveDisabledReasonRequiresStylePromptAndChanges() {
        let state = AppShellState()
        state.localDevTokenInput = "fixture-token"
        state.saveLocalDevToken()
        state.chatPersonaStylePromptDraft = ""
        state.chatPersonaGuardrailsDraft = ""
        state.chatPersonaPolicyItem = ChatPersonaPolicyItem(
            workspaceID: "ws1",
            principalActorID: nil,
            channelID: nil,
            stylePrompt: "",
            guardrails: [],
            source: "default",
            updatedAtRaw: nil,
            updatedAtLabel: nil
        )

        XCTAssertEqual(state.chatPersonaSaveDisabledReason, "Style prompt is required.")
    }
}
