import SwiftUI

public struct AppShellView: View {
    @ObservedObject private var state: AppShellState
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    public init(state: AppShellState) {
        self.state = state
    }

    public var body: some View {
        NavigationSplitView(columnVisibility: splitViewVisibilityBinding) {
            sidebarList
                .navigationSplitViewColumnWidth(min: 194, ideal: 212, max: 248)
                .safeAreaInset(edge: .bottom) {
                    sidebarRuntimeFooter
                }
        } detail: {
            mainPanelContainer
                .navigationTitle(detailNavigationTitle)
        }
        .navigationSplitViewStyle(.balanced)
        .background(Color(nsColor: .windowBackgroundColor))
        .animation(reduceMotion ? nil : .snappy(duration: 0.2), value: state.isSidebarVisible)
        .animation(reduceMotion ? nil : .snappy(duration: 0.2), value: state.selectedSection)
        .toolbar {
            ToolbarItem(placement: .primaryAction) {
                Button {
                    state.presentDoEntryPoint()
                } label: {
                    Label("Do", systemImage: "sparkles")
                }
                .help("Open Do Entrypoint (Shift-Command-D)")
            }
            ToolbarItem(placement: .primaryAction) {
                Button {
                    state.presentCommandPalette()
                } label: {
                    Label("Commands", systemImage: "command.square")
                }
                .help("Open Command Palette (Shift-Command-P)")
            }
            ToolbarItem(placement: .primaryAction) {
                Button {
                    state.presentNotificationCenter()
                } label: {
                    Label(
                        "Notifications",
                        systemImage: state.unreadNotificationCount > 0 ? "bell.badge.fill" : "bell"
                    )
                }
                .help("Open Notification Center")
            }
            ToolbarItem(placement: .primaryAction) {
                Menu {
                    Picker("Information Density", selection: informationDensityModeBinding) {
                        ForEach(AppInformationDensityMode.allCases, id: \.rawValue) { mode in
                            Text(mode.title).tag(mode)
                        }
                    }
                    .pickerStyle(.inline)
                } label: {
                    Label("Density", systemImage: state.informationDensityMode.symbolName)
                }
                .help("Set information density")
            }
        }
        .sheet(isPresented: $state.isNotificationCenterPresented) {
            NotificationCenterPanelView(state: state)
                .frame(minWidth: 560, minHeight: 420)
        }
        .sheet(isPresented: $state.isCommandPalettePresented) {
            CommandPaletteView(state: state)
                .frame(minWidth: 560, minHeight: 420)
        }
        .overlay(alignment: .topTrailing) {
            NotificationToastStackView(state: state)
                .padding(.top, 10)
                .padding(.trailing, 12)
        }
        .overlay(alignment: .bottomTrailing) {
            UndoActionPromptView(state: state)
                .padding(.trailing, 12)
                .padding(.bottom, 10)
        }
        .alert("Discard Unsaved Changes?", isPresented: $state.showsUnsavedChangesNavigationAlert) {
            Button("Stay", role: .cancel) {
                state.cancelPendingSectionNavigation()
            }
            Button("Discard Changes", role: .destructive) {
                state.discardPendingSectionNavigationChanges()
            }
        } message: {
            Text(
                state.pendingSectionNavigationSummary
                    ?? "Draft edits in this section will be lost if you continue."
            )
        }
        .alert(
            state.pendingHighImpactActionConfirmation?.title ?? "Confirm Action",
            isPresented: highImpactActionAlertBinding,
            presenting: state.pendingHighImpactActionConfirmation
        ) { confirmation in
            Button("Cancel", role: .cancel) {
                state.cancelPendingHighImpactAction()
            }
            if confirmation.isDestructive {
                Button(confirmation.confirmButtonTitle, role: .destructive) {
                    state.confirmPendingHighImpactAction()
                }
            } else {
                Button(confirmation.confirmButtonTitle) {
                    state.confirmPendingHighImpactAction()
                }
            }
        } message: { confirmation in
            Text(confirmation.fullMessage)
        }
    }

