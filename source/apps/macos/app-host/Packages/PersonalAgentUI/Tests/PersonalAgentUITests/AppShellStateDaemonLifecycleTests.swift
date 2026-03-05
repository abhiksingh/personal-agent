import Foundation
import XCTest
@testable import PersonalAgentUI

@MainActor
final class AppShellStateDaemonLifecycleTests: XCTestCase {
    private let tokenDefaultsKey = "personalagent.ui.local_dev_token"
    private let onboardingDefaultsKey = "personalagent.ui.onboarding_complete"

    override func setUp() {
        super.setUp()
        AppShellState._test_setLocalDevTokenSecretReference(
            service: "personalagent.ui.tests.daemon-lifecycle.\(UUID().uuidString)",
            account: "daemon_auth_token"
        )
        AppShellState._test_clearPersistedLocalDevToken()
    }

    override func tearDown() {
        AppShellState._test_clearPersistedLocalDevToken()
        AppShellState._test_resetLocalDevTokenPersistenceHooks()
        AppShellState._test_resetDaemonLocalServiceInstallHooks()
        AppShellState._test_resetDaemonLifecycleControlHooks()
        super.tearDown()
    }

    func testInitWithoutStoredTokenStartsUnconfigured() {
        let defaults = appShellStateTestUserDefaults()
        let priorToken = defaults.object(forKey: tokenDefaultsKey)
        let priorOnboarding = defaults.object(forKey: onboardingDefaultsKey)
        defer {
            if let priorToken {
                defaults.set(priorToken, forKey: tokenDefaultsKey)
            } else {
                defaults.removeObject(forKey: tokenDefaultsKey)
            }
            if let priorOnboarding {
                defaults.set(priorOnboarding, forKey: onboardingDefaultsKey)
            } else {
                defaults.removeObject(forKey: onboardingDefaultsKey)
            }
        }

        defaults.removeObject(forKey: tokenDefaultsKey)
        defaults.removeObject(forKey: onboardingDefaultsKey)

        let state = AppShellState()

        XCTAssertFalse(state.localDevTokenConfigured)
        XCTAssertEqual(state.localDevTokenLastUpdated, "Not configured")
        XCTAssertEqual(state.daemonControlAuthState, .unknown)
    }

    func testInitIgnoresEnvironmentTokenWhenStoredTokenIsMissing() {
        let defaults = appShellStateTestUserDefaults()
        let priorToken = defaults.object(forKey: tokenDefaultsKey)
        let priorOnboarding = defaults.object(forKey: onboardingDefaultsKey)
        let envKey = "PERSONAL_AGENT_DAEMON_TOKEN"
        let priorEnv = getenv(envKey).map { String(cString: $0) }
        defer {
            if let priorToken {
                defaults.set(priorToken, forKey: tokenDefaultsKey)
            } else {
                defaults.removeObject(forKey: tokenDefaultsKey)
            }
            if let priorOnboarding {
                defaults.set(priorOnboarding, forKey: onboardingDefaultsKey)
            } else {
                defaults.removeObject(forKey: onboardingDefaultsKey)
            }
            if let priorEnv {
                setenv(envKey, priorEnv, 1)
            } else {
                unsetenv(envKey)
            }
        }

        defaults.removeObject(forKey: tokenDefaultsKey)
        defaults.removeObject(forKey: onboardingDefaultsKey)
        setenv(envKey, "env-only-token", 1)

        let state = AppShellState()

        XCTAssertFalse(state.localDevTokenConfigured)
        XCTAssertEqual(state.localDevTokenLastUpdated, "Not configured")
    }

    func testInitLoadsTokenFromKeychainPersistence() {
        let state = AppShellState()
        state.localDevTokenInput = "persisted-token"
        state.saveLocalDevToken()

        let reloadedState = AppShellState()

        XCTAssertTrue(reloadedState.localDevTokenConfigured)
        XCTAssertEqual(reloadedState.localDevTokenLastUpdated, "Stored locally")
        XCTAssertEqual(AppShellState._test_readPersistedLocalDevToken(), "persisted-token")
    }

