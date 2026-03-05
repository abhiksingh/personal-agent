import XCTest

@MainActor
final class PersonalAgentHostUITests: XCTestCase {
    override func setUpWithError() throws {
        continueAfterFailure = false
    }

    func testReadyFixtureSmokeJourneys() throws {
        let app = launchApp(scenario: .ready)

        XCTAssertTrue(
            panel(identifier: "panel-home", app: app).waitForExistence(timeout: 8),
            "Home panel should be visible on launch in ready fixture scenario."
        )
        openSidebarSection(id: "chat", title: "Chat", app: app)

        let chatSendButton = elementWithIdentifier(
            "chat-send-button",
            fallback: app.buttons["Send message"].firstMatch,
            app: app
        )
        XCTAssertTrue(chatSendButton.waitForExistence(timeout: 8))
        XCTAssertFalse(chatSendButton.isEnabled, "Send should be disabled before draft input.")

        let composerInput = elementWithIdentifier(
            "chat-composer-native-input",
            fallback: app.textViews.firstMatch,
            app: app
        )
        XCTAssertTrue(composerInput.waitForExistence(timeout: 8))
        composerInput.click()
        composerInput.typeText("Smoke fixture chat prompt")
        waitForElementEnabled(chatSendButton)
        chatSendButton.click()
        XCTAssertTrue(
            app.staticTexts["Fixture response: smoke chat completed."].waitForExistence(timeout: 8),
            "Fixture assistant response should render after send."
        )

        openSidebarSection(id: "tasks", title: "Tasks", app: app)
        XCTAssertTrue(app.staticTexts["Tasks"].waitForExistence(timeout: 8))
        let viewRunDetailButton = app.buttons.matching(NSPredicate(format: "label == %@", "View Run Detail")).firstMatch
        XCTAssertTrue(viewRunDetailButton.waitForExistence(timeout: 8))
        viewRunDetailButton.click()
        let closeRunDetailButton = app.buttons.matching(NSPredicate(format: "label == %@", "Close")).firstMatch
        XCTAssertTrue(closeRunDetailButton.waitForExistence(timeout: 8))
        closeRunDetailButton.click()

        openSidebarSection(id: "approvals", title: "Approvals", app: app)
        XCTAssertTrue(app.staticTexts["Approvals"].waitForExistence(timeout: 8))
        let openTaskDetailButton = app.buttons.matching(NSPredicate(format: "label == %@", "Open Task Detail")).firstMatch
        XCTAssertTrue(openTaskDetailButton.waitForExistence(timeout: 8))
        openTaskDetailButton.click()
        XCTAssertTrue(closeRunDetailButton.waitForExistence(timeout: 8))
        closeRunDetailButton.click()

        openSidebarSection(id: "channels", title: "Channels", app: app)
        let channelsPanel = panel(identifier: "panel-channels", app: app)
        XCTAssertTrue(channelsPanel.waitForExistence(timeout: 8))
        expandFirstDisclosureCard(in: channelsPanel, panelName: "Channels")

        openSidebarSection(id: "connectors", title: "Connectors", app: app)
        let connectorsPanel = panel(identifier: "panel-connectors", app: app)
        XCTAssertTrue(connectorsPanel.waitForExistence(timeout: 8))
        expandFirstDisclosureCard(in: connectorsPanel, panelName: "Connectors")
    }

    func testOnboardingFixtureShowsGateAndKeepsSetupSectionsAccessible() throws {
        let app = launchApp(scenario: .onboarding)

        XCTAssertTrue(
            app.staticTexts["Finish Setup"].waitForExistence(timeout: 8),
            "Onboarding fixture should gate workflow sections on launch."
        )
        openSidebarSection(id: "models", title: "Models", app: app)

        XCTAssertTrue(
            app.descendants(matching: .any).matching(identifier: "panel-models").firstMatch.waitForExistence(timeout: 8),
            "Models should remain accessible while onboarding is incomplete."
        )
        XCTAssertTrue(
            app.descendants(matching: .any).matching(identifier: "setup-blocker-ribbon").firstMatch.waitForExistence(timeout: 8),
            "Models panel should show current setup blocker ribbon when onboarding is incomplete."
        )

        openSidebarSection(id: "chat", title: "Chat", app: app)
        XCTAssertTrue(
            app.staticTexts["Finish Setup"].waitForExistence(timeout: 8),
            "Returning to Chat should show onboarding gate until setup blockers are resolved."
        )
    }

