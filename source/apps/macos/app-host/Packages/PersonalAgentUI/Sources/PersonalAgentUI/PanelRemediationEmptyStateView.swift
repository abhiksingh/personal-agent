import SwiftUI

struct PanelRemediationEmptyStateView: View {
    let title: String
    let systemImage: String
    let description: String
    let statusMessage: String?
    let actions: [EmptyStateRemediationAction]
    let onAction: (EmptyStateRemediationActionID) -> Void

    init(
        title: String,
        systemImage: String,
        description: String,
        statusMessage: String? = nil,
        headerStatusMessage: String? = nil,
        actions: [EmptyStateRemediationAction] = [],
        onAction: @escaping (EmptyStateRemediationActionID) -> Void
    ) {
        self.title = title
        self.systemImage = systemImage
        self.description = description
        self.statusMessage = Self.dedupedStatusMessage(
            statusMessage: statusMessage,
            headerStatusMessage: headerStatusMessage
        )
        self.actions = actions
        self.onAction = onAction
    }

    var body: some View {
        ContentUnavailableView {
            Label(title, systemImage: systemImage)
        } description: {
            VStack(alignment: .center, spacing: 6) {
                Text(description)
                if let statusMessage {
                    Text(statusMessage)
                        .foregroundStyle(.secondary)
                }
            }
        } actions: {
            ForEach(actions) { action in
                emptyStateActionButton(action)
            }
        }
    }

    @ViewBuilder
    private func emptyStateActionButton(_ action: EmptyStateRemediationAction) -> some View {
        if action.isProminent {
            Button {
                onAction(action.actionID)
            } label: {
                Label(action.title, systemImage: action.symbolName)
            }
            .buttonStyle(.borderedProminent)
            .disabled(action.isDisabled)
        } else {
            Button {
                onAction(action.actionID)
            } label: {
                Label(action.title, systemImage: action.symbolName)
            }
            .buttonStyle(.bordered)
            .disabled(action.isDisabled)
        }
    }

    private static func dedupedStatusMessage(
        statusMessage: String?,
        headerStatusMessage: String?
    ) -> String? {
        guard let normalizedStatus = normalizedStatusToken(statusMessage) else {
            return nil
        }
        if let normalizedHeader = normalizedStatusToken(headerStatusMessage),
           normalizedStatus.caseInsensitiveCompare(normalizedHeader) == .orderedSame {
            return nil
        }
        return normalizedStatus
    }

    private static func normalizedStatusToken(_ value: String?) -> String? {
        let trimmed = value?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        return trimmed.isEmpty ? nil : trimmed
    }
}
