import XCTest
@testable import PersonalAgentUI

@MainActor
final class AppShellStateSidebarNavigationTests: XCTestCase {
    func testAdvancedSidebarSectionsAreCollapsedByDefault() {
        let state = AppShellState()

        XCTAssertFalse(state.isAdvancedSidebarNavigationExpanded)
        XCTAssertFalse(state.isAdvancedSidebarNavigationVisible)
        XCTAssertEqual(state.visibleSidebarNavigationSections, AppSection.primarySidebarSections)
    }

    func testAdvancedSidebarDisclosureCanExpandAndCollapseOnWorkflowSections() {
        let state = AppShellState()

        state.setSidebarAdvancedNavigationExpanded(true)
        XCTAssertTrue(state.isAdvancedSidebarNavigationExpanded)
        XCTAssertTrue(state.isAdvancedSidebarNavigationVisible)
        XCTAssertEqual(state.visibleSidebarNavigationSections, AppSection.middleSidebarSections)

        state.setSidebarAdvancedNavigationExpanded(false)
        XCTAssertFalse(state.isAdvancedSidebarNavigationExpanded)
        XCTAssertFalse(state.isAdvancedSidebarNavigationVisible)
        XCTAssertEqual(state.visibleSidebarNavigationSections, AppSection.primarySidebarSections)
    }

    func testSelectingAdvancedSectionAutoExpandsDisclosureAndPreservesShortcutNavigation() {
        let state = AppShellState()
        state.setSidebarAdvancedNavigationExpanded(false)

        state.performAppCommand(.openConnectors)

        XCTAssertEqual(state.selectedSection, .connectors)
        XCTAssertTrue(state.isAdvancedSidebarNavigationExpanded)
        XCTAssertTrue(state.isAdvancedSidebarNavigationVisible)
        XCTAssertEqual(state.visibleSidebarNavigationSections, AppSection.middleSidebarSections)
    }

    func testAdvancedDisclosureCannotHideActiveAdvancedDestination() {
        let state = AppShellState()

        state.performAppCommand(.openInspect)
        XCTAssertEqual(state.selectedSection, .inspect)

        state.setSidebarAdvancedNavigationExpanded(false)

        XCTAssertTrue(state.isAdvancedSidebarNavigationExpanded)
        XCTAssertTrue(state.isAdvancedSidebarNavigationVisible)

        state.performAppCommand(.openChat)
        state.setSidebarAdvancedNavigationExpanded(false)

        XCTAssertEqual(state.selectedSection, .chat)
        XCTAssertFalse(state.isAdvancedSidebarNavigationExpanded)
        XCTAssertFalse(state.isAdvancedSidebarNavigationVisible)
    }
}
