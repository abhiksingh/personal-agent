import SwiftUI

struct CommunicationsPanelView: View {
    @ObservedObject private var state: AppShellState
    @FocusState private var isSearchFieldFocused: Bool
    @State private var searchText = ""
    @State private var selectedChannelFilterID = FilterID.allChannels
    @State private var selectedDirectionFilter: CommunicationDirectionFilter = .all
    @State private var selectedThreadFilterID = FilterID.allThreads
    @State private var activeComposeContext: ComposeContext? = nil
    @State private var composeSourceChannelDraft = "message"
    @State private var composeThreadIDDraft = ""
    @State private var composeConnectorHintDraft = ""
    @State private var composeDestinationDraft = ""
    @State private var composeMessageDraft = ""
    @State private var composeReceiptBaselineID: String? = nil
    @State private var composeAdvancedExpanded = false
    @State private var compactScanModeEnabled = false
    @State private var handledThreadIDs: Set<String> = []
    @State private var followUpThreadIDs: Set<String> = []
    @State private var seenThreadIDs: Set<String> = []
    @State private var newThreadIDs: Set<String> = []
    @State private var expandedThreadDetailIDs: Set<String> = []
    @State private var expandedEventDetailIDs: Set<String> = []
    @State private var expandedCallSessionDetailIDs: Set<String> = []
    @State private var expandedDeliveryAttemptDetailIDs: Set<String> = []
    @State private var expandedContinuityDetailIDs: Set<String> = []
    @State private var latestSendDetailsExpanded = false
    @State private var isResetContextConfirmationPresented = false

    private enum FilterID {
        static let allChannels = CommunicationsFilterContext.allChannelsID
        static let allThreads = CommunicationsFilterContext.allThreadsID
    }

    private struct ComposeContext: Identifiable {
        enum Flow {
            case newMessage
            case reply
            case startCall

            var persistenceID: String {
                switch self {
                case .newMessage:
                    return "new_message"
                case .reply:
                    return "reply"
                case .startCall:
                    return "start_call"
                }
            }

            init?(persistenceID: String) {
                switch persistenceID.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
                case "new_message":
                    self = .newMessage
                case "reply":
                    self = .reply
                case "start_call":
                    self = .startCall
                default:
                    return nil
                }
            }

            var helperCopy: String {
                switch self {
                case .newMessage:
                    return "Send a message quickly. Use Advanced only when you need channel or conversation overrides."
                case .reply:
                    return "Reply keeps the current conversation context. Leave Recipient empty to send within that conversation."
                case .startCall:
                    return "Start an outbound call quickly. Use Advanced to adjust routing context."
                }
            }

            var submitActionTitle: String {
                switch self {
                case .newMessage, .reply:
                    return "Send"
                case .startCall:
                    return "Start Call"
                }
            }
        }

