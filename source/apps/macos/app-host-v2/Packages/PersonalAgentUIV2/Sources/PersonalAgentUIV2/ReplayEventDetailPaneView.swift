import SwiftUI

struct ReplayEventDetailPaneView: View {
    @State private var showSourceContext: Bool = false
    @State private var showDecisionTrace: Bool = false

    let event: ReplayEvent?
    let onApprove: () -> Void
    let onReject: () -> Void
    let onRetry: () -> Void
    let onCompleteRunning: () -> Void
    let onAskWhy: () -> Void
    let globalActionDisabledReason: String?
    let detailEvidence: V2ReplayDetailEvidenceState?
    let onRefreshEvidence: () -> Void
    let onOpenMaintenance: () -> Void

    var body: some View {
        PASurfaceCard("Replay Detail", tone: .neutral) {
            if let event {
                let whatCameIn = detailEvidence?.whatCameIn ?? event.instruction
                let whatAssistantUnderstood = detailEvidence?.whatAssistantUnderstood ?? event.interpretedIntent
                let whatHappened = detailEvidence?.whatHappened ?? event.actionSummary
                let approvalContext = detailEvidence?.approvalContext ?? event.approvalReason
                let failureHint = detailEvidence?.failureHint ?? event.failureRecoveryHint
                let channelsTouched = detailEvidence?.channelsTouched ?? event.channelsTouched
                let confidenceScore = detailEvidence?.confidenceScore ?? event.confidenceScore
                let sourceContextFields = detailEvidence?.sourceContextFields ?? event.sourceContext.fields
                let decisionTrace = detailEvidence?.decisionTrace ?? event.decisionTrace

                VStack(alignment: .leading, spacing: 8) {
                    nextActionCard(for: event)
                    actionRow(event)
                    evidenceBanner()

                    ScrollView {
                        VStack(alignment: .leading, spacing: 10) {
                            detailBlock(title: "What came in", text: whatCameIn)
                            detailBlock(title: "What the assistant understood", text: whatAssistantUnderstood)
                            detailBlock(title: "What happened", text: whatHappened)

                            if let approvalReason = approvalContext {
                                detailBlock(title: "Approval context", text: approvalReason)
                            }

                            if let failureRecoveryHint = failureHint {
                                detailBlock(title: "Recovery hint", text: failureRecoveryHint)
                            }

                            detailBlock(
                                title: "Channels touched",
                                text: channelsTouched.joined(separator: " • ")
                            )

                            detailBlock(title: "Confidence", text: "\(confidenceScore)%")

                            DisclosureGroup(isExpanded: $showSourceContext) {
                                sourceContextCard(sourceContextFields)
                                    .padding(.top, 4)
                            } label: {
                                Text("Source Context")
                                    .font(.paCaption.weight(.semibold))
                                    .foregroundStyle(Color.paTextSecondary)
                            }
                            .paSubsurface()

                            DisclosureGroup(isExpanded: $showDecisionTrace) {
                                decisionTraceCard(decisionTrace)
                                    .padding(.top, 4)
                            } label: {
                                Text("Decision Trace")
                                    .font(.paCaption.weight(.semibold))
                                    .foregroundStyle(Color.paTextSecondary)
                            }
                            .paSubsurface()
                        }
                        .frame(maxWidth: .infinity, alignment: .topLeading)
                    }
                }
            } else {
                ContentUnavailableView(
                    "No replay selected",
                    systemImage: "tray",
                    description: Text("Pick an instruction to inspect its decisions and actions.")
                )
                .frame(maxWidth: .infinity, maxHeight: .infinity)
            }
        }
    }

    private func nextActionCard(for event: ReplayEvent) -> some View {
        let descriptor = nextActionDescriptor(for: event)
        return PASurfaceCard("Next Action", tone: descriptor.cardTone) {
            HStack(spacing: 8) {
                PAStatusChip(label: descriptor.statusLabel, systemImage: descriptor.systemImage, tone: descriptor.tone)
                Text(descriptor.message)
                    .font(.paBody)
                    .foregroundStyle(Color.paTextPrimary)
                    .lineLimit(2)
                    .multilineTextAlignment(.leading)
                    .frame(maxWidth: .infinity, alignment: .leading)
            }
            .controlSize(.small)
        }
    }

