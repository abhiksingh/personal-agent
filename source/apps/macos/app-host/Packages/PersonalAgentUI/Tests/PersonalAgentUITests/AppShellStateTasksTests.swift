import Foundation
import XCTest
@testable import PersonalAgentUI

@MainActor
final class AppShellStateTasksTests: XCTestCase {
    private let tokenDefaultsKey = "personalagent.ui.local_dev_token"

    override func setUp() {
        super.setUp()
        AppShellState._test_setLocalDevTokenSecretReference(
            service: "personalagent.ui.tests.tasks.\(UUID().uuidString)",
            account: "daemon_auth_token"
        )
        AppShellState._test_clearPersistedLocalDevToken()
    }

    override func tearDown() {
        AppShellState._test_clearPersistedLocalDevToken()
        AppShellState._test_resetLocalDevTokenPersistenceHooks()
        super.tearDown()
    }

    func testOpenInspectForTaskRunSetsRunFocusAndNavigates() {
        let state = AppShellState()

        state.openInspectForTaskRun(makeTaskRow(runID: "run-123"))

        XCTAssertEqual(state.selectedSection, .inspect)
        XCTAssertEqual(state.inspectFocusedRunID, "run-123")
        XCTAssertEqual(state.inspectStatusMessage, "Loading inspect logs for task run run-123…")
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.sourceSection, .tasks)
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.destinationSection, .inspect)
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.chips.first, "Run: run-123")
    }

    func testOpenApprovalsForTaskRunSeedsSearchAndNavigates() {
        let state = AppShellState()

        state.openApprovalsForTaskRun(makeTaskRow(runID: "run-abc"))

        XCTAssertEqual(state.selectedSection, .approvals)
        XCTAssertEqual(state.approvalsSearchSeed, "run-abc")
        XCTAssertEqual(state.approvalsStatusMessage, "Opened Approvals for task run run-abc.")
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.sourceSection, .tasks)
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.destinationSection, .approvals)
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.chips.first, "Run: run-abc")
    }

    func testOpenApprovalsForTaskRunDetailSeedsSearchAndNavigates() {
        let state = AppShellState()

        state.openApprovalsForTaskRunDetail(makeTaskDetail(runID: "run-detail-1"))

        XCTAssertEqual(state.selectedSection, .approvals)
        XCTAssertEqual(state.approvalsSearchSeed, "run-detail-1")
        XCTAssertEqual(state.approvalsStatusMessage, "Opened Approvals for run run-detail-1.")
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.sourceSection, .tasks)
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.destinationSection, .approvals)
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.chips.first, "Run: run-detail-1")
    }

    func testSubmitTaskWithoutTokenSetsStatusMessage() async {
        let defaults = appShellStateTestUserDefaults()
        let priorToken = defaults.object(forKey: tokenDefaultsKey)
        defer {
            if let priorToken {
                defaults.set(priorToken, forKey: tokenDefaultsKey)
            } else {
                defaults.removeObject(forKey: tokenDefaultsKey)
            }
        }

        defaults.removeObject(forKey: tokenDefaultsKey)

        let state = AppShellState()
        state.clearLocalDevToken()

        state.submitTask(
            title: "Review approvals queue",
            description: "Validate pending approvals and summarize risk.",
            taskClass: "approval",
            requestedByActorID: "default",
            subjectPrincipalActorID: "default"
        )
        try? await Task.sleep(for: .milliseconds(50))

        XCTAssertEqual(
            state.taskSubmitStatusMessage,
            "Set Assistant Access Token before submitting tasks."
        )
        XCTAssertFalse(state.isTaskSubmitInFlight)
        XCTAssertNil(state.latestTaskSubmissionReceipt)
    }

    func testSubmitTaskWithoutGoalSetsValidationMessage() async {
        let state = AppShellState()

        state.submitTask(
            title: "   ",
            description: "Draft details",
            taskClass: "chat",
            requestedByActorID: "default",
            subjectPrincipalActorID: "default"
        )
        try? await Task.sleep(for: .milliseconds(50))

        XCTAssertEqual(state.taskSubmitStatusMessage, "Goal is required.")
        XCTAssertFalse(state.isTaskSubmitInFlight)
        XCTAssertNil(state.latestTaskSubmissionReceipt)
    }

    func testSubmitTaskWithOutOfScopeRequestedByPrincipalSetsValidationMessage() async {
        let state = AppShellState()
        state.identityPrincipalItems = [
            IdentityPrincipalItem(
                id: "default",
                displayName: "Default",
                actorType: "human",
                actorStatus: "ACTIVE",
                principalStatus: "ACTIVE",
                isActive: true,
                handles: []
            )
        ]

        state.submitTask(
            title: "Schedule follow-up",
            description: nil,
            taskClass: "comm",
            requestedByActorID: "unknown.actor",
            subjectPrincipalActorID: "default"
        )
        try? await Task.sleep(for: .milliseconds(50))

        XCTAssertEqual(
            state.taskSubmitStatusMessage,
            "Requested By actor `unknown.actor` is not in the active workspace directory."
        )
        XCTAssertFalse(state.isTaskSubmitInFlight)
        XCTAssertNil(state.latestTaskSubmissionReceipt)
    }

    func testSubmitTaskWithOutOfScopeSubjectPrincipalSetsValidationMessage() async {
        let state = AppShellState()
        state.identityPrincipalItems = [
            IdentityPrincipalItem(
                id: "default",
                displayName: "Default",
                actorType: "human",
                actorStatus: "ACTIVE",
                principalStatus: "ACTIVE",
                isActive: true,
                handles: []
            )
        ]

        state.submitTask(
            title: "Collect runtime diagnostics",
            description: "Capture inspect and daemon state context.",
            taskClass: "agent",
            requestedByActorID: "default",
            subjectPrincipalActorID: "unknown.subject"
        )
        try? await Task.sleep(for: .milliseconds(50))

        XCTAssertEqual(
            state.taskSubmitStatusMessage,
            "Subject principal actor `unknown.subject` is not in the active workspace directory."
        )
        XCTAssertFalse(state.isTaskSubmitInFlight)
        XCTAssertNil(state.latestTaskSubmissionReceipt)
    }

    func testRequestTaskRunControlPresentsConfirmationWhenActionAvailable() {
        let state = AppShellState()
        let row = makeTaskRow(
            runID: "run-123",
            actions: TaskRunActionAvailabilityItem(
                canCancel: false,
                canRetry: true,
                canRequeue: false
            )
        )

        state.requestTaskRunControl(.retry, item: row)

        XCTAssertEqual(state.pendingHighImpactActionConfirmation?.title, "Retry Task Run?")
        XCTAssertEqual(state.pendingHighImpactActionConfirmation?.confirmButtonTitle, "Retry Run")
        XCTAssertEqual(state.pendingHighImpactActionConfirmation?.isDestructive, false)
    }

    func testRequestTaskRunControlUnavailableActionSetsStatusWithoutConfirmation() {
        let state = AppShellState()
        let row = makeTaskRow(
            runID: "run-123",
            actions: .unavailable
        )

        state.requestTaskRunControl(.retry, item: row)

        XCTAssertNil(state.pendingHighImpactActionConfirmation)
        XCTAssertEqual(state.taskRunControlStatus(runID: "run-123"), "Retry is available only for failed or cancelled runs.")
        XCTAssertEqual(state.tasksStatusMessage, "Retry is available only for failed or cancelled runs.")
    }

    func testTaskRunControlDisabledReasonReflectsInFlightStatus() {
        let state = AppShellState()
        state.taskRunControlInFlightRunIDs = ["run-123"]

        let reason = state.taskRunControlDisabledReason(
            .cancel,
            runID: "run-123",
            actions: TaskRunActionAvailabilityItem(canCancel: true, canRetry: false, canRequeue: false)
        )

        XCTAssertEqual(reason, "Another run control action is already in progress for this run.")
        XCTAssertFalse(
            state.canPerformTaskRunControl(
                .cancel,
                runID: "run-123",
                actions: TaskRunActionAvailabilityItem(canCancel: true, canRetry: false, canRequeue: false)
            )
        )
    }

    func testTaskRunControlDisabledReasonForMissingRunID() {
        let state = AppShellState()

        let reason = state.taskRunControlDisabledReason(
            .requeue,
            runID: nil,
            actions: TaskRunActionAvailabilityItem(canCancel: false, canRetry: false, canRequeue: true)
        )

        XCTAssertEqual(reason, "Run control is unavailable because this task row has no run id.")
    }

    func testTaskRunCardSummaryAwaitingApprovalHighlightsApprovalPath() {
        let state = AppShellState()
        state.setInformationDensityMode(.simple)
        defer { state.setInformationDensityMode(.simple) }
        let row = makeTaskRow(
            runID: "run-awaiting",
            effectiveState: .awaitingApproval
        )

        let summary = state.taskRunCardSummary(for: row)

        XCTAssertEqual(summary.whatHappened, "Task is paused and waiting for an approval decision.")
        XCTAssertEqual(
            summary.whatNeedsAction,
            "Open Related Approvals and submit a decision to continue this task."
        )
        XCTAssertTrue(summary.whatNext.contains("Open Related Approvals to continue this task."))

        state.setInformationDensityMode(.advanced)
        let advancedSummary = state.taskRunCardSummary(for: row)
        XCTAssertEqual(advancedSummary.whatHappened, "Run is paused waiting for an approval decision.")
        XCTAssertEqual(
            advancedSummary.whatNeedsAction,
            "Open Related Approvals and submit a decision to continue this run."
        )
        XCTAssertTrue(advancedSummary.whatNext.contains("Open Related Approvals to unblock execution."))
    }

    func testTaskRunCardSummaryFailedStateUsesErrorAndRetryGuidance() {
        let state = AppShellState()
        state.setInformationDensityMode(.simple)
        defer { state.setInformationDensityMode(.simple) }
        let row = makeTaskRow(
            runID: "run-failed",
            actions: TaskRunActionAvailabilityItem(
                canCancel: false,
                canRetry: true,
                canRequeue: false
            ),
            effectiveState: .failed,
            lastError: "connector timeout while waiting for provider response"
        )

        let summary = state.taskRunCardSummary(for: row)

        XCTAssertTrue(summary.whatHappened.contains("Task reported an error:"))
        XCTAssertTrue(summary.whatHappened.contains("connector timeout"))
        XCTAssertEqual(
            summary.whatNeedsAction,
            "Review the issue, then retry or requeue once blockers are addressed."
        )
        XCTAssertTrue(summary.whatNext.contains("Use Retry Run after fixing the issue."))

        state.setInformationDensityMode(.advanced)
        let advancedSummary = state.taskRunCardSummary(for: row)
        XCTAssertTrue(advancedSummary.whatHappened.contains("Run reported an error:"))
        XCTAssertEqual(
            advancedSummary.whatNeedsAction,
            "Review failure context, then retry or requeue once blockers are addressed."
        )
        XCTAssertTrue(advancedSummary.whatNext.contains("Use Retry Run after fixing the reported issue."))
    }

    private func makeTaskRow(
        runID: String?,
        actions: TaskRunActionAvailabilityItem = .unavailable,
        effectiveState: TaskRunWorkflowState = .running,
        taskState: String? = nil,
        runState: String? = nil,
        lastError: String? = nil
    ) -> TaskRunListRowItem {
        let resolvedTaskState = taskState ?? effectiveState.label
        let resolvedRunState = runState ?? (runID == nil ? "Unknown" : effectiveState.label)
        return TaskRunListRowItem(
            id: "task-1::\(runID ?? "no-run")",
            title: "Sample Task",
            taskID: "task-1",
            runID: runID,
            taskState: resolvedTaskState,
            runState: resolvedRunState,
            effectiveState: effectiveState,
            priority: 2,
            priorityLabel: "Priority Medium",
            requestedByActorID: "owner",
            subjectPrincipalActorID: "default",
            actingAsActorID: "default",
            taskCreatedAtLabel: "now",
            taskUpdatedAtLabel: "now",
            runCreatedAtLabel: nil,
            runUpdatedAtLabel: nil,
            startedAtLabel: nil,
            finishedAtLabel: nil,
            lastError: lastError,
            actions: actions,
            sortTimestamp: .now,
            route: WorkflowRouteContext(
                available: true,
                taskClass: "chat",
                provider: "openai",
                modelKey: "gpt-5-codex",
                taskClassSource: "policy",
                routeSource: "explicit"
            )
        )
    }

    private func makeTaskDetail(
        runID: String,
        actions: TaskRunActionAvailabilityItem = .unavailable
    ) -> TaskRunDetailItem {
        TaskRunDetailItem(
            id: runID,
            taskID: "task-1",
            runID: runID,
            title: "Sample Task",
            taskState: "Running",
            runState: "Running",
            priorityLabel: "Priority Medium",
            requestedByActorID: "owner",
            subjectPrincipalActorID: "default",
            actingAsActorID: "default",
            startedAtLabel: nil,
            finishedAtLabel: nil,
            updatedAtLabel: "now",
            lastError: nil,
            actions: actions,
            route: WorkflowRouteContext(
                available: true,
                taskClass: "chat",
                provider: "openai",
                modelKey: "gpt-5-codex",
                taskClassSource: "policy",
                routeSource: "explicit"
            ),
            steps: [],
            artifacts: [],
            auditEntries: []
        )
    }
}
