import XCTest
@testable import PersonalAgentUI

@MainActor
final class ChatTurnContextStoreTests: XCTestCase {
    func testResetForNewTurnClearsAllContextAndSetsWaitingStatus() {
        let store = ChatTurnContextStore()
        store.updateTraceabilityFromTurnResponse(
            DaemonChatTurnTaskRunCorrelation(
                available: true,
                source: "audit",
                taskID: "task-1",
                runID: "run-1",
                taskState: "completed",
                runState: "completed"
            ),
            fallbackCorrelationID: "corr-1",
            taskClass: "chat",
            provider: "openai",
            modelKey: "gpt-4.1",
            routeSource: "policy",
            channelID: "app",
            turnContractVersion: "chat_turn.v2",
            turnItemSchemaVersion: "chat_turn_item.v2",
            realtimeEventContractVersion: "chat_realtime_lifecycle.v2",
            items: [DaemonChatTurnItem(type: "assistant_message", content: "done")]
        )

        store.applyExplainabilitySuccess(
            decodeExplainabilityResponse(
                workspaceID: "ws1",
                selectedProvider: "openai",
                selectedModelKey: "gpt-4.1",
                selectedSource: "policy"
            ),
            defaultWorkspaceID: "ws1"
        )
        store.resetForNewTurn()

        XCTAssertNil(store.latestTurnTraceability)
        XCTAssertNil(store.latestTurnExplainability)
        XCTAssertNil(store.explainabilityErrorMessage)
        XCTAssertEqual(store.explainabilityStatusMessage, "Waiting for chat explainability.")
    }

    func testUpdateTraceabilityFromTurnResponseUsesTurnSignalsAndContractMetadata() throws {
        let store = ChatTurnContextStore()
        store.updateTraceabilityFromTurnResponse(
            DaemonChatTurnTaskRunCorrelation(
                available: true,
                source: "audit",
                taskID: "task-42",
                runID: "run-42",
                taskState: "running",
                runState: "awaiting_approval"
            ),
            fallbackCorrelationID: "corr-42",
            taskClass: "chat",
            provider: "openai",
            modelKey: "gpt-4.1",
            routeSource: "policy",
            channelID: "message",
            turnContractVersion: "chat_turn.v2",
            turnItemSchemaVersion: "chat_turn_item.v2",
            realtimeEventContractVersion: "chat_realtime_lifecycle.v2",
            items: [
                DaemonChatTurnItem(
                    type: "approval_request",
                    output: [
                        "approval_required": .bool(true),
                        "clarification_required": .bool(true),
                        "clarification_prompt": .string("Need final approval")
                    ],
                    approvalRequestID: "approval-123",
                    metadata: DaemonChatTurnItemMetadata(
                        responseShapingChannel: "message",
                        responseShapingProfile: "message.compact",
                        responseShapingGuardrailCount: 3,
                        responseShapingInstructionCount: 2,
                        personaPolicySource: "persisted"
                    )
                )
            ]
        )

        let traceability = try XCTUnwrap(store.latestTurnTraceability)
        XCTAssertEqual(traceability.taskID, "task-42")
        XCTAssertEqual(traceability.runID, "run-42")
        XCTAssertEqual(traceability.correlationID, "corr-42")
        XCTAssertEqual(traceability.turnContractVersion, "chat_turn.v2")
        XCTAssertEqual(traceability.turnItemSchemaVersion, "chat_turn_item.v2")
        XCTAssertEqual(traceability.realtimeEventContractVersion, "chat_realtime_lifecycle.v2")
        XCTAssertTrue(traceability.approvalRequired)
        XCTAssertEqual(traceability.approvalRequestID, "approval-123")
        XCTAssertTrue(traceability.clarificationRequired)
        XCTAssertEqual(traceability.clarificationPrompt, "Need final approval")
        XCTAssertEqual(traceability.responseShapingChannel, "message")
        XCTAssertEqual(traceability.responseShapingProfile, "message.compact")
        XCTAssertEqual(traceability.personaPolicySource, "persisted")
        XCTAssertEqual(traceability.responseShapingGuardrailCount, 3)
        XCTAssertEqual(traceability.responseShapingInstructionCount, 2)
    }

