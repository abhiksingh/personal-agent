import XCTest
@testable import PersonalAgentUI

@MainActor
final class AppShellStateChannelConnectorConfigTests: XCTestCase {
    func testChannelCardDefaultsToCollapsedState() {
        let card = ChannelCardItem(
            id: "app_chat",
            name: "App Chat",
            status: .active,
            summary: "Ready",
            details: [:],
            editableConfiguration: [:],
            editableConfigurationKinds: [:],
            readOnlyConfiguration: [:],
            actions: [],
            unavailableActionReason: "n/a"
        )

        XCTAssertFalse(card.isExpanded)
    }

    func testConnectorCardDefaultsToCollapsedState() {
        let card = ConnectorCardItem(
            id: "mail",
            name: "Mail",
            health: .ready,
            permissionState: .granted,
            permissionScope: "Mail data access",
            summary: "Ready",
            details: [:],
            editableConfiguration: [:],
            editableConfigurationKinds: [:],
            readOnlyConfiguration: [:],
            actions: [],
            unavailableActionReason: "n/a"
        )

        XCTAssertFalse(card.isExpanded)
    }

    func testSaveChannelConfigurationWithoutTokenSetsDeterministicStatus() async {
        let state = AppShellState()
        state.clearLocalDevToken()

        state.saveChannelConfiguration(channelID: "app_chat")
        try? await Task.sleep(for: .milliseconds(120))

        XCTAssertEqual(
            state.channelConfigActionStatusByID["app_chat"],
            "Set Assistant Access Token before saving channel configuration."
        )
    }

    func testRunConnectorHealthCheckWithoutTokenSetsDeterministicStatus() async {
        let state = AppShellState()
        state.clearLocalDevToken()

        state.runConnectorHealthCheck(connectorID: "mail")
        try? await Task.sleep(for: .milliseconds(120))

        XCTAssertEqual(
            state.connectorConfigActionStatusByID["mail"],
            "Set Assistant Access Token before running connector health check."
        )
    }

    func testSaveChannelDeliveryPolicyWithoutTokenSetsDeterministicStatus() async {
        let state = AppShellState()
        state.clearLocalDevToken()

        state.saveChannelDeliveryPolicy(channelID: "app_chat")
        try? await Task.sleep(for: .milliseconds(120))

        XCTAssertEqual(
            state.channelDeliveryPolicyActionStatusByID["app"],
            "Set Assistant Access Token before saving delivery policy."
        )
    }

    func testChannelDeliveryPolicyDraftCanonicalizesLegacyChannelIdentifier() {
        let state = AppShellState()

        state.startNewChannelDeliveryPolicyDraft(channelID: "twilio_sms")

        XCTAssertNotNil(state.channelDeliveryPolicyDraftByID["message"])
        XCTAssertNil(state.channelDeliveryPolicyDraftByID["twilio_sms"])
        XCTAssertEqual(state.channelDeliveryPolicyDraft(channelID: "twilio_sms").primaryChannel, "message")
    }

    func testChannelDeliveryRouteOptionsCanonicalizeLegacyChannelAliases() {
        let state = AppShellState()
        state.channelCards = [
            makeChannelCard(id: "app_chat", name: "App Chat", status: .active),
            makeChannelCard(id: "imessage_sms_bridge", name: "iMessage Bridge", status: .active),
            makeChannelCard(id: "twilio_voice", name: "Twilio Voice", status: .active)
        ]
        state.channelDeliveryPoliciesByChannelID["message"] = [
            ChannelDeliveryPolicyItem(
                id: "policy-message",
                workspaceID: "ws1",
                sourceChannel: "message",
                endpointPattern: nil,
                isDefault: true,
                primaryChannel: "imessage_sms",
                retryCount: 1,
                fallbackChannels: ["twilio_sms", "app_chat"],
                createdAtLabel: "n/a",
                updatedAtLabel: "n/a",
                sortTimestamp: Date.now
            )
        ]

        let options = state.channelDeliveryRouteOptions(channelID: "imessage_sms_bridge")

        XCTAssertEqual(options, ["app", "message", "voice"])
    }

    func testAddChannelConfigDraftFieldInfersBoolKind() {
        let state = AppShellState()

        state.addChannelConfigDraftField(channelID: "app_chat", key: "enabled", value: "true")

        XCTAssertEqual(state.channelConfigDraftValue(channelID: "app_chat", key: "enabled"), "true")
        XCTAssertEqual(state.channelConfigDraftKind(channelID: "app_chat", key: "enabled"), .bool)
    }

