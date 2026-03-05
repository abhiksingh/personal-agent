import Foundation
import SwiftUI

@MainActor
final class AppModelsRouteStore: ObservableObject {
    @Published var providerReadinessItems: [ProviderReadinessItem] = []
    @Published var providerEndpointDraftByID: [String: String] = [:]
    @Published var providerAPIKeySecretNameDraftByID: [String: String] = [:]
    @Published var providerAPIKeySecretValueDraftByID: [String: String] = [:]
    @Published var providerSetupStatusByID: [String: String] = [:]
    @Published var providerSetupInFlightIDs: Set<String> = []
    @Published var providerCheckInFlightIDs: Set<String> = []
    @Published var modelCatalogStatusMessage: String? = "Waiting for model catalog."
    @Published var modelCatalogItems: [ModelCatalogEntryItem] = []
    @Published var modelPolicyItems: [ModelPolicyItem] = []
    @Published var modelMutationStatusByID: [String: String] = [:]
    @Published var modelMutationInFlightIDs: Set<String> = []
    @Published var modelCatalogManagementStatusByProviderID: [String: String] = [:]
    @Published var modelCatalogDiscoverInFlightProviderIDs: Set<String> = []
    @Published var modelCatalogManageInFlightProviderIDs: Set<String> = []
    @Published var discoveredModelsByProviderID: [String: [DiscoveredModelEntryItem]] = [:]
    @Published var modelManualAddDraftByProviderID: [String: String] = [:]
    @Published var isModelRoutePolicySaveInFlight = false
    @Published var modelRoutePolicySaveStatusMessage: String? = "No route policy changes submitted."
    @Published var modelRouteSimulationTaskClass = "chat"
    @Published var modelRouteSimulationPrincipalActorID = ""
    @Published var isModelRouteSimulationInFlight = false
    @Published var isModelRouteExplainInFlight = false
    @Published var modelRouteSimulationStatusMessage: String? = "No route simulation run yet."
    @Published var modelRouteExplainStatusMessage: String? = "No route explainability run yet."
    @Published var modelRouteSimulationResult: ModelRouteSimulationResultItem? = nil
    @Published var modelRouteExplainResult: ModelRouteExplainResultItem? = nil
    @Published var providerEndpointSourceByID: [String: String] = [:]
    @Published var providerSecretNameSourceByID: [String: String] = [:]

    func modelRouteSimulationTaskClassOptions(contextTaskClassOptions: [String]) -> [String] {
        var values = contextTaskClassOptions
        for taskClass in modelPolicyItems.map(\.taskClass) {
            let normalized = taskClass.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
            guard !normalized.isEmpty else {
                continue
            }
            if !values.contains(where: { $0.caseInsensitiveCompare(normalized) == .orderedSame }) {
                values.append(normalized)
            }
        }
        return values
    }

    func resetProviderEndpointDraft(
        providerID: String,
        normalizedProviderID: (String) -> String,
        providerDefaultEndpoints: [String: String]
    ) {
        let normalizedProvider = normalizedProviderID(providerID)
        providerEndpointDraftByID[normalizedProvider] = providerDefaultEndpoints[normalizedProvider] ?? ""
        providerSetupStatusByID[normalizedProvider] = "Endpoint reset to default. Save Provider to persist the change."
    }

    func providerEndpointDraft(
        for providerID: String,
        normalizedProviderID: (String) -> String,
        providerDefaultEndpoints: [String: String]
    ) -> String {
        let normalizedProvider = normalizedProviderID(providerID)
        return providerEndpointDraftByID[normalizedProvider]
            ?? providerDefaultEndpoints[normalizedProvider]
            ?? ""
    }

    func providerSecretNameDraft(
        for providerID: String,
        normalizedProviderID: (String) -> String,
        defaultProviderSecretName: (String) -> String
    ) -> String {
        let normalizedProvider = normalizedProviderID(providerID)
        return providerAPIKeySecretNameDraftByID[normalizedProvider]
            ?? defaultProviderSecretName(normalizedProvider)
    }

