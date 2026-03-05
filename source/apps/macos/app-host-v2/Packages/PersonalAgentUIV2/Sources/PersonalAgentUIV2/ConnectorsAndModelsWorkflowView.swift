import SwiftUI

struct ConnectorsAndModelsWorkflowView: View {
    @ObservedObject var store: AppShellV2Store

    var body: some View {
        VStack(alignment: .leading, spacing: V2WorkflowLayout.panelSpacing) {
            PASectionHeader(
                title: "Connectors and model routing",
                subtitle: "Keep operational complexity here so Replay stays clean, explainable, and trustable."
            )
            summaryStrip
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
            V2PanelStateBannerView(
                state: store.panelLifecycleState(for: .connectorsAndModels),
                onAction: { actionID in
                    store.performPanelStateAction(actionID, workflow: .connectorsAndModels)
                }
            )
            if let feedback = store.lastFeedback {
                feedbackRow(feedback)
            }
            contentGrid
            advancedCard
            Spacer(minLength: 0)
        }
        .onAppear {
            Task {
                await store.refreshConnectorsInventoryIfNeeded()
                await store.refreshModelsInventoryIfNeeded()
            }
        }
    }

    private var contentGrid: some View {
        ViewThatFits(in: .horizontal) {
            HStack(alignment: .top, spacing: V2WorkflowLayout.sectionSpacing) {
                connectorsCard
                modelsCard
            }

            VStack(alignment: .leading, spacing: V2WorkflowLayout.sectionSpacing) {
                connectorsCard
                modelsCard
            }
        }
    }

