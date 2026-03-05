import Foundation
import SwiftUI

@MainActor
final class AppConnectionConfigStore: ObservableObject {
    struct ConfigurationMutationValidationError: Error {
        let message: String
    }

    @Published var channelConnectorMappingFallbackPolicy = "priority_order"
    @Published var channelConnectorMappingsByChannelID: [String: [ChannelConnectorMappingItem]] = [:]
    @Published var channelConnectorMappingDraftByChannelID: [String: [ChannelConnectorMappingItem]] = [:]
    @Published var channelConnectorMappingActionStatusByChannelID: [String: String] = [:]
    @Published var channelConnectorMappingSaveInFlightChannelIDs: Set<String> = []
    @Published var channelConfigDraftByID: [String: [String: String]] = [:]
    @Published var channelConfigKindsByID: [String: [String: ConfigurationDraftValueKind]] = [:]
    @Published var channelConfigActionStatusByID: [String: String] = [:]
    @Published var channelConfigSaveInFlightIDs: Set<String> = []
    @Published var channelTestInFlightIDs: Set<String> = []
    @Published var channelLastTestResultByID: [String: ConfigurationTestResultItem] = [:]
    @Published var channelDeliveryPoliciesByChannelID: [String: [ChannelDeliveryPolicyItem]] = [:]
    @Published var channelDeliveryPolicyDraftByID: [String: ChannelDeliveryPolicyDraft] = [:]
    @Published var channelDeliveryPolicyActionStatusByID: [String: String] = [:]
    @Published var channelDeliveryPolicySaveInFlightIDs: Set<String> = []
    @Published var connectorConfigDraftByID: [String: [String: String]] = [:]
    @Published var connectorConfigKindsByID: [String: [String: ConfigurationDraftValueKind]] = [:]
    @Published var connectorConfigActionStatusByID: [String: String] = [:]
    @Published var connectorConfigSaveInFlightIDs: Set<String> = []
    @Published var connectorTestInFlightIDs: Set<String> = []
    @Published var connectorLastTestResultByID: [String: ConfigurationTestResultItem] = [:]
    @Published var connectorPermissionActionStatusByID: [String: String] = [:]
    @Published var connectorPermissionRequestInFlightIDs: Set<String> = []
    @Published var connectorPermissionStatesByID: [String: ConnectorPermissionState] = [
        "imessage": .unknown,
        "mail": .unknown,
        "calendar": .unknown,
        "browser": .unknown,
        "finder": .unknown,
    ]
    @Published var connectorPermissionRefreshPendingIDs: Set<String> = []

    func channelConnectorMappings(
        channelID: String,
        normalizeChannelID: (String) -> String,
        sortedMappings: ([ChannelConnectorMappingItem]) -> [ChannelConnectorMappingItem]
    ) -> [ChannelConnectorMappingItem] {
        let normalizedChannelID = normalizeChannelID(channelID)
        return sortedMappings(
            channelConnectorMappingDraftByChannelID[normalizedChannelID]
                ?? channelConnectorMappingsByChannelID[normalizedChannelID]
                ?? []
        )
    }

    func channelConnectorMappingStatusMessage(
        channelID: String,
        normalizeChannelID: (String) -> String
    ) -> String? {
        channelConnectorMappingActionStatusByChannelID[normalizeChannelID(channelID)]
    }

    func channelConnectorMappingHasDraftChanges(
        channelID: String,
        normalizeChannelID: (String) -> String,
        sortedMappings: ([ChannelConnectorMappingItem]) -> [ChannelConnectorMappingItem]
    ) -> Bool {
        let normalizedChannelID = normalizeChannelID(channelID)
        let source = sortedMappings(channelConnectorMappingsByChannelID[normalizedChannelID] ?? [])
        let draft = sortedMappings(channelConnectorMappingDraftByChannelID[normalizedChannelID] ?? source)
        return source != draft
    }

