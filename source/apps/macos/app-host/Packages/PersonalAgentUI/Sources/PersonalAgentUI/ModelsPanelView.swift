import SwiftUI

struct ModelsPanelView: View {
    private struct ModelsQuickstartAction {
        let title: String
        let detail: String?
        let isEnabled: Bool
        let perform: () -> Void
    }

    private struct ModelsQuickstartStep: Identifiable {
        let id: String
        let title: String
        let detail: String
        let status: OnboardingSetupStepStatus
        let action: ModelsQuickstartAction?
    }

    @ObservedObject private var state: AppShellState
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @State private var collapsedProviderIDs: Set<String> = []
    @State private var knownProviderIDs: Set<String> = []
    @State private var isQuickstartChecklistExpanded = false
    @State private var routeTaskClassSelection = "chat"
    @State private var routeProviderSelection = ""
    @State private var routeModelSelection = ""

    init(state: AppShellState) {
        self.state = state
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: UIStyle.sectionSpacing) {
                header
                if let runtimeBannerMessage {
                    RuntimeStateBanner(message: runtimeBannerMessage)
                }
                if let panelProblemRemediation {
                    PanelProblemRemediationCardView(context: panelProblemRemediation) { actionID in
                        state.performPanelProblemRemediationAction(actionID, section: .models)
                    }
                }
                quickstartCard
                providerSectionHeader
                providerCards
                routeSummaryCard
                if state.modelRouteReadinessNeedsAttention {
                    routeReadinessChecklistCard
                }
                routePolicyEditorCard
                if isAdvancedInformationDensityEnabled {
                    routeSimulationCard
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(UIStyle.panelPadding)
        }
        .background(UIStyle.panelGradient)
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .onAppear {
            synchronizeProviderCardCollapsedState()
            synchronizeRoutePolicyEditorSelection()
            synchronizeRouteSimulationSelection()
        }
        .onChange(of: state.providerReadinessItems.map(\.id)) { _, _ in
            synchronizeProviderCardCollapsedState()
        }
        .onChange(of: state.modelCatalogItems.map(\.id)) { _, _ in
            synchronizeRoutePolicyEditorSelection()
            synchronizeRouteSimulationSelection()
        }
        .onChange(of: state.modelPolicyItems.map(\.id)) { _, _ in
            synchronizeRoutePolicyEditorSelection()
            synchronizeRouteSimulationSelection()
        }
        .onChange(of: routeTaskClassSelection) { _, _ in
            synchronizeRoutePolicyEditorSelection()
        }
        .onChange(of: routeProviderSelection) { _, _ in
            synchronizeRoutePolicyEditorSelection()
        }
        .onChange(of: state.selectedPrincipal) { _, _ in
            synchronizeRouteSimulationSelection()
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
        state.panelProblemRemediation(for: .models)
    }

    private var isAdvancedInformationDensityEnabled: Bool {
        state.isAdvancedInformationDensityEnabled
    }

    private var header: some View {
        TahoeSectionHeader(
            title: "Models",
            subtitle: "Provider readiness, model route resolution, catalog, and policy visibility"
        ) {
            EmptyView()
        }
    }

    private var quickstartCard: some View {
        let steps = quickstartSteps
        let currentStep = quickstartCurrentStep(from: steps)
        let completedCount = steps.filter { $0.status.isComplete }.count
        let progressFraction = steps.isEmpty ? 0.0 : Double(completedCount) / Double(steps.count)

        return VStack(alignment: .leading, spacing: 10) {
            Text("Models Quickstart")
                .font(.subheadline.weight(.semibold))

            VStack(alignment: .leading, spacing: 6) {
                ProgressView(value: progressFraction)
                    .progressViewStyle(.linear)
                Text("\(completedCount) of \(steps.count) quickstart steps ready")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            if let currentStep {
                GroupBox("Current Step") {
                    VStack(alignment: .leading, spacing: 8) {
                        HStack(alignment: .firstTextBaseline, spacing: 8) {
                            Label(currentStep.title, systemImage: currentStep.status.symbolName)
                                .font(.caption.weight(.semibold))
                                .foregroundStyle(currentStep.status.tint)
                            Spacer(minLength: 0)
                            TahoeStatusBadge(
                                text: currentStep.status.label,
                                symbolName: currentStep.status.symbolName,
                                tint: currentStep.status.tint
                            )
                            .controlSize(.small)
                        }

                        Text(currentStep.detail)
                            .font(.caption2)
                            .foregroundStyle(.secondary)

                        if let action = currentStep.action {
                            HStack(alignment: .firstTextBaseline, spacing: 8) {
                                Button(action.title) {
                                    action.perform()
                                }
                                .buttonStyle(.borderedProminent)
                                .disabled(!action.isEnabled)

                                if let detail = action.detail {
                                    Text(detail)
                                        .font(.caption2)
                                        .foregroundStyle(.secondary)
                                }
                            }
                        }
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                }
            }

            DisclosureGroup("All Quickstart Steps", isExpanded: $isQuickstartChecklistExpanded) {
                VStack(alignment: .leading, spacing: 8) {
                    ForEach(steps) { step in
                        quickstartStepRow(step)
                    }
                }
                .padding(.top, 4)
            }
            .font(.caption.weight(.semibold))

            Text("Use quickstart for first route setup. Advanced provider/catalog/policy controls remain below.")
                .font(.caption2)
                .foregroundStyle(.secondary)
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .cardSurface(.standard)
    }

    private func quickstartStepRow(_ step: ModelsQuickstartStep) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(alignment: .top, spacing: 8) {
                Image(systemName: step.status.symbolName)
                    .foregroundStyle(step.status.tint)
                    .padding(.top, 1)

                VStack(alignment: .leading, spacing: 2) {
                    Text(step.title)
                        .font(.caption.weight(.semibold))
                    Text(step.detail)
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }

                Spacer(minLength: 0)

                TahoeStatusBadge(
                    text: step.status.label,
                    symbolName: step.status.symbolName,
                    tint: step.status.tint
                )
                .controlSize(.small)
            }

            if let action = step.action {
                Button(action.title) {
                    action.perform()
                }
                .buttonStyle(.bordered)
                .controlSize(.small)
                .disabled(!action.isEnabled)
            }
        }
    }

    private var providerSectionHeader: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Text("Provider + Model Readiness")
                    .font(.subheadline.weight(.semibold))
                Spacer()
                if state.isProviderStatusLoading {
                    ProgressView()
                        .controlSize(.small)
                }
            }

            if let providerStatusMessage = state.providerStatusMessage {
                Text(providerStatusMessage)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            if let modelCatalogStatusMessage = state.modelCatalogStatusMessage {
                Text(modelCatalogStatusMessage)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            HStack(spacing: 8) {
                if state.modelsHasUnsavedDraftChanges {
                    Text("Unsaved changes")
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(.orange)
                }

                Button("Discard All") {
                    state.discardDraftChanges(for: .models)
                }
                .buttonStyle(.bordered)
                .disabled(!state.modelsHasUnsavedDraftChanges || isModelBulkMutationInFlight)

                Button("Save All") {
                    state.saveAllDraftChanges(for: .models)
                }
                .buttonStyle(.borderedProminent)
                .successSymbolEffect(
                    state.successNotificationPulse(for: "models"),
                    reduceMotion: reduceMotion
                )
                .disabled(!state.modelsHasUnsavedDraftChanges || isModelBulkMutationInFlight)

                Button {
                    state.refreshProviderInventory()
                } label: {
                    Label("Refresh Inventory", systemImage: "arrow.clockwise")
                }
                .quietButtonChrome()
                .disabled(state.isProviderStatusLoading)

                Button {
                    state.runProviderConnectivityChecks()
                } label: {
                    Label("Run Checks", systemImage: "checkmark.seal")
                }
                .quietButtonChrome()
                .disabled(state.isProviderStatusLoading)
            }
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .cardSurface(.standard)
    }

    private var routeSummaryCard: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("Chat Model Route")
                .font(.caption.weight(.semibold))
                .foregroundStyle(.secondary)

            if let route = state.modelRouteSummary {
                HStack(spacing: 8) {
                    TahoeStatusBadge(
                        text: "\(providerDisplayName(route.provider)) • \(route.modelKey)",
                        symbolName: "sparkles",
                        tint: .blue
                    )
                    if isAdvancedInformationDensityEnabled {
                        Text("Source: \(route.source)")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                }
                if let notes = route.notes, !notes.isEmpty {
                    Text(notes)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            } else {
                Text(state.modelRouteStatusMessage ?? "Route not resolved.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
        .padding(10)
        .frame(maxWidth: .infinity, alignment: .leading)
        .cardSurface(.subtle)
    }

    private var routeReadinessChecklistCard: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                Text("Route Readiness Checklist")
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(.secondary)
                if state.modelRouteReadinessBlockerCount > 0 {
                    TahoeStatusBadge(
                        text: "\(state.modelRouteReadinessBlockerCount) blocker\(state.modelRouteReadinessBlockerCount == 1 ? "" : "s")",
                        symbolName: "exclamationmark.triangle.fill",
                        tint: .orange
                    )
                } else {
                    TahoeStatusBadge(
                        text: "Checking",
                        symbolName: "clock.arrow.circlepath",
                        tint: .secondary
                    )
                }
                Spacer(minLength: 0)
            }

            Text("Resolve these checks to make chat route selection deterministic.")
                .font(.caption2)
                .foregroundStyle(.secondary)

            ForEach(state.modelRouteReadinessChecklistSteps) { step in
                HStack(alignment: .top, spacing: 10) {
                    TahoeStatusBadge(
                        text: step.status.label,
                        symbolName: step.status.symbolName,
                        tint: step.status.tint
                    )
                    .controlSize(.small)

                    VStack(alignment: .leading, spacing: 3) {
                        Text(step.title)
                            .font(.caption.weight(.semibold))
                        Text(step.detail)
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                    }

                    Spacer(minLength: 0)

                    if !step.status.isComplete, let action = step.remediationAction {
                        Button(action.title) {
                            state.performOnboardingSetupAction(action)
                        }
                        .buttonStyle(.bordered)
                        .controlSize(.small)
                        .disabled(!action.isEnabled)
                    }
                }
                .padding(.vertical, 2)
            }
        }
        .padding(10)
        .frame(maxWidth: .infinity, alignment: .leading)
        .cardSurface(.subtle)
    }

    private var routePolicyEditorCard: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("Route Policy Editor")
                .font(.caption.weight(.semibold))
                .foregroundStyle(.secondary)

            LabeledContent("Task Class") {
                Picker("Task Class", selection: $routeTaskClassSelection) {
                    ForEach(routeTaskClassOptions, id: \.self) { taskClass in
                        Text(taskClass).tag(taskClass)
                    }
                }
                .pickerStyle(.menu)
                .frame(maxWidth: 220)
                .disabled(routeTaskClassOptions.isEmpty || state.isModelRoutePolicySaveInFlight)
            }

            LabeledContent("Provider") {
                Picker("Provider", selection: $routeProviderSelection) {
                    ForEach(routeProviderOptions, id: \.self) { providerID in
                        Text(providerDisplayName(providerID)).tag(providerID)
                    }
                }
                .pickerStyle(.menu)
                .frame(maxWidth: 220)
                .disabled(routeProviderOptions.isEmpty || state.isModelRoutePolicySaveInFlight)
            }

            LabeledContent("Model") {
                Picker("Model", selection: $routeModelSelection) {
                    ForEach(routeModelOptions, id: \.self) { modelKey in
                        Text(modelKey).tag(modelKey)
                    }
                }
                .pickerStyle(.menu)
                .frame(maxWidth: 320)
                .disabled(routeModelOptions.isEmpty || state.isModelRoutePolicySaveInFlight)
            }

            HStack(spacing: 8) {
                Button("Save Route Policy") {
                    state.requestSaveModelRoutePolicy(
                        taskClass: routeTaskClassSelection,
                        providerID: routeProviderSelection,
                        modelKey: routeModelSelection
                    )
                }
                .buttonStyle(.borderedProminent)
                .successSymbolEffect(
                    state.successNotificationPulse(for: "models"),
                    reduceMotion: reduceMotion
                )
                .disabled(
                    routeTaskClassSelection.isEmpty ||
                    routeProviderSelection.isEmpty ||
                    routeModelSelection.isEmpty ||
                    state.isModelRoutePolicySaveInFlight
                )

                if state.isModelRoutePolicySaveInFlight {
                    ProgressView()
                        .controlSize(.small)
                }
            }

            if let statusMessage = state.modelRoutePolicySaveStatusMessage {
                Text(statusMessage)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            if routeProviderOptions.isEmpty {
                Text("No model catalog entries are available yet. Refresh inventory after provider setup to enable route selection.")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }
        }
        .padding(10)
        .frame(maxWidth: .infinity, alignment: .leading)
        .cardSurface(.subtle)
    }

    private var routeSimulationCard: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("Route Simulation + Explainability")
                .font(.caption.weight(.semibold))
                .foregroundStyle(.secondary)

            Text("Test task-class route outcomes with optional principal context, then inspect reason-coded decisions and fallback-chain ranking.")
                .font(.caption2)
                .foregroundStyle(.secondary)

            LabeledContent("Task Class") {
                Picker("Task Class", selection: simulationTaskClassBinding) {
                    ForEach(simulationTaskClassOptions, id: \.self) { taskClass in
                        Text(taskClass).tag(taskClass)
                    }
                }
                .pickerStyle(.menu)
                .frame(maxWidth: 220)
                .disabled(simulationTaskClassOptions.isEmpty || state.isModelRouteSimulationInFlight || state.isModelRouteExplainInFlight)
            }

            LabeledContent("Principal Actor ID") {
                TextField("Optional principal actor id", text: $state.modelRouteSimulationPrincipalActorID)
                    .textFieldStyle(.roundedBorder)
                    .frame(maxWidth: 320)
                    .disabled(state.isModelRouteSimulationInFlight || state.isModelRouteExplainInFlight)
            }

            HStack(spacing: 8) {
                Button("Use Active Principal") {
                    state.applyActivePrincipalToModelRouteSimulation()
                }
                .buttonStyle(.bordered)
                .disabled(state.isModelRouteSimulationInFlight || state.isModelRouteExplainInFlight)

                Menu("Pick Principal") {
                    ForEach(state.modelRouteSimulationPrincipalOptions, id: \.self) { principal in
                        Button(principal) {
                            state.modelRouteSimulationPrincipalActorID = principal
                        }
                    }
                }
                .disabled(
                    state.modelRouteSimulationPrincipalOptions.isEmpty ||
                    state.isModelRouteSimulationInFlight ||
                    state.isModelRouteExplainInFlight
                )

                Button("Clear Principal") {
                    state.clearModelRouteSimulationPrincipal()
                }
                .buttonStyle(.bordered)
                .disabled(
                    state.modelRouteSimulationPrincipalActorID.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ||
                    state.isModelRouteSimulationInFlight ||
                    state.isModelRouteExplainInFlight
                )
            }

            HStack(spacing: 8) {
                Button("Simulate Route") {
                    state.runModelRouteSimulation()
                }
                .buttonStyle(.borderedProminent)
                .disabled(
                    simulationTaskClassOptions.isEmpty ||
                    state.modelRouteSimulationTaskClass.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ||
                    state.isModelRouteSimulationInFlight ||
                    state.isModelRouteExplainInFlight
                )

                Button("Explain Route") {
                    state.runModelRouteExplainability()
                }
                .buttonStyle(.bordered)
                .disabled(
                    simulationTaskClassOptions.isEmpty ||
                    state.modelRouteSimulationTaskClass.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ||
                    state.isModelRouteSimulationInFlight ||
                    state.isModelRouteExplainInFlight
                )

                Button("Reset Output") {
                    state.resetModelRouteSimulationOutputs()
                }
                .buttonStyle(.bordered)
                .disabled(state.modelRouteSimulationResult == nil && state.modelRouteExplainResult == nil)

                if state.isModelRouteSimulationInFlight || state.isModelRouteExplainInFlight {
                    ProgressView()
                        .controlSize(.small)
                }
            }

            if let statusMessage = state.modelRouteSimulationStatusMessage {
                Text(statusMessage)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            if let routeSelection = routeAnalysisSelectionContext {
                routeSelectionSummary(
                    taskClass: routeSelection.taskClass,
                    principalActorID: routeSelection.principalActorID,
                    provider: routeSelection.provider,
                    modelKey: routeSelection.modelKey,
                    source: routeSelection.source
                )
            }

            GroupBox("Simulation Result") {
                if let result = state.modelRouteSimulationResult {
                    routeSimulationResultBlock(result)
                } else {
                    Text("Run `Simulate Route` to inspect selected route, reason codes, and fallback chain.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }

            if let statusMessage = state.modelRouteExplainStatusMessage {
                Text(statusMessage)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            GroupBox("Explainability") {
                if let result = state.modelRouteExplainResult {
                    routeExplainResultBlock(result)
                } else {
                    Text("Run `Explain Route` to load summary guidance and decision rationale.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }
        }
        .padding(10)
        .frame(maxWidth: .infinity, alignment: .leading)
        .cardSurface(.subtle)
    }

    @ViewBuilder
    private var providerCards: some View {
        if showProviderLoadingSkeleton {
            PanelLoadingSkeletonView(
                title: "Loading Providers",
                subtitle: "Fetching provider readiness, catalog entries, and route policy context.",
                rowCount: 3
            )
            .frame(maxWidth: .infinity, alignment: .leading)
        } else if state.providerReadinessItems.isEmpty {
            PanelRemediationEmptyStateView(
                title: "No Providers Found",
                systemImage: "cube.box",
                description: "Provider readiness appears here after daemon model/provider inventory loads.",
                statusMessage: state.providerStatusMessage,
                headerStatusMessage: state.providerStatusMessage,
                actions: state.modelsEmptyStateRemediationActions
            ) { actionID in
                state.performEmptyStateRemediationAction(actionID)
            }
        } else {
            LazyVStack(alignment: .leading, spacing: UIStyle.standardSpacing) {
                ForEach(state.providerReadinessItems) { provider in
                    providerCard(provider)
                }
            }
        }
    }

    private var showProviderLoadingSkeleton: Bool {
        (state.isProviderStatusLoading || !state.hasLoadedProviderStatus) && state.providerReadinessItems.isEmpty
    }

    private func providerCard(_ provider: ProviderReadinessItem) -> some View {
        GroupBox {
            DisclosureGroup(isExpanded: expansionBinding(for: provider.id)) {
                VStack(alignment: .leading, spacing: 10) {
                    Divider()

                    providerSetupBlock(provider: provider)

                    Text(provider.detail)
                        .font(.caption)
                        .foregroundStyle(.secondary)

                    Text("Updated: \(provider.updatedAtLabel)")
                        .font(.caption2)
                        .foregroundStyle(.secondary)

                    providerModelCatalogBlock(providerID: provider.provider)
                    providerPolicyBlock(providerID: provider.provider)
                }
                .padding(.top, 8)
            } label: {
                HStack(alignment: .firstTextBaseline, spacing: 10) {
                    VStack(alignment: .leading, spacing: 3) {
                        Text(providerDisplayName(provider.provider))
                            .font(.headline)
                        Text(providerCardSummary(providerID: provider.provider))
                            .font(.caption)
                            .foregroundStyle(.secondary)
                        if state.providerSetupHasDraftChanges(providerID: provider.provider) {
                            Text("Unsaved setup draft")
                                .font(.caption2.weight(.semibold))
                                .foregroundStyle(.orange)
                        }
                    }

                    Spacer(minLength: 0)

                    TahoeStatusBadge(
                        text: provider.status.label,
                        symbolName: provider.status.symbolName,
                        tint: provider.status.tint
                    )
                }
            }
        }
        .groupBoxStyle(.automatic)
    }

    private func providerSetupBlock(provider: ProviderReadinessItem) -> some View {
        let normalizedProviderID = provider.provider.lowercased()
        let isSetupInFlight = state.providerSetupInFlightIDs.contains(normalizedProviderID)
        let isCheckInFlight = state.providerCheckInFlightIDs.contains(normalizedProviderID)

        return VStack(alignment: .leading, spacing: 8) {
            Text("Provider Setup")
                .font(.caption.weight(.semibold))
                .foregroundStyle(.secondary)

            LabeledContent("Endpoint") {
                TextField("Endpoint", text: endpointBinding(for: provider.provider))
                    .textFieldStyle(.roundedBorder)
                    .frame(maxWidth: 360)
            }

            if providerRequiresAPIKey(provider.provider) {
                LabeledContent("Secret Name") {
                    TextField("API key secret name", text: secretNameBinding(for: provider.provider))
                        .textFieldStyle(.roundedBorder)
                        .frame(maxWidth: 260)
                }

                LabeledContent("API Key") {
                    SecureField("Enter API key (write-only)", text: secretValueBinding(for: provider.provider))
                        .textFieldStyle(.roundedBorder)
                        .frame(maxWidth: 360)
                }

                Text("API key values are stored in local Keychain and never rendered after save.")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            } else {
                Text("This provider does not require an API key by default.")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }

            HStack(spacing: 8) {
                Button("Save Provider") {
                    state.saveProviderSetup(for: provider.provider)
                }
                .buttonStyle(.borderedProminent)
                .successSymbolEffect(
                    state.successNotificationPulse(for: "models"),
                    reduceMotion: reduceMotion
                )
                .disabled(isSetupInFlight || isCheckInFlight)

                Button("Check") {
                    state.runProviderConnectivityCheck(for: provider.provider)
                }
                .buttonStyle(.bordered)
                .disabled(isSetupInFlight || isCheckInFlight)

                Button("Reset Endpoint") {
                    state.resetProviderEndpointDraft(for: provider.provider)
                }
                .buttonStyle(.bordered)
                .disabled(isSetupInFlight || isCheckInFlight)

                if isSetupInFlight || isCheckInFlight {
                    ProgressView()
                        .controlSize(.small)
                }
            }

            if let statusMessage = state.providerSetupStatusByID[normalizedProviderID] {
                Text(statusMessage)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            } else {
                Text("Current endpoint: \(provider.endpoint)")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }
        }
        .padding(10)
        .cardSurface(.subtle)
    }

    private var isModelBulkMutationInFlight: Bool {
        state.isProviderStatusLoading ||
            !state.providerSetupInFlightIDs.isEmpty ||
            !state.providerCheckInFlightIDs.isEmpty
    }

    @ViewBuilder
    private func providerModelCatalogBlock(providerID: String) -> some View {
        let normalizedProviderID = normalizedProviderID(providerID)
        let models = modelsForProvider(providerID)
        let discoveredModels = discoveredModelsForProvider(providerID)
        let isDiscoverInFlight = state.modelCatalogDiscoverInFlightProviderIDs.contains(normalizedProviderID)
        let isManageInFlight = state.modelCatalogManageInFlightProviderIDs.contains(normalizedProviderID)

        VStack(alignment: .leading, spacing: 6) {
            HStack(spacing: 8) {
                Text("Model Catalog")
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(.secondary)
                Spacer(minLength: 0)
                Button(isDiscoverInFlight ? "Discovering…" : "Discover") {
                    state.discoverModels(for: providerID)
                }
                .buttonStyle(.bordered)
                .disabled(isDiscoverInFlight || isManageInFlight)
                if isDiscoverInFlight {
                    ProgressView()
                        .controlSize(.small)
                }
            }

            HStack(spacing: 8) {
                TextField("Add model key", text: manualAddBinding(for: providerID))
                    .textFieldStyle(.roundedBorder)
                    .frame(maxWidth: 260)

                Button("Add Model") {
                    state.addModelToCatalog(
                        providerID: providerID,
                        modelKey: state.modelManualAddDraft(for: providerID),
                        enabled: false
                    )
                }
                .buttonStyle(.bordered)
                .disabled(
                    isManageInFlight ||
                    isDiscoverInFlight ||
                    state.modelManualAddDraft(for: providerID).trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
                )

                if isManageInFlight {
                    ProgressView()
                        .controlSize(.small)
                }
            }

            if let managementStatus = state.modelCatalogManagementStatusByProviderID[normalizedProviderID] {
                Text(managementStatus)
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }

            if models.isEmpty {
                Text("No model catalog entries for \(providerDisplayName(providerID)).")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            } else {
                ForEach(models) { model in
                    providerModelRow(model, isManageInFlight: isManageInFlight)
                }
            }

            if !discoveredModels.isEmpty {
                Divider()
                Text("Discovered")
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(.secondary)
                ForEach(discoveredModels) { model in
                    discoveredModelRow(model, isManageInFlight: isManageInFlight || isDiscoverInFlight)
                }
            }
        }
        .padding(10)
        .cardSurface(.subtle)
    }

    @ViewBuilder
    private func providerPolicyBlock(providerID: String) -> some View {
        let policies = policiesForProvider(providerID)

        VStack(alignment: .leading, spacing: 6) {
            Text("Routing Policies")
                .font(.caption.weight(.semibold))
                .foregroundStyle(.secondary)

            if policies.isEmpty {
                Text("No routing policies for \(providerDisplayName(providerID)).")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            } else {
                ForEach(policies) { policy in
                    providerPolicyRow(policy)
                }
            }
        }
        .padding(10)
        .cardSurface(.subtle)
    }

    private func providerModelRow(_ item: ModelCatalogEntryItem, isManageInFlight: Bool) -> some View {
        let isMutationInFlight = state.modelMutationInFlightIDs.contains(item.id)
        let isBusy = isMutationInFlight || isManageInFlight
        let isCurrentChatRoute = modelIsCurrentChatRoute(item)

        return VStack(alignment: .leading, spacing: 6) {
            HStack(spacing: 8) {
                Text(item.modelKey)
                    .font(.callout.weight(.semibold))
                Spacer(minLength: 0)
                TahoeStatusBadge(
                    text: item.enabled ? "Enabled" : "Disabled",
                    symbolName: item.enabled ? "checkmark.circle.fill" : "pause.circle.fill",
                    tint: item.enabled ? .green : .secondary
                )
                TahoeStatusBadge(
                    text: item.providerReady ? "Provider Ready" : "Provider Not Ready",
                    symbolName: item.providerReady ? "checkmark.seal.fill" : "exclamationmark.triangle.fill",
                    tint: item.providerReady ? .green : .orange
                )
                if isCurrentChatRoute {
                    TahoeStatusBadge(
                        text: "Chat Route",
                        symbolName: "sparkles",
                        tint: .blue
                    )
                }
            }

            HStack(spacing: 8) {
                Button(item.enabled ? "Disable" : "Enable") {
                    state.setModelEnabled(
                        providerID: item.provider,
                        modelKey: item.modelKey,
                        enabled: !item.enabled
                    )
                }
                .buttonStyle(.bordered)
                .disabled(isBusy)

                Button(isCurrentChatRoute ? "Chat Route Set" : "Set as Chat Route") {
                    state.setModelAsChatRoute(
                        providerID: item.provider,
                        modelKey: item.modelKey
                    )
                }
                .buttonStyle(.borderedProminent)
                .disabled(isBusy || state.isModelRoutePolicySaveInFlight)

                Button("Remove") {
                    state.removeModelFromCatalog(providerID: item.provider, modelKey: item.modelKey)
                }
                .buttonStyle(.bordered)
                .tint(.red)
                .disabled(isBusy)

                if isMutationInFlight || state.isModelRoutePolicySaveInFlight {
                    ProgressView()
                        .controlSize(.small)
                }
            }

            if let status = state.modelMutationStatusByID[item.id] {
                Text(status)
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }

            if isAdvancedInformationDensityEnabled {
                Text(item.providerEndpoint)
                    .font(.caption2)
                    .foregroundStyle(.secondary)
                    .lineLimit(2)
                    .textSelection(.enabled)
            }
        }
        .padding(10)
        .frame(maxWidth: .infinity, alignment: .leading)
        .cardSurface(.standard)
    }

    private func discoveredModelRow(_ item: DiscoveredModelEntryItem, isManageInFlight: Bool) -> some View {
        HStack(alignment: .top, spacing: 8) {
            VStack(alignment: .leading, spacing: 2) {
                Text(item.displayName)
                    .font(.caption.weight(.semibold))
                if item.displayName.caseInsensitiveCompare(item.modelKey) != .orderedSame {
                    Text(item.modelKey)
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }
                if isAdvancedInformationDensityEnabled {
                    Text("Source: \(item.source)")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }
            }
            Spacer(minLength: 0)
            if item.inCatalog {
                TahoeStatusBadge(
                    text: item.enabled ? "In Catalog • Enabled" : "In Catalog",
                    symbolName: item.enabled ? "checkmark.circle.fill" : "checkmark.circle",
                    tint: item.enabled ? .green : .secondary
                )
            } else {
                Button("Add to Catalog") {
                    state.addModelToCatalog(providerID: item.provider, modelKey: item.modelKey, enabled: false)
                }
                .buttonStyle(.bordered)
                .disabled(isManageInFlight)
            }
        }
        .padding(8)
        .frame(maxWidth: .infinity, alignment: .leading)
        .cardSurface(.standard)
    }

    private func providerPolicyRow(_ item: ModelPolicyItem) -> some View {
        HStack(alignment: .top, spacing: 8) {
            Text(item.taskClass)
                .font(.caption.weight(.semibold))
                .foregroundStyle(.secondary)
                .frame(width: 106, alignment: .leading)
            VStack(alignment: .leading, spacing: 2) {
                Text(item.modelKey)
                    .font(.caption)
                Text("Updated: \(item.updatedAtLabel)")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }
            Spacer(minLength: 0)
        }
    }

    private func routeSimulationResultBlock(_ result: ModelRouteSimulationResultItem) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            if let notes = result.notes, !notes.isEmpty {
                Text(notes)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            routeReasonCodesView(result.reasonCodes)
            routeDecisionTraceList(result.decisions)
            routeFallbackChainList(result.fallbackChain)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    private func routeExplainResultBlock(_ result: ModelRouteExplainResultItem) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(result.summary)
                .font(.caption)
                .foregroundStyle(.secondary)

            if !result.explanations.isEmpty {
                VStack(alignment: .leading, spacing: 4) {
                    Text("Explanations")
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(.secondary)
                    ForEach(Array(result.explanations.enumerated()), id: \.offset) { _, explanation in
                        Text("• \(explanation)")
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                    }
                }
            }

            routeReasonCodesView(result.reasonCodes)
            routeDecisionTraceList(result.decisions)
            routeFallbackChainList(result.fallbackChain)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    private func routeSelectionSummary(
        taskClass: String,
        principalActorID: String?,
        provider: String,
        modelKey: String,
        source: String
    ) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            HStack(spacing: 8) {
                TahoeStatusBadge(
                    text: "\(providerDisplayName(provider)) • \(modelKey)",
                    symbolName: "sparkles",
                    tint: .blue
                )
                if isAdvancedInformationDensityEnabled {
                    Text("Source: \(source)")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }
            }
            Text(
                "Task class: \(taskClass) • Principal: \(principalActorID ?? "auto")"
            )
            .font(.caption2)
            .foregroundStyle(.secondary)
        }
    }

    private var routeAnalysisSelectionContext: (
        taskClass: String,
        principalActorID: String?,
        provider: String,
        modelKey: String,
        source: String
    )? {
        if let result = state.modelRouteSimulationResult {
            return (
                taskClass: result.taskClass,
                principalActorID: result.principalActorID,
                provider: result.selectedProvider,
                modelKey: result.selectedModelKey,
                source: result.selectedSource
            )
        }
        if let result = state.modelRouteExplainResult {
            return (
                taskClass: result.taskClass,
                principalActorID: result.principalActorID,
                provider: result.selectedProvider,
                modelKey: result.selectedModelKey,
                source: result.selectedSource
            )
        }
        return nil
    }

    private func routeReasonCodesView(_ reasonCodes: [String]) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Text("Reason Codes")
                .font(.caption.weight(.semibold))
                .foregroundStyle(.secondary)
            if reasonCodes.isEmpty {
                Text("No reason codes returned.")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            } else {
                Text(reasonCodes.joined(separator: ", "))
                    .font(.caption2)
                    .foregroundStyle(.secondary)
                    .textSelection(.enabled)
            }
        }
    }

    private func routeDecisionTraceList(_ decisions: [ModelRouteDecisionTraceItem]) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Text("Decision Trace (\(decisions.count))")
                .font(.caption.weight(.semibold))
                .foregroundStyle(.secondary)
            if decisions.isEmpty {
                Text("No decision trace rows returned.")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            } else {
                ForEach(decisions) { decision in
                    VStack(alignment: .leading, spacing: 2) {
                        Text("\(decision.step) • \(decision.decision) • \(decision.reasonCode)")
                            .font(.caption2.weight(.semibold))
                        if let provider = decision.provider, let modelKey = decision.modelKey {
                            Text("\(providerDisplayName(provider)) • \(modelKey)")
                                .font(.caption2)
                                .foregroundStyle(.secondary)
                        }
                        if let note = decision.note, !note.isEmpty {
                            Text(note)
                                .font(.caption2)
                                .foregroundStyle(.secondary)
                        }
                    }
                    .padding(6)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .cardSurface(.standard)
                }
            }
        }
    }

    private func routeFallbackChainList(_ fallbackChain: [ModelRouteFallbackTraceItem]) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Text("Fallback Chain (\(fallbackChain.count))")
                .font(.caption.weight(.semibold))
                .foregroundStyle(.secondary)
            if fallbackChain.isEmpty {
                Text("No fallback candidates returned.")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            } else {
                ForEach(fallbackChain) { candidate in
                    HStack(alignment: .firstTextBaseline, spacing: 8) {
                        Text("#\(candidate.rank)")
                            .font(.caption2.weight(.semibold))
                            .foregroundStyle(.secondary)
                            .frame(width: 26, alignment: .leading)
                        Text("\(providerDisplayName(candidate.provider)) • \(candidate.modelKey)")
                            .font(.caption2.weight(candidate.selected ? .semibold : .regular))
                        Spacer(minLength: 0)
                        TahoeStatusBadge(
                            text: candidate.selected ? "Selected" : candidate.reasonCode,
                            symbolName: candidate.selected ? "checkmark.circle.fill" : "arrow.triangle.branch",
                            tint: candidate.selected ? .green : .secondary
                        )
                        .controlSize(.small)
                    }
                    .padding(6)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .cardSurface(.standard)
                }
            }
        }
    }

