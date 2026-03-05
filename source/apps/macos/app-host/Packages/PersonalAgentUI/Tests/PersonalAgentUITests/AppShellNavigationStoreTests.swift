import XCTest
@testable import PersonalAgentUI

@MainActor
final class AppShellNavigationStoreTests: XCTestCase {
    func testRequestSectionSelectionSelectsTargetWhenNoUnsavedDrafts() {
        let store = AppShellNavigationStore()
        store.selectedSection = .chat

        let outcome = store.requestSectionSelection(
            .models,
            preservingDrillInContext: false,
            hasUnsavedDraftChanges: false,
            unsavedDraftSummary: nil
        )

        XCTAssertEqual(outcome, .selected)
        XCTAssertEqual(store.selectedSection, .models)
        XCTAssertFalse(store.showsUnsavedChangesNavigationAlert)
    }

    func testRequestSectionSelectionStagesPendingNavigationWhenUnsavedDraftsExist() {
        let store = AppShellNavigationStore()
        store.selectedSection = .channels

        let outcome = store.requestSectionSelection(
            .connectors,
            preservingDrillInContext: false,
            hasUnsavedDraftChanges: true,
            unsavedDraftSummary: "Unsaved channel draft"
        )

        XCTAssertEqual(outcome, .pendingDiscardConfirmation)
        XCTAssertEqual(store.pendingSectionNavigationSource, .channels)
        XCTAssertEqual(store.pendingSectionNavigationTarget, .connectors)
        XCTAssertEqual(store.pendingSectionNavigationSummary, "Unsaved channel draft")
        XCTAssertTrue(store.showsUnsavedChangesNavigationAlert)
        XCTAssertEqual(store.selectedSection, .channels)
    }

    func testApplySelectedSectionChangeSideEffectsClearsMismatchedDrillInAndExpandsAdvanced() {
        let store = AppShellNavigationStore()
        store.activeDrillInNavigationContext = DrillInNavigationContext(
            sourceSection: .tasks,
            destinationSection: .inspect,
            chips: ["Task: t1"]
        )
        store.selectedSection = .chat

        store.applySelectedSectionChangeSideEffects()
        XCTAssertNil(store.activeDrillInNavigationContext)

        store.selectedSection = .inspect
        store.applySelectedSectionChangeSideEffects()
        XCTAssertTrue(store.isAdvancedSidebarNavigationVisible)
    }

    func testDiscardPendingSectionNavigationAndSelectTargetClearsPromptState() {
        let store = AppShellNavigationStore()
        store.selectedSection = .models
        _ = store.requestSectionSelection(
            .connectors,
            preservingDrillInContext: false,
            hasUnsavedDraftChanges: true,
            unsavedDraftSummary: "Unsaved model draft"
        )

        let target = store.discardPendingSectionNavigationAndSelectTarget()

        XCTAssertEqual(target, .connectors)
        XCTAssertEqual(store.selectedSection, .connectors)
        XCTAssertNil(store.pendingSectionNavigationSource)
        XCTAssertNil(store.pendingSectionNavigationTarget)
        XCTAssertNil(store.pendingSectionNavigationSummary)
        XCTAssertFalse(store.showsUnsavedChangesNavigationAlert)
    }
}
