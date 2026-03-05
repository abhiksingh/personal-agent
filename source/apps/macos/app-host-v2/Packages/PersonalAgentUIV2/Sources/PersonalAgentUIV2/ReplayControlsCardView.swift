import SwiftUI

struct ReplayControlsCardView: View {
    @ObservedObject var store: AppShellV2Store
    private let filterControlWidth: CGFloat = 190

    var body: some View {
        PASurfaceCard(tone: .neutral) {
            filterRow
                .controlSize(.small)
                .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private var filterRow: some View {
        ViewThatFits(in: .horizontal) {
            HStack(spacing: V2WorkflowLayout.compactSpacing) {
                statusControl
                sourceControl
                searchField
            }

            VStack(alignment: .leading, spacing: V2WorkflowLayout.compactSpacing) {
                HStack(spacing: V2WorkflowLayout.compactSpacing) {
                    statusControl
                    sourceControl
                }
                searchField
            }

            VStack(alignment: .leading, spacing: V2WorkflowLayout.compactSpacing) {
                statusControl
                sourceControl
                searchField
            }
        }
    }

    private var statusControl: some View {
        statusMenu
            .buttonStyle(.bordered)
            .frame(width: filterControlWidth, alignment: .leading)
            .fixedSize(horizontal: false, vertical: true)
            .accessibilityLabel("Status Filter")
            .accessibilityHint("Filter replay by instruction status.")
            .accessibilityIdentifier("v2-replay-status-filter")
    }

    private var sourceControl: some View {
        sourceMenu
            .buttonStyle(.bordered)
            .frame(width: filterControlWidth, alignment: .leading)
            .fixedSize(horizontal: false, vertical: true)
            .accessibilityLabel("Source Filter")
            .accessibilityHint("Filter replay by source channel.")
            .accessibilityIdentifier("v2-replay-source-filter")
    }

    private var searchField: some View {
        TextField("Search instruction, intent, channel, or action", text: $store.searchQuery)
            .textFieldStyle(.plain)
            .paInputSurface()
            .foregroundStyle(Color.paTextPrimary)
            .frame(minWidth: 180, maxWidth: .infinity)
            .accessibilityLabel("Replay Search")
            .accessibilityHint("Search instructions, intent, channel, or action.")
            .accessibilityIdentifier("v2-replay-search-field")
    }

    private var sourceMenu: some View {
        Menu {
            Button {
                store.selectedSources.removeAll()
            } label: {
                HStack {
                    Text("All Sources")
                    Spacer(minLength: 8)
                    if store.selectedSources.isEmpty || store.selectedSources.count == ReplaySource.allCases.count {
                        Image(systemName: "checkmark")
                    }
                }
            }

            Divider()

            ForEach(ReplaySource.allCases) { source in
                Button {
                    store.toggleSource(source)
                } label: {
                    HStack {
                        Label(source.label, systemImage: source.systemImage)
                        Spacer(minLength: 8)
                        if store.selectedSources.contains(source) {
                            Image(systemName: "checkmark")
                        }
                    }
                }
            }
        } label: {
            Label(sourceMenuLabel, systemImage: "line.3.horizontal.decrease.circle")
                .font(.system(size: 11, weight: .semibold, design: .rounded))
        }
    }

    private var statusMenu: some View {
        Menu {
            Button {
                store.statusFilter = .all
            } label: {
                HStack {
                    Text("All Statuses")
                    Spacer(minLength: 8)
                    if store.statusFilter == .all {
                        Image(systemName: "checkmark")
                    }
                }
            }

            Divider()

            ForEach(ReplayStatusFilter.allCases.filter { $0 != .all }) { filter in
                Button {
                    store.statusFilter = filter
                } label: {
                    HStack {
                        Text(statusLabel(for: filter))
                        Spacer(minLength: 8)
                        if store.statusFilter == filter {
                            Image(systemName: "checkmark")
                        }
                    }
                }
            }
        } label: {
            Label(statusMenuLabel, systemImage: "line.3.horizontal.decrease.circle")
                .font(.system(size: 11, weight: .semibold, design: .rounded))
        }
    }

    private var sourceMenuLabel: String {
        let count = store.selectedSources.count
        if count == 1, let source = store.selectedSources.first {
            return source.label
        }
        if count == 0 || count == ReplaySource.allCases.count {
            return "Source"
        }
        return "\(count) Sources"
    }

    private var statusMenuLabel: String {
        if store.statusFilter == .all {
            return "Status"
        }
        return statusLabel(for: store.statusFilter)
    }

    private func statusLabel(for filter: ReplayStatusFilter) -> String {
        if filter == .waiting, store.metrics.waiting > 0 {
            return "Needs Attention (\(store.metrics.waiting))"
        }
        if filter == .needsApproval, store.metrics.needsApproval > 0 {
            return "Needs Approval (\(store.metrics.needsApproval))"
        }
        return filter.label
    }
}
