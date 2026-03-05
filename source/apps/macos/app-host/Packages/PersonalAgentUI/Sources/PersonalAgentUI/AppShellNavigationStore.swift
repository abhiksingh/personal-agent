import Foundation
import SwiftUI

@MainActor
final class AppShellNavigationStore: ObservableObject {
    enum SectionSelectionOutcome: Equatable {
        case unchanged
        case selected
        case pendingDiscardConfirmation
    }

    @Published var selectedSection: AppSection = .home
    @Published var activeDrillInNavigationContext: DrillInNavigationContext? = nil
    @Published var pendingSectionNavigationSource: AppSection? = nil
    @Published var pendingSectionNavigationTarget: AppSection? = nil
    @Published var pendingSectionNavigationSummary: String? = nil
    @Published var showsUnsavedChangesNavigationAlert = false
    @Published var isSidebarVisible = true
    @Published private(set) var isAdvancedSidebarNavigationExpanded = false

    var isAdvancedSidebarNavigationVisible: Bool {
        isAdvancedSidebarNavigationExpanded || selectedSection.isAdvancedSidebarDestination
    }

    var visibleSidebarNavigationSections: [AppSection] {
        if isAdvancedSidebarNavigationVisible {
            return AppSection.middleSidebarSections
        }
        return AppSection.primarySidebarSections
    }

    func toggleSidebar() {
        isSidebarVisible.toggle()
    }

    func setSidebarAdvancedNavigationExpanded(_ isExpanded: Bool) {
        if selectedSection.isAdvancedSidebarDestination {
            isAdvancedSidebarNavigationExpanded = true
        } else {
            isAdvancedSidebarNavigationExpanded = isExpanded
        }
    }

    func toggleSidebarAdvancedNavigationExpanded() {
        setSidebarAdvancedNavigationExpanded(!isAdvancedSidebarNavigationExpanded)
    }

    func applySelectedSectionChangeSideEffects() {
        if let context = activeDrillInNavigationContext,
           context.destinationSection != selectedSection {
            activeDrillInNavigationContext = nil
        }
        if selectedSection.isAdvancedSidebarDestination {
            isAdvancedSidebarNavigationExpanded = true
        }
    }

    func activeDrillInContextForSelectedSection() -> DrillInNavigationContext? {
        guard let context = activeDrillInNavigationContext,
              context.destinationSection == selectedSection else {
            return nil
        }
        return context
    }

    func clearActiveDrillInNavigationContext() {
        activeDrillInNavigationContext = nil
    }

    @discardableResult
    func requestSectionSelection(
        _ section: AppSection,
        preservingDrillInContext: Bool,
        hasUnsavedDraftChanges: Bool,
        unsavedDraftSummary: String?
    ) -> SectionSelectionOutcome {
        if selectedSection == section {
            return .unchanged
        }
        if !preservingDrillInContext {
            activeDrillInNavigationContext = nil
        }
        guard hasUnsavedDraftChanges else {
            selectedSection = section
            return .selected
        }
        pendingSectionNavigationSource = selectedSection
        pendingSectionNavigationTarget = section
        pendingSectionNavigationSummary = unsavedDraftSummary
        showsUnsavedChangesNavigationAlert = true
        return .pendingDiscardConfirmation
    }

    func clearPendingSectionNavigation() {
        pendingSectionNavigationSource = nil
        pendingSectionNavigationTarget = nil
        pendingSectionNavigationSummary = nil
        showsUnsavedChangesNavigationAlert = false
    }

    func discardPendingSectionNavigationAndSelectTarget() -> AppSection? {
        let targetSection = pendingSectionNavigationTarget
        clearPendingSectionNavigation()
        if let targetSection {
            selectedSection = targetSection
        }
        return targetSection
    }
}