    private var splitViewVisibilityBinding: Binding<NavigationSplitViewVisibility> {
        Binding(
            get: { state.isSidebarVisible ? .all : .detailOnly },
            set: { visibility in
                state.isSidebarVisible = visibility != .detailOnly
            }
        )
    }

    private var highImpactActionAlertBinding: Binding<Bool> {
        Binding(
            get: { state.pendingHighImpactActionConfirmation != nil },
            set: { newValue in
                if !newValue {
                    state.cancelPendingHighImpactAction()
                }
            }
        )
    }

    private var informationDensityModeBinding: Binding<AppInformationDensityMode> {
        Binding(
            get: { state.informationDensityMode },
            set: { mode in
                state.setInformationDensityMode(mode)
            }
        )
    }

    private var sidebarSelectionBinding: Binding<AppSection?> {
        Binding(
            get: { state.selectedSection },
            set: { selection in
                guard let selection else {
                    return
                }
                state.requestSectionSelection(selection)
            }
        )
    }

    private var advancedSidebarDisclosureBinding: Binding<Bool> {
        Binding(
            get: { state.isAdvancedSidebarNavigationVisible },
            set: { isExpanded in
                state.setSidebarAdvancedNavigationExpanded(isExpanded)
            }
        )
    }

    private var sidebarList: some View {
        List(selection: sidebarSelectionBinding) {
            Section("Workspace") {
                sidebarNavigationLink(for: .configuration)
            }

            Section("Workflow") {
                ForEach(AppSection.primarySidebarSections) { section in
                    sidebarNavigationLink(for: section)
                }
            }

            Section(isExpanded: advancedSidebarDisclosureBinding) {
                ForEach(AppSection.advancedSidebarSections) { section in
                    sidebarNavigationLink(for: section)
                }
            } header: {
                Button {
                    advancedSidebarDisclosureBinding.wrappedValue.toggle()
                } label: {
                    Text("Advanced")
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .contentShape(Rectangle())
                }
                .buttonStyle(.plain)
                    .accessibilityIdentifier("sidebar-advanced-disclosure")
                    .accessibilityHint("Expands or collapses advanced destinations.")
            }
        }
        .listStyle(.sidebar)
        .accessibilityElement(children: .contain)
        .accessibilityLabel(UIAccessibilityContract.sidebarNavigationLabel)
        .accessibilityHint(UIAccessibilityContract.sidebarNavigationHint)
        .accessibilityIdentifier(UIAccessibilityContract.sidebarNavigationIdentifier)
    }

    private func sidebarNavigationLink(for section: AppSection) -> some View {
        NavigationLink(value: section) {
            sidebarRowLabel(for: section)
        }
        .accessibilityIdentifier("sidebar-section-\(section.rawValue)")
    }

    @ViewBuilder
    private func sidebarRowLabel(for section: AppSection) -> some View {
        let activeFilterCount = state.activeFilterCount(for: section)
        let activeFilterSummary = state.activeFilterSummary(for: section)
        HStack(spacing: 8) {
            Label(section.title, systemImage: section.symbolName)
            Spacer(minLength: 0)
            if activeFilterCount > 0 {
                Text("\(activeFilterCount)")
                    .font(.caption2.monospacedDigit().weight(.semibold))
                    .foregroundStyle(.secondary)
                    .padding(.horizontal, 6)
                    .padding(.vertical, 2)
                    .background(
                        Capsule()
                            .fill(Color.secondary.opacity(0.16))
                    )
                    .help(activeFilterSummary ?? "Active filters are applied.")
                    .accessibilityLabel("\(activeFilterCount) active filters")
                    .accessibilityHint(activeFilterSummary ?? "")
            }
        }
    }