    func testChannelAdvancedConfigDraftKeysExcludeGuidedDescriptorFields() {
        let state = AppShellState()
        let descriptor = ConfigurationFieldDescriptorItem(
            key: "mode",
            label: "Mode",
            required: true,
            enumOptions: ["daemon", "local"],
            editable: true,
            secret: false,
            writeOnly: false,
            helpText: "Dispatch mode.",
            draftKind: .string
        )
        state.channelCards = [
            ChannelCardItem(
                id: "app_chat",
                name: "App Chat",
                status: .active,
                summary: "Ready",
                details: [:],
                editableConfiguration: ["mode": "daemon", "legacy_hint": "true"],
                editableConfigurationKinds: ["mode": .string, "legacy_hint": .bool],
                configurationFieldDescriptors: [descriptor],
                readOnlyConfiguration: [:],
                actions: [],
                unavailableActionReason: "n/a"
            )
        ]
        state.channelConfigDraftByID["app_chat"] = ["mode": "daemon", "legacy_hint": "true"]

        XCTAssertEqual(state.channelGuidedConfigFieldKeys(channelID: "app_chat"), ["mode"])
        XCTAssertEqual(state.channelAdvancedConfigDraftKeys(channelID: "app_chat"), ["legacy_hint"])
    }

    func testChannelGuidedConfigFieldKeysSynthesizeDescriptorsWhenDaemonDescriptorsMissing() {
        let state = AppShellState()
        state.channelCards = [
            ChannelCardItem(
                id: "message",
                name: "Message",
                status: .active,
                summary: "Ready",
                details: [:],
                editableConfiguration: [
                    "enabled": "true",
                    "retry_count": "2"
                ],
                editableConfigurationKinds: [
                    "enabled": .bool,
                    "retry_count": .number
                ],
                readOnlyConfiguration: [:],
                actions: [],
                unavailableActionReason: "n/a"
            )
        ]
        state.channelConfigDraftByID["message"] = [
            "enabled": "true",
            "retry_count": "2",
            "custom_override": "on"
        ]

        let descriptors = state.channelConfigFieldDescriptors(channelID: "message")

        XCTAssertEqual(descriptors.map(\.key), ["enabled", "retry_count"])
        XCTAssertEqual(descriptors.first(where: { $0.key == "enabled" })?.draftKind, .bool)
        XCTAssertEqual(descriptors.first(where: { $0.key == "retry_count" })?.draftKind, .number)
        XCTAssertEqual(state.channelGuidedConfigFieldKeys(channelID: "message"), ["enabled", "retry_count"])
        XCTAssertEqual(state.channelAdvancedConfigDraftKeys(channelID: "message"), ["custom_override"])
    }

    func testRemoveChannelConfigDraftFieldClearsDraftValue() {
        let state = AppShellState()
        state.addChannelConfigDraftField(channelID: "app_chat", key: "mode", value: "daemon")

        state.removeChannelConfigDraftField(channelID: "app_chat", key: "mode")

        XCTAssertEqual(state.channelConfigDraftValue(channelID: "app_chat", key: "mode"), "")
        XCTAssertEqual(state.channelConfigDraftKind(channelID: "app_chat", key: "mode"), .string)
    }

    func testChannelConfigHasDraftChangesTracksDiffAgainstCardBaseline() {
        let state = AppShellState()
        state.channelCards = [
            ChannelCardItem(
                id: "app_chat",
                name: "App Chat",
                status: .active,
                summary: "Ready",
                details: [:],
                editableConfiguration: ["mode": "daemon"],
                editableConfigurationKinds: ["mode": .string],
                readOnlyConfiguration: [:],
                actions: [],
                unavailableActionReason: "n/a"
            )
        ]
        state.channelConfigDraftByID["app_chat"] = ["mode": "daemon"]
        XCTAssertFalse(state.channelConfigHasDraftChanges(channelID: "app_chat"))

        state.channelConfigDraftByID["app_chat"] = ["mode": "manual"]
        XCTAssertTrue(state.channelConfigHasDraftChanges(channelID: "app_chat"))
    }

    func testAddConnectorConfigDraftFieldRequiresNonEmptyKey() {
        let state = AppShellState()

        state.addConnectorConfigDraftField(connectorID: "mail", key: "   ", value: "inbox")

        XCTAssertEqual(state.connectorConfigDraftKeys(connectorID: "mail"), [])
        XCTAssertEqual(state.connectorConfigActionStatusByID["mail"], "Configuration key is required.")
    }