    func testMergeRealtimeLifecycleMetadataBackfillsAndPreservesTraceability() throws {
        let store = ChatTurnContextStore()
        store.updateTraceabilityFromTurnResponse(
            DaemonChatTurnTaskRunCorrelation(),
            fallbackCorrelationID: "corr-a",
            taskClass: "chat",
            provider: nil,
            modelKey: nil,
            routeSource: nil,
            channelID: "voice",
            turnContractVersion: "chat_turn.v2",
            turnItemSchemaVersion: "chat_turn_item.v2",
            realtimeEventContractVersion: "chat_realtime_lifecycle.v2",
            items: []
        )

        let merged = store.mergeRealtimeLifecycleContractMetadata(
            from: DaemonRealtimeEventEnvelope(
                eventID: "evt-1",
                sequence: 1,
                eventType: "turn_item_started",
                occurredAt: "2026-03-03T00:00:00Z",
                correlationID: "corr-b",
                contractVersion: "chat_realtime_lifecycle.v2",
                lifecycleSchemaVersion: "chat_realtime_item.v2",
                payload: DaemonRealtimeEventPayload()
            ),
            fallbackCorrelationID: "corr-fallback",
            routeProvider: "ollama",
            routeModelKey: "gpt-oss:20b",
            routeSource: "policy"
        )

        XCTAssertTrue(merged)
        let traceability = try XCTUnwrap(store.latestTurnTraceability)
        XCTAssertEqual(traceability.correlationID, "corr-a")
        XCTAssertEqual(traceability.provider, "ollama")
        XCTAssertEqual(traceability.modelKey, "gpt-oss:20b")
        XCTAssertEqual(traceability.routeSource, "policy")
        XCTAssertEqual(traceability.realtimeLifecycleContractVersion, "chat_realtime_lifecycle.v2")
        XCTAssertEqual(traceability.realtimeLifecycleSchemaVersion, "chat_realtime_item.v2")
        XCTAssertEqual(traceability.responseShapingChannel, "voice")
        XCTAssertEqual(traceability.responseShapingProfile, "voice.spoken")
    }

    func testExplainabilityLifecycleSupportsLoadingSuccessFailureAndMissingTokenState() {
        let store = ChatTurnContextStore()

        let started = store.markExplainabilityLoading(userInitiated: true, isAlreadyInFlight: false)
        XCTAssertTrue(started)
        XCTAssertEqual(store.explainabilityStatusMessage, "Loading chat explainability trace…")

        let alreadyLoading = store.markExplainabilityLoading(userInitiated: true, isAlreadyInFlight: true)
        XCTAssertFalse(alreadyLoading)
        XCTAssertEqual(store.explainabilityStatusMessage, "Chat explainability is already loading.")

        store.applyExplainabilitySuccess(
            decodeExplainabilityResponse(
                workspaceID: "ws2",
                selectedProvider: "openai",
                selectedModelKey: "gpt-4.1",
                selectedSource: "policy"
            ),
            defaultWorkspaceID: "ws-default"
        )
        XCTAssertEqual(store.latestTurnExplainability?.workspaceID, "ws2")
        XCTAssertEqual(store.explainabilityStatusMessage, "Loaded chat explainability for openai • gpt-4.1.")

        store.applyExplainabilityFailure(message: "Chat explainability failed", retainPreviousOnFailure: true)
        XCTAssertNotNil(store.latestTurnExplainability)
        XCTAssertEqual(store.explainabilityErrorMessage, "Chat explainability failed")
        XCTAssertEqual(store.explainabilityStatusMessage, "Chat explainability failed")

        store.clearExplainabilityForMissingToken()
        XCTAssertNil(store.latestTurnExplainability)
        XCTAssertEqual(store.explainabilityStatusMessage, "Set Assistant Access Token before loading chat explainability.")
    }

    private func decodeExplainabilityResponse(
        workspaceID: String,
        selectedProvider: String,
        selectedModelKey: String,
        selectedSource: String
    ) -> DaemonChatTurnExplainResponse {
        let json = """
        {
          "workspace_id": "\(workspaceID)",
          "task_class": "chat",
          "requested_by_actor_id": "actor.requester.ws1",
          "subject_actor_id": "actor.requester.ws1",
          "acting_as_actor_id": "actor.requester.ws1",
          "contract_version": "chat_turn_explain.v1",
          "selected_route": {
            "workspace_id": "\(workspaceID)",
            "task_class": "chat",
            "principal_actor_id": "actor.requester.ws1",
            "selected_provider": "\(selectedProvider)",
            "selected_model_key": "\(selectedModelKey)",
            "selected_source": "\(selectedSource)",
            "summary": "Selected route summary.",
            "explanations": ["Matched chat policy"],
            "reason_codes": ["policy_match"],
            "decisions": [],
            "fallback_chain": []
          },
          "tool_catalog": [
            {
              "name": "send_email",
              "description": "Send an email.",
              "capability_keys": ["connector.mail.send"],
              "input_schema": {"to": "string", "subject": "string"}
            }
          ],
          "policy_decisions": [
            {
              "tool_name": "send_email",
              "capability_key": "connector.mail.send",
              "decision": "allow",
              "reason": "Connector ready"
            }
          ]
        }
        """
        let data = Data(json.utf8)
        return try! JSONDecoder().decode(DaemonChatTurnExplainResponse.self, from: data)
    }
}
