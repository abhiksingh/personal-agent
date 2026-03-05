import XCTest
@testable import PersonalAgentUI

@MainActor
final class AppConnectionConfigStoreTests: XCTestCase {
    func testConfigurationMutationPayloadParsesExpectedKinds() throws {
        let store = AppConnectionConfigStore()
        let draft: [String: String] = [
            "mode": "daemon",
            "retry_count": "3",
            "enabled": "true",
            "optional_note": "null",
        ]
        let kindMap: [String: ConfigurationDraftValueKind] = [
            "mode": .string,
            "retry_count": .number,
            "enabled": .bool,
            "optional_note": .null,
        ]

        let payload = try store.configurationMutationPayload(draft: draft, kindMap: kindMap)

        assertMutationValue(payload["mode"], expectedString: "daemon")
        assertMutationValue(payload["retry_count"], expectedNumber: 3)
        assertMutationValue(payload["enabled"], expectedBool: true)
        assertMutationValue(payload["optional_note"], expectedNull: true)
    }

    func testConfigurationMutationPayloadReturnsValidationMessageForInvalidNumber() {
        let store = AppConnectionConfigStore()
        let draft = ["retry_count": "not-a-number"]
        let kindMap: [String: ConfigurationDraftValueKind] = ["retry_count": .number]

        do {
            _ = try store.configurationMutationPayload(draft: draft, kindMap: kindMap)
            XCTFail("Expected payload validation failure.")
        } catch let error as AppConnectionConfigStore.ConfigurationMutationValidationError {
            XCTAssertEqual(error.message, "Configuration field retry_count expects a number.")
        } catch {
            XCTFail("Unexpected error: \(error)")
        }
    }

    func testSynchronizeChannelConfigurationDraftsMergesExistingAndPrunesStaleState() {
        let store = AppConnectionConfigStore()
        store.channelConfigDraftByID = [
            "app_chat": ["custom_threshold": "5"],
            "stale_channel": ["legacy": "true"],
        ]
        store.channelConfigKindsByID = [
            "app_chat": ["custom_threshold": .number],
            "stale_channel": ["legacy": .bool],
        ]
        store.channelConfigActionStatusByID = [
            "app_chat": "Saved.",
            "stale_channel": "Stale",
        ]
        store.channelLastTestResultByID = [
            "stale_channel": ConfigurationTestResultItem(
                operation: "health",
                success: false,
                status: "failed",
                summary: "stale",
                checkedAtLabel: "n/a",
                details: [:]
            )
        ]
        store.channelConfigSaveInFlightIDs = ["app_chat", "stale_channel"]
        store.channelTestInFlightIDs = ["stale_channel"]

        store.synchronizeChannelConfigurationDrafts(
            with: [
                makeChannelCard(
                    id: "app_chat",
                    editableConfiguration: ["mode": "daemon"],
                    editableKinds: ["mode": .string]
                )
            ]
        )

        XCTAssertEqual(store.channelConfigDraftByID["app_chat"]?["mode"], "daemon")
        XCTAssertEqual(store.channelConfigDraftByID["app_chat"]?["custom_threshold"], "5")
        XCTAssertEqual(store.channelConfigKindsByID["app_chat"]?["custom_threshold"], .number)
        XCTAssertNil(store.channelConfigDraftByID["stale_channel"])
        XCTAssertNil(store.channelConfigActionStatusByID["stale_channel"])
        XCTAssertFalse(store.channelConfigSaveInFlightIDs.contains("stale_channel"))
        XCTAssertFalse(store.channelTestInFlightIDs.contains("stale_channel"))
    }

    func testSynchronizeChannelConnectorMappingDraftsPrunesStaleAndUsesMergedSource() {
        let store = AppConnectionConfigStore()
        store.channelConnectorMappingActionStatusByChannelID = [
            "app": "Ready",
            "stale": "Stale",
        ]
        store.channelConnectorMappingSaveInFlightChannelIDs = ["stale", "app"]

        let observed: [String: [ChannelConnectorMappingItem]] = [
            "app": [
                makeMapping(channelID: "app", connectorID: "mail", enabled: true, priority: 1)
            ]
        ]

        store.synchronizeChannelConnectorMappingDrafts(
            with: [makeChannelCard(id: "app_chat", editableConfiguration: [:], editableKinds: [:])],
            mappingsByChannelID: observed,
            normalizeChannelID: { raw in
                raw == "app_chat" ? "app" : raw
            },
            inferredMappingsByLogicalChannelID: { _ in
                ["app": [makeMapping(channelID: "app", connectorID: "finder", enabled: false, priority: 2)]]
            },
            mergeMappings: { observedMappings, inferredMappings, _ in
                (observedMappings + inferredMappings).sorted { $0.priority < $1.priority }
            }
        )

        XCTAssertEqual(store.channelConnectorMappingsByChannelID.keys.sorted(), ["app"])
        XCTAssertEqual(store.channelConnectorMappingDraftByChannelID["app"]?.count, 2)
        XCTAssertNil(store.channelConnectorMappingActionStatusByChannelID["stale"])
        XCTAssertFalse(store.channelConnectorMappingSaveInFlightChannelIDs.contains("stale"))
    }

