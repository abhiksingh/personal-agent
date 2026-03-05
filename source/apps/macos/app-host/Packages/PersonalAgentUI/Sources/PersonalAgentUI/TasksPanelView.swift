import SwiftUI

struct TasksPanelView: View {
    @ObservedObject private var state: AppShellState
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @FocusState private var isSearchFieldFocused: Bool
    @State private var searchText = ""
    @State private var selectedStateFilter = "All States"
    @State private var selectedPriorityFilter: TaskPriorityFilter = .all
    @State private var selectedPrincipalFilter = "All Principals"
    @State private var isAutoRefreshEnabled = false
    @State private var selectedAutoRefreshInterval: AutoRefreshInterval = .thirtySeconds
    @State private var autoRefreshTask: Task<Void, Never>? = nil
    @State private var lastAutoRefreshAt: Date? = nil
    @State private var isPresentingTaskSubmitSheet = false
    @State private var taskSubmitTitleDraft = ""
    @State private var taskSubmitDescriptionDraft = ""
    @State private var taskSubmitPriorityDraft: TaskSubmitPriority = .medium
    @State private var taskSubmitTaskClassDraft = "chat"
    @State private var taskSubmitRequestedByActorDraft = "default"
    @State private var taskSubmitSubjectPrincipalDraft = "default"
    @State private var taskSubmitContextOverrideExpanded = false
    @State private var detailsExpandedByID: [String: Bool] = [:]
    @State private var isResetContextConfirmationPresented = false

    init(state: AppShellState) {
        self.state = state
    }

    var body: some View {
        PanelScaffoldView(
            activeFilterSummaryParts: activeFilterSummaryParts,
            clearFiltersButtonTitle: "Clear Filters",
            clearFiltersAction: clearFilters,
            runtimeBannerMessage: runtimeBannerMessage,
            header: { header },
            filterBar: { filterToolbar },
            supplementary: { supplementaryCards },
            content: { content }
        )
        .sheet(item: taskRunDetailBinding) { detail in
            taskRunDetailSheet(detail)
                .frame(minWidth: 760, minHeight: 560)
        }
        .sheet(isPresented: $isPresentingTaskSubmitSheet) {
            taskSubmitSheet
                .frame(minWidth: 560, minHeight: 420)
        }
        .confirmationDialog(
            "Reset task context for this workspace?",
            isPresented: $isResetContextConfirmationPresented
        ) {
            Button("Reset Context", role: .destructive) {
                resetWorkspaceContinuityContext()
            }
            Button("Cancel", role: .cancel) {}
        } message: {
            Text("This clears saved task draft, panel filters, and continuity state for the current workspace.")
        }
        .onAppear {
            applyPersistedFilterContext()
            normalizeFilterSelections()
            applyExternalSearchSeedIfNeeded()
            focusSearchFieldForKeyboardTraversal()
            prepareTaskSubmitDraft()
            applyPersistedTaskSubmitDraftContext()
            applyExternalTaskSubmitDraftIfNeeded()
            restartAutoRefreshLoop()
        }
        .onDisappear {
            stopAutoRefreshLoop()
        }
        .onChange(of: state.workspaceLabel) { _, _ in
            applyPersistedFilterContext()
            normalizeFilterSelections()
            applyPersistedTaskSubmitDraftContext()
            applyExternalTaskSubmitDraftIfNeeded()
        }
        .onChange(of: state.taskRunItems.map(\.id)) { _, _ in
            normalizeFilterSelections()
        }
        .onChange(of: state.tasksSearchSeed) { _, _ in
            applyExternalSearchSeedIfNeeded()
        }
        .onChange(of: state.taskSubmitDraftSeed?.id) { _, _ in
            applyExternalTaskSubmitDraftIfNeeded()
        }
        .onChange(of: taskSubmitDraftPersistenceSignature) { _, _ in
            persistTaskSubmitDraftContext()
        }
        .onChange(of: searchText) { _, _ in
            persistFilterContext()
        }
        .onChange(of: selectedStateFilter) { _, _ in
            persistFilterContext()
        }
        .onChange(of: selectedPriorityFilter) { _, _ in
            persistFilterContext()
        }
        .onChange(of: selectedPrincipalFilter) { _, _ in
            persistFilterContext()
        }
        .onChange(of: isAutoRefreshEnabled) { _, _ in
            restartAutoRefreshLoop()
        }
        .onChange(of: selectedAutoRefreshInterval) { _, _ in
            restartAutoRefreshLoop()
        }
        .onChange(of: state.taskSubmissionPrincipalOptions) { _, _ in
            normalizeTaskSubmitPrincipalDrafts()
        }
        .onChange(of: state.latestTaskSubmissionReceipt?.id) { _, _ in
            closeTaskSubmitSheetIfSuccessful()
        }
    }

    @ViewBuilder
    private var supplementaryCards: some View {
        if let panelProblemRemediation {
            PanelProblemRemediationCardView(context: panelProblemRemediation) { actionID in
                state.performPanelProblemRemediationAction(actionID, section: .tasks)
            }
            .padding(.horizontal, UIStyle.panelPadding)
            .padding(.bottom, 12)
        }
        if let latestReceipt = state.latestTaskSubmissionReceipt {
            latestTaskSubmissionCard(latestReceipt)
                .padding(.horizontal, UIStyle.panelPadding)
                .padding(.bottom, 12)
        }
    }

    private enum TaskPriorityFilter: String, CaseIterable, Identifiable {
        case all
        case high
        case medium
        case low

        var id: String { rawValue }

        var label: String {
            switch self {
            case .all:
                return "All Priorities"
            case .high:
                return "Priority High"
            case .medium:
                return "Priority Medium"
            case .low:
                return "Priority Low"
            }
        }

        func matches(priority: Int) -> Bool {
            switch self {
            case .all:
                return true
            case .high:
                return priority >= 3
            case .medium:
                return priority == 2
            case .low:
                return priority < 2
            }
        }
    }

    private enum AutoRefreshInterval: String, CaseIterable, Identifiable {
        case fifteenSeconds
        case thirtySeconds
        case sixtySeconds

        var id: String { rawValue }

        var seconds: Int {
            switch self {
            case .fifteenSeconds:
                return 15
            case .thirtySeconds:
                return 30
            case .sixtySeconds:
                return 60
            }
        }

