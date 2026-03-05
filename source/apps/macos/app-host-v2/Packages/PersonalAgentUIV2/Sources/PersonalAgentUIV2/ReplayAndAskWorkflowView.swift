import SwiftUI

struct ReplayAndAskWorkflowView: View {
    @ObservedObject var store: AppShellV2Store
    @State private var showAskComposer: Bool = false

    var body: some View {
        VStack(alignment: .leading, spacing: V2WorkflowLayout.panelSpacing) {
            headerRow
            if store.shouldShowSetupBlockerRibbon, let blocker = store.currentSetupBlocker {
                SetupBlockerBannerView(
                    blocker: blocker,
                    onFixNext: {
                        store.fixNextSetupBlocker()
                    },
                    onOpenGetStarted: {
                        store.selectedSection = .getStarted
                    }
                )
            }
            ReplayControlsCardView(store: store)
            activitySurface
            if let feedback = store.lastFeedback {
                feedbackBanner(feedback)
            }
        }
        .onAppear {
            store.seedAskFromSelectedEventIfEmpty()
            store.startReplayRealtimeIfNeeded()
            Task {
                await store.refreshReplayFeedIfNeeded()
                await store.refreshSelectedReplayDetailEvidenceIfNeeded()
            }
        }
        .onDisappear {
            Task {
                await store.stopReplayRealtimeStream()
            }
        }
        .onChange(of: store.selectedEventID) { _, _ in
            store.seedAskFromSelectedEventIfEmpty()
            Task {
                await store.refreshSelectedReplayDetailEvidenceIfNeeded()
            }
        }
    }

    private var headerRow: some View {
        ViewThatFits(in: .horizontal) {
            HStack(alignment: .top, spacing: 10) {
                PASectionHeader(
                    title: "Replay first. Ask second.",
                    subtitle: "Inspect what arrived, how it was interpreted, which constraints fired, and what action actually ran."
                )
                Spacer(minLength: 8)
                trustPulseStrip
            }

            VStack(alignment: .leading, spacing: 6) {
                PASectionHeader(
                    title: "Replay first. Ask second.",
                    subtitle: "Inspect what arrived, how it was interpreted, which constraints fired, and what action actually ran."
                )
                trustPulseStrip
            }
        }
    }

    private var trustPulseStrip: some View {
        ViewThatFits(in: .horizontal) {
            HStack(spacing: 5) {
                trustPulseButton(
                    label: "Needs Attention",
                    value: "\(store.metrics.waiting)",
                    tone: .warning,
                    isActive: store.statusFilter == .waiting || store.statusFilter == .needsApproval || store.statusFilter == .running
                ) {
                    store.statusFilter = .waiting
                }
                trustPulseButton(
                    label: "Failed",
                    value: "\(store.metrics.atRisk)",
                    tone: .danger,
                    isActive: store.statusFilter == .failed
                ) {
                    store.statusFilter = .failed
                }
                trustPulseButton(
                    label: "Automated Safely",
                    value: "\(store.metrics.automatedSafely)",
                    tone: .success,
                    isActive: store.statusFilter == .completed
                ) {
                    store.statusFilter = .completed
                }
            }
            .frame(maxWidth: .infinity, alignment: .trailing)
            .fixedSize(horizontal: true, vertical: false)

            HStack(spacing: 5) {
                trustPulseButton(
                    label: "Needs Attention",
                    value: "\(store.metrics.waiting)",
                    tone: .warning,
                    isActive: store.statusFilter == .waiting || store.statusFilter == .needsApproval || store.statusFilter == .running
                ) {
                    store.statusFilter = .waiting
                }
                trustPulseButton(
                    label: "Failed",
                    value: "\(store.metrics.atRisk)",
                    tone: .danger,
                    isActive: store.statusFilter == .failed
                ) {
                    store.statusFilter = .failed
                }
            }
            .frame(maxWidth: .infinity, alignment: .trailing)
            .fixedSize(horizontal: true, vertical: false)
        }
        .frame(maxWidth: .infinity, alignment: .trailing)
    }

    private var activitySurface: some View {
        VStack(alignment: .leading, spacing: V2WorkflowLayout.sectionSpacing) {
            V2PanelStateBannerView(
                state: store.panelLifecycleState(for: .replayAndAsk),
                onAction: { actionID in
                    store.performPanelStateAction(actionID, workflow: .replayAndAsk)
                }
            )

            if store.filteredEvents.isEmpty {
                filteredEmptyState
            } else {
                activityInspectorLayout
            }

            askComposer
        }
    }

