import Foundation
import SwiftUI

struct AutomationPanelView: View {
    @ObservedObject private var state: AppShellState
    @State private var isEditorPresented = false
    @State private var editorMode: AutomationEditorMode = .create
    @State private var editorDraft = AutomationEditorDraft.newSchedule(defaultActorID: "default")
    @State private var editorValidationMessage: String?
    @State private var pendingDeleteTrigger: AutomationTriggerItem?
    @State private var commFilterTokenDraftByFieldID: [String: String] = [:]
    @State private var isCommEventRawFilterExpanded = false
    @State private var useCommEventRawFilterOverride = false
    @State private var commEventRawFilterOverrideJSON = "{}"

    init(state: AppShellState) {
        self.state = state
    }

    var body: some View {
        VStack(spacing: 0) {
            header
                .padding(.horizontal, UIStyle.panelPadding)
                .padding(.vertical, 12)

            if let runtimeBannerMessage {
                RuntimeStateBanner(message: runtimeBannerMessage)
                    .padding(.horizontal, UIStyle.panelPadding)
                    .padding(.bottom, 12)
            }

            if let panelProblemRemediation {
                PanelProblemRemediationCardView(context: panelProblemRemediation) { actionID in
                    state.performPanelProblemRemediationAction(actionID, section: .automation)
                }
                .padding(.horizontal, UIStyle.panelPadding)
                .padding(.bottom, 12)
            }

            Divider()

            content
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(UIStyle.panelGradient)
        .sheet(isPresented: $isEditorPresented) {
            editorSheet
                .frame(minWidth: 560, minHeight: 520)
        }
        .confirmationDialog(
            "Delete automation trigger?",
            isPresented: deleteConfirmationBinding,
            presenting: pendingDeleteTrigger
        ) { trigger in
            Button("Delete Trigger", role: .destructive) {
                state.deleteAutomationTrigger(triggerID: trigger.id)
            }
            Button("Cancel", role: .cancel) {}
        } message: { trigger in
            Text("This removes \(trigger.directiveTitle) and its trigger configuration from daemon state.")
        }
    }

    private enum AutomationEditorMode: Equatable {
        case create
        case edit(triggerID: String)

        var title: String {
            switch self {
            case .create:
                return "New Automation Trigger"
            case .edit:
                return "Edit Automation Trigger"
            }
        }

        var submitLabel: String {
            switch self {
            case .create:
                return "Create Trigger"
            case .edit:
                return "Save Changes"
            }
        }

        var triggerID: String? {
            switch self {
            case .create:
                return nil
            case .edit(let triggerID):
                return triggerID
            }
        }
    }

    private struct AutomationCommEventFilterDraft {
        var channels: String
        var principalActorIDs: String
        var senderAllowlist: String
        var threadIDs: String
        var keywordContainsAny: String
        var keywordContainsAll: String
        var keywordExactPhrases: String

        static var empty: AutomationCommEventFilterDraft {
            AutomationCommEventFilterDraft(
                channels: "",
                principalActorIDs: "",
                senderAllowlist: "",
                threadIDs: "",
                keywordContainsAny: "",
                keywordContainsAll: "",
                keywordExactPhrases: ""
            )
        }
    }

    private enum CommEventFilterFieldID: String {
        case channels
        case principalActorIDs
        case senderAllowlist
        case threadIDs
        case keywordContainsAny
        case keywordContainsAll
        case keywordExactPhrases
    }

    private struct AutomationEditorDraft {
        var triggerType: String
        var subjectActorID: String
        var title: String
        var instruction: String
        var enabled: Bool
        var cooldownSeconds: Int
        var scheduleIntervalSeconds: Int
        var commEventFilter: AutomationCommEventFilterDraft

        static func newSchedule(defaultActorID: String) -> AutomationEditorDraft {
            AutomationEditorDraft(
                triggerType: "SCHEDULE",
                subjectActorID: defaultActorID,
                title: "",
                instruction: "",
                enabled: true,
                cooldownSeconds: 0,
                scheduleIntervalSeconds: 300,
                commEventFilter: .empty
            )
        }
    }

    private struct AutomationCommEventKeywordFilterPayload: Codable {
        var containsAny: [String]
        var containsAll: [String]
        var exactPhrases: [String]

        enum CodingKeys: String, CodingKey {
            case containsAny = "contains_any"
            case containsAll = "contains_all"
            case exactPhrases = "exact_phrases"
        }

        init(
            containsAny: [String] = [],
            containsAll: [String] = [],
            exactPhrases: [String] = []
        ) {
            self.containsAny = containsAny
            self.containsAll = containsAll
            self.exactPhrases = exactPhrases
        }

        init(from decoder: Decoder) throws {
            let container = try decoder.container(keyedBy: CodingKeys.self)
            containsAny = try container.decodeIfPresent([String].self, forKey: .containsAny) ?? []
            containsAll = try container.decodeIfPresent([String].self, forKey: .containsAll) ?? []
            exactPhrases = try container.decodeIfPresent([String].self, forKey: .exactPhrases) ?? []
        }
    }

    private struct AutomationCommEventFilterPayload: Codable {
        var channels: [String]
        var principalActorIDs: [String]
        var senderAllowlist: [String]
        var threadIDs: [String]
        var keywords: AutomationCommEventKeywordFilterPayload

        enum CodingKeys: String, CodingKey {
            case channels
            case principalActorIDs = "principal_actor_ids"
            case senderAllowlist = "sender_allowlist"
            case threadIDs = "thread_ids"
            case keywords
        }

        init(
            channels: [String] = [],
            principalActorIDs: [String] = [],
            senderAllowlist: [String] = [],
            threadIDs: [String] = [],
            keywords: AutomationCommEventKeywordFilterPayload = AutomationCommEventKeywordFilterPayload()
        ) {
            self.channels = channels
            self.principalActorIDs = principalActorIDs
            self.senderAllowlist = senderAllowlist
            self.threadIDs = threadIDs
            self.keywords = keywords
        }

        init(from decoder: Decoder) throws {
            let container = try decoder.container(keyedBy: CodingKeys.self)
            channels = try container.decodeIfPresent([String].self, forKey: .channels) ?? []
            principalActorIDs = try container.decodeIfPresent([String].self, forKey: .principalActorIDs) ?? []
            senderAllowlist = try container.decodeIfPresent([String].self, forKey: .senderAllowlist) ?? []
            threadIDs = try container.decodeIfPresent([String].self, forKey: .threadIDs) ?? []
            keywords = try container.decodeIfPresent(AutomationCommEventKeywordFilterPayload.self, forKey: .keywords)
                ?? AutomationCommEventKeywordFilterPayload()
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
        state.panelProblemRemediation(for: .automation)
    }

    private var isAdvancedInformationDensityEnabled: Bool {
        state.isAdvancedInformationDensityEnabled
    }

    private var header: some View {
        TahoeSectionHeader(
            title: "Automation",
            subtitle: state.automationStatusMessage ?? "Daemon-backed trigger inventory and management"
        ) {
            HStack(spacing: 8) {
                if hasActivitySpinner {
                    ProgressView()
                        .controlSize(.small)
                }

                Button {
                    openCreateEditor()
                } label: {
                    Label("New Trigger", systemImage: "plus")
                }
                .buttonStyle(.borderedProminent)
                .disabled(state.isAutomationCreateInFlight || state.isAutomationSimulationInFlight)

                Menu {
                    Button("Run Schedule Simulation") {
                        state.simulateAutomationScheduleNow()
                    }
                    Button("Run Comm Event Simulation") {
                        state.simulateAutomationCommEvent()
                    }
                } label: {
                    Label("Simulate", systemImage: "play.circle")
                }
                .quietButtonChrome()
                .disabled(state.isAutomationSimulationInFlight)

                Button {
                    state.refreshAutomationTriggers()
                } label: {
                    Label("Refresh", systemImage: "arrow.clockwise")
                }
                .quietButtonChrome()
                .disabled(
                    state.isAutomationLoading
                        || state.isAutomationFireHistoryLoading
                        || state.isAutomationSimulationInFlight
                )
                .accessibilityLabel("Refresh automation triggers")
            }
        }
    }

    private var hasActivitySpinner: Bool {
        state.isAutomationLoading
            || state.isAutomationFireHistoryLoading
            || state.isAutomationSimulationInFlight
            || state.isAutomationCreateInFlight
            || !state.automationUpdateInFlightIDs.isEmpty
            || !state.automationDeleteInFlightIDs.isEmpty
    }

    private var showLoadingSkeleton: Bool {
        (state.isAutomationLoading || state.isAutomationFireHistoryLoading || !state.hasLoadedAutomationPanelData)
            && state.automationTriggers.isEmpty
            && state.automationFireHistoryItems.isEmpty
    }

    private var automationSupplementaryStatusMessage: String? {
        let ignoredDefaults: Set<String> = [
            "no create/edit action run yet.",
            "no simulation run yet.",
            "waiting for trigger fire history."
        ]
        let headerStatus = normalizedStatusMessage(state.automationStatusMessage)?.lowercased()
        let candidates = [
            state.automationManagementStatusMessage,
            state.automationSimulationStatusMessage,
            state.automationFireHistoryStatusMessage
        ].compactMap(normalizedStatusMessage)

        return candidates.first { candidate in
            let normalized = candidate.lowercased()
            guard !ignoredDefaults.contains(normalized) else {
                return false
            }
            if let headerStatus {
                return normalized != headerStatus
            }
            return true
        }
    }

    @ViewBuilder
    private var content: some View {
        if showLoadingSkeleton {
            PanelLoadingSkeletonView(
                title: "Loading Automation",
                subtitle: "Fetching trigger inventory and recent fire history.",
                rowCount: 4
            )
            .frame(maxWidth: .infinity, maxHeight: .infinity)
        } else if state.automationTriggers.isEmpty && state.automationFireHistoryItems.isEmpty {
            ContentUnavailableView {
                Label("No Automation Activity Yet", systemImage: "clock.arrow.trianglehead.counterclockwise.rotate.90")
            } description: {
                VStack(alignment: .center, spacing: 6) {
                    Text("Create schedule or communication-event directives to populate this view.")
                    if let statusMessage = automationSupplementaryStatusMessage {
                        Text(statusMessage)
                            .foregroundStyle(.secondary)
                    }
                }
            } actions: {
                if state.localDevTokenConfigured {
                    Button {
                        openCreateEditor()
                    } label: {
                        Label("Create Trigger", systemImage: "plus")
                    }
                    .buttonStyle(.borderedProminent)
                }

                ForEach(state.automationEmptyStateRemediationActions) { action in
                    automationEmptyStateActionButton(action)
                }
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)
            .padding(UIStyle.panelPadding)
        } else {
            ScrollView {
                LazyVStack(alignment: .leading, spacing: UIStyle.standardSpacing) {
                    if let statusMessage = automationSupplementaryStatusMessage {
                        statusLine(statusMessage)
                    }

                    fireHistorySection

                    ForEach(state.automationTriggers) { trigger in
                        triggerCard(trigger)
                    }
                }
                .padding(UIStyle.panelPadding)
            }
        }
    }

    @ViewBuilder
    private func automationEmptyStateActionButton(_ action: EmptyStateRemediationAction) -> some View {
        if action.isProminent {
            Button {
                state.performEmptyStateRemediationAction(action.actionID)
            } label: {
                Label(action.title, systemImage: action.symbolName)
            }
            .buttonStyle(.borderedProminent)
            .disabled(action.isDisabled)
        } else {
            Button {
                state.performEmptyStateRemediationAction(action.actionID)
            } label: {
                Label(action.title, systemImage: action.symbolName)
            }
            .buttonStyle(.bordered)
            .disabled(action.isDisabled)
        }
    }

    private func statusLine(_ text: String) -> some View {
        Text(text)
            .font(.caption)
            .foregroundStyle(.secondary)
            .padding(.horizontal, 4)
    }

    private func normalizedStatusMessage(_ value: String?) -> String? {
        let trimmed = value?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        return trimmed.isEmpty ? nil : trimmed
    }

    private var fireHistorySection: some View {
        GroupBox {
            VStack(alignment: .leading, spacing: 10) {
                HStack(spacing: 8) {
                    Text("Recent Trigger Evaluations")
                        .font(.headline)
                    Spacer(minLength: 0)
                    Button("Open Tasks") {
                        state.tasksSearchSeed = nil
                        state.navigateToSection(.tasks)
                    }
                    .buttonStyle(.bordered)
                    .controlSize(.small)
                    Button("Open Inspect") {
                        state.clearInspectRunFocus()
                        state.navigateToSection(.inspect)
                    }
                    .buttonStyle(.bordered)
                    .controlSize(.small)
                }

                if state.isAutomationFireHistoryLoading && state.automationFireHistoryItems.isEmpty {
                    HStack(spacing: 8) {
                        ProgressView()
                            .controlSize(.small)
                        Text("Loading trigger fire history…")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                } else if state.automationFireHistoryItems.isEmpty {
                    Text("No trigger fire records yet. Run a simulation or wait for an automation event.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                } else {
                    VStack(alignment: .leading, spacing: 8) {
                        ForEach(state.automationFireHistoryItems.prefix(10)) { item in
                            fireHistoryRow(item)
                            if item.id != state.automationFireHistoryItems.prefix(10).last?.id {
                                Divider()
                            }
                        }
                    }
                }
            }
            .padding(.vertical, 2)
        }
        .groupBoxStyle(.automatic)
    }

    private func fireHistoryRow(_ item: AutomationFireHistoryItem) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(spacing: 8) {
                TahoeStatusBadge(
                    text: item.status.label,
                    symbolName: item.status.symbolName,
                    tint: item.status.tint
                )
                TahoeStatusBadge(
                    text: triggerTypeLabel(item.triggerType),
                    symbolName: "clock.arrow.circlepath",
                    tint: .orange
                )
                Spacer(minLength: 0)
                Text(item.firedAtLabel)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            VStack(alignment: .leading, spacing: 3) {
                if isAdvancedInformationDensityEnabled {
                    detailRow(label: "Trigger ID", value: item.triggerID)
                }
                detailRow(label: "Outcome", value: item.outcome)
                if isAdvancedInformationDensityEnabled {
                    detailRow(label: "Idempotency", value: item.idempotencySignal)
                    if let taskID = item.taskID {
                        detailRow(label: "Task ID", value: taskID)
                    }
                    if let runID = item.runID {
                        detailRow(label: "Run ID", value: runID)
                    }
                }
                if let taskClass = item.route.taskClass {
                    detailRow(label: "Task Class", value: taskClass)
                }
                if let provider = item.route.provider {
                    detailRow(label: "Provider", value: provider)
                }
                if let modelKey = item.route.modelKey {
                    detailRow(label: "Model", value: modelKey)
                }
                if isAdvancedInformationDensityEnabled, let routeSource = item.route.routeSource {
                    detailRow(label: "Route Source", value: routeSource)
                }
            }

            HStack(spacing: 8) {
                Button("Open Related Tasks") {
                    state.openTasksForAutomationFireHistory(item)
                }
                .buttonStyle(.bordered)
                .controlSize(.small)

                Button("Open Related Inspect") {
                    state.openInspectForAutomationFireHistory(item)
                }
                .buttonStyle(.bordered)
                .controlSize(.small)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    private func triggerCard(_ trigger: AutomationTriggerItem) -> some View {
        GroupBox {
            VStack(alignment: .leading, spacing: 10) {
                HStack(spacing: 8) {
                    Text(trigger.directiveTitle)
                        .font(.headline)
                    Spacer(minLength: 0)
                    TahoeStatusBadge(
                        text: trigger.enabled ? "Enabled" : "Disabled",
                        symbolName: trigger.enabled ? "checkmark.circle.fill" : "pause.circle.fill",
                        tint: trigger.enabled ? .green : .secondary
                    )
                    TahoeStatusBadge(
                        text: triggerTypeLabel(trigger.triggerType),
                        symbolName: "clock.arrow.trianglehead.counterclockwise.rotate.90",
                        tint: .orange
                    )
                }

                Text(trigger.directiveInstruction)
                    .font(.callout)
                    .foregroundStyle(.secondary)
                    .lineLimit(4)

                Divider()

                VStack(alignment: .leading, spacing: 4) {
                    detailRow(label: "Acting As", value: trigger.subjectPrincipalActor)
                    detailRow(label: "Cooldown", value: "\(trigger.cooldownSeconds)s")
                    detailRow(label: "Filter", value: trigger.filterSummary)
                    detailRow(label: "Updated", value: trigger.updatedAtLabel)
                    if isAdvancedInformationDensityEnabled {
                        detailRow(label: "Trigger ID", value: trigger.id)
                    }
                }

                HStack(spacing: 8) {
                    Button("Edit") {
                        openEditEditor(for: trigger)
                    }
                    .buttonStyle(.bordered)
                    .disabled(isTriggerActionInFlight(trigger.id))

                    Button(trigger.enabled ? "Disable" : "Enable") {
                        toggleTriggerEnabled(trigger)
                    }
                    .buttonStyle(.bordered)
                    .disabled(isTriggerActionInFlight(trigger.id))

                    Button("Delete", role: .destructive) {
                        pendingDeleteTrigger = trigger
                    }
                    .buttonStyle(.bordered)
                    .disabled(isTriggerActionInFlight(trigger.id))

                    if isTriggerActionInFlight(trigger.id) {
                        ProgressView()
                            .controlSize(.small)
                    }
                }

                if let actionStatus = state.automationActionStatusByID[trigger.id] {
                    Text(actionStatus)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }
            .padding(.vertical, 2)
        }
        .groupBoxStyle(.automatic)
    }

    private func detailRow(label: String, value: String) -> some View {
        HStack(alignment: .top, spacing: 8) {
            Text(label)
                .font(.caption.weight(.semibold))
                .foregroundStyle(.secondary)
                .frame(width: 106, alignment: .leading)
            Text(value)
                .font(.caption)
                .foregroundStyle(.secondary)
                .frame(maxWidth: .infinity, alignment: .leading)
                .textSelection(.enabled)
        }
    }

    private var editorSheet: some View {
        VStack(spacing: 0) {
            HStack(spacing: 10) {
                Text(editorMode.title)
                    .font(.title3.weight(.semibold))
                Spacer(minLength: 0)
                Button("Cancel") {
                    closeEditor()
                }
                .buttonStyle(.bordered)
                Button(editorMode.submitLabel) {
                    submitEditor()
                }
                .buttonStyle(.borderedProminent)
                .disabled(isEditorSubmitDisabled)
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 12)

            Divider()

            Form {
                Section("Trigger") {
                    if editorMode == .create {
                        Picker("Trigger Type", selection: $editorDraft.triggerType) {
                            Text("Schedule").tag("SCHEDULE")
                            Text("Comm Event").tag("ON_COMM_EVENT")
                        }
                    } else {
                        LabeledContent("Trigger Type") {
                            Text(triggerTypeLabel(editorDraft.triggerType))
                                .foregroundStyle(.secondary)
                        }
                    }

                    Picker("Acting As", selection: $editorDraft.subjectActorID) {
                        ForEach(automationActingAsOptions(selectedActorID: editorDraft.subjectActorID), id: \.self) { actorID in
                            Text(actorID).tag(actorID)
                        }
                    }
                    .pickerStyle(.menu)
                    Toggle("Enabled", isOn: $editorDraft.enabled)
                    Stepper("Cooldown Seconds: \(editorDraft.cooldownSeconds)", value: $editorDraft.cooldownSeconds, in: 0...86_400)
                }

                Section("Directive") {
                    TextField("Title", text: $editorDraft.title)
                    TextEditor(text: $editorDraft.instruction)
                        .font(.body)
                        .frame(minHeight: 88)
                }

                if normalizedTriggerType(editorDraft.triggerType) == "SCHEDULE" {
                    Section("Schedule Configuration") {
                        Stepper(
                            "Interval Seconds: \(editorDraft.scheduleIntervalSeconds)",
                            value: $editorDraft.scheduleIntervalSeconds,
                            in: 1...86_400
                        )
                    }
                } else {
                    Section("Comm Event Source Defaults") {
                        LabeledContent("Event Type") {
                            Text("MESSAGE")
                                .foregroundStyle(.secondary)
                        }
                        LabeledContent("Direction") {
                            Text("INBOUND")
                                .foregroundStyle(.secondary)
                        }
                        LabeledContent("Assistant Emitted") {
                            Text("false")
                                .foregroundStyle(.secondary)
                        }
                    }

                    Section("Comm Event Filter") {
                        commEventTokenEditor(
                            title: "Channels",
                            fieldID: .channels,
                            csvBinding: $editorDraft.commEventFilter.channels,
                            addPlaceholder: "message",
                            emptyStateText: "No channel filter. All logical channels are eligible.",
                            isChannelField: true
                        )

                        commEventTokenEditor(
                            title: "Principal Actor IDs",
                            fieldID: .principalActorIDs,
                            csvBinding: $editorDraft.commEventFilter.principalActorIDs,
                            addPlaceholder: "actor.requester",
                            emptyStateText: "No actor filter. Any actor ID can match."
                        )

                        commEventTokenEditor(
                            title: "Sender Allowlist",
                            fieldID: .senderAllowlist,
                            csvBinding: $editorDraft.commEventFilter.senderAllowlist,
                            addPlaceholder: "user@example.com",
                            emptyStateText: "No sender filter. Any sender can match."
                        )

                        commEventTokenEditor(
                            title: "Thread IDs",
                            fieldID: .threadIDs,
                            csvBinding: $editorDraft.commEventFilter.threadIDs,
                            addPlaceholder: "thread-123",
                            emptyStateText: "No thread filter. Any conversation thread can match."
                        )

                        commEventTokenEditor(
                            title: "Keywords Any",
                            fieldID: .keywordContainsAny,
                            csvBinding: $editorDraft.commEventFilter.keywordContainsAny,
                            addPlaceholder: "urgent",
                            emptyStateText: "Optional. Match when any listed keyword appears."
                        )

                        commEventTokenEditor(
                            title: "Keywords All",
                            fieldID: .keywordContainsAll,
                            csvBinding: $editorDraft.commEventFilter.keywordContainsAll,
                            addPlaceholder: "invoice",
                            emptyStateText: "Optional. Match only when all listed keywords appear."
                        )

                        commEventTokenEditor(
                            title: "Keywords Exact Phrases",
                            fieldID: .keywordExactPhrases,
                            csvBinding: $editorDraft.commEventFilter.keywordExactPhrases,
                            addPlaceholder: "go ahead",
                            emptyStateText: "Optional. Match when an exact phrase appears."
                        )

                        Button("Reset Filter") {
                            editorDraft.commEventFilter = .empty
                            useCommEventRawFilterOverride = false
                            syncCommEventRawFilterOverrideFromGuidedDraft()
                        }
                        .buttonStyle(.bordered)
                    }

                    Section {
                        DisclosureGroup("Advanced Raw Filter JSON", isExpanded: $isCommEventRawFilterExpanded) {
                            VStack(alignment: .leading, spacing: 8) {
                                Toggle("Use raw JSON override when saving", isOn: $useCommEventRawFilterOverride)
                                    .toggleStyle(.switch)

                                TextEditor(text: $commEventRawFilterOverrideJSON)
                                    .font(.system(.caption, design: .monospaced))
                                    .frame(minHeight: 100)
                                    .overlay(
                                        RoundedRectangle(cornerRadius: 8)
                                            .stroke(Color.secondary.opacity(0.25))
                                    )

                                HStack(spacing: 8) {
                                    Button("Load from Guided") {
                                        syncCommEventRawFilterOverrideFromGuidedDraft()
                                    }
                                    .buttonStyle(.bordered)

                                    Button("Apply to Guided") {
                                        applyCommEventRawFilterOverrideToGuidedDraft()
                                    }
                                    .buttonStyle(.bordered)
                                    .disabled(!isCommEventRawFilterJSONObjectValid)
                                }

                                Text(commEventRawFilterValidationMessage)
                                    .font(.caption)
                                    .foregroundStyle(
                                        isCommEventRawFilterJSONObjectValid ? Color.secondary : Color.orange
                                    )
                            }
                            .padding(.top, 4)
                        }
                    }

                    if isAdvancedInformationDensityEnabled {
                        Section("Filter Preview") {
                            Text(commEventFilterPreviewSummary(for: editorDraft.commEventFilter))
                                .font(.callout)
                                .foregroundStyle(.secondary)
                            Text(commEventFilterJSONString(fromDraft: editorDraft.commEventFilter))
                                .font(.system(.caption, design: .monospaced))
                                .foregroundStyle(.secondary)
                                .textSelection(.enabled)
                        }

                        let commHints = commEventFilterInlineHints(for: editorDraft.commEventFilter)
                        if !commHints.isEmpty {
                            Section("Validation Hints") {
                                ForEach(commHints, id: \.self) { hint in
                                    Text(hint)
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                }
                            }
                        }
                    }
                }

                if let editorValidationMessage {
                    Section("Validation") {
                        Text(editorValidationMessage)
                            .font(.callout)
                            .foregroundStyle(.orange)
                    }
                }

                if let managementStatus = state.automationManagementStatusMessage {
                    Section("Last Action") {
                        Text(managementStatus)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                }
            }
        }
    }

    private var isEditorSubmitDisabled: Bool {
        if state.isAutomationSimulationInFlight {
            return true
        }
        if case .create = editorMode {
            return state.isAutomationCreateInFlight
        }
        if let triggerID = editorMode.triggerID {
            return state.automationUpdateInFlightIDs.contains(triggerID)
                || state.automationDeleteInFlightIDs.contains(triggerID)
        }
        return false
    }

    private var deleteConfirmationBinding: Binding<Bool> {
        Binding(
            get: { pendingDeleteTrigger != nil },
            set: { shouldPresent in
                if !shouldPresent {
                    pendingDeleteTrigger = nil
                }
            }
        )
    }

    private func openCreateEditor() {
        editorMode = .create
        editorDraft = .newSchedule(defaultActorID: state.activePrincipalLabel)
        syncCommEventRawFilterOverrideFromGuidedDraft()
        useCommEventRawFilterOverride = false
        isCommEventRawFilterExpanded = false
        commFilterTokenDraftByFieldID = [:]
        editorValidationMessage = nil
        isEditorPresented = true
    }

    private func openEditEditor(for trigger: AutomationTriggerItem) {
        let normalizedType = normalizedTriggerType(trigger.triggerType)
        editorMode = .edit(triggerID: trigger.id)
        editorDraft = AutomationEditorDraft(
            triggerType: normalizedType,
            subjectActorID: trigger.subjectPrincipalActor,
            title: trigger.directiveTitle,
            instruction: trigger.directiveInstruction,
            enabled: trigger.enabled,
            cooldownSeconds: max(0, trigger.cooldownSeconds),
            scheduleIntervalSeconds: scheduleInterval(fromFilterJSON: trigger.filterSummary),
            commEventFilter: commEventFilterDraft(fromFilterJSON: trigger.filterSummary)
        )
        commEventRawFilterOverrideJSON = normalizedRawFilterJSONString(trigger.filterSummary)
        useCommEventRawFilterOverride = false
        isCommEventRawFilterExpanded = false
        commFilterTokenDraftByFieldID = [:]
        editorValidationMessage = nil
        isEditorPresented = true
    }

    private func closeEditor() {
        isEditorPresented = false
        editorValidationMessage = nil
    }

    private func submitEditor() {
        if let validationMessage = validateEditorDraft(editorDraft) {
            editorValidationMessage = validationMessage
            return
        }
        editorValidationMessage = nil

        let normalizedTriggerType = normalizedTriggerType(editorDraft.triggerType)
        let resolvedCommEventFilterJSON: String?
        if normalizedTriggerType == "ON_COMM_EVENT" {
            if useCommEventRawFilterOverride {
                guard isCommEventRawFilterJSONObjectValid else {
                    editorValidationMessage = commEventRawFilterValidationMessage
                    return
                }
                resolvedCommEventFilterJSON = GuidedEditorSupport.normalizedRawJSONObject(commEventRawFilterOverrideJSON)
            } else {
                resolvedCommEventFilterJSON = commEventFilterJSONString(fromDraft: editorDraft.commEventFilter)
            }
        } else {
            resolvedCommEventFilterJSON = nil
        }

        let input = AutomationTriggerMutationInput(
            triggerType: normalizedTriggerType,
            subjectActorID: editorDraft.subjectActorID,
            title: editorDraft.title,
            instruction: editorDraft.instruction,
            enabled: editorDraft.enabled,
            cooldownSeconds: editorDraft.cooldownSeconds,
            scheduleIntervalSeconds: normalizedTriggerType == "SCHEDULE"
                ? editorDraft.scheduleIntervalSeconds
                : nil,
            commEventFilterJSON: resolvedCommEventFilterJSON
        )

        switch editorMode {
        case .create:
            state.createAutomationTrigger(input)
        case .edit(let triggerID):
            state.updateAutomationTrigger(triggerID: triggerID, input: input)
        }

        closeEditor()
    }

    private func toggleTriggerEnabled(_ trigger: AutomationTriggerItem) {
        let draft = AutomationTriggerMutationInput(
            triggerType: normalizedTriggerType(trigger.triggerType),
            subjectActorID: trigger.subjectPrincipalActor,
            title: trigger.directiveTitle,
            instruction: trigger.directiveInstruction,
            enabled: !trigger.enabled,
            cooldownSeconds: max(0, trigger.cooldownSeconds),
            scheduleIntervalSeconds: normalizedTriggerType(trigger.triggerType) == "SCHEDULE"
                ? scheduleInterval(fromFilterJSON: trigger.filterSummary)
                : nil,
            commEventFilterJSON: normalizedTriggerType(trigger.triggerType) == "ON_COMM_EVENT"
                ? normalizedRawFilterJSONString(trigger.filterSummary)
                : nil
        )
        state.updateAutomationTrigger(triggerID: trigger.id, input: draft)
    }

    private func validateEditorDraft(_ draft: AutomationEditorDraft) -> String? {
        if draft.subjectActorID.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            return "Acting-as principal is required."
        }
        if let actingAsValidationMessage = state.actingAsValidationMessage(for: draft.subjectActorID) {
            return actingAsValidationMessage
        }
        if draft.title.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            return "Directive title is required."
        }
        if draft.instruction.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            return "Directive instruction is required."
        }
        if draft.cooldownSeconds < 0 {
            return "Cooldown seconds cannot be negative."
        }

        if normalizedTriggerType(draft.triggerType) == "SCHEDULE" {
            if draft.scheduleIntervalSeconds <= 0 {
                return "Schedule interval must be greater than zero."
            }
        } else if useCommEventRawFilterOverride, !isCommEventRawFilterJSONObjectValid {
            return commEventRawFilterValidationMessage
        }
        return nil
    }

    private func isTriggerActionInFlight(_ triggerID: String) -> Bool {
        state.automationUpdateInFlightIDs.contains(triggerID)
            || state.automationDeleteInFlightIDs.contains(triggerID)
    }

    private func normalizedTriggerType(_ raw: String) -> String {
        switch raw.trimmingCharacters(in: .whitespacesAndNewlines).uppercased() {
        case "SCHEDULE":
            return "SCHEDULE"
        case "ON_COMM_EVENT", "ON COMM EVENT":
            return "ON_COMM_EVENT"
        default:
            return "SCHEDULE"
        }
    }

    private func automationActingAsOptions(selectedActorID: String) -> [String] {
        state.actingAsOptions(including: selectedActorID)
    }

    private func scheduleInterval(fromFilterJSON raw: String) -> Int {
        guard let data = raw.data(using: .utf8),
              let object = try? JSONSerialization.jsonObject(with: data) as? [String: Any] else {
            return 300
        }

        if let intValue = object["interval_seconds"] as? Int, intValue > 0 {
            return intValue
        }
        if let numberValue = object["interval_seconds"] as? NSNumber, numberValue.intValue > 0 {
            return numberValue.intValue
        }
        return 300
    }

    private func normalizedRawFilterJSONString(_ raw: String) -> String {
        GuidedEditorSupport.normalizedRawJSONObject(raw)
    }

    private func commEventFilterDraft(fromFilterJSON raw: String) -> AutomationCommEventFilterDraft {
        let payload = decodedCommEventFilterPayload(fromFilterJSON: raw)
        return AutomationCommEventFilterDraft(
            channels: csvString(from: payload.channels),
            principalActorIDs: csvString(from: payload.principalActorIDs),
            senderAllowlist: csvString(from: payload.senderAllowlist),
            threadIDs: csvString(from: payload.threadIDs),
            keywordContainsAny: csvString(from: payload.keywords.containsAny),
            keywordContainsAll: csvString(from: payload.keywords.containsAll),
            keywordExactPhrases: csvString(from: payload.keywords.exactPhrases)
        )
    }

    private func decodedCommEventFilterPayload(fromFilterJSON raw: String) -> AutomationCommEventFilterPayload {
        let normalizedRaw = normalizedRawFilterJSONString(raw)
        guard let data = normalizedRaw.data(using: .utf8),
              let payload = try? JSONDecoder().decode(AutomationCommEventFilterPayload.self, from: data) else {
            return AutomationCommEventFilterPayload()
        }
        return AutomationCommEventFilterPayload(
            channels: normalizedChannelTokenList(from: payload.channels),
            principalActorIDs: normalizedTokenList(from: payload.principalActorIDs),
            senderAllowlist: normalizedTokenList(from: payload.senderAllowlist),
            threadIDs: normalizedTokenList(from: payload.threadIDs),
            keywords: AutomationCommEventKeywordFilterPayload(
                containsAny: normalizedTokenList(from: payload.keywords.containsAny),
                containsAll: normalizedTokenList(from: payload.keywords.containsAll),
                exactPhrases: normalizedTokenList(from: payload.keywords.exactPhrases)
            )
        )
    }

    private func commEventFilterPayload(fromDraft draft: AutomationCommEventFilterDraft) -> AutomationCommEventFilterPayload {
        AutomationCommEventFilterPayload(
            channels: normalizedChannelTokenList(fromCommaSeparated: draft.channels),
            principalActorIDs: normalizedTokenList(fromCommaSeparated: draft.principalActorIDs),
            senderAllowlist: normalizedTokenList(fromCommaSeparated: draft.senderAllowlist),
            threadIDs: normalizedTokenList(fromCommaSeparated: draft.threadIDs),
            keywords: AutomationCommEventKeywordFilterPayload(
                containsAny: normalizedTokenList(fromCommaSeparated: draft.keywordContainsAny),
                containsAll: normalizedTokenList(fromCommaSeparated: draft.keywordContainsAll),
                exactPhrases: normalizedTokenList(fromCommaSeparated: draft.keywordExactPhrases)
            )
        )
    }

    private func commEventFilterJSONString(fromDraft draft: AutomationCommEventFilterDraft) -> String {
        let payload = commEventFilterPayload(fromDraft: draft)
        let encoder = JSONEncoder()
        encoder.outputFormatting = [.sortedKeys]
        guard let data = try? encoder.encode(payload),
              let encoded = String(data: data, encoding: .utf8) else {
            return "{}"
        }
        return encoded
    }

    private func commEventFilterPreviewSummary(for draft: AutomationCommEventFilterDraft) -> String {
        let payload = commEventFilterPayload(fromDraft: draft)
        var clauses: [String] = []
        if !payload.channels.isEmpty {
            clauses.append("channels = \(payload.channels.joined(separator: ", "))")
        }
        if !payload.principalActorIDs.isEmpty {
            clauses.append("principal actors = \(payload.principalActorIDs.joined(separator: ", "))")
        }
        if !payload.senderAllowlist.isEmpty {
            clauses.append("senders = \(payload.senderAllowlist.joined(separator: ", "))")
        }
        if !payload.threadIDs.isEmpty {
            clauses.append("threads = \(payload.threadIDs.joined(separator: ", "))")
        }
        if !payload.keywords.containsAny.isEmpty {
            clauses.append("keywords(any) = \(payload.keywords.containsAny.joined(separator: ", "))")
        }
        if !payload.keywords.containsAll.isEmpty {
            clauses.append("keywords(all) = \(payload.keywords.containsAll.joined(separator: ", "))")
        }
        if !payload.keywords.exactPhrases.isEmpty {
            clauses.append("keywords(phrase) = \(payload.keywords.exactPhrases.joined(separator: ", "))")
        }

        if clauses.isEmpty {
            return "Broad match: evaluates every inbound MESSAGE event for this workspace."
        }
        return "Matches inbound MESSAGE events where " + clauses.joined(separator: " • ")
    }

    private func commEventFilterInlineHints(for draft: AutomationCommEventFilterDraft) -> [String] {
        let payload = commEventFilterPayload(fromDraft: draft)
        var hints: [String] = []

        if payload.channels.isEmpty &&
            payload.principalActorIDs.isEmpty &&
            payload.senderAllowlist.isEmpty &&
            payload.threadIDs.isEmpty &&
            payload.keywords.containsAny.isEmpty &&
            payload.keywords.containsAll.isEmpty &&
            payload.keywords.exactPhrases.isEmpty {
            hints.append("No filter values set. This trigger will evaluate every inbound MESSAGE event.")
        }

        let selectedActor = editorDraft.subjectActorID.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        if !selectedActor.isEmpty &&
            !payload.principalActorIDs.isEmpty &&
            !payload.principalActorIDs.contains(selectedActor) {
            hints.append("Principal Actor IDs do not include the selected Acting As value. Daemon validation may warn.")
        }

        if let duplicateHint = duplicateChannelNormalizationHint(raw: draft.channels) {
            hints.append(duplicateHint)
        }
        if let duplicateHint = duplicateNormalizationHint(label: "Principal Actor IDs", raw: draft.principalActorIDs) {
            hints.append(duplicateHint)
        }
        if let duplicateHint = duplicateNormalizationHint(label: "Sender Allowlist", raw: draft.senderAllowlist) {
            hints.append(duplicateHint)
        }
        if let duplicateHint = duplicateNormalizationHint(label: "Thread IDs", raw: draft.threadIDs) {
            hints.append(duplicateHint)
        }
        if let duplicateHint = duplicateNormalizationHint(label: "Keywords Any", raw: draft.keywordContainsAny) {
            hints.append(duplicateHint)
        }
        if let duplicateHint = duplicateNormalizationHint(label: "Keywords All", raw: draft.keywordContainsAll) {
            hints.append(duplicateHint)
        }
        if let duplicateHint = duplicateNormalizationHint(label: "Keywords Exact Phrases", raw: draft.keywordExactPhrases) {
            hints.append(duplicateHint)
        }
        if hints.isEmpty {
            hints.append("Filter values are normalized before save; channels are canonicalized to app/message/voice.")
        }
        return hints
    }

    private var isCommEventRawFilterJSONObjectValid: Bool {
        GuidedEditorSupport.isValidRawJSONObject(commEventRawFilterOverrideJSON)
    }

    private var commEventRawFilterValidationMessage: String {
        isCommEventRawFilterJSONObjectValid
            ? "Raw filter JSON is valid. Guided fields are ignored while override is enabled."
            : "Raw filter JSON must be a valid JSON object before save."
    }

    private func syncCommEventRawFilterOverrideFromGuidedDraft() {
        commEventRawFilterOverrideJSON = commEventFilterJSONString(fromDraft: editorDraft.commEventFilter)
    }

    private func applyCommEventRawFilterOverrideToGuidedDraft() {
        guard isCommEventRawFilterJSONObjectValid else {
            editorValidationMessage = "Raw filter JSON must be a valid object before applying."
            return
        }
        editorDraft.commEventFilter = commEventFilterDraft(fromFilterJSON: commEventRawFilterOverrideJSON)
        useCommEventRawFilterOverride = false
        editorValidationMessage = nil
    }

    @ViewBuilder
    private func commEventTokenEditor(
        title: String,
        fieldID: CommEventFilterFieldID,
        csvBinding: Binding<String>,
        addPlaceholder: String,
        emptyStateText: String,
        isChannelField: Bool = false
    ) -> some View {
        let tokens = normalizedTokens(from: csvBinding.wrappedValue, isChannelField: isChannelField)
        VStack(alignment: .leading, spacing: 6) {
            Text(title)
                .font(.caption.weight(.semibold))
                .foregroundStyle(.secondary)

            if tokens.isEmpty {
                Text(emptyStateText)
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            } else {
                LazyVGrid(columns: [GridItem(.adaptive(minimum: 160), spacing: 6)], spacing: 6) {
                    ForEach(tokens, id: \.self) { token in
                        HStack(spacing: 6) {
                            Text(token)
                                .font(.caption)
                                .lineLimit(1)
                                .truncationMode(.tail)
                            Spacer(minLength: 0)
                            Button {
                                removeCommEventToken(
                                    token,
                                    from: csvBinding,
                                    isChannelField: isChannelField
                                )
                            } label: {
                                Image(systemName: "xmark.circle.fill")
                                    .foregroundStyle(.secondary)
                            }
                            .buttonStyle(.plain)
                            .accessibilityLabel("Remove \(token)")
                        }
                        .padding(.horizontal, 8)
                        .padding(.vertical, 4)
                        .background(
                            Capsule()
                                .fill(Color.secondary.opacity(0.16))
                        )
                    }
                }
            }

            HStack(spacing: 8) {
                TextField(
                    addPlaceholder,
                    text: commEventTokenDraftBinding(for: fieldID)
                )
                .textFieldStyle(.roundedBorder)

                Button("Add") {
                    addCommEventToken(
                        to: csvBinding,
                        fieldID: fieldID,
                        isChannelField: isChannelField
                    )
                }
                .buttonStyle(.bordered)
                .disabled(
                    commEventTokenDraftBinding(for: fieldID)
                        .wrappedValue
                        .trimmingCharacters(in: .whitespacesAndNewlines)
                        .isEmpty
                )
            }
        }
        .padding(.vertical, 2)
    }

    private func commEventTokenDraftBinding(for fieldID: CommEventFilterFieldID) -> Binding<String> {
        Binding(
            get: { commFilterTokenDraftByFieldID[fieldID.rawValue] ?? "" },
            set: { commFilterTokenDraftByFieldID[fieldID.rawValue] = $0 }
        )
    }

    private func normalizedTokens(from raw: String, isChannelField: Bool) -> [String] {
        if isChannelField {
            return GuidedEditorSupport.normalizeChannelEntries(fromCommaSeparated: raw).values
        }
        return GuidedEditorSupport.normalizeTokenEntries(fromCommaSeparated: raw).values
    }

    private func addCommEventToken(
        to csvBinding: Binding<String>,
        fieldID: CommEventFilterFieldID,
        isChannelField: Bool
    ) {
        let draft = commEventTokenDraftBinding(for: fieldID).wrappedValue
            .trimmingCharacters(in: .whitespacesAndNewlines)
        guard !draft.isEmpty else {
            return
        }
        let existing = normalizedTokens(from: csvBinding.wrappedValue, isChannelField: isChannelField)
        let appended = existing + [draft]
        if isChannelField {
            csvBinding.wrappedValue = csvString(
                from: GuidedEditorSupport.normalizeChannelEntries(
                    fromCommaSeparated: appended.joined(separator: ",")
                ).values
            )
        } else {
            csvBinding.wrappedValue = csvString(
                from: GuidedEditorSupport.normalizeTokenEntries(
                    fromCommaSeparated: appended.joined(separator: ",")
                ).values
            )
        }
        commFilterTokenDraftByFieldID[fieldID.rawValue] = ""
        syncCommEventRawFilterOverrideFromGuidedDraft()
    }

    private func removeCommEventToken(
        _ token: String,
        from csvBinding: Binding<String>,
        isChannelField: Bool
    ) {
        let values = normalizedTokens(from: csvBinding.wrappedValue, isChannelField: isChannelField)
            .filter { $0 != token }
        csvBinding.wrappedValue = csvString(from: values)
        syncCommEventRawFilterOverrideFromGuidedDraft()
    }

    private func duplicateChannelNormalizationHint(raw: String) -> String? {
        let normalized = GuidedEditorSupport.normalizeChannelEntries(fromCommaSeparated: raw)
        if normalized.duplicateCount > 0 {
            return "Channels: \(normalized.duplicateCount) duplicate entr\(normalized.duplicateCount == 1 ? "y" : "ies") will be removed."
        }
        return nil
    }

    private func duplicateNormalizationHint(label: String, raw: String) -> String? {
        let normalized = GuidedEditorSupport.normalizeTokenEntries(fromCommaSeparated: raw)
        if normalized.duplicateCount > 0 {
            return "\(label): \(normalized.duplicateCount) duplicate entr\(normalized.duplicateCount == 1 ? "y" : "ies") will be removed."
        }
        return nil
    }

    private func csvString(from values: [String]) -> String {
        values.joined(separator: ", ")
    }

    private func normalizedTokenList(fromCommaSeparated raw: String) -> [String] {
        GuidedEditorSupport.normalizeTokenEntries(fromCommaSeparated: raw).values
    }

    private func normalizedTokenList(from values: [String]) -> [String] {
        normalizedTokenList(fromCommaSeparated: values.joined(separator: ","))
    }

    private func normalizedChannelTokenList(fromCommaSeparated raw: String) -> [String] {
        GuidedEditorSupport.normalizeChannelEntries(fromCommaSeparated: raw).values
    }

    private func normalizedChannelTokenList(from values: [String]) -> [String] {
        normalizedChannelTokenList(fromCommaSeparated: values.joined(separator: ","))
    }

    private func triggerTypeLabel(_ raw: String) -> String {
        switch normalizedTriggerType(raw) {
        case "SCHEDULE":
            return "Schedule"
        case "ON_COMM_EVENT":
            return "Comm Event"
        default:
            return raw.replacingOccurrences(of: "_", with: " ").capitalized
        }
    }
}