    private var quickstartSteps: [ModelsQuickstartStep] {
        let checklistByID = Dictionary(uniqueKeysWithValues: state.modelRouteReadinessChecklistSteps.map { ($0.id, $0) })
        let prerequisiteStep = [checklistByID["token"], checklistByID["daemon"]]
            .compactMap { $0 }
            .first { !$0.status.isComplete }
        let providerChecklistStep = checklistByID["provider"]
        let modelChecklistStep = checklistByID["model_catalog"]
        let routeChecklistStep = checklistByID["chat_route"]
        let preferredProvider = quickstartPreferredProvider
        let preferredProviderID = preferredProvider?.provider ?? ""
        let preferredProviderLabel = preferredProvider.map { providerDisplayName($0.provider) } ?? "provider"
        let preferredModel = preferredProvider.flatMap { quickstartPreferredModel(for: $0.provider) }
        let preferredModelKey = preferredModel?.modelKey ?? ""
        let sendMessageMilestoneComplete = state.homeFirstSessionSteps.first(where: { $0.id == .sendMessage })?.isComplete ?? false

        let connectProviderStep: ModelsQuickstartStep
        if let prerequisiteStep {
            connectProviderStep = ModelsQuickstartStep(
                id: "connect_provider",
                title: "Connect Provider",
                detail: prerequisiteStep.detail,
                status: prerequisiteStep.status,
                action: quickstartAction(from: prerequisiteStep.remediationAction)
            )
        } else if let providerChecklistStep, providerChecklistStep.status == .loading {
            connectProviderStep = ModelsQuickstartStep(
                id: "connect_provider",
                title: "Connect Provider",
                detail: providerChecklistStep.detail,
                status: .loading,
                action: nil
            )
        } else if let preferredProvider, providerIsQuickstartReady(preferredProvider) {
            connectProviderStep = ModelsQuickstartStep(
                id: "connect_provider",
                title: "Connect Provider",
                detail: "\(preferredProviderLabel) is configured and ready.",
                status: .complete,
                action: nil
            )
        } else if !preferredProviderID.isEmpty {
            connectProviderStep = ModelsQuickstartStep(
                id: "connect_provider",
                title: "Connect Provider",
                detail: providerChecklistStep?.detail ?? "Save endpoint and credentials for \(preferredProviderLabel).",
                status: .blocked,
                action: ModelsQuickstartAction(
                    title: "Save + Check \(preferredProviderLabel)",
                    detail: "Saves provider setup and runs one readiness check.",
                    isEnabled: !isProviderSetupOrCheckInFlight(providerID: preferredProviderID)
                ) {
                    state.runProviderQuickstartSaveAndCheck(providerID: preferredProviderID)
                }
            )
        } else {
            connectProviderStep = ModelsQuickstartStep(
                id: "connect_provider",
                title: "Connect Provider",
                detail: providerChecklistStep?.detail ?? "Refresh provider inventory to start quickstart.",
                status: state.isProviderStatusLoading ? .loading : .blocked,
                action: ModelsQuickstartAction(
                    title: "Refresh Inventory",
                    detail: nil,
                    isEnabled: !state.isProviderStatusLoading
                ) {
                    state.refreshProviderInventory()
                }
            )
        }

        let chooseModelStep: ModelsQuickstartStep
        if connectProviderStep.status != .complete {
            chooseModelStep = ModelsQuickstartStep(
                id: "choose_model",
                title: "Choose and Enable Model",
                detail: "Connect a provider before selecting a chat model.",
                status: connectProviderStep.status == .loading ? .loading : .blocked,
                action: connectProviderStep.action
            )
        } else if let provider = preferredProvider {
            let providerModels = modelsForProvider(provider.provider)
            if providerModels.isEmpty {
                let discoverDetail = modelChecklistStep?.detail
                    ?? "No catalog models found for \(preferredProviderLabel). Discover or add one model."
                let discoverAction = ModelsQuickstartAction(
                    title: "Discover Models",
                    detail: "Loads provider-backed model inventory for \(preferredProviderLabel).",
                    isEnabled: !isProviderModelCatalogMutationInFlight(providerID: provider.provider)
                ) {
                    state.discoverModels(for: provider.provider)
                }
                chooseModelStep = ModelsQuickstartStep(
                    id: "choose_model",
                    title: "Choose and Enable Model",
                    detail: discoverDetail,
                    status: state.isProviderStatusLoading ? .loading : .blocked,
                    action: discoverAction
                )
            } else if let enabledModel = providerModels.first(where: \.enabled) {
                let enabledCount = providerModels.filter(\.enabled).count
                let detail = enabledCount > 1
                    ? "\(enabledCount) enabled models available. \(enabledModel.modelKey) is selected for quickstart."
                    : "\(enabledModel.modelKey) is enabled and ready for routing."
                chooseModelStep = ModelsQuickstartStep(
                    id: "choose_model",
                    title: "Choose and Enable Model",
                    detail: detail,
                    status: .complete,
                    action: nil
                )
            } else {
                let candidateModel = preferredModel ?? providerModels[0]
                chooseModelStep = ModelsQuickstartStep(
                    id: "choose_model",
                    title: "Choose and Enable Model",
                    detail: "Enable \(candidateModel.modelKey) for \(preferredProviderLabel) to make it routeable.",
                    status: .blocked,
                    action: ModelsQuickstartAction(
                        title: "Enable \(candidateModel.modelKey)",
                        detail: nil,
                        isEnabled: !state.modelMutationInFlightIDs.contains(candidateModel.id) &&
                            !isProviderModelCatalogMutationInFlight(providerID: provider.provider)
                    ) {
                        state.setModelEnabled(
                            providerID: provider.provider,
                            modelKey: candidateModel.modelKey,
                            enabled: true
                        )
                    }
                )
            }
        } else {
            chooseModelStep = ModelsQuickstartStep(
                id: "choose_model",
                title: "Choose and Enable Model",
                detail: "Provider inventory is unavailable. Refresh provider inventory to continue.",
                status: .blocked,
                action: ModelsQuickstartAction(
                    title: "Refresh Inventory",
                    detail: nil,
                    isEnabled: !state.isProviderStatusLoading
                ) {
                    state.refreshProviderInventory()
                }
            )
        }

        let setRouteStep: ModelsQuickstartStep
        if chooseModelStep.status != .complete {
            setRouteStep = ModelsQuickstartStep(
                id: "set_route",
                title: "Set Chat Route",
                detail: "Choose and enable a model before setting chat route.",
                status: chooseModelStep.status == .loading ? .loading : .blocked,
                action: chooseModelStep.action
            )
        } else if !preferredProviderID.isEmpty && !preferredModelKey.isEmpty && modelRouteMatches(
            providerID: preferredProviderID,
            modelKey: preferredModelKey
        ) {
            setRouteStep = ModelsQuickstartStep(
                id: "set_route",
                title: "Set Chat Route",
                detail: "Chat route points to \(preferredProviderLabel) • \(preferredModelKey).",
                status: .complete,
                action: nil
            )
        } else if let routeChecklistStep, routeChecklistStep.status == .loading {
            setRouteStep = ModelsQuickstartStep(
                id: "set_route",
                title: "Set Chat Route",
                detail: routeChecklistStep.detail,
                status: .loading,
                action: nil
            )
        } else if !preferredProviderID.isEmpty && !preferredModelKey.isEmpty {
            setRouteStep = ModelsQuickstartStep(
                id: "set_route",
                title: "Set Chat Route",
                detail: routeChecklistStep?.detail ?? "Set chat route to \(preferredProviderLabel) • \(preferredModelKey).",
                status: .blocked,
                action: ModelsQuickstartAction(
                    title: "Set Route to \(preferredModelKey)",
                    detail: nil,
                    isEnabled: !state.isModelRoutePolicySaveInFlight
                ) {
                    state.setModelAsChatRoute(
                        providerID: preferredProviderID,
                        modelKey: preferredModelKey
                    )
                }
            )
        } else {
            setRouteStep = ModelsQuickstartStep(
                id: "set_route",
                title: "Set Chat Route",
                detail: routeChecklistStep?.detail ?? "Select an enabled model before setting chat route.",
                status: .blocked,
                action: chooseModelStep.action
            )
        }

        let testInChatStep: ModelsQuickstartStep
        if setRouteStep.status != .complete {
            testInChatStep = ModelsQuickstartStep(
                id: "test_chat",
                title: "Test in Chat",
                detail: "Set chat route before validating with a message.",
                status: setRouteStep.status == .loading ? .loading : .blocked,
                action: setRouteStep.action
            )
        } else if sendMessageMilestoneComplete {
            testInChatStep = ModelsQuickstartStep(
                id: "test_chat",
                title: "Test in Chat",
                detail: "Chat quickstart test is complete for this workspace.",
                status: .complete,
                action: nil
            )
        } else {
            testInChatStep = ModelsQuickstartStep(
                id: "test_chat",
                title: "Test in Chat",
                detail: "Open Chat with a starter prompt and send one message to finish setup.",
                status: .blocked,
                action: ModelsQuickstartAction(
                    title: "Open Chat Test",
                    detail: "Seeds a quickstart prompt for \(preferredProviderLabel) • \(preferredModelKey).",
                    isEnabled: !preferredProviderID.isEmpty && !preferredModelKey.isEmpty
                ) {
                    state.openChatForModelsQuickstartTest(
                        providerID: preferredProviderID,
                        modelKey: preferredModelKey
                    )
                }
            )
        }

        return [
            connectProviderStep,
            chooseModelStep,
            setRouteStep,
            testInChatStep
        ]
    }