    func setChannelConnectorMappingEnabled(
        channelID: String,
        connectorID: String,
        enabled: Bool,
        normalizeChannelID: (String) -> String,
        normalizeConnectorID: (String) -> String,
        sortedMappings: ([ChannelConnectorMappingItem]) -> [ChannelConnectorMappingItem]
    ) {
        let normalizedChannelID = normalizeChannelID(channelID)
        let normalizedConnectorID = normalizeConnectorID(connectorID)
        guard !normalizedChannelID.isEmpty, !normalizedConnectorID.isEmpty else {
            return
        }
        var draft = channelConnectorMappings(
            channelID: normalizedChannelID,
            normalizeChannelID: normalizeChannelID,
            sortedMappings: sortedMappings
        )
        guard let index = draft.firstIndex(where: { $0.connectorID == normalizedConnectorID }) else {
            return
        }
        draft[index].enabled = enabled
        channelConnectorMappingDraftByChannelID[normalizedChannelID] = sortedMappings(draft)
    }

    func canMoveChannelConnectorMapping(
        channelID: String,
        connectorID: String,
        direction: Int,
        normalizeChannelID: (String) -> String,
        normalizeConnectorID: (String) -> String,
        sortedMappings: ([ChannelConnectorMappingItem]) -> [ChannelConnectorMappingItem]
    ) -> Bool {
        let normalizedChannelID = normalizeChannelID(channelID)
        let normalizedConnectorID = normalizeConnectorID(connectorID)
        let draft = channelConnectorMappings(
            channelID: normalizedChannelID,
            normalizeChannelID: normalizeChannelID,
            sortedMappings: sortedMappings
        )
        guard let index = draft.firstIndex(where: { $0.connectorID == normalizedConnectorID }) else {
            return false
        }
        let targetIndex = index + direction
        return targetIndex >= 0 && targetIndex < draft.count
    }

    func resetChannelConnectorMappingDraft(
        channelID: String,
        normalizeChannelID: (String) -> String,
        sortedMappings: ([ChannelConnectorMappingItem]) -> [ChannelConnectorMappingItem]
    ) {
        let normalizedChannelID = normalizeChannelID(channelID)
        channelConnectorMappingDraftByChannelID[normalizedChannelID] =
            sortedMappings(channelConnectorMappingsByChannelID[normalizedChannelID] ?? [])
        channelConnectorMappingActionStatusByChannelID[normalizedChannelID] = "Reset channel connector mapping draft."
    }

    func channelConfigHasDraftChanges(
        channelID: String,
        baseline: [String: String]
    ) -> Bool {
        let draft = channelConfigDraftByID[channelID] ?? baseline
        return draft != baseline
    }

    func channelConfigDraftKeys(channelID: String) -> [String] {
        (channelConfigDraftByID[channelID] ?? [:]).keys.sorted()
    }

    func channelConfigDraftValue(channelID: String, key: String) -> String {
        channelConfigDraftByID[channelID]?[key] ?? ""
    }

    func channelConfigDraftKind(channelID: String, key: String) -> ConfigurationDraftValueKind {
        channelConfigKindsByID[channelID]?[key] ?? .string
    }

    func setChannelConfigDraftValue(channelID: String, key: String, value: String) {
        var draft = channelConfigDraftByID[channelID] ?? [:]
        draft[key] = value
        channelConfigDraftByID[channelID] = draft
    }

    func addChannelConfigDraftField(
        channelID: String,
        key: String,
        value: String,
        inferDraftKind: (String) -> ConfigurationDraftValueKind
    ) {
        let normalizedKey = key.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !normalizedKey.isEmpty else {
            channelConfigActionStatusByID[channelID] = "Configuration key is required."
            return
        }
        var draft = channelConfigDraftByID[channelID] ?? [:]
        draft[normalizedKey] = value
        channelConfigDraftByID[channelID] = draft
        var kindMap = channelConfigKindsByID[channelID] ?? [:]
        kindMap[normalizedKey] = inferDraftKind(value)
        channelConfigKindsByID[channelID] = kindMap
        channelConfigActionStatusByID[channelID] = "Added draft field \(normalizedKey)."
    }

