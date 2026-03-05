import SwiftUI

struct ChatPanelView: View {
    @ObservedObject private var state: AppShellState
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @State private var chatWorkflowDetailsExpanded = false
    @State private var chatWorkflowContextExpanded = false
    @State private var chatExplainabilitySectionExpanded = false
    @State private var chatExplainabilityToolCatalogExpanded = false
    @State private var chatExplainabilityPolicyExpanded = false
    @State private var timelineDetailsExpandedItemIDs: Set<String> = []
    @State private var chatActingAsExpanded = false
    @State private var inlineApprovalIntentDrafts: [String: InlineApprovalDecisionIntent] = [:]
    @State private var inlineApprovalActorDrafts: [String: String] = [:]
    @State private var inlineApprovalPhraseDrafts: [String: String] = [:]
    @State private var inlineApprovalExpandedItemIDs: Set<String> = []
    @State private var pendingInlineApprovalConfirmation: InlineApprovalConfirmation?

    private enum InlineApprovalDecisionIntent: String, CaseIterable, Identifiable {
        case approve
        case reject

        var id: String { rawValue }
        var title: String {
            switch self {
            case .approve:
                return "Approve"
            case .reject:
                return "Reject"
            }
        }

        var submitTitle: String {
            switch self {
            case .approve:
                return "Approve Here"
            case .reject:
                return "Reject Here"
            }
        }
    }

    private struct InlineApprovalConfirmation: Identifiable {
        let itemID: String
        let approvalID: String
        let intent: InlineApprovalDecisionIntent
        let actorID: String
        let decisionPhrase: String

        var id: String { "\(itemID)::\(approvalID)::\(intent.rawValue)" }

        var title: String {
            switch intent {
            case .approve:
                return "Confirm Inline Approval?"
            case .reject:
                return "Confirm Inline Rejection?"
            }
        }

        var message: String {
            switch intent {
            case .approve:
                return "This low-risk approval will be submitted immediately by \(actorID)."
            case .reject:
                return "This low-risk approval will be rejected immediately by \(actorID)."
            }
        }

        var confirmButtonTitle: String {
            switch intent {
            case .approve:
                return "Confirm Approve"
            case .reject:
                return "Confirm Reject"
            }
        }
    }

    init(state: AppShellState) {
        self.state = state
    }