    func testSaveConnectorConfigurationMissingRequiredDescriptorSetsDeterministicStatus() async {
        let state = AppShellState()
        let descriptor = ConfigurationFieldDescriptorItem(
            key: "account_sid",
            label: "Account SID",
            required: true,
            enumOptions: [],
            editable: true,
            secret: false,
            writeOnly: false,
            helpText: "Twilio account SID.",
            draftKind: .string
        )
        state.connectorCards = [
            ConnectorCardItem(
                id: "twilio",
                name: "Twilio",
                health: .unavailable,
                permissionState: .granted,
                permissionScope: "Twilio",
                statusReason: "missing_config",
                summary: "Missing setup",
                details: [:],
                editableConfiguration: [:],
                editableConfigurationKinds: [:],
                configurationFieldDescriptors: [descriptor],
                readOnlyConfiguration: [:],
                actions: [],
                unavailableActionReason: "n/a"
            )
        ]
        state.connectorConfigDraftByID["twilio"] = [:]
        state.localDevTokenInput = "test-token"
        state.saveLocalDevToken()

        state.saveConnectorConfiguration(connectorID: "twilio")
        try? await Task.sleep(for: .milliseconds(120))

        XCTAssertEqual(
            state.connectorConfigActionStatusByID["twilio"],
            "Connector configuration field `Account SID` is required."
        )
    }

    func testConnectorGuidedConfigFieldKeysSynthesizeDescriptorsWhenDaemonDescriptorsMissing() {
        let state = AppShellState()
        state.connectorCards = [
            ConnectorCardItem(
                id: "twilio",
                name: "Twilio",
                health: .unavailable,
                permissionState: .granted,
                permissionScope: "Twilio",
                statusReason: "missing_config",
                summary: "Missing setup",
                details: [:],
                editableConfiguration: [
                    "sms_enabled": "true",
                    "send_rate_limit": "10"
                ],
                editableConfigurationKinds: [
                    "sms_enabled": .bool,
                    "send_rate_limit": .number
                ],
                readOnlyConfiguration: [:],
                actions: [],
                unavailableActionReason: "n/a"
            )
        ]
        state.connectorConfigDraftByID["twilio"] = [
            "sms_enabled": "true",
            "send_rate_limit": "10",
            "custom_override": "value"
        ]

        let descriptors = state.connectorConfigFieldDescriptors(connectorID: "twilio")

        XCTAssertEqual(descriptors.map(\.key), ["send_rate_limit", "sms_enabled"])
        XCTAssertEqual(descriptors.first(where: { $0.key == "sms_enabled" })?.draftKind, .bool)
        XCTAssertEqual(descriptors.first(where: { $0.key == "send_rate_limit" })?.draftKind, .number)
        XCTAssertEqual(state.connectorGuidedConfigFieldKeys(connectorID: "twilio"), ["send_rate_limit", "sms_enabled"])
        XCTAssertEqual(state.connectorAdvancedConfigDraftKeys(connectorID: "twilio"), ["custom_override"])
    }

    func testResetConnectorConfigDraftUsesCardEditableConfiguration() {
        let state = AppShellState()
        state.connectorCards = [
            ConnectorCardItem(
                id: "mail",
                name: "Mail",
                health: .ready,
                permissionState: .granted,
                permissionScope: "Mail data access",
                statusReason: nil,
                summary: "Ready",
                details: ["Connector ID": "mail"],
                editableConfiguration: [
                    "scope": "inbox",
                    "enabled": "true"
                ],
                editableConfigurationKinds: [
                    "scope": .string,
                    "enabled": .bool
                ],
                readOnlyConfiguration: [
                    "metadata": "owner=ops"
                ],
                actions: [],
                unavailableActionReason: "No actions"
            )
        ]

        state.setConnectorConfigDraftValue(connectorID: "mail", key: "scope", value: "archive")
        state.resetConnectorConfigDraft(connectorID: "mail")

        XCTAssertEqual(state.connectorConfigDraftValue(connectorID: "mail", key: "scope"), "inbox")
        XCTAssertEqual(state.connectorConfigDraftValue(connectorID: "mail", key: "enabled"), "true")
        XCTAssertEqual(state.connectorConfigDraftKind(connectorID: "mail", key: "enabled"), .bool)
    }

    func testConnectorConfigHasDraftChangesTracksDiffAgainstCardBaseline() {
        let state = AppShellState()
        state.connectorCards = [
            makeConnectorCard(
                id: "mail",
                name: "Mail",
                health: .ready,
                permissionState: .granted,
                permissionScope: "Mail access",
                statusReason: nil,
                capabilities: "mail",
                actions: []
            )
        ]
        state.connectorConfigDraftByID["mail"] = [:]
        XCTAssertFalse(state.connectorConfigHasDraftChanges(connectorID: "mail"))

        state.connectorConfigDraftByID["mail"] = ["scope": "inbox"]
        XCTAssertTrue(state.connectorConfigHasDraftChanges(connectorID: "mail"))
    }

    func testProviderSetupHasDraftChangesTracksDraftsAndSecretValue() {
        let state = AppShellState()

        XCTAssertFalse(state.providerSetupHasDraftChanges(providerID: "openai"))

        state.setProviderEndpointDraft("https://example.invalid", for: "openai")
        XCTAssertTrue(state.providerSetupHasDraftChanges(providerID: "openai"))

        state.discardDraftChanges(for: .models)
        XCTAssertFalse(state.providerSetupHasDraftChanges(providerID: "openai"))

        state.setProviderSecretValueDraft("top-secret", for: "openai")
        XCTAssertTrue(state.providerSetupHasDraftChanges(providerID: "openai"))
    }

