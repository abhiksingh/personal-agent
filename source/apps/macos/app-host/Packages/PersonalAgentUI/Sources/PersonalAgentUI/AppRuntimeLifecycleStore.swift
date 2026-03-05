import Foundation
import SwiftUI

@MainActor
final class AppRuntimeLifecycleStore: ObservableObject {
    @Published var daemonStatus: DaemonStatus = .unknown
    @Published var connectionStatus: ConnectionStatus = .disconnected
    @Published var daemonCanStart = true
    @Published var daemonCanStop = false
    @Published var daemonCanRestart = false
    @Published var daemonCanInstall = false
    @Published var daemonCanUninstall = false
    @Published var daemonCanRepair = false
    @Published var daemonNeedsInstall = false
    @Published var daemonNeedsRepair = false
    @Published var daemonControlOperationAction = ""
    @Published var daemonControlOperationState = "idle"
    @Published var daemonStatusDetail = "Waiting for daemon status..."
    @Published var isDaemonLifecycleLoading = false
    @Published var isDaemonControlInFlight = false
    @Published var hasLoadedDaemonStatus = false
    @Published var daemonWorkerSummary = DaemonLifecycleWorkerSummary()
    @Published var daemonDatabaseReady = false
    @Published var daemonSetupState = "unknown"
    @Published var daemonRepairHint = ""
    @Published var daemonLifecycleOverallState = "unknown"
    @Published var daemonCoreRuntimeState = "unknown"
    @Published var daemonPluginRuntimeState = "unknown"
    @Published var daemonLifecycleBlocking = false
    @Published var daemonControlAuthState: DaemonControlAuthState = .unknown
    @Published var daemonControlAuthSource = "unknown"
    @Published var daemonControlAuthRemediationHints: [String] = []

    private var daemonHasStructuredLifecycleClassification: Bool {
        daemonLifecycleOverallState != "unknown" &&
            daemonCoreRuntimeState != "unknown" &&
            daemonPluginRuntimeState != "unknown"
    }

    func effectiveDaemonControlAuthState(localDevTokenConfigured: Bool) -> DaemonControlAuthState {
        if hasLoadedDaemonStatus {
            return daemonControlAuthState
        }
        return localDevTokenConfigured ? .configured : .missing
    }

    func daemonControlAuthNeedsRemediation(localDevTokenConfigured: Bool) -> Bool {
        effectiveDaemonControlAuthState(localDevTokenConfigured: localDevTokenConfigured) != .configured
    }

    func daemonControlAuthSetupDetail(localDevTokenConfigured: Bool) -> String {
        switch effectiveDaemonControlAuthState(localDevTokenConfigured: localDevTokenConfigured) {
        case .configured:
            return "Token configured for authenticated daemon requests."
        case .missing:
            if let hint = daemonControlAuthRemediationHints.first(where: { !$0.isEmpty }) {
                return hint
            }
            return "Save an Assistant Access Token to authorize daemon requests."
        case .unknown:
            return "Checking daemon auth state."
        }
    }

    var daemonHasWorkerFailureRepairState: Bool {
        if daemonHasStructuredLifecycleClassification {
            return daemonCoreRuntimeState == "ready" && daemonPluginRuntimeState == "degraded"
        }
        return daemonNeedsRepair &&
            daemonDatabaseReady &&
            daemonSetupState == "repair_required" &&
            daemonWorkerSummary.failed > 0
    }

    var daemonNeedsInfrastructureRepair: Bool {
        if daemonHasStructuredLifecycleClassification {
            if daemonLifecycleBlocking {
                return daemonCoreRuntimeState != "install_required"
            }
            return daemonNeedsRepair && !daemonHasWorkerFailureRepairState
        }
        return daemonNeedsRepair && !daemonHasWorkerFailureRepairState
    }

    func applyMissingTokenState() {
        daemonStatus = .unknown
        connectionStatus = .disconnected
        daemonCanStart = false
        daemonCanStop = false
        daemonCanRestart = false
        daemonCanInstall = false
        daemonCanUninstall = false
        daemonCanRepair = false
        daemonNeedsInstall = false
        daemonNeedsRepair = false
        daemonWorkerSummary = DaemonLifecycleWorkerSummary()
        daemonDatabaseReady = false
        daemonSetupState = "unknown"
        daemonRepairHint = ""
        daemonLifecycleOverallState = "unknown"
        daemonCoreRuntimeState = "unknown"
        daemonPluginRuntimeState = "unknown"
        daemonLifecycleBlocking = false
        daemonControlAuthState = .missing
        daemonControlAuthSource = "local_token_missing"
        daemonControlAuthRemediationHints = ["Save an Assistant Access Token to authorize daemon requests."]
        daemonControlOperationAction = ""
        daemonControlOperationState = "idle"
        daemonStatusDetail = "Set Assistant Access Token to query daemon lifecycle."
    }

