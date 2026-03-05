import SwiftUI

struct InspectPanelView: View {
    @ObservedObject private var state: AppShellState
    @State private var selectedInspectMode: InspectPresentationMode = .activity
    @State private var selectedStatusFilter: InspectStatusFilter = .all
    @State private var selectedMetadataFilterScope: InspectMetadataFilterScope = .all
    @State private var selectedGrouping: InspectGroupingMode = .none
    @State private var metadataFilterText = ""

    init(state: AppShellState) {
        self.state = state
    }

    private enum InspectStatusFilter: String, CaseIterable, Identifiable {
        case all
        case success
        case running
        case failure

        var id: String { rawValue }

        var label: String {
            switch self {
            case .all:
                return "All Statuses"
            case .success:
                return "Success"
            case .running:
                return "Running"
            case .failure:
                return "Failure"
            }
        }

        func matches(_ status: InspectLogStatus) -> Bool {
            switch self {
            case .all:
                return true
            case .success:
                return status == .success
            case .running:
                return status == .running
            case .failure:
                return status == .failure
            }
        }
    }

    private enum InspectMetadataFilterScope: String, CaseIterable, Identifiable {
        case all
        case task
        case run
        case correlation
        case provider
        case model

        var id: String { rawValue }

        var label: String {
            switch self {
            case .all:
                return "All Fields"
            case .task:
                return "Task"
            case .run:
                return "Run"
            case .correlation:
                return "Correlation"
            case .provider:
                return "Provider"
            case .model:
                return "Model"
            }
        }
    }

    private enum InspectGroupingMode: String, CaseIterable, Identifiable {
        case none
        case task
        case run
        case correlation
        case provider
        case model

        var id: String { rawValue }

        var label: String {
            switch self {
            case .none:
                return "No Grouping"
            case .task:
                return "Group by Task"
            case .run:
                return "Group by Run"
            case .correlation:
                return "Group by Correlation"
            case .provider:
                return "Group by Provider"
            case .model:
                return "Group by Model"
            }
        }
    }

    private struct InspectGroupBucket: Identifiable {
        let id: String
        let title: String
        let items: [InspectLogItem]
    }

    private var isTraceMode: Bool {
        selectedInspectMode == .trace
    }

    private var isGalleryMode: Bool {
        selectedInspectMode == .gallery
    }

    private var filteredLogs: [InspectLogItem] {
        guard !isGalleryMode else {
            return []
        }
        return state.inspectLogs.filter { log in
            selectedStatusFilter.matches(log.status) && matchesMetadataFilter(log)
        }
    }

    private var metadataFilterScopeForFiltering: InspectMetadataFilterScope {
        isTraceMode ? selectedMetadataFilterScope : .all
    }

    private var groupingModeForDisplay: InspectGroupingMode {
        isTraceMode ? selectedGrouping : .none
    }

    private var groupedLogs: [InspectGroupBucket] {
        guard groupingModeForDisplay != .none else {
            return []
        }

        var buckets: [String: [InspectLogItem]] = [:]
        var orderedKeys: [String] = []
        for log in filteredLogs {
            let key = groupKey(for: log)
            if buckets[key] == nil {
                orderedKeys.append(key)
            }
            buckets[key, default: []].append(log)
        }

        return orderedKeys.map { key in
            InspectGroupBucket(
                id: key,
                title: key,
                items: buckets[key] ?? []
            )
        }
    }

    private var filterSummaryLabel: String {
        if isGalleryMode {
            return "Gallery mode renders deterministic component references for shared visual baselines."
        }
        if state.inspectLogs.isEmpty {
            return "Filters apply after inspect logs are loaded."
        }
        return "Showing \(filteredLogs.count) of \(state.inspectLogs.count) inspect rows."
    }

