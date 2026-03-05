import SwiftUI

struct PanelFilterBarCard<Content: View>: View {
    let summaryText: String?
    @ViewBuilder let content: Content

    init(
        summaryText: String? = nil,
        @ViewBuilder content: () -> Content
    ) {
        self.summaryText = summaryText
        self.content = content()
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            content

            if let summaryText,
               !summaryText.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                Text(summaryText)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
        .padding(10)
        .cardSurface(.subtle)
    }
}

struct PanelScaffoldView<Header: View, FilterBar: View, Supplementary: View, Content: View>: View {
    let activeFilterSummaryParts: [String]
    let clearFiltersButtonTitle: String
    let clearFiltersAction: (() -> Void)?
    let runtimeBannerMessage: RuntimeStateBannerMessage?
    let showFilterBar: Bool
    @ViewBuilder let header: Header
    @ViewBuilder let filterBar: FilterBar
    @ViewBuilder let supplementary: Supplementary
    @ViewBuilder let content: Content

    init(
        activeFilterSummaryParts: [String] = [],
        clearFiltersButtonTitle: String = "Clear Filters",
        clearFiltersAction: (() -> Void)? = nil,
        runtimeBannerMessage: RuntimeStateBannerMessage? = nil,
        showFilterBar: Bool = true,
        @ViewBuilder header: () -> Header,
        @ViewBuilder filterBar: () -> FilterBar,
        @ViewBuilder supplementary: () -> Supplementary,
        @ViewBuilder content: () -> Content
    ) {
        self.activeFilterSummaryParts = activeFilterSummaryParts
        self.clearFiltersButtonTitle = clearFiltersButtonTitle
        self.clearFiltersAction = clearFiltersAction
        self.runtimeBannerMessage = runtimeBannerMessage
        self.showFilterBar = showFilterBar
        self.header = header()
        self.filterBar = filterBar()
        self.supplementary = supplementary()
        self.content = content()
    }

    var body: some View {
        VStack(spacing: 0) {
            header
                .padding(.horizontal, UIStyle.panelPadding)
                .padding(.vertical, 12)

            if let clearFiltersAction, !activeFilterSummaryParts.isEmpty {
                PanelActiveFilterIndicatorView(
                    summaryParts: activeFilterSummaryParts,
                    clearButtonTitle: clearFiltersButtonTitle,
                    clearAction: clearFiltersAction
                )
                .padding(.horizontal, UIStyle.panelPadding)
                .padding(.bottom, 12)
            }

            if let runtimeBannerMessage {
                RuntimeStateBanner(message: runtimeBannerMessage)
                    .padding(.horizontal, UIStyle.panelPadding)
                    .padding(.bottom, 12)
            }

            if showFilterBar {
                filterBar
                    .padding(.horizontal, UIStyle.panelPadding)
                    .padding(.bottom, 12)
            }

            supplementary

            Divider()

            content
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(UIStyle.panelGradient)
    }
}