    private var sidebarRuntimeFooter: some View {
        VStack(alignment: .leading, spacing: 8) {
            Divider()
                .padding(.horizontal, 12)

            VStack(alignment: .leading, spacing: 6) {
                Text("Runtime")
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(.secondary)

                if state.isRuntimeStatusBootstrapLoading {
                    Label("Daemon: Checking", systemImage: "clock.arrow.circlepath")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                    Label("App Connection: Checking", systemImage: "clock.arrow.circlepath")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                } else {
                    Label(state.daemonStatus.label, systemImage: state.daemonStatus.symbolName)
                        .font(.caption)
                        .foregroundStyle(state.daemonStatus.tint)

                    Label(state.connectionStatus.label, systemImage: state.connectionStatus.symbolName)
                        .font(.caption)
                        .foregroundStyle(state.connectionStatus.tint)
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(.horizontal, 12)
            .padding(.bottom, 10)
        }
        .background(.regularMaterial)
    }

    @ViewBuilder
    private var mainPanel: some View {
        if state.selectedSectionOnboardingGateApplies {
            OnboardingPanelView(state: state)
        } else {
            switch state.selectedSection {
            case .home:
                HomePanelView(state: state)
            case .chat:
                ChatPanelView(state: state)
            case .communications:
                CommunicationsPanelView(state: state)
            case .configuration:
                ConfigurationPanelView(state: state)
            case .automation:
                AutomationPanelView(state: state)
            case .approvals:
                ApprovalsPanelView(state: state)
            case .tasks:
                TasksPanelView(state: state)
            case .inspect:
                InspectPanelView(state: state)
            case .channels:
                ChannelsPanelView(state: state)
            case .connectors:
                ConnectorsPanelView(state: state)
            case .models:
                ModelsPanelView(state: state)
            }
        }
    }

    private var mainPanelContainer: some View {
        let showsSetupBlockerRibbon =
            state.shouldShowCurrentSetupBlockerRibbon(for: state.selectedSection) &&
            state.selectedSection != .home &&
            state.selectedSection != .configuration &&
            !state.selectedSectionOnboardingGateApplies
        let firstSessionGuidanceContext: HomeFirstSessionGuidanceContext? =
            state.selectedSectionOnboardingGateApplies || state.selectedSection == .home
            ? nil
            : state.homeFirstSessionGuidanceContext
        return VStack(spacing: 0) {
            if showsSetupBlockerRibbon {
                setupBlockerRibbon
                    .padding(.horizontal, UIStyle.panelPadding)
                    .padding(.top, UIStyle.panelPadding)
                    .accessibilityIdentifier("setup-blocker-ribbon")
            }

            if let guidance = firstSessionGuidanceContext {
                firstSessionGuidanceRibbon(guidance)
                    .padding(.horizontal, UIStyle.panelPadding)
                    .padding(.top, showsSetupBlockerRibbon ? 10 : UIStyle.panelPadding)
                    .accessibilityIdentifier("first-session-guidance-ribbon")
            }

            if let drillInContext = state.activeDrillInContextForSelectedSection {
                drillInContextRibbon(drillInContext)
                    .padding(.horizontal, UIStyle.panelPadding)
                    .padding(.top, (showsSetupBlockerRibbon || firstSessionGuidanceContext != nil) ? 10 : UIStyle.panelPadding)
                    .accessibilityIdentifier("drill-in-context-ribbon")
            }

            mainPanel
                .frame(maxWidth: .infinity, maxHeight: .infinity)
                .accessibilityElement(children: .contain)
                .accessibilityLabel(
                    UIAccessibilityContract.panelLandmarkLabel(for: detailNavigationTitle)
                )
                .accessibilityIdentifier("panel-\(state.selectedSection.rawValue)")
        }
        .background(Color(nsColor: .windowBackgroundColor))
    }

    private var detailNavigationTitle: String {
        if state.selectedSectionOnboardingGateApplies {
            return "Setup"
        }
        return state.selectedSection.title
    }

    private var setupBlockerRibbon: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack(alignment: .top, spacing: 10) {
                Image(systemName: state.currentSetupBlockerStatus.symbolName)
                    .foregroundStyle(state.currentSetupBlockerStatus.tint)
                    .padding(.top, 1)
                    .accessibilityHidden(true)

                VStack(alignment: .leading, spacing: 4) {
                    Text("Current Blocker")
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(.secondary)
                    Text(state.currentSetupBlockerTitle)
                        .font(.callout.weight(.semibold))
                    Text(state.currentSetupBlockerSummary)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }

                Spacer(minLength: 0)

                TahoeStatusBadge(
                    text: state.currentSetupBlockerStatus.label,
                    symbolName: state.currentSetupBlockerStatus.symbolName,
                    tint: state.currentSetupBlockerStatus.tint
                )
            }

            HStack(spacing: 8) {
                Button("Fix Next") {
                    state.performOnboardingFixNextStep()
                }
                .buttonStyle(.borderedProminent)
                .controlSize(.small)
                .disabled(state.onboardingFixNextStep == nil || state.onboardingSetupChecksLoading)
                .accessibilityIdentifier("setup-blocker-fix-next")

                if let secondaryAction = state.currentSetupBlockerSecondaryAction {
                    Button(secondaryAction.title) {
                        state.performOnboardingSetupAction(secondaryAction)
                    }
                    .buttonStyle(.bordered)
                    .controlSize(.small)
                    .disabled(!secondaryAction.isEnabled)
                    .accessibilityIdentifier("setup-blocker-secondary-action")
                }
            }
        }
        .padding(10)
        .frame(maxWidth: .infinity, alignment: .leading)
        .cardSurface(.subtle)
    }

