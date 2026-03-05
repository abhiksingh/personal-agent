import XCTest
@testable import PersonalAgentUI

@MainActor
final class AppCommunicationsStoreTests: XCTestCase {
    func testCommunicationsContextReducersPersistByWorkspace() {
        let retentionStore = makeContextRetentionStore()
        let store = AppCommunicationsStore()

        let wsAFilter = CommunicationsFilterContext(
            searchText: "urgent",
            channelFilterID: "message",
            directionFilterRawValue: "inbound",
            threadFilterID: "thread-a",
            compactScanModeEnabled: true
        )
        store.setCommunicationsFilterContext(
            wsAFilter,
            workspaceID: "ws-a",
            contextRetentionStore: retentionStore
        )
        store.setCommunicationsTriageContext(
            CommunicationsTriageContext(
                handledThreadIDs: ["thread-a"],
                followUpThreadIDs: ["thread-b"],
                seenThreadIDs: ["thread-a", "thread-b"]
            ),
            workspaceID: "ws-a",
            contextRetentionStore: retentionStore
        )
        store.setCommunicationsComposeDraftContext(
            CommunicationsComposeDraftContext(
                isPresented: true,
                flowID: "reply",
                sourceChannel: "message",
                threadID: "thread-a",
                connectorID: "imessage",
                destination: "person@example.com",
                message: "Follow-up draft"
            ),
            workspaceID: "ws-a",
            contextRetentionStore: retentionStore
        )

        XCTAssertEqual(
            store.communicationsFilterContext(
                workspaceID: "ws-a",
                contextRetentionStore: retentionStore
            ),
            wsAFilter
        )
        XCTAssertEqual(
            store.communicationsTriageContext(
                workspaceID: "ws-a",
                contextRetentionStore: retentionStore
            ).handledThreadIDs,
            ["thread-a"]
        )
        XCTAssertEqual(
            store.communicationsComposeDraftContext(
                workspaceID: "ws-a",
                contextRetentionStore: retentionStore
            )?.destination,
            "person@example.com"
        )

        XCTAssertEqual(
            store.communicationsFilterContext(
                workspaceID: "ws-b",
                contextRetentionStore: retentionStore
            ),
            CommunicationsFilterContext()
        )
    }

    func testThreadSelectionFilterContextResetsChannelDirectionAndPreservesDensityMode() {
        let store = AppCommunicationsStore()
        let current = CommunicationsFilterContext(
            searchText: "old",
            channelFilterID: "voice",
            directionFilterRawValue: "outbound",
            threadFilterID: "thread-old",
            compactScanModeEnabled: true
        )

        let updated = store.threadSelectionFilterContext(
            threadID: "thread-42",
            currentContext: current
        )

        XCTAssertEqual(updated.searchText, "thread-42")
        XCTAssertEqual(updated.threadFilterID, "thread-42")
        XCTAssertEqual(updated.channelFilterID, CommunicationsFilterContext.allChannelsID)
        XCTAssertEqual(updated.directionFilterRawValue, "all")
        XCTAssertTrue(updated.compactScanModeEnabled)
    }

    func testCommunicationAttemptContextThreadSelectionNormalizesWhitespace() {
        let store = AppCommunicationsStore()

        XCTAssertEqual(store.setCommunicationAttemptContextThreadID("  thread-1  "), "thread-1")
        XCTAssertEqual(store.communicationAttemptContextThreadID, "thread-1")

        XCTAssertNil(store.setCommunicationAttemptContextThreadID("   "))
        XCTAssertNil(store.communicationAttemptContextThreadID)
    }

