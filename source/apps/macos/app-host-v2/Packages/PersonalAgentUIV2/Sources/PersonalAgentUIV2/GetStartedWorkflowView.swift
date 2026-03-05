import SwiftUI

struct GetStartedWorkflowView: View {
    @ObservedObject var store: AppShellV2Store

    var body: some View {
        VStack(alignment: .leading, spacing: V2WorkflowLayout.panelSpacing) {
            PASectionHeader(
                title: "Get to your first trusted result fast",
                subtitle: "Use this short checklist and get one real run through Replay before deeper setup."
            )
            V2PanelStateBannerView(
                state: store.panelLifecycleState(for: .getStarted),
                onAction: { actionID in
                    store.performPanelStateAction(actionID, workflow: .getStarted)
                }
            )
            sessionConfigCard
            nextActionCard
            checklistCard
            Spacer(minLength: 0)
        }
        .onAppear {
            Task {
                await store.refreshGetStartedReadinessIfNeeded()
            }
        }
    }

    private var sessionConfigCard: some View {
        PASurfaceCard("Session Setup", tone: store.sessionReadiness.isReadyForDaemonMutations ? .emerald : .warm) {
            VStack(alignment: .leading, spacing: V2WorkflowLayout.sectionSpacing) {
                HStack(spacing: 6) {
                    PAStatusChip(
                        label: store.sessionReadiness.hasValidDaemonBaseURL ? "Daemon URL Ready" : "Daemon URL Missing",
                        systemImage: store.sessionReadiness.hasValidDaemonBaseURL ? "checkmark.circle.fill" : "exclamationmark.circle.fill",
                        tone: store.sessionReadiness.hasValidDaemonBaseURL ? .success : .warning
                    )
                    PAStatusChip(
                        label: store.sessionReadiness.hasStoredAccessToken ? "Token Stored" : "Token Missing",
                        systemImage: store.sessionReadiness.hasStoredAccessToken ? "lock.shield.fill" : "lock.slash.fill",
                        tone: store.sessionReadiness.hasStoredAccessToken ? .success : .warning
                    )
                    PAStatusChip(
                        label: "Density: \(store.informationDensityMode.title)",
                        systemImage: "dial.low",
                        tone: .info
                    )
                }

                HStack(spacing: 6) {
                    PAStatusChip(
                        label: lifecycleChipLabel,
                        systemImage: lifecycleChipSymbol,
                        tone: lifecycleChipTone
                    )
                    PAStatusChip(
                        label: routeChipLabel,
                        systemImage: routeChipSymbol,
                        tone: routeChipTone
                    )
                    PAStatusChip(
                        label: connectorChipLabel,
                        systemImage: connectorChipSymbol,
                        tone: connectorChipTone
                    )
                    PAStatusChip(
                        label: replayChipLabel,
                        systemImage: replayChipSymbol,
                        tone: replayChipTone
                    )
                }

                Text(store.sessionReadiness.setupSummary)
                    .font(.paBody)
                    .foregroundStyle(Color.paTextSecondary)

                HStack(spacing: 8) {
                    Text("Daemon URL")
                        .font(.paCaption)
                        .foregroundStyle(Color.paTextSecondary)
                        .frame(width: 82, alignment: .leading)

                    TextField("http://127.0.0.1:7071", text: daemonBaseURLBinding)
                        .textFieldStyle(.plain)
                        .paInputSurface()
                        .accessibilityLabel("Daemon URL")
                        .accessibilityIdentifier("v2-get-started-daemon-url")
                }

                HStack(spacing: 8) {
                    Text("Workspace")
                        .font(.paCaption)
                        .foregroundStyle(Color.paTextSecondary)
                        .frame(width: 82, alignment: .leading)

                    TextField("ws1", text: workspaceIDBinding)
                        .textFieldStyle(.plain)
                        .paInputSurface()
                        .accessibilityLabel("Workspace ID")
                        .accessibilityIdentifier("v2-get-started-workspace")

                    Text("Principal")
                        .font(.paCaption)
                        .foregroundStyle(Color.paTextSecondary)
                        .frame(width: 58, alignment: .leading)

                    TextField("default", text: principalActorIDBinding)
                        .textFieldStyle(.plain)
                        .paInputSurface()
                        .accessibilityLabel("Principal Actor")
                        .accessibilityIdentifier("v2-get-started-principal")
                }

                HStack(spacing: 8) {
                    Text("Density")
                        .font(.paCaption)
                        .foregroundStyle(Color.paTextSecondary)
                        .frame(width: 82, alignment: .leading)

                    Picker("Density", selection: densityModeBinding) {
                        ForEach(V2InformationDensityMode.allCases, id: \.rawValue) { mode in
                            Text(mode.title).tag(mode)
                        }
                    }
                    .pickerStyle(.segmented)
                    .accessibilityIdentifier("v2-get-started-density")
                }

                HStack(spacing: 8) {
                    Text("Access Token")
                        .font(.paCaption)
                        .foregroundStyle(Color.paTextSecondary)
                        .frame(width: 82, alignment: .leading)

                    SecureField("Paste assistant access token", text: $store.accessTokenInput)
                        .textFieldStyle(.plain)
                        .paInputSurface()
                        .accessibilityLabel("Assistant Access Token")
                        .accessibilityIdentifier("v2-get-started-access-token")
                }

                HStack(spacing: 8) {
                    Button("Save Token") {
                        store.saveAccessTokenFromInput()
                    }
                    .buttonStyle(.borderedProminent)
                    .tint(.paInfo)
                    .disabled(tokenSaveLifecycle.isDisabled || tokenSaveLifecycle.isInFlight)
                    .accessibilityIdentifier("v2-get-started-save-token")

                    Button("Clear Token", role: .destructive) {
                        store.clearStoredAccessToken()
                    }
                    .buttonStyle(.bordered)
                    .disabled(tokenClearLifecycle.isDisabled || tokenClearLifecycle.isInFlight)
                    .accessibilityIdentifier("v2-get-started-clear-token")

                    Spacer(minLength: 8)

                    Button(daemonProbeLifecycle.isInFlight ? "Verifying…" : "Verify Daemon") {
                        Task {
                            await store.probeDaemonConnection()
                        }
                    }
                    .buttonStyle(.borderedProminent)
                    .tint(.paInfo)
                    .disabled(daemonProbeLifecycle.isDisabled || daemonProbeLifecycle.isInFlight)
                    .accessibilityIdentifier("v2-get-started-verify-daemon")
                }
                .controlSize(.small)

                Text("Token reference: \(store.tokenReferenceLabel)")
                    .font(.paCaption)
                    .foregroundStyle(Color.paTextTertiary)

                if let lastUpdatedAt = store.getStartedReadinessSnapshot.lastUpdatedAt {
                    Text("Live readiness updated \(lastUpdatedAt.formatted(date: .omitted, time: .shortened))")
                        .font(.paCaption)
                        .foregroundStyle(Color.paTextTertiary)
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private var nextActionCard: some View {
        PASurfaceCard("Next Step", tone: nextStep == nil ? .emerald : .neutral) {
            VStack(alignment: .leading, spacing: 8) {
                HStack(spacing: 8) {
                    ProgressView(value: store.setupProgress)
                        .tint(.paInfo)
                    PAStatusChip(
                        label: "\(completedStepCount) of \(totalStepCount) complete",
                        systemImage: "checkmark.circle",
                        tone: .info
                    )
                }

                Group {
                    if let nextStep {
                        VStack(alignment: .leading, spacing: 3) {
                            Text(nextStep.title)
                                .font(.system(size: 14, weight: .semibold, design: .rounded))
                            Text(nextStep.detail)
                                .font(.paBody)
                                .foregroundStyle(Color.paTextSecondary)
                        }
                        HStack(spacing: 8) {
                            Button(nextStep.actionTitle) {
                                nextStep.action()
                            }
                            .buttonStyle(.borderedProminent)
                            .tint(.paInfo)

                            Button("Open Replay & Ask") {
                                store.selectedSection = .replayAndAsk
                            }
                            .buttonStyle(.bordered)
                        }
                    } else {
                        HStack(spacing: 8) {
                            PAInlineBanner(text: "Setup complete. You are ready to audit live assistant behavior.", tone: .success)

                            Button("Open Replay & Ask") {
                                store.selectedSection = .replayAndAsk
                            }
                            .buttonStyle(.borderedProminent)
                            .tint(.paInfo)
                        }
                        .frame(maxWidth: .infinity, alignment: .leading)
                    }
                }
                .controlSize(.small)
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private var checklistCard: some View {
        PASurfaceCard("Checklist") {
            VStack(alignment: .leading, spacing: 8) {
                ForEach(Array(store.setupChecklist.enumerated()), id: \.element.id) { index, item in
                    HStack(alignment: .top, spacing: 10) {
                        stepMarker(index: index + 1, isDone: item.isDone)

                        VStack(alignment: .leading, spacing: 3) {
                            HStack(spacing: 6) {
                                Text(item.title)
                                    .font(.system(size: 13, weight: .semibold, design: .rounded))
                                if item.isDone {
                                    PAStatusChip(label: "Done", systemImage: "checkmark", tone: .success)
                                }
                            }
                            Text(item.detail)
                                .font(.paCaption)
                                .foregroundStyle(Color.paTextSecondary)

                            if let statusMessage = store.setupActionStatus(for: item.id), !item.isDone {
                                Text(statusMessage)
                                    .font(.paCaption)
                                    .foregroundStyle(Color.paInfo)
                            }
                        }
                        .frame(maxWidth: .infinity, alignment: .leading)

                        Spacer(minLength: 8)

                        if !item.isDone {
                            Button(item.actionTitle) {
                                item.action()
                            }
                            .buttonStyle(.bordered)
                            .tint(.paInfo)
                            .controlSize(.small)
                        }
                    }
                    .padding(6)
                    .paSubsurface(rowTone(for: index, isDone: item.isDone))

                    if index < store.setupChecklist.count - 1 {
                        Divider().overlay(Color.paStrokeSoft)
                    }
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private var nextStep: SetupChecklistItem? {
        store.setupChecklist.first(where: { !$0.isDone })
    }

    private var completedStepCount: Int {
        store.setupChecklist.filter(\.isDone).count
    }

    private var totalStepCount: Int {
        store.setupChecklist.count
    }

    private var daemonBaseURLBinding: Binding<String> {
        Binding(
            get: { store.daemonBaseURL },
            set: { store.daemonBaseURL = $0 }
        )
    }

    private var workspaceIDBinding: Binding<String> {
        Binding(
            get: { store.workspaceID },
            set: { store.workspaceID = $0 }
        )
    }

    private var principalActorIDBinding: Binding<String> {
        Binding(
            get: { store.principalActorID },
            set: { store.principalActorID = $0 }
        )
    }

    private var densityModeBinding: Binding<V2InformationDensityMode> {
        Binding(
            get: { store.informationDensityMode },
            set: { store.informationDensityMode = $0 }
        )
    }

    private var firstPendingIndex: Int? {
        store.setupChecklist.firstIndex(where: { !$0.isDone })
    }

    private var tokenSaveLifecycle: V2MutationLifecycleState {
        store.mutationLifecycle(for: .tokenSave)
    }

    private var tokenClearLifecycle: V2MutationLifecycleState {
        store.mutationLifecycle(for: .tokenClear)
    }

    private var daemonProbeLifecycle: V2MutationLifecycleState {
        store.mutationLifecycle(for: .daemonProbe)
    }

    private var lifecycleChipLabel: String {
        if !store.sessionReadiness.isReadyForDaemonMutations {
            return "Lifecycle Pending"
        }
        if store.getStartedReadinessSnapshot.lifecycleIsOperational {
            return "Lifecycle Ready"
        }
        if store.getStartedReadinessSnapshot.lifecycleError != nil {
            return "Lifecycle Check Failed"
        }
        return "Lifecycle Needs Attention"
    }

    private var lifecycleChipSymbol: String {
        if store.getStartedReadinessSnapshot.lifecycleIsOperational {
            return "checkmark.shield.fill"
        }
        return "exclamationmark.triangle.fill"
    }

    private var lifecycleChipTone: PAStatusTone {
        store.getStartedReadinessSnapshot.lifecycleIsOperational ? .success : .warning
    }

    private var routeChipLabel: String {
        if !store.sessionReadiness.isReadyForDaemonMutations {
            return "Route Pending"
        }
        if store.getStartedReadinessSnapshot.hasRouteResolution {
            return "Route Ready"
        }
        if store.getStartedReadinessSnapshot.modelRouteError != nil {
            return "Route Check Failed"
        }
        return "Route Missing"
    }

    private var routeChipSymbol: String {
        store.getStartedReadinessSnapshot.hasRouteResolution ? "bolt.fill" : "bolt.slash.fill"
    }

    private var routeChipTone: PAStatusTone {
        store.getStartedReadinessSnapshot.hasRouteResolution ? .success : .warning
    }

    private var connectorChipLabel: String {
        if !store.sessionReadiness.isReadyForDaemonMutations {
            return "Connectors Pending"
        }
        if store.connectedLiveConnectorCount > 0 {
            return "\(store.connectedLiveConnectorCount) Connector(s) Ready"
        }
        if store.getStartedReadinessSnapshot.connectorError != nil {
            return "Connector Check Failed"
        }
        return "No Healthy Connector"
    }

    private var connectorChipSymbol: String {
        store.connectedLiveConnectorCount > 0 ? "link.circle.fill" : "link.badge.plus"
    }

    private var connectorChipTone: PAStatusTone {
        store.connectedLiveConnectorCount > 0 ? .success : .warning
    }

    private var replayChipLabel: String {
        if !store.sessionReadiness.isReadyForDaemonMutations {
            return "Replay Pending"
        }
        if store.liveReplayInstructionCount > 0 {
            return "\(store.liveReplayInstructionCount) Replay Signal(s)"
        }
        if store.getStartedReadinessSnapshot.replayError != nil {
            return "Replay Check Failed"
        }
        return "No Replay Signals"
    }

    private var replayChipSymbol: String {
        store.liveReplayInstructionCount > 0 ? "timeline.selection" : "tray"
    }

    private var replayChipTone: PAStatusTone {
        store.liveReplayInstructionCount > 0 ? .info : .warning
    }

    private func rowTone(for index: Int, isDone: Bool) -> PACardTone {
        if isDone {
            return .neutral
        }
        return firstPendingIndex == index ? .cool : .neutral
    }

    private func stepMarker(index: Int, isDone: Bool) -> some View {
        ZStack {
            Circle()
                .fill((isDone ? Color.paSuccess : Color.paSurfaceMuted).opacity(0.28))
                .frame(width: 20, height: 20)
            if isDone {
                Image(systemName: "checkmark")
                    .font(.system(size: 10, weight: .semibold))
                    .foregroundStyle(Color.paSuccess)
            } else {
                Text("\(index)")
                    .font(.system(size: 10, weight: .semibold, design: .rounded))
                    .foregroundStyle(Color.paTextSecondary)
            }
        }
    }
}