    func removeChannelConfigDraftField(channelID: String, key: String) {
        var draft = channelConfigDraftByID[channelID] ?? [:]
        draft.removeValue(forKey: key)
        channelConfigDraftByID[channelID] = draft
        var kindMap = channelConfigKindsByID[channelID] ?? [:]
        kindMap.removeValue(forKey: key)
        channelConfigKindsByID[channelID] = kindMap
        channelConfigActionStatusByID[channelID] = "Removed draft field \(key)."
    }

    func resetChannelConfigDraft(channelID: String, channelCards: [ChannelCardItem]) {
        guard let card = channelCards.first(where: { $0.id == channelID }) else {
            channelConfigActionStatusByID[channelID] = "Unable to reset draft for unknown channel \(channelID)."
            return
        }
        channelConfigDraftByID[channelID] = card.editableConfiguration
        channelConfigKindsByID[channelID] = card.editableConfigurationKinds
        channelConfigActionStatusByID[channelID] = "Reset channel configuration draft."
    }

    func channelDeliveryPolicies(
        channelID: String,
        normalizeChannelID: (String) -> String
    ) -> [ChannelDeliveryPolicyItem] {
        let normalizedChannelID = normalizeChannelID(channelID)
        guard !normalizedChannelID.isEmpty else {
            return []
        }
        return channelDeliveryPoliciesByChannelID[normalizedChannelID] ?? []
    }

    func channelDeliveryPolicyDraft(
        channelID: String,
        normalizeChannelID: (String) -> String,
        defaultDraft: (String, [ChannelDeliveryPolicyItem]) -> ChannelDeliveryPolicyDraft
    ) -> ChannelDeliveryPolicyDraft {
        let normalizedChannelID = normalizeChannelID(channelID)
        guard !normalizedChannelID.isEmpty else {
            return defaultDraft("unknown", [])
        }
        return channelDeliveryPolicyDraftByID[normalizedChannelID]
            ?? defaultDraft(
                normalizedChannelID,
                channelDeliveryPolicies(channelID: normalizedChannelID, normalizeChannelID: normalizeChannelID)
            )
    }

    func channelDeliveryPolicyHasDraftChanges(
        channelID: String,
        normalizeChannelID: (String) -> String,
        defaultDraft: (String, [ChannelDeliveryPolicyItem]) -> ChannelDeliveryPolicyDraft
    ) -> Bool {
        let normalizedChannelID = normalizeChannelID(channelID)
        guard !normalizedChannelID.isEmpty else {
            return false
        }
        let baseline = defaultDraft(
            normalizedChannelID,
            channelDeliveryPoliciesByChannelID[normalizedChannelID] ?? []
        )
        let draft = channelDeliveryPolicyDraftByID[normalizedChannelID] ?? baseline
        return draft != baseline
    }

    func setChannelDeliveryPolicyPrimaryChannel(
        channelID: String,
        primaryChannel: String,
        normalizeChannelID: (String) -> String,
        defaultDraft: (String, [ChannelDeliveryPolicyItem]) -> ChannelDeliveryPolicyDraft
    ) {
        let normalizedChannelID = normalizeChannelID(channelID)
        guard !normalizedChannelID.isEmpty else {
            return
        }
        var draft = channelDeliveryPolicyDraft(
            channelID: normalizedChannelID,
            normalizeChannelID: normalizeChannelID,
            defaultDraft: defaultDraft
        )
        let normalizedPrimary = normalizeChannelID(primaryChannel)
        draft.primaryChannel = normalizedPrimary.isEmpty ? normalizedChannelID : normalizedPrimary
        channelDeliveryPolicyDraftByID[normalizedChannelID] = draft
    }

    func setChannelDeliveryPolicyEndpointPattern(
        channelID: String,
        endpointPattern: String,
        normalizeChannelID: (String) -> String,
        defaultDraft: (String, [ChannelDeliveryPolicyItem]) -> ChannelDeliveryPolicyDraft
    ) {
        let normalizedChannelID = normalizeChannelID(channelID)
        guard !normalizedChannelID.isEmpty else {
            return
        }
        var draft = channelDeliveryPolicyDraft(
            channelID: normalizedChannelID,
            normalizeChannelID: normalizeChannelID,
            defaultDraft: defaultDraft
        )
        draft.endpointPattern = endpointPattern
        channelDeliveryPolicyDraftByID[normalizedChannelID] = draft
    }