    private var summaryStrip: some View {
        HStack(spacing: V2WorkflowLayout.compactSpacing) {
            PAStatusChip(
                label: "\(connectedConnectorCount) Connected",
                systemImage: "checkmark.seal.fill",
                tone: .success
            )
            PAStatusChip(
                label: "\(attentionConnectorCount) Needs Attention",
                systemImage: "exclamationmark.triangle.fill",
                tone: .warning
            )
            PAStatusChip(
                label: store.modelRouteResolution == nil ? "Route Missing" : "Route Ready",
                systemImage: store.modelRouteResolution == nil ? "bolt.slash" : "bolt.fill",
                tone: store.modelRouteResolution == nil ? .warning : .info
            )
        }
        .padding(.top, 3)
        .padding(.bottom, 2)
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    private var connectorsCard: some View {
        let connectors = store.connectors
        return PASurfaceCard("Connectors", tone: .neutral) {
            VStack(alignment: .leading, spacing: 9) {
                ForEach(connectors) { connector in
                    VStack(alignment: .leading, spacing: 7) {
                        HStack(alignment: .top, spacing: 8) {
                            VStack(alignment: .leading, spacing: 5) {
                                HStack(spacing: 6) {
                                    Label(connector.name, systemImage: iconName(for: connector.name))
                                        .font(.system(size: 13, weight: .semibold, design: .rounded))
                                    PAStatusChip(
                                        label: connector.status.label,
                                        systemImage: statusSymbol(for: connector.status),
                                        tone: connector.status.statusTone
                                    )
                                    if let permissionState = connector.permissionState,
                                       !permissionState.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                                        PAStatusChip(
                                            label: permissionChipLabel(permissionState),
                                            systemImage: permissionState.lowercased() == "granted" ? "lock.open.fill" : "lock.fill",
                                            tone: permissionState.lowercased() == "granted" ? .success : .warning
                                        )
                                    }
                                }

                                Text(connector.summary)
                                    .font(.paBody)
                                    .foregroundStyle(Color.paTextSecondary)

                                Text(checkStatusText(for: connector))
                                    .font(.paCaption)
                                    .foregroundStyle(Color.paTextTertiary)
                            }
                            .frame(maxWidth: .infinity, alignment: .leading)

                            HStack(spacing: 6) {
                                connectorToggleButton(connector)
                                Button("Run Check") {
                                    store.runConnectorCheck(connector.id)
                                }
                                .buttonStyle(.bordered)
                                .tint(.paInfo)
                                .disabled(connectorCheckDisabled(connector))
                            }
                        }
                        .controlSize(.small)

                        DisclosureGroup("Configuration") {
                            VStack(alignment: .leading, spacing: 6) {
                                ForEach(store.connectorConfigurationDraftKeys(connector.id), id: \.self) { key in
                                    HStack(spacing: 8) {
                                        Text(key)
                                            .font(.paCaption)
                                            .foregroundStyle(Color.paTextSecondary)
                                            .frame(width: 120, alignment: .leading)
                                        TextField(
                                            "Value",
                                            text: connectorConfigDraftBinding(connector.id, key: key)
                                        )
                                        .textFieldStyle(.plain)
                                        .paInputSurface()
                                    }
                                }

                                HStack(spacing: 8) {
                                    Button("Reset Draft") {
                                        store.resetConnectorConfigurationDraft(connector.id)
                                    }
                                    .buttonStyle(.bordered)
                                    .tint(.paNeutral)
                                    .disabled(!connector.hasConfigDraftChanges || store.isConnectorActionInFlight(connector.id))

                                    Button("Save Config") {
                                        store.saveConnectorConfiguration(connector.id)
                                    }
                                    .buttonStyle(.borderedProminent)
                                    .tint(.paInfo)
                                    .disabled(!connector.hasConfigDraftChanges || connectorSaveDisabled(connector))
                                }
                                .controlSize(.small)
                            }
                            .padding(.top, 4)
                        }

                        HStack(spacing: 8) {
                            if shouldShowPermissionAction(connector) {
                                Button("Request Permission") {
                                    store.requestConnectorPermission(connector.id)
                                }
                                .buttonStyle(.bordered)
                                .tint(.paWarning)
                                .disabled(permissionRequestDisabled(connector))
                            }

                            if !secondaryRemediationActions(for: connector).isEmpty {
                                Menu("Remediation") {
                                    ForEach(secondaryRemediationActions(for: connector), id: \.identifier) { action in
                                        Button(action.label) {
                                            store.performConnectorRemediation(connectorID: connector.id, actionID: action.identifier)
                                        }
                                        .disabled(!action.enabled || store.isConnectorActionInFlight(connector.id))
                                    }
                                }
                                .menuStyle(.borderlessButton)
                                .disabled(store.isConnectorActionInFlight(connector.id))
                            }
                        }
                        .controlSize(.small)

                        if let actionStatus = store.connectorActionStatus(for: connector.id),
                           !actionStatus.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                            Text(actionStatus)
                                .font(.paCaption)
                                .foregroundStyle(Color.paTextSecondary)
                        }
                    }
                    .paSubsurface(connector.status == .connected ? .neutral : .warm)
                    .padding(.vertical, 2)

                    if connector.id != connectors.last?.id {
                        Divider().overlay(Color.paStrokeSoft)
                    }
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private var modelsCard: some View {
        let models = store.models
        return PASurfaceCard("Model Routing", tone: .neutral) {
            VStack(alignment: .leading, spacing: 9) {
                routeSummaryCard
                routeActionsRow
                routeStatusRows
                routeEvidenceSection

                if !models.isEmpty {
                    Divider().overlay(Color.paStrokeSoft)
                }

                ForEach(models) { model in
                    let disableReason = store.modelActionDisabledReason(for: model.id, action: .toggle)
                    let setPrimaryDisabledReason = store.modelActionDisabledReason(for: model.id, action: .setPrimary)
                    VStack(alignment: .leading, spacing: 7) {
                        HStack(alignment: .top, spacing: 8) {
                            VStack(alignment: .leading, spacing: 5) {
                                Text(model.routeLabel)
                                    .font(.system(size: 13, weight: .semibold, design: .rounded))
                                HStack(spacing: 6) {
                                    PAStatusChip(
                                        label: model.enabled ? "Enabled" : "Disabled",
                                        systemImage: model.enabled ? "checkmark" : "pause.fill",
                                        tone: model.enabled ? .success : .neutral
                                    )
                                    PAStatusChip(
                                        label: model.providerReady ? "Provider Ready" : "Provider Setup Needed",
                                        systemImage: model.providerReady ? "checkmark.seal.fill" : "exclamationmark.triangle.fill",
                                        tone: model.providerReady ? .success : .warning
                                    )
                                    if store.isModelActive(model.id) {
                                        PAStatusChip(label: "Primary", systemImage: "bolt.fill", tone: .info)
                                    }
                                }
                                if let endpoint = model.providerEndpoint?.trimmingCharacters(in: .whitespacesAndNewlines),
                                   !endpoint.isEmpty {
                                    Text(endpoint)
                                        .font(.paCaption)
                                        .foregroundStyle(Color.paTextTertiary)
                                }
                            }
                            .frame(maxWidth: .infinity, alignment: .leading)

                            HStack(spacing: 6) {
                                Button(model.enabled ? "Disable" : "Enable") {
                                    store.toggleModelEnabled(model.id)
                                }
                                .buttonStyle(.bordered)
                                .disabled(disableReason != nil || store.mutationLifecycle(for: .modelToggle).isDisabled || store.isModelActionInFlight(model.id))
                                .tint(.paInfo)

                                if store.isModelActive(model.id) {
                                    Button("Primary") {
                                        store.setActiveModel(model.id)
                                    }
                                    .buttonStyle(.bordered)
                                    .disabled(true)
                                    .tint(.paNeutral)
                                } else {
                                    Button("Set Primary") {
                                        store.setActiveModel(model.id)
                                    }
                                    .buttonStyle(.borderedProminent)
                                    .disabled(setPrimaryDisabledReason != nil || store.mutationLifecycle(for: .modelSetPrimary).isDisabled || store.isModelActionInFlight(model.id))
                                    .tint(.paInfo)
                                }
                            }
                        }
                        .controlSize(.small)
                        .paSubsurface(store.isModelActive(model.id) ? .cool : .neutral)
                        .padding(.vertical, 2)

                        if let disableReason {
                            Text(disableReason)
                                .font(.paCaption)
                                .foregroundStyle(Color.paTextSecondary)
                        }

                        if let actionStatus = store.modelActionStatus(for: model.id),
                           !actionStatus.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                            Text(actionStatus)
                                .font(.paCaption)
                                .foregroundStyle(Color.paTextSecondary)
                        }
                    }

                    if model.id != models.last?.id {
                        Divider().overlay(Color.paStrokeSoft)
                    }
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private var routeSummaryCard: some View {
        VStack(alignment: .leading, spacing: 5) {
            if let route = store.modelRouteResolution {
                HStack(spacing: 6) {
                    Text("Primary route: \(providerDisplayName(route.provider)) / \(route.modelKey)")
                        .font(.system(size: 13, weight: .semibold, design: .rounded))
                    PAStatusChip(label: "Active", systemImage: "bolt.fill", tone: .info)
                }
                if let status = store.modelRouteStatusMessage?.trimmingCharacters(in: .whitespacesAndNewlines),
                   !status.isEmpty {
                    Text(status)
                        .font(.paCaption)
                        .foregroundStyle(Color.paTextSecondary)
                }
            } else {
                HStack(spacing: 6) {
                    Text("Primary route not resolved")
                        .font(.system(size: 13, weight: .semibold, design: .rounded))
                    PAStatusChip(label: "Needs Setup", systemImage: "exclamationmark.triangle.fill", tone: .warning)
                }
                Text("Resolve provider and model routing before trusting automated execution.")
                    .font(.paCaption)
                    .foregroundStyle(Color.paTextSecondary)
            }
        }
        .paSubsurface(store.modelRouteResolution == nil ? .warm : .cool)
    }

    private var routeActionsRow: some View {
        HStack(spacing: V2WorkflowLayout.sectionSpacing) {
            Menu {
                ForEach(routeTaskClassOptions, id: \.self) { taskClass in
                    Button {
                        store.modelRouteTaskClass = taskClass
                    } label: {
                        if taskClass == normalizedRouteTaskClass {
                            Label(taskClass.uppercased(), systemImage: "checkmark")
                        } else {
                            Text(taskClass.uppercased())
                        }
                    }
                }
            } label: {
                Label("Task: \(normalizedRouteTaskClass.uppercased())", systemImage: "line.3.horizontal.decrease")
            }
            .menuStyle(.borderlessButton)
            .accessibilityLabel("Model Route Task Class")
            .accessibilityHint("Choose the task class for route simulation and explainability.")
            .accessibilityIdentifier("v2-model-route-task-class")

            Button("Simulate Route") {
                store.simulateModelRoute()
            }
            .buttonStyle(.bordered)
            .tint(.paInfo)
            .disabled(store.mutationLifecycle(for: .modelRouteSimulate).isDisabled || store.mutationLifecycle(for: .modelRouteSimulate).isInFlight)
            .accessibilityIdentifier("v2-model-route-simulate")

            Button("Explain Route") {
                store.explainModelRoute()
            }
            .buttonStyle(.borderedProminent)
            .tint(.paInfo)
            .disabled(store.mutationLifecycle(for: .modelRouteExplain).isDisabled || store.mutationLifecycle(for: .modelRouteExplain).isInFlight)
            .accessibilityIdentifier("v2-model-route-explain")

            Spacer(minLength: 0)

            Button("Refresh") {
                Task {
                    await store.refreshModelsInventory(force: true)
                }
            }
            .buttonStyle(.bordered)
            .disabled(store.isModelInventoryRefreshInFlight)
            .accessibilityIdentifier("v2-model-route-refresh")
        }
        .controlSize(.small)
    }

    @ViewBuilder
    private var routeStatusRows: some View {
        if let status = store.modelRouteSimulationStatusMessage?.trimmingCharacters(in: .whitespacesAndNewlines),
           !status.isEmpty {
            Text(status)
                .font(.paCaption)
                .foregroundStyle(Color.paTextSecondary)
        }

        if let status = store.modelRouteExplainStatusMessage?.trimmingCharacters(in: .whitespacesAndNewlines),
           !status.isEmpty {
            Text(status)
                .font(.paCaption)
                .foregroundStyle(Color.paTextSecondary)
        }
    }

    @ViewBuilder
    private var routeEvidenceSection: some View {
        if let simulation = store.modelRouteSimulation {
            DisclosureGroup("Route Simulation") {
                VStack(alignment: .leading, spacing: 5) {
                    Text("Selected: \(providerDisplayName(simulation.selectedProvider)) / \(simulation.selectedModelKey)")
                        .font(.paBody)
                        .foregroundStyle(Color.paTextPrimary)
                    if !simulation.reasonCodes.isEmpty {
                        Text("Reason codes: \(simulation.reasonCodes.joined(separator: ", "))")
                            .font(.paCaption)
                            .foregroundStyle(Color.paTextSecondary)
                    }
                    if !simulation.decisions.isEmpty {
                        Text("Decisions")
                            .font(.paCaption)
                            .foregroundStyle(Color.paTextSecondary)
                        ForEach(Array(simulation.decisions.enumerated()), id: \.offset) { _, decision in
                            Text("• \(decision.step): \(decision.decision) (\(decision.reasonCode))")
                                .font(.paCaption)
                                .foregroundStyle(Color.paTextSecondary)
                        }
                    }
                }
                .padding(.top, 3)
            }
        }

        if let explainability = store.modelRouteExplainability {
            DisclosureGroup("Route Explainability") {
                VStack(alignment: .leading, spacing: 5) {
                    Text(explainability.summary)
                        .font(.paBody)
                        .foregroundStyle(Color.paTextPrimary)
                    if !explainability.explanations.isEmpty {
                        ForEach(Array(explainability.explanations.enumerated()), id: \.offset) { _, explanation in
                            Text("• \(explanation)")
                                .font(.paCaption)
                                .foregroundStyle(Color.paTextSecondary)
                        }
                    }
                    if !explainability.fallbackChain.isEmpty {
                        Text("Fallback chain")
                            .font(.paCaption)
                            .foregroundStyle(Color.paTextSecondary)
                        ForEach(explainability.fallbackChain, id: \.rank) { option in
                            Text("• #\(option.rank) \(providerDisplayName(option.provider)) / \(option.modelKey)\(option.selected ? " (selected)" : "")")
                                .font(.paCaption)
                                .foregroundStyle(Color.paTextSecondary)
                        }
                    }
                }
                .padding(.top, 3)
            }
        }
    }

    private var advancedCard: some View {
        PASurfaceCard("Maintenance", tone: .warm) {
            DisclosureGroup("Operator-only controls") {
                VStack(alignment: .leading, spacing: 6) {
                    Text("Keep this hidden for everyday usage. Use only for debugging and handoff checks.")
                        .font(.paBody)
                        .foregroundStyle(Color.paTextSecondary)

                    HStack(spacing: 8) {
                        Button("Open Replay & Ask") {
                            store.selectedSection = .replayAndAsk
                        }
                        .buttonStyle(.bordered)
                        .tint(.paInfo)

                        Button("Clear Status Message") {
                            store.dismissFeedback()
                        }
                        .buttonStyle(.bordered)
                        .tint(.paWarning)
                        .disabled(store.lastFeedback == nil)
                    }
                    .controlSize(.small)
                }
                .padding(.top, 4)
            }
        }
    }

    private var activeModel: ModelOption? {
        store.activeModel
    }

    private var normalizedRouteTaskClass: String {
        let trimmed = store.modelRouteTaskClass.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        return trimmed.isEmpty ? "chat" : trimmed
    }

    private var routeTaskClassOptions: [String] {
        ["chat", "message", "voice", "automation"]
    }

    private var connectedConnectorCount: Int {
        store.connectors.filter { $0.status == .connected }.count
    }

    private var attentionConnectorCount: Int {
        store.connectors.filter { $0.status == .needsAttention }.count
    }

    private func checkStatusText(for connector: ConnectorState) -> String {
        if let summary = connector.lastCheckSummary?.trimmingCharacters(in: .whitespacesAndNewlines),
           !summary.isEmpty,
           let lastCheckAt = connector.lastCheckAt {
            return "\(summary) • Checked \(lastCheckAt.formatted(date: .omitted, time: .shortened))"
        }
        if let lastCheckAt = connector.lastCheckAt {
            return "Checked \(lastCheckAt.formatted(date: .omitted, time: .shortened))"
        }
        return "No checks run yet"
    }

    private func iconName(for connectorName: String) -> String {
        let lowercasedName = connectorName.lowercased()
        if lowercasedName.contains("message") {
            return "message"
        }
        if lowercasedName.contains("whats") {
            return "phone.bubble"
        }
        if lowercasedName.contains("telegram") {
            return "paperplane"
        }
        if lowercasedName.contains("email") || lowercasedName.contains("mail") {
            return "envelope"
        }
        if lowercasedName.contains("voice") {
            return "waveform"
        }
        return "link"
    }

    private func feedbackRow(_ feedback: String) -> some View {
        PAInlineBanner(text: feedback, tone: .info)
    }

    private func connectorConfigDraftBinding(_ connectorID: ConnectorState.ID, key: String) -> Binding<String> {
        Binding(
            get: { store.connectorConfigurationDraftValue(connectorID: connectorID, key: key) },
            set: { store.setConnectorConfigurationDraftValue(connectorID: connectorID, key: key, value: $0) }
        )
    }

    @ViewBuilder
    private func connectorToggleButton(_ connector: ConnectorState) -> some View {
        if connector.enabled {
            Button(connector.connectActionLabel) {
                store.toggleConnector(connector.id)
            }
            .buttonStyle(.bordered)
            .tint(.paNeutral)
            .disabled(connectorToggleDisabled(connector))
        } else {
            Button(connector.connectActionLabel) {
                store.toggleConnector(connector.id)
            }
            .buttonStyle(.borderedProminent)
            .tint(.paInfo)
            .disabled(connectorToggleDisabled(connector))
        }
    }

    private func connectorToggleDisabled(_ connector: ConnectorState) -> Bool {
        store.isConnectorActionInFlight(connector.id) ||
        store.mutationLifecycle(for: .connectorToggle).isDisabled ||
        store.connectorActionDisabledReason(for: connector.id, action: .toggle) != nil
    }

    private func connectorCheckDisabled(_ connector: ConnectorState) -> Bool {
        store.isConnectorActionInFlight(connector.id) ||
        store.mutationLifecycle(for: .connectorCheck).isDisabled ||
        store.connectorActionDisabledReason(for: connector.id, action: .check) != nil
    }

    private func connectorSaveDisabled(_ connector: ConnectorState) -> Bool {
        store.isConnectorActionInFlight(connector.id) ||
        store.mutationLifecycle(for: .connectorSaveConfig).isDisabled ||
        store.connectorActionDisabledReason(for: connector.id, action: .saveConfig) != nil
    }

    private func permissionRequestDisabled(_ connector: ConnectorState) -> Bool {
        store.isConnectorActionInFlight(connector.id) ||
        store.mutationLifecycle(for: .connectorPermission).isDisabled ||
        store.connectorActionDisabledReason(for: connector.id, action: .requestPermission) != nil
    }

    private func shouldShowPermissionAction(_ connector: ConnectorState) -> Bool {
        let permission = connector.permissionState?.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        if permission == nil || permission == "unknown" || permission == "missing" {
            return true
        }
        return connector.remediationActions.contains(where: { remediation in
            remediation.intent.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() == "request_permission"
        })
    }

    private func secondaryRemediationActions(for connector: ConnectorState) -> [V2DaemonDiagnosticsRemediationAction] {
        connector.remediationActions.filter { remediation in
            remediation.intent.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() != "request_permission"
        }
    }

    private func permissionChipLabel(_ permissionState: String) -> String {
        switch permissionState.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
        case "granted":
            return "Permission Granted"
        case "missing":
            return "Permission Missing"
        case "unknown", "":
            return "Permission Unknown"
        default:
            return permissionState
        }
    }

    private func statusSymbol(for status: ConnectorStatus) -> String {
        switch status {
        case .connected:
            return "checkmark.seal.fill"
        case .notConnected:
            return "pause.circle.fill"
        case .needsAttention:
            return "exclamationmark.triangle.fill"
        }
    }

    private func providerDisplayName(_ providerID: String) -> String {
        switch providerID.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
        case "openai":
            return "OpenAI"
        case "anthropic":
            return "Anthropic"
        case "built_in", "builtin", "personalagent":
            return "Built-In"
        default:
            let trimmed = providerID.trimmingCharacters(in: .whitespacesAndNewlines)
            if trimmed.isEmpty {
                return "Unknown"
            }
            return trimmed
                .replacingOccurrences(of: "_", with: " ")
                .replacingOccurrences(of: "-", with: " ")
                .split(separator: " ")
                .map { component in
                    component.prefix(1).uppercased() + component.dropFirst().lowercased()
                }
                .joined(separator: " ")
        }
    }
}
