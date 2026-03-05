import SwiftUI

struct OnboardingPanelView: View {
    @ObservedObject private var state: AppShellState
    @State private var isChecklistExpanded = false

    init(state: AppShellState) {
        self.state = state
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: UIStyle.sectionSpacing) {
                TahoeSectionHeader(
                    title: "Finish Setup",
                    subtitle: "Follow one step at a time to unlock full workflows."
                ) {
                    EmptyView()
                }

                setupWizardCard
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(UIStyle.panelPadding)
        }
        .background(UIStyle.panelGradient)
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    private var setupWizardCard: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("Setup Wizard")
                .font(.subheadline.weight(.semibold))

            VStack(alignment: .leading, spacing: 6) {
                ProgressView(value: state.onboardingSetupProgressFraction)
                    .progressViewStyle(.linear)
                Text("\(state.onboardingSetupCompletedCount) of \(state.onboardingSetupTotalCount) checks ready")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            if let currentStep = state.onboardingCurrentWizardStep {
                currentStepCard(currentStep)
            } else {
                setupCompleteCard
            }

            DisclosureGroup("All Setup Checks", isExpanded: $isChecklistExpanded) {
                VStack(alignment: .leading, spacing: 8) {
                    ForEach(state.onboardingSetupSteps) { step in
                        setupStepRow(step: step)
                    }
                }
                .padding(.top, 4)
            }
            .font(.caption.weight(.semibold))

            Text(state.onboardingStatusMessage)
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .cardSurface(.standard)
    }

    private func currentStepCard(_ step: OnboardingSetupStep) -> some View {
        GroupBox("Current Step") {
            VStack(alignment: .leading, spacing: 10) {
                HStack(alignment: .firstTextBaseline, spacing: 8) {
                    Label(step.title, systemImage: step.status.symbolName)
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(step.status.tint)
                    Spacer(minLength: 0)
                    TahoeStatusBadge(
                        text: step.status.label,
                        symbolName: step.status.symbolName,
                        tint: step.status.tint
                    )
                }

                Text(step.detail)
                    .font(.caption)
                    .foregroundStyle(.secondary)

                if let nextStep = state.onboardingNextWizardStep {
                    Text("Up next: \(nextStep.title)")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }

                HStack(spacing: 8) {
                    if let action = step.remediationAction {
                        Button(action.title) {
                            state.performOnboardingSetupAction(action)
                        }
                        .buttonStyle(.borderedProminent)
                        .disabled(!action.isEnabled)

                        if let actionDetail = action.detail {
                            Text(actionDetail)
                                .font(.caption2)
                                .foregroundStyle(.secondary)
                        }
                    }

                    Button("Refresh Checks") {
                        state.refreshOnboardingReadiness()
                    }
                    .buttonStyle(.bordered)
                    .disabled(state.onboardingSetupChecksLoading)
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private var setupCompleteCard: some View {
        GroupBox("Current Step") {
            VStack(alignment: .leading, spacing: 10) {
                Label("Setup Complete", systemImage: "checkmark.circle.fill")
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(.green)

                Text("Your workspace is ready for chat, tasks, approvals, and automation workflows.")
                    .font(.caption)
                    .foregroundStyle(.secondary)

                HStack(spacing: 8) {
                    Button("Open Chat") {
                        state.navigateToSection(.chat)
                    }
                    .buttonStyle(.borderedProminent)

                    Button("Open Communications") {
                        state.navigateToSection(.communications)
                    }
                    .buttonStyle(.bordered)

                    Button("Open Configuration") {
                        state.navigateToSection(.configuration)
                    }
                    .buttonStyle(.bordered)
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private func setupStepRow(step: OnboardingSetupStep) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(alignment: .top, spacing: 10) {
                Image(systemName: step.status.symbolName)
                    .foregroundStyle(step.status.tint)
                    .padding(.top, 1)

                VStack(alignment: .leading, spacing: 3) {
                    HStack(alignment: .firstTextBaseline, spacing: 8) {
                        Text(step.title)
                            .font(.callout.weight(.semibold))
                        Label(step.status.label, systemImage: step.status.symbolName)
                            .font(.caption2.weight(.semibold))
                            .foregroundStyle(step.status.tint)
                            .labelStyle(.titleAndIcon)
                    }
                    Text(step.detail)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }

                Spacer(minLength: 0)
            }

            if let action = step.remediationAction, step.status.isBlocked {
                Button(action.title) {
                    state.performOnboardingSetupAction(action)
                }
                .buttonStyle(.bordered)
                .disabled(!action.isEnabled)
            }
        }
    }
}