    func testInitMigratesLegacyDefaultsTokenToKeychainAndClearsDefaults() {
        let defaults = appShellStateTestUserDefaults()
        let priorToken = defaults.object(forKey: tokenDefaultsKey)
        defer {
            if let priorToken {
                defaults.set(priorToken, forKey: tokenDefaultsKey)
            } else {
                defaults.removeObject(forKey: tokenDefaultsKey)
            }
        }

        defaults.set("legacy-defaults-token", forKey: tokenDefaultsKey)
        XCTAssertEqual(defaults.string(forKey: tokenDefaultsKey), "legacy-defaults-token")
        XCTAssertEqual(AppShellState._test_loadPersistedLocalDevToken(), "legacy-defaults-token")
        XCTAssertEqual(AppShellState._test_readPersistedLocalDevToken(), "legacy-defaults-token")
        XCTAssertNil(defaults.string(forKey: tokenDefaultsKey))

        let state = AppShellState()

        XCTAssertTrue(state.localDevTokenConfigured)
        XCTAssertEqual(AppShellState._test_loadPersistedLocalDevToken(), "legacy-defaults-token")
    }

    func testSaveLocalDevTokenPersistsToKeychainAndClearsLegacyDefaults() {
        let defaults = appShellStateTestUserDefaults()
        defaults.set("stale-defaults-token", forKey: tokenDefaultsKey)

        let state = AppShellState()
        state.localDevTokenInput = "new-keychain-token"
        state.saveLocalDevToken()

        XCTAssertTrue(state.localDevTokenConfigured)
        XCTAssertEqual(AppShellState._test_readPersistedLocalDevToken(), "new-keychain-token")
        XCTAssertNil(defaults.string(forKey: tokenDefaultsKey))
    }

    func testClearLocalDevTokenRemovesPersistedKeychainToken() {
        let state = AppShellState()
        state.localDevTokenInput = "ephemeral-token"
        state.saveLocalDevToken()

        state.clearLocalDevToken()

        XCTAssertFalse(state.localDevTokenConfigured)
        XCTAssertNil(AppShellState._test_readPersistedLocalDevToken())
    }

    func testEffectiveDaemonControlAuthStateUsesLoadedDaemonStateWhenDisconnected() {
        let state = AppShellState()
        state.localDevTokenConfigured = true
        state.hasLoadedDaemonStatus = true
        state.connectionStatus = .disconnected
        state.daemonControlAuthState = .missing

        XCTAssertEqual(state.effectiveDaemonControlAuthState, .missing)
        XCTAssertTrue(state.daemonControlAuthNeedsRemediation)
    }

    func testInstallDaemonWithoutTokenShowsLifecycleTokenGuidance() async {
        let defaults = appShellStateTestUserDefaults()
        let priorToken = defaults.object(forKey: tokenDefaultsKey)
        let priorOnboarding = defaults.object(forKey: onboardingDefaultsKey)
        defer {
            if let priorToken {
                defaults.set(priorToken, forKey: tokenDefaultsKey)
            } else {
                defaults.removeObject(forKey: tokenDefaultsKey)
            }
            if let priorOnboarding {
                defaults.set(priorOnboarding, forKey: onboardingDefaultsKey)
            } else {
                defaults.removeObject(forKey: onboardingDefaultsKey)
            }
        }

        defaults.removeObject(forKey: tokenDefaultsKey)
        defaults.removeObject(forKey: onboardingDefaultsKey)

        let state = AppShellState()
        state.clearLocalDevToken()
        state.installDaemon()

        try? await Task.sleep(for: .milliseconds(50))

        XCTAssertEqual(
            state.daemonStatusDetail,
            "Set Assistant Access Token before controlling daemon lifecycle."
        )
        XCTAssertEqual(state.isDaemonControlInFlight, false)
    }

    func testInstallDaemonUsesLocalServiceInstallerRunner() async {
        let state = AppShellState()
        state.localDevTokenInput = "install-token"
        state.saveLocalDevToken()

        AppShellState._test_setDaemonLocalServiceInstallRunner { action, authToken in
            XCTAssertEqual(action, "install")
            XCTAssertEqual(authToken, "install-token")
            return DaemonLocalServiceInstallResult(
                daemonAppPath: "/tmp/daemon.app",
                daemonExecutablePath: "/tmp/daemon",
                launchAgentPath: "/tmp/daemon.plist",
                authTokenFilePath: "/tmp/token",
                helperUpdated: false
            )
        }

        await state._test_performDaemonLifecycleControl(action: "install")

        XCTAssertEqual(
            state.daemonStatusDetail,
            "Install completed. Daemon launch agent refreshed."
        )
    }

    func testInstallDaemonReportsDeterministicApplicationsRemediation() async {
        let state = AppShellState()
        state.localDevTokenInput = "install-token"
        state.saveLocalDevToken()

        AppShellState._test_setDaemonLocalServiceInstallRunner { _, _ in
            throw DaemonLocalServiceInstallError.appNotInApplications("/tmp/PersonalAgent.app")
        }

        await state._test_performDaemonLifecycleControl(action: "install")

        XCTAssertEqual(state.daemonControlOperationState, "failed")
        XCTAssertEqual(
            state.daemonStatusDetail,
            "Move PersonalAgent.app to /Applications before running daemon install or repair."
        )
    }

