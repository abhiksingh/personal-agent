import SwiftUI

extension ConfigurationPanelView {
    var advancedModeContent: some View {
        ConfigurationAdvancedModeContent {
            operatorDisclosure(
                title: "Daemon Lifecycle Controls",
                isExpanded: $isAdvancedLifecycleControlsExpanded
            ) {
                advancedDaemonSection
            }
        } timeline: {
            operatorDisclosure(
                title: "Runtime Supervisor Timeline",
                isExpanded: $isRuntimeSupervisorTimelineExpanded
            ) {
                runtimeSupervisorTimelineSection
            }
        } performance: {
            operatorDisclosure(
                title: "Panel Latency Budgets",
                isExpanded: $isPanelLatencyDiagnosticsExpanded
            ) {
                panelLatencyDiagnosticsSection
            }
        }
    }

    var runtimeSupervisorTimelineSection: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text(
                state.daemonPluginLifecycleHistoryStatusMessage
                    ?? "Runtime plugin lifecycle history has not been queried yet."
            )
            .font(.caption)
            .foregroundStyle(.secondary)

            GroupBox("Filters") {
                VStack(alignment: .leading, spacing: 8) {
                    TextField("Plugin ID (optional)", text: $state.daemonPluginLifecycleHistoryFilterPluginID)
                        .textFieldStyle(.roundedBorder)

                    HStack(spacing: 8) {
                        Picker("Kind", selection: $state.daemonPluginLifecycleHistoryFilterKind) {
                            ForEach(state.daemonPluginLifecycleHistoryKindOptions, id: \.self) { option in
                                Text(runtimeLifecycleFilterOptionLabel(option)).tag(option)
                            }
                        }
                        .pickerStyle(.menu)

                        Picker("State", selection: $state.daemonPluginLifecycleHistoryFilterState) {
                            ForEach(state.daemonPluginLifecycleHistoryStateOptions, id: \.self) { option in
                                Text(runtimeLifecycleFilterOptionLabel(option)).tag(option)
                            }
                        }
                        .pickerStyle(.menu)
                    }

                    Picker("Event Type", selection: $state.daemonPluginLifecycleHistoryFilterEventType) {
                        ForEach(state.daemonPluginLifecycleHistoryEventTypeOptions, id: \.self) { option in
                            Text(runtimeLifecycleFilterOptionLabel(option)).tag(option)
                        }
                    }
                    .pickerStyle(.menu)

                    Stepper(
                        "Limit: \(state.daemonPluginLifecycleHistoryLimit)",
                        value: $state.daemonPluginLifecycleHistoryLimit,
                        in: 10...200,
                        step: 10
                    )

                    HStack(spacing: 8) {
                        Button("Refresh Timeline") {
                            state.refreshDaemonPluginLifecycleHistory()
                        }
                        .buttonStyle(.bordered)
                        .disabled(state.isDaemonPluginLifecycleHistoryLoading)

                        Button("Reset Filters") {
                            state.resetDaemonPluginLifecycleHistoryFilters()
                            state.refreshDaemonPluginLifecycleHistory()
                        }
                        .buttonStyle(.bordered)
                        .disabled(state.isDaemonPluginLifecycleHistoryLoading)

                        if state.isDaemonPluginLifecycleHistoryLoading {
                            ProgressView()
                                .controlSize(.small)
                        }
                    }
                }
            }

            GroupBox("Plugin Health Trend (\(state.daemonPluginLifecycleTrendItems.count))") {
                VStack(alignment: .leading, spacing: 8) {
                    if state.daemonPluginLifecycleTrendItems.isEmpty {
                        Text("No plugin lifecycle trend data yet for the current filter set.")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    } else {
                        ForEach(state.daemonPluginLifecycleTrendItems) { item in
                            runtimePluginTrendRow(item)
                        }
                    }
                }
            }

