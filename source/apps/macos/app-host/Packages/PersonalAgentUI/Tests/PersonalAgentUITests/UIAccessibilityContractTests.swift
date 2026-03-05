import XCTest
@testable import PersonalAgentUI

final class UIAccessibilityContractTests: XCTestCase {
    func testCoreAccessibilityIdentifiersAreStableAndUnique() {
        let identifiers = [
            UIAccessibilityContract.sidebarNavigationIdentifier,
            UIAccessibilityContract.commandPaletteSearchIdentifier,
            UIAccessibilityContract.approvalsSearchIdentifier,
            UIAccessibilityContract.tasksSearchIdentifier,
            UIAccessibilityContract.communicationsSearchIdentifier
        ]

        XCTAssertEqual(identifiers.count, Set(identifiers).count)
        XCTAssertFalse(identifiers.contains(where: { $0.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty }))
        XCTAssertEqual(UIAccessibilityContract.commandPaletteSearchIdentifier, "command-palette-search-field")
    }

    func testVoiceOverLabelsAndHintsRemainNonEmptyForCoreLandmarksAndActions() {
        let strings = [
            UIAccessibilityContract.sidebarNavigationLabel,
            UIAccessibilityContract.sidebarNavigationHint,
            UIAccessibilityContract.commandPaletteSearchLabel,
            UIAccessibilityContract.commandPaletteSearchHint,
            UIAccessibilityContract.approvalsSearchLabel,
            UIAccessibilityContract.approvalsSearchHint,
            UIAccessibilityContract.tasksSearchLabel,
            UIAccessibilityContract.tasksSearchHint,
            UIAccessibilityContract.communicationsSearchLabel,
            UIAccessibilityContract.communicationsSearchHint,
            UIAccessibilityContract.drillInDismissLabel,
            UIAccessibilityContract.drillInDismissHint
        ]

        XCTAssertFalse(strings.contains(where: { $0.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty }))
    }

    func testPanelLandmarkLabelIncludesSelectedSectionTitle() {
        XCTAssertEqual(
            UIAccessibilityContract.panelLandmarkLabel(for: "Chat"),
            "Chat workflow panel"
        )
    }
}