    private var filteredEmptyState: some View {
        PASurfaceCard("Replay", tone: .neutral) {
            ContentUnavailableView(
                "No matches",
                systemImage: "line.3.horizontal.decrease.circle",
                description: Text("Filters are too strict. Clear filters to see incoming instructions.")
            )
            .frame(maxWidth: .infinity, minHeight: 120)

            HStack {
                Spacer(minLength: 0)
                Button("Clear Filters") {
                    store.clearFilters()
                }
                .buttonStyle(.borderedProminent)
                .tint(.paInfo)
                Spacer(minLength: 0)
            }
            .padding(.bottom, 2)
        }
    }

    private var askComposer: some View {
        PASurfaceCard("Ask", tone: .neutral) {
            Group {
                if showAskComposer {
                    VStack(alignment: .leading, spacing: 6) {
                        Text("Ask from context. Keep wording specific so the assistant can answer with evidence.")
                            .font(.paCaption)
                            .foregroundStyle(Color.paTextSecondary)

                        if askSendLifecycle.phase == .inFlight {
                            HStack(spacing: 6) {
                                ProgressView()
                                    .controlSize(.small)
                                Text(askSendLifecycle.message ?? "Sending question…")
                                    .font(.paCaption)
                                    .foregroundStyle(Color.paTextSecondary)
                            }
                        } else if let message = askSendLifecycle.message,
                                  askSendLifecycle.phase == .failed || askSendLifecycle.phase == .disabled {
                            Text(message)
                                .font(.paCaption)
                                .foregroundStyle(Color.paWarning)
                        }

                        TextField("Example: Why did you request approval for this action?", text: $store.askDraft, axis: .vertical)
                            .textFieldStyle(.plain)
                            .paInputSurface()
                            .lineLimit(1...2)
                            .disabled(askSendLifecycle.isInFlight)
                            .accessibilityLabel("Ask Replay Question")
                            .accessibilityHint("Ask a question about the selected replay item.")
                            .accessibilityIdentifier("v2-replay-ask-field")

                        HStack {
                            Button("Reseed from Selection") {
                                store.seedAskFromSelectedEvent()
                            }
                            .buttonStyle(.bordered)
                            .tint(.paInfo)

                            Spacer(minLength: 8)

                            Button("Collapse") {
                                showAskComposer = false
                            }
                            .buttonStyle(.bordered)

                            Button("Send Question") {
                                store.sendAsk()
                            }
                            .buttonStyle(.borderedProminent)
                            .tint(.paInfo)
                            .keyboardShortcut(.return, modifiers: [.command])
                            .disabled(
                                store.askDraft.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
                                    || askSendLifecycle.isInFlight
                                    || askSendLifecycle.isDisabled
                            )
                            .accessibilityLabel("Send Question")
                            .accessibilityHint("Submits this question to the assistant with replay context.")
                            .accessibilityIdentifier("v2-replay-send-question")
                        }
                        .controlSize(.small)
                    }
                } else {
                    HStack(spacing: 8) {
                        Text(store.askDraft)
                            .font(.paCaption)
                            .foregroundStyle(Color.paTextSecondary)
                            .lineLimit(1)
                            .frame(maxWidth: .infinity, alignment: .leading)

                        Button("Open Ask") {
                            showAskComposer = true
                        }
                        .buttonStyle(.borderedProminent)
                        .tint(.paInfo)
                        .controlSize(.small)
                        .accessibilityIdentifier("v2-replay-open-ask")
                    }
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private func trustPulseButton(
        label: String,
        value: String,
        tone: PAStatusTone,
        isActive: Bool,
        action: @escaping () -> Void
    ) -> some View {
        Button(action: action) {
            ReplayMetricChip(label: label, value: value, tone: tone, isActive: isActive)
        }
        .buttonStyle(.plain)
        .accessibilityLabel("\(label): \(value)")
        .accessibilityHint("Applies the \(label) replay status filter.")
    }

    private func feedbackBanner(_ feedback: String) -> some View {
        HStack(spacing: 8) {
            Image(systemName: "checkmark.seal")
                .foregroundStyle(Color.paSuccess)
            Text(feedback)
                .font(.paBody)
            Spacer(minLength: 8)
            Button("Dismiss") {
                store.dismissFeedback()
            }
            .buttonStyle(.borderless)
            .foregroundStyle(Color.paTextSecondary)
            .accessibilityLabel("Dismiss Feedback Banner")
        }
        .padding(8)
        .background(Color.paSuccess.opacity(0.13), in: RoundedRectangle(cornerRadius: PATokens.radiusMD, style: .continuous))
        .overlay(
            RoundedRectangle(cornerRadius: PATokens.radiusMD, style: .continuous)
                .stroke(Color.paSuccess.opacity(0.3), lineWidth: 1)
        )
    }

    private var activityInspectorLayout: some View {
        ViewThatFits(in: .horizontal) {
            HSplitView {
                activityListPane

                activityDetailPane
                    .frame(minWidth: 330)
                    .padding(.leading, 6)
            }
            .frame(minHeight: 280)

            VStack(spacing: 8) {
                activityListPane
                    .frame(maxWidth: .infinity, minHeight: 220, maxHeight: 360)

                activityDetailPane
                    .frame(maxWidth: .infinity, minHeight: 300)
            }
            .frame(minHeight: 540)
        }
    }

    private var activityListPane: some View {
        ReplayEventListPaneView(
            events: store.filteredEvents,
            selectedEventID: Binding(
                get: { store.selectedEventID },
                set: { store.selectEvent($0) }
            ),
            canLoadMore: store.replayFeedQueryState.canLoadMore,
            isLoadingMore: store.replayFeedQueryState.isLoadingMore,
            onLoadMore: {
                Task {
                    await store.loadMoreReplayFeed()
                }
            }
        )
    }

    private var activityDetailPane: some View {
        ReplayEventDetailPaneView(
            event: store.selectedEvent,
            onApprove: {
                store.approveSelectedEvent()
            },
            onReject: {
                store.rejectSelectedEvent()
            },
            onRetry: {
                store.retrySelectedEvent()
            },
            onCompleteRunning: {
                store.completeSelectedRunningEvent()
            },
            onAskWhy: {
                store.seedAskFromSelectedEvent()
            },
            globalActionDisabledReason: replayActionDisabledReason,
            detailEvidence: store.replayDetailEvidence(for: store.selectedEvent),
            onRefreshEvidence: {
                Task {
                    await store.refreshSelectedReplayDetailEvidence(force: true)
                }
            },
            onOpenMaintenance: {
                store.openReplayMaintenanceFromDetail()
            }
        )
    }

    private var replayActionDisabledReason: String? {
        if let event = store.selectedEvent {
            switch event.status {
            case .awaitingApproval:
                if event.daemonLocator?.approvalRequestID?.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty != false {
                    return "Missing approval reference. Refresh Replay to sync this instruction before deciding."
                }
            case .failed, .running:
                if event.daemonLocator?.taskID?.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty != false {
                    return "Missing task reference. Refresh Replay to sync run controls for this instruction."
                }
            case .completed:
                break
            }
        }

        let approve = store.mutationLifecycle(for: .replayApprove)
        let reject = store.mutationLifecycle(for: .replayReject)
        let retry = store.mutationLifecycle(for: .replayRetry)
        let complete = store.mutationLifecycle(for: .replayComplete)

        if approve.isInFlight || reject.isInFlight || retry.isInFlight || complete.isInFlight {
            return "Replay mutation in progress."
        }
        if approve.isDisabled {
            return approve.message
        }
        return nil
    }

    private var askSendLifecycle: V2MutationLifecycleState {
        store.mutationLifecycle(for: .askSend)
    }

}

private struct ReplayMetricChip: View {
    let label: String
    let value: String
    let tone: PAStatusTone
    var isActive: Bool = false

    var body: some View {
        HStack(spacing: 4) {
            Circle()
                .fill(tone.foreground)
                .frame(width: 5, height: 5)
            Text(value)
                .font(.system(size: 11, weight: .bold, design: .rounded))
                .foregroundStyle(Color.paTextPrimary)
            Text(label)
                .font(.system(size: 10, weight: .medium, design: .rounded))
                .foregroundStyle(Color.paTextSecondary)
        }
        .padding(.horizontal, 7)
        .padding(.vertical, 4)
        .background(
            Capsule(style: .continuous)
                .fill(.thinMaterial)
                .overlay(
                    Capsule(style: .continuous)
                        .fill(tone.background.opacity(0.7))
                )
        )
        .overlay(
            Capsule(style: .continuous)
                .stroke(isActive ? Color.paInfo.opacity(0.75) : tone.stroke, lineWidth: isActive ? 1.5 : 1)
        )
        .shadow(color: isActive ? Color.paInfo.opacity(0.35) : .clear, radius: 8, x: 0, y: 0)
    }
}
