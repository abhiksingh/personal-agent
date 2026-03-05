import AppKit
import SwiftUI

struct ConnectorsPanelView: View {
    @ObservedObject private var state: AppShellState
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @State private var pendingConfigKeyByConnectorID: [String: String] = [:]
    @State private var pendingConfigValueByConnectorID: [String: String] = [:]
    @State private var advancedExpandedByConnectorID: [String: Bool] = [:]

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
                    state.performPanelProblemRemediationAction(actionID, section: .connectors)
                }
                .padding(.horizontal, UIStyle.panelPadding)
                .padding(.bottom, 12)
            }

            Divider()

            content
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    @ViewBuilder
    private var content: some View {
        if showLoadingSkeleton {
            PanelLoadingSkeletonView(
                title: "Loading Connectors",
                subtitle: "Fetching connector status, permissions, and remediation actions.",
                rowCount: 3
            )
            .frame(maxWidth: .infinity, maxHeight: .infinity)
        } else if logicalConnectorCards.isEmpty {
            PanelRemediationEmptyStateView(
                title: "No Connectors Found",
                systemImage: "cable.connector",
                description: "Connector status will appear once daemon connector inventory is available.",
                statusMessage: state.connectorsStatusMessage,
                headerStatusMessage: state.connectorsStatusMessage,
                actions: state.connectorsEmptyStateRemediationActions
            ) { actionID in
                state.performEmptyStateRemediationAction(actionID)
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)
            .background(UIStyle.panelGradient)
        } else {
            ScrollView {
                LazyVStack(spacing: UIStyle.standardSpacing) {
                    ForEach(logicalConnectorCards) { logicalCard in
                        connectorCard(logicalCard)
                    }
                }
                .padding(UIStyle.panelPadding)
            }
            .background(UIStyle.panelGradient)
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
        state.panelProblemRemediation(for: .connectors)
    }

    private var isAdvancedInformationDensityEnabled: Bool {
        state.isAdvancedInformationDensityEnabled
    }

    private var header: some View {
        TahoeSectionHeader(
            title: "Connectors",
            subtitle: state.connectorsStatusMessage ?? "Connector health and permission controls"
        ) {
            HStack(spacing: 8) {
                if state.isConnectorsLoading {
                    ProgressView()
                        .controlSize(.small)
                }

                if state.connectorsHasUnsavedDraftChanges {
                    Text("Unsaved changes")
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(.orange)
                }

                Button("Discard All") {
                    state.discardDraftChanges(for: .connectors)
                }
                .buttonStyle(.bordered)
                .disabled(!state.connectorsHasUnsavedDraftChanges || isConnectorBulkMutationInFlight)

                Button("Save All") {
                    state.saveAllDraftChanges(for: .connectors)
                }
                .buttonStyle(.borderedProminent)
                .successSymbolEffect(
                    state.successNotificationPulse(for: "connectors"),
                    reduceMotion: reduceMotion
                )
                .disabled(!state.connectorsHasUnsavedDraftChanges || isConnectorBulkMutationInFlight)

                Button {
                    state.refreshConnectorCards()
                } label: {
                    Label("Refresh", systemImage: "arrow.clockwise")
                }
                .buttonStyle(.bordered)
                .disabled(state.isConnectorsLoading)
            }
        }
    }

    private var logicalConnectorCards: [LogicalConnectorCardItem] {
        state.logicalConnectorCards
    }

    private var showLoadingSkeleton: Bool {
        (state.isConnectorsLoading || !state.hasLoadedConnectorStatus) && logicalConnectorCards.isEmpty
    }

    @ViewBuilder
    private func connectorCard(_ logicalCard: LogicalConnectorCardItem) -> some View {
        if let card = state.connectorCardItem(connectorID: logicalCard.primaryConnectorCardID) {
            GroupBox {
                DisclosureGroup(isExpanded: expansionBinding(for: card)) {
                VStack(alignment: .leading, spacing: 10) {
                    let remediationActions = connectorRemediationActions(
                        for: logicalCard,
                        primaryConnectorID: card.id
                    )
                    let requestPermissionAction = connectorRequestPermissionAction(in: remediationActions)
                    let openSystemSettingsAction = connectorOpenSystemSettingsAction(in: remediationActions)
                    let diagnosticsActions = connectorSecondaryDiagnosticsActions(from: remediationActions)
                    let showsRequestPermission = requestPermissionAction != nil

                    Divider()

                    HStack(spacing: 8) {
                        TahoeStatusBadge(
                            text: logicalCard.permissionState.label,
                            symbolName: logicalCard.permissionState.symbolName,
                            tint: logicalCard.permissionState.tint
                        )
                        Text(logicalCard.permissionScope)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }

                    if isAdvancedInformationDensityEnabled {
                        VStack(alignment: .leading, spacing: 8) {
                            ForEach(logicalCard.details.keys.sorted(), id: \.self) { key in
                                detailRow(key: key, value: logicalCard.details[key] ?? "")
                            }
                        }
                        Divider()
                    }

                    VStack(alignment: .leading, spacing: 8) {
                        connectorConfigurationSection(card: card)

                        HStack(spacing: 8) {
                            Button("Reset Draft") {
                                state.resetConnectorConfigDraft(connectorID: card.id)
                            }
                            .buttonStyle(.bordered)
                            .disabled(state.connectorConfigSaveInFlightIDs.contains(card.id) || state.isConnectorsLoading)

                            Button(state.connectorConfigSaveInFlightIDs.contains(card.id) ? "Saving…" : "Save Config") {
                                state.saveConnectorConfiguration(connectorID: card.id)
                            }
                            .buttonStyle(.borderedProminent)
                            .disabled(state.connectorConfigSaveInFlightIDs.contains(card.id) || state.isConnectorsLoading)

                            Button(state.connectorTestInFlightIDs.contains(card.id) ? "Running…" : "Run Health Check") {
                                state.runConnectorHealthCheck(connectorID: card.id)
                            }
                            .buttonStyle(.bordered)
                            .disabled(
                                state.connectorTestInFlightIDs.contains(card.id)
                                    || state.connectorConfigSaveInFlightIDs.contains(card.id)
                                    || state.isConnectorsLoading
                            )
                        }

                        if let status = state.connectorConfigActionStatusByID[card.id] {
                            Text(status)
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }

                        if let testResult = state.connectorLastTestResultByID[card.id] {
                            HStack(spacing: 8) {
                                TahoeStatusBadge(
                                    text: testResult.success ? "Healthy" : "Needs Attention",
                                    symbolName: testResult.success ? "checkmark.circle.fill" : "exclamationmark.triangle.fill",
                                    tint: testResult.success ? .green : .orange
                                )
                                Text("Status: \(testResult.status) • \(testResult.checkedAtLabel)")
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }
                            Text(testResult.summary)
                                .font(.caption)
                                .foregroundStyle(.secondary)
                            if isAdvancedInformationDensityEnabled && !testResult.details.isEmpty {
                                VStack(alignment: .leading, spacing: 4) {
                                    ForEach(testResult.details.keys.sorted(), id: \.self) { key in
                                        detailRow(key: key, value: testResult.details[key] ?? "")
                                    }
                                }
                            }
                        }
                    }

                    HStack(spacing: 8) {
                        if let requestPermissionAction {
                            Button(
                                state.connectorPermissionRequestInFlightIDs.contains(card.id)
                                    ? "Requesting…"
                                    : requestPermissionAction.title
                            ) {
                                state.performConnectorDiagnosticsAction(connectorID: card.id, action: requestPermissionAction)
                            }
                            .buttonStyle(.bordered)
                            .disabled(
                                logicalCard.permissionState == .granted
                                    || !requestPermissionAction.enabled
                                    || state.connectorPermissionRequestInFlightIDs.contains(card.id)
                                    || state.isConnectorsLoading
                            )
                        }

                        Button(openSystemSettingsAction?.title ?? "Open System Settings") {
                            openSystemSettings(for: card.id, action: openSystemSettingsAction)
                        }
                        .buttonStyle(.bordered)
                        .disabled(
                            !(openSystemSettingsAction?.enabled ?? true)
                                || state.isConnectorsLoading
                        )
                    }

                    if !diagnosticsActions.isEmpty {
                        ScrollView(.horizontal, showsIndicators: false) {
                            HStack(spacing: 8) {
                                ForEach(diagnosticsActions) { action in
                                    Button(action.title) {
                                        state.performConnectorDiagnosticsAction(connectorID: card.id, action: action)
                                    }
                                    .buttonStyle(.bordered)
                                    .disabled(
                                        !action.enabled
                                            || state.isConnectorsLoading
                                            || state.isDaemonControlInFlight
                                    )
                                }
                            }
                        }
                        .frame(maxWidth: .infinity, alignment: .leading)
                    }

                    if let helperStatus = connectorCardHelperStatusMessage(
                        logicalCard: logicalCard,
                        card: card,
                        remediationActions: remediationActions,
                        showsRequestPermission: showsRequestPermission,
                        actionStatus: state.connectorPermissionActionStatusByID[card.id]
                    ) {
                        Text(helperStatus)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                }
                .padding(.top, 8)
            } label: {
                HStack(alignment: .firstTextBaseline, spacing: 10) {
                    VStack(alignment: .leading, spacing: 3) {
                        Text(logicalCard.title)
                            .font(.headline)
                        Text(logicalCard.summary)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                        if state.connectorConfigHasDraftChanges(connectorID: card.id) {
                            Text("Unsaved draft changes")
                                .font(.caption2.weight(.semibold))
                                .foregroundStyle(.orange)
                        }
                    }

                    Spacer(minLength: 0)

                    TahoeStatusBadge(
                        text: logicalCard.health.label,
                        symbolName: logicalCard.health.symbolName,
                        tint: logicalCard.health.tint
                    )
                }
            }
        }
        .groupBoxStyle(.automatic)
        .animation(reduceMotion ? nil : .snappy(duration: 0.2), value: card.isExpanded)
        } else {
            GroupBox {
                VStack(alignment: .leading, spacing: 6) {
                    Text(logicalCard.title)
                        .font(.headline)
                    Text("Connector data is loading. Refresh to resolve logical connector mapping.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }
            .groupBoxStyle(.automatic)
        }
    }

    private var isConnectorBulkMutationInFlight: Bool {
        state.isConnectorsLoading ||
            !state.connectorConfigSaveInFlightIDs.isEmpty ||
            !state.connectorTestInFlightIDs.isEmpty ||
            !state.connectorPermissionRequestInFlightIDs.isEmpty
    }

    private func connectorConfigurationSection(card: ConnectorCardItem) -> some View {
        let descriptors = state.connectorConfigFieldDescriptors(connectorID: card.id)
        return VStack(alignment: .leading, spacing: 10) {
            Text("Configuration")
                .font(.subheadline.weight(.semibold))

            if descriptors.isEmpty {
                Text("No guided configuration descriptors are available yet. Use Advanced to edit raw key/value fields.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            } else {
                ForEach(descriptors) { descriptor in
                    guidedConnectorConfigFieldRow(connectorID: card.id, descriptor: descriptor)
                }
            }

            DisclosureGroup(
                isExpanded: advancedConnectorConfigExpansionBinding(connectorID: card.id)
            ) {
                VStack(alignment: .leading, spacing: 8) {
                    Text("Raw key-value fallback for fields not covered by guided descriptors.")
                        .font(.caption)
                        .foregroundStyle(.secondary)

                    let advancedDraftKeys = state.connectorAdvancedConfigDraftKeys(connectorID: card.id)
                    let readOnlyKeys = advancedReadOnlyConnectorConfigKeys(card: card)
                    if advancedDraftKeys.isEmpty && readOnlyKeys.isEmpty {
                        Text("No advanced-only configuration fields are present.")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    } else {
                        ForEach(advancedDraftKeys, id: \.self) { key in
                            VStack(alignment: .leading, spacing: 4) {
                                HStack(spacing: 8) {
                                    Text(key)
                                        .font(.caption.weight(.semibold))
                                        .foregroundStyle(.secondary)
                                        .frame(width: 150, alignment: .leading)
                                    if state.connectorConfigDraftKind(connectorID: card.id, key: key).supportsInlineEditing {
                                        TextField(
                                            "Value",
                                            text: connectorConfigDraftBinding(connectorID: card.id, key: key)
                                        )
                                        .textFieldStyle(.roundedBorder)
                                    } else {
                                        Text(state.connectorConfigDraftValue(connectorID: card.id, key: key))
                                            .font(.callout)
                                            .frame(maxWidth: .infinity, alignment: .leading)
                                    }
                                    Button {
                                        state.removeConnectorConfigDraftField(connectorID: card.id, key: key)
                                    } label: {
                                        Image(systemName: "minus.circle")
                                    }
                                    .buttonStyle(.plain)
                                    .accessibilityLabel("Remove advanced field \(key)")
                                    .accessibilityHint("Deletes this draft configuration field.")
                                }
                                Text("Type: \(state.connectorConfigDraftKind(connectorID: card.id, key: key).label)")
                                    .font(.caption2)
                                    .foregroundStyle(.secondary)
                            }
                        }

                        ForEach(readOnlyKeys, id: \.self) { key in
                            VStack(alignment: .leading, spacing: 4) {
                                detailRow(
                                    key: key,
                                    value: card.readOnlyConfiguration[key] ?? ""
                                )
                                Text("Read-only complex value from daemon.")
                                    .font(.caption2)
                                    .foregroundStyle(.secondary)
                                    .padding(.leading, 146)
                            }
                        }
                    }

                    HStack(spacing: 8) {
                        TextField(
                            "Config key",
                            text: pendingConnectorConfigKeyBinding(connectorID: card.id)
                        )
                        .textFieldStyle(.roundedBorder)
                        TextField(
                            "Config value",
                            text: pendingConnectorConfigValueBinding(connectorID: card.id)
                        )
                        .textFieldStyle(.roundedBorder)
                        Button("Add Field") {
                            addConnectorConfigField(connectorID: card.id)
                        }
                        .buttonStyle(.bordered)
                    }
                }
                .padding(.top, 6)
            } label: {
                Label("Advanced", systemImage: "slider.horizontal.3")
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(.secondary)
            }
        }
    }

    @ViewBuilder
    private func guidedConnectorConfigFieldRow(
        connectorID: String,
        descriptor: ConfigurationFieldDescriptorItem
    ) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                Text(descriptor.label)
                    .font(.caption.weight(.semibold))
                if descriptor.required {
                    Text("Required")
                        .font(.caption2.weight(.semibold))
                        .foregroundStyle(.orange)
                }
                if descriptor.secret {
                    Text(descriptor.writeOnly ? "Secret • Write-only" : "Secret")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                } else if !descriptor.editable {
                    Text("Read-only")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }
            }

            if descriptor.editable {
                if !descriptor.enumOptions.isEmpty {
                    Picker(
                        descriptor.label,
                        selection: connectorConfigDraftBinding(connectorID: connectorID, key: descriptor.key)
                    ) {
                        if !descriptor.required {
                            Text("Not Set").tag("")
                        }
                        ForEach(descriptor.enumOptions, id: \.self) { option in
                            Text(option).tag(option)
                        }
                    }
                    .pickerStyle(.menu)
                } else if descriptor.draftKind == .bool {
                    Toggle(
                        descriptor.label,
                        isOn: connectorConfigBoolBinding(connectorID: connectorID, key: descriptor.key)
                    )
                    .toggleStyle(.switch)
                } else if descriptor.secret {
                    SecureField(
                        descriptor.writeOnly ? "Enter new secret value" : "Secret value",
                        text: connectorConfigDraftBinding(connectorID: connectorID, key: descriptor.key)
                    )
                    .textFieldStyle(.roundedBorder)
                } else {
                    TextField(
                        descriptor.draftKind == .number ? "Number" : "Value",
                        text: connectorConfigDraftBinding(connectorID: connectorID, key: descriptor.key)
                    )
                    .textFieldStyle(.roundedBorder)
                }
            } else {
                Text(nonEmptyStateValue(state.connectorConfigDraftValue(connectorID: connectorID, key: descriptor.key)))
                    .font(.callout)
                    .foregroundStyle(.secondary)
            }

            if descriptor.writeOnly && descriptor.editable {
                Text("Write-only field. Leave blank to keep existing secret unchanged.")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }
            if let helpText = descriptor.helpText {
                Text(helpText)
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }
        }
    }

    private func advancedReadOnlyConnectorConfigKeys(card: ConnectorCardItem) -> [String] {
        let guidedKeys = Set(state.connectorGuidedConfigFieldKeys(connectorID: card.id))
        return card.readOnlyConfiguration.keys
            .filter { !guidedKeys.contains($0) }
            .sorted()
    }

    private func advancedConnectorConfigExpansionBinding(connectorID: String) -> Binding<Bool> {
        Binding(
            get: { advancedExpandedByConnectorID[connectorID] ?? false },
            set: { advancedExpandedByConnectorID[connectorID] = $0 }
        )
    }

    private func connectorConfigBoolBinding(connectorID: String, key: String) -> Binding<Bool> {
        Binding(
            get: {
                parseBool(state.connectorConfigDraftValue(connectorID: connectorID, key: key)) ?? false
            },
            set: { value in
                state.setConnectorConfigDraftValue(connectorID: connectorID, key: key, value: value ? "true" : "false")
            }
        )
    }

    private func parseBool(_ raw: String) -> Bool? {
        switch raw.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
        case "true", "yes", "1":
            return true
        case "false", "no", "0":
            return false
        default:
            return nil
        }
    }

    private func nonEmptyStateValue(_ value: String) -> String {
        value.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ? "Not set" : value
    }

    private func detailRow(key: String, value: String) -> some View {
        HStack(alignment: .top, spacing: 10) {
            Text(key)
                .font(.caption.weight(.semibold))
                .foregroundStyle(.secondary)
                .frame(width: 136, alignment: .leading)
            Text(value)
                .font(.callout)
                .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private func expansionBinding(for card: ConnectorCardItem) -> Binding<Bool> {
        Binding(
            get: { card.isExpanded },
            set: { expanded in
                guard expanded != card.isExpanded else {
                    return
                }
                state.toggleConnectorCard(card.id)
            }
        )
    }

    private func openSystemSettings(for connectorID: String, action: DiagnosticsActionItem?) {
        if let action {
            state.performConnectorDiagnosticsAction(connectorID: connectorID, action: action)
        } else {
            state.noteConnectorSystemSettingsOpened(connectorID: connectorID)
        }
        NSWorkspace.shared.open(
            state.systemSettingsURLForConnectorAction(connectorID: connectorID, action: action)
        )
    }

    private func connectorRemediationActions(
        for card: LogicalConnectorCardItem,
        primaryConnectorID: String
    ) -> [DiagnosticsActionItem] {
        card.actions
            .filter { action in
                state.canPerformConnectorDiagnosticsAction(connectorID: primaryConnectorID, action: action)
            }
            .sorted(by: connectorActionSortOrder)
    }

    private func connectorRequestPermissionAction(in actions: [DiagnosticsActionItem]) -> DiagnosticsActionItem? {
        actions.first(where: { connectorActionIntent($0) == "request_permission" })
    }

    private func connectorOpenSystemSettingsAction(in actions: [DiagnosticsActionItem]) -> DiagnosticsActionItem? {
        actions.first(where: { connectorActionIntent($0) == "open_system_settings" })
    }

    private func connectorSecondaryDiagnosticsActions(from actions: [DiagnosticsActionItem]) -> [DiagnosticsActionItem] {
        actions.filter { action in
            let intent = connectorActionIntent(action)
            return intent != "request_permission" && intent != "open_system_settings"
        }
    }

    private func connectorActionIntent(_ action: DiagnosticsActionItem) -> String {
        action.intent.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
    }

    private func connectorActionSortOrder(_ lhs: DiagnosticsActionItem, _ rhs: DiagnosticsActionItem) -> Bool {
        let lhsTuple = (
            lhs.recommended ? 0 : 1,
            connectorActionIntentPriority(lhs),
            lhs.title.lowercased(),
            lhs.id.lowercased()
        )
        let rhsTuple = (
            rhs.recommended ? 0 : 1,
            connectorActionIntentPriority(rhs),
            rhs.title.lowercased(),
            rhs.id.lowercased()
        )
        return lhsTuple < rhsTuple
    }

    private func connectorActionIntentPriority(_ action: DiagnosticsActionItem) -> Int {
        switch connectorActionIntent(action) {
        case "request_permission":
            return 0
        case "open_system_settings":
            return 1
        case "daemon_lifecycle_control":
            return 2
        case "navigate":
            return 3
        case "refresh_status":
            return 4
        default:
            return 5
        }
    }

    private func connectorCardHelperStatusMessage(
        logicalCard: LogicalConnectorCardItem,
        card: ConnectorCardItem,
        remediationActions: [DiagnosticsActionItem],
        showsRequestPermission: Bool,
        actionStatus: String?
    ) -> String? {
        let disabledReason = remediationActions.first(where: { !$0.enabled })?.reason
        let statusReasonSummary = connectorStatusReasonSummary(logicalCard.statusReason)
        let unavailableReason = remediationActions.isEmpty ? card.unavailableActionReason : nil
        let permissionFallback = connectorPermissionFallbackMessage(
            logicalCard: logicalCard,
            showsRequestPermission: showsRequestPermission
        )

        let candidates = [
            actionStatus,
            disabledReason,
            statusReasonSummary,
            unavailableReason,
            permissionFallback
        ]

        for candidate in candidates {
            if let normalized = normalizedStatusMessage(candidate) {
                return normalized
            }
        }
        return nil
    }

    private func connectorPermissionFallbackMessage(
        logicalCard: LogicalConnectorCardItem,
        showsRequestPermission: Bool
    ) -> String? {
        if !showsRequestPermission {
            return "In-app permission request is unavailable for this connector. Use Open System Settings for remediation."
        }
        switch logicalCard.permissionState {
        case .granted:
            return "Permission already granted. Request action is intentionally disabled."
        case .unknown:
            return "Permission status is unknown. Request permission to prompt macOS or open System Settings."
        case .missing:
            return "Permission request is available because required access is missing."
        }
    }

    private func normalizedStatusMessage(_ value: String?) -> String? {
        let trimmed = value?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        return trimmed.isEmpty ? nil : trimmed
    }

    private func connectorStatusReasonSummary(_ rawReason: String?) -> String? {
        switch rawReason?.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
        case "permission_missing":
            return "Daemon classified this connector as blocked by missing macOS permission."
        case "worker_missing":
            return "Connector worker is not registered with daemon supervision."
        case "worker_starting":
            return "Connector worker is still starting. Refresh after startup completes."
        case "worker_stopped":
            return "Connector worker is stopped and may require restart or repair."
        case "worker_failed":
            return "Connector worker failed. Review Inspect logs and use remediation actions."
        case "runtime_failure":
            return "Connector runtime reported a degraded state. Review Inspect logs for details."
        case "cloudflared_binary_missing":
            return "Cloudflared binary is unavailable for this connector runtime."
        case "cloudflared_runtime_failure":
            return "Cloudflared runtime probe failed. Review connector diagnostics and logs."
        case "ready", .none, "":
            return nil
        default:
            return "Daemon status reason: \(rawReason ?? "unknown")."
        }
    }

    private func connectorConfigDraftBinding(connectorID: String, key: String) -> Binding<String> {
        Binding(
            get: { state.connectorConfigDraftValue(connectorID: connectorID, key: key) },
            set: { newValue in
                state.setConnectorConfigDraftValue(connectorID: connectorID, key: key, value: newValue)
            }
        )
    }

    private func pendingConnectorConfigKeyBinding(connectorID: String) -> Binding<String> {
        Binding(
            get: { pendingConfigKeyByConnectorID[connectorID] ?? "" },
            set: { pendingConfigKeyByConnectorID[connectorID] = $0 }
        )
    }

    private func pendingConnectorConfigValueBinding(connectorID: String) -> Binding<String> {
        Binding(
            get: { pendingConfigValueByConnectorID[connectorID] ?? "" },
            set: { pendingConfigValueByConnectorID[connectorID] = $0 }
        )
    }

    private func addConnectorConfigField(connectorID: String) {
        let key = pendingConfigKeyByConnectorID[connectorID] ?? ""
        let value = pendingConfigValueByConnectorID[connectorID] ?? ""
        state.addConnectorConfigDraftField(connectorID: connectorID, key: key, value: value)
        pendingConfigKeyByConnectorID[connectorID] = ""
        pendingConfigValueByConnectorID[connectorID] = ""
    }
}
