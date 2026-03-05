import XCTest
@testable import PersonalAgentUI

@MainActor
final class AppShellStateConnectorPermissionTests: XCTestCase {
    func testResolveConnectorPermissionStateUsesDaemonPermissionMissingReason() {
        let state = AppShellState()

        let resolved = state.resolveConnectorPermissionState(
            daemonPermissionStateRaw: "granted",
            statusReason: "permission_missing",
            connectorStatus: "degraded",
            fallback: .unknown
        )

        XCTAssertEqual(resolved, .missing)
    }

    func testResolveConnectorPermissionStateDoesNotInferGrantedFromReadyStatusAlone() {
        let state = AppShellState()

        let resolved = state.resolveConnectorPermissionState(
            daemonPermissionStateRaw: nil,
            statusReason: "ready",
            connectorStatus: "ready",
            fallback: .unknown
        )

        XCTAssertEqual(resolved, .unknown)
    }

    func testResolveConnectorPermissionStateFallsBackWhenDaemonStateUnknownAndNotReady() {
        let state = AppShellState()

        let resolved = state.resolveConnectorPermissionState(
            daemonPermissionStateRaw: nil,
            statusReason: "runtime_failure",
            connectorStatus: "degraded",
            fallback: .missing
        )

        XCTAssertEqual(resolved, .missing)
    }

    func testResolveConnectorPermissionStateDaemonGrantedOverridesFallback() {
        let state = AppShellState()

        let resolved = state.resolveConnectorPermissionState(
            daemonPermissionStateRaw: "granted",
            statusReason: "runtime_failure",
            connectorStatus: "degraded",
            fallback: .missing
        )

        XCTAssertEqual(resolved, .granted)
    }

    func testResolveConnectorPermissionStateDaemonMissingOverridesFallback() {
        let state = AppShellState()

        let resolved = state.resolveConnectorPermissionState(
            daemonPermissionStateRaw: "missing",
            statusReason: "ready",
            connectorStatus: "ready",
            fallback: .granted
        )

        XCTAssertEqual(resolved, .missing)
    }
}