    func testFirstTenMinuteJourneyCoversSetupSendTaskSubmitApprovalsAndRecovery() throws {
        let onboardingApp = launchApp(scenario: .onboarding)
        XCTAssertTrue(
            onboardingApp.staticTexts["Finish Setup"].waitForExistence(timeout: 8),
            "First-run onboarding gate should appear before setup is complete."
        )
        openSidebarSection(id: "models", title: "Models", app: onboardingApp)
        XCTAssertTrue(
            onboardingApp.descendants(matching: .any).matching(identifier: "setup-blocker-ribbon").firstMatch.waitForExistence(timeout: 8),
            "Setup-accessible panels should surface a deterministic recovery/blocker ribbon."
        )
        XCTAssertTrue(
            onboardingApp.buttons["Fix Next"].firstMatch.waitForExistence(timeout: 8),
            "First-run recovery path should expose a primary Fix Next action."
        )
        onboardingApp.terminate()

        let app = launchApp(scenario: .ready)

        XCTAssertTrue(
            panel(identifier: "panel-home", app: app).waitForExistence(timeout: 8),
            "Home panel should be visible after setup-ready launch."
        )
        openSidebarSection(id: "chat", title: "Chat", app: app)

        let composerInput = elementWithIdentifier(
            "chat-composer-native-input",
            fallback: app.textViews.firstMatch,
            app: app
        )
        XCTAssertTrue(composerInput.waitForExistence(timeout: 8))
        composerInput.click()
        composerInput.typeText("First ten minute validation chat prompt")

        let chatSendButton = elementWithIdentifier(
            "chat-send-button",
            fallback: app.buttons["Send message"].firstMatch,
            app: app
        )
        waitForElementEnabled(chatSendButton)
        chatSendButton.click()
        XCTAssertTrue(
            app.staticTexts["Fixture response: smoke chat completed."].waitForExistence(timeout: 8),
            "First-send chat workflow should succeed in the ready fixture."
        )

        openSidebarSection(id: "tasks", title: "Tasks", app: app)
        XCTAssertTrue(app.staticTexts["Tasks"].waitForExistence(timeout: 8))
        let viewRunDetailButton = app.buttons.matching(NSPredicate(format: "label == %@", "View Run Detail")).firstMatch
        XCTAssertTrue(viewRunDetailButton.waitForExistence(timeout: 8))
        viewRunDetailButton.click()
        let closeRunDetailButton = app.buttons.matching(NSPredicate(format: "label == %@", "Close")).firstMatch
        XCTAssertTrue(closeRunDetailButton.waitForExistence(timeout: 8))
        closeRunDetailButton.click()

        openSidebarSection(id: "approvals", title: "Approvals", app: app)
        XCTAssertTrue(
            app.buttons["Use Required Phrase"].firstMatch.waitForExistence(timeout: 8),
            "Pending approval should expose guided phrase helper before submit."
        )
        app.buttons["Use Required Phrase"].firstMatch.click()

        let approveButton = app.buttons["Approve and Continue"].firstMatch
        waitForElementEnabled(approveButton)
        approveButton.click()

        let decisionSubmittedStatus = staticTextContaining("Decision submitted", app: app)
        let approvalSavedBadge = app.staticTexts["Saved"].firstMatch
        XCTAssertTrue(
            decisionSubmittedStatus.waitForExistence(timeout: 8)
                || approvalSavedBadge.waitForExistence(timeout: 4),
            "Approval submit should confirm a successful decision outcome."
        )
    }

    func testAutonomousChatEmailPromptShowsApprovalOutcome() throws {
        let app = launchReadyChatApp()
        assertAutonomousChatOutcome(
            prompt: "Send an email update to finance",
            expectation: AutonomousChatExpectation(
                workflow: "send_email",
                expectedResultBadge: "Blocked",
                expectedAssistantText: "send_email is waiting for approval (request approval-fixture-email).",
                expectedApprovalControlLabel: "Approve and Continue"
            ),
            app: app
        )

        openSidebarSection(id: "approvals", title: "Approvals", app: app)
        XCTAssertTrue(
            app.staticTexts["Approvals"].waitForExistence(timeout: 8),
            "Approval inbox should remain reachable after approval-required autonomous chat outcome."
        )
    }