            GroupBox("Lifecycle Events (\(state.daemonPluginLifecycleHistoryItems.count))") {
                VStack(alignment: .leading, spacing: 8) {
                    if state.isDaemonPluginLifecycleHistoryLoading && state.daemonPluginLifecycleHistoryItems.isEmpty {
                        HStack(spacing: 8) {
                            ProgressView()
                                .controlSize(.small)
                            Text("Loading runtime plugin lifecycle events…")
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                    } else if state.daemonPluginLifecycleHistoryItems.isEmpty {
                        Text("No runtime plugin lifecycle events matched the current filters.")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    } else {
                        ForEach(Array(state.daemonPluginLifecycleHistoryItems.enumerated()), id: \.element.id) { index, item in
                            if index > 0 {
                                Divider()
                            }
                            runtimePluginEventRow(item)
                        }
                    }

                    if state.daemonPluginLifecycleHistoryHasMore {
                        Text("More lifecycle events are available. Narrow filters or lower timeline noise to inspect quickly.")
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                    }
                }
            }
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .cardSurface(.standard)
    }

    var panelLatencyDiagnosticsSection: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text(state.panelLatencyStatusMessage)
                .font(.caption)
                .foregroundStyle(.secondary)

            HStack(spacing: 8) {
                TahoeStatusBadge(
                    text: "Samples \(state.panelLatencySampleCount)",
                    symbolName: "speedometer",
                    tint: .secondary
                )
                .controlSize(.small)

                TahoeStatusBadge(
                    text: "Regressions \(state.panelLatencyRegressionSamples.count)",
                    symbolName: state.panelLatencyRegressionSamples.isEmpty
                        ? "checkmark.circle.fill"
                        : "exclamationmark.triangle.fill",
                    tint: state.panelLatencyRegressionSamples.isEmpty ? .green : .orange
                )
                .controlSize(.small)
            }

            GroupBox("Latest by Section (\(state.panelLatencyLatestSamplesSorted.count))") {
                VStack(alignment: .leading, spacing: 8) {
                    if state.panelLatencyLatestSamplesSorted.isEmpty {
                        Text("No panel latency samples yet. Refresh any panel to capture instrumentation.")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    } else {
                        ForEach(Array(state.panelLatencyLatestSamplesSorted.enumerated()), id: \.element.id) { index, sample in
                            if index > 0 {
                                Divider()
                            }
                            panelLatencySampleRow(sample)
                        }
                    }
                }
            }

