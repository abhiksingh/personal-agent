import Foundation
import XCTest
@testable import PersonalAgentUI

@MainActor
final class AppShellStateModelsSetupTests: XCTestCase {
    private let tokenDefaultsKey = "personalagent.ui.local_dev_token"
    private let onboardingDefaultsKey = "personalagent.ui.onboarding_complete"

    override func setUp() {
        super.setUp()
        AppShellState._test_setLocalDevTokenSecretReference(
            service: "personalagent.ui.tests.models-setup.\(UUID().uuidString)",
            account: "daemon_auth_token"
        )
        AppShellState._test_clearPersistedLocalDevToken()
    }

    override func tearDown() {
        AppShellState._test_clearPersistedLocalDevToken()
        AppShellState._test_resetLocalDevTokenPersistenceHooks()
        super.tearDown()
    }

    func testProviderDraftDefaultsExposeEndpointAndSecretName() {
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

        XCTAssertEqual(state.providerEndpointDraft(for: "openai"), "https://api.openai.com/v1")
        XCTAssertEqual(state.providerSecretNameDraft(for: "openai"), "OPENAI_API_KEY")
        XCTAssertEqual(state.providerEndpointDraft(for: "ollama"), "http://127.0.0.1:11434")
        XCTAssertEqual(state.providerSecretNameDraft(for: "ollama"), "")
    }

    func testModelRouteReadinessChecklistIncludesExpectedStepOrder() {
        let state = AppShellState()

        let stepIDs = state.modelRouteReadinessChecklistSteps.map(\.id)

        XCTAssertEqual(stepIDs, ["token", "daemon", "provider", "model_catalog", "chat_route"])
    }

    func testModelRouteReadinessChecklistFlagsChatRouteBlockerWhenRouteMissing() {
        let state = AppShellState()
        state.localDevTokenConfigured = true
        state.hasLoadedDaemonStatus = true
        state.daemonControlAuthState = .configured
        state.daemonStatus = .running
        state.connectionStatus = .connected
        state.hasLoadedProviderStatus = true
        state.providerReadinessItems = [
            ProviderReadinessItem(
                id: "openai",
                provider: "openai",
                endpoint: "https://api.openai.com/v1",
                status: .healthy,
                detail: "Ready",
                updatedAtLabel: "now"
            )
        ]
        state.modelCatalogItems = [
            ModelCatalogEntryItem(
                id: "openai::gpt-5",
                provider: "openai",
                modelKey: "gpt-5",
                enabled: true,
                providerReady: true,
                providerEndpoint: "https://api.openai.com/v1"
            )
        ]
        state.modelRouteSummary = nil

        XCTAssertTrue(state.modelRouteReadinessNeedsAttention)
        XCTAssertEqual(state.modelRouteReadinessBlockerCount, 1)
        XCTAssertEqual(
            state.modelRouteReadinessChecklistSteps.last?.id,
            "chat_route"
        )
        XCTAssertEqual(
            state.modelRouteReadinessChecklistSteps.last?.status,
            .blocked
        )
    }

    func testSetModelAsChatRouteWithoutTokenSetsStatusMessage() async {
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
        state.setModelAsChatRoute(providerID: "openai", modelKey: "gpt-5")

        try? await Task.sleep(for: .milliseconds(50))

        XCTAssertEqual(
            state.modelRoutePolicySaveStatusMessage,
            "Set Assistant Access Token before setting chat route."
        )
    }

    func testSaveProviderSetupWithoutTokenSetsStatusMessage() async {
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
        state.saveProviderSetup(for: "openai")

        try? await Task.sleep(for: .milliseconds(50))

        XCTAssertEqual(
            state.providerSetupStatusByID["openai"],
            "Set Assistant Access Token before saving provider setup."
        )
    }

    func testRunProviderCheckWithoutTokenSetsStatusMessage() async {
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
        state.runProviderConnectivityCheck(for: "openai")

        try? await Task.sleep(for: .milliseconds(50))

        XCTAssertEqual(
            state.providerSetupStatusByID["openai"],
            "Set Assistant Access Token before running provider checks."
        )
    }

    func testResetProviderEndpointDraftUsesDefaultAndSetsStatus() {
        let state = AppShellState()
        state.setProviderEndpointDraft("http://custom-endpoint.local", for: "openai")

        state.resetProviderEndpointDraft(for: "openai")

        XCTAssertEqual(state.providerEndpointDraft(for: "openai"), "https://api.openai.com/v1")
        XCTAssertEqual(
            state.providerSetupStatusByID["openai"],
            "Endpoint reset to default. Save Provider to persist the change."
        )
    }

    func testSetModelEnabledWithoutTokenSetsStatusMessage() async {
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
        state.setModelEnabled(providerID: "openai", modelKey: "gpt-5-codex", enabled: true)

        try? await Task.sleep(for: .milliseconds(50))

        XCTAssertEqual(
            state.modelMutationStatusByID["openai::gpt-5-codex"],
            "Set Assistant Access Token before changing model enablement."
        )
    }

    func testDiscoverModelsWithoutTokenSetsProviderStatusMessage() async {
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
        state.discoverModels(for: "ollama")

        try? await Task.sleep(for: .milliseconds(50))

        XCTAssertEqual(
            state.modelCatalogManagementStatusByProviderID["ollama"],
            "Set Assistant Access Token before discovering models."
        )
    }

