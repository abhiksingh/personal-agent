import XCTest
@testable import PersonalAgentUI

@MainActor
final class AppShellStateInformationDensityTests: XCTestCase {
    private let informationDensityDefaultsKey = "personalagent.ui.information_density_mode.v1"

    private func withIsolatedInformationDensityDefaults(_ body: () -> Void) {
        let defaults = appShellStateTestUserDefaults()
        let priorValue = defaults.object(forKey: informationDensityDefaultsKey)
        defer {
            if let priorValue {
                defaults.set(priorValue, forKey: informationDensityDefaultsKey)
            } else {
                defaults.removeObject(forKey: informationDensityDefaultsKey)
            }
        }
        defaults.removeObject(forKey: informationDensityDefaultsKey)
        body()
    }

    func testInformationDensityModeDefaultsToSimple() {
        withIsolatedInformationDensityDefaults {
            let state = AppShellState()
            XCTAssertEqual(state.informationDensityMode, .simple)
            XCTAssertFalse(state.isAdvancedInformationDensityEnabled)
        }
    }

    func testInformationDensityModePersistsPerWorkspace() {
        withIsolatedInformationDensityDefaults {
            let state = AppShellState()

            state._test_applyWorkspaceContextExplicitSelection("ws-density-a")
            XCTAssertEqual(state.informationDensityMode, .simple)

            state.setInformationDensityMode(.advanced)
            XCTAssertEqual(state.informationDensityMode, .advanced)

            state._test_applyWorkspaceContextExplicitSelection("ws-density-b")
            XCTAssertEqual(state.informationDensityMode, .simple)

            state.setInformationDensityMode(.simple)
            XCTAssertEqual(state.informationDensityMode, .simple)

            state._test_applyWorkspaceContextExplicitSelection("ws-density-a")
            XCTAssertEqual(state.informationDensityMode, .advanced)
        }
    }

    func testInformationDensityModeReloadsFromDefaults() {
        withIsolatedInformationDensityDefaults {
            let workspaceID = "ws-density-reload-\(UUID().uuidString.lowercased())"

            let first = AppShellState()
            first._test_applyWorkspaceContextExplicitSelection(workspaceID)
            first.setInformationDensityMode(.advanced)

            let second = AppShellState()
            second._test_applyWorkspaceContextExplicitSelection(workspaceID)

            XCTAssertEqual(second.informationDensityMode, .advanced)
            XCTAssertTrue(second.isAdvancedInformationDensityEnabled)

            second._test_applyWorkspaceContextExplicitSelection("ws-density-fallback")
            XCTAssertEqual(second.informationDensityMode, .simple)
        }
    }

    func testLegacyDefaultInformationDensityKeyIsIgnoredWithoutMigration() {
        withIsolatedInformationDensityDefaults {
            let defaults = appShellStateTestUserDefaults()
            defaults.set(
                try? JSONEncoder().encode(["default": AppInformationDensityMode.advanced.rawValue]),
                forKey: informationDensityDefaultsKey
            )

            let state = AppShellState()
            XCTAssertEqual(state.workspaceID, "ws1")
            XCTAssertEqual(state.informationDensityMode, .simple)

            guard
                let data = defaults.data(forKey: informationDensityDefaultsKey),
                let persisted = try? JSONDecoder().decode([String: String].self, from: data)
            else {
                XCTFail("Expected persisted information-density defaults payload.")
                return
            }

            XCTAssertEqual(persisted["default"], AppInformationDensityMode.advanced.rawValue)
            XCTAssertNil(persisted["ws1"])

            let restored = AppShellState()
            XCTAssertEqual(restored.informationDensityMode, .simple)
        }
    }
}