        var label: String {
            "\(seconds)s"
        }
    }

    private enum TaskSubmitPriority: String, CaseIterable, Identifiable {
        case high
        case medium
        case low

        var id: String { rawValue }

        var label: String {
            rawValue.capitalized
        }

        var annotationLine: String? {
            switch self {
            case .high:
                return "Priority: High"
            case .medium:
                return nil
            case .low:
                return "Priority: Low"
            }
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

    private var panelProblemRemediation: PanelProblemRemediationContext? {
        state.panelProblemRemediation(for: .tasks)
    }

    private var isAdvancedInformationDensityEnabled: Bool {
        state.isAdvancedInformationDensityEnabled
    }

    private var header: some View {
        TahoeSectionHeader(
            title: "Tasks",
            subtitle: state.tasksStatusMessage ?? "Task progress and run status"
        ) {
            HStack(spacing: 8) {
                if state.isTasksLoading {
                    ProgressView()
                        .controlSize(.small)
                }
                if state.isTaskRunDetailLoading {
                    ProgressView()
                        .controlSize(.small)
                }
                if state.isTaskSubmitInFlight {
                    ProgressView()
                        .controlSize(.small)
                }

                Button {
                    prepareTaskSubmitDraft()
                    isPresentingTaskSubmitSheet = true
                } label: {
                    Label("New Task", systemImage: "plus")
                }
                .panelActionStyle(.primary)
                .disabled(state.isTaskSubmitInFlight)
                .accessibilityLabel("Create new task")
                .accessibilityIdentifier("tasks-new-button")

                Button {
                    state.refreshTaskRunList()
                } label: {
                    Label("Refresh", systemImage: "arrow.clockwise")
                }
                .panelActionStyle(.secondary)
                .disabled(state.isTasksLoading)
                .accessibilityLabel("Refresh task run list")
            }
        }
    }

    private var filterToolbar: some View {
        PanelFilterBarCard {
            HStack(spacing: 8) {
                HStack(spacing: 6) {
                    Image(systemName: "magnifyingglass")
                        .foregroundStyle(.secondary)
                        .accessibilityHidden(true)
                    TextField(taskSearchPlaceholder, text: $searchText)
                        .textFieldStyle(.plain)
                        .focused($isSearchFieldFocused)
                        .accessibilityLabel(UIAccessibilityContract.tasksSearchLabel)
                        .accessibilityHint(UIAccessibilityContract.tasksSearchHint)
                        .accessibilityIdentifier(UIAccessibilityContract.tasksSearchIdentifier)
                }
                .padding(.horizontal, 10)
                .padding(.vertical, 8)
                .background(
                    RoundedRectangle(cornerRadius: UIStyle.controlCornerRadius, style: .continuous)
                        .fill(Color(nsColor: .textBackgroundColor).opacity(0.82))
                )

                Button("Clear Filters") {
                    clearFilters()
                }
                .panelActionStyle(.secondary)
                .disabled(activeFilterSummaryParts.isEmpty)

                Button("Reset Context") {
                    isResetContextConfirmationPresented = true
                }
                .panelActionStyle(.destructive)
            }

            HStack(spacing: 8) {
                Picker("State", selection: $selectedStateFilter) {
                    ForEach(stateFilterOptions, id: \.self) { option in
                        Text(option).tag(option)
                    }
                }
                .pickerStyle(.menu)
                .frame(maxWidth: 210)

                Picker("Priority", selection: $selectedPriorityFilter) {
                    ForEach(TaskPriorityFilter.allCases) { option in
                        Text(option.label).tag(option)
                    }
                }
                .pickerStyle(.menu)
                .frame(maxWidth: 180)

                Picker("Principal", selection: $selectedPrincipalFilter) {
                    ForEach(principalFilterOptions, id: \.self) { option in
                        Text(principalFilterDisplayName(option)).tag(option)
                    }
                }
                .pickerStyle(.menu)
                .frame(maxWidth: 240)

                Spacer(minLength: 0)

                Toggle("Auto-refresh", isOn: $isAutoRefreshEnabled)
                    .toggleStyle(.switch)
                    .controlSize(.small)

                if isAutoRefreshEnabled {
                    Picker("Interval", selection: $selectedAutoRefreshInterval) {
                        ForEach(AutoRefreshInterval.allCases) { option in
                            Text(option.label).tag(option)
                        }
                    }
                    .pickerStyle(.menu)
                    .frame(maxWidth: 112)
                }
            }

            HStack(spacing: 8) {
                Text(filteredSummaryLabel)
                    .font(.caption)
                    .foregroundStyle(.secondary)

                if state.isTasksLoading {
                    ProgressView()
                        .controlSize(.small)
                    Text("Refreshing task list…")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }

                if isAutoRefreshEnabled {
                    Text(autoRefreshStatusLabel)
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }
            }
        }
    }

    private var stateFilterOptions: [String] {
        let states = Set(state.taskRunItems.map(\.effectiveState.label))
        return ["All States"] + states.sorted { lhs, rhs in
            lhs.localizedCaseInsensitiveCompare(rhs) == .orderedAscending
        }
    }

    private var principalFilterOptions: [String] {
        let principals = Set(
            state.taskRunItems.flatMap {
                [$0.requestedByActorID, $0.subjectPrincipalActorID, $0.actingAsActorID]
            }.filter { !$0.isEmpty }
        )
        return ["All Principals"] + principals.sorted { lhs, rhs in
            principalFilterDisplayName(lhs).localizedCaseInsensitiveCompare(
                principalFilterDisplayName(rhs)
            ) == .orderedAscending
        }
    }

    private var filteredSummaryLabel: String {
        guard !state.taskRunItems.isEmpty else {
            return "Filters apply after tasks are loaded."
        }
        return "Showing \(filteredTaskRunItems.count) of \(state.taskRunItems.count) task/run rows."
    }

    private var activeFilterSummaryParts: [String] {
        var parts = TasksFilterContext(
            searchText: searchText,
            stateFilter: selectedStateFilter,
            priorityFilterRawValue: selectedPriorityFilter.rawValue,
            principalFilter: selectedPrincipalFilter
        ).activeFilterSummaryParts
        let principalPrefix = "Principal: "
        if selectedPrincipalFilter != "All Principals",
           let index = parts.firstIndex(where: { $0.hasPrefix(principalPrefix) }) {
            parts[index] = "\(principalPrefix)\(principalFilterDisplayName(selectedPrincipalFilter))"
        }
        return parts
    }

    private func focusSearchFieldForKeyboardTraversal() {
        DispatchQueue.main.async {
            isSearchFieldFocused = true
        }
    }

    private var autoRefreshStatusLabel: String {
        if let lastAutoRefreshAt {
            return "Auto-refresh \(selectedAutoRefreshInterval.label) • last \(lastAutoRefreshAt.formatted(date: .omitted, time: .standard))"
        }
        return "Auto-refresh \(selectedAutoRefreshInterval.label)"
    }

    private var filteredTaskRunItems: [TaskRunListRowItem] {
        state.taskRunItems.filter { item in
            matchesStateFilter(item) &&
            selectedPriorityFilter.matches(priority: item.priority) &&
            matchesPrincipalFilter(item) &&
            matchesSearch(item)
        }
    }

    @ViewBuilder
    private var content: some View {
        if showLoadingSkeleton {
            PanelLoadingSkeletonView(
                title: "Loading Tasks",
                subtitle: "Fetching tasks and run progress.",
                rowCount: 4
            )
            .frame(maxWidth: .infinity, maxHeight: .infinity)
        } else if state.taskRunItems.isEmpty {
            PanelRemediationEmptyStateView(
                title: "No Tasks Yet",
                systemImage: "list.bullet.rectangle.portrait",
                description: "Submitted work appears here with progress, owner context, and timestamps.",
                statusMessage: state.tasksStatusMessage,
                headerStatusMessage: state.tasksStatusMessage,
                actions: state.tasksEmptyStateRemediationActions
            ) { actionID in
                state.performEmptyStateRemediationAction(actionID)
            }
            .padding(UIStyle.panelPadding)
            .frame(maxWidth: .infinity, maxHeight: .infinity)
        } else if filteredTaskRunItems.isEmpty {
            VStack(alignment: .leading, spacing: 10) {
                Text("No tasks match current filters")
                    .font(.headline)
                Text("Adjust status, priority, or actor filters to see more results.")
                    .font(.callout)
                    .foregroundStyle(.secondary)
                HStack(spacing: 8) {
                    Button("Clear Filters") {
                        clearFilters()
                    }
                    .panelActionStyle(.primary)
                    .disabled(
                        searchText.isEmpty &&
                        selectedStateFilter == "All States" &&
                        selectedPriorityFilter == .all &&
                        selectedPrincipalFilter == "All Principals"
                    )

                    if state.isTasksLoading {
                        ProgressView()
                            .controlSize(.small)
                    }
                }
            }
            .padding(UIStyle.panelPadding)
            .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
        } else {
            ScrollView {
                LazyVStack(alignment: .leading, spacing: UIStyle.standardSpacing) {
                    if let detailStatus = state.taskRunDetailStatusMessage {
                        Text(detailStatus)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                            .padding(.horizontal, 4)
                    }
                    ForEach(filteredTaskRunItems) { item in
                        taskCard(item)
                    }
                }
                .padding(UIStyle.panelPadding)
            }
        }
    }

    private func routeSummaryLine(route: WorkflowRouteContext) -> String? {
        guard route.available else {
            return nil
        }
        if !isAdvancedInformationDensityEnabled {
            var simpleParts: [String] = []
            if let taskClass = route.taskClass {
                simpleParts.append("\(taskClass.capitalized) workflow")
            }
            if let routeLabel = route.routeLabel {
                simpleParts.append("Using \(routeLabel)")
            } else if let provider = route.provider {
                simpleParts.append("Using \(provider)")
            } else if let modelKey = route.modelKey {
                simpleParts.append("Using \(modelKey)")
            }
            return simpleParts.isEmpty ? nil : simpleParts.joined(separator: " • ")
        }
        var parts: [String] = []
        if let taskClass = route.taskClass {
            parts.append("task class \(taskClass)")
        }
        if let provider = route.provider, let modelKey = route.modelKey {
            parts.append("\(provider) • \(modelKey)")
        } else if let provider = route.provider {
            parts.append(provider)
        } else if let modelKey = route.modelKey {
            parts.append(modelKey)
        }
        if isAdvancedInformationDensityEnabled, let source = route.routeSource {
            parts.append("source \(source)")
        }
        return parts.isEmpty ? nil : parts.joined(separator: " • ")
    }

    private func taskCardTimingSummary(_ item: TaskRunListRowItem) -> String {
        if let runUpdated = item.runUpdatedAtLabel {
            return "Run updated \(runUpdated)"
        }
        return "Task updated \(item.taskUpdatedAtLabel)"
    }

    private var taskSearchPlaceholder: String {
        isAdvancedInformationDensityEnabled
            ? "Search tasks, runs, actors, or route details"
            : "Search tasks, people, or model"
    }

    private func taskCard(_ item: TaskRunListRowItem) -> some View {
        let summary = state.taskRunCardSummary(for: item)
        return GroupBox {
            VStack(alignment: .leading, spacing: 10) {
                HStack(alignment: .top, spacing: 10) {
                    VStack(alignment: .leading, spacing: 3) {
                        Text(item.title)
                            .font(.headline)
                        Text(taskCardTimingSummary(item))
                            .font(.caption)
                            .foregroundStyle(.secondary)
                        if let routeSummary = routeSummaryLine(route: item.route) {
                            Text(routeSummary)
                                .font(.caption2)
                                .foregroundStyle(.secondary)
                        }
                    }
                    Spacer(minLength: 0)
                    TahoeStatusBadge(
                        text: item.priorityLabel,
                        symbolName: "flag.fill",
                        tint: priorityTint(item.priority)
                    )
                    TahoeStatusBadge(
                        text: item.effectiveState.label,
                        symbolName: item.effectiveState.symbolName,
                        tint: item.effectiveState.tint
                    )
                }

                WorkflowCardSummaryView(summary: summary)

                HStack(spacing: 8) {
                    if item.runID != nil {
                        Button {
                            state.showTaskRunDetail(for: item)
                        } label: {
                            Label("View Run Detail", systemImage: "doc.text.magnifyingglass")
                        }
                        .panelActionStyle(.primary)
                        .controlSize(.small)
                        .disabled(state.isTaskRunDetailLoading)
                    }

                    Button("Open Related Inspect") {
                        state.openInspectForTaskRun(item)
                    }
                    .panelActionStyle(.secondary)
                    .controlSize(.small)

                    Button("Open Related Approvals") {
                        state.openApprovalsForTaskRun(item)
                    }
                    .panelActionStyle(.secondary)
                    .controlSize(.small)
                }

                taskRunControlRow(
                    taskID: item.taskID,
                    runID: item.runID,
                    actions: item.actions,
                    controlSize: .small
                )

                if let status = state.taskRunControlStatus(runID: item.runID) {
                    Text(status)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }

                DisclosureGroup(isExpanded: taskDetailsExpandedBinding(for: item)) {
                    VStack(alignment: .leading, spacing: 6) {
                        identityDetailRow(label: "Requested By", actorID: item.requestedByActorID)
                        identityDetailRow(label: "Subject", actorID: item.subjectPrincipalActorID)
                        identityDetailRow(label: "Acting As", actorID: item.actingAsActorID)
                        if let taskClass = item.route.taskClass {
                            detailRow(label: "Task Class", value: taskClass)
                        }
                        if let routeLabel = item.route.routeLabel {
                            detailRow(label: "Route", value: routeLabel)
                        }
                        detailRow(label: "Task Created", value: item.taskCreatedAtLabel)
                        detailRow(label: "Task Updated", value: item.taskUpdatedAtLabel)
                        if let runCreated = item.runCreatedAtLabel {
                            detailRow(label: "Run Created", value: runCreated)
                        }
                        if let runUpdated = item.runUpdatedAtLabel {
                            detailRow(label: "Run Updated", value: runUpdated)
                        }
                        if let startedAt = item.startedAtLabel {
                            detailRow(label: "Started", value: startedAt)
                        }
                        if let finishedAt = item.finishedAtLabel {
                            detailRow(label: "Finished", value: finishedAt)
                        }
                        if let lastError = item.lastError {
                            Text(lastError)
                                .font(.caption)
                                .foregroundStyle(.red)
                                .frame(maxWidth: .infinity, alignment: .leading)
                        }
                        if isAdvancedInformationDensityEnabled {
                            detailRow(label: "Task ID", value: item.taskID)
                            if let runID = item.runID {
                                detailRow(label: "Run ID", value: runID)
                            }
                        }
                    }
                    .padding(.top, 2)
                } label: {
                    Label("Details", systemImage: "info.circle")
                        .font(.caption.weight(.semibold))
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(.vertical, 2)
        }
        .groupBoxStyle(.automatic)
    }

    private func matchesStateFilter(_ item: TaskRunListRowItem) -> Bool {
        if selectedStateFilter == "All States" {
            return true
        }
        return item.effectiveState.label.caseInsensitiveCompare(selectedStateFilter) == .orderedSame
    }

    private func matchesPrincipalFilter(_ item: TaskRunListRowItem) -> Bool {
        if selectedPrincipalFilter == "All Principals" {
            return true
        }
        return item.requestedByActorID.caseInsensitiveCompare(selectedPrincipalFilter) == .orderedSame ||
            item.subjectPrincipalActorID.caseInsensitiveCompare(selectedPrincipalFilter) == .orderedSame ||
            item.actingAsActorID.caseInsensitiveCompare(selectedPrincipalFilter) == .orderedSame
    }

    private func matchesSearch(_ item: TaskRunListRowItem) -> Bool {
        let query = searchText.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !query.isEmpty else {
            return true
        }
        let normalizedQuery = query.lowercased()
        return item.taskID.lowercased().contains(normalizedQuery) ||
            (item.runID?.lowercased().contains(normalizedQuery) ?? false) ||
            item.title.lowercased().contains(normalizedQuery) ||
            state.principalIdentityDisplayValue(for: item.requestedByActorID).displayText
            .lowercased()
            .contains(normalizedQuery) ||
            state.principalIdentityDisplayValue(for: item.subjectPrincipalActorID).displayText
            .lowercased()
            .contains(normalizedQuery) ||
            state.principalIdentityDisplayValue(for: item.actingAsActorID).displayText
            .lowercased()
            .contains(normalizedQuery) ||
            (item.route.taskClass?.lowercased().contains(normalizedQuery) ?? false) ||
            (item.route.provider?.lowercased().contains(normalizedQuery) ?? false) ||
            (item.route.modelKey?.lowercased().contains(normalizedQuery) ?? false)
    }

    private var showLoadingSkeleton: Bool {
        (state.isTasksLoading || !state.hasLoadedTaskRunList) && state.taskRunItems.isEmpty
    }

    private func clearFilters() {
        let reset = state.resetTasksFilterContext()
        applyFilterContext(reset)
    }

    private func normalizeFilterSelections() {
        if !stateFilterOptions.contains(where: { $0 == selectedStateFilter }) {
            selectedStateFilter = "All States"
        }
        if !principalFilterOptions.contains(where: { $0 == selectedPrincipalFilter }) {
            selectedPrincipalFilter = "All Principals"
        }
    }

    private func applyPersistedFilterContext() {
        applyFilterContext(state.tasksFilterContext())
    }

    private func applyFilterContext(_ context: TasksFilterContext) {
        searchText = context.searchText
        selectedStateFilter = context.stateFilter
        selectedPriorityFilter = TaskPriorityFilter(rawValue: context.priorityFilterRawValue) ?? .all
        selectedPrincipalFilter = context.principalFilter
    }

    private func persistFilterContext() {
        state.updateTasksFilterContext(
            TasksFilterContext(
                searchText: searchText,
                stateFilter: selectedStateFilter,
                priorityFilterRawValue: selectedPriorityFilter.rawValue,
                principalFilter: selectedPrincipalFilter
            )
        )
    }

    private func restartAutoRefreshLoop() {
        stopAutoRefreshLoop()
        guard isAutoRefreshEnabled else {
            return
        }
        autoRefreshTask = Task {
            while !Task.isCancelled {
                try? await Task.sleep(for: .seconds(selectedAutoRefreshInterval.seconds))
                guard !Task.isCancelled else {
                    return
                }
                await MainActor.run {
                    guard isAutoRefreshEnabled else {
                        return
                    }
                    if !state.isTasksLoading {
                        state.refreshTaskRunList()
                        lastAutoRefreshAt = Date.now
                    }
                }
            }
        }
    }

    private func stopAutoRefreshLoop() {
        autoRefreshTask?.cancel()
        autoRefreshTask = nil
    }

    private func taskRunControlRow(
        taskID: String,
        runID: String?,
        actions: TaskRunActionAvailabilityItem,
        controlSize: ControlSize
    ) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(spacing: 8) {
                ForEach(TaskRunControlAction.allCases, id: \.rawValue) { action in
                    Button {
                        state.requestTaskRunControl(
                            action,
                            taskID: taskID,
                            runID: runID,
                            actions: actions
                        )
                    } label: {
                        Label(action.title, systemImage: action.symbolName)
                    }
                    .panelActionStyle(action.isDestructive ? .destructive : .secondary)
                    .successSymbolEffect(
                        state.successNotificationPulse(for: "tasks"),
                        reduceMotion: reduceMotion
                    )
                    .controlSize(controlSize)
                    .disabled(
                        !state.canPerformTaskRunControl(
                            action,
                            runID: runID,
                            actions: actions
                        )
                    )
                }

                if state.isTaskRunControlInFlight(runID: runID) {
                    ProgressView()
                        .controlSize(.small)
                }
            }

            if let disabledReason = taskRunControlDisabledReason(runID: runID, actions: actions) {
                Text(disabledReason)
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }
        }
    }

    private func taskRunControlDisabledReason(
        runID: String?,
        actions: TaskRunActionAvailabilityItem
    ) -> String? {
        let anyEnabled = TaskRunControlAction.allCases.contains { action in
            state.canPerformTaskRunControl(action, runID: runID, actions: actions)
        }
        if anyEnabled {
            return nil
        }
        for action in TaskRunControlAction.allCases {
            if let reason = state.taskRunControlDisabledReason(action, runID: runID, actions: actions) {
                return reason
            }
        }
        return nil
    }

    private func taskDetailsExpandedBinding(for item: TaskRunListRowItem) -> Binding<Bool> {
        Binding(
            get: { detailsExpandedByID[item.id] ?? false },
            set: { detailsExpandedByID[item.id] = $0 }
        )
    }

    private func detailRow(label: String, value: String) -> some View {
        HStack(alignment: .top, spacing: 10) {
            Text(label)
                .font(.caption.weight(.semibold))
                .foregroundStyle(.secondary)
                .frame(width: 92, alignment: .leading)
            Text(value)
                .font(.caption)
                .frame(maxWidth: .infinity, alignment: .leading)
                .textSelection(.enabled)
        }
    }

    private func identityDetailRow(
        label: String,
        actorID: String?,
        labelWidth: CGFloat = 92,
        valueFont: Font = .caption
    ) -> some View {
        let identity = state.principalIdentityDisplayValue(for: actorID)
        return IdentityDetailRowView(
            label: label,
            displayText: identity.displayText,
            rawID: identity.rawID,
            labelWidth: labelWidth,
            valueFont: valueFont
        )
    }

    private func principalFilterDisplayName(_ rawValue: String) -> String {
        if rawValue == "All Principals" {
            return rawValue
        }
        return state.principalIdentityDisplayValue(for: rawValue).displayText
    }

    private func priorityTint(_ priority: Int) -> Color {
        switch priority {
        case ..<2:
            return .secondary
        case 2:
            return .orange
        default:
            return .red
        }
    }

    private var taskRunDetailBinding: Binding<TaskRunDetailItem?> {
        Binding(
            get: { state.selectedTaskRunDetail },
            set: { detail in
                if detail == nil {
                    state.clearTaskRunDetail()
                }
            }
        )
    }

    private func taskRunDetailSheet(_ detail: TaskRunDetailItem) -> some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: UIStyle.sectionSpacing) {
                    GroupBox("Summary") {
                        VStack(alignment: .leading, spacing: 8) {
                            detailRow(label: "Title", value: detail.title)
                            if isAdvancedInformationDensityEnabled {
                                detailRow(label: "Task ID", value: detail.taskID)
                                detailRow(label: "Run ID", value: detail.runID)
                            }
                            detailRow(label: "Task State", value: detail.taskState)
                            detailRow(label: "Run State", value: detail.runState)
                            if let taskClass = detail.route.taskClass {
                                detailRow(label: "Task Class", value: taskClass)
                            }
                            if let routeLabel = detail.route.routeLabel {
                                detailRow(label: "Route", value: routeLabel)
                            }
                            if isAdvancedInformationDensityEnabled, let routeSource = detail.route.routeSource {
                                detailRow(label: "Route Source", value: routeSource)
                            }
                            detailRow(label: "Priority", value: detail.priorityLabel)
                            identityDetailRow(label: "Requested By", actorID: detail.requestedByActorID)
                            identityDetailRow(label: "Subject", actorID: detail.subjectPrincipalActorID)
                            identityDetailRow(label: "Acting As", actorID: detail.actingAsActorID)
                            detailRow(label: "Updated", value: detail.updatedAtLabel)
                            if let started = detail.startedAtLabel {
                                detailRow(label: "Started", value: started)
                            }
                            if let finished = detail.finishedAtLabel {
                                detailRow(label: "Finished", value: finished)
                            }

                            HStack(spacing: 8) {
                                Button("Open Related Inspect") {
                                    state.openInspectForTaskRunDetail(detail)
                                }
                                .panelActionStyle(.secondary)
                                .controlSize(.small)

                                Button("Open Related Approvals") {
                                    state.openApprovalsForTaskRunDetail(detail)
                                }
                                .panelActionStyle(.secondary)
                                .controlSize(.small)
                            }

                            taskRunControlRow(
                                taskID: detail.taskID,
                                runID: detail.runID,
                                actions: detail.actions,
                                controlSize: .small
                            )

                            if let status = state.taskRunControlStatus(runID: detail.runID) {
                                Text(status)
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }
                        }
                    }

                    if let lastError = detail.lastError {
                        GroupBox("Last Error") {
                            Text(lastError)
                                .font(.caption)
                                .foregroundStyle(.red)
                                .frame(maxWidth: .infinity, alignment: .leading)
                        }
                    }

                    GroupBox("Steps (\(detail.steps.count))") {
                        VStack(alignment: .leading, spacing: 8) {
                            if detail.steps.isEmpty {
                                Text("No steps found for this run.")
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            } else {
                                ForEach(detail.steps) { step in
                                    VStack(alignment: .leading, spacing: 4) {
                                        Text("\(step.index). \(step.name)")
                                            .font(.caption.weight(.semibold))
                                        HStack(spacing: 8) {
                                            Text(step.statusLabel)
                                                .font(.caption2)
                                            Text("Retry \(step.retryLabel)")
                                                .font(.caption2)
                                                .foregroundStyle(.secondary)
                                            Text("Timeout \(step.timeoutLabel)")
                                                .font(.caption2)
                                                .foregroundStyle(.secondary)
                                        }
                                        if isAdvancedInformationDensityEnabled,
                                           let capability = step.capability {
                                            Text("Capability: \(capability)")
                                                .font(.caption2)
                                                .foregroundStyle(.secondary)
                                        }
                                        if isAdvancedInformationDensityEnabled,
                                           let interaction = step.interactionLevel {
                                            Text("Interaction: \(interaction)")
                                                .font(.caption2)
                                                .foregroundStyle(.secondary)
                                        }
                                        if let lastError = step.lastError {
                                            Text(lastError)
                                                .font(.caption2)
                                                .foregroundStyle(.red)
                                        }
                                        Text("Updated \(step.updatedAtLabel)")
                                            .font(.caption2)
                                            .foregroundStyle(.secondary)
                                    }
                                    .frame(maxWidth: .infinity, alignment: .leading)
                                    .padding(8)
                                    .cardSurface(.subtle)
                                }
                            }
                        }
                    }

                    GroupBox("Artifacts (\(detail.artifacts.count))") {
                        VStack(alignment: .leading, spacing: 8) {
                            if detail.artifacts.isEmpty {
                                Text("No artifacts recorded for this run.")
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            } else {
                                ForEach(detail.artifacts) { artifact in
                                    VStack(alignment: .leading, spacing: 3) {
                                        Text(artifact.type)
                                            .font(.caption.weight(.semibold))
                                        if isAdvancedInformationDensityEnabled,
                                           let stepID = artifact.stepID {
                                            Text("Step: \(stepID)")
                                                .font(.caption2)
                                                .foregroundStyle(.secondary)
                                        }
                                        if let uri = artifact.uri {
                                            Text(uri)
                                                .font(.caption2)
                                                .foregroundStyle(.secondary)
                                                .textSelection(.enabled)
                                        }
                                        if isAdvancedInformationDensityEnabled,
                                           let hash = artifact.contentHash {
                                            Text("Hash: \(hash)")
                                                .font(.caption2)
                                                .foregroundStyle(.secondary)
                                                .textSelection(.enabled)
                                        }
                                        Text("Created \(artifact.createdAtLabel)")
                                            .font(.caption2)
                                            .foregroundStyle(.secondary)
                                    }
                                    .frame(maxWidth: .infinity, alignment: .leading)
                                    .padding(8)
                                    .cardSurface(.subtle)
                                }
                            }
                        }
                    }

                    GroupBox("Audit Entries (\(detail.auditEntries.count))") {
                        VStack(alignment: .leading, spacing: 8) {
                            if detail.auditEntries.isEmpty {
                                Text("No audit entries found for this run.")
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            } else {
                                ForEach(detail.auditEntries) { audit in
                                    VStack(alignment: .leading, spacing: 3) {
                                        Text(audit.eventType)
                                            .font(.caption.weight(.semibold))
                                        if isAdvancedInformationDensityEnabled,
                                           let actor = audit.actorID {
                                            identityDetailRow(
                                                label: "Actor",
                                                actorID: actor,
                                                labelWidth: 72,
                                                valueFont: .caption2
                                            )
                                        }
                                        if isAdvancedInformationDensityEnabled,
                                           let actingAs = audit.actingAsActorID {
                                            identityDetailRow(
                                                label: "Acting As",
                                                actorID: actingAs,
                                                labelWidth: 72,
                                                valueFont: .caption2
                                            )
                                        }
                                        if isAdvancedInformationDensityEnabled,
                                           let correlation = audit.correlationID {
                                            Text("Correlation: \(correlation)")
                                                .font(.caption2)
                                                .foregroundStyle(.secondary)
                                                .textSelection(.enabled)
                                        }
                                        if isAdvancedInformationDensityEnabled,
                                           let payload = audit.payloadSummary {
                                            Text(payload)
                                                .font(.caption2)
                                                .foregroundStyle(.secondary)
                                                .textSelection(.enabled)
                                        }
                                        Text("Created \(audit.createdAtLabel)")
                                            .font(.caption2)
                                            .foregroundStyle(.secondary)
                                    }
                                    .frame(maxWidth: .infinity, alignment: .leading)
                                    .padding(8)
                                    .cardSurface(.subtle)
                                }
                            }
                        }
                    }
                }
                .padding(UIStyle.panelPadding)
            }
            .navigationTitle("Run Detail")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") {
                        state.clearTaskRunDetail()
                    }
                }
            }
        }
    }

    private func latestTaskSubmissionCard(_ receipt: TaskSubmissionReceiptItem) -> some View {
        GroupBox {
            VStack(alignment: .leading, spacing: 8) {
                HStack(alignment: .firstTextBaseline, spacing: 8) {
                    Text("Latest Submitted Task")
                        .font(.headline)
                    Spacer(minLength: 0)
                    TahoeStatusBadge(
                        text: receipt.state.replacingOccurrences(of: "_", with: " ").capitalized,
                        symbolName: "paperplane.fill",
                        tint: .green
                    )
                }

                if isAdvancedInformationDensityEnabled {
                    detailRow(label: "Task ID", value: receipt.taskID)
                    detailRow(label: "Run ID", value: receipt.runID)
                    if let correlationID = receipt.correlationID {
                        detailRow(label: "Correlation", value: correlationID)
                    }
                }
                detailRow(
                    label: "Submitted",
                    value: receipt.submittedAt.formatted(date: .abbreviated, time: .standard)
                )

                if let status = state.taskSubmitStatusMessage {
                    Text(status)
                        .font(.caption)
                        .foregroundStyle(taskSubmitStatusColor)
                }

                HStack(spacing: 8) {
                    Button("Filter to Run") {
                        searchText = receipt.runID
                        persistFilterContext()
                    }
                    .panelActionStyle(.secondary)
                    .controlSize(.small)

                    Button("Refresh Tasks") {
                        state.refreshTaskRunList()
                    }
                    .panelActionStyle(.secondary)
                    .controlSize(.small)
                }
            }
            .padding(.vertical, 2)
        }
        .groupBoxStyle(.automatic)
    }

    private var taskSubmitSheet: some View {
        NavigationStack {
            Form {
                Section("Goal") {
                    TextField("Goal", text: $taskSubmitTitleDraft)
                        .accessibilityIdentifier("tasks-submit-goal-field")
                    Text("Describe the outcome you want this task to achieve.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }

                Section("Details") {
                    TextField("Details (optional)", text: $taskSubmitDescriptionDraft, axis: .vertical)
                        .lineLimit(3...6)
                }

                Section("Priority") {
                    Picker("Priority", selection: $taskSubmitPriorityDraft) {
                        ForEach(TaskSubmitPriority.allCases) { priority in
                            Text(priority.label).tag(priority)
                        }
                    }
                    .pickerStyle(.segmented)
                }

                Section("Execution Context") {
                    Text(taskSubmitContextSummary)
                        .font(.callout)

                    DisclosureGroup("Override Context", isExpanded: $taskSubmitContextOverrideExpanded) {
                        Picker("Task Class", selection: $taskSubmitTaskClassDraft) {
                            ForEach(taskSubmitTaskClassOptions, id: \.self) { taskClass in
                                Text(taskClass).tag(taskClass)
                            }
                        }
                        .pickerStyle(.menu)

                        Picker("Requested By", selection: $taskSubmitRequestedByActorDraft) {
                            ForEach(taskSubmitPrincipalOptions, id: \.self) { principalID in
                                Text(state.principalOptionDisplayName(for: principalID)).tag(principalID)
                            }
                        }
                        .pickerStyle(.menu)

                        Picker("Subject Principal", selection: $taskSubmitSubjectPrincipalDraft) {
                            ForEach(taskSubmitPrincipalOptions, id: \.self) { principalID in
                                Text(state.principalOptionDisplayName(for: principalID)).tag(principalID)
                            }
                        }
                        .pickerStyle(.menu)

                        Text("Requested By identifies who initiated the task. Subject principal scopes execution context.")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                }

                Section("Status") {
                    Text(state.taskSubmitStatusMessage ?? "No task submit action run yet.")
                        .font(.callout)
                        .foregroundStyle(taskSubmitStatusColor)
                }
            }
            .navigationTitle("New Task")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") {
                        taskSubmitContextOverrideExpanded = false
                        isPresentingTaskSubmitSheet = false
                    }
                    .disabled(state.isTaskSubmitInFlight)
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Submit Task") {
                        submitTaskDraft()
                    }
                    .panelActionStyle(.primary)
                    .successSymbolEffect(
                        state.successNotificationPulse(for: "tasks"),
                        reduceMotion: reduceMotion
                    )
                    .disabled(!isTaskSubmitDraftValid || state.isTaskSubmitInFlight)
                    .accessibilityIdentifier("tasks-submit-button")
                }
            }
        }
    }

    private var taskSubmitTaskClassOptions: [String] {
        var options = state.modelRouteSimulationTaskClassOptions
            .map { $0.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() }
            .filter { !$0.isEmpty }
        if !options.contains(where: { $0.caseInsensitiveCompare(taskSubmitTaskClassDraft) == .orderedSame }) {
            options.insert(taskSubmitTaskClassDraft, at: 0)
        }
        return options
    }

    private var taskSubmitPrincipalOptions: [String] {
        var options = state.taskSubmissionPrincipalOptions
            .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
            .filter { !$0.isEmpty }
        if options.isEmpty {
            options = ["default"]
        }
        if !options.contains(where: { $0.caseInsensitiveCompare(taskSubmitRequestedByActorDraft) == .orderedSame }) {
            options.insert(taskSubmitRequestedByActorDraft, at: 0)
        }
        if !options.contains(where: { $0.caseInsensitiveCompare(taskSubmitSubjectPrincipalDraft) == .orderedSame }) {
            options.insert(taskSubmitSubjectPrincipalDraft, at: 0)
        }
        return Array(NSOrderedSet(array: options)) as? [String] ?? options
    }

    private var taskSubmitContextSummary: String {
        let requestedBy = state.principalOptionDisplayName(for: taskSubmitRequestedByActorDraft)
        let subjectPrincipal = state.principalOptionDisplayName(for: taskSubmitSubjectPrincipalDraft)
        let taskClass = taskSubmitTaskClassDraft.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        let classLabel = taskClass.isEmpty ? "chat" : taskClass
        return "Task class \(classLabel) • requested by \(requestedBy) • subject \(subjectPrincipal)"
    }

    private var taskSubmitStatusColor: Color {
        let normalized = state.taskSubmitStatusMessage?
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .lowercased() ?? ""
        if normalized.contains("failed")
            || normalized.contains("required")
            || normalized.contains("not in the active workspace") {
            return .red
        }
        if normalized.contains("submitted") || normalized.contains("accepted") {
            return .green
        }
        return .secondary
    }

    private var isTaskSubmitDraftValid: Bool {
        !taskSubmitTitleDraft.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty &&
            !taskSubmitRequestedByActorDraft.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty &&
            !taskSubmitSubjectPrincipalDraft.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }

    private var taskSubmitDraftPersistenceSignature: String {
        [
            isPresentingTaskSubmitSheet ? "1" : "0",
            taskSubmitTitleDraft,
            taskSubmitDescriptionDraft,
            taskSubmitPriorityDraft.rawValue,
            taskSubmitTaskClassDraft,
            taskSubmitRequestedByActorDraft,
            taskSubmitSubjectPrincipalDraft,
        ].joined(separator: "\u{1F}")
    }

    private func submitTaskDraft() {
        let payloadDescription = taskSubmitPayloadDescription()
        let taskClass = taskSubmitTaskClassDraft
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .lowercased()
        state.submitTask(
            title: taskSubmitTitleDraft,
            description: payloadDescription,
            taskClass: taskClass.isEmpty ? nil : taskClass,
            requestedByActorID: taskSubmitRequestedByActorDraft,
            subjectPrincipalActorID: taskSubmitSubjectPrincipalDraft
        )
    }

    private func taskSubmitPayloadDescription() -> String? {
        let trimmedDetails = taskSubmitDescriptionDraft.trimmingCharacters(in: .whitespacesAndNewlines)
        let trimmedDetailValue = trimmedDetails.isEmpty ? nil : trimmedDetails
        let priorityAnnotation = taskSubmitPriorityDraft.annotationLine
        switch (priorityAnnotation, trimmedDetailValue) {
        case let (annotation?, details?):
            return "\(annotation)\n\(details)"
        case let (annotation?, nil):
            return annotation
        case let (nil, details?):
            return details
        case (nil, nil):
            return nil
        }
    }

    private func prepareTaskSubmitDraft() {
        taskSubmitTitleDraft = ""
        taskSubmitDescriptionDraft = ""
        taskSubmitPriorityDraft = .medium
        taskSubmitTaskClassDraft = "chat"
        taskSubmitContextOverrideExpanded = false

        let preferredPrincipal = state.activePrincipalLabel
            .trimmingCharacters(in: .whitespacesAndNewlines)
        let principalOptions = state.taskSubmissionPrincipalOptions
            .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
            .filter { !$0.isEmpty }
        if let matchingPrincipal = principalOptions.first(
            where: { $0.caseInsensitiveCompare(preferredPrincipal) == .orderedSame }
        ) {
            taskSubmitRequestedByActorDraft = matchingPrincipal
            taskSubmitSubjectPrincipalDraft = matchingPrincipal
        } else if let firstOption = principalOptions.first {
            taskSubmitRequestedByActorDraft = firstOption
            taskSubmitSubjectPrincipalDraft = firstOption
        } else {
            taskSubmitRequestedByActorDraft = "default"
            taskSubmitSubjectPrincipalDraft = "default"
        }
    }

    private func normalizeTaskSubmitPrincipalDrafts() {
        let options = taskSubmitPrincipalOptions
        guard !options.isEmpty else {
            return
        }
        if !options.contains(where: { $0.caseInsensitiveCompare(taskSubmitRequestedByActorDraft) == .orderedSame }) {
            taskSubmitRequestedByActorDraft = options[0]
        }
        if !options.contains(where: { $0.caseInsensitiveCompare(taskSubmitSubjectPrincipalDraft) == .orderedSame }) {
            taskSubmitSubjectPrincipalDraft = options[0]
        }
    }

    private func closeTaskSubmitSheetIfSuccessful() {
        guard isPresentingTaskSubmitSheet, !state.isTaskSubmitInFlight else {
            return
        }
        let message = state.taskSubmitStatusMessage?
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .lowercased() ?? ""
        if message.contains("submitted") {
            isPresentingTaskSubmitSheet = false
            taskSubmitContextOverrideExpanded = false
        }
    }

    private func applyPersistedTaskSubmitDraftContext() {
        guard let context = state.tasksSubmitDraftContext() else {
            prepareTaskSubmitDraft()
            isPresentingTaskSubmitSheet = false
            return
        }
        taskSubmitTitleDraft = context.title
        taskSubmitDescriptionDraft = context.description
        taskSubmitPriorityDraft = TaskSubmitPriority(rawValue: context.priorityRawValue) ?? .medium
        taskSubmitTaskClassDraft = context.taskClass
        taskSubmitRequestedByActorDraft = context.requestedByActorID
        taskSubmitSubjectPrincipalDraft = context.subjectPrincipalActorID
        normalizeTaskSubmitPrincipalDrafts()
        isPresentingTaskSubmitSheet = context.isPresented
        taskSubmitContextOverrideExpanded = false
    }

    private func persistTaskSubmitDraftContext() {
        let hasDraftContent = !taskSubmitTitleDraft.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ||
            !taskSubmitDescriptionDraft.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
        guard hasDraftContent || isPresentingTaskSubmitSheet else {
            state.updateTasksSubmitDraftContext(nil)
            return
        }
        state.updateTasksSubmitDraftContext(
            TasksSubmitDraftContext(
                isPresented: isPresentingTaskSubmitSheet,
                title: taskSubmitTitleDraft,
                description: taskSubmitDescriptionDraft,
                taskClass: taskSubmitTaskClassDraft,
                priorityRawValue: taskSubmitPriorityDraft.rawValue,
                requestedByActorID: taskSubmitRequestedByActorDraft,
                subjectPrincipalActorID: taskSubmitSubjectPrincipalDraft
            )
        )
    }

    private func applyExternalSearchSeedIfNeeded() {
        guard let seed = state.tasksSearchSeed?
            .trimmingCharacters(in: .whitespacesAndNewlines),
              !seed.isEmpty else {
            return
        }
        searchText = seed
        state.tasksSearchSeed = nil
    }

    private func applyExternalTaskSubmitDraftIfNeeded() {
        guard let draftSeed = state.taskSubmitDraftSeed else {
            return
        }

        taskSubmitTitleDraft = draftSeed.title
        taskSubmitDescriptionDraft = draftSeed.description ?? ""
        taskSubmitPriorityDraft = .medium
        taskSubmitTaskClassDraft = draftSeed.taskClass

        let principalOptions = taskSubmitPrincipalOptions
        if let requestedBy = draftSeed.requestedByActorID,
           principalOptions.contains(where: { $0.caseInsensitiveCompare(requestedBy) == .orderedSame }) {
            taskSubmitRequestedByActorDraft = requestedBy
        }
        if let subjectPrincipal = draftSeed.subjectPrincipalActorID,
           principalOptions.contains(where: { $0.caseInsensitiveCompare(subjectPrincipal) == .orderedSame }) {
            taskSubmitSubjectPrincipalDraft = subjectPrincipal
        }

        normalizeTaskSubmitPrincipalDrafts()
        isPresentingTaskSubmitSheet = true
        taskSubmitContextOverrideExpanded = false
        state.clearTaskSubmitDraftSeed()
    }

    private func resetWorkspaceContinuityContext() {
        state.resetWorkspaceContinuityContext()
        prepareTaskSubmitDraft()
        isPresentingTaskSubmitSheet = false
        taskSubmitContextOverrideExpanded = false
        clearFilters()
        state.updateTasksSubmitDraftContext(nil)
    }
}
