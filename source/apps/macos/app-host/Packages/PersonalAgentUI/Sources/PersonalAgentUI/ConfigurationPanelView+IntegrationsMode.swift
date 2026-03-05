import SwiftUI

extension ConfigurationPanelView {
    var integrationsModeContent: some View {
        ConfigurationIntegrationsModeContent {
            operatorDisclosure(
                title: "Capability Grants Governance",
                isExpanded: $isCapabilityGrantsExpanded
            ) {
                capabilityGrantsSection
            }
        } trust: {
            operatorDisclosure(
                title: "Communication Trust Receipts",
                isExpanded: $isTrustReceiptsExpanded
            ) {
                trustReceiptsSection
            }
        }
    }

    var capabilityGrantsSection: some View {
        VStack(alignment: .leading, spacing: 10) {
            GroupBox("Upsert Capability Grant") {
                VStack(alignment: .leading, spacing: 8) {
                    TextField("Grant ID (optional for update/revoke)", text: $configurationDraftStore.capabilityGrantDraftGrantID)
                        .textFieldStyle(.roundedBorder)

                    Picker("Actor", selection: $configurationDraftStore.capabilityGrantDraftActorID) {
                        ForEach(capabilityGrantActorOptions, id: \.self) { actorID in
                            Text(state.principalOptionDisplayName(for: actorID)).tag(actorID)
                        }
                    }
                    .pickerStyle(.menu)

                    TextField("Capability Key", text: $configurationDraftStore.capabilityGrantDraftCapabilityKey)
                        .textFieldStyle(.roundedBorder)

                    Picker("Status", selection: $configurationDraftStore.capabilityGrantDraftStatus) {
                        ForEach(state.capabilityGrantStatusFilterOptions.filter { $0.lowercased() != "all" }, id: \.self) { status in
                            Text(contextFilterOptionLabel(status)).tag(status)
                        }
                    }
                    .pickerStyle(.menu)

                    VStack(alignment: .leading, spacing: 8) {
                        Text("Scope (Guided)")
                            .font(.caption.weight(.semibold))
                            .foregroundStyle(.secondary)

                        if configurationDraftStore.capabilityGrantScopeEntries.isEmpty {
                            Text("No scope entries. Grant applies broadly unless raw override is enabled.")
                                .font(.caption2)
                                .foregroundStyle(.secondary)
                        } else {
                            ForEach(configurationDraftStore.capabilityGrantScopeEntries) { entry in
                                HStack(spacing: 8) {
                                    TextField(
                                        "key",
                                        text: Binding(
                                            get: { scopeEntry(for: entry.id)?.key ?? entry.key },
                                            set: { updateCapabilityGrantScopeEntry(entry.id, key: $0, value: nil) }
                                        )
                                    )
                                    .textFieldStyle(.roundedBorder)

                                    TextField(
                                        "value",
                                        text: Binding(
                                            get: { scopeEntry(for: entry.id)?.value ?? entry.value },
                                            set: { updateCapabilityGrantScopeEntry(entry.id, key: nil, value: $0) }
                                        )
                                    )
                                    .textFieldStyle(.roundedBorder)

                                    Button {
                                        removeCapabilityGrantScopeEntry(entry.id)
                                    } label: {
                                        Image(systemName: "minus.circle")
                                    }
                                    .buttonStyle(.plain)
                                    .accessibilityLabel("Remove scope entry")
                                }
                            }
                        }

                        HStack(spacing: 8) {
                            TextField("key", text: $configurationDraftStore.capabilityGrantScopeDraftKey)
                                .textFieldStyle(.roundedBorder)
                            TextField("value", text: $configurationDraftStore.capabilityGrantScopeDraftValue)
                                .textFieldStyle(.roundedBorder)
                            Button("Add Entry") {
                                addCapabilityGrantScopeEntry()
                            }
                            .buttonStyle(.bordered)
                            .disabled(
                                configurationDraftStore.capabilityGrantScopeDraftKey
                                    .trimmingCharacters(in: .whitespacesAndNewlines)
                                    .isEmpty
                                || configurationDraftStore.capabilityGrantScopeDraftValue
                                    .trimmingCharacters(in: .whitespacesAndNewlines)
                                    .isEmpty
                            )
                        }

                        HStack(spacing: 8) {
                            Button("Reset Scope") {
                                resetCapabilityGrantScopeEntries()
                            }
                            .buttonStyle(.bordered)
                            .disabled(configurationDraftStore.capabilityGrantScopeEntries.isEmpty)

                            Button("Sync to Raw JSON") {
                                syncCapabilityGrantRawScopeFromGuided()
                            }
                            .buttonStyle(.bordered)
                        }
                    }

                    DisclosureGroup("Advanced Raw Scope JSON", isExpanded: $configurationDraftStore.isCapabilityGrantRawScopeExpanded) {
                        VStack(alignment: .leading, spacing: 8) {
                            Toggle("Use raw JSON override when saving", isOn: $configurationDraftStore.useCapabilityGrantRawScopeOverride)
                                .toggleStyle(.switch)

                            TextEditor(text: $configurationDraftStore.capabilityGrantDraftScopeJSON)
                                .font(.system(.caption, design: .monospaced))
                                .frame(minHeight: 92)
                                .overlay(
                                    RoundedRectangle(cornerRadius: 8)
                                        .stroke(Color.secondary.opacity(0.25))
                                )

                            HStack(spacing: 8) {
                                Button("Load from Guided") {
                                    syncCapabilityGrantRawScopeFromGuided()
                                }
                                .buttonStyle(.bordered)

                                Button("Apply to Guided") {
                                    applyCapabilityGrantRawScopeToGuided()
                                }
                                .buttonStyle(.bordered)
                                .disabled(!isCapabilityGrantRawScopeJSONValid)
                            }

                            Text(capabilityGrantRawScopeValidationMessage)
                                .font(.caption)
                                .foregroundStyle(
                                    isCapabilityGrantRawScopeJSONValid ? Color.secondary : Color.orange
                                )
                        }
                        .padding(.top, 4)
                    }

                    TextField("Expires At (RFC3339, optional)", text: $configurationDraftStore.capabilityGrantDraftExpiresAt)
                        .textFieldStyle(.roundedBorder)

                    HStack(spacing: 8) {
                        Button("Save Capability Grant") {
                            submitCapabilityGrantUpsert()
                        }
                        .buttonStyle(.bordered)
                        .disabled(isCapabilityGrantMutationDisabled)

                        if state.isCapabilityGrantMutationInFlight {
                            ProgressView()
                                .controlSize(.small)
                        }
                    }
                }
            }

            if let message = state.capabilityGrantMutationStatusMessage {
                Text(message)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            GroupBox("Inventory Filters") {
                VStack(alignment: .leading, spacing: 8) {
                    TextField("Actor ID (optional)", text: $state.capabilityGrantActorFilter)
                        .textFieldStyle(.roundedBorder)

                    TextField("Capability Key (optional)", text: $state.capabilityGrantKeyFilter)
                        .textFieldStyle(.roundedBorder)

                    Picker("Status", selection: $state.capabilityGrantStatusFilter) {
                        ForEach(state.capabilityGrantStatusFilterOptions, id: \.self) { option in
                            Text(contextFilterOptionLabel(option)).tag(option)
                        }
                    }
                    .pickerStyle(.menu)

                    Stepper(
                        "Limit: \(state.capabilityGrantLimit)",
                        value: $state.capabilityGrantLimit,
                        in: 5...200,
                        step: 5
                    )

                    HStack(spacing: 8) {
                        Button("Refresh Grants") {
                            state.refreshCapabilityGrantInventory()
                        }
                        .buttonStyle(.bordered)
                        .disabled(state.isCapabilityGrantInventoryLoading || state.isCapabilityGrantMutationInFlight)

                        Button("Reset Filters") {
                            state.resetCapabilityGrantFilters()
                            state.refreshCapabilityGrantInventory()
                        }
                        .buttonStyle(.bordered)
                        .disabled(state.isCapabilityGrantInventoryLoading || state.isCapabilityGrantMutationInFlight)

                        if state.isCapabilityGrantInventoryLoading {
                            ProgressView()
                                .controlSize(.small)
                        }
                    }
                }
            }

            if let message = state.capabilityGrantStatusMessage {
                Text(message)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            GroupBox("Capability Grants (\(state.capabilityGrantItems.count))") {
                VStack(alignment: .leading, spacing: 8) {
                    if state.isCapabilityGrantInventoryLoading && state.capabilityGrantItems.isEmpty {
                        HStack(spacing: 8) {
                            ProgressView()
                                .controlSize(.small)
                            Text("Loading capability grants…")
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                    } else if state.capabilityGrantItems.isEmpty {
                        Text("No capability grants matched the current filters.")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    } else {
                        ForEach(Array(state.capabilityGrantItems.enumerated()), id: \.element.id) { index, item in
                            if index > 0 {
                                Divider()
                            }
                            capabilityGrantRow(item)
                        }
                    }

                    if state.capabilityGrantInventoryHasMore {
                        Text("More capability grants are available. Narrow filters for targeted governance review.")
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                    }
                }
            }
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .cardSurface(.standard)
        .onAppear {
            seedCapabilityGrantDraftIfNeeded()
        }
        .onChange(of: state.delegationPrincipalOptions) { _, _ in
            seedCapabilityGrantDraftIfNeeded()
        }
    }

    var trustReceiptsSection: some View {
        VStack(alignment: .leading, spacing: 10) {
            Picker("Receipt Inventory", selection: $configurationDraftStore.selectedTrustReceiptInventory) {
                ForEach(TrustReceiptInventoryKind.allCases) { option in
                    Text(option.label).tag(option)
                }
            }
            .pickerStyle(.segmented)

            switch configurationDraftStore.selectedTrustReceiptInventory {
            case .webhook:
                webhookTrustReceiptsInventory
            case .ingest:
                ingestTrustReceiptsInventory
            }
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .cardSurface(.standard)
    }

    var webhookTrustReceiptsInventory: some View {
        VStack(alignment: .leading, spacing: 10) {
            GroupBox("Webhook Filters") {
                VStack(alignment: .leading, spacing: 8) {
                    TextField("Provider (optional)", text: $state.webhookReceiptProviderFilter)
                        .textFieldStyle(.roundedBorder)

                    TextField("Provider Event ID (optional)", text: $state.webhookReceiptProviderEventIDFilter)
                        .textFieldStyle(.roundedBorder)

                    TextField("Provider Event Query (optional)", text: $state.webhookReceiptProviderEventQueryFilter)
                        .textFieldStyle(.roundedBorder)

                    TextField("Event ID (optional)", text: $state.webhookReceiptEventIDFilter)
                        .textFieldStyle(.roundedBorder)

                    Stepper(
                        "Limit: \(state.webhookReceiptLimit)",
                        value: $state.webhookReceiptLimit,
                        in: 5...200,
                        step: 5
                    )

                    HStack(spacing: 8) {
                        Button("Refresh Webhook Receipts") {
                            state.refreshWebhookTrustReceipts()
                        }
                        .buttonStyle(.bordered)
                        .disabled(state.isWebhookReceiptsLoading)

                        Button("Reset Filters") {
                            state.resetWebhookTrustReceiptFilters()
                            state.refreshWebhookTrustReceipts()
                        }
                        .buttonStyle(.bordered)
                        .disabled(state.isWebhookReceiptsLoading)

                        if state.isWebhookReceiptsLoading {
                            ProgressView()
                                .controlSize(.small)
                        }
                    }
                }
            }

            configurationSecondaryStatusMessage(state.webhookReceiptsStatusMessage)

            configurationInventoryGroup(
                title: "Webhook Receipts (\(state.webhookReceiptItems.count))",
                items: state.webhookReceiptItems,
                isLoading: state.isWebhookReceiptsLoading,
                loadingMessage: "Loading webhook trust receipts…",
                emptyMessage: "No webhook trust receipts matched the current filters.",
                hasMore: state.webhookReceiptsHasMore,
                hasMoreMessage: "More webhook trust receipts are available. Narrow filters for focused review."
            ) { item in
                webhookTrustReceiptRow(item)
            }
        }
    }

    var ingestTrustReceiptsInventory: some View {
        VStack(alignment: .leading, spacing: 10) {
            GroupBox("Ingest Filters") {
                VStack(alignment: .leading, spacing: 8) {
                    TextField("Source (optional)", text: $state.ingestReceiptSourceFilter)
                        .textFieldStyle(.roundedBorder)

                    TextField("Source Scope (optional)", text: $state.ingestReceiptSourceScopeFilter)
                        .textFieldStyle(.roundedBorder)

                    TextField("Source Event ID (optional)", text: $state.ingestReceiptSourceEventIDFilter)
                        .textFieldStyle(.roundedBorder)

                    TextField("Source Event Query (optional)", text: $state.ingestReceiptSourceEventQueryFilter)
                        .textFieldStyle(.roundedBorder)

                    Picker("Trust State", selection: $state.ingestReceiptTrustStateFilter) {
                        ForEach(state.receiptTrustStateFilterOptions, id: \.self) { option in
                            Text(contextFilterOptionLabel(option)).tag(option)
                        }
                    }
                    .pickerStyle(.menu)

                    TextField("Event ID (optional)", text: $state.ingestReceiptEventIDFilter)
                        .textFieldStyle(.roundedBorder)

                    Stepper(
                        "Limit: \(state.ingestReceiptLimit)",
                        value: $state.ingestReceiptLimit,
                        in: 5...200,
                        step: 5
                    )

                    HStack(spacing: 8) {
                        Button("Refresh Ingest Receipts") {
                            state.refreshIngestTrustReceipts()
                        }
                        .buttonStyle(.bordered)
                        .disabled(state.isIngestReceiptsLoading)

                        Button("Reset Filters") {
                            state.resetIngestTrustReceiptFilters()
                            state.refreshIngestTrustReceipts()
                        }
                        .buttonStyle(.bordered)
                        .disabled(state.isIngestReceiptsLoading)

                        if state.isIngestReceiptsLoading {
                            ProgressView()
                                .controlSize(.small)
                        }
                    }
                }
            }

            configurationSecondaryStatusMessage(state.ingestReceiptsStatusMessage)

            configurationInventoryGroup(
                title: "Ingest Receipts (\(state.ingestReceiptItems.count))",
                items: state.ingestReceiptItems,
                isLoading: state.isIngestReceiptsLoading,
                loadingMessage: "Loading ingest trust receipts…",
                emptyMessage: "No ingest trust receipts matched the current filters.",
                hasMore: state.ingestReceiptsHasMore,
                hasMoreMessage: "More ingest trust receipts are available. Narrow filters for focused review."
            ) { item in
                ingestTrustReceiptRow(item)
            }
        }
    }


    var capabilityGrantActorOptions: [String] {
        configurationDraftStore.capabilityGrantActorOptions(
            principalOptions: state.delegationPrincipalOptions
        )
    }

    var isCapabilityGrantMutationDisabled: Bool {
        configurationDraftStore.isCapabilityGrantMutationDisabled(
            isInFlight: state.isCapabilityGrantMutationInFlight
        )
    }

    func seedCapabilityGrantDraftIfNeeded() {
        configurationDraftStore.seedCapabilityGrantDraftIfNeeded(
            principalOptions: state.delegationPrincipalOptions
        )
    }

    func submitCapabilityGrantUpsert() {
        if configurationDraftStore.useCapabilityGrantRawScopeOverride && !configurationDraftStore.isCapabilityGrantRawScopeJSONValid {
            state.capabilityGrantMutationStatusMessage = "Scope JSON must be a valid JSON object."
            return
        }
        if !configurationDraftStore.useCapabilityGrantRawScopeOverride {
            configurationDraftStore.syncCapabilityGrantRawScopeFromGuided()
        }
        state.upsertCapabilityGrant(configurationDraftStore.capabilityGrantMutationInput())
    }

    func loadCapabilityGrantDraft(_ item: CapabilityGrantItem) {
        configurationDraftStore.loadCapabilityGrantDraft(item)
    }

    var isCapabilityGrantRawScopeJSONValid: Bool {
        configurationDraftStore.isCapabilityGrantRawScopeJSONValid
    }

    var capabilityGrantRawScopeValidationMessage: String {
        configurationDraftStore.capabilityGrantRawScopeValidationMessage
    }

    func scopeEntry(for id: UUID) -> GuidedEditorScopeEntry? {
        configurationDraftStore.scopeEntry(for: id)
    }

    func updateCapabilityGrantScopeEntry(_ id: UUID, key: String?, value: String?) {
        configurationDraftStore.updateCapabilityGrantScopeEntry(id, key: key, value: value)
    }

    func addCapabilityGrantScopeEntry() {
        configurationDraftStore.addCapabilityGrantScopeEntry()
    }

    func removeCapabilityGrantScopeEntry(_ id: UUID) {
        configurationDraftStore.removeCapabilityGrantScopeEntry(id)
    }

    func resetCapabilityGrantScopeEntries() {
        configurationDraftStore.resetCapabilityGrantScopeEntries()
    }

    func syncCapabilityGrantRawScopeFromGuided() {
        configurationDraftStore.syncCapabilityGrantRawScopeFromGuided()
    }

    func applyCapabilityGrantRawScopeToGuided() {
        guard configurationDraftStore.applyCapabilityGrantRawScopeToGuided() else {
            state.capabilityGrantMutationStatusMessage = "Scope JSON must be a valid JSON object before applying to guided fields."
            return
        }
    }

    func capabilityGrantRow(_ item: CapabilityGrantItem) -> some View {
        let actorIdentity = state.principalIdentityDisplayValue(for: item.actorID)
        return configurationRecordCard {
            HStack(spacing: 8) {
                Text(item.capabilityKey)
                    .font(.caption.weight(.semibold))
                    .textSelection(.enabled)

                TahoeStatusBadge(
                    text: contextFilterOptionLabel(item.status),
                    symbolName: capabilityGrantStatusSymbol(item.status),
                    tint: capabilityGrantStatusTint(item.status)
                )
                .controlSize(.small)

                Spacer(minLength: 0)
                Text(item.createdAtLabel)
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }

            Text("Grant ID \(item.id)")
                .font(.caption2)
                .foregroundStyle(.secondary)
                .textSelection(.enabled)
            HStack(alignment: .firstTextBaseline, spacing: 6) {
                Text("Actor")
                    .font(.caption2.weight(.semibold))
                    .foregroundStyle(.secondary)
                IdentityValueInlineView(
                    displayText: actorIdentity.displayText,
                    rawID: actorIdentity.rawID,
                    valueFont: .caption2
                )
                Text("• Scope \(item.scopeSummary)")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
                    .textSelection(.enabled)
            }

            if let expiresAtLabel = item.expiresAtLabel {
                Text("Expires \(expiresAtLabel)")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }

            HStack(spacing: 8) {
                Button("Load into Form") {
                    loadCapabilityGrantDraft(item)
                }
                .buttonStyle(.bordered)
                .controlSize(.small)

                if !item.isRevoked {
                    Button("Revoke", role: .destructive) {
                        state.requestRevokeCapabilityGrant(grantID: item.id)
                    }
                    .buttonStyle(.bordered)
                    .controlSize(.small)
                    .disabled(state.capabilityGrantRevokeInFlightIDs.contains(item.id))
                }

                Button("Open Inspect") {
                    state.openInspectForCapabilityGrant(item)
                }
                .buttonStyle(.bordered)
                .controlSize(.small)

                if state.capabilityGrantRevokeInFlightIDs.contains(item.id) {
                    ProgressView()
                        .controlSize(.small)
                }
            }

            if let actionStatus = state.capabilityGrantActionStatusByID[item.id] {
                Text(actionStatus)
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }
        }
    }

    func webhookTrustReceiptRow(_ item: WebhookTrustReceiptItem) -> some View {
        configurationRecordCard {
            HStack(spacing: 8) {
                Text(item.providerEventID)
                    .font(.caption.weight(.semibold))
                    .textSelection(.enabled)
                TahoeStatusBadge(
                    text: receiptTrustLabel(item.trustState),
                    symbolName: receiptTrustSymbol(item.trustState),
                    tint: receiptTrustTint(item.trustState)
                )
                .controlSize(.small)
                Spacer(minLength: 0)
                Text(item.createdAtLabel)
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }

            Text("Receipt \(item.id) • Provider \(item.provider)")
                .font(.caption2)
                .foregroundStyle(.secondary)
                .textSelection(.enabled)
            Text("Signature valid: \(item.signatureValid ? "true" : "false") • Signature value present: \(item.signatureValuePresent ? "true" : "false")")
                .font(.caption2)
                .foregroundStyle(.secondary)
            Text("Payload hash: \(item.payloadHash)")
                .font(.caption2)
                .foregroundStyle(.secondary)
                .textSelection(.enabled)
            if let eventID = item.eventID {
                Text("Event ID: \(eventID)")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
                    .textSelection(.enabled)
            }
            if let threadID = item.threadID {
                Text("Thread ID: \(threadID)")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
                    .textSelection(.enabled)
            }
            if let receivedAtLabel = item.receivedAtLabel {
                Text("Received \(receivedAtLabel)")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }

            HStack(spacing: 8) {
                Button("Open Inspect") {
                    state.openInspectForTrustReceipt(
                        receiptID: item.id,
                        preferredSeed: item.eventID,
                        fallbackSeed: item.providerEventID
                    )
                }
                .buttonStyle(.bordered)
                .controlSize(.small)
            }

            trustReceiptAuditLinksView(item.auditLinks)
        }
    }

    func ingestTrustReceiptRow(_ item: IngestTrustReceiptItem) -> some View {
        configurationRecordCard {
            HStack(spacing: 8) {
                Text(item.sourceEventID)
                    .font(.caption.weight(.semibold))
                    .textSelection(.enabled)
                TahoeStatusBadge(
                    text: receiptTrustLabel(item.trustState),
                    symbolName: receiptTrustSymbol(item.trustState),
                    tint: receiptTrustTint(item.trustState)
                )
                .controlSize(.small)
                Spacer(minLength: 0)
                Text(item.createdAtLabel)
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }

            Text("Receipt \(item.id) • Source \(item.source) • Scope \(item.sourceScope)")
                .font(.caption2)
                .foregroundStyle(.secondary)
                .textSelection(.enabled)
            if let sourceCursor = item.sourceCursor {
                Text("Source cursor: \(sourceCursor)")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
                    .textSelection(.enabled)
            }
            Text("Payload hash: \(item.payloadHash)")
                .font(.caption2)
                .foregroundStyle(.secondary)
                .textSelection(.enabled)
            if let eventID = item.eventID {
                Text("Event ID: \(eventID)")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
                    .textSelection(.enabled)
            }
            if let threadID = item.threadID {
                Text("Thread ID: \(threadID)")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
                    .textSelection(.enabled)
            }
            if let receivedAtLabel = item.receivedAtLabel {
                Text("Received \(receivedAtLabel)")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }

            HStack(spacing: 8) {
                Button("Open Inspect") {
                    state.openInspectForTrustReceipt(
                        receiptID: item.id,
                        preferredSeed: item.eventID,
                        fallbackSeed: item.sourceEventID
                    )
                }
                .buttonStyle(.bordered)
                .controlSize(.small)
            }

            trustReceiptAuditLinksView(item.auditLinks)
        }
    }

    func trustReceiptAuditLinksView(_ links: [TrustReceiptAuditLinkItem]) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            if links.isEmpty {
                Text("No audit links matched this receipt.")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            } else {
                Text("Audit Links")
                    .font(.caption2.weight(.semibold))
                    .foregroundStyle(.secondary)
                ForEach(links) { link in
                    HStack(alignment: .firstTextBaseline, spacing: 8) {
                        Button(link.id) {
                            state.openInspectForTrustReceiptAuditLink(link)
                        }
                        .buttonStyle(.link)
                        .lineLimit(1)
                        .truncationMode(.middle)
                        Text(contextFilterOptionLabel(link.eventType))
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                        Spacer(minLength: 0)
                        Text(link.createdAtLabel)
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                    }
                }
            }
        }
    }

    func capabilityGrantStatusSymbol(_ raw: String) -> String {
        switch raw.uppercased() {
        case "ACTIVE":
            return "checkmark.circle.fill"
        case "DISABLED":
            return "pause.circle.fill"
        case "REVOKED":
            return "xmark.octagon.fill"
        default:
            return "questionmark.circle.fill"
        }
    }

    func capabilityGrantStatusTint(_ raw: String) -> Color {
        switch raw.uppercased() {
        case "ACTIVE":
            return .green
        case "DISABLED":
            return .orange
        case "REVOKED":
            return .red
        default:
            return .secondary
        }
    }

    func receiptTrustLabel(_ raw: String) -> String {
        switch raw.lowercased() {
        case "accepted":
            return "Accepted"
        case "rejected":
            return "Rejected"
        default:
            return contextFilterOptionLabel(raw)
        }
    }

    func receiptTrustSymbol(_ raw: String) -> String {
        switch raw.lowercased() {
        case "accepted":
            return "checkmark.shield.fill"
        case "rejected":
            return "xmark.shield.fill"
        default:
            return "questionmark.shield.fill"
        }
    }

    func receiptTrustTint(_ raw: String) -> Color {
        switch raw.lowercased() {
        case "accepted":
            return .green
        case "rejected":
            return .red
        default:
            return .secondary
        }
    }
}