    func testLogicalChannelCardsGroupLegacyChannelCardsIntoAppMessageVoice() {
        let state = AppShellState()
        state.channelCards = [
            makeChannelCard(id: "app_chat", name: "App Chat", status: .active),
            makeChannelCard(id: "imessage_sms", name: "iMessage/SMS Bridge", status: .degraded),
            makeChannelCard(id: "twilio_sms", name: "Twilio SMS", status: .active),
            makeChannelCard(id: "twilio_voice", name: "Twilio Voice", status: .setupRequired)
        ]

        let logicalCards = state.logicalChannelCards

        XCTAssertEqual(logicalCards.count, 3)

        let app = try! XCTUnwrap(logicalCards.first(where: { $0.channelID == "app" }))
        XCTAssertEqual(app.primaryChannelCardID, "app_chat")
        XCTAssertEqual(app.channelCardIDs, ["app_chat"])
        XCTAssertEqual(app.status, .active)

        let message = try! XCTUnwrap(logicalCards.first(where: { $0.channelID == "message" }))
        XCTAssertEqual(message.primaryChannelCardID, "imessage_sms")
        XCTAssertEqual(Set(message.channelCardIDs), Set(["imessage_sms", "twilio_sms"]))
        XCTAssertEqual(message.status, .degraded)
        XCTAssertEqual(message.details["Logical Channel"], "Message")
        XCTAssertEqual(message.details["Mapped Channel IDs"], "imessage_sms, twilio_sms")

        let voice = try! XCTUnwrap(logicalCards.first(where: { $0.channelID == "voice" }))
        XCTAssertEqual(voice.primaryChannelCardID, "twilio_voice")
        XCTAssertEqual(voice.channelCardIDs, ["twilio_voice"])
        XCTAssertEqual(voice.status, .setupRequired)
    }

    func testLogicalChannelCardsRollUpMappedConnectorReasonsAndActions() {
        let state = AppShellState()
        state.channelCards = [
            makeChannelCard(id: "imessage_sms", name: "iMessage/SMS Bridge", status: .active),
            makeChannelCard(id: "twilio_sms", name: "Twilio SMS", status: .active),
            makeChannelCard(id: "twilio_voice", name: "Twilio Voice", status: .active)
        ]
        state.channelConnectorMappingsByChannelID["message"] = [
            makeChannelConnectorMapping(channelID: "message", connectorID: "imessage", enabled: true, priority: 1),
            makeChannelConnectorMapping(channelID: "message", connectorID: "twilio", enabled: true, priority: 2)
        ]
        state.channelConnectorMappingsByChannelID["voice"] = [
            makeChannelConnectorMapping(channelID: "voice", connectorID: "twilio", enabled: true, priority: 1)
        ]
        state.connectorCards = [
            makeConnectorCard(
                id: "messages",
                name: "Messages",
                health: .needsPermission,
                permissionScope: "Messages access",
                statusReason: "Accessibility not granted",
                capabilities: "imessage,sms",
                actions: [
                    makeAction(id: "request_permission", title: "Request Permission"),
                    makeAction(id: "open_system_settings", title: "Open System Settings")
                ]
            ),
            makeConnectorCard(
                id: "twilio",
                name: "Twilio",
                health: .unavailable,
                permissionScope: "Twilio access",
                statusReason: "Missing account SID",
                capabilities: "sms,voice",
                actions: [
                    makeAction(id: "open_system_settings", title: "Open System Settings"),
                    makeAction(id: "check_connector", title: "Run Connector Check")
                ]
            )
        ]

        let logicalCards = state.logicalChannelCards
        let message = try! XCTUnwrap(logicalCards.first(where: { $0.channelID == "message" }))
        let voice = try! XCTUnwrap(logicalCards.first(where: { $0.channelID == "voice" }))

        XCTAssertEqual(
            Set(message.mappedConnectorRollups.map(\.connectorID)),
            Set(["imessage", "twilio"])
        )
        XCTAssertEqual(Set(voice.mappedConnectorRollups.map(\.connectorID)), Set(["twilio"]))

        XCTAssertEqual(
            Set(message.connectorActionTitles),
            Set(["Request Permission", "Open System Settings", "Run Connector Check"])
        )

        let messageReasonSummary = try! XCTUnwrap(message.connectorReasonSummary)
        XCTAssertTrue(messageReasonSummary.contains("Messages: Accessibility not granted"))
        XCTAssertTrue(messageReasonSummary.contains("Twilio: Missing account SID"))
    }