    func testAutonomousChatTextPromptShowsCompletedOutcome() throws {
        let app = launchReadyChatApp()
        assertAutonomousChatOutcome(
            prompt: "Text the team that launch is complete",
            expectation: AutonomousChatExpectation(
                workflow: "send_message",
                expectedResultBadge: "Complete",
                expectedAssistantText: "send_message completed successfully."
            ),
            app: app
        )
    }

    func testAutonomousChatFindFilesPromptShowsBlockedOutcome() throws {
        let app = launchReadyChatApp()
        assertAutonomousChatOutcome(
            prompt: "Find files for Q1 report",
            expectation: AutonomousChatExpectation(
                workflow: "find_files",
                expectedResultBadge: "Blocked",
                expectedAssistantText: "find_files is blocked: connector permission is missing."
            ),
            app: app
        )
    }

    func testAutonomousChatBrowsePromptShowsCompletedOutcome() throws {
        let app = launchReadyChatApp()
        assertAutonomousChatOutcome(
            prompt: "Browse website for competitive updates",
            expectation: AutonomousChatExpectation(
                workflow: "browse_web",
                expectedResultBadge: "Complete",
                expectedAssistantText: "browse_web completed successfully."
            ),
            app: app
        )
    }

    func testCommandPaletteNaturalLanguageEnterExecutesRankedDestination() throws {
        let app = launchApp(scenario: .ready)

        let commandButton = app.buttons["Commands"].firstMatch
        XCTAssertTrue(commandButton.waitForExistence(timeout: 8))
        commandButton.click()

        XCTAssertTrue(app.staticTexts["Command Palette"].waitForExistence(timeout: 8))
        let queryField = commandPaletteSearchField(app: app)
        XCTAssertTrue(queryField.waitForExistence(timeout: 8), "Command palette search field should be present.")
        queryField.click()
        queryField.typeText("settings setup")
        queryField.typeKey(XCUIKeyboardKey.return, modifierFlags: [])

        XCTAssertTrue(
            app.descendants(matching: .any).matching(identifier: "panel-configuration").firstMatch.waitForExistence(timeout: 8),
            "Pressing Enter should execute the ranked first enabled match and navigate to Configuration."
        )
    }

    func testMissingAuthRecoveryJourneyRoutesToConfigurationFromChat() throws {
        let app = launchApp(scenario: .ready, authToken: nil)
        openSidebarSection(id: "chat", title: "Chat", app: app)

        XCTAssertTrue(
            app.staticTexts["Finish Setup"].waitForExistence(timeout: 8),
            "Missing-auth recovery should route workflow panels through setup gate."
        )

        let openConfigurationButton = app.buttons["Open Configuration"].firstMatch
        XCTAssertTrue(
            openConfigurationButton.waitForExistence(timeout: 8),
            "Missing-auth recovery should expose an in-flow Open Configuration action."
        )
        openConfigurationButton.click()

        XCTAssertTrue(
            panel(identifier: "panel-configuration", app: app).waitForExistence(timeout: 8),
            "Open Configuration remediation action should navigate to Configuration."
        )
        XCTAssertTrue(
            app.staticTexts["Assistant Access Token"].firstMatch.waitForExistence(timeout: 8),
            "Configuration should expose Assistant Access Token setup after missing-auth remediation."
        )
    }

    func testRouteMissingRecoveryJourneySurfacesOnboardingModelsRemediation() throws {
        let app = launchApp(scenario: .onboarding)
        XCTAssertTrue(
            app.staticTexts["Finish Setup"].waitForExistence(timeout: 8),
            "Route-missing recovery journey should present setup gate."
        )
        XCTAssertTrue(
            app.staticTexts["Chat Route"].firstMatch.waitForExistence(timeout: 8),
            "Onboarding route-missing recovery should keep chat-route blocker visible."
        )

        let openModelsButton = app.buttons["Open Models"].firstMatch
        XCTAssertTrue(
            openModelsButton.waitForExistence(timeout: 8),
            "Route-missing setup gate should expose Open Models remediation."
        )
        openModelsButton.click()

        XCTAssertTrue(
            panel(identifier: "panel-models", app: app).waitForExistence(timeout: 8),
            "Open Models remediation should navigate to Models panel."
        )
        XCTAssertTrue(
            app.descendants(matching: .any).matching(identifier: "setup-blocker-ribbon").firstMatch.waitForExistence(timeout: 8),
            "Models panel should retain deterministic setup blocker ribbon after route-missing remediation navigation."
        )
    }

