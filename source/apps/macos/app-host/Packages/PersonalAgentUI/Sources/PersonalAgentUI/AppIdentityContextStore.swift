import Foundation
import SwiftUI

@MainActor
final class AppIdentityContextStore: ObservableObject {
    enum WorkspaceContextUpdateIntent {
        case passiveResponse
        case identityDirectorySync
        case explicitSelection
    }

    @Published var workspaceID = "ws1"
    @Published var principalOptions: [String] = ["default"]
    @Published var selectedPrincipal = "default"
    @Published var isPrincipalOptionsLoading = false
    @Published var principalStatusMessage: String? = "Waiting for principal context."
    @Published var isIdentityDirectoryLoading = false
    @Published var identityStatusMessage: String? = "Identity directory has not been queried yet."
    @Published var identityWorkspaceItems: [IdentityWorkspaceItem] = []
    @Published var identityPrincipalItems: [IdentityPrincipalItem] = []
    @Published var identityActiveContext: IdentityActiveContextItem? = nil
    @Published var identityDeviceUserIDFilter = ""
    @Published var identityDeviceTypeFilter = ""
    @Published var identityDevicePlatformFilter = ""
    @Published var identityDeviceLimit = 25
    @Published var isIdentityDeviceInventoryLoading = false
    @Published var identityDeviceInventoryStatusMessage: String?
        = "Identity device inventory has not been queried yet."
    @Published var identityDeviceItems: [IdentityDeviceItem] = []
    @Published var identityDeviceInventoryHasMore = false
    @Published var identitySessionDeviceIDFilter = ""
    @Published var identitySessionUserIDFilter = ""
    @Published var identitySessionHealthFilter = "all"
    @Published var identitySessionLimit = 25
    @Published var isIdentitySessionInventoryLoading = false
    @Published var identitySessionInventoryStatusMessage: String?
        = "Identity session inventory has not been queried yet."
    @Published var identitySessionItems: [IdentitySessionItem] = []
    @Published var identitySessionInventoryHasMore = false
    @Published var identitySessionActionStatusByID: [String: String] = [:]
    @Published var identitySessionRevokeInFlightIDs: Set<String> = []

    private var workspaceContextPinnedByUser = false

    func configureInitialWorkspaceContext(
        envWorkspaceID: String?,
        storedWorkspaceID: String?,
        defaultWorkspaceID: String,
        canonicalWorkspaceID: (String?, Bool) -> String?,
        persistWorkspaceSelection: (String) -> Void
    ) {
        let normalizedEnvWorkspaceID = canonicalWorkspaceID(envWorkspaceID, false)
        let normalizedStoredWorkspaceID = canonicalWorkspaceID(storedWorkspaceID, false)

        if let normalizedEnvWorkspaceID, !normalizedEnvWorkspaceID.isEmpty {
            workspaceID = normalizedEnvWorkspaceID
            workspaceContextPinnedByUser = true
            return
        }
        if let normalizedStoredWorkspaceID, !normalizedStoredWorkspaceID.isEmpty {
            workspaceID = normalizedStoredWorkspaceID
            workspaceContextPinnedByUser = true
            persistWorkspaceSelection(normalizedStoredWorkspaceID)
            return
        }

        workspaceID = defaultWorkspaceID
        workspaceContextPinnedByUser = false
    }

    func updateWorkspaceContext(
        from daemonWorkspaceID: String?,
        intent: WorkspaceContextUpdateIntent = .passiveResponse,
        persistSelection: Bool = true,
        canonicalWorkspaceID: (String?, Bool) -> String?,
        applyWorkspaceScopedInformationDensityMode: (String) -> Void,
        persistWorkspaceSelection: (String) -> Void
    ) {
        guard let resolvedWorkspaceID = canonicalWorkspaceID(daemonWorkspaceID, false) else {
            return
        }

        switch intent {
        case .passiveResponse:
            return
        case .identityDirectorySync:
            guard !workspaceContextPinnedByUser else {
                return
            }
        case .explicitSelection:
            workspaceContextPinnedByUser = true
            if persistSelection {
                persistWorkspaceSelection(resolvedWorkspaceID)
            }
        }

        if workspaceID != resolvedWorkspaceID {
            workspaceID = resolvedWorkspaceID
        }
        applyWorkspaceScopedInformationDensityMode(resolvedWorkspaceID)
    }

    func resetIdentityDeviceInventoryFilters(defaultLimit: Int = 25) {
        identityDeviceUserIDFilter = ""
        identityDeviceTypeFilter = ""
        identityDevicePlatformFilter = ""
        identityDeviceLimit = defaultLimit
    }

    func resetIdentitySessionInventoryFilters(defaultLimit: Int = 25) {
        identitySessionDeviceIDFilter = ""
        identitySessionUserIDFilter = ""
        identitySessionHealthFilter = "all"
        identitySessionLimit = defaultLimit
    }

