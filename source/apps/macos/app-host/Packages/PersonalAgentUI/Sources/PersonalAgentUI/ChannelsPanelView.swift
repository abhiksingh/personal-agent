import AppKit
import SwiftUI

struct ChannelsPanelView: View {
    @ObservedObject private var state: AppShellState
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @State private var pendingConfigKeyByChannelID: [String: String] = [:]
    @State private var pendingConfigValueByChannelID: [String: String] = [:]
    @State private var advancedExpandedByChannelID: [String: Bool] = [:]

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
                    state.performPanelProblemRemediationAction(actionID, section: .channels)
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
                title: "Loading Channels",
                subtitle: "Fetching channel status and connector mappings.",
                rowCount: 3
            )
            .frame(maxWidth: .infinity, maxHeight: .infinity)
        } else if logicalChannelCards.isEmpty {
            PanelRemediationEmptyStateView(
                title: "No Channels Found",
                systemImage: "point.3.connected.trianglepath.dotted",
                description: "Channel status will appear once daemon channel inventory is available.",
                statusMessage: state.channelsStatusMessage,
                headerStatusMessage: state.channelsStatusMessage,
                actions: state.channelsEmptyStateRemediationActions
            ) { actionID in
                state.performEmptyStateRemediationAction(actionID)
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)
            .background(UIStyle.panelGradient)
        } else {
            ScrollView {
                LazyVStack(spacing: UIStyle.standardSpacing) {
                    ForEach(logicalChannelCards) { logicalCard in
                        channelCard(logicalCard)
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
        state.panelProblemRemediation(for: .channels)
    }

    private var isAdvancedInformationDensityEnabled: Bool {
        state.isAdvancedInformationDensityEnabled
    }

    private var logicalChannelCards: [LogicalChannelCardItem] {
        state.logicalChannelCards
    }

    private var showLoadingSkeleton: Bool {
        (state.isChannelsLoading || !state.hasLoadedChannelStatus) && state.channelCards.isEmpty
    }

    private var header: some View {
        TahoeSectionHeader(
            title: "Channels",
            subtitle: state.channelsStatusMessage ?? "Daemon-backed channel status and configuration"
        ) {
            HStack(spacing: 8) {
                if state.isChannelsLoading {
                    ProgressView()
                        .controlSize(.small)
                }

                if state.channelsHasUnsavedDraftChanges {
                    Text("Unsaved changes")
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(.orange)
                }

                Button("Discard All") {
                    state.discardDraftChanges(for: .channels)
                }
                .buttonStyle(.bordered)
                .disabled(!state.channelsHasUnsavedDraftChanges || isChannelBulkMutationInFlight)

                Button("Save All") {
                    state.saveAllDraftChanges(for: .channels)
                }
                .buttonStyle(.borderedProminent)
                .successSymbolEffect(
                    state.successNotificationPulse(for: "channels"),
                    reduceMotion: reduceMotion
                )
                .disabled(!state.channelsHasUnsavedDraftChanges || isChannelBulkMutationInFlight)

                Button {
                    state.refreshChannelCards()
                } label: {
                    Label("Refresh", systemImage: "arrow.clockwise")
                }
                .buttonStyle(.bordered)
                .disabled(state.isChannelsLoading)
            }
        }
    }

    @ViewBuilder
    private func channelCard(_ logicalCard: LogicalChannelCardItem) -> some View {
        if let card = state.channelCardItem(channelID: logicalCard.primaryChannelCardID) {
            GroupBox {
                DisclosureGroup(isExpanded: expansionBinding(for: card)) {
                VStack(alignment: .leading, spacing: 10) {
                    Divider()

                    if isAdvancedInformationDensityEnabled {
                        VStack(alignment: .leading, spacing: 8) {
                            ForEach(logicalCard.details.keys.sorted(), id: \.self) { key in
                                detailRow(key: key, value: logicalCard.details[key] ?? "")
                            }
                        }
                        Divider()
                    }

                    VStack(alignment: .leading, spacing: 8) {
                        channelConfigurationSection(card: card)

                        HStack(spacing: 8) {
                            Button("Reset Draft") {
                                state.resetChannelConfigDraft(channelID: card.id)
                            }
                            .buttonStyle(.bordered)
                            .disabled(state.channelConfigSaveInFlightIDs.contains(card.id) || state.isChannelsLoading)

                            Button(state.channelConfigSaveInFlightIDs.contains(card.id) ? "Saving…" : "Save Config") {
                                state.saveChannelConfiguration(channelID: card.id)
                            }
                            .buttonStyle(.borderedProminent)
                            .disabled(state.channelConfigSaveInFlightIDs.contains(card.id) || state.isChannelsLoading)
                        }

                        HStack(spacing: 8) {
                            Button(state.channelTestInFlightIDs.contains(card.id) ? "Running…" : "Run Health Check") {
                                state.runChannelHealthCheck(channelID: card.id)
                            }
                            .buttonStyle(.bordered)
                            .disabled(
                                state.channelTestInFlightIDs.contains(card.id)
                                    || state.channelConfigSaveInFlightIDs.contains(card.id)
                                    || state.isChannelsLoading
                            )
                        }

                        if let status = state.channelConfigActionStatusByID[card.id] {
                            Text(status)
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }

                        if let testResult = state.channelLastTestResultByID[card.id] {
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

                    Divider()

                    deliveryPolicySection(logicalChannelID: logicalCard.channelID)

                    Divider()

                    channelConnectorMappingsSection(logicalCard: logicalCard)

                    Divider()

                    let actions = logicalCard.actions.filter { action in
                        state.canPerformChannelDiagnosticsAction(channelID: card.id, action: action)
                    }
                    if actions.isEmpty {
                        Text(logicalCard.unavailableActionReason)
                            .font(.caption)
                            .foregroundStyle(.secondary)

                        if !logicalCard.mappedConnectorRollups.isEmpty {
                            Button("Open Connectors") {
                                state.navigateToSection(.connectors)
                            }
                            .buttonStyle(.bordered)
                            .controlSize(.small)
                        }
                    } else {
                        ScrollView(.horizontal, showsIndicators: false) {
                            HStack(spacing: 8) {
                                ForEach(actions) { action in
                                    Button(action.title) {
                                        performChannelDiagnosticsAction(channelID: card.id, action: action)
                                    }
                                    .buttonStyle(.bordered)
                                    .disabled(
                                        !action.enabled
                                            || state.isChannelsLoading
                                            || state.isDaemonControlInFlight
                                    )
                                }
                            }
                        }
                        .frame(maxWidth: .infinity, alignment: .leading)

                        if let disabledReason = actions.first(where: { !$0.enabled })?.reason {
                            Text(disabledReason)
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
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
                        if channelCardHasUnsavedChanges(card: card, logicalCard: logicalCard) {
                            Text("Unsaved draft changes")
                                .font(.caption2.weight(.semibold))
                                .foregroundStyle(.orange)
                        }
                    }

                    Spacer(minLength: 0)

                    TahoeStatusBadge(
                        text: logicalCard.status.label,
                        symbolName: logicalCard.status.symbolName,
                        tint: logicalCard.status.tint
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
                    Text("Channel data is loading. Refresh to resolve logical card mapping.")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }
            .groupBoxStyle(.automatic)
        }
    }

    private var isChannelBulkMutationInFlight: Bool {
        state.isChannelsLoading ||
            !state.channelConfigSaveInFlightIDs.isEmpty ||
            !state.channelDeliveryPolicySaveInFlightIDs.isEmpty ||
            !state.channelConnectorMappingSaveInFlightChannelIDs.isEmpty
    }

    private func channelCardHasUnsavedChanges(
        card: ChannelCardItem,
        logicalCard: LogicalChannelCardItem
    ) -> Bool {
        state.channelConfigHasDraftChanges(channelID: card.id) ||
            state.channelDeliveryPolicyHasDraftChanges(channelID: logicalCard.channelID) ||
            state.channelConnectorMappingHasDraftChanges(channelID: logicalCard.channelID)
    }

    private func deliveryPolicySection(logicalChannelID: String) -> some View {
        let policies = state.channelDeliveryPolicies(channelID: logicalChannelID)
        let draft = state.channelDeliveryPolicyDraft(channelID: logicalChannelID)
        let routeOptions = state.channelDeliveryRouteOptions(channelID: logicalChannelID)
        let isSaving = state.channelDeliveryPolicySaveInFlightIDs.contains(logicalChannelID)

        return VStack(alignment: .leading, spacing: 8) {
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                Text("Delivery Policy")
                    .font(.subheadline.weight(.semibold))
                Text("\(policies.count)")
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(.secondary)
                Spacer(minLength: 0)
                Button("New Policy") {
                    state.startNewChannelDeliveryPolicyDraft(channelID: logicalChannelID)
                }
                .buttonStyle(.bordered)
                .controlSize(.small)
            }

            if policies.isEmpty {
                Text("No daemon policies found for this source channel. Save one to control retry and fallback behavior.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            } else {
                ForEach(policies) { policy in
                    HStack(alignment: .firstTextBaseline, spacing: 8) {
                        VStack(alignment: .leading, spacing: 2) {
                            Text(policy.endpointPattern ?? "Default Endpoint Match")
                                .font(.caption.weight(.semibold))
                            Text(
                                "\(policy.primaryChannel) • retry \(policy.retryCount) • fallback \(policy.fallbackChannels.isEmpty ? "none" : policy.fallbackChannels.joined(separator: ", "))"
                            )
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                        }
                        Spacer(minLength: 0)
                        if policy.isDefault {
                            TahoeStatusBadge(
                                text: "Default",
                                symbolName: "star.fill",
                                tint: .secondary
                            )
                        }
                        Button("Edit") {
                            state.loadChannelDeliveryPolicyDraft(channelID: logicalChannelID, policyID: policy.id)
                        }
                        .buttonStyle(.bordered)
                        .controlSize(.small)
                    }
                }
            }

            TextField(
                "Endpoint pattern (optional)",
                text: channelDeliveryEndpointPatternBinding(channelID: logicalChannelID)
            )
            .textFieldStyle(.roundedBorder)

            HStack(spacing: 8) {
                Picker("Primary Channel", selection: channelDeliveryPrimaryBinding(channelID: logicalChannelID)) {
                    ForEach(routeOptions, id: \.self) { option in
                        Text(option).tag(option)
                    }
                }
                .pickerStyle(.menu)
                .frame(maxWidth: 220)

                Stepper(
                    value: channelDeliveryRetryCountBinding(channelID: logicalChannelID),
                    in: 0...5
                ) {
                    Text("Retry Count: \(draft.retryCount)")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }

            TextField(
                "Fallback channels (comma separated)",
                text: channelDeliveryFallbackBinding(channelID: logicalChannelID)
            )
            .textFieldStyle(.roundedBorder)

            Toggle("Mark as default policy", isOn: channelDeliveryIsDefaultBinding(channelID: logicalChannelID))
                .toggleStyle(.switch)

            HStack(spacing: 8) {
                Button("Reset Policy Draft") {
                    state.resetChannelDeliveryPolicyDraft(channelID: logicalChannelID)
                }
                .buttonStyle(.bordered)
                .disabled(isSaving)

                Button(isSaving ? "Saving…" : "Save Policy") {
                    state.requestSaveChannelDeliveryPolicy(channelID: logicalChannelID)
                }
                .buttonStyle(.borderedProminent)
                .disabled(isSaving)
            }

            if let status = state.channelDeliveryPolicyActionStatusByID[logicalChannelID] {
                Text(status)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
    }

    private func channelConnectorMappingsSection(logicalCard: LogicalChannelCardItem) -> some View {
        let logicalChannelID = logicalCard.channelID
        let mappings = state.channelConnectorMappings(channelID: logicalChannelID)
        let isSaving = state.channelConnectorMappingSaveInFlightChannelIDs.contains(logicalChannelID)

        return VStack(alignment: .leading, spacing: 8) {
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                Text("Connector Mapping")
                    .font(.subheadline.weight(.semibold))
                Text("\(mappings.count)")
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(.secondary)
                Spacer(minLength: 0)
                if state.isChannelConnectorMappingsLoading {
                    ProgressView()
                        .controlSize(.small)
                }
                Button("Reset Draft") {
                    state.resetChannelConnectorMappingDraft(channelID: logicalChannelID)
                }
                .buttonStyle(.bordered)
                .controlSize(.small)
                .disabled(isSaving || state.isChannelConnectorMappingsLoading)
                Button(isSaving ? "Saving…" : "Save Mapping") {
                    state.saveChannelConnectorMappings(channelID: logicalChannelID)
                }
                .buttonStyle(.borderedProminent)
                .controlSize(.small)
                .disabled(isSaving || !state.channelConnectorMappingHasDraftChanges(channelID: logicalChannelID))
            }

            Text(state.channelConnectorMappingConstraintsSummary(channelID: logicalChannelID))
                .font(.caption)
                .foregroundStyle(.secondary)

            if mappings.isEmpty {
                Text("No connector mappings are available yet for this logical channel.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            } else {
                ForEach(mappings) { mapping in
                    VStack(alignment: .leading, spacing: 6) {
                        HStack(spacing: 8) {
                            Toggle(
                                isOn: channelConnectorMappingEnabledBinding(
                                    channelID: logicalChannelID,
                                    connectorID: mapping.connectorID
                                )
                            ) {
                                VStack(alignment: .leading, spacing: 2) {
                                    Text(state.channelConnectorDisplayName(connectorID: mapping.connectorID))
                                        .font(.caption.weight(.semibold))
                                    if isAdvancedInformationDensityEnabled {
                                        Text("\(mapping.connectorID) • priority \(mapping.priority)")
                                            .font(.caption2)
                                            .foregroundStyle(.secondary)
                                    } else {
                                        Text("Priority \(mapping.priority)")
                                            .font(.caption2)
                                            .foregroundStyle(.secondary)
                                    }
                                }
                            }
                            .toggleStyle(.switch)
                            .disabled(isSaving || state.isChannelConnectorMappingsLoading)

                            Spacer(minLength: 0)

                            Button {
                                state.moveChannelConnectorMappingUp(
                                    channelID: logicalChannelID,
                                    connectorID: mapping.connectorID
                                )
                            } label: {
                                Image(systemName: "chevron.up")
                            }
                            .buttonStyle(.bordered)
                            .controlSize(.small)
                            .accessibilityLabel(
                                "Move \(state.channelConnectorDisplayName(connectorID: mapping.connectorID)) up"
                            )
                            .accessibilityHint("Raises connector priority for this logical channel.")
                            .disabled(
                                isSaving
                                    || !state.canMoveChannelConnectorMapping(
                                        channelID: logicalChannelID,
                                        connectorID: mapping.connectorID,
                                        direction: -1
                                    )
                            )

                            Button {
                                state.moveChannelConnectorMappingDown(
                                    channelID: logicalChannelID,
                                    connectorID: mapping.connectorID
                                )
                            } label: {
                                Image(systemName: "chevron.down")
                            }
                            .buttonStyle(.bordered)
                            .controlSize(.small)
                            .accessibilityLabel(
                                "Move \(state.channelConnectorDisplayName(connectorID: mapping.connectorID)) down"
                            )
                            .accessibilityHint("Lowers connector priority for this logical channel.")
                            .disabled(
                                isSaving
                                    || !state.canMoveChannelConnectorMapping(
                                        channelID: logicalChannelID,
                                        connectorID: mapping.connectorID,
                                        direction: 1
                                    )
                            )
                        }

                        if isAdvancedInformationDensityEnabled && !mapping.capabilities.isEmpty {
                            Text("Capabilities: \(mapping.capabilities.joined(separator: ", "))")
                                .font(.caption2)
                                .foregroundStyle(.secondary)
                        }
                    }
                }
            }

            Text("Fallback policy: \(state.channelConnectorMappingFallbackPolicy)")
                .font(.caption2)
                .foregroundStyle(.secondary)

            if let status = state.channelConnectorMappingStatusMessage(channelID: logicalChannelID) {
                Text(status)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
    }

    private func channelConfigurationSection(card: ChannelCardItem) -> some View {
        let descriptors = state.channelConfigFieldDescriptors(channelID: card.id)
        return VStack(alignment: .leading, spacing: 10) {
            Text("Configuration")
                .font(.subheadline.weight(.semibold))

            if descriptors.isEmpty {
                Text("No guided configuration descriptors are available yet. Use Advanced to edit raw key/value fields.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            } else {
                ForEach(descriptors) { descriptor in
                    guidedChannelConfigFieldRow(channelID: card.id, descriptor: descriptor)
                }
            }

            DisclosureGroup(
                isExpanded: advancedChannelConfigExpansionBinding(channelID: card.id)
            ) {
                VStack(alignment: .leading, spacing: 8) {
                    Text("Raw key-value fallback for fields not covered by guided descriptors.")
                        .font(.caption)
                        .foregroundStyle(.secondary)

                    let advancedDraftKeys = state.channelAdvancedConfigDraftKeys(channelID: card.id)
                    let readOnlyKeys = advancedReadOnlyChannelConfigKeys(card: card)
                    if advancedDraftKeys.isEmpty && readOnlyKeys.isEmpty {
                        Text("No advanced-only configuration fields are present.")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    } else {
                        ForEach(advancedDraftKeys, id: \.self) { key in
                            VStack(alignment: .leading, spacing: 4) {
                                HStack(alignment: .center, spacing: 8) {
                                    Text(key)
                                        .font(.caption.weight(.semibold))
                                        .foregroundStyle(.secondary)
                                        .frame(width: 150, alignment: .leading)
                                    if state.channelConfigDraftKind(channelID: card.id, key: key).supportsInlineEditing {
                                        TextField(
                                            "Value",
                                            text: channelConfigDraftBinding(channelID: card.id, key: key)
                                        )
                                        .textFieldStyle(.roundedBorder)
                                    } else {
                                        Text(state.channelConfigDraftValue(channelID: card.id, key: key))
                                            .font(.callout)
                                            .frame(maxWidth: .infinity, alignment: .leading)
                                    }
                                    Button {
                                        state.removeChannelConfigDraftField(channelID: card.id, key: key)
                                    } label: {
                                        Image(systemName: "minus.circle")
                                    }
                                    .buttonStyle(.plain)
                                    .accessibilityLabel("Remove advanced field \(key)")
                                    .accessibilityHint("Deletes this draft configuration field.")
                                }
                                Text("Type: \(state.channelConfigDraftKind(channelID: card.id, key: key).label)")
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
                            text: pendingChannelConfigKeyBinding(channelID: card.id)
                        )
                        .textFieldStyle(.roundedBorder)
                        TextField(
                            "Config value",
                            text: pendingChannelConfigValueBinding(channelID: card.id)
                        )
                        .textFieldStyle(.roundedBorder)
                        Button("Add Field") {
                            addChannelConfigField(channelID: card.id)
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
    private func guidedChannelConfigFieldRow(
        channelID: String,
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
                        selection: channelConfigDraftBinding(channelID: channelID, key: descriptor.key)
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
                        isOn: channelConfigBoolBinding(channelID: channelID, key: descriptor.key)
                    )
                    .toggleStyle(.switch)
                } else if descriptor.secret {
                    SecureField(
                        descriptor.writeOnly ? "Enter new secret value" : "Secret value",
                        text: channelConfigDraftBinding(channelID: channelID, key: descriptor.key)
                    )
                    .textFieldStyle(.roundedBorder)
                } else {
                    TextField(
                        descriptor.draftKind == .number ? "Number" : "Value",
                        text: channelConfigDraftBinding(channelID: channelID, key: descriptor.key)
                    )
                    .textFieldStyle(.roundedBorder)
                }
            } else {
                Text(nonEmptyStateValue(state.channelConfigDraftValue(channelID: channelID, key: descriptor.key)))
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

    private func advancedReadOnlyChannelConfigKeys(card: ChannelCardItem) -> [String] {
        let guidedKeys = Set(state.channelGuidedConfigFieldKeys(channelID: card.id))
        return card.readOnlyConfiguration.keys
            .filter { !guidedKeys.contains($0) }
            .sorted()
    }

    private func advancedChannelConfigExpansionBinding(channelID: String) -> Binding<Bool> {
        Binding(
            get: { advancedExpandedByChannelID[channelID] ?? false },
            set: { advancedExpandedByChannelID[channelID] = $0 }
        )
    }

    private func channelConfigBoolBinding(channelID: String, key: String) -> Binding<Bool> {
        Binding(
            get: {
                parseBool(state.channelConfigDraftValue(channelID: channelID, key: key)) ?? false
            },
            set: { value in
                state.setChannelConfigDraftValue(channelID: channelID, key: key, value: value ? "true" : "false")
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

    private func expansionBinding(for card: ChannelCardItem) -> Binding<Bool> {
        Binding(
            get: { card.isExpanded },
            set: { expanded in
                guard expanded != card.isExpanded else {
                    return
                }
                state.toggleChannelCard(card.id)
            }
        )
    }

    private func channelConfigDraftBinding(channelID: String, key: String) -> Binding<String> {
        Binding(
            get: { state.channelConfigDraftValue(channelID: channelID, key: key) },
            set: { newValue in
                state.setChannelConfigDraftValue(channelID: channelID, key: key, value: newValue)
            }
        )
    }

    private func channelConnectorMappingEnabledBinding(
        channelID: String,
        connectorID: String
    ) -> Binding<Bool> {
        Binding(
            get: {
                state.channelConnectorMappings(channelID: channelID)
                    .first(where: { $0.connectorID == connectorID })?.enabled ?? false
            },
            set: { enabled in
                state.setChannelConnectorMappingEnabled(
                    channelID: channelID,
                    connectorID: connectorID,
                    enabled: enabled
                )
            }
        )
    }

    private func channelDeliveryPrimaryBinding(channelID: String) -> Binding<String> {
        Binding(
            get: { state.channelDeliveryPolicyDraft(channelID: channelID).primaryChannel },
            set: { newValue in
                state.setChannelDeliveryPolicyPrimaryChannel(channelID: channelID, primaryChannel: newValue)
            }
        )
    }

    private func channelDeliveryEndpointPatternBinding(channelID: String) -> Binding<String> {
        Binding(
            get: { state.channelDeliveryPolicyDraft(channelID: channelID).endpointPattern },
            set: { newValue in
                state.setChannelDeliveryPolicyEndpointPattern(channelID: channelID, endpointPattern: newValue)
            }
        )
    }

    private func channelDeliveryRetryCountBinding(channelID: String) -> Binding<Int> {
        Binding(
            get: { state.channelDeliveryPolicyDraft(channelID: channelID).retryCount },
            set: { newValue in
                state.setChannelDeliveryPolicyRetryCount(channelID: channelID, retryCount: newValue)
            }
        )
    }

    private func channelDeliveryFallbackBinding(channelID: String) -> Binding<String> {
        Binding(
            get: { state.channelDeliveryPolicyDraft(channelID: channelID).fallbackChannelsText },
            set: { newValue in
                state.setChannelDeliveryPolicyFallbackChannelsText(channelID: channelID, fallbackChannelsText: newValue)
            }
        )
    }

    private func channelDeliveryIsDefaultBinding(channelID: String) -> Binding<Bool> {
        Binding(
            get: { state.channelDeliveryPolicyDraft(channelID: channelID).isDefault },
            set: { newValue in
                state.setChannelDeliveryPolicyIsDefault(channelID: channelID, isDefault: newValue)
            }
        )
    }

    private func pendingChannelConfigKeyBinding(channelID: String) -> Binding<String> {
        Binding(
            get: { pendingConfigKeyByChannelID[channelID] ?? "" },
            set: { pendingConfigKeyByChannelID[channelID] = $0 }
        )
    }

    private func pendingChannelConfigValueBinding(channelID: String) -> Binding<String> {
        Binding(
            get: { pendingConfigValueByChannelID[channelID] ?? "" },
            set: { pendingConfigValueByChannelID[channelID] = $0 }
        )
    }

    private func performChannelDiagnosticsAction(channelID: String, action: DiagnosticsActionItem) {
        if channelActionIntent(action) == "open_system_settings" {
            state.performChannelDiagnosticsAction(channelID: channelID, action: action)
            NSWorkspace.shared.open(
                state.systemSettingsURLForChannelAction(channelID: channelID, action: action)
            )
            return
        }
        state.performChannelDiagnosticsAction(channelID: channelID, action: action)
    }

    private func channelActionIntent(_ action: DiagnosticsActionItem) -> String {
        action.intent.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
    }

    private func addChannelConfigField(channelID: String) {
        let key = pendingConfigKeyByChannelID[channelID] ?? ""
        let value = pendingConfigValueByChannelID[channelID] ?? ""
        state.addChannelConfigDraftField(channelID: channelID, key: key, value: value)
        pendingConfigKeyByChannelID[channelID] = ""
        pendingConfigValueByChannelID[channelID] = ""
    }
}