    private var activeFilterSummaryParts: [String] {
        guard !isGalleryMode else {
            return []
        }
        return InspectFilterContext(
            metadataFilterText: metadataFilterText,
            statusFilterRawValue: selectedStatusFilter.rawValue,
            metadataScopeRawValue: metadataFilterScopeForFiltering.rawValue,
            groupingRawValue: groupingModeForDisplay.rawValue,
            inspectModeRawValue: selectedInspectMode.rawValue
        ).activeFilterSummaryParts
    }

    private var showLoadingSkeleton: Bool {
        guard !isGalleryMode else {
            return false
        }
        return (state.isInspectLoading || !state.hasLoadedInspectLogs) && state.inspectLogs.isEmpty
    }

    var body: some View {
        PanelScaffoldView(
            activeFilterSummaryParts: activeFilterSummaryParts,
            clearFiltersButtonTitle: "Clear Filters",
            clearFiltersAction: resetFilters,
            runtimeBannerMessage: runtimeBannerMessage,
            showFilterBar: !isGalleryMode,
            header: { header },
            filterBar: { filterToolbar },
            supplementary: { EmptyView() },
            content: { content }
        )
        .onAppear {
            applyPersistedFilterContext()
            applyExternalSearchSeedIfNeeded()
        }
        .onChange(of: state.workspaceLabel) { _, _ in
            applyPersistedFilterContext()
        }
        .onChange(of: state.inspectSearchSeed) { _, _ in
            applyExternalSearchSeedIfNeeded()
        }
        .onChange(of: selectedStatusFilter) { _, _ in
            persistFilterContext()
        }
        .onChange(of: selectedInspectMode) { _, _ in
            persistFilterContext()
        }
        .onChange(of: selectedMetadataFilterScope) { _, _ in
            persistFilterContext()
        }
        .onChange(of: selectedGrouping) { _, _ in
            persistFilterContext()
        }
        .onChange(of: metadataFilterText) { _, _ in
            persistFilterContext()
        }
    }

    private var runtimeBannerMessage: RuntimeStateBannerMessage? {
        RuntimeStateBannerMessage.resolve(
            daemonStatus: state.daemonStatus,
            connectionStatus: state.connectionStatus,
            detail: state.daemonStatusDetail,
            hasLoadedDaemonStatus: state.hasLoadedDaemonStatus
        )
    }

    private var header: some View {
        TahoeSectionHeader(
            title: "Inspect",
            subtitle: isGalleryMode
                ? "Canonical shared UI component references and snapshot anchors."
                : (state.inspectStatusMessage ?? "Newest-first daemon operational logs")
        ) {
            HStack(spacing: 8) {
                if state.isInspectLoading && !isGalleryMode {
                    ProgressView()
                        .controlSize(.small)
                }

                if let focusedRunID = state.inspectFocusedRunID, !isGalleryMode {
                    TahoeStatusBadge(
                        text: "Run \(shortIdentifier(focusedRunID))",
                        symbolName: "line.3.horizontal.decrease.circle.fill",
                        tint: .blue
                    )

                    Button("Clear Run Filter") {
                        state.clearInspectRunFocus()
                        state.refreshInspectLogs()
                    }
                    .panelActionStyle(.secondary)
                    .controlSize(.small)
                }

                Picker("View", selection: $selectedInspectMode) {
                    Text("Activity").tag(InspectPresentationMode.activity)
                    Text("Trace").tag(InspectPresentationMode.trace)
                    Text("Gallery").tag(InspectPresentationMode.gallery)
                }
                .pickerStyle(.segmented)
                .frame(maxWidth: 260)
                .accessibilityIdentifier("inspect-mode-picker")

                if isGalleryMode {
                    Button("Back to Activity") {
                        selectedInspectMode = .activity
                    }
                    .panelActionStyle(.secondary)
                    .controlSize(.small)
                    .accessibilityIdentifier("inspect-mode-activity-button")
                } else {
                    Button("Open Gallery") {
                        selectedInspectMode = .gallery
                    }
                    .panelActionStyle(.secondary)
                    .controlSize(.small)
                    .accessibilityIdentifier("inspect-mode-gallery-button")
                }

                if !isGalleryMode {
                    Picker("Status", selection: $selectedStatusFilter) {
                        ForEach(InspectStatusFilter.allCases) { filter in
                            Text(filter.label).tag(filter)
                        }
                    }
                    .pickerStyle(.menu)
                    .frame(maxWidth: 164)

                    Button(state.isInspectLiveTailEnabled ? "Pause Tail" : "Resume Tail") {
                        state.setInspectLiveTailEnabled(!state.isInspectLiveTailEnabled)
                    }
                    .panelActionStyle(.secondary)
                    .disabled(state.isInspectLoading)

                    Button {
                        state.refreshInspectLogs()
                    } label: {
                        Label("Refresh", systemImage: "arrow.clockwise")
                    }
                    .panelActionStyle(.secondary)
                }
            }
        }
    }