    func providerSecretValueDraft(
        for providerID: String,
        normalizedProviderID: (String) -> String
    ) -> String {
        providerAPIKeySecretValueDraftByID[normalizedProviderID(providerID)] ?? ""
    }

    func providerSetupHasDraftChanges(
        providerID: String,
        normalizedProviderID: (String) -> String,
        providerDefaultEndpoints: [String: String],
        defaultProviderSecretName: (String) -> String
    ) -> Bool {
        let normalizedProvider = normalizedProviderID(providerID)
        let endpointDraft = providerEndpointDraft(
            for: normalizedProvider,
            normalizedProviderID: normalizedProviderID,
            providerDefaultEndpoints: providerDefaultEndpoints
        )
        let endpointSource = providerEndpointSourceByID[normalizedProvider]
            ?? providerDefaultEndpoints[normalizedProvider]
            ?? ""
        let secretNameDraft = providerSecretNameDraft(
            for: normalizedProvider,
            normalizedProviderID: normalizedProviderID,
            defaultProviderSecretName: defaultProviderSecretName
        )
        let secretNameSource = providerSecretNameSourceByID[normalizedProvider]
            ?? defaultProviderSecretName(normalizedProvider)
        let secretValueDraft = providerSecretValueDraft(
            for: normalizedProvider,
            normalizedProviderID: normalizedProviderID
        )
        return endpointDraft != endpointSource ||
            secretNameDraft != secretNameSource ||
            !secretValueDraft.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }

    func dirtyProviderSetupIDs(
        providerReadinessProviders: [String],
        normalizedProviderID: (String) -> String,
        canonicalProviderOrder: [String],
        providerDefaultEndpoints: [String: String],
        defaultProviderSecretName: (String) -> String
    ) -> [String] {
        var providerIDs = Set(providerReadinessProviders.map(normalizedProviderID))
        providerIDs.formUnion(providerEndpointDraftByID.keys)
        providerIDs.formUnion(providerAPIKeySecretNameDraftByID.keys)
        providerIDs.formUnion(providerAPIKeySecretValueDraftByID.keys)
        providerIDs = Set(providerIDs.filter { canonicalProviderOrder.contains($0) })
        return providerIDs
            .filter {
                providerSetupHasDraftChanges(
                    providerID: $0,
                    normalizedProviderID: normalizedProviderID,
                    providerDefaultEndpoints: providerDefaultEndpoints,
                    defaultProviderSecretName: defaultProviderSecretName
                )
            }
            .sorted()
    }

    func resetProviderSetupDraft(
        providerID: String,
        normalizedProviderID: (String) -> String,
        providerDefaultEndpoints: [String: String],
        defaultProviderSecretName: (String) -> String
    ) {
        let normalizedProvider = normalizedProviderID(providerID)
        providerEndpointDraftByID[normalizedProvider] = providerEndpointSourceByID[normalizedProvider]
            ?? providerDefaultEndpoints[normalizedProvider]
            ?? ""
        providerAPIKeySecretNameDraftByID[normalizedProvider] = providerSecretNameSourceByID[normalizedProvider]
            ?? defaultProviderSecretName(normalizedProvider)
        providerAPIKeySecretValueDraftByID[normalizedProvider] = ""
    }

    func setProviderEndpointDraft(
        _ value: String,
        providerID: String,
        normalizedProviderID: (String) -> String
    ) {
        providerEndpointDraftByID[normalizedProviderID(providerID)] = value
    }

    func setProviderSecretNameDraft(
        _ value: String,
        providerID: String,
        normalizedProviderID: (String) -> String
    ) {
        providerAPIKeySecretNameDraftByID[normalizedProviderID(providerID)] = value
    }