    private func quickstartCurrentStep(from steps: [ModelsQuickstartStep]) -> ModelsQuickstartStep? {
        steps.first(where: { !$0.status.isComplete }) ?? steps.last
    }

    private func quickstartAction(from onboardingAction: OnboardingSetupAction?) -> ModelsQuickstartAction? {
        guard let onboardingAction else {
            return nil
        }
        return ModelsQuickstartAction(
            title: onboardingAction.title,
            detail: onboardingAction.detail,
            isEnabled: onboardingAction.isEnabled
        ) {
            state.performOnboardingSetupAction(onboardingAction)
        }
    }

    private var quickstartPreferredProvider: ProviderReadinessItem? {
        if let routeProviderID = state.modelRouteSummary?.provider,
           let item = providerReadinessItem(for: routeProviderID) {
            return item
        }
        if let item = providerReadinessItem(for: routeProviderSelection) {
            return item
        }
        if let needsSetup = state.providerReadinessItems.first(where: { !providerIsQuickstartReady($0) }) {
            return needsSetup
        }
        return state.providerReadinessItems.first
    }

    private func quickstartPreferredModel(for providerID: String) -> ModelCatalogEntryItem? {
        let providerModels = modelsForProvider(providerID)
        guard !providerModels.isEmpty else {
            return nil
        }
        if let routeSummary = state.modelRouteSummary,
           routeSummary.provider.caseInsensitiveCompare(providerID) == .orderedSame,
           let model = providerModels.first(where: {
               $0.modelKey.caseInsensitiveCompare(routeSummary.modelKey) == .orderedSame
           }) {
            return model
        }
        if let selectedModelKey = canonicalOption(routeModelSelection, within: providerModels.map(\.modelKey)),
           let selected = providerModels.first(where: {
               $0.modelKey.caseInsensitiveCompare(selectedModelKey) == .orderedSame
           }) {
            return selected
        }
        if let enabled = providerModels.first(where: \.enabled) {
            return enabled
        }
        return providerModels.first
    }

