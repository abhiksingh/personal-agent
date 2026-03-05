import Foundation
import XCTest
@testable import PersonalAgentUI

@MainActor
final class AppShellStateAutomationTests: XCTestCase {
    private let tokenDefaultsKey = "personalagent.ui.local_dev_token"
    private let onboardingDefaultsKey = "personalagent.ui.onboarding_complete"

    override func setUp() {
        super.setUp()
        AppShellState._test_setLocalDevTokenSecretReference(
            service: "personalagent.ui.tests.automation.\(UUID().uuidString)",
            account: "daemon_auth_token"
        )
        AppShellState._test_clearPersistedLocalDevToken()
    }

    override func tearDown() {
        AppShellState._test_clearPersistedLocalDevToken()
        AppShellState._test_resetLocalDevTokenPersistenceHooks()
        super.tearDown()
    }

    func testRefreshAutomationFireHistoryWithoutTokenSetsDeterministicStatus() async {
        let defaults = appShellStateTestUserDefaults()
        let priorToken = defaults.object(forKey: tokenDefaultsKey)
        let priorOnboarding = defaults.object(forKey: onboardingDefaultsKey)
        defer {
            if let priorToken {
                defaults.set(priorToken, forKey: tokenDefaultsKey)
            } else {
                defaults.removeObject(forKey: tokenDefaultsKey)
            }
            if let priorOnboarding {
                defaults.set(priorOnboarding, forKey: onboardingDefaultsKey)
            } else {
                defaults.removeObject(forKey: onboardingDefaultsKey)
            }
        }

        defaults.removeObject(forKey: tokenDefaultsKey)
        defaults.removeObject(forKey: onboardingDefaultsKey)

        let state = AppShellState()
        state.clearLocalDevToken()
        state.refreshAutomationFireHistory()

        try? await Task.sleep(for: .milliseconds(50))

        XCTAssertEqual(state.automationFireHistoryItems.count, 0)
        XCTAssertEqual(
            state.automationFireHistoryStatusMessage,
            "Set Assistant Access Token to query trigger fire history."
        )
        XCTAssertEqual(state.isAutomationFireHistoryLoading, false)
    }

    func testOpenTasksForAutomationFireHistorySeedsSearchWithRunID() {
        let state = AppShellState()
        let item = makeFireHistoryItem(taskID: "task-1", runID: "run-1")

        state.openTasksForAutomationFireHistory(item)

        XCTAssertEqual(state.selectedSection, .tasks)
        XCTAssertEqual(state.tasksSearchSeed, "run-1")
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.sourceSection, .automation)
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.destinationSection, .tasks)
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.chips.first, "Run: run-1")
    }

    func testOpenInspectForAutomationFireHistorySetsRunFilterAndNavigates() {
        let state = AppShellState()
        let item = makeFireHistoryItem(taskID: "task-2", runID: "run-2")

        state.openInspectForAutomationFireHistory(item)

        XCTAssertEqual(state.selectedSection, .inspect)
        XCTAssertEqual(state.inspectFocusedRunID, "run-2")
        XCTAssertEqual(state.inspectStatusMessage, "Loading inspect logs for automation run run-2…")
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.sourceSection, .automation)
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.destinationSection, .inspect)
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.chips.first, "Run: run-2")
    }

    func testOpenTasksForAutomationFireHistorySeedsRouteModelWhenIdentityMissing() {
        let state = AppShellState()
        let item = makeFireHistoryItem(
            taskID: nil,
            runID: nil,
            route: WorkflowRouteContext(
                available: true,
                taskClass: "chat",
                provider: "openai",
                modelKey: "gpt-5-codex",
                taskClassSource: "policy",
                routeSource: "explicit",
                notes: nil
            )
        )

        state.openTasksForAutomationFireHistory(item)

        XCTAssertEqual(state.selectedSection, .tasks)
        XCTAssertEqual(state.tasksSearchSeed, "gpt-5-codex")
        XCTAssertEqual(state.tasksStatusMessage, "Opened Tasks for automation route openai/gpt-5-codex.")
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.chips.first, "Route: openai/gpt-5-codex")
    }

    func testOpenInspectForAutomationFireHistoryUsesRouteSummaryWhenIdentityMissing() {
        let state = AppShellState()
        let item = makeFireHistoryItem(
            taskID: nil,
            runID: nil,
            route: WorkflowRouteContext(
                available: true,
                taskClass: "chat",
                provider: "openai",
                modelKey: "gpt-5-codex",
                taskClassSource: "policy",
                routeSource: "explicit",
                notes: nil
            )
        )

        state.openInspectForAutomationFireHistory(item)

        XCTAssertEqual(state.selectedSection, .inspect)
        XCTAssertNil(state.inspectFocusedRunID)
        XCTAssertEqual(state.inspectStatusMessage, "Opened Inspect from automation route openai/gpt-5-codex.")
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.chips.first, "Route: openai/gpt-5-codex")
    }

    func testClearInspectRunFocusResetsFocusedRunID() {
        let state = AppShellState()
        state.inspectFocusedRunID = "run-3"

        state.clearInspectRunFocus()

        XCTAssertNil(state.inspectFocusedRunID)
        XCTAssertEqual(state.inspectStatusMessage, "Inspect run filter cleared.")
    }

    func testCanonicalizedAutomationCommEventFilterJSONNormalizesChannelAliases() throws {
        let state = AppShellState()
        let raw = """
        {
          "channels": ["twilio_sms", "app_chat", "message", "imessage_sms_bridge", "voice"],
          "principal_actor_ids": ["actor.requester"]
        }
        """

        let normalized = state.canonicalizedAutomationCommEventFilterJSON(raw)
        let data = try XCTUnwrap(normalized.data(using: .utf8))
        let object = try XCTUnwrap(try JSONSerialization.jsonObject(with: data) as? [String: Any])
        let channels = try XCTUnwrap(object["channels"] as? [String])

        XCTAssertEqual(channels, ["message", "app", "voice"])
    }

    func testCanonicalizedAutomationCommEventFilterJSONPreservesNonObjectJSON() {
        let state = AppShellState()
        let raw = "[\"twilio_sms\"]"

        let normalized = state.canonicalizedAutomationCommEventFilterJSON(raw)

        XCTAssertEqual(normalized, raw)
    }

    private func makeFireHistoryItem(
        taskID: String?,
        runID: String?,
        route: WorkflowRouteContext = WorkflowRouteContext()
    ) -> AutomationFireHistoryItem {
        AutomationFireHistoryItem(
            id: "fire-1",
            triggerID: "trigger-1",
            triggerType: "SCHEDULE",
            status: .createdTask,
            outcome: "created_task",
            idempotencySignal: "created",
            idempotencyKey: "idemp-1",
            firedAtLabel: "now",
            taskID: taskID,
            runID: runID,
            sortTimestamp: Date.now,
            route: route
        )
    }
}