    func setChannelDeliveryPolicyRetryCount(
        channelID: String,
        retryCount: Int,
        normalizeChannelID: (String) -> String,
        defaultDraft: (String, [ChannelDeliveryPolicyItem]) -> ChannelDeliveryPolicyDraft
    ) {
        let normalizedChannelID = normalizeChannelID(channelID)
        guard !normalizedChannelID.isEmpty else {
            return
        }
        var draft = channelDeliveryPolicyDraft(
            channelID: normalizedChannelID,
            normalizeChannelID: normalizeChannelID,
            defaultDraft: defaultDraft
        )
        draft.retryCount = max(retryCount, 0)
        channelDeliveryPolicyDraftByID[normalizedChannelID] = draft
    }

    func setChannelDeliveryPolicyFallbackChannelsText(
        channelID: String,
        fallbackChannelsText: String,
        normalizeChannelID: (String) -> String,
        defaultDraft: (String, [ChannelDeliveryPolicyItem]) -> ChannelDeliveryPolicyDraft
    ) {
        let normalizedChannelID = normalizeChannelID(channelID)
        guard !normalizedChannelID.isEmpty else {
            return
        }
        var draft = channelDeliveryPolicyDraft(
            channelID: normalizedChannelID,
            normalizeChannelID: normalizeChannelID,
            defaultDraft: defaultDraft
        )
        draft.fallbackChannelsText = fallbackChannelsText
        channelDeliveryPolicyDraftByID[normalizedChannelID] = draft
    }

    func setChannelDeliveryPolicyIsDefault(
        channelID: String,
        isDefault: Bool,
        normalizeChannelID: (String) -> String,
        defaultDraft: (String, [ChannelDeliveryPolicyItem]) -> ChannelDeliveryPolicyDraft
    ) {
        let normalizedChannelID = normalizeChannelID(channelID)
        guard !normalizedChannelID.isEmpty else {
            return
        }
        var draft = channelDeliveryPolicyDraft(
            channelID: normalizedChannelID,
            normalizeChannelID: normalizeChannelID,
            defaultDraft: defaultDraft
        )
        draft.isDefault = isDefault
        channelDeliveryPolicyDraftByID[normalizedChannelID] = draft
    }

    func loadChannelDeliveryPolicyDraft(
        channelID: String,
        policyID: String,
        normalizeChannelID: (String) -> String,
        channelDeliveryDraft: (ChannelDeliveryPolicyItem) -> ChannelDeliveryPolicyDraft
    ) {
        let normalizedChannelID = normalizeChannelID(channelID)
        guard !normalizedChannelID.isEmpty else {
            return
        }
        guard let policy = channelDeliveryPoliciesByChannelID[normalizedChannelID]?.first(where: { $0.id == policyID }) else {
            channelDeliveryPolicyActionStatusByID[normalizedChannelID] = "Unable to load policy \(policyID) for edit."
            return
        }
        channelDeliveryPolicyDraftByID[normalizedChannelID] = channelDeliveryDraft(policy)
        channelDeliveryPolicyActionStatusByID[normalizedChannelID] = "Loaded policy \(policyID) for edit."
    }

    func startNewChannelDeliveryPolicyDraft(
        channelID: String,
        normalizeChannelID: (String) -> String
    ) {
        let normalizedChannelID = normalizeChannelID(channelID)
        guard !normalizedChannelID.isEmpty else {
            return
        }
        channelDeliveryPolicyDraftByID[normalizedChannelID] = ChannelDeliveryPolicyDraft(
            policyID: nil,
            endpointPattern: "",
            primaryChannel: normalizedChannelID,
            retryCount: 0,
            fallbackChannelsText: "",
            isDefault: (channelDeliveryPoliciesByChannelID[normalizedChannelID] ?? []).isEmpty
        )
        channelDeliveryPolicyActionStatusByID[normalizedChannelID] = "Started new delivery policy draft."
    }