    private func providerReadinessItem(for providerID: String) -> ProviderReadinessItem? {
        state.providerReadinessItems.first {
            $0.provider.caseInsensitiveCompare(providerID) == .orderedSame
        }
    }

    private func providerIsQuickstartReady(_ item: ProviderReadinessItem) -> Bool {
        switch item.status {
        case .configured, .healthy:
            return true
        case .missingSetup, .checkFailed:
            return false
        }
    }

    private func isProviderSetupOrCheckInFlight(providerID: String) -> Bool {
        let normalizedProvider = normalizedProviderID(providerID)
        return state.providerSetupInFlightIDs.contains(normalizedProvider) ||
            state.providerCheckInFlightIDs.contains(normalizedProvider)
    }

    private func isProviderModelCatalogMutationInFlight(providerID: String) -> Bool {
        let normalizedProvider = normalizedProviderID(providerID)
        return state.modelCatalogDiscoverInFlightProviderIDs.contains(normalizedProvider) ||
            state.modelCatalogManageInFlightProviderIDs.contains(normalizedProvider)
    }

    private func modelRouteMatches(providerID: String, modelKey: String) -> Bool {
        guard let route = state.modelRouteSummary else {
            return false
        }
        return route.provider.caseInsensitiveCompare(providerID) == .orderedSame &&
            route.modelKey.caseInsensitiveCompare(modelKey) == .orderedSame
    }

