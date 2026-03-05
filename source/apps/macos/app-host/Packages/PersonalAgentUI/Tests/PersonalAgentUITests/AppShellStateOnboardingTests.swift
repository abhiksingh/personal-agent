import XCTest
@testable import PersonalAgentUI

@MainActor
final class AppShellStateOnboardingTests: XCTestCase {
    func testOnboardingReadinessRequiresProviderCatalogRouteAndMappingChecks() {
        let state = AppShellState()
        configureReadyOnboardingState(state)
        state.providerReadinessItems = [
            ProviderReadinessItem(
                id: "openai",
                provider: "openai",
                endpoint: "https://api.openai.com/v1",
                status: .missingSetup,
                detail: "Setup Required",
                updatedAtLabel: "n/a"
            )
        ]
        state.modelRouteSummary = nil

        XCTAssertFalse(state.onboardingProviderReady)
        XCTAssertFalse(state.onboardingChatRouteReady)
        XCTAssertFalse(state.onboardingReadinessMet)
        XCTAssertEqual(state.onboardingFixNextStep?.id, "provider")
        XCTAssertTrue(state.onboardingStatusMessage.contains("Fix Next"))
    }

    func testSetupReadinessChecksLoadingUntilInitialDaemonAndProviderLoadsComplete() {
        let state = AppShellState()
        state.isDaemonLifecycleLoading = false
        state.isProviderStatusLoading = false
        state.hasLoadedDaemonStatus = false
        state.hasLoadedProviderStatus = false

        XCTAssertTrue(state.setupReadinessChecksLoading)

        state.hasLoadedDaemonStatus = true
        XCTAssertTrue(state.setupReadinessChecksLoading)

        state.hasLoadedProviderStatus = true
        XCTAssertFalse(state.setupReadinessChecksLoading)
    }

    func testSetupReadinessChecksLoadingTracksActiveRefreshFlags() {
        let state = AppShellState()
        state.hasLoadedDaemonStatus = true
        state.hasLoadedProviderStatus = true
        state.isDaemonLifecycleLoading = true

        XCTAssertTrue(state.setupReadinessChecksLoading)

        state.isDaemonLifecycleLoading = false
        state.isProviderStatusLoading = true
        XCTAssertTrue(state.setupReadinessChecksLoading)

        state.isProviderStatusLoading = false
        XCTAssertFalse(state.setupReadinessChecksLoading)
    }

    func testOnboardingReadinessCompletesWhenAllGuidedSetupChecksAreReady() {
        let state = AppShellState()
        configureReadyOnboardingState(state)

        XCTAssertTrue(state.onboardingProviderReady)
        XCTAssertTrue(state.onboardingModelCatalogReady)
        XCTAssertTrue(state.onboardingChatRouteReady)
        XCTAssertTrue(state.onboardingChannelConnectorMappingReady)
        XCTAssertTrue(state.onboardingReadinessMet)
        XCTAssertNil(state.onboardingFixNextStep)
        XCTAssertTrue(state.onboardingSetupProgressSummary.contains("Setup is complete"))
    }

    func testOnboardingFixNextPrioritizesTokenBeforeDaemonAndNavigatesToConfiguration() {
        let state = AppShellState()
        state.clearLocalDevToken()
        state.daemonStatus = .missing

        XCTAssertEqual(state.onboardingFixNextStep?.id, "token")

        state.performOnboardingFixNextStep()

        XCTAssertEqual(state.selectedSection, .configuration)
    }

    func testOnboardingFixNextTreatsMissingControlAuthAsTokenBlocker() {
        let state = AppShellState()
        configureReadyOnboardingState(state)
        state.daemonControlAuthState = .missing
        state.daemonControlAuthRemediationHints = [
            "Rotate Assistant Access Token and restart daemon."
        ]

        XCTAssertEqual(state.onboardingFixNextStep?.id, "token")
        XCTAssertTrue(state.onboardingStatusMessage.contains("Fix Next"))
        XCTAssertFalse(state.onboardingReadinessMet)
    }

    func testOnboardingFixNextForChatRouteNavigatesToModels() {
        let state = AppShellState()
        configureReadyOnboardingState(state)
        state.modelRouteSummary = nil

        XCTAssertEqual(state.onboardingFixNextStep?.id, "chat_route")

        state.performOnboardingFixNextStep()

        XCTAssertEqual(state.selectedSection, .models)
    }