    private var filterToolbar: some View {
        PanelFilterBarCard(summaryText: filterSummaryLabel) {
            HStack(spacing: 8) {
                HStack(spacing: 6) {
                    Image(systemName: "magnifyingglass")
                        .foregroundStyle(.secondary)
                    TextField(
                        isTraceMode
                            ? "Filter task/run/correlation/provider/model"
                            : "Search activity by event, task, run, or summary",
                        text: $metadataFilterText
                    )
                    .textFieldStyle(.plain)
                }
                .padding(.horizontal, 10)
                .padding(.vertical, 8)
                .background(
                    RoundedRectangle(cornerRadius: UIStyle.controlCornerRadius, style: .continuous)
                        .fill(Color(nsColor: .textBackgroundColor).opacity(0.82))
                )

                Button("Clear Filters") {
                    resetFilters()
                }
                .panelActionStyle(.secondary)
                .disabled(activeFilterSummaryParts.isEmpty)
            }

            HStack(spacing: 8) {
                if isTraceMode {
                    Picker("Match", selection: $selectedMetadataFilterScope) {
                        ForEach(InspectMetadataFilterScope.allCases) { scope in
                            Text(scope.label).tag(scope)
                        }
                    }
                    .pickerStyle(.menu)
                    .frame(maxWidth: 170)

                    Picker("Group", selection: $selectedGrouping) {
                        ForEach(InspectGroupingMode.allCases) { grouping in
                            Text(grouping.label).tag(grouping)
                        }
                    }
                    .pickerStyle(.menu)
                    .frame(maxWidth: 170)
                }

                Spacer(minLength: 0)
            }
        }
    }

    @ViewBuilder
    private var content: some View {
        if isGalleryMode {
            InspectComponentGalleryView()
        } else if showLoadingSkeleton {
            PanelLoadingSkeletonView(
                title: "Loading Inspect",
                subtitle: "Fetching newest-first operational logs and trace metadata.",
                rowCount: 4
            )
            .frame(maxWidth: .infinity, maxHeight: .infinity)
        } else if !filteredLogs.isEmpty {
            ScrollView {
                LazyVStack(spacing: UIStyle.standardSpacing) {
                    if groupingModeForDisplay == .none {
                        ForEach(filteredLogs) { log in
                            inspectRowCard(log)
                        }
                    } else {
                        ForEach(groupedLogs) { group in
                            groupedLogSection(group)
                        }
                    }
                }
                .padding(UIStyle.panelPadding)
            }
            .background(UIStyle.panelGradient)
        } else if !state.inspectLogs.isEmpty {
            ContentUnavailableView {
                Label(
                    isTraceMode ? "No Trace Rows Match Filter" : "No Activity Matches Filter",
                    systemImage: "line.3.horizontal.decrease.circle"
                )
            } description: {
                Text(
                    isTraceMode
                        ? "Adjust trace filters or grouping options to show rows from this inspect snapshot."
                        : "Adjust activity filters to show events from this inspect snapshot."
                )
            } actions: {
                Button("Clear Filters") {
                    resetFilters()
                }
                .panelActionStyle(.primary)
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)
            .background(UIStyle.panelGradient)
        } else {
            PanelRemediationEmptyStateView(
                title: isTraceMode ? "No Trace Rows Yet" : "No Activity Yet",
                systemImage: "doc.text.magnifyingglass",
                description: {
                    if let focusedRunID = state.inspectFocusedRunID {
                        return isTraceMode
                            ? "No trace entries found for run \(focusedRunID)."
                            : "No activity entries found for run \(focusedRunID)."
                    }
                    return isTraceMode
                        ? "Trace rows from daemon-backed workflows will appear here when available."
                        : "Workflow activity updates will appear here when available."
                }(),
                statusMessage: state.inspectStatusMessage,
                headerStatusMessage: state.inspectStatusMessage,
                actions: state.inspectEmptyStateRemediationActions
            ) { actionID in
                state.performEmptyStateRemediationAction(actionID)
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)
            .background(UIStyle.panelGradient)
        }
    }