    private func modelsForProvider(_ providerID: String) -> [ModelCatalogEntryItem] {
        state.modelCatalogItems
            .filter { $0.provider.caseInsensitiveCompare(providerID) == .orderedSame }
            .sorted { lhs, rhs in
                lhs.modelKey.localizedCaseInsensitiveCompare(rhs.modelKey) == .orderedAscending
            }
    }

    private func discoveredModelsForProvider(_ providerID: String) -> [DiscoveredModelEntryItem] {
        state.discoveredModelsByProviderID[normalizedProviderID(providerID)]?
            .sorted { lhs, rhs in
                lhs.modelKey.localizedCaseInsensitiveCompare(rhs.modelKey) == .orderedAscending
            } ?? []
    }

    private func policiesForProvider(_ providerID: String) -> [ModelPolicyItem] {
        state.modelPolicyItems
            .filter { $0.provider.caseInsensitiveCompare(providerID) == .orderedSame }
            .sorted { lhs, rhs in
                lhs.taskClass.localizedCaseInsensitiveCompare(rhs.taskClass) == .orderedAscending
            }
    }

    private func providerCardSummary(providerID: String) -> String {
        let modelCount = modelsForProvider(providerID).count
        let policyCount = policiesForProvider(providerID).count
        let discoveredCount = discoveredModelsForProvider(providerID).count
        if discoveredCount > 0 {
            return "\(modelCount) models • \(policyCount) policies • \(discoveredCount) discovered"
        }
        return "\(modelCount) models • \(policyCount) policies"
    }