    func applyLifecycleError(detail: String) {
        daemonStatus = .unknown
        daemonCanStart = false
        daemonCanStop = false
        daemonCanRestart = false
        daemonCanInstall = false
        daemonCanUninstall = false
        daemonCanRepair = false
        daemonNeedsInstall = false
        daemonNeedsRepair = false
        daemonWorkerSummary = DaemonLifecycleWorkerSummary()
        daemonDatabaseReady = false
        daemonSetupState = "unknown"
        daemonRepairHint = ""
        daemonLifecycleOverallState = "unknown"
        daemonCoreRuntimeState = "unknown"
        daemonPluginRuntimeState = "unknown"
        daemonLifecycleBlocking = false
        daemonControlAuthState = .unknown
        daemonControlAuthSource = "unknown"
        daemonControlAuthRemediationHints = []
        daemonControlOperationAction = ""
        daemonControlOperationState = "idle"
        daemonStatusDetail = detail
    }

    func applyDaemonLifecycleStatus(_ lifecycle: DaemonLifecycleStatusResponse) {
        daemonStatus = daemonStatus(from: lifecycle)
        daemonCanStart = lifecycle.controls.start
        daemonCanStop = lifecycle.controls.stop
        daemonCanRestart = lifecycle.controls.restart
        daemonCanInstall = lifecycle.controls.install
        daemonCanUninstall = lifecycle.controls.uninstall
        daemonCanRepair = lifecycle.controls.repair
        daemonNeedsInstall = lifecycle.needsInstall || lifecycle.setupState == "install_required"
        daemonNeedsRepair = lifecycle.needsRepair || lifecycle.setupState == "repair_required"
        daemonWorkerSummary = lifecycle.workerSummary
        daemonDatabaseReady = lifecycle.databaseReady
        daemonSetupState = lifecycle.setupState
        daemonRepairHint = nonEmpty(lifecycle.repairHint) ?? ""
        daemonLifecycleOverallState = lifecycle.healthClassification.overallState
        daemonCoreRuntimeState = lifecycle.healthClassification.coreRuntimeState
        daemonPluginRuntimeState = lifecycle.healthClassification.pluginRuntimeState
        daemonLifecycleBlocking = lifecycle.healthClassification.blocking
        daemonControlAuthState = daemonControlAuthState(from: lifecycle.controlAuth.state)
        daemonControlAuthSource = lifecycle.controlAuth.source
        daemonControlAuthRemediationHints = lifecycle.controlAuth.remediationHints
        daemonControlOperationAction = lifecycle.controlOperation.action
        daemonControlOperationState = lifecycle.controlOperation.state
        connectionStatus = .connected
        daemonStatusDetail = daemonLifecycleSummary(lifecycle)
    }

    func markDaemonMissing() {
        daemonStatus = .missing
        connectionStatus = .disconnected
        daemonCanStart = false
        daemonCanStop = false
        daemonCanRestart = false
        daemonCanInstall = false
        daemonCanUninstall = false
        daemonCanRepair = false
        daemonNeedsInstall = true
        daemonNeedsRepair = false
        daemonWorkerSummary = DaemonLifecycleWorkerSummary()
        daemonDatabaseReady = false
        daemonSetupState = "install_required"
        daemonRepairHint = ""
        daemonLifecycleOverallState = "blocked"
        daemonCoreRuntimeState = "install_required"
        daemonPluginRuntimeState = "unknown"
        daemonLifecycleBlocking = true
        daemonControlAuthState = .unknown
        daemonControlAuthSource = "unknown"
        daemonControlAuthRemediationHints = []
        daemonControlOperationAction = ""
        daemonControlOperationState = "idle"
        hasLoadedDaemonStatus = true
    }

    func markDaemonBroken() {
        daemonStatus = .broken
        connectionStatus = .degraded
        daemonCanStart = false
        daemonCanStop = false
        daemonCanRestart = false
        daemonCanInstall = false
        daemonCanUninstall = false
        daemonCanRepair = false
        daemonNeedsInstall = false
        daemonNeedsRepair = true
        daemonWorkerSummary = DaemonLifecycleWorkerSummary()
        daemonDatabaseReady = false
        daemonSetupState = "repair_required"
        daemonRepairHint = "Runtime controls may fail until setup is repaired."
        daemonLifecycleOverallState = "blocked"
        daemonCoreRuntimeState = "database_unavailable"
        daemonPluginRuntimeState = "unknown"
        daemonLifecycleBlocking = true
        daemonControlAuthState = .unknown
        daemonControlAuthSource = "unknown"
        daemonControlAuthRemediationHints = []
        daemonControlOperationAction = ""
        daemonControlOperationState = "idle"
        hasLoadedDaemonStatus = true
    }

