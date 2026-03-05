import SwiftUI

public struct TaskbarMenuView: View {
    @ObservedObject private var state: AppShellState
    private let openMainWindow: () -> Void
    private let closeMainWindow: () -> Void
    private let quitApp: () -> Void

    public init(
        state: AppShellState,
        openMainWindow: @escaping () -> Void,
        closeMainWindow: @escaping () -> Void,
        quitApp: @escaping () -> Void
    ) {
        self.state = state
        self.openMainWindow = openMainWindow
        self.closeMainWindow = closeMainWindow
        self.quitApp = quitApp
    }

    public var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            header

            runtimeSection

            Divider()

            readinessSection

            Divider()

            daemonControlSection

            Divider()

            appSection
        }
        .padding(10)
        .frame(maxWidth: .infinity, alignment: .leading)
        .controlSize(.small)
    }

    private var header: some View {
        HStack(spacing: 8) {
            Text("Personal Agent")
                .font(.headline)
            Spacer(minLength: 6)
            if state.isDaemonLifecycleLoading || state.isDaemonControlInFlight {
                ProgressView()
                    .controlSize(.small)
            }
        }
    }

    private var runtimeSection: some View {
        VStack(alignment: .leading, spacing: 6) {
            statusRow(
                label: "Daemon",
                value: runtimeDaemonStatusValue,
                symbolName: runtimeDaemonStatusSymbol,
                tint: runtimeDaemonStatusTint
            )
            statusRow(
                label: "Connection",
                value: runtimeConnectionStatusValue,
                symbolName: runtimeConnectionStatusSymbol,
                tint: runtimeConnectionStatusTint
            )

            Button {
                state.refreshDaemonStatus()
            } label: {
                Label("Refresh", systemImage: "arrow.clockwise")
            }
            .buttonStyle(.bordered)
            .disabled(state.isDaemonLifecycleLoading)
        }
    }

    private var daemonControlSection: some View {
        VStack(alignment: .leading, spacing: 6) {
            Text("Daemon Controls")
                .font(.caption.weight(.semibold))
                .foregroundStyle(.secondary)

            ControlGroup {
                Button("Start") { state.requestStartDaemon() }
                    .disabled(!state.daemonCanStart || state.isDaemonControlInFlight)
                Button("Stop") { state.requestStopDaemon() }
                    .disabled(!state.daemonCanStop || state.isDaemonControlInFlight)
                Button("Restart") { state.requestRestartDaemon() }
                    .disabled(!state.daemonCanRestart || state.isDaemonControlInFlight)
            }
            .controlGroupStyle(.automatic)
            .buttonStyle(.bordered)
        }
    }

    private var readinessSection: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(spacing: 8) {
                Text("Readiness")
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(.secondary)
                Spacer(minLength: 0)
                if state.setupReadinessChecksLoading {
                    Label("Checking", systemImage: "clock.arrow.circlepath")
                        .font(.caption2.weight(.semibold))
                        .foregroundStyle(.secondary)
                } else if readinessIssues.isEmpty {
                    Label("Ready", systemImage: "checkmark.circle.fill")
                        .font(.caption2.weight(.semibold))
                        .foregroundStyle(.green)
                }
            }

            if state.setupReadinessChecksLoading {
                Text("Setup checks in progress.")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            } else if readinessIssues.isEmpty {
                Text("Core setup checks are healthy.")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            } else {
                ForEach(readinessIssues) { issue in
                    readinessRow(issue)
                }
            }
        }
    }

    private func readinessRow(_ issue: TaskbarReadinessIssue) -> some View {
        HStack(alignment: .top, spacing: 8) {
            Image(systemName: issue.symbolName)
                .font(.caption.weight(.semibold))
                .foregroundStyle(issue.tint)
                .frame(width: 12, alignment: .center)

            VStack(alignment: .leading, spacing: 2) {
                Text(issue.title)
                    .font(.caption.weight(.semibold))
                Text(issue.detail)
                    .font(.caption2)
                    .foregroundStyle(.secondary)
                    .lineLimit(2)
            }

            Spacer(minLength: 6)

            Button(issue.actionTitle) {
                runReadinessAction(issue)
            }
            .buttonStyle(.bordered)
            .disabled(!isReadinessActionEnabled(issue))
        }
    }

    private var appSection: some View {
        HStack(spacing: 8) {
            Button {
                if state.mainWindowVisible {
                    closeMainWindow()
                } else {
                    openMainWindow()
                }
            } label: {
                Text(state.mainWindowVisible ? "Close Window" : "Open Window")
            }
            .buttonStyle(.bordered)

            Spacer(minLength: 6)

            Button(role: .destructive) {
                quitApp()
            } label: {
                Text("Quit")
            }
            .buttonStyle(.bordered)
        }
    }

    private func statusRow(label: String, value: String, symbolName: String, tint: Color) -> some View {
        LabeledContent(label) {
            Label(value, systemImage: symbolName)
                .font(.caption)
                .foregroundStyle(tint)
        }
    }

    private var readinessIssues: [TaskbarReadinessIssue] {
        var issues: [TaskbarReadinessIssue] = []
        switch state.effectiveDaemonControlAuthState {
        case .missing:
            issues.append(.tokenMissing)
        case .configured, .unknown:
            break
        }
        if state.daemonNeedsInstall {
            issues.append(.daemonInstall)
        } else if state.daemonHasWorkerFailureRepairState {
            issues.append(.workerDegraded)
        } else if state.daemonNeedsRepair {
            issues.append(.daemonRepair)
        } else if state.daemonStatus != .running {
            issues.append(.daemonNotRunning)
        }
        if !state.onboardingProviderReady {
            issues.append(.providerSetup)
        } else if !state.onboardingChatRouteReady {
            issues.append(.chatRoute)
        }
        return Array(issues.prefix(2))
    }

    private var runtimeDaemonStatusValue: String {
        if state.isRuntimeStatusBootstrapLoading {
            return "Checking"
        }
        return state.daemonStatus.label.replacingOccurrences(of: "Daemon: ", with: "")
    }

    private var runtimeDaemonStatusSymbol: String {
        state.isRuntimeStatusBootstrapLoading ? "clock.arrow.circlepath" : state.daemonStatus.symbolName
    }

    private var runtimeDaemonStatusTint: Color {
        state.isRuntimeStatusBootstrapLoading ? .secondary : state.daemonStatus.tint
    }

    private var runtimeConnectionStatusValue: String {
        if state.isRuntimeStatusBootstrapLoading {
            return "Checking"
        }
        return state.connectionStatus.label.replacingOccurrences(of: "App Connection: ", with: "")
    }

    private var runtimeConnectionStatusSymbol: String {
        state.isRuntimeStatusBootstrapLoading ? "clock.arrow.circlepath" : state.connectionStatus.symbolName
    }

    private var runtimeConnectionStatusTint: Color {
        state.isRuntimeStatusBootstrapLoading ? .secondary : state.connectionStatus.tint
    }

    private func runReadinessAction(_ issue: TaskbarReadinessIssue) {
        switch issue {
        case .tokenMissing:
            openMainWindow()
            state.navigateToSection(.configuration)
        case .daemonInstall:
            state.requestInstallDaemon()
        case .workerDegraded:
            openMainWindow()
            state.navigateToSection(.channels)
        case .daemonRepair:
            state.requestRepairDaemonInstallation()
        case .daemonNotRunning:
            state.requestStartDaemon()
        case .providerSetup, .chatRoute:
            openMainWindow()
            state.navigateToSection(.models)
        }
    }

    private func isReadinessActionEnabled(_ issue: TaskbarReadinessIssue) -> Bool {
        switch issue {
        case .daemonInstall:
            return state.daemonCanInstallFromBundle
        case .workerDegraded:
            return true
        case .daemonRepair:
            return state.daemonCanRepairFromBundle
        case .daemonNotRunning:
            return state.daemonCanStart && !state.isDaemonControlInFlight
        case .tokenMissing, .providerSetup, .chatRoute:
            return true
        }
    }
}

