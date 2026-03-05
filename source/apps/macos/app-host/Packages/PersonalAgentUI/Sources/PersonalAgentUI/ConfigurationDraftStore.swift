import Foundation
import Combine

@MainActor
final class ConfigurationDraftStore: ObservableObject {
    @Published var delegationFromActorID = "default"
    @Published var delegationToActorID = "default"
    @Published var delegationScopeType = "EXECUTION"
    @Published var delegationScopeKey = ""
    @Published var delegationExpiresAt = ""

    @Published var capabilityGrantDraftGrantID = ""
    @Published var capabilityGrantDraftActorID = "default"
    @Published var capabilityGrantDraftCapabilityKey = ""
    @Published var capabilityGrantDraftScopeJSON = "{}"
    @Published var capabilityGrantScopeEntries: [GuidedEditorScopeEntry] = []
    @Published var capabilityGrantScopeDraftKey = ""
    @Published var capabilityGrantScopeDraftValue = ""
    @Published var isCapabilityGrantRawScopeExpanded = false
    @Published var useCapabilityGrantRawScopeOverride = false
    @Published var capabilityGrantDraftStatus = "ACTIVE"
    @Published var capabilityGrantDraftExpiresAt = ""

    @Published var selectedTrustReceiptInventory: ConfigurationPanelView.TrustReceiptInventoryKind = .webhook

    func delegationActorOptions(principalOptions: [String]) -> [String] {
        var options = normalizedPrincipalOptions(principalOptions)

        if !options.contains(delegationFromActorID), !delegationFromActorID.isEmpty {
            options.append(delegationFromActorID)
        }
        if !options.contains(delegationToActorID), !delegationToActorID.isEmpty {
            options.append(delegationToActorID)
        }

        if options.isEmpty {
            options = ["default"]
        }

        return Array(Set(options)).sorted()
    }

    func seedDelegationDraftIfNeeded(principalOptions: [String]) {
        let options = delegationActorOptions(principalOptions: principalOptions)
        guard !options.isEmpty else {
            return
        }
        if !options.contains(delegationFromActorID) {
            delegationFromActorID = options.first ?? "default"
        }
        if !options.contains(delegationToActorID) || delegationToActorID == delegationFromActorID {
            if let alternate = options.first(where: { $0 != delegationFromActorID }) {
                delegationToActorID = alternate
            } else {
                delegationToActorID = options.first ?? "default"
            }
        }
    }

    func isDelegationGrantDisabled(isGrantInFlight: Bool, isLoading: Bool) -> Bool {
        let fromActor = delegationFromActorID.trimmingCharacters(in: .whitespacesAndNewlines)
        let toActor = delegationToActorID.trimmingCharacters(in: .whitespacesAndNewlines)
        if isGrantInFlight || isLoading {
            return true
        }
        if fromActor.isEmpty || toActor.isEmpty {
            return true
        }
        if fromActor == toActor {
            return true
        }
        return false
    }

    func delegationGrantInput() -> DelegationGrantInput {
        DelegationGrantInput(
            fromActorID: delegationFromActorID,
            toActorID: delegationToActorID,
            scopeType: delegationScopeType,
            scopeKey: delegationScopeType == "ALL" ? nil : delegationScopeKey,
            expiresAt: delegationExpiresAt
        )
    }

    func capabilityGrantActorOptions(principalOptions: [String]) -> [String] {
        var options = normalizedPrincipalOptions(principalOptions)
        if !capabilityGrantDraftActorID.isEmpty, !options.contains(capabilityGrantDraftActorID) {
            options.append(capabilityGrantDraftActorID)
        }
        if options.isEmpty {
            options = ["default"]
        }
        return Array(Set(options)).sorted()
    }