    private func firstSessionGuidanceRibbon(_ context: HomeFirstSessionGuidanceContext) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack(alignment: .top, spacing: 10) {
                Image(systemName: "arrow.triangle.branch")
                    .foregroundStyle(.orange)
                    .padding(.top, 1)
                    .accessibilityHidden(true)

                VStack(alignment: .leading, spacing: 4) {
                    Text("Guided First Session")
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(.secondary)
                    Text(context.step.title)
                        .font(.callout.weight(.semibold))
                    Text(
                        context.isCurrentSectionDestination
                            ? "Complete this milestone in \(context.step.destinationSection.title). The next guided action unlocks automatically."
                            : context.step.detail
                    )
                    .font(.caption)
                    .foregroundStyle(.secondary)
                }

                Spacer(minLength: 0)

                TahoeStatusBadge(
                    text: context.progressLabel,
                    symbolName: "list.number",
                    tint: .orange
                )
            }

            HStack(spacing: 8) {
                if context.isCurrentSectionDestination {
                    Label("Current guided step", systemImage: "checkmark.seal")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                } else {
                    Button(context.step.actionTitle) {
                        state.performHomeFirstSessionGuidancePrimaryAction()
                    }
                    .buttonStyle(.borderedProminent)
                    .controlSize(.small)
                    .accessibilityIdentifier("first-session-guidance-primary-action")
                }

                Button("Open Home Checklist") {
                    state.openHomeFirstSessionChecklist()
                }
                .buttonStyle(.bordered)
                .controlSize(.small)
                .accessibilityIdentifier("first-session-guidance-open-home")
            }
        }
        .padding(10)
        .cardSurface(.subtle)
    }

    private func drillInContextRibbon(_ context: DrillInNavigationContext) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack(spacing: 8) {
                Label("Opened from \(context.sourceSection.title)", systemImage: "arrow.turn.down.right")
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(.secondary)
                Spacer(minLength: 0)
                Button("Back to \(context.sourceSection.title)") {
                    state.returnToDrillInSourceSection()
                }
                .buttonStyle(.bordered)
                .controlSize(.small)
                Button {
                    state.clearActiveDrillInNavigationContext()
                } label: {
                    Image(systemName: "xmark.circle.fill")
                        .foregroundStyle(.secondary)
                }
                .buttonStyle(.plain)
                .help("Dismiss drill-in context")
                .accessibilityLabel(UIAccessibilityContract.drillInDismissLabel)
                .accessibilityHint(UIAccessibilityContract.drillInDismissHint)
                .accessibilityIdentifier("drill-in-context-dismiss-button")
            }

            if !context.chips.isEmpty {
                ScrollView(.horizontal, showsIndicators: false) {
                    HStack(spacing: 6) {
                        ForEach(context.chips, id: \.self) { chip in
                            Text(chip)
                                .font(.caption2)
                                .foregroundStyle(.secondary)
                                .padding(.horizontal, 8)
                                .padding(.vertical, 4)
                                .background(
                                    Capsule()
                                        .fill(Color.secondary.opacity(0.16))
                                )
                        }
                    }
                }
            }
        }
        .padding(10)
        .cardSurface(.subtle)
    }
}