private enum TaskbarReadinessIssue: String, Identifiable {
    case tokenMissing
    case daemonInstall
    case workerDegraded
    case daemonRepair
    case daemonNotRunning
    case providerSetup
    case chatRoute

    var id: String { rawValue }

    var title: String {
        switch self {
        case .tokenMissing:
            return "Auth token required"
        case .daemonInstall:
            return "Daemon install required"
        case .workerDegraded:
            return "Plugin worker degraded"
        case .daemonRepair:
            return "Daemon setup needs repair"
        case .daemonNotRunning:
            return "Daemon is not running"
        case .providerSetup:
            return "Provider setup required"
        case .chatRoute:
            return "Chat model route required"
        }
    }

    var detail: String {
        switch self {
        case .tokenMissing:
            return "Save Assistant Access Token in Configuration."
        case .daemonInstall:
            return "Install daemon runtime before chat workflows."
        case .workerDegraded:
            return "Open Channels to inspect failed plugin workers."
        case .daemonRepair:
            return "Run repair to restore daemon runtime health."
        case .daemonNotRunning:
            return "Start daemon to restore runtime connectivity."
        case .providerSetup:
            return "Configure at least one provider in Models."
        case .chatRoute:
            return "Select an enabled chat model route in Models."
        }
    }

    var actionTitle: String {
        switch self {
        case .tokenMissing:
            return "Open Config"
        case .daemonInstall:
            return "Install"
        case .workerDegraded:
            return "Open Channels"
        case .daemonRepair:
            return "Repair"
        case .daemonNotRunning:
            return "Start"
        case .providerSetup, .chatRoute:
            return "Open Models"
        }
    }

    var symbolName: String {
        switch self {
        case .tokenMissing:
            return "key.fill"
        case .daemonInstall, .daemonRepair:
            return "wrench.and.screwdriver.fill"
        case .workerDegraded:
            return "exclamationmark.triangle.fill"
        case .daemonNotRunning:
            return "power"
        case .providerSetup, .chatRoute:
            return "cpu"
        }
    }

    var tint: Color {
        switch self {
        case .daemonNotRunning, .providerSetup, .chatRoute, .workerDegraded:
            return .orange
        case .tokenMissing, .daemonInstall, .daemonRepair:
            return .secondary
        }
    }
}