            HStack(spacing: 8) {
                Button("Capture Current Panel") {
                    state.navigateToSection(state.selectedSection)
                }
                .buttonStyle(.bordered)

                Button("Clear Samples") {
                    state.clearPanelLatencySamples()
                }
                .buttonStyle(.bordered)
                .disabled(state.panelLatencySampleCount == 0)
            }
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .cardSurface(.standard)
    }

    @ViewBuilder
    func panelLatencySampleRow(_ sample: UIPanelLatencySample) -> some View {
        HStack(alignment: .firstTextBaseline, spacing: 8) {
            VStack(alignment: .leading, spacing: 4) {
                Text(sample.sectionTitle)
                    .font(.subheadline.weight(.semibold))
                Text("\(sample.category.title) • Captured \(panelLatencyTimestampLabel(sample.capturedAt))")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            Spacer(minLength: 8)
            Text("\(sample.durationMS) ms / \(sample.budgetMS) ms")
                .font(.caption.monospacedDigit())
                .foregroundStyle(.secondary)
            TahoeStatusBadge(
                text: sample.isOverBudget ? "Over Budget" : "Within Budget",
                symbolName: sample.isOverBudget ? "exclamationmark.triangle.fill" : "checkmark.circle.fill",
                tint: sample.isOverBudget ? .orange : .green
            )
            .controlSize(.small)
        }
    }

    func panelLatencyTimestampLabel(_ rawTimestamp: String) -> String {
        let parserWithFractionalSeconds = ISO8601DateFormatter()
        parserWithFractionalSeconds.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        let parser = ISO8601DateFormatter()
        parser.formatOptions = [.withInternetDateTime]
        guard let timestamp = parserWithFractionalSeconds.date(from: rawTimestamp)
            ?? parser.date(from: rawTimestamp) else {
            return rawTimestamp
        }
        return timestamp.formatted(date: .omitted, time: .standard)
    }


    var advancedDaemonSection: some View {
        VStack(alignment: .leading, spacing: 10) {
            sectionTitle("Advanced Daemon Lifecycle")

            HStack(spacing: 8) {
                TahoeStatusBadge(
                    text: daemonInstallStatusText,
                    symbolName: daemonInstallStatusSymbol,
                    tint: daemonInstallStatusTint
                )
                Spacer()
            }

            HStack(spacing: 8) {
                Button("Install") {
                    state.requestInstallDaemon()
                }
                .quietButtonChrome()
                .disabled(!state.daemonCanInstallFromBundle)

                Button("Uninstall") {
                    state.requestUninstallDaemon()
                }
                .quietButtonChrome()
                .disabled(!state.daemonCanUninstall || state.isDaemonControlInFlight)

                Button("Repair") {
                    state.requestRepairDaemonInstallation()
                }
                .quietButtonChrome()
                .disabled(!state.daemonCanRepairFromBundle)
            }

            HStack(spacing: 8) {
                Button("Start") {
                    state.requestStartDaemon()
                }
                .quietButtonChrome()
                .disabled(!state.daemonCanStart || state.isDaemonControlInFlight)

                Button("Stop") {
                    state.requestStopDaemon()
                }
                .quietButtonChrome()
                .disabled(!state.daemonCanStop || state.isDaemonControlInFlight)

                Button("Restart") {
                    state.requestRestartDaemon()
                }
                .quietButtonChrome()
                .disabled(!state.daemonCanRestart || state.isDaemonControlInFlight)

                if state.isDaemonControlInFlight {
                    ProgressView()
                        .controlSize(.small)
                }
            }

            Text("Install/Repair use bundled helper assets; runtime controls remain daemon API-backed.")
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .cardSurface(.standard)
    }

    func runtimePluginTrendRow(_ item: RuntimePluginLifecycleTrendItem) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(spacing: 8) {
                Text(item.pluginID)
                    .font(.caption.weight(.semibold))
                    .textSelection(.enabled)
                TahoeStatusBadge(
                    text: item.kindLabel,
                    symbolName: item.kind.lowercased() == "channel" ? "dot.radiowaves.left.and.right" : "puzzlepiece.extension",
                    tint: item.kind.lowercased() == "channel" ? .blue : .teal
                )
                TahoeStatusBadge(
                    text: runtimeLifecycleStateLabel(item.latestState),
                    symbolName: "circle.fill",
                    tint: runtimeLifecycleStateTint(item.latestState)
                )
                Spacer(minLength: 0)
                Text(item.latestOccurredAtLabel)
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }

            HStack(spacing: 8) {
                TahoeStatusBadge(
                    text: "Restarts \(item.restartEvents)",
                    symbolName: "arrow.clockwise.circle.fill",
                    tint: .orange
                )
                TahoeStatusBadge(
                    text: "Failures \(item.failureEvents)",
                    symbolName: "xmark.octagon.fill",
                    tint: item.failureEvents > 0 ? .red : .secondary
                )
                TahoeStatusBadge(
                    text: "Recoveries \(item.recoveryEvents)",
                    symbolName: "checkmark.circle.fill",
                    tint: item.recoveryEvents > 0 ? .green : .secondary
                )
                Text("Events \(item.totalEvents)")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(8)
        .cardSurface(.subtle)
    }

    func runtimePluginEventRow(_ item: RuntimePluginLifecycleEventItem) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(spacing: 8) {
                Text(item.pluginID)
                    .font(.caption.weight(.semibold))
                    .textSelection(.enabled)
                TahoeStatusBadge(
                    text: item.kindLabel,
                    symbolName: item.kind.lowercased() == "channel" ? "dot.radiowaves.left.and.right" : "puzzlepiece.extension",
                    tint: item.kind.lowercased() == "channel" ? .blue : .teal
                )
                TahoeStatusBadge(
                    text: runtimeLifecycleStateLabel(item.state),
                    symbolName: "circle.fill",
                    tint: runtimeLifecycleStateTint(item.state)
                )
                TahoeStatusBadge(
                    text: runtimeLifecycleEventTypeLabel(item.eventType),
                    symbolName: runtimeLifecycleEventSymbol(item: item),
                    tint: runtimeLifecycleEventTint(item: item)
                )
                Spacer(minLength: 0)
                Text(item.occurredAtLabel)
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }

            Text("Reason: \(runtimeLifecycleReasonLabel(item.reason)) • Restarts: \(item.restartCount) • PID: \(item.processID)")
                .font(.caption2)
                .foregroundStyle(.secondary)

            if let error = item.error {
                Text("Error: \(error)")
                    .font(.caption2)
                    .foregroundStyle(.orange)
            }

            if let heartbeat = item.lastHeartbeatAtLabel {
                Text("Last heartbeat: \(heartbeat)")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }
            if let transition = item.lastTransitionAtLabel {
                Text("Last transition: \(transition)")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }

            HStack(spacing: 8) {
                Button("Open Inspect") {
                    state.openInspectForRuntimePluginLifecycle(item)
                }
                .buttonStyle(.bordered)
                .controlSize(.small)

                if let runtimeDestinationTitle = runtimeLifecycleDestinationTitle(for: item.kind) {
                    Button(runtimeDestinationTitle) {
                        state.openRuntimeDiagnosticsForPluginLifecycle(item)
                    }
                    .buttonStyle(.bordered)
                    .controlSize(.small)
                }
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    func runtimeLifecycleFilterOptionLabel(_ raw: String) -> String {
        let trimmed = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmed.lowercased() == "all" {
            return "All"
        }
        return trimmed
            .replacingOccurrences(of: "PLUGIN_", with: "")
            .replacingOccurrences(of: "_", with: " ")
            .capitalized
    }

    func contextFilterOptionLabel(_ raw: String) -> String {
        let trimmed = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmed.lowercased() == "all" {
            return "All"
        }
        return trimmed.replacingOccurrences(of: "_", with: " ").capitalized
    }

    func runtimeLifecycleEventTypeLabel(_ raw: String) -> String {
        raw
            .replacingOccurrences(of: "PLUGIN_", with: "")
            .replacingOccurrences(of: "_", with: " ")
            .capitalized
    }

    func runtimeLifecycleReasonLabel(_ raw: String) -> String {
        raw.replacingOccurrences(of: "_", with: " ").capitalized
    }

    func runtimeLifecycleStateLabel(_ raw: String) -> String {
        raw.replacingOccurrences(of: "_", with: " ").capitalized
    }

    func runtimeLifecycleStateTint(_ raw: String) -> Color {
        switch raw.lowercased() {
        case "running":
            return .green
        case "starting", "registered", "restarting":
            return .orange
        case "failed":
            return .red
        case "stopped":
            return .secondary
        default:
            return .secondary
        }
    }

    func runtimeLifecycleEventTint(item: RuntimePluginLifecycleEventItem) -> Color {
        if item.failureEvent {
            return .red
        }
        if item.recoveryEvent {
            return .green
        }
        if item.restartEvent {
            return .orange
        }
        return .secondary
    }

    func runtimeLifecycleEventSymbol(item: RuntimePluginLifecycleEventItem) -> String {
        if item.failureEvent {
            return "xmark.octagon.fill"
        }
        if item.recoveryEvent {
            return "checkmark.arrow.trianglehead.counterclockwise"
        }
        if item.restartEvent {
            return "arrow.clockwise.circle.fill"
        }
        return "clock.arrow.circlepath"
    }

    func runtimeLifecycleDestinationTitle(for kind: String) -> String? {
        switch kind.lowercased() {
        case "channel":
            return "Open Channels"
        case "connector":
            return "Open Connectors"
        default:
            return nil
        }
    }
}
