import Foundation
import XCTest
@testable import PersonalAgentUI

@MainActor
final class AppShellStateCommunicationsTests: XCTestCase {
    private let tokenDefaultsKey = "personalagent.ui.local_dev_token"
    private let onboardingDefaultsKey = "personalagent.ui.onboarding_complete"

    override func setUp() {
        super.setUp()
        AppShellState._test_setLocalDevTokenSecretReference(
            service: "personalagent.ui.tests.communications.\(UUID().uuidString)",
            account: "daemon_auth_token"
        )
        AppShellState._test_clearPersistedLocalDevToken()
    }

    override func tearDown() {
        AppShellState._test_clearPersistedLocalDevToken()
        AppShellState._test_resetLocalDevTokenPersistenceHooks()
        super.tearDown()
    }

    func testRefreshCommunicationsInboxWithoutTokenSetsDeterministicStatus() async {
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
        state.refreshCommunicationsInbox()

        try? await Task.sleep(for: .milliseconds(80))

        XCTAssertEqual(state.communicationThreads.count, 0)
        XCTAssertEqual(state.communicationEvents.count, 0)
        XCTAssertEqual(state.communicationCallSessions.count, 0)
        XCTAssertFalse(state.communicationThreadsHasMore)
        XCTAssertFalse(state.communicationEventsHasMore)
        XCTAssertFalse(state.communicationCallSessionsHasMore)
        XCTAssertTrue(state.communicationContinuityItems.isEmpty)
        XCTAssertFalse(state.communicationContinuityHasMore)
        XCTAssertEqual(
            state.communicationContinuityStatusMessage,
            "Set Assistant Access Token to query conversation continuity."
        )
        XCTAssertEqual(
            state.communicationsStatusMessage,
            "Set Assistant Access Token to query communications inbox."
        )
    }

    func testRefreshCommunicationAttemptsWithoutTokenSetsDeterministicStatus() async {
        let state = AppShellState()
        state.clearLocalDevToken()

        state.refreshCommunicationAttempts(threadID: "thread-1")
        try? await Task.sleep(for: .milliseconds(120))

        XCTAssertTrue(state.communicationDeliveryAttempts.isEmpty)
        XCTAssertFalse(state.communicationDeliveryAttemptsHasMore)
        XCTAssertEqual(
            state.communicationAttemptsStatusMessage,
            "Set Assistant Access Token to query delivery attempts."
        )
    }

    func testRefreshCommunicationAttemptsWithoutThreadSetsSelectionGuidance() async {
        let state = AppShellState()
        state.localDevTokenInput = "test-token"
        state.saveLocalDevToken()

        state.refreshCommunicationAttempts(threadID: nil)
        try? await Task.sleep(for: .milliseconds(120))

        XCTAssertTrue(state.communicationDeliveryAttempts.isEmpty)
        XCTAssertFalse(state.communicationDeliveryAttemptsHasMore)
        XCTAssertEqual(
            state.communicationAttemptsStatusMessage,
            "Select a conversation to load delivery attempts."
        )
    }

    func testSendCommunicationWithoutMessageSetsValidationStatus() async {
        let state = AppShellState()

        state.sendCommunication(
            sourceChannel: "message",
            destination: "+15550001111",
            message: "   "
        )
        try? await Task.sleep(for: .milliseconds(80))

        XCTAssertEqual(state.communicationSendStatusMessage, "Message body is required.")
        XCTAssertFalse(state.isCommunicationSendInFlight)
    }

    func testSendCommunicationWithoutDestinationOrThreadSetsValidationStatus() async {
        let state = AppShellState()

        state.sendCommunication(
            sourceChannel: "message",
            destination: nil,
            message: "hello"
        )
        try? await Task.sleep(for: .milliseconds(80))

        XCTAssertEqual(
            state.communicationSendStatusMessage,
            "Destination is required unless a thread context is selected."
        )
        XCTAssertFalse(state.isCommunicationSendInFlight)
    }

    func testSendCommunicationWithThreadContextAllowsEmptyDestinationValidationPath() async {
        let state = AppShellState()
        state.clearLocalDevToken()

        state.sendCommunication(
            sourceChannel: "message",
            destination: nil,
            message: "hello",
            threadID: "thread-1"
        )
        try? await Task.sleep(for: .milliseconds(120))

        XCTAssertEqual(
            state.communicationSendStatusMessage,
            "Set Assistant Access Token before sending communications."
        )
        XCTAssertFalse(state.isCommunicationSendInFlight)
    }

    func testSendCommunicationWithWhitespaceThreadStillRequiresDestination() async {
        let state = AppShellState()

        state.sendCommunication(
            sourceChannel: "message",
            destination: nil,
            message: "hello",
            threadID: "   "
        )
        try? await Task.sleep(for: .milliseconds(80))

        XCTAssertEqual(
            state.communicationSendStatusMessage,
            "Destination is required unless a thread context is selected."
        )
        XCTAssertFalse(state.isCommunicationSendInFlight)
    }