    private var routeTaskClassOptions: [String] {
        var options = state.contextTaskClassOptions
        for taskClass in state.modelPolicyItems.map(\.taskClass) where !taskClass.isEmpty {
            if !options.contains(where: { $0.caseInsensitiveCompare(taskClass) == .orderedSame }) {
                options.append(taskClass)
            }
        }
        return options
    }

    private var simulationTaskClassOptions: [String] {
        state.modelRouteSimulationTaskClassOptions
    }

    private var simulationTaskClassBinding: Binding<String> {
        Binding(
            get: {
                if let existing = canonicalOption(state.modelRouteSimulationTaskClass, within: simulationTaskClassOptions) {
                    return existing
                }
                return simulationTaskClassOptions.first ?? state.modelRouteSimulationTaskClass
            },
            set: { selection in
                state.modelRouteSimulationTaskClass = selection
            }
        )
    }

    private var routeProviderOptions: [String] {
        let providers = Set(state.modelCatalogItems.map { normalizedProviderID($0.provider) })
        return providers.sorted { lhs, rhs in
            providerDisplayName(lhs).localizedCaseInsensitiveCompare(providerDisplayName(rhs)) == .orderedAscending
        }
    }

    private var routeModelOptions: [String] {
        modelsForProvider(routeProviderSelection)
            .map(\.modelKey)
            .sorted { lhs, rhs in
                lhs.localizedCaseInsensitiveCompare(rhs) == .orderedAscending
            }
    }