    func resetChannelDeliveryPolicyDraft(
        channelID: String,
        normalizeChannelID: (String) -> String,
        defaultDraft: (String, [ChannelDeliveryPolicyItem]) -> ChannelDeliveryPolicyDraft
    ) {
        let normalizedChannelID = normalizeChannelID(channelID)
        guard !normalizedChannelID.isEmpty else {
            return
        }
        channelDeliveryPolicyDraftByID[normalizedChannelID] = defaultDraft(
            normalizedChannelID,
            channelDeliveryPoliciesByChannelID[normalizedChannelID] ?? []
        )
        channelDeliveryPolicyActionStatusByID[normalizedChannelID] = "Reset delivery policy draft."
    }

    func connectorConfigDraftKeys(connectorID: String) -> [String] {
        (connectorConfigDraftByID[connectorID] ?? [:]).keys.sorted()
    }

    func connectorConfigHasDraftChanges(
        connectorID: String,
        baseline: [String: String]
    ) -> Bool {
        let draft = connectorConfigDraftByID[connectorID] ?? baseline
        return draft != baseline
    }

    func connectorConfigDraftValue(connectorID: String, key: String) -> String {
        connectorConfigDraftByID[connectorID]?[key] ?? ""
    }

    func connectorConfigDraftKind(connectorID: String, key: String) -> ConfigurationDraftValueKind {
        connectorConfigKindsByID[connectorID]?[key] ?? .string
    }

    func setConnectorConfigDraftValue(connectorID: String, key: String, value: String) {
        var draft = connectorConfigDraftByID[connectorID] ?? [:]
        draft[key] = value
        connectorConfigDraftByID[connectorID] = draft
    }

    func addConnectorConfigDraftField(
        connectorID: String,
        key: String,
        value: String,
        inferDraftKind: (String) -> ConfigurationDraftValueKind
    ) {
        let normalizedKey = key.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !normalizedKey.isEmpty else {
            connectorConfigActionStatusByID[connectorID] = "Configuration key is required."
            return
        }
        var draft = connectorConfigDraftByID[connectorID] ?? [:]
        draft[normalizedKey] = value
        connectorConfigDraftByID[connectorID] = draft
        var kindMap = connectorConfigKindsByID[connectorID] ?? [:]
        kindMap[normalizedKey] = inferDraftKind(value)
        connectorConfigKindsByID[connectorID] = kindMap
        connectorConfigActionStatusByID[connectorID] = "Added draft field \(normalizedKey)."
    }

    func removeConnectorConfigDraftField(connectorID: String, key: String) {
        var draft = connectorConfigDraftByID[connectorID] ?? [:]
        draft.removeValue(forKey: key)
        connectorConfigDraftByID[connectorID] = draft
        var kindMap = connectorConfigKindsByID[connectorID] ?? [:]
        kindMap.removeValue(forKey: key)
        connectorConfigKindsByID[connectorID] = kindMap
        connectorConfigActionStatusByID[connectorID] = "Removed draft field \(key)."
    }

    func resetConnectorConfigDraft(connectorID: String, connectorCards: [ConnectorCardItem]) {
        guard let card = connectorCards.first(where: { $0.id == connectorID }) else {
            connectorConfigActionStatusByID[connectorID] = "Unable to reset draft for unknown connector \(connectorID)."
            return
        }
        connectorConfigDraftByID[connectorID] = card.editableConfiguration
        connectorConfigKindsByID[connectorID] = card.editableConfigurationKinds
        connectorConfigActionStatusByID[connectorID] = "Reset connector configuration draft."
    }

