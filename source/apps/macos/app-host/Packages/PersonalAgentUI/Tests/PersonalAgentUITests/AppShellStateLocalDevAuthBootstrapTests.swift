import Foundation
import XCTest
@testable import PersonalAgentUI

@MainActor
final class AppShellStateLocalDevAuthBootstrapTests: XCTestCase {
    private let tokenDefaultsKey = "personalagent.ui.local_dev_token"
    private let onboardingDefaultsKey = "personalagent.ui.onboarding_complete"

    override func setUp() {
        super.setUp()
        AppShellState._test_setLocalDevTokenSecretReference(
            service: "personalagent.ui.tests.local-dev-bootstrap.\(UUID().uuidString)",
            account: "daemon_auth_token"
        )
        AppShellState._test_clearPersistedLocalDevToken()
    }

    override func tearDown() {
        AppShellState._test_clearPersistedLocalDevToken()
        AppShellState._test_resetLocalDevAuthBootstrapHooks()
        super.tearDown()
    }

    func testLocalDevAuthBootstrapCommandUsesCurrentWorkspaceAndDaemonEndpoint() {
        AppShellState._test_resetLocalDevAuthBootstrapHooks()

        let state = AppShellState()

        let command = state.localDevAuthBootstrapCommand

        XCTAssertTrue(command.contains("auth bootstrap-local-dev"))
        XCTAssertTrue(command.contains("--workspace ws1"))
        XCTAssertTrue(command.contains("--mode tcp"))
        XCTAssertTrue(command.contains("--address 127.0.0.1:7071"))
    }

    func testPerformLocalDevAuthBootstrapLoadsTokenAndRefreshesState() async throws {
        AppShellState._test_resetLocalDevAuthBootstrapHooks()
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
            AppShellState._test_resetLocalDevAuthBootstrapHooks()
        }

        defaults.removeObject(forKey: tokenDefaultsKey)
        defaults.removeObject(forKey: onboardingDefaultsKey)

        let tokenFile = URL(fileURLWithPath: NSTemporaryDirectory())
            .appendingPathComponent("pa-ui-bootstrap-token-\(UUID().uuidString).txt")
        try "bootstrap-token\n".write(to: tokenFile, atomically: true, encoding: .utf8)

        AppShellState._test_setLocalDevAuthBootstrapCommandRunner { _ in
            (
                exitCode: 0,
                stdout: self.bootstrapCommandJSONOutput(
                    tokenFilePath: tokenFile.path,
                    tokenCreated: true,
                    tokenRotated: false
                ),
                stderr: ""
            )
        }
        AppShellState._test_setLocalDevAuthBootstrapRefreshHandler { _ in }

        let state = AppShellState()
        state.clearLocalDevToken()

        await state.performLocalDevAuthBootstrap()

        XCTAssertTrue(state.localDevTokenConfigured)
        XCTAssertNotEqual(state.localDevTokenLastUpdated, "Not configured")
        XCTAssertEqual(AppShellState._test_readPersistedLocalDevToken(), "bootstrap-token")
        XCTAssertNil(defaults.string(forKey: tokenDefaultsKey))
        XCTAssertEqual(state.localDevAuthBootstrapStatusMessage, "Bootstrap completed and readiness checks refreshed.")
        XCTAssertFalse(state.isLocalDevAuthBootstrapInFlight)
    }

    func testPerformLocalDevAuthBootstrapMissingCLIHasDeterministicError() async {
        AppShellState._test_resetLocalDevAuthBootstrapHooks()
        defer { AppShellState._test_resetLocalDevAuthBootstrapHooks() }

        AppShellState._test_setLocalDevAuthBootstrapCommandRunner { _ in
            (
                exitCode: 127,
                stdout: "",
                stderr: "/usr/bin/env: personal-agent: No such file or directory"
            )
        }
        AppShellState._test_setLocalDevAuthBootstrapRefreshHandler { _ in }

        let state = AppShellState()
        state.clearLocalDevToken()

        await state.performLocalDevAuthBootstrap()

        XCTAssertFalse(state.localDevTokenConfigured)
        XCTAssertEqual(
            state.localDevAuthBootstrapStatusMessage,
            "Bootstrap command failed because `personal-agent` is not available on PATH. Build or install CLI, then retry."
        )
        XCTAssertFalse(state.isLocalDevAuthBootstrapInFlight)
    }

    private func bootstrapCommandJSONOutput(
        tokenFilePath: String,
        tokenCreated: Bool,
        tokenRotated: Bool
    ) -> String {
        let payload: [String: Any] = [
            "operation": "bootstrap_local_dev",
            "token_file": tokenFilePath,
            "token_created": tokenCreated,
            "token_rotated": tokenRotated,
            "active_profile": "local-daemon",
            "next_step_reminder": "Start/restart daemon with --auth-token-file and use configured profile"
        ]
        let data = try! JSONSerialization.data(withJSONObject: payload, options: [.sortedKeys])
        return String(data: data, encoding: .utf8) ?? ""
    }
}
