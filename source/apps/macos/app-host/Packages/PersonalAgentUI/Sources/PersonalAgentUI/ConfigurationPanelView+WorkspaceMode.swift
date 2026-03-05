import SwiftUI

extension ConfigurationPanelView {
    var workspaceModeContent: some View {
        ConfigurationWorkspaceModeContent {
            identityHubSection
        } persona: {
            chatPersonaPolicySection
        } delegation: {
            operatorDisclosure(
                title: "Delegation Rules",
                isExpanded: $isWorkspaceDelegationExpanded
            ) {
                delegationSection
            }
        } devices: {
            operatorDisclosure(
                title: "Identity Devices and Sessions",
                isExpanded: $isWorkspaceIdentityOperationsExpanded
            ) {
                identityDevicesAndSessionsSection
            }
        }
    }

    var identityHubSection: some View {
        VStack(alignment: .leading, spacing: 10) {
            sectionTitle("Identity Hub")

            Picker("Workspace", selection: workspaceSelectionBinding) {
                ForEach(state.identityWorkspaceOptions, id: \.self) { workspaceID in
                    Text(state.identityWorkspaceDisplayName(for: workspaceID)).tag(workspaceID)
                }
            }
            .pickerStyle(.menu)

            Picker("Active Principal", selection: $state.selectedPrincipal) {
                ForEach(state.principalOptions, id: \.self) { principal in
                    Text(state.principalOptionDisplayName(for: principal)).tag(principal)
                }
            }
            .pickerStyle(.menu)

            if let activeContext = state.identityActiveContext {
                GroupBox("Active Context") {
                    VStack(alignment: .leading, spacing: 6) {
                        identitySettingRow(
                            label: "Workspace",
                            displayValue: state.workspaceIdentityDisplayValue(for: activeContext.workspaceID),
                            labelWidth: 122
                        )
                        identitySettingRow(
                            label: "Principal",
                            displayValue: state.principalIdentityDisplayValue(for: activeContext.principalActorID),
                            labelWidth: 122
                        )
                        settingRow(label: "Workspace Source", value: activeContext.workspaceSource)
                        settingRow(label: "Principal Source", value: activeContext.principalSource)
                        settingRow(label: "Resolved", value: activeContext.workspaceResolved ? "true" : "false")
                        if let updated = activeContext.lastUpdatedLabel {
                            settingRow(label: "Last Updated", value: updated)
                        }
                    }
                }
            } else {
                Text("Active context has not been loaded yet.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            configurationInventoryGroup(
                title: "Workspace Directory (\(state.identityWorkspaceItems.count))",
                items: state.identityWorkspaceItems,
                isLoading: false,
                loadingMessage: "Loading workspace directory…",
                emptyMessage: "No workspace directory records returned for the active identity context."
            ) { workspace in
                identityWorkspaceRow(workspace)
            }

            configurationInventoryGroup(
                title: "Principal Directory (\(state.identityPrincipalItems.count))",
                items: state.identityPrincipalItems,
                isLoading: false,
                loadingMessage: "Loading principal directory…",
                emptyMessage: "No principal directory records returned for this workspace."
            ) { principal in
                identityPrincipalRow(principal)
            }

            HStack(spacing: 8) {
                Button("Refresh Identity Directory") {
                    state.refreshIdentityDirectory()
                }
                .buttonStyle(.bordered)

                if state.isIdentityDirectoryLoading || state.isPrincipalOptionsLoading {
                    ProgressView()
                        .controlSize(.small)
                }
            }

            if let message = state.identityStatusMessage ?? state.principalStatusMessage {
                Text(message)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .cardSurface(.standard)
    }

    var chatPersonaPolicySection: some View {
        VStack(alignment: .leading, spacing: 10) {
            sectionTitle("Chat Persona Policy")

            Text("Define assistant tone and guardrails for workspace, principal, and channel scopes.")
                .font(.caption)
                .foregroundStyle(.secondary)

            Picker("Scope", selection: $state.chatPersonaScopeType) {
                ForEach(ChatPersonaScopeType.allCases, id: \.rawValue) { scope in
                    Text(scope.title).tag(scope)
                }
            }
            .pickerStyle(.menu)
            .accessibilityIdentifier("configuration-chat-persona-scope-picker")

            Text(state.chatPersonaScopeType.subtitle)
                .font(.caption2)
                .foregroundStyle(.secondary)

            HStack(spacing: 8) {
                TahoeStatusBadge(
                    text: "Test Channel \(chatPersonaChannelLabel(state.chatPersonaResponseShapingChannelID))",
                    symbolName: "macwindow",
                    tint: .blue
                )
                .controlSize(.small)
                TahoeStatusBadge(
                    text: "Profile \(state.chatPersonaResponseShapingProfileID)",
                    symbolName: "wand.and.stars",
                    tint: .indigo
                )
                .controlSize(.small)
            }

            if let scopedChannelID = state.chatPersonaResolvedChannelID,
               scopedChannelID != state.chatPersonaResponseShapingChannelID {
                Text("`Test in Chat` validates app-channel shaping only. Use channel activity in Communications to validate \(chatPersonaChannelLabel(scopedChannelID)) profile behavior.")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }

            if personaScopeIncludesPrincipal {
                Picker("Principal", selection: $state.chatPersonaScopePrincipalActorID) {
                    ForEach(state.chatPersonaPrincipalOptions, id: \.self) { actorID in
                        Text(state.principalOptionDisplayName(for: actorID)).tag(actorID)
                    }
                }
                .pickerStyle(.menu)
                .accessibilityIdentifier("configuration-chat-persona-principal-picker")

                Button("Use Active Principal") {
                    state.useActivePrincipalForChatPersonaScope()
                }
                .buttonStyle(.bordered)
                .controlSize(.small)
                .disabled(state.selectedPrincipal.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
            }

            if personaScopeIncludesChannel {
                Picker("Channel", selection: $state.chatPersonaScopeChannelID) {
                    ForEach(state.chatPersonaChannelOptions, id: \.self) { channelID in
                        Text(chatPersonaChannelLabel(channelID)).tag(channelID)
                    }
                }
                .pickerStyle(.menu)
                .accessibilityIdentifier("configuration-chat-persona-channel-picker")
            }

            GroupBox("Simple Persona") {
                VStack(alignment: .leading, spacing: 8) {
                    TextField("Style prompt (required)", text: $state.chatPersonaStylePromptDraft, axis: .vertical)
                        .lineLimit(4...10)
                        .textFieldStyle(.roundedBorder)
                        .accessibilityIdentifier("configuration-chat-persona-style-prompt")

                    Text("Applied as the default persona prompt for the selected scope.")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }
            }

            operatorDisclosure(
                title: "Advanced Guardrails",
                isExpanded: $isChatPersonaAdvancedExpanded
            ) {
                GroupBox("Guardrails (one per line)") {
                    VStack(alignment: .leading, spacing: 8) {
                        ZStack(alignment: .topLeading) {
                            RoundedRectangle(cornerRadius: 8, style: .continuous)
                                .fill(Color.primary.opacity(0.03))
                            RoundedRectangle(cornerRadius: 8, style: .continuous)
                                .stroke(Color.primary.opacity(0.08), lineWidth: 1)

                            if state.chatPersonaGuardrailsDraft.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                                Text("Never reveal secrets\nAsk for clarification before destructive actions")
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                                    .padding(.horizontal, 10)
                                    .padding(.vertical, 10)
                            }

                            TextEditor(text: $state.chatPersonaGuardrailsDraft)
                                .scrollContentBackground(.hidden)
                                .padding(6)
                                .font(.caption)
                                .accessibilityIdentifier("configuration-chat-persona-guardrails")
                        }
                        .frame(minHeight: 90)

                        Text("Parsed \(state.chatPersonaNormalizedGuardrails.count) guardrail\(state.chatPersonaNormalizedGuardrails.count == 1 ? "" : "s"). Empty lines and duplicates are ignored.")
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                    }
                }
            }

            if let policy = state.chatPersonaPolicyItem {
                HStack(spacing: 8) {
                    TahoeStatusBadge(
                        text: "Source \(policy.source.capitalized)",
                        symbolName: policy.source == "persisted" ? "checkmark.circle.fill" : "sparkles",
                        tint: policy.source == "persisted" ? .green : .secondary
                    )
                    .controlSize(.small)
                    if let updatedAt = policy.updatedAtLabel {
                        TahoeStatusBadge(
                            text: "Updated \(updatedAt)",
                            symbolName: "clock.arrow.circlepath",
                            tint: .secondary
                        )
                        .controlSize(.small)
                    }
                }
            }

            HStack(spacing: 8) {
                Button("Refresh Scope") {
                    state.refreshChatPersonaPolicy()
                }
                .buttonStyle(.bordered)
                .disabled(state.isChatPersonaPolicyLoading || state.isChatPersonaPolicySaveInFlight)
                .accessibilityIdentifier("configuration-chat-persona-refresh")

                Button("Save Policy") {
                    state.saveChatPersonaPolicy()
                }
                .buttonStyle(.borderedProminent)
                .disabled(state.chatPersonaSaveDisabledReason != nil)
                .help(state.chatPersonaSaveDisabledReason ?? "Save persona policy for the selected scope.")
                .accessibilityIdentifier("configuration-chat-persona-save")

                Button("Reset Draft") {
                    state.resetChatPersonaPolicyDraft()
                }
                .buttonStyle(.bordered)
                .disabled(!state.chatPersonaHasDraftChanges && !state.chatPersonaHasLoadedPolicy)
                .accessibilityIdentifier("configuration-chat-persona-reset")

                Button("Test in Chat") {
                    state.testChatPersonaPolicyInChat()
                }
                .buttonStyle(.bordered)
                .disabled(state.chatPersonaStylePromptDraft.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                .accessibilityIdentifier("configuration-chat-persona-test")

                if state.isChatPersonaPolicyLoading || state.isChatPersonaPolicySaveInFlight {
                    ProgressView()
                        .controlSize(.small)
                }
            }

            if let message = state.chatPersonaPolicyStatusMessage {
                Text(message)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .cardSurface(.standard)
        .onAppear {
            if !state.chatPersonaHasLoadedPolicy {
                state.refreshChatPersonaPolicy()
            }
        }
        .onChange(of: state.chatPersonaScopeType) { _, _ in
            state.refreshChatPersonaPolicy()
        }
        .onChange(of: state.chatPersonaScopePrincipalActorID) { _, _ in
            if personaScopeIncludesPrincipal {
                state.refreshChatPersonaPolicy()
            }
        }
        .onChange(of: state.chatPersonaScopeChannelID) { _, _ in
            if personaScopeIncludesChannel {
                state.refreshChatPersonaPolicy()
            }
        }
    }

    var identityDevicesAndSessionsSection: some View {
        VStack(alignment: .leading, spacing: 10) {
            sectionTitle("Identity Devices and Sessions")

            GroupBox("Device Inventory Filters") {
                VStack(alignment: .leading, spacing: 8) {
                    TextField("User ID (optional)", text: $state.identityDeviceUserIDFilter)
                        .textFieldStyle(.roundedBorder)
                    TextField("Device Type (optional)", text: $state.identityDeviceTypeFilter)
                        .textFieldStyle(.roundedBorder)
                    TextField("Platform (optional)", text: $state.identityDevicePlatformFilter)
                        .textFieldStyle(.roundedBorder)
                    Stepper(
                        "Limit: \(state.identityDeviceLimit)",
                        value: $state.identityDeviceLimit,
                        in: 5...200,
                        step: 5
                    )

                    HStack(spacing: 8) {
                        Button("Refresh Devices") {
                            state.refreshIdentityDeviceInventory()
                        }
                        .buttonStyle(.bordered)
                        .disabled(state.isIdentityDeviceInventoryLoading)

                        Button("Reset Filters") {
                            state.resetIdentityDeviceInventoryFilters()
                            state.refreshIdentityDeviceInventory()
                        }
                        .buttonStyle(.bordered)
                        .disabled(state.isIdentityDeviceInventoryLoading)

                        if state.isIdentityDeviceInventoryLoading {
                            ProgressView()
                                .controlSize(.small)
                        }
                    }
                }
            }

            configurationSecondaryStatusMessage(state.identityDeviceInventoryStatusMessage)

            configurationInventoryGroup(
                title: "Devices (\(state.identityDeviceItems.count))",
                items: state.identityDeviceItems,
                isLoading: state.isIdentityDeviceInventoryLoading,
                loadingMessage: "Loading identity devices…",
                emptyMessage: "No identity devices matched the current filters.",
                hasMore: state.identityDeviceInventoryHasMore,
                hasMoreMessage: "More identity devices are available. Narrow filters for focused review."
            ) { item in
                identityDeviceRow(item)
            }

            GroupBox("Session Inventory Filters") {
                VStack(alignment: .leading, spacing: 8) {
                    TextField("Device ID (optional)", text: $state.identitySessionDeviceIDFilter)
                        .textFieldStyle(.roundedBorder)
                    TextField("User ID (optional)", text: $state.identitySessionUserIDFilter)
                        .textFieldStyle(.roundedBorder)
                    Picker("Session Health", selection: $state.identitySessionHealthFilter) {
                        ForEach(state.identitySessionHealthFilterOptions, id: \.self) { option in
                            Text(contextFilterOptionLabel(option)).tag(option)
                        }
                    }
                    .pickerStyle(.menu)
                    Stepper(
                        "Limit: \(state.identitySessionLimit)",
                        value: $state.identitySessionLimit,
                        in: 5...200,
                        step: 5
                    )

                    HStack(spacing: 8) {
                        Button("Refresh Sessions") {
                            state.refreshIdentitySessionInventory()
                        }
                        .buttonStyle(.bordered)
                        .disabled(state.isIdentitySessionInventoryLoading)

                        Button("Reset Filters") {
                            state.resetIdentitySessionInventoryFilters()
                            state.refreshIdentitySessionInventory()
                        }
                        .buttonStyle(.bordered)
                        .disabled(state.isIdentitySessionInventoryLoading)

                        if state.isIdentitySessionInventoryLoading {
                            ProgressView()
                                .controlSize(.small)
                        }
                    }
                }
            }

            configurationSecondaryStatusMessage(state.identitySessionInventoryStatusMessage)

            configurationInventoryGroup(
                title: "Sessions (\(state.identitySessionItems.count))",
                items: state.identitySessionItems,
                isLoading: state.isIdentitySessionInventoryLoading,
                loadingMessage: "Loading identity sessions…",
                emptyMessage: "No identity sessions matched the current filters.",
                hasMore: state.identitySessionInventoryHasMore,
                hasMoreMessage: "More identity sessions are available. Narrow filters for focused review."
            ) { item in
                identitySessionRow(item)
            }
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .cardSurface(.standard)
    }

    var delegationSection: some View {
        VStack(alignment: .leading, spacing: 10) {
            sectionTitle("Delegation Rules")

            GroupBox("Grant Delegation") {
                VStack(alignment: .leading, spacing: 8) {
                    Picker("From Actor", selection: $configurationDraftStore.delegationFromActorID) {
                        ForEach(delegationActorOptions, id: \.self) { actorID in
                            Text(state.principalOptionDisplayName(for: actorID)).tag(actorID)
                        }
                    }
                    .pickerStyle(.menu)

                    Picker("To Actor", selection: $configurationDraftStore.delegationToActorID) {
                        ForEach(delegationActorOptions, id: \.self) { actorID in
                            Text(state.principalOptionDisplayName(for: actorID)).tag(actorID)
                        }
                    }
                    .pickerStyle(.menu)

                    Picker("Scope Type", selection: $configurationDraftStore.delegationScopeType) {
                        ForEach(state.delegationScopeOptions, id: \.self) { scopeType in
                            Text(scopeType).tag(scopeType)
                        }
                    }
                    .pickerStyle(.menu)

                    if configurationDraftStore.delegationScopeType != "ALL" {
                        TextField("Scope Key (optional)", text: $configurationDraftStore.delegationScopeKey)
                            .textFieldStyle(.roundedBorder)
                    } else {
                        Text("Scope key is not allowed when scope type is ALL.")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }

                    TextField("Expires At (RFC3339, optional)", text: $configurationDraftStore.delegationExpiresAt)
                        .textFieldStyle(.roundedBorder)

                    HStack(spacing: 8) {
                        Button("Grant Rule") {
                            submitDelegationGrant()
                        }
                        .buttonStyle(.bordered)
                        .disabled(isDelegationGrantDisabled)

                        if state.isDelegationGrantInFlight {
                            ProgressView()
                                .controlSize(.small)
                        }
                    }
                }
            }

            GroupBox("Delegation Inventory (\(state.delegationRules.count))") {
                VStack(alignment: .leading, spacing: 8) {
                    if state.isDelegationLoading && state.delegationRules.isEmpty {
                        HStack(spacing: 8) {
                            ProgressView()
                                .controlSize(.small)
                            Text("Loading delegation rules…")
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                    } else if state.delegationRules.isEmpty {
                        Text("No delegation rules are configured for this workspace.")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    } else {
                        ForEach(state.delegationRules) { rule in
                            delegationRuleRow(rule)
                        }
                    }
                }
            }

            HStack(spacing: 8) {
                Button("Refresh Delegation Rules") {
                    state.refreshDelegationRules()
                }
                .buttonStyle(.bordered)
                .disabled(state.isDelegationLoading || state.isDelegationGrantInFlight)

                if state.isDelegationLoading {
                    ProgressView()
                        .controlSize(.small)
                }
            }

            if let message = state.delegationStatusMessage {
                Text(message)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .cardSurface(.standard)
        .onAppear {
            seedDelegationDraftIfNeeded()
        }
        .onChange(of: state.delegationPrincipalOptions) { _, _ in
            seedDelegationDraftIfNeeded()
        }
    }


    var personaScopeIncludesPrincipal: Bool {
        switch state.chatPersonaScopeType {
        case .principal, .principalChannel:
            return true
        case .workspace, .channel:
            return false
        }
    }

    var personaScopeIncludesChannel: Bool {
        switch state.chatPersonaScopeType {
        case .channel, .principalChannel:
            return true
        case .workspace, .principal:
            return false
        }
    }

    func chatPersonaChannelLabel(_ raw: String) -> String {
        switch raw.lowercased() {
        case "app":
            return "App"
        case "message":
            return "Message"
        case "voice":
            return "Voice"
        default:
            return raw.capitalized
        }
    }

    var delegationActorOptions: [String] {
        configurationDraftStore.delegationActorOptions(
            principalOptions: state.delegationPrincipalOptions
        )
    }

    var isDelegationGrantDisabled: Bool {
        configurationDraftStore.isDelegationGrantDisabled(
            isGrantInFlight: state.isDelegationGrantInFlight,
            isLoading: state.isDelegationLoading
        )
    }

    func seedDelegationDraftIfNeeded() {
        configurationDraftStore.seedDelegationDraftIfNeeded(
            principalOptions: state.delegationPrincipalOptions
        )
    }

    func submitDelegationGrant() {
        state.createDelegationRule(configurationDraftStore.delegationGrantInput())
    }

    func delegationRuleRow(_ rule: DelegationRuleItem) -> some View {
        let fromIdentity = state.principalIdentityDisplayValue(for: rule.fromActorID)
        let toIdentity = state.principalIdentityDisplayValue(for: rule.toActorID)
        return VStack(alignment: .leading, spacing: 6) {
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                HStack(alignment: .firstTextBaseline, spacing: 6) {
                    IdentityValueInlineView(
                        displayText: fromIdentity.displayText,
                        rawID: fromIdentity.rawID,
                        valueFont: .caption.weight(.semibold)
                    )
                    Text("→")
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(.secondary)
                    IdentityValueInlineView(
                        displayText: toIdentity.displayText,
                        rawID: toIdentity.rawID,
                        valueFont: .caption.weight(.semibold)
                    )
                }
                Spacer(minLength: 0)
                TahoeStatusBadge(
                    text: rule.status.uppercased() == "ACTIVE" ? "Active" : rule.status,
                    symbolName: rule.status.uppercased() == "ACTIVE" ? "checkmark.circle.fill" : "circle",
                    tint: rule.status.uppercased() == "ACTIVE" ? .green : .secondary
                )
                .controlSize(.small)
            }

            Text("Scope \(delegationScopeSummary(scopeType: rule.scopeType, scopeKey: rule.scopeKey))")
                .font(.caption2)
                .foregroundStyle(.secondary)
            Text("Created \(rule.createdAtLabel)")
                .font(.caption2)
                .foregroundStyle(.secondary)
            if let expiresAtLabel = rule.expiresAtLabel {
                Text("Expires \(expiresAtLabel)")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }

            HStack(spacing: 8) {
                Button("Revoke", role: .destructive) {
                    state.requestRevokeDelegationRule(ruleID: rule.id)
                }
                .buttonStyle(.bordered)
                .disabled(state.delegationRevokeInFlightRuleIDs.contains(rule.id) || state.isDelegationGrantInFlight)

                if state.delegationRevokeInFlightRuleIDs.contains(rule.id) {
                    ProgressView()
                        .controlSize(.small)
                }
            }

            if let actionStatus = state.delegationActionStatusByRuleID[rule.id] {
                Text(actionStatus)
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(8)
        .cardSurface(.subtle)
    }

    func delegationScopeSummary(scopeType: String, scopeKey: String?) -> String {
        configurationDraftStore.delegationScopeSummary(
            scopeType: scopeType,
            scopeKey: scopeKey
        )
    }

    var workspaceSelectionBinding: Binding<String> {
        Binding(
            get: { state.workspaceLabel },
            set: { selectedWorkspaceID in
                let trimmed = selectedWorkspaceID.trimmingCharacters(in: .whitespacesAndNewlines)
                guard !trimmed.isEmpty, trimmed != state.workspaceLabel else {
                    return
                }
                state.selectIdentityWorkspace(trimmed)
            }
        )
    }

    func identityWorkspaceRow(_ item: IdentityWorkspaceItem) -> some View {
        let workspaceIdentity = state.workspaceIdentityDisplayValue(for: item.id)
        return configurationRecordCard {
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                IdentityValueInlineView(
                    displayText: workspaceIdentity.displayText,
                    rawID: workspaceIdentity.rawID,
                    valueFont: .caption.weight(.semibold)
                )
                Spacer(minLength: 0)
                TahoeStatusBadge(
                    text: item.isActive ? "Active" : "Listed",
                    symbolName: item.isActive ? "checkmark.circle.fill" : "circle",
                    tint: item.isActive ? .green : .secondary
                )
                .controlSize(.small)
            }
            Text("Status \(item.status) • Principals \(item.principalCount) • Actors \(item.actorCount) • Handles \(item.handleCount)")
                .font(.caption2)
                .foregroundStyle(.secondary)
            Text("Updated \(item.updatedAtLabel)")
                .font(.caption2)
                .foregroundStyle(.secondary)
        }
    }

    func identityPrincipalRow(_ item: IdentityPrincipalItem) -> some View {
        let principalIdentity = state.principalIdentityDisplayValue(for: item.id)
        return configurationRecordCard {
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                IdentityValueInlineView(
                    displayText: principalIdentity.displayText,
                    rawID: principalIdentity.rawID,
                    valueFont: .caption.weight(.semibold)
                )
                Spacer(minLength: 0)
                TahoeStatusBadge(
                    text: item.isActive ? "Active" : "Listed",
                    symbolName: item.isActive ? "checkmark.circle.fill" : "circle",
                    tint: item.isActive ? .green : .secondary
                )
                .controlSize(.small)
            }
            Text("Actor \(item.actorType) • Actor Status \(item.actorStatus) • Principal Status \(item.principalStatus)")
                .font(.caption2)
                .foregroundStyle(.secondary)

            if item.handles.isEmpty {
                Text("No actor handles mapped for this principal.")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            } else {
                ForEach(item.handles) { handle in
                    HStack(alignment: .firstTextBaseline, spacing: 6) {
                        Text(handle.channel)
                            .font(.caption2.weight(.semibold))
                        Text(handle.handleValue)
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                            .textSelection(.enabled)
                        if handle.isPrimary {
                            Text("Primary")
                                .font(.caption2)
                                .foregroundStyle(.green)
                        }
                        Spacer(minLength: 0)
                        Text(handle.updatedAtLabel)
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                    }
                }
            }
        }
    }

    func identityDeviceRow(_ item: IdentityDeviceItem) -> some View {
        configurationRecordCard {
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                Text(item.label ?? item.id)
                    .font(.caption.weight(.semibold))
                Text(item.id)
                    .font(.caption2)
                    .foregroundStyle(.secondary)
                    .textSelection(.enabled)
                Spacer(minLength: 0)
                TahoeStatusBadge(
                    text: "\(item.sessionActiveCount) Active",
                    symbolName: item.sessionActiveCount > 0 ? "checkmark.circle.fill" : "circle",
                    tint: item.sessionActiveCount > 0 ? .green : .secondary
                )
                .controlSize(.small)
            }
            Text("User \(item.userID) • \(item.deviceType) • \(item.platform)")
                .font(.caption2)
                .foregroundStyle(.secondary)
            Text("Sessions \(item.sessionTotal) • active \(item.sessionActiveCount) • expired \(item.sessionExpiredCount) • revoked \(item.sessionRevokedCount)")
                .font(.caption2)
                .foregroundStyle(.secondary)
            if let lastSeen = item.lastSeenAtLabel {
                Text("Last seen \(lastSeen)")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }
            if let latestSession = item.sessionLatestStartedAtLabel {
                Text("Latest session start \(latestSession)")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }
            Text("Created \(item.createdAtLabel)")
                .font(.caption2)
                .foregroundStyle(.secondary)
        }
    }

    func identitySessionRow(_ item: IdentitySessionItem) -> some View {
        configurationRecordCard {
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                Text(item.id)
                    .font(.caption.weight(.semibold))
                    .textSelection(.enabled)
                Spacer(minLength: 0)
                TahoeStatusBadge(
                    text: identitySessionHealthLabel(item.sessionHealth),
                    symbolName: identitySessionHealthSymbol(item.sessionHealth),
                    tint: identitySessionHealthTint(item.sessionHealth)
                )
                .controlSize(.small)
            }
            Text("Device \(item.deviceLabel ?? item.deviceID) • User \(item.userID) • \(item.deviceType) • \(item.platform)")
                .font(.caption2)
                .foregroundStyle(.secondary)
            Text("Started \(item.startedAtLabel) • Expires \(item.expiresAtLabel)")
                .font(.caption2)
                .foregroundStyle(.secondary)
            if let revokedAt = item.revokedAtLabel {
                Text("Revoked \(revokedAt)")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }
            if let lastSeen = item.deviceLastSeenAtLabel {
                Text("Device last seen \(lastSeen)")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }

            HStack(spacing: 8) {
                Button(identitySessionRevokeButtonTitle(item.sessionHealth)) {
                    state.requestRevokeIdentitySession(sessionID: item.id)
                }
                .buttonStyle(.bordered)
                .disabled(!item.canRevoke || state.identitySessionRevokeInFlightIDs.contains(item.id))

                if state.identitySessionRevokeInFlightIDs.contains(item.id) {
                    ProgressView()
                        .controlSize(.small)
                }
            }

            if let message = state.identitySessionActionStatusByID[item.id] {
                Text(message)
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }
        }
    }

    func identitySessionHealthLabel(_ raw: String) -> String {
        switch raw.lowercased() {
        case "active":
            return "Active"
        case "expired":
            return "Expired"
        case "revoked":
            return "Revoked"
        default:
            return contextFilterOptionLabel(raw)
        }
    }

    func identitySessionHealthSymbol(_ raw: String) -> String {
        switch raw.lowercased() {
        case "active":
            return "checkmark.circle.fill"
        case "expired":
            return "clock.badge.exclamationmark"
        case "revoked":
            return "xmark.circle.fill"
        default:
            return "questionmark.circle.fill"
        }
    }

    func identitySessionHealthTint(_ raw: String) -> Color {
        switch raw.lowercased() {
        case "active":
            return .green
        case "expired":
            return .orange
        case "revoked":
            return .secondary
        default:
            return .secondary
        }
    }

    func identitySessionRevokeButtonTitle(_ rawHealth: String) -> String {
        switch rawHealth.lowercased() {
        case "active":
            return "Revoke Session"
        case "revoked":
            return "Session Revoked"
        default:
            return "Revoke Unavailable"
        }
    }
}