    var body: some View {
        VStack(spacing: 0) {
            header
                .padding(.horizontal, UIStyle.panelPadding)
                .padding(.vertical, 10)

            if let runtimeBannerMessage {
                RuntimeStateBanner(message: runtimeBannerMessage)
                    .padding(.horizontal, UIStyle.panelPadding)
                    .padding(.bottom, 12)
            }

            if let panelProblemRemediation {
                PanelProblemRemediationCardView(context: panelProblemRemediation) { actionID in
                    state.performPanelProblemRemediationAction(actionID, section: .chat)
                }
                .padding(.horizontal, UIStyle.panelPadding)
                .padding(.bottom, 12)
            }

            if let chatRouteRemediationMessage = state.chatRouteRemediationMessage {
                chatRouteRemediationCard(message: chatRouteRemediationMessage)
                    .padding(.horizontal, UIStyle.panelPadding)
                    .padding(.bottom, 12)
            }

            if panelProblemRemediation == nil,
               let chatFailureRemediationMessage = state.chatFailureRemediationMessage {
                chatFailureRemediationCard(message: chatFailureRemediationMessage)
                    .padding(.horizontal, UIStyle.panelPadding)
                    .padding(.bottom, 12)
            }

            if showsWorkflowContextCard {
                chatWorkflowContextCard
                    .padding(.horizontal, UIStyle.panelPadding)
                    .padding(.bottom, 12)
            }

            Divider()

            transcriptView

            Divider()

            composer
                .padding(UIStyle.panelPadding)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .confirmationDialog(
            pendingInlineApprovalConfirmation?.title ?? "Confirm",
            isPresented: inlineApprovalConfirmationBinding,
            titleVisibility: .visible
        ) {
            if let confirmation = pendingInlineApprovalConfirmation {
                Button(
                    confirmation.confirmButtonTitle,
                    role: confirmation.intent == .reject ? .destructive : nil
                ) {
                    submitInlineApprovalDecision(confirmation)
                }
            }
            Button("Cancel", role: .cancel) {}
        } message: {
            if let confirmation = pendingInlineApprovalConfirmation {
                Text(confirmation.message)
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
        state.panelProblemRemediation(for: .chat)
    }

    private var header: some View {
        HStack(alignment: .top, spacing: UIStyle.standardSpacing) {
            VStack(alignment: .leading, spacing: 4) {
                Text("Assistant")
                    .font(.title3.weight(.semibold))
                Text(state.chatStatusMessage ?? "Conversation and workflow status")
                    .font(.subheadline)
                    .foregroundStyle(.secondary)
            }

            Spacer()

            if state.isChatStreaming {
                VStack(alignment: .trailing, spacing: 6) {
                    HStack(spacing: 6) {
                        ProgressView()
                            .controlSize(.small)
                        Text(state.chatProgressDetail ?? "Streaming")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }

                    Button {
                        state.interruptActiveChat()
                    } label: {
                        Label("Interrupt", systemImage: "stop.fill")
                    }
                    .buttonStyle(.bordered)
                    .controlSize(.small)
                    .disabled(state.isChatInterruptInFlight)

                }
            }

            Button {
                state.refreshDaemonStatus()
            } label: {
                Label("Refresh", systemImage: "arrow.clockwise")
            }
            .quietButtonChrome()

            Menu {
                Button("Refresh Daemon Status") {
                    state.refreshDaemonStatus()
                }
                Button("Retry Realtime Stream") {
                    state.retryChatRealtimeStream()
                }
                .disabled(state.isChatStreaming || state.isChatRealtimeRetryInFlight)
                Button("Open Configuration") {
                    state.navigateToSection(.configuration)
                }
                Button("Open Models") {
                    state.openModelsForChatRemediation()
                }
            } label: {
                Label("Actions", systemImage: "ellipsis.circle")
            }
            .controlSize(.regular)
        }
    }

    private var transcriptView: some View {
        ScrollViewReader { proxy in
            ScrollView {
                LazyVStack(alignment: .leading, spacing: UIStyle.standardSpacing) {
                    if state.chatTimelineItems.isEmpty, !state.isChatStreaming {
                        chatEmptyState
                    }

                    ForEach(state.chatTimelineItems) { item in
                        timelineRow(item)
                            .id(item.id)
                            .accessibilityIdentifier("chat-timeline-item-\(item.id)")
                    }

                    if state.isChatStreaming {
                        HStack(spacing: 8) {
                            ProgressView()
                                .controlSize(.small)
                            Text(state.chatProgressDetail ?? "Assistant is preparing a response…")
                                .font(.callout)
                                .foregroundStyle(.secondary)
                        }
                        .padding(.horizontal, 12)
                        .padding(.vertical, 10)
                        .cardSurface(.subtle)
                    }

                    Color.clear
                        .frame(height: 1)
                        .id("chat-bottom-anchor")
                }
                .padding(UIStyle.panelPadding)
            }
            .background(UIStyle.panelGradient)
            .accessibilityIdentifier("chat-timeline-scroll")
            .cardSurface(.subtle)
            .padding(.horizontal, UIStyle.panelPadding)
            .padding(.vertical, 10)
            .onChange(of: transcriptAutoScrollToken) { _, _ in
                scheduleTranscriptAutoScroll(proxy: proxy)
            }
            .onAppear {
                scheduleTranscriptAutoScroll(proxy: proxy)
            }
        }
    }

    private var transcriptAutoScrollToken: String {
        let lastItemSignature: String
        if let lastItem = state.chatTimelineItems.last {
            lastItemSignature = [
                lastItem.id,
                lastItem.kind.rawValue,
                lastItem.state.rawValue,
                lastItem.title,
                lastItem.summary,
                lastItem.content ?? "",
                "\(lastItem.details.count)"
            ].joined(separator: "|")
        } else {
            lastItemSignature = "none"
        }
        return [
            "\(state.chatTimelineItems.count)",
            state.isChatStreaming ? "streaming" : "idle",
            state.chatProgressDetail ?? "",
            lastItemSignature
        ].joined(separator: "||")
    }

    private var composer: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack {
                Text("Message")
                    .font(.subheadline.weight(.semibold))
                Spacer()
                Text("Enter to send  ·  Shift+Enter for newline")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            DisclosureGroup(
                isExpanded: chatActingAsExpandedBinding
            ) {
                VStack(alignment: .leading, spacing: 8) {
                    HStack(alignment: .center, spacing: 8) {
                        Text("Acting As")
                            .font(.caption.weight(.semibold))
                            .foregroundStyle(.secondary)
                        Picker("Acting As", selection: $state.selectedPrincipal) {
                            ForEach(chatActingAsOptions, id: \.self) { actorID in
                                Text(state.principalIdentityDisplayValue(for: actorID).displayText)
                                    .tag(actorID)
                            }
                        }
                        .pickerStyle(.menu)
                        .labelsHidden()
                        .accessibilityIdentifier("chat-acting-as-picker")

                        if !chatIsUsingDefaultActingAs {
                            Button("Use Default Actor") {
                                state.selectedPrincipal = "default"
                            }
                            .buttonStyle(.bordered)
                            .controlSize(.small)
                        }
                        Spacer(minLength: 0)
                    }

                    if let validationMessage = chatActingAsValidationMessage {
                        Text(validationMessage)
                            .font(.caption)
                            .foregroundStyle(.orange)
                    }
                }
                .padding(.top, 4)
            } label: {
                HStack(spacing: 6) {
                    Text("Advanced Override")
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(.secondary)
                    Text(chatActingAsSummary)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }
            .accessibilityIdentifier("chat-acting-as-disclosure")

            ChatComposerTextView(
                text: $state.chatDraft,
                onSubmit: {
                    if !isSendDisabled {
                        state.sendChatDraft()
                    }
                }
            )
            .accessibilityIdentifier("chat-composer-input")
            .frame(minHeight: 96, maxHeight: 168)
            .padding(6)
            .background(
                RoundedRectangle(cornerRadius: UIStyle.controlCornerRadius, style: .continuous)
                    .fill(Color(nsColor: .textBackgroundColor).opacity(0.85))
            )

            HStack {
                Text(composerFootnote)
                    .font(.caption)
                    .foregroundStyle(.secondary)

                if showsChatSendSuccessBadge {
                    TahoeStatusBadge(
                        text: "Turn Sent",
                        symbolName: "checkmark.circle.fill",
                        tint: .green
                    )
                    .controlSize(.small)
                    .transition(.opacity)
                }

                Spacer()
                Button {
                    state.sendChatDraft()
                } label: {
                    Label("Send", systemImage: "paperplane.fill")
                        .font(.callout.weight(.semibold))
                }
                .buttonStyle(.borderedProminent)
                .successSymbolEffect(
                    state.successNotificationPulse(for: "chat"),
                    reduceMotion: reduceMotion
                )
                .disabled(isSendDisabled)
                .accessibilityLabel("Send message")
                .accessibilityHint("Sends this message and lets the assistant choose response or tool execution automatically.")
                .accessibilityIdentifier("chat-send-button")
            }
        }
        .padding(12)
        .cardSurface(.standard)
    }

    private var isSendDisabled: Bool {
        state.chatDraft.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
            || state.isChatStreaming
            || chatActingAsValidationMessage != nil
    }

    private var chatActingAsOptions: [String] {
        state.actingAsOptions(including: state.selectedPrincipal)
    }

    private var chatActingAsValidationMessage: String? {
        state.selectedActingAsValidationMessage
    }

    private var chatIsUsingDefaultActingAs: Bool {
        let trimmed = state.selectedPrincipal.trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty || trimmed.caseInsensitiveCompare("default") == .orderedSame
    }

    private var chatActingAsRequiresExplicitSelection: Bool {
        !chatIsUsingDefaultActingAs || chatActingAsValidationMessage != nil
    }

    private var chatActingAsExpandedBinding: Binding<Bool> {
        Binding(
            get: { chatActingAsRequiresExplicitSelection || chatActingAsExpanded },
            set: { expanded in
                chatActingAsExpanded = expanded
            }
        )
    }

    private var chatActingAsSummary: String {
        if chatIsUsingDefaultActingAs {
            return "Auto"
        }
        return state.principalIdentityDisplayValue(for: state.selectedPrincipal).displayText
    }

    private var showsChatSendSuccessBadge: Bool {
        guard !state.isChatStreaming else {
            return false
        }
        let normalized = state.chatStatusMessage?
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .lowercased()
        return normalized?.hasPrefix("provider:") == true
    }

    @ViewBuilder
    private func timelineRow(_ item: ChatTimelineItem) -> some View {
        switch item.kind {
        case .userMessage, .assistantMessage:
            timelineMessageRow(item)
        case .toolCall, .toolResult, .approvalRequest, .approvalDecision, .systemStatus:
            timelineWorkflowRow(item)
        }
    }

    private func timelineMessageRow(_ item: ChatTimelineItem) -> some View {
        let isUserMessage = item.isUserMessage
        let renderedText = nonEmpty(item.content) ?? item.summary
        let assistantMessagePrefersExpandedWidth = !isUserMessage
            && ChatMarkdownParser.prefersExpandedTimelineWidth(renderedText)
        let bubble = VStack(alignment: .leading, spacing: 6) {
            Text(item.title)
                .font(.caption.weight(.semibold))
                .foregroundStyle(.secondary)

            ChatMarkdownContentView(
                text: renderedText,
                style: .body
            )
        }
        .padding(.horizontal, 13)
        .padding(.vertical, 11)
        .background(timelineMessageBackground(isUserMessage: isUserMessage))
        .clipShape(RoundedRectangle(cornerRadius: UIStyle.cardCornerRadius, style: .continuous))

        return HStack(alignment: .top, spacing: 0) {
            if isUserMessage {
                Spacer(minLength: 86)
            }
            if isUserMessage {
                bubble
                    .fixedSize(horizontal: false, vertical: true)
            } else {
                bubble
                    .frame(
                        maxWidth: assistantMessagePrefersExpandedWidth ? .infinity : 660,
                        alignment: .leading
                    )
            }
            if !isUserMessage, !assistantMessagePrefersExpandedWidth {
                Spacer(minLength: 86)
            }
        }
    }

    private func timelineWorkflowRow(_ item: ChatTimelineItem) -> some View {
        let actions = state.chatTimelineActions(for: item)
        let firstDisabledReason = actions
            .first(where: { !$0.enabled })?
            .disabledReason
            .flatMap(nonEmpty)
        let actionStatus = state.chatTimelineActionStatus(for: item.id)
        let actionInFlight = state.isChatTimelineActionInFlight(itemID: item.id)
        return VStack(alignment: .leading, spacing: 8) {
            HStack(spacing: 8) {
                Label(item.title, systemImage: item.state.symbolName)
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(item.state.tint)
                Spacer(minLength: 0)
                TahoeStatusBadge(
                    text: item.state.label,
                    symbolName: item.state.symbolName,
                    tint: item.state.tint
                )
                .controlSize(.small)
            }

            Text(item.summary)
                .font(.subheadline)
                .foregroundStyle(.primary)

            if let toolChainLabel = item.toolChainLabel {
                HStack(spacing: 6) {
                    Text(toolChainLabel)
                        .font(.caption2.weight(.semibold))
                        .foregroundStyle(.secondary)
                        .accessibilityIdentifier("chat-timeline-chain-label")
                    if let toolChainStepLabel = item.toolChainStepLabel {
                        Text("•")
                            .font(.caption2)
                            .foregroundStyle(.tertiary)
                        Text(toolChainStepLabel)
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                            .accessibilityIdentifier("chat-timeline-chain-step-label")
                    }
                }
            }

            if let content = nonEmpty(item.content) {
                ChatMarkdownContentView(
                    text: content,
                    style: .caption
                )
                    .foregroundStyle(.secondary)
            }

            if item.kind == .approvalRequest {
                timelineApprovalHandoffSection(item)
            }

            if !actions.isEmpty {
                HStack(spacing: 8) {
                    ForEach(actions) { action in
                        timelineActionButton(action, itemID: item.id)
                    }
                }
            }

            if let firstDisabledReason {
                Text(firstDisabledReason)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            if actionInFlight {
                HStack(spacing: 6) {
                    ProgressView()
                        .controlSize(.small)
                    Text("Applying action…")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            } else if let actionStatus {
                Text(actionStatus)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            if item.hasDetails {
                DisclosureGroup(
                    "Details",
                    isExpanded: timelineDetailsExpandedBinding(for: item.id)
                ) {
                    VStack(alignment: .leading, spacing: 4) {
                        ForEach(item.details) { detail in
                            timelineDetailRow(label: detail.label, value: detail.value)
                        }
                    }
                    .padding(.top, 4)
                }
                .font(.caption.weight(.semibold))
            }
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .cardSurface(.standard)
        .onAppear {
            state.loadChatApprovalContextIfNeeded(for: item)
        }
    }

    @ViewBuilder
    private func timelineApprovalHandoffSection(_ item: ChatTimelineItem) -> some View {
        let approvalID = state.chatApprovalRequestID(for: item)
        let approval = state.chatApprovalInboxItem(for: item)

        VStack(alignment: .leading, spacing: 8) {
            Divider()

            Text("Approval")
                .font(.caption.weight(.semibold))
                .foregroundStyle(.secondary)

            if approvalID == nil {
                Text("Approval request ID is missing. Open Approvals to continue this workflow.")
                    .font(.caption)
                    .foregroundStyle(.orange)
            } else if state.isApprovalsLoading, approval == nil {
                HStack(spacing: 8) {
                    ProgressView()
                        .controlSize(.small)
                    Text("Loading approval context…")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            } else if let approval {
                if approval.decisionState == .pending {
                    Text("Decision required. Open Approvals to review evidence and submit the required decision phrase.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    if let inlineFastPathBlockedReason = state.chatInlineApprovalFastPathBlockedReason(for: approval) {
                        Text(inlineFastPathBlockedReason)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    } else {
                        timelineInlineApprovalFastPath(item: item, approval: approval)
                    }
                } else {
                    Text("Decision recorded in Approvals. Use Resume Turn to continue once available.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            } else {
                Text("Open Approvals for full decision controls and evidence details.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
    }

    private func timelineInlineApprovalFastPath(item: ChatTimelineItem, approval: ApprovalInboxItem) -> some View {
        DisclosureGroup(
            isExpanded: inlineApprovalExpandedBinding(for: item.id)
        ) {
            VStack(alignment: .leading, spacing: 8) {
                HStack(spacing: 8) {
                    TahoeStatusBadge(
                        text: "Low-Risk Fast Path",
                        symbolName: "bolt.fill",
                        tint: .green
                    )
                    .controlSize(.small)
                    TahoeStatusBadge(
                        text: approval.riskLevel.label,
                        symbolName: approval.riskLevel.symbolName,
                        tint: approval.riskLevel.tint
                    )
                    .controlSize(.small)
                    Spacer(minLength: 0)
                }

                Text("Scope: \(approval.taskTitle) • Acting as \(state.principalIdentityDisplayValue(for: approval.actingAsActorID).displayText).")
                    .font(.caption)
                    .foregroundStyle(.secondary)

                Picker("Action", selection: inlineApprovalIntentBinding(for: approval)) {
                    ForEach(InlineApprovalDecisionIntent.allCases) { intent in
                        Text(intent.title).tag(intent)
                    }
                }
                .pickerStyle(.segmented)

                HStack(alignment: .center, spacing: 8) {
                    Text("Decision By")
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(.secondary)
                    Picker("Decision By", selection: inlineApprovalActorBinding(for: approval)) {
                        ForEach(inlineApprovalActorOptions(for: approval), id: \.self) { actorID in
                            Text(state.principalIdentityDisplayValue(for: actorID).displayText)
                                .tag(actorID)
                        }
                    }
                    .pickerStyle(.menu)
                    .labelsHidden()
                    Spacer(minLength: 0)
                }

                if inlineApprovalIntent(for: approval) == .approve {
                    TextField("Use Required Phrase", text: inlineApprovalPhraseBinding(for: approval))
                        .textFieldStyle(.roundedBorder)
                    HStack(spacing: 8) {
                        Button("Use Required Phrase") {
                            inlineApprovalPhraseDrafts[approval.id] = state.approvalRequiredPhrase(for: approval)
                        }
                        .buttonStyle(.bordered)
                        .controlSize(.small)
                        Text("Required: \(state.approvalRequiredPhrase(for: approval))")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                }

                if let actorValidationMessage = inlineApprovalActorValidationMessage(for: approval) {
                    Text(actorValidationMessage)
                        .font(.caption)
                        .foregroundStyle(.orange)
                } else if let phraseValidationMessage = inlineApprovalPhraseValidationMessage(for: approval) {
                    Text(phraseValidationMessage)
                        .font(.caption)
                        .foregroundStyle(.orange)
                }

                HStack(spacing: 8) {
                    Button(
                        state.isChatApprovalDecisionInFlight(for: item)
                            ? "Submitting…"
                            : inlineApprovalIntent(for: approval).submitTitle
                    ) {
                        requestInlineApprovalDecision(item: item, approval: approval)
                    }
                    .buttonStyle(.borderedProminent)
                    .disabled(inlineApprovalSubmitDisabled(item: item, approval: approval))

                    Button("Undo Draft") {
                        resetInlineApprovalDrafts(for: approval)
                    }
                    .buttonStyle(.bordered)
                    .disabled(state.isChatApprovalDecisionInFlight(for: item))
                }

                if state.isChatApprovalDecisionInFlight(for: item) {
                    HStack(spacing: 6) {
                        ProgressView()
                            .controlSize(.small)
                        Text("Submitting approval decision…")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                } else if let actionStatus = state.chatApprovalActionStatus(for: item) {
                    Text(actionStatus)
                        .font(.caption)
                        .foregroundStyle(chatApprovalActionStatusTint(actionStatus))
                }

                Text("Need full evidence or detailed rationale? Open Approvals.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            .padding(.top, 6)
        } label: {
            Text("Low-Risk Inline Decision")
                .font(.caption.weight(.semibold))
                .foregroundStyle(.secondary)
        }
        .font(.caption)
    }

    private var inlineApprovalConfirmationBinding: Binding<Bool> {
        Binding(
            get: { pendingInlineApprovalConfirmation != nil },
            set: { presented in
                if !presented {
                    pendingInlineApprovalConfirmation = nil
                }
            }
        )
    }

    private func inlineApprovalExpandedBinding(for itemID: String) -> Binding<Bool> {
        Binding(
            get: { inlineApprovalExpandedItemIDs.contains(itemID) },
            set: { expanded in
                if expanded {
                    inlineApprovalExpandedItemIDs.insert(itemID)
                } else {
                    inlineApprovalExpandedItemIDs.remove(itemID)
                }
            }
        )
    }

    private func inlineApprovalIntent(for approval: ApprovalInboxItem) -> InlineApprovalDecisionIntent {
        inlineApprovalIntentDrafts[approval.id] ?? .approve
    }

    private func inlineApprovalIntentBinding(for approval: ApprovalInboxItem) -> Binding<InlineApprovalDecisionIntent> {
        Binding(
            get: { inlineApprovalIntent(for: approval) },
            set: { intent in
                inlineApprovalIntentDrafts[approval.id] = intent
                if intent == .approve {
                    let trimmedDraft = inlineApprovalPhraseDrafts[approval.id]?
                        .trimmingCharacters(in: .whitespacesAndNewlines)
                    if trimmedDraft == nil || trimmedDraft?.isEmpty == true {
                        inlineApprovalPhraseDrafts[approval.id] = state.approvalRequiredPhrase(for: approval)
                    }
                }
            }
        )
    }

    private func inlineApprovalActorBinding(for approval: ApprovalInboxItem) -> Binding<String> {
        Binding(
            get: {
                if let draft = inlineApprovalActorDrafts[approval.id] {
                    return draft
                }
                return state.defaultApprovalDecisionActor(for: approval)
            },
            set: { inlineApprovalActorDrafts[approval.id] = $0 }
        )
    }

    private func inlineApprovalPhraseBinding(for approval: ApprovalInboxItem) -> Binding<String> {
        Binding(
            get: {
                if let draft = inlineApprovalPhraseDrafts[approval.id] {
                    return draft
                }
                if inlineApprovalIntent(for: approval) == .approve {
                    return state.approvalRequiredPhrase(for: approval)
                }
                return ""
            },
            set: { inlineApprovalPhraseDrafts[approval.id] = $0 }
        )
    }

    private func inlineApprovalActorOptions(for approval: ApprovalInboxItem) -> [String] {
        let selectedActorID = inlineApprovalActorBinding(for: approval).wrappedValue
            .trimmingCharacters(in: .whitespacesAndNewlines)
        return state.approvalDecisionActorOptions(including: selectedActorID)
    }

    private func inlineApprovalActorValidationMessage(for approval: ApprovalInboxItem) -> String? {
        let actorID = inlineApprovalActorBinding(for: approval).wrappedValue
            .trimmingCharacters(in: .whitespacesAndNewlines)
        return state.approvalDecisionActorValidationMessage(actorID: actorID)
    }

    private func inlineApprovalPhraseValidationMessage(for approval: ApprovalInboxItem) -> String? {
        guard inlineApprovalIntent(for: approval) == .approve else {
            return nil
        }
        let phraseDraft = inlineApprovalPhraseBinding(for: approval).wrappedValue
        return state.approvalApprovePhraseValidationMessage(phrase: phraseDraft, item: approval)
    }

    private func inlineApprovalSubmitDisabled(item: ChatTimelineItem, approval: ApprovalInboxItem) -> Bool {
        if state.isChatApprovalDecisionInFlight(for: item) {
            return true
        }
        if inlineApprovalActorValidationMessage(for: approval) != nil {
            return true
        }
        if inlineApprovalIntent(for: approval) == .approve,
           inlineApprovalPhraseValidationMessage(for: approval) != nil {
            return true
        }
        return false
    }

    private func resetInlineApprovalDrafts(for approval: ApprovalInboxItem) {
        inlineApprovalIntentDrafts[approval.id] = .approve
        inlineApprovalActorDrafts[approval.id] = state.defaultApprovalDecisionActor(for: approval)
        inlineApprovalPhraseDrafts[approval.id] = state.approvalRequiredPhrase(for: approval)
        state.approvalsActionStatusByID[approval.id] = "Inline draft reset. Ready to submit."
    }

    private func requestInlineApprovalDecision(item: ChatTimelineItem, approval: ApprovalInboxItem) {
        if let actorValidationMessage = inlineApprovalActorValidationMessage(for: approval) {
            state.approvalsActionStatusByID[approval.id] = actorValidationMessage
            return
        }
        if let phraseValidationMessage = inlineApprovalPhraseValidationMessage(for: approval) {
            state.approvalsActionStatusByID[approval.id] = phraseValidationMessage
            return
        }

        let intent = inlineApprovalIntent(for: approval)
        let actorID = inlineApprovalActorBinding(for: approval).wrappedValue
            .trimmingCharacters(in: .whitespacesAndNewlines)
        let phraseDraft = inlineApprovalPhraseBinding(for: approval).wrappedValue
            .trimmingCharacters(in: .whitespacesAndNewlines)
        let decisionPhrase: String
        switch intent {
        case .approve:
            decisionPhrase = state.approvalRequiredPhrase(for: approval)
        case .reject:
            decisionPhrase = phraseDraft.isEmpty ? "REJECT" : phraseDraft
        }

        pendingInlineApprovalConfirmation = InlineApprovalConfirmation(
            itemID: item.id,
            approvalID: approval.id,
            intent: intent,
            actorID: actorID,
            decisionPhrase: decisionPhrase
        )
    }

    private func submitInlineApprovalDecision(_ confirmation: InlineApprovalConfirmation) {
        pendingInlineApprovalConfirmation = nil
        state.submitApprovalDecision(
            approvalID: confirmation.approvalID,
            decisionPhrase: confirmation.decisionPhrase,
            decisionByActorID: confirmation.actorID,
            rationale: "Inline low-risk decision from Chat."
        )
    }

    private func chatApprovalActionStatusTint(_ value: String) -> Color {
        let normalized = value.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        if normalized.contains("approved")
            || normalized.contains("decision submitted")
            || normalized.contains("resumed") {
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

    private func timelineDetailsExpandedBinding(for itemID: String) -> Binding<Bool> {
        Binding(
            get: { timelineDetailsExpandedItemIDs.contains(itemID) },
            set: { expanded in
                if expanded {
                    timelineDetailsExpandedItemIDs.insert(itemID)
                } else {
                    timelineDetailsExpandedItemIDs.remove(itemID)
                }
            }
        )
    }

    private func timelineDetailRow(label: String, value: String) -> some View {
        HStack(alignment: .firstTextBaseline, spacing: 8) {
            Text(label)
                .font(.caption.weight(.semibold))
                .foregroundStyle(.secondary)
                .frame(width: 96, alignment: .leading)
            Text(value)
                .font(.caption.monospaced())
                .foregroundStyle(.secondary)
                .frame(maxWidth: .infinity, alignment: .leading)
                .textSelection(.enabled)
        }
    }

    @ViewBuilder
    private func timelineActionButton(_ action: ChatTimelineActionItem, itemID: String) -> some View {
        let actionInFlight = state.isChatTimelineActionInFlight(itemID: itemID)
        let button = Button(action.title) {
            state.performChatTimelineAction(itemID: itemID, intent: action.intent)
        }
        .disabled(!action.enabled || actionInFlight)

        switch action.style {
        case .primary:
            button.buttonStyle(.borderedProminent)
        case .secondary:
            button.buttonStyle(.bordered)
        case .destructive:
            button
                .buttonStyle(.bordered)
                .tint(.red)
        }
    }

    @ViewBuilder
    private func timelineMessageBackground(isUserMessage: Bool) -> some View {
        if isUserMessage {
            RoundedRectangle(cornerRadius: UIStyle.cardCornerRadius, style: .continuous)
                .fill(
                    LinearGradient(
                        colors: [
                            Color.accentColor.opacity(0.28),
                            Color.accentColor.opacity(0.18)
                        ],
                        startPoint: .topLeading,
                        endPoint: .bottomTrailing
                    )
                )
        } else {
            RoundedRectangle(cornerRadius: UIStyle.cardCornerRadius, style: .continuous)
                .fill(Color(nsColor: .controlBackgroundColor).opacity(0.72))
        }
    }

    private func scheduleTranscriptAutoScroll(proxy: ScrollViewProxy) {
        scrollToBottom(proxy: proxy, animated: !state.isChatStreaming)
        DispatchQueue.main.async {
            scrollToBottom(proxy: proxy, animated: false)
        }
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.12) {
            scrollToBottom(proxy: proxy, animated: false)
        }
    }

    private func scrollToBottom(proxy: ScrollViewProxy, animated: Bool = true) {
        let anchorID = "chat-bottom-anchor"
        let performScroll = {
            proxy.scrollTo(anchorID, anchor: .bottom)
        }
        if reduceMotion || !animated {
            performScroll()
        } else {
            withAnimation(.easeOut(duration: 0.18)) {
                performScroll()
            }
        }
    }

    private var composerFootnote: String {
        if state.modelRouteSummary != nil {
            return "Assistant selects route and tools automatically. Expand Effective Workflow Context for route details."
        }
        return "Responses stream live when available and automatically fall back to one-shot reply mode."
    }

    private func chatRouteRemediationCard(message: String) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Label("Set Up Chat Route", systemImage: "wrench.and.screwdriver.fill")
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(.orange)

            Text(message)
                .font(.caption)
                .foregroundStyle(.secondary)

            HStack(spacing: 8) {
                Button(state.isChatFixAndContinueInFlight ? "Fixing…" : "Fix and Continue") {
                    state.runChatFixAndContinueFromRouteRemediation()
                }
                .buttonStyle(.borderedProminent)
                .disabled(state.isChatFixAndContinueInFlight)

                Button("Open Models") {
                    state.openModelsForChatRemediation()
                }
                .buttonStyle(.bordered)

                Button("Check Again") {
                    state.refreshChatRoutePreflight()
                }
                .buttonStyle(.bordered)
            }

            if let status = state.chatFixAndContinueStatusMessage {
                HStack(spacing: 6) {
                    if state.isChatFixAndContinueInFlight {
                        ProgressView()
                            .controlSize(.small)
                    }
                    Text(status)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .cardSurface(.standard)
    }

    private func chatFailureRemediationCard(message: String) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Label("Couldn't Send Message", systemImage: "exclamationmark.triangle.fill")
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(.orange)

            Text(message)
                .font(.caption)
                .foregroundStyle(.secondary)

            HStack(spacing: 8) {
                Button(state.isChatFixAndContinueInFlight ? "Fixing…" : "Fix and Continue") {
                    state.runChatFixAndContinueFromFailureRemediation()
                }
                .buttonStyle(.borderedProminent)
                .disabled(state.isChatFixAndContinueInFlight)

                Button("Refresh Daemon") {
                    state.refreshDaemonStatus()
                }
                .buttonStyle(.bordered)

                Button("Restore Prompt") {
                    state.restoreLastFailedChatDraftForRetry()
                }
                .buttonStyle(.bordered)
                .disabled(state.chatLastFailedDraft == nil)

                Button("Open Configuration") {
                    state.navigateToSection(.configuration)
                }
                .buttonStyle(.bordered)
            }

            if let status = state.chatFixAndContinueStatusMessage {
                HStack(spacing: 6) {
                    if state.isChatFixAndContinueInFlight {
                        ProgressView()
                            .controlSize(.small)
                    }
                    Text(status)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .cardSurface(.standard)
    }

    private var showsWorkflowContextCard: Bool {
        state.isChatStreaming
            || state.chatLatestTurnTraceability != nil
            || state.modelRouteSummary != nil
            || state.chatLatestTurnExplainability != nil
            || state.isChatExplainabilityInFlight
            || state.chatExplainabilityErrorMessage != nil
    }

    private var chatWorkflowContextCard: some View {
        let traceability = state.chatLatestTurnTraceability
        let explainability = state.chatLatestTurnExplainability
        let contextState = workflowContextState(traceability: traceability)
        let workflowSummary = state.chatWorkflowCardSummary()
        let taskClass = nonEmpty(traceability?.taskClass)
            ?? nonEmpty(explainability?.taskClass)
            ?? ((state.modelRouteSummary != nil || state.isChatStreaming) ? "chat" : nil)
        let providerID = nonEmpty(traceability?.provider)
            ?? nonEmpty(explainability?.selectedProvider)
            ?? nonEmpty(state.modelRouteSummary?.provider)
        let modelKey = nonEmpty(traceability?.modelKey)
            ?? nonEmpty(explainability?.selectedModelKey)
            ?? nonEmpty(state.modelRouteSummary?.modelKey)
        let routeSource = nonEmpty(traceability?.routeSource)
            ?? nonEmpty(explainability?.selectedSource)
            ?? nonEmpty(state.modelRouteSummary?.source)
        let correlationID = nonEmpty(traceability?.correlationID) ?? nonEmpty(state.chatActiveCorrelationID)
        let responseShapingChannel = nonEmpty(traceability?.responseShapingChannel)
        let responseShapingProfile = nonEmpty(traceability?.responseShapingProfile)
        let personaPolicySource = nonEmpty(traceability?.personaPolicySource)
        let detailRows = ProgressiveDisclosureDetails.chatWorkflowDetails(
            traceability: traceability,
            routeSource: routeSource,
            correlationID: correlationID
        )

        return VStack(alignment: .leading, spacing: 8) {
            HStack(spacing: 8) {
                Label("Effective Workflow Context", systemImage: "point.3.connected.trianglepath.dotted")
                    .font(.subheadline.weight(.semibold))
                Spacer(minLength: 0)
                TahoeStatusBadge(
                    text: workflowContextStateLabel(contextState),
                    symbolName: workflowContextStateSymbol(contextState),
                    tint: workflowContextStateTint(contextState)
                )
                Button {
                    chatWorkflowContextExpanded.toggle()
                } label: {
                    Label(
                        chatWorkflowContextExpanded ? "Hide" : "Show",
                        systemImage: chatWorkflowContextExpanded ? "chevron.up" : "chevron.down"
                    )
                }
                .buttonStyle(.bordered)
                .controlSize(.small)
                .accessibilityIdentifier("chat-workflow-context-toggle")
            }

            if chatWorkflowContextExpanded {
                WorkflowCardSummaryView(summary: workflowSummary)

                if let taskClass {
                    traceabilityDetailRow(label: "Task Class", value: taskClass, monospacedValue: false)
                }
                if let providerID {
                    traceabilityDetailRow(
                        label: "Provider",
                        value: providerDisplayName(providerID),
                        monospacedValue: false
                    )
                }
                if let modelKey {
                    traceabilityDetailRow(label: "Model", value: modelKey, monospacedValue: false)
                }
                if responseShapingChannel != nil
                    || responseShapingProfile != nil
                    || personaPolicySource != nil {
                    HStack(spacing: 8) {
                        if let responseShapingChannel {
                            TahoeStatusBadge(
                                text: responseShapingChannelLabel(responseShapingChannel),
                                symbolName: responseShapingChannelSymbol(responseShapingChannel),
                                tint: responseShapingChannelTint(responseShapingChannel)
                            )
                            .controlSize(.small)
                        }
                        if let responseShapingProfile {
                            TahoeStatusBadge(
                                text: responseShapingProfileLabel(responseShapingProfile),
                                symbolName: "wand.and.stars",
                                tint: .indigo
                            )
                            .controlSize(.small)
                        }
                        if let personaPolicySource {
                            TahoeStatusBadge(
                                text: personaPolicySourceLabel(personaPolicySource),
                                symbolName: personaPolicySourceSymbol(personaPolicySource),
                                tint: personaPolicySourceTint(personaPolicySource)
                            )
                            .controlSize(.small)
                        }
                        Spacer(minLength: 0)
                    }
                }
                if !detailRows.isEmpty {
                    DisclosureGroup("Details", isExpanded: $chatWorkflowDetailsExpanded) {
                        VStack(alignment: .leading, spacing: 4) {
                            ForEach(detailRows) { row in
                                traceabilityDetailRow(label: row.label, value: row.value)
                            }
                        }
                        .padding(.top, 4)
                    }
                    .font(.caption.weight(.semibold))
                }

                chatExplainabilitySection(
                    traceability: traceability,
                    explainability: explainability
                )

                HStack(spacing: 8) {
                    Button("Open Related Tasks") {
                        state.openTasksForChatTraceability()
                    }
                    .buttonStyle(.bordered)

                    Button("Open Related Inspect") {
                        state.openInspectForChatTraceability()
                    }
                    .buttonStyle(.bordered)
                }

                if contextState == .active {
                    Text("Active turn context will resolve to task and run IDs when this response completes.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                } else if traceability?.approvalRequired == true {
                    let approvalRequestID = nonEmpty(traceability?.approvalRequestID) ?? "unknown"
                    Text("Action is waiting for approval (\(approvalRequestID)). Open Approvals to continue.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                } else if traceability?.clarificationRequired == true {
                    Text(nonEmpty(traceability?.clarificationPrompt) ?? "Action requires clarification before execution can continue.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                } else if let traceability, !traceability.hasTaskOrRunIdentity {
                    Text("Recent turn has no linked task or run IDs yet. Route and trace details are still available.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                } else if contextState == .preview {
                    Text("Route preview is ready. Send a message to attach task and run traceability.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .cardSurface(.standard)
    }

    private enum ChatExplainabilityState {
        case loading
        case ready
        case failed
        case empty
    }

    private func chatExplainabilityState(
        explainability: ChatTurnExplainabilityItem?
    ) -> ChatExplainabilityState {
        if state.isChatExplainabilityInFlight {
            return .loading
        }
        if explainability != nil {
            return .ready
        }
        if state.chatExplainabilityErrorMessage != nil {
            return .failed
        }
        return .empty
    }

    @ViewBuilder
    private func chatExplainabilitySection(
        traceability: ChatTaskRunTraceabilityItem?,
        explainability: ChatTurnExplainabilityItem?
    ) -> some View {
        let sectionState = chatExplainabilityState(explainability: explainability)
        let canOpenInspect = traceability != nil
            || explainability != nil
            || state.modelRouteSummary != nil
            || nonEmpty(state.chatActiveCorrelationID) != nil
        let inspectDisabledReason = canOpenInspect
            ? nil
            : "Open Inspect becomes available after route or trace context is loaded."
        let refreshDisabledReason = state.isChatExplainabilityInFlight
            ? "Explainability refresh is already in progress."
            : nil

        VStack(alignment: .leading, spacing: 8) {
            HStack(spacing: 8) {
                Label("Route + Tool Explainability", systemImage: "list.bullet.clipboard")
                    .font(.caption.weight(.semibold))
                Spacer(minLength: 0)
                TahoeStatusBadge(
                    text: chatExplainabilityStateLabel(sectionState),
                    symbolName: chatExplainabilityStateSymbol(sectionState),
                    tint: chatExplainabilityStateTint(sectionState)
                )
                Button {
                    chatExplainabilitySectionExpanded.toggle()
                } label: {
                    Label(
                        chatExplainabilitySectionExpanded ? "Hide" : "Show",
                        systemImage: chatExplainabilitySectionExpanded ? "chevron.up" : "chevron.down"
                    )
                }
                .buttonStyle(.bordered)
                .controlSize(.small)
                .accessibilityIdentifier("chat-explainability-toggle")
            }

            if chatExplainabilitySectionExpanded {
                switch sectionState {
                case .loading:
                    HStack(spacing: 8) {
                        ProgressView()
                            .controlSize(.small)
                        Text(state.chatExplainabilityStatusMessage ?? "Loading chat explainability trace…")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                case .failed:
                    Text(state.chatExplainabilityErrorMessage ?? "Chat explainability failed.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                case .empty:
                    Text(state.chatExplainabilityStatusMessage ?? "No chat explainability loaded yet.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                case .ready:
                    if let explainability {
                        traceabilityDetailRow(
                            label: "Explain Contract",
                            value: explainability.contractVersion,
                            monospacedValue: false
                        )
                        if !explainability.routeReasonCodes.isEmpty {
                            Text("Reason Codes: \(explainability.routeReasonCodes.joined(separator: ", "))")
                                .font(.caption2)
                                .foregroundStyle(.secondary)
                        }
                        if !explainability.routeExplanations.isEmpty {
                            VStack(alignment: .leading, spacing: 2) {
                                ForEach(explainability.routeExplanations, id: \.self) { explanation in
                                    Text("• \(explanation)")
                                        .font(.caption2)
                                        .foregroundStyle(.secondary)
                                }
                            }
                        }
                        if explainability.toolCatalog.isEmpty {
                            Text("Tool catalog is empty for this turn context.")
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        } else {
                            DisclosureGroup(
                                "Tool Catalog (\(explainability.toolCatalog.count))",
                                isExpanded: $chatExplainabilityToolCatalogExpanded
                            ) {
                                VStack(alignment: .leading, spacing: 6) {
                                    ForEach(explainability.toolCatalog) { tool in
                                        VStack(alignment: .leading, spacing: 2) {
                                            Text(tool.name)
                                                .font(.caption.weight(.semibold))
                                            if let description = nonEmpty(tool.description) {
                                                Text(description)
                                                    .font(.caption2)
                                                    .foregroundStyle(.secondary)
                                            }
                                            if !tool.capabilityKeys.isEmpty {
                                                Text("Capabilities: \(tool.capabilityKeys.joined(separator: ", "))")
                                                    .font(.caption2)
                                                    .foregroundStyle(.secondary)
                                            }
                                            if let schemaSummary = nonEmpty(tool.inputSchemaSummary) {
                                                Text("Input Schema: \(schemaSummary)")
                                                    .font(.caption2)
                                                    .foregroundStyle(.secondary)
                                            }
                                        }
                                        .frame(maxWidth: .infinity, alignment: .leading)
                                    }
                                }
                                .padding(.top, 4)
                            }
                            .font(.caption.weight(.semibold))
                        }
                        if explainability.policyDecisions.isEmpty {
                            Text("No explicit tool policy decisions were returned.")
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        } else {
                            DisclosureGroup(
                                "Policy Decisions (\(explainability.policyDecisions.count))",
                                isExpanded: $chatExplainabilityPolicyExpanded
                            ) {
                                VStack(alignment: .leading, spacing: 4) {
                                    ForEach(explainability.policyDecisions) { decision in
                                        VStack(alignment: .leading, spacing: 2) {
                                            Text("\(decision.toolName): \(decision.decision)")
                                                .font(.caption.weight(.semibold))
                                            if let capabilityKey = nonEmpty(decision.capabilityKey) {
                                                Text("Capability: \(capabilityKey)")
                                                    .font(.caption2)
                                                    .foregroundStyle(.secondary)
                                            }
                                            if let reason = nonEmpty(decision.reason) {
                                                Text(reason)
                                                    .font(.caption2)
                                                    .foregroundStyle(.secondary)
                                            }
                                        }
                                        .frame(maxWidth: .infinity, alignment: .leading)
                                    }
                                }
                                .padding(.top, 4)
                            }
                            .font(.caption.weight(.semibold))
                        }
                    }
                }

                HStack(spacing: 8) {
                    Button("Refresh") {
                        state.refreshChatTurnExplainability()
                    }
                    .buttonStyle(.borderedProminent)
                    .disabled(state.isChatExplainabilityInFlight)
                    .accessibilityIdentifier("chat-explainability-refresh")

                    Button("Open Models") {
                        state.openModelsForChatRemediation()
                    }
                    .buttonStyle(.bordered)
                    .accessibilityIdentifier("chat-explainability-open-models")

                    Button("Open Inspect") {
                        state.openInspectForChatTraceability()
                    }
                    .buttonStyle(.bordered)
                    .disabled(!canOpenInspect)
                    .accessibilityIdentifier("chat-explainability-open-inspect")
                }

                if let refreshDisabledReason {
                    Text(refreshDisabledReason)
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }
                if let inspectDisabledReason {
                    Text(inspectDisabledReason)
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }
            }
        }
        .padding(10)
        .frame(maxWidth: .infinity, alignment: .leading)
        .cardSurface(.subtle)
    }

    private func chatExplainabilityStateLabel(_ state: ChatExplainabilityState) -> String {
        switch state {
        case .loading:
            return "Loading"
        case .ready:
            return "Ready"
        case .failed:
            return "Needs Attention"
        case .empty:
            return "Not Loaded"
        }
    }

    private func chatExplainabilityStateSymbol(_ state: ChatExplainabilityState) -> String {
        switch state {
        case .loading:
            return "clock.arrow.circlepath"
        case .ready:
            return "checkmark.circle.fill"
        case .failed:
            return "exclamationmark.triangle.fill"
        case .empty:
            return "questionmark.circle"
        }
    }

    private func chatExplainabilityStateTint(_ state: ChatExplainabilityState) -> Color {
        switch state {
        case .loading:
            return .secondary
        case .ready:
            return .green
        case .failed:
            return .orange
        case .empty:
            return .secondary
        }
    }

    private func workflowContextState(traceability: ChatTaskRunTraceabilityItem?) -> ChatWorkflowContextState {
        if state.isChatStreaming {
            return .active
        }
        if traceability != nil || state.chatLatestTurnExplainability != nil {
            return .recent
        }
        return .preview
    }

    private enum ChatWorkflowContextState {
        case active
        case recent
        case preview
    }

    private func workflowContextStateLabel(_ state: ChatWorkflowContextState) -> String {
        switch state {
        case .active:
            return "Active Turn"
        case .recent:
            return "Recent Turn"
        case .preview:
            return "Route Preview"
        }
    }

    private func workflowContextStateSymbol(_ state: ChatWorkflowContextState) -> String {
        switch state {
        case .active:
            return "bolt.horizontal.circle.fill"
        case .recent:
            return "clock.arrow.trianglehead.counterclockwise.rotate.90"
        case .preview:
            return "map"
        }
    }

    private func workflowContextStateTint(_ state: ChatWorkflowContextState) -> Color {
        switch state {
        case .active:
            return .blue
        case .recent:
            return .green
        case .preview:
            return .secondary
        }
    }

    private func responseShapingChannelLabel(_ rawChannel: String) -> String {
        switch rawChannel.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
        case "app":
            return "App Channel"
        case "message":
            return "Message Channel"
        case "voice":
            return "Voice Channel"
        default:
            return "Channel \(rawChannel)"
        }
    }

    private func responseShapingChannelSymbol(_ rawChannel: String) -> String {
        switch rawChannel.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
        case "app":
            return "macwindow"
        case "message":
            return "message"
        case "voice":
            return "waveform"
        default:
            return "point.3.connected.trianglepath.dotted"
        }
    }

    private func responseShapingChannelTint(_ rawChannel: String) -> Color {
        switch rawChannel.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
        case "app":
            return .blue
        case "message":
            return .teal
        case "voice":
            return .purple
        default:
            return .secondary
        }
    }

    private func responseShapingProfileLabel(_ rawProfile: String) -> String {
        switch rawProfile.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
        case "app.default":
            return "Profile App Default"
        case "message.compact":
            return "Profile Message Compact"
        case "voice.spoken":
            return "Profile Voice Spoken"
        default:
            return "Profile \(rawProfile)"
        }
    }

    private func personaPolicySourceLabel(_ rawSource: String) -> String {
        "Persona \(rawSource.capitalized)"
    }

    private func personaPolicySourceSymbol(_ rawSource: String) -> String {
        switch rawSource.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
        case "persisted":
            return "checkmark.circle.fill"
        case "default":
            return "sparkles"
        default:
            return "person.text.rectangle"
        }
    }

    private func personaPolicySourceTint(_ rawSource: String) -> Color {
        switch rawSource.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
        case "persisted":
            return .green
        case "default":
            return .secondary
        default:
            return .secondary
        }
    }

    private func traceabilityDetailRow(
        label: String,
        value: String,
        monospacedValue: Bool = true
    ) -> some View {
        HStack(alignment: .firstTextBaseline, spacing: 8) {
            Text(label)
                .font(.caption.weight(.semibold))
                .foregroundStyle(.secondary)
                .frame(width: 74, alignment: .leading)
            Text(value)
                .font(monospacedValue ? .caption.monospaced() : .caption)
                .foregroundStyle(.secondary)
                .frame(maxWidth: .infinity, alignment: .leading)
                .textSelection(.enabled)
        }
    }

    private var chatEmptyState: some View {
        VStack(alignment: .leading, spacing: 8) {
            Label("Start Conversation", systemImage: "sparkles")
                .font(.headline)
            Text(chatEmptyStateDetail)
                .font(.subheadline)
                .foregroundStyle(.secondary)
            if let status = state.chatStatusMessage {
                Text(status)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            if !state.chatEmptyStateRemediationActions.isEmpty {
                HStack(spacing: 8) {
                    ForEach(state.chatEmptyStateRemediationActions) { action in
                        chatEmptyStateActionButton(action)
                    }
                }
                .padding(.top, 2)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(14)
        .cardSurface(.subtle)
    }

    @ViewBuilder
    private func chatEmptyStateActionButton(_ action: EmptyStateRemediationAction) -> some View {
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

    private var chatEmptyStateDetail: String {
        if !state.localDevTokenConfigured {
            return "Add an Assistant Access Token in Configuration before sending your first message."
        }
        if state.connectionStatus == .degraded {
            return "Connection is degraded. You can still send messages and the app will use fallback behavior if needed."
        }
        if state.connectionStatus == .disconnected {
            return "Start or reconnect the daemon from the menu bar or Configuration, then send your first message."
        }
        return "Use the message box below to send your first message."
    }

    private func nonEmpty(_ value: String?) -> String? {
        guard let value else {
            return nil
        }
        let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? nil : trimmed
    }

    private func providerDisplayName(_ providerID: String) -> String {
        switch providerID.lowercased() {
        case "openai":
            return "OpenAI"
        case "anthropic":
            return "Anthropic"
        case "google":
            return "Google"
        case "ollama":
            return "Ollama"
        default:
            return providerID.capitalized
        }
    }

}