    func testOnboardingWizardCurrentStepUsesFixNextBlockedStep() {
        let state = AppShellState()
        state.clearLocalDevToken()
        state.daemonStatus = .missing

        XCTAssertEqual(state.onboardingCurrentWizardStep?.id, "token")
        XCTAssertEqual(state.onboardingNextWizardStep?.id, "daemon")
    }

    func testOnboardingWizardCurrentStepFallsBackToLoadingStepWhenNoBlockedSteps() {
        let state = AppShellState()
        configureReadyOnboardingState(state)
        state.hasLoadedProviderStatus = false
        state.isProviderStatusLoading = true

        XCTAssertNil(state.onboardingFixNextStep)
        XCTAssertEqual(state.onboardingCurrentWizardStep?.id, "provider")
        XCTAssertEqual(state.onboardingCurrentWizardStep?.status, .loading)
    }

    func testOnboardingSetupProgressFractionUsesCompletedCount() {
        let state = AppShellState()
        configureReadyOnboardingState(state)

        XCTAssertEqual(state.onboardingSetupCompletedCount, state.onboardingSetupTotalCount)
        XCTAssertEqual(state.onboardingSetupProgressFraction, 1.0, accuracy: 0.0001)
    }

    func testOnboardingDetectsMissingChannelMappingAndNavigatesToChannels() {
        let state = AppShellState()
        configureReadyOnboardingState(state)
        state.channelConnectorMappingsByChannelID["voice"] = []

        XCTAssertFalse(state.onboardingChannelConnectorMappingReady)
        XCTAssertEqual(state.onboardingFixNextStep?.id, "channel_connector_mapping")

        state.performOnboardingFixNextStep()

        XCTAssertEqual(state.selectedSection, .channels)
    }

    func testOnboardingGateLeavesSetupSectionsAccessibleWhenIncomplete() {
        let state = AppShellState()
        state.clearLocalDevToken()

        XCTAssertTrue(state.needsFirstRunOnboarding)
        XCTAssertFalse(state.onboardingGateApplies(to: .configuration))
        XCTAssertFalse(state.onboardingGateApplies(to: .channels))
        XCTAssertFalse(state.onboardingGateApplies(to: .connectors))
        XCTAssertFalse(state.onboardingGateApplies(to: .models))
    }

    func testOnboardingGateStillAppliesToWorkflowSectionsWhenIncomplete() {
        let state = AppShellState()
        state.clearLocalDevToken()

        XCTAssertTrue(state.needsFirstRunOnboarding)
        XCTAssertTrue(state.onboardingGateApplies(to: .chat))
        XCTAssertTrue(state.onboardingGateApplies(to: .communications))
        XCTAssertTrue(state.onboardingGateApplies(to: .automation))
        XCTAssertTrue(state.onboardingGateApplies(to: .inspect))
    }

    func testCurrentSetupBlockerRibbonVisibilityForConfigurationAndNonConfigurationSections() {
        let state = AppShellState()
        state.clearLocalDevToken()

        XCTAssertFalse(state.shouldShowCurrentSetupBlockerRibbon(for: .configuration))
        XCTAssertTrue(state.shouldShowCurrentSetupBlockerRibbon(for: .chat))
        XCTAssertTrue(state.shouldShowCurrentSetupBlockerRibbon(for: .channels))
    }

    func testCurrentSetupBlockerSecondaryActionTargetsModelsForRouteBlocker() {
        let state = AppShellState()
        configureReadyOnboardingState(state)
        state.modelRouteSummary = nil

        XCTAssertEqual(state.onboardingFixNextStep?.id, "chat_route")
        XCTAssertEqual(state.currentSetupBlockerSecondaryAction?.kind, .openModels)
    }

    func testCurrentSetupBlockerSecondaryActionTargetsConfigurationForTokenBlocker() {
        let state = AppShellState()
        state.clearLocalDevToken()

        XCTAssertEqual(state.onboardingFixNextStep?.id, "token")
        XCTAssertEqual(state.currentSetupBlockerSecondaryAction?.kind, .openConfiguration)
    }

