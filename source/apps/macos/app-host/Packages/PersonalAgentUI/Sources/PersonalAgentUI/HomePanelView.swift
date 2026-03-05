import SwiftUI

struct HomePanelView: View {
    @ObservedObject private var state: AppShellState

    init(state: AppShellState) {
        self.state = state
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: UIStyle.sectionSpacing) {
                TahoeSectionHeader(
                    title: "Home",
                    subtitle: "Start and complete end-to-end workflows from one place."
                ) {
                    EmptyView()
                }

                primaryActionCard
                if prioritizeQuickActions {
                    quickActionsCard
                }
                firstSessionChecklistCard
                if !prioritizeQuickActions {
                    quickActionsCard
                }
                if state.isAdvancedInformationDensityEnabled {
                    funnelDiagnosticsCard
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(UIStyle.panelPadding)
        }
        .background(UIStyle.panelGradient)
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    private var prioritizeQuickActions: Bool {
        state.onboardingReadinessMet && state.homeNextFirstSessionStep == nil
    }

    @ViewBuilder
    private var primaryActionCard: some View {
        if !state.onboardingReadinessMet {
            setupRecoveryCard
        } else if let nextStep = state.homeNextFirstSessionStep {
            firstSessionNextStepCard(nextStep)
        } else {
            firstSessionCompletedCard
        }
    }

    private var setupRecoveryCard: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack(alignment: .firstTextBaseline) {
                Text("Finish Setup")
                    .font(.subheadline.weight(.semibold))
                Spacer(minLength: 0)
                TahoeStatusBadge(
                    text: state.currentSetupBlockerStatus.label,
                    symbolName: state.currentSetupBlockerStatus.symbolName,
                    tint: state.currentSetupBlockerStatus.tint
                )
            }

            Text(state.currentSetupBlockerSummary)
                .font(.caption)
                .foregroundStyle(.secondary)

            HStack(spacing: 8) {
                Button("Fix Next") {
                    state.performOnboardingFixNextStep()
                }
                .buttonStyle(.borderedProminent)
                .controlSize(.small)
                .disabled(state.onboardingFixNextStep == nil || state.onboardingSetupChecksLoading)
                .accessibilityIdentifier("home-primary-action-button")

                if let secondaryAction = state.currentSetupBlockerSecondaryAction {
                    Button(secondaryAction.title) {
                        state.performOnboardingSetupAction(secondaryAction)
                    }
                    .buttonStyle(.bordered)
                    .controlSize(.small)
                    .disabled(!secondaryAction.isEnabled)
                }
            }

            Divider()

            Text("First-Run Trust Steps")
                .font(.caption.weight(.semibold))

            Text(state.distributionTrustGuidanceSummary)
                .font(.caption2)
                .foregroundStyle(.secondary)

            VStack(alignment: .leading, spacing: 3) {
                ForEach(Array(state.distributionTrustGuidanceChecklist.prefix(2).enumerated()), id: \.offset) { entry in
                    Text("\(entry.offset + 1). \(entry.element)")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }
            }

            Text(state.distributionTrustRetryGuidance)
                .font(.caption2)
                .foregroundStyle(.secondary)

            HStack(spacing: 8) {
                Button("Open Security Settings") {
                    state.openDistributionTrustSecuritySettings()
                }
                .buttonStyle(.bordered)
                .controlSize(.small)

                Button("Retry Setup Checks") {
                    state.retrySetupChecksAfterTrustGuidance()
                }
                .buttonStyle(.bordered)
                .controlSize(.small)
                .disabled(state.onboardingSetupChecksLoading)
            }
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .cardSurface(.standard)
    }

    private func firstSessionNextStepCard(_ step: HomeFirstSessionStep) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack(alignment: .firstTextBaseline) {
                Text("Next Best Action")
                    .font(.subheadline.weight(.semibold))
                Spacer(minLength: 0)
                TahoeStatusBadge(
                    text: "In Progress",
                    symbolName: "bolt.fill",
                    tint: .orange
                )
            }

            Text(step.title)
                .font(.callout.weight(.semibold))
            Text(step.detail)
                .font(.caption)
                .foregroundStyle(.secondary)