    func testDegradedRuntimeJourneyKeepsDiagnosticsNavigationAvailable() throws {
        let app = launchApp(scenario: .degraded)

        let commandButton = app.buttons["Commands"].firstMatch
        XCTAssertTrue(
            commandButton.waitForExistence(timeout: 8),
            "Degraded runtime journey should keep command palette access available."
        )
        commandButton.click()

        XCTAssertTrue(app.staticTexts["Command Palette"].waitForExistence(timeout: 8))
        let queryField = commandPaletteSearchField(app: app)
        XCTAssertTrue(queryField.waitForExistence(timeout: 8), "Command palette search should open in degraded runtime.")
        queryField.click()
        queryField.typeText("connectors")
        queryField.typeKey(XCUIKeyboardKey.return, modifierFlags: [])

        XCTAssertTrue(
            panel(identifier: "panel-connectors", app: app).waitForExistence(timeout: 8),
            "Degraded runtime journey should keep Connectors reachable through command palette routing."
        )
        XCTAssertTrue(
            app.staticTexts["App connector degraded in fixture."].firstMatch.waitForExistence(timeout: 8),
            "Degraded runtime journey should surface deterministic degraded connector evidence."
        )

        XCTAssertTrue(
            app.buttons["Refresh"].firstMatch.waitForExistence(timeout: 8),
            "Degraded runtime journey should preserve diagnostics remediation controls in Connectors."
        )
    }

    private func launchApp(
        scenario: FixtureScenario,
        authToken: String? = "fixture-token"
    ) -> XCUIApplication {
        let app = XCUIApplication()
        app.launchArguments += [
            "-ApplePersistenceIgnoreState",
            "YES",
        ]
        app.launchEnvironment["PA_UI_SMOKE_FIXTURE_SCENARIO"] = scenario.rawValue
        app.launchEnvironment["PERSONAL_AGENT_DAEMON_URL"] = "http://127.0.0.1:7071"
        if let authToken {
            app.launchEnvironment["PERSONAL_AGENT_DAEMON_TOKEN"] = authToken
        }
        app.launchEnvironment["PERSONAL_AGENT_WORKSPACE_ID"] = "ws1"
        app.launchEnvironment["PA_UI_DEFAULTS_SUITE"] = "com.personalagent.app.xcuitests.\(UUID().uuidString)"
        app.launch()
        return app
    }