    func testAddModelWithoutTokenSetsProviderStatusMessage() async {
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
        state.addModelToCatalog(providerID: "ollama", modelKey: "llama3.2")

        try? await Task.sleep(for: .milliseconds(50))

        XCTAssertEqual(
            state.modelCatalogManagementStatusByProviderID["ollama"],
            "Set Assistant Access Token before adding models."
        )
    }

    func testRemoveModelWithoutTokenSetsProviderStatusMessage() async {
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
        state.removeModelFromCatalog(providerID: "ollama", modelKey: "llama3.2")

        try? await Task.sleep(for: .milliseconds(50))

        XCTAssertEqual(
            state.modelCatalogManagementStatusByProviderID["ollama"],
            "Set Assistant Access Token before removing models."
        )
    }

    func testModelManualAddDraftRoundTripsPerProvider() {
        let state = AppShellState()

        state.setModelManualAddDraft("llama3.2", for: "ollama")

        XCTAssertEqual(state.modelManualAddDraft(for: "ollama"), "llama3.2")
    }

    func testSaveModelRoutePolicyWithoutTokenSetsStatusMessage() async {
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
        state.saveModelRoutePolicy(taskClass: "chat", providerID: "openai", modelKey: "gpt-5-codex")

        try? await Task.sleep(for: .milliseconds(50))

        XCTAssertEqual(
            state.modelRoutePolicySaveStatusMessage,
            "Set Assistant Access Token before saving route policy."
        )
    }

    func testRunProviderQuickstartSaveAndCheckWithoutTokenStopsAtSaveValidation() async {
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
        state.runProviderQuickstartSaveAndCheck(providerID: "openai")

        try? await Task.sleep(for: .milliseconds(50))

        XCTAssertEqual(
            state.providerSetupStatusByID["openai"],
            "Set Assistant Access Token before saving provider setup."
        )
        XCTAssertFalse(state.providerCheckInFlightIDs.contains("openai"))
    }

    func testOpenChatForModelsQuickstartTestSeedsDraftWhenComposerIsEmpty() {
        let state = AppShellState()
        state.chatDraft = ""
        state.selectedSection = .models

        state.openChatForModelsQuickstartTest(providerID: "openai", modelKey: "gpt-5-codex")

        XCTAssertEqual(state.selectedSection, .chat)
        XCTAssertEqual(
            state.chatDraft,
            "Quickstart check: confirm the OpenAI • gpt-5-codex route is ready and respond with one short sentence."
        )
        XCTAssertEqual(
            state.chatStatusMessage,
            "Models quickstart is ready for OpenAI • gpt-5-codex. Send a message to validate chat routing."
        )
        XCTAssertEqual(
            state.activeDrillInNavigationContext,
            DrillInNavigationContext(
                sourceSection: .models,
                destinationSection: .chat,
                chips: ["Route: OpenAI • gpt-5-codex"]
            )
        )
    }

    func testOpenChatForModelsQuickstartTestPreservesExistingDraft() {
        let state = AppShellState()
        state.chatDraft = "Existing user draft"

        state.openChatForModelsQuickstartTest(providerID: "ollama", modelKey: "gpt-oss:20b")

        XCTAssertEqual(state.chatDraft, "Existing user draft")
        XCTAssertEqual(
            state.chatStatusMessage,
            "Models quickstart is ready for Ollama • gpt-oss:20b. Send a message to validate chat routing."
        )
    }

    func testRunModelRouteSimulationWithoutTokenSetsStatusMessage() async {
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
        state.modelRouteSimulationTaskClass = "chat"
        state.runModelRouteSimulation()

        try? await Task.sleep(for: .milliseconds(50))

        XCTAssertEqual(
            state.modelRouteSimulationStatusMessage,
            "Set Assistant Access Token before running route simulation."
        )
        XCTAssertNil(state.modelRouteSimulationResult)
    }

    func testRunModelRouteExplainWithoutTokenSetsStatusMessage() async {
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
        state.modelRouteSimulationTaskClass = "chat"
        state.runModelRouteExplainability()

        try? await Task.sleep(for: .milliseconds(50))

        XCTAssertEqual(
            state.modelRouteExplainStatusMessage,
            "Set Assistant Access Token before running route explainability."
        )
        XCTAssertNil(state.modelRouteExplainResult)
    }

    func testRunModelRouteSimulationWithMissingTaskClassSetsValidationMessage() async {
        let state = AppShellState()
        state.localDevTokenInput = "test-token"
        state.saveLocalDevToken()
        state.modelRouteSimulationTaskClass = "   "
        state.runModelRouteSimulation()

        try? await Task.sleep(for: .milliseconds(50))

        XCTAssertEqual(
            state.modelRouteSimulationStatusMessage,
            "Task class is required for route simulation."
        )
    }

    func testSaveModelRoutePolicyWithMissingTaskClassSetsValidationMessage() async {
        let state = AppShellState()
        state.saveModelRoutePolicy(taskClass: "   ", providerID: "openai", modelKey: "gpt-5-codex")

        try? await Task.sleep(for: .milliseconds(50))

        XCTAssertEqual(state.modelRoutePolicySaveStatusMessage, "Task class is required.")
    }
}