    func testLogicalChannelCardsSelectPrimaryImplementationFromMappingPriority() {
        let state = AppShellState()
        state.channelCards = [
            makeChannelCard(id: "imessage_sms", name: "iMessage/SMS Bridge", status: .active),
            makeChannelCard(id: "twilio_sms", name: "Twilio SMS", status: .active)
        ]
        state.channelConnectorMappingsByChannelID["message"] = [
            makeChannelConnectorMapping(channelID: "message", connectorID: "twilio", enabled: true, priority: 1),
            makeChannelConnectorMapping(channelID: "message", connectorID: "imessage", enabled: true, priority: 2)
        ]

        let message = try! XCTUnwrap(state.logicalChannelCards.first(where: { $0.channelID == "message" }))
        XCTAssertEqual(message.primaryChannelCardID, "twilio_sms")
    }

    func testLogicalChannelCardsConnectorRollupsFollowCurrentWorkspaceMappings() {
        let state = AppShellState()
        state.channelCards = [
            makeChannelCard(id: "imessage_sms", name: "iMessage/SMS Bridge", status: .active),
            makeChannelCard(id: "twilio_sms", name: "Twilio SMS", status: .active)
        ]
        state.connectorCards = [
            makeConnectorCard(
                id: "messages",
                name: "Messages",
                health: .ready,
                permissionState: .granted,
                permissionScope: "iMessage",
                statusReason: nil,
                capabilities: "imessage,sms",
                actions: []
            ),
            makeConnectorCard(
                id: "twilio",
                name: "Twilio",
                health: .ready,
                permissionState: .granted,
                permissionScope: "Twilio",
                statusReason: nil,
                capabilities: "sms,voice",
                actions: []
            )
        ]

        state.channelConnectorMappingsByChannelID["message"] = [
            makeChannelConnectorMapping(channelID: "message", connectorID: "imessage", enabled: true, priority: 1)
        ]
        var message = try! XCTUnwrap(state.logicalChannelCards.first(where: { $0.channelID == "message" }))
        XCTAssertEqual(message.mappedConnectorRollups.map(\.connectorID), ["imessage"])

        state.channelConnectorMappingsByChannelID["message"] = [
            makeChannelConnectorMapping(channelID: "message", connectorID: "twilio", enabled: true, priority: 1)
        ]
        message = try! XCTUnwrap(state.logicalChannelCards.first(where: { $0.channelID == "message" }))
        XCTAssertEqual(message.mappedConnectorRollups.map(\.connectorID), ["twilio"])
    }

    func testLogicalChannelCardsPreferSetupRequiredStatusWhenAnyMappedCardNeedsSetup() {
        let state = AppShellState()
        state.channelCards = [
            makeChannelCard(id: "imessage_sms", name: "iMessage/SMS Bridge", status: .degraded),
            makeChannelCard(id: "twilio_sms", name: "Twilio SMS", status: .setupRequired)
        ]

        let message = try! XCTUnwrap(state.logicalChannelCards.first(where: { $0.channelID == "message" }))
        XCTAssertEqual(message.status, .setupRequired)
    }

    func testLogicalChannelCardsDoNotDuplicateWhenCanonicalAndLegacyIDsCoexist() {
        let state = AppShellState()
        state.channelCards = [
            makeChannelCard(id: "app", name: "App", status: .active),
            makeChannelCard(id: "app_chat", name: "App Chat", status: .active),
            makeChannelCard(id: "message", name: "Message", status: .active),
            makeChannelCard(id: "imessage_sms", name: "iMessage/SMS Bridge", status: .degraded),
            makeChannelCard(id: "voice", name: "Voice", status: .active),
            makeChannelCard(id: "twilio_voice", name: "Twilio Voice", status: .setupRequired)
        ]

        let logicalCards = state.logicalChannelCards

        XCTAssertEqual(logicalCards.count, 3)
        XCTAssertEqual(Set(logicalCards.map(\.channelID)), Set(["app", "message", "voice"]))
        let app = try! XCTUnwrap(logicalCards.first(where: { $0.channelID == "app" }))
        XCTAssertEqual(Set(app.channelCardIDs), Set(["app", "app_chat"]))
        let message = try! XCTUnwrap(logicalCards.first(where: { $0.channelID == "message" }))
        XCTAssertEqual(Set(message.channelCardIDs), Set(["message", "imessage_sms"]))
        let voice = try! XCTUnwrap(logicalCards.first(where: { $0.channelID == "voice" }))
        XCTAssertEqual(Set(voice.channelCardIDs), Set(["voice", "twilio_voice"]))
    }