    func testStopDaemonForTerminationDispatchesStopWithoutConfirmation() async throws {
        let state = AppShellState()
        state.localDevTokenInput = "stop-token"
        state.saveLocalDevToken()
        state.daemonCanStop = true
        state.isDaemonControlInFlight = false

        var callCount = 0
        var capturedAction: String?
        var capturedReason: String?
        AppShellState._test_setDaemonLifecycleControlRunner { _, authToken, action, reason in
            XCTAssertEqual(authToken, "stop-token")
            callCount += 1
            capturedAction = action
            capturedReason = reason
            return try self.decodeDaemonLifecycleControlResponse(operationState: "in_progress")
        }

        await state.stopDaemonForTermination(maxWaitSeconds: 0.25)

        XCTAssertEqual(callCount, 1)
        XCTAssertEqual(capturedAction, "stop")
        XCTAssertEqual(capturedReason, "ui:taskbar_quit")
        XCTAssertNil(state.pendingHighImpactActionConfirmation)
        XCTAssertFalse(state.isDaemonControlInFlight)
    }

    func testStopDaemonForTerminationSkipsDispatchWhenStopUnavailable() async throws {
        let state = AppShellState()
        state.localDevTokenInput = "stop-token"
        state.saveLocalDevToken()
        state.daemonCanStop = false

        var callCount = 0
        AppShellState._test_setDaemonLifecycleControlRunner { _, _, _, _ in
            callCount += 1
            return try self.decodeDaemonLifecycleControlResponse(operationState: "succeeded")
        }

        await state.stopDaemonForTermination(maxWaitSeconds: 0.25)

        XCTAssertEqual(callCount, 0)
    }

    func testMarkDaemonMissingResetsSetupControlFlags() {
        let state = AppShellState()
        state.daemonCanInstall = true
        state.daemonCanUninstall = true
        state.daemonCanRepair = true
        state.daemonNeedsRepair = true
        state.daemonControlOperationAction = "repair"
        state.daemonControlOperationState = "in_progress"

        state.markDaemonMissing()

        XCTAssertEqual(state.daemonStatus, .missing)
        XCTAssertEqual(state.daemonNeedsInstall, true)
        XCTAssertEqual(state.daemonNeedsRepair, false)
        XCTAssertEqual(state.daemonCanInstall, false)
        XCTAssertEqual(state.daemonCanUninstall, false)
        XCTAssertEqual(state.daemonCanRepair, false)
        XCTAssertEqual(state.daemonControlOperationState, "idle")
    }

    func testSelectingConfigurationDoesNotResetLoadedLifecycleState() {
        let state = AppShellState()
        state.hasLoadedDaemonStatus = true
        state.selectedSection = .configuration

        XCTAssertTrue(state.hasLoadedDaemonStatus)
    }

    func testWorkerFailureRepairStateDoesNotBlockOnboardingWithGenericRepairMessage() {
        let state = AppShellState()
        state.localDevTokenConfigured = true
        state.daemonNeedsRepair = true
        state.daemonDatabaseReady = true
        state.daemonSetupState = "repair_required"
        state.daemonWorkerSummary = DaemonLifecycleWorkerSummary(failed: 1)

        XCTAssertTrue(state.daemonHasWorkerFailureRepairState)
        XCTAssertFalse(state.daemonNeedsInfrastructureRepair)
        XCTAssertFalse(state.onboardingStatusMessage.contains("needs repair"))
    }

    func testStructuredLifecycleClassificationDrivesWorkerDegradationFlags() {
        let state = AppShellState()
        state.daemonNeedsRepair = true
        state.daemonLifecycleOverallState = "degraded"
        state.daemonCoreRuntimeState = "ready"
        state.daemonPluginRuntimeState = "degraded"
        state.daemonLifecycleBlocking = false

        XCTAssertTrue(state.daemonHasWorkerFailureRepairState)
        XCTAssertFalse(state.daemonNeedsInfrastructureRepair)
    }