    @ViewBuilder
    private func evidenceBanner() -> some View {
        if let detailEvidence {
            switch detailEvidence.phase {
            case .loading:
                HStack(spacing: 8) {
                    ProgressView()
                        .controlSize(.small)
                    Text(detailEvidence.summary ?? "Loading replay evidence…")
                        .font(.paBody)
                        .foregroundStyle(Color.paTextSecondary)
                    Spacer(minLength: 0)
                }
                .padding(8)
                .paSubsurface(.cool)
            case .failed, .empty:
                VStack(alignment: .leading, spacing: 6) {
                    PAInlineBanner(
                        text: detailEvidence.summary ?? "Replay evidence could not be loaded.",
                        tone: detailEvidence.phase == .failed ? .warning : .info
                    )
                    HStack(spacing: 8) {
                        Button("Refresh Evidence") {
                            onRefreshEvidence()
                        }
                        .buttonStyle(.bordered)
                        .tint(.paInfo)
                        .accessibilityIdentifier("v2-replay-refresh-evidence")

                        Button("Open Maintenance") {
                            onOpenMaintenance()
                        }
                        .buttonStyle(.bordered)
                        .tint(.paNeutral)
                        .accessibilityIdentifier("v2-replay-open-maintenance")
                    }
                    .controlSize(.small)
                }
            case .ready:
                if let updatedAt = detailEvidence.lastUpdatedAt {
                    Text("Evidence refreshed \(updatedAt.formatted(date: .omitted, time: .shortened))")
                        .font(.paCaption)
                        .foregroundStyle(Color.paTextTertiary)
                }
            }
        }
    }

