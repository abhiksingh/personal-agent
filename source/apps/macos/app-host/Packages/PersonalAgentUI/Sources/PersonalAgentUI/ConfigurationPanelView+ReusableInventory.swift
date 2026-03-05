import SwiftUI

extension ConfigurationPanelView {
    @ViewBuilder
    func configurationSecondaryStatusMessage(_ message: String?) -> some View {
        if let message {
            Text(message)
                .font(.caption)
                .foregroundStyle(.secondary)
        }
    }

    func configurationInventoryGroup<Item: Identifiable, RowContent: View>(
        title: String,
        items: [Item],
        isLoading: Bool,
        loadingMessage: String,
        emptyMessage: String,
        hasMore: Bool = false,
        hasMoreMessage: String? = nil,
        @ViewBuilder rowContent: @escaping (Item) -> RowContent
    ) -> some View {
        GroupBox(title) {
            VStack(alignment: .leading, spacing: 8) {
                if isLoading && items.isEmpty {
                    HStack(spacing: 8) {
                        ProgressView()
                            .controlSize(.small)
                        Text(loadingMessage)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                } else if items.isEmpty {
                    Text(emptyMessage)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                } else {
                    ForEach(Array(items.enumerated()), id: \.element.id) { index, item in
                        if index > 0 {
                            Divider()
                        }
                        rowContent(item)
                    }
                }

                if hasMore, let hasMoreMessage, !hasMoreMessage.isEmpty {
                    Text(hasMoreMessage)
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }
            }
        }
    }

    func configurationRecordCard<Content: View>(
        @ViewBuilder _ content: () -> Content
    ) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            content()
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(8)
        .cardSurface(.subtle)
    }
}