            Button(step.actionTitle) {
                state.performHomeFirstSessionStep(step.id)
            }
            .buttonStyle(.borderedProminent)
            .controlSize(.small)
            .accessibilityIdentifier("home-primary-action-button")
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .cardSurface(.standard)
    }

    private var firstSessionCompletedCard: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack(alignment: .firstTextBaseline) {
                Text("First Session Complete")
                    .font(.subheadline.weight(.semibold))
                Spacer(minLength: 0)
                TahoeStatusBadge(
                    text: "Complete",
                    symbolName: "checkmark.circle.fill",
                    tint: .green
                )
            }

            Text("Core workflows are ready. Continue in Chat, Tasks, Approvals, or Communications.")
                .font(.caption)
                .foregroundStyle(.secondary)

            Button("Open Chat") {
                state.navigateToSection(.chat)
            }
            .buttonStyle(.borderedProminent)
            .controlSize(.small)
            .accessibilityIdentifier("home-primary-action-button")
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .cardSurface(.standard)
    }

    private var firstSessionChecklistCard: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("Guided First Session")
                .font(.subheadline.weight(.semibold))
            Text(state.homeFirstSessionCompletionSummary)
                .font(.caption)
                .foregroundStyle(.secondary)

            VStack(alignment: .leading, spacing: 8) {
                ForEach(state.homeFirstSessionSteps) { step in
                    HStack(alignment: .top, spacing: 10) {
                        Image(systemName: step.isComplete ? "checkmark.circle.fill" : "circle")
                            .foregroundStyle(step.isComplete ? Color.green : Color.secondary)
                            .padding(.top, 1)
                            .accessibilityHidden(true)

                        VStack(alignment: .leading, spacing: 3) {
                            Text(step.title)
                                .font(.callout.weight(.semibold))
                            Text(step.detail)
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                        Spacer(minLength: 0)
                        if !step.isComplete {
                            Button(step.actionTitle) {
                                state.performHomeFirstSessionStep(step.id)
                            }
                            .buttonStyle(.bordered)
                            .controlSize(.small)
                        } else {
                            TahoeStatusBadge(
                                text: "Done",
                                symbolName: "checkmark.circle.fill",
                                tint: .green
                            )
                        }
                    }
                    .accessibilityIdentifier("home-first-session-step-\(step.id.rawValue)")
                }
            }
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .cardSurface(.standard)
    }

    private var funnelDiagnosticsCard: some View {
        let diagnostics = state.homeFirstSessionFunnelDiagnostics
        return VStack(alignment: .leading, spacing: 10) {
            Text("First-Success Funnel Diagnostics")
                .font(.subheadline.weight(.semibold))
            Text(
                "Workspace \(diagnostics.workspaceID) • \(diagnostics.completedCount) of \(diagnostics.totalCount) completed (\(diagnostics.completionRateLabel))."
            )
            .font(.caption)
            .foregroundStyle(.secondary)

            HStack(spacing: 8) {
                if let firstCompletedAtLabel = diagnostics.firstCompletedAtLabel {
                    Text("First completion: \(firstCompletedAtLabel)")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                } else {
                    Text("First completion: pending")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }
                if let latestCompletedAtLabel = diagnostics.latestCompletedAtLabel {
                    Text("Latest completion: \(latestCompletedAtLabel)")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                } else {
                    Text("Latest completion: pending")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }
            }

            VStack(alignment: .leading, spacing: 8) {
                ForEach(diagnostics.milestones) { milestone in
                    VStack(alignment: .leading, spacing: 2) {
                        HStack(alignment: .firstTextBaseline, spacing: 8) {
                            Text(milestone.title)
                                .font(.callout.weight(.semibold))
                            Spacer(minLength: 0)
                            TahoeStatusBadge(
                                text: milestone.isComplete ? "Completed" : "Pending",
                                symbolName: milestone.isComplete ? "checkmark.circle.fill" : "clock.badge.exclamationmark.fill",
                                tint: milestone.isComplete ? .green : .orange
                            )
                        }
                        if let completedAtLabel = milestone.completedAtLabel {
                            let sourceLabel = milestone.completionSourceLabel ?? "Unknown source"
                            Text("Completed at \(completedAtLabel) • Source: \(sourceLabel)")
                                .font(.caption2)
                                .foregroundStyle(.secondary)
                        } else {
                            Text("No completion evidence captured yet.")
                                .font(.caption2)
                                .foregroundStyle(.secondary)
                        }
                    }
                    .accessibilityIdentifier("home-funnel-milestone-\(milestone.id.rawValue)")
                }
            }
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .cardSurface(.subtle)
        .accessibilityIdentifier("home-funnel-diagnostics")
    }

    private var quickActionsCard: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("Quick Actions")
                .font(.subheadline.weight(.semibold))

            HStack(spacing: 8) {
                Button("Send Message") {
                    state.navigateToSection(.chat)
                }
                .buttonStyle(.borderedProminent)
                .controlSize(.small)

                Button("Create Task") {
                    state.navigateToSection(.tasks)
                }
                .buttonStyle(.bordered)
                .controlSize(.small)

                Button("Review Approvals") {
                    state.navigateToSection(.approvals)
                }
                .buttonStyle(.bordered)
                .controlSize(.small)

                Button("Send Communication") {
                    state.navigateToSection(.communications)
                }
                .buttonStyle(.bordered)
                .controlSize(.small)
            }
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .cardSurface(.subtle)
    }
}