    private func synchronizeRoutePolicyEditorSelection() {
        guard !routeTaskClassOptions.isEmpty else {
            routeTaskClassSelection = ""
            routeProviderSelection = ""
            routeModelSelection = ""
            return
        }

        if let existingTaskClass = canonicalOption(routeTaskClassSelection, within: routeTaskClassOptions) {
            routeTaskClassSelection = existingTaskClass
        } else if let chatOption = canonicalOption("chat", within: routeTaskClassOptions) {
            routeTaskClassSelection = chatOption
        } else {
            routeTaskClassSelection = routeTaskClassOptions[0]
        }

        guard !routeProviderOptions.isEmpty else {
            routeProviderSelection = ""
            routeModelSelection = ""
            return
        }

        let selectedPolicy = policyForTaskClass(routeTaskClassSelection)

        if let existingProvider = canonicalOption(routeProviderSelection, within: routeProviderOptions) {
            routeProviderSelection = existingProvider
        } else if let policyProvider = selectedPolicy.flatMap({ canonicalOption($0.provider, within: routeProviderOptions) }) {
            routeProviderSelection = policyProvider
        } else if routeTaskClassSelection.caseInsensitiveCompare("chat") == .orderedSame,
                  let routeProvider = state.modelRouteSummary.flatMap({ canonicalOption($0.provider, within: routeProviderOptions) }) {
            routeProviderSelection = routeProvider
        } else {
            routeProviderSelection = routeProviderOptions[0]
        }

        guard !routeModelOptions.isEmpty else {
            routeModelSelection = ""
            return
        }

        if let existingModel = canonicalOption(routeModelSelection, within: routeModelOptions) {
            routeModelSelection = existingModel
        } else if let policyModel = selectedPolicy.flatMap({ canonicalOption($0.modelKey, within: routeModelOptions) }) {
            routeModelSelection = policyModel
        } else if routeTaskClassSelection.caseInsensitiveCompare("chat") == .orderedSame,
                  let routeModel = state.modelRouteSummary.flatMap({ canonicalOption($0.modelKey, within: routeModelOptions) }) {
            routeModelSelection = routeModel
        } else {
            routeModelSelection = routeModelOptions[0]
        }
    }

