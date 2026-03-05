import XCTest

@MainActor
final class PersonalAgentHostVisualRegressionTests: XCTestCase {
    override func setUpWithError() throws {
        continueAfterFailure = false
    }

    func testReadyFixtureSectionBaselines() throws {
        let app = launchApp(scenario: .ready)
        let mainWindow = primaryWindow(app: app)
        XCTAssertTrue(mainWindow.waitForExistence(timeout: 8), "Main window should be visible for visual regression snapshots.")

        let snapshotAssert = VisualSnapshotAssert()

        waitForPanel(identifier: "panel-home", app: app)

        captureSnapshot(
            sectionID: "chat",
            title: "Chat",
            panelIdentifier: "panel-chat",
            snapshotName: "window-chat-shell",
            app: app,
            window: mainWindow,
            snapshotAssert: snapshotAssert
        )

        captureSnapshot(
            sectionID: "configuration",
            title: "Configuration",
            panelIdentifier: "panel-configuration",
            snapshotName: "window-configuration",
            app: app,
            window: mainWindow,
            snapshotAssert: snapshotAssert
        )
        captureSnapshot(
            sectionID: "communications",
            title: "Communications",
            panelIdentifier: "panel-communications",
            snapshotName: "window-communications",
            app: app,
            window: mainWindow,
            snapshotAssert: snapshotAssert
        )
        captureSnapshot(
            sectionID: "automation",
            title: "Automation",
            panelIdentifier: "panel-automation",
            snapshotName: "window-automation",
            app: app,
            window: mainWindow,
            snapshotAssert: snapshotAssert
        )
        captureSnapshot(
            sectionID: "tasks",
            title: "Tasks",
            panelIdentifier: "panel-tasks",
            snapshotName: "window-tasks",
            app: app,
            window: mainWindow,
            snapshotAssert: snapshotAssert
        )
        captureSnapshot(
            sectionID: "approvals",
            title: "Approvals",
            panelIdentifier: "panel-approvals",
            snapshotName: "window-approvals",
            app: app,
            window: mainWindow,
            snapshotAssert: snapshotAssert
        )
        captureSnapshot(
            sectionID: "inspect",
            title: "Inspect",
            panelIdentifier: "panel-inspect",
            snapshotName: "window-inspect",
            app: app,
            window: mainWindow,
            snapshotAssert: snapshotAssert
        )
        captureInspectGallerySnapshot(
            app: app,
            window: mainWindow,
            snapshotAssert: snapshotAssert
        )
        captureSnapshot(
            sectionID: "channels",
            title: "Channels",
            panelIdentifier: "panel-channels",
            snapshotName: "window-channels",
            app: app,
            window: mainWindow,
            snapshotAssert: snapshotAssert
        )
        captureSnapshot(
            sectionID: "connectors",
            title: "Connectors",
            panelIdentifier: "panel-connectors",
            snapshotName: "window-connectors",
            app: app,
            window: mainWindow,
            snapshotAssert: snapshotAssert
        )
        captureSnapshot(
            sectionID: "models",
            title: "Models",
            panelIdentifier: "panel-models",
            snapshotName: "window-models",
            app: app,
            window: mainWindow,
            snapshotAssert: snapshotAssert
        )
    }

    private func launchApp(scenario: FixtureScenario) -> XCUIApplication {
        let app = XCUIApplication()
        app.launchArguments += [
            "-ApplePersistenceIgnoreState",
            "YES",
        ]
        app.launchEnvironment["PA_UI_SMOKE_FIXTURE_SCENARIO"] = scenario.rawValue
        app.launchEnvironment["PA_UI_APP_HOST_VISUAL_MODE"] = "1"
        app.launchEnvironment["PERSONAL_AGENT_DAEMON_URL"] = "http://127.0.0.1:7071"
        app.launchEnvironment["PERSONAL_AGENT_DAEMON_TOKEN"] = "fixture-token"
        app.launchEnvironment["PERSONAL_AGENT_WORKSPACE_ID"] = "ws1"
        app.launchEnvironment["PA_UI_DEFAULTS_SUITE"] = "com.personalagent.app.visualtests.\(UUID().uuidString)"
        app.launch()
        return app
    }

    private func captureSnapshot(
        sectionID: String,
        title: String,
        panelIdentifier: String,
        snapshotName: String,
        app: XCUIApplication,
        window: XCUIElement,
        snapshotAssert: VisualSnapshotAssert
    ) {
        openSidebarSection(id: sectionID, title: title, app: app)
        waitForPanel(identifier: panelIdentifier, app: app)
        settleUI()
        snapshotAssert.assertSnapshot(of: window, named: snapshotName)
    }