    func testLogicalChannelCardsCanonicalAndLegacyInputsPreserveMappingAndActionParity() {
        let channelActions = [
            makeAction(id: "open_system_settings", title: "Open System Settings"),
            makeAction(id: "open_channel_logs", title: "Open Logs")
        ]

        let canonicalState = AppShellState()
        canonicalState.channelCards = [
            makeChannelCard(id: "message", name: "Message", status: .degraded, actions: channelActions)
        ]
        canonicalState.channelConnectorMappingsByChannelID["message"] = [
            makeChannelConnectorMapping(channelID: "message", connectorID: "imessage", enabled: true, priority: 1)
        ]
        canonicalState.connectorCards = [
            makeConnectorCard(
                id: "imessage",
                name: "Messages",
                health: .needsPermission,
                permissionState: .missing,
                permissionScope: "iMessage",
                statusReason: "permission_missing",
                capabilities: "imessage,sms",
                actions: [makeAction(id: "open_system_settings", title: "Open System Settings")]
            )
        ]

        let legacyState = AppShellState()
        legacyState.channelCards = [
            makeChannelCard(id: "imessage_sms", name: "iMessage/SMS Bridge", status: .degraded, actions: channelActions)
        ]
        legacyState.channelConnectorMappingsByChannelID["message"] = [
            makeChannelConnectorMapping(channelID: "message", connectorID: "messages", enabled: true, priority: 1)
        ]
        legacyState.connectorCards = [
            makeConnectorCard(
                id: "messages",
                name: "Messages",
                health: .needsPermission,
                permissionState: .missing,
                permissionScope: "iMessage",
                statusReason: "permission_missing",
                capabilities: "imessage,sms",
                actions: [makeAction(id: "open_system_settings", title: "Open System Settings")]
            )
        ]

        let canonicalMessage = try! XCTUnwrap(canonicalState.logicalChannelCards.first(where: { $0.channelID == "message" }))
        let legacyMessage = try! XCTUnwrap(legacyState.logicalChannelCards.first(where: { $0.channelID == "message" }))

        XCTAssertEqual(canonicalMessage.status, legacyMessage.status)
        XCTAssertEqual(Set(canonicalMessage.actions.map(\.title)), Set(legacyMessage.actions.map(\.title)))
        XCTAssertEqual(canonicalMessage.mappedConnectorRollups.map(\.connectorID), ["imessage"])
        XCTAssertEqual(legacyMessage.mappedConnectorRollups.map(\.connectorID), ["imessage"])
        XCTAssertEqual(canonicalMessage.connectorActionTitles, legacyMessage.connectorActionTitles)
    }

    func testChannelCardItemLookupReturnsMappedPrimaryCard() {
        let state = AppShellState()
        state.channelCards = [
            makeChannelCard(id: "app_chat", name: "App Chat", status: .active)
        ]

        let logicalApp = try! XCTUnwrap(state.logicalChannelCards.first(where: { $0.channelID == "app" }))
        let primary = state.channelCardItem(channelID: logicalApp.primaryChannelCardID)

        XCTAssertEqual(primary?.id, "app_chat")
        XCTAssertEqual(primary?.name, "App Chat")
    }

    func testLogicalConnectorCardsMergeTwilioCardsIntoUnifiedConnector() {
        let state = AppShellState()
        state.connectorCards = [
            makeConnectorCard(
                id: "twilio_sms",
                name: "Twilio SMS",
                health: .ready,
                permissionState: .granted,
                permissionScope: "SMS send",
                statusReason: nil,
                capabilities: "sms",
                actions: [
                    makeAction(id: "open_system_settings", title: "Open System Settings")
                ]
            ),
            makeConnectorCard(
                id: "twilio_voice",
                name: "Twilio Voice",
                health: .needsPermission,
                permissionState: .missing,
                permissionScope: "Voice calls",
                statusReason: "permission_missing",
                capabilities: "voice",
                actions: [
                    makeAction(id: "request_permission", title: "Request Permission"),
                    makeAction(id: "open_system_settings", title: "Open System Settings")
                ]
            ),
            makeConnectorCard(
                id: "messages",
                name: "Messages",
                health: .ready,
                permissionState: .granted,
                permissionScope: "iMessage",
                statusReason: nil,
                capabilities: "imessage,sms",
                actions: [
                    makeAction(id: "open_system_settings", title: "Open System Settings")
                ]
            )
        ]

        let logicalCards = state.logicalConnectorCards

        XCTAssertEqual(Set(logicalCards.map(\.id)), Set(["twilio", "imessage"]))
        let twilio = try! XCTUnwrap(logicalCards.first(where: { $0.id == "twilio" }))
        XCTAssertEqual(twilio.title, "Twilio")
        XCTAssertEqual(twilio.primaryConnectorCardID, "twilio_sms")
        XCTAssertEqual(Set(twilio.connectorCardIDs), Set(["twilio_sms", "twilio_voice"]))
        XCTAssertEqual(twilio.health, .needsPermission)
        XCTAssertEqual(twilio.permissionState, .missing)
        XCTAssertEqual(twilio.permissionScope, "SMS send • Voice calls")
        XCTAssertEqual(twilio.details["Mapped Connector IDs"], "twilio_sms, twilio_voice")
        XCTAssertEqual(twilio.details["Capabilities"], "sms, voice")
        XCTAssertTrue(twilio.summary.contains("2 mapped connectors"))

        let messages = try! XCTUnwrap(logicalCards.first(where: { $0.id == "imessage" }))
        XCTAssertEqual(messages.title, "iMessage")
        XCTAssertEqual(messages.primaryConnectorCardID, "messages")
        XCTAssertEqual(messages.details["Canonical Connector ID"], "imessage")
    }