        let id = UUID()
        let flow: Flow
    }

    private enum CommunicationDirectionFilter: String, CaseIterable, Identifiable {
        case all
        case inbound
        case outbound
        case other

        var id: String { rawValue }

        var label: String {
            switch self {
            case .all:
                return "All Directions"
            case .inbound:
                return "Inbound"
            case .outbound:
                return "Outbound"
            case .other:
                return "Other"
            }
        }

        func matches(_ raw: String?) -> Bool {
            switch self {
            case .all:
                return true
            case .inbound:
                return Self.normalize(raw) == "inbound"
            case .outbound:
                return Self.normalize(raw) == "outbound"
            case .other:
                let normalized = Self.normalize(raw)
                guard !normalized.isEmpty else {
                    return true
                }
                return normalized != "inbound" && normalized != "outbound"
            }
        }

        private static func normalize(_ raw: String?) -> String {
            raw?
                .trimmingCharacters(in: .whitespacesAndNewlines)
                .lowercased() ?? ""
        }
    }

    private struct ThreadFilterOption: Identifiable {
        let id: String
        let label: String
    }

    private struct ChannelGroup<Item>: Identifiable {
        let channelID: String
        let items: [Item]
        var id: String { channelID }
    }

    private struct ContinuityDetailRow: Identifiable {
        let id: String
        let label: String
        let value: String
    }

    init(state: AppShellState) {
        self.state = state
    }

    var body: some View {
        lifecycleBoundBody
    }

    private var panelBody: some View {
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
        .sheet(item: $activeComposeContext) { context in
            composeSheet(context)
                .frame(minWidth: 560, minHeight: 460)
        }
        .confirmationDialog(
            "Reset communications context for this workspace?",
            isPresented: $isResetContextConfirmationPresented
        ) {
            Button("Reset Context", role: .destructive) {
                resetWorkspaceContinuityContext()
            }
            Button("Cancel", role: .cancel) {}
        } message: {
            Text("This clears panel filters, compact mode, triage markers, and any saved compose draft for the current workspace.")
        }
    }

    @ViewBuilder
    private var supplementaryCards: some View {
        identityContextBar
            .padding(.horizontal, UIStyle.panelPadding)
            .padding(.bottom, 12)

        if let latestReceipt = state.latestCommunicationSendReceipt {
            latestCommunicationSendCard(latestReceipt)
                .padding(.horizontal, UIStyle.panelPadding)
                .padding(.bottom, 12)
        } else if let sendStatus = nonEmpty(state.communicationSendStatusMessage),
                  sendStatus != "No outbound communication sent yet." {
            Text(sendStatus)
                .font(.caption)
                .foregroundStyle(.secondary)
                .frame(maxWidth: .infinity, alignment: .leading)
                .padding(.horizontal, UIStyle.panelPadding)
                .padding(.bottom, 12)
        }
    }

    private var lifecycleBoundBody: some View {
        composeLifecycleBoundBody(
            applyingTo: filterLifecycleBoundBody(
                applyingTo: dataLifecycleBoundBody(
                    applyingTo: panelBody.onAppear(perform: handlePanelAppear)
                )
            )
        )
        .onChange(of: state.latestCommunicationSendReceipt?.id) { _, newValue in
            handleLatestSendReceiptChanged(newValue)
        }
    }

    private func dataLifecycleBoundBody<Content: View>(applyingTo content: Content) -> some View {
        content
            .onChange(of: state.communicationThreads.map(\.id)) { _, _ in
                normalizeThreadSelection()
                reconcileThreadTriageState()
                invalidateComposeThreadContextIfMissing()
                reconcileDetailDisclosureState()
            }
            .onChange(of: state.communicationEvents.map(\.id)) { _, _ in
                reconcileDetailDisclosureState()
            }
            .onChange(of: state.communicationCallSessions.map(\.id)) { _, _ in
                reconcileDetailDisclosureState()
            }
            .onChange(of: state.communicationDeliveryAttempts.map(\.id)) { _, _ in
                reconcileDetailDisclosureState()
            }
            .onChange(of: state.communicationContinuityItems.map(\.id)) { _, _ in
                reconcileDetailDisclosureState()
            }
            .onChange(of: state.workspaceLabel) { _, _ in
                handleWorkspaceChanged()
            }
    }

    private func filterLifecycleBoundBody<Content: View>(applyingTo content: Content) -> some View {
        content
            .onChange(of: searchText) { _, _ in
                persistFilterContext()
            }
            .onChange(of: selectedChannelFilterID) { _, _ in
                persistFilterContext()
            }
            .onChange(of: selectedDirectionFilter) { _, _ in
                persistFilterContext()
            }
            .onChange(of: selectedThreadFilterID) { _, _ in
                persistFilterContext()
                state.refreshCommunicationAttempts(threadID: selectedThreadIDForAttemptQuery)
            }
            .onChange(of: compactScanModeEnabled) { _, _ in
                persistFilterContext()
            }
    }

    private func composeLifecycleBoundBody<Content: View>(applyingTo content: Content) -> some View {
        content
            .onChange(of: activeComposeContext?.id) { _, _ in
                persistComposeDraftContext()
            }
            .onChange(of: composeSourceChannelDraft) { _, _ in
                persistComposeDraftContext()
            }
            .onChange(of: composeThreadIDDraft) { _, _ in
                persistComposeDraftContext()
            }
            .onChange(of: composeConnectorHintDraft) { _, _ in
                persistComposeDraftContext()
            }
            .onChange(of: composeDestinationDraft) { _, _ in
                persistComposeDraftContext()
            }
            .onChange(of: composeMessageDraft) { _, _ in
                persistComposeDraftContext()
            }
    }

    private func handlePanelAppear() {
        applyPersistedFilterContext()
        applyPersistedTriageContext()
        applyPersistedComposeDraftContext()
        reconcileThreadTriageState(bootstrapSeenState: true)
        reconcileDetailDisclosureState()
        state.refreshCommunicationAttempts(threadID: selectedThreadIDForAttemptQuery)
        focusSearchFieldForKeyboardTraversal()
    }

    private func handleWorkspaceChanged() {
        applyPersistedFilterContext()
        applyPersistedTriageContext()
        applyPersistedComposeDraftContext()
        reconcileThreadTriageState(bootstrapSeenState: true)
        reconcileDetailDisclosureState()
        state.refreshCommunicationAttempts(threadID: selectedThreadIDForAttemptQuery)
    }

    private func handleLatestSendReceiptChanged(_ newValue: String?) {
        reconcileDetailDisclosureState()
        guard let composeContext = activeComposeContext,
              let newValue,
              newValue != composeReceiptBaselineID,
              let receipt = state.latestCommunicationSendReceipt,
              receipt.success else {
            return
        }
        composeReceiptBaselineID = receipt.id
        switch composeContext.flow {
        case .newMessage, .reply, .startCall:
            activeComposeContext = nil
            composeAdvancedExpanded = false
            persistComposeDraftContext()
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

    private var workspaceIdentityDisplay: IdentityDisplayValue {
        state.workspaceIdentityDisplayValue(for: state.workspaceLabel)
    }

    private var principalIdentityDisplay: IdentityDisplayValue {
        state.principalIdentityDisplayValue(for: state.selectedPrincipal)
    }

    private var header: some View {
        TahoeSectionHeader(
            title: "Communications",
            subtitle: state.communicationsStatusMessage ?? "Conversations, events, and calls"
        ) {
            HStack(spacing: 8) {
                if state.isCommunicationSendInFlight {
                    ProgressView()
                        .controlSize(.small)
                }
                if state.isCommunicationsLoading {
                    ProgressView()
                        .controlSize(.small)
                }

                Button {
                    presentCompose(
                        flow: .newMessage,
                        sourceChannel: "message",
                        threadID: nil,
                        connectorID: nil,
                        destination: nil,
                        message: ""
                    )
                } label: {
                    Label("New Message", systemImage: "square.and.pencil")
                }
                .panelActionStyle(.primary)
                .disabled(state.isCommunicationSendInFlight)
                .accessibilityLabel("Compose new communication message")

                Button {
                    presentCompose(
                        flow: .startCall,
                        sourceChannel: "voice",
                        threadID: nil,
                        connectorID: "twilio",
                        destination: nil,
                        message: "Starting a call from Personal Agent."
                    )
                } label: {
                    Label("Start Call", systemImage: "phone.badge.plus")
                }
                .panelActionStyle(.secondary)
                .disabled(state.isCommunicationSendInFlight)
                .accessibilityLabel("Compose new outbound call")

                Button {
                    state.refreshCommunicationsInbox()
                } label: {
                    Label("Refresh", systemImage: "arrow.clockwise")
                }
                .panelActionStyle(.secondary)
                .disabled(state.isCommunicationsLoading || state.isCommunicationSendInFlight)
                .accessibilityLabel("Refresh communications inbox")
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
                    TextField(
                        "Search conversations, events, addresses, and calls",
                        text: $searchText
                    )
                    .textFieldStyle(.plain)
                    .focused($isSearchFieldFocused)
                    .accessibilityLabel(UIAccessibilityContract.communicationsSearchLabel)
                    .accessibilityHint(UIAccessibilityContract.communicationsSearchHint)
                    .accessibilityIdentifier(UIAccessibilityContract.communicationsSearchIdentifier)
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
                Picker("Channel", selection: $selectedChannelFilterID) {
                    Text("All Channels").tag(FilterID.allChannels)
                    ForEach(availableChannels, id: \.self) { channel in
                        Text(channel).tag(channel)
                    }
                }
                .pickerStyle(.menu)
                .frame(maxWidth: 200)

                Picker("Direction", selection: $selectedDirectionFilter) {
                    ForEach(CommunicationDirectionFilter.allCases) { filter in
                        Text(filter.label).tag(filter)
                    }
                }
                .pickerStyle(.menu)
                .frame(maxWidth: 180)

                Picker("Thread", selection: $selectedThreadFilterID) {
                    ForEach(threadFilterOptions) { option in
                        Text(option.label).tag(option.id)
                    }
                }
                .pickerStyle(.menu)
                .frame(maxWidth: 240)

                Spacer(minLength: 0)

                Text(filteredSummaryLabel)
                    .font(.caption)
                    .foregroundStyle(.secondary)

                if let triageSummaryLabel {
                    Text(triageSummaryLabel)
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }

                Toggle("Compact Scan", isOn: $compactScanModeEnabled)
                    .toggleStyle(.switch)
                    .controlSize(.small)
            }
        }
    }

    private var identityContextBar: some View {
        HStack(spacing: 12) {
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                Text("Workspace")
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(.secondary)
                IdentityValueInlineView(
                    displayText: workspaceIdentityDisplay.displayText,
                    rawID: workspaceIdentityDisplay.rawID,
                    valueFont: .caption
                )
            }

            HStack(alignment: .firstTextBaseline, spacing: 8) {
                Text("Principal")
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(.secondary)
                IdentityValueInlineView(
                    displayText: principalIdentityDisplay.displayText,
                    rawID: principalIdentityDisplay.rawID,
                    valueFont: .caption
                )
            }

            Spacer(minLength: 0)
        }
        .padding(10)
        .cardSurface(.subtle)
    }

    @ViewBuilder
    private var content: some View {
        if showLoadingSkeleton {
            PanelLoadingSkeletonView(
                title: "Loading Communications",
                subtitle: "Fetching conversations, events, calls, and delivery attempts.",
                rowCount: 4
            )
            .frame(maxWidth: .infinity, maxHeight: .infinity)
        } else if state.communicationThreads.isEmpty
            && state.communicationEvents.isEmpty
            && state.communicationCallSessions.isEmpty
            && state.communicationContinuityItems.isEmpty {
            PanelRemediationEmptyStateView(
                title: "No Communication Activity",
                systemImage: "tray.full",
                description: "Conversations, events, and calls appear here when activity starts.",
                statusMessage: state.communicationsStatusMessage,
                headerStatusMessage: state.communicationsStatusMessage,
                actions: state.communicationsEmptyStateRemediationActions
            ) { actionID in
                state.performEmptyStateRemediationAction(actionID)
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)
            .padding(UIStyle.panelPadding)
        } else if filteredThreads.isEmpty
            && filteredEvents.isEmpty
            && filteredCallSessions.isEmpty
            && filteredDeliveryAttempts.isEmpty
            && filteredContinuityItems.isEmpty {
            ContentUnavailableView {
                Label("No Results for Current Filters", systemImage: "line.3.horizontal.decrease.circle")
            } description: {
                Text("Adjust search, channel, conversation, or direction filters to see results.")
            } actions: {
                Button("Clear Filters") {
                    clearFilters()
                }
                .panelActionStyle(.primary)
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)
            .padding(UIStyle.panelPadding)
        } else {
            ScrollView {
                LazyVStack(alignment: .leading, spacing: UIStyle.standardSpacing) {
                    continuitySection
                    threadSection
                    eventSection
                    deliveryAttemptSection
                    callSessionSection
                }
                .padding(UIStyle.panelPadding)
            }
        }
    }

    private var availableChannels: [String] {
        let threadChannels = state.communicationThreads.map(\.channel)
        let eventChannels = state.communicationEvents.map(\.channel)
        let attemptChannels = state.communicationDeliveryAttempts.map(\.channel)
        let continuityChannels = state.communicationContinuityItems.map(\.channel)
        return Array(
            Set(threadChannels + eventChannels + attemptChannels + continuityChannels)
                .filter { !$0.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty }
        )
        .sorted { lhs, rhs in
            lhs.localizedCaseInsensitiveCompare(rhs) == .orderedAscending
        }
    }

    private var threadFilterOptions: [ThreadFilterOption] {
        var options: [ThreadFilterOption] = [
            ThreadFilterOption(id: FilterID.allThreads, label: "All Threads")
        ]
        options.append(contentsOf: state.communicationThreads.map { item in
            let title = threadDisplayTitle(item)
            let connectorSummary = connectorDisplayLabel(item.connectorID).map { " • \($0)" } ?? ""
            let eventSummary = item.eventCount == 1 ? "1 event" : "\(item.eventCount) events"
            return ThreadFilterOption(
                id: item.id,
                label: "\(title)\(connectorSummary) • \(eventSummary)"
            )
        })
        return options
    }

    private var activeFilterSummaryParts: [String] {
        var summary = CommunicationsFilterContext(
            searchText: searchText,
            channelFilterID: selectedChannelFilterID,
            directionFilterRawValue: selectedDirectionFilter.rawValue,
            threadFilterID: selectedThreadFilterID,
            compactScanModeEnabled: compactScanModeEnabled
        ).activeFilterSummaryParts

        if selectedThreadFilterID != FilterID.allThreads,
            let threadSummaryIndex = summary.firstIndex(where: { $0.hasPrefix("Thread: ") }) {
            if let selectedThread = state.communicationThreads.first(where: { $0.id == selectedThreadFilterID }) {
                let threadTitle = threadDisplayTitle(selectedThread)
                summary[threadSummaryIndex] = "Thread: \(threadTitle)"
            } else {
                summary[threadSummaryIndex] = "Thread: \(shortIdentifier(selectedThreadFilterID, limit: 14))"
            }
        }

        return summary
    }

    private var filteredThreads: [CommunicationThreadItem] {
        let query = normalizedSearchQuery
        return state.communicationThreads.filter { item in
            if selectedChannelFilterID != FilterID.allChannels, item.channel != selectedChannelFilterID {
                return false
            }
            if selectedThreadFilterID != FilterID.allThreads, item.id != selectedThreadFilterID {
                return false
            }
            if !selectedDirectionFilter.matches(item.lastDirection) {
                return false
            }
            guard !query.isEmpty else {
                return true
            }
            return communicationThreadSearchFields(item).contains { field in
                field.localizedCaseInsensitiveContains(query)
            }
        }
    }

    private var filteredEvents: [CommunicationEventItem] {
        let query = normalizedSearchQuery
        return state.communicationEvents.filter { item in
            if selectedChannelFilterID != FilterID.allChannels, item.channel != selectedChannelFilterID {
                return false
            }
            if selectedThreadFilterID != FilterID.allThreads, item.threadID != selectedThreadFilterID {
                return false
            }
            if !selectedDirectionFilter.matches(item.direction) {
                return false
            }
            guard !query.isEmpty else {
                return true
            }
            return communicationEventSearchFields(item).contains { field in
                field.localizedCaseInsensitiveContains(query)
            }
        }
    }

    private var filteredCallSessions: [CommunicationCallSessionItem] {
        let query = normalizedSearchQuery
        return state.communicationCallSessions.filter { item in
            if selectedThreadFilterID != FilterID.allThreads, item.threadID != selectedThreadFilterID {
                return false
            }
            if !selectedDirectionFilter.matches(item.direction) {
                return false
            }
            guard !query.isEmpty else {
                return true
            }
            return communicationCallSessionSearchFields(item).contains { field in
                field.localizedCaseInsensitiveContains(query)
            }
        }
    }

    private var filteredDeliveryAttempts: [CommunicationDeliveryAttemptItem] {
        let query = normalizedSearchQuery
        return state.communicationDeliveryAttempts.filter { item in
            if selectedChannelFilterID != FilterID.allChannels, item.channel != selectedChannelFilterID {
                return false
            }
            if selectedThreadFilterID != FilterID.allThreads, item.threadID != selectedThreadFilterID {
                return false
            }
            guard !query.isEmpty else {
                return true
            }
            return communicationDeliveryAttemptSearchFields(item).contains { field in
                field.localizedCaseInsensitiveContains(query)
            }
        }
    }

    private var filteredContinuityItems: [CommunicationContinuityItem] {
        let query = normalizedSearchQuery
        return state.communicationContinuityItems.filter { item in
            if selectedChannelFilterID != FilterID.allChannels, item.channel != selectedChannelFilterID {
                return false
            }
            if selectedThreadFilterID != FilterID.allThreads, item.threadID != selectedThreadFilterID {
                return false
            }
            guard !query.isEmpty else {
                return true
            }
            return communicationContinuitySearchFields(item).contains { field in
                field.localizedCaseInsensitiveContains(query)
            }
        }
    }

    private var filteredSummaryLabel: String {
        let threadCount = filteredThreads.count
        let eventCount = filteredEvents.count
        let callCount = filteredCallSessions.count
        let attemptCount = filteredDeliveryAttempts.count
        let continuityCount = filteredContinuityItems.count
        return "Continuity \(continuityCount) • Conversations \(threadCount) • Events \(eventCount) • Calls \(callCount) • Attempts \(attemptCount)"
    }

    private var triageSummaryLabel: String? {
        let unreadCount = filteredThreads.filter(isThreadUnread).count
        let newCount = filteredThreads.filter(isThreadNew).count
        let followUpCount = filteredThreads.filter(isThreadFollowUp).count
        guard unreadCount > 0 || newCount > 0 || followUpCount > 0 else {
            return nil
        }
        return "Unread \(unreadCount) • New \(newCount) • Follow Up \(followUpCount)"
    }

    private var groupedFilteredThreads: [ChannelGroup<CommunicationThreadItem>] {
        groupedByLogicalChannel(filteredThreads, channel: \.channel)
    }

    private var groupedFilteredEvents: [ChannelGroup<CommunicationEventItem>] {
        groupedByLogicalChannel(filteredEvents, channel: \.channel)
    }

    private var groupedFilteredAttempts: [ChannelGroup<CommunicationDeliveryAttemptItem>] {
        groupedByLogicalChannel(filteredDeliveryAttempts, channel: \.channel)
    }

    private var groupedFilteredContinuity: [ChannelGroup<CommunicationContinuityItem>] {
        groupedByLogicalChannel(filteredContinuityItems, channel: \.channel)
    }

    private var normalizedSearchQuery: String {
        searchText.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    private func focusSearchFieldForKeyboardTraversal() {
        DispatchQueue.main.async {
            isSearchFieldFocused = true
        }
    }

    private var selectedThreadIDForAttemptQuery: String? {
        selectedThreadFilterID == FilterID.allThreads ? nil : selectedThreadFilterID
    }

    private var showLoadingSkeleton: Bool {
        (state.isCommunicationsLoading || !state.hasLoadedCommunicationsInbox)
            && state.communicationThreads.isEmpty
            && state.communicationEvents.isEmpty
            && state.communicationCallSessions.isEmpty
            && state.communicationContinuityItems.isEmpty
    }

    private var continuitySection: some View {
        GroupBox {
            VStack(alignment: .leading, spacing: 8) {
                sectionHeaderRow(
                    title: "Conversation Continuity",
                    count: filteredContinuityItems.count,
                    hasMore: state.communicationContinuityHasMore
                )
                if state.isCommunicationContinuityLoading && state.communicationContinuityItems.isEmpty {
                    HStack(spacing: 8) {
                        ProgressView()
                            .controlSize(.small)
                        Text("Loading conversation continuity…")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                } else if filteredContinuityItems.isEmpty {
                    Text(state.communicationContinuityStatusMessage ?? "No conversation continuity entries match current filters.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                } else {
                    ForEach(groupedFilteredContinuity) { group in
                        channelGroupHeader(
                            channelID: group.channelID,
                            count: group.items.count
                        )
                        ForEach(group.items) { item in
                            communicationContinuityRow(item)
                            if item.id != group.items.last?.id {
                                Divider()
                            }
                        }
                        if group.channelID != groupedFilteredContinuity.last?.channelID {
                            Divider()
                        }
                    }
                }
            }
            .padding(.vertical, 2)
        }
        .groupBoxStyle(.automatic)
        .accessibilityIdentifier("communications-continuity-section")
    }

    private var threadSection: some View {
        GroupBox {
            VStack(alignment: .leading, spacing: 8) {
                sectionHeaderRow(
                    title: "Conversations by Channel",
                    count: filteredThreads.count,
                    hasMore: state.communicationThreadsHasMore
                )
                if filteredThreads.isEmpty {
                    Text("No conversations match the current filters.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                } else {
                    ForEach(groupedFilteredThreads) { group in
                        channelGroupHeader(
                            channelID: group.channelID,
                            count: group.items.count
                        )
                        ForEach(group.items) { item in
                            communicationThreadRow(item)
                            if item.id != group.items.last?.id {
                                Divider()
                            }
                        }
                        if group.channelID != groupedFilteredThreads.last?.channelID {
                            Divider()
                        }
                    }
                }
            }
            .padding(.vertical, 2)
        }
        .groupBoxStyle(.automatic)
    }

    private var eventSection: some View {
        GroupBox {
            VStack(alignment: .leading, spacing: 8) {
                sectionHeaderRow(
                    title: "Event Timeline by Channel",
                    count: filteredEvents.count,
                    hasMore: state.communicationEventsHasMore
                )
                if filteredEvents.isEmpty {
                    Text("No events match the current filters.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                } else {
                    ForEach(groupedFilteredEvents) { group in
                        channelGroupHeader(
                            channelID: group.channelID,
                            count: group.items.count
                        )
                        ForEach(group.items) { item in
                            communicationEventRow(item)
                            if item.id != group.items.last?.id {
                                Divider()
                            }
                        }
                        if group.channelID != groupedFilteredEvents.last?.channelID {
                            Divider()
                        }
                    }
                }
            }
            .padding(.vertical, 2)
        }
        .groupBoxStyle(.automatic)
    }

    private var deliveryAttemptSection: some View {
        GroupBox {
            VStack(alignment: .leading, spacing: 8) {
                sectionHeaderRow(
                    title: "Delivery Attempts",
                    count: filteredDeliveryAttempts.count,
                    hasMore: state.communicationDeliveryAttemptsHasMore
                )
                if selectedThreadFilterID == FilterID.allThreads {
                    Text("Select a conversation to load delivery-attempt history.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                } else if state.isCommunicationAttemptsLoading && state.communicationDeliveryAttempts.isEmpty {
                    HStack(spacing: 8) {
                        ProgressView()
                            .controlSize(.small)
                        Text("Loading delivery attempts…")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                } else if filteredDeliveryAttempts.isEmpty {
                    Text(state.communicationAttemptsStatusMessage ?? "No delivery attempts match the current filters.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                } else {
                    ForEach(groupedFilteredAttempts) { group in
                        channelGroupHeader(
                            channelID: group.channelID,
                            count: group.items.count
                        )
                        ForEach(group.items) { item in
                            communicationDeliveryAttemptRow(item)
                            if item.id != group.items.last?.id {
                                Divider()
                            }
                        }
                        if group.channelID != groupedFilteredAttempts.last?.channelID {
                            Divider()
                        }
                    }
                }
            }
            .padding(.vertical, 2)
        }
        .groupBoxStyle(.automatic)
    }

    private var callSessionSection: some View {
        GroupBox {
            VStack(alignment: .leading, spacing: 8) {
                sectionHeaderRow(
                    title: "Call Sessions",
                    count: filteredCallSessions.count,
                    hasMore: state.communicationCallSessionsHasMore
                )
                if filteredCallSessions.isEmpty {
                    Text("No call sessions match the current filters.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                } else {
                    ForEach(filteredCallSessions) { item in
                        communicationCallSessionRow(item)
                        if item.id != filteredCallSessions.last?.id {
                            Divider()
                        }
                    }
                }
            }
            .padding(.vertical, 2)
        }
        .groupBoxStyle(.automatic)
    }

    private func sectionHeaderRow(title: String, count: Int, hasMore: Bool) -> some View {
        HStack(alignment: .firstTextBaseline, spacing: 8) {
            Text(title)
                .font(.headline)
            Text("\(count)")
                .font(.caption.weight(.semibold))
                .foregroundStyle(.secondary)
            if hasMore {
                TahoeStatusBadge(
                    text: "More Available",
                    symbolName: "ellipsis.circle",
                    tint: .secondary
                )
            }
            Spacer(minLength: 0)
        }
    }

    private func channelGroupHeader(
        channelID: String,
        count: Int
    ) -> some View {
        HStack(alignment: .firstTextBaseline, spacing: 8) {
            Text(logicalChannelLabel(channelID))
                .font(.subheadline.weight(.semibold))
            Text("\(count)")
                .font(.caption.weight(.semibold))
                .foregroundStyle(.secondary)
            Spacer(minLength: 0)
        }
        .padding(.top, 2)
    }

    private func communicationContinuityRow(_ item: CommunicationContinuityItem) -> some View {
        let detailRows = continuityDetailRows(item)
        return VStack(alignment: .leading, spacing: 6) {
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                VStack(alignment: .leading, spacing: 2) {
                    Text(continuityPrimaryTitle(item))
                        .font(.subheadline.weight(.semibold))
                }
                Spacer(minLength: 0)
                TahoeStatusBadge(
                    text: continuityItemTypeLabel(item.itemType),
                    symbolName: continuityItemTypeSymbol(item.itemType),
                    tint: continuityItemTypeTint(item.itemType)
                )
                TahoeStatusBadge(
                    text: continuityStatusLabel(item.itemStatus),
                    symbolName: continuityStatusSymbol(item.itemStatus),
                    tint: continuityStatusTint(item.itemStatus)
                )
            }

            Text(item.summary)
                .font(.callout)
                .foregroundStyle(.secondary)
                .lineLimit(compactScanModeEnabled ? 2 : 3)

            if !compactScanModeEnabled {
                HStack(spacing: 10) {
                    detailPill("Updated", value: item.createdAtLabel)
                }
            }

            if !detailRows.isEmpty {
                DisclosureGroup(
                    "Details",
                    isExpanded: disclosureBinding(for: item.id, expandedIDs: $expandedContinuityDetailIDs)
                ) {
                    VStack(alignment: .leading, spacing: 4) {
                        ForEach(detailRows) { row in
                            detailRow(label: row.label, value: row.value)
                        }
                    }
                    .padding(.top, 4)
                }
                .font(.caption.weight(.semibold))
            }

            HStack(spacing: 8) {
                if compactScanModeEnabled {
                    Menu("Actions") {
                        Button("Open Chat") {
                            state.openChatForCommunicationContinuity(item)
                        }
                        Button("Open Related Tasks") {
                            state.openTasksForCommunicationContinuity(item)
                        }
                        Button("Open Related Inspect") {
                            state.openInspectForCommunicationContinuity(item)
                        }
                    }
                    .menuStyle(.borderlessButton)
                } else {
                    Button("Open Chat") {
                        state.openChatForCommunicationContinuity(item)
                    }
                    .panelActionStyle(.primary)
                    .controlSize(.small)

                    Button("Open Related Tasks") {
                        state.openTasksForCommunicationContinuity(item)
                    }
                    .panelActionStyle(.secondary)
                    .controlSize(.small)

                    Button("Open Related Inspect") {
                        state.openInspectForCommunicationContinuity(item)
                    }
                    .panelActionStyle(.secondary)
                    .controlSize(.small)
                }
            }
        }
    }

    private func communicationThreadRow(_ item: CommunicationThreadItem) -> some View {
        let threadIsNew = isThreadNew(item)
        let threadIsUnread = isThreadUnread(item)
        let threadNeedsFollowUp = isThreadFollowUp(item)
        let threadHandled = isThreadHandled(item)
        let detailRows = ProgressiveDisclosureDetails.communicationThreadDetails(item)
        return VStack(alignment: .leading, spacing: 6) {
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                VStack(alignment: .leading, spacing: 2) {
                    Text(threadDisplayTitle(item))
                        .font(.subheadline.weight(.semibold))
                }
                Spacer(minLength: 0)
                if threadNeedsFollowUp {
                    TahoeStatusBadge(
                        text: "Follow Up",
                        symbolName: "flag.fill",
                        tint: .orange
                    )
                }
                if threadIsNew {
                    TahoeStatusBadge(
                        text: "New",
                        symbolName: "sparkle",
                        tint: .blue
                    )
                }
                if threadIsUnread {
                    TahoeStatusBadge(
                        text: "Unread",
                        symbolName: "circle.fill",
                        tint: .green
                    )
                } else if threadHandled, !threadNeedsFollowUp {
                    TahoeStatusBadge(
                        text: "Handled",
                        symbolName: "checkmark.circle",
                        tint: .secondary
                    )
                }
                if let connectorSummary = connectorDisplayLabel(item.connectorID) {
                    TahoeStatusBadge(
                        text: connectorSummary,
                        symbolName: "cable.connector",
                        tint: .secondary
                    )
                }
                TahoeStatusBadge(
                    text: item.eventCount == 1 ? "1 event" : "\(item.eventCount) events",
                    symbolName: "text.bubble",
                    tint: .secondary
                )
            }

            if !compactScanModeEnabled, let preview = nonEmpty(item.lastBodyPreview) {
                Text(preview)
                    .font(.callout)
                    .foregroundStyle(.secondary)
                    .lineLimit(2)
            }

            if !compactScanModeEnabled, !item.participantAddresses.isEmpty {
                Text("Participants: \(item.participantAddresses.joined(separator: ", "))")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            if !compactScanModeEnabled {
                HStack(spacing: 10) {
                    if let lastOccurredAtLabel = item.lastOccurredAtLabel {
                        detailPill("Last Event", value: lastOccurredAtLabel)
                    }
                    detailPill("Updated", value: item.updatedAtLabel)
                    if let direction = nonEmpty(item.lastDirection) {
                        detailPill("Direction", value: direction)
                    }
                }
            }

            if !detailRows.isEmpty {
                DisclosureGroup(
                    "Details",
                    isExpanded: disclosureBinding(for: item.id, expandedIDs: $expandedThreadDetailIDs)
                ) {
                    VStack(alignment: .leading, spacing: 4) {
                        ForEach(detailRows) { row in
                            detailRow(label: row.label, value: row.value)
                        }
                    }
                    .padding(.top, 4)
                }
                .font(.caption.weight(.semibold))
            }

            HStack(spacing: 8) {
                Button("Reply") {
                    acknowledgeThread(item.id)
                    presentReplyCompose(from: item)
                }
                .panelActionStyle(.primary)
                .controlSize(.small)
                .disabled(state.isCommunicationSendInFlight)

                Button(threadHandled ? "Reopen" : "Mark Handled") {
                    toggleThreadHandled(item)
                }
                .panelActionStyle(.secondary)
                .controlSize(.small)

                Button(threadNeedsFollowUp ? "Clear Follow Up" : "Follow Up") {
                    toggleThreadFollowUp(item)
                }
                .panelActionStyle(.secondary)
                .controlSize(.small)

                Button("Open Task Draft") {
                    acknowledgeThread(item.id)
                    state.openTaskDraftForCommunicationThread(item)
                }
                .panelActionStyle(.secondary)
                .controlSize(.small)

                if compactScanModeEnabled {
                    Menu("More") {
                        Button("Start Call") {
                            acknowledgeThread(item.id)
                            presentStartCallCompose(from: item)
                        }
                        .disabled(state.isCommunicationSendInFlight)

                        Button("Filter Events") {
                            acknowledgeThread(item.id)
                            selectedThreadFilterID = item.id
                        }

                        Button("Open Related Inspect") {
                            acknowledgeThread(item.id)
                            state.openInspectForCommunicationThread(item)
                        }

                        Button("Open Related Channels") {
                            state.openChannelsForCommunicationChannel(item.channel)
                        }
                    }
                    .menuStyle(.borderlessButton)
                } else {
                    Button("Start Call") {
                        acknowledgeThread(item.id)
                        presentStartCallCompose(from: item)
                    }
                    .panelActionStyle(.secondary)
                    .controlSize(.small)
                    .disabled(state.isCommunicationSendInFlight)

                    Button("Filter Events") {
                        acknowledgeThread(item.id)
                        selectedThreadFilterID = item.id
                    }
                    .panelActionStyle(.secondary)
                    .controlSize(.small)

                    Button("Open Related Inspect") {
                        acknowledgeThread(item.id)
                        state.openInspectForCommunicationThread(item)
                    }
                    .panelActionStyle(.secondary)
                    .controlSize(.small)

                    Button("Open Related Channels") {
                        state.openChannelsForCommunicationChannel(item.channel)
                    }
                    .panelActionStyle(.secondary)
                    .controlSize(.small)
                }
            }
        }
    }

    private func communicationEventRow(_ item: CommunicationEventItem) -> some View {
        let parentThread = state.communicationThreads.first(where: { $0.id == item.threadID })
        let threadIsNew = parentThread.map(isThreadNew) ?? false
        let threadIsUnread = parentThread.map(isThreadUnread) ?? false
        let threadNeedsFollowUp = parentThread.map(isThreadFollowUp) ?? false
        let threadHandled = parentThread.map(isThreadHandled) ?? false
        let detailRows = ProgressiveDisclosureDetails.communicationEventDetails(item)
        return VStack(alignment: .leading, spacing: 6) {
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                VStack(alignment: .leading, spacing: 2) {
                    Text(item.eventType)
                        .font(.subheadline.weight(.semibold))
                }
                Spacer(minLength: 0)
                TahoeStatusBadge(
                    text: item.direction,
                    symbolName: directionSymbol(item.direction),
                    tint: directionTint(item.direction)
                )
                if let connectorSummary = connectorDisplayLabel(item.connectorID) {
                    TahoeStatusBadge(
                        text: connectorSummary,
                        symbolName: "cable.connector",
                        tint: .secondary
                    )
                }
                if item.assistantEmitted {
                    TahoeStatusBadge(
                        text: "Assistant",
                        symbolName: "sparkles",
                        tint: .purple
                    )
                }
                if threadNeedsFollowUp {
                    TahoeStatusBadge(
                        text: "Follow Up",
                        symbolName: "flag.fill",
                        tint: .orange
                    )
                }
                if threadIsNew {
                    TahoeStatusBadge(
                        text: "New",
                        symbolName: "sparkle",
                        tint: .blue
                    )
                }
                if threadIsUnread {
                    TahoeStatusBadge(
                        text: "Unread",
                        symbolName: "circle.fill",
                        tint: .green
                    )
                } else if threadHandled, !threadNeedsFollowUp {
                    TahoeStatusBadge(
                        text: "Handled",
                        symbolName: "checkmark.circle",
                        tint: .secondary
                    )
                }
            }

            if !compactScanModeEnabled, let bodyText = nonEmpty(item.bodyText) {
                Text(bodyText)
                    .font(.callout)
                    .lineLimit(3)
            }

            if !compactScanModeEnabled, !item.addresses.isEmpty {
                Text(addressSummary(item.addresses))
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            if !compactScanModeEnabled {
                HStack(spacing: 10) {
                    detailPill("Occurred", value: item.occurredAtLabel)
                    detailPill("Created", value: item.createdAtLabel)
                }
            }

            if !detailRows.isEmpty {
                DisclosureGroup(
                    "Details",
                    isExpanded: disclosureBinding(for: item.id, expandedIDs: $expandedEventDetailIDs)
                ) {
                    VStack(alignment: .leading, spacing: 4) {
                        ForEach(detailRows) { row in
                            detailRow(label: row.label, value: row.value)
                        }
                    }
                    .padding(.top, 4)
                }
                .font(.caption.weight(.semibold))
            }

            HStack(spacing: 8) {
                Button("Reply") {
                    acknowledgeThread(item.threadID)
                    presentReplyCompose(from: item)
                }
                .panelActionStyle(.primary)
                .controlSize(.small)
                .disabled(state.isCommunicationSendInFlight)

                if let thread = parentThread {
                    Button(threadHandled ? "Reopen" : "Mark Handled") {
                        toggleThreadHandled(thread)
                    }
                    .panelActionStyle(.secondary)
                    .controlSize(.small)

                    Button(threadNeedsFollowUp ? "Clear Follow Up" : "Follow Up") {
                        toggleThreadFollowUp(thread)
                    }
                    .panelActionStyle(.secondary)
                    .controlSize(.small)

                    Button("Open Task Draft") {
                        acknowledgeThread(thread.id)
                        state.openTaskDraftForCommunicationThread(thread)
                    }
                    .panelActionStyle(.secondary)
                    .controlSize(.small)
                }

                Button("Filter Thread") {
                    acknowledgeThread(item.threadID)
                    selectedThreadFilterID = item.threadID
                }
                .panelActionStyle(.secondary)
                .controlSize(.small)

                Button("Open Related Inspect") {
                    state.openInspectForCommunicationEvent(item)
                }
                .panelActionStyle(.secondary)
                .controlSize(.small)
            }
        }
    }

    private func communicationCallSessionRow(_ item: CommunicationCallSessionItem) -> some View {
        let detailRows = ProgressiveDisclosureDetails.communicationCallSessionDetails(item)
        return VStack(alignment: .leading, spacing: 6) {
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                VStack(alignment: .leading, spacing: 2) {
                    Text(item.provider)
                        .font(.subheadline.weight(.semibold))
                }
                Spacer(minLength: 0)
                TahoeStatusBadge(
                    text: item.status,
                    symbolName: callSessionStatusSymbol(item.status),
                    tint: callSessionStatusTint(item.status)
                )
                if let connectorSummary = connectorDisplayLabel(item.connectorID) {
                    TahoeStatusBadge(
                        text: connectorSummary,
                        symbolName: "cable.connector",
                        tint: .secondary
                    )
                }
                TahoeStatusBadge(
                    text: item.direction,
                    symbolName: directionSymbol(item.direction),
                    tint: directionTint(item.direction)
                )
            }

            if compactScanModeEnabled {
                HStack(spacing: 10) {
                    if let fromAddress = nonEmpty(item.fromAddress) {
                        detailPill("From", value: fromAddress)
                    }
                    if let toAddress = nonEmpty(item.toAddress) {
                        detailPill("To", value: toAddress)
                    }
                    detailPill("Updated", value: item.updatedAtLabel)
                }
            } else {
                VStack(alignment: .leading, spacing: 3) {
                    if let fromAddress = nonEmpty(item.fromAddress) {
                        detailRow(label: "From", value: fromAddress)
                    }
                    if let toAddress = nonEmpty(item.toAddress) {
                        detailRow(label: "To", value: toAddress)
                    }
                    if let startedAt = item.startedAtLabel {
                        detailRow(label: "Started", value: startedAt)
                    }
                    if let endedAt = item.endedAtLabel {
                        detailRow(label: "Ended", value: endedAt)
                    }
                    detailRow(label: "Updated", value: item.updatedAtLabel)
                }
            }

            if !detailRows.isEmpty {
                DisclosureGroup(
                    "Details",
                    isExpanded: disclosureBinding(for: item.id, expandedIDs: $expandedCallSessionDetailIDs)
                ) {
                    VStack(alignment: .leading, spacing: 4) {
                        ForEach(detailRows) { row in
                            detailRow(label: row.label, value: row.value)
                        }
                    }
                    .padding(.top, 4)
                }
                .font(.caption.weight(.semibold))
            }

            HStack(spacing: 8) {
                Button("Start Call") {
                    presentStartCallCompose(from: item)
                }
                .panelActionStyle(.primary)
                .controlSize(.small)
                .disabled(state.isCommunicationSendInFlight)

                if compactScanModeEnabled {
                    Menu("More") {
                        if let threadID = nonEmpty(item.threadID) {
                            Button("Filter Thread") {
                                selectedThreadFilterID = threadID
                            }
                        }
                        Button("Open Related Inspect") {
                            state.openInspectForCommunicationCallSession(item)
                        }
                    }
                    .menuStyle(.borderlessButton)
                } else {
                    if let threadID = nonEmpty(item.threadID) {
                        Button("Filter Thread") {
                            selectedThreadFilterID = threadID
                        }
                        .panelActionStyle(.secondary)
                        .controlSize(.small)
                    }

                    Button("Open Related Inspect") {
                        state.openInspectForCommunicationCallSession(item)
                    }
                    .panelActionStyle(.secondary)
                    .controlSize(.small)
                }
            }
        }
    }

    private func communicationDeliveryAttemptRow(_ item: CommunicationDeliveryAttemptItem) -> some View {
        let detailRows = ProgressiveDisclosureDetails.communicationDeliveryAttemptDetails(item)
        return VStack(alignment: .leading, spacing: 6) {
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                VStack(alignment: .leading, spacing: 2) {
                    Text("Delivery Attempt")
                        .font(.subheadline.weight(.semibold))
                }
                Spacer(minLength: 0)
                TahoeStatusBadge(
                    text: item.status,
                    symbolName: deliveryAttemptStatusSymbol(item.status),
                    tint: deliveryAttemptStatusTint(item.status)
                )
                TahoeStatusBadge(
                    text: item.routePhase,
                    symbolName: "arrow.triangle.branch",
                    tint: deliveryAttemptRoutePhaseTint(item.routePhase)
                )
            }

            if compactScanModeEnabled {
                HStack(spacing: 10) {
                    detailPill("Attempted", value: item.attemptedAtLabel)
                    detailPill("Destination", value: item.destinationEndpoint)
                    if let error = nonEmpty(item.error) {
                        detailPill("Error", value: error)
                    }
                }
            } else {
                VStack(alignment: .leading, spacing: 3) {
                    detailRow(label: "Attempted", value: item.attemptedAtLabel)
                    detailRow(label: "Destination", value: item.destinationEndpoint)
                    if let fallbackFrom = nonEmpty(item.fallbackFromChannel) {
                        detailRow(label: "Fallback From", value: fallbackFrom)
                    }
                    if let error = nonEmpty(item.error) {
                        detailRow(label: "Error", value: error)
                    }
                }
            }

            if !detailRows.isEmpty {
                DisclosureGroup(
                    "Details",
                    isExpanded: disclosureBinding(for: item.id, expandedIDs: $expandedDeliveryAttemptDetailIDs)
                ) {
                    VStack(alignment: .leading, spacing: 4) {
                        ForEach(detailRows) { row in
                            detailRow(label: row.label, value: row.value)
                        }
                    }
                    .padding(.top, 4)
                }
                .font(.caption.weight(.semibold))
            }

            HStack(spacing: 8) {
                if compactScanModeEnabled {
                    Menu("Actions") {
                        Button("Open Related Tasks") {
                            state.openTasksForCommunicationAttempt(item)
                        }
                        Button("Open Related Inspect") {
                            state.openInspectForCommunicationAttempt(item)
                        }
                        Button("Open Related Channels") {
                            state.openChannelsForCommunicationChannel(item.channel)
                        }
                    }
                    .menuStyle(.borderlessButton)
                } else {
                    Button("Open Related Tasks") {
                        state.openTasksForCommunicationAttempt(item)
                    }
                    .panelActionStyle(.secondary)
                    .controlSize(.small)

                    Button("Open Related Inspect") {
                        state.openInspectForCommunicationAttempt(item)
                    }
                    .panelActionStyle(.secondary)
                    .controlSize(.small)

                    Button("Open Related Channels") {
                        state.openChannelsForCommunicationChannel(item.channel)
                    }
                    .panelActionStyle(.secondary)
                    .controlSize(.small)
                }
            }
        }
    }

    private func latestCommunicationSendCard(_ receipt: CommunicationSendReceiptItem) -> some View {
        let detailRows = ProgressiveDisclosureDetails.communicationSendReceiptDetails(receipt)
        return GroupBox {
            VStack(alignment: .leading, spacing: 8) {
                HStack(alignment: .firstTextBaseline, spacing: 8) {
                    Text("Latest Outbound Send")
                        .font(.headline)
                    Spacer(minLength: 0)
                    TahoeStatusBadge(
                        text: receipt.success ? "Delivered" : "Failed",
                        symbolName: receipt.success ? "checkmark.circle.fill" : "xmark.circle.fill",
                        tint: receipt.success ? .green : .orange
                    )
                }
                if let statusMessage = nonEmpty(state.communicationSendStatusMessage) {
                    Text(statusMessage)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                HStack(spacing: 10) {
                    detailPill("Source", value: logicalChannelLabel(receipt.sourceChannel))
                    if let connector = connectorDisplayLabel(receipt.connectorID) {
                        detailPill("Connector", value: connector)
                    }
                }
                if !detailRows.isEmpty {
                    DisclosureGroup("Details", isExpanded: $latestSendDetailsExpanded) {
                        VStack(alignment: .leading, spacing: 4) {
                            ForEach(detailRows) { row in
                                detailRow(label: row.label, value: row.value)
                            }
                        }
                        .padding(.top, 4)
                    }
                    .font(.caption.weight(.semibold))
                }
                if let destination = nonEmpty(receipt.destination) {
                    Text("Destination: \(destination)")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                } else if nonEmpty(receipt.threadID) != nil {
                    Text("Destination resolved from thread context.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                Text("Sent \(receipt.sentAt.formatted(date: .abbreviated, time: .shortened))")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }
            .padding(.vertical, 2)
        }
        .groupBoxStyle(.automatic)
    }

    @ViewBuilder
    private func composeSheet(_ context: ComposeContext) -> some View {
        NavigationStack {
            Form {
                Section("Quick Send") {
                    Text(context.flow.helperCopy)
                        .font(.caption)
                        .foregroundStyle(.secondary)

                    TextField("Recipient", text: $composeDestinationDraft)
                        .textFieldStyle(.roundedBorder)

                    TextEditor(text: $composeMessageDraft)
                        .frame(minHeight: 140)
                        .overlay(
                            RoundedRectangle(cornerRadius: 8, style: .continuous)
                                .stroke(Color(nsColor: .separatorColor), lineWidth: 1)
                        )

                    if let contextHint = composeContextHintText(for: context.flow) {
                        Text(contextHint)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }

                    if let statusMessage = nonEmpty(state.communicationSendStatusMessage) {
                        Text(statusMessage)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                }

                Section {
                    DisclosureGroup("Advanced", isExpanded: $composeAdvancedExpanded) {
                        Picker("Source Channel", selection: $composeSourceChannelDraft) {
                            Text("App").tag("app")
                            Text("Message").tag("message")
                            Text("Voice").tag("voice")
                        }
                        .pickerStyle(.menu)

                        TextField("Conversation ID (optional)", text: $composeThreadIDDraft)
                            .textFieldStyle(.roundedBorder)

                        TextField("Connector Hint (optional)", text: $composeConnectorHintDraft)
                            .textFieldStyle(.roundedBorder)
                    }
                    .font(.subheadline.weight(.semibold))
                }
            }
            .navigationTitle("Quick Send")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") {
                        composeAdvancedExpanded = false
                        activeComposeContext = nil
                    }
                    .disabled(state.isCommunicationSendInFlight)
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button(state.isCommunicationSendInFlight ? "Sending…" : context.flow.submitActionTitle) {
                        submitComposeDraft()
                    }
                    .disabled(!composeCanSend || state.isCommunicationSendInFlight)
                }
            }
        }
    }

    private var composeCanSend: Bool {
        let normalizedSource = composeSourceChannelDraft
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .lowercased()
        let hasMessage = nonEmpty(composeMessageDraft) != nil
        let hasThreadContext = nonEmpty(composeThreadIDDraft) != nil
        let hasDestination = nonEmpty(composeDestinationDraft) != nil
        return ["app", "message", "voice"].contains(normalizedSource) && hasMessage && (hasThreadContext || hasDestination)
    }

    private func composeContextHintText(for flow: ComposeContext.Flow) -> String? {
        if nonEmpty(composeThreadIDDraft) != nil {
            return "Conversation context is attached. Recipient can stay empty."
        }
        if nonEmpty(composeConnectorHintDraft) != nil {
            return "Connector hint is set in Advanced."
        }
        if flow == .startCall {
            return "Quick Send uses voice routing for this call."
        }
        return nil
    }

    private func submitComposeDraft() {
        state.sendCommunication(
            sourceChannel: composeSourceChannelDraft,
            destination: nonEmpty(composeDestinationDraft),
            message: composeMessageDraft,
            threadID: nonEmpty(composeThreadIDDraft),
            connectorID: nonEmpty(composeConnectorHintDraft)
        )
    }

    private func presentReplyCompose(from thread: CommunicationThreadItem) {
        presentCompose(
            flow: .reply,
            sourceChannel: thread.channel,
            threadID: thread.id,
            connectorID: thread.connectorID,
            destination: suggestedThreadDestination(thread),
            message: ""
        )
    }

    private func presentReplyCompose(from event: CommunicationEventItem) {
        let thread = state.communicationThreads.first(where: { $0.id == event.threadID })
        presentCompose(
            flow: .reply,
            sourceChannel: event.channel,
            threadID: event.threadID,
            connectorID: event.connectorID ?? thread?.connectorID,
            destination: thread.flatMap { suggestedThreadDestination($0) },
            message: ""
        )
    }

    private func presentStartCallCompose(from thread: CommunicationThreadItem) {
        let connectorHint = connectorHintForVoiceFlow(from: thread.connectorID)
        presentCompose(
            flow: .startCall,
            sourceChannel: "voice",
            threadID: thread.id,
            connectorID: connectorHint,
            destination: suggestedThreadDestination(thread),
            message: "Starting a call from Personal Agent."
        )
    }

    private func presentStartCallCompose(from callSession: CommunicationCallSessionItem) {
        let destination = nonEmpty(callSession.toAddress) ?? nonEmpty(callSession.fromAddress)
        let connectorHint = connectorHintForVoiceFlow(from: callSession.connectorID)
        presentCompose(
            flow: .startCall,
            sourceChannel: "voice",
            threadID: nonEmpty(callSession.threadID),
            connectorID: connectorHint,
            destination: destination,
            message: "Starting a follow-up call from Personal Agent."
        )
    }

    private func presentCompose(
        flow: ComposeContext.Flow,
        sourceChannel: String,
        threadID: String?,
        connectorID: String?,
        destination: String?,
        message: String
    ) {
        composeSourceChannelDraft = normalizedComposeSourceChannel(sourceChannel)
        composeThreadIDDraft = threadID ?? ""
        composeConnectorHintDraft = connectorID ?? ""
        composeDestinationDraft = destination ?? ""
        composeMessageDraft = message
        composeReceiptBaselineID = state.latestCommunicationSendReceipt?.id
        composeAdvancedExpanded = false
        activeComposeContext = ComposeContext(flow: flow)
    }

    private func suggestedThreadDestination(_ thread: CommunicationThreadItem) -> String? {
        let participants = thread.participantAddresses
            .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
            .filter { !$0.isEmpty }
        guard participants.count == 1 else {
            return nil
        }
        return participants.first
    }

    private func normalizedComposeSourceChannel(_ value: String) -> String {
        switch value
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .lowercased() {
        case "app", "app_chat":
            return "app"
        case "voice", "twilio_voice":
            return "voice"
        case "message", "sms", "imessage", "imessage_sms", "twilio_sms":
            return "message"
        default:
            return "message"
        }
    }

    private func connectorHintForVoiceFlow(from connectorID: String?) -> String? {
        guard let connectorID = nonEmpty(connectorID) else {
            return "twilio"
        }
        let normalized = connectorID.lowercased()
        if normalized == "twilio" || normalized == "twilio_voice" {
            return "twilio"
        }
        return "twilio"
    }

    private func clearFilters() {
        let reset = state.resetCommunicationsFilterContext()
        applyFilterContext(reset)
    }

    private func applyPersistedFilterContext() {
        applyFilterContext(state.communicationsFilterContext())
    }

    private func applyFilterContext(_ context: CommunicationsFilterContext) {
        searchText = context.searchText
        selectedChannelFilterID = context.channelFilterID
        selectedDirectionFilter = CommunicationDirectionFilter(rawValue: context.directionFilterRawValue) ?? .all
        selectedThreadFilterID = context.threadFilterID
        compactScanModeEnabled = context.compactScanModeEnabled
        normalizeThreadSelection()
    }

    private func persistFilterContext() {
        state.updateCommunicationsFilterContext(
            CommunicationsFilterContext(
                searchText: searchText,
                channelFilterID: selectedChannelFilterID,
                directionFilterRawValue: selectedDirectionFilter.rawValue,
                threadFilterID: selectedThreadFilterID,
                compactScanModeEnabled: compactScanModeEnabled
            )
        )
    }

    private func normalizeThreadSelection() {
        guard selectedThreadFilterID != FilterID.allThreads else {
            return
        }
        let knownThreadIDs = Set(state.communicationThreads.map(\.id))
        if !knownThreadIDs.contains(selectedThreadFilterID) {
            selectedThreadFilterID = FilterID.allThreads
        }
    }

    private func applyPersistedTriageContext() {
        let context = state.communicationsTriageContext()
        handledThreadIDs = triageSet(context.handledThreadIDs)
        followUpThreadIDs = triageSet(context.followUpThreadIDs)
        seenThreadIDs = triageSet(context.seenThreadIDs)
        newThreadIDs = []
    }

    private func persistTriageContext() {
        state.updateCommunicationsTriageContext(
            CommunicationsTriageContext(
                handledThreadIDs: Array(handledThreadIDs).sorted(),
                followUpThreadIDs: Array(followUpThreadIDs).sorted(),
                seenThreadIDs: Array(seenThreadIDs).sorted()
            )
        )
    }

    private func applyPersistedComposeDraftContext() {
        guard let context = state.communicationsComposeDraftContext() else {
            activeComposeContext = nil
            resetComposeDraftFields()
            return
        }
        composeSourceChannelDraft = context.sourceChannel
        composeThreadIDDraft = context.threadID
        composeConnectorHintDraft = context.connectorID
        composeDestinationDraft = context.destination
        composeMessageDraft = context.message
        if context.isPresented,
           let flow = ComposeContext.Flow(persistenceID: context.flowID) {
            activeComposeContext = ComposeContext(flow: flow)
        } else {
            activeComposeContext = nil
        }
        composeAdvancedExpanded = false
        invalidateComposeThreadContextIfMissing()
    }

    private func persistComposeDraftContext() {
        let hasDraftContent = !composeSourceChannelDraft.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ||
            !composeThreadIDDraft.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ||
            !composeConnectorHintDraft.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ||
            !composeDestinationDraft.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ||
            !composeMessageDraft.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
        let isPresented = activeComposeContext != nil

        guard hasDraftContent || isPresented else {
            state.updateCommunicationsComposeDraftContext(nil)
            return
        }

        let flowID = activeComposeContext?.flow.persistenceID ?? ComposeContext.Flow.newMessage.persistenceID
        state.updateCommunicationsComposeDraftContext(
            CommunicationsComposeDraftContext(
                isPresented: isPresented,
                flowID: flowID,
                sourceChannel: composeSourceChannelDraft,
                threadID: composeThreadIDDraft,
                connectorID: composeConnectorHintDraft,
                destination: composeDestinationDraft,
                message: composeMessageDraft
            )
        )
    }

    private func invalidateComposeThreadContextIfMissing() {
        guard let threadID = nonEmpty(composeThreadIDDraft) else {
            return
        }
        let knownThreadIDs = Set(state.communicationThreads.map(\.id))
        guard !knownThreadIDs.isEmpty else {
            return
        }
        if !knownThreadIDs.contains(threadID) {
            composeThreadIDDraft = ""
        }
    }

    private func resetComposeDraftFields() {
        composeSourceChannelDraft = "message"
        composeThreadIDDraft = ""
        composeConnectorHintDraft = ""
        composeDestinationDraft = ""
        composeMessageDraft = ""
        composeAdvancedExpanded = false
    }

    private func reconcileThreadTriageState(bootstrapSeenState: Bool = false) {
        let currentThreadIDs = triageSet(state.communicationThreads.map(\.id))
        guard !currentThreadIDs.isEmpty else {
            handledThreadIDs = []
            followUpThreadIDs = []
            seenThreadIDs = []
            newThreadIDs = []
            persistTriageContext()
            return
        }

        let priorHandled = handledThreadIDs
        let priorFollowUp = followUpThreadIDs
        let priorSeen = seenThreadIDs

        handledThreadIDs.formIntersection(currentThreadIDs)
        followUpThreadIDs.formIntersection(currentThreadIDs)
        seenThreadIDs.formIntersection(currentThreadIDs)
        newThreadIDs.formIntersection(currentThreadIDs)

        if seenThreadIDs.isEmpty || bootstrapSeenState {
            seenThreadIDs = currentThreadIDs
            newThreadIDs = []
        } else {
            let newlyArrived = currentThreadIDs.subtracting(seenThreadIDs).subtracting(handledThreadIDs)
            if !newlyArrived.isEmpty {
                newThreadIDs.formUnion(newlyArrived)
                seenThreadIDs.formUnion(newlyArrived)
            }
        }

        if handledThreadIDs != priorHandled || followUpThreadIDs != priorFollowUp || seenThreadIDs != priorSeen {
            persistTriageContext()
        }
    }

    private func toggleThreadHandled(_ thread: CommunicationThreadItem) {
        let threadID = thread.id.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !threadID.isEmpty else {
            return
        }
        if handledThreadIDs.contains(threadID) {
            handledThreadIDs.remove(threadID)
        } else {
            handledThreadIDs.insert(threadID)
            followUpThreadIDs.remove(threadID)
            newThreadIDs.remove(threadID)
        }
        seenThreadIDs.insert(threadID)
        persistTriageContext()
    }

    private func toggleThreadFollowUp(_ thread: CommunicationThreadItem) {
        let threadID = thread.id.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !threadID.isEmpty else {
            return
        }
        if followUpThreadIDs.contains(threadID) {
            followUpThreadIDs.remove(threadID)
        } else {
            followUpThreadIDs.insert(threadID)
            handledThreadIDs.remove(threadID)
            newThreadIDs.remove(threadID)
        }
        seenThreadIDs.insert(threadID)
        persistTriageContext()
    }

    private func acknowledgeThread(_ threadID: String) {
        let normalizedThreadID = threadID.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !normalizedThreadID.isEmpty else {
            return
        }
        newThreadIDs.remove(normalizedThreadID)
        seenThreadIDs.insert(normalizedThreadID)
        persistTriageContext()
    }

    private func triageSet(_ values: [String]) -> Set<String> {
        Set(
            values
                .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
                .filter { !$0.isEmpty }
        )
    }

    private func isThreadHandled(_ thread: CommunicationThreadItem) -> Bool {
        handledThreadIDs.contains(thread.id.trimmingCharacters(in: .whitespacesAndNewlines))
    }

    private func isThreadFollowUp(_ thread: CommunicationThreadItem) -> Bool {
        followUpThreadIDs.contains(thread.id.trimmingCharacters(in: .whitespacesAndNewlines))
    }

    private func isThreadNew(_ thread: CommunicationThreadItem) -> Bool {
        newThreadIDs.contains(thread.id.trimmingCharacters(in: .whitespacesAndNewlines))
    }

    private func isThreadUnread(_ thread: CommunicationThreadItem) -> Bool {
        guard !isThreadHandled(thread) else {
            return false
        }
        let direction = thread.lastDirection?
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .lowercased()
        return direction == "inbound"
    }

    private func resetWorkspaceContinuityContext() {
        state.resetWorkspaceContinuityContext()
        state.updateCommunicationsTriageContext(CommunicationsTriageContext())
        handledThreadIDs = []
        followUpThreadIDs = []
        seenThreadIDs = []
        newThreadIDs = []
        expandedThreadDetailIDs = []
        expandedEventDetailIDs = []
        expandedCallSessionDetailIDs = []
        expandedDeliveryAttemptDetailIDs = []
        expandedContinuityDetailIDs = []
        latestSendDetailsExpanded = false
        activeComposeContext = nil
        resetComposeDraftFields()
        clearFilters()
        persistComposeDraftContext()
    }

    private func reconcileDetailDisclosureState() {
        expandedThreadDetailIDs.formIntersection(Set(state.communicationThreads.map(\.id)))
        expandedEventDetailIDs.formIntersection(Set(state.communicationEvents.map(\.id)))
        expandedCallSessionDetailIDs.formIntersection(Set(state.communicationCallSessions.map(\.id)))
        expandedDeliveryAttemptDetailIDs.formIntersection(Set(state.communicationDeliveryAttempts.map(\.id)))
        expandedContinuityDetailIDs.formIntersection(Set(state.communicationContinuityItems.map(\.id)))
        if state.latestCommunicationSendReceipt == nil {
            latestSendDetailsExpanded = false
        }
    }

    private func threadDisplayTitle(_ thread: CommunicationThreadItem) -> String {
        if let title = nonEmpty(thread.title) {
            return title
        }
        if let firstParticipant = thread.participantAddresses
            .map({ $0.trimmingCharacters(in: .whitespacesAndNewlines) })
            .first(where: { !$0.isEmpty }) {
            return firstParticipant
        }
        return "Conversation"
    }

    private func disclosureBinding(
        for id: String,
        expandedIDs: Binding<Set<String>>
    ) -> Binding<Bool> {
        Binding(
            get: { expandedIDs.wrappedValue.contains(id) },
            set: { isExpanded in
                if isExpanded {
                    expandedIDs.wrappedValue.insert(id)
                } else {
                    expandedIDs.wrappedValue.remove(id)
                }
            }
        )
    }

    private func communicationThreadSearchFields(_ item: CommunicationThreadItem) -> [String] {
        [
            item.id,
            item.channel,
            logicalChannelLabel(item.channel),
            item.connectorID,
            connectorDisplayLabel(item.connectorID),
            item.title,
            item.externalRef,
            item.lastEventID,
            item.lastEventType,
            item.lastDirection,
            item.lastBodyPreview,
            item.participantAddresses.joined(separator: " ")
        ].compactMap(nonEmpty)
    }

    private func communicationEventSearchFields(_ item: CommunicationEventItem) -> [String] {
        let addressValues = item.addresses.flatMap { address in
            [address.role, address.value, address.display].compactMap(nonEmpty)
        }
        return [
            item.id,
            item.threadID,
            item.channel,
            logicalChannelLabel(item.channel),
            item.connectorID,
            connectorDisplayLabel(item.connectorID),
            item.eventType,
            item.direction,
            item.bodyText
        ]
        .compactMap(nonEmpty) + addressValues
    }

    private func communicationCallSessionSearchFields(_ item: CommunicationCallSessionItem) -> [String] {
        [
            item.id,
            item.provider,
            item.connectorID,
            connectorDisplayLabel(item.connectorID),
            item.providerCallID,
            item.threadID,
            item.direction,
            item.status,
            item.fromAddress,
            item.toAddress
        ].compactMap(nonEmpty)
    }

    private func communicationDeliveryAttemptSearchFields(_ item: CommunicationDeliveryAttemptItem) -> [String] {
        [
            item.id,
            item.operationID,
            item.taskID,
            item.runID,
            item.stepID,
            item.eventID,
            item.threadID,
            item.channel,
            logicalChannelLabel(item.channel),
            item.routePhase,
            item.destinationEndpoint,
            item.idempotencyKey,
            item.fallbackFromChannel,
            item.status,
            item.providerReceipt,
            item.error
        ].compactMap(nonEmpty)
    }

    private func communicationContinuitySearchFields(_ item: CommunicationContinuityItem) -> [String] {
        [
            item.id,
            item.turnID,
            item.channel,
            logicalChannelLabel(item.channel),
            item.connectorID,
            connectorDisplayLabel(item.connectorID),
            item.threadID,
            item.correlationID,
            item.taskClass,
            item.itemType,
            item.itemStatus,
            item.summary,
            item.taskID,
            item.runID,
            item.taskState,
            item.runState,
            item.responseShapingChannel,
            item.responseShapingProfile,
            item.personaPolicySource
        ]
        .compactMap(nonEmpty)
    }

    private func groupedByLogicalChannel<Item>(
        _ items: [Item],
        channel: KeyPath<Item, String>
    ) -> [ChannelGroup<Item>] {
        let grouped = Dictionary(grouping: items) { item in
            item[keyPath: channel]
        }
        let orderedChannels = grouped.keys.sorted(by: communicationChannelSortOrder)
        return orderedChannels.map { channelID in
            ChannelGroup(
                channelID: channelID,
                items: items.filter { $0[keyPath: channel] == channelID }
            )
        }
    }

    private func communicationChannelSortOrder(_ lhs: String, _ rhs: String) -> Bool {
        let lhsRank = communicationChannelPriority(lhs)
        let rhsRank = communicationChannelPriority(rhs)
        if lhsRank != rhsRank {
            return lhsRank < rhsRank
        }
        return lhs.localizedCaseInsensitiveCompare(rhs) == .orderedAscending
    }

    private func communicationChannelPriority(_ channelID: String) -> Int {
        switch channelID.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
        case "app":
            return 0
        case "message":
            return 1
        case "voice":
            return 2
        default:
            return 3
        }
    }

    private func logicalChannelLabel(_ channelID: String) -> String {
        switch channelID.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
        case "app":
            return "App"
        case "message":
            return "Message"
        case "voice":
            return "Voice"
        case let value where !value.isEmpty:
            return value
                .replacingOccurrences(of: "_", with: " ")
                .split(separator: " ")
                .map { $0.capitalized }
                .joined(separator: " ")
        default:
            return "Unknown"
        }
    }

    private func connectorDisplayLabel(_ connectorID: String?) -> String? {
        guard let connectorID else {
            return nil
        }
        let normalized = connectorID
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .lowercased()
        guard !normalized.isEmpty else {
            return nil
        }
        switch normalized {
        case "twilio":
            return "Twilio"
        case "imessage":
            return "iMessage"
        case "builtin.app":
            return "App"
        default:
            return normalized
                .replacingOccurrences(of: "_", with: " ")
                .replacingOccurrences(of: ".", with: " ")
                .split(separator: " ")
                .map { $0.capitalized }
                .joined(separator: " ")
        }
    }

    private func continuityPrimaryTitle(_ item: CommunicationContinuityItem) -> String {
        if let threadID = nonEmpty(item.threadID) {
            return "Thread \(shortIdentifier(threadID, limit: 16))"
        }
        if let correlationID = nonEmpty(item.correlationID) {
            return "Correlation \(shortIdentifier(correlationID, limit: 16))"
        }
        return "Turn \(shortIdentifier(item.turnID, limit: 16))"
    }

    private func continuityDetailRows(_ item: CommunicationContinuityItem) -> [ContinuityDetailRow] {
        var rows: [ContinuityDetailRow] = []
        func append(_ label: String, _ value: String?) {
            guard let value = nonEmpty(value) else {
                return
            }
            rows.append(ContinuityDetailRow(id: "\(label):\(value)", label: label, value: value))
        }

        append("Turn ID", item.turnID)
        append("Correlation", item.correlationID)
        append("Thread ID", item.threadID)
        append("Connector", connectorDisplayLabel(item.connectorID))
        append("Task Class", item.taskClass)
        append("Shaping Channel", item.responseShapingChannel.map(logicalChannelLabel))
        append("Shaping Profile", continuityResponseShapingProfileLabel(item.responseShapingProfile))
        append("Persona Source", continuityPersonaSourceLabel(item.personaPolicySource))
        append("Shaping Guardrails", item.responseShapingGuardrailCount.map(String.init))
        append("Shaping Instructions", item.responseShapingInstructionCount.map(String.init))
        append("Task ID", item.taskID)
        append("Run ID", item.runID)
        append("Task State", item.taskState)
        append("Run State", item.runState)
        return rows
    }

    private func continuityResponseShapingProfileLabel(_ raw: String?) -> String? {
        guard let normalized = nonEmpty(raw)?.lowercased() else {
            return nil
        }
        switch normalized {
        case "app.default":
            return "Profile App Default"
        case "message.compact":
            return "Profile Message Compact"
        case "voice.spoken":
            return "Profile Voice Spoken"
        default:
            return "Profile \(normalized)"
        }
    }

    private func continuityPersonaSourceLabel(_ raw: String?) -> String? {
        guard let normalized = nonEmpty(raw) else {
            return nil
        }
        return "Persona \(normalized.capitalized)"
    }

    private func continuityItemTypeLabel(_ itemType: String) -> String {
        switch itemType.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
        case "assistant_message":
            return "Assistant"
        case "user_message":
            return "User"
        case "tool_call":
            return "Tool Call"
        case "tool_result":
            return "Tool Result"
        case "approval_request":
            return "Approval"
        case "approval_decision":
            return "Decision"
        default:
            return itemType
                .replacingOccurrences(of: "_", with: " ")
                .split(separator: " ")
                .map { $0.capitalized }
                .joined(separator: " ")
        }
    }

    private func continuityItemTypeSymbol(_ itemType: String) -> String {
        switch itemType.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
        case "assistant_message":
            return "sparkles"
        case "user_message":
            return "person.crop.circle"
        case "tool_call":
            return "wrench.and.screwdriver"
        case "tool_result":
            return "checkmark.seal"
        case "approval_request":
            return "hand.raised"
        case "approval_decision":
            return "checkmark.shield"
        default:
            return "text.bubble"
        }
    }

    private func continuityItemTypeTint(_ itemType: String) -> Color {
        switch itemType.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
        case "assistant_message":
            return .blue
        case "user_message":
            return .secondary
        case "tool_call", "tool_result":
            return .purple
        case "approval_request", "approval_decision":
            return .orange
        default:
            return .secondary
        }
    }

    private func continuityStatusLabel(_ status: String) -> String {
        switch status.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
        case "completed", "done", "success":
            return "Completed"
        case "failed", "error":
            return "Failed"
        case "awaiting_approval":
            return "Awaiting Approval"
        case "running", "in_progress":
            return "In Progress"
        case "queued", "pending":
            return "Queued"
        case let value where !value.isEmpty:
            return value
                .replacingOccurrences(of: "_", with: " ")
                .split(separator: " ")
                .map { $0.capitalized }
                .joined(separator: " ")
        default:
            return "Unknown"
        }
    }

    private func continuityStatusSymbol(_ status: String) -> String {
        switch status.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
        case "completed", "done", "success":
            return "checkmark.circle.fill"
        case "failed", "error":
            return "xmark.circle.fill"
        case "awaiting_approval":
            return "hand.raised.fill"
        case "running", "in_progress":
            return "clock.arrow.circlepath"
        case "queued", "pending":
            return "clock.fill"
        default:
            return "questionmark.circle"
        }
    }

    private func continuityStatusTint(_ status: String) -> Color {
        switch status.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
        case "completed", "done", "success":
            return .green
        case "failed", "error":
            return .red
        case "awaiting_approval":
            return .orange
        case "running", "in_progress", "queued", "pending":
            return .secondary
        default:
            return .secondary
        }
    }

    private func addressSummary(_ addresses: [CommunicationEventAddressItem]) -> String {
        addresses.map { address in
            let role = address.role.uppercased()
            let value = nonEmpty(address.display) ?? address.value
            return "\(role): \(value)"
        }.joined(separator: " • ")
    }

    private func directionSymbol(_ rawDirection: String) -> String {
        switch rawDirection.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
        case "inbound":
            return "arrow.down.left"
        case "outbound":
            return "arrow.up.right"
        default:
            return "arrow.left.arrow.right"
        }
    }

    private func directionTint(_ rawDirection: String) -> Color {
        switch rawDirection.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
        case "inbound":
            return .green
        case "outbound":
            return .blue
        default:
            return .secondary
        }
    }

    private func callSessionStatusSymbol(_ rawStatus: String) -> String {
        switch rawStatus.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
        case "in_progress":
            return "phone.fill.arrow.up.right"
        case "ringing":
            return "phone.connection.fill"
        case "completed":
            return "phone.down.fill"
        case "failed":
            return "phone.down.waves.left.and.right"
        default:
            return "phone.fill"
        }
    }

    private func callSessionStatusTint(_ rawStatus: String) -> Color {
        switch rawStatus.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
        case "completed":
            return .green
        case "failed":
            return .red
        case "in_progress", "ringing":
            return .orange
        default:
            return .secondary
        }
    }

    private func deliveryAttemptStatusSymbol(_ rawStatus: String) -> String {
        switch rawStatus.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
        case "delivered", "sent", "success":
            return "checkmark.circle.fill"
        case "failed", "error":
            return "xmark.circle.fill"
        case "queued", "retrying", "pending":
            return "clock.fill"
        default:
            return "questionmark.circle.fill"
        }
    }

    private func deliveryAttemptStatusTint(_ rawStatus: String) -> Color {
        switch rawStatus.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
        case "delivered", "sent", "success":
            return .green
        case "failed", "error":
            return .red
        case "queued", "retrying", "pending":
            return .orange
        default:
            return .secondary
        }
    }

    private func deliveryAttemptRoutePhaseTint(_ rawPhase: String) -> Color {
        switch rawPhase.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
        case "primary":
            return .blue
        case "retry":
            return .orange
        case "fallback":
            return .purple
        default:
            return .secondary
        }
    }

    private func detailRow(label: String, value: String) -> some View {
        HStack(alignment: .top, spacing: 8) {
            Text(label)
                .font(.caption.weight(.semibold))
                .foregroundStyle(.secondary)
                .frame(width: 104, alignment: .leading)
            Text(value)
                .font(.caption)
                .foregroundStyle(.secondary)
                .textSelection(.enabled)
        }
    }

    private func detailPill(_ title: String, value: String) -> some View {
        HStack(spacing: 5) {
            Text(title)
                .font(.caption2.weight(.semibold))
                .foregroundStyle(.secondary)
            Text(value)
                .font(.caption2)
                .foregroundStyle(.secondary)
        }
        .padding(.horizontal, 8)
        .padding(.vertical, 4)
        .background(
            Capsule(style: .continuous)
                .fill(Color(nsColor: .controlBackgroundColor).opacity(0.72))
        )
    }

    private func shortIdentifier(_ value: String, limit: Int = 12) -> String {
        guard value.count > limit else {
            return value
        }
        let prefixCount = max(4, (limit / 2) - 1)
        let suffixCount = max(3, limit - prefixCount - 1)
        return "\(value.prefix(prefixCount))…\(value.suffix(suffixCount))"
    }

    private func nonEmpty(_ value: String?) -> String? {
        guard let value else {
            return nil
        }
        let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? nil : trimmed
    }
}