    private func synchronizeRouteSimulationSelection() {
        guard !simulationTaskClassOptions.isEmpty else {
            state.modelRouteSimulationTaskClass = ""
            return
        }

        if let existing = canonicalOption(state.modelRouteSimulationTaskClass, within: simulationTaskClassOptions) {
            state.modelRouteSimulationTaskClass = existing
        } else if let chatOption = canonicalOption("chat", within: simulationTaskClassOptions) {
            state.modelRouteSimulationTaskClass = chatOption
        } else {
            state.modelRouteSimulationTaskClass = simulationTaskClassOptions[0]
        }

        if state.modelRouteSimulationPrincipalActorID.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            state.applyActivePrincipalToModelRouteSimulation()
        }
    }

    private func synchronizeProviderCardCollapsedState() {
        let providerIDs = Set(state.providerReadinessItems.map(\.id))
        collapsedProviderIDs = collapsedProviderIDs.intersection(providerIDs)
        let newProviderIDs = providerIDs.subtracting(knownProviderIDs)
        collapsedProviderIDs.formUnion(newProviderIDs)
        knownProviderIDs = providerIDs
    }

    private func policyForTaskClass(_ taskClass: String) -> ModelPolicyItem? {
        state.modelPolicyItems.first {
            $0.taskClass.caseInsensitiveCompare(taskClass) == .orderedSame
        }
    }

    private func modelIsCurrentChatRoute(_ item: ModelCatalogEntryItem) -> Bool {
        guard let chatPolicy = policyForTaskClass("chat") else {
            return false
        }
        return chatPolicy.provider.caseInsensitiveCompare(item.provider) == .orderedSame &&
            chatPolicy.modelKey.caseInsensitiveCompare(item.modelKey) == .orderedSame
    }

    private func canonicalOption(_ value: String, within options: [String]) -> String? {
        options.first { $0.caseInsensitiveCompare(value) == .orderedSame }
    }

    private func normalizedProviderID(_ providerID: String) -> String {
        providerID.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
    }

    private func expansionBinding(for providerID: String) -> Binding<Bool> {
        Binding(
            get: { !collapsedProviderIDs.contains(providerID) },
            set: { expanded in
                if expanded {
                    collapsedProviderIDs.remove(providerID)
                } else {
                    collapsedProviderIDs.insert(providerID)
                }
            }
        )
    }

    private func endpointBinding(for providerID: String) -> Binding<String> {
        Binding(
            get: { state.providerEndpointDraft(for: providerID) },
            set: { state.setProviderEndpointDraft($0, for: providerID) }
        )
    }

    private func manualAddBinding(for providerID: String) -> Binding<String> {
        Binding(
            get: { state.modelManualAddDraft(for: providerID) },
            set: { state.setModelManualAddDraft($0, for: providerID) }
        )
    }

    private func secretNameBinding(for providerID: String) -> Binding<String> {
        Binding(
            get: { state.providerSecretNameDraft(for: providerID) },
            set: { state.setProviderSecretNameDraft($0, for: providerID) }
        )
    }

    private func secretValueBinding(for providerID: String) -> Binding<String> {
        Binding(
            get: { state.providerSecretValueDraft(for: providerID) },
            set: { state.setProviderSecretValueDraft($0, for: providerID) }
        )
    }

    private func providerRequiresAPIKey(_ providerID: String) -> Bool {
        switch providerID.lowercased() {
        case "openai", "anthropic", "google":
            return true
        default:
            return false
        }
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