    func testContinuityReducersMapLatestItemsAndSortByTimestampDescending() throws {
        let store = AppCommunicationsStore()
        let records = try decodeHistoryRecords(
            """
            {
              "workspace_id": "ws1",
              "items": [
                {
                  "record_id": "turn-1-item-0",
                  "turn_id": "turn-1",
                  "workspace_id": "ws1",
                  "task_class": "chat",
                  "correlation_id": "corr-1",
                  "channel_id": "twilio_sms",
                  "connector_id": "twilio",
                  "thread_id": "thread-1",
                  "item_index": 0,
                  "item": { "type": "user_message", "content": "Send a status update" },
                  "task_run_reference": { "task_id": "task-1", "run_id": "run-1", "run_state": "completed" },
                  "created_at": "2026-03-04T10:00:00Z"
                },
                {
                  "record_id": "turn-1-item-1",
                  "turn_id": "turn-1",
                  "workspace_id": "ws1",
                  "task_class": "chat",
                  "correlation_id": "corr-1",
                  "channel_id": "twilio_sms",
                  "connector_id": "twilio",
                  "thread_id": "thread-1",
                  "item_index": 1,
                  "item": {
                    "type": "assistant_message",
                    "content": "Draft created and sent.",
                    "metadata": { "response_shaping_channel": "message" }
                  },
                  "task_run_reference": { "task_id": "task-1", "run_id": "run-1", "run_state": "completed" },
                  "created_at": "2026-03-04T10:00:01Z"
                },
                {
                  "record_id": "turn-2-item-0",
                  "turn_id": "turn-2",
                  "workspace_id": "ws1",
                  "task_class": "chat",
                  "correlation_id": "corr-2",
                  "channel_id": "voice",
                  "connector_id": "twilio",
                  "thread_id": "thread-2",
                  "item_index": 0,
                  "item": {
                    "type": "tool_result",
                    "status": "failed",
                    "tool_name": "browse_web",
                    "error_message": "Browse target rejected request."
                  },
                  "task_run_reference": { "task_id": "task-2", "run_id": "run-2", "run_state": "failed" },
                  "created_at": "2026-03-04T11:00:00Z"
                }
              ],
              "has_more": false
            }
            """
        )

        let mapped = store.mapCommunicationContinuityRecords(
            records,
            workspaceID: "ws1",
            logicalCommunicationChannelID: { raw in
                switch raw.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
                case "twilio_sms", "message":
                    return "message"
                case "voice":
                    return "voice"
                default:
                    return raw
                }
            },
            parseDaemonTimestamp: { raw in
                ISO8601DateFormatter().date(from: raw)
            },
            formattedWorkflowTimestamp: { $0 },
            truncateText: { value, limit in
                guard value.count > limit else {
                    return value
                }
                let endIndex = value.index(value.startIndex, offsetBy: limit)
                return "\(value[..<endIndex])…"
            }
        )

        XCTAssertEqual(mapped.count, 2)
        XCTAssertEqual(mapped.first?.turnID, "turn-2")
        XCTAssertEqual(mapped.first?.channel, "voice")
        XCTAssertEqual(mapped.first?.summary, "Browse target rejected request.")
        XCTAssertEqual(mapped[1].turnID, "turn-1")
        XCTAssertEqual(mapped[1].channel, "message")
        XCTAssertEqual(mapped[1].summary, "Draft created and sent.")
        XCTAssertEqual(mapped[1].responseShapingChannel, "message")
        XCTAssertEqual(mapped[1].responseShapingProfile, "message.compact")
    }

    private func decodeHistoryRecords(_ payload: String) throws -> [DaemonChatTurnHistoryRecord] {
        let response = try JSONDecoder().decode(
            DaemonChatTurnHistoryResponse.self,
            from: Data(payload.utf8)
        )
        return response.items
    }

    private func makeContextRetentionStore() -> AppContextRetentionStore {
        let suffix = UUID().uuidString.lowercased()
        return AppContextRetentionStore(
            userDefaults: appShellStateTestUserDefaults(),
            defaultWorkspaceID: "ws1",
            canonicalWorkspaceID: Self.canonicalWorkspaceID,
            panelFilterContextDefaultsKey: "test.u257.panel_filter.\(suffix)",
            communicationsTriageDefaultsKey: "test.u257.triage.\(suffix)",
            workspaceContinuityDefaultsKey: "test.u257.continuity.\(suffix)",
            informationDensityModeDefaultsKey: "test.u257.density.\(suffix)",
            homeFirstSessionProgressDefaultsKey: "test.u257.home_progress.\(suffix)"
        )
    }

    private static func canonicalWorkspaceID(_ raw: String?, _ fallbackToDefault: Bool) -> String? {
        let trimmed = raw?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if trimmed.isEmpty {
            return fallbackToDefault ? "ws1" : nil
        }
        return trimmed
    }
}