    func testRefreshDaemonPluginLifecycleHistoryWithoutTokenSetsDeterministicStatus() async {
        let defaults = appShellStateTestUserDefaults()
        let priorToken = defaults.object(forKey: tokenDefaultsKey)
        let priorOnboarding = defaults.object(forKey: onboardingDefaultsKey)
        defer {
            if let priorToken {
                defaults.set(priorToken, forKey: tokenDefaultsKey)
            } else {
                defaults.removeObject(forKey: tokenDefaultsKey)
            }
            if let priorOnboarding {
                defaults.set(priorOnboarding, forKey: onboardingDefaultsKey)
            } else {
                defaults.removeObject(forKey: onboardingDefaultsKey)
            }
        }

        defaults.removeObject(forKey: tokenDefaultsKey)
        defaults.removeObject(forKey: onboardingDefaultsKey)

        let state = AppShellState()
        state.clearLocalDevToken()
        state.refreshDaemonPluginLifecycleHistory()

        try? await Task.sleep(for: .milliseconds(50))

        XCTAssertEqual(state.daemonPluginLifecycleHistoryItems.count, 0)
        XCTAssertEqual(
            state.daemonPluginLifecycleHistoryStatusMessage,
            "Set Assistant Access Token to query runtime plugin lifecycle history."
        )
        XCTAssertEqual(state.isDaemonPluginLifecycleHistoryLoading, false)
    }

    func testRefreshDaemonStatusWithoutTokenMarksControlAuthMissing() async {
        let defaults = appShellStateTestUserDefaults()
        let priorToken = defaults.object(forKey: tokenDefaultsKey)
        let priorOnboarding = defaults.object(forKey: onboardingDefaultsKey)
        defer {
            if let priorToken {
                defaults.set(priorToken, forKey: tokenDefaultsKey)
            } else {
                defaults.removeObject(forKey: tokenDefaultsKey)
            }
            if let priorOnboarding {
                defaults.set(priorOnboarding, forKey: onboardingDefaultsKey)
            } else {
                defaults.removeObject(forKey: onboardingDefaultsKey)
            }
        }

        defaults.removeObject(forKey: tokenDefaultsKey)
        defaults.removeObject(forKey: onboardingDefaultsKey)

        let state = AppShellState()
        state.clearLocalDevToken()
        state.refreshDaemonStatus()

        try? await Task.sleep(for: .milliseconds(50))

        XCTAssertEqual(state.daemonControlAuthState, .missing)
        XCTAssertEqual(state.daemonControlAuthSource, "local_token_missing")
        XCTAssertEqual(state.effectiveDaemonControlAuthState, .missing)
        XCTAssertTrue(state.daemonControlAuthNeedsRemediation)
    }

    func testOpenInspectForRuntimePluginLifecycleSeedsSearchAndNavigates() {
        let state = AppShellState()
        let item = makeRuntimePluginLifecycleItem(kind: "channel")

        state.openInspectForRuntimePluginLifecycle(item)

        XCTAssertEqual(state.selectedSection, .inspect)
        XCTAssertEqual(state.inspectSearchSeed, "messages.daemon")
        XCTAssertEqual(state.inspectStatusMessage, "Opened Inspect for runtime plugin messages.daemon.")
    }

    func testOpenRuntimeDiagnosticsForPluginLifecycleNavigatesByKind() {
        let state = AppShellState()
        let channelItem = makeRuntimePluginLifecycleItem(kind: "channel")

        state.openRuntimeDiagnosticsForPluginLifecycle(channelItem)

        XCTAssertEqual(state.selectedSection, .channels)
        XCTAssertEqual(
            state.channelsStatusMessage,
            "Opened Channels for runtime plugin messages.daemon."
        )
    }

    private func makeRuntimePluginLifecycleItem(kind: String) -> RuntimePluginLifecycleEventItem {
        RuntimePluginLifecycleEventItem(
            id: "audit-1",
            workspaceID: "daemon",
            pluginID: "messages.daemon",
            kind: kind,
            state: "running",
            eventType: "PLUGIN_HANDSHAKE_ACCEPTED",
            processID: 1001,
            restartCount: 1,
            reason: "worker_recovered",
            error: nil,
            restartEvent: false,
            failureEvent: false,
            recoveryEvent: true,
            lastHeartbeatAtLabel: nil,
            lastTransitionAtLabel: nil,
            occurredAtLabel: "now",
            sortTimestamp: Date.now
        )
    }

    private func decodeDaemonLifecycleControlResponse(
        operationState: String
    ) throws -> DaemonLifecycleControlResponse {
        let rawJSON = """
        {
          "action": "stop",
          "accepted": true,
          "idempotent": false,
          "lifecycle_state": "running",
          "message": "Stop requested.",
          "operation_state": "\(operationState)"
        }
        """
        return try JSONDecoder().decode(
            DaemonLifecycleControlResponse.self,
            from: Data(rawJSON.utf8)
        )
    }
}