    private func sourceContextCard(_ fields: [ReplaySourceContextField]) -> some View {
        PASurfaceCard(tone: .neutral) {
            VStack(alignment: .leading, spacing: 6) {
                ForEach(fields) { field in
                    HStack(alignment: .top) {
                        Text(field.label)
                            .font(.paCaption)
                            .foregroundStyle(Color.paTextSecondary)
                            .frame(width: 110, alignment: .leading)
                        Text(field.value)
                            .font(.paBody)
                            .foregroundStyle(Color.paTextPrimary)
                            .lineLimit(2)
                        Spacer(minLength: 0)
                    }
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private func decisionTraceCard(_ trace: [ReplayDecisionStage]) -> some View {
        PASurfaceCard {
            VStack(alignment: .leading, spacing: 8) {
                ForEach(trace) { stage in
                    HStack(alignment: .top, spacing: 8) {
                        Image(systemName: stage.status.systemImage)
                            .foregroundStyle(stage.status.statusTone.foreground)
                            .padding(.top, 2)

                        VStack(alignment: .leading, spacing: 2) {
                            Text(stage.title)
                                .font(.system(size: 13, weight: .semibold, design: .rounded))
                                .foregroundStyle(Color.paTextPrimary)
                            Text(stage.detail)
                                .font(.paBody)
                                .foregroundStyle(Color.paTextSecondary)
                            PAStatusChip(label: stage.status.label, tone: stage.status.statusTone)
                        }
                    }
                    .paSubsurface()
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private func actionRow(_ event: ReplayEvent) -> some View {
        let disabledReason = globalActionDisabledReason ?? event.inlineApprovalDisabledReason
        return VStack(alignment: .leading, spacing: 6) {
            ViewThatFits(in: .horizontal) {
                HStack(spacing: 8) {
                    primaryActionButton(for: event, disabledReason: disabledReason)
                    secondaryActionButton(for: event)
                    if event.requiresManualApproval {
                        Spacer(minLength: 8)
                        destructiveActionButton(for: event, disabledReason: disabledReason)
                    }
                }
                VStack(alignment: .leading, spacing: 6) {
                    primaryActionButton(for: event, disabledReason: disabledReason)
                    secondaryActionButton(for: event)
                    if event.requiresManualApproval {
                        destructiveActionButton(for: event, disabledReason: disabledReason)
                    }
                }
            }
            .controlSize(.small)

            if let disabledReason {
                Text(disabledReason)
                    .font(.paCaption)
                    .foregroundStyle(Color.paTextSecondary)
            }
        }
        .paSubsurface(.warm)
    }

    @ViewBuilder
    private func primaryActionButton(for event: ReplayEvent, disabledReason: String?) -> some View {
        if event.requiresManualApproval {
            Button("Approve") {
                onApprove()
            }
            .buttonStyle(.borderedProminent)
            .tint(.paSuccess)
            .disabled(disabledReason != nil)
            .accessibilityIdentifier("v2-replay-approve")
        }

        if event.status == .failed && !event.requiresManualApproval {
            Button("Retry Action") {
                onRetry()
            }
            .buttonStyle(.borderedProminent)
            .tint(.paInfo)
            .disabled(disabledReason != nil)
            .accessibilityIdentifier("v2-replay-retry")
        }

        if event.status == .running && !event.requiresManualApproval {
            Button("Stop Run") {
                onCompleteRunning()
            }
            .buttonStyle(.borderedProminent)
            .tint(.paWarning)
            .disabled(disabledReason != nil)
            .accessibilityIdentifier("v2-replay-stop-run")
        }
    }

    @ViewBuilder
    private func secondaryActionButton(for event: ReplayEvent) -> some View {
        Button("Ask Why") {
            onAskWhy()
        }
        .buttonStyle(.bordered)
        .tint(.paInfo)
        .accessibilityIdentifier("v2-replay-ask-why")
    }

    @ViewBuilder
    private func destructiveActionButton(for event: ReplayEvent, disabledReason: String?) -> some View {
        if event.requiresManualApproval {
            Button("Reject", role: .destructive) {
                onReject()
            }
            .buttonStyle(.bordered)
            .tint(.paDanger)
            .disabled(disabledReason != nil)
            .accessibilityIdentifier("v2-replay-reject")
        }
    }

    private func nextActionDescriptor(for event: ReplayEvent) -> NextActionDescriptor {
        switch event.status {
        case .awaitingApproval:
            return NextActionDescriptor(
                statusLabel: "Needs Approval",
                message: "Review and decide here. Approve to execute now, or reject to stop this request.",
                systemImage: "hand.raised.fill",
                tone: .warning,
                cardTone: .warm
            )
        case .failed:
            return NextActionDescriptor(
                statusLabel: "Action Failed",
                message: "Retry after reviewing the recovery hint and confirming the instruction is still valid.",
                systemImage: "exclamationmark.triangle.fill",
                tone: .danger,
                cardTone: .warm
            )
        case .running:
            return NextActionDescriptor(
                statusLabel: "In Progress",
                message: "Stop this run if the instruction is no longer valid or needs correction.",
                systemImage: "pause.circle.fill",
                tone: .warning,
                cardTone: .cool
            )
        case .completed:
            return NextActionDescriptor(
                statusLabel: "Completed",
                message: "Use Ask Why for audit clarity, or select another instruction to continue review.",
                systemImage: "checkmark.circle.fill",
                tone: .success,
                cardTone: .emerald
            )
        }
    }

    private func detailBlock(title: String, text: String) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(title)
                .font(.paCaption)
                .foregroundStyle(Color.paTextSecondary)
            Text(text)
                .font(.paBody)
                .foregroundStyle(Color.paTextPrimary)
        }
        .paSubsurface()
    }
}

private struct NextActionDescriptor {
    var statusLabel: String
    var message: String
    var systemImage: String
    var tone: PAStatusTone
    var cardTone: PACardTone
}