    func synchronizeChannelConnectorMappingDrafts(
        with cards: [ChannelCardItem],
        mappingsByChannelID: [String: [ChannelConnectorMappingItem]],
        normalizeChannelID: (String) -> String,
        inferredMappingsByLogicalChannelID: ([ChannelCardItem]) -> [String: [ChannelConnectorMappingItem]],
        mergeMappings: (
            _ observed: [ChannelConnectorMappingItem],
            _ inferred: [ChannelConnectorMappingItem],
            _ channelID: String
        ) -> [ChannelConnectorMappingItem]
    ) {
        var validChannelIDs = Set(cards.map { normalizeChannelID($0.id) })
        validChannelIDs.formUnion(mappingsByChannelID.keys)
        validChannelIDs = Set(validChannelIDs.filter { !$0.isEmpty })

        let inferredByLogicalChannel = inferredMappingsByLogicalChannelID(cards)
        var normalizedSource: [String: [ChannelConnectorMappingItem]] = [:]
        for channelID in validChannelIDs {
            let source = mergeMappings(
                mappingsByChannelID[channelID] ?? [],
                inferredByLogicalChannel[channelID] ?? [],
                channelID
            )
            normalizedSource[channelID] = source
        }

        channelConnectorMappingsByChannelID = normalizedSource
        channelConnectorMappingDraftByChannelID = normalizedSource
        channelConnectorMappingActionStatusByChannelID =
            channelConnectorMappingActionStatusByChannelID.filter { validChannelIDs.contains($0.key) }
        channelConnectorMappingSaveInFlightChannelIDs.formIntersection(validChannelIDs)
    }

    func reorderChannelConnectorMapping(
        channelID: String,
        connectorID: String,
        direction: Int,
        normalizeChannelID: (String) -> String,
        normalizeConnectorID: (String) -> String,
        sortedMappings: ([ChannelConnectorMappingItem]) -> [ChannelConnectorMappingItem],
        rebalanceMappings: ([ChannelConnectorMappingItem]) -> [ChannelConnectorMappingItem],
        connectorDisplayName: (String) -> String
    ) {
        let normalizedChannelID = normalizeChannelID(channelID)
        let normalizedConnectorID = normalizeConnectorID(connectorID)
        guard !normalizedChannelID.isEmpty, !normalizedConnectorID.isEmpty else {
            return
        }

        var draft = channelConnectorMappings(
            channelID: normalizedChannelID,
            normalizeChannelID: normalizeChannelID,
            sortedMappings: sortedMappings
        )
        guard let index = draft.firstIndex(where: { $0.connectorID == normalizedConnectorID }) else {
            return
        }
        let targetIndex = index + direction
        guard targetIndex >= 0 && targetIndex < draft.count else {
            return
        }
        draft.swapAt(index, targetIndex)
        draft = rebalanceMappings(draft)
        channelConnectorMappingDraftByChannelID[normalizedChannelID] = draft

        if let updated = draft.first(where: { $0.connectorID == normalizedConnectorID }) {
            channelConnectorMappingActionStatusByChannelID[normalizedChannelID] =
                "Updated \(connectorDisplayName(normalizedConnectorID)) priority to \(updated.priority)."
        }
    }

    func inferConfigurationDraftKind(from value: String) -> ConfigurationDraftValueKind {
        let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmed.isEmpty {
            return .string
        }
        if parseBoolString(trimmed) != nil {
            return .bool
        }
        if trimmed.lowercased() == "null" {
            return .null
        }
        if Double(trimmed) != nil {
            return .number
        }
        return .string
    }

    func missingRequiredConfigurationFieldMessage(
        draft: [String: String],
        descriptors: [ConfigurationFieldDescriptorItem],
        subject: String
    ) -> String? {
        for descriptor in descriptors where descriptor.required && descriptor.editable && !descriptor.writeOnly {
            let value = draft[descriptor.key]?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
            if value.isEmpty {
                return "\(subject) configuration field `\(descriptor.label)` is required."
            }
        }
        return nil
    }

    func configurationMutationPayload(
        draft: [String: String],
        kindMap: [String: ConfigurationDraftValueKind]
    ) throws -> [String: DaemonConfigMutationValue] {
        var payload: [String: DaemonConfigMutationValue] = [:]
        for key in draft.keys.sorted() {
            let normalizedKey = key.trimmingCharacters(in: .whitespacesAndNewlines)
            guard !normalizedKey.isEmpty else {
                continue
            }
            let rawValue = draft[key] ?? ""
            let expectedKind = kindMap[key] ?? inferConfigurationDraftKind(from: rawValue)
            payload[normalizedKey] = try parseConfigurationMutationValue(
                rawValue,
                expectedKind: expectedKind,
                key: normalizedKey
            )
        }
        return payload
    }