    func setProviderSecretValueDraft(
        _ value: String,
        providerID: String,
        normalizedProviderID: (String) -> String
    ) {
        providerAPIKeySecretValueDraftByID[normalizedProviderID(providerID)] = value
    }

    func modelManualAddDraft(
        providerID: String,
        normalizedProviderID: (String) -> String
    ) -> String {
        modelManualAddDraftByProviderID[normalizedProviderID(providerID)] ?? ""
    }

    func setModelManualAddDraft(
        _ value: String,
        providerID: String,
        normalizedProviderID: (String) -> String
    ) {
        modelManualAddDraftByProviderID[normalizedProviderID(providerID)] = value
    }

    func applyActivePrincipalToModelRouteSimulation(selectedPrincipal: String) {
        modelRouteSimulationPrincipalActorID = selectedPrincipal.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    func clearModelRouteSimulationPrincipal() {
        modelRouteSimulationPrincipalActorID = ""
    }

    func resetModelRouteSimulationOutputs() {
        modelRouteSimulationResult = nil
        modelRouteExplainResult = nil
        modelRouteSimulationStatusMessage = "No route simulation run yet."
        modelRouteExplainStatusMessage = "No route explainability run yet."
    }

    func syncProviderSetupDrafts(
        configuredByProvider: [String: DaemonProviderConfigRecord],
        canonicalProviderOrder: [String],
        providerDefaultEndpoints: [String: String],
        providerRequiresAPIKey: (String) -> Bool,
        defaultProviderSecretName: (String) -> String
    ) {
        var endpointDrafts: [String: String] = [:]
        var secretNameDrafts: [String: String] = [:]

        for providerID in canonicalProviderOrder {
            let config = configuredByProvider[providerID]
            endpointDrafts[providerID] = nonEmpty(config?.endpoint)
                ?? providerDefaultEndpoints[providerID]
                ?? ""

            if providerRequiresAPIKey(providerID) {
                secretNameDrafts[providerID] = nonEmpty(config?.apiKeySecretName)
                    ?? defaultProviderSecretName(providerID)
            } else {
                secretNameDrafts[providerID] = nonEmpty(config?.apiKeySecretName) ?? ""
            }
        }

        providerEndpointDraftByID = endpointDrafts
        providerAPIKeySecretNameDraftByID = secretNameDrafts
        providerEndpointSourceByID = endpointDrafts
        providerSecretNameSourceByID = secretNameDrafts
        providerSetupStatusByID = providerSetupStatusByID.filter { endpointDrafts[$0.key] != nil }
        providerSetupInFlightIDs = Set(providerSetupInFlightIDs.filter { endpointDrafts[$0] != nil })
    }

    func resetProviderSetupDraftsToDefaults(
        defaultProviderEndpointDrafts: [String: String],
        defaultProviderSecretNameDrafts: [String: String]
    ) {
        providerEndpointDraftByID = defaultProviderEndpointDrafts
        providerAPIKeySecretNameDraftByID = defaultProviderSecretNameDrafts
        providerEndpointSourceByID = defaultProviderEndpointDrafts
        providerSecretNameSourceByID = defaultProviderSecretNameDrafts
        providerSetupInFlightIDs.removeAll()
        providerCheckInFlightIDs.removeAll()
    }

    func resetModelCatalogManagementState() {
        modelCatalogManagementStatusByProviderID.removeAll()
        modelCatalogDiscoverInFlightProviderIDs.removeAll()
        modelCatalogManageInFlightProviderIDs.removeAll()
        discoveredModelsByProviderID.removeAll()
        modelManualAddDraftByProviderID.removeAll()
    }

    func modelCatalogIdentifier(
        providerID: String,
        modelKey: String,
        normalizedProviderID: (String) -> String
    ) -> String {
        "\(normalizedProviderID(providerID))::\(modelKey)"
    }

    func mapModelCatalogRecord(_ record: DaemonModelCatalogRecord) -> ModelCatalogEntryItem {
        let endpoint = record.providerEndpoint?
            .trimmingCharacters(in: .whitespacesAndNewlines)
        let endpointLabel: String
        if let endpoint, endpoint.isEmpty == false {
            endpointLabel = endpoint
        } else {
            endpointLabel = "Provider endpoint unavailable"
        }

        return ModelCatalogEntryItem(
            id: "\(record.provider)::\(record.modelKey)",
            provider: record.provider,
            modelKey: record.modelKey,
            enabled: record.enabled,
            providerReady: record.providerReady,
            providerEndpoint: endpointLabel
        )
    }

    func mapDiscoveredModelRecord(
        _ record: DaemonModelDiscoverItem,
        modelCatalogIdentifier: (String, String) -> String
    ) -> DiscoveredModelEntryItem {
        let displayName = nonEmpty(record.displayName) ?? record.modelKey
        let source = nonEmpty(record.source) ?? "provider_discovery"
        return DiscoveredModelEntryItem(
            id: modelCatalogIdentifier(record.provider, record.modelKey),
            provider: record.provider,
            modelKey: record.modelKey,
            displayName: displayName,
            source: source,
            inCatalog: record.inCatalog,
            enabled: record.enabled
        )
    }

    func syncDiscoveredModelCatalogFlags(
        providerID: String? = nil,
        normalizedProviderID: (String) -> String,
        modelCatalogIdentifier: (String, String) -> String
    ) {
        let providers: [String]
        if let providerID {
            providers = [normalizedProviderID(providerID)]
        } else {
            providers = Array(discoveredModelsByProviderID.keys)
        }

        for provider in providers {
            guard let items = discoveredModelsByProviderID[provider] else {
                continue
            }
            discoveredModelsByProviderID[provider] = items.map { item in
                let modelID = modelCatalogIdentifier(item.provider, item.modelKey)
                let catalogMatch = modelCatalogItems.first { $0.id.caseInsensitiveCompare(modelID) == .orderedSame }
                return DiscoveredModelEntryItem(
                    id: item.id,
                    provider: item.provider,
                    modelKey: item.modelKey,
                    displayName: item.displayName,
                    source: item.source,
                    inCatalog: catalogMatch != nil,
                    enabled: catalogMatch?.enabled ?? false
                )
            }
        }
    }

    func mapModelPolicyRecord(
        _ record: DaemonModelRoutingPolicyRecord,
        parseDaemonTimestamp: (String) -> Date?
    ) -> ModelPolicyItem {
        let updatedAtLabel: String
        if let parsed = parseDaemonTimestamp(record.updatedAt) {
            updatedAtLabel = parsed.formatted(date: .abbreviated, time: .shortened)
        } else {
            updatedAtLabel = nonEmpty(record.updatedAt) ?? "n/a"
        }
        return ModelPolicyItem(
            id: "\(record.taskClass)::\(record.provider)::\(record.modelKey)",
            taskClass: record.taskClass,
            provider: record.provider,
            modelKey: record.modelKey,
            updatedAtLabel: updatedAtLabel
        )
    }

    func mapModelRouteSimulationResponse(
        _ response: DaemonModelRouteSimulationResponse,
        workspaceID: String
    ) -> ModelRouteSimulationResultItem {
        ModelRouteSimulationResultItem(
            workspaceID: nonEmpty(response.workspaceID) ?? workspaceID,
            taskClass: nonEmpty(response.taskClass) ?? "chat",
            principalActorID: nonEmpty(response.principalActorID),
            selectedProvider: nonEmpty(response.selectedProvider) ?? "unknown",
            selectedModelKey: nonEmpty(response.selectedModelKey) ?? "unknown",
            selectedSource: nonEmpty(response.selectedSource) ?? "unknown",
            notes: nonEmpty(response.notes),
            reasonCodes: response.reasonCodes.compactMap { nonEmpty($0) },
            decisions: mapModelRouteDecisionItems(response.decisions),
            fallbackChain: mapModelRouteFallbackItems(response.fallbackChain)
        )
    }

    func mapModelRouteExplainResponse(
        _ response: DaemonModelRouteExplainResponse,
        workspaceID: String
    ) -> ModelRouteExplainResultItem {
        ModelRouteExplainResultItem(
            workspaceID: nonEmpty(response.workspaceID) ?? workspaceID,
            taskClass: nonEmpty(response.taskClass) ?? "chat",
            principalActorID: nonEmpty(response.principalActorID),
            selectedProvider: nonEmpty(response.selectedProvider) ?? "unknown",
            selectedModelKey: nonEmpty(response.selectedModelKey) ?? "unknown",
            selectedSource: nonEmpty(response.selectedSource) ?? "unknown",
            summary: nonEmpty(response.summary) ?? "No explainability summary returned.",
            explanations: response.explanations.compactMap { nonEmpty($0) },
            reasonCodes: response.reasonCodes.compactMap { nonEmpty($0) },
            decisions: mapModelRouteDecisionItems(response.decisions),
            fallbackChain: mapModelRouteFallbackItems(response.fallbackChain)
        )
    }

    func modelRouteSimulationSummaryMessage(
        _ item: ModelRouteSimulationResultItem,
        providerDisplayName: (String) -> String
    ) -> String {
        let principalLabel = item.principalActorID ?? "auto principal"
        return "Simulated \(item.taskClass) route for \(principalLabel): \(providerDisplayName(item.selectedProvider)) • \(item.selectedModelKey)."
    }

    func modelRouteExplainSummaryMessage(_ item: ModelRouteExplainResultItem) -> String {
        let principalLabel = item.principalActorID ?? "auto principal"
        return "Loaded explainability trace for \(item.taskClass) (\(principalLabel))."
    }

    private func mapModelRouteDecisionItems(
        _ records: [DaemonModelRouteDecision]
    ) -> [ModelRouteDecisionTraceItem] {
        records.enumerated().map { index, record in
            let idSeed = [
                nonEmpty(record.step) ?? "step",
                nonEmpty(record.decision) ?? "decision",
                nonEmpty(record.reasonCode) ?? "reason",
                nonEmpty(record.provider) ?? "provider",
                nonEmpty(record.modelKey) ?? "model",
                "\(index)"
            ].joined(separator: "::")
            return ModelRouteDecisionTraceItem(
                id: idSeed,
                step: nonEmpty(record.step) ?? "n/a",
                decision: nonEmpty(record.decision) ?? "n/a",
                reasonCode: nonEmpty(record.reasonCode) ?? "unknown",
                provider: nonEmpty(record.provider),
                modelKey: nonEmpty(record.modelKey),
                note: nonEmpty(record.note)
            )
        }
    }

    private func mapModelRouteFallbackItems(
        _ records: [DaemonModelRouteFallbackDecision]
    ) -> [ModelRouteFallbackTraceItem] {
        records.map { record in
            ModelRouteFallbackTraceItem(
                id: "\(max(record.rank, 0))::\(record.provider)::\(record.modelKey)",
                rank: max(record.rank, 0),
                provider: nonEmpty(record.provider) ?? "unknown",
                modelKey: nonEmpty(record.modelKey) ?? "unknown",
                selected: record.selected,
                reasonCode: nonEmpty(record.reasonCode) ?? "unknown"
            )
        }
        .sorted { lhs, rhs in
            if lhs.rank == rhs.rank {
                return lhs.id.localizedCaseInsensitiveCompare(rhs.id) == .orderedAscending
            }
            return lhs.rank < rhs.rank
        }
    }

    private func nonEmpty(_ value: String?) -> String? {
        let trimmed = value?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        return trimmed.isEmpty ? nil : trimmed
    }
}
