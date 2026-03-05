import SwiftUI

struct CommandPaletteView: View {
    @ObservedObject var state: AppShellState
    @FocusState private var isSearchFieldFocused: Bool

    private var normalizedQuery: String {
        state.commandPaletteSearchQuery
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .lowercased()
    }

    private var isSearching: Bool {
        !normalizedQuery.isEmpty
    }

    private var filteredItems: [AppCommandActionItem] {
        state.rankedAppCommandActionItems(for: state.commandPaletteSearchQuery)
    }

    private var objectItems: [CommandPaletteObjectSearchItem] {
        state.rankedCommandPaletteObjectItems(for: state.commandPaletteSearchQuery)
    }

    private var recentItems: [AppCommandActionItem] {
        guard !isSearching else {
            return []
        }
        let recentIDs = Set(state.recentAppCommandActionIDs)
        return filteredItems.filter { recentIDs.contains($0.actionID) }
    }

    private var groupedItems: [(AppCommandActionCategory, [AppCommandActionItem])] {
        guard !isSearching else {
            return []
        }
        let recentIDs = Set(recentItems.map(\.actionID))
        return AppCommandActionCategory.allCases.compactMap { category -> (AppCommandActionCategory, [AppCommandActionItem])? in
            let items = filteredItems.filter { $0.category == category && !recentIDs.contains($0.actionID) }
            guard !items.isEmpty else {
                return nil
            }
            return (category, items)
        }
    }

    private var firstEnabledItem: AppCommandActionItem? {
        state.firstEnabledAppCommandAction(for: state.commandPaletteSearchQuery)
    }

    private var firstObjectItem: CommandPaletteObjectSearchItem? {
        state.firstCommandPaletteObjectMatch(for: state.commandPaletteSearchQuery)
    }

    var body: some View {
        NavigationStack {
            List {
                Section {
                    TextField("Search commands and objects", text: $state.commandPaletteSearchQuery)
                        .textFieldStyle(.roundedBorder)
                        .focused($isSearchFieldFocused)
                        .accessibilityLabel(UIAccessibilityContract.commandPaletteSearchLabel)
                        .accessibilityHint(UIAccessibilityContract.commandPaletteSearchHint)
                        .accessibilityIdentifier(UIAccessibilityContract.commandPaletteSearchIdentifier)
                        .onSubmit {
                            runFirstEnabledMatch()
                        }
                }

                if isSearching {
                    if !objectItems.isEmpty {
                        Section("Objects") {
                            ForEach(objectItems) { item in
                                objectRow(item)
                            }
                        }
                    }

                    if !filteredItems.isEmpty {
                        Section("Commands") {
                            ForEach(filteredItems) { item in
                                commandRow(item)
                            }
                        }
                    }
                } else if !recentItems.isEmpty {
                    Section("Recent") {
                        ForEach(recentItems) { item in
                            commandRow(item)
                        }
                    }
                }

                if !isSearching {
                    ForEach(groupedItems, id: \.0) { category, items in
                        Section(category.title) {
                            ForEach(items) { item in
                                commandRow(item)
                            }
                        }
                    }
                }
            }
            .navigationTitle("Command Palette")
            .onAppear {
                DispatchQueue.main.async {
                    isSearchFieldFocused = true
                }
            }
            .onDisappear {
                isSearchFieldFocused = false
            }
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("Run First Match") {
                        runFirstEnabledMatch()
                    }
                    .disabled(firstEnabledItem == nil)
                }
                ToolbarItem(placement: .cancellationAction) {
                    Button("Close") {
                        state.dismissCommandPalette()
                    }
                }
            }
            .overlay {
                if filteredItems.isEmpty && objectItems.isEmpty {
                    ContentUnavailableView(
                        "No Matching Results",
                        systemImage: "magnifyingglass",
                        description: Text("Try a broader query or clear search.")
                    )
                }
            }
        }
    }

    @ViewBuilder
    private func commandRow(_ item: AppCommandActionItem) -> some View {
        Button {
            run(item)
        } label: {
            HStack(alignment: .top, spacing: 10) {
                Label(item.title, systemImage: item.symbolName)
                    .labelStyle(.titleAndIcon)
                    .font(.body)
                    .frame(maxWidth: .infinity, alignment: .leading)

                if let shortcutHint = item.shortcutHint {
                    Text(shortcutHint)
                        .font(.caption.monospaced())
                        .foregroundStyle(.secondary)
                }
            }
        }
        .disabled(!item.isEnabled)
        .accessibilityLabel(item.title)
        .accessibilityHint(item.subtitle ?? item.category.title)

        if let subtitle = item.subtitle, !subtitle.isEmpty {
            Text(subtitle)
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        if !item.isEnabled, let disabledReason = item.disabledReason, !disabledReason.isEmpty {
            Text(disabledReason)
                .font(.caption2)
                .foregroundStyle(.secondary)
        }
    }

    @ViewBuilder
    private func objectRow(_ item: CommandPaletteObjectSearchItem) -> some View {
        Button {
            run(item)
        } label: {
            HStack(alignment: .top, spacing: 10) {
                Label(item.title, systemImage: item.symbolName)
                    .labelStyle(.titleAndIcon)
                    .font(.body)
                    .frame(maxWidth: .infinity, alignment: .leading)
                Text(item.kind.title)
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }
        }
        .accessibilityLabel(item.title)
        .accessibilityHint(item.subtitle ?? item.kind.title)

        if let subtitle = item.subtitle, !subtitle.isEmpty {
            Text(subtitle)
                .font(.caption)
                .foregroundStyle(.secondary)
        }
    }

    private func run(_ item: AppCommandActionItem) {
        state.dismissCommandPalette()
        state.performAppCommand(item.actionID)
    }

    private func run(_ item: CommandPaletteObjectSearchItem) {
        state.dismissCommandPalette()
        state.performCommandPaletteObjectAction(item.target)
    }

    private func runFirstEnabledMatch() {
        if let firstObjectItem {
            run(firstObjectItem)
            return
        }
        guard let firstEnabledItem else {
            return
        }
        run(firstEnabledItem)
    }
}