    private func captureInspectGallerySnapshot(
        app: XCUIApplication,
        window: XCUIElement,
        snapshotAssert: VisualSnapshotAssert
    ) {
        openSidebarSection(id: "inspect", title: "Inspect", app: app)
        waitForPanel(identifier: "panel-inspect", app: app)
        let galleryToggle = app.descendants(matching: .any)
            .matching(identifier: "inspect-mode-gallery-button")
            .firstMatch
        XCTAssertTrue(
            galleryToggle.waitForExistence(timeout: 8),
            "Inspect gallery action should be available for visual snapshot capture."
        )
        galleryToggle.click()
        let gallery = app.descendants(matching: .any).matching(identifier: "inspect-component-gallery").firstMatch
        XCTAssertTrue(gallery.waitForExistence(timeout: 8), "Inspect gallery should be visible before snapshot capture.")
        settleUI()
        snapshotAssert.assertSnapshot(of: window, named: "window-inspect-gallery")
    }

    private func waitForPanel(identifier: String, app: XCUIApplication) {
        let panel = app.descendants(matching: .any).matching(identifier: identifier).firstMatch
        XCTAssertTrue(panel.waitForExistence(timeout: 8), "Panel \(identifier) should exist.")
    }

    private func primaryWindow(app: XCUIApplication) -> XCUIElement {
        let identifiedWindow = app.windows.matching(identifier: "main").firstMatch
        if identifiedWindow.exists || identifiedWindow.waitForExistence(timeout: 2) {
            return identifiedWindow
        }
        return app.windows.firstMatch
    }

    private func settleUI() {
        RunLoop.main.run(until: Date(timeIntervalSinceNow: 0.35))
    }

    private func openSidebarSection(id: String, title: String, app: XCUIApplication) {
        let identifier = "sidebar-section-\(id)"
        let identifiedRow = app.descendants(matching: .any).matching(identifier: identifier).firstMatch
        if identifiedRow.exists || identifiedRow.waitForExistence(timeout: 1) {
            identifiedRow.click()
            return
        }

        if triggerSectionKeyboardShortcut(id: id, app: app) {
            return
        }

        let advancedDisclosure = app.descendants(matching: .any)
            .matching(identifier: "sidebar-advanced-disclosure")
            .firstMatch
        if advancedDisclosure.exists || advancedDisclosure.waitForExistence(timeout: 2) {
            let initiallyVisibleRow = app.descendants(matching: .any).matching(identifier: identifier).firstMatch
            if initiallyVisibleRow.exists || initiallyVisibleRow.waitForExistence(timeout: 1) {
                initiallyVisibleRow.click()
                return
            }
            advancedDisclosure.click()
            let firstToggleRow = app.descendants(matching: .any).matching(identifier: identifier).firstMatch
            if firstToggleRow.waitForExistence(timeout: 8) {
                firstToggleRow.click()
                return
            }
            advancedDisclosure.click()
            let secondToggleRow = app.descendants(matching: .any).matching(identifier: identifier).firstMatch
            if secondToggleRow.waitForExistence(timeout: 8) {
                secondToggleRow.click()
                return
            }
        }

        let fallbackByButton = app.buttons[title]
        if fallbackByButton.waitForExistence(timeout: 8) {
            fallbackByButton.click()
            return
        }

        let fallbackByStaticText = app.staticTexts[title].firstMatch
        XCTAssertTrue(fallbackByStaticText.waitForExistence(timeout: 8), "Sidebar section \(title) should exist.")
        fallbackByStaticText.click()
    }

    @discardableResult
    private func triggerSectionKeyboardShortcut(id: String, app: XCUIApplication) -> Bool {
        guard let shortcutKey = sectionShortcutKey(for: id) else {
            return false
        }
        app.typeKey(shortcutKey, modifierFlags: [.command])
        return true
    }

    private func sectionShortcutKey(for id: String) -> String? {
        switch id {
        case "configuration":
            return "1"
        case "home":
            return "2"
        case "chat":
            return "3"
        case "communications":
            return "4"
        case "automation":
            return "5"
        case "approvals":
            return "6"
        case "inspect":
            return "7"
        case "channels":
            return "8"
        case "connectors":
            return "9"
        case "models":
            return "0"
        default:
            return nil
        }
    }
}

private enum FixtureScenario: String {
    case ready
}