    func daemonLifecycleActionLabel(_ action: String) -> String {
        switch action.lowercased() {
        case "install":
            return "Install"
        case "uninstall":
            return "Uninstall"
        case "repair":
            return "Repair"
        case "start":
            return "Start"
        case "stop":
            return "Stop"
        case "restart":
            return "Restart"
        default:
            return action.capitalized
        }
    }

    func daemonLifecycleControlResponseSummary(_ response: DaemonLifecycleControlResponse) -> String {
        let label = daemonLifecycleActionLabel(response.action)
        let message = nonEmpty(response.message)
        let errorText = nonEmpty(response.error)
        switch response.operationState.lowercased() {
        case "in_progress":
            if let message {
                return "\(label) in progress. \(message)"
            }
            return "\(label) in progress."
        case "failed":
            if let errorText {
                return "\(label) failed. \(errorText)"
            }
            if let message {
                return "\(label) failed. \(message)"
            }
            return "\(label) failed."
        case "succeeded":
            if let message {
                return "\(label) completed. \(message)"
            }
            return "\(label) completed."
        default:
            if let message {
                return message
            }
            return "\(label) \(response.accepted ? "accepted" : "rejected")."
        }
    }

    private func daemonControlAuthState(from rawState: String) -> DaemonControlAuthState {
        switch rawState.lowercased() {
        case "configured":
            return .configured
        case "missing":
            return .missing
        default:
            return .unknown
        }
    }

    private func daemonStatus(from lifecycle: DaemonLifecycleStatusResponse) -> DaemonStatus {
        let health = lifecycle.healthClassification
        if health.overallState == "blocked" {
            if health.coreRuntimeState == "install_required" {
                return .missing
            }
            return .broken
        }
        if health.overallState == "degraded" {
            return .running
        }

        if lifecycle.needsInstall || lifecycle.setupState == "install_required" || lifecycle.installState == "missing" {
            return .missing
        }
        if lifecycle.needsRepair || lifecycle.setupState == "repair_required" {
            if lifecycle.databaseReady && lifecycle.workerSummary.failed > 0 {
                return .running
            }
            return .broken
        }

        switch lifecycle.lifecycleState.lowercased() {
        case "running", "restart_requested":
            return .running
        case "stop_requested":
            return .stopped
        default:
            return .unknown
        }
    }

    private func daemonLifecycleSummary(_ lifecycle: DaemonLifecycleStatusResponse) -> String {
        var parts: [String] = []
        parts.append("state=\(lifecycle.lifecycleState)")
        if lifecycle.healthClassification.overallState != "unknown" {
            parts.append("health=\(lifecycle.healthClassification.overallState)")
            if lifecycle.healthClassification.coreRuntimeState != "unknown" {
                parts.append("core=\(lifecycle.healthClassification.coreRuntimeState)")
            }
            if lifecycle.healthClassification.pluginRuntimeState != "unknown" {
                parts.append("plugins=\(lifecycle.healthClassification.pluginRuntimeState)")
            }
        }
        if let runtime = nonEmpty(lifecycle.runtimeMode) {
            parts.append("mode=\(runtime)")
        }
        if let bound = nonEmpty(lifecycle.boundAddress) {
            parts.append("bound=\(bound)")
        }
        if lifecycle.controlAuth.state != "unknown" {
            parts.append("auth=\(lifecycle.controlAuth.state)")
        }
        if lifecycle.workerSummary.failed > 0 {
            parts.append("workers_failed=\(lifecycle.workerSummary.failed)")
        }
        let operation = lifecycle.controlOperation
        if operation.state.lowercased() != "idle" {
            let actionLabel = nonEmpty(operation.action).map(daemonLifecycleActionLabel) ?? "daemon"
            var operationSummary = "operation=\(actionLabel.lowercased()):\(operation.state)"
            if let message = nonEmpty(operation.message) {
                operationSummary.append(" (\(message))")
            } else if let error = nonEmpty(operation.error) {
                operationSummary.append(" (\(error))")
            }
            parts.append(operationSummary)
        }
        if let repairHint = nonEmpty(lifecycle.repairHint) {
            parts.append(repairHint)
        }
        return parts.joined(separator: " • ")
    }

    private func nonEmpty(_ value: String?) -> String? {
        guard let trimmed = value?.trimmingCharacters(in: .whitespacesAndNewlines), !trimmed.isEmpty else {
            return nil
        }
        return trimmed
    }
}