    private func openSidebarSection(id: String, title: String, app: XCUIApplication) {
        let identifier = "sidebar-section-\(id)"
        let identifiedRow = app.descendants(matching: .any).matching(identifier: identifier).firstMatch
        if identifiedRow.waitForExistence(timeout: 8) {
            identifiedRow.click()
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

    private func waitForElementEnabled(
        _ element: XCUIElement,
        timeout: TimeInterval = 8,
        file: StaticString = #filePath,
        line: UInt = #line
    ) {
        let predicate = NSPredicate(format: "enabled == true")
        let expectation = XCTNSPredicateExpectation(predicate: predicate, object: element)
        let waiterResult = XCTWaiter().wait(for: [expectation], timeout: timeout)
        XCTAssertEqual(waiterResult, .completed, "Element did not become enabled in time.", file: file, line: line)
    }

    private func elementWithIdentifier(
        _ identifier: String,
        fallback: XCUIElement,
        app: XCUIApplication
    ) -> XCUIElement {
        let identified = app.descendants(matching: .any).matching(identifier: identifier).firstMatch
        if identified.exists || identified.waitForExistence(timeout: 2) {
            return identified
        }
        return fallback
    }

    private func staticTextContaining(_ value: String, app: XCUIApplication) -> XCUIElement {
        app.staticTexts.containing(NSPredicate(format: "label CONTAINS %@", value)).firstMatch
    }

    private func launchReadyChatApp() -> XCUIApplication {
        let app = launchApp(scenario: .ready)
        openSidebarSection(id: "chat", title: "Chat", app: app)
        selectDefaultActingAsPrincipal(app: app)
        let composerInput = elementWithIdentifier(
            "chat-composer-native-input",
            fallback: app.textViews.firstMatch,
            app: app
        )
        XCTAssertTrue(
            composerInput.waitForExistence(timeout: 8),
            "Chat composer should be available in the ready fixture scenario."
        )
        return app
    }

    private func assertAutonomousChatOutcome(
        prompt: String,
        expectation: AutonomousChatExpectation,
        app: XCUIApplication
    ) {
        let timeline = chatTimeline(app: app)
        let composerInput = elementWithIdentifier(
            "chat-composer-native-input",
            fallback: app.textViews.firstMatch,
            app: app
        )
        XCTAssertTrue(composerInput.waitForExistence(timeout: 8))
        composerInput.click()
        composerInput.typeKey("a", modifierFlags: [.command])
        composerInput.typeKey(XCUIKeyboardKey.delete, modifierFlags: [.command])
        composerInput.typeText(prompt)
        composerInput.typeKey(XCUIKeyboardKey.return, modifierFlags: [])

        let toolResultRow = timeline.descendants(matching: .any)
            .matching(identifier: expectation.toolResultRowIdentifier)
            .firstMatch
        XCTAssertTrue(
            waitForTimelineElement(toolResultRow, in: timeline, timeout: 20),
            "Expected tool-result row `\(expectation.toolResultRowIdentifier)` after prompt `\(prompt)`."
        )

        let assistantRow = timeline.descendants(matching: .any)
            .matching(identifier: expectation.assistantRowIdentifier)
            .firstMatch
        scrollTimelineToBottom(timeline)
        XCTAssertTrue(
            waitForTimelineElement(assistantRow, in: timeline, timeout: 20),
            "Expected assistant row `\(expectation.assistantRowIdentifier)` after prompt `\(prompt)`."
        )

        let expectedBadge = toolResultRow.staticTexts.containing(
            NSPredicate(
                format: "(label == %@) OR (value == %@)",
                expectation.expectedResultBadge,
                expectation.expectedResultBadge
            )
        ).firstMatch
        XCTAssertTrue(
            expectedBadge.waitForExistence(timeout: 8),
            "Expected tool-result badge `\(expectation.expectedResultBadge)` for workflow `\(expectation.workflow)`."
        )

        let assistantText = assistantRow.staticTexts.containing(
            NSPredicate(
                format: "(label CONTAINS[c] %@) OR (value CONTAINS[c] %@)",
                expectation.expectedAssistantText,
                expectation.expectedAssistantText
            )
        ).firstMatch
        XCTAssertTrue(
            assistantText.waitForExistence(timeout: 8),
            "Expected assistant outcome text `\(expectation.expectedAssistantText)` for workflow `\(expectation.workflow)`."
        )

        if let expectedApprovalControlLabel = expectation.expectedApprovalControlLabel {
            let approvalRow = timeline.descendants(matching: .any)
                .matching(identifier: expectation.approvalRowIdentifier)
                .firstMatch
            XCTAssertTrue(
                waitForTimelineElement(approvalRow, in: timeline, timeout: 12),
                "Expected approval row `\(expectation.approvalRowIdentifier)` for workflow `\(expectation.workflow)`."
            )

            scrollTimelineToBottom(timeline)
            let approvalControl = approvalRow.buttons[expectedApprovalControlLabel].firstMatch
            XCTAssertTrue(
                waitForTimelineElement(approvalControl, in: timeline, timeout: 8),
                "Expected approval control `\(expectedApprovalControlLabel)` for workflow `\(expectation.workflow)`."
            )
        }
    }

    private func waitForTimelineElement(
        _ element: XCUIElement,
        in timeline: XCUIElement,
        timeout: TimeInterval
    ) -> Bool {
        if element.waitForExistence(timeout: min(timeout, 2)) {
            return true
        }

        let deadline = Date().addingTimeInterval(max(timeout - 2, 0))
        while Date() < deadline {
            scrollTimelineToBottom(timeline, maxAttempts: 1)
            if element.waitForExistence(timeout: 0.5) {
                return true
            }
        }
        return element.exists
    }

    private func scrollTimelineToBottom(_ timeline: XCUIElement, maxAttempts: Int = 4) {
        guard timeline.exists || timeline.waitForExistence(timeout: 2) else {
            return
        }

        for _ in 0..<maxAttempts {
            timeline.swipeUp()
        }
    }

    private func selectDefaultActingAsPrincipal(app: XCUIApplication) {
        let actingAsPicker = elementWithIdentifier(
            "chat-acting-as-picker",
            fallback: app.popUpButtons.firstMatch,
            app: app
        )
        guard actingAsPicker.exists || actingAsPicker.waitForExistence(timeout: 2) else {
            return
        }
        actingAsPicker.click()
        let defaultActorOption = app.menuItems["default"].firstMatch
        if defaultActorOption.exists || defaultActorOption.waitForExistence(timeout: 2) {
            defaultActorOption.click()
        }
    }

    private func commandPaletteSearchField(app: XCUIApplication) -> XCUIElement {
        let identified = app.descendants(matching: .any).matching(identifier: "command-palette-search-field").firstMatch
        if identified.exists || identified.waitForExistence(timeout: 2) {
            return identified
        }

        let named = app.searchFields["Search commands and objects"].firstMatch
        if named.exists || named.waitForExistence(timeout: 2) {
            return named
        }

        let legacyNamed = app.searchFields["Search commands"].firstMatch
        if legacyNamed.exists || legacyNamed.waitForExistence(timeout: 2) {
            return legacyNamed
        }

        let namedTextField = app.textFields["Search commands and objects"].firstMatch
        if namedTextField.exists || namedTextField.waitForExistence(timeout: 2) {
            return namedTextField
        }

        let legacyNamedTextField = app.textFields["Search commands"].firstMatch
        if legacyNamedTextField.exists || legacyNamedTextField.waitForExistence(timeout: 2) {
            return legacyNamedTextField
        }

        return app.textFields.firstMatch
    }

    private func chatTimeline(app: XCUIApplication) -> XCUIElement {
        let identified = app.descendants(matching: .any).matching(identifier: "chat-timeline-scroll").firstMatch
        if identified.exists || identified.waitForExistence(timeout: 2) {
            return identified
        }
        return panel(identifier: "panel-chat", app: app)
    }

    private func panel(identifier: String, app: XCUIApplication) -> XCUIElement {
        let scrollView = app.scrollViews.matching(identifier: identifier).firstMatch
        if scrollView.exists || scrollView.waitForExistence(timeout: 2) {
            return scrollView
        }

        return app.descendants(matching: .any).matching(identifier: identifier).firstMatch
    }

    private func expandFirstDisclosureCard(
        in panel: XCUIElement,
        panelName: String,
        timeout: TimeInterval = 8,
        file: StaticString = #filePath,
        line: UInt = #line
    ) {
        let disclosureTriangle = panel.descendants(matching: .disclosureTriangle).firstMatch
        XCTAssertTrue(
            disclosureTriangle.waitForExistence(timeout: timeout),
            "\(panelName) should expose at least one card disclosure toggle.",
            file: file,
            line: line
        )

        if disclosureControlIsCollapsed(disclosureTriangle) {
            disclosureTriangle.click()
        }
    }

    private func disclosureControlIsCollapsed(_ control: XCUIElement) -> Bool {
        if let value = control.value as? NSNumber {
            return value.intValue == 0
        }
        if let value = control.value as? String {
            return value == "0"
        }
        return true
    }
}

private struct AutonomousChatExpectation {
    let workflow: String
    let expectedResultBadge: String
    let expectedAssistantText: String
    let expectedApprovalControlLabel: String?

    init(
        workflow: String,
        expectedResultBadge: String,
        expectedAssistantText: String,
        expectedApprovalControlLabel: String? = nil
    ) {
        self.workflow = workflow
        self.expectedResultBadge = expectedResultBadge
        self.expectedAssistantText = expectedAssistantText
        self.expectedApprovalControlLabel = expectedApprovalControlLabel
    }

    var toolResultRowIdentifier: String {
        "chat-timeline-item-item-tool-result-\(workflow)"
    }

    var assistantRowIdentifier: String {
        "chat-timeline-item-item-assistant-\(workflow)"
    }

    var approvalRowIdentifier: String {
        "chat-timeline-item-item-approval-\(workflow)"
    }
}

private enum FixtureScenario: String {
    case ready
    case onboarding
    case degraded
}