    func testReorderChannelConnectorMappingRebalancesPrioritiesAndSetsStatus() {
        let store = AppConnectionConfigStore()
        store.channelConnectorMappingDraftByChannelID["app"] = [
            makeMapping(channelID: "app", connectorID: "mail", enabled: true, priority: 1),
            makeMapping(channelID: "app", connectorID: "finder", enabled: true, priority: 2),
        ]

        store.reorderChannelConnectorMapping(
            channelID: "app",
            connectorID: "mail",
            direction: 1,
            normalizeChannelID: { $0 },
            normalizeConnectorID: { $0 },
            sortedMappings: { $0.sorted { $0.priority < $1.priority } },
            rebalanceMappings: { mappings in
                mappings.enumerated().map { index, item in
                    var mutable = item
                    mutable.priority = index + 1
                    return mutable
                }
            },
            connectorDisplayName: { $0.capitalized }
        )

        XCTAssertEqual(
            store.channelConnectorMappingDraftByChannelID["app"]?.map(\.connectorID),
            ["finder", "mail"]
        )
        XCTAssertEqual(
            store.channelConnectorMappingDraftByChannelID["app"]?.map(\.priority),
            [1, 2]
        )
        XCTAssertEqual(
            store.channelConnectorMappingActionStatusByChannelID["app"],
            "Updated Mail priority to 2."
        )
    }

    func testMapChannelConfigurationTestResultNormalizesDetails() {
        let store = AppConnectionConfigStore()
        let response = DaemonChannelTestOperationResponse(
            workspaceID: "ws1",
            channelID: "app_chat",
            operation: "health",
            success: true,
            status: "ready",
            summary: "All checks passed.",
            checkedAt: "2026-03-04T12:00:00Z",
            details: DaemonUIStatusTestOperationDetails(
                pluginID: "chat-worker",
                workerRegistered: true,
                additional: ["round_trip_ms": .number(18)]
            )
        )

        let result = store.mapConfigurationTestResult(
            from: response,
            formattedWorkflowTimestamp: { _ in "formatted-ts" }
        )

        XCTAssertEqual(result.operation, "health")
        XCTAssertTrue(result.success)
        XCTAssertEqual(result.status, "ready")
        XCTAssertEqual(result.checkedAtLabel, "formatted-ts")
        XCTAssertEqual(result.details["plugin_id"], "chat-worker")
        XCTAssertEqual(result.details["worker_registered"], "true")
        XCTAssertEqual(result.details["round_trip_ms"], "18")
    }

    private func makeChannelCard(
        id: String,
        editableConfiguration: [String: String],
        editableKinds: [String: ConfigurationDraftValueKind]
    ) -> ChannelCardItem {
        ChannelCardItem(
            id: id,
            name: id,
            status: .active,
            summary: "Ready",
            details: [:],
            editableConfiguration: editableConfiguration,
            editableConfigurationKinds: editableKinds,
            readOnlyConfiguration: [:],
            actions: [],
            unavailableActionReason: "n/a"
        )
    }

    private func makeMapping(
        channelID: String,
        connectorID: String,
        enabled: Bool,
        priority: Int
    ) -> ChannelConnectorMappingItem {
        ChannelConnectorMappingItem(
            channelID: channelID,
            connectorID: connectorID,
            enabled: enabled,
            priority: priority,
            capabilities: [],
            createdAtLabel: nil,
            updatedAtLabel: nil
        )
    }

    private func assertMutationValue(_ value: DaemonConfigMutationValue?, expectedString: String) {
        guard case .string(let actual)? = value else {
            XCTFail("Expected string mutation value.")
            return
        }
        XCTAssertEqual(actual, expectedString)
    }

    private func assertMutationValue(_ value: DaemonConfigMutationValue?, expectedNumber: Double) {
        guard case .number(let actual)? = value else {
            XCTFail("Expected number mutation value.")
            return
        }
        XCTAssertEqual(actual, expectedNumber, accuracy: 0.0001)
    }

    private func assertMutationValue(_ value: DaemonConfigMutationValue?, expectedBool: Bool) {
        guard case .bool(let actual)? = value else {
            XCTFail("Expected bool mutation value.")
            return
        }
        XCTAssertEqual(actual, expectedBool)
    }

    private func assertMutationValue(_ value: DaemonConfigMutationValue?, expectedNull: Bool) {
        guard case .null? = value else {
            XCTFail("Expected null mutation value.")
            return
        }
        XCTAssertTrue(expectedNull)
    }
}