    func seedCapabilityGrantDraftIfNeeded(principalOptions: [String]) {
        let options = capabilityGrantActorOptions(principalOptions: principalOptions)
        guard !options.isEmpty else {
            return
        }
        if !options.contains(capabilityGrantDraftActorID) {
            capabilityGrantDraftActorID = options.first ?? "default"
        }
        if capabilityGrantDraftCapabilityKey.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            capabilityGrantDraftCapabilityKey = "messages_send_sms"
        }
        if capabilityGrantScopeEntries.isEmpty {
            capabilityGrantScopeEntries = GuidedEditorSupport.scopeEntries(from: capabilityGrantDraftScopeJSON) ?? []
        }
    }

    func isCapabilityGrantMutationDisabled(isInFlight: Bool) -> Bool {
        if isInFlight {
            return true
        }
        if capabilityGrantDraftActorID.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            return true
        }
        if capabilityGrantDraftCapabilityKey.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            return true
        }
        if useCapabilityGrantRawScopeOverride && !isCapabilityGrantRawScopeJSONValid {
            return true
        }
        return false
    }

    func capabilityGrantMutationInput() -> CapabilityGrantMutationInput {
        CapabilityGrantMutationInput(
            grantID: capabilityGrantDraftGrantID,
            actorID: capabilityGrantDraftActorID,
            capabilityKey: capabilityGrantDraftCapabilityKey,
            scopeJSON: useCapabilityGrantRawScopeOverride
                ? GuidedEditorSupport.normalizedRawJSONObject(capabilityGrantDraftScopeJSON)
                : GuidedEditorSupport.scopeJSON(from: capabilityGrantScopeEntries),
            status: capabilityGrantDraftStatus,
            expiresAt: capabilityGrantDraftExpiresAt
        )
    }

    func loadCapabilityGrantDraft(_ item: CapabilityGrantItem) {
        capabilityGrantDraftGrantID = item.id
        capabilityGrantDraftActorID = item.actorID
        capabilityGrantDraftCapabilityKey = item.capabilityKey
        capabilityGrantDraftScopeJSON = item.scopeJSON
        capabilityGrantScopeEntries = GuidedEditorSupport.scopeEntries(from: item.scopeJSON) ?? []
        capabilityGrantScopeDraftKey = ""
        capabilityGrantScopeDraftValue = ""
        useCapabilityGrantRawScopeOverride = false
        isCapabilityGrantRawScopeExpanded = false
        capabilityGrantDraftStatus = item.status.uppercased()
        capabilityGrantDraftExpiresAt = item.expiresAtRaw ?? ""
    }

    var isCapabilityGrantRawScopeJSONValid: Bool {
        GuidedEditorSupport.isValidRawJSONObject(capabilityGrantDraftScopeJSON)
    }

    var capabilityGrantRawScopeValidationMessage: String {
        isCapabilityGrantRawScopeJSONValid
            ? "Raw scope JSON is valid. Guided scope entries are ignored while override is enabled."
            : "Raw scope JSON must be a valid JSON object before save."
    }

    func scopeEntry(for id: UUID) -> GuidedEditorScopeEntry? {
        capabilityGrantScopeEntries.first(where: { $0.id == id })
    }

    func updateCapabilityGrantScopeEntry(_ id: UUID, key: String?, value: String?) {
        guard let index = capabilityGrantScopeEntries.firstIndex(where: { $0.id == id }) else {
            return
        }
        if let key {
            capabilityGrantScopeEntries[index].key = key
        }
        if let value {
            capabilityGrantScopeEntries[index].value = value
        }
        syncCapabilityGrantRawScopeFromGuided()
    }

    func addCapabilityGrantScopeEntry() {
        let key = capabilityGrantScopeDraftKey.trimmingCharacters(in: .whitespacesAndNewlines)
        let value = capabilityGrantScopeDraftValue.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !key.isEmpty, !value.isEmpty else {
            return
        }
        capabilityGrantScopeEntries.append(GuidedEditorScopeEntry(key: key, value: value))
        capabilityGrantScopeDraftKey = ""
        capabilityGrantScopeDraftValue = ""
        syncCapabilityGrantRawScopeFromGuided()
    }

    func removeCapabilityGrantScopeEntry(_ id: UUID) {
        capabilityGrantScopeEntries.removeAll { $0.id == id }
        syncCapabilityGrantRawScopeFromGuided()
    }

    func resetCapabilityGrantScopeEntries() {
        capabilityGrantScopeEntries = []
        capabilityGrantScopeDraftKey = ""
        capabilityGrantScopeDraftValue = ""
        syncCapabilityGrantRawScopeFromGuided()
    }

    func syncCapabilityGrantRawScopeFromGuided() {
        capabilityGrantDraftScopeJSON = GuidedEditorSupport.scopeJSON(from: capabilityGrantScopeEntries)
    }

    func applyCapabilityGrantRawScopeToGuided() -> Bool {
        guard isCapabilityGrantRawScopeJSONValid else {
            return false
        }
        capabilityGrantScopeEntries = GuidedEditorSupport.scopeEntries(from: capabilityGrantDraftScopeJSON) ?? []
        useCapabilityGrantRawScopeOverride = false
        return true
    }

    func delegationScopeSummary(scopeType: String, scopeKey: String?) -> String {
        let normalizedScopeType = scopeType.trimmingCharacters(in: .whitespacesAndNewlines).uppercased()
        let trimmedScopeKey = scopeKey?.trimmingCharacters(in: .whitespacesAndNewlines)
        if let trimmedScopeKey, !trimmedScopeKey.isEmpty {
            return "\(normalizedScopeType):\(trimmedScopeKey)"
        }
        return normalizedScopeType
    }

    private func normalizedPrincipalOptions(_ principalOptions: [String]) -> [String] {
        principalOptions
            .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
            .filter { !$0.isEmpty }
    }
}
