import XCTest
@testable import PersonalAgentUI

@MainActor
final class AppRuntimeLifecycleStoreTests: XCTestCase {
    func testApplyMissingTokenStateSetsDeterministicDefaults() {
        let store = AppRuntimeLifecycleStore()

        store.applyMissingTokenState()

        XCTAssertEqual(store.daemonStatus, .unknown)
        XCTAssertEqual(store.connectionStatus, .disconnected)
        XCTAssertEqual(store.daemonControlAuthState, .missing)
        XCTAssertEqual(store.daemonControlAuthSource, "local_token_missing")
        XCTAssertEqual(
            store.daemonStatusDetail,
            "Set Assistant Access Token to query daemon lifecycle."
        )
        XCTAssertFalse(store.daemonCanStart)
        XCTAssertFalse(store.daemonCanStop)
        XCTAssertFalse(store.daemonCanRepair)
    }

    func testApplyDaemonLifecycleStatusProjectsStructuredRuntimeAndReadinessState() throws {
        let store = AppRuntimeLifecycleStore()
        let lifecycle = try decodeLifecycleStatus(
            """
            {
              "lifecycle_state": "running",
              "runtime_mode": "tcp",
              "bound_address": "127.0.0.1:7071",
              "setup_state": "repair_required",
              "install_state": "installed",
              "needs_install": false,
              "needs_repair": true,
              "database_ready": true,
              "repair_hint": "run repair if plugins fail",
              "health_classification": {
                "overall_state": "degraded",
                "core_runtime_state": "ready",
                "plugin_runtime_state": "degraded",
                "blocking": false
              },
              "control_auth": {
                "state": "configured",
                "source": "auth_token_flag",
                "remediation_hints": []
              },
              "worker_summary": {
                "failed": 1
              },
              "controls": {
                "start": false,
                "stop": true,
                "restart": true,
                "install": false,
                "uninstall": false,
                "repair": true
              },
              "control_operation": {
                "action": "",
                "state": "idle"
              }
            }
            """
        )

        store.applyDaemonLifecycleStatus(lifecycle)

        XCTAssertEqual(store.daemonStatus, .running)
        XCTAssertEqual(store.connectionStatus, .connected)
        XCTAssertEqual(store.daemonControlAuthState, .configured)
        XCTAssertTrue(store.daemonHasWorkerFailureRepairState)
        XCTAssertFalse(store.daemonNeedsInfrastructureRepair)
        XCTAssertTrue(store.daemonCanRepair)
        XCTAssertTrue(store.daemonStatusDetail.contains("health=degraded"))
        XCTAssertTrue(store.daemonStatusDetail.contains("workers_failed=1"))
    }

    func testDaemonControlAuthSetupDetailUsesLoadedMissingHint() {
        let store = AppRuntimeLifecycleStore()
        store.hasLoadedDaemonStatus = true
        store.daemonControlAuthState = .missing
        store.daemonControlAuthRemediationHints = ["Save a token first."]

        XCTAssertTrue(store.daemonControlAuthNeedsRemediation(localDevTokenConfigured: true))
        XCTAssertEqual(
            store.daemonControlAuthSetupDetail(localDevTokenConfigured: true),
            "Save a token first."
        )
    }

    private func decodeLifecycleStatus(_ rawJSON: String) throws -> DaemonLifecycleStatusResponse {
        let data = Data(rawJSON.utf8)
        return try JSONDecoder().decode(DaemonLifecycleStatusResponse.self, from: data)
    }
}