    func testLogicalConnectorCardsInjectConnectorIDIntoRolledUpActions() {
        let state = AppShellState()
        state.connectorCards = [
            makeConnectorCard(
                id: "twilio_sms",
                name: "Twilio SMS",
                health: .ready,
                permissionState: .granted,
                permissionScope: "SMS send",
                statusReason: nil,
                capabilities: "sms",
                actions: [
                    makeAction(id: "open_system_settings", title: "Open System Settings"),
                    makeAction(id: "check_connector", title: "Run Connector Check")
                ]
            ),
            makeConnectorCard(
                id: "twilio_voice",
                name: "Twilio Voice",
                health: .ready,
                permissionState: .granted,
                permissionScope: "Voice calls",
                statusReason: nil,
                capabilities: "voice",
                actions: [
                    makeAction(id: "open_system_settings", title: "Open System Settings")
                ]
            )
        ]

        let twilio = try! XCTUnwrap(state.logicalConnectorCards.first(where: { $0.id == "twilio" }))

        XCTAssertEqual(Set(twilio.actions.map(\.title)), Set(["Open System Settings", "Run Connector Check"]))
        for action in twilio.actions {
            let connectorID = action.parameters["connector_id"]
            XCTAssertNotNil(connectorID)
            XCTAssertTrue(twilio.connectorCardIDs.contains(connectorID ?? ""))
        }
    }

    func testChannelConnectorMappingToggleTracksDraftChanges() {
        let state = AppShellState()
        state.channelConnectorMappingsByChannelID["message"] = [
            makeChannelConnectorMapping(channelID: "message", connectorID: "imessage", enabled: true, priority: 1),
            makeChannelConnectorMapping(channelID: "message", connectorID: "twilio", enabled: true, priority: 2)
        ]
        state.channelConnectorMappingDraftByChannelID["message"] = state.channelConnectorMappingsByChannelID["message"]

        XCTAssertFalse(state.channelConnectorMappingHasDraftChanges(channelID: "message"))
        state.setChannelConnectorMappingEnabled(channelID: "message", connectorID: "twilio", enabled: false)
        XCTAssertTrue(state.channelConnectorMappingHasDraftChanges(channelID: "message"))
    }

    func testMoveChannelConnectorMappingDownReordersPriorityDeterministically() {
        let state = AppShellState()
        state.channelConnectorMappingsByChannelID["message"] = [
            makeChannelConnectorMapping(channelID: "message", connectorID: "imessage", enabled: true, priority: 1),
            makeChannelConnectorMapping(channelID: "message", connectorID: "twilio", enabled: true, priority: 2)
        ]
        state.channelConnectorMappingDraftByChannelID["message"] = state.channelConnectorMappingsByChannelID["message"]

        state.moveChannelConnectorMappingDown(channelID: "message", connectorID: "imessage")

        let reordered = state.channelConnectorMappings(channelID: "message")
        XCTAssertEqual(reordered.map(\.connectorID), ["twilio", "imessage"])
        XCTAssertEqual(reordered.map(\.priority), [1, 2])
    }

    func testResetChannelConnectorMappingDraftRestoresSourceMappings() {
        let state = AppShellState()
        state.channelConnectorMappingsByChannelID["message"] = [
            makeChannelConnectorMapping(channelID: "message", connectorID: "imessage", enabled: true, priority: 1),
            makeChannelConnectorMapping(channelID: "message", connectorID: "twilio", enabled: true, priority: 2)
        ]
        state.channelConnectorMappingDraftByChannelID["message"] = state.channelConnectorMappingsByChannelID["message"]
        state.setChannelConnectorMappingEnabled(channelID: "message", connectorID: "twilio", enabled: false)
        XCTAssertTrue(state.channelConnectorMappingHasDraftChanges(channelID: "message"))

        state.resetChannelConnectorMappingDraft(channelID: "message")

        XCTAssertFalse(state.channelConnectorMappingHasDraftChanges(channelID: "message"))
        XCTAssertEqual(state.channelConnectorMappings(channelID: "message").first(where: { $0.connectorID == "twilio" })?.enabled, true)
    }

