import XCTest
@testable import PersonalAgentUI

@MainActor
final class AppWorkflowMutationStoreTests: XCTestCase {
    func testConfirmPendingHighImpactActionExecutesHandler() {
        let store = AppWorkflowMutationStore()
        var didExecute = false

        store.presentHighImpactActionConfirmation(
            title: "Confirm",
            message: "Message",
            confirmButtonTitle: "Run",
            isDestructive: false
        ) {
            didExecute = true
        }

        XCTAssertNotNil(store.pendingHighImpactActionConfirmation)
        store.confirmPendingHighImpactAction()
        XCTAssertTrue(didExecute)
        XCTAssertNil(store.pendingHighImpactActionConfirmation)
    }

    func testCancelPendingHighImpactActionClearsPromptWithoutExecutingHandler() {
        let store = AppWorkflowMutationStore()
        var didExecute = false

        store.presentHighImpactActionConfirmation(
            title: "Confirm",
            message: "Message",
            confirmButtonTitle: "Run",
            isDestructive: true
        ) {
            didExecute = true
        }

        store.cancelPendingHighImpactAction()
        XCTAssertFalse(didExecute)
        XCTAssertNil(store.pendingHighImpactActionConfirmation)
    }

    func testPerformActiveUndoActionExecutesHandlerAndClearsPrompt() {
        let store = AppWorkflowMutationStore()
        var didUndo = false

        store.presentUndoActionPrompt(
            title: "Undo",
            message: "Message",
            visibleForSeconds: 0
        ) {
            didUndo = true
        }

        XCTAssertNotNil(store.activeUndoActionPrompt)
        store.performActiveUndoAction()
        XCTAssertTrue(didUndo)
        XCTAssertNil(store.activeUndoActionPrompt)
    }
}
