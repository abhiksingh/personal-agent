import AppKit
import SwiftUI

extension ConfigurationPanelView {
    var setupModeContent: some View {
        ConfigurationSetupModeContent {
            setupDiagnosticsSection
        } token: {
            tokenSection
        } runtime: {
            operatorDisclosure(
                title: "Runtime Details",
                isExpanded: $isSetupRuntimeDetailsExpanded
            ) {
                runtimeSection
            }
        }
    }

    var runtimeSection: some View {
        VStack(alignment: .leading, spacing: 10) {
            sectionTitle("Workspace Runtime")

            settingRow(label: "Workspace", value: state.workspaceLabel)
            settingRow(label: "Principal Context", value: state.activePrincipalLabel)
            settingRow(label: "Daemon Endpoint", value: state.daemonEndpointLabel)

            Text("Detailed runtime readiness and remediation appear in the Setup Matrix above.")
                .font(.caption)
                .foregroundStyle(.secondary)

            Button {
                state.refreshDaemonStatus()
            } label: {
                Label("Refresh Daemon Status", systemImage: "arrow.clockwise")
            }
            .quietButtonChrome()
            .disabled(state.isDaemonLifecycleLoading)
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .cardSurface(.standard)
    }

    var setupDiagnosticsSection: some View {
        VStack(alignment: .leading, spacing: 10) {
            sectionTitle("Setup Overview")

            HStack(spacing: 8) {
                TahoeStatusBadge(
                    text: "\(setupReadyCount) Ready",
                    symbolName: SetupMatrixStatus.ready.symbolName,
                    tint: SetupMatrixStatus.ready.tint
                )
                .controlSize(.small)

                TahoeStatusBadge(
                    text: "\(setupAttentionCount) Needs Attention",
                    symbolName: SetupMatrixStatus.attention.symbolName,
                    tint: SetupMatrixStatus.attention.tint
                )
                .controlSize(.small)

                if setupLoadingCount > 0 {
                    TahoeStatusBadge(
                        text: "\(setupLoadingCount) Checking",
                        symbolName: SetupMatrixStatus.loading.symbolName,
                        tint: SetupMatrixStatus.loading.tint
                    )
                    .controlSize(.small)
                }
            }

            Text(setupDiagnosticsSummary)
                .font(.caption)
                .foregroundStyle(.secondary)

            HStack(spacing: 8) {
                if let fixNextStep = state.onboardingFixNextStep {
                    Button("Fix Next") {
                        state.performOnboardingFixNextStep()
                    }
                    .buttonStyle(.borderedProminent)
                    .disabled(state.onboardingSetupChecksLoading)
                    .help("Fix Next: \(fixNextStep.title). \(fixNextStep.detail)")
                }

                if let primaryRemediation = setupPrimaryRemediation {
                    Button(primaryRemediation.title) {
                        performSetupRemediation(primaryRemediation)
                    }
                    .buttonStyle(.bordered)
                    .disabled(isSetupRemediationDisabled(primaryRemediation))
                }

                if !setupAdditionalRemediations.isEmpty {
                    Menu("More Actions") {
                        ForEach(setupAdditionalRemediations) { remediation in
                            Button(remediation.title) {
                                performSetupRemediation(remediation)
                            }
                            .disabled(isSetupRemediationDisabled(remediation))
                        }
                    }
                    .menuStyle(.borderlessButton)
                }

                Button("Refresh Checks") {
                    refreshSetupChecks()
                }
                .quietButtonChrome()
                .disabled(isSetupQuickRefreshLoading)

                if isSetupQuickRefreshLoading {
                    ProgressView()
                        .controlSize(.small)
                }
            }

            if let actionGuidance = setupActionGuidanceMessage {
                Text(actionGuidance)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            DisclosureGroup(isExpanded: $isSetupMatrixExpanded) {
                VStack(alignment: .leading, spacing: 8) {
                    ForEach(setupMatrixItems) { item in
                        setupMatrixRow(item)
                    }
                }
                .padding(.top, 6)
            } label: {
                HStack(spacing: 8) {
                    Text("Setup Matrix Details")
                        .font(.subheadline.weight(.semibold))
                    Spacer(minLength: 0)
                    Text("\(setupMatrixItems.count) checks")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }
            .accessibilityIdentifier("configuration-setup-matrix-disclosure")

            if state.setupReadinessChecksLoading {
                HStack(spacing: 8) {
                    ProgressView()
                        .controlSize(.small)
                    Text("Refreshing setup diagnostics…")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }

            distributionTrustGuidanceSection
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .cardSurface(.standard)
    }

    var distributionTrustGuidanceSection: some View {
        GroupBox("First-Run Trust Guidance (Unsigned Build)") {
            VStack(alignment: .leading, spacing: 8) {
                Text(state.distributionTrustGuidanceSummary)
                    .font(.caption)
                    .foregroundStyle(.secondary)

                VStack(alignment: .leading, spacing: 4) {
                    ForEach(Array(state.distributionTrustGuidanceChecklist.enumerated()), id: \.offset) { entry in
                        Text("\(entry.offset + 1). \(entry.element)")
                            .font(.caption)
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
                    .quietButtonChrome()

                    Button("Retry Setup Checks") {
                        state.retrySetupChecksAfterTrustGuidance()
                    }
                    .quietButtonChrome()
                    .disabled(isSetupQuickRefreshLoading)
                }
            }
        }
    }

    var tokenSection: some View {
        VStack(alignment: .leading, spacing: 10) {
            sectionTitle("Assistant Access Token")

            HStack(spacing: 8) {
                SecureField("Enter access token", text: $state.localDevTokenInput)
                    .textFieldStyle(.roundedBorder)
                Button("Save Token") {
                    state.saveLocalDevToken()
                }
                .quietButtonChrome()
                .disabled(tokenInputTrimmed.isEmpty)
            }

            HStack(spacing: 8) {
                TahoeStatusBadge(
                    text: tokenValidationBadgeText,
                    symbolName: tokenValidationBadgeSymbol,
                    tint: tokenValidationBadgeTint
                )
                Text("Last updated: \(state.localDevTokenLastUpdated)")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            GroupBox("Bootstrap from CLI") {
                VStack(alignment: .leading, spacing: 8) {
                    Text("Generate or reuse an access token and update CLI defaults for this workspace and daemon endpoint.")
                        .font(.caption)
                        .foregroundStyle(.secondary)

                    Text(state.localDevAuthBootstrapCommand)
                        .textSelection(.enabled)
                        .font(.system(.caption, design: .monospaced))
                        .padding(8)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .background(
                            RoundedRectangle(cornerRadius: 8, style: .continuous)
                                .fill(Color.primary.opacity(0.04))
                        )

                    HStack(spacing: 8) {
                        Button("Copy Command") {
                            copyLocalDevAuthBootstrapCommand()
                        }
                        .quietButtonChrome()

                        Button("Run Bootstrap") {
                            state.runLocalDevAuthBootstrap()
                        }
                        .buttonStyle(.borderedProminent)
                        .disabled(state.isLocalDevAuthBootstrapInFlight)

                        if state.isLocalDevAuthBootstrapInFlight {
                            ProgressView()
                                .controlSize(.small)
                        }
                    }

                    if let status = state.localDevAuthBootstrapStatusMessage {
                        Text(status)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                }
            }

            Button("Clear Stored Token") {
                state.clearLocalDevToken()
            }
            .quietButtonChrome()
            .disabled(!state.localDevTokenConfigured)

            Text("Token values are write-only in app UI and are never shown after save.")
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .cardSurface(.standard)
    }

    func setupMatrixRow(_ item: SetupMatrixItem) -> some View {
        HStack(alignment: .top, spacing: 8) {
            Image(systemName: item.status.symbolName)
                .font(.callout)
                .foregroundStyle(item.status.tint)

            VStack(alignment: .leading, spacing: 2) {
                Text(item.title)
                    .font(.callout.weight(.semibold))
                Text(item.detail)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            .frame(maxWidth: .infinity, alignment: .leading)

            TahoeStatusBadge(
                text: item.status.label,
                symbolName: item.status.symbolName,
                tint: item.status.tint
            )
            .controlSize(.small)
        }
        .padding(10)
        .background(
            RoundedRectangle(cornerRadius: 10, style: .continuous)
                .fill(Color.primary.opacity(0.04))
        )
    }

    var tokenInputTrimmed: String {
        state.localDevTokenInput.trimmingCharacters(in: .whitespacesAndNewlines)
    }

    func copyLocalDevAuthBootstrapCommand() {
        NSPasteboard.general.clearContents()
        NSPasteboard.general.setString(state.localDevAuthBootstrapCommand, forType: .string)
        state.noteLocalDevAuthBootstrapCommandCopied()
    }

    var tokenValidationBadgeText: String {
        switch state.effectiveDaemonControlAuthState {
        case .configured:
            return "Configured"
        case .missing:
            return "Not Configured"
        case .unknown:
            return "Checking"
        }
    }

    var tokenValidationBadgeSymbol: String {
        switch state.effectiveDaemonControlAuthState {
        case .configured:
            return "checkmark.circle.fill"
        case .missing:
            return "xmark.circle"
        case .unknown:
            return "clock.arrow.circlepath"
        }
    }

    var tokenValidationBadgeTint: Color {
        switch state.effectiveDaemonControlAuthState {
        case .configured:
            return .green
        case .missing, .unknown:
            return .secondary
        }
    }

    var daemonInstallStatusText: String {
        switch state.daemonStatus {
        case .running, .stopped:
            return "Installed"
        case .missing:
            return "Not Installed"
        case .broken:
            return "Needs Repair"
        case .unknown:
            return "Unknown"
        }
    }

    var daemonInstallStatusSymbol: String {
        switch state.daemonStatus {
        case .running, .stopped:
            return "checkmark.seal.fill"
        case .missing:
            return "exclamationmark.triangle.fill"
        case .broken:
            return "wrench.and.screwdriver.fill"
        case .unknown:
            return "questionmark.circle.fill"
        }
    }

    var daemonInstallStatusTint: Color {
        switch state.daemonStatus {
        case .running, .stopped:
            return .green
        case .missing, .broken:
            return .orange
        case .unknown:
            return .secondary
        }
    }

    var setupMatrixItems: [SetupMatrixItem] {
        [
            daemonSetupMatrixItem,
            tokenSetupMatrixItem,
            providerSetupMatrixItem,
            catalogSetupMatrixItem,
            chatRouteSetupMatrixItem
        ]
    }

    var daemonSetupMatrixItem: SetupMatrixItem {
        if state.isDaemonLifecycleLoading || !state.hasLoadedDaemonStatus {
            return SetupMatrixItem(
                id: "daemon_lifecycle",
                title: "Daemon Lifecycle",
                detail: "Refreshing daemon lifecycle and reachability checks.",
                status: .loading,
                remediations: []
            )
        }

        let isReady = state.daemonStatus == .running &&
            state.connectionStatus == .connected &&
            !state.daemonNeedsInstall &&
            !state.daemonNeedsRepair
        if isReady {
            return SetupMatrixItem(
                id: "daemon_lifecycle",
                title: "Daemon Lifecycle",
                detail: "Daemon is running and reachable for workspace \(state.workspaceLabel).",
                status: .ready,
                remediations: []
            )
        }

        if state.daemonNeedsInstall || state.daemonStatus == .missing {
            return SetupMatrixItem(
                id: "daemon_lifecycle",
                title: "Daemon Lifecycle",
                detail: state.daemonStatusDetail,
                status: .attention,
                remediations: [.installDaemon]
            )
        }

        if state.daemonNeedsRepair || state.daemonStatus == .broken {
            if state.daemonHasWorkerFailureRepairState {
                let failedCount = state.daemonWorkerSummary.failed
                let pluralSuffix = failedCount == 1 ? "" : "s"
                return SetupMatrixItem(
                    id: "daemon_lifecycle",
                    title: "Daemon Lifecycle",
                    detail: "Daemon control plane is reachable, but \(failedCount) plugin worker\(pluralSuffix) failed to start. Open Channels for worker diagnostics before using daemon repair/restart.",
                    status: .attention,
                    remediations: [.openChannels, .repairDaemon]
                )
            }
            return SetupMatrixItem(
                id: "daemon_lifecycle",
                title: "Daemon Lifecycle",
                detail: state.daemonStatusDetail,
                status: .attention,
                remediations: [.repairDaemon]
            )
        }

        return SetupMatrixItem(
            id: "daemon_lifecycle",
            title: "Daemon Lifecycle",
            detail: state.daemonStatusDetail,
            status: .attention,
            remediations: [.startDaemon]
        )
    }

    var tokenSetupMatrixItem: SetupMatrixItem {
        if state.isDaemonLifecycleLoading && state.localDevTokenConfigured {
            return SetupMatrixItem(
                id: "auth_token",
                title: "Assistant Access Token",
                detail: "Validating daemon auth token state.",
                status: .loading,
                remediations: []
            )
        }

        let authState = state.effectiveDaemonControlAuthState
        return SetupMatrixItem(
            id: "auth_token",
            title: "Assistant Access Token",
            detail: state.daemonControlAuthSetupDetail,
            status: authState == .configured ? .ready : .attention,
            remediations: []
        )
    }

    var providerSetupMatrixItem: SetupMatrixItem {
        if state.isProviderStatusLoading || !state.hasLoadedProviderStatus {
            return SetupMatrixItem(
                id: "provider_setup",
                title: "Provider Setup",
                detail: "Refreshing provider inventory and readiness checks.",
                status: .loading,
                remediations: []
            )
        }

        let readyCount = state.providerReadinessItems.filter { item in
            item.status == .configured || item.status == .healthy
        }.count

        if readyCount > 0 {
            return SetupMatrixItem(
                id: "provider_setup",
                title: "Provider Setup",
                detail: "\(readyCount) provider\(readyCount == 1 ? "" : "s") configured and ready for routing.",
                status: .ready,
                remediations: []
            )
        }

        return SetupMatrixItem(
            id: "provider_setup",
            title: "Provider Setup",
            detail: state.providerStatusMessage ?? "Open Models and save provider setup.",
            status: .attention,
            remediations: [.openModels]
        )
    }

    var catalogSetupMatrixItem: SetupMatrixItem {
        if (state.isProviderStatusLoading || !state.hasLoadedProviderStatus) && state.modelCatalogItems.isEmpty {
            return SetupMatrixItem(
                id: "catalog_readiness",
                title: "Model Catalog",
                detail: "Refreshing model catalog inventory.",
                status: .loading,
                remediations: []
            )
        }

        let totalCount = state.modelCatalogItems.count
        let enabledCount = state.modelCatalogItems.filter(\.enabled).count
        if totalCount > 0 && enabledCount > 0 {
            return SetupMatrixItem(
                id: "catalog_readiness",
                title: "Model Catalog",
                detail: "Catalog loaded with \(enabledCount) enabled model\(enabledCount == 1 ? "" : "s") out of \(totalCount).",
                status: .ready,
                remediations: []
            )
        }

        if totalCount > 0 {
            return SetupMatrixItem(
                id: "catalog_readiness",
                title: "Model Catalog",
                detail: "Catalog loaded with \(totalCount) model\(totalCount == 1 ? "" : "s"), but none are enabled for routing.",
                status: .attention,
                remediations: [.openModels]
            )
        }

        return SetupMatrixItem(
            id: "catalog_readiness",
            title: "Model Catalog",
            detail: state.modelCatalogStatusMessage ?? "No catalog models available yet.",
            status: .attention,
            remediations: [.openModels]
        )
    }

    var chatRouteSetupMatrixItem: SetupMatrixItem {
        if (state.isProviderStatusLoading || !state.hasLoadedProviderStatus) && state.modelRouteSummary == nil {
            return SetupMatrixItem(
                id: "chat_route",
                title: "Chat Route",
                detail: "Refreshing chat route resolution.",
                status: .loading,
                remediations: []
            )
        }

        if let summary = state.modelRouteSummary {
            let sourceSuffix = summary.source.isEmpty ? "" : " (\(summary.source))"
            return SetupMatrixItem(
                id: "chat_route",
                title: "Chat Route",
                detail: "Chat route resolves to \(summary.provider) • \(summary.modelKey)\(sourceSuffix).",
                status: .ready,
                remediations: []
            )
        }

        return SetupMatrixItem(
            id: "chat_route",
            title: "Chat Route",
            detail: state.chatRouteRemediationMessage
                ?? state.modelRouteStatusMessage
                ?? "Open Models and save a `chat` route policy to an enabled model.",
            status: .attention,
            remediations: [.openModels, .openOnboarding]
        )
    }

    var setupMatrixRemediations: [SetupMatrixRemediation] {
        var ordered: [SetupMatrixRemediation] = []
        var seen: Set<String> = []
        for item in setupMatrixItems where item.status == .attention {
            for remediation in item.remediations where seen.insert(remediation.id).inserted {
                ordered.append(remediation)
            }
        }
        return ordered
    }

    var setupPrimaryRemediation: SetupMatrixRemediation? {
        setupMatrixRemediations.first
    }

    var setupAdditionalRemediations: [SetupMatrixRemediation] {
        Array(setupMatrixRemediations.dropFirst())
    }

    var setupLoadingCount: Int {
        setupMatrixItems.filter { $0.status == .loading }.count
    }

    var setupAttentionCount: Int {
        setupMatrixItems.filter { $0.status == .attention }.count
    }

    var setupReadyCount: Int {
        setupMatrixItems.filter { $0.status == .ready }.count
    }

    var setupActionGuidanceMessage: String? {
        if state.onboardingSetupChecksLoading, state.onboardingFixNextStep != nil {
            return "Fix Next is unavailable while setup checks are refreshing."
        }
        if let primaryRemediation = setupPrimaryRemediation {
            return setupRemediationDisabledReason(primaryRemediation)
        }
        return nil
    }

    var isSetupQuickRefreshLoading: Bool {
        state.setupReadinessChecksLoading || state.isDaemonLifecycleLoading || state.isProviderStatusLoading
    }

    func performSetupRemediation(_ remediation: SetupMatrixRemediation) {
        switch remediation {
        case .startDaemon:
            state.requestStartDaemon()
        case .installDaemon:
            state.requestInstallDaemon()
        case .repairDaemon:
            state.requestRepairDaemonInstallation()
        case .openChannels:
            state.navigateToSection(.channels)
        case .openModels:
            state.openModelsForChatRemediation()
        case .openOnboarding:
            state.openOnboardingFromConfiguration()
        }
    }

    func isSetupRemediationDisabled(_ remediation: SetupMatrixRemediation) -> Bool {
        setupRemediationDisabledReason(remediation) != nil
    }

    func setupRemediationDisabledReason(_ remediation: SetupMatrixRemediation) -> String? {
        switch remediation {
        case .startDaemon:
            if !state.localDevTokenConfigured {
                return "Save Assistant Access Token before starting daemon lifecycle actions."
            }
            if state.isDaemonControlInFlight {
                return "A daemon lifecycle action is already in progress."
            }
            if !state.daemonCanStart {
                return "Start is currently unavailable for this daemon state."
            }
            return nil
        case .installDaemon:
            return state.daemonInstallFromBundleDisabledReason
        case .repairDaemon:
            return state.daemonRepairFromBundleDisabledReason
        case .openChannels:
            return nil
        case .openModels:
            return nil
        case .openOnboarding:
            if state.onboardingReadinessMet {
                return "Onboarding is available only when setup still needs attention."
            }
            return nil
        }
    }

    func refreshSetupChecks() {
        state.refreshOnboardingReadiness()
        state.refreshDaemonStatus()
        state.refreshProviderInventory()
    }

    var setupDiagnosticsSummary: String {
        if setupLoadingCount > 0 {
            return "Refreshing setup matrix checks…"
        }

        if setupAttentionCount == 0 {
            return "All setup matrix checks are ready for workflow panels."
        }
        if setupAttentionCount == 1 {
            return "One setup matrix check still needs attention."
        }
        return "\(setupAttentionCount) setup matrix checks still need attention."
    }
}