    func testSaveChannelConnectorMappingsWithoutTokenSetsDeterministicStatus() async {
        let state = AppShellState()
        state.clearLocalDevToken()
        state.channelConnectorMappingsByChannelID["message"] = [
            makeChannelConnectorMapping(channelID: "message", connectorID: "imessage", enabled: true, priority: 1)
        ]
        state.channelConnectorMappingDraftByChannelID["message"] = [
            makeChannelConnectorMapping(channelID: "message", connectorID: "imessage", enabled: false, priority: 1)
        ]

        state.saveChannelConnectorMappings(channelID: "message")
        try? await Task.sleep(for: .milliseconds(120))

        XCTAssertEqual(
            state.channelConnectorMappingActionStatusByChannelID["message"],
            "Set Assistant Access Token before saving connector mappings."
        )
    }

    func testConnectorCardItemLookupReturnsPrimaryConnectorByID() {
        let state = AppShellState()
        state.connectorCards = [
            makeConnectorCard(
                id: "messages",
                name: "Messages",
                health: .ready,
                permissionState: .granted,
                permissionScope: "iMessage",
                statusReason: nil,
                capabilities: "imessage,sms",
                actions: []
            )
        ]

        let logical = try! XCTUnwrap(state.logicalConnectorCards.first(where: { $0.id == "imessage" }))
        let primary = state.connectorCardItem(connectorID: logical.primaryConnectorCardID)

        XCTAssertEqual(primary?.id, "messages")
        XCTAssertEqual(primary?.name, "Messages")
    }

    func testLogicalConnectorCardsPreferCanonicalPrimaryWhenCanonicalAndLegacyCardsExist() {
        let state = AppShellState()
        state.connectorCards = [
            makeConnectorCard(
                id: "twilio_sms",
                name: "Twilio SMS",
                health: .ready,
                permissionState: .granted,
                permissionScope: "SMS",
                statusReason: nil,
                capabilities: "sms",
                actions: []
            ),
            makeConnectorCard(
                id: "twilio",
                name: "Twilio",
                health: .ready,
                permissionState: .granted,
                permissionScope: "SMS+Voice",
                statusReason: nil,
                capabilities: "sms,voice",
                actions: []
            ),
            makeConnectorCard(
                id: "imessage",
                name: "Messages",
                health: .ready,
                permissionState: .granted,
                permissionScope: "iMessage",
                statusReason: nil,
                capabilities: "imessage,sms",
                actions: []
            ),
            makeConnectorCard(
                id: "messages",
                name: "Messages (Legacy)",
                health: .ready,
                permissionState: .granted,
                permissionScope: "iMessage",
                statusReason: nil,
                capabilities: "imessage,sms",
                actions: []
            )
        ]

        let logicalCards = state.logicalConnectorCards
        XCTAssertEqual(logicalCards.count, 2)
        XCTAssertEqual(Set(logicalCards.map(\.id)), Set(["twilio", "imessage"]))

        let twilio = try! XCTUnwrap(logicalCards.first(where: { $0.id == "twilio" }))
        XCTAssertEqual(twilio.primaryConnectorCardID, "twilio")

        let messages = try! XCTUnwrap(logicalCards.first(where: { $0.id == "imessage" }))
        XCTAssertEqual(messages.primaryConnectorCardID, "imessage")
        XCTAssertEqual(Set(messages.connectorCardIDs), Set(["imessage", "messages"]))
    }

    private func makeChannelCard(
        id: String,
        name: String,
        status: ChannelCardStatus,
        actions: [DiagnosticsActionItem]? = nil
    ) -> ChannelCardItem {
        ChannelCardItem(
            id: id,
            name: name,
            status: status,
            summary: "\(name) summary",
            details: [:],
            editableConfiguration: [:],
            editableConfigurationKinds: [:],
            readOnlyConfiguration: [:],
            actions: actions ?? [makeAction(id: "open_channel_logs", title: "Open Logs")],
            unavailableActionReason: "No channel action available."
        )
    }

    private func makeConnectorCard(
        id: String,
        name: String,
        health: ConnectorHealthStatus,
        permissionState: ConnectorPermissionState = .granted,
        permissionScope: String,
        statusReason: String?,
        capabilities: String,
        actions: [DiagnosticsActionItem]
    ) -> ConnectorCardItem {
        ConnectorCardItem(
            id: id,
            name: name,
            health: health,
            permissionState: permissionState,
            permissionScope: permissionScope,
            statusReason: statusReason,
            summary: "\(name) summary",
            details: ["Capabilities": capabilities],
            editableConfiguration: [:],
            editableConfigurationKinds: [:],
            readOnlyConfiguration: [:],
            actions: actions,
            unavailableActionReason: "No connector action available."
        )
    }

    private func makeAction(id: String, title: String) -> DiagnosticsActionItem {
        DiagnosticsActionItem(
            id: id,
            title: title,
            intent: "navigate",
            destination: "ui://connectors",
            enabled: true,
            recommended: false,
            reason: nil
        )
    }

    private func makeChannelConnectorMapping(
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
}