    func ensurePrincipalFallbackOptions() -> [String] {
        var options = Set<String>(["default"])
        if let selected = nonEmpty(selectedPrincipal) {
            options.insert(selected)
        }
        return options.sorted()
    }

    func principalOptionsForPrincipalSelection(including actorID: String?) -> [String] {
        var options = Set<String>(["default"])
        for option in principalOptions {
            let trimmed = option.trimmingCharacters(in: .whitespacesAndNewlines)
            if !trimmed.isEmpty {
                options.insert(trimmed)
            }
        }
        for principal in identityPrincipalItems {
            let trimmed = principal.id.trimmingCharacters(in: .whitespacesAndNewlines)
            if !trimmed.isEmpty {
                options.insert(trimmed)
            }
        }
        if let selected = nonEmpty(selectedPrincipal) {
            options.insert(selected)
        }
        if let actorID = nonEmpty(actorID) {
            options.insert(actorID)
        }
        return options.sorted()
    }

    func actingAsValidationMessage(for actorID: String) -> String? {
        let trimmedActorID = actorID.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmedActorID.isEmpty else {
            return "Select an acting-as principal before sending requests."
        }
        guard trimmedActorID != "default" else {
            return nil
        }

        let knownPrincipalIDs = Set(
            identityPrincipalItems
                .map(\.id)
                .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
                .filter { !$0.isEmpty }
        )
        guard !knownPrincipalIDs.isEmpty else {
            return nil
        }
        guard knownPrincipalIDs.contains(trimmedActorID) else {
            return "Selected acting-as principal `\(trimmedActorID)` is not in the active workspace directory. Refresh Identity Directory in Configuration."
        }
        return nil
    }

    func identityWorkspaceDisplayName(
        for workspaceID: String,
        canonicalWorkspaceID: (String?, Bool) -> String?
    ) -> String {
        guard let normalizedWorkspaceID = canonicalWorkspaceID(workspaceID, false) else {
            return "Unknown Workspace"
        }
        if let item = identityWorkspaceItems.first(where: { $0.id == normalizedWorkspaceID }),
           !item.name.isEmpty {
            return item.name
        }
        return normalizedWorkspaceID
    }

    func workspaceIdentityDisplayValue(
        for workspaceID: String?,
        canonicalWorkspaceID: (String?, Bool) -> String?,
        defaultWorkspaceID: String
    ) -> IdentityDisplayValue {
        guard let normalizedWorkspaceID = canonicalWorkspaceID(workspaceID, false) else {
            return IdentityDisplayValue(displayText: "Unknown Workspace", rawID: nil)
        }

        if let item = identityWorkspaceItems.first(
            where: { $0.id.caseInsensitiveCompare(normalizedWorkspaceID) == .orderedSame }
        ), let displayName = nonEmpty(item.name) {
            return IdentityDisplayValue(displayText: displayName, rawID: normalizedWorkspaceID)
        }

        if normalizedWorkspaceID.caseInsensitiveCompare(defaultWorkspaceID) == .orderedSame {
            return IdentityDisplayValue(displayText: "Default Workspace", rawID: normalizedWorkspaceID)
        }

        return IdentityDisplayValue(displayText: "Unrecognized Workspace", rawID: normalizedWorkspaceID)
    }

    func principalIdentityDisplayValue(for actorID: String?) -> IdentityDisplayValue {
        guard let normalizedActorID = normalizedIdentityID(actorID) else {
            return IdentityDisplayValue(displayText: "Unknown Principal", rawID: nil)
        }

        if let item = identityPrincipalItems.first(
            where: { $0.id.caseInsensitiveCompare(normalizedActorID) == .orderedSame }
        ), let displayName = nonEmpty(item.displayName) {
            return IdentityDisplayValue(displayText: displayName, rawID: normalizedActorID)
        }

        if normalizedActorID.caseInsensitiveCompare("default") == .orderedSame {
            return IdentityDisplayValue(displayText: "Default Principal", rawID: normalizedActorID)
        }
        if normalizedActorID.caseInsensitiveCompare("unknown") == .orderedSame {
            return IdentityDisplayValue(displayText: "Unknown Principal", rawID: nil)
        }

        return IdentityDisplayValue(displayText: "Unrecognized Principal", rawID: normalizedActorID)
    }

    private func normalizedIdentityID(_ value: String?) -> String? {
        let trimmed = value?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard !trimmed.isEmpty else {
            return nil
        }
        return trimmed
    }

    private func nonEmpty(_ value: String?) -> String? {
        let trimmed = value?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard !trimmed.isEmpty else {
            return nil
        }
        return trimmed
    }
}