    func synchronizeChannelConfigurationDrafts(with cards: [ChannelCardItem]) {
        let validIDs = Set(cards.map(\.id))
        var nextDrafts: [String: [String: String]] = [:]
        var nextKinds: [String: [String: ConfigurationDraftValueKind]] = [:]

        for card in cards {
            var mergedDraft = card.editableConfiguration
            if let existingDraft = channelConfigDraftByID[card.id] {
                for (key, value) in existingDraft where !key.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                    mergedDraft[key] = value
                }
            }

            var mergedKinds = card.editableConfigurationKinds
            if let existingKinds = channelConfigKindsByID[card.id] {
                for (key, kind) in existingKinds where mergedDraft[key] != nil {
                    mergedKinds[key] = kind
                }
            }
            for (key, value) in mergedDraft where mergedKinds[key] == nil {
                mergedKinds[key] = inferConfigurationDraftKind(from: value)
            }

            nextDrafts[card.id] = mergedDraft
            nextKinds[card.id] = mergedKinds
        }

        channelConfigDraftByID = nextDrafts
        channelConfigKindsByID = nextKinds
        channelConfigActionStatusByID = channelConfigActionStatusByID.filter { validIDs.contains($0.key) }
        channelLastTestResultByID = channelLastTestResultByID.filter { validIDs.contains($0.key) }
        channelConfigSaveInFlightIDs.formIntersection(validIDs)
        channelTestInFlightIDs.formIntersection(validIDs)
    }

    func synchronizeConnectorConfigurationDrafts(with cards: [ConnectorCardItem]) {
        let validIDs = Set(cards.map(\.id))
        var nextDrafts: [String: [String: String]] = [:]
        var nextKinds: [String: [String: ConfigurationDraftValueKind]] = [:]

        for card in cards {
            var mergedDraft = card.editableConfiguration
            if let existingDraft = connectorConfigDraftByID[card.id] {
                for (key, value) in existingDraft where !key.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                    mergedDraft[key] = value
                }
            }

            var mergedKinds = card.editableConfigurationKinds
            if let existingKinds = connectorConfigKindsByID[card.id] {
                for (key, kind) in existingKinds where mergedDraft[key] != nil {
                    mergedKinds[key] = kind
                }
            }
            for (key, value) in mergedDraft where mergedKinds[key] == nil {
                mergedKinds[key] = inferConfigurationDraftKind(from: value)
            }

            nextDrafts[card.id] = mergedDraft
            nextKinds[card.id] = mergedKinds
        }

        connectorConfigDraftByID = nextDrafts
        connectorConfigKindsByID = nextKinds
        connectorConfigActionStatusByID = connectorConfigActionStatusByID.filter { validIDs.contains($0.key) }
        connectorLastTestResultByID = connectorLastTestResultByID.filter { validIDs.contains($0.key) }
        connectorConfigSaveInFlightIDs.formIntersection(validIDs)
        connectorTestInFlightIDs.formIntersection(validIDs)
    }

    func applyChannelConfigurationFromDaemon(
        channelID: String,
        editable: [String: String],
        editableKinds: [String: ConfigurationDraftValueKind]
    ) {
        channelConfigDraftByID[channelID] = editable
        channelConfigKindsByID[channelID] = editableKinds
    }

    func applyConnectorConfigurationFromDaemon(
        connectorID: String,
        editable: [String: String],
        editableKinds: [String: ConfigurationDraftValueKind]
    ) {
        connectorConfigDraftByID[connectorID] = editable
        connectorConfigKindsByID[connectorID] = editableKinds
    }

    func mapConfigurationTestResult(
        from response: DaemonChannelTestOperationResponse,
        formattedWorkflowTimestamp: (String) -> String
    ) -> ConfigurationTestResultItem {
        ConfigurationTestResultItem(
            operation: nonEmpty(response.operation) ?? "health",
            success: response.success,
            status: nonEmpty(response.status) ?? "unknown",
            summary: nonEmpty(response.summary) ?? "No channel test summary returned.",
            checkedAtLabel: formattedWorkflowTimestamp(response.checkedAt),
            details: normalizeTestDetails(response.details)
        )
    }

    func mapConfigurationTestResult(
        from response: DaemonConnectorTestOperationResponse,
        formattedWorkflowTimestamp: (String) -> String
    ) -> ConfigurationTestResultItem {
        ConfigurationTestResultItem(
            operation: nonEmpty(response.operation) ?? "health",
            success: response.success,
            status: nonEmpty(response.status) ?? "unknown",
            summary: nonEmpty(response.summary) ?? "No connector test summary returned.",
            checkedAtLabel: formattedWorkflowTimestamp(response.checkedAt),
            details: normalizeTestDetails(response.details)
        )
    }

    func resetForMissingToken() {
        channelConnectorMappingFallbackPolicy = "priority_order"
        channelConnectorMappingsByChannelID = [:]
        channelConnectorMappingDraftByChannelID = [:]
        channelConnectorMappingActionStatusByChannelID = [:]
        channelConnectorMappingSaveInFlightChannelIDs = []
        channelConfigDraftByID = [:]
        channelConfigKindsByID = [:]
        channelConfigActionStatusByID = [:]
        channelConfigSaveInFlightIDs = []
        channelTestInFlightIDs = []
        channelLastTestResultByID = [:]
        channelDeliveryPoliciesByChannelID = [:]
        channelDeliveryPolicyDraftByID = [:]
        channelDeliveryPolicyActionStatusByID = [:]
        channelDeliveryPolicySaveInFlightIDs = []
        connectorConfigDraftByID = [:]
        connectorConfigKindsByID = [:]
        connectorConfigActionStatusByID = [:]
        connectorConfigSaveInFlightIDs = []
        connectorTestInFlightIDs = []
        connectorLastTestResultByID = [:]
        connectorPermissionActionStatusByID = [:]
        connectorPermissionRequestInFlightIDs = []
        connectorPermissionRefreshPendingIDs = []
    }

    private func parseConfigurationMutationValue(
        _ raw: String,
        expectedKind: ConfigurationDraftValueKind,
        key: String
    ) throws -> DaemonConfigMutationValue {
        let trimmed = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        switch expectedKind {
        case .string:
            return .string(raw)
        case .number:
            guard let value = Double(trimmed) else {
                throw ConfigurationMutationValidationError(
                    message: "Configuration field \(key) expects a number."
                )
            }
            return .number(value)
        case .bool:
            guard let value = parseBoolString(trimmed) else {
                throw ConfigurationMutationValidationError(
                    message: "Configuration field \(key) expects true or false."
                )
            }
            return .bool(value)
        case .null:
            if trimmed.isEmpty || trimmed.lowercased() == "null" {
                return .null
            }
            return inferredMutationValue(from: raw)
        case .object, .array:
            throw ConfigurationMutationValidationError(
                message: "Configuration field \(key) is read-only."
            )
        }
    }

    private func parseBoolString(_ raw: String) -> Bool? {
        switch raw.lowercased() {
        case "true", "yes", "1":
            return true
        case "false", "no", "0":
            return false
        default:
            return nil
        }
    }

    private func inferredMutationValue(from raw: String) -> DaemonConfigMutationValue {
        let trimmed = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        if let boolValue = parseBoolString(trimmed) {
            return .bool(boolValue)
        }
        if trimmed.lowercased() == "null" {
            return .null
        }
        if let numberValue = Double(trimmed) {
            return .number(numberValue)
        }
        return .string(raw)
    }

    private func normalizeTestDetails(_ details: DaemonUIStatusTestOperationDetails?) -> [String: String] {
        guard let details else {
            return [:]
        }
        let values = details.allValues
        guard !values.isEmpty else {
            return [:]
        }
        var normalized: [String: String] = [:]
        for key in values.keys.sorted() {
            normalized[key] = values[key]?.displayText ?? ""
        }
        return normalized
    }

    private func nonEmpty(_ value: String?) -> String? {
        let trimmed = value?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        return trimmed.isEmpty ? nil : trimmed
    }
}