    func testCurrentSetupBlockerSecondaryActionUsesDisabledRefreshDuringLoading() {
        let state = AppShellState()
        configureReadyOnboardingState(state)
        state.hasLoadedProviderStatus = false
        state.isProviderStatusLoading = true

        XCTAssertTrue(state.onboardingSetupChecksLoading)
        XCTAssertEqual(state.currentSetupBlockerStatus, .loading)
        XCTAssertEqual(state.currentSetupBlockerSecondaryAction?.kind, .refreshChecks)
        XCTAssertEqual(state.currentSetupBlockerSecondaryAction?.isEnabled, false)
    }

    func testOpenOnboardingFromConfigurationSelectsChatSection() {
        let state = AppShellState()
        state.selectedSection = .configuration

        state.openOnboardingFromConfiguration()

        XCTAssertEqual(state.selectedSection, .chat)
    }

    private func configureReadyOnboardingState(_ state: AppShellState) {
        state.localDevTokenConfigured = true
        state.daemonStatus = .running
        state.connectionStatus = .connected
        state.hasLoadedDaemonStatus = true
        state.daemonControlAuthState = .configured
        state.daemonControlAuthSource = "auth_token_flag"
        state.daemonControlAuthRemediationHints = []
        state.hasLoadedProviderStatus = true
        state.hasLoadedChannelStatus = true
        state.hasLoadedConnectorStatus = true
        state.isDaemonLifecycleLoading = false
        state.isProviderStatusLoading = false
        state.isChannelsLoading = false
        state.isConnectorsLoading = false
        state.isChannelConnectorMappingsLoading = false
        state.providerReadinessItems = [
            ProviderReadinessItem(
                id: "openai",
                provider: "openai",
                endpoint: "https://api.openai.com/v1",
                status: .healthy,
                detail: "Healthy",
                updatedAtLabel: "n/a"
            )
        ]
        state.modelCatalogItems = [
            ModelCatalogEntryItem(
                id: "openai/gpt-5-codex",
                provider: "openai",
                modelKey: "gpt-5-codex",
                enabled: true,
                providerReady: true,
                providerEndpoint: "https://api.openai.com/v1"
            )
        ]
        state.modelRouteSummary = ModelRouteSummary(
            provider: "openai",
            modelKey: "gpt-5-codex",
            source: "policy",
            notes: nil
        )
        state.channelConnectorMappingsByChannelID = [
            "app": [
                ChannelConnectorMappingItem(
                    channelID: "app",
                    connectorID: "builtin.app",
                    enabled: true,
                    priority: 1,
                    capabilities: ["channel.app_chat.send"],
                    createdAtLabel: nil,
                    updatedAtLabel: nil
                )
            ],
            "message": [
                ChannelConnectorMappingItem(
                    channelID: "message",
                    connectorID: "imessage",
                    enabled: true,
                    priority: 1,
                    capabilities: ["channel.messages.send"],
                    createdAtLabel: nil,
                    updatedAtLabel: nil
                ),
                ChannelConnectorMappingItem(
                    channelID: "message",
                    connectorID: "twilio",
                    enabled: true,
                    priority: 2,
                    capabilities: ["channel.twilio.sms.send"],
                    createdAtLabel: nil,
                    updatedAtLabel: nil
                )
            ],
            "voice": [
                ChannelConnectorMappingItem(
                    channelID: "voice",
                    connectorID: "twilio",
                    enabled: true,
                    priority: 1,
                    capabilities: ["channel.twilio.voice.start_call"],
                    createdAtLabel: nil,
                    updatedAtLabel: nil
                )
            ]
        ]
        state.connectorCards = [
            makeConnectorCard(id: "builtin.app", name: "App"),
            makeConnectorCard(id: "imessage", name: "Messages"),
            makeConnectorCard(id: "twilio", name: "Twilio")
        ]
    }

    private func makeConnectorCard(
        id: String,
        name: String
    ) -> ConnectorCardItem {
        ConnectorCardItem(
            id: id,
            name: name,
            health: .ready,
            permissionState: .granted,
            permissionScope: "n/a",
            statusReason: nil,
            summary: "Ready",
            details: [:],
            editableConfiguration: [:],
            editableConfigurationKinds: [:],
            readOnlyConfiguration: [:],
            actions: [],
            unavailableActionReason: "",
            isExpanded: false
        )
    }
}
