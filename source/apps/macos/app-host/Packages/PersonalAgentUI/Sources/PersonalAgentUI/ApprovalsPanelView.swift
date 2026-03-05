import SwiftUI

struct ApprovalsPanelView: View {
    @ObservedObject private var state: AppShellState
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @FocusState private var isSearchFieldFocused: Bool
    @State private var searchText = ""
    @State private var decisionPhraseDrafts: [String: String] = [:]
    @State private var decisionActorDrafts: [String: String] = [:]
    @State private var decisionIntentDrafts: [String: ApprovalDecisionIntent] = [:]
    @State private var decisionRationaleDrafts: [String: String] = [:]
    @State private var evidenceExpandedByID: [String: Bool] = [:]
    @State private var detailsExpandedByID: [String: Bool] = [:]

    private enum ApprovalDecisionIntent: String, CaseIterable, Identifiable {
        case approve
        case reject

        var id: String { rawValue }

        var label: String {
            switch self {
            case .approve:
                return "Approve"
            case .reject:
                return "Reject"
            }
        }

        var submitLabel: String {
            switch self {
            case .approve:
                return "Approve and Continue"
            case .reject:
                return "Submit Rejection"
            }
        }

        var submitSymbolName: String {
            switch self {
            case .approve:
                return "checkmark.circle.fill"
            case .reject:
                return "xmark.circle"
            }
        }

    }

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
            supplementary: { supplementaryContent },
            content: { content }
        )
        .onAppear {
            applyPersistedFilterContext()
            applyExternalSearchSeedIfNeeded()
            focusSearchFieldForKeyboardTraversal()
        }
        .onChange(of: state.workspaceLabel) { _, _ in
            applyPersistedFilterContext()
        }
        .onChange(of: state.approvalsSearchSeed) { _, _ in
            applyExternalSearchSeedIfNeeded()
        }
        .onChange(of: searchText) { _, _ in
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

    private var panelProblemRemediation: PanelProblemRemediationContext? {
        state.panelProblemRemediation(for: .approvals)
    }

    @ViewBuilder
    private var supplementaryContent: some View {
        if let panelProblemRemediation {
            PanelProblemRemediationCardView(context: panelProblemRemediation) { actionID in
                state.performPanelProblemRemediationAction(actionID, section: .approvals)
            }
            .padding(.horizontal, UIStyle.panelPadding)
            .padding(.bottom, 12)
        }
    }

    private var isAdvancedInformationDensityEnabled: Bool {
        state.isAdvancedInformationDensityEnabled
    }

    private var header: some View {
        TahoeSectionHeader(
            title: "Approvals",
            subtitle: state.approvalsStatusMessage ?? "Pending and completed decisions"
        ) {
            HStack(spacing: 8) {
                if state.isApprovalsLoading {
                    ProgressView()
                        .controlSize(.small)
                }

                Button {
                    state.refreshApprovalsInbox()
                } label: {
                    Label("Refresh", systemImage: "arrow.clockwise")
                }
                .panelActionStyle(.secondary)
                .disabled(state.isApprovalsLoading)
                .accessibilityLabel("Refresh approvals inbox")
            }
        }
    }

    private var filterToolbar: some View {
        PanelFilterBarCard(summaryText: filteredSummaryLabel) {
            HStack(spacing: 8) {
                HStack(spacing: 6) {
                    Image(systemName: "magnifyingglass")
                        .foregroundStyle(.secondary)
                        .accessibilityHidden(true)
                    TextField(approvalsSearchPlaceholder, text: $searchText)
                        .textFieldStyle(.plain)
                        .focused($isSearchFieldFocused)
                        .accessibilityLabel(UIAccessibilityContract.approvalsSearchLabel)
                        .accessibilityHint(UIAccessibilityContract.approvalsSearchHint)
                        .accessibilityIdentifier(UIAccessibilityContract.approvalsSearchIdentifier)
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
            }
        }
    }

    @ViewBuilder
    private var content: some View {
        if showLoadingSkeleton {
            PanelLoadingSkeletonView(
                title: "Loading Approvals",
                subtitle: "Fetching pending and completed approval decisions.",
                rowCount: 4
            )
            .frame(maxWidth: .infinity, maxHeight: .infinity)
        } else if state.approvalInboxItems.isEmpty {
            PanelRemediationEmptyStateView(
                title: "No Approvals Yet",
                systemImage: "checkmark.shield",
                description: "Approvals appear here when a workflow needs a decision.",
                statusMessage: state.approvalsStatusMessage,
                headerStatusMessage: state.approvalsStatusMessage,
                actions: state.approvalsEmptyStateRemediationActions
            ) { actionID in
                state.performEmptyStateRemediationAction(actionID)
            }
            .padding(UIStyle.panelPadding)
            .frame(maxWidth: .infinity, maxHeight: .infinity)
        } else if filteredApprovalItems.isEmpty {
            VStack(alignment: .leading, spacing: 10) {
                Text("No approvals match current filters")
                    .font(.headline)
                Text("Adjust filters or clear search text to include all approvals.")
                    .font(.callout)
                    .foregroundStyle(.secondary)
                Button("Clear Filters") {
                    clearFilters()
                }
                .panelActionStyle(.primary)
            }
            .padding(UIStyle.panelPadding)
            .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
        } else {
            ScrollView {
                LazyVStack(alignment: .leading, spacing: UIStyle.standardSpacing) {
                    if !pendingApprovals.isEmpty {
                        sectionTitle("Pending (\(pendingApprovals.count))")
                        ForEach(pendingApprovals) { item in
                            approvalCard(item)
                        }
                    }

                    if !finalizedApprovals.isEmpty {
                        sectionTitle("Recent Decisions (\(finalizedApprovals.count))")
                            .padding(.top, pendingApprovals.isEmpty ? 0 : 4)
                        ForEach(finalizedApprovals) { item in
                            approvalCard(item)
                        }
                    }
                }
                .padding(UIStyle.panelPadding)
            }
        }
    }

    private var pendingApprovals: [ApprovalInboxItem] {
        filteredApprovalItems.filter { $0.decisionState == .pending }
    }

    private var finalizedApprovals: [ApprovalInboxItem] {
        filteredApprovalItems.filter { $0.decisionState == .final }
    }

    private var filteredApprovalItems: [ApprovalInboxItem] {
        let query = searchText.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !query.isEmpty else {
            return state.approvalInboxItems
        }
        let normalizedQuery = query.lowercased()
        return state.approvalInboxItems.filter { item in
            approvalSearchFields(for: item).contains { field in
                field.lowercased().contains(normalizedQuery)
            }
        }
    }

    private var filteredSummaryLabel: String {
        if state.approvalInboxItems.isEmpty {
            return "No approvals loaded yet."
        }
        return "Showing \(filteredApprovalItems.count) of \(state.approvalInboxItems.count) approvals."
    }

    private var activeFilterSummaryParts: [String] {
        ApprovalsFilterContext(searchText: searchText).activeFilterSummaryParts
    }

    private var showLoadingSkeleton: Bool {
        (state.isApprovalsLoading || !state.hasLoadedApprovalsInbox) && state.approvalInboxItems.isEmpty
    }

    private func approvalSearchFields(for item: ApprovalInboxItem) -> [String] {
        [
            item.id,
            item.taskTitle,
            item.stepName,
            item.riskRationale,
            item.taskState,
            item.runState,
            item.requestedByActorID,
            item.subjectPrincipalActorID,
            item.actingAsActorID,
            state.principalIdentityDisplayValue(for: item.requestedByActorID).displayText,
            state.principalIdentityDisplayValue(for: item.subjectPrincipalActorID).displayText,
            state.principalIdentityDisplayValue(for: item.actingAsActorID).displayText,
            item.route.taskClass,
            item.route.provider,
            item.route.modelKey,
            item.route.routeSource,
            item.taskID,
            item.runID,
            item.stepID
        ].compactMap { value in
            guard let value else {
                return nil
            }
            let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines)
            return trimmed.isEmpty ? nil : trimmed
        }
    }

    private func applyExternalSearchSeedIfNeeded() {
        guard let seed = state.approvalsSearchSeed?.trimmingCharacters(in: .whitespacesAndNewlines), !seed.isEmpty else {
            return
        }
        searchText = seed
        state.approvalsSearchSeed = nil
    }

    private func focusSearchFieldForKeyboardTraversal() {
        DispatchQueue.main.async {
            isSearchFieldFocused = true
        }
    }

    private func clearFilters() {
        let reset = state.resetApprovalsFilterContext()
        applyFilterContext(reset)
    }

    private func applyPersistedFilterContext() {
        applyFilterContext(state.approvalsFilterContext())
    }

    private func applyFilterContext(_ context: ApprovalsFilterContext) {
        searchText = context.searchText
    }

    private func persistFilterContext() {
        state.updateApprovalsFilterContext(
            ApprovalsFilterContext(searchText: searchText)
        )
    }

    private func routeSummaryLine(for item: ApprovalInboxItem) -> String? {
        guard item.route.available else {
            return nil
        }
        if !isAdvancedInformationDensityEnabled {
            var simpleParts: [String] = []
            if let taskClass = item.route.taskClass {
                simpleParts.append("\(taskClass.capitalized) workflow")
            }
            if let routeLabel = item.route.routeLabel {
                simpleParts.append("Using \(routeLabel)")
            }
            return simpleParts.isEmpty ? nil : simpleParts.joined(separator: " • ")
        }
        var parts: [String] = []
        if let taskClass = item.route.taskClass {
            parts.append("task class \(taskClass)")
        }
        if let routeLabel = item.route.routeLabel {
            parts.append(routeLabel)
        }
        if isAdvancedInformationDensityEnabled, let routeSource = item.route.routeSource {
            parts.append("source \(routeSource)")
        }
        return parts.isEmpty ? nil : parts.joined(separator: " • ")
    }

    private var approvalsSearchPlaceholder: String {
        isAdvancedInformationDensityEnabled
            ? "Search approvals, tasks, runs, and steps"
            : "Search approvals and workflow details"
    }

    private func sectionTitle(_ text: String) -> some View {
        Text(text)
            .font(.subheadline.weight(.semibold))
            .foregroundStyle(.secondary)
    }

    private func approvalCard(_ item: ApprovalInboxItem) -> some View {
        let summary = state.approvalCardSummary(for: item)
        return GroupBox {
            VStack(alignment: .leading, spacing: 10) {
                HStack(alignment: .top, spacing: 10) {
                    VStack(alignment: .leading, spacing: 3) {
                        Text(item.taskTitle)
                            .font(.headline)
                        Text(item.stepName)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                        if let routeSummary = routeSummaryLine(for: item) {
                            Text(routeSummary)
                                .font(.caption2)
                                .foregroundStyle(.secondary)
                        }
                    }
                    Spacer(minLength: 0)
                    TahoeStatusBadge(
                        text: item.riskLevel.label,
                        symbolName: item.riskLevel.symbolName,
                        tint: item.riskLevel.tint
                    )
                    TahoeStatusBadge(
                        text: item.decisionState.label,
                        symbolName: item.decisionState.symbolName,
                        tint: item.decisionState.tint
                    )
                    if let decisionOutcome = item.decisionOutcome {
                        TahoeStatusBadge(
                            text: decisionOutcome.label,
                            symbolName: decisionOutcome.symbolName,
                            tint: decisionOutcome.tint
                        )
                    }
                }

                WorkflowCardSummaryView(summary: summary)

                if item.taskID != nil
                    || item.runID != nil
                    || item.route.available
                    || item.route.taskClass != nil
                    || item.route.provider != nil
                    || item.route.modelKey != nil {
                    HStack(spacing: 8) {
                        if item.runID != nil {
                            Button("Open Task Detail") {
                                state.openTaskRunDetailForApproval(item)
                            }
                            .panelActionStyle(.secondary)
                            .controlSize(.small)
                        }

                        Button("Open Related Tasks") {
                            state.openTasksForApproval(item)
                        }
                        .panelActionStyle(.secondary)
                        .controlSize(.small)

                        Button("Open Related Inspect") {
                            state.openInspectForApproval(item)
                        }
                        .panelActionStyle(.secondary)
                        .controlSize(.small)
                    }
                }

                if item.decisionState == .pending {
                    Divider()

                    VStack(alignment: .leading, spacing: 8) {
                        Text("Decision")
                            .font(.caption.weight(.semibold))
                            .foregroundStyle(.secondary)
                        Text("Choose action, confirm who is deciding, then submit.")
                            .font(.caption2)
                            .foregroundStyle(.secondary)

                        Picker("Action", selection: decisionIntentBinding(for: item)) {
                            ForEach(ApprovalDecisionIntent.allCases) { intent in
                                Text(intent.label).tag(intent)
                            }
                        }
                        .pickerStyle(.segmented)

                        Picker("Decision By", selection: decisionActorBinding(for: item)) {
                            ForEach(decisionActorOptions(for: item), id: \.self) { actorID in
                                Text(state.principalOptionDisplayName(for: actorID)).tag(actorID)
                            }
                        }
                        .pickerStyle(.menu)

                        if let actorValidationMessage = decisionActorValidationMessage(for: item) {
                            Text(actorValidationMessage)
                                .font(.caption)
                                .foregroundStyle(.orange)
                        }

                        if decisionIntent(for: item) == .approve {
                            let requiredPhrase = state.approvalRequiredPhrase(for: item)
                            TextField(
                                "Approve phrase",
                                text: decisionPhraseBinding(for: item)
                            )
                            .textFieldStyle(.roundedBorder)

                            HStack(spacing: 8) {
                                Button("Use Required Phrase") {
                                    decisionPhraseDrafts[item.id] = requiredPhrase
                                }
                                .panelActionStyle(.secondary)
                                .controlSize(.small)
                                .disabled(
                                    decisionPhraseBinding(for: item)
                                        .wrappedValue
                                        .trimmingCharacters(in: .whitespacesAndNewlines) == requiredPhrase
                                )

                                Text("Required: `\(requiredPhrase)`")
                                    .font(.caption2)
                                    .foregroundStyle(.secondary)
                                    .textSelection(.enabled)
                            }

                            if let phraseValidationMessage = approvalPhraseValidationMessage(for: item) {
                                Text(phraseValidationMessage)
                                    .font(.caption)
                                    .foregroundStyle(.orange)
                            }
                        } else {
                            TextField(
                                "Reject phrase (optional, default REJECT)",
                                text: decisionPhraseBinding(for: item)
                            )
                            .textFieldStyle(.roundedBorder)
                            Text("Leave reject phrase empty to send `REJECT`.")
                                .font(.caption2)
                                .foregroundStyle(.secondary)
                        }

                        TextField(
                            "Decision note (optional)",
                            text: decisionRationaleBinding(for: item),
                            axis: .vertical
                        )
                        .lineLimit(2...4)
                        .textFieldStyle(.roundedBorder)

                        HStack(spacing: 8) {
                            Button {
                                submitDecision(item)
                            } label: {
                                Label(
                                    decisionIntent(for: item).submitLabel,
                                    systemImage: decisionIntent(for: item).submitSymbolName
                                )
                            }
                            .panelActionStyle(decisionIntent(for: item) == .approve ? .primary : .destructive)
                            .successSymbolEffect(
                                state.successNotificationPulse(for: "approvals"),
                                reduceMotion: reduceMotion
                            )
                            .disabled(isDecisionSubmitDisabled(for: item))

                            if state.approvalDecisionInFlightIDs.contains(item.id) {
                                ProgressView()
                                    .controlSize(.small)
                            }
                        }
                    }
                }

                approvalEvidenceSection(item)

                DisclosureGroup(isExpanded: detailsExpandedBinding(for: item)) {
                    VStack(alignment: .leading, spacing: 6) {
                        detailRow(label: "Step", value: item.stepName)
                        detailRow(label: "Risk", value: item.riskLevel.label)
                        identityDetailRow(label: "Requested By", actorID: item.requestedByActorID)
                        identityDetailRow(label: "Subject", actorID: item.subjectPrincipalActorID)
                        identityDetailRow(label: "Acting As", actorID: item.actingAsActorID)
                        if let taskClass = item.route.taskClass {
                            detailRow(label: "Task Class", value: taskClass)
                        }
                        if let routeLabel = item.route.routeLabel {
                            detailRow(label: "Route", value: routeLabel)
                        }
                        if isAdvancedInformationDensityEnabled, let routeSource = item.route.routeSource {
                            detailRow(label: "Route Source", value: routeSource)
                        }
                        detailRow(label: "Task State", value: item.taskState)
                        detailRow(label: "Run State", value: item.runState)
                        detailRow(label: "Requested At", value: item.requestedAtLabel)
                        if let decidedAt = item.decidedAtLabel {
                            detailRow(label: "Decided At", value: decidedAt)
                        }
                        if let decisionBy = item.decisionByActorID {
                            identityDetailRow(label: "Decision By", actorID: decisionBy)
                        }
                        detailRow(label: "Risk Rationale", value: item.riskRationale)
                        if let rationale = item.decisionRationale {
                            detailRow(label: "Decision Rationale", value: rationale)
                        }
                        if isAdvancedInformationDensityEnabled, let requestedPhrase = item.requestedPhrase {
                            detailRow(label: "Phrase", value: requestedPhrase)
                        }
                        if isAdvancedInformationDensityEnabled, let capability = item.stepCapabilityKey {
                            detailRow(label: "Capability", value: capability)
                        }
                        if isAdvancedInformationDensityEnabled {
                            if let taskID = item.taskID {
                                detailRow(label: "Task ID", value: taskID)
                            }
                            if let runID = item.runID {
                                detailRow(label: "Run ID", value: runID)
                            }
                            if let stepID = item.stepID {
                                detailRow(label: "Step ID", value: stepID)
                            }
                        }
                    }
                    .padding(.top, 2)
                } label: {
                    Label("Details", systemImage: "info.circle")
                        .font(.caption.weight(.semibold))
                }

                if let actionStatus = state.approvalsActionStatusByID[item.id] {
                    HStack(spacing: 8) {
                        Text(actionStatus)
                            .font(.caption)
                            .foregroundStyle(approvalActionStatusTint(actionStatus))
                        if approvalActionStatusIsSuccess(actionStatus) {
                            TahoeStatusBadge(
                                text: "Saved",
                                symbolName: "checkmark.circle.fill",
                                tint: .green
                            )
                            .controlSize(.small)
                        }
                    }
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(.vertical, 2)
        }
        .groupBoxStyle(.automatic)
    }

    @ViewBuilder
    private func approvalEvidenceSection(_ item: ApprovalInboxItem) -> some View {
        DisclosureGroup(isExpanded: approvalEvidenceExpandedBinding(for: item)) {
            VStack(alignment: .leading, spacing: 8) {
                if state.approvalEvidenceInFlightIDs.contains(item.id),
                   state.approvalEvidenceByID[item.id] == nil {
                    HStack(spacing: 8) {
                        ProgressView()
                            .controlSize(.small)
                        Text("Loading step/artifact evidence…")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                }

                if let evidence = state.approvalEvidenceByID[item.id] {
                    GroupBox("Step Context") {
                        VStack(alignment: .leading, spacing: 6) {
                            detailRow(label: "Title", value: evidence.title)
                            if isAdvancedInformationDensityEnabled {
                                detailRow(label: "Task ID", value: evidence.taskID)
                                detailRow(label: "Run ID", value: evidence.runID)
                            }
                            detailRow(label: "Updated", value: evidence.updatedAtLabel)
                            if let step = evidence.step {
                                if isAdvancedInformationDensityEnabled {
                                    detailRow(label: "Step", value: "\(step.stepID) • \(step.name)")
                                } else {
                                    detailRow(label: "Step", value: step.name)
                                }
                                detailRow(label: "Status", value: step.statusLabel)
                                if isAdvancedInformationDensityEnabled, let capability = step.capability {
                                    detailRow(label: "Capability", value: capability)
                                }
                                if isAdvancedInformationDensityEnabled, let interaction = step.interactionLevel {
                                    detailRow(label: "Interaction", value: interaction)
                                }
                                detailRow(label: "Step Updated", value: step.updatedAtLabel)
                                GroupBox("Step Input") {
                                    Text(step.inputSummary)
                                        .font(.caption)
                                        .frame(maxWidth: .infinity, alignment: .leading)
                                        .textSelection(.enabled)
                                }
                                GroupBox("Step Output") {
                                    Text(step.outputSummary)
                                        .font(.caption)
                                        .frame(maxWidth: .infinity, alignment: .leading)
                                        .textSelection(.enabled)
                                }
                                if let lastError = step.lastError {
                                    Text(lastError)
                                        .font(.caption2)
                                        .foregroundStyle(.red)
                                        .frame(maxWidth: .infinity, alignment: .leading)
                                }
                            } else {
                                Text("No step metadata found for the linked run.")
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }
                        }
                    }

                    GroupBox("Artifacts (\(evidence.artifacts.count))") {
                        VStack(alignment: .leading, spacing: 6) {
                            if evidence.artifacts.isEmpty {
                                Text("No related artifacts found.")
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            } else {
                                ForEach(evidence.artifacts) { artifact in
                                    VStack(alignment: .leading, spacing: 3) {
                                        Text(artifact.type)
                                            .font(.caption.weight(.semibold))
                                        if isAdvancedInformationDensityEnabled, let stepID = artifact.stepID {
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
                                        if isAdvancedInformationDensityEnabled, let hash = artifact.contentHash {
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

                    GroupBox("Audit Snippets (\(evidence.auditEntries.count))") {
                        VStack(alignment: .leading, spacing: 6) {
                            if evidence.auditEntries.isEmpty {
                                Text("No related audit entries found.")
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            } else {
                                ForEach(evidence.auditEntries) { entry in
                                    VStack(alignment: .leading, spacing: 3) {
                                        Text(entry.eventType)
                                            .font(.caption.weight(.semibold))
                                        if isAdvancedInformationDensityEnabled,
                                           let payloadSummary = entry.payloadSummary {
                                            Text(payloadSummary)
                                                .font(.caption2)
                                                .foregroundStyle(.secondary)
                                                .textSelection(.enabled)
                                        }
                                        Text("Created \(entry.createdAtLabel)")
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
                } else if item.runID == nil {
                    Text("Run detail is unavailable for this approval row, so inline evidence cannot be loaded.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                } else {
                    Text("Expand this section to load step inputs/outputs and related artifacts inline.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }

                HStack(spacing: 8) {
                    Button("Reload Evidence") {
                        state.loadApprovalEvidence(for: item, forceRefresh: true)
                    }
                    .panelActionStyle(.secondary)
                    .controlSize(.small)
                    .disabled(state.approvalEvidenceInFlightIDs.contains(item.id))

                    if state.approvalEvidenceInFlightIDs.contains(item.id) {
                        ProgressView()
                            .controlSize(.small)
                    }
                }

                if let status = state.approvalEvidenceStatusByID[item.id] {
                    Text(status)
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }
            }
            .padding(.top, 2)
        } label: {
            HStack(spacing: 8) {
                Label("Evidence", systemImage: "doc.text.magnifyingglass")
                    .font(.caption.weight(.semibold))
                if state.approvalEvidenceInFlightIDs.contains(item.id) {
                    ProgressView()
                        .controlSize(.small)
                } else if let evidence = state.approvalEvidenceByID[item.id] {
                    Text(isAdvancedInformationDensityEnabled ? "Run \(evidence.runID)" : "Loaded")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                } else if item.runID == nil {
                    Text("Run unavailable")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }
                Spacer(minLength: 0)
            }
        }
    }

    private func approvalEvidenceExpandedBinding(for item: ApprovalInboxItem) -> Binding<Bool> {
        Binding(
            get: { evidenceExpandedByID[item.id] ?? false },
            set: { isExpanded in
                evidenceExpandedByID[item.id] = isExpanded
                if isExpanded {
                    state.loadApprovalEvidence(for: item)
                }
            }
        )
    }

    private func detailsExpandedBinding(for item: ApprovalInboxItem) -> Binding<Bool> {
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

    private func identityDetailRow(label: String, actorID: String?) -> some View {
        let identity = state.principalIdentityDisplayValue(for: actorID)
        return IdentityDetailRowView(
            label: label,
            displayText: identity.displayText,
            rawID: identity.rawID
        )
    }

    private func decisionPhraseBinding(for item: ApprovalInboxItem) -> Binding<String> {
        Binding(
            get: {
                if let draft = decisionPhraseDrafts[item.id] {
                    return draft
                }
                if decisionIntent(for: item) == .approve {
                    return state.approvalRequiredPhrase(for: item)
                }
                return ""
            },
            set: { decisionPhraseDrafts[item.id] = $0 }
        )
    }

    private func decisionIntentBinding(for item: ApprovalInboxItem) -> Binding<ApprovalDecisionIntent> {
        Binding(
            get: { decisionIntent(for: item) },
            set: { intent in
                decisionIntentDrafts[item.id] = intent
                if intent == .approve {
                    if let phrase = decisionPhraseDrafts[item.id],
                       phrase.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                        decisionPhraseDrafts[item.id] = state.approvalRequiredPhrase(for: item)
                    } else if decisionPhraseDrafts[item.id] == nil {
                        decisionPhraseDrafts[item.id] = state.approvalRequiredPhrase(for: item)
                    }
                }
            }
        )
    }

    private func decisionIntent(for item: ApprovalInboxItem) -> ApprovalDecisionIntent {
        decisionIntentDrafts[item.id] ?? .approve
    }

    private func decisionActorBinding(for item: ApprovalInboxItem) -> Binding<String> {
        Binding(
            get: {
                if let draft = decisionActorDrafts[item.id] {
                    return draft
                }
                return state.defaultApprovalDecisionActor(for: item)
            },
            set: { decisionActorDrafts[item.id] = $0 }
        )
    }

    private func decisionRationaleBinding(for item: ApprovalInboxItem) -> Binding<String> {
        Binding(
            get: { decisionRationaleDrafts[item.id] ?? "" },
            set: { decisionRationaleDrafts[item.id] = $0 }
        )
    }

    private func decisionActorOptions(for item: ApprovalInboxItem) -> [String] {
        let selectedActorID = decisionActorBinding(for: item).wrappedValue
            .trimmingCharacters(in: .whitespacesAndNewlines)
        return state.approvalDecisionActorOptions(including: selectedActorID)
    }

    private func decisionActorValidationMessage(for item: ApprovalInboxItem) -> String? {
        let actorID = decisionActorBinding(for: item).wrappedValue.trimmingCharacters(in: .whitespacesAndNewlines)
        return state.approvalDecisionActorValidationMessage(actorID: actorID)
    }

    private func approvalPhraseValidationMessage(for item: ApprovalInboxItem) -> String? {
        guard decisionIntent(for: item) == .approve else {
            return nil
        }
        let phraseDraft = decisionPhraseBinding(for: item).wrappedValue
        return state.approvalApprovePhraseValidationMessage(phrase: phraseDraft, item: item)
    }

    private func submitDecision(_ item: ApprovalInboxItem) {
        let actorID = decisionActorBinding(for: item).wrappedValue.trimmingCharacters(in: .whitespacesAndNewlines)
        let phraseDraft = decisionPhraseBinding(for: item).wrappedValue.trimmingCharacters(in: .whitespacesAndNewlines)
        let rationale = decisionRationaleBinding(for: item).wrappedValue
        let intent = decisionIntent(for: item)

        if let actorValidationMessage = decisionActorValidationMessage(for: item) {
            state.approvalsActionStatusByID[item.id] = actorValidationMessage
            return
        }

        if let phraseValidationMessage = approvalPhraseValidationMessage(for: item) {
            state.approvalsActionStatusByID[item.id] = phraseValidationMessage
            return
        }

        let phrase: String
        switch intent {
        case .approve:
            phrase = state.approvalRequiredPhrase(for: item)
        case .reject:
            phrase = phraseDraft.isEmpty ? "REJECT" : phraseDraft
        }

        state.submitApprovalDecision(
            approvalID: item.id,
            decisionPhrase: phrase,
            decisionByActorID: actorID,
            rationale: rationale
        )
    }

    private func isDecisionSubmitDisabled(for item: ApprovalInboxItem) -> Bool {
        if state.approvalDecisionInFlightIDs.contains(item.id) {
            return true
        }
        if decisionActorValidationMessage(for: item) != nil {
            return true
        }
        if decisionIntent(for: item) == .approve,
           approvalPhraseValidationMessage(for: item) != nil {
            return true
        }
        return false
    }

    private func approvalActionStatusIsSuccess(_ value: String) -> Bool {
        let normalized = value.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        return normalized.contains("approved")
            || normalized.contains("decision recorded")
            || normalized.contains("resumed execution")
    }

    private func approvalActionStatusTint(_ value: String) -> Color {
        let normalized = value.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        if approvalActionStatusIsSuccess(value) {
            return .green
        }
        if normalized.contains("failed")
            || normalized.contains("error")
            || normalized.contains("required")
            || normalized.contains("set assistant access token")
            || normalized.contains("set local dev auth token") {
            return .orange
        }
        return .secondary
    }
}