    private func groupedLogSection(_ group: InspectGroupBucket) -> some View {
        GroupBox {
            VStack(alignment: .leading, spacing: 10) {
                HStack(spacing: 8) {
                    Text(group.title)
                        .font(.subheadline.weight(.semibold))
                    Spacer(minLength: 0)
                    Text("\(group.items.count)")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }

                ForEach(group.items) { item in
                    inspectRowCard(item)
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
        .groupBoxStyle(.automatic)
    }

    @ViewBuilder
    private func inspectRowCard(_ log: InspectLogItem) -> some View {
        if isTraceMode {
            traceLogCard(log)
        } else {
            activityLogCard(log)
        }
    }

    private func activityLogCard(_ log: InspectLogItem) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                TahoeStatusBadge(
                    text: log.status.label,
                    symbolName: log.status.symbolName,
                    tint: log.status.tint
                )
                Spacer()
                Text(log.timestamp.formatted(date: .omitted, time: .standard))
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            Text(activityEventTitle(for: log))
                .font(.headline)

            Text(activityEventSummary(for: log))
                .font(.callout)
                .foregroundStyle(.secondary)
                .lineLimit(4)

            if let contextSummary = activityContextSummary(for: log) {
                Text(contextSummary)
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .textSelection(.enabled)
            }

            if log.hasCrossViewContext {
                HStack(spacing: 8) {
                    Button("Open Related Tasks") {
                        state.openTasksForInspectLog(log)
                    }
                    .panelActionStyle(.secondary)
                    .controlSize(.small)

                    Button("Open Related Approvals") {
                        state.openApprovalsForInspectLog(log)
                    }
                    .panelActionStyle(.secondary)
                    .controlSize(.small)
                }
            }
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .cardSurface(.standard)
    }

    private func traceLogCard(_ log: InspectLogItem) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack(alignment: .firstTextBaseline) {
                TahoeStatusBadge(
                    text: log.status.label,
                    symbolName: log.status.symbolName,
                    tint: log.status.tint
                )
                Spacer()
                Text(log.timestamp.formatted(date: .omitted, time: .standard))
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            Text(log.event)
                .font(.headline)

            if let identifierSummary = identifierSummaryLine(for: log) {
                Text(identifierSummary)
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .textSelection(.enabled)
            }

            if let routeSummary = routeSummaryLine(for: log) {
                Text(routeSummary)
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .textSelection(.enabled)
            }

            detailsBlock(label: "Input", value: log.inputSummary)
            detailsBlock(label: "Output", value: log.outputSummary)
            detailsBlock(label: "Metadata", value: log.metadataSummary, isSecondary: true)

            if log.hasCrossViewContext {
                HStack(spacing: 8) {
                    Button("Open Related Tasks") {
                        state.openTasksForInspectLog(log)
                    }
                    .panelActionStyle(.secondary)
                    .controlSize(.small)

                    Button("Open Related Approvals") {
                        state.openApprovalsForInspectLog(log)
                    }
                    .panelActionStyle(.secondary)
                    .controlSize(.small)
                }
            }
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .cardSurface(.standard)
    }

    private func detailsBlock(label: String, value: String, isSecondary: Bool = false) -> some View {
        VStack(alignment: .leading, spacing: 5) {
            Text(label)
                .font(.caption.weight(.semibold))
                .foregroundStyle(.secondary)
                .textCase(.uppercase)
            Text(value)
                .font(isSecondary ? .caption : .callout)
                .foregroundStyle(isSecondary ? .secondary : .primary)
                .textSelection(.enabled)
        }
    }

    private func activityEventTitle(for log: InspectLogItem) -> String {
        log.event
            .replacingOccurrences(of: ".", with: " ")
            .replacingOccurrences(of: "_", with: " ")
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .capitalized
    }

    private func activityEventSummary(for log: InspectLogItem) -> String {
        if let output = nonEmpty(log.outputSummary),
           output != "No output summary." {
            return output
        }
        if let input = nonEmpty(log.inputSummary),
           input != "No input summary." {
            return input
        }
        return "No event summary available."
    }

    private func activityContextSummary(for log: InspectLogItem) -> String? {
        var parts: [String] = []
        if let taskID = nonEmpty(log.taskID) {
            parts.append("Task \(shortIdentifier(taskID, limit: 18))")
        }
        if let runID = nonEmpty(log.runID) {
            parts.append("Run \(shortIdentifier(runID, limit: 18))")
        }
        if let provider = nonEmpty(log.route.provider),
           let model = nonEmpty(log.route.modelKey) {
            parts.append("\(provider) • \(model)")
        }
        return parts.isEmpty ? nil : parts.joined(separator: "  •  ")
    }

    private func matchesMetadataFilter(_ log: InspectLogItem) -> Bool {
        let query = metadataFilterText.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !query.isEmpty else {
            return true
        }

        let normalizedQuery = query.lowercased()
        switch metadataFilterScopeForFiltering {
        case .all:
            return fieldValues(for: log).contains { value in
                value.lowercased().contains(normalizedQuery)
            }
        case .task:
            return contains(query: normalizedQuery, in: log.taskID)
        case .run:
            return contains(query: normalizedQuery, in: log.runID)
        case .correlation:
            return contains(query: normalizedQuery, in: log.correlationID)
        case .provider:
            return contains(query: normalizedQuery, in: log.route.provider)
        case .model:
            return contains(query: normalizedQuery, in: log.route.modelKey)
        }
    }

    private func fieldValues(for log: InspectLogItem) -> [String] {
        var values: [String?] = [
            log.event,
            log.taskID,
            log.runID,
            log.stepID,
            log.correlationID,
            log.route.taskClass,
            log.route.provider,
            log.route.modelKey,
        ]
        if isTraceMode {
            values.append(contentsOf: [
                log.route.taskClassSource,
                log.route.routeSource,
                log.route.notes,
                log.metadataSummary
            ])
        }
        return values.compactMap { nonEmpty($0) }
    }

    private func contains(query: String, in value: String?) -> Bool {
        guard let value = nonEmpty(value) else {
            return false
        }
        return value.lowercased().contains(query)
    }

    private func groupKey(for log: InspectLogItem) -> String {
        switch groupingModeForDisplay {
        case .none:
            return "All Logs"
        case .task:
            return nonEmpty(log.taskID).map { "Task \(shortIdentifier($0, limit: 18))" } ?? "Task Unassigned"
        case .run:
            return nonEmpty(log.runID).map { "Run \(shortIdentifier($0, limit: 18))" } ?? "Run Unassigned"
        case .correlation:
            return nonEmpty(log.correlationID).map { "Correlation \(shortIdentifier($0, limit: 18))" } ?? "Correlation Unassigned"
        case .provider:
            return nonEmpty(log.route.provider).map { "Provider \($0)" } ?? "Provider Unassigned"
        case .model:
            return nonEmpty(log.route.modelKey).map { "Model \($0)" } ?? "Model Unassigned"
        }
    }

    private func identifierSummaryLine(for log: InspectLogItem) -> String? {
        var parts: [String] = []
        if let taskID = nonEmpty(log.taskID) {
            parts.append("task \(shortIdentifier(taskID))")
        }
        if let runID = nonEmpty(log.runID) {
            parts.append("run \(shortIdentifier(runID))")
        }
        if let stepID = nonEmpty(log.stepID) {
            parts.append("step \(shortIdentifier(stepID))")
        }
        if let correlationID = nonEmpty(log.correlationID) {
            parts.append("corr \(shortIdentifier(correlationID))")
        }
        return parts.isEmpty ? nil : parts.joined(separator: " • ")
    }

    private func routeSummaryLine(for log: InspectLogItem) -> String? {
        guard log.route.available else {
            return nil
        }
        var parts: [String] = []
        if let taskClass = nonEmpty(log.route.taskClass) {
            parts.append("class \(taskClass)")
        }
        if let provider = nonEmpty(log.route.provider), let model = nonEmpty(log.route.modelKey) {
            parts.append("route \(provider) • \(model)")
        } else if let provider = nonEmpty(log.route.provider) {
            parts.append("provider \(provider)")
        } else if let model = nonEmpty(log.route.modelKey) {
            parts.append("model \(model)")
        }
        if isTraceMode,
           let routeSource = nonEmpty(log.route.routeSource) {
            parts.append("source \(routeSource)")
        }
        return parts.isEmpty ? nil : parts.joined(separator: " • ")
    }

    private func applyExternalSearchSeedIfNeeded() {
        guard let seed = state.inspectSearchSeed?
            .trimmingCharacters(in: .whitespacesAndNewlines),
              !seed.isEmpty else {
            return
        }
        metadataFilterText = seed
        selectedMetadataFilterScope = .all
        state.inspectSearchSeed = nil
    }

    private func resetFilters() {
        let reset = InspectFilterContext(
            inspectModeRawValue: selectedInspectMode.rawValue
        )
        applyFilterContext(reset)
        state.updateInspectFilterContext(reset)
    }

    private func applyPersistedFilterContext() {
        applyFilterContext(state.inspectFilterContext())
    }

    private func applyFilterContext(_ context: InspectFilterContext) {
        metadataFilterText = context.metadataFilterText
        selectedStatusFilter = InspectStatusFilter(rawValue: context.statusFilterRawValue) ?? .all
        selectedMetadataFilterScope = InspectMetadataFilterScope(rawValue: context.metadataScopeRawValue) ?? .all
        selectedGrouping = InspectGroupingMode(rawValue: context.groupingRawValue) ?? .none
        selectedInspectMode = context.inspectMode
    }

    private func persistFilterContext() {
        state.updateInspectFilterContext(
            InspectFilterContext(
                metadataFilterText: metadataFilterText,
                statusFilterRawValue: selectedStatusFilter.rawValue,
                metadataScopeRawValue: selectedMetadataFilterScope.rawValue,
                groupingRawValue: selectedGrouping.rawValue,
                inspectModeRawValue: selectedInspectMode.rawValue
            )
        )
    }

    private func nonEmpty(_ value: String?) -> String? {
        guard let value else {
            return nil
        }
        let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? nil : trimmed
    }

    private func shortIdentifier(_ value: String, limit: Int = 12) -> String {
        if value.count <= limit {
            return value
        }
        let index = value.index(value.startIndex, offsetBy: limit)
        return "\(value[..<index])…"
    }
}
