import Foundation
import XCTest
@testable import PersonalAgentUI

@MainActor
final class AppShellStateContextMemoryTests: XCTestCase {
    private let tokenDefaultsKey = "personalagent.ui.local_dev_token"
    private let onboardingDefaultsKey = "personalagent.ui.onboarding_complete"

    override func setUp() {
        super.setUp()
        AppShellState._test_setLocalDevTokenSecretReference(
            service: "personalagent.ui.tests.context-memory.\(UUID().uuidString)",
            account: "daemon_auth_token"
        )
        AppShellState._test_clearPersistedLocalDevToken()
    }

    override func tearDown() {
        AppShellState._test_clearPersistedLocalDevToken()
        AppShellState._test_resetLocalDevTokenPersistenceHooks()
        super.tearDown()
    }

    func testRefreshContextMemoryInventoryWithoutTokenSetsDeterministicStatus() async {
        let defaults = appShellStateTestUserDefaults()
        let priorToken = defaults.object(forKey: tokenDefaultsKey)
        let priorOnboarding = defaults.object(forKey: onboardingDefaultsKey)
        defer {
            if let priorToken {
                defaults.set(priorToken, forKey: tokenDefaultsKey)
            } else {
                defaults.removeObject(forKey: tokenDefaultsKey)
            }
            if let priorOnboarding {
                defaults.set(priorOnboarding, forKey: onboardingDefaultsKey)
            } else {
                defaults.removeObject(forKey: onboardingDefaultsKey)
            }
        }

        defaults.removeObject(forKey: tokenDefaultsKey)
        defaults.removeObject(forKey: onboardingDefaultsKey)

        let state = AppShellState()
        state.clearLocalDevToken()
        state.refreshContextMemoryInventory()
        try? await Task.sleep(for: .milliseconds(50))

        XCTAssertEqual(state.contextMemoryInventoryItems.count, 0)
        XCTAssertEqual(
            state.contextMemoryInventoryStatusMessage,
            "Set Assistant Access Token to query context memory inventory."
        )
        XCTAssertEqual(state.isContextMemoryInventoryLoading, false)
    }

    func testRefreshContextMemoryCandidatesWithoutTokenSetsDeterministicStatus() async {
        let defaults = appShellStateTestUserDefaults()
        let priorToken = defaults.object(forKey: tokenDefaultsKey)
        let priorOnboarding = defaults.object(forKey: onboardingDefaultsKey)
        defer {
            if let priorToken {
                defaults.set(priorToken, forKey: tokenDefaultsKey)
            } else {
                defaults.removeObject(forKey: tokenDefaultsKey)
            }
            if let priorOnboarding {
                defaults.set(priorOnboarding, forKey: onboardingDefaultsKey)
            } else {
                defaults.removeObject(forKey: onboardingDefaultsKey)
            }
        }

        defaults.removeObject(forKey: tokenDefaultsKey)
        defaults.removeObject(forKey: onboardingDefaultsKey)

        let state = AppShellState()
        state.clearLocalDevToken()
        state.refreshContextMemoryCandidates()
        try? await Task.sleep(for: .milliseconds(50))

        XCTAssertEqual(state.contextMemoryCandidateItems.count, 0)
        XCTAssertEqual(
            state.contextMemoryCandidatesStatusMessage,
            "Set Assistant Access Token to query memory compaction candidates."
        )
        XCTAssertEqual(state.isContextMemoryCandidatesLoading, false)
    }

    func testRefreshContextRetrievalDocumentsWithoutTokenSetsDeterministicStatus() async {
        let defaults = appShellStateTestUserDefaults()
        let priorToken = defaults.object(forKey: tokenDefaultsKey)
        let priorOnboarding = defaults.object(forKey: onboardingDefaultsKey)
        defer {
            if let priorToken {
                defaults.set(priorToken, forKey: tokenDefaultsKey)
            } else {
                defaults.removeObject(forKey: tokenDefaultsKey)
            }
            if let priorOnboarding {
                defaults.set(priorOnboarding, forKey: onboardingDefaultsKey)
            } else {
                defaults.removeObject(forKey: onboardingDefaultsKey)
            }
        }

        defaults.removeObject(forKey: tokenDefaultsKey)
        defaults.removeObject(forKey: onboardingDefaultsKey)

        let state = AppShellState()
        state.clearLocalDevToken()
        state.refreshContextRetrievalDocuments()
        try? await Task.sleep(for: .milliseconds(50))

        XCTAssertEqual(state.contextRetrievalDocumentItems.count, 0)
        XCTAssertEqual(
            state.contextRetrievalDocumentsStatusMessage,
            "Set Assistant Access Token to query retrieval documents."
        )
        XCTAssertEqual(state.selectedContextRetrievalDocumentID, "")
        XCTAssertEqual(state.isContextRetrievalDocumentsLoading, false)
    }

    func testRefreshContextRetrievalChunksWithoutDocumentShowsSelectionGuidance() async {
        let state = AppShellState()
        state.selectedContextRetrievalDocumentID = ""
        state.refreshContextRetrievalChunks()
        try? await Task.sleep(for: .milliseconds(20))

        XCTAssertEqual(state.contextRetrievalChunkItems.count, 0)
        XCTAssertEqual(
            state.contextRetrievalChunksStatusMessage,
            "Select a retrieval document to inspect retrieval chunks."
        )
        XCTAssertEqual(state.isContextRetrievalChunksLoading, false)
    }

    func testResetContextMemoryInventoryFiltersDefaultsToSelectedPrincipal() {
        let state = AppShellState()
        state.selectedPrincipal = "actor.context.a"
        state.contextMemoryOwnerActorFilter = "actor.other"
        state.contextMemoryScopeTypeFilter = "conversation"
        state.contextMemoryStatusFilter = "ACTIVE"
        state.contextMemorySourceTypeFilter = "comm_event"
        state.contextMemorySourceRefQuery = "event://manual"
        state.contextMemoryLimit = 80

        state.resetContextMemoryInventoryFilters()

        XCTAssertEqual(state.contextMemoryOwnerActorFilter, "actor.context.a")
        XCTAssertEqual(state.contextMemoryScopeTypeFilter, "")
        XCTAssertEqual(state.contextMemoryStatusFilter, "all")
        XCTAssertEqual(state.contextMemorySourceTypeFilter, "")
        XCTAssertEqual(state.contextMemorySourceRefQuery, "")
        XCTAssertEqual(state.contextMemoryLimit, 25)
    }
}
