import XCTest
@testable import PersonalAgentUI

@MainActor
final class AppShellStateApprovalsTests: XCTestCase {
    func testOpenTasksForApprovalSeedsRunIDAndNavigates() {
        let state = AppShellState()
        let item = makeApprovalItem(taskID: "task-1", runID: "run-1")

        state.openTasksForApproval(item)

        XCTAssertEqual(state.selectedSection, .tasks)
        XCTAssertEqual(state.tasksSearchSeed, "run-1")
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.sourceSection, .approvals)
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.destinationSection, .tasks)
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.chips.first, "Run: run-1")
    }

    func testOpenInspectForApprovalSetsRunFocusAndNavigates() {
        let state = AppShellState()
        let item = makeApprovalItem(taskID: "task-2", runID: "run-2")

        state.openInspectForApproval(item)

        XCTAssertEqual(state.selectedSection, .inspect)
        XCTAssertEqual(state.inspectFocusedRunID, "run-2")
        XCTAssertEqual(state.inspectStatusMessage, "Loading inspect logs for approval run run-2…")
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.sourceSection, .approvals)
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.destinationSection, .inspect)
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.chips.first, "Run: run-2")
    }

    func testOpenTaskRunDetailForApprovalSeedsRunAndNavigates() {
        let state = AppShellState()
        let item = makeApprovalItem(taskID: "task-3", runID: "run-3")

        state.openTaskRunDetailForApproval(item)

        XCTAssertEqual(state.selectedSection, .tasks)
        XCTAssertEqual(state.tasksSearchSeed, "run-3")
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.sourceSection, .approvals)
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.destinationSection, .tasks)
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.chips.first, "Run: run-3")
    }

    func testOpenTasksForApprovalFallsBackToRouteContextWhenTaskRunMissing() {
        let state = AppShellState()
        let item = makeApprovalItem(taskID: nil, runID: nil)

        state.openTasksForApproval(item)

        XCTAssertEqual(state.selectedSection, .tasks)
        XCTAssertEqual(state.tasksSearchSeed, "llama3.2")
        XCTAssertEqual(state.tasksStatusMessage, "Opened Tasks for approval route ollama/llama3.2.")
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.chips.first, "Route: ollama/llama3.2")
    }

    func testOpenInspectForApprovalFallsBackToRouteContextWhenTaskRunMissing() {
        let state = AppShellState()
        let item = makeApprovalItem(taskID: nil, runID: nil)

        state.openInspectForApproval(item)

        XCTAssertEqual(state.selectedSection, .inspect)
        XCTAssertNil(state.inspectFocusedRunID)
        XCTAssertEqual(state.inspectSearchSeed, "llama3.2")
        XCTAssertEqual(state.inspectStatusMessage, "Opened Inspect from approval route ollama/llama3.2.")
        XCTAssertEqual(state.activeDrillInContextForSelectedSection?.chips.first, "Model: llama3.2")
    }

    func testApprovalEvidenceInputOutputSummariesExtractsStructuredFields() {
        let auditEntries = [
            DaemonInspectRunAuditEntry(
                auditID: "audit-step",
                workspaceID: "ws1",
                runID: "run-1",
                stepID: "step-1",
                eventType: "STEP_EXECUTED",
                actorID: nil,
                actingAsActorID: nil,
                correlationID: "corr-1",
                payloadJSON: #"{"status":"COMPLETED","summary":"Calendar event created","input":{"title":"Planning Sync","time":"10:00"},"output":{"event_id":"evt-1"}}"#,
                createdAt: "2026-02-25T10:00:00Z"
            )
        ]

        let summaries = AppShellState.approvalEvidenceInputOutputSummaries(from: auditEntries)

        XCTAssertEqual(summaries.output, "Calendar event created")
        XCTAssertNotNil(summaries.input)
        XCTAssertTrue(summaries.input?.contains("Planning Sync") == true)
    }

    func testApprovalEvidencePayloadSummaryFormatsKnownPayloadFields() {
        let summary = AppShellState.approvalEvidencePayloadSummary(
            from: #"{"requested_phrase":"GO AHEAD","capability_key":"finder_delete","summary":"Approval requested for delete"}"#
        )

        XCTAssertNotNil(summary)
        XCTAssertTrue(summary?.contains("summary: Approval requested for delete") == true)
        XCTAssertTrue(summary?.contains("requested phrase: GO AHEAD") == true)
    }

    func testApprovalEvidencePayloadSummaryFormatsJSONArraysWithoutDynamicDictionaryBridge() {
        let summary = AppShellState.approvalEvidencePayloadSummary(
            from: #"[{"step":"draft","status":"ok"},{"step":"send","status":"pending"}]"#
        )

        XCTAssertNotNil(summary)
        XCTAssertTrue(summary?.contains("step: draft") == true)
        XCTAssertTrue(summary?.contains("status: ok") == true)
    }

    func testInspectRunAuditEntryPayloadObjectDecodesTypedJSONValue() {
        let entry = DaemonInspectRunAuditEntry(
            auditID: "audit-payload",
            workspaceID: "ws1",
            runID: "run-1",
            stepID: "step-1",
            eventType: "STEP_EXECUTED",
            actorID: nil,
            actingAsActorID: nil,
            correlationID: "corr-payload",
            payloadJSON: #"{"summary":"Calendar event created","output":{"event_id":"evt-1"}}"#,
            createdAt: "2026-02-25T10:00:00Z"
        )

        XCTAssertEqual(entry.payloadObject?["summary"]?.stringValue, "Calendar event created")
        XCTAssertEqual(entry.payloadObject?["output"]?.objectValue?["event_id"]?.stringValue, "evt-1")
    }

    func testApprovalCardSummaryPendingHighlightsDecisionRequirements() {
        let state = AppShellState()
        state.setInformationDensityMode(.simple)
        defer { state.setInformationDensityMode(.simple) }
        let item = makeApprovalItem(taskID: "task-1", runID: "run-1")

        let summary = state.approvalCardSummary(for: item)

        XCTAssertTrue(summary.whatHappened.contains("Approval needed for Delete."))
        XCTAssertTrue(summary.whatNeedsAction.contains("exact phrase `GO AHEAD`"))
        XCTAssertEqual(
            summary.whatNext,
            "Review details, then submit your decision."
        )

        state.setInformationDensityMode(.advanced)
        let advancedSummary = state.approvalCardSummary(for: item)
        XCTAssertTrue(advancedSummary.whatHappened.contains("Approval requested for Delete."))
        XCTAssertEqual(
            advancedSummary.whatNext,
            "Review Evidence or Open Task Detail, then submit a decision."
        )
    }

    func testApprovalCardSummaryFinalApprovedHighlightsFollowThrough() {
        let state = AppShellState()
        state.setInformationDensityMode(.simple)
        defer { state.setInformationDensityMode(.simple) }
        let item = makeApprovalItem(
            taskID: "task-9",
            runID: "run-9",
            decisionState: .final,
            decisionOutcome: .approved,
            decisionRationale: "Confirmed safe scope."
        )

        let summary = state.approvalCardSummary(for: item)

        XCTAssertTrue(summary.whatHappened.contains("Decision recorded: Approved."))
        XCTAssertEqual(summary.whatNeedsAction, "No further action is required.")
        XCTAssertEqual(
            summary.whatNext,
            "Open Related Tasks to confirm what happened next."
        )

        state.setInformationDensityMode(.advanced)
        let advancedSummary = state.approvalCardSummary(for: item)
        XCTAssertEqual(advancedSummary.whatNeedsAction, "No further decision action is required.")
        XCTAssertEqual(
            advancedSummary.whatNext,
            "Open Related Tasks or Inspect to confirm post-approval execution."
        )
    }

    func testDefaultApprovalDecisionActorPrefersActingAsThenRequestedByThenSelectedPrincipal() {
        let state = AppShellState()
        state.selectedPrincipal = "actor.selected"
        state.identityPrincipalItems = [
            makePrincipal(id: "actor.acting"),
            makePrincipal(id: "actor.requester"),
            makePrincipal(id: "actor.selected")
        ]

        let actingAsItem = makeApprovalItem(taskID: "task-1", runID: "run-1")
        XCTAssertEqual(state.defaultApprovalDecisionActor(for: actingAsItem), "actor.subject")

        let fallbackToRequestedBy = ApprovalInboxItem(
            id: "approval-actor-2",
            taskTitle: "Archive file",
            decisionState: .pending,
            decisionOutcome: nil,
            riskLevel: .policy,
            riskRationale: "Policy review required.",
            requestedAtLabel: "now",
            decidedAtLabel: nil,
            decisionByActorID: nil,
            decisionRationale: nil,
            requestedPhrase: nil,
            taskState: "awaiting_approval",
            runState: "awaiting_approval",
            stepName: "Archive",
            stepCapabilityKey: nil,
            requestedByActorID: "actor.requester",
            subjectPrincipalActorID: "actor.subject",
            actingAsActorID: "unknown",
            taskID: "task-2",
            runID: "run-2",
            stepID: "step-2",
            route: WorkflowRouteContext(
                available: false,
                taskClass: nil,
                provider: nil,
                modelKey: nil,
                taskClassSource: nil,
                routeSource: nil
            )
        )
        XCTAssertEqual(state.defaultApprovalDecisionActor(for: fallbackToRequestedBy), "actor.requester")

        let fallbackToSelectedPrincipal = ApprovalInboxItem(
            id: "approval-actor-3",
            taskTitle: "Review draft",
            decisionState: .pending,
            decisionOutcome: nil,
            riskLevel: .policy,
            riskRationale: "Supervisor approval required.",
            requestedAtLabel: "now",
            decidedAtLabel: nil,
            decisionByActorID: nil,
            decisionRationale: nil,
            requestedPhrase: nil,
            taskState: "awaiting_approval",
            runState: "awaiting_approval",
            stepName: "Review",
            stepCapabilityKey: nil,
            requestedByActorID: "unknown",
            subjectPrincipalActorID: "actor.subject",
            actingAsActorID: "unknown",
            taskID: "task-3",
            runID: "run-3",
            stepID: "step-3",
            route: WorkflowRouteContext(
                available: false,
                taskClass: nil,
                provider: nil,
                modelKey: nil,
                taskClassSource: nil,
                routeSource: nil
            )
        )
        XCTAssertEqual(state.defaultApprovalDecisionActor(for: fallbackToSelectedPrincipal), "actor.selected")
    }

    func testApprovalDecisionActorValidationRequiresKnownActorWhenDirectoryAvailable() {
        let state = AppShellState()
        state.identityPrincipalItems = [makePrincipal(id: "actor.approver")]

        XCTAssertEqual(
            state.approvalDecisionActorValidationMessage(actorID: ""),
            "Select `Decision By` before submitting."
        )
        XCTAssertEqual(
            state.approvalDecisionActorValidationMessage(actorID: "actor.missing"),
            "Selected decision actor `actor.missing` is not in the active workspace directory. Refresh Identity Directory in Configuration."
        )
        XCTAssertNil(state.approvalDecisionActorValidationMessage(actorID: "actor.approver"))
    }

    func testApprovalRequiredPhraseAndValidationUseRequestedPhraseFallback() {
        let state = AppShellState()
        let defaultPhraseItem = makeApprovalItem(taskID: "task-1", runID: "run-1")
        XCTAssertEqual(state.approvalRequiredPhrase(for: defaultPhraseItem), "GO AHEAD")

        let customPhraseItem = ApprovalInboxItem(
            id: "approval-custom-phrase",
            taskTitle: "Publish report",
            decisionState: .pending,
            decisionOutcome: nil,
            riskLevel: .policy,
            riskRationale: "Needs supervisor review.",
            requestedAtLabel: "now",
            decidedAtLabel: nil,
            decisionByActorID: nil,
            decisionRationale: nil,
            requestedPhrase: "APPROVE NOW",
            taskState: "awaiting_approval",
            runState: "awaiting_approval",
            stepName: "Publish",
            stepCapabilityKey: nil,
            requestedByActorID: "actor.requester",
            subjectPrincipalActorID: "actor.subject",
            actingAsActorID: "actor.subject",
            taskID: "task-custom",
            runID: "run-custom",
            stepID: "step-custom",
            route: WorkflowRouteContext(
                available: false,
                taskClass: nil,
                provider: nil,
                modelKey: nil,
                taskClassSource: nil,
                routeSource: nil
            )
        )

        XCTAssertEqual(state.approvalRequiredPhrase(for: customPhraseItem), "APPROVE NOW")
        XCTAssertNil(
            state.approvalApprovePhraseValidationMessage(
                phrase: "APPROVE NOW",
                item: customPhraseItem
            )
        )
        XCTAssertEqual(
            state.approvalApprovePhraseValidationMessage(
                phrase: "GO AHEAD",
                item: customPhraseItem
            ),
            "Approve requires exact phrase `APPROVE NOW`. Use `Use Required Phrase` or type it exactly."
        )
    }

    private func makeApprovalItem(
        taskID: String?,
        runID: String?,
        decisionState: ApprovalInboxDecisionState = .pending,
        decisionOutcome: ApprovalInboxDecisionOutcome? = nil,
        decisionRationale: String? = nil
    ) -> ApprovalInboxItem {
        let decidedAtLabel: String?
        switch decisionState {
        case .pending:
            decidedAtLabel = nil
        case .final:
            decidedAtLabel = "later"
        }

        return ApprovalInboxItem(
            id: "approval-1",
            taskTitle: "Delete file",
            decisionState: decisionState,
            decisionOutcome: decisionOutcome,
            riskLevel: .destructive,
            riskRationale: "Destructive action requires approval.",
            requestedAtLabel: "now",
            decidedAtLabel: decidedAtLabel,
            decisionByActorID: nil,
            decisionRationale: decisionRationale,
            requestedPhrase: "GO AHEAD",
            taskState: "awaiting_approval",
            runState: "awaiting_approval",
            stepName: "Delete",
            stepCapabilityKey: "finder_delete",
            requestedByActorID: "actor.requester",
            subjectPrincipalActorID: "actor.subject",
            actingAsActorID: "actor.subject",
            taskID: taskID,
            runID: runID,
            stepID: "step-1",
            route: WorkflowRouteContext(
                available: true,
                taskClass: "finder",
                provider: "ollama",
                modelKey: "llama3.2",
                taskClassSource: "step_capability",
                routeSource: "fallback_enabled"
            )
        )
    }

    private func makePrincipal(id: String) -> IdentityPrincipalItem {
        return IdentityPrincipalItem(
            id: id,
            displayName: id,
            actorType: "human",
            actorStatus: "active",
            principalStatus: "active",
            isActive: true,
            handles: []
        )
    }
}