    func testSendCommunicationWithoutTokenSetsDeterministicStatus() async {
        let state = AppShellState()
        state.clearLocalDevToken()

        state.sendCommunication(
            sourceChannel: "message",
            destination: "+15550001111",
            message: "hello"
        )
        try? await Task.sleep(for: .milliseconds(120))

        XCTAssertEqual(
            state.communicationSendStatusMessage,
            "Set Assistant Access Token before sending communications."
        )
        XCTAssertFalse(state.isCommunicationSendInFlight)
    }

    func testOpenInspectForCommunicationEventSeedsSearchAndNavigates() {
        let state = AppShellState()
        let event = CommunicationEventItem(
            id: "event-1",
            workspaceID: "ws1",
            threadID: "thread-1",
            channel: "message",
            connectorID: "imessage",
            eventType: "MESSAGE",
            direction: "inbound",
            assistantEmitted: false,
            bodyText: "hello",
            occurredAtLabel: "now",
            createdAtLabel: "now",
            addresses: [],
            sortTimestamp: .now
        )

        state.openInspectForCommunicationEvent(event)

        XCTAssertEqual(state.selectedSection, .inspect)
        XCTAssertEqual(state.inspectSearchSeed, "event-1")
        XCTAssertEqual(state.inspectStatusMessage, "Opened Inspect for communication event event-1.")
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.sourceSection, .communications)
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.destinationSection, .inspect)
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.chips.first, "Event: event-1")
    }

    func testOpenInspectForCommunicationCallSessionPrefersProviderCallIDSearchSeed() {
        let state = AppShellState()
        let session = CommunicationCallSessionItem(
            id: "session-1",
            workspaceID: "ws1",
            provider: "twilio_voice",
            connectorID: "twilio",
            providerCallID: "CA42",
            threadID: "thread-voice",
            direction: "outbound",
            fromAddress: "+15550000001",
            toAddress: "+15550000002",
            status: "in_progress",
            startedAtLabel: "now",
            endedAtLabel: nil,
            updatedAtLabel: "now",
            sortTimestamp: .now
        )

        state.openInspectForCommunicationCallSession(session)

        XCTAssertEqual(state.selectedSection, .inspect)
        XCTAssertEqual(state.inspectSearchSeed, "CA42")
        XCTAssertEqual(state.inspectStatusMessage, "Opened Inspect for call session session-1.")
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.sourceSection, .communications)
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.destinationSection, .inspect)
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.chips.first, "Call: session-1")
    }

    func testOpenChannelsForCommunicationChannelNavigatesWithStatusCopy() {
        let state = AppShellState()

        state.openChannelsForCommunicationChannel("twilio_voice")

        XCTAssertEqual(state.selectedSection, .channels)
        XCTAssertEqual(
            state.channelsStatusMessage,
            "Opened Channels for communication channel voice."
        )
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.sourceSection, .communications)
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.destinationSection, .channels)
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.chips.first, "Channel: voice")
    }

    func testOpenChannelsForCommunicationMessageAliasNormalizesToLogicalMessage() {
        let state = AppShellState()

        state.openChannelsForCommunicationChannel("twilio_sms")

        XCTAssertEqual(state.selectedSection, .channels)
        XCTAssertEqual(
            state.channelsStatusMessage,
            "Opened Channels for communication channel message."
        )
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.chips.first, "Channel: message")
    }

    func testOpenTasksForCommunicationAttemptUsesRunIDWhenAvailable() {
        let state = AppShellState()
        let attempt = CommunicationDeliveryAttemptItem(
            id: "attempt-1",
            workspaceID: "ws1",
            operationID: "op-1",
            taskID: "task-1",
            runID: "run-1",
            stepID: "step-1",
            eventID: "event-1",
            threadID: "thread-1",
            destinationEndpoint: "+15550000001",
            idempotencyKey: "idem-1",
            channel: "sms",
            routeIndex: 1,
            routePhase: "fallback",
            retryOrdinal: 1,
            fallbackFromChannel: "imessage",
            status: "delivered",
            providerReceipt: "receipt-1",
            error: nil,
            attemptedAtLabel: "now",
            sortTimestamp: .now
        )

        state.openTasksForCommunicationAttempt(attempt)

        XCTAssertEqual(state.selectedSection, .tasks)
        XCTAssertEqual(state.tasksSearchSeed, "run-1")
        XCTAssertEqual(state.tasksStatusMessage, "Opened Tasks for delivery run run-1.")
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.sourceSection, .communications)
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.destinationSection, .tasks)
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.chips.first, "Run: run-1")
    }

    func testOpenTaskDraftForCommunicationThreadSeedsTasksDraftAndNavigates() {
        let state = AppShellState()
        let thread = CommunicationThreadItem(
            id: "thread-42",
            workspaceID: "ws1",
            channel: "message",
            connectorID: "imessage",
            title: "Family Follow Up",
            externalRef: "ext-42",
            lastEventID: "event-42",
            lastEventType: "MESSAGE",
            lastDirection: "inbound",
            lastOccurredAtLabel: "now",
            lastBodyPreview: "Can we sync tomorrow morning?",
            participantAddresses: ["mom@example.com"],
            eventCount: 12,
            createdAtLabel: "today",
            updatedAtLabel: "today",
            sortTimestamp: .now
        )

        state.openTaskDraftForCommunicationThread(thread)

        XCTAssertEqual(state.selectedSection, .tasks)
        XCTAssertEqual(state.tasksSearchSeed, "thread-42")
        XCTAssertEqual(state.tasksStatusMessage, "Opened task draft for communication thread thread-42.")
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.sourceSection, .communications)
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.destinationSection, .tasks)
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.chips.first, "Thread: thread-42")
        XCTAssertEqual(state.taskSubmitDraftSeed?.taskClass, "chat")
        XCTAssertEqual(state.taskSubmitDraftSeed?.title, "Follow up: Family Follow Up")
        XCTAssertTrue(state.taskSubmitDraftSeed?.description?.contains("thread thread-42") == true)
    }

    func testOpenChatForCommunicationContinuityNavigatesWithDrillInContext() {
        let state = AppShellState()
        let item = CommunicationContinuityItem(
            id: "message::turn-1::corr-1",
            turnID: "turn-1",
            workspaceID: "ws1",
            channel: "message",
            connectorID: "twilio",
            threadID: "thread-1",
            correlationID: "corr-1",
            taskClass: "chat",
            itemType: "assistant_message",
            itemStatus: "completed",
            summary: "Sent a message.",
            taskID: "task-1",
            runID: "run-1",
            taskState: "completed",
            runState: "completed",
            createdAtLabel: "now",
            sortTimestamp: .now
        )

        state.openChatForCommunicationContinuity(item)

        XCTAssertEqual(state.selectedSection, .chat)
        XCTAssertEqual(state.chatStatusMessage, "Opened Chat for Message continuity.")
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.sourceSection, .communications)
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.destinationSection, .chat)
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.chips.first, "Channel: message")
    }

    func testOpenTasksForCommunicationContinuityPrefersRunIDSearchSeed() {
        let state = AppShellState()
        let item = CommunicationContinuityItem(
            id: "voice::turn-2::corr-2",
            turnID: "turn-2",
            workspaceID: "ws1",
            channel: "voice",
            connectorID: "twilio",
            threadID: "thread-voice-1",
            correlationID: "corr-2",
            taskClass: "chat",
            itemType: "tool_result",
            itemStatus: "failed",
            summary: "start_call failed.",
            taskID: "task-voice-1",
            runID: "run-voice-1",
            taskState: "failed",
            runState: "failed",
            createdAtLabel: "now",
            sortTimestamp: .now
        )

        state.openTasksForCommunicationContinuity(item)

        XCTAssertEqual(state.selectedSection, .tasks)
        XCTAssertEqual(state.tasksSearchSeed, "run-voice-1")
        XCTAssertEqual(state.tasksStatusMessage, "Opened Tasks for continuity run run-voice-1.")
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.sourceSection, .communications)
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.destinationSection, .tasks)
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.chips.first, "Run: run-voice-1")
    }

    func testOpenInspectForCommunicationContinuityPrefersRunFocusedInspectContext() {
        let state = AppShellState()
        let item = CommunicationContinuityItem(
            id: "app::turn-3::corr-3",
            turnID: "turn-3",
            workspaceID: "ws1",
            channel: "app",
            connectorID: "builtin.app",
            threadID: "thread-app-1",
            correlationID: "corr-3",
            taskClass: "chat",
            itemType: "approval_request",
            itemStatus: "awaiting_approval",
            summary: "Approval is required.",
            taskID: "task-app-1",
            runID: "run-app-1",
            taskState: "awaiting_approval",
            runState: "awaiting_approval",
            createdAtLabel: "now",
            sortTimestamp: .now
        )

        state.openInspectForCommunicationContinuity(item)

        XCTAssertEqual(state.selectedSection, .inspect)
        XCTAssertEqual(state.inspectFocusedRunID, "run-app-1")
        XCTAssertNil(state.inspectSearchSeed)
        XCTAssertEqual(state.inspectStatusMessage, "Loading inspect logs for continuity run run-app-1…")
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.sourceSection, .communications)
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.destinationSection, .inspect)
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.chips.first, "Run: run-app-1")
    }
}
