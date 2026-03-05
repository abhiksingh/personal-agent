import Combine
import Foundation
import AppKit
import SwiftUI

private func daemonTimestampString(_ date: Date) -> String {
    let formatter = ISO8601DateFormatter()
    formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
    return formatter.string(from: date)
}

private func daemonTimestampParser() -> ISO8601DateFormatter {
    let formatter = ISO8601DateFormatter()
    formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
    return formatter
}

@MainActor
public final class AppShellState: ObservableObject {
    private typealias WorkspaceContextUpdateIntent = AppIdentityContextStore.WorkspaceContextUpdateIntent
    private typealias HomeFirstSessionProgress = AppContextRetentionStore.HomeFirstSessionProgress
    private typealias PanelLatencyTrigger = AppPanelLatencyStore.Trigger

    private struct OnboardingChannelConnectorAssessment {
        let missingChannels: [String]
        let missingConnectorIDs: [String]
    }

    private struct ChatRealtimeFallbackContext {
        let reason: ChatRealtimeFallbackReason
        let statusMessage: String
        let progressDetail: String
        let remediationDetail: String
        let notificationSummary: String
    }

    private enum ChatFixAndContinueSource {
        case routeRemediation
        case failureRemediation
    }

    private struct PendingChatFixAndContinue {
        let source: ChatFixAndContinueSource
        let draft: String?
        let requiredSection: AppSection
    }

    private let navigationStore = AppShellNavigationStore()
    public var selectedSection: AppSection {
        get { navigationStore.selectedSection }
        set {
            guard navigationStore.selectedSection != newValue else {
                return
            }
            navigationStore.selectedSection = newValue
            handleSelectedSectionDidChange()
        }
    }
    public var activeDrillInNavigationContext: DrillInNavigationContext? {
        get { navigationStore.activeDrillInNavigationContext }
        set { navigationStore.activeDrillInNavigationContext = newValue }
    }
    @Published public private(set) var informationDensityMode: AppInformationDensityMode = .simple
    public var pendingSectionNavigationSource: AppSection? {
        get { navigationStore.pendingSectionNavigationSource }
        set { navigationStore.pendingSectionNavigationSource = newValue }
    }
    public var pendingSectionNavigationTarget: AppSection? {
        get { navigationStore.pendingSectionNavigationTarget }
        set { navigationStore.pendingSectionNavigationTarget = newValue }
    }
    public var pendingSectionNavigationSummary: String? {
        get { navigationStore.pendingSectionNavigationSummary }
        set { navigationStore.pendingSectionNavigationSummary = newValue }
    }
    public var showsUnsavedChangesNavigationAlert: Bool {
        get { navigationStore.showsUnsavedChangesNavigationAlert }
        set { navigationStore.showsUnsavedChangesNavigationAlert = newValue }
    }
    public var isSidebarVisible: Bool {
        get { navigationStore.isSidebarVisible }
        set { navigationStore.isSidebarVisible = newValue }
    }
    public var isAdvancedSidebarNavigationExpanded: Bool {
        navigationStore.isAdvancedSidebarNavigationExpanded
    }
    private let runtimeLifecycleStore = AppRuntimeLifecycleStore()
    private let panelProblemStore = AppPanelProblemStore()
    public var daemonStatus: DaemonStatus {
        get { runtimeLifecycleStore.daemonStatus }
        set { runtimeLifecycleStore.daemonStatus = newValue }
    }
    public var connectionStatus: ConnectionStatus {
        get { runtimeLifecycleStore.connectionStatus }
        set { runtimeLifecycleStore.connectionStatus = newValue }
    }
    @Published public var mainWindowVisible = true
    @Published public var isCommandPalettePresented = false
    @Published public var commandPaletteSearchQuery = ""
    @Published public var isNotificationCenterPresented = false
    @Published public var notificationCenterSearchQuery = ""
    @Published public var notificationCenterSourceFilter = "all"
    public var notificationItems: [AppNotificationItem] {
        notificationStore.notificationItems
    }
    public var notificationToastItems: [AppNotificationItem] {
        notificationStore.notificationToastItems
    }
    public var successNotificationPulseBySource: [String: Int] {
        notificationStore.successNotificationPulseBySource
    }
    public var pendingHighImpactActionConfirmation: HighImpactActionConfirmation? {
        get { workflowMutationStore.pendingHighImpactActionConfirmation }
        set { workflowMutationStore.pendingHighImpactActionConfirmation = newValue }
    }
    public var activeUndoActionPrompt: UndoActionPrompt? {
        get { workflowMutationStore.activeUndoActionPrompt }
        set { workflowMutationStore.activeUndoActionPrompt = newValue }
    }
    public var panelLatencySamples: [UIPanelLatencySample] {
        panelLatencyStore.panelLatencySamples
    }
    public var panelLatencyLatestBySectionID: [String: UIPanelLatencySample] {
        panelLatencyStore.panelLatencyLatestBySectionID
    }
    public var panelLatencyStatusMessage: String {
        panelLatencyStore.panelLatencyStatusMessage
    }
    public var daemonCanStart: Bool {
        get { runtimeLifecycleStore.daemonCanStart }
        set { runtimeLifecycleStore.daemonCanStart = newValue }
    }
    public var daemonCanStop: Bool {
        get { runtimeLifecycleStore.daemonCanStop }
        set { runtimeLifecycleStore.daemonCanStop = newValue }
    }
    public var daemonCanRestart: Bool {
        get { runtimeLifecycleStore.daemonCanRestart }
        set { runtimeLifecycleStore.daemonCanRestart = newValue }
    }
    public var daemonCanInstall: Bool {
        get { runtimeLifecycleStore.daemonCanInstall }
        set { runtimeLifecycleStore.daemonCanInstall = newValue }
    }
    public var daemonCanUninstall: Bool {
        get { runtimeLifecycleStore.daemonCanUninstall }
        set { runtimeLifecycleStore.daemonCanUninstall = newValue }
    }
    public var daemonCanRepair: Bool {
        get { runtimeLifecycleStore.daemonCanRepair }
        set { runtimeLifecycleStore.daemonCanRepair = newValue }
    }
    public var daemonCanInstallFromBundle: Bool {
        daemonInstallFromBundleDisabledReason == nil
    }
    public var daemonCanRepairFromBundle: Bool {
        daemonRepairFromBundleDisabledReason == nil
    }
    public var daemonInstallFromBundleDisabledReason: String? {
        daemonLocalSetupActionDisabledReason(for: "install")
    }
    public var daemonRepairFromBundleDisabledReason: String? {
        daemonLocalSetupActionDisabledReason(for: "repair")
    }
    public var daemonNeedsInstall: Bool {
        get { runtimeLifecycleStore.daemonNeedsInstall }
        set { runtimeLifecycleStore.daemonNeedsInstall = newValue }
    }
    public var daemonNeedsRepair: Bool {
        get { runtimeLifecycleStore.daemonNeedsRepair }
        set { runtimeLifecycleStore.daemonNeedsRepair = newValue }
    }
    public var daemonControlOperationAction: String {
        get { runtimeLifecycleStore.daemonControlOperationAction }
        set { runtimeLifecycleStore.daemonControlOperationAction = newValue }
    }
    public var daemonControlOperationState: String {
        get { runtimeLifecycleStore.daemonControlOperationState }
        set { runtimeLifecycleStore.daemonControlOperationState = newValue }
    }
    public var daemonStatusDetail: String {
        get { runtimeLifecycleStore.daemonStatusDetail }
        set { runtimeLifecycleStore.daemonStatusDetail = newValue }
    }
    public var isDaemonLifecycleLoading: Bool {
        get { runtimeLifecycleStore.isDaemonLifecycleLoading }
        set { runtimeLifecycleStore.isDaemonLifecycleLoading = newValue }
    }
    public var isDaemonControlInFlight: Bool {
        get { runtimeLifecycleStore.isDaemonControlInFlight }
        set { runtimeLifecycleStore.isDaemonControlInFlight = newValue }
    }
    public var hasLoadedDaemonStatus: Bool {
        get { runtimeLifecycleStore.hasLoadedDaemonStatus }
        set { runtimeLifecycleStore.hasLoadedDaemonStatus = newValue }
    }
    public var daemonWorkerSummary: DaemonLifecycleWorkerSummary {
        get { runtimeLifecycleStore.daemonWorkerSummary }
        set { runtimeLifecycleStore.daemonWorkerSummary = newValue }
    }
    public var daemonDatabaseReady: Bool {
        get { runtimeLifecycleStore.daemonDatabaseReady }
        set { runtimeLifecycleStore.daemonDatabaseReady = newValue }
    }
    public var daemonSetupState: String {
        get { runtimeLifecycleStore.daemonSetupState }
        set { runtimeLifecycleStore.daemonSetupState = newValue }
    }
    public var daemonRepairHint: String {
        get { runtimeLifecycleStore.daemonRepairHint }
        set { runtimeLifecycleStore.daemonRepairHint = newValue }
    }
    public var daemonLifecycleOverallState: String {
        get { runtimeLifecycleStore.daemonLifecycleOverallState }
        set { runtimeLifecycleStore.daemonLifecycleOverallState = newValue }
    }
    public var daemonCoreRuntimeState: String {
        get { runtimeLifecycleStore.daemonCoreRuntimeState }
        set { runtimeLifecycleStore.daemonCoreRuntimeState = newValue }
    }
    public var daemonPluginRuntimeState: String {
        get { runtimeLifecycleStore.daemonPluginRuntimeState }
        set { runtimeLifecycleStore.daemonPluginRuntimeState = newValue }
    }
    public var daemonLifecycleBlocking: Bool {
        get { runtimeLifecycleStore.daemonLifecycleBlocking }
        set { runtimeLifecycleStore.daemonLifecycleBlocking = newValue }
    }
    public var daemonControlAuthState: DaemonControlAuthState {
        get { runtimeLifecycleStore.daemonControlAuthState }
        set { runtimeLifecycleStore.daemonControlAuthState = newValue }
    }
    public var daemonControlAuthSource: String {
        get { runtimeLifecycleStore.daemonControlAuthSource }
        set { runtimeLifecycleStore.daemonControlAuthSource = newValue }
    }
    public var daemonControlAuthRemediationHints: [String] {
        get { runtimeLifecycleStore.daemonControlAuthRemediationHints }
        set { runtimeLifecycleStore.daemonControlAuthRemediationHints = newValue }
    }
    @Published public var daemonPluginLifecycleHistoryFilterPluginID = ""
    @Published public var daemonPluginLifecycleHistoryFilterKind = RuntimePluginLifecycleProjection.defaultFilterSelection
    @Published public var daemonPluginLifecycleHistoryFilterState = RuntimePluginLifecycleProjection.defaultFilterSelection
    @Published public var daemonPluginLifecycleHistoryFilterEventType = RuntimePluginLifecycleProjection.defaultFilterSelection
    @Published public var daemonPluginLifecycleHistoryLimit = RuntimePluginLifecycleProjection.defaultHistoryLimit
    @Published public var isDaemonPluginLifecycleHistoryLoading = false
    @Published public var daemonPluginLifecycleHistoryStatusMessage: String? = "Runtime plugin lifecycle history has not been queried yet."
    @Published public var daemonPluginLifecycleHistoryItems: [RuntimePluginLifecycleEventItem] = []
    @Published public var daemonPluginLifecycleTrendItems: [RuntimePluginLifecycleTrendItem] = []
    @Published public var daemonPluginLifecycleHistoryHasMore = false
    @Published public var localDevTokenInput = ""
    @Published public var localDevTokenConfigured = false
    @Published public var localDevTokenLastUpdated = "Not configured"
    @Published public var isLocalDevAuthBootstrapInFlight = false
    @Published public var localDevAuthBootstrapStatusMessage: String?
        = "Use Copy Command or Run Bootstrap to configure assistant access."
    public private(set) var workspaceID: String {
        get { identityContextStore.workspaceID }
        set { identityContextStore.workspaceID = newValue }
    }
    public var principalOptions: [String] {
        get { identityContextStore.principalOptions }
        set { identityContextStore.principalOptions = newValue }
    }
    public var selectedPrincipal: String {
        get { identityContextStore.selectedPrincipal }
        set { identityContextStore.selectedPrincipal = newValue }
    }
    public var isPrincipalOptionsLoading: Bool {
        get { identityContextStore.isPrincipalOptionsLoading }
        set { identityContextStore.isPrincipalOptionsLoading = newValue }
    }
    @Published public private(set) var hasCompletedFirstRunOnboarding = false
    public var principalStatusMessage: String? {
        get { identityContextStore.principalStatusMessage }
        set { identityContextStore.principalStatusMessage = newValue }
    }
    public var isIdentityDirectoryLoading: Bool {
        get { identityContextStore.isIdentityDirectoryLoading }
        set { identityContextStore.isIdentityDirectoryLoading = newValue }
    }
    public var identityStatusMessage: String? {
        get { identityContextStore.identityStatusMessage }
        set { identityContextStore.identityStatusMessage = newValue }
    }
    public var identityWorkspaceItems: [IdentityWorkspaceItem] {
        get { identityContextStore.identityWorkspaceItems }
        set { identityContextStore.identityWorkspaceItems = newValue }
    }
    public var identityPrincipalItems: [IdentityPrincipalItem] {
        get { identityContextStore.identityPrincipalItems }
        set { identityContextStore.identityPrincipalItems = newValue }
    }
    public var identityActiveContext: IdentityActiveContextItem? {
        get { identityContextStore.identityActiveContext }
        set { identityContextStore.identityActiveContext = newValue }
    }
    public var identityDeviceUserIDFilter: String {
        get { identityContextStore.identityDeviceUserIDFilter }
        set { identityContextStore.identityDeviceUserIDFilter = newValue }
    }
    public var identityDeviceTypeFilter: String {
        get { identityContextStore.identityDeviceTypeFilter }
        set { identityContextStore.identityDeviceTypeFilter = newValue }
    }
    public var identityDevicePlatformFilter: String {
        get { identityContextStore.identityDevicePlatformFilter }
        set { identityContextStore.identityDevicePlatformFilter = newValue }
    }
    public var identityDeviceLimit: Int {
        get { identityContextStore.identityDeviceLimit }
        set { identityContextStore.identityDeviceLimit = newValue }
    }
    public var isIdentityDeviceInventoryLoading: Bool {
        get { identityContextStore.isIdentityDeviceInventoryLoading }
        set { identityContextStore.isIdentityDeviceInventoryLoading = newValue }
    }
    public var identityDeviceInventoryStatusMessage: String? {
        get { identityContextStore.identityDeviceInventoryStatusMessage }
        set { identityContextStore.identityDeviceInventoryStatusMessage = newValue }
    }
    public var identityDeviceItems: [IdentityDeviceItem] {
        get { identityContextStore.identityDeviceItems }
        set { identityContextStore.identityDeviceItems = newValue }
    }
    public var identityDeviceInventoryHasMore: Bool {
        get { identityContextStore.identityDeviceInventoryHasMore }
        set { identityContextStore.identityDeviceInventoryHasMore = newValue }
    }
    public var identitySessionDeviceIDFilter: String {
        get { identityContextStore.identitySessionDeviceIDFilter }
        set { identityContextStore.identitySessionDeviceIDFilter = newValue }
    }
    public var identitySessionUserIDFilter: String {
        get { identityContextStore.identitySessionUserIDFilter }
        set { identityContextStore.identitySessionUserIDFilter = newValue }
    }
    public var identitySessionHealthFilter: String {
        get { identityContextStore.identitySessionHealthFilter }
        set { identityContextStore.identitySessionHealthFilter = newValue }
    }
    public var identitySessionLimit: Int {
        get { identityContextStore.identitySessionLimit }
        set { identityContextStore.identitySessionLimit = newValue }
    }
    public var isIdentitySessionInventoryLoading: Bool {
        get { identityContextStore.isIdentitySessionInventoryLoading }
        set { identityContextStore.isIdentitySessionInventoryLoading = newValue }
    }
    public var identitySessionInventoryStatusMessage: String? {
        get { identityContextStore.identitySessionInventoryStatusMessage }
        set { identityContextStore.identitySessionInventoryStatusMessage = newValue }
    }
    public var identitySessionItems: [IdentitySessionItem] {
        get { identityContextStore.identitySessionItems }
        set { identityContextStore.identitySessionItems = newValue }
    }
    public var identitySessionInventoryHasMore: Bool {
        get { identityContextStore.identitySessionInventoryHasMore }
        set { identityContextStore.identitySessionInventoryHasMore = newValue }
    }
    public var identitySessionActionStatusByID: [String: String] {
        get { identityContextStore.identitySessionActionStatusByID }
        set { identityContextStore.identitySessionActionStatusByID = newValue }
    }
    public var identitySessionRevokeInFlightIDs: Set<String> {
        get { identityContextStore.identitySessionRevokeInFlightIDs }
        set { identityContextStore.identitySessionRevokeInFlightIDs = newValue }
    }
    @Published public var delegationRules: [DelegationRuleItem] = []
    @Published public var isDelegationLoading = false
    @Published public var isDelegationGrantInFlight = false
    @Published public var delegationRevokeInFlightRuleIDs: Set<String> = []
    @Published public var delegationStatusMessage: String? = "Delegation rules have not been queried yet."
    @Published public var delegationActionStatusByRuleID: [String: String] = [:]
    @Published public var chatPersonaScopeType: ChatPersonaScopeType = .workspace
    @Published public var chatPersonaScopePrincipalActorID = "default"
    @Published public var chatPersonaScopeChannelID = "app"
    @Published public var chatPersonaStylePromptDraft = ""
    @Published public var chatPersonaGuardrailsDraft = ""
    @Published public var chatPersonaPolicyStatusMessage: String?
        = "Chat persona policy has not been queried yet."
    @Published public var chatPersonaPolicyItem: ChatPersonaPolicyItem? = nil
    @Published public var isChatPersonaPolicyLoading = false
    @Published public var isChatPersonaPolicySaveInFlight = false
    @Published public private(set) var chatPersonaHasLoadedPolicy = false
    @Published public var capabilityGrantActorFilter = ""
    @Published public var capabilityGrantKeyFilter = ""
    @Published public var capabilityGrantStatusFilter = "all"
    @Published public var capabilityGrantLimit = 25
    @Published public var isCapabilityGrantInventoryLoading = false
    @Published public var capabilityGrantStatusMessage: String? = "Capability grants have not been queried yet."
    @Published public var capabilityGrantItems: [CapabilityGrantItem] = []
    @Published public var capabilityGrantInventoryHasMore = false
    @Published public var isCapabilityGrantMutationInFlight = false
    @Published public var capabilityGrantMutationStatusMessage: String? = "No capability-grant mutation submitted."
    @Published public var capabilityGrantActionStatusByID: [String: String] = [:]
    @Published public var capabilityGrantRevokeInFlightIDs: Set<String> = []
    @Published public var webhookReceiptProviderFilter = ""
    @Published public var webhookReceiptProviderEventIDFilter = ""
    @Published public var webhookReceiptProviderEventQueryFilter = ""
    @Published public var webhookReceiptEventIDFilter = ""
    @Published public var webhookReceiptLimit = 25
    @Published public var isWebhookReceiptsLoading = false
    @Published public var webhookReceiptsStatusMessage: String? = "Webhook trust receipts have not been queried yet."
    @Published public var webhookReceiptItems: [WebhookTrustReceiptItem] = []
    @Published public var webhookReceiptsHasMore = false
    @Published public var ingestReceiptSourceFilter = ""
    @Published public var ingestReceiptSourceScopeFilter = ""
    @Published public var ingestReceiptSourceEventIDFilter = ""
    @Published public var ingestReceiptSourceEventQueryFilter = ""
    @Published public var ingestReceiptTrustStateFilter = "all"
    @Published public var ingestReceiptEventIDFilter = ""
    @Published public var ingestReceiptLimit = 25
    @Published public var isIngestReceiptsLoading = false
    @Published public var ingestReceiptsStatusMessage: String? = "Ingest trust receipts have not been queried yet."
    @Published public var ingestReceiptItems: [IngestTrustReceiptItem] = []
    @Published public var ingestReceiptsHasMore = false
    @Published public var retentionTraceDays = 7
    @Published public var retentionTranscriptDays = 7
    @Published public var retentionMemoryDays = 7
    @Published public var retentionTokenThreshold = 1000
    @Published public var retentionStaleAfterHours = 168
    @Published public var retentionCompactionLimit = 500
    @Published public var retentionCompactionApply = false
    @Published public var isRetentionActionInFlight = false
    @Published public var retentionStatusMessage: String? = "No retention action run yet."
    @Published public var contextTaskClass = "chat"
    @Published public var contextSamplesLimit = 20
    @Published public var isContextActionInFlight = false
    @Published public var contextStatusMessage: String? = "No context action run yet."
    @Published public var contextMemoryOwnerActorFilter = ""
    @Published public var contextMemoryScopeTypeFilter = ""
    @Published public var contextMemoryStatusFilter = "all"
    @Published public var contextMemorySourceTypeFilter = ""
    @Published public var contextMemorySourceRefQuery = ""
    @Published public var contextMemoryLimit = 25
    @Published public var isContextMemoryInventoryLoading = false
    @Published public var contextMemoryInventoryStatusMessage: String? = "Memory inventory has not been queried yet."
    @Published public var contextMemoryInventoryItems: [MemoryInventoryItem] = []
    @Published public var contextMemoryInventoryHasMore = false
    @Published public var contextMemoryCandidatesOwnerActorFilter = ""
    @Published public var contextMemoryCandidatesStatusFilter = "all"
    @Published public var contextMemoryCandidatesLimit = 25
    @Published public var isContextMemoryCandidatesLoading = false
    @Published public var contextMemoryCandidatesStatusMessage: String? = "Compaction candidates have not been queried yet."
    @Published public var contextMemoryCandidateItems: [MemoryCompactionCandidateItem] = []
    @Published public var contextMemoryCandidatesHasMore = false
    @Published public var contextRetrievalOwnerActorFilter = ""
    @Published public var contextRetrievalSourceURIQuery = ""
    @Published public var contextRetrievalDocumentsLimit = 25
    @Published public var isContextRetrievalDocumentsLoading = false
    @Published public var contextRetrievalDocumentsStatusMessage: String? = "Retrieval documents have not been queried yet."
    @Published public var contextRetrievalDocumentItems: [RetrievalDocumentItem] = []
    @Published public var contextRetrievalDocumentsHasMore = false
    @Published public var selectedContextRetrievalDocumentID = ""
    @Published public var contextRetrievalChunkTextQuery = ""
    @Published public var contextRetrievalChunksLimit = 25
    @Published public var isContextRetrievalChunksLoading = false
    @Published public var contextRetrievalChunksStatusMessage: String? = "Select a document to inspect retrieval chunks."
    @Published public var contextRetrievalChunkItems: [RetrievalChunkItem] = []
    @Published public var contextRetrievalChunksHasMore = false
    @Published public var chatDraft = ""
    @Published public var chatTimelineItems: [ChatTimelineItem] = [] {
        didSet {
            chatTimelineStore.synchronizeTimeline(chatTimelineItems)
        }
    }
    @Published public var isChatStreaming = false
    @Published public var chatStatusMessage: String? = "Checking assistant connection." {
        didSet {
            recordStatusNotification(
                source: "chat",
                oldValue: oldValue,
                newValue: chatStatusMessage
            )
        }
    }
    @Published public var chatProgressDetail: String? = nil
    @Published public var chatRouteRemediationMessage: String? = nil
    @Published public var chatFailureRemediationMessage: String? = nil
    @Published public var chatLastFailedDraft: String? = nil
    @Published public var isChatFixAndContinueInFlight = false
    @Published public var chatFixAndContinueStatusMessage: String? = nil
    @Published public var chatTimelineActionStatusByItemID: [String: String] = [:]
    @Published public var chatTimelineActionInFlightItemIDs: Set<String> = []
    @Published public var isChatInterruptInFlight = false
    @Published public var isChatRealtimeRetryInFlight = false
    @Published public var chatActiveCorrelationID: String? = nil
    @Published public var chatLatestTurnTraceability: ChatTaskRunTraceabilityItem? = nil
    @Published public var isChatExplainabilityInFlight = false
    @Published public var chatExplainabilityStatusMessage: String? = "No chat explainability loaded yet."
    @Published public var chatExplainabilityErrorMessage: String? = nil
    @Published public var chatLatestTurnExplainability: ChatTurnExplainabilityItem? = nil
    public var isInspectLoading: Bool {
        get { inspectStore.isInspectLoading }
        set { inspectStore.isInspectLoading = newValue }
    }
    public var hasLoadedInspectLogs: Bool {
        get { inspectStore.hasLoadedInspectLogs }
        set { inspectStore.hasLoadedInspectLogs = newValue }
    }
    public var inspectStatusMessage: String? {
        get { inspectStore.inspectStatusMessage }
        set { inspectStore.inspectStatusMessage = newValue }
    }
    public var isInspectLiveTailEnabled: Bool {
        get { inspectStore.isInspectLiveTailEnabled }
        set { inspectStore.isInspectLiveTailEnabled = newValue }
    }
    public var inspectFocusedRunID: String? {
        get { inspectStore.inspectFocusedRunID }
        set { inspectStore.inspectFocusedRunID = newValue }
    }
    public var inspectSearchSeed: String? {
        get { inspectStore.inspectSearchSeed }
        set { inspectStore.inspectSearchSeed = newValue }
    }
    public var inspectLogs: [InspectLogItem] {
        get { inspectStore.inspectLogs }
        set { inspectStore.inspectLogs = newValue }
    }
    @Published public var isCommunicationsLoading = false
    @Published public var hasLoadedCommunicationsInbox = false
    @Published public var communicationsStatusMessage: String? = "Checking communications inbox." {
        didSet {
            recordStatusNotification(
                source: "communications",
                oldValue: oldValue,
                newValue: communicationsStatusMessage
            )
        }
    }
    @Published public var communicationThreads: [CommunicationThreadItem] = []
    @Published public var communicationEvents: [CommunicationEventItem] = []
    @Published public var communicationCallSessions: [CommunicationCallSessionItem] = []
    @Published public var communicationThreadsHasMore = false
    @Published public var communicationEventsHasMore = false
    @Published public var communicationCallSessionsHasMore = false
    @Published public var isCommunicationContinuityLoading = false
    @Published public var communicationContinuityStatusMessage: String?
        = "Waiting for conversation continuity."
    @Published public var communicationContinuityItems: [CommunicationContinuityItem] = []
    @Published public var communicationContinuityHasMore = false
    @Published public var isCommunicationAttemptsLoading = false
    @Published public var communicationAttemptsStatusMessage: String? = "Select a conversation to load delivery attempts."
    @Published public var communicationDeliveryAttempts: [CommunicationDeliveryAttemptItem] = []
    @Published public var communicationDeliveryAttemptsHasMore = false
    @Published public var isCommunicationSendInFlight = false
    @Published public var communicationSendStatusMessage: String? = "No outbound communication sent yet."
    @Published public var latestCommunicationSendReceipt: CommunicationSendReceiptItem? = nil
    @Published public var isChannelsLoading = false
    @Published public var hasLoadedChannelStatus = false
    @Published public var channelsStatusMessage: String? = "Waiting for channel status." {
        didSet {
            recordStatusNotification(
                source: "channels",
                oldValue: oldValue,
                newValue: channelsStatusMessage
            )
        }
    }
    @Published public var channelCards: [ChannelCardItem] = []
    @Published public var isChannelConnectorMappingsLoading = false
    public var channelConnectorMappingFallbackPolicy: String {
        get { connectionConfigStore.channelConnectorMappingFallbackPolicy }
        set { connectionConfigStore.channelConnectorMappingFallbackPolicy = newValue }
    }
    public var channelConnectorMappingsByChannelID: [String: [ChannelConnectorMappingItem]] {
        get { connectionConfigStore.channelConnectorMappingsByChannelID }
        set { connectionConfigStore.channelConnectorMappingsByChannelID = newValue }
    }
    public var channelConnectorMappingDraftByChannelID: [String: [ChannelConnectorMappingItem]] {
        get { connectionConfigStore.channelConnectorMappingDraftByChannelID }
        set { connectionConfigStore.channelConnectorMappingDraftByChannelID = newValue }
    }
    public var channelConnectorMappingActionStatusByChannelID: [String: String] {
        get { connectionConfigStore.channelConnectorMappingActionStatusByChannelID }
        set { connectionConfigStore.channelConnectorMappingActionStatusByChannelID = newValue }
    }
    public var channelConnectorMappingSaveInFlightChannelIDs: Set<String> {
        get { connectionConfigStore.channelConnectorMappingSaveInFlightChannelIDs }
        set { connectionConfigStore.channelConnectorMappingSaveInFlightChannelIDs = newValue }
    }
    public var channelConfigDraftByID: [String: [String: String]] {
        get { connectionConfigStore.channelConfigDraftByID }
        set { connectionConfigStore.channelConfigDraftByID = newValue }
    }
    public var channelConfigActionStatusByID: [String: String] {
        get { connectionConfigStore.channelConfigActionStatusByID }
        set { connectionConfigStore.channelConfigActionStatusByID = newValue }
    }
    public var channelConfigSaveInFlightIDs: Set<String> {
        get { connectionConfigStore.channelConfigSaveInFlightIDs }
        set { connectionConfigStore.channelConfigSaveInFlightIDs = newValue }
    }
    public var channelTestInFlightIDs: Set<String> {
        get { connectionConfigStore.channelTestInFlightIDs }
        set { connectionConfigStore.channelTestInFlightIDs = newValue }
    }
    public var channelLastTestResultByID: [String: ConfigurationTestResultItem] {
        get { connectionConfigStore.channelLastTestResultByID }
        set { connectionConfigStore.channelLastTestResultByID = newValue }
    }
    public var channelDeliveryPoliciesByChannelID: [String: [ChannelDeliveryPolicyItem]] {
        get { connectionConfigStore.channelDeliveryPoliciesByChannelID }
        set { connectionConfigStore.channelDeliveryPoliciesByChannelID = newValue }
    }
    public var channelDeliveryPolicyDraftByID: [String: ChannelDeliveryPolicyDraft] {
        get { connectionConfigStore.channelDeliveryPolicyDraftByID }
        set { connectionConfigStore.channelDeliveryPolicyDraftByID = newValue }
    }
    public var channelDeliveryPolicyActionStatusByID: [String: String] {
        get { connectionConfigStore.channelDeliveryPolicyActionStatusByID }
        set { connectionConfigStore.channelDeliveryPolicyActionStatusByID = newValue }
    }
    public var channelDeliveryPolicySaveInFlightIDs: Set<String> {
        get { connectionConfigStore.channelDeliveryPolicySaveInFlightIDs }
        set { connectionConfigStore.channelDeliveryPolicySaveInFlightIDs = newValue }
    }
    @Published public var isConnectorsLoading = false
    @Published public var hasLoadedConnectorStatus = false
    @Published public var connectorsStatusMessage: String? = "Waiting for connector status." {
        didSet {
            recordStatusNotification(
                source: "connectors",
                oldValue: oldValue,
                newValue: connectorsStatusMessage
            )
        }
    }
    @Published public var connectorCards: [ConnectorCardItem] = []
    public var connectorConfigDraftByID: [String: [String: String]] {
        get { connectionConfigStore.connectorConfigDraftByID }
        set { connectionConfigStore.connectorConfigDraftByID = newValue }
    }
    public var connectorConfigActionStatusByID: [String: String] {
        get { connectionConfigStore.connectorConfigActionStatusByID }
        set { connectionConfigStore.connectorConfigActionStatusByID = newValue }
    }
    public var connectorConfigSaveInFlightIDs: Set<String> {
        get { connectionConfigStore.connectorConfigSaveInFlightIDs }
        set { connectionConfigStore.connectorConfigSaveInFlightIDs = newValue }
    }
    public var connectorTestInFlightIDs: Set<String> {
        get { connectionConfigStore.connectorTestInFlightIDs }
        set { connectionConfigStore.connectorTestInFlightIDs = newValue }
    }
    public var connectorLastTestResultByID: [String: ConfigurationTestResultItem] {
        get { connectionConfigStore.connectorLastTestResultByID }
        set { connectionConfigStore.connectorLastTestResultByID = newValue }
    }
    public var connectorPermissionActionStatusByID: [String: String] {
        get { connectionConfigStore.connectorPermissionActionStatusByID }
        set { connectionConfigStore.connectorPermissionActionStatusByID = newValue }
    }
    public var connectorPermissionRequestInFlightIDs: Set<String> {
        get { connectionConfigStore.connectorPermissionRequestInFlightIDs }
        set { connectionConfigStore.connectorPermissionRequestInFlightIDs = newValue }
    }
    @Published public var isProviderStatusLoading = false
    @Published public var hasLoadedProviderStatus = false
    @Published public var providerStatusMessage: String? = "Waiting for provider inventory." {
        didSet {
            recordStatusNotification(
                source: "models",
                oldValue: oldValue,
                newValue: providerStatusMessage
            )
        }
    }
    public var providerReadinessItems: [ProviderReadinessItem] {
        get { modelsRouteStore.providerReadinessItems }
        set { modelsRouteStore.providerReadinessItems = newValue }
    }
    public var providerEndpointDraftByID: [String: String] {
        get { modelsRouteStore.providerEndpointDraftByID }
        set { modelsRouteStore.providerEndpointDraftByID = newValue }
    }
    public var providerAPIKeySecretNameDraftByID: [String: String] {
        get { modelsRouteStore.providerAPIKeySecretNameDraftByID }
        set { modelsRouteStore.providerAPIKeySecretNameDraftByID = newValue }
    }
    public var providerAPIKeySecretValueDraftByID: [String: String] {
        get { modelsRouteStore.providerAPIKeySecretValueDraftByID }
        set { modelsRouteStore.providerAPIKeySecretValueDraftByID = newValue }
    }
    public var providerSetupStatusByID: [String: String] {
        get { modelsRouteStore.providerSetupStatusByID }
        set { modelsRouteStore.providerSetupStatusByID = newValue }
    }
    public var providerSetupInFlightIDs: Set<String> {
        get { modelsRouteStore.providerSetupInFlightIDs }
        set { modelsRouteStore.providerSetupInFlightIDs = newValue }
    }
    public var providerCheckInFlightIDs: Set<String> {
        get { modelsRouteStore.providerCheckInFlightIDs }
        set { modelsRouteStore.providerCheckInFlightIDs = newValue }
    }
    @Published public var modelRouteSummary: ModelRouteSummary?
    @Published public var modelRouteStatusMessage: String? = "Waiting for model route resolution." {
        didSet {
            recordStatusNotification(
                source: "models",
                oldValue: oldValue,
                newValue: modelRouteStatusMessage
            )
        }
    }
    public var modelCatalogStatusMessage: String? {
        get { modelsRouteStore.modelCatalogStatusMessage }
        set { modelsRouteStore.modelCatalogStatusMessage = newValue }
    }
    public var modelCatalogItems: [ModelCatalogEntryItem] {
        get { modelsRouteStore.modelCatalogItems }
        set { modelsRouteStore.modelCatalogItems = newValue }
    }
    public var modelPolicyItems: [ModelPolicyItem] {
        get { modelsRouteStore.modelPolicyItems }
        set { modelsRouteStore.modelPolicyItems = newValue }
    }
    public var modelMutationStatusByID: [String: String] {
        get { modelsRouteStore.modelMutationStatusByID }
        set { modelsRouteStore.modelMutationStatusByID = newValue }
    }
    public var modelMutationInFlightIDs: Set<String> {
        get { modelsRouteStore.modelMutationInFlightIDs }
        set { modelsRouteStore.modelMutationInFlightIDs = newValue }
    }
    public var modelCatalogManagementStatusByProviderID: [String: String] {
        get { modelsRouteStore.modelCatalogManagementStatusByProviderID }
        set { modelsRouteStore.modelCatalogManagementStatusByProviderID = newValue }
    }
    public var modelCatalogDiscoverInFlightProviderIDs: Set<String> {
        get { modelsRouteStore.modelCatalogDiscoverInFlightProviderIDs }
        set { modelsRouteStore.modelCatalogDiscoverInFlightProviderIDs = newValue }
    }
    public var modelCatalogManageInFlightProviderIDs: Set<String> {
        get { modelsRouteStore.modelCatalogManageInFlightProviderIDs }
        set { modelsRouteStore.modelCatalogManageInFlightProviderIDs = newValue }
    }
    public var discoveredModelsByProviderID: [String: [DiscoveredModelEntryItem]] {
        get { modelsRouteStore.discoveredModelsByProviderID }
        set { modelsRouteStore.discoveredModelsByProviderID = newValue }
    }
    public var modelManualAddDraftByProviderID: [String: String] {
        get { modelsRouteStore.modelManualAddDraftByProviderID }
        set { modelsRouteStore.modelManualAddDraftByProviderID = newValue }
    }
    public var isModelRoutePolicySaveInFlight: Bool {
        get { modelsRouteStore.isModelRoutePolicySaveInFlight }
        set { modelsRouteStore.isModelRoutePolicySaveInFlight = newValue }
    }
    public var modelRoutePolicySaveStatusMessage: String? {
        get { modelsRouteStore.modelRoutePolicySaveStatusMessage }
        set { modelsRouteStore.modelRoutePolicySaveStatusMessage = newValue }
    }
    public var modelRouteSimulationTaskClass: String {
        get { modelsRouteStore.modelRouteSimulationTaskClass }
        set { modelsRouteStore.modelRouteSimulationTaskClass = newValue }
    }
    public var modelRouteSimulationPrincipalActorID: String {
        get { modelsRouteStore.modelRouteSimulationPrincipalActorID }
        set { modelsRouteStore.modelRouteSimulationPrincipalActorID = newValue }
    }
    public var isModelRouteSimulationInFlight: Bool {
        get { modelsRouteStore.isModelRouteSimulationInFlight }
        set { modelsRouteStore.isModelRouteSimulationInFlight = newValue }
    }
    public var isModelRouteExplainInFlight: Bool {
        get { modelsRouteStore.isModelRouteExplainInFlight }
        set { modelsRouteStore.isModelRouteExplainInFlight = newValue }
    }
    public var modelRouteSimulationStatusMessage: String? {
        get { modelsRouteStore.modelRouteSimulationStatusMessage }
        set { modelsRouteStore.modelRouteSimulationStatusMessage = newValue }
    }
    public var modelRouteExplainStatusMessage: String? {
        get { modelsRouteStore.modelRouteExplainStatusMessage }
        set { modelsRouteStore.modelRouteExplainStatusMessage = newValue }
    }
    public var modelRouteSimulationResult: ModelRouteSimulationResultItem? {
        get { modelsRouteStore.modelRouteSimulationResult }
        set { modelsRouteStore.modelRouteSimulationResult = newValue }
    }
    public var modelRouteExplainResult: ModelRouteExplainResultItem? {
        get { modelsRouteStore.modelRouteExplainResult }
        set { modelsRouteStore.modelRouteExplainResult = newValue }
    }
    @Published public var isAutomationLoading = false
    @Published public var hasLoadedAutomationPanelData = false
    @Published public var automationStatusMessage: String? = "Waiting for automation trigger inventory." {
        didSet {
            recordStatusNotification(
                source: "automation",
                oldValue: oldValue,
                newValue: automationStatusMessage
            )
        }
    }
    @Published public var automationTriggers: [AutomationTriggerItem] = []
    @Published public var automationManagementStatusMessage: String? = "No create/edit action run yet." {
        didSet {
            recordStatusNotification(
                source: "automation",
                oldValue: oldValue,
                newValue: automationManagementStatusMessage
            )
        }
    }
    public var isAutomationCreateInFlight: Bool {
        get { workflowQueueStore.isAutomationCreateInFlight }
        set { workflowQueueStore.isAutomationCreateInFlight = newValue }
    }
    public var automationUpdateInFlightIDs: Set<String> {
        get { workflowQueueStore.automationUpdateInFlightIDs }
        set { workflowQueueStore.automationUpdateInFlightIDs = newValue }
    }
    public var automationDeleteInFlightIDs: Set<String> {
        get { workflowQueueStore.automationDeleteInFlightIDs }
        set { workflowQueueStore.automationDeleteInFlightIDs = newValue }
    }
    public var automationActionStatusByID: [String: String] {
        get { workflowQueueStore.automationActionStatusByID }
        set { workflowQueueStore.automationActionStatusByID = newValue }
    }
    @Published public var isAutomationSimulationInFlight = false
    @Published public var automationSimulationStatusMessage: String? = "No simulation run yet."
    @Published public var isAutomationFireHistoryLoading = false
    @Published public var automationFireHistoryStatusMessage: String? = "Waiting for trigger fire history."
    @Published public var automationFireHistoryItems: [AutomationFireHistoryItem] = []
    @Published public var isApprovalsLoading = false
    @Published public var hasLoadedApprovalsInbox = false
    @Published public var approvalsStatusMessage: String? = "Checking approvals inbox." {
        didSet {
            recordStatusNotification(
                source: "approvals",
                oldValue: oldValue,
                newValue: approvalsStatusMessage
            )
        }
    }
    @Published public var approvalsSearchSeed: String? = nil
    @Published public var approvalInboxItems: [ApprovalInboxItem] = []
    public var approvalsActionStatusByID: [String: String] {
        get { workflowQueueStore.approvalsActionStatusByID }
        set { workflowQueueStore.approvalsActionStatusByID = newValue }
    }
    public var approvalDecisionInFlightIDs: Set<String> {
        get { workflowQueueStore.approvalDecisionInFlightIDs }
        set { workflowQueueStore.approvalDecisionInFlightIDs = newValue }
    }
    @Published public var approvalEvidenceByID: [String: ApprovalEvidenceItem] = [:]
    @Published public var approvalEvidenceStatusByID: [String: String] = [:]
    @Published public var approvalEvidenceInFlightIDs: Set<String> = []
    @Published public var isTasksLoading = false
    @Published public var hasLoadedTaskRunList = false
    @Published public var tasksStatusMessage: String? = "Checking tasks and runs." {
        didSet {
            recordStatusNotification(
                source: "tasks",
                oldValue: oldValue,
                newValue: tasksStatusMessage
            )
        }
    }
    @Published public var tasksSearchSeed: String? = nil
    @Published public var taskSubmitDraftSeed: TaskSubmitDraftSeed? = nil
    @Published public var taskRunItems: [TaskRunListRowItem] = []
    @Published public var taskRunDetailStatusMessage: String? = nil
    @Published public var isTaskRunDetailLoading = false
    @Published public var selectedTaskRunDetail: TaskRunDetailItem? = nil
    public var taskRunControlStatusByRunID: [String: String] {
        get { workflowQueueStore.taskRunControlStatusByRunID }
        set { workflowQueueStore.taskRunControlStatusByRunID = newValue }
    }
    public var taskRunControlInFlightRunIDs: Set<String> {
        get { workflowQueueStore.taskRunControlInFlightRunIDs }
        set { workflowQueueStore.taskRunControlInFlightRunIDs = newValue }
    }
    public var isTaskSubmitInFlight: Bool {
        get { workflowQueueStore.isTaskSubmitInFlight }
        set { workflowQueueStore.isTaskSubmitInFlight = newValue }
    }
    public var taskSubmitStatusMessage: String? {
        get { workflowQueueStore.taskSubmitStatusMessage }
        set { workflowQueueStore.taskSubmitStatusMessage = newValue }
    }
    public var latestTaskSubmissionReceipt: TaskSubmissionReceiptItem? {
        get { workflowQueueStore.latestTaskSubmissionReceipt }
        set { workflowQueueStore.latestTaskSubmissionReceipt = newValue }
    }
    
    private let daemonClient = DaemonAPIClient()
    private let daemonBaseURL: URL
    private var daemonAuthToken: String?
    private var workflowMutationStoreCancellable: AnyCancellable?
    private var navigationStoreCancellable: AnyCancellable?
    private var identityContextStoreCancellable: AnyCancellable?
    private var runtimeLifecycleStoreCancellable: AnyCancellable?
    private var panelProblemStoreCancellable: AnyCancellable?
    private var contextRetentionStoreCancellable: AnyCancellable?
    private var commandHistoryStoreCancellable: AnyCancellable?
    private var notificationStoreCancellable: AnyCancellable?
    private var panelLatencyStoreCancellable: AnyCancellable?
    private var connectionConfigStoreCancellable: AnyCancellable?
    private var modelsRouteStoreCancellable: AnyCancellable?
    private var workflowQueueStoreCancellable: AnyCancellable?
    private var communicationsStoreCancellable: AnyCancellable?
    private var inspectStoreCancellable: AnyCancellable?
    private var inspectStatusNotificationCancellable: AnyCancellable?
    private var lastInspectStatusNotificationValue: String?
    private var inspectStreamTask: Task<Void, Never>?
    private let workflowMutationStore = AppWorkflowMutationStore()
    private let identityContextStore = AppIdentityContextStore()
    private let contextRetentionStore: AppContextRetentionStore
    private let commandHistoryStore: AppCommandHistoryStore
    private let notificationStore: AppNotificationCenterStore
    private let panelLatencyStore = AppPanelLatencyStore()
    private let connectionConfigStore = AppConnectionConfigStore()
    private let modelsRouteStore = AppModelsRouteStore()
    private let workflowQueueStore = AppWorkflowQueueStore()
    private let communicationsStore = AppCommunicationsStore()
    private let inspectStore = AppInspectStore()
    private let chatOrchestrationStore = ChatOrchestrationStore()
    private let chatTurnExecutionStore = ChatTurnExecutionStore()
    private let chatTimelineStore = ChatTimelineEventStore()
    private let chatTurnContextStore = ChatTurnContextStore()
    private var pendingChatFixAndContinue: PendingChatFixAndContinue? = nil
    private var lifecyclePollingTask: Task<Void, Never>?
    private var isDaemonLifecycleRequestInFlight = false
    private var isDaemonPluginLifecycleHistoryRequestInFlight = false
    private var isContextMemoryInventoryRequestInFlight = false
    private var isContextMemoryCandidatesRequestInFlight = false
    private var isContextRetrievalDocumentsRequestInFlight = false
    private var isContextRetrievalChunksRequestInFlight = false
    private var isCapabilityGrantInventoryRequestInFlight = false
    private var isCapabilityGrantMutationRequestInFlight = false
    private var isWebhookReceiptsRequestInFlight = false
    private var isIngestReceiptsRequestInFlight = false
    private var isIdentityDeviceInventoryRequestInFlight = false
    private var isIdentitySessionInventoryRequestInFlight = false
    private var isChatPersonaPolicyRequestInFlight = false
    private var isChatPersonaPolicySaveRequestInFlight = false
    private var chatPersonaLoadedStylePrompt = ""
    private var chatPersonaLoadedGuardrails: [String] = []
    private var connectorPermissionStatesByID: [String: ConnectorPermissionState] {
        get { connectionConfigStore.connectorPermissionStatesByID }
        set { connectionConfigStore.connectorPermissionStatesByID = newValue }
    }
    private var channelConfigKindsByID: [String: [String: ConfigurationDraftValueKind]] {
        get { connectionConfigStore.channelConfigKindsByID }
        set { connectionConfigStore.channelConfigKindsByID = newValue }
    }
    private var connectorConfigKindsByID: [String: [String: ConfigurationDraftValueKind]] {
        get { connectionConfigStore.connectorConfigKindsByID }
        set { connectionConfigStore.connectorConfigKindsByID = newValue }
    }
    private var connectorPermissionRefreshPendingIDs: Set<String> {
        get { connectionConfigStore.connectorPermissionRefreshPendingIDs }
        set { connectionConfigStore.connectorPermissionRefreshPendingIDs = newValue }
    }
    private var providerEndpointSourceByID: [String: String] {
        get { modelsRouteStore.providerEndpointSourceByID }
        set { modelsRouteStore.providerEndpointSourceByID = newValue }
    }
    private var providerSecretNameSourceByID: [String: String] {
        get { modelsRouteStore.providerSecretNameSourceByID }
        set { modelsRouteStore.providerSecretNameSourceByID = newValue }
    }
    private var inspectCursorCreatedAt: String? {
        get { inspectStore.inspectCursorCreatedAt }
        set { inspectStore.inspectCursorCreatedAt = newValue }
    }
    private var inspectCursorID: String? {
        get { inspectStore.inspectCursorID }
        set { inspectStore.inspectCursorID = newValue }
    }
    typealias LocalDevAuthBootstrapCommandExecution = (
        exitCode: Int32,
        stdout: String,
        stderr: String
    )
    typealias LocalDevAuthBootstrapCommandRunner = (
        _ args: [String]
    ) async -> LocalDevAuthBootstrapCommandExecution
    typealias LocalDevAuthBootstrapRefreshHandler = (
        _ state: AppShellState
    ) async -> Void
    typealias DaemonLocalServiceInstallRunner = (
        _ action: String,
        _ authToken: String
    ) async throws -> DaemonLocalServiceInstallResult
    typealias DaemonLifecycleControlRunner = (
        _ state: AppShellState,
        _ authToken: String,
        _ action: String,
        _ reason: String
    ) async throws -> DaemonLifecycleControlResponse

    private struct LocalDevTokenSecretReference {
        let service: String
        let account: String
    }

    private static let localDevTokenDefaultsKey = "personalagent.ui.local_dev_token"
    private static let localDevTokenKeychainService = "personalagent.ui.local_dev_token.v1"
    private static let localDevTokenKeychainAccount = "daemon_auth_token"
    private static let onboardingCompleteDefaultsKey = "personalagent.ui.onboarding_complete"
    private static let workspaceDefaultsKey = "personalagent.ui.workspace_id"
    private static let notificationsDefaultsKey = "personalagent.ui.notifications.v1"
    private static let panelFilterContextDefaultsKey = "personalagent.ui.panel_filter_context.v1"
    private static let communicationsTriageDefaultsKey = "personalagent.ui.communications_triage_context.v1"
    private static let workspaceContinuityDefaultsKey = "personalagent.ui.workspace_continuity_context.v1"
    private static let informationDensityModeDefaultsKey = "personalagent.ui.information_density_mode.v1"
    private static let homeFirstSessionProgressDefaultsKey = "personalagent.ui.home_first_session_progress.v1"
    private static let recentAppCommandsDefaultsKey = "personalagent.ui.recent_app_commands.v1"
    private static let uiDefaultsSuiteEnvKey = "PA_UI_DEFAULTS_SUITE"
    private static let defaultDaemonAddress = "http://127.0.0.1:7071"
    private static let distributionTrustSecuritySettingsURLString =
        "x-apple.systempreferences:com.apple.preference.security?General"
    private static let distributionTrustGuidanceChecklist: [String] = [
        "If macOS blocks first launch, Control-click PersonalAgent.app and choose Open.",
        "If launch was already blocked once, open System Settings > Privacy & Security and click Open Anyway for PersonalAgent.",
        "After launch succeeds, run Install or Repair in Configuration > Advanced to bootstrap Personal Agent Daemon.",
        "Approve connector permission prompts so access is attributed to Personal Agent Daemon."
    ]
    private static let defaultWorkspaceID = "ws1"
    private static let notificationHistoryLimit = 250
    private static let recentAppCommandHistoryLimit = 8
    private static let canonicalProviderOrder = ["openai", "anthropic", "google", "ollama"]
    private static let daemonPluginLifecycleKindOptions = ["all", "channel", "connector"]
    private static let daemonPluginLifecycleStateOptions = ["all", "registered", "starting", "running", "restarting", "stopped", "failed"]
    private static let daemonPluginLifecycleEventTypeOptions = [
        "all",
        "PLUGIN_WORKER_STARTED",
        "PLUGIN_HANDSHAKE_ACCEPTED",
        "PLUGIN_HEALTH_TIMEOUT",
        "PLUGIN_WORKER_RESTARTING",
        "PLUGIN_WORKER_EXITED",
        "PLUGIN_WORKER_STOPPED",
        "PLUGIN_WORKER_RESTART_LIMIT",
    ]
    private static let contextMemoryStatusOptions = ["all", "ACTIVE", "DISABLED", "DELETED"]
    private static let contextMemoryCandidateStatusOptions = ["all", "PENDING", "APPLIED", "SKIPPED", "REJECTED"]
    private static let identitySessionHealthOptions = ["all", "active", "expired", "revoked"]
    private static let capabilityGrantStatusOptions = ["all", "ACTIVE", "DISABLED", "REVOKED"]
    private static let receiptTrustStateOptions = ["all", "accepted", "rejected"]
    private static let chatPersonaChannelOptions = ["app", "message", "voice"]
    private static let canonicalLogicalChannelSortOrder: [String: Int] = [
        "app": 0,
        "message": 1,
        "voice": 2
    ]
    private static let providerDefaultEndpoints: [String: String] = [
        "openai": "https://api.openai.com/v1",
        "anthropic": "https://api.anthropic.com/v1",
        "google": "https://generativelanguage.googleapis.com/v1beta",
        "ollama": "http://127.0.0.1:11434"
    ]
    private static let modelRouteReadinessStepIDs = [
        "token",
        "daemon",
        "provider",
        "model_catalog",
        "chat_route"
    ]
    private static let defaultLocalDevAuthBootstrapCommandRunner: LocalDevAuthBootstrapCommandRunner = { args in
        await executeLocalDevAuthBootstrapShellCommand(arguments: ["personal-agent"] + args)
    }
    private static let defaultLocalDevAuthBootstrapRefreshHandler: LocalDevAuthBootstrapRefreshHandler = { state in
        await state.bootstrapFromDaemon()
    }
    private static let defaultDaemonLocalServiceInstallRunner: DaemonLocalServiceInstallRunner = { action, authToken in
        try await Task.detached(priority: .userInitiated) {
            try DaemonLocalServiceInstaller.installOrRepair(
                action: action,
                authToken: authToken
            )
        }.value
    }
    private static let defaultDaemonLifecycleControlRunner: DaemonLifecycleControlRunner = {
        state,
        authToken,
        action,
        reason in
        try await state.daemonClient.lifecycle.daemonLifecycleControl(
            baseURL: state.daemonBaseURL,
            authToken: authToken,
            action: action,
            reason: reason
        )
    }
    static var localDevAuthBootstrapCommandRunner: LocalDevAuthBootstrapCommandRunner = defaultLocalDevAuthBootstrapCommandRunner
    static var localDevAuthBootstrapRefreshHandler: LocalDevAuthBootstrapRefreshHandler = defaultLocalDevAuthBootstrapRefreshHandler
    static var daemonLocalServiceInstallRunner: DaemonLocalServiceInstallRunner = defaultDaemonLocalServiceInstallRunner
    static var daemonLifecycleControlRunner: DaemonLifecycleControlRunner = defaultDaemonLifecycleControlRunner
    private static var localDevTokenSecretReferenceOverride: LocalDevTokenSecretReference?
    private static var userDefaultsStore: UserDefaults {
        let suiteName = ProcessInfo.processInfo.environment[uiDefaultsSuiteEnvKey]?
            .trimmingCharacters(in: .whitespacesAndNewlines)
        if let suiteName, !suiteName.isEmpty, let suiteDefaults = UserDefaults(suiteName: suiteName) {
            return suiteDefaults
        }
        return .standard
    }

    private static func localDevTokenSecretReference() -> LocalDevTokenSecretReference {
        if let localDevTokenSecretReferenceOverride {
            return localDevTokenSecretReferenceOverride
        }
        return LocalDevTokenSecretReference(
            service: localDevTokenKeychainService,
            account: localDevTokenKeychainAccount
        )
    }

    private static func loadPersistedLocalDevToken() -> String? {
        let secretReference = localDevTokenSecretReference()
        if let keychainToken = try? LocalSecretStore.readSecret(
            service: secretReference.service,
            account: secretReference.account
        )?.trimmingCharacters(in: .whitespacesAndNewlines),
            !keychainToken.isEmpty
        {
            userDefaultsStore.removeObject(forKey: localDevTokenDefaultsKey)
            return keychainToken
        }

        guard let legacyDefaultsToken = userDefaultsStore.string(forKey: localDevTokenDefaultsKey)?
            .trimmingCharacters(in: .whitespacesAndNewlines),
            !legacyDefaultsToken.isEmpty
        else {
            return nil
        }

        do {
            try LocalSecretStore.upsertSecret(
                value: legacyDefaultsToken,
                service: secretReference.service,
                account: secretReference.account
            )
        } catch {
            return legacyDefaultsToken
        }

        userDefaultsStore.removeObject(forKey: localDevTokenDefaultsKey)
        return legacyDefaultsToken
    }

    private static func persistLocalDevToken(_ token: String) throws {
        let secretReference = localDevTokenSecretReference()
        try LocalSecretStore.upsertSecret(
            value: token,
            service: secretReference.service,
            account: secretReference.account
        )
        userDefaultsStore.removeObject(forKey: localDevTokenDefaultsKey)
    }

    private static func clearPersistedLocalDevToken() throws {
        let secretReference = localDevTokenSecretReference()
        try LocalSecretStore.deleteSecret(
            service: secretReference.service,
            account: secretReference.account
        )
        userDefaultsStore.removeObject(forKey: localDevTokenDefaultsKey)
    }

    private static func canonicalWorkspaceID(
        _ rawWorkspaceID: String?,
        fallbackToDefault: Bool = true
    ) -> String? {
        guard let trimmedWorkspaceID = rawWorkspaceID?
            .trimmingCharacters(in: .whitespacesAndNewlines),
            !trimmedWorkspaceID.isEmpty
        else {
            return fallbackToDefault ? Self.defaultWorkspaceID : nil
        }
        return trimmedWorkspaceID
    }

    public var workspaceLabel: String { workspaceID }
    public var distributionTrustGuidanceSummary: String {
        "This local/internal build is unsigned and not notarized. First launch may require Gatekeeper override before daemon setup checks can pass."
    }
    public var distributionTrustGuidanceChecklist: [String] {
        Self.distributionTrustGuidanceChecklist
    }
    public var distributionTrustRetryGuidance: String {
        "If prompts were dismissed or permissions were denied, reopen System Settings and then retry setup checks or connector permission actions."
    }
    public var isAdvancedInformationDensityEnabled: Bool {
        informationDensityMode == .advanced
    }
    public func setInformationDensityMode(_ mode: AppInformationDensityMode) {
        guard informationDensityMode != mode else {
            return
        }
        informationDensityMode = mode
        setCurrentWorkspaceInformationDensityMode(mode)
    }
    public var activePrincipalLabel: String { nonEmpty(selectedPrincipal) ?? "default" }
    public var delegationScopeOptions: [String] { ["EXECUTION", "APPROVAL", "ALL"] }
    public var delegationPrincipalOptions: [String] {
        principalOptionsForPrincipalSelection(including: nil)
    }
    public var chatPersonaPrincipalOptions: [String] {
        principalOptionsForPrincipalSelection(including: chatPersonaScopePrincipalActorID)
    }
    public var chatPersonaChannelOptions: [String] {
        Self.chatPersonaChannelOptions
    }
    public var chatPersonaResolvedPrincipalActorID: String? {
        resolvedChatPersonaPrincipalActorID()
    }
    public var chatPersonaResolvedChannelID: String? {
        resolvedChatPersonaChannelID()
    }
    public var chatPersonaScopeSummary: String {
        chatPersonaScopeSummary(
            principalActorID: chatPersonaResolvedPrincipalActorID,
            channelID: chatPersonaResolvedChannelID
        )
    }
    public var chatPersonaResponseShapingChannelID: String { "app" }
    public var chatPersonaResponseShapingProfileID: String {
        responseShapingProfileID(for: chatPersonaResponseShapingChannelID)
    }
    public var chatPersonaNormalizedGuardrails: [String] {
        normalizedChatPersonaGuardrails(chatPersonaGuardrailsDraft)
    }
    public var chatPersonaHasDraftChanges: Bool {
        let normalizedStylePrompt = chatPersonaStylePromptDraft
            .trimmingCharacters(in: .whitespacesAndNewlines)
        let normalizedLoadedStylePrompt = chatPersonaLoadedStylePrompt
            .trimmingCharacters(in: .whitespacesAndNewlines)
        return normalizedStylePrompt != normalizedLoadedStylePrompt
            || chatPersonaNormalizedGuardrails != chatPersonaLoadedGuardrails
    }
    public var chatPersonaSaveDisabledReason: String? {
        if isChatPersonaPolicyLoading {
            return "Wait for persona policy loading to finish."
        }
        if isChatPersonaPolicySaveInFlight {
            return "Persona policy save is already in progress."
        }
        if resolvedAuthToken() == nil {
            return "Set Assistant Access Token before saving persona policy."
        }
        if chatPersonaStylePromptDraft.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
            return "Style prompt is required."
        }
        if !chatPersonaHasDraftChanges {
            return "No unsaved persona policy changes."
        }
        return nil
    }
    public var taskSubmissionPrincipalOptions: [String] {
        principalOptionsForPrincipalSelection(including: nil)
    }
    public var actingAsPrincipalOptions: [String] {
        principalOptionsForPrincipalSelection(including: selectedPrincipal)
    }
    public var selectedActingAsValidationMessage: String? {
        actingAsValidationMessage(for: selectedPrincipal)
    }
    public var identityWorkspaceOptions: [String] {
        var ids = Set(identityWorkspaceItems.map(\.id))
        ids.insert(workspaceID)
        if ids.isEmpty {
            return [workspaceID]
        }
        return ids.sorted()
    }
    public var daemonPluginLifecycleHistoryKindOptions: [String] {
        Self.daemonPluginLifecycleKindOptions
    }
    public var daemonPluginLifecycleHistoryStateOptions: [String] {
        Self.daemonPluginLifecycleStateOptions
    }
    public var daemonPluginLifecycleHistoryEventTypeOptions: [String] {
        Self.daemonPluginLifecycleEventTypeOptions
    }
    public var contextMemoryStatusFilterOptions: [String] {
        Self.contextMemoryStatusOptions
    }
    public var contextMemoryCandidateStatusFilterOptions: [String] {
        Self.contextMemoryCandidateStatusOptions
    }
    public var identitySessionHealthFilterOptions: [String] {
        Self.identitySessionHealthOptions
    }
    public var capabilityGrantStatusFilterOptions: [String] {
        Self.capabilityGrantStatusOptions
    }
    public var receiptTrustStateFilterOptions: [String] {
        Self.receiptTrustStateOptions
    }
    public var notificationSourceOptions: [String] {
        notificationStore.notificationSourceOptions()
    }
    public var unreadNotificationCount: Int {
        notificationStore.unreadNotificationCount()
    }
    public var filteredNotificationItems: [AppNotificationItem] {
        notificationStore.filteredNotificationItems(
            query: notificationCenterSearchQuery,
            sourceFilter: notificationCenterSourceFilter
        )
    }
    public var groupedFilteredNotificationSections: [NotificationInboxSection] {
        let grouped = Dictionary(grouping: filteredNotificationItems) { item in
            notificationInboxIntent(for: item)
        }
        return NotificationInboxIntent.allCases
            .sorted { $0.sortPriority < $1.sortPriority }
            .compactMap { intent in
                guard let items = grouped[intent], !items.isEmpty else {
                    return nil
                }
                return NotificationInboxSection(intent: intent, items: items)
            }
    }

    public func notificationInboxIntent(for item: AppNotificationItem) -> NotificationInboxIntent {
        NotificationInboxRouting.inboxIntent(for: item)
    }

    public func notificationInboxActions(for item: AppNotificationItem) -> [NotificationInboxAction] {
        NotificationInboxRouting.inboxActions(for: item)
    }
    public var chatEmptyStateRemediationActions: [EmptyStateRemediationAction] {
        if daemonControlAuthNeedsRemediation {
            return [
                remediationAction(
                    .openConfiguration,
                    title: "Open Configuration",
                    symbolName: "gearshape",
                    prominent: true
                )
            ]
        }
        if chatRouteRemediationMessage != nil {
            return [
                remediationAction(
                    .openModels,
                    title: "Open Models",
                    symbolName: "cube.box",
                    prominent: true
                ),
                remediationAction(
                    .recheckChatRoute,
                    title: "Check Again",
                    symbolName: "arrow.clockwise"
                )
            ]
        }
        if connectionStatus == .degraded || connectionStatus == .disconnected {
            return [
                remediationAction(
                    .refreshDaemonStatus,
                    title: "Refresh Runtime",
                    symbolName: "arrow.clockwise",
                    prominent: true
                ),
                remediationAction(
                    .openConfiguration,
                    title: "Open Configuration",
                    symbolName: "gearshape"
                )
            ]
        }
        return [
            remediationAction(
                .openModels,
                title: "Open Models",
                symbolName: "cube.box"
            )
        ]
    }
    public var communicationsEmptyStateRemediationActions: [EmptyStateRemediationAction] {
        if daemonControlAuthNeedsRemediation {
            return [
                remediationAction(
                    .openConfiguration,
                    title: "Open Configuration",
                    symbolName: "gearshape",
                    prominent: true
                ),
                remediationAction(
                    .openChannels,
                    title: "Open Channels",
                    symbolName: "point.3.connected.trianglepath.dotted"
                )
            ]
        }
        return [
            remediationAction(
                .refreshCommunications,
                title: "Refresh Inbox",
                symbolName: "arrow.clockwise",
                prominent: true
            ),
            remediationAction(
                .openChannels,
                title: "Open Channels",
                symbolName: "point.3.connected.trianglepath.dotted"
            )
        ]
    }
    public var automationEmptyStateRemediationActions: [EmptyStateRemediationAction] {
        if daemonControlAuthNeedsRemediation {
            return [
                remediationAction(
                    .openConfiguration,
                    title: "Open Configuration",
                    symbolName: "gearshape",
                    prominent: true
                )
            ]
        }
        return [
            remediationAction(
                .refreshAutomation,
                title: "Refresh Automation",
                symbolName: "arrow.clockwise"
            ),
            remediationAction(
                .openChat,
                title: "Open Chat",
                symbolName: "message"
            )
        ]
    }
    public var approvalsEmptyStateRemediationActions: [EmptyStateRemediationAction] {
        if daemonControlAuthNeedsRemediation {
            return [
                remediationAction(
                    .openConfiguration,
                    title: "Open Configuration",
                    symbolName: "gearshape",
                    prominent: true
                )
            ]
        }
        return [
            remediationAction(
                .refreshApprovals,
                title: "Refresh Approvals",
                symbolName: "arrow.clockwise",
                prominent: true
            ),
            remediationAction(
                .openTasks,
                title: "Open Tasks",
                symbolName: "list.bullet.rectangle.portrait"
            )
        ]
    }
    public var tasksEmptyStateRemediationActions: [EmptyStateRemediationAction] {
        if daemonControlAuthNeedsRemediation {
            return [
                remediationAction(
                    .openConfiguration,
                    title: "Open Configuration",
                    symbolName: "gearshape",
                    prominent: true
                )
            ]
        }
        return [
            remediationAction(
                .openChat,
                title: "Open Chat",
                symbolName: "message",
                prominent: true
            ),
            remediationAction(
                .refreshTasks,
                title: "Refresh Tasks",
                symbolName: "arrow.clockwise"
            )
        ]
    }
    public var inspectEmptyStateRemediationActions: [EmptyStateRemediationAction] {
        if daemonControlAuthNeedsRemediation {
            return [
                remediationAction(
                    .openConfiguration,
                    title: "Open Configuration",
                    symbolName: "gearshape",
                    prominent: true
                )
            ]
        }
        return [
            remediationAction(
                .refreshInspect,
                title: "Refresh Logs",
                symbolName: "arrow.clockwise",
                prominent: true
            ),
            remediationAction(
                .openTasks,
                title: "Open Tasks",
                symbolName: "list.bullet.rectangle.portrait"
            )
        ]
    }
    public var channelsEmptyStateRemediationActions: [EmptyStateRemediationAction] {
        if daemonControlAuthNeedsRemediation {
            return [
                remediationAction(
                    .openConfiguration,
                    title: "Open Configuration",
                    symbolName: "gearshape",
                    prominent: true
                ),
                remediationAction(
                    .openConnectors,
                    title: "Open Connectors",
                    symbolName: "cable.connector"
                )
            ]
        }
        return [
            remediationAction(
                .refreshChannels,
                title: "Refresh Channels",
                symbolName: "arrow.clockwise",
                prominent: true
            ),
            remediationAction(
                .openConnectors,
                title: "Open Connectors",
                symbolName: "cable.connector"
            )
        ]
    }
    public var connectorsEmptyStateRemediationActions: [EmptyStateRemediationAction] {
        if daemonControlAuthNeedsRemediation {
            return [
                remediationAction(
                    .openConfiguration,
                    title: "Open Configuration",
                    symbolName: "gearshape",
                    prominent: true
                ),
                remediationAction(
                    .openChannels,
                    title: "Open Channels",
                    symbolName: "point.3.connected.trianglepath.dotted"
                )
            ]
        }
        return [
            remediationAction(
                .refreshConnectors,
                title: "Refresh Connectors",
                symbolName: "arrow.clockwise",
                prominent: true
            ),
            remediationAction(
                .openChannels,
                title: "Open Channels",
                symbolName: "point.3.connected.trianglepath.dotted"
            )
        ]
    }
    public var modelsEmptyStateRemediationActions: [EmptyStateRemediationAction] {
        if daemonControlAuthNeedsRemediation {
            return [
                remediationAction(
                    .openConfiguration,
                    title: "Open Configuration",
                    symbolName: "gearshape",
                    prominent: true
                )
            ]
        }
        return [
            remediationAction(
                .refreshModels,
                title: "Refresh Inventory",
                symbolName: "arrow.clockwise",
                prominent: true
            ),
            remediationAction(
                .runProviderChecks,
                title: "Run Checks",
                symbolName: "checkmark.seal"
            )
        ]
    }
    private func remediationAction(
        _ actionID: EmptyStateRemediationActionID,
        title: String,
        symbolName: String,
        prominent: Bool = false,
        disabled: Bool = false
    ) -> EmptyStateRemediationAction {
        EmptyStateRemediationAction(
            actionID: actionID,
            title: title,
            symbolName: symbolName,
            isProminent: prominent,
            isDisabled: disabled
        )
    }
    public var daemonEndpointLabel: String { daemonBaseURL.absoluteString }
    public var localDevAuthBootstrapCommand: String {
        let commandParts = ["personal-agent"] + localDevAuthBootstrapArguments()
        return commandParts.map(Self.shellEscapedCommandArgument).joined(separator: " ")
    }
    public var contextTaskClassOptions: [String] { ["chat", "agent", "automation", "comm", "approval"] }
    public var modelRouteSimulationTaskClassOptions: [String] {
        modelsRouteStore.modelRouteSimulationTaskClassOptions(
            contextTaskClassOptions: contextTaskClassOptions
        )
    }
    public var modelRouteSimulationPrincipalOptions: [String] {
        principalOptionsForPrincipalSelection(including: modelRouteSimulationPrincipalActorID)
    }
    public var logicalChannelCards: [LogicalChannelCardItem] {
        buildLogicalChannelCards(
            channelCards: channelCards,
            connectorCards: connectorCards,
            channelConnectorMappingsByChannelID: channelConnectorMappingsByChannelID
        )
    }
    public var logicalConnectorCards: [LogicalConnectorCardItem] {
        Self.buildLogicalConnectorCards(connectorCards: connectorCards)
    }
    public func channelCardItem(channelID: String) -> ChannelCardItem? {
        channelCards.first { $0.id == channelID }
    }
    public func connectorCardItem(connectorID: String) -> ConnectorCardItem? {
        connectorCards.first { $0.id == connectorID }
    }
    public var needsFirstRunOnboarding: Bool { !hasCompletedFirstRunOnboarding }
    public func isOnboardingSetupSection(_ section: AppSection) -> Bool {
        switch section {
        case .configuration, .channels, .connectors, .models:
            return true
        default:
            return false
        }
    }
    public func onboardingGateApplies(to section: AppSection) -> Bool {
        needsFirstRunOnboarding && !isOnboardingSetupSection(section)
    }
    public var selectedSectionOnboardingGateApplies: Bool {
        onboardingGateApplies(to: selectedSection)
    }
    public var onboardingProviderReady: Bool {
        providerReadinessItems.contains { item in
            item.status == .configured || item.status == .healthy
        }
    }
    public var onboardingModelCatalogReady: Bool {
        modelCatalogItems.contains { $0.enabled }
    }
    public var onboardingChatRouteReady: Bool {
        modelRouteSummary != nil
    }
    public var onboardingChannelConnectorMappingReady: Bool {
        let assessment = onboardingChannelConnectorAssessment()
        return assessment.missingChannels.isEmpty && assessment.missingConnectorIDs.isEmpty
    }
    public var effectiveDaemonControlAuthState: DaemonControlAuthState {
        runtimeLifecycleStore.effectiveDaemonControlAuthState(
            localDevTokenConfigured: localDevTokenConfigured
        )
    }
    public var daemonControlAuthNeedsRemediation: Bool {
        runtimeLifecycleStore.daemonControlAuthNeedsRemediation(
            localDevTokenConfigured: localDevTokenConfigured
        )
    }
    public var daemonControlAuthSetupDetail: String {
        runtimeLifecycleStore.daemonControlAuthSetupDetail(
            localDevTokenConfigured: localDevTokenConfigured
        )
    }
    public var onboardingReadinessMet: Bool {
        !daemonControlAuthNeedsRemediation &&
            daemonStatus == .running &&
            connectionStatus == .connected &&
            onboardingProviderReady &&
            onboardingModelCatalogReady &&
            onboardingChatRouteReady &&
            onboardingChannelConnectorMappingReady
    }
    public var onboardingSetupSteps: [OnboardingSetupStep] {
        [
            onboardingTokenSetupStep(),
            onboardingDaemonSetupStep(),
            onboardingProviderSetupStep(),
            onboardingModelCatalogSetupStep(),
            onboardingChatRouteSetupStep(),
            onboardingChannelConnectorMappingSetupStep()
        ]
        .sorted { lhs, rhs in
            if lhs.priority == rhs.priority {
                return lhs.title.localizedCaseInsensitiveCompare(rhs.title) == .orderedAscending
            }
            return lhs.priority < rhs.priority
        }
    }
    public var onboardingFixNextStep: OnboardingSetupStep? {
        onboardingSetupSteps.first { $0.status == .blocked }
    }
    public var onboardingSetupCompletedCount: Int {
        onboardingSetupSteps.filter { $0.status.isComplete }.count
    }
    public var onboardingSetupTotalCount: Int {
        onboardingSetupSteps.count
    }
    public var onboardingSetupProgressFraction: Double {
        let total = onboardingSetupTotalCount
        guard total > 0 else {
            return 0
        }
        return Double(onboardingSetupCompletedCount) / Double(total)
    }
    public var onboardingCurrentWizardStep: OnboardingSetupStep? {
        if let fixNext = onboardingFixNextStep {
            return fixNext
        }
        return onboardingSetupSteps.first { $0.status == .loading }
    }
    public var onboardingNextWizardStep: OnboardingSetupStep? {
        guard let current = onboardingCurrentWizardStep else {
            return nil
        }
        return onboardingSetupSteps.first { step in
            step.id != current.id && !step.status.isComplete
        }
    }
    public var onboardingSetupChecksLoading: Bool {
        onboardingSetupSteps.contains { $0.status == .loading }
    }
    public var onboardingSetupProgressSummary: String {
        let completeCount = onboardingSetupCompletedCount
        let blockedCount = onboardingSetupSteps.filter { $0.status.isBlocked }.count
        let totalCount = onboardingSetupTotalCount
        if blockedCount == 0, completeCount == totalCount {
            return "Setup is complete. Workflow panels are ready."
        }
        if onboardingSetupChecksLoading {
            return "Refreshing setup checks (\(completeCount)/\(totalCount) ready)…"
        }
        return "\(completeCount)/\(totalCount) checks ready • \(blockedCount) blocker\(blockedCount == 1 ? "" : "s") remaining."
    }
    public var setupReadinessChecksLoading: Bool {
        isDaemonLifecycleLoading ||
            !hasLoadedDaemonStatus ||
            isProviderStatusLoading ||
            !hasLoadedProviderStatus
    }
    public var isRuntimeStatusBootstrapLoading: Bool {
        !hasLoadedDaemonStatus
    }
    public var daemonHasWorkerFailureRepairState: Bool {
        runtimeLifecycleStore.daemonHasWorkerFailureRepairState
    }
    public var daemonNeedsInfrastructureRepair: Bool {
        runtimeLifecycleStore.daemonNeedsInfrastructureRepair
    }
    public var onboardingStatusMessage: String {
        if onboardingReadinessMet {
            return "Setup complete. All workflow sections are available."
        }
        if onboardingSetupChecksLoading, let loadingStep = onboardingSetupSteps.first(where: { $0.status == .loading }) {
            return "Checking \(loadingStep.title.lowercased())."
        }
        if let fixNext = onboardingFixNextStep {
            return "Fix Next: \(fixNext.title). \(fixNext.detail)"
        }
        return "Refresh setup checks to confirm readiness."
    }
    public func shouldShowCurrentSetupBlockerRibbon(for section: AppSection) -> Bool {
        guard section != .configuration else {
            return false
        }
        return onboardingSetupChecksLoading || onboardingFixNextStep != nil
    }
    public var currentSetupBlockerStatus: OnboardingSetupStepStatus {
        if onboardingSetupChecksLoading {
            return .loading
        }
        return onboardingFixNextStep?.status ?? .complete
    }
    public var currentSetupBlockerTitle: String {
        if let fixNext = onboardingFixNextStep {
            return fixNext.title
        }
        if onboardingSetupChecksLoading {
            return "Refreshing setup checks"
        }
        return "Setup checks complete"
    }
    public var currentSetupBlockerDetail: String {
        if onboardingSetupChecksLoading,
           let loadingStep = onboardingSetupSteps.first(where: { $0.status == .loading }) {
            return loadingStep.detail
        }
        if let fixNext = onboardingFixNextStep {
            return fixNext.detail
        }
        return onboardingStatusMessage
    }
    public var currentSetupBlockerSummary: String {
        if onboardingSetupChecksLoading {
            return onboardingSetupProgressSummary
        }
        if let fixNext = onboardingFixNextStep {
            return "Use Fix Next to resolve \(fixNext.title) and continue setup."
        }
        return "Setup checks are complete."
    }
    public var currentSetupBlockerSecondaryAction: OnboardingSetupAction? {
        if onboardingSetupChecksLoading {
            return onboardingSetupAction(
                .refreshChecks,
                title: "Refresh",
                detail: "Setup checks are already refreshing.",
                isEnabled: false
            )
        }
        guard let fixNext = onboardingFixNextStep else {
            return nil
        }
        let kind = onboardingSecondaryActionKind(for: fixNext.id)
        return onboardingSetupAction(kind, title: onboardingRibbonSecondaryActionTitle(for: kind))
    }
    public var onboardingCheckItems: [OnboardingCheckItem] {
        onboardingSetupSteps.map { step in
            OnboardingCheckItem(
                id: step.id,
                title: step.title,
                detail: step.detail,
                isComplete: step.status.isComplete
            )
        }
    }
    public var homeFirstSessionSteps: [HomeFirstSessionStep] {
        [
            HomeFirstSessionStep(
                id: .sendMessage,
                title: "Send your first chat message",
                detail: "Ask the assistant to complete one real workflow step.",
                actionTitle: "Open Chat",
                destinationSection: .chat,
                isComplete: isHomeFirstSessionStepComplete(.sendMessage)
            ),
            HomeFirstSessionStep(
                id: .sendCommunication,
                title: "Send one communication",
                detail: "Deliver a message from `Communications` using a configured channel.",
                actionTitle: "Open Communications",
                destinationSection: .communications,
                isComplete: isHomeFirstSessionStepComplete(.sendCommunication)
            ),
            HomeFirstSessionStep(
                id: .createTask,
                title: "Create one task",
                detail: "Submit a task and confirm the run appears in `Tasks`.",
                actionTitle: "Open Tasks",
                destinationSection: .tasks,
                isComplete: isHomeFirstSessionStepComplete(.createTask)
            ),
            HomeFirstSessionStep(
                id: .reviewApprovals,
                title: "Review approvals",
                detail: "Open `Approvals` and verify pending or recent decision outcomes.",
                actionTitle: "Open Approvals",
                destinationSection: .approvals,
                isComplete: isHomeFirstSessionStepComplete(.reviewApprovals)
            )
        ]
    }
    public var homeNextFirstSessionStep: HomeFirstSessionStep? {
        homeFirstSessionSteps.first { !$0.isComplete }
    }
    public var homeFirstSessionCompletionSummary: String {
        let steps = homeFirstSessionSteps
        let completedCount = steps.filter(\.isComplete).count
        if completedCount == steps.count {
            return "All first-session milestones are complete."
        }
        return "\(completedCount) of \(steps.count) first-session milestones complete."
    }
    public var homeFirstSessionFunnelDiagnostics: HomeFirstSessionFunnelDiagnostics {
        let steps = homeFirstSessionSteps
        let progress = currentWorkspaceHomeFirstSessionProgress()
        let milestoneItems = steps.map { step -> HomeFirstSessionFunnelMilestoneItem in
            let evidence = progress.milestoneEvidenceByStepID[step.id.rawValue]
            let completedAtRaw = nonEmpty(evidence?.completedAtRaw)
            let completionSource = nonEmpty(evidence?.source)
            let completedAtLabel = completedAtRaw
                .flatMap(parseDaemonTimestamp)?
                .formatted(date: .abbreviated, time: .shortened)
            return HomeFirstSessionFunnelMilestoneItem(
                id: step.id,
                title: step.title,
                isComplete: step.isComplete,
                completedAtRaw: completedAtRaw,
                completedAtLabel: completedAtLabel,
                completionSource: completionSource,
                completionSourceLabel: completionSource.map(homeFirstSessionMilestoneSourceLabel)
            )
        }
        let completedMilestoneTimestamps = milestoneItems
            .compactMap { $0.completedAtRaw }
            .compactMap(parseDaemonTimestamp)
            .sorted()
        let firstCompletedAt = completedMilestoneTimestamps.first
        let latestCompletedAt = completedMilestoneTimestamps.last
        return HomeFirstSessionFunnelDiagnostics(
            workspaceID: currentPanelFilterWorkspaceID(),
            completedCount: milestoneItems.filter(\.isComplete).count,
            totalCount: milestoneItems.count,
            firstCompletedAtRaw: firstCompletedAt.map(daemonTimestampString),
            firstCompletedAtLabel: firstCompletedAt?.formatted(date: .abbreviated, time: .shortened),
            latestCompletedAtRaw: latestCompletedAt.map(daemonTimestampString),
            latestCompletedAtLabel: latestCompletedAt?.formatted(date: .abbreviated, time: .shortened),
            milestones: milestoneItems
        )
    }
    public var homeFirstSessionGuidanceContext: HomeFirstSessionGuidanceContext? {
        guard onboardingReadinessMet,
              selectedSection != .home,
              !isOnboardingSetupSection(selectedSection),
              let nextStep = homeNextFirstSessionStep else {
            return nil
        }
        let steps = homeFirstSessionSteps
        let index = steps.firstIndex(where: { $0.id == nextStep.id }) ?? 0
        return HomeFirstSessionGuidanceContext(
            step: nextStep,
            stepNumber: index + 1,
            totalSteps: steps.count,
            isCurrentSectionDestination: selectedSection == nextStep.destinationSection
        )
    }
    public func performHomeFirstSessionStep(_ stepID: HomeFirstSessionStepID) {
        switch stepID {
        case .sendMessage:
            navigateToSection(.chat)
        case .sendCommunication:
            navigateToSection(.communications)
        case .createTask:
            navigateToSection(.tasks)
        case .reviewApprovals:
            markHomeFirstSessionStepComplete(.reviewApprovals, source: "home_checklist")
            navigateToSection(.approvals)
        }
    }
    public func performHomeFirstSessionGuidancePrimaryAction() {
        guard let context = homeFirstSessionGuidanceContext else {
            navigateToSection(.home)
            return
        }
        performHomeFirstSessionStep(context.step.id)
    }
    public func openHomeFirstSessionChecklist() {
        navigateToSection(.home)
    }
    public var modelRouteReadinessChecklistSteps: [OnboardingSetupStep] {
        let stepsByID = Dictionary(uniqueKeysWithValues: onboardingSetupSteps.map { ($0.id, $0) })
        return Self.modelRouteReadinessStepIDs.compactMap { stepsByID[$0] }
    }
    public var modelRouteReadinessNeedsAttention: Bool {
        modelRouteReadinessChecklistSteps.contains { !$0.status.isComplete }
    }
    public var modelRouteReadinessBlockerCount: Int {
        modelRouteReadinessChecklistSteps.filter { $0.status.isBlocked }.count
    }

    private func onboardingSetupAction(
        _ kind: OnboardingSetupActionKind,
        title: String? = nil,
        detail: String? = nil,
        isEnabled: Bool = true
    ) -> OnboardingSetupAction {
        let resolvedTitle: String
        if let title {
            resolvedTitle = title
        } else {
            switch kind {
            case .openConfiguration:
                resolvedTitle = "Open Configuration"
            case .openModels:
                resolvedTitle = "Open Models"
            case .openChannels:
                resolvedTitle = "Open Channels"
            case .openConnectors:
                resolvedTitle = "Open Connectors"
            case .refreshChecks:
                resolvedTitle = "Refresh Checks"
            case .startDaemon:
                resolvedTitle = "Start Daemon"
            case .installDaemon:
                resolvedTitle = "Install Daemon"
            case .repairDaemon:
                resolvedTitle = "Repair Daemon"
            }
        }
        return OnboardingSetupAction(
            kind: kind,
            title: resolvedTitle,
            detail: detail,
            isEnabled: isEnabled
        )
    }

    private func onboardingSecondaryActionKind(for stepID: String) -> OnboardingSetupActionKind {
        switch stepID {
        case "provider", "catalog_readiness", "chat_route":
            return .openModels
        case "token", "daemon":
            return .openConfiguration
        case "channel_connector_mapping":
            return .openChannels
        default:
            return .refreshChecks
        }
    }

    private func onboardingRibbonSecondaryActionTitle(for kind: OnboardingSetupActionKind) -> String {
        switch kind {
        case .refreshChecks:
            return "Refresh"
        case .openConfiguration:
            return "Open Configuration"
        case .openModels:
            return "Open Models"
        case .openChannels:
            return "Open Channels"
        case .openConnectors:
            return "Open Connectors"
        case .startDaemon:
            return "Start Daemon"
        case .installDaemon:
            return "Install Daemon"
        case .repairDaemon:
            return "Repair Daemon"
        }
    }

    private func currentWorkspaceHomeFirstSessionProgress() -> HomeFirstSessionProgress {
        contextRetentionStore.homeFirstSessionProgress(for: workspaceID)
    }

    private func updateCurrentWorkspaceHomeFirstSessionProgress(
        _ mutate: (inout HomeFirstSessionProgress) -> Void
    ) {
        contextRetentionStore.updateHomeFirstSessionProgress(for: workspaceID, mutate)
    }

    private func markHomeFirstSessionStepComplete(
        _ stepID: HomeFirstSessionStepID,
        source: String = "unknown",
        completedAt: Date = .now
    ) {
        updateCurrentWorkspaceHomeFirstSessionProgress { progress in
            switch stepID {
            case .sendMessage:
                progress.sentMessage = true
            case .sendCommunication:
                progress.sentCommunication = true
            case .createTask:
                progress.createdTask = true
            case .reviewApprovals:
                progress.reviewedApprovals = true
            }

            let normalizedSource = source
                .trimmingCharacters(in: .whitespacesAndNewlines)
                .lowercased()
            let resolvedSource = normalizedSource.isEmpty ? "unknown" : normalizedSource
            if progress.milestoneEvidenceByStepID[stepID.rawValue] == nil {
                progress.milestoneEvidenceByStepID[stepID.rawValue] = HomeFirstSessionProgress.MilestoneEvidence(
                    completedAtRaw: daemonTimestampString(completedAt),
                    source: resolvedSource
                )
            }
        }
    }

    private func homeFirstSessionMilestoneSourceLabel(_ source: String) -> String {
        switch source {
        case "chat_turn":
            return "Chat Turn"
        case "comm_send":
            return "Communication Send"
        case "task_submit":
            return "Task Submit"
        case "approvals_decision":
            return "Approval Decision"
        case "home_checklist":
            return "Guided Checklist"
        default:
            return source
                .replacingOccurrences(of: "_", with: " ")
                .capitalized
        }
    }

    private func isHomeFirstSessionStepComplete(_ stepID: HomeFirstSessionStepID) -> Bool {
        let progress = currentWorkspaceHomeFirstSessionProgress()
        switch stepID {
        case .sendMessage:
            return progress.sentMessage || chatTimelineItems.contains { item in
                item.kind == .assistantMessage || item.kind == .toolCall || item.kind == .toolResult
            }
        case .sendCommunication:
            return progress.sentCommunication || (latestCommunicationSendReceipt?.success ?? false)
        case .createTask:
            return progress.createdTask || latestTaskSubmissionReceipt != nil
        case .reviewApprovals:
            return progress.reviewedApprovals
                || approvalInboxItems.contains { $0.decisionState != .pending }
                || approvalsActionStatusByID.values.contains { status in
                    status.localizedCaseInsensitiveContains("decision submitted")
                }
        }
    }

    private func onboardingTokenSetupStep() -> OnboardingSetupStep {
        if isDaemonLifecycleLoading && localDevTokenConfigured {
            return OnboardingSetupStep(
                id: "token",
                title: "Assistant Access Token",
                priority: 10,
                status: .loading,
                detail: "Validating daemon auth token state.",
                remediationAction: onboardingSetupAction(
                    .refreshChecks,
                    isEnabled: false
                )
            )
        }

        let authState = effectiveDaemonControlAuthState
        return OnboardingSetupStep(
            id: "token",
            title: "Assistant Access Token",
            priority: 10,
            status: authState == .configured ? .complete : .blocked,
            detail: daemonControlAuthSetupDetail,
            remediationAction: authState == .configured
                ? nil
                : onboardingSetupAction(.openConfiguration)
        )
    }

    private func onboardingDaemonSetupStep() -> OnboardingSetupStep {
        if daemonControlAuthNeedsRemediation {
            return OnboardingSetupStep(
                id: "daemon",
                title: "Daemon Reachability",
                priority: 20,
                status: .blocked,
                detail: daemonControlAuthSetupDetail,
                remediationAction: onboardingSetupAction(.openConfiguration)
            )
        }

        if isDaemonLifecycleLoading || !hasLoadedDaemonStatus {
            return OnboardingSetupStep(
                id: "daemon",
                title: "Daemon Reachability",
                priority: 20,
                status: .loading,
                detail: "Refreshing daemon lifecycle and connection status.",
                remediationAction: onboardingSetupAction(
                    .refreshChecks,
                    isEnabled: !isDaemonLifecycleLoading
                )
            )
        }

        let daemonReady = daemonStatus == .running &&
            connectionStatus == .connected &&
            !daemonNeedsInstall &&
            !daemonNeedsInfrastructureRepair
        if daemonReady {
            return OnboardingSetupStep(
                id: "daemon",
                title: "Daemon Reachability",
                priority: 20,
                status: .complete,
                detail: "Daemon is running and reachable for workspace \(workspaceLabel).",
                remediationAction: nil
            )
        }

        if daemonNeedsInstall || daemonStatus == .missing {
            let action: OnboardingSetupAction
            if daemonCanInstallFromBundle {
                action = onboardingSetupAction(
                    .installDaemon,
                    isEnabled: !isDaemonControlInFlight
                )
            } else {
                action = onboardingSetupAction(
                    .openConfiguration,
                    title: "Open Configuration",
                    detail: "Open Configuration > Advanced to install daemon setup."
                )
            }
            return OnboardingSetupStep(
                id: "daemon",
                title: "Daemon Reachability",
                priority: 20,
                status: .blocked,
                detail: daemonStatusDetail,
                remediationAction: action
            )
        }

        if daemonNeedsInfrastructureRepair || daemonStatus == .broken {
            let action: OnboardingSetupAction
            if daemonCanRepairFromBundle {
                action = onboardingSetupAction(
                    .repairDaemon,
                    isEnabled: !isDaemonControlInFlight
                )
            } else {
                action = onboardingSetupAction(
                    .openConfiguration,
                    title: "Open Configuration",
                    detail: "Open Configuration > Advanced to repair daemon setup."
                )
            }
            return OnboardingSetupStep(
                id: "daemon",
                title: "Daemon Reachability",
                priority: 20,
                status: .blocked,
                detail: daemonStatusDetail,
                remediationAction: action
            )
        }

        if daemonStatus == .running && connectionStatus == .degraded {
            return OnboardingSetupStep(
                id: "daemon",
                title: "Daemon Reachability",
                priority: 20,
                status: .blocked,
                detail: "Daemon is running but connection is degraded. Refresh checks or restart daemon.",
                remediationAction: onboardingSetupAction(
                    .refreshChecks,
                    isEnabled: !isDaemonLifecycleLoading
                )
            )
        }

        let action: OnboardingSetupAction
        if daemonCanStart {
            action = onboardingSetupAction(
                .startDaemon,
                isEnabled: !isDaemonControlInFlight
            )
        } else {
            action = onboardingSetupAction(
                .openConfiguration,
                title: "Open Configuration",
                detail: "Open Configuration > Advanced to start daemon."
            )
        }
        return OnboardingSetupStep(
            id: "daemon",
            title: "Daemon Reachability",
            priority: 20,
            status: .blocked,
            detail: daemonStatusDetail,
            remediationAction: action
        )
    }

    private func onboardingProviderSetupStep() -> OnboardingSetupStep {
        if daemonControlAuthNeedsRemediation {
            return OnboardingSetupStep(
                id: "provider",
                title: "Provider Setup",
                priority: 30,
                status: .blocked,
                detail: daemonControlAuthSetupDetail,
                remediationAction: onboardingSetupAction(.openConfiguration)
            )
        }

        if isProviderStatusLoading || !hasLoadedProviderStatus {
            return OnboardingSetupStep(
                id: "provider",
                title: "Provider Setup",
                priority: 30,
                status: .loading,
                detail: "Refreshing provider inventory and readiness checks.",
                remediationAction: onboardingSetupAction(
                    .refreshChecks,
                    isEnabled: !isProviderStatusLoading
                )
            )
        }

        if onboardingProviderReady {
            let readyCount = providerReadinessItems.filter {
                $0.status == .configured || $0.status == .healthy
            }.count
            return OnboardingSetupStep(
                id: "provider",
                title: "Provider Setup",
                priority: 30,
                status: .complete,
                detail: "\(readyCount) provider\(readyCount == 1 ? "" : "s") configured and ready for routing.",
                remediationAction: nil
            )
        }

        return OnboardingSetupStep(
            id: "provider",
            title: "Provider Setup",
            priority: 30,
            status: .blocked,
            detail: providerStatusMessage
                ?? "Open Models and save endpoint/credential setup for at least one provider.",
            remediationAction: onboardingSetupAction(.openModels)
        )
    }

    private func onboardingModelCatalogSetupStep() -> OnboardingSetupStep {
        if daemonControlAuthNeedsRemediation {
            return OnboardingSetupStep(
                id: "model_catalog",
                title: "Model Catalog",
                priority: 40,
                status: .blocked,
                detail: daemonControlAuthSetupDetail,
                remediationAction: onboardingSetupAction(.openConfiguration)
            )
        }

        if (isProviderStatusLoading || !hasLoadedProviderStatus) && modelCatalogItems.isEmpty {
            return OnboardingSetupStep(
                id: "model_catalog",
                title: "Model Catalog",
                priority: 40,
                status: .loading,
                detail: "Refreshing model catalog inventory.",
                remediationAction: onboardingSetupAction(
                    .refreshChecks,
                    isEnabled: !isProviderStatusLoading
                )
            )
        }

        let totalCount = modelCatalogItems.count
        let enabledCount = modelCatalogItems.filter(\.enabled).count
        if totalCount > 0 && enabledCount > 0 {
            return OnboardingSetupStep(
                id: "model_catalog",
                title: "Model Catalog",
                priority: 40,
                status: .complete,
                detail: "Catalog includes \(enabledCount) enabled model\(enabledCount == 1 ? "" : "s") out of \(totalCount).",
                remediationAction: nil
            )
        }

        if totalCount > 0 {
            return OnboardingSetupStep(
                id: "model_catalog",
                title: "Model Catalog",
                priority: 40,
                status: .blocked,
                detail: "Catalog is loaded but no models are enabled for routing.",
                remediationAction: onboardingSetupAction(.openModels)
            )
        }

        return OnboardingSetupStep(
            id: "model_catalog",
            title: "Model Catalog",
            priority: 40,
            status: .blocked,
            detail: modelCatalogStatusMessage
                ?? "No model catalog entries are available yet.",
            remediationAction: onboardingSetupAction(.openModels)
        )
    }

    private func onboardingChatRouteSetupStep() -> OnboardingSetupStep {
        if daemonControlAuthNeedsRemediation {
            return OnboardingSetupStep(
                id: "chat_route",
                title: "Chat Route",
                priority: 50,
                status: .blocked,
                detail: daemonControlAuthSetupDetail,
                remediationAction: onboardingSetupAction(.openConfiguration)
            )
        }

        if (isProviderStatusLoading || !hasLoadedProviderStatus) && modelRouteSummary == nil {
            return OnboardingSetupStep(
                id: "chat_route",
                title: "Chat Route",
                priority: 50,
                status: .loading,
                detail: "Refreshing chat route resolution.",
                remediationAction: onboardingSetupAction(
                    .refreshChecks,
                    isEnabled: !isProviderStatusLoading
                )
            )
        }

        if let summary = modelRouteSummary {
            return OnboardingSetupStep(
                id: "chat_route",
                title: "Chat Route",
                priority: 50,
                status: .complete,
                detail: "Chat route resolves to \(providerDisplayName(summary.provider)) • \(summary.modelKey).",
                remediationAction: nil
            )
        }

        return OnboardingSetupStep(
            id: "chat_route",
            title: "Chat Route",
            priority: 50,
            status: .blocked,
            detail: chatRouteRemediationMessage
                ?? modelRouteStatusMessage
                ?? "Open Models and save a chat route policy to an enabled model.",
            remediationAction: onboardingSetupAction(.openModels)
        )
    }

    private func onboardingChannelConnectorMappingSetupStep() -> OnboardingSetupStep {
        if daemonControlAuthNeedsRemediation {
            return OnboardingSetupStep(
                id: "channel_connector_mapping",
                title: "Channel + Connector Mapping",
                priority: 60,
                status: .blocked,
                detail: daemonControlAuthSetupDetail,
                remediationAction: onboardingSetupAction(.openConfiguration)
            )
        }

        if isChannelsLoading ||
            isConnectorsLoading ||
            isChannelConnectorMappingsLoading ||
            !hasLoadedChannelStatus ||
            !hasLoadedConnectorStatus {
            return OnboardingSetupStep(
                id: "channel_connector_mapping",
                title: "Channel + Connector Mapping",
                priority: 60,
                status: .loading,
                detail: "Refreshing logical channel mappings and connector status cards.",
                remediationAction: onboardingSetupAction(
                    .refreshChecks,
                    isEnabled: !isChannelsLoading && !isConnectorsLoading
                )
            )
        }

        let assessment = onboardingChannelConnectorAssessment()
        if !assessment.missingChannels.isEmpty {
            let channelNames = assessment.missingChannels.map(onboardingLogicalChannelTitle(for:))
            let detail: String
            if channelNames.count == 1, let first = channelNames.first {
                detail = "\(first) channel has no enabled connector mapping. Open Channels and enable at least one mapped connector."
            } else {
                detail = "\(channelNames.joined(separator: ", ")) channels have no enabled connector mapping. Open Channels and enable mapped connectors."
            }
            return OnboardingSetupStep(
                id: "channel_connector_mapping",
                title: "Channel + Connector Mapping",
                priority: 60,
                status: .blocked,
                detail: detail,
                remediationAction: onboardingSetupAction(.openChannels)
            )
        }

        if !assessment.missingConnectorIDs.isEmpty {
            let connectorNames = assessment.missingConnectorIDs.map(onboardingConnectorTitle(for:))
            return OnboardingSetupStep(
                id: "channel_connector_mapping",
                title: "Channel + Connector Mapping",
                priority: 60,
                status: .blocked,
                detail: "Connector status is missing for \(connectorNames.joined(separator: ", ")). Open Connectors and refresh connector status.",
                remediationAction: onboardingSetupAction(.openConnectors)
            )
        }

        let mappingCount = channelConnectorMappingsByChannelID.values.reduce(0) { partialResult, items in
            partialResult + items.filter(\.enabled).count
        }
        return OnboardingSetupStep(
            id: "channel_connector_mapping",
            title: "Channel + Connector Mapping",
            priority: 60,
            status: .complete,
            detail: "Logical channels include enabled connector mappings (\(mappingCount) enabled total) with loaded connector cards.",
            remediationAction: nil
        )
    }

    private func onboardingLogicalChannelTitle(for channelID: String) -> String {
        let normalizedChannelID = normalizedChannelConnectorMappingChannelID(channelID)
        if let logicalCard = logicalChannelCards.first(where: {
            normalizedChannelConnectorMappingChannelID($0.channelID) == normalizedChannelID
        }) {
            return logicalCard.title
        }
        return logicalChannelDisplayName(for: normalizedChannelID, memberDisplayNames: [])
    }

    private func onboardingConnectorTitle(for connectorID: String) -> String {
        channelConnectorDisplayName(connectorID: connectorID)
    }

    private func onboardingChannelConnectorAssessment() -> OnboardingChannelConnectorAssessment {
        let normalizedMappings = normalizedChannelConnectorMappingsByLogicalChannelID(
            channelConnectorMappingsByChannelID
        )
        let inferredMappings = inferredChannelConnectorMappingsByLogicalChannelID(from: channelCards)
        let availableConnectors = Set(logicalConnectorCards.map {
            normalizedChannelConnectorMappingConnectorID($0.id)
        })

        var channelIDs = Set(Self.canonicalLogicalChannelSortOrder.keys)
        channelIDs.formUnion(normalizedMappings.keys)
        channelIDs.formUnion(inferredMappings.keys)
        channelIDs.formUnion(
            logicalChannelCards.map { normalizedChannelConnectorMappingChannelID($0.channelID) }
        )
        channelIDs = Set(channelIDs.filter { !$0.isEmpty })

        var missingChannels: [String] = []
        var missingConnectorIDs: Set<String> = []
        for channelID in channelIDs {
            let mappings = mergedChannelConnectorMappings(
                observed: normalizedMappings[channelID] ?? [],
                inferred: inferredMappings[channelID] ?? [],
                channelID: channelID
            )
            let enabledConnectorIDs = Set(
                mappings
                    .filter(\.enabled)
                    .map { normalizedChannelConnectorMappingConnectorID($0.connectorID) }
            )
            let requiredConnectorIDs = onboardingAllowedConnectorIDs(for: channelID)
            let compatibleConnectorIDs: Set<String>
            if let requiredConnectorIDs {
                compatibleConnectorIDs = enabledConnectorIDs.intersection(requiredConnectorIDs)
            } else {
                compatibleConnectorIDs = enabledConnectorIDs
            }

            if compatibleConnectorIDs.isEmpty {
                missingChannels.append(channelID)
                continue
            }
            for connectorID in compatibleConnectorIDs where !availableConnectors.contains(connectorID) {
                missingConnectorIDs.insert(connectorID)
            }
        }

        return OnboardingChannelConnectorAssessment(
            missingChannels: missingChannels.sorted(),
            missingConnectorIDs: missingConnectorIDs.sorted()
        )
    }

    private func onboardingAllowedConnectorIDs(for channelID: String) -> Set<String>? {
        switch Self.canonicalLogicalChannelID(from: channelID) {
        case "app":
            return ["builtin.app"]
        case "message":
            return ["imessage", "twilio"]
        case "voice":
            return ["twilio"]
        default:
            return nil
        }
    }

    public init() {
        let daemonAddress = ProcessInfo.processInfo.environment["PERSONAL_AGENT_DAEMON_URL"]
            ?? Self.defaultDaemonAddress
        self.daemonBaseURL = URL(string: daemonAddress) ?? URL(string: Self.defaultDaemonAddress)!
        self.contextRetentionStore = AppContextRetentionStore(
            userDefaults: Self.userDefaultsStore,
            defaultWorkspaceID: Self.defaultWorkspaceID,
            canonicalWorkspaceID: Self.canonicalWorkspaceID,
            panelFilterContextDefaultsKey: Self.panelFilterContextDefaultsKey,
            communicationsTriageDefaultsKey: Self.communicationsTriageDefaultsKey,
            workspaceContinuityDefaultsKey: Self.workspaceContinuityDefaultsKey,
            informationDensityModeDefaultsKey: Self.informationDensityModeDefaultsKey,
            homeFirstSessionProgressDefaultsKey: Self.homeFirstSessionProgressDefaultsKey
        )
        self.commandHistoryStore = AppCommandHistoryStore(
            userDefaults: Self.userDefaultsStore,
            defaultsKey: Self.recentAppCommandsDefaultsKey,
            historyLimit: Self.recentAppCommandHistoryLimit
        )
        self.notificationStore = AppNotificationCenterStore(
            userDefaults: Self.userDefaultsStore,
            defaultsKey: Self.notificationsDefaultsKey,
            defaultWorkspaceID: Self.defaultWorkspaceID,
            notificationHistoryLimit: Self.notificationHistoryLimit
        )
        modelsRouteStore.providerReadinessItems = Self.defaultProviderReadinessItems()
        modelsRouteStore.providerEndpointDraftByID = Self.defaultProviderEndpointDrafts()
        modelsRouteStore.providerAPIKeySecretNameDraftByID = Self.defaultProviderSecretNameDrafts()
        modelsRouteStore.providerEndpointSourceByID = Self.defaultProviderEndpointDrafts()
        modelsRouteStore.providerSecretNameSourceByID = Self.defaultProviderSecretNameDrafts()
        workflowMutationStoreCancellable = workflowMutationStore.objectWillChange.sink { [weak self] _ in
            self?.objectWillChange.send()
        }
        navigationStoreCancellable = navigationStore.objectWillChange.sink { [weak self] _ in
            self?.objectWillChange.send()
        }
        identityContextStoreCancellable = identityContextStore.objectWillChange.sink { [weak self] _ in
            self?.objectWillChange.send()
        }
        runtimeLifecycleStoreCancellable = runtimeLifecycleStore.objectWillChange.sink { [weak self] _ in
            self?.objectWillChange.send()
        }
        panelProblemStoreCancellable = panelProblemStore.objectWillChange.sink { [weak self] _ in
            self?.objectWillChange.send()
        }
        contextRetentionStoreCancellable = contextRetentionStore.objectWillChange.sink { [weak self] _ in
            self?.objectWillChange.send()
        }
        commandHistoryStoreCancellable = commandHistoryStore.objectWillChange.sink { [weak self] _ in
            self?.objectWillChange.send()
        }
        notificationStoreCancellable = notificationStore.objectWillChange.sink { [weak self] _ in
            self?.objectWillChange.send()
        }
        panelLatencyStoreCancellable = panelLatencyStore.objectWillChange.sink { [weak self] _ in
            self?.objectWillChange.send()
        }
        connectionConfigStoreCancellable = connectionConfigStore.objectWillChange.sink { [weak self] _ in
            self?.objectWillChange.send()
        }
        modelsRouteStoreCancellable = modelsRouteStore.objectWillChange.sink { [weak self] _ in
            self?.objectWillChange.send()
        }
        workflowQueueStoreCancellable = workflowQueueStore.objectWillChange.sink { [weak self] _ in
            self?.objectWillChange.send()
        }
        communicationsStoreCancellable = communicationsStore.objectWillChange.sink { [weak self] _ in
            self?.objectWillChange.send()
        }
        inspectStoreCancellable = inspectStore.objectWillChange.sink { [weak self] _ in
            self?.objectWillChange.send()
        }
        lastInspectStatusNotificationValue = inspectStore.inspectStatusMessage
        inspectStatusNotificationCancellable = inspectStore.$inspectStatusMessage
            .dropFirst()
            .sink { [weak self] newValue in
                guard let self else {
                    return
                }
                self.recordStatusNotification(
                    source: "inspect",
                    oldValue: self.lastInspectStatusNotificationValue,
                    newValue: newValue
                )
                self.lastInspectStatusNotificationValue = newValue
            }
        identityContextStore.configureInitialWorkspaceContext(
            envWorkspaceID: ProcessInfo.processInfo.environment["PERSONAL_AGENT_WORKSPACE_ID"],
            storedWorkspaceID: Self.userDefaultsStore.string(forKey: Self.workspaceDefaultsKey),
            defaultWorkspaceID: Self.defaultWorkspaceID,
            canonicalWorkspaceID: Self.canonicalWorkspaceID,
            persistWorkspaceSelection: { [weak self] workspaceID in
                self?.persistWorkspaceSelection(workspaceID)
            }
        )
        loadPersistedInformationDensityModes()
        applyWorkspaceScopedInformationDensityMode(for: workspaceID)
        hasCompletedFirstRunOnboarding = Self.userDefaultsStore.bool(forKey: Self.onboardingCompleteDefaultsKey)
        loadPersistedNotifications()
        loadPersistedPanelFilterContexts()
        loadPersistedCommunicationsTriageContexts()
        loadPersistedWorkspaceContinuityContexts()
        loadPersistedHomeFirstSessionProgress()
        loadPersistedRecentAppCommands()

        if let persistedToken = Self.loadPersistedLocalDevToken(), !persistedToken.isEmpty {
            daemonAuthToken = persistedToken
            localDevTokenConfigured = true
            localDevTokenLastUpdated = "Stored locally"
        } else {
            daemonAuthToken = nil
            localDevTokenConfigured = false
            localDevTokenLastUpdated = "Not configured"
        }
        contextMemoryOwnerActorFilter = selectedPrincipal
        contextMemoryCandidatesOwnerActorFilter = selectedPrincipal
        contextRetrievalOwnerActorFilter = selectedPrincipal
        capabilityGrantActorFilter = selectedPrincipal
        modelRouteSimulationTaskClass = "chat"
        modelRouteSimulationPrincipalActorID = selectedPrincipal
        updateOnboardingCompletionState()

        Task { [weak self] in
            guard let self else {
                return
            }
            await self.bootstrapFromDaemon()
            self.startLifecyclePolling()
        }
    }

    public func toggleSidebar() {
        navigationStore.toggleSidebar()
    }

    public var isAdvancedSidebarNavigationVisible: Bool {
        navigationStore.isAdvancedSidebarNavigationVisible
    }

    public var visibleSidebarNavigationSections: [AppSection] {
        navigationStore.visibleSidebarNavigationSections
    }

    public func setSidebarAdvancedNavigationExpanded(_ isExpanded: Bool) {
        navigationStore.setSidebarAdvancedNavigationExpanded(isExpanded)
    }

    public func toggleSidebarAdvancedNavigationExpanded() {
        navigationStore.toggleSidebarAdvancedNavigationExpanded()
    }

    public func requestStartDaemon() {
        presentHighImpactActionConfirmation(
            title: "Start Daemon?",
            message: "Start `Personal Agent Daemon` now for this workspace context.",
            confirmButtonTitle: "Start Daemon",
            isDestructive: false
        ) { [weak self] in
            self?.startDaemon()
            self?.presentUndoActionPrompt(
                title: "Daemon start requested",
                message: "Undo if you want to stop it again.",
                actionTitle: "Undo Start"
            ) { [weak self] in
                self?.stopDaemon()
            }
        }
    }

    public func requestStopDaemon() {
        presentHighImpactActionConfirmation(
            title: "Stop Daemon?",
            message: "Stopping daemon runtime will pause in-app workflows until it is started again.",
            confirmButtonTitle: "Stop Daemon",
            isDestructive: true
        ) { [weak self] in
            self?.stopDaemon()
            self?.presentUndoActionPrompt(
                title: "Daemon stop requested",
                message: "Undo if this was accidental.",
                actionTitle: "Undo Stop"
            ) { [weak self] in
                self?.startDaemon()
            }
        }
    }

    public func requestRestartDaemon() {
        presentHighImpactActionConfirmation(
            title: "Restart Daemon?",
            message: "Restart will interrupt active runtime operations while daemon restarts.",
            confirmButtonTitle: "Restart Daemon",
            isDestructive: true
        ) { [weak self] in
            self?.restartDaemon()
        }
    }

    public func requestInstallDaemon() {
        presentHighImpactActionConfirmation(
            title: "Install Daemon Setup?",
            message: "Install will apply daemon host setup for this machine.",
            confirmButtonTitle: "Install",
            isDestructive: true,
            irreversibleNote: "Install/uninstall operations change host lifecycle configuration."
        ) { [weak self] in
            self?.installDaemon()
        }
    }

    public func requestUninstallDaemon() {
        presentHighImpactActionConfirmation(
            title: "Uninstall Daemon Setup?",
            message: "Uninstall removes daemon host setup and may disable automatic runtime startup.",
            confirmButtonTitle: "Uninstall",
            isDestructive: true,
            irreversibleNote: "Uninstall actions are not automatically reversible."
        ) { [weak self] in
            self?.uninstallDaemon()
        }
    }

    public func requestRepairDaemonInstallation() {
        presentHighImpactActionConfirmation(
            title: "Repair Daemon Setup?",
            message: "Repair will run daemon host setup remediation for this machine.",
            confirmButtonTitle: "Repair",
            isDestructive: true
        ) { [weak self] in
            self?.repairDaemonInstallation()
        }
    }

    public func startDaemon() {
        requestDaemonLifecycleControl(action: "start")
    }

    public func stopDaemon() {
        requestDaemonLifecycleControl(action: "stop")
    }

    public func stopDaemonForTermination(maxWaitSeconds: TimeInterval = 2.5) async {
        guard daemonCanStop else {
            return
        }
        guard !isDaemonControlInFlight else {
            return
        }
        guard resolvedAuthToken() != nil else {
            return
        }

        let boundedWaitSeconds = min(max(maxWaitSeconds, 0.25), 10)
        let deadline = Date().addingTimeInterval(boundedWaitSeconds)
        let startupGraceDeadline = Date().addingTimeInterval(0.2)
        let stopTask = Task { [weak self] in
            guard let self else {
                return
            }
            await self.performDaemonLifecycleControl(
                action: "stop",
                waitForOperationCompletion: false,
                reasonContext: "ui:taskbar_quit"
            )
        }

        var observedControlInFlight = false
        while Date() < deadline {
            if isDaemonControlInFlight {
                observedControlInFlight = true
            }
            if observedControlInFlight && !isDaemonControlInFlight {
                break
            }
            if !observedControlInFlight, Date() >= startupGraceDeadline, !isDaemonControlInFlight {
                break
            }
            try? await Task.sleep(for: .milliseconds(40))
        }

        stopTask.cancel()
    }

    public func restartDaemon() {
        requestDaemonLifecycleControl(action: "restart")
    }

    public func markDaemonMissing() {
        runtimeLifecycleStore.markDaemonMissing()
    }

    public func markDaemonBroken() {
        runtimeLifecycleStore.markDaemonBroken()
    }

    public func installDaemon() {
        requestDaemonLifecycleControl(action: "install")
    }

    public func uninstallDaemon() {
        requestDaemonLifecycleControl(action: "uninstall")
    }

    public func repairDaemonInstallation() {
        requestDaemonLifecycleControl(action: "repair")
    }

    public func refreshDaemonStatus() {
        Task { [weak self] in
            guard let self else {
                return
            }
            let startedAtNanoseconds = DispatchTime.now().uptimeNanoseconds
            await self.refreshDaemonLifecycleStatus()
            let elapsedNanoseconds = DispatchTime.now().uptimeNanoseconds - startedAtNanoseconds
            self.panelLatencyStore.recordPanelLatencySample(
                section: .configuration,
                category: .refresh,
                durationMS: Int(elapsedNanoseconds / 1_000_000)
            )
        }
    }

    public func refreshDaemonPluginLifecycleHistory() {
        Task { [weak self] in
            await self?.fetchDaemonPluginLifecycleHistory()
        }
    }

    public func resetDaemonPluginLifecycleHistoryFilters() {
        daemonPluginLifecycleHistoryFilterPluginID = ""
        daemonPluginLifecycleHistoryFilterKind = RuntimePluginLifecycleProjection.defaultFilterSelection
        daemonPluginLifecycleHistoryFilterState = RuntimePluginLifecycleProjection.defaultFilterSelection
        daemonPluginLifecycleHistoryFilterEventType = RuntimePluginLifecycleProjection.defaultFilterSelection
        daemonPluginLifecycleHistoryLimit = RuntimePluginLifecycleProjection.defaultHistoryLimit
    }

    public func handleAppDidBecomeActive() {
        Task { [weak self] in
            await self?.refreshOnAppActivation()
        }
    }

    public var channelsHasUnsavedDraftChanges: Bool {
        !dirtyChannelConfigurationIDs().isEmpty ||
            !dirtyChannelDeliveryPolicyIDs().isEmpty ||
            !dirtyChannelConnectorMappingIDs().isEmpty
    }

    public var connectorsHasUnsavedDraftChanges: Bool {
        !dirtyConnectorConfigurationIDs().isEmpty
    }

    public var modelsHasUnsavedDraftChanges: Bool {
        !dirtyProviderSetupIDs().isEmpty
    }

    public func hasUnsavedDraftChanges(for section: AppSection) -> Bool {
        switch section {
        case .channels:
            return channelsHasUnsavedDraftChanges
        case .connectors:
            return connectorsHasUnsavedDraftChanges
        case .models:
            return modelsHasUnsavedDraftChanges
        default:
            return false
        }
    }

    public func unsavedDraftSummary(for section: AppSection) -> String? {
        switch section {
        case .channels:
            let configCount = dirtyChannelConfigurationIDs().count
            let policyCount = dirtyChannelDeliveryPolicyIDs().count
            let mappingCount = dirtyChannelConnectorMappingIDs().count
            let total = configCount + policyCount + mappingCount
            guard total > 0 else {
                return nil
            }
            return "Channels has \(total) unsaved draft change(s) across configuration, delivery policy, or connector mappings."
        case .connectors:
            let dirtyCount = dirtyConnectorConfigurationIDs().count
            guard dirtyCount > 0 else {
                return nil
            }
            return "Connectors has \(dirtyCount) unsaved configuration draft change(s)."
        case .models:
            let dirtyCount = dirtyProviderSetupIDs().count
            guard dirtyCount > 0 else {
                return nil
            }
            return "Models has \(dirtyCount) unsaved provider setup draft change(s)."
        default:
            return nil
        }
    }

    public var recentAppCommandActionIDs: [AppCommandActionID] {
        commandHistoryStore.recentUsage
    }

    public var appCommandActionItems: [AppCommandActionItem] {
        AppCommandPaletteSearchEngine.actionItems(
            onboardingFixNextStepDetail: onboardingFixNextStep?.detail,
            recentUsage: commandHistoryStore.recentUsage,
            isEnabled: { [self] actionID in isAppCommandEnabled(actionID) },
            disabledReason: { [self] actionID in commandDisabledReason(for: actionID) }
        )
    }

    public func rankedAppCommandActionItems(for query: String) -> [AppCommandActionItem] {
        AppCommandPaletteSearchEngine.rankedActionItems(
            for: query,
            from: appCommandActionItems
        )
    }

    public func firstEnabledAppCommandAction(for query: String) -> AppCommandActionItem? {
        AppCommandPaletteSearchEngine.firstEnabledAction(
            for: query,
            from: appCommandActionItems
        )
    }

    public func rankedCommandPaletteObjectItems(for query: String) -> [CommandPaletteObjectSearchItem] {
        AppCommandPaletteSearchEngine.rankedObjectItems(
            for: query,
            from: commandPaletteObjectSearchCandidates()
        )
    }

    public func firstCommandPaletteObjectMatch(for query: String) -> CommandPaletteObjectSearchItem? {
        AppCommandPaletteSearchEngine.firstObjectMatch(
            for: query,
            from: commandPaletteObjectSearchCandidates()
        )
    }

    public func performCommandPaletteObjectAction(_ target: CommandPaletteObjectTarget) {
        switch target {
        case .taskRun(let rowID):
            guard let row = taskRunItems.first(where: { $0.id == rowID }) else {
                requestSectionSelection(.tasks)
                tasksStatusMessage = "Task result is no longer available. Refresh Tasks."
                return
            }
            let searchSeed = nonEmpty(row.runID) ?? row.taskID
            tasksSearchSeed = searchSeed
            tasksStatusMessage = "Opened task result for \(searchSeed)."
            requestSectionSelection(.tasks)

        case .approval(let approvalID):
            approvalsSearchSeed = approvalID
            approvalsStatusMessage = "Opened approval \(approvalID)."
            requestSectionSelection(.approvals)

        case .thread(let threadID):
            let currentContext = communicationsFilterContext()
            updateCommunicationsFilterContext(
                communicationsStore.threadSelectionFilterContext(
                    threadID: threadID,
                    currentContext: currentContext
                )
            )
            communicationsStatusMessage = "Opened thread \(threadID)."
            requestSectionSelection(.communications)

        case .connector(let connectorID):
            if let connectorCard = connectorCardItem(connectorID: connectorID), !connectorCard.isExpanded {
                toggleConnectorCard(connectorID)
            }
            connectorsStatusMessage = "Opened connector \(connectorID)."
            requestSectionSelection(.connectors)

        case .model(let providerID, let modelKey):
            modelCatalogStatusMessage = "Opened model \(providerID)/\(modelKey)."
            requestSectionSelection(.models)
        }
    }

    public func presentCommandPalette() {
        commandPaletteSearchQuery = ""
        isCommandPalettePresented = true
    }

    public func presentDoEntryPoint() {
        commandPaletteSearchQuery = "do "
        isCommandPalettePresented = true
    }

    public func dismissCommandPalette() {
        isCommandPalettePresented = false
        commandPaletteSearchQuery = ""
    }

    public func isAppCommandEnabled(_ actionID: AppCommandActionID) -> Bool {
        switch actionID {
        case .doSendMessage,
                .doSendEmail,
                .doCreateTask,
                .doReviewApprovals,
                .doInspectIssue,
                .openConfiguration,
                .openChat,
                .openCommunications,
                .openAutomation,
                .openApprovals,
                .openTasks,
                .openModels,
                .openInspect,
                .openChannels,
                .openConnectors,
                .refreshCurrentSection,
                .openNotificationCenter:
            return true
        case .setSimpleDensityMode:
            return informationDensityMode != .simple
        case .setAdvancedDensityMode:
            return informationDensityMode != .advanced
        case .performOnboardingFixNextStep:
            return onboardingFixNextStep != nil || onboardingSetupChecksLoading
        case .startDaemon:
            return daemonCanStart && !isDaemonControlInFlight
        case .stopDaemon:
            return daemonCanStop && !isDaemonControlInFlight
        case .restartDaemon:
            return daemonCanRestart && !isDaemonControlInFlight
        }
    }

    public func isRecentAppCommandAction(_ actionID: AppCommandActionID) -> Bool {
        commandHistoryStore.contains(actionID)
    }

    public func commandDisabledReason(for actionID: AppCommandActionID) -> String? {
        guard !isAppCommandEnabled(actionID) else {
            return nil
        }

        switch actionID {
        case .setSimpleDensityMode:
            return "Information density is already set to Simple."
        case .setAdvancedDensityMode:
            return "Information density is already set to Advanced."
        case .performOnboardingFixNextStep:
            return "Setup checks are complete or no unresolved setup blocker remains."
        case .startDaemon:
            if isDaemonControlInFlight {
                return "Daemon lifecycle action is already in progress."
            }
            return "Daemon start control is unavailable in the current lifecycle state."
        case .stopDaemon:
            if isDaemonControlInFlight {
                return "Daemon lifecycle action is already in progress."
            }
            return "Daemon stop control is unavailable in the current lifecycle state."
        case .restartDaemon:
            if isDaemonControlInFlight {
                return "Daemon lifecycle action is already in progress."
            }
            return "Daemon restart control is unavailable in the current lifecycle state."
        default:
            return "Action is currently unavailable."
        }
    }

    public func performAppCommand(_ actionID: AppCommandActionID) {
        guard isAppCommandEnabled(actionID) else {
            return
        }
        switch actionID {
        case .doSendMessage:
            applyDoEntryPointChatDraftIfEmpty("Send a message to ")
            chatStatusMessage = "Do: Send a Message ready. Enter recipient and details, then send."
            requestSectionSelection(.chat)
        case .doSendEmail:
            applyDoEntryPointChatDraftIfEmpty("Draft and send an email to ")
            chatStatusMessage = "Do: Send an Email ready. Enter recipient and intent, then send."
            requestSectionSelection(.chat)
        case .doCreateTask:
            tasksStatusMessage = "Do: Create a Task ready. Use New Task to draft and submit."
            requestSectionSelection(.tasks)
        case .doReviewApprovals:
            approvalsStatusMessage = "Do: Review Approvals ready. Open a pending item and submit a decision."
            requestSectionSelection(.approvals)
        case .doInspectIssue:
            inspectStatusMessage = "Do: Inspect an Issue ready. Review recent activity and drill into evidence."
            requestSectionSelection(.inspect)
        case .openConfiguration:
            requestSectionSelection(.configuration)
        case .openChat:
            requestSectionSelection(.chat)
        case .openCommunications:
            requestSectionSelection(.communications)
        case .openAutomation:
            requestSectionSelection(.automation)
        case .openApprovals:
            requestSectionSelection(.approvals)
        case .openTasks:
            requestSectionSelection(.tasks)
        case .openModels:
            requestSectionSelection(.models)
        case .openInspect:
            requestSectionSelection(.inspect)
        case .openChannels:
            requestSectionSelection(.channels)
        case .openConnectors:
            requestSectionSelection(.connectors)
        case .refreshCurrentSection:
            navigateToSection(selectedSection)
        case .openNotificationCenter:
            presentNotificationCenter()
        case .setSimpleDensityMode:
            setInformationDensityMode(.simple)
        case .setAdvancedDensityMode:
            setInformationDensityMode(.advanced)
        case .performOnboardingFixNextStep:
            performOnboardingFixNextStep()
        case .startDaemon:
            requestStartDaemon()
        case .stopDaemon:
            requestStopDaemon()
        case .restartDaemon:
            requestRestartDaemon()
        }
        recordAppCommandUsage(actionID)
    }

    private func applyDoEntryPointChatDraftIfEmpty(_ starterText: String) {
        guard chatDraft.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else {
            return
        }
        chatDraft = starterText
    }

    private func statePrincipalDisplayName(_ actorID: String?) -> String? {
        nonEmpty(principalIdentityDisplayValue(for: actorID).displayText)
    }

    private func commandPaletteObjectSearchCandidates() -> [CommandPaletteObjectSearchCandidate] {
        CommandPaletteObjectCandidateBuilder.build(
            taskRunItems: taskRunItems,
            approvalInboxItems: approvalInboxItems,
            communicationThreads: communicationThreads,
            logicalConnectorCards: logicalConnectorCards,
            modelCatalogItems: modelCatalogItems,
            principalDisplayName: { [self] actorID in statePrincipalDisplayName(actorID) },
            providerDisplayName: { [self] providerID in providerDisplayName(providerID) }
        )
    }

    public var panelLatencyLatestSamplesSorted: [UIPanelLatencySample] {
        panelLatencyStore.panelLatencyLatestSamplesSorted
    }

    public var panelLatencyRegressionSamples: [UIPanelLatencySample] {
        panelLatencyStore.panelLatencyRegressionSamples
    }

    public var panelLatencySampleCount: Int {
        panelLatencyStore.panelLatencySampleCount
    }

    public func successNotificationPulse(for source: String) -> Int {
        notificationStore.successNotificationPulse(for: source)
    }

    public func clearPanelLatencySamples() {
        panelLatencyStore.clearPanelLatencySamples()
    }

    public func communicationsFilterContext() -> CommunicationsFilterContext {
        communicationsStore.communicationsFilterContext(
            workspaceID: workspaceID,
            contextRetentionStore: contextRetentionStore
        )
    }

    public func updateCommunicationsFilterContext(_ context: CommunicationsFilterContext) {
        communicationsStore.setCommunicationsFilterContext(
            context,
            workspaceID: workspaceID,
            contextRetentionStore: contextRetentionStore
        )
    }

    public func communicationsTriageContext() -> CommunicationsTriageContext {
        communicationsStore.communicationsTriageContext(
            workspaceID: workspaceID,
            contextRetentionStore: contextRetentionStore
        )
    }

    public func updateCommunicationsTriageContext(_ context: CommunicationsTriageContext) {
        communicationsStore.setCommunicationsTriageContext(
            context,
            workspaceID: workspaceID,
            contextRetentionStore: contextRetentionStore
        )
    }

    public func communicationsComposeDraftContext() -> CommunicationsComposeDraftContext? {
        communicationsStore.communicationsComposeDraftContext(
            workspaceID: workspaceID,
            contextRetentionStore: contextRetentionStore
        )
    }

    public func updateCommunicationsComposeDraftContext(_ context: CommunicationsComposeDraftContext?) {
        communicationsStore.setCommunicationsComposeDraftContext(
            context,
            workspaceID: workspaceID,
            contextRetentionStore: contextRetentionStore
        )
    }

    public func tasksSubmitDraftContext() -> TasksSubmitDraftContext? {
        currentWorkspaceContinuityContext().tasksSubmitDraft
    }

    public func updateTasksSubmitDraftContext(_ context: TasksSubmitDraftContext?) {
        updateCurrentWorkspaceContinuityContext { value in
            value.tasksSubmitDraft = context
        }
    }

    public func expandedChannelCardIDsContinuity() -> Set<String> {
        Set(currentWorkspaceContinuityContext().expandedChannelCardIDs)
    }

    public func expandedConnectorCardIDsContinuity() -> Set<String> {
        Set(currentWorkspaceContinuityContext().expandedConnectorCardIDs)
    }

    public func resetWorkspaceContinuityContext() {
        contextRetentionStore.resetWorkspaceContinuityContext(for: workspaceID)
        taskSubmitDraftSeed = nil

        for index in channelCards.indices {
            channelCards[index].isExpanded = false
        }
        for index in connectorCards.indices {
            connectorCards[index].isExpanded = false
        }
    }

    @discardableResult
    public func resetCommunicationsFilterContext() -> CommunicationsFilterContext {
        communicationsStore.resetCommunicationsFilterContext(
            workspaceID: workspaceID,
            contextRetentionStore: contextRetentionStore
        )
    }

    public func tasksFilterContext() -> TasksFilterContext {
        currentWorkspacePanelFilterContext().tasks
    }

    public func updateTasksFilterContext(_ context: TasksFilterContext) {
        updateCurrentWorkspacePanelFilterContext { value in
            value.tasks = context
        }
    }

    @discardableResult
    public func resetTasksFilterContext() -> TasksFilterContext {
        let reset = TasksFilterContext()
        updateTasksFilterContext(reset)
        return reset
    }

    public func approvalsFilterContext() -> ApprovalsFilterContext {
        currentWorkspacePanelFilterContext().approvals
    }

    public func updateApprovalsFilterContext(_ context: ApprovalsFilterContext) {
        updateCurrentWorkspacePanelFilterContext { value in
            value.approvals = context
        }
    }

    @discardableResult
    public func resetApprovalsFilterContext() -> ApprovalsFilterContext {
        let reset = ApprovalsFilterContext()
        updateApprovalsFilterContext(reset)
        return reset
    }

    public func inspectFilterContext() -> InspectFilterContext {
        currentWorkspacePanelFilterContext().inspect
    }

    public func updateInspectFilterContext(_ context: InspectFilterContext) {
        updateCurrentWorkspacePanelFilterContext { value in
            value.inspect = context
        }
    }

    @discardableResult
    public func resetInspectFilterContext() -> InspectFilterContext {
        let reset = InspectFilterContext()
        updateInspectFilterContext(reset)
        return reset
    }

    public func activeFilterCount(for section: AppSection) -> Int {
        activeFilterSummaryParts(for: section).count
    }

    public func activeFilterSummary(for section: AppSection) -> String? {
        let summaryParts = activeFilterSummaryParts(for: section)
        guard !summaryParts.isEmpty else {
            return nil
        }
        return summaryParts.joined(separator: " • ")
    }

    private func activeFilterSummaryParts(for section: AppSection) -> [String] {
        switch section {
        case .communications:
            return communicationsFilterContext().activeFilterSummaryParts
        case .tasks:
            return tasksFilterContext().activeFilterSummaryParts
        case .approvals:
            return approvalsFilterContext().activeFilterSummaryParts
        case .inspect:
            return inspectFilterContext().activeFilterSummaryParts
        default:
            return []
        }
    }

    public var activeDrillInContextForSelectedSection: DrillInNavigationContext? {
        navigationStore.activeDrillInContextForSelectedSection()
    }

    public func clearActiveDrillInNavigationContext() {
        navigationStore.clearActiveDrillInNavigationContext()
    }

    public func returnToDrillInSourceSection() {
        guard let context = activeDrillInContextForSelectedSection else {
            return
        }
        navigationStore.clearActiveDrillInNavigationContext()
        requestSectionSelection(context.sourceSection)
    }

    public func requestSectionSelection(_ section: AppSection, preservingDrillInContext: Bool = false) {
        let sourceSection = selectedSection
        let hasUnsavedChanges = hasUnsavedDraftChanges(for: sourceSection)
        let unsavedSummary = unsavedDraftSummary(for: sourceSection)
        _ = navigationStore.requestSectionSelection(
            section,
            preservingDrillInContext: preservingDrillInContext,
            hasUnsavedDraftChanges: hasUnsavedChanges,
            unsavedDraftSummary: unsavedSummary
        )
        if navigationStore.selectedSection == section {
            handleSelectedSectionDidChange()
        }
    }

    public func cancelPendingSectionNavigation() {
        clearPendingSectionNavigation()
    }

    public func discardPendingSectionNavigationChanges() {
        let sourceSection = pendingSectionNavigationSource ?? selectedSection
        discardDraftChanges(for: sourceSection)
        let targetSection = navigationStore.discardPendingSectionNavigationAndSelectTarget()
        if targetSection != nil {
            handleSelectedSectionDidChange()
        }
    }

    public func discardDraftChanges(for section: AppSection) {
        switch section {
        case .channels:
            discardAllChannelDraftChanges()
        case .connectors:
            discardAllConnectorDraftChanges()
        case .models:
            discardAllProviderDraftChanges()
        default:
            break
        }
    }

    public func saveAllDraftChanges(for section: AppSection) {
        switch section {
        case .channels:
            Task { [weak self] in
                await self?.saveAllChannelDraftChanges()
            }
        case .connectors:
            Task { [weak self] in
                await self?.saveAllConnectorDraftChanges()
            }
        case .models:
            Task { [weak self] in
                await self?.saveAllProviderDraftChanges()
            }
        default:
            break
        }
    }

    public func navigateToSection(_ section: AppSection, drillInContext: DrillInNavigationContext? = nil) {
        if let drillInContext {
            activeDrillInNavigationContext = normalizedDrillInContext(drillInContext)
        }
        guard selectedSection == section else {
            requestSectionSelection(section, preservingDrillInContext: drillInContext != nil)
            return
        }
        Task { [weak self] in
            await self?.refreshDataForCurrentSection(trigger: .refresh)
        }
    }

    public func openModelsForChatRemediation() {
        navigateToSection(.models)
    }

    public func openOnboardingFromConfiguration() {
        navigateToSection(.chat)
    }

    public func openDistributionTrustSecuritySettings() {
        guard let url = URL(string: Self.distributionTrustSecuritySettingsURLString) else {
            return
        }
        let opened = NSWorkspace.shared.open(url)
        let message: String
        let level: AppNotificationLevel
        if opened {
            message = "Opened System Settings > Privacy & Security for Gatekeeper override guidance."
            level = .success
        } else {
            message = "Could not open System Settings automatically. Open Privacy & Security and run Open Anyway for PersonalAgent."
            level = .error
        }
        if selectedSection == .configuration {
            daemonStatusDetail = message
        }
        appendNotification(
            source: "configuration",
            action: "open_distribution_trust_settings",
            message: message,
            level: level
        )
    }

    public func retrySetupChecksAfterTrustGuidance() {
        appendNotification(
            source: "configuration",
            action: "retry_setup_checks_after_trust_guidance",
            message: "Retrying setup checks after trust remediation.",
            level: .progress
        )
        refreshOnboardingReadiness()
        refreshDaemonStatus()
        refreshProviderInventory()
    }

    public func performOnboardingFixNextStep() {
        guard let fixNextStep = onboardingFixNextStep else {
            refreshOnboardingReadiness()
            return
        }
        if let remediationAction = fixNextStep.remediationAction {
            performOnboardingSetupAction(remediationAction)
        } else {
            refreshOnboardingReadiness()
        }
    }

    public func performOnboardingSetupAction(_ action: OnboardingSetupAction) {
        guard action.isEnabled else {
            return
        }
        switch action.kind {
        case .openConfiguration:
            navigateToSection(.configuration)
        case .openModels:
            openModelsForChatRemediation()
        case .openChannels:
            channelsStatusMessage = "Opened Channels for setup remediation."
            navigateToSection(.channels)
        case .openConnectors:
            connectorsStatusMessage = "Opened Connectors for setup remediation."
            navigateToSection(.connectors)
        case .refreshChecks:
            refreshOnboardingReadiness()
        case .startDaemon:
            requestStartDaemon()
        case .installDaemon:
            requestInstallDaemon()
        case .repairDaemon:
            requestRepairDaemonInstallation()
        }
    }

    public func performEmptyStateRemediationAction(_ actionID: EmptyStateRemediationActionID) {
        switch actionID {
        case .openConfiguration:
            navigateToSection(.configuration)
        case .openModels:
            openModelsForChatRemediation()
        case .openChannels:
            channelsStatusMessage = "Opened Channels for remediation."
            navigateToSection(.channels)
        case .openConnectors:
            connectorsStatusMessage = "Opened Connectors for remediation."
            navigateToSection(.connectors)
        case .openTasks:
            navigateToSection(.tasks)
        case .openChat:
            navigateToSection(.chat)
        case .openAutomation:
            navigateToSection(.automation)
        case .refreshDaemonStatus:
            refreshDaemonStatus()
        case .recheckChatRoute:
            refreshChatRoutePreflight()
        case .refreshCommunications:
            refreshCommunicationsInbox()
        case .refreshAutomation:
            refreshAutomationTriggers()
            refreshAutomationFireHistory()
        case .refreshApprovals:
            refreshApprovalsInbox()
        case .refreshTasks:
            refreshTaskRunList()
        case .refreshInspect:
            refreshInspectLogs()
        case .refreshChannels:
            refreshChannelCards()
        case .refreshConnectors:
            refreshConnectorCards()
        case .refreshModels:
            refreshProviderInventory()
        case .runProviderChecks:
            runProviderConnectivityChecks()
        }
    }

    func panelProblemRemediation(for section: AppSection) -> PanelProblemRemediationContext? {
        panelProblemStore.remediationContext(
            for: section,
            retryInFlight: isPanelProblemRetryInFlight(for: section)
        )
    }

    func performPanelProblemRemediationAction(
        _ actionID: PanelProblemRemediationActionID,
        section: AppSection
    ) {
        switch actionID {
        case .openConfiguration:
            navigateToSection(.configuration)
        case .openInspect:
            inspectStatusMessage = "Opened Inspect for \(section.title) remediation."
            navigateToSection(.inspect)
        case .retry:
            guard !isPanelProblemRetryInFlight(for: section) else {
                return
            }
            switch section {
            case .chat:
                refreshChatRoutePreflight()
            case .models:
                refreshProviderInventory()
            case .channels:
                refreshChannelCards()
            case .connectors:
                refreshConnectorCards()
            case .automation:
                refreshAutomationTriggers()
                refreshAutomationFireHistory()
            case .approvals:
                refreshApprovalsInbox()
            case .tasks:
                refreshTaskRunList()
            default:
                break
            }
        }
    }

    public func confirmPendingHighImpactAction() {
        workflowMutationStore.confirmPendingHighImpactAction()
    }

    public func cancelPendingHighImpactAction() {
        workflowMutationStore.cancelPendingHighImpactAction()
    }

    public func performActiveUndoAction() {
        workflowMutationStore.performActiveUndoAction()
    }

    public func dismissActiveUndoActionPrompt() {
        workflowMutationStore.dismissActiveUndoActionPrompt()
    }

    public func presentNotificationCenter() {
        isNotificationCenterPresented = true
    }

    public func postNotification(
        source: String,
        action: String,
        message: String,
        level: AppNotificationLevel
    ) {
        notificationStore.postNotification(
            workspaceID: nonEmpty(workspaceID) ?? Self.defaultWorkspaceID,
            source: source,
            action: action,
            message: message,
            level: level
        )
    }

    public func dismissNotificationCenter() {
        isNotificationCenterPresented = false
    }

    public func dismissNotificationToast(notificationID: String, markAsRead: Bool = true) {
        notificationStore.dismissNotificationToast(
            notificationID: notificationID,
            markAsRead: markAsRead
        )
    }

    public func markNotificationRead(notificationID: String) {
        notificationStore.markNotificationRead(notificationID: notificationID)
    }

    public func markAllNotificationsRead() {
        notificationStore.markAllNotificationsRead()
    }

    public func clearNotification(notificationID: String) {
        notificationStore.clearNotification(notificationID: notificationID)
    }

    public func clearReadNotifications() {
        notificationStore.clearReadNotifications()
    }

    public func clearAllNotifications() {
        notificationStore.clearAllNotifications()
    }

    public func performNotificationInboxAction(
        _ action: NotificationInboxAction,
        notificationID: String? = nil
    ) {
        if let notificationID {
            markNotificationRead(notificationID: notificationID)
        }

        switch action.kind {
        case .openSection(let section):
            navigateToSection(section)
        }
    }

    public func openTasksForChatTraceability() {
        let route = effectiveChatRouteContext()
        var drillInChips: [String] = []
        if let runID = nonEmpty(chatLatestTurnTraceability?.runID) {
            tasksSearchSeed = runID
            tasksStatusMessage = "Opened Tasks for chat run \(runID)."
            drillInChips.append(makeDrillInChip(label: "Run", value: runID) ?? "")
        } else if let taskID = nonEmpty(chatLatestTurnTraceability?.taskID) {
            tasksSearchSeed = taskID
            tasksStatusMessage = "Opened Tasks for chat task \(taskID)."
            drillInChips.append(makeDrillInChip(label: "Task", value: taskID) ?? "")
        } else if let correlationID = nonEmpty(chatLatestTurnTraceability?.correlationID) ?? nonEmpty(chatActiveCorrelationID) {
            tasksSearchSeed = correlationID
            tasksStatusMessage = "Opened Tasks for chat correlation \(correlationID)."
            drillInChips.append(makeDrillInChip(label: "Correlation", value: correlationID) ?? "")
        } else if let modelKey = nonEmpty(route.modelKey),
                  let provider = nonEmpty(route.provider) {
            tasksSearchSeed = modelKey
            tasksStatusMessage = "Opened Tasks for chat route \(provider)/\(modelKey)."
            drillInChips.append("Route: \(provider)/\(modelKey)")
        } else if let modelKey = nonEmpty(route.modelKey) {
            tasksSearchSeed = modelKey
            tasksStatusMessage = "Opened Tasks for chat model \(modelKey)."
            drillInChips.append(makeDrillInChip(label: "Model", value: modelKey) ?? "")
        } else if let provider = nonEmpty(route.provider) {
            tasksSearchSeed = provider
            tasksStatusMessage = "Opened Tasks for chat provider \(provider)."
            drillInChips.append(makeDrillInChip(label: "Provider", value: provider) ?? "")
        } else if let taskClass = nonEmpty(route.taskClass) {
            tasksSearchSeed = taskClass
            tasksStatusMessage = "Opened Tasks for chat task class \(taskClass)."
            drillInChips.append(makeDrillInChip(label: "Task Class", value: taskClass) ?? "")
        } else {
            tasksSearchSeed = nil
            tasksStatusMessage = "Opened Tasks from chat context."
        }
        navigateToSection(
            .tasks,
            drillInContext: DrillInNavigationContext(
                sourceSection: .chat,
                destinationSection: .tasks,
                chips: drillInChips
            )
        )
    }

    public func openInspectForChatTraceability() {
        let route = effectiveChatRouteContext()
        let focusedRunID = nonEmpty(chatLatestTurnTraceability?.runID)
        var drillInChips: [String] = []
        if let focusedRunID {
            inspectStore.transitionInspectContext(
                focusedRunID: focusedRunID,
                searchSeed: nil,
                statusMessage: "Loading inspect logs for chat run \(focusedRunID)…"
            )
            drillInChips.append(makeDrillInChip(label: "Run", value: focusedRunID) ?? "")
        } else if let taskID = nonEmpty(chatLatestTurnTraceability?.taskID) {
            inspectStore.transitionInspectContext(
                focusedRunID: nil,
                searchSeed: taskID,
                statusMessage: "Opened Inspect for chat task \(taskID)."
            )
            drillInChips.append(makeDrillInChip(label: "Task", value: taskID) ?? "")
        } else if let correlationID = nonEmpty(chatLatestTurnTraceability?.correlationID) ?? nonEmpty(chatActiveCorrelationID) {
            inspectStore.transitionInspectContext(
                focusedRunID: nil,
                searchSeed: correlationID,
                statusMessage: "Opened Inspect for chat correlation \(correlationID)."
            )
            drillInChips.append(makeDrillInChip(label: "Correlation", value: correlationID) ?? "")
        } else if let routeSummary = automationRouteSummary(route) {
            if let modelKey = nonEmpty(route.modelKey) {
                inspectStore.transitionInspectContext(
                    focusedRunID: nil,
                    searchSeed: modelKey,
                    statusMessage: "Opened Inspect from chat route \(routeSummary)."
                )
                drillInChips.append(makeDrillInChip(label: "Model", value: modelKey) ?? "")
            } else if let provider = nonEmpty(route.provider) {
                inspectStore.transitionInspectContext(
                    focusedRunID: nil,
                    searchSeed: provider,
                    statusMessage: "Opened Inspect from chat route \(routeSummary)."
                )
                drillInChips.append(makeDrillInChip(label: "Provider", value: provider) ?? "")
            } else if let taskClass = nonEmpty(route.taskClass) {
                inspectStore.transitionInspectContext(
                    focusedRunID: nil,
                    searchSeed: taskClass,
                    statusMessage: "Opened Inspect from chat route \(routeSummary)."
                )
                drillInChips.append(makeDrillInChip(label: "Task Class", value: taskClass) ?? "")
            } else {
                inspectStore.transitionInspectContext(
                    focusedRunID: nil,
                    searchSeed: nil,
                    statusMessage: "Opened Inspect from chat route \(routeSummary)."
                )
            }
        } else {
            inspectStore.transitionInspectContext(
                focusedRunID: nil,
                searchSeed: nil,
                statusMessage: "Opened Inspect from chat context."
            )
        }
        navigateToSection(
            .inspect,
            drillInContext: DrillInNavigationContext(
                sourceSection: .chat,
                destinationSection: .inspect,
                chips: drillInChips
            )
        )
    }

    private func openTasksForChatTimelineItem(_ item: ChatTimelineItem) {
        if let runID = nonEmpty(item.runID) {
            tasksSearchSeed = runID
            tasksStatusMessage = "Opened Tasks for chat run \(runID)."
            navigateToSection(
                .tasks,
                drillInContext: DrillInNavigationContext(
                    sourceSection: .chat,
                    destinationSection: .tasks,
                    chips: [makeDrillInChip(label: "Run", value: runID)].compactMap { $0 }
                )
            )
            return
        }
        if let taskID = nonEmpty(item.taskID) {
            tasksSearchSeed = taskID
            tasksStatusMessage = "Opened Tasks for chat task \(taskID)."
            navigateToSection(
                .tasks,
                drillInContext: DrillInNavigationContext(
                    sourceSection: .chat,
                    destinationSection: .tasks,
                    chips: [makeDrillInChip(label: "Task", value: taskID)].compactMap { $0 }
                )
            )
            return
        }
        if let correlationID = nonEmpty(item.correlationID) {
            tasksSearchSeed = correlationID
            tasksStatusMessage = "Opened Tasks for chat correlation \(correlationID)."
            navigateToSection(
                .tasks,
                drillInContext: DrillInNavigationContext(
                    sourceSection: .chat,
                    destinationSection: .tasks,
                    chips: [makeDrillInChip(label: "Correlation", value: correlationID)].compactMap { $0 }
                )
            )
            return
        }
        openTasksForChatTraceability()
    }

    private func openInspectForChatTimelineItem(_ item: ChatTimelineItem) {
        if let runID = nonEmpty(item.runID) {
            inspectStore.transitionInspectContext(
                focusedRunID: runID,
                searchSeed: nil,
                statusMessage: "Loading inspect logs for chat run \(runID)…"
            )
            navigateToSection(
                .inspect,
                drillInContext: DrillInNavigationContext(
                    sourceSection: .chat,
                    destinationSection: .inspect,
                    chips: [makeDrillInChip(label: "Run", value: runID)].compactMap { $0 }
                )
            )
            return
        }
        if let taskID = nonEmpty(item.taskID) {
            inspectStore.transitionInspectContext(
                focusedRunID: nil,
                searchSeed: taskID,
                statusMessage: "Opened Inspect for chat task \(taskID).",
                forceResetSnapshot: true
            )
            navigateToSection(
                .inspect,
                drillInContext: DrillInNavigationContext(
                    sourceSection: .chat,
                    destinationSection: .inspect,
                    chips: [makeDrillInChip(label: "Task", value: taskID)].compactMap { $0 }
                )
            )
            return
        }
        if let correlationID = nonEmpty(item.correlationID) {
            inspectStore.transitionInspectContext(
                focusedRunID: nil,
                searchSeed: correlationID,
                statusMessage: "Opened Inspect for chat correlation \(correlationID).",
                forceResetSnapshot: true
            )
            navigateToSection(
                .inspect,
                drillInContext: DrillInNavigationContext(
                    sourceSection: .chat,
                    destinationSection: .inspect,
                    chips: [makeDrillInChip(label: "Correlation", value: correlationID)].compactMap { $0 }
                )
            )
            return
        }
        openInspectForChatTraceability()
    }

    public func refreshChatRoutePreflight() {
        Task { [weak self] in
            await self?.performChatRoutePreflightOnly()
        }
    }

    public func runChatFixAndContinueFromRouteRemediation() {
        runChatFixAndContinue(source: .routeRemediation)
    }

    public func runChatFixAndContinueFromFailureRemediation() {
        runChatFixAndContinue(source: .failureRemediation)
    }

    public func refreshChatTurnExplainability() {
        Task { [weak self] in
            await self?.performChatTurnExplainabilityRefresh(userInitiated: true)
        }
    }

    private func runChatFixAndContinue(source: ChatFixAndContinueSource) {
        guard !isChatFixAndContinueInFlight else {
            return
        }
        Task { [weak self] in
            await self?.performChatFixAndContinue(
                source: source,
                allowNavigationToRemediationOwner: true
            )
        }
    }

    private func resumePendingChatFixAndContinueIfNeeded() async {
        guard selectedSection == .chat, let pending = pendingChatFixAndContinue else {
            return
        }
        await performChatFixAndContinue(
            source: pending.source,
            allowNavigationToRemediationOwner: false
        )
    }

    private func performChatFixAndContinue(
        source: ChatFixAndContinueSource,
        allowNavigationToRemediationOwner: Bool
    ) async {
        guard !isChatStreaming, !isChatFixAndContinueInFlight else {
            return
        }
        isChatFixAndContinueInFlight = true
        defer {
            isChatFixAndContinueInFlight = false
        }

        let preservedDraft = nonEmpty(chatDraft)
            ?? nonEmpty(chatLastFailedDraft)
            ?? pendingChatFixAndContinue?.draft

        if source == .failureRemediation,
           nonEmpty(chatDraft) == nil,
           let failedDraft = nonEmpty(chatLastFailedDraft) {
            chatDraft = failedDraft
        }

        guard resolvedAuthToken() != nil else {
            presentChatFixAndContinuePendingState(
                source: source,
                requiredSection: .configuration,
                preservedDraft: preservedDraft,
                allowNavigationToRemediationOwner: allowNavigationToRemediationOwner
            )
            return
        }

        if let validationMessage = actingAsValidationMessage(for: selectedPrincipal) {
            chatFixAndContinueStatusMessage = validationMessage
            chatStatusMessage = validationMessage
            return
        }

        chatFixAndContinueStatusMessage = "Refreshing runtime and route checks…"
        await refreshDaemonLifecycleStatus(showLoadingState: false)

        guard let authToken = resolvedAuthToken() else {
            presentChatFixAndContinuePendingState(
                source: source,
                requiredSection: .configuration,
                preservedDraft: preservedDraft,
                allowNavigationToRemediationOwner: allowNavigationToRemediationOwner
            )
            return
        }

        let routeReady = await validateChatRouteBeforeSend(
            authToken: authToken,
            updateChatStatusOnSuccess: false,
            clearFailedDraftOnRouteFailure: false
        )
        guard routeReady else {
            let remediationOwner = chatFixAndContinueRemediationOwner()
            presentChatFixAndContinuePendingState(
                source: source,
                requiredSection: remediationOwner,
                preservedDraft: preservedDraft,
                allowNavigationToRemediationOwner: allowNavigationToRemediationOwner
            )
            return
        }

        pendingChatFixAndContinue = nil
        if nonEmpty(chatDraft) == nil, let preservedDraft {
            chatDraft = preservedDraft
        }

        if nonEmpty(chatDraft) != nil {
            chatFixAndContinueStatusMessage = "Route is ready. Sending your saved message…"
            sendChatDraft()
            if !isChatStreaming {
                chatFixAndContinueStatusMessage = chatStatusMessage ?? "Route is ready. Press Send to continue."
            }
        } else {
            chatFixAndContinueStatusMessage = "Route is ready. Continue in Chat."
            chatStatusMessage = "Chat route is ready. Continue in Chat."
            chatProgressDetail = nil
        }
    }

    private func presentChatFixAndContinuePendingState(
        source: ChatFixAndContinueSource,
        requiredSection: AppSection,
        preservedDraft: String?,
        allowNavigationToRemediationOwner: Bool
    ) {
        pendingChatFixAndContinue = PendingChatFixAndContinue(
            source: source,
            draft: preservedDraft,
            requiredSection: requiredSection
        )

        let ownerTitle = requiredSection.title
        let missingToken = resolvedAuthToken() == nil
        if allowNavigationToRemediationOwner {
            if requiredSection == .configuration, missingToken {
                chatFixAndContinueStatusMessage = "Set Assistant Access Token in Configuration, then return to Chat to continue automatically."
            } else {
                chatFixAndContinueStatusMessage = "Finish setup in \(ownerTitle), then return to Chat to continue automatically."
            }
            if selectedSection != requiredSection {
                navigateToSection(requiredSection)
            }
        } else {
            if requiredSection == .configuration, missingToken {
                chatFixAndContinueStatusMessage = "Assistant Access Token is still missing. Open Configuration, then return to Chat to continue."
            } else {
                chatFixAndContinueStatusMessage = "\(ownerTitle) still needs attention before Chat can continue."
            }
        }
    }

    private func chatFixAndContinueRemediationOwner() -> AppSection {
        if resolvedAuthToken() == nil {
            return .configuration
        }
        if chatRouteRemediationMessage != nil {
            return .models
        }
        let diagnosticText = [chatFailureRemediationMessage, chatStatusMessage]
            .compactMap(nonEmpty)
            .joined(separator: " ")
            .lowercased()
        if diagnosticText.contains("model")
            || diagnosticText.contains("provider")
            || diagnosticText.contains("route")
            || diagnosticText.contains("open models") {
            return .models
        }
        return .configuration
    }

    private func performChatTurnExplainabilityRefresh(userInitiated: Bool) async {
        guard let authToken = resolvedAuthToken() else {
            chatTurnContextStore.clearExplainabilityForMissingToken()
            syncChatTurnContextProjectionFromStore()
            return
        }
        let selectedActorID = selectedActorIDForChatSubmission()
        await performChatTurnExplainabilityFetch(
            authToken: authToken,
            requestedByActorID: selectedActorID,
            subjectActorID: selectedActorID,
            actingAsActorID: selectedActorID,
            userInitiated: userInitiated,
            retainPreviousOnFailure: true
        )
    }

    private func performChatTurnExplainabilityFetch(
        authToken: String,
        requestedByActorID: String?,
        subjectActorID: String?,
        actingAsActorID: String?,
        userInitiated: Bool,
        retainPreviousOnFailure: Bool
    ) async {
        guard chatTurnContextStore.markExplainabilityLoading(
            userInitiated: userInitiated,
            isAlreadyInFlight: isChatExplainabilityInFlight
        ) else {
            syncChatTurnContextProjectionFromStore()
            return
        }

        isChatExplainabilityInFlight = true
        syncChatTurnContextProjectionFromStore()
        defer {
            isChatExplainabilityInFlight = false
        }

        do {
            let response = try await daemonClient.chat.chatTurnExplain(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                taskClass: "chat",
                requestedByActorID: requestedByActorID,
                subjectActorID: subjectActorID,
                actingAsActorID: actingAsActorID
            )
            updateWorkspaceContext(from: response.workspaceID)
            chatTurnContextStore.applyExplainabilitySuccess(
                response,
                defaultWorkspaceID: workspaceID
            )
            syncChatTurnContextProjectionFromStore()
            connectionStatus = .connected
        } catch {
            if !userInitiated,
               let daemonError = error as? DaemonAPIError,
               daemonError.isConnectivityIssue {
                chatTurnContextStore.applyExplainabilityFailure(
                    message: "Chat explainability is temporarily unavailable due to a transport interruption. Turn output is still available.",
                    retainPreviousOnFailure: true
                )
                syncChatTurnContextProjectionFromStore()
                return
            }
            let message = daemonErrorMessage(
                error,
                fallbackContext: "Chat explainability failed",
                updateConnectionStatus: false,
                panelContext: .chat
            )
            chatTurnContextStore.applyExplainabilityFailure(
                message: message,
                retainPreviousOnFailure: retainPreviousOnFailure
            )
            syncChatTurnContextProjectionFromStore()
        }
    }

    public func restoreLastFailedChatDraftForRetry() {
        guard let failedDraft = nonEmpty(chatLastFailedDraft) else {
            return
        }
        chatDraft = failedDraft
        chatFailureRemediationMessage = nil
        chatFixAndContinueStatusMessage = nil
        chatStatusMessage = "Restored your last message. Press Send to retry."
        chatProgressDetail = nil
    }

    public func chatWorkflowCardSummary() -> WorkflowCardSummary {
        let traceability = chatLatestTurnTraceability
        let latestItem = latestChatWorkflowTimelineItem()
        let primaryRecoveryActionTitle = chatPrimaryRecoveryActionTitle(for: latestItem)
        return WorkflowCardSummary(
            whatHappened: chatWhatHappenedSummary(traceability: traceability, latestItem: latestItem),
            whatNeedsAction: chatWhatNeedsActionSummary(traceability: traceability, latestItem: latestItem),
            whatNext: chatWhatNextSummary(
                traceability: traceability,
                latestItem: latestItem,
                primaryRecoveryActionTitle: primaryRecoveryActionTitle
            )
        )
    }

    public func chatTimelineActionStatus(for itemID: String) -> String? {
        nonEmpty(chatTimelineActionStatusByItemID[itemID])
    }

    public func isChatTimelineActionInFlight(itemID: String) -> Bool {
        chatTimelineActionInFlightItemIDs.contains(itemID)
    }

    public func chatApprovalInboxItem(for item: ChatTimelineItem) -> ApprovalInboxItem? {
        guard let approvalID = resolvedChatApprovalRequestID(for: item) else {
            return nil
        }
        return approvalInboxItems.first(where: { inboxItem in
            inboxItem.id.caseInsensitiveCompare(approvalID) == .orderedSame
        })
    }

    public func chatApprovalRequestID(for item: ChatTimelineItem) -> String? {
        resolvedChatApprovalRequestID(for: item)
    }

    public func chatApprovalRequiredPhrase(for item: ChatTimelineItem) -> String {
        if let approval = chatApprovalInboxItem(for: item) {
            return approvalRequiredPhrase(for: approval)
        }
        return "GO AHEAD"
    }

    public func chatApprovalApprovePhraseValidationMessage(
        phrase: String,
        item: ChatTimelineItem
    ) -> String? {
        let requiredPhrase = chatApprovalRequiredPhrase(for: item)
        let trimmedPhrase = phrase.trimmingCharacters(in: .whitespacesAndNewlines)
        guard trimmedPhrase == requiredPhrase else {
            return "Approve requires exact phrase `\(requiredPhrase)`. Use `Use Required Phrase` or type it exactly."
        }
        return nil
    }

    public func loadChatApprovalContextIfNeeded(for item: ChatTimelineItem) {
        guard item.kind == .approvalRequest || item.kind == .approvalDecision else {
            return
        }
        if hasLoadedApprovalsInbox, chatApprovalInboxItem(for: item) != nil {
            return
        }
        guard !isApprovalsLoading else {
            return
        }
        refreshApprovalsInbox()
    }

    public func chatApprovalActionStatus(for item: ChatTimelineItem) -> String? {
        guard let approvalID = resolvedChatApprovalRequestID(for: item) else {
            return nil
        }
        return nonEmpty(approvalsActionStatusByID[approvalID])
    }

    public func isChatApprovalDecisionInFlight(for item: ChatTimelineItem) -> Bool {
        guard let approvalID = resolvedChatApprovalRequestID(for: item) else {
            return false
        }
        return approvalDecisionInFlightIDs.contains(approvalID)
    }

    public func chatInlineApprovalFastPathBlockedReason(for approval: ApprovalInboxItem) -> String? {
        guard approval.decisionState == .pending else {
            return "Decision is already finalized. Open Approvals to review history."
        }
        switch approval.riskLevel {
        case .policy:
            return nil
        case .destructive:
            return "High-risk approvals require full review in Approvals before submitting."
        case .other:
            return "This approval requires full review in Approvals before submitting."
        }
    }

    public func chatTimelineActions(for item: ChatTimelineItem) -> [ChatTimelineActionItem] {
        switch item.kind {
        case .toolCall, .toolResult:
            let isFailureState = item.state == .failed || item.state == .blocked
            let canRetry = !isChatStreaming && isFailureState && nonEmpty(chatLastFailedDraft) != nil
            let canCancel = isChatStreaming && (item.state == .inFlight || item.state == .pending)
            let canInspect = nonEmpty(item.runID) != nil
                || nonEmpty(item.taskID) != nil
                || nonEmpty(item.correlationID) != nil
                || chatLatestTurnTraceability != nil
            var actions: [ChatTimelineActionItem] = [
                ChatTimelineActionItem(
                    id: "retry_turn",
                    title: "Retry Turn",
                    intent: .retryTurn,
                    style: .primary,
                    enabled: canRetry,
                    disabledReason: canRetry ? nil : "Retry is available after a failed or blocked turn."
                ),
                ChatTimelineActionItem(
                    id: "open_inspect",
                    title: "Open Inspect",
                    intent: .openInspect,
                    style: .secondary,
                    enabled: canInspect,
                    disabledReason: canInspect ? nil : "Inspect is unavailable until run context is available."
                ),
                ChatTimelineActionItem(
                    id: "cancel_turn",
                    title: "Cancel",
                    intent: .cancelTurn,
                    style: .destructive,
                    enabled: canCancel,
                    disabledReason: canCancel ? nil : "Turn is not running."
                )
            ]
            if let remediationAction = chatToolRemediationAction(for: item) {
                actions.append(remediationAction)
            }
            if let approvalRequestID = nonEmpty(item.approvalRequestID) {
                actions.append(
                    ChatTimelineActionItem(
                        id: "open_approvals_tool_\(approvalRequestID)",
                        title: "Open Approvals",
                        intent: .openApprovals,
                        style: .secondary,
                        enabled: true
                    )
                )
            }
            return actions
        case .approvalRequest:
            let canOpenApprovals = resolvedChatApprovalRequestID(for: item) != nil
            let canOpenTasks = nonEmpty(item.runID) != nil
                || nonEmpty(item.taskID) != nil
                || nonEmpty(item.correlationID) != nil
                || chatLatestTurnTraceability != nil
            let resumeDisabledReason = chatResumeDisabledReason(for: item)
            return [
                ChatTimelineActionItem(
                    id: "open_approvals",
                    title: "Open Approvals",
                    intent: .openApprovals,
                    style: .primary,
                    enabled: canOpenApprovals,
                    disabledReason: canOpenApprovals ? nil : "Approval request ID is missing."
                ),
                ChatTimelineActionItem(
                    id: "resume_turn",
                    title: "Resume Turn",
                    intent: .resumeTurn,
                    style: .secondary,
                    enabled: resumeDisabledReason == nil,
                    disabledReason: resumeDisabledReason
                ),
                ChatTimelineActionItem(
                    id: "open_tasks",
                    title: "Open Tasks",
                    intent: .openTasks,
                    style: .secondary,
                    enabled: canOpenTasks,
                    disabledReason: canOpenTasks ? nil : "Task context is unavailable until a run is created."
                ),
                ChatTimelineActionItem(
                    id: "open_inspect",
                    title: "Open Inspect",
                    intent: .openInspect,
                    style: .secondary,
                    enabled: canOpenTasks,
                    disabledReason: canOpenTasks ? nil : "Inspect is unavailable until traceability context exists."
                )
            ]
        case .approvalDecision:
            let canOpenTasks = nonEmpty(item.runID) != nil
                || nonEmpty(item.taskID) != nil
                || nonEmpty(item.correlationID) != nil
                || chatLatestTurnTraceability != nil
            let resumeDisabledReason = chatResumeDisabledReason(for: item)
            return [
                ChatTimelineActionItem(
                    id: "resume_turn",
                    title: "Resume Turn",
                    intent: .resumeTurn,
                    style: .primary,
                    enabled: resumeDisabledReason == nil,
                    disabledReason: resumeDisabledReason
                ),
                ChatTimelineActionItem(
                    id: "open_tasks",
                    title: "Open Tasks",
                    intent: .openTasks,
                    style: .secondary,
                    enabled: canOpenTasks,
                    disabledReason: canOpenTasks ? nil : "Task context is unavailable until a run is created."
                ),
                ChatTimelineActionItem(
                    id: "open_inspect",
                    title: "Open Inspect",
                    intent: .openInspect,
                    style: .secondary,
                    enabled: canOpenTasks,
                    disabledReason: canOpenTasks ? nil : "Inspect is unavailable until traceability context exists."
                )
            ]
        default:
            return []
        }
    }

    private func latestChatWorkflowTimelineItem() -> ChatTimelineItem? {
        if let workflowItem = chatTimelineItems.last(where: { item in
            item.kind != .userMessage && item.kind != .assistantMessage
        }) {
            return workflowItem
        }
        return chatTimelineItems.last
    }

    private func chatPrimaryRecoveryActionTitle(for item: ChatTimelineItem?) -> String? {
        guard let item else {
            return nil
        }
        let actions = chatTimelineActions(for: item)
        if let primary = actions.first(where: { $0.style == .primary && $0.enabled }) {
            return primary.title
        }
        if let enabled = actions.first(where: \.enabled) {
            return enabled.title
        }
        return nil
    }

    private func chatWhatHappenedSummary(
        traceability: ChatTaskRunTraceabilityItem?,
        latestItem: ChatTimelineItem?
    ) -> String {
        if isChatStreaming {
            return isAdvancedInformationDensityEnabled
                ? "Assistant is actively executing your latest request."
                : "Assistant is working on your latest request."
        }
        if let latestItem {
            switch latestItem.state {
            case .failed:
                if isAdvancedInformationDensityEnabled {
                    return "Latest workflow step failed: \(truncateText(latestItem.summary, limit: 160))"
                }
                return "Something went wrong: \(truncateText(latestItem.summary, limit: 160))"
            case .blocked:
                if isAdvancedInformationDensityEnabled {
                    return "Latest workflow step is blocked: \(truncateText(latestItem.summary, limit: 160))"
                }
                return "This request is blocked: \(truncateText(latestItem.summary, limit: 160))"
            case .inFlight, .pending:
                return isAdvancedInformationDensityEnabled
                    ? "Latest workflow step is still running."
                    : "The assistant is still working on this request."
            case .completed:
                if isAdvancedInformationDensityEnabled {
                    return "Latest workflow step completed: \(truncateText(latestItem.summary, limit: 160))"
                }
                return "Latest action completed: \(truncateText(latestItem.summary, limit: 160))"
            }
        }
        if let traceability {
            if traceability.approvalRequired {
                return "Assistant prepared an action and is waiting for approval."
            }
            if traceability.clarificationRequired {
                return "Assistant needs clarification before continuing the action."
            }
            if traceability.hasTaskOrRunIdentity {
                return isAdvancedInformationDensityEnabled
                    ? "Recent turn includes linked workflow context for audit and follow-up."
                    : "Related workflow details are available if you want to review what ran."
            }
        }
        if modelRouteSummary != nil {
            return isAdvancedInformationDensityEnabled
                ? "Route preview is ready for the next workflow request."
                : "Assistant is ready for your next request."
        }
        return "No workflow action has started yet."
    }

    private func chatWhatNeedsActionSummary(
        traceability: ChatTaskRunTraceabilityItem?,
        latestItem: ChatTimelineItem?
    ) -> String {
        if isChatStreaming {
            return isAdvancedInformationDensityEnabled
                ? "No action is required unless you need to interrupt this run."
                : "No action is required unless you want to stop this request."
        }
        if let traceability {
            if traceability.approvalRequired {
                return "Review the approval request and submit Approve or Reject to continue."
            }
            if traceability.clarificationRequired {
                return "Reply with the missing clarification requested by the assistant."
            }
        }
        if let latestItem {
            switch latestItem.state {
            case .failed:
                return isAdvancedInformationDensityEnabled
                    ? "Review failure details and choose a recovery action before retrying."
                    : "Review the issue, then choose a recovery action before retrying."
            case .blocked:
                return isAdvancedInformationDensityEnabled
                    ? "Resolve the blocker, then resume or retry the workflow."
                    : "Resolve the blocker, then continue this request."
            case .pending, .inFlight:
                return "No action is required while execution is in progress."
            case .completed:
                return "No immediate action is required."
            }
        }
        if chatFailureRemediationMessage != nil {
            return "Resolve setup/runtime guidance, then retry the same request."
        }
        return isAdvancedInformationDensityEnabled
            ? "Send a message to start a workflow turn."
            : "Send a message to get started."
    }

    private func chatWhatNextSummary(
        traceability: ChatTaskRunTraceabilityItem?,
        latestItem: ChatTimelineItem?,
        primaryRecoveryActionTitle: String?
    ) -> String {
        if isChatStreaming {
            return isAdvancedInformationDensityEnabled
                ? "Wait for completion, then open Tasks or Inspect to verify the outcome."
                : "Wait for completion, then open Tasks to verify the outcome."
        }
        if let traceability {
            if traceability.approvalRequired {
                if isAdvancedInformationDensityEnabled,
                   let approvalRequestID = nonEmpty(traceability.approvalRequestID) {
                    return "Open Approvals (\(approvalRequestID)), submit a decision, then resume the turn."
                }
                return "Open Approvals, submit a decision, then resume the turn."
            }
            if traceability.clarificationRequired {
                if let prompt = nonEmpty(traceability.clarificationPrompt) {
                    return "Reply with clarification (\(truncateText(prompt, limit: 140))), then retry."
                }
                return "Reply with clarification, then retry the action."
            }
        }
        if let latestItem {
            switch latestItem.state {
            case .failed, .blocked:
                if let primaryRecoveryActionTitle {
                    if isAdvancedInformationDensityEnabled {
                        return "Use `\(primaryRecoveryActionTitle)` or open related workflow context for remediation."
                    }
                    return "Use `\(primaryRecoveryActionTitle)` or open related details to continue."
                }
                return isAdvancedInformationDensityEnabled
                    ? "Open related Tasks/Inspect context and resolve blockers before retrying."
                    : "Open related Tasks or Inspect details, resolve blockers, then retry."
            case .completed:
                if nonEmpty(latestItem.runID) != nil || nonEmpty(latestItem.taskID) != nil {
                    return isAdvancedInformationDensityEnabled
                        ? "Open Related Tasks or Inspect to confirm completion details."
                        : "Open Related Tasks to confirm what completed."
                }
                return "Continue with your next message when ready."
            case .pending, .inFlight:
                return "Wait for execution to finish, or use Cancel if you need to stop."
            }
        }
        if modelRouteSummary != nil {
            return isAdvancedInformationDensityEnabled
                ? "Send your next message to execute with the active route."
                : "Send your next message when you're ready."
        }
        return isAdvancedInformationDensityEnabled
            ? "Send your first message to start a workflow."
            : "Send your first message to get started."
    }

    private func chatResumeDisabledReason(for item: ChatTimelineItem) -> String? {
        if isChatStreaming {
            return "Turn is already running."
        }
        guard nonEmpty(chatLastFailedDraft) != nil else {
            return "No interrupted turn draft is available to resume."
        }
        if let approval = chatApprovalInboxItem(for: item) {
            if approval.decisionState == .pending {
                return "Submit approve or reject before resuming the turn."
            }
            if let decisionOutcome = approval.decisionOutcome {
                switch decisionOutcome {
                case .rejected:
                    return "Turn cannot resume after rejection. Update your prompt and retry."
                default:
                    break
                }
            }
            return nil
        }
        if item.kind == .approvalRequest {
            return "Load approval context or open Approvals before resuming."
        }
        if item.kind == .approvalDecision, item.state == .failed {
            return "Turn cannot resume after rejection. Update your prompt and retry."
        }
        return nil
    }

    private func resolvedChatApprovalRequestID(for item: ChatTimelineItem) -> String? {
        if let approvalID = nonEmpty(item.approvalRequestID) {
            return approvalID
        }
        return item.details
            .first(where: { detail in
                detail.label.trimmingCharacters(in: .whitespacesAndNewlines).caseInsensitiveCompare("Approval Request") == .orderedSame
            })
            .flatMap { nonEmpty($0.value) }
    }

    private func chatToolRemediationAction(for item: ChatTimelineItem) -> ChatTimelineActionItem? {
        guard item.kind == .toolCall || item.kind == .toolResult else {
            return nil
        }
        guard item.state == .failed || item.state == .blocked else {
            return nil
        }

        let diagnosticText = ([item.summary, item.content] + item.details.map(\.value))
            .compactMap(nonEmpty)
            .joined(separator: " ")
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .lowercased()
        guard !diagnosticText.isEmpty else {
            return nil
        }

        let connectorIndicators = [
            "permission",
            "tcc",
            "accessibility",
            "calendar",
            "mail",
            "contacts",
            "apple events",
            "eventkit",
            "connector"
        ]
        if connectorIndicators.contains(where: diagnosticText.contains) {
            return ChatTimelineActionItem(
                id: "open_connectors",
                title: "Open Connectors",
                intent: .openConnectors
            )
        }

        let channelIndicators = [
            "channel",
            "delivery",
            "destination",
            "thread",
            "twilio",
            "imessage",
            "sms",
            "voice"
        ]
        if channelIndicators.contains(where: diagnosticText.contains) {
            return ChatTimelineActionItem(
                id: "open_channels",
                title: "Open Channels",
                intent: .openChannels
            )
        }

        let configurationIndicators = [
            "route",
            "model",
            "provider",
            "auth",
            "token",
            "unauthorized",
            "service_not_configured",
            "no enabled models",
            "setup"
        ]
        if configurationIndicators.contains(where: diagnosticText.contains) {
            return ChatTimelineActionItem(
                id: "open_configuration",
                title: "Open Configuration",
                intent: .openConfiguration
            )
        }

        return nil
    }

    public func performChatTimelineAction(itemID: String, intent: ChatTimelineActionIntent) {
        guard let item = chatTimelineItems.first(where: { $0.id == itemID }) else {
            return
        }
        chatTimelineActionInFlightItemIDs.insert(itemID)
        defer {
            chatTimelineActionInFlightItemIDs.remove(itemID)
        }
        switch intent {
        case .resumeTurn:
            if let disabledReason = chatResumeDisabledReason(for: item) {
                chatTimelineActionStatusByItemID[itemID] = disabledReason
                return
            }
            restoreLastFailedChatDraftForRetry()
            sendChatDraft()
            chatTimelineActionStatusByItemID[itemID] = isChatStreaming
                ? "Resuming turn with your restored prompt."
                : (chatStatusMessage ?? "Resume requested.")
        case .retryTurn:
            restoreLastFailedChatDraftForRetry()
            chatTimelineActionStatusByItemID[itemID] = "Restored last failed prompt so you can retry."
        case .cancelTurn:
            interruptActiveChat()
            chatTimelineActionStatusByItemID[itemID] = "Interrupt requested for the active turn."
        case .openApprovals:
            let approvalID = resolvedChatApprovalRequestID(for: item)
            approvalsSearchSeed = approvalID
            approvalsStatusMessage = approvalID.map { "Opened Approvals for request \($0)." } ?? "Opened Approvals from chat context."
            navigateToSection(
                .approvals,
                drillInContext: DrillInNavigationContext(
                    sourceSection: .chat,
                    destinationSection: .approvals,
                    chips: [makeDrillInChip(label: "Approval", value: approvalID)].compactMap { $0 }
                )
            )
            chatTimelineActionStatusByItemID[itemID] = approvalID.map { "Opened Approvals (\($0))." } ?? "Opened Approvals."
        case .openTasks:
            openTasksForChatTimelineItem(item)
            chatTimelineActionStatusByItemID[itemID] = "Opened Tasks for related workflow context."
        case .openInspect:
            openInspectForChatTimelineItem(item)
            chatTimelineActionStatusByItemID[itemID] = "Opened Inspect for related trace context."
        case .openConfiguration:
            chatTimelineActionStatusByItemID[itemID] = "Opened Configuration for chat remediation."
            navigateToSection(.configuration)
        case .openConnectors:
            chatTimelineActionStatusByItemID[itemID] = "Opened Connectors for permission remediation."
            navigateToSection(.connectors)
        case .openChannels:
            chatTimelineActionStatusByItemID[itemID] = "Opened Channels for delivery remediation."
            navigateToSection(.channels)
        }
    }

    public func sendChatDraft() {
        guard !isChatStreaming else {
            return
        }
        let trimmed = chatDraft.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            return
        }
        pendingChatFixAndContinue = nil
        isChatFixAndContinueInFlight = false
        chatFixAndContinueStatusMessage = nil
        guard resolvedAuthToken() != nil else {
            chatStatusMessage = "Set Assistant Access Token before sending chat turns."
            chatProgressDetail = nil
            chatRouteRemediationMessage = nil
            chatFailureRemediationMessage = nil
            return
        }
        if let validationMessage = actingAsValidationMessage(for: selectedPrincipal) {
            chatStatusMessage = validationMessage
            chatProgressDetail = nil
            chatRouteRemediationMessage = nil
            chatFailureRemediationMessage = nil
            return
        }

        isChatStreaming = true
        chatRouteRemediationMessage = nil
        chatFailureRemediationMessage = nil
        chatLastFailedDraft = nil
        chatTimelineActionStatusByItemID = [:]
        chatTimelineActionInFlightItemIDs = []
        chatStatusMessage = "Checking chat route…"
        chatProgressDetail = "Validating model availability before send…"
        chatOrchestrationStore.turnTask?.cancel()
        chatOrchestrationStore.turnTask = Task { [weak self] in
            await self?.performChatSendWithPreflight(draft: trimmed)
        }
    }

    public func retryChatRealtimeStream() {
        guard !isChatStreaming else {
            chatStatusMessage = "Wait for the active turn to finish before retrying realtime."
            chatProgressDetail = nil
            return
        }
        guard !isChatRealtimeRetryInFlight else {
            return
        }
        guard let authToken = resolvedAuthToken() else {
            chatStatusMessage = "Set Assistant Access Token before retrying realtime."
            chatProgressDetail = nil
            return
        }

        isChatRealtimeRetryInFlight = true
        chatProgressDetail = "Retrying realtime stream connectivity…"
        Task { [weak self] in
            await self?.performChatRealtimeRetry(authToken: authToken)
        }
    }

    public func interruptActiveChat() {
        guard isChatStreaming, !isChatInterruptInFlight else {
            return
        }
        Task { [weak self] in
            await self?.performChatInterrupt()
        }
    }

    public func refreshInspectLogs() {
        Task { [weak self] in
            guard let self else {
                return
            }
            let startedAtNanoseconds = DispatchTime.now().uptimeNanoseconds
            await self.fetchInspectLogs()
            let elapsedNanoseconds = DispatchTime.now().uptimeNanoseconds - startedAtNanoseconds
            self.panelLatencyStore.recordPanelLatencySample(
                section: .inspect,
                category: .refresh,
                durationMS: Int(elapsedNanoseconds / 1_000_000)
            )
        }
    }

    public func refreshCommunicationsInbox() {
        Task { [weak self] in
            guard let self else {
                return
            }
            let startedAtNanoseconds = DispatchTime.now().uptimeNanoseconds
            await self.fetchCommunicationsInbox()
            await self.fetchCommunicationAttempts(threadID: self.communicationsStore.communicationAttemptContextThreadID)
            let elapsedNanoseconds = DispatchTime.now().uptimeNanoseconds - startedAtNanoseconds
            self.panelLatencyStore.recordPanelLatencySample(
                section: .communications,
                category: .refresh,
                durationMS: Int(elapsedNanoseconds / 1_000_000)
            )
        }
    }

    public func refreshCommunicationAttempts(threadID: String?) {
        let normalizedThreadID = communicationsStore.setCommunicationAttemptContextThreadID(threadID)
        Task { [weak self] in
            await self?.fetchCommunicationAttempts(threadID: normalizedThreadID)
        }
    }

    public func sendCommunication(
        sourceChannel: String,
        destination: String?,
        message: String,
        threadID: String? = nil,
        connectorID: String? = nil
    ) {
        Task { [weak self] in
            await self?.performCommunicationSend(
                sourceChannel: sourceChannel,
                destination: destination,
                message: message,
                threadID: threadID,
                connectorID: connectorID
            )
        }
    }

    public func openInspectForCommunicationThread(_ item: CommunicationThreadItem) {
        inspectStore.transitionInspectContext(
            focusedRunID: nil,
            searchSeed: item.id,
            statusMessage: "Opened Inspect for communication thread \(item.id).",
            forceResetSnapshot: true
        )
        navigateToSection(
            .inspect,
            drillInContext: DrillInNavigationContext(
                sourceSection: .communications,
                destinationSection: .inspect,
                chips: [
                    makeDrillInChip(label: "Thread", value: item.id),
                    makeDrillInChip(label: "Channel", value: logicalCommunicationChannelID(rawChannelID: item.channel))
                ].compactMap { $0 }
            )
        )
    }

    public func openInspectForCommunicationEvent(_ item: CommunicationEventItem) {
        inspectStore.transitionInspectContext(
            focusedRunID: nil,
            searchSeed: item.id,
            statusMessage: "Opened Inspect for communication event \(item.id).",
            forceResetSnapshot: true
        )
        navigateToSection(
            .inspect,
            drillInContext: DrillInNavigationContext(
                sourceSection: .communications,
                destinationSection: .inspect,
                chips: [
                    makeDrillInChip(label: "Event", value: item.id),
                    makeDrillInChip(label: "Thread", value: item.threadID)
                ].compactMap { $0 }
            )
        )
    }

    public func openInspectForCommunicationCallSession(_ item: CommunicationCallSessionItem) {
        inspectStore.transitionInspectContext(
            focusedRunID: nil,
            searchSeed: nonEmpty(item.providerCallID) ?? item.id,
            statusMessage: "Opened Inspect for call session \(item.id).",
            forceResetSnapshot: true
        )
        navigateToSection(
            .inspect,
            drillInContext: DrillInNavigationContext(
                sourceSection: .communications,
                destinationSection: .inspect,
                chips: [
                    makeDrillInChip(label: "Call", value: item.id),
                    makeDrillInChip(label: "Provider Call", value: item.providerCallID)
                ].compactMap { $0 }
            )
        )
    }

    public func openChannelsForCommunicationChannel(_ channel: String) {
        let normalizedChannel = logicalCommunicationChannelID(rawChannelID: channel)
        channelsStatusMessage = "Opened Channels for communication channel \(normalizedChannel)."
        navigateToSection(
            .channels,
            drillInContext: DrillInNavigationContext(
                sourceSection: .communications,
                destinationSection: .channels,
                chips: [makeDrillInChip(label: "Channel", value: normalizedChannel)].compactMap { $0 }
            )
        )
    }

    public func openInspectForRuntimePluginLifecycle(_ item: RuntimePluginLifecycleEventItem) {
        inspectStore.transitionInspectContext(
            focusedRunID: nil,
            searchSeed: item.pluginID,
            statusMessage: "Opened Inspect for runtime plugin \(item.pluginID).",
            forceResetSnapshot: true
        )
        navigateToSection(.inspect)
    }

    public func openInspectForCapabilityGrant(_ item: CapabilityGrantItem) {
        inspectStore.transitionInspectContext(
            focusedRunID: nil,
            searchSeed: nonEmpty(item.id) ?? nonEmpty(item.capabilityKey) ?? nonEmpty(item.actorID),
            statusMessage: "Opened Inspect for capability grant \(item.id).",
            forceResetSnapshot: true
        )
        navigateToSection(.inspect)
    }

    public func openInspectForTrustReceipt(
        receiptID: String,
        preferredSeed: String?,
        fallbackSeed: String?
    ) {
        inspectStore.transitionInspectContext(
            focusedRunID: nil,
            searchSeed: nonEmpty(preferredSeed) ?? nonEmpty(fallbackSeed) ?? nonEmpty(receiptID),
            statusMessage: "Opened Inspect for trust receipt \(receiptID).",
            forceResetSnapshot: true
        )
        navigateToSection(.inspect)
    }

    public func openInspectForTrustReceiptAuditLink(_ auditLink: TrustReceiptAuditLinkItem) {
        inspectStore.transitionInspectContext(
            focusedRunID: nil,
            searchSeed: nonEmpty(auditLink.id),
            statusMessage: "Opened Inspect for receipt audit \(auditLink.id).",
            forceResetSnapshot: true
        )
        navigateToSection(.inspect)
    }

    public func openRuntimeDiagnosticsForPluginLifecycle(_ item: RuntimePluginLifecycleEventItem) {
        let normalizedPluginID = nonEmpty(item.pluginID) ?? "unknown"
        switch item.kind.lowercased() {
        case "channel":
            channelsStatusMessage = "Opened Channels for runtime plugin \(normalizedPluginID)."
            navigateToSection(.channels)
        case "connector":
            connectorsStatusMessage = "Opened Connectors for runtime plugin \(normalizedPluginID)."
            navigateToSection(.connectors)
        default:
            inspectStore.transitionInspectContext(
                focusedRunID: nil,
                searchSeed: normalizedPluginID,
                statusMessage: "Opened Inspect for runtime plugin \(normalizedPluginID).",
                forceResetSnapshot: true
            )
            navigateToSection(.inspect)
        }
    }

    public func openTasksForCommunicationAttempt(_ item: CommunicationDeliveryAttemptItem) {
        var drillInChips: [String] = []
        if let runID = nonEmpty(item.runID) {
            tasksSearchSeed = runID
            tasksStatusMessage = "Opened Tasks for delivery run \(runID)."
            drillInChips.append(makeDrillInChip(label: "Run", value: runID) ?? "")
        } else if let taskID = nonEmpty(item.taskID) {
            tasksSearchSeed = taskID
            tasksStatusMessage = "Opened Tasks for delivery task \(taskID)."
            drillInChips.append(makeDrillInChip(label: "Task", value: taskID) ?? "")
        } else if let operationID = nonEmpty(item.operationID) {
            tasksSearchSeed = operationID
            tasksStatusMessage = "Opened Tasks for delivery operation \(operationID)."
            drillInChips.append(makeDrillInChip(label: "Operation", value: operationID) ?? "")
        } else if let threadID = nonEmpty(item.threadID) {
            tasksSearchSeed = threadID
            tasksStatusMessage = "Opened Tasks for delivery thread \(threadID)."
            drillInChips.append(makeDrillInChip(label: "Thread", value: threadID) ?? "")
        } else {
            tasksSearchSeed = nil
            tasksStatusMessage = "Opened Tasks from delivery attempt context."
        }
        navigateToSection(
            .tasks,
            drillInContext: DrillInNavigationContext(
                sourceSection: .communications,
                destinationSection: .tasks,
                chips: drillInChips
            )
        )
    }

    public func openTaskDraftForCommunicationThread(_ item: CommunicationThreadItem) {
        let normalizedThreadID = nonEmpty(item.id) ?? "unknown-thread"
        let normalizedChannel = logicalCommunicationChannelID(rawChannelID: item.channel)
        let threadTitle = nonEmpty(item.title) ?? "Thread \(normalizedThreadID)"
        var descriptionParts: [String] = [
            "Communication follow-up for thread \(normalizedThreadID).",
            "Channel: \(normalizedChannel).",
        ]
        if let connectorID = nonEmpty(item.connectorID) {
            descriptionParts.append("Connector: \(connectorID).")
        }
        if !item.participantAddresses.isEmpty {
            descriptionParts.append("Participants: \(item.participantAddresses.joined(separator: ", ")).")
        }
        if let preview = nonEmpty(item.lastBodyPreview) {
            descriptionParts.append("Latest preview: \(preview)")
        }
        let seedPrincipal = preferredTaskDraftPrincipal()
        taskSubmitDraftSeed = TaskSubmitDraftSeed(
            title: "Follow up: \(threadTitle)",
            description: descriptionParts.joined(separator: " "),
            taskClass: "chat",
            requestedByActorID: seedPrincipal,
            subjectPrincipalActorID: seedPrincipal
        )
        tasksSearchSeed = normalizedThreadID
        tasksStatusMessage = "Opened task draft for communication thread \(normalizedThreadID)."
        navigateToSection(
            .tasks,
            drillInContext: DrillInNavigationContext(
                sourceSection: .communications,
                destinationSection: .tasks,
                chips: [
                    makeDrillInChip(label: "Thread", value: normalizedThreadID),
                    makeDrillInChip(label: "Channel", value: normalizedChannel)
                ].compactMap { $0 }
            )
        )
    }

    public func clearTaskSubmitDraftSeed() {
        taskSubmitDraftSeed = nil
    }

    private func preferredTaskDraftPrincipal() -> String {
        let preferred = nonEmpty(activePrincipalLabel)
        let options = taskSubmissionPrincipalOptions
            .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
            .filter { !$0.isEmpty }
        if let preferred, let match = options.first(where: { $0.caseInsensitiveCompare(preferred) == .orderedSame }) {
            return match
        }
        if let first = options.first {
            return first
        }
        return "default"
    }

    public func openInspectForCommunicationAttempt(_ item: CommunicationDeliveryAttemptItem) {
        let focusedRunID = nonEmpty(item.runID)
        var drillInChips: [String] = []
        if let focusedRunID {
            inspectStore.transitionInspectContext(
                focusedRunID: focusedRunID,
                searchSeed: nil,
                statusMessage: "Loading inspect logs for delivery run \(focusedRunID)…"
            )
            drillInChips.append(makeDrillInChip(label: "Run", value: focusedRunID) ?? "")
        } else if let operationID = nonEmpty(item.operationID) {
            inspectStore.transitionInspectContext(
                focusedRunID: nil,
                searchSeed: operationID,
                statusMessage: "Opened Inspect for delivery operation \(operationID).",
                forceResetSnapshot: true
            )
            drillInChips.append(makeDrillInChip(label: "Operation", value: operationID) ?? "")
        } else if let attemptID = nonEmpty(item.id) {
            inspectStore.transitionInspectContext(
                focusedRunID: nil,
                searchSeed: attemptID,
                statusMessage: "Opened Inspect for delivery attempt \(attemptID).",
                forceResetSnapshot: true
            )
            drillInChips.append(makeDrillInChip(label: "Attempt", value: attemptID) ?? "")
        } else {
            inspectStore.transitionInspectContext(
                focusedRunID: nil,
                searchSeed: nil,
                statusMessage: "Opened Inspect from delivery attempt context.",
                forceResetSnapshot: true
            )
        }
        navigateToSection(
            .inspect,
            drillInContext: DrillInNavigationContext(
                sourceSection: .communications,
                destinationSection: .inspect,
                chips: drillInChips
            )
        )
    }

    public func openChatForCommunicationContinuity(_ item: CommunicationContinuityItem) {
        let channelLabel = item.channel
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .capitalized
        var drillInChips: [String] = []
        drillInChips.append(makeDrillInChip(label: "Channel", value: item.channel) ?? "")
        if let threadID = nonEmpty(item.threadID) {
            drillInChips.append(makeDrillInChip(label: "Thread", value: threadID) ?? "")
        }
        if let correlationID = nonEmpty(item.correlationID) {
            drillInChips.append(makeDrillInChip(label: "Correlation", value: correlationID) ?? "")
        }
        chatStatusMessage = "Opened Chat for \(channelLabel) continuity."
        navigateToSection(
            .chat,
            drillInContext: DrillInNavigationContext(
                sourceSection: .communications,
                destinationSection: .chat,
                chips: drillInChips
            )
        )
    }

    public func openTasksForCommunicationContinuity(_ item: CommunicationContinuityItem) {
        var drillInChips: [String] = []
        if let runID = nonEmpty(item.runID) {
            tasksSearchSeed = runID
            tasksStatusMessage = "Opened Tasks for continuity run \(runID)."
            drillInChips.append(makeDrillInChip(label: "Run", value: runID) ?? "")
        } else if let taskID = nonEmpty(item.taskID) {
            tasksSearchSeed = taskID
            tasksStatusMessage = "Opened Tasks for continuity task \(taskID)."
            drillInChips.append(makeDrillInChip(label: "Task", value: taskID) ?? "")
        } else if let correlationID = nonEmpty(item.correlationID) {
            tasksSearchSeed = correlationID
            tasksStatusMessage = "Opened Tasks for continuity correlation \(correlationID)."
            drillInChips.append(makeDrillInChip(label: "Correlation", value: correlationID) ?? "")
        } else {
            tasksSearchSeed = nil
            tasksStatusMessage = "Opened Tasks from conversation continuity context."
        }
        drillInChips.append(makeDrillInChip(label: "Channel", value: item.channel) ?? "")
        navigateToSection(
            .tasks,
            drillInContext: DrillInNavigationContext(
                sourceSection: .communications,
                destinationSection: .tasks,
                chips: drillInChips
            )
        )
    }

    public func openInspectForCommunicationContinuity(_ item: CommunicationContinuityItem) {
        let focusedRunID = nonEmpty(item.runID)
        var drillInChips: [String] = []
        if let focusedRunID {
            inspectStore.transitionInspectContext(
                focusedRunID: focusedRunID,
                searchSeed: nil,
                statusMessage: "Loading inspect logs for continuity run \(focusedRunID)…"
            )
            drillInChips.append(makeDrillInChip(label: "Run", value: focusedRunID) ?? "")
        } else if let correlationID = nonEmpty(item.correlationID) {
            inspectStore.transitionInspectContext(
                focusedRunID: nil,
                searchSeed: correlationID,
                statusMessage: "Opened Inspect for continuity correlation \(correlationID).",
                forceResetSnapshot: true
            )
            drillInChips.append(makeDrillInChip(label: "Correlation", value: correlationID) ?? "")
        } else if let threadID = nonEmpty(item.threadID) {
            inspectStore.transitionInspectContext(
                focusedRunID: nil,
                searchSeed: threadID,
                statusMessage: "Opened Inspect for continuity thread \(threadID).",
                forceResetSnapshot: true
            )
            drillInChips.append(makeDrillInChip(label: "Thread", value: threadID) ?? "")
        } else if let turnID = nonEmpty(item.turnID) {
            inspectStore.transitionInspectContext(
                focusedRunID: nil,
                searchSeed: turnID,
                statusMessage: "Opened Inspect for continuity turn \(turnID).",
                forceResetSnapshot: true
            )
            drillInChips.append(makeDrillInChip(label: "Turn", value: turnID) ?? "")
        } else {
            inspectStore.transitionInspectContext(
                focusedRunID: nil,
                searchSeed: nil,
                statusMessage: "Opened Inspect from conversation continuity context.",
                forceResetSnapshot: true
            )
        }
        drillInChips.append(makeDrillInChip(label: "Channel", value: item.channel) ?? "")
        navigateToSection(
            .inspect,
            drillInContext: DrillInNavigationContext(
                sourceSection: .communications,
                destinationSection: .inspect,
                chips: drillInChips
            )
        )
    }

    public func setInspectLiveTailEnabled(_ enabled: Bool) {
        inspectStore.isInspectLiveTailEnabled = enabled
        guard selectedSection == .inspect else {
            return
        }
        if enabled {
            inspectStatusMessage = "Inspect live tail resumed."
            startInspectStream()
        } else {
            stopInspectStream()
            inspectStatusMessage = "Inspect live tail paused. Use Refresh for a one-time snapshot."
        }
    }

    public func refreshChannelCards() {
        Task { [weak self] in
            guard let self else {
                return
            }
            let startedAtNanoseconds = DispatchTime.now().uptimeNanoseconds
            await self.fetchChannelCards()
            let elapsedNanoseconds = DispatchTime.now().uptimeNanoseconds - startedAtNanoseconds
            self.panelLatencyStore.recordPanelLatencySample(
                section: .channels,
                category: .refresh,
                durationMS: Int(elapsedNanoseconds / 1_000_000)
            )
        }
    }

    public func channelConnectorMappings(channelID: String) -> [ChannelConnectorMappingItem] {
        connectionConfigStore.channelConnectorMappings(
            channelID: channelID,
            normalizeChannelID: normalizedChannelConnectorMappingChannelID(_:),
            sortedMappings: sortedChannelConnectorMappings(_:)
        )
    }

    public func channelConnectorMappingStatusMessage(channelID: String) -> String? {
        connectionConfigStore.channelConnectorMappingStatusMessage(
            channelID: channelID,
            normalizeChannelID: normalizedChannelConnectorMappingChannelID(_:)
        )
    }

    public func channelConnectorMappingHasDraftChanges(channelID: String) -> Bool {
        connectionConfigStore.channelConnectorMappingHasDraftChanges(
            channelID: channelID,
            normalizeChannelID: normalizedChannelConnectorMappingChannelID(_:),
            sortedMappings: sortedChannelConnectorMappings(_:)
        )
    }

    public func channelConfigHasDraftChanges(channelID: String) -> Bool {
        let baseline = channelCardItem(channelID: channelID)?.editableConfiguration ?? [:]
        return connectionConfigStore.channelConfigHasDraftChanges(
            channelID: channelID,
            baseline: baseline
        )
    }

    public func channelDeliveryPolicyHasDraftChanges(channelID: String) -> Bool {
        connectionConfigStore.channelDeliveryPolicyHasDraftChanges(
            channelID: channelID,
            normalizeChannelID: normalizedChannelDeliveryPolicyChannelID(_:),
            defaultDraft: defaultChannelDeliveryPolicyDraft(channelID:policies:)
        )
    }

    public func channelConnectorDisplayName(connectorID: String) -> String {
        let normalizedConnectorID = normalizedChannelConnectorMappingConnectorID(connectorID)
        guard !normalizedConnectorID.isEmpty else {
            return connectorID
        }
        if let logicalCard = logicalConnectorCards.first(where: {
            normalizedChannelConnectorMappingConnectorID($0.id) == normalizedConnectorID
        }) {
            return logicalCard.title
        }
        if let connectorCard = connectorCards.first(where: { card in
            normalizedChannelConnectorMappingConnectorID(card.logicalConnectorID) == normalizedConnectorID
                || normalizedChannelConnectorMappingConnectorID(card.id) == normalizedConnectorID
        }) {
            return connectorCard.name
        }
        return Self.humanizedIdentifierTitle(normalizedConnectorID)
    }

    public func channelConnectorMappingConstraintsSummary(channelID: String) -> String {
        let normalizedChannelID = normalizedChannelConnectorMappingChannelID(channelID)
        let mappings = channelConnectorMappings(channelID: normalizedChannelID)
        guard !mappings.isEmpty else {
            return "No connector mappings are available yet. Refresh channel status to load daemon-declared mappings."
        }

        let connectorSummary = Self.deduplicatedPreservingOrder(
            mappings.map { channelConnectorDisplayName(connectorID: $0.connectorID) }
        )
        let capabilityFamilies = Self.deduplicatedPreservingOrder(
            mappings
                .flatMap(\.capabilities)
                .compactMap(Self.capabilityFamilyLabel(from:))
        )

        if capabilityFamilies.isEmpty {
            return "Mapped connectors: \(connectorSummary.joined(separator: ", ")). Daemon validates capability compatibility for this channel."
        }
        return "Mapped connectors: \(connectorSummary.joined(separator: ", ")). Capability families: \(capabilityFamilies.joined(separator: ", "))."
    }

    public func setChannelConnectorMappingEnabled(
        channelID: String,
        connectorID: String,
        enabled: Bool
    ) {
        connectionConfigStore.setChannelConnectorMappingEnabled(
            channelID: channelID,
            connectorID: connectorID,
            enabled: enabled,
            normalizeChannelID: normalizedChannelConnectorMappingChannelID(_:),
            normalizeConnectorID: normalizedChannelConnectorMappingConnectorID(_:),
            sortedMappings: sortedChannelConnectorMappings(_:)
        )
    }

    public func moveChannelConnectorMappingUp(channelID: String, connectorID: String) {
        connectionConfigStore.reorderChannelConnectorMapping(
            channelID: channelID,
            connectorID: connectorID,
            direction: -1,
            normalizeChannelID: normalizedChannelConnectorMappingChannelID(_:),
            normalizeConnectorID: normalizedChannelConnectorMappingConnectorID(_:),
            sortedMappings: sortedChannelConnectorMappings(_:),
            rebalanceMappings: rebalancedChannelConnectorMappingsPreservingCurrentOrder(_:),
            connectorDisplayName: channelConnectorDisplayName(connectorID:)
        )
    }

    public func moveChannelConnectorMappingDown(channelID: String, connectorID: String) {
        connectionConfigStore.reorderChannelConnectorMapping(
            channelID: channelID,
            connectorID: connectorID,
            direction: 1,
            normalizeChannelID: normalizedChannelConnectorMappingChannelID(_:),
            normalizeConnectorID: normalizedChannelConnectorMappingConnectorID(_:),
            sortedMappings: sortedChannelConnectorMappings(_:),
            rebalanceMappings: rebalancedChannelConnectorMappingsPreservingCurrentOrder(_:),
            connectorDisplayName: channelConnectorDisplayName(connectorID:)
        )
    }

    public func canMoveChannelConnectorMapping(
        channelID: String,
        connectorID: String,
        direction: Int
    ) -> Bool {
        connectionConfigStore.canMoveChannelConnectorMapping(
            channelID: channelID,
            connectorID: connectorID,
            direction: direction,
            normalizeChannelID: normalizedChannelConnectorMappingChannelID(_:),
            normalizeConnectorID: normalizedChannelConnectorMappingConnectorID(_:),
            sortedMappings: sortedChannelConnectorMappings(_:)
        )
    }

    public func resetChannelConnectorMappingDraft(channelID: String) {
        connectionConfigStore.resetChannelConnectorMappingDraft(
            channelID: channelID,
            normalizeChannelID: normalizedChannelConnectorMappingChannelID(_:),
            sortedMappings: sortedChannelConnectorMappings(_:)
        )
    }

    public func saveChannelConnectorMappings(channelID: String) {
        Task { [weak self] in
            await self?.performChannelConnectorMappingSave(channelID: channelID)
        }
    }

    public func channelConfigDraftKeys(channelID: String) -> [String] {
        connectionConfigStore.channelConfigDraftKeys(channelID: channelID)
    }

    public func channelConfigFieldDescriptors(channelID: String) -> [ConfigurationFieldDescriptorItem] {
        guard let card = channelCardItem(channelID: channelID) else {
            return []
        }
        if !card.configurationFieldDescriptors.isEmpty {
            return card.configurationFieldDescriptors
        }
        return synthesizedConfigurationFieldDescriptors(
            editableConfiguration: card.editableConfiguration,
            editableConfigurationKinds: card.editableConfigurationKinds
        )
    }

    public func channelConfigFieldDescriptor(
        channelID: String,
        key: String
    ) -> ConfigurationFieldDescriptorItem? {
        channelConfigFieldDescriptors(channelID: channelID).first(where: { $0.key == key })
    }

    public func channelGuidedConfigFieldKeys(channelID: String) -> [String] {
        channelConfigFieldDescriptors(channelID: channelID).map(\.key)
    }

    public func channelAdvancedConfigDraftKeys(channelID: String) -> [String] {
        let guidedKeys = Set(channelGuidedConfigFieldKeys(channelID: channelID))
        return (channelConfigDraftByID[channelID] ?? [:]).keys
            .filter { !guidedKeys.contains($0) }
            .sorted()
    }

    public func channelConfigDraftValue(channelID: String, key: String) -> String {
        connectionConfigStore.channelConfigDraftValue(channelID: channelID, key: key)
    }

    public func channelConfigDraftKind(channelID: String, key: String) -> ConfigurationDraftValueKind {
        connectionConfigStore.channelConfigDraftKind(channelID: channelID, key: key)
    }

    public func setChannelConfigDraftValue(channelID: String, key: String, value: String) {
        connectionConfigStore.setChannelConfigDraftValue(channelID: channelID, key: key, value: value)
    }

    public func addChannelConfigDraftField(channelID: String, key: String, value: String) {
        connectionConfigStore.addChannelConfigDraftField(
            channelID: channelID,
            key: key,
            value: value,
            inferDraftKind: connectionConfigStore.inferConfigurationDraftKind(from:)
        )
    }

    public func removeChannelConfigDraftField(channelID: String, key: String) {
        connectionConfigStore.removeChannelConfigDraftField(channelID: channelID, key: key)
    }

    public func resetChannelConfigDraft(channelID: String) {
        connectionConfigStore.resetChannelConfigDraft(
            channelID: channelID,
            channelCards: channelCards
        )
    }

    public func saveChannelConfiguration(channelID: String) {
        Task { [weak self] in
            await self?.performChannelConfigurationSave(channelID: channelID)
        }
    }

    public func runChannelHealthCheck(channelID: String) {
        Task { [weak self] in
            await self?.performChannelHealthCheck(channelID: channelID)
        }
    }

    public func channelDeliveryPolicies(channelID: String) -> [ChannelDeliveryPolicyItem] {
        connectionConfigStore.channelDeliveryPolicies(
            channelID: channelID,
            normalizeChannelID: normalizedChannelDeliveryPolicyChannelID(_:)
        )
    }

    public func channelDeliveryPolicyDraft(channelID: String) -> ChannelDeliveryPolicyDraft {
        connectionConfigStore.channelDeliveryPolicyDraft(
            channelID: channelID,
            normalizeChannelID: normalizedChannelDeliveryPolicyChannelID(_:),
            defaultDraft: defaultChannelDeliveryPolicyDraft(channelID:policies:)
        )
    }

    public func channelDeliveryRouteOptions(channelID: String) -> [String] {
        let normalizedChannelID = normalizedChannelDeliveryPolicyChannelID(channelID)
        var candidates: Set<String> = []
        if !normalizedChannelID.isEmpty {
            candidates.insert(normalizedChannelID)
        }
        candidates.formUnion(logicalChannelCards.map { normalizedChannelDeliveryPolicyChannelID($0.channelID) })
        for card in channelCards {
            let candidate = normalizedChannelDeliveryPolicyChannelID(card.id)
            if !candidate.isEmpty {
                candidates.insert(candidate)
            }
        }
        for policy in channelDeliveryPolicies(channelID: normalizedChannelID) {
            let primary = normalizedChannelDeliveryPolicyChannelID(policy.primaryChannel)
            if !primary.isEmpty {
                candidates.insert(primary)
            }
            for fallback in policy.fallbackChannels {
                let fallbackChannel = normalizedChannelDeliveryPolicyChannelID(fallback)
                if !fallbackChannel.isEmpty {
                    candidates.insert(fallbackChannel)
                }
            }
        }
        return candidates
            .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
            .filter { !$0.isEmpty }
            .sorted { lhs, rhs in
                lhs.localizedCaseInsensitiveCompare(rhs) == .orderedAscending
            }
    }

    public func setChannelDeliveryPolicyPrimaryChannel(channelID: String, primaryChannel: String) {
        connectionConfigStore.setChannelDeliveryPolicyPrimaryChannel(
            channelID: channelID,
            primaryChannel: primaryChannel,
            normalizeChannelID: normalizedChannelDeliveryPolicyChannelID(_:),
            defaultDraft: defaultChannelDeliveryPolicyDraft(channelID:policies:)
        )
    }

    public func setChannelDeliveryPolicyEndpointPattern(channelID: String, endpointPattern: String) {
        connectionConfigStore.setChannelDeliveryPolicyEndpointPattern(
            channelID: channelID,
            endpointPattern: endpointPattern,
            normalizeChannelID: normalizedChannelDeliveryPolicyChannelID(_:),
            defaultDraft: defaultChannelDeliveryPolicyDraft(channelID:policies:)
        )
    }

    public func setChannelDeliveryPolicyRetryCount(channelID: String, retryCount: Int) {
        connectionConfigStore.setChannelDeliveryPolicyRetryCount(
            channelID: channelID,
            retryCount: retryCount,
            normalizeChannelID: normalizedChannelDeliveryPolicyChannelID(_:),
            defaultDraft: defaultChannelDeliveryPolicyDraft(channelID:policies:)
        )
    }

    public func setChannelDeliveryPolicyFallbackChannelsText(channelID: String, fallbackChannelsText: String) {
        connectionConfigStore.setChannelDeliveryPolicyFallbackChannelsText(
            channelID: channelID,
            fallbackChannelsText: fallbackChannelsText,
            normalizeChannelID: normalizedChannelDeliveryPolicyChannelID(_:),
            defaultDraft: defaultChannelDeliveryPolicyDraft(channelID:policies:)
        )
    }

    public func setChannelDeliveryPolicyIsDefault(channelID: String, isDefault: Bool) {
        connectionConfigStore.setChannelDeliveryPolicyIsDefault(
            channelID: channelID,
            isDefault: isDefault,
            normalizeChannelID: normalizedChannelDeliveryPolicyChannelID(_:),
            defaultDraft: defaultChannelDeliveryPolicyDraft(channelID:policies:)
        )
    }

    public func loadChannelDeliveryPolicyDraft(channelID: String, policyID: String) {
        connectionConfigStore.loadChannelDeliveryPolicyDraft(
            channelID: channelID,
            policyID: policyID,
            normalizeChannelID: normalizedChannelDeliveryPolicyChannelID(_:),
            channelDeliveryDraft: channelDeliveryDraft(from:)
        )
    }

    public func startNewChannelDeliveryPolicyDraft(channelID: String) {
        connectionConfigStore.startNewChannelDeliveryPolicyDraft(
            channelID: channelID,
            normalizeChannelID: normalizedChannelDeliveryPolicyChannelID(_:)
        )
    }

    public func resetChannelDeliveryPolicyDraft(channelID: String) {
        connectionConfigStore.resetChannelDeliveryPolicyDraft(
            channelID: channelID,
            normalizeChannelID: normalizedChannelDeliveryPolicyChannelID(_:),
            defaultDraft: defaultChannelDeliveryPolicyDraft(channelID:policies:)
        )
    }

    public func saveChannelDeliveryPolicy(channelID: String) {
        Task { [weak self] in
            await self?.performChannelDeliveryPolicySave(channelID: channelID)
        }
    }

    public func requestSaveChannelDeliveryPolicy(channelID: String) {
        let normalizedChannelID = normalizedChannelDeliveryPolicyChannelID(channelID)
        guard !normalizedChannelID.isEmpty else {
            return
        }
        let draft = channelDeliveryPolicyDraft(channelID: normalizedChannelID)
        let fallbackSummary = nonEmpty(draft.fallbackChannelsText) ?? "none"
        let endpointSummary = nonEmpty(draft.endpointPattern) ?? "default endpoint match"
        presentHighImpactActionConfirmation(
            title: "Save Delivery Policy?",
            message: "Save \(normalizedChannelID) policy with primary `\(draft.primaryChannel)`, fallback `\(fallbackSummary)`, endpoint `\(endpointSummary)`.",
            confirmButtonTitle: "Save Policy",
            isDestructive: false
        ) { [weak self] in
            self?.saveChannelDeliveryPolicy(channelID: normalizedChannelID)
        }
    }

    public func channelSetupFallbackActionTitle(channelID: String) -> String {
        channelSetupDestination(channelID: channelID).actionTitle
    }

    public func openChannelSetupDestination(channelID: String) {
        let destination = channelSetupDestination(channelID: channelID)
        navigateToSection(destination.section)
        channelsStatusMessage = destination.message
    }

    public func refreshConnectorCards() {
        Task { [weak self] in
            guard let self else {
                return
            }
            let startedAtNanoseconds = DispatchTime.now().uptimeNanoseconds
            await self.fetchConnectorCards()
            let elapsedNanoseconds = DispatchTime.now().uptimeNanoseconds - startedAtNanoseconds
            self.panelLatencyStore.recordPanelLatencySample(
                section: .connectors,
                category: .refresh,
                durationMS: Int(elapsedNanoseconds / 1_000_000)
            )
        }
    }

    public func connectorConfigDraftKeys(connectorID: String) -> [String] {
        connectionConfigStore.connectorConfigDraftKeys(connectorID: connectorID)
    }

    public func connectorConfigFieldDescriptors(connectorID: String) -> [ConfigurationFieldDescriptorItem] {
        guard let card = connectorCardItem(connectorID: connectorID) else {
            return []
        }
        if !card.configurationFieldDescriptors.isEmpty {
            return card.configurationFieldDescriptors
        }
        return synthesizedConfigurationFieldDescriptors(
            editableConfiguration: card.editableConfiguration,
            editableConfigurationKinds: card.editableConfigurationKinds
        )
    }

    public func connectorConfigFieldDescriptor(
        connectorID: String,
        key: String
    ) -> ConfigurationFieldDescriptorItem? {
        connectorConfigFieldDescriptors(connectorID: connectorID).first(where: { $0.key == key })
    }

    public func connectorGuidedConfigFieldKeys(connectorID: String) -> [String] {
        connectorConfigFieldDescriptors(connectorID: connectorID).map(\.key)
    }

    public func connectorAdvancedConfigDraftKeys(connectorID: String) -> [String] {
        let guidedKeys = Set(connectorGuidedConfigFieldKeys(connectorID: connectorID))
        return (connectorConfigDraftByID[connectorID] ?? [:]).keys
            .filter { !guidedKeys.contains($0) }
            .sorted()
    }

    public func connectorConfigDraftValue(connectorID: String, key: String) -> String {
        connectionConfigStore.connectorConfigDraftValue(connectorID: connectorID, key: key)
    }

    public func connectorConfigHasDraftChanges(connectorID: String) -> Bool {
        let baseline = connectorCardItem(connectorID: connectorID)?.editableConfiguration ?? [:]
        return connectionConfigStore.connectorConfigHasDraftChanges(
            connectorID: connectorID,
            baseline: baseline
        )
    }

    public func connectorConfigDraftKind(connectorID: String, key: String) -> ConfigurationDraftValueKind {
        connectionConfigStore.connectorConfigDraftKind(connectorID: connectorID, key: key)
    }

    public func setConnectorConfigDraftValue(connectorID: String, key: String, value: String) {
        connectionConfigStore.setConnectorConfigDraftValue(connectorID: connectorID, key: key, value: value)
    }

    public func addConnectorConfigDraftField(connectorID: String, key: String, value: String) {
        connectionConfigStore.addConnectorConfigDraftField(
            connectorID: connectorID,
            key: key,
            value: value,
            inferDraftKind: connectionConfigStore.inferConfigurationDraftKind(from:)
        )
    }

    public func removeConnectorConfigDraftField(connectorID: String, key: String) {
        connectionConfigStore.removeConnectorConfigDraftField(connectorID: connectorID, key: key)
    }

    public func resetConnectorConfigDraft(connectorID: String) {
        connectionConfigStore.resetConnectorConfigDraft(
            connectorID: connectorID,
            connectorCards: connectorCards
        )
    }

    public func saveConnectorConfiguration(connectorID: String) {
        Task { [weak self] in
            await self?.performConnectorConfigurationSave(connectorID: connectorID)
        }
    }

    public func runConnectorHealthCheck(connectorID: String) {
        Task { [weak self] in
            await self?.performConnectorHealthCheck(connectorID: connectorID)
        }
    }

    public func canPerformChannelDiagnosticsAction(channelID: String, action: DiagnosticsActionItem) -> Bool {
        resolveChannelDiagnosticsExecution(channelID: channelID, action: action) != nil
    }

    public func performChannelDiagnosticsAction(channelID: String, action: DiagnosticsActionItem) {
        guard action.enabled else {
            channelsStatusMessage = action.reason ?? "\(action.title) is currently unavailable for \(channelID)."
            return
        }

        guard let execution = resolveChannelDiagnosticsExecution(channelID: channelID, action: action) else {
            channelsStatusMessage = "Unsupported channel action `\(action.id)`."
            return
        }

        switch execution {
        case .refreshStatus:
            refreshChannelCards()
        case .openSection(let section, let message):
            navigateToSection(section)
            channelsStatusMessage = message
        case .openSystemSettings(let resolvedChannelID, _):
            channelsStatusMessage = "Opened System Settings for \(resolvedChannelID) channel remediation."
        case .channelSetup:
            openChannelSetupDestination(channelID: channelID)
        case .daemonLifecycleControl(let action):
            requestDaemonLifecycleControlWithConfirmation(action: action)
        }
    }

    public func canPerformConnectorDiagnosticsAction(connectorID: String, action: DiagnosticsActionItem) -> Bool {
        resolveConnectorDiagnosticsExecution(connectorID: connectorID, action: action) != nil
    }

    public func performConnectorDiagnosticsAction(connectorID: String, action: DiagnosticsActionItem) {
        guard action.enabled else {
            connectorsStatusMessage = action.reason ?? "\(action.title) is currently unavailable for \(connectorID)."
            return
        }

        guard let execution = resolveConnectorDiagnosticsExecution(connectorID: connectorID, action: action) else {
            connectorsStatusMessage = "Unsupported connector action `\(action.id)`."
            return
        }

        switch execution {
        case .refreshStatus:
            refreshConnectorCards()
        case .openSection(let section, let message):
            navigateToSection(section)
            connectorsStatusMessage = message
        case .requestPermission(let resolvedConnectorID):
            requestConnectorPermission(resolvedConnectorID)
        case .openSystemSettings(let resolvedConnectorID, _):
            noteConnectorSystemSettingsOpened(connectorID: resolvedConnectorID)
        case .daemonLifecycleControl(let action):
            requestDaemonLifecycleControlWithConfirmation(action: action)
        }
    }

    func systemSettingsURLForConnectorAction(connectorID: String, action: DiagnosticsActionItem?) -> URL {
        let resolvedConnectorID = normalizedConnectorIdentifier(
            connectorID,
            parameters: action?.parameters ?? [:]
        )
        if let daemonDestinationURL = ConnectorPermissionManager.systemSettingsURL(
            fromDaemonDestination: action?.destination
        ) {
            return daemonDestinationURL
        }
        return ConnectorPermissionManager.systemSettingsURL(for: resolvedConnectorID)
    }

    func systemSettingsURLForChannelAction(channelID: String, action: DiagnosticsActionItem?) -> URL {
        if let daemonDestinationURL = ConnectorPermissionManager.systemSettingsURL(
            fromDaemonDestination: action?.destination
        ) {
            return daemonDestinationURL
        }

        let resolvedChannelID = normalizedChannelIdentifier(
            channelID,
            parameters: action?.parameters ?? [:]
        )
        let connectorID = normalizedConnectorIdentifier(
            defaultConnectorIDForChannelSystemSettings(channelID: resolvedChannelID),
            parameters: action?.parameters ?? [:]
        )
        return ConnectorPermissionManager.systemSettingsURL(for: connectorID)
    }

    public func refreshPrincipalOptions() {
        Task { [weak self] in
            await self?.fetchPrincipalOptions()
        }
    }

    public func refreshIdentityDirectory() {
        refreshPrincipalOptions()
    }

    public func refreshIdentityDeviceInventory() {
        Task { [weak self] in
            await self?.fetchIdentityDeviceInventory()
        }
    }

    public func resetIdentityDeviceInventoryFilters() {
        identityContextStore.resetIdentityDeviceInventoryFilters(
            defaultLimit: IdentityInventoryProjection.defaultQueryLimit
        )
    }

    public func refreshIdentitySessionInventory() {
        Task { [weak self] in
            await self?.fetchIdentitySessionInventory()
        }
    }

    public func resetIdentitySessionInventoryFilters() {
        identityContextStore.resetIdentitySessionInventoryFilters(
            defaultLimit: IdentityInventoryProjection.defaultQueryLimit
        )
    }

    public func requestRevokeIdentitySession(sessionID: String) {
        let trimmedSessionID = sessionID.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmedSessionID.isEmpty else {
            return
        }
        presentHighImpactActionConfirmation(
            title: "Revoke Session?",
            message: "Session \(trimmedSessionID) will be revoked for the selected workspace.",
            confirmButtonTitle: "Revoke Session",
            isDestructive: true,
            irreversibleNote: "Session revoke is intended to be irreversible."
        ) { [weak self] in
            self?.revokeIdentitySession(sessionID: trimmedSessionID)
        }
    }

    public func revokeIdentitySession(sessionID: String) {
        Task { [weak self] in
            await self?.performIdentitySessionRevoke(sessionID: sessionID)
        }
    }

    public func refreshDelegationRules() {
        Task { [weak self] in
            await self?.fetchDelegationRules()
        }
    }

    public func createDelegationRule(_ input: DelegationGrantInput) {
        Task { [weak self] in
            await self?.performDelegationGrant(input)
        }
    }

    public func refreshChatPersonaPolicy() {
        Task { [weak self] in
            await self?.fetchChatPersonaPolicy()
        }
    }

    public func useActivePrincipalForChatPersonaScope() {
        chatPersonaScopePrincipalActorID = nonEmpty(selectedPrincipal) ?? "default"
    }

    public func resetChatPersonaPolicyDraft() {
        if let policy = chatPersonaPolicyItem {
            applyChatPersonaPolicy(policy, statusMessage: "Reverted unsaved persona policy changes.")
            return
        }
        chatPersonaStylePromptDraft = ""
        chatPersonaGuardrailsDraft = ""
        chatPersonaPolicyStatusMessage = "No loaded persona policy to reset."
    }

    public func saveChatPersonaPolicy() {
        saveChatPersonaPolicy(
            ChatPersonaPolicyMutationInput(
                principalActorID: chatPersonaResolvedPrincipalActorID,
                channelID: chatPersonaResolvedChannelID,
                stylePrompt: chatPersonaStylePromptDraft,
                guardrailsText: chatPersonaGuardrailsDraft
            )
        )
    }

    public func saveChatPersonaPolicy(_ input: ChatPersonaPolicyMutationInput) {
        Task { [weak self] in
            await self?.performChatPersonaPolicySave(input)
        }
    }

    public func testChatPersonaPolicyInChat() {
        guard !chatPersonaStylePromptDraft.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else {
            chatPersonaPolicyStatusMessage = "Save a style prompt before testing chat persona behavior."
            return
        }

        let responseShapingChannelID = chatPersonaResponseShapingChannelID
        let responseShapingProfileID = responseShapingProfileID(for: responseShapingChannelID)
        let responseShapingSummary = "\(responseShapingChannelID) • \(responseShapingProfileID)"
        let scopeSummary = chatPersonaScopeSummary(
            principalActorID: chatPersonaResolvedPrincipalActorID,
            channelID: chatPersonaResolvedChannelID
        )
        chatStatusMessage = "Persona test scope active (\(scopeSummary) • \(responseShapingSummary)). Send a message to validate tone and guardrails."
        chatProgressDetail = nil
        chatFailureRemediationMessage = nil
        navigateToSection(
            .chat,
            drillInContext: DrillInNavigationContext(
                sourceSection: .configuration,
                destinationSection: .chat,
                chips: [
                    "Persona Scope: \(scopeSummary)",
                    "Response Profile: \(responseShapingProfileID)"
                ]
            )
        )
    }

    public func requestRevokeDelegationRule(ruleID: String) {
        let normalizedRuleID = ruleID.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !normalizedRuleID.isEmpty else {
            return
        }
        presentHighImpactActionConfirmation(
            title: "Revoke Delegation Rule?",
            message: "Rule \(normalizedRuleID) will be revoked and no longer authorize delegated actions.",
            confirmButtonTitle: "Revoke Rule",
            isDestructive: true
        ) { [weak self] in
            self?.revokeDelegationRule(ruleID: normalizedRuleID)
        }
    }

    public func revokeDelegationRule(ruleID: String) {
        Task { [weak self] in
            await self?.performDelegationRevoke(ruleID: ruleID)
        }
    }

    public func refreshCapabilityGrantInventory() {
        Task { [weak self] in
            await self?.fetchCapabilityGrantInventory()
        }
    }

    public func resetCapabilityGrantFilters() {
        capabilityGrantActorFilter = nonEmpty(selectedPrincipal) ?? ""
        capabilityGrantKeyFilter = ""
        capabilityGrantStatusFilter = "all"
        capabilityGrantLimit = 25
    }

    public func upsertCapabilityGrant(_ input: CapabilityGrantMutationInput) {
        Task { [weak self] in
            await self?.performCapabilityGrantUpsert(input)
        }
    }

    public func requestRevokeCapabilityGrant(grantID: String) {
        let normalizedGrantID = grantID.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !normalizedGrantID.isEmpty else {
            return
        }
        presentHighImpactActionConfirmation(
            title: "Revoke Capability Grant?",
            message: "Grant \(normalizedGrantID) will be marked revoked for this workspace.",
            confirmButtonTitle: "Revoke Grant",
            isDestructive: true
        ) { [weak self] in
            self?.revokeCapabilityGrant(grantID: normalizedGrantID)
        }
    }

    public func revokeCapabilityGrant(grantID: String) {
        Task { [weak self] in
            await self?.performCapabilityGrantRevoke(grantID: grantID)
        }
    }

    public func refreshWebhookTrustReceipts() {
        Task { [weak self] in
            await self?.fetchWebhookTrustReceipts()
        }
    }

    public func resetWebhookTrustReceiptFilters() {
        webhookReceiptProviderFilter = ""
        webhookReceiptProviderEventIDFilter = ""
        webhookReceiptProviderEventQueryFilter = ""
        webhookReceiptEventIDFilter = ""
        webhookReceiptLimit = 25
    }

    public func refreshIngestTrustReceipts() {
        Task { [weak self] in
            await self?.fetchIngestTrustReceipts()
        }
    }

    public func resetIngestTrustReceiptFilters() {
        ingestReceiptSourceFilter = ""
        ingestReceiptSourceScopeFilter = ""
        ingestReceiptSourceEventIDFilter = ""
        ingestReceiptSourceEventQueryFilter = ""
        ingestReceiptTrustStateFilter = "all"
        ingestReceiptEventIDFilter = ""
        ingestReceiptLimit = 25
    }

    public func actingAsOptions(including actorID: String? = nil) -> [String] {
        identityContextStore.principalOptionsForPrincipalSelection(including: actorID)
    }

    public func actingAsValidationMessage(for actorID: String) -> String? {
        identityContextStore.actingAsValidationMessage(for: actorID)
    }

    public func identityWorkspaceDisplayName(for workspaceID: String) -> String {
        identityContextStore.identityWorkspaceDisplayName(
            for: workspaceID,
            canonicalWorkspaceID: Self.canonicalWorkspaceID
        )
    }

    public func workspaceIdentityDisplayValue(for workspaceID: String?) -> IdentityDisplayValue {
        identityContextStore.workspaceIdentityDisplayValue(
            for: workspaceID,
            canonicalWorkspaceID: Self.canonicalWorkspaceID,
            defaultWorkspaceID: Self.defaultWorkspaceID
        )
    }

    public func principalIdentityDisplayValue(for actorID: String?) -> IdentityDisplayValue {
        identityContextStore.principalIdentityDisplayValue(for: actorID)
    }

    public func principalOptionDisplayName(for actorID: String) -> String {
        principalIdentityDisplayValue(for: actorID).displayText
    }

    public func selectIdentityWorkspace(_ workspaceID: String) {
        Task { [weak self] in
            await self?.performIdentityWorkspaceSelection(workspaceID: workspaceID)
        }
    }

    public func refreshOnboardingReadiness() {
        Task { [weak self] in
            guard let self else {
                return
            }
            async let lifecycleRefresh: Void = self.refreshDaemonLifecycleStatus()
            async let principalRefresh: Void = self.fetchPrincipalOptions()
            async let providerRefresh: Void = self.fetchProviderAndModelStatus(runChecks: false)
            async let channelsRefresh: Void = self.fetchChannelCards()
            async let connectorsRefresh: Void = self.fetchConnectorCards()
            _ = await (
                lifecycleRefresh,
                principalRefresh,
                providerRefresh,
                channelsRefresh,
                connectorsRefresh
            )
        }
    }

    public func runRetentionPurge() {
        Task { [weak self] in
            await self?.performRetentionPurge()
        }
    }

    public func requestRunRetentionPurge() {
        presentHighImpactActionConfirmation(
            title: "Run Retention Purge?",
            message: "Purge applies the current retention windows and removes expired runtime data.",
            confirmButtonTitle: "Run Purge",
            isDestructive: true,
            irreversibleNote: "Purged records cannot be restored from app history."
        ) { [weak self] in
            self?.runRetentionPurge()
        }
    }

    public func runRetentionCompactMemory() {
        Task { [weak self] in
            await self?.performRetentionCompactMemory()
        }
    }

    public func requestRunRetentionCompactMemory() {
        let applyWrites = retentionCompactionApply
        presentHighImpactActionConfirmation(
            title: applyWrites ? "Run Memory Compaction?" : "Preview Memory Compaction?",
            message: applyWrites
                ? "Compaction writes optimized memory records using the current thresholds."
                : "Preview evaluates memory compaction candidates without writing changes.",
            confirmButtonTitle: applyWrites ? "Run Compaction" : "Run Preview",
            isDestructive: applyWrites,
            irreversibleNote: applyWrites ? "Compaction writes mutate retained memory entries." : nil
        ) { [weak self] in
            self?.runRetentionCompactMemory()
        }
    }

    public func loadContextSamples() {
        Task { [weak self] in
            await self?.performContextSamplesQuery()
        }
    }

    public func runContextTune() {
        Task { [weak self] in
            await self?.performContextTune()
        }
    }

    public func refreshContextMemoryInventory() {
        Task { [weak self] in
            await self?.fetchContextMemoryInventory()
        }
    }

    public func resetContextMemoryInventoryFilters() {
        contextMemoryOwnerActorFilter = nonEmpty(selectedPrincipal) ?? ""
        contextMemoryScopeTypeFilter = ""
        contextMemoryStatusFilter = "all"
        contextMemorySourceTypeFilter = ""
        contextMemorySourceRefQuery = ""
        contextMemoryLimit = 25
    }

    public func refreshContextMemoryCandidates() {
        Task { [weak self] in
            await self?.fetchContextMemoryCandidates()
        }
    }

    public func resetContextMemoryCandidatesFilters() {
        contextMemoryCandidatesOwnerActorFilter = nonEmpty(selectedPrincipal) ?? ""
        contextMemoryCandidatesStatusFilter = "all"
        contextMemoryCandidatesLimit = 25
    }

    public func refreshContextRetrievalDocuments() {
        Task { [weak self] in
            await self?.fetchContextRetrievalDocuments()
        }
    }

    public func resetContextRetrievalDocumentFilters() {
        contextRetrievalOwnerActorFilter = nonEmpty(selectedPrincipal) ?? ""
        contextRetrievalSourceURIQuery = ""
        contextRetrievalDocumentsLimit = 25
    }

    public func refreshContextRetrievalChunks() {
        Task { [weak self] in
            await self?.fetchContextRetrievalChunks()
        }
    }

    public func selectContextRetrievalDocument(_ documentID: String) {
        selectedContextRetrievalDocumentID = documentID
        Task { [weak self] in
            await self?.fetchContextRetrievalChunks()
        }
    }

    public func refreshProviderInventory() {
        Task { [weak self] in
            guard let self else {
                return
            }
            let startedAtNanoseconds = DispatchTime.now().uptimeNanoseconds
            await self.fetchProviderAndModelStatus(runChecks: false)
            let elapsedNanoseconds = DispatchTime.now().uptimeNanoseconds - startedAtNanoseconds
            self.panelLatencyStore.recordPanelLatencySample(
                section: .models,
                category: .refresh,
                durationMS: Int(elapsedNanoseconds / 1_000_000)
            )
        }
    }

    public func runProviderConnectivityChecks() {
        Task { [weak self] in
            await self?.fetchProviderAndModelStatus(runChecks: true)
        }
    }

    public func runProviderConnectivityCheck(for providerID: String) {
        Task { [weak self] in
            await self?.performProviderConnectivityCheck(providerID: providerID)
        }
    }

    public func runProviderQuickstartSaveAndCheck(providerID: String) {
        Task { [weak self] in
            guard let self else {
                return
            }
            let normalizedProvider = self.normalizedProviderID(providerID)
            await self.performProviderSetupSave(providerID: normalizedProvider)
            guard self.providerSetupStatusByID[normalizedProvider]?.hasPrefix("Saved ") == true else {
                return
            }
            await self.performProviderConnectivityCheck(providerID: normalizedProvider)
        }
    }

    public func resetProviderEndpointDraft(for providerID: String) {
        modelsRouteStore.resetProviderEndpointDraft(
            providerID: providerID,
            normalizedProviderID: normalizedProviderID(_:),
            providerDefaultEndpoints: Self.providerDefaultEndpoints
        )
    }

    public func providerEndpointDraft(for providerID: String) -> String {
        modelsRouteStore.providerEndpointDraft(
            for: providerID,
            normalizedProviderID: normalizedProviderID(_:),
            providerDefaultEndpoints: Self.providerDefaultEndpoints
        )
    }

    public func providerSecretNameDraft(for providerID: String) -> String {
        modelsRouteStore.providerSecretNameDraft(
            for: providerID,
            normalizedProviderID: normalizedProviderID(_:),
            defaultProviderSecretName: defaultProviderSecretName(for:)
        )
    }

    public func providerSecretValueDraft(for providerID: String) -> String {
        modelsRouteStore.providerSecretValueDraft(
            for: providerID,
            normalizedProviderID: normalizedProviderID(_:)
        )
    }

    public func providerSetupHasDraftChanges(providerID: String) -> Bool {
        modelsRouteStore.providerSetupHasDraftChanges(
            providerID: providerID,
            normalizedProviderID: normalizedProviderID(_:),
            providerDefaultEndpoints: Self.providerDefaultEndpoints,
            defaultProviderSecretName: defaultProviderSecretName(for:)
        )
    }

    private func clearPendingSectionNavigation() {
        navigationStore.clearPendingSectionNavigation()
    }

    private func dirtyChannelConfigurationIDs() -> [String] {
        channelCards
            .map(\.id)
            .filter { channelConfigHasDraftChanges(channelID: $0) }
            .sorted()
    }

    private func dirtyChannelDeliveryPolicyIDs() -> [String] {
        var channelIDs = Set(channelCards.map { normalizedChannelDeliveryPolicyChannelID($0.id) })
        channelIDs.formUnion(channelDeliveryPoliciesByChannelID.keys.map(normalizedChannelDeliveryPolicyChannelID))
        channelIDs.formUnion(channelDeliveryPolicyDraftByID.keys.map(normalizedChannelDeliveryPolicyChannelID))
        return channelIDs
            .filter { !$0.isEmpty }
            .filter { channelDeliveryPolicyHasDraftChanges(channelID: $0) }
            .sorted()
    }

    private func dirtyChannelConnectorMappingIDs() -> [String] {
        var ids = Set(channelConnectorMappingsByChannelID.keys)
        ids.formUnion(channelConnectorMappingDraftByChannelID.keys)
        ids.formUnion(logicalChannelCards.map { normalizedChannelConnectorMappingChannelID($0.channelID) })
        return ids
            .map(normalizedChannelConnectorMappingChannelID)
            .filter { !$0.isEmpty }
            .filter { channelConnectorMappingHasDraftChanges(channelID: $0) }
            .sorted()
    }

    private func dirtyConnectorConfigurationIDs() -> [String] {
        connectorCards
            .map(\.id)
            .filter { connectorConfigHasDraftChanges(connectorID: $0) }
            .sorted()
    }

    private func dirtyProviderSetupIDs() -> [String] {
        modelsRouteStore.dirtyProviderSetupIDs(
            providerReadinessProviders: providerReadinessItems.map(\.provider),
            normalizedProviderID: normalizedProviderID(_:),
            canonicalProviderOrder: Self.canonicalProviderOrder,
            providerDefaultEndpoints: Self.providerDefaultEndpoints,
            defaultProviderSecretName: defaultProviderSecretName(for:)
        )
    }

    private func discardAllChannelDraftChanges() {
        let channelIDs = Set(channelCards.map(\.id))
        for channelID in channelIDs where channelConfigHasDraftChanges(channelID: channelID) {
            resetChannelConfigDraft(channelID: channelID)
        }
        for channelID in channelIDs where channelDeliveryPolicyHasDraftChanges(channelID: channelID) {
            resetChannelDeliveryPolicyDraft(channelID: channelID)
        }
        for logicalChannelID in dirtyChannelConnectorMappingIDs() {
            resetChannelConnectorMappingDraft(channelID: logicalChannelID)
        }
        channelsStatusMessage = "Discarded unsaved channel drafts."
    }

    private func discardAllConnectorDraftChanges() {
        for connectorID in dirtyConnectorConfigurationIDs() {
            resetConnectorConfigDraft(connectorID: connectorID)
        }
        connectorsStatusMessage = "Discarded unsaved connector drafts."
    }

    private func resetProviderSetupDraft(providerID: String) {
        modelsRouteStore.resetProviderSetupDraft(
            providerID: providerID,
            normalizedProviderID: normalizedProviderID(_:),
            providerDefaultEndpoints: Self.providerDefaultEndpoints,
            defaultProviderSecretName: defaultProviderSecretName(for:)
        )
    }

    private func discardAllProviderDraftChanges() {
        for providerID in dirtyProviderSetupIDs() {
            resetProviderSetupDraft(providerID: providerID)
        }
        providerStatusMessage = "Discarded unsaved provider setup drafts."
    }

    private func saveAllChannelDraftChanges() async {
        let channelConfigIDs = dirtyChannelConfigurationIDs()
        let deliveryPolicyIDs = dirtyChannelDeliveryPolicyIDs()
        let mappingIDs = dirtyChannelConnectorMappingIDs()

        guard !channelConfigIDs.isEmpty || !deliveryPolicyIDs.isEmpty || !mappingIDs.isEmpty else {
            channelsStatusMessage = "No unsaved channel drafts to save."
            return
        }

        channelsStatusMessage = "Saving channel drafts…"
        for channelID in channelConfigIDs {
            await performChannelConfigurationSave(channelID: channelID)
        }
        for channelID in deliveryPolicyIDs {
            await performChannelDeliveryPolicySave(channelID: channelID)
        }
        for logicalChannelID in mappingIDs {
            await performChannelConnectorMappingSave(channelID: logicalChannelID)
        }
        channelsStatusMessage = "Saved all channel drafts."
    }

    private func saveAllConnectorDraftChanges() async {
        let connectorIDs = dirtyConnectorConfigurationIDs()
        guard !connectorIDs.isEmpty else {
            connectorsStatusMessage = "No unsaved connector drafts to save."
            return
        }

        connectorsStatusMessage = "Saving connector drafts…"
        for connectorID in connectorIDs {
            await performConnectorConfigurationSave(connectorID: connectorID)
        }
        connectorsStatusMessage = "Saved all connector drafts."
    }

    private func saveAllProviderDraftChanges() async {
        let providerIDs = dirtyProviderSetupIDs()
        guard !providerIDs.isEmpty else {
            providerStatusMessage = "No unsaved provider setup drafts to save."
            return
        }

        providerStatusMessage = "Saving provider setup drafts…"
        for providerID in providerIDs {
            await performProviderSetupSave(providerID: providerID)
        }
        providerStatusMessage = "Saved all provider setup drafts."
    }

    public func setProviderEndpointDraft(_ value: String, for providerID: String) {
        modelsRouteStore.setProviderEndpointDraft(
            value,
            providerID: providerID,
            normalizedProviderID: normalizedProviderID(_:)
        )
    }

    public func setProviderSecretNameDraft(_ value: String, for providerID: String) {
        modelsRouteStore.setProviderSecretNameDraft(
            value,
            providerID: providerID,
            normalizedProviderID: normalizedProviderID(_:)
        )
    }

    public func setProviderSecretValueDraft(_ value: String, for providerID: String) {
        modelsRouteStore.setProviderSecretValueDraft(
            value,
            providerID: providerID,
            normalizedProviderID: normalizedProviderID(_:)
        )
    }

    public func saveProviderSetup(for providerID: String) {
        Task { [weak self] in
            await self?.performProviderSetupSave(providerID: providerID)
        }
    }

    public func setModelEnabled(
        providerID: String,
        modelKey: String,
        enabled: Bool
    ) {
        Task { [weak self] in
            await self?.performModelEnabledMutation(
                providerID: providerID,
                modelKey: modelKey,
                enabled: enabled
            )
        }
    }

    public func setModelAsChatRoute(providerID: String, modelKey: String) {
        Task { [weak self] in
            await self?.performSetModelAsChatRoute(providerID: providerID, modelKey: modelKey)
        }
    }

    public func openChatForModelsQuickstartTest(providerID: String, modelKey: String) {
        let normalizedProvider = normalizedProviderID(providerID)
        let normalizedModelKey = modelKey.trimmingCharacters(in: .whitespacesAndNewlines)
        let providerLabel = providerDisplayName(normalizedProvider)
        let routeLabel = normalizedModelKey.isEmpty
            ? providerLabel
            : "\(providerLabel) • \(normalizedModelKey)"

        if nonEmpty(chatDraft) == nil {
            chatDraft = "Quickstart check: confirm the \(routeLabel) route is ready and respond with one short sentence."
        }

        chatStatusMessage = "Models quickstart is ready for \(routeLabel). Send a message to validate chat routing."
        navigateToSection(
            .chat,
            drillInContext: DrillInNavigationContext(
                sourceSection: .models,
                destinationSection: .chat,
                chips: ["Route: \(routeLabel)"]
            )
        )
    }

    public func modelManualAddDraft(for providerID: String) -> String {
        modelsRouteStore.modelManualAddDraft(
            providerID: providerID,
            normalizedProviderID: normalizedProviderID(_:)
        )
    }

    public func setModelManualAddDraft(_ value: String, for providerID: String) {
        modelsRouteStore.setModelManualAddDraft(
            value,
            providerID: providerID,
            normalizedProviderID: normalizedProviderID(_:)
        )
    }

    public func discoverModels(for providerID: String) {
        Task { [weak self] in
            await self?.performModelDiscover(providerID: providerID)
        }
    }

    public func addModelToCatalog(providerID: String, modelKey: String, enabled: Bool = false) {
        Task { [weak self] in
            await self?.performModelCatalogAdd(providerID: providerID, modelKey: modelKey, enabled: enabled)
        }
    }

    public func removeModelFromCatalog(providerID: String, modelKey: String) {
        Task { [weak self] in
            await self?.performModelCatalogRemove(providerID: providerID, modelKey: modelKey)
        }
    }

    public func saveModelRoutePolicy(
        taskClass: String,
        providerID: String,
        modelKey: String
    ) {
        Task { [weak self] in
            await self?.performModelRoutePolicySave(
                taskClass: taskClass,
                providerID: providerID,
                modelKey: modelKey
            )
        }
    }

    public func requestSaveModelRoutePolicy(
        taskClass: String,
        providerID: String,
        modelKey: String
    ) {
        let normalizedTaskClass = taskClass.trimmingCharacters(in: .whitespacesAndNewlines)
        let normalizedProviderID = providerID.trimmingCharacters(in: .whitespacesAndNewlines)
        let normalizedModelKey = modelKey.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !normalizedTaskClass.isEmpty, !normalizedProviderID.isEmpty, !normalizedModelKey.isEmpty else {
            return
        }
        presentHighImpactActionConfirmation(
            title: "Save Route Policy?",
            message: "Route `\(normalizedTaskClass)` tasks to `\(normalizedProviderID)` • `\(normalizedModelKey)` for this workspace.",
            confirmButtonTitle: "Save Route Policy",
            isDestructive: false
        ) { [weak self] in
            self?.saveModelRoutePolicy(
                taskClass: normalizedTaskClass,
                providerID: normalizedProviderID,
                modelKey: normalizedModelKey
            )
        }
    }

    public func applyActivePrincipalToModelRouteSimulation() {
        modelsRouteStore.applyActivePrincipalToModelRouteSimulation(
            selectedPrincipal: selectedPrincipal
        )
    }

    public func clearModelRouteSimulationPrincipal() {
        modelsRouteStore.clearModelRouteSimulationPrincipal()
    }

    public func resetModelRouteSimulationOutputs() {
        modelsRouteStore.resetModelRouteSimulationOutputs()
    }

    public func runModelRouteSimulation() {
        Task { [weak self] in
            await self?.performModelRouteSimulation()
        }
    }

    public func runModelRouteExplainability() {
        Task { [weak self] in
            await self?.performModelRouteExplainability()
        }
    }

    public func refreshAutomationTriggers() {
        Task { [weak self] in
            guard let self else {
                return
            }
            let startedAtNanoseconds = DispatchTime.now().uptimeNanoseconds
            await self.fetchAutomationPanelData()
            let elapsedNanoseconds = DispatchTime.now().uptimeNanoseconds - startedAtNanoseconds
            self.panelLatencyStore.recordPanelLatencySample(
                section: .automation,
                category: .refresh,
                durationMS: Int(elapsedNanoseconds / 1_000_000)
            )
        }
    }

    public func refreshAutomationFireHistory() {
        Task { [weak self] in
            guard let self else {
                return
            }
            self.clearPanelProblemSignal(for: .automation)
            let startedAtNanoseconds = DispatchTime.now().uptimeNanoseconds
            await self.fetchAutomationFireHistory()
            let elapsedNanoseconds = DispatchTime.now().uptimeNanoseconds - startedAtNanoseconds
            self.panelLatencyStore.recordPanelLatencySample(
                section: .automation,
                category: .refresh,
                durationMS: Int(elapsedNanoseconds / 1_000_000)
            )
        }
    }

    public func openTasksForAutomationFireHistory(_ item: AutomationFireHistoryItem) {
        var drillInChips: [String] = []
        if let runID = nonEmpty(item.runID) {
            tasksSearchSeed = runID
            tasksStatusMessage = "Opened Tasks for automation run \(runID)."
            drillInChips.append(makeDrillInChip(label: "Run", value: runID) ?? "")
        } else if let taskID = nonEmpty(item.taskID) {
            tasksSearchSeed = taskID
            tasksStatusMessage = "Opened Tasks for automation task \(taskID)."
            drillInChips.append(makeDrillInChip(label: "Task", value: taskID) ?? "")
        } else if let modelKey = nonEmpty(item.route.modelKey),
                  let provider = nonEmpty(item.route.provider) {
            tasksSearchSeed = modelKey
            tasksStatusMessage = "Opened Tasks for automation route \(provider)/\(modelKey)."
            drillInChips.append("Route: \(provider)/\(modelKey)")
        } else if let modelKey = nonEmpty(item.route.modelKey) {
            tasksSearchSeed = modelKey
            tasksStatusMessage = "Opened Tasks for automation model \(modelKey)."
            drillInChips.append(makeDrillInChip(label: "Model", value: modelKey) ?? "")
        } else if let provider = nonEmpty(item.route.provider) {
            tasksSearchSeed = provider
            tasksStatusMessage = "Opened Tasks for automation provider \(provider)."
            drillInChips.append(makeDrillInChip(label: "Provider", value: provider) ?? "")
        } else if let taskClass = nonEmpty(item.route.taskClass) {
            tasksSearchSeed = taskClass
            tasksStatusMessage = "Opened Tasks for automation task class \(taskClass)."
            drillInChips.append(makeDrillInChip(label: "Task Class", value: taskClass) ?? "")
        } else {
            tasksSearchSeed = nil
            tasksStatusMessage = "Opened Tasks from automation fire history."
        }
        navigateToSection(
            .tasks,
            drillInContext: DrillInNavigationContext(
                sourceSection: .automation,
                destinationSection: .tasks,
                chips: drillInChips
            )
        )
    }

    public func openInspectForAutomationFireHistory(_ item: AutomationFireHistoryItem) {
        let focusedRunID = nonEmpty(item.runID)
        var drillInChips: [String] = []
        if let focusedRunID {
            inspectStore.transitionInspectContext(
                focusedRunID: focusedRunID,
                searchSeed: nil,
                statusMessage: "Loading inspect logs for automation run \(focusedRunID)…"
            )
            drillInChips.append(makeDrillInChip(label: "Run", value: focusedRunID) ?? "")
        } else if let taskID = nonEmpty(item.taskID) {
            inspectStore.transitionInspectContext(
                focusedRunID: nil,
                searchSeed: taskID,
                statusMessage: "Opened Inspect from automation task \(taskID).",
                forceResetSnapshot: true
            )
            drillInChips.append(makeDrillInChip(label: "Task", value: taskID) ?? "")
        } else if let routeSummary = automationRouteSummary(item.route) {
            inspectStore.transitionInspectContext(
                focusedRunID: nil,
                searchSeed: nil,
                statusMessage: "Opened Inspect from automation route \(routeSummary).",
                forceResetSnapshot: true
            )
            drillInChips.append("Route: \(routeSummary)")
        } else {
            inspectStore.transitionInspectContext(
                focusedRunID: nil,
                searchSeed: nil,
                statusMessage: "Opened Inspect from automation fire history.",
                forceResetSnapshot: true
            )
        }
        navigateToSection(
            .inspect,
            drillInContext: DrillInNavigationContext(
                sourceSection: .automation,
                destinationSection: .inspect,
                chips: drillInChips
            )
        )
    }

    public func clearInspectRunFocus() {
        _ = inspectStore.clearInspectRunFocus()
    }

    public func openTasksForInspectLog(_ item: InspectLogItem) {
        var drillInChips: [String] = []
        if let runID = nonEmpty(item.runID) {
            tasksSearchSeed = runID
            tasksStatusMessage = "Opened Tasks for inspect run \(runID)."
            drillInChips.append(makeDrillInChip(label: "Run", value: runID) ?? "")
        } else if let taskID = nonEmpty(item.taskID) {
            tasksSearchSeed = taskID
            tasksStatusMessage = "Opened Tasks for inspect task \(taskID)."
            drillInChips.append(makeDrillInChip(label: "Task", value: taskID) ?? "")
        } else if let correlationID = nonEmpty(item.correlationID) {
            tasksSearchSeed = correlationID
            tasksStatusMessage = "Opened Tasks for inspect correlation \(correlationID)."
            drillInChips.append(makeDrillInChip(label: "Correlation", value: correlationID) ?? "")
        } else {
            tasksSearchSeed = nil
            tasksStatusMessage = "Opened Tasks from inspect context."
        }
        navigateToSection(
            .tasks,
            drillInContext: DrillInNavigationContext(
                sourceSection: .inspect,
                destinationSection: .tasks,
                chips: drillInChips
            )
        )
    }

    public func openApprovalsForInspectLog(_ item: InspectLogItem) {
        var drillInChips: [String] = []
        if let runID = nonEmpty(item.runID) {
            approvalsSearchSeed = runID
            approvalsStatusMessage = "Opened Approvals for inspect run \(runID)."
            drillInChips.append(makeDrillInChip(label: "Run", value: runID) ?? "")
        } else if let taskID = nonEmpty(item.taskID) {
            approvalsSearchSeed = taskID
            approvalsStatusMessage = "Opened Approvals for inspect task \(taskID)."
            drillInChips.append(makeDrillInChip(label: "Task", value: taskID) ?? "")
        } else if let stepID = nonEmpty(item.stepID) {
            approvalsSearchSeed = stepID
            approvalsStatusMessage = "Opened Approvals for inspect step \(stepID)."
            drillInChips.append(makeDrillInChip(label: "Step", value: stepID) ?? "")
        } else {
            approvalsSearchSeed = nil
            approvalsStatusMessage = "Opened Approvals from inspect context."
        }
        navigateToSection(
            .approvals,
            drillInContext: DrillInNavigationContext(
                sourceSection: .inspect,
                destinationSection: .approvals,
                chips: drillInChips
            )
        )
    }

    public func createAutomationTrigger(_ input: AutomationTriggerMutationInput) {
        Task { [weak self] in
            await self?.performAutomationCreate(input)
        }
    }

    public func updateAutomationTrigger(triggerID: String, input: AutomationTriggerMutationInput) {
        Task { [weak self] in
            await self?.performAutomationUpdate(triggerID: triggerID, input: input)
        }
    }

    public func deleteAutomationTrigger(triggerID: String) {
        Task { [weak self] in
            await self?.performAutomationDelete(triggerID: triggerID)
        }
    }

    public func refreshApprovalsInbox() {
        Task { [weak self] in
            guard let self else {
                return
            }
            let startedAtNanoseconds = DispatchTime.now().uptimeNanoseconds
            await self.fetchApprovalsInbox()
            let elapsedNanoseconds = DispatchTime.now().uptimeNanoseconds - startedAtNanoseconds
            self.panelLatencyStore.recordPanelLatencySample(
                section: .approvals,
                category: .refresh,
                durationMS: Int(elapsedNanoseconds / 1_000_000)
            )
        }
    }

    public func loadApprovalEvidence(for item: ApprovalInboxItem, forceRefresh: Bool = false) {
        let approvalID = item.id.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !approvalID.isEmpty else {
            return
        }

        guard let runID = nonEmpty(item.runID) else {
            approvalEvidenceStatusByID[approvalID] = "Run detail is unavailable for this approval row."
            return
        }

        if !forceRefresh,
           let cached = approvalEvidenceByID[approvalID],
           cached.runID == runID {
            if approvalEvidenceStatusByID[approvalID] == nil {
                approvalEvidenceStatusByID[approvalID] = "Evidence loaded from run \(runID)."
            }
            return
        }

        guard !approvalEvidenceInFlightIDs.contains(approvalID) else {
            return
        }

        Task { [weak self] in
            await self?.fetchApprovalEvidence(for: item, runID: runID)
        }
    }

    public func openTasksForApproval(_ item: ApprovalInboxItem) {
        var drillInChips: [String] = []
        if let runID = nonEmpty(item.runID) {
            tasksSearchSeed = runID
            tasksStatusMessage = "Opened Tasks for approval run \(runID)."
            drillInChips.append(makeDrillInChip(label: "Run", value: runID) ?? "")
        } else if let taskID = nonEmpty(item.taskID) {
            tasksSearchSeed = taskID
            tasksStatusMessage = "Opened Tasks for approval task \(taskID)."
            drillInChips.append(makeDrillInChip(label: "Task", value: taskID) ?? "")
        } else if let modelKey = nonEmpty(item.route.modelKey),
                  let provider = nonEmpty(item.route.provider) {
            tasksSearchSeed = modelKey
            tasksStatusMessage = "Opened Tasks for approval route \(provider)/\(modelKey)."
            drillInChips.append("Route: \(provider)/\(modelKey)")
        } else if let modelKey = nonEmpty(item.route.modelKey) {
            tasksSearchSeed = modelKey
            tasksStatusMessage = "Opened Tasks for approval model \(modelKey)."
            drillInChips.append(makeDrillInChip(label: "Model", value: modelKey) ?? "")
        } else if let provider = nonEmpty(item.route.provider) {
            tasksSearchSeed = provider
            tasksStatusMessage = "Opened Tasks for approval provider \(provider)."
            drillInChips.append(makeDrillInChip(label: "Provider", value: provider) ?? "")
        } else if let taskClass = nonEmpty(item.route.taskClass) {
            tasksSearchSeed = taskClass
            tasksStatusMessage = "Opened Tasks for approval task class \(taskClass)."
            drillInChips.append(makeDrillInChip(label: "Task Class", value: taskClass) ?? "")
        } else {
            tasksSearchSeed = nil
            tasksStatusMessage = "Opened Tasks from approval context."
        }
        navigateToSection(
            .tasks,
            drillInContext: DrillInNavigationContext(
                sourceSection: .approvals,
                destinationSection: .tasks,
                chips: drillInChips
            )
        )
    }

    public func openTaskRunDetailForApproval(_ item: ApprovalInboxItem) {
        guard let runID = nonEmpty(item.runID) else {
            openTasksForApproval(item)
            return
        }

        tasksSearchSeed = runID
        tasksStatusMessage = "Opening task run detail for approval run \(runID)…"
        navigateToSection(
            .tasks,
            drillInContext: DrillInNavigationContext(
                sourceSection: .approvals,
                destinationSection: .tasks,
                chips: [
                    makeDrillInChip(label: "Run", value: runID),
                    makeDrillInChip(label: "Approval", value: item.id)
                ].compactMap { $0 }
            )
        )

        Task { [weak self] in
            guard let self else {
                return
            }
            await self.fetchTaskRunList()
            guard let row = self.taskRunItems.first(where: { nonEmpty($0.runID) == runID }) else {
                self.tasksStatusMessage = "Run \(runID) was not found in the current task list."
                return
            }
            self.showTaskRunDetail(for: row)
            self.tasksStatusMessage = "Opened task run detail for approval run \(runID)."
        }
    }

    public func openInspectForApproval(_ item: ApprovalInboxItem) {
        let focusedRunID = nonEmpty(item.runID)
        var drillInChips: [String] = []
        if let focusedRunID {
            inspectStore.transitionInspectContext(
                focusedRunID: focusedRunID,
                searchSeed: nil,
                statusMessage: "Loading inspect logs for approval run \(focusedRunID)…"
            )
            drillInChips.append(makeDrillInChip(label: "Run", value: focusedRunID) ?? "")
        } else if let taskID = nonEmpty(item.taskID) {
            inspectStore.transitionInspectContext(
                focusedRunID: nil,
                searchSeed: taskID,
                statusMessage: "Opened Inspect from approval task \(taskID).",
                forceResetSnapshot: true
            )
            drillInChips.append(makeDrillInChip(label: "Task", value: taskID) ?? "")
        } else if let routeSummary = automationRouteSummary(item.route) {
            var routeSearchSeed: String?
            if let modelKey = nonEmpty(item.route.modelKey) {
                routeSearchSeed = modelKey
                drillInChips.append(makeDrillInChip(label: "Model", value: modelKey) ?? "")
            } else if let provider = nonEmpty(item.route.provider) {
                routeSearchSeed = provider
                drillInChips.append(makeDrillInChip(label: "Provider", value: provider) ?? "")
            } else if let taskClass = nonEmpty(item.route.taskClass) {
                routeSearchSeed = taskClass
                drillInChips.append(makeDrillInChip(label: "Task Class", value: taskClass) ?? "")
            }
            inspectStore.transitionInspectContext(
                focusedRunID: nil,
                searchSeed: routeSearchSeed,
                statusMessage: "Opened Inspect from approval route \(routeSummary).",
                forceResetSnapshot: true
            )
        } else {
            inspectStore.transitionInspectContext(
                focusedRunID: nil,
                searchSeed: nil,
                statusMessage: "Opened Inspect from approval context.",
                forceResetSnapshot: true
            )
        }
        navigateToSection(
            .inspect,
            drillInContext: DrillInNavigationContext(
                sourceSection: .approvals,
                destinationSection: .inspect,
                chips: drillInChips
            )
        )
    }

    public func approvalCardSummary(for item: ApprovalInboxItem) -> WorkflowCardSummary {
        WorkflowCardSummary(
            whatHappened: approvalWhatHappenedSummary(for: item),
            whatNeedsAction: approvalWhatNeedsActionSummary(for: item),
            whatNext: approvalWhatNextSummary(for: item)
        )
    }

    public func taskRunCardSummary(for item: TaskRunListRowItem) -> WorkflowCardSummary {
        WorkflowCardSummary(
            whatHappened: taskRunWhatHappenedSummary(for: item),
            whatNeedsAction: taskRunWhatNeedsActionSummary(for: item),
            whatNext: taskRunWhatNextSummary(for: item)
        )
    }

    public func refreshTaskRunList() {
        Task { [weak self] in
            guard let self else {
                return
            }
            let startedAtNanoseconds = DispatchTime.now().uptimeNanoseconds
            await self.fetchTaskRunList()
            let elapsedNanoseconds = DispatchTime.now().uptimeNanoseconds - startedAtNanoseconds
            self.panelLatencyStore.recordPanelLatencySample(
                section: .tasks,
                category: .refresh,
                durationMS: Int(elapsedNanoseconds / 1_000_000)
            )
        }
    }

    public func taskRunControlStatus(runID: String?) -> String? {
        guard let runID = nonEmpty(runID) else {
            return nil
        }
        return taskRunControlStatusByRunID[runID]
    }

    public func isTaskRunControlInFlight(runID: String?) -> Bool {
        guard let runID = nonEmpty(runID) else {
            return false
        }
        return taskRunControlInFlightRunIDs.contains(runID)
    }

    public func canPerformTaskRunControl(
        _ action: TaskRunControlAction,
        item: TaskRunListRowItem
    ) -> Bool {
        canPerformTaskRunControl(action, runID: item.runID, actions: item.actions)
    }

    public func canPerformTaskRunControl(
        _ action: TaskRunControlAction,
        runID: String?,
        actions: TaskRunActionAvailabilityItem
    ) -> Bool {
        guard let runID = nonEmpty(runID) else {
            return false
        }
        guard !taskRunControlInFlightRunIDs.contains(runID) else {
            return false
        }
        switch action {
        case .cancel:
            return actions.canCancel
        case .retry:
            return actions.canRetry
        case .requeue:
            return actions.canRequeue
        }
    }

    public func taskRunControlDisabledReason(
        _ action: TaskRunControlAction,
        item: TaskRunListRowItem
    ) -> String? {
        taskRunControlDisabledReason(action, runID: item.runID, actions: item.actions)
    }

    public func taskRunControlDisabledReason(
        _ action: TaskRunControlAction,
        runID: String?,
        actions: TaskRunActionAvailabilityItem
    ) -> String? {
        guard nonEmpty(runID) != nil else {
            return "Run control is unavailable because this task row has no run id."
        }
        guard !isTaskRunControlInFlight(runID: runID) else {
            return "Another run control action is already in progress for this run."
        }
        guard !canPerformTaskRunControl(action, runID: runID, actions: actions) else {
            return nil
        }
        switch action {
        case .cancel:
            return "Cancel is unavailable for the current run state."
        case .retry:
            return "Retry is available only for failed or cancelled runs."
        case .requeue:
            return "Requeue is available only for queued, planning, awaiting-approval, or blocked runs."
        }
    }

    public func requestTaskRunControl(
        _ action: TaskRunControlAction,
        item: TaskRunListRowItem
    ) {
        requestTaskRunControl(
            action,
            taskID: item.taskID,
            runID: item.runID,
            actions: item.actions
        )
    }

    public func requestTaskRunControl(
        _ action: TaskRunControlAction,
        taskID: String,
        runID: String?,
        actions: TaskRunActionAvailabilityItem
    ) {
        guard let runID = nonEmpty(runID) else {
            tasksStatusMessage = "Run control is unavailable because this task row has no run id."
            return
        }
        guard canPerformTaskRunControl(action, runID: runID, actions: actions) else {
            if let reason = taskRunControlDisabledReason(action, runID: runID, actions: actions) {
                taskRunControlStatusByRunID[runID] = reason
                tasksStatusMessage = reason
            }
            return
        }

        let runLabel = runID
        let taskLabel = taskID
        presentHighImpactActionConfirmation(
            title: action.confirmationTitle,
            message: "\(action.title) for task \(taskLabel), run \(runLabel)?",
            confirmButtonTitle: action.confirmationButtonTitle,
            isDestructive: action.isDestructive
        ) { [weak self] in
            Task { [weak self] in
                await self?.performTaskRunControl(
                    action,
                    taskID: taskLabel,
                    runID: runLabel
                )
            }
        }
    }

    public func submitTask(
        title: String,
        description: String?,
        taskClass: String?,
        requestedByActorID: String,
        subjectPrincipalActorID: String
    ) {
        Task { [weak self] in
            await self?.performTaskSubmit(
                title: title,
                description: description,
                taskClass: taskClass,
                requestedByActorID: requestedByActorID,
                subjectPrincipalActorID: subjectPrincipalActorID
            )
        }
    }

    public func openInspectForTaskRun(_ item: TaskRunListRowItem) {
        let focusedRunID = nonEmpty(item.runID)
        let drillInChips = [
            makeDrillInChip(label: "Run", value: focusedRunID),
            makeDrillInChip(label: "Task", value: item.taskID)
        ].compactMap { $0 }
        if let focusedRunID {
            inspectStore.transitionInspectContext(
                focusedRunID: focusedRunID,
                searchSeed: nil,
                statusMessage: "Loading inspect logs for task run \(focusedRunID)…"
            )
        } else {
            inspectStore.transitionInspectContext(
                focusedRunID: nil,
                searchSeed: item.taskID,
                statusMessage: "Opened Inspect for task \(item.taskID).",
                forceResetSnapshot: true
            )
        }
        navigateToSection(
            .inspect,
            drillInContext: DrillInNavigationContext(
                sourceSection: .tasks,
                destinationSection: .inspect,
                chips: drillInChips
            )
        )
    }

    public func openApprovalsForTaskRun(_ item: TaskRunListRowItem) {
        let drillInChips = [
            makeDrillInChip(label: "Run", value: item.runID),
            makeDrillInChip(label: "Task", value: item.taskID)
        ].compactMap { $0 }
        if let runID = nonEmpty(item.runID) {
            approvalsSearchSeed = runID
            approvalsStatusMessage = "Opened Approvals for task run \(runID)."
        } else {
            approvalsSearchSeed = item.taskID
            approvalsStatusMessage = "Opened Approvals for task \(item.taskID)."
        }
        navigateToSection(
            .approvals,
            drillInContext: DrillInNavigationContext(
                sourceSection: .tasks,
                destinationSection: .approvals,
                chips: drillInChips
            )
        )
    }

    public func showTaskRunDetail(for item: TaskRunListRowItem) {
        guard let runID = nonEmpty(item.runID) else {
            taskRunDetailStatusMessage = "Run detail is unavailable for this task because no run id is associated."
            selectedTaskRunDetail = nil
            return
        }
        Task { [weak self] in
            await self?.fetchTaskRunDetail(runID: runID, actions: item.actions)
        }
    }

    public func clearTaskRunDetail() {
        selectedTaskRunDetail = nil
    }

    public func openInspectForTaskRunDetail(_ detail: TaskRunDetailItem) {
        let drillInChips = [
            makeDrillInChip(label: "Run", value: detail.runID),
            makeDrillInChip(label: "Task", value: detail.taskID)
        ].compactMap { $0 }
        inspectStore.transitionInspectContext(
            focusedRunID: detail.runID,
            searchSeed: nil,
            statusMessage: "Loading inspect logs for run \(detail.runID)…"
        )
        navigateToSection(
            .inspect,
            drillInContext: DrillInNavigationContext(
                sourceSection: .tasks,
                destinationSection: .inspect,
                chips: drillInChips
            )
        )
    }

    public func openApprovalsForTaskRunDetail(_ detail: TaskRunDetailItem) {
        let drillInChips = [
            makeDrillInChip(label: "Run", value: detail.runID),
            makeDrillInChip(label: "Task", value: detail.taskID)
        ].compactMap { $0 }
        approvalsSearchSeed = detail.runID
        approvalsStatusMessage = "Opened Approvals for run \(detail.runID)."
        navigateToSection(
            .approvals,
            drillInContext: DrillInNavigationContext(
                sourceSection: .tasks,
                destinationSection: .approvals,
                chips: drillInChips
            )
        )
    }

    public func submitApprovalDecision(
        approvalID: String,
        decisionPhrase: String,
        decisionByActorID: String,
        rationale: String?
    ) {
        Task { [weak self] in
            await self?.performApprovalDecision(
                approvalID: approvalID,
                decisionPhrase: decisionPhrase,
                decisionByActorID: decisionByActorID,
                rationale: rationale
            )
        }
    }

    public func approvalDecisionActorOptions(including actorID: String? = nil) -> [String] {
        let fallbackActor = nonEmpty(actorID)
        var options = actingAsOptions(including: fallbackActor)
        if let fallbackActor,
           !options.contains(where: { $0.caseInsensitiveCompare(fallbackActor) == .orderedSame }) {
            options.insert(fallbackActor, at: 0)
        }
        if options.isEmpty {
            options = ["default"]
        }
        return Array(NSOrderedSet(array: options)) as? [String] ?? options
    }

    public func defaultApprovalDecisionActor(for item: ApprovalInboxItem) -> String {
        if let actingAs = normalizedApprovalDecisionActorCandidate(item.actingAsActorID) {
            return actingAs
        }
        if let requestedBy = normalizedApprovalDecisionActorCandidate(item.requestedByActorID) {
            return requestedBy
        }
        if let selectedPrincipalActorID = normalizedApprovalDecisionActorCandidate(selectedPrincipal) {
            return selectedPrincipalActorID
        }

        let availableOptions = approvalDecisionActorOptions()
        if let fallback = availableOptions.first(where: { normalizedApprovalDecisionActorCandidate($0) != nil }) {
            return fallback
        }
        return availableOptions.first ?? "default"
    }

    private func normalizedApprovalDecisionActorCandidate(_ raw: String?) -> String? {
        guard let candidate = nonEmpty(raw) else {
            return nil
        }
        switch candidate.lowercased() {
        case "unknown", "none", "n/a", "default":
            return nil
        default:
            return candidate
        }
    }

    public func approvalDecisionActorValidationMessage(actorID: String) -> String? {
        let trimmedActorID = actorID.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmedActorID.isEmpty else {
            return "Select `Decision By` before submitting."
        }
        guard trimmedActorID.caseInsensitiveCompare("default") != .orderedSame else {
            return nil
        }
        let knownPrincipalIDs = Set(
            identityPrincipalItems
                .map(\.id)
                .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
                .filter { !$0.isEmpty }
        )
        guard !knownPrincipalIDs.isEmpty else {
            return nil
        }
        guard knownPrincipalIDs.contains(trimmedActorID) else {
            return "Selected decision actor `\(trimmedActorID)` is not in the active workspace directory. Refresh Identity Directory in Configuration."
        }
        return nil
    }

    public func approvalRequiredPhrase(for item: ApprovalInboxItem) -> String {
        nonEmpty(item.requestedPhrase) ?? "GO AHEAD"
    }

    public func approvalApprovePhraseValidationMessage(
        phrase: String,
        item: ApprovalInboxItem
    ) -> String? {
        let requiredPhrase = approvalRequiredPhrase(for: item)
        let trimmedPhrase = phrase.trimmingCharacters(in: .whitespacesAndNewlines)
        guard trimmedPhrase == requiredPhrase else {
            return "Approve requires exact phrase `\(requiredPhrase)`. Use `Use Required Phrase` or type it exactly."
        }
        return nil
    }

    public func simulateAutomationScheduleNow() {
        Task { [weak self] in
            await self?.runAutomationScheduleSimulation()
        }
    }

    private func approvalWhatHappenedSummary(for item: ApprovalInboxItem) -> String {
        switch item.decisionState {
        case .pending:
            let risk = truncateText(item.riskRationale, limit: 140)
            if isAdvancedInformationDensityEnabled {
                return "Approval requested for \(item.stepName). Risk context: \(risk)"
            }
            return "Approval needed for \(item.stepName). \(risk)"
        case .final:
            var parts: [String] = []
            if let outcome = item.decisionOutcome {
                parts.append("Decision recorded: \(outcome.label).")
            } else {
                parts.append("Decision recorded.")
            }
            if let decidedAt = nonEmpty(item.decidedAtLabel) {
                parts.append("Updated \(decidedAt).")
            }
            if let rationale = nonEmpty(item.decisionRationale) {
                parts.append("Rationale: \(truncateText(rationale, limit: 140))")
            }
            return parts.joined(separator: " ")
        }
    }

    private func approvalWhatNeedsActionSummary(for item: ApprovalInboxItem) -> String {
        switch item.decisionState {
        case .pending:
            switch item.riskLevel {
            case .destructive:
                return isAdvancedInformationDensityEnabled
                    ? "Pick a decision actor, then approve with exact phrase `GO AHEAD` or reject with rationale."
                    : "Choose who is deciding, then approve with exact phrase `GO AHEAD` or reject with rationale."
            case .policy, .other:
                return isAdvancedInformationDensityEnabled
                    ? "Pick a decision actor, then submit approve or reject with optional rationale."
                    : "Choose who is deciding, then approve or reject (rationale optional)."
            }
        case .final:
            return isAdvancedInformationDensityEnabled
                ? "No further decision action is required."
                : "No further action is required."
        }
    }

    private func approvalWhatNextSummary(for item: ApprovalInboxItem) -> String {
        switch item.decisionState {
        case .pending:
            if nonEmpty(item.runID) != nil {
                return isAdvancedInformationDensityEnabled
                    ? "Review Evidence or Open Task Detail, then submit a decision."
                    : "Review details, then submit your decision."
            }
            return isAdvancedInformationDensityEnabled
                ? "Review context in related Tasks or Inspect, then submit a decision."
                : "Open Tasks or Inspect for context, then submit your decision."
        case .final:
            if let outcome = item.decisionOutcome {
                switch outcome {
                case .approved:
                    return isAdvancedInformationDensityEnabled
                        ? "Open Related Tasks or Inspect to confirm post-approval execution."
                        : "Open Related Tasks to confirm what happened next."
                case .rejected:
                    return isAdvancedInformationDensityEnabled
                        ? "Open Related Tasks to verify follow-up handling, or Inspect for audit trace."
                        : "Open Related Tasks for follow-up, or Inspect for details."
                case .other:
                    break
                }
            }
            return isAdvancedInformationDensityEnabled
                ? "Open Related Inspect for final audit context."
                : "Open Related Inspect for final details."
        }
    }

    private func taskRunWhatHappenedSummary(for item: TaskRunListRowItem) -> String {
        if let lastError = nonEmpty(item.lastError) {
            if isAdvancedInformationDensityEnabled {
                return "Run reported an error: \(truncateText(lastError, limit: 160))"
            }
            return "Task reported an error: \(truncateText(lastError, limit: 160))"
        }
        switch item.effectiveState {
        case .queued:
            return isAdvancedInformationDensityEnabled
                ? "Run is queued and waiting for execution."
                : "Task is queued and waiting to start."
        case .planning:
            return isAdvancedInformationDensityEnabled
                ? "Run is preparing execution steps."
                : "Task is preparing to run."
        case .awaitingApproval:
            return isAdvancedInformationDensityEnabled
                ? "Run is paused waiting for an approval decision."
                : "Task is paused and waiting for an approval decision."
        case .running:
            return isAdvancedInformationDensityEnabled
                ? "Run is actively executing."
                : "Task is actively running."
        case .blocked:
            return isAdvancedInformationDensityEnabled
                ? "Run is blocked and cannot continue yet."
                : "Task is blocked and cannot continue yet."
        case .completed:
            return isAdvancedInformationDensityEnabled
                ? "Run completed successfully."
                : "Task completed successfully."
        case .failed:
            return isAdvancedInformationDensityEnabled
                ? "Run finished in a failed state."
                : "Task finished in a failed state."
        case .cancelled:
            return isAdvancedInformationDensityEnabled
                ? "Run was cancelled before completion."
                : "Task was cancelled before completion."
        case .unknown(let rawState):
            if isAdvancedInformationDensityEnabled {
                return "Run state is currently \(rawState.replacingOccurrences(of: "_", with: " "))."
            }
            return "Task state is currently \(rawState.replacingOccurrences(of: "_", with: " "))."
        }
    }

    private func taskRunWhatNeedsActionSummary(for item: TaskRunListRowItem) -> String {
        switch item.effectiveState {
        case .awaitingApproval:
            return isAdvancedInformationDensityEnabled
                ? "Open Related Approvals and submit a decision to continue this run."
                : "Open Related Approvals and submit a decision to continue this task."
        case .failed, .blocked:
            if item.actions.canRetry || item.actions.canRequeue {
                return isAdvancedInformationDensityEnabled
                    ? "Review failure context, then retry or requeue once blockers are addressed."
                    : "Review the issue, then retry or requeue once blockers are addressed."
            }
            return isAdvancedInformationDensityEnabled
                ? "Review failure context in Inspect. Retry/requeue controls are unavailable."
                : "Review details in Inspect. Retry/requeue controls are unavailable."
        case .running, .planning, .queued:
            if item.actions.canCancel {
                return isAdvancedInformationDensityEnabled
                    ? "No immediate action required unless you need to cancel this run."
                    : "No immediate action required unless you need to cancel this task."
            }
            return "No immediate action required while this run is in progress."
        case .completed, .cancelled:
            return "No immediate action required."
        case .unknown:
            return "Refresh the list if this state appears stale."
        }
    }

    private func taskRunWhatNextSummary(for item: TaskRunListRowItem) -> String {
        var nextSteps: [String] = []
        if item.runID != nil {
            nextSteps.append(
                isAdvancedInformationDensityEnabled
                    ? "View Run Detail for steps and artifacts."
                    : "Open Run Detail to review what happened."
            )
        }
        nextSteps.append(
            isAdvancedInformationDensityEnabled
                ? "Open Related Inspect for full execution trace."
                : "Open Related Inspect for more details."
        )

        switch item.effectiveState {
        case .awaitingApproval:
            nextSteps.append(
                isAdvancedInformationDensityEnabled
                    ? "Open Related Approvals to unblock execution."
                    : "Open Related Approvals to continue this task."
            )
        case .failed:
            if item.actions.canRetry {
                nextSteps.append(
                    isAdvancedInformationDensityEnabled
                        ? "Use Retry Run after fixing the reported issue."
                        : "Use Retry Run after fixing the issue."
                )
            } else if item.actions.canRequeue {
                nextSteps.append(
                    isAdvancedInformationDensityEnabled
                        ? "Use Requeue Run to place work back in queue."
                        : "Use Requeue Run to try again."
                )
            }
        default:
            break
        }

        return nextSteps.joined(separator: " ")
    }

    public func simulateAutomationCommEvent() {
        Task { [weak self] in
            await self?.runAutomationCommEventSimulation()
        }
    }

    private func persistExpandedChannelCardContinuity() {
        let expandedIDs = channelCards
            .filter(\.isExpanded)
            .map(\.id)
            .sorted()
        updateCurrentWorkspaceContinuityContext { value in
            value.expandedChannelCardIDs = expandedIDs
        }
    }

    private func persistExpandedConnectorCardContinuity() {
        let expandedIDs = connectorCards
            .filter(\.isExpanded)
            .map(\.id)
            .sorted()
        updateCurrentWorkspaceContinuityContext { value in
            value.expandedConnectorCardIDs = expandedIDs
        }
    }

    public func toggleChannelCard(_ id: String) {
        guard let index = channelCards.firstIndex(where: { $0.id == id }) else {
            return
        }
        channelCards[index].isExpanded.toggle()
        persistExpandedChannelCardContinuity()
    }

    public func toggleConnectorCard(_ id: String) {
        guard let index = connectorCards.firstIndex(where: { $0.id == id }) else {
            return
        }
        connectorCards[index].isExpanded.toggle()
        persistExpandedConnectorCardContinuity()
    }

    public func requestConnectorPermissionWithConfirmation(connectorID: String) {
        let normalizedConnectorID = normalizedConnectorIdentifier(connectorID, parameters: [:])
        guard !normalizedConnectorID.isEmpty else {
            return
        }
        requestConnectorPermission(normalizedConnectorID)
    }

    public func requestConnectorPermission(_ id: String) {
        Task { [weak self] in
            await self?.performConnectorPermissionRequest(id: id)
        }
    }

    public func noteConnectorSystemSettingsOpened(connectorID: String) {
        connectorPermissionRefreshPendingIDs.insert(connectorID)
        connectorPermissionActionStatusByID[connectorID] = "Opened System Settings. Return to the app to refresh permission status."
        connectorsStatusMessage = "Opened System Settings for \(connectorID) permission checks."
    }

    public func saveLocalDevToken() {
        let trimmed = localDevTokenInput.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            return
        }
        do {
            try Self.persistLocalDevToken(trimmed)
        } catch {
            localDevAuthBootstrapStatusMessage = "Failed to save Assistant Access Token in local Keychain."
            appendNotification(
                source: "configuration",
                action: "save_local_dev_token",
                message: "Failed to save Assistant Access Token in local Keychain.",
                level: .error
            )
            return
        }
        daemonAuthToken = trimmed
        localDevTokenConfigured = true
        localDevTokenLastUpdated = Date.now.formatted(date: .abbreviated, time: .shortened)
        daemonControlAuthState = .unknown
        daemonControlAuthSource = "unknown"
        daemonControlAuthRemediationHints = []
        localDevTokenInput = ""
        hasLoadedProviderStatus = false
        localDevAuthBootstrapStatusMessage = "Saved Assistant Access Token manually."
        appendNotification(
            source: "configuration",
            action: "save_local_dev_token",
            message: "Saved Assistant Access Token.",
            level: .success
        )
        updateOnboardingCompletionState()
        Task { [weak self] in
            await self?.bootstrapFromDaemon()
        }
    }

    public func clearLocalDevToken() {
        daemonAuthToken = nil
        localDevTokenConfigured = false
        localDevTokenLastUpdated = "Not configured"
        localDevTokenInput = ""
        isLocalDevAuthBootstrapInFlight = false
        localDevAuthBootstrapStatusMessage = "Use Copy Command or Run Bootstrap to configure assistant access."
        hasLoadedProviderStatus = true
        isChatExplainabilityInFlight = false
        chatTurnContextStore.resetAllForMissingToken()
        syncChatTurnContextProjectionFromStore()
        _ = try? Self.clearPersistedLocalDevToken()
        connectionStatus = .disconnected
        daemonControlAuthState = .missing
        daemonControlAuthSource = "local_token_missing"
        daemonControlAuthRemediationHints = ["Save an Assistant Access Token to authorize daemon requests."]
        daemonStatusDetail = "Set Assistant Access Token to connect to daemon."
        daemonPluginLifecycleHistoryFilterPluginID = ""
        daemonPluginLifecycleHistoryFilterKind = RuntimePluginLifecycleProjection.defaultFilterSelection
        daemonPluginLifecycleHistoryFilterState = RuntimePluginLifecycleProjection.defaultFilterSelection
        daemonPluginLifecycleHistoryFilterEventType = RuntimePluginLifecycleProjection.defaultFilterSelection
        daemonPluginLifecycleHistoryLimit = RuntimePluginLifecycleProjection.defaultHistoryLimit
        isDaemonPluginLifecycleHistoryLoading = false
        daemonPluginLifecycleHistoryStatusMessage = RuntimePluginLifecycleProjection.missingTokenStatusMessage
        daemonPluginLifecycleHistoryItems = []
        daemonPluginLifecycleTrendItems = []
        daemonPluginLifecycleHistoryHasMore = false
        principalStatusMessage = "Set Assistant Access Token to query principal context."
        identityStatusMessage = principalStatusMessage
        identityWorkspaceItems = []
        identityPrincipalItems = []
        identityActiveContext = nil
        identityDeviceUserIDFilter = ""
        identityDeviceTypeFilter = ""
        identityDevicePlatformFilter = ""
        identityDeviceLimit = IdentityInventoryProjection.defaultQueryLimit
        isIdentityDeviceInventoryLoading = false
        identityDeviceInventoryStatusMessage = IdentityInventoryProjection.missingTokenDeviceMessage
        identityDeviceItems = []
        identityDeviceInventoryHasMore = false
        identitySessionDeviceIDFilter = ""
        identitySessionUserIDFilter = ""
        identitySessionHealthFilter = IdentityInventoryProjection.defaultHealthFilterSelection
        identitySessionLimit = IdentityInventoryProjection.defaultQueryLimit
        isIdentitySessionInventoryLoading = false
        identitySessionInventoryStatusMessage = IdentityInventoryProjection.missingTokenSessionMessage
        identitySessionItems = []
        identitySessionInventoryHasMore = false
        identitySessionActionStatusByID = [:]
        identitySessionRevokeInFlightIDs = []
        delegationRules = []
        delegationStatusMessage = "Set Assistant Access Token to query delegation rules."
        delegationActionStatusByRuleID = [:]
        chatPersonaScopeType = .workspace
        chatPersonaScopePrincipalActorID = "default"
        chatPersonaScopeChannelID = "app"
        chatPersonaStylePromptDraft = ""
        chatPersonaGuardrailsDraft = ""
        chatPersonaPolicyStatusMessage = "Set Assistant Access Token to load chat persona policy."
        chatPersonaPolicyItem = nil
        chatPersonaLoadedStylePrompt = ""
        chatPersonaLoadedGuardrails = []
        isChatPersonaPolicyRequestInFlight = false
        isChatPersonaPolicySaveRequestInFlight = false
        isChatPersonaPolicyLoading = false
        isChatPersonaPolicySaveInFlight = false
        chatPersonaHasLoadedPolicy = false
        capabilityGrantActorFilter = ""
        capabilityGrantKeyFilter = ""
        capabilityGrantStatusFilter = "all"
        capabilityGrantLimit = 25
        isCapabilityGrantInventoryLoading = false
        capabilityGrantStatusMessage = "Set Assistant Access Token to query capability grants."
        capabilityGrantItems = []
        capabilityGrantInventoryHasMore = false
        isCapabilityGrantMutationInFlight = false
        capabilityGrantMutationStatusMessage = "Set Assistant Access Token before mutating capability grants."
        capabilityGrantActionStatusByID = [:]
        capabilityGrantRevokeInFlightIDs = []
        webhookReceiptProviderFilter = ""
        webhookReceiptProviderEventIDFilter = ""
        webhookReceiptProviderEventQueryFilter = ""
        webhookReceiptEventIDFilter = ""
        webhookReceiptLimit = 25
        isWebhookReceiptsLoading = false
        webhookReceiptsStatusMessage = "Set Assistant Access Token to query webhook trust receipts."
        webhookReceiptItems = []
        webhookReceiptsHasMore = false
        ingestReceiptSourceFilter = ""
        ingestReceiptSourceScopeFilter = ""
        ingestReceiptSourceEventIDFilter = ""
        ingestReceiptSourceEventQueryFilter = ""
        ingestReceiptTrustStateFilter = "all"
        ingestReceiptEventIDFilter = ""
        ingestReceiptLimit = 25
        isIngestReceiptsLoading = false
        ingestReceiptsStatusMessage = "Set Assistant Access Token to query ingest trust receipts."
        ingestReceiptItems = []
        ingestReceiptsHasMore = false
        contextMemoryOwnerActorFilter = ""
        contextMemoryScopeTypeFilter = ""
        contextMemoryStatusFilter = "all"
        contextMemorySourceTypeFilter = ""
        contextMemorySourceRefQuery = ""
        contextMemoryLimit = 25
        isContextMemoryInventoryLoading = false
        contextMemoryInventoryStatusMessage = "Set Assistant Access Token to query context memory inventory."
        contextMemoryInventoryItems = []
        contextMemoryInventoryHasMore = false
        contextMemoryCandidatesOwnerActorFilter = ""
        contextMemoryCandidatesStatusFilter = "all"
        contextMemoryCandidatesLimit = 25
        isContextMemoryCandidatesLoading = false
        contextMemoryCandidatesStatusMessage = "Set Assistant Access Token to query memory compaction candidates."
        contextMemoryCandidateItems = []
        contextMemoryCandidatesHasMore = false
        contextRetrievalOwnerActorFilter = ""
        contextRetrievalSourceURIQuery = ""
        contextRetrievalDocumentsLimit = 25
        isContextRetrievalDocumentsLoading = false
        contextRetrievalDocumentsStatusMessage = "Set Assistant Access Token to query retrieval documents."
        contextRetrievalDocumentItems = []
        contextRetrievalDocumentsHasMore = false
        selectedContextRetrievalDocumentID = ""
        contextRetrievalChunkTextQuery = ""
        contextRetrievalChunksLimit = 25
        isContextRetrievalChunksLoading = false
        contextRetrievalChunksStatusMessage = "Set Assistant Access Token to query retrieval chunks."
        contextRetrievalChunkItems = []
        contextRetrievalChunksHasMore = false
        hasLoadedInspectLogs = false
        inspectLogs = []
        inspectStatusMessage = "Set Assistant Access Token to query inspect logs."
        communicationThreads = []
        communicationEvents = []
        communicationCallSessions = []
        communicationThreadsHasMore = false
        communicationEventsHasMore = false
        communicationCallSessionsHasMore = false
        isCommunicationContinuityLoading = false
        communicationContinuityStatusMessage = "Set Assistant Access Token to query conversation continuity."
        communicationContinuityItems = []
        communicationContinuityHasMore = false
        hasLoadedCommunicationsInbox = false
        isCommunicationAttemptsLoading = false
        communicationAttemptsStatusMessage = "Set Assistant Access Token to query delivery attempts."
        communicationDeliveryAttempts = []
        communicationDeliveryAttemptsHasMore = false
        communicationsStore.resetCommunicationAttemptContextThreadID()
        communicationsStatusMessage = "Set Assistant Access Token to query communications inbox."
        channelCards = []
        hasLoadedChannelStatus = false
        isChannelConnectorMappingsLoading = false
        channelConnectorMappingFallbackPolicy = "priority_order"
        channelConnectorMappingsByChannelID = [:]
        channelConnectorMappingDraftByChannelID = [:]
        channelConnectorMappingActionStatusByChannelID = [:]
        channelConnectorMappingSaveInFlightChannelIDs = []
        channelConfigDraftByID = [:]
        channelConfigKindsByID = [:]
        channelConfigActionStatusByID = [:]
        channelConfigSaveInFlightIDs = []
        channelTestInFlightIDs = []
        channelLastTestResultByID = [:]
        channelDeliveryPoliciesByChannelID = [:]
        channelDeliveryPolicyDraftByID = [:]
        channelDeliveryPolicyActionStatusByID = [:]
        channelDeliveryPolicySaveInFlightIDs = []
        connectorCards = []
        hasLoadedConnectorStatus = false
        connectorConfigDraftByID = [:]
        connectorConfigKindsByID = [:]
        connectorConfigActionStatusByID = [:]
        connectorConfigSaveInFlightIDs = []
        connectorTestInFlightIDs = []
        connectorLastTestResultByID = [:]
        connectorPermissionActionStatusByID = [:]
        connectorPermissionRequestInFlightIDs = []
        connectorPermissionRefreshPendingIDs = []
        modelRouteSimulationTaskClass = "chat"
        modelRouteSimulationPrincipalActorID = selectedPrincipal
        isModelRouteSimulationInFlight = false
        isModelRouteExplainInFlight = false
        modelRouteSimulationStatusMessage = "Set Assistant Access Token before running route simulation."
        modelRouteExplainStatusMessage = "Set Assistant Access Token before running route explainability."
        modelRouteSimulationResult = nil
        modelRouteExplainResult = nil
        hasLoadedAutomationPanelData = false
        automationTriggers = []
        automationStatusMessage = "Set Assistant Access Token to query automation triggers."
        automationFireHistoryItems = []
        automationFireHistoryStatusMessage = "Set Assistant Access Token to query trigger fire history."
        hasLoadedApprovalsInbox = false
        approvalInboxItems = []
        approvalsStatusMessage = "Set Assistant Access Token to query approval inbox."
        hasLoadedTaskRunList = false
        taskRunItems = []
        taskRunControlStatusByRunID = [:]
        taskRunControlInFlightRunIDs = []
        tasksStatusMessage = "Set Assistant Access Token to query task/runs."
        appendNotification(
            source: "configuration",
            action: "clear_local_dev_token",
            message: "Cleared Assistant Access Token.",
            level: .info
        )
        hasCompletedFirstRunOnboarding = false
        Self.userDefaultsStore.set(false, forKey: Self.onboardingCompleteDefaultsKey)
    }

    public func noteLocalDevAuthBootstrapCommandCopied() {
        localDevAuthBootstrapStatusMessage = "Copied bootstrap command. Run it in Terminal or use Run Bootstrap."
    }

    public func runLocalDevAuthBootstrap() {
        Task { [weak self] in
            await self?.performLocalDevAuthBootstrap()
        }
    }

    func performLocalDevAuthBootstrap() async {
        guard !isLocalDevAuthBootstrapInFlight else {
            return
        }
        isLocalDevAuthBootstrapInFlight = true
        defer { isLocalDevAuthBootstrapInFlight = false }

        let bootstrapArgs = localDevAuthBootstrapArguments()
        localDevAuthBootstrapStatusMessage = "Running assistant access setup..."
        let execution = await Self.localDevAuthBootstrapCommandRunner(bootstrapArgs)
        guard execution.exitCode == 0 else {
            let failureMessage = localDevAuthBootstrapFailureMessage(
                stderr: execution.stderr,
                exitCode: execution.exitCode
            )
            localDevAuthBootstrapStatusMessage = failureMessage
            appendNotification(
                source: "configuration",
                action: "bootstrap_local_dev_auth",
                message: failureMessage,
                level: .error
            )
            return
        }

        let response: LocalDevAuthBootstrapCLIResponse
        do {
            response = try decodeLocalDevAuthBootstrapCLIResponse(stdout: execution.stdout)
        } catch {
            let statusMessage = "Bootstrap command returned unexpected output. Retry or run command manually."
            localDevAuthBootstrapStatusMessage = statusMessage
            appendNotification(
                source: "configuration",
                action: "bootstrap_local_dev_auth",
                message: statusMessage,
                level: .error
            )
            return
        }

        let token: String
        do {
            token = try loadLocalDevAuthTokenFromFile(response.tokenFile)
        } catch {
            let statusMessage = "Bootstrap completed but token file could not be loaded. Check token path and retry."
            localDevAuthBootstrapStatusMessage = statusMessage
            appendNotification(
                source: "configuration",
                action: "bootstrap_local_dev_auth",
                message: statusMessage,
                level: .error
            )
            return
        }

        do {
            try Self.persistLocalDevToken(token)
        } catch {
            let statusMessage = "Bootstrap completed but token could not be saved to local Keychain. Retry setup."
            localDevAuthBootstrapStatusMessage = statusMessage
            appendNotification(
                source: "configuration",
                action: "bootstrap_local_dev_auth",
                message: statusMessage,
                level: .error
            )
            return
        }

        daemonAuthToken = token
        localDevTokenConfigured = true
        localDevTokenInput = ""
        localDevTokenLastUpdated = Date.now.formatted(date: .abbreviated, time: .shortened)
        daemonControlAuthState = .unknown
        daemonControlAuthSource = "unknown"
        daemonControlAuthRemediationHints = []
        hasLoadedProviderStatus = false
        updateOnboardingCompletionState()

        let tokenMaterialSummary: String
        if response.tokenRotated {
            tokenMaterialSummary = "rotated"
        } else if response.tokenCreated {
            tokenMaterialSummary = "created"
        } else {
            tokenMaterialSummary = "reused"
        }

        localDevAuthBootstrapStatusMessage = "Bootstrap completed (\(tokenMaterialSummary) token file). Refreshing setup checks..."
        appendNotification(
            source: "configuration",
            action: "bootstrap_local_dev_auth",
            message: "Bootstrap command completed. Refreshing setup checks.",
            level: .success
        )
        await Self.localDevAuthBootstrapRefreshHandler(self)
        if daemonControlAuthNeedsRemediation {
            let reminder = nonEmpty(response.nextStepReminder)
                ?? "Start or restart daemon with --auth-token-file, then retry."
            localDevAuthBootstrapStatusMessage = "Bootstrap completed, but daemon still reports auth setup is incomplete. \(reminder)"
        } else {
            localDevAuthBootstrapStatusMessage = "Bootstrap completed and readiness checks refreshed."
        }
    }

    private func bootstrapFromDaemon() async {
        await refreshDaemonLifecycleStatus()
        await fetchPrincipalOptions()
        await refreshDataForCurrentSection(trigger: .bootstrap)
        if needsFirstRunOnboarding {
            async let providerRefresh: Void = fetchProviderAndModelStatus(runChecks: false)
            async let channelsRefresh: Void = fetchChannelCards()
            async let connectorsRefresh: Void = fetchConnectorCards()
            _ = await (providerRefresh, channelsRefresh, connectorsRefresh)
        }
    }

    private func refreshOnAppActivation() async {
        let shouldRefreshConnectorCards = selectedSection == .connectors
        connectorPermissionRefreshPendingIDs.removeAll()
        guard shouldRefreshConnectorCards, resolvedAuthToken() != nil else {
            return
        }
        await fetchConnectorCards()
    }

    private func handleSelectedSectionDidChange() {
        navigationStore.applySelectedSectionChangeSideEffects()
        Task { [weak self] in
            guard let self else {
                return
            }
            await self.refreshDataForCurrentSection(trigger: .transition)
            await self.resumePendingChatFixAndContinueIfNeeded()
        }
    }

    private func startLifecyclePolling() {
        lifecyclePollingTask?.cancel()
        lifecyclePollingTask = Task { [weak self] in
            guard let self else {
                return
            }
            while !Task.isCancelled {
                try? await Task.sleep(for: .seconds(4))
                await self.refreshDaemonLifecycleStatus(showLoadingState: false)
            }
        }
    }

    private func presentHighImpactActionConfirmation(
        title: String,
        message: String,
        confirmButtonTitle: String,
        isDestructive: Bool,
        irreversibleNote: String? = nil,
        onConfirm: @escaping () -> Void
    ) {
        workflowMutationStore.presentHighImpactActionConfirmation(
            title: title,
            message: message,
            confirmButtonTitle: confirmButtonTitle,
            isDestructive: isDestructive,
            irreversibleNote: irreversibleNote,
            onConfirm: onConfirm
        )
    }

    private func presentUndoActionPrompt(
        title: String,
        message: String,
        actionTitle: String = "Undo",
        visibleForSeconds: TimeInterval = 8,
        onUndo: @escaping () -> Void
    ) {
        workflowMutationStore.presentUndoActionPrompt(
            title: title,
            message: message,
            actionTitle: actionTitle,
            visibleForSeconds: visibleForSeconds,
            onUndo: onUndo
        )
    }

    private func daemonLocalSetupActionDisabledReason(for action: String) -> String? {
        let normalizedAction = action.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        guard normalizedAction == "install" || normalizedAction == "repair" else {
            return "Action is currently unavailable."
        }
        if !localDevTokenConfigured {
            return normalizedAction == "install"
                ? "Save Assistant Access Token before daemon installation actions."
                : "Save Assistant Access Token before daemon repair actions."
        }
        if isDaemonControlInFlight {
            return "A daemon lifecycle action is already in progress."
        }
        return nil
    }

    private func requestDaemonLifecycleControlWithConfirmation(action: String) {
        let normalizedAction = action.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        switch normalizedAction {
        case "start":
            requestStartDaemon()
        case "stop":
            requestStopDaemon()
        case "restart":
            requestRestartDaemon()
        case "install":
            requestInstallDaemon()
        case "uninstall":
            requestUninstallDaemon()
        case "repair":
            requestRepairDaemonInstallation()
        default:
            requestDaemonLifecycleControl(action: normalizedAction)
        }
    }

    private func requestDaemonLifecycleControl(action: String) {
        Task { [weak self] in
            await self?.performDaemonLifecycleControl(action: action)
        }
    }

    private func performDaemonLifecycleControl(
        action: String,
        waitForOperationCompletion: Bool = true,
        reasonContext: String? = nil
    ) async {
        guard let authToken = resolvedAuthToken() else {
            daemonStatusDetail = "Set Assistant Access Token before controlling daemon lifecycle."
            return
        }

        let normalizedAction = action.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        let resolvedReasonContext = reasonContext ?? "ui:\(selectedSection.title.lowercased())"
        isDaemonControlInFlight = true
        daemonStatusDetail = "\(runtimeLifecycleStore.daemonLifecycleActionLabel(normalizedAction)) requested..."
        appendNotification(
            source: "configuration",
            action: "daemon_\(normalizedAction)",
            message: daemonStatusDetail,
            level: .progress
        )
        defer {
            isDaemonControlInFlight = false
        }

        if normalizedAction == "install" || normalizedAction == "repair" {
            await performLocalDaemonInstallOrRepair(
                action: normalizedAction,
                authToken: authToken
            )
            return
        }

        do {
            let response = try await Self.daemonLifecycleControlRunner(
                self,
                authToken,
                normalizedAction,
                resolvedReasonContext
            )
            daemonStatusDetail = runtimeLifecycleStore.daemonLifecycleControlResponseSummary(response)
            let operationState = response.operationState.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
            let level: AppNotificationLevel
            switch operationState {
            case "failed":
                level = .error
            case "in_progress":
                level = .progress
            default:
                level = .success
            }
            appendNotification(
                source: "configuration",
                action: "daemon_\(normalizedAction)",
                message: daemonStatusDetail,
                level: level
            )
            await refreshDaemonLifecycleStatus()
            if waitForOperationCompletion && response.operationState.lowercased() == "in_progress" {
                await waitForDaemonControlOperationToComplete(
                    authToken: authToken,
                    expectedAction: normalizedAction
                )
            }
        } catch is CancellationError {
            return
        } catch {
            daemonStatusDetail = daemonErrorMessage(error, fallbackContext: "Failed to \(normalizedAction) daemon")
            appendNotification(
                source: "configuration",
                action: "daemon_\(normalizedAction)",
                message: daemonStatusDetail,
                level: .error
            )
        }
    }

    private func performLocalDaemonInstallOrRepair(
        action: String,
        authToken: String
    ) async {
        guard localDevTokenConfigured else {
            daemonStatusDetail = "Save Assistant Access Token before daemon \(action) actions."
            appendNotification(
                source: "configuration",
                action: "daemon_\(action)",
                message: daemonStatusDetail,
                level: .error
            )
            return
        }

        daemonControlOperationAction = action
        daemonControlOperationState = "in_progress"

        do {
            let result = try await Self.daemonLocalServiceInstallRunner(action, authToken)
            daemonControlOperationState = "succeeded"
            let completionSummary = "\(runtimeLifecycleStore.daemonLifecycleActionLabel(action)) completed. \(result.summaryText)"
            daemonStatusDetail = completionSummary
            if action == "install" {
                daemonNeedsInstall = false
            } else if action == "repair" {
                daemonNeedsRepair = false
            }
            appendNotification(
                source: "configuration",
                action: "daemon_\(action)",
                message: daemonStatusDetail,
                level: .success
            )
            await refreshDaemonLifecycleStatus(showLoadingState: false)
            if connectionStatus != .connected {
                daemonStatusDetail = completionSummary
            }
        } catch {
            daemonControlOperationState = "failed"
            if let installError = error as? DaemonLocalServiceInstallError,
               let detail = installError.errorDescription,
               !detail.isEmpty {
                daemonStatusDetail = detail
            } else {
                daemonStatusDetail = daemonErrorMessage(
                    error,
                    fallbackContext: "Failed to \(action) daemon setup"
                )
            }
            appendNotification(
                source: "configuration",
                action: "daemon_\(action)",
                message: daemonStatusDetail,
                level: .error
            )
        }
    }

    private func refreshDaemonLifecycleStatus(showLoadingState: Bool = true) async {
        if isDaemonLifecycleRequestInFlight {
            return
        }
        isDaemonLifecycleRequestInFlight = true
        if showLoadingState {
            isDaemonLifecycleLoading = true
        }
        defer {
            isDaemonLifecycleRequestInFlight = false
            if showLoadingState {
                isDaemonLifecycleLoading = false
            }
            hasLoadedDaemonStatus = true
            updateOnboardingCompletionState()
        }

        guard let authToken = resolvedAuthToken() else {
            runtimeLifecycleStore.applyMissingTokenState()
            return
        }

        do {
            let lifecycle = try await daemonClient.lifecycle.daemonLifecycleStatus(
                baseURL: daemonBaseURL,
                authToken: authToken
            )
            applyDaemonLifecycleStatus(lifecycle)
        } catch {
            let detail = daemonErrorMessage(error, fallbackContext: "Daemon lifecycle query failed")
            runtimeLifecycleStore.applyLifecycleError(detail: detail)
        }
    }

    private func applyDaemonLifecycleStatus(_ lifecycle: DaemonLifecycleStatusResponse) {
        runtimeLifecycleStore.applyDaemonLifecycleStatus(lifecycle)
    }

    private func waitForDaemonControlOperationToComplete(
        authToken: String,
        expectedAction: String
    ) async {
        let deadline = Date().addingTimeInterval(20)
        while Date() < deadline, !Task.isCancelled {
            try? await Task.sleep(for: .milliseconds(350))
            do {
                let lifecycle = try await daemonClient.lifecycle.daemonLifecycleStatus(
                    baseURL: daemonBaseURL,
                    authToken: authToken
                )
                applyDaemonLifecycleStatus(lifecycle)
                let operation = lifecycle.controlOperation
                let operationAction = operation.action.lowercased()
                guard operationAction == expectedAction.lowercased() else {
                    continue
                }
                let operationState = operation.state.lowercased()
                if operationState != "in_progress" {
                    return
                }
            } catch {
                daemonStatusDetail = daemonErrorMessage(
                    error,
                    fallbackContext: "Failed to query daemon operation status"
                )
                return
            }
        }
    }

    private func fetchDaemonPluginLifecycleHistory() async {
        if isDaemonPluginLifecycleHistoryRequestInFlight {
            return
        }
        isDaemonPluginLifecycleHistoryRequestInFlight = true
        isDaemonPluginLifecycleHistoryLoading = true
        defer {
            isDaemonPluginLifecycleHistoryRequestInFlight = false
            isDaemonPluginLifecycleHistoryLoading = false
        }

        guard let authToken = resolvedAuthToken() else {
            daemonPluginLifecycleHistoryItems = []
            daemonPluginLifecycleTrendItems = []
            daemonPluginLifecycleHistoryHasMore = false
            daemonPluginLifecycleHistoryStatusMessage = RuntimePluginLifecycleProjection.missingTokenStatusMessage
            return
        }

        let normalizedPluginID = nonEmpty(daemonPluginLifecycleHistoryFilterPluginID)
        let normalizedKind = RuntimePluginLifecycleProjection.normalizedFilter(daemonPluginLifecycleHistoryFilterKind)
        let normalizedState = RuntimePluginLifecycleProjection.normalizedFilter(daemonPluginLifecycleHistoryFilterState)
        let normalizedEventType = RuntimePluginLifecycleProjection.normalizedFilter(daemonPluginLifecycleHistoryFilterEventType)
        let requestLimit = RuntimePluginLifecycleProjection.clampedLimit(daemonPluginLifecycleHistoryLimit)
        if daemonPluginLifecycleHistoryLimit != requestLimit {
            daemonPluginLifecycleHistoryLimit = requestLimit
        }

        do {
            let response = try await daemonClient.lifecycle.daemonPluginLifecycleHistory(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: nil,
                pluginID: normalizedPluginID,
                kind: normalizedKind,
                state: normalizedState,
                eventType: normalizedEventType,
                cursorCreatedAt: nil,
                cursorID: nil,
                limit: requestLimit
            )
            let mappedItems = RuntimePluginLifecycleProjection.mapAndSortRecords(
                response.items,
                parseTimestamp: { [self] value in
                    parseDaemonTimestamp(value)
                },
                formatTimestamp: { [self] value in
                    formattedWorkflowTimestamp(value)
                }
            )
            daemonPluginLifecycleHistoryItems = mappedItems
            daemonPluginLifecycleTrendItems = RuntimePluginLifecycleProjection.buildTrendItems(from: mappedItems)
            daemonPluginLifecycleHistoryHasMore = response.hasMore
            daemonPluginLifecycleHistoryStatusMessage = RuntimePluginLifecycleProjection.summaryMessage(
                itemCount: mappedItems.count,
                hasMore: response.hasMore,
                pluginID: normalizedPluginID,
                kind: normalizedKind,
                state: normalizedState,
                eventType: normalizedEventType
            )
            connectionStatus = .connected
        } catch {
            daemonPluginLifecycleHistoryItems = []
            daemonPluginLifecycleTrendItems = []
            daemonPluginLifecycleHistoryHasMore = false
            daemonPluginLifecycleHistoryStatusMessage = daemonErrorMessage(
                error,
                fallbackContext: "Runtime plugin lifecycle history query failed",
                updateConnectionStatus: false
            )
        }
    }

    private func refreshDataForCurrentSection(trigger: PanelLatencyTrigger = .transition) async {
        let section = selectedSection
        let category = panelLatencyStore.panelLatencyCategory(
            for: section,
            trigger: trigger,
            hasLoadedPanelData: { [self] target in
                hasLoadedPanelData(for: target)
            }
        )
        let startedAtNanoseconds = DispatchTime.now().uptimeNanoseconds
        defer {
            let elapsedNanoseconds = DispatchTime.now().uptimeNanoseconds - startedAtNanoseconds
            let durationMS = Int(elapsedNanoseconds / 1_000_000)
            panelLatencyStore.recordPanelLatencySample(
                section: section,
                category: category,
                durationMS: durationMS
            )
        }

        switch section {
        case .home:
            stopInspectStream()
        case .inspect:
            await fetchInspectLogs()
            startInspectStream()
        case .communications:
            await fetchCommunicationsInbox()
            await fetchCommunicationAttempts(threadID: communicationsStore.communicationAttemptContextThreadID)
            stopInspectStream()
        case .channels:
            await fetchChannelCards()
            stopInspectStream()
        case .connectors:
            await fetchConnectorCards()
            stopInspectStream()
        case .models:
            async let lifecycleRefresh: Void = refreshDaemonLifecycleStatus(
                showLoadingState: !hasLoadedDaemonStatus
            )
            async let providerRefresh: Void = fetchProviderAndModelStatus(runChecks: false)
            _ = await (lifecycleRefresh, providerRefresh)
            stopInspectStream()
        case .configuration:
            async let lifecycleRefresh: Void = refreshDaemonLifecycleStatus(
                showLoadingState: !hasLoadedDaemonStatus
            )
            async let principalRefresh: Void = fetchPrincipalOptions()
            async let identityInventoryRefresh: Void = fetchIdentityDeviceAndSessionPanels()
            async let delegationRefresh: Void = fetchDelegationRules()
            async let personaRefresh: Void = fetchChatPersonaPolicy()
            async let governanceRefresh: Void = fetchCapabilityGrantAndTrustReceiptPanels()
            async let providerRefresh: Void = fetchProviderAndModelStatus(runChecks: false)
            async let pluginHistoryRefresh: Void = fetchDaemonPluginLifecycleHistory()
            async let contextMemoryRefresh: Void = fetchContextMemoryAndRetrievalPanels()
            _ = await (
                lifecycleRefresh,
                principalRefresh,
                identityInventoryRefresh,
                delegationRefresh,
                personaRefresh,
                governanceRefresh,
                providerRefresh,
                pluginHistoryRefresh,
                contextMemoryRefresh
            )
            stopInspectStream()
        case .chat:
            await hydrateChatTimelineFromHistoryIfEmpty()
            stopInspectStream()
        case .automation:
            await fetchAutomationPanelData()
            stopInspectStream()
        case .approvals:
            await fetchApprovalsInbox()
            stopInspectStream()
        case .tasks:
            await fetchTaskRunList()
            stopInspectStream()
        }
    }

    private func hasLoadedPanelData(for section: AppSection) -> Bool {
        switch section {
        case .configuration:
            return hasLoadedDaemonStatus
        case .home:
            return true
        case .chat:
            return true
        case .communications:
            return hasLoadedCommunicationsInbox
        case .automation:
            return hasLoadedAutomationPanelData
        case .approvals:
            return hasLoadedApprovalsInbox
        case .tasks:
            return hasLoadedTaskRunList
        case .inspect:
            return hasLoadedInspectLogs
        case .channels:
            return hasLoadedChannelStatus
        case .connectors:
            return hasLoadedConnectorStatus
        case .models:
            return hasLoadedProviderStatus
        }
    }

    private func performChatRoutePreflightOnly() async {
        pendingChatFixAndContinue = nil
        chatFixAndContinueStatusMessage = nil
        guard let authToken = resolvedAuthToken() else {
            clearPanelProblemSignal(for: .chat)
            chatStatusMessage = "Set Assistant Access Token before checking chat route."
            chatProgressDetail = nil
            chatRouteRemediationMessage = nil
            chatFailureRemediationMessage = nil
            return
        }

        chatStatusMessage = "Checking chat route…"
        chatProgressDetail = "Validating model availability before send…"
        _ = await validateChatRouteBeforeSend(authToken: authToken, updateChatStatusOnSuccess: true)
    }

    private func performChatSendWithPreflight(draft: String) async {
        guard let authToken = resolvedAuthToken() else {
            isChatStreaming = false
            isChatInterruptInFlight = false
            chatProgressDetail = nil
            chatStatusMessage = "Set Assistant Access Token before sending chat turns."
            chatRouteRemediationMessage = nil
            chatFailureRemediationMessage = nil
            chatOrchestrationStore.turnTask = nil
            return
        }

        let routeReady = await validateChatRouteBeforeSend(authToken: authToken, updateChatStatusOnSuccess: false)
        if !routeReady {
            if isChatInterruptInFlight || Task.isCancelled {
                chatStatusMessage = "Chat interrupted."
                chatRouteRemediationMessage = nil
                chatFailureRemediationMessage = nil
            }
            isChatStreaming = false
            isChatInterruptInFlight = false
            chatProgressDetail = nil
            chatOrchestrationStore.turnTask = nil
            return
        }

        guard !Task.isCancelled else {
            chatStatusMessage = "Chat interrupted."
            chatProgressDetail = nil
            chatRouteRemediationMessage = nil
            chatFailureRemediationMessage = nil
            isChatStreaming = false
            isChatInterruptInFlight = false
            chatOrchestrationStore.turnTask = nil
            return
        }

        appendChatTimelineUserMessage(content: draft)
        chatDraft = ""
        chatStatusMessage = "Sending to daemon…"
        chatProgressDetail = "Connecting realtime stream…"
        await sendChatTurnToDaemon(
            authToken: authToken,
            submittedDraft: draft
        )
    }

    private func validateChatRouteBeforeSend(
        authToken: String,
        updateChatStatusOnSuccess: Bool,
        clearFailedDraftOnRouteFailure: Bool = true
    ) async -> Bool {
        do {
            let route = try await daemonClient.models.modelResolve(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                taskClass: "chat"
            )
            modelRouteSummary = ModelRouteSummary(
                provider: route.provider,
                modelKey: route.modelKey,
                source: route.source,
                notes: nonEmpty(route.notes)
            )
            modelRouteStatusMessage = "Resolved chat route from \(route.source)."
            chatRouteRemediationMessage = nil
            chatFailureRemediationMessage = nil
            clearPanelProblemSignal(for: .chat)
            connectionStatus = .connected
            if updateChatStatusOnSuccess {
                chatStatusMessage = "Chat route is ready. Send a message to continue."
                chatProgressDetail = nil
            }
            return true
        } catch is CancellationError {
            return false
        } catch {
            if isMissingReadyChatModelRoute(error) {
                let guidance = missingReadyChatModelRouteGuidance
                modelRouteSummary = nil
                modelRouteStatusMessage = guidance
                chatRouteRemediationMessage = guidance
                chatStatusMessage = guidance
                chatProgressDetail = nil
                chatFailureRemediationMessage = nil
                if clearFailedDraftOnRouteFailure {
                    chatLastFailedDraft = nil
                }
                clearPanelProblemSignal(for: .chat)
                connectionStatus = .connected
                return false
            }

            let remediation = chatSendFailureGuidance(error)
            chatRouteRemediationMessage = nil
            chatFailureRemediationMessage = remediation
            let message = daemonErrorMessage(
                error,
                fallbackContext: "Chat route check failed",
                updateConnectionStatus: true,
                panelContext: .chat
            )
            modelRouteStatusMessage = message
            chatStatusMessage = message
            chatProgressDetail = nil
            return false
        }
    }

    private func sendChatTurnToDaemon(
        authToken: String,
        submittedDraft: String
    ) async {
        await chatTurnExecutionStore.executeTurn(
            authToken: authToken,
            submittedDraft: submittedDraft,
            bindings: ChatTurnExecutionStore.Bindings(
                daemonClient: daemonClient,
                daemonBaseURL: daemonBaseURL,
                workspaceID: workspaceID,
                parseDaemonTimestamp: { [weak self] value in
                    guard let self else {
                        return nil
                    }
                    return self.parseDaemonTimestamp(value)
                },
                beginTurnExecution: { [weak self] correlationID in
                    self?.beginChatTurnExecution(correlationID: correlationID)
                },
                daemonMessagesForSubmission: { [weak self] in
                    self?.chatDaemonMessagesForSubmission() ?? []
                },
                selectedActorIDForSubmission: { [weak self] in
                    self?.selectedActorIDForChatSubmission()
                },
                handleRealtimeConnected: { [weak self] session, correlationID in
                    self?.handleRealtimeSessionConnectedForChatTurn(
                        session: session,
                        correlationID: correlationID
                    )
                },
                handleRealtimeConnectionFailure: { [weak self] error in
                    self?.handleRealtimeSessionConnectionFailureForChatTurn(error)
                },
                handleTurnResponse: { [weak self] response, correlationID, authToken, selectedActorID in
                    guard let self else {
                        return
                    }
                    await self.handleChatTurnResponseFromExecutionStore(
                        response: response,
                        correlationID: correlationID,
                        authToken: authToken,
                        selectedActorID: selectedActorID
                    )
                },
                handleTurnCancellation: { [weak self] correlationID in
                    self?.handleChatTurnCancellationFromExecutionStore(correlationID: correlationID)
                },
                handleRecoveredSnapshot: { [weak self] snapshot, authToken, requestedByActorID, subjectActorID, actingAsActorID in
                    guard let self else {
                        return
                    }
                    await self.applyRecoveredChatTurnSnapshotFromExecutionStore(
                        snapshot: snapshot,
                        authToken: authToken,
                        requestedByActorID: requestedByActorID,
                        subjectActorID: subjectActorID,
                        actingAsActorID: actingAsActorID
                    )
                },
                handleTurnFailure: { [weak self] error, correlationID, submittedDraft in
                    self?.handleChatTurnFailureFromExecutionStore(
                        error: error,
                        correlationID: correlationID,
                        submittedDraft: submittedDraft
                    )
                },
                finishTurnExecution: { [weak self] in
                    guard let self else {
                        return
                    }
                    await self.finishChatTurnExecutionFromExecutionStore()
                }
            )
        )
    }

    private func beginChatTurnExecution(correlationID: String) {
        chatActiveCorrelationID = correlationID
        chatOrchestrationStore.resetRealtimeTrackingForNewTurn()
        chatTurnContextStore.resetForNewTurn()
        syncChatTurnContextProjectionFromStore()
        chatTimelineStore.resetForNewTurn(existingTimeline: chatTimelineItems)
    }

    private func chatDaemonMessagesForSubmission() -> [(role: String, content: String)] {
        chatTimelineItems.compactMap(\.daemonContextMessage)
    }

    private func handleRealtimeSessionConnectedForChatTurn(
        session: DaemonRealtimeSession,
        correlationID: String
    ) {
        chatOrchestrationStore.realtimeSession = session
        chatOrchestrationStore.realtimeConnectedForActiveTurn = true
        chatProgressDetail = "Realtime stream connected."
        chatStatusMessage = "Streaming via daemon realtime…"
        chatOrchestrationStore.realtimeTask?.cancel()
        chatOrchestrationStore.realtimeTask = Task { [weak self] in
            await self?.consumeChatRealtimeEvents(
                session: session,
                correlationID: correlationID
            )
        }
    }

    private func handleRealtimeSessionConnectionFailureForChatTurn(_ error: Error) {
        chatOrchestrationStore.realtimeSession = nil
        let fallbackContext = chatRealtimeFallbackContext(from: error)
        applyChatRealtimeFallbackContext(fallbackContext)
    }

    private func handleChatTurnResponseFromExecutionStore(
        response: DaemonChatTurnResponse,
        correlationID: String,
        authToken: String,
        selectedActorID: String?
    ) async {
        chatTurnContextStore.updateTraceabilityFromTurnResponse(
            response.taskRunCorrelation,
            fallbackCorrelationID: nonEmpty(response.correlationID) ?? correlationID,
            taskClass: nonEmpty(response.taskClass),
            provider: nonEmpty(response.provider),
            modelKey: nonEmpty(response.modelKey),
            routeSource: nonEmpty(modelRouteSummary?.source),
            channelID: nonEmpty(response.channel?.channelID),
            turnContractVersion: nonEmpty(response.contractVersion),
            turnItemSchemaVersion: nonEmpty(response.turnItemSchemaVersion),
            realtimeEventContractVersion: nonEmpty(response.realtimeEventContractVersion),
            items: response.items
        )
        syncChatTurnContextProjectionFromStore()
        reconcileChatTurnTimelineItems(
            items: response.items,
            correlationID: nonEmpty(response.correlationID) ?? correlationID,
            taskCorrelation: response.taskRunCorrelation
        )
        let realtimeConnectedForTurn = chatRealtimeTransportConnectedForActiveTurn()
        chatStatusMessage = chatTurnCompletionStatusMessage(
            response: response,
            realtimeConnected: realtimeConnectedForTurn
        )
        chatProgressDetail = nil
        connectionStatus = realtimeConnectedForTurn ? .connected : .degraded
        pendingChatFixAndContinue = nil
        chatFixAndContinueStatusMessage = nil
        chatRouteRemediationMessage = nil
        chatFailureRemediationMessage = nil
        chatLastFailedDraft = nil
        clearPanelProblemSignal(for: .chat)
        appendNotification(
            source: "chat",
            action: "chat_turn",
            message: chatTurnNotificationSummary(
                response: response,
                realtimeConnected: realtimeConnectedForTurn
            ),
            level: .success
        )
        markHomeFirstSessionStepComplete(
            .sendMessage,
            source: "chat_turn",
            completedAt: chatTimelineItems.last?.timestamp ?? .now
        )
        Task { [weak self] in
            await self?.performChatTurnExplainabilityFetch(
                authToken: authToken,
                requestedByActorID: selectedActorID,
                subjectActorID: selectedActorID,
                actingAsActorID: selectedActorID,
                userInitiated: false,
                retainPreviousOnFailure: false
            )
        }
    }

    private func handleChatTurnCancellationFromExecutionStore(correlationID: String) {
        if chatTimelineStore.streamedAssistantText.isEmpty {
            appendChatTimelineSystemStatus(
                state: .failed,
                title: "Turn Interrupted",
                summary: "Chat interrupted before daemon response completed.",
                correlationID: correlationID
            )
        }
        chatStatusMessage = "Chat interrupted."
        chatProgressDetail = nil
        chatFailureRemediationMessage = nil
        chatTurnContextStore.markExplainabilityInterrupted()
        syncChatTurnContextProjectionFromStore()
    }

    private func applyRecoveredChatTurnSnapshotFromExecutionStore(
        snapshot: ChatTurnExecutionStore.RecoveredChatTurnSnapshot,
        authToken: String,
        requestedByActorID: String?,
        subjectActorID: String?,
        actingAsActorID: String?
    ) async {
        updateWorkspaceContext(from: snapshot.workspaceID)

        chatTurnContextStore.updateTraceabilityFromTurnResponse(
            snapshot.taskRunCorrelation,
            fallbackCorrelationID: snapshot.correlationID,
            taskClass: nonEmpty(snapshot.taskClass),
            provider: nonEmpty(modelRouteSummary?.provider),
            modelKey: nonEmpty(modelRouteSummary?.modelKey),
            routeSource: nonEmpty(modelRouteSummary?.source),
            channelID: nonEmpty(snapshot.channelID),
            turnContractVersion: nil,
            turnItemSchemaVersion: nil,
            realtimeEventContractVersion: nil,
            items: snapshot.items
        )
        syncChatTurnContextProjectionFromStore()
        reconcileChatTurnTimelineItems(
            items: snapshot.items,
            correlationID: snapshot.correlationID,
            taskCorrelation: snapshot.taskRunCorrelation
        )

        let realtimeConnectedForTurn = chatRealtimeTransportConnectedForActiveTurn()
        chatStatusMessage = recoveredChatTurnCompletionStatusMessage(
            items: snapshot.items,
            taskRunCorrelation: snapshot.taskRunCorrelation,
            realtimeConnected: realtimeConnectedForTurn
        )
        chatProgressDetail = nil
        connectionStatus = realtimeConnectedForTurn ? .connected : .degraded
        chatRouteRemediationMessage = nil
        chatFailureRemediationMessage = nil
        chatLastFailedDraft = nil
        clearPanelProblemSignal(for: .chat)
        appendNotification(
            source: "chat",
            action: "chat_turn",
            message: "Recovered chat turn from daemon history after a temporary transport interruption.",
            level: .success
        )
        Task { [weak self] in
            await self?.performChatTurnExplainabilityFetch(
                authToken: authToken,
                requestedByActorID: requestedByActorID,
                subjectActorID: subjectActorID,
                actingAsActorID: actingAsActorID,
                userInitiated: false,
                retainPreviousOnFailure: true
            )
        }
    }

    private func handleChatTurnFailureFromExecutionStore(
        error: Error,
        correlationID: String,
        submittedDraft: String
    ) {
        if isMissingReadyChatModelRoute(error) {
            let guidance = missingReadyChatModelRouteGuidance
            modelRouteSummary = nil
            modelRouteStatusMessage = guidance
            chatRouteRemediationMessage = guidance
            chatFailureRemediationMessage = nil
            chatLastFailedDraft = nil
            chatStatusMessage = guidance
            clearPanelProblemSignal(for: .chat)
            chatFixAndContinueStatusMessage = nil
        } else if (error as? DaemonAPIError)?.isConnectivityIssue == true, chatTimelineStore.realtimeCompletionReceived {
            if let realtimeErrorMessage = nonEmpty(chatTimelineStore.realtimeErrorMessage) {
                if chatTimelineStore.streamedAssistantText.isEmpty {
                    appendChatTimelineSystemStatus(
                        state: .failed,
                        title: "Realtime Error",
                        summary: realtimeErrorMessage,
                        correlationID: correlationID
                    )
                }
                chatStatusMessage = realtimeErrorMessage
                chatFailureRemediationMessage = chatSendFailureGuidance(error)
                chatLastFailedDraft = nonEmpty(submittedDraft)
                connectionStatus = .degraded
                appendNotification(
                    source: "chat",
                    action: "chat_turn",
                    message: realtimeErrorMessage,
                    level: .error
                )
            } else {
                chatStatusMessage = "Chat completed via realtime stream. Final daemon receipt was unavailable, but streamed output was preserved."
                chatFailureRemediationMessage = nil
                chatLastFailedDraft = nil
                connectionStatus = chatRealtimeTransportConnectedForActiveTurn() ? .connected : .degraded
                appendNotification(
                    source: "chat",
                    action: "chat_turn",
                    message: "Chat completed via realtime stream (daemon receipt unavailable).",
                    level: .success
                )
            }
            chatRouteRemediationMessage = nil
            chatFixAndContinueStatusMessage = nil
        } else {
            let message = daemonErrorMessage(
                error,
                fallbackContext: "Chat request failed",
                updateConnectionStatus: false,
                panelContext: .chat
            )
            if chatTimelineStore.streamedAssistantText.isEmpty {
                appendChatTimelineSystemStatus(
                    state: .failed,
                    title: "Turn Failed",
                    summary: message,
                    correlationID: correlationID
                )
                chatStatusMessage = message
            } else {
                chatStatusMessage = "\(message) Partial streamed output was preserved."
            }
            chatRouteRemediationMessage = nil
            chatFailureRemediationMessage = chatSendFailureGuidance(error)
            chatLastFailedDraft = nonEmpty(submittedDraft)
            chatFixAndContinueStatusMessage = nil
        }

        chatProgressDetail = nil
        if !isMissingReadyChatModelRoute(error) {
            chatTurnContextStore.markExplainabilityUnavailableUntilTurnCompletes()
        } else {
            chatTurnContextStore.markExplainabilityUnavailableForRouteSetup()
        }
        syncChatTurnContextProjectionFromStore()
        appendNotification(
            source: "chat",
            action: "chat_turn",
            message: chatStatusMessage ?? "Chat request failed.",
            level: .error
        )
    }

    private func finishChatTurnExecutionFromExecutionStore() async {
        await closeChatRealtimeSession()
        isChatStreaming = false
        isChatInterruptInFlight = false
        chatActiveCorrelationID = nil
        chatOrchestrationStore.turnTask = nil
    }

    private func recoveredChatTurnCompletionStatusMessage(
        items: [DaemonChatTurnItem],
        taskRunCorrelation: DaemonChatTurnTaskRunCorrelation,
        realtimeConnected: Bool
    ) -> String {
        let transportLabel = realtimeConnected ? "Realtime" : "Fallback"
        let provider = nonEmpty(chatLatestTurnTraceability?.provider)
            ?? nonEmpty(modelRouteSummary?.provider)
            ?? "unknown"
        let modelKey = nonEmpty(chatLatestTurnTraceability?.modelKey)
            ?? nonEmpty(modelRouteSummary?.modelKey)
            ?? "unknown"
        let signals = chatTurnContextStore.chatTurnSignals(from: items)
        if signals.clarificationRequired {
            let prompt = nonEmpty(signals.clarificationPrompt) ?? "additional details are required"
            return "Action needs clarification: \(prompt)"
        }
        if signals.approvalRequired {
            if let approvalRequestID = nonEmpty(signals.approvalRequestID) {
                return "Action awaiting approval (\(approvalRequestID)) • \(transportLabel)"
            }
            return "Action awaiting approval • \(transportLabel)"
        }
        let hasToolCalls = items.contains {
            $0.type.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() == "tool_call"
        }
        let hasToolFailure = items.contains { item in
            item.type.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() == "tool_result"
                && ChatTimelineEventStore.normalizedTimelineItemState(item.status) == .failed
        }
        if hasToolFailure {
            return "Tool execution failed • Provider: \(providerDisplayName(provider)) • Model: \(modelKey) • \(transportLabel)"
        }
        if hasToolCalls {
            let runState = nonEmpty(taskRunCorrelation.runState)?
                .replacingOccurrences(of: "_", with: " ")
                .capitalized ?? "Completed"
            return "Action run state: \(runState) • Provider: \(providerDisplayName(provider)) • Model: \(modelKey) • \(transportLabel)"
        }
        return "Provider: \(providerDisplayName(provider)) • Model: \(modelKey) • \(transportLabel)"
    }

    private func performChatInterrupt() async {
        guard isChatStreaming else {
            return
        }
        isChatInterruptInFlight = true
        chatProgressDetail = "Interrupt requested…"

        if let session = chatOrchestrationStore.realtimeSession {
            do {
                try await session.sendSignal(
                    DaemonRealtimeClientSignal(
                        signalType: "cancel",
                        taskID: nil,
                        runID: nil,
                        reason: "ui:chat_interrupt",
                        correlationID: chatActiveCorrelationID
                    )
                )
            } catch {
                chatProgressDetail = "Realtime signal failed; cancelling local request."
            }
        } else {
            chatProgressDetail = "Realtime unavailable; cancelling local request."
        }
        chatOrchestrationStore.turnTask?.cancel()
    }

    private func performChatRealtimeRetry(authToken: String) async {
        defer {
            isChatRealtimeRetryInFlight = false
        }

        let probeCorrelationID = "chat-realtime-retry-\(UUID().uuidString.lowercased())"
        do {
            let session = try daemonClient.chat.connectRealtime(
                baseURL: daemonBaseURL,
                authToken: authToken,
                correlationID: probeCorrelationID
            )
            try await session.ping()
            await session.close()
            guard !isChatStreaming else {
                return
            }
            chatOrchestrationStore.realtimeFallbackReason = nil
            chatOrchestrationStore.realtimeFallbackDetail = nil
            connectionStatus = .connected
            chatStatusMessage = "Realtime stream reachable. Next turn will stream live."
            chatProgressDetail = nil
            appendNotification(
                source: "chat",
                action: "realtime_retry",
                message: "Realtime stream reconnected.",
                level: .success
            )
        } catch {
            guard !isChatStreaming else {
                return
            }
            let fallbackContext = chatRealtimeFallbackContext(from: error)
            chatOrchestrationStore.realtimeFallbackReason = fallbackContext.reason
            chatOrchestrationStore.realtimeFallbackDetail = fallbackContext.remediationDetail
            connectionStatus = .degraded
            chatStatusMessage = fallbackContext.statusMessage
            chatProgressDetail = fallbackContext.progressDetail
            appendNotification(
                source: "chat",
                action: "realtime_retry",
                message: fallbackContext.notificationSummary,
                level: .error
            )
        }
    }

    private func consumeChatRealtimeEvents(
        session: DaemonRealtimeSession,
        correlationID: String
    ) async {
        while !Task.isCancelled {
            do {
                let event = try await session.receive()
                if let eventCorrelationID = nonEmpty(event.correlationID), eventCorrelationID != correlationID {
                    continue
                }
                if chatTurnContextStore.mergeRealtimeLifecycleContractMetadata(
                    from: event,
                    fallbackCorrelationID: correlationID,
                    routeProvider: nonEmpty(modelRouteSummary?.provider),
                    routeModelKey: nonEmpty(modelRouteSummary?.modelKey),
                    routeSource: nonEmpty(modelRouteSummary?.source)
                ) {
                    syncChatTurnContextProjectionFromStore()
                }

                switch event.eventType {
                case "chat_completed":
                    chatTimelineStore.markRealtimeCompleted()
                    chatProgressDetail = "Realtime response completed. Finalizing daemon receipt…"
                case "chat_error":
                    let realtimeErrorSummary = chatRealtimeErrorSummaryForDisplay(
                        errorCode: nonEmpty(event.payload.errorCode),
                        message: nonEmpty(event.payload.message)
                    )
                    chatTimelineStore.markRealtimeError(realtimeErrorSummary)
                    let fallbackContext = chatRealtimeFallbackContext(
                        fromRealtimeEventCode: nonEmpty(event.payload.errorCode),
                        message: nonEmpty(event.payload.message),
                        defaultReason: .unavailable
                    )
                    chatOrchestrationStore.realtimeFallbackReason = fallbackContext.reason
                    chatOrchestrationStore.realtimeFallbackDetail = fallbackContext.remediationDetail
                    connectionStatus = .degraded
                    chatStatusMessage = fallbackContext.statusMessage
                    chatProgressDetail = chatTimelineStore.streamedAssistantText.isEmpty
                        ? fallbackContext.progressDetail
                        : "Realtime stream reported an error; preserving streamed output."
                case "turn_item_delta":
                    appendChatTimelineFromRealtimeEvent(event, correlationID: correlationID)
                    if isAssistantTurnItemDelta(event.payload) {
                        chatProgressDetail = "Streaming assistant response…"
                    }
                case "tool_call_started", "tool_call_output", "tool_call_completed", "turn_item_started", "turn_item_completed":
                    appendChatTimelineFromRealtimeEvent(event, correlationID: correlationID)
                default:
                    continue
                }
            } catch is CancellationError {
                return
            } catch {
                let fallbackContext = chatRealtimeFallbackContext(from: error, defaultReason: .disconnected)
                chatOrchestrationStore.realtimeFallbackReason = fallbackContext.reason
                chatOrchestrationStore.realtimeFallbackDetail = fallbackContext.remediationDetail
                connectionStatus = .degraded
                chatStatusMessage = fallbackContext.statusMessage
                if chatTimelineStore.streamedAssistantText.isEmpty {
                    chatProgressDetail = fallbackContext.progressDetail
                } else {
                    chatProgressDetail = "Realtime stream disconnected; preserving streamed output."
                }
                return
            }
        }
    }

    private func isAssistantTurnItemDelta(_ payload: DaemonRealtimeEventPayload) -> Bool {
        guard let itemType = nonEmpty(payload.itemType)?.lowercased(), itemType == "assistant_message" else {
            return false
        }
        guard let delta = payload.delta else {
            return false
        }
        return !delta.isEmpty
    }

    private func reconcileChatTurnTimelineItems(
        items: [DaemonChatTurnItem],
        correlationID: String,
        taskCorrelation: DaemonChatTurnTaskRunCorrelation
    ) {
        chatTimelineStore.reconcileChatTurnTimelineItems(
            items: items,
            correlationID: correlationID,
            taskCorrelation: taskCorrelation,
            activeCorrelationID: nonEmpty(chatActiveCorrelationID)
        )
        syncChatTimelineProjectionFromStore()
    }

    private func appendChatTimelineUserMessage(content: String) {
        chatTimelineStore.appendTimelineUserMessage(content: content)
        syncChatTimelineProjectionFromStore()
    }

    private func appendChatTimelineSystemStatus(
        state: ChatTimelineItemState,
        title: String,
        summary: String,
        correlationID: String?
    ) {
        chatTimelineStore.appendTimelineSystemStatus(
            state: state,
            title: title,
            summary: summary,
            correlationID: correlationID
        )
        syncChatTimelineProjectionFromStore()
    }

    private func appendChatTimelineFromRealtimeEvent(
        _ event: DaemonRealtimeEventEnvelope,
        correlationID: String
    ) {
        chatTimelineStore.appendTimelineFromRealtimeEvent(
            event,
            correlationID: correlationID,
            activeCorrelationID: nonEmpty(chatActiveCorrelationID)
        )
        syncChatTimelineProjectionFromStore()
    }

    private func syncChatTimelineProjectionFromStore() {
        chatTimelineItems = chatTimelineStore.timelineItems
    }

    private func syncChatTurnContextProjectionFromStore() {
        chatLatestTurnTraceability = chatTurnContextStore.latestTurnTraceability
        chatLatestTurnExplainability = chatTurnContextStore.latestTurnExplainability
        chatExplainabilityStatusMessage = chatTurnContextStore.explainabilityStatusMessage
        chatExplainabilityErrorMessage = chatTurnContextStore.explainabilityErrorMessage
    }

    private func normalizedBoolean(_ value: DaemonJSONValue?) -> Bool? {
        switch value {
        case .bool(let flag):
            return flag
        case .string(let raw):
            switch raw.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
            case "true", "1", "yes", "ok":
                return true
            case "false", "0", "no":
                return false
            default:
                return nil
            }
        default:
            return nil
        }
    }

    private func closeChatRealtimeSession() async {
        chatOrchestrationStore.realtimeTask?.cancel()
        chatOrchestrationStore.realtimeTask = nil
        if let session = chatOrchestrationStore.realtimeSession {
            await session.close()
            chatOrchestrationStore.realtimeSession = nil
        }
    }

    private func fetchInspectLogs() async {
        isInspectLoading = true
        defer {
            isInspectLoading = false
            hasLoadedInspectLogs = true
        }

        guard let authToken = resolvedAuthToken() else {
            inspectStatusMessage = "Set Assistant Access Token to query inspect logs."
            return
        }

        let focusedRunID = nonEmpty(inspectFocusedRunID)

        do {
            let response = try await daemonClient.inspect.inspectLogsQuery(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                runID: focusedRunID,
                limit: 120
            )
            updateWorkspaceContext(from: response.workspaceID)
            let mappedLogs = response.logs.map { record in
                inspectStore.mapInspectLogRecord(
                    record,
                    parseDaemonTimestamp: parseDaemonTimestamp(_:),
                    mapWorkflowRoute: mapWorkflowRoute(_:)
                )
            }
            inspectStore.applyInspectQuerySnapshot(
                logs: mappedLogs,
                focusedRunID: focusedRunID,
                workspaceID: response.workspaceID
            )
            connectionStatus = .connected
        } catch is CancellationError {
            return
        } catch {
            inspectStatusMessage = daemonErrorMessage(
                error,
                fallbackContext: "Inspect log query failed",
                updateConnectionStatus: false
            )
        }
    }

    private func startInspectStream() {
        guard isInspectLiveTailEnabled else {
            stopInspectStream()
            return
        }
        inspectStreamTask?.cancel()
        inspectStreamTask = Task { [weak self] in
            guard let self else {
                return
            }
            while !Task.isCancelled {
                await self.streamInspectLogsOnce()
            }
        }
    }

    private func stopInspectStream() {
        inspectStreamTask?.cancel()
        inspectStreamTask = nil
    }

    private func streamInspectLogsOnce() async {
        guard selectedSection == .inspect else {
            try? await Task.sleep(for: .milliseconds(200))
            return
        }
        guard let authToken = resolvedAuthToken() else {
            return
        }

        do {
            let response = try await daemonClient.inspect.inspectLogsStream(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                runID: nonEmpty(inspectFocusedRunID),
                cursorCreatedAt: inspectCursorCreatedAt,
                cursorID: inspectCursorID,
                limit: 120,
                timeoutMS: 1500,
                pollIntervalMS: 200
            )
            updateWorkspaceContext(from: response.workspaceID)
            if !response.logs.isEmpty {
                let incoming = response.logs.map { record in
                    inspectStore.mapInspectLogRecord(
                        record,
                        parseDaemonTimestamp: parseDaemonTimestamp(_:),
                        mapWorkflowRoute: mapWorkflowRoute(_:)
                    )
                }
                inspectStore.mergeInspectLogs(incoming)
                inspectStatusMessage = "Inspect live update received at \(Date.now.formatted(date: .omitted, time: .standard))."
            }
            inspectStore.updateInspectCursor(createdAt: response.cursorCreatedAt, cursorID: response.cursorID)
            connectionStatus = .connected
        } catch is CancellationError {
            return
        } catch {
            inspectStatusMessage = daemonErrorMessage(
                error,
                fallbackContext: "Inspect stream failed",
                updateConnectionStatus: false
            )
            try? await Task.sleep(for: .seconds(2))
        }
    }

    private func fetchCommunicationsInbox() async {
        isCommunicationsLoading = true
        defer {
            isCommunicationsLoading = false
            hasLoadedCommunicationsInbox = true
        }

        guard let authToken = resolvedAuthToken() else {
            communicationThreads = []
            communicationEvents = []
            communicationCallSessions = []
            communicationThreadsHasMore = false
            communicationEventsHasMore = false
            communicationCallSessionsHasMore = false
            isCommunicationContinuityLoading = false
            communicationContinuityItems = []
            communicationContinuityHasMore = false
            communicationContinuityStatusMessage = "Set Assistant Access Token to query conversation continuity."
            communicationsStatusMessage = "Set Assistant Access Token to query communications inbox."
            return
        }

        do {
            async let threadResponseTask: DaemonCommThreadListResponse = daemonClient.communications.commThreadList(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                limit: 120
            )
            async let eventResponseTask: DaemonCommEventTimelineResponse = daemonClient.communications.commEventTimeline(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                limit: 200
            )
            async let callResponseTask: DaemonCommCallSessionListResponse = daemonClient.communications.commCallSessionList(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                limit: 120
            )
            let (threadResponse, eventResponse, callResponse) = try await (
                threadResponseTask,
                eventResponseTask,
                callResponseTask
            )

            updateWorkspaceContext(from: threadResponse.workspaceID)
            updateWorkspaceContext(from: eventResponse.workspaceID)
            updateWorkspaceContext(from: callResponse.workspaceID)

            communicationThreads = threadResponse.items
                .map(mapCommunicationThreadRecord)
                .sorted { lhs, rhs in
                    if lhs.sortTimestamp == rhs.sortTimestamp {
                        return lhs.id > rhs.id
                    }
                    return lhs.sortTimestamp > rhs.sortTimestamp
                }
            communicationEvents = eventResponse.items
                .map(mapCommunicationEventRecord)
                .sorted { lhs, rhs in
                    if lhs.sortTimestamp == rhs.sortTimestamp {
                        return lhs.id > rhs.id
                    }
                    return lhs.sortTimestamp > rhs.sortTimestamp
                }
            communicationCallSessions = callResponse.items
                .map(mapCommunicationCallSessionRecord)
                .sorted { lhs, rhs in
                    if lhs.sortTimestamp == rhs.sortTimestamp {
                        return lhs.id > rhs.id
                    }
                    return lhs.sortTimestamp > rhs.sortTimestamp
                }

            communicationThreadsHasMore = threadResponse.hasMore
            communicationEventsHasMore = eventResponse.hasMore
            communicationCallSessionsHasMore = callResponse.hasMore

            await fetchCommunicationContinuity(authToken: authToken)

            let hasMoreSources = [
                threadResponse.hasMore,
                eventResponse.hasMore,
                callResponse.hasMore
            ].contains(true)
            let hasMoreSuffix = hasMoreSources ? " (showing latest page)." : "."

            communicationsStatusMessage =
                "Workspace \(workspaceID) • Threads \(communicationThreads.count) • Events \(communicationEvents.count) • Calls \(communicationCallSessions.count)\(hasMoreSuffix)"
            connectionStatus = .connected
        } catch {
            isCommunicationContinuityLoading = false
            communicationContinuityItems = []
            communicationContinuityHasMore = false
            communicationContinuityStatusMessage = daemonErrorMessage(
                error,
                fallbackContext: "Conversation continuity query failed",
                updateConnectionStatus: false
            )
            communicationsStatusMessage = daemonErrorMessage(
                error,
                fallbackContext: "Communications inbox query failed",
                updateConnectionStatus: false
            )
        }
    }

    private func fetchCommunicationContinuity(authToken: String) async {
        isCommunicationContinuityLoading = true
        defer {
            isCommunicationContinuityLoading = false
        }

        do {
            let response = try await daemonClient.chat.chatTurnHistory(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                limit: 180
            )
            updateWorkspaceContext(from: response.workspaceID)
            communicationContinuityItems = communicationsStore.mapCommunicationContinuityRecords(
                response.items,
                workspaceID: workspaceID,
                logicalCommunicationChannelID: { [unowned self] rawChannelID in
                    logicalCommunicationChannelID(rawChannelID: rawChannelID)
                },
                parseDaemonTimestamp: { [unowned self] rawTimestamp in
                    parseDaemonTimestamp(rawTimestamp)
                },
                formattedWorkflowTimestamp: { [unowned self] rawTimestamp in
                    formattedWorkflowTimestamp(rawTimestamp)
                },
                truncateText: { [unowned self] value, limit in
                    truncateText(value, limit: limit)
                }
            )
            communicationContinuityHasMore = response.hasMore

            if communicationContinuityItems.isEmpty {
                communicationContinuityStatusMessage = "No conversation continuity records yet."
            } else {
                let hasMoreSuffix = response.hasMore ? " (showing latest page)." : "."
                communicationContinuityStatusMessage =
                    "Conversation continuity loaded (\(communicationContinuityItems.count) turn(s))\(hasMoreSuffix)"
            }
        } catch {
            communicationContinuityItems = []
            communicationContinuityHasMore = false
            communicationContinuityStatusMessage = daemonErrorMessage(
                error,
                fallbackContext: "Conversation continuity query failed",
                updateConnectionStatus: false
            )
        }
    }

    private func hydrateChatTimelineFromHistoryIfEmpty() async {
        guard chatTimelineItems.isEmpty,
              !isChatStreaming,
              let authToken = resolvedAuthToken() else {
            return
        }

        do {
            let response = try await daemonClient.chat.chatTurnHistory(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                channelID: "app",
                limit: 240
            )
            updateWorkspaceContext(from: response.workspaceID)
            let hydratedItems = mapChatTimelineItemsFromHistory(response.items)
            guard !hydratedItems.isEmpty else {
                return
            }
            chatTimelineItems = hydratedItems
        } catch {
            // Keep chat usable even if continuity hydration fails.
        }
    }

    private func mapChatTimelineItemsFromHistory(
        _ records: [DaemonChatTurnHistoryRecord]
    ) -> [ChatTimelineItem] {
        guard !records.isEmpty else {
            return []
        }

        let appRecords = records.filter { record in
            logicalCommunicationChannelID(rawChannelID: record.channelID) == "app"
        }
        guard !appRecords.isEmpty else {
            return []
        }

        let groupedByTurnID = Dictionary(grouping: appRecords) { record in
            nonEmpty(record.turnID)
                ?? nonEmpty(record.recordID)
                ?? UUID().uuidString.lowercased()
        }
        let orderedTurns = groupedByTurnID.values.sorted { lhs, rhs in
            let lhsSort = lhs.compactMap { parseDaemonTimestamp($0.createdAt) }.min() ?? .distantPast
            let rhsSort = rhs.compactMap { parseDaemonTimestamp($0.createdAt) }.min() ?? .distantPast
            if lhsSort == rhsSort {
                let lhsTurnID = nonEmpty(lhs.first?.turnID) ?? ""
                let rhsTurnID = nonEmpty(rhs.first?.turnID) ?? ""
                return lhsTurnID.localizedCaseInsensitiveCompare(rhsTurnID) == .orderedAscending
            }
            return lhsSort < rhsSort
        }

        let store = ChatTimelineEventStore()
        store.resetForNewTurn(existingTimeline: [])
        for turnRecords in orderedTurns {
            let orderedRecords = turnRecords.sorted { lhs, rhs in
                if lhs.itemIndex == rhs.itemIndex {
                    let lhsCreatedAt = parseDaemonTimestamp(lhs.createdAt) ?? .distantPast
                    let rhsCreatedAt = parseDaemonTimestamp(rhs.createdAt) ?? .distantPast
                    if lhsCreatedAt == rhsCreatedAt {
                        return lhs.recordID.localizedCaseInsensitiveCompare(rhs.recordID) == .orderedAscending
                    }
                    return lhsCreatedAt < rhsCreatedAt
                }
                return lhs.itemIndex < rhs.itemIndex
            }
            let items = orderedRecords.map(\.item)
            guard !items.isEmpty else {
                continue
            }
            let correlationID = nonEmpty(orderedRecords.last?.correlationID)
                ?? nonEmpty(orderedRecords.first?.correlationID)
                ?? UUID().uuidString.lowercased()
            let taskCorrelation = orderedRecords.last?.taskRunReference
                ?? orderedRecords.first?.taskRunReference
                ?? DaemonChatTurnTaskRunCorrelation()
            store.reconcileChatTurnTimelineItems(
                items: items,
                correlationID: correlationID,
                taskCorrelation: taskCorrelation,
                activeCorrelationID: nil
            )
        }

        return store.timelineItems
    }

    private func fetchCommunicationAttempts(threadID: String?) async {
        let normalizedThreadID = communicationsStore.setCommunicationAttemptContextThreadID(threadID)
        isCommunicationAttemptsLoading = true
        defer {
            isCommunicationAttemptsLoading = false
        }

        guard let authToken = resolvedAuthToken() else {
            communicationDeliveryAttempts = []
            communicationDeliveryAttemptsHasMore = false
            communicationAttemptsStatusMessage = "Set Assistant Access Token to query delivery attempts."
            return
        }

        guard let normalizedThreadID else {
            communicationDeliveryAttempts = []
            communicationDeliveryAttemptsHasMore = false
            communicationAttemptsStatusMessage = "Select a conversation to load delivery attempts."
            return
        }

        do {
            let response = try await daemonClient.communications.commAttempts(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                threadID: normalizedThreadID,
                limit: 120
            )
            updateWorkspaceContext(from: response.workspaceID)
            communicationDeliveryAttempts = response.attempts
                .map(mapCommunicationDeliveryAttemptRecord)
                .sorted { lhs, rhs in
                    if lhs.sortTimestamp == rhs.sortTimestamp {
                        return lhs.id > rhs.id
                    }
                    return lhs.sortTimestamp > rhs.sortTimestamp
                }
            communicationDeliveryAttemptsHasMore = response.hasMore

            var contextComponents: [String] = []
            if let operationID = nonEmpty(response.operationID) {
                contextComponents.append("op \(operationID)")
            }
            if let taskID = nonEmpty(response.taskID) {
                contextComponents.append("task \(taskID)")
            }
            if let runID = nonEmpty(response.runID) {
                contextComponents.append("run \(runID)")
            }
            if let stepID = nonEmpty(response.stepID) {
                contextComponents.append("step \(stepID)")
            }
            let contextLabel = contextComponents.isEmpty
                ? "thread \(normalizedThreadID)"
                : contextComponents.joined(separator: " • ")
            let hasMoreSuffix = response.hasMore ? " (showing latest page)." : "."
            communicationAttemptsStatusMessage = "Delivery attempts for \(contextLabel) • \(communicationDeliveryAttempts.count) record(s)\(hasMoreSuffix)"
            connectionStatus = .connected
        } catch {
            communicationDeliveryAttempts = []
            communicationDeliveryAttemptsHasMore = false
            communicationAttemptsStatusMessage = daemonErrorMessage(
                error,
                fallbackContext: "Delivery attempt history query failed",
                updateConnectionStatus: false
            )
        }
    }

    private func performCommunicationSend(
        sourceChannel: String,
        destination: String?,
        message: String,
        threadID: String?,
        connectorID: String?
    ) async {
        let normalizedSourceChannel = logicalCommunicationChannelID(rawChannelID: sourceChannel)
        let normalizedDestination = nonEmpty(destination)
        let normalizedThreadID = nonEmpty(threadID)
        let normalizedConnectorID = nonEmpty(connectorID)
        let trimmedMessage = message.trimmingCharacters(in: .whitespacesAndNewlines)

        guard normalizedSourceChannel != "unknown" else {
            communicationSendStatusMessage = "Select a valid source channel before sending."
            return
        }
        guard !trimmedMessage.isEmpty else {
            communicationSendStatusMessage = "Message body is required."
            return
        }
        guard normalizedThreadID != nil || normalizedDestination != nil else {
            communicationSendStatusMessage = "Destination is required unless a thread context is selected."
            return
        }
        guard let authToken = resolvedAuthToken() else {
            communicationSendStatusMessage = "Set Assistant Access Token before sending communications."
            return
        }

        let operationID = "ui-\(UUID().uuidString.lowercased())"
        isCommunicationSendInFlight = true
        communicationSendStatusMessage = "Sending \(normalizedSourceChannel) communication…"
        defer {
            isCommunicationSendInFlight = false
        }

        do {
            let response = try await daemonClient.communications.commSend(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                operationID: operationID,
                sourceChannel: normalizedSourceChannel,
                threadID: normalizedThreadID,
                connectorID: normalizedConnectorID,
                destination: normalizedDestination,
                message: trimmedMessage
            )
            updateWorkspaceContext(from: response.workspaceID)

            let resolvedThreadID = nonEmpty(response.threadID) ?? normalizedThreadID
            let resolvedSourceChannel = nonEmpty(response.resolvedSourceChannel) ?? normalizedSourceChannel
            let resolvedConnectorID = nonEmpty(response.resolvedConnectorID) ?? normalizedConnectorID
            let resolvedDestination = nonEmpty(response.resolvedDestination) ?? normalizedDestination
            let resolvedOperationID = nonEmpty(response.operationID) ?? operationID
            let responseError = nonEmpty(response.error)

            latestCommunicationSendReceipt = CommunicationSendReceiptItem(
                operationID: resolvedOperationID,
                threadID: resolvedThreadID,
                sourceChannel: resolvedSourceChannel,
                connectorID: resolvedConnectorID,
                destination: resolvedDestination,
                success: response.success && responseError == nil,
                sentAt: Date.now
            )

            if response.success, responseError == nil {
                let destinationLabel = resolvedDestination ?? "thread context"
                communicationSendStatusMessage = "Sent via \(resolvedSourceChannel) to \(destinationLabel)."
                communicationsStatusMessage = "Communication send accepted. Refreshing inbox…"
                connectionStatus = .connected
                appendNotification(
                    source: "communications",
                    action: "comm_send",
                    message: "Sent via \(resolvedSourceChannel) to \(destinationLabel).",
                    level: .success
                )
                markHomeFirstSessionStepComplete(
                    .sendCommunication,
                    source: "comm_send",
                    completedAt: latestCommunicationSendReceipt?.sentAt ?? .now
                )
                await fetchCommunicationsInbox()
                if let resolvedThreadID {
                    await fetchCommunicationAttempts(threadID: resolvedThreadID)
                }
            } else {
                let failureMessage = responseError ?? "Daemon reported communication delivery failure."
                communicationSendStatusMessage = failureMessage
                communicationsStatusMessage = failureMessage
                appendNotification(
                    source: "communications",
                    action: "comm_send",
                    message: failureMessage,
                    level: .error
                )
            }
        } catch {
            let failureMessage = daemonErrorMessage(
                error,
                fallbackContext: "Communication send failed",
                updateConnectionStatus: false
            )
            communicationSendStatusMessage = failureMessage
            communicationsStatusMessage = failureMessage
            appendNotification(
                source: "communications",
                action: "comm_send",
                message: failureMessage,
                level: .error
            )
        }
    }

    private func fetchChannelCards() async {
        isChannelsLoading = true
        isChannelConnectorMappingsLoading = true
        clearPanelProblemSignal(for: .channels)
        defer {
            isChannelsLoading = false
            isChannelConnectorMappingsLoading = false
            hasLoadedChannelStatus = true
            updateOnboardingCompletionState()
        }

        guard let authToken = resolvedAuthToken() else {
            channelsStatusMessage = "Set Assistant Access Token to load channel status."
            let tokenMessage = "Set Assistant Access Token before loading channel connector mappings."
            for channelID in discoveredLogicalChannelIDsForMappings()
            where channelConnectorMappingActionStatusByChannelID[channelID] == nil {
                channelConnectorMappingActionStatusByChannelID[channelID] = tokenMessage
            }
            return
        }

        let persistedExpandedIDs = expandedChannelCardIDsContinuity()

        do {
            var mappingsByChannelID: [String: [ChannelConnectorMappingItem]] = channelConnectorMappingsByChannelID
            var mappingQueryWarning: String?
            do {
                let mappingResponse = try await daemonClient.channels.channelConnectorMappingsList(
                    baseURL: daemonBaseURL,
                    authToken: authToken,
                    workspaceID: workspaceID
                )
                updateWorkspaceContext(from: mappingResponse.workspaceID)
                channelConnectorMappingFallbackPolicy = nonEmpty(mappingResponse.fallbackPolicy) ?? "priority_order"
                mappingsByChannelID = mapChannelConnectorMappingsByChannel(mappingResponse.bindings)
            } catch {
                mappingQueryWarning = daemonErrorMessage(
                    error,
                    fallbackContext: "Channel connector mapping query failed",
                    updateConnectionStatus: false,
                    panelContext: .channels
                )
            }

            let statusResponse = try await daemonClient.channels.channelStatus(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID
            )

            var diagnosticsByID: [String: [DiagnosticsActionItem]] = [:]
            do {
                let diagnosticsResponse = try await daemonClient.channels.channelDiagnostics(
                    baseURL: daemonBaseURL,
                    authToken: authToken,
                    workspaceID: workspaceID
                )
                updateWorkspaceContext(from: diagnosticsResponse.workspaceID)
                diagnosticsByID = Dictionary(
                    uniqueKeysWithValues: diagnosticsResponse.diagnostics.map { summary in
                        (
                            summary.channelID,
                            summary.remediationActions.map(mapDiagnosticsAction)
                        )
                    }
                )
            } catch {
                channelsStatusMessage = daemonErrorMessage(
                    error,
                    fallbackContext: "Channel diagnostics query failed",
                    updateConnectionStatus: false,
                    panelContext: .channels
                )
            }

            var policiesByChannelID: [String: [ChannelDeliveryPolicyItem]] = channelDeliveryPoliciesByChannelID
            var policyQueryWarning: String?
            do {
                let policyResponse = try await daemonClient.communications.commPolicyList(
                    baseURL: daemonBaseURL,
                    authToken: authToken,
                    workspaceID: workspaceID,
                    sourceChannel: nil
                )
                updateWorkspaceContext(from: policyResponse.workspaceID)
                policiesByChannelID = mapChannelDeliveryPoliciesBySource(policyResponse.policies)
            } catch {
                policyQueryWarning = daemonErrorMessage(
                    error,
                    fallbackContext: "Channel delivery policy query failed",
                    updateConnectionStatus: false,
                    panelContext: .channels
                )
            }

            updateWorkspaceContext(from: statusResponse.workspaceID)
            channelCards = statusResponse.channels.map { card in
                let statusActions = card.remediationActions?.map(mapDiagnosticsAction) ?? []
                let resolvedActions = diagnosticsByID[card.channelID] ?? statusActions
                let mapped = mapChannelCard(
                    card,
                    diagnosticsActions: resolvedActions
                )
                return ChannelCardItem(
                    id: mapped.id,
                    name: mapped.name,
                    status: mapped.status,
                    summary: mapped.summary,
                    details: mapped.details,
                    editableConfiguration: mapped.editableConfiguration,
                    editableConfigurationKinds: mapped.editableConfigurationKinds,
                    readOnlyConfiguration: mapped.readOnlyConfiguration,
                    actions: mapped.actions,
                    unavailableActionReason: mapped.unavailableActionReason,
                    isExpanded: persistedExpandedIDs.contains(mapped.id)
                )
            }
            persistExpandedChannelCardContinuity()
            connectionConfigStore.synchronizeChannelConnectorMappingDrafts(
                with: channelCards,
                mappingsByChannelID: mappingsByChannelID,
                normalizeChannelID: normalizedChannelConnectorMappingChannelID(_:),
                inferredMappingsByLogicalChannelID: inferredChannelConnectorMappingsByLogicalChannelID(from:),
                mergeMappings: mergedChannelConnectorMappings(observed:inferred:channelID:)
            )
            channelDeliveryPoliciesByChannelID = policiesByChannelID
            connectionConfigStore.synchronizeChannelConfigurationDrafts(with: channelCards)
            synchronizeChannelDeliveryPolicyDrafts(with: channelCards, policiesByChannelID: policiesByChannelID)
            let statusPrefix = "Workspace \(statusResponse.workspaceID) • Updated \(Date.now.formatted(date: .omitted, time: .standard))"
            var warningParts: [String] = []
            if let policyQueryWarning {
                warningParts.append("Delivery policy unavailable.")
                for channelID in logicalChannelCards.map(\.channelID)
                where channelDeliveryPolicyActionStatusByID[channelID] == nil {
                    channelDeliveryPolicyActionStatusByID[channelID] = policyQueryWarning
                }
            }
            if let mappingQueryWarning {
                warningParts.append("Connector mappings unavailable.")
                for channelID in logicalChannelCards.map(\.channelID)
                where channelConnectorMappingActionStatusByChannelID[channelID] == nil {
                    channelConnectorMappingActionStatusByChannelID[channelID] = mappingQueryWarning
                }
            }
            if warningParts.isEmpty {
                channelsStatusMessage = statusPrefix
            } else {
                channelsStatusMessage = "\(statusPrefix) • \(warningParts.joined(separator: " "))"
            }
            connectionStatus = .connected
        } catch {
            channelsStatusMessage = daemonErrorMessage(
                error,
                fallbackContext: "Channel status query failed",
                updateConnectionStatus: false,
                panelContext: .channels
            )
        }
    }

    private func fetchConnectorCards() async {
        isConnectorsLoading = true
        clearPanelProblemSignal(for: .connectors)
        defer {
            isConnectorsLoading = false
            hasLoadedConnectorStatus = true
            updateOnboardingCompletionState()
        }

        guard let authToken = resolvedAuthToken() else {
            connectorsStatusMessage = "Set Assistant Access Token to load connector status."
            return
        }

        let persistedExpandedIDs = expandedConnectorCardIDsContinuity()

        do {
            let statusResponse = try await daemonClient.connectors.connectorStatus(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID
            )

            var diagnosticsByID: [String: [DiagnosticsActionItem]] = [:]
            do {
                let diagnosticsResponse = try await daemonClient.connectors.connectorDiagnostics(
                    baseURL: daemonBaseURL,
                    authToken: authToken,
                    workspaceID: workspaceID
                )
                updateWorkspaceContext(from: diagnosticsResponse.workspaceID)
                diagnosticsByID = Dictionary(
                    uniqueKeysWithValues: diagnosticsResponse.diagnostics.map { summary in
                        (
                            summary.connectorID,
                            summary.remediationActions.map(mapDiagnosticsAction)
                        )
                    }
                )
            } catch {
                connectorsStatusMessage = daemonErrorMessage(
                    error,
                    fallbackContext: "Connector diagnostics query failed",
                    updateConnectionStatus: false,
                    panelContext: .connectors
                )
            }

            updateWorkspaceContext(from: statusResponse.workspaceID)
            connectorCards = statusResponse.connectors.map { card in
                let permissionState = connectorPermissionState(for: card)
                connectorPermissionStatesByID[card.connectorID] = permissionState
                let statusActions = card.remediationActions?.map(mapDiagnosticsAction) ?? []
                let resolvedActions = diagnosticsByID[card.connectorID] ?? statusActions
                return mapConnectorCard(
                    card,
                    diagnosticsActions: resolvedActions,
                    permissionState: permissionState,
                    isExpanded: persistedExpandedIDs.contains(card.connectorID)
                )
            }
            persistExpandedConnectorCardContinuity()
            connectionConfigStore.synchronizeConnectorConfigurationDrafts(with: connectorCards)
            connectorsStatusMessage = "Workspace \(statusResponse.workspaceID) • Updated \(Date.now.formatted(date: .omitted, time: .standard))"
            connectionStatus = .connected
        } catch {
            connectorsStatusMessage = daemonErrorMessage(
                error,
                fallbackContext: "Connector status query failed",
                updateConnectionStatus: false,
                panelContext: .connectors
            )
        }
    }

    private func performIdentityWorkspaceSelection(workspaceID requestedWorkspaceID: String) async {
        guard let workspaceCandidate = Self.canonicalWorkspaceID(requestedWorkspaceID, fallbackToDefault: false) else {
            identityStatusMessage = "Workspace selection requires a valid workspace id."
            principalStatusMessage = identityStatusMessage
            return
        }

        guard let authToken = resolvedAuthToken() else {
            identityStatusMessage = "Set Assistant Access Token to switch workspace context."
            principalStatusMessage = identityStatusMessage
            return
        }

        isIdentityDirectoryLoading = true
        isPrincipalOptionsLoading = true
        defer {
            isIdentityDirectoryLoading = false
            isPrincipalOptionsLoading = false
        }

        do {
            let response = try await daemonClient.identity.identitySelectWorkspace(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceCandidate,
                principalActorID: nil,
                source: "ui.configuration.identity_hub"
            )
            applyIdentityActiveContext(
                response.activeContext,
                fallbackWorkspaceID: workspaceCandidate,
                fallbackPrincipalActorID: nonEmpty(selectedPrincipal),
                contextUpdateIntent: .explicitSelection
            )
            await fetchPrincipalOptions(authToken: authToken)

            if selectedSection == .configuration {
                async let lifecycleRefresh: Void = refreshDaemonLifecycleStatus(
                    showLoadingState: !hasLoadedDaemonStatus
                )
                async let delegationRefresh: Void = fetchDelegationRules(authToken: authToken)
                async let personaRefresh: Void = fetchChatPersonaPolicy(authToken: authToken)
                async let identityInventoryRefresh: Void = fetchIdentityDeviceAndSessionPanels(authToken: authToken)
                async let governanceRefresh: Void = fetchCapabilityGrantAndTrustReceiptPanels(authToken: authToken)
                async let providerRefresh: Void = fetchProviderAndModelStatus(runChecks: false)
                _ = await (
                    lifecycleRefresh,
                    delegationRefresh,
                    personaRefresh,
                    identityInventoryRefresh,
                    governanceRefresh,
                    providerRefresh
                )
            } else {
                await refreshDataForCurrentSection(trigger: .refresh)
            }

            let workspaceLabel = identityWorkspaceDisplayName(for: workspaceID)
            identityStatusMessage = "Workspace context switched to \(workspaceLabel) (\(workspaceID))."
            principalStatusMessage = identityStatusMessage
            connectionStatus = .connected
        } catch {
            identityStatusMessage = daemonErrorMessage(
                error,
                fallbackContext: "Workspace context switch failed",
                updateConnectionStatus: false
            )
            principalStatusMessage = identityStatusMessage
        }
    }

    private func fetchPrincipalOptions() async {
        await fetchPrincipalOptions(authToken: nil)
    }

    private func fetchPrincipalOptions(authToken providedAuthToken: String?) async {
        isPrincipalOptionsLoading = true
        isIdentityDirectoryLoading = true
        defer {
            isPrincipalOptionsLoading = false
            isIdentityDirectoryLoading = false
        }

        guard let authToken = providedAuthToken ?? resolvedAuthToken() else {
            principalOptions = ensurePrincipalFallbackOptions()
            principalStatusMessage = "Set Assistant Access Token to query principal context."
            identityStatusMessage = principalStatusMessage
            identityWorkspaceItems = []
            identityPrincipalItems = []
            identityActiveContext = nil
            return
        }

        do {
            let requestedWorkspaceHint = nonEmpty(workspaceID)
            let activeContextResponse = try await daemonClient.identity.identityActiveContext(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: requestedWorkspaceHint
            )
            let workspaceResponse = try await daemonClient.identity.identityWorkspaces(
                baseURL: daemonBaseURL,
                authToken: authToken,
                includeInactive: true
            )
            let requestedWorkspaceID = nonEmpty(activeContextResponse.activeContext?.workspaceID)
                ?? nonEmpty(workspaceResponse.activeContext?.workspaceID)
                ?? workspaceID
            let principalsResponse = try await daemonClient.identity.identityPrincipals(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: requestedWorkspaceID,
                includeInactive: true
            )

            let resolvedWorkspaceID = nonEmpty(principalsResponse.workspaceID) ?? requestedWorkspaceID
            applyIdentityActiveContext(
                principalsResponse.activeContext
                    ?? activeContextResponse.activeContext
                    ?? workspaceResponse.activeContext,
                fallbackWorkspaceID: resolvedWorkspaceID,
                fallbackPrincipalActorID: nonEmpty(selectedPrincipal),
                contextUpdateIntent: .identityDirectorySync
            )
            mapIdentityDirectoryRecords(
                workspaceResponse: workspaceResponse,
                principalsResponse: principalsResponse
            )

            var options = Set<String>(["default"])
            for principal in identityPrincipalItems {
                options.insert(principal.id)
            }
            principalOptions = options.sorted()
            let activePrincipalCandidate = nonEmpty(identityActiveContext?.principalActorID)
                ?? nonEmpty(selectedPrincipal)
                ?? "default"
            selectedPrincipal = principalOptions.contains(activePrincipalCandidate)
                ? activePrincipalCandidate
                : (principalOptions.first ?? "default")
            let resolvedPersonaPrincipal = nonEmpty(chatPersonaScopePrincipalActorID) ?? "default"
            if !principalOptions.contains(resolvedPersonaPrincipal) {
                chatPersonaScopePrincipalActorID = selectedPrincipal
            }
            let activePrincipal = nonEmpty(selectedPrincipal) ?? "default"
            if nonEmpty(modelRouteSimulationPrincipalActorID) == nil {
                modelRouteSimulationPrincipalActorID = activePrincipal
            }
            let workspaceLabel = identityWorkspaceDisplayName(for: workspaceID)
            let principalCount = identityPrincipalItems.count
            principalStatusMessage = principalCount == 0
                ? "No principal records found for workspace \(workspaceLabel). Active principal: \(activePrincipal)."
                : "Workspace \(workspaceLabel) • \(principalCount) principal(s) loaded. Active principal: \(activePrincipal)."
            identityStatusMessage = principalStatusMessage
            connectionStatus = .connected
        } catch {
            principalOptions = ensurePrincipalFallbackOptions()
            principalStatusMessage = daemonErrorMessage(
                error,
                fallbackContext: "Principal context query failed",
                updateConnectionStatus: false
            )
            identityStatusMessage = principalStatusMessage
            identityWorkspaceItems = []
            identityPrincipalItems = []
            identityActiveContext = nil
        }
    }

    private func fetchIdentityDeviceAndSessionPanels() async {
        await fetchIdentityDeviceAndSessionPanels(authToken: nil)
    }

    private func fetchIdentityDeviceAndSessionPanels(authToken providedAuthToken: String?) async {
        async let deviceRefresh: Void = fetchIdentityDeviceInventory(authToken: providedAuthToken)
        async let sessionRefresh: Void = fetchIdentitySessionInventory(authToken: providedAuthToken)
        _ = await (deviceRefresh, sessionRefresh)
    }

    private func fetchIdentityDeviceInventory() async {
        await fetchIdentityDeviceInventory(authToken: nil)
    }

    private func fetchIdentityDeviceInventory(authToken providedAuthToken: String?) async {
        if isIdentityDeviceInventoryRequestInFlight {
            return
        }
        isIdentityDeviceInventoryRequestInFlight = true
        isIdentityDeviceInventoryLoading = true
        defer {
            isIdentityDeviceInventoryRequestInFlight = false
            isIdentityDeviceInventoryLoading = false
        }

        guard let authToken = providedAuthToken ?? resolvedAuthToken() else {
            identityDeviceItems = []
            identityDeviceInventoryHasMore = false
            identityDeviceInventoryStatusMessage = IdentityInventoryProjection.missingTokenDeviceMessage
            return
        }

        let userID = nonEmpty(identityDeviceUserIDFilter)
        let deviceType = nonEmpty(identityDeviceTypeFilter)
        let platform = nonEmpty(identityDevicePlatformFilter)
        let requestLimit = IdentityInventoryProjection.clampedQueryLimit(identityDeviceLimit)
        if requestLimit != identityDeviceLimit {
            identityDeviceLimit = requestLimit
        }

        do {
            let response = try await daemonClient.identity.identityDevicesList(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                userID: userID,
                deviceType: deviceType,
                platform: platform,
                cursorCreatedAt: nil,
                cursorID: nil,
                limit: requestLimit
            )
            updateWorkspaceContext(from: response.workspaceID)
            let mappedItems = IdentityInventoryProjection.mapAndSortDeviceItems(
                response.items,
                fallbackWorkspaceID: workspaceID,
                canonicalWorkspaceID: { raw in
                    Self.canonicalWorkspaceID(raw)
                },
                parseTimestamp: { [self] value in
                    parseDaemonTimestamp(value)
                },
                formatTimestamp: { [self] value in
                    formattedWorkflowTimestamp(value)
                }
            )
            identityDeviceItems = mappedItems
            identityDeviceInventoryHasMore = response.hasMore
            identityDeviceInventoryStatusMessage = IdentityInventoryProjection.deviceSummaryMessage(
                itemCount: mappedItems.count,
                hasMore: response.hasMore,
                userID: userID,
                deviceType: deviceType,
                platform: platform
            )
            connectionStatus = .connected
        } catch {
            identityDeviceItems = []
            identityDeviceInventoryHasMore = false
            identityDeviceInventoryStatusMessage = daemonErrorMessage(
                error,
                fallbackContext: "Identity device query failed",
                updateConnectionStatus: false
            )
        }
    }

    private func fetchIdentitySessionInventory() async {
        await fetchIdentitySessionInventory(authToken: nil)
    }

    private func fetchIdentitySessionInventory(authToken providedAuthToken: String?) async {
        if isIdentitySessionInventoryRequestInFlight {
            return
        }
        isIdentitySessionInventoryRequestInFlight = true
        isIdentitySessionInventoryLoading = true
        defer {
            isIdentitySessionInventoryRequestInFlight = false
            isIdentitySessionInventoryLoading = false
        }

        guard let authToken = providedAuthToken ?? resolvedAuthToken() else {
            identitySessionItems = []
            identitySessionInventoryHasMore = false
            identitySessionInventoryStatusMessage = IdentityInventoryProjection.missingTokenSessionMessage
            identitySessionActionStatusByID = [:]
            identitySessionRevokeInFlightIDs = []
            return
        }

        let deviceID = nonEmpty(identitySessionDeviceIDFilter)
        let userID = nonEmpty(identitySessionUserIDFilter)
        let sessionHealth = IdentityInventoryProjection.normalizedSessionHealthFilter(identitySessionHealthFilter)
        let requestLimit = IdentityInventoryProjection.clampedQueryLimit(identitySessionLimit)
        if requestLimit != identitySessionLimit {
            identitySessionLimit = requestLimit
        }

        do {
            let response = try await daemonClient.identity.identitySessionsList(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                deviceID: deviceID,
                userID: userID,
                sessionHealth: sessionHealth,
                cursorStartedAt: nil,
                cursorID: nil,
                limit: requestLimit
            )
            updateWorkspaceContext(from: response.workspaceID)
            let mappedItems = IdentityInventoryProjection.mapAndSortSessionItems(
                response.items,
                fallbackWorkspaceID: workspaceID,
                canonicalWorkspaceID: { raw in
                    Self.canonicalWorkspaceID(raw)
                },
                parseTimestamp: { [self] value in
                    parseDaemonTimestamp(value)
                },
                formatTimestamp: { [self] value in
                    formattedWorkflowTimestamp(value)
                }
            )
            identitySessionItems = mappedItems
            identitySessionInventoryHasMore = response.hasMore
            let activeSessionIDs = Set(mappedItems.map(\.id))
            let prunedSessionState = IdentityInventoryProjection.pruneSessionActionState(
                activeSessionIDs: activeSessionIDs,
                actionStatusByID: identitySessionActionStatusByID,
                revokeInFlightIDs: identitySessionRevokeInFlightIDs
            )
            identitySessionActionStatusByID = prunedSessionState.actionStatusByID
            identitySessionRevokeInFlightIDs = prunedSessionState.revokeInFlightIDs
            identitySessionInventoryStatusMessage = IdentityInventoryProjection.sessionSummaryMessage(
                itemCount: mappedItems.count,
                hasMore: response.hasMore,
                deviceID: deviceID,
                userID: userID,
                sessionHealth: sessionHealth
            )
            connectionStatus = .connected
        } catch {
            identitySessionItems = []
            identitySessionInventoryHasMore = false
            identitySessionInventoryStatusMessage = daemonErrorMessage(
                error,
                fallbackContext: "Identity session query failed",
                updateConnectionStatus: false
            )
        }
    }

    private func performIdentitySessionRevoke(sessionID: String) async {
        let normalizedSessionID = sessionID.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !normalizedSessionID.isEmpty else {
            return
        }
        guard let authToken = resolvedAuthToken() else {
            let message = "Set Assistant Access Token before revoking identity sessions."
            identitySessionActionStatusByID[normalizedSessionID] = message
            identitySessionInventoryStatusMessage = message
            return
        }
        guard !identitySessionRevokeInFlightIDs.contains(normalizedSessionID) else {
            return
        }

        identitySessionRevokeInFlightIDs.insert(normalizedSessionID)
        identitySessionActionStatusByID[normalizedSessionID] = "Revoking session…"
        defer {
            identitySessionRevokeInFlightIDs.remove(normalizedSessionID)
        }

        do {
            let response = try await daemonClient.identity.identitySessionRevoke(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                sessionID: normalizedSessionID
            )
            updateWorkspaceContext(from: response.workspaceID)
            let statusDetail = nonEmpty(response.sessionHealth)?.lowercased() ?? "unknown"
            let message = response.idempotent
                ? "Session \(normalizedSessionID) was already revoked (\(statusDetail))."
                : "Session \(normalizedSessionID) revoked (\(statusDetail))."
            identitySessionActionStatusByID[normalizedSessionID] = message
            identitySessionInventoryStatusMessage = message
            connectionStatus = .connected
            await fetchIdentitySessionInventory(authToken: authToken)
        } catch {
            let message = daemonErrorMessage(
                error,
                fallbackContext: "Identity session revoke failed",
                updateConnectionStatus: false
            )
            identitySessionActionStatusByID[normalizedSessionID] = message
            identitySessionInventoryStatusMessage = message
        }
    }

    private struct DelegationGrantPayload {
        let fromActorID: String
        let toActorID: String
        let scopeType: String
        let scopeKey: String?
        let expiresAt: String?
    }

    private struct DelegationGrantValidationError: Error {
        let message: String
    }

    private func fetchDelegationRules() async {
        await fetchDelegationRules(authToken: nil)
    }

    private func fetchDelegationRules(authToken providedAuthToken: String?) async {
        isDelegationLoading = true
        defer {
            isDelegationLoading = false
        }

        guard let authToken = providedAuthToken ?? resolvedAuthToken() else {
            delegationRules = []
            delegationStatusMessage = "Set Assistant Access Token to query delegation rules."
            return
        }

        do {
            let response = try await daemonClient.identity.delegationList(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID
            )
            updateWorkspaceContext(from: response.workspaceID)
            delegationRules = response.rules
                .sorted { lhs, rhs in
                    let lhsTimestamp = parseDaemonTimestamp(lhs.createdAt) ?? .distantPast
                    let rhsTimestamp = parseDaemonTimestamp(rhs.createdAt) ?? .distantPast
                    if lhsTimestamp == rhsTimestamp {
                        return lhs.id > rhs.id
                    }
                    return lhsTimestamp > rhsTimestamp
                }
                .map(mapDelegationRuleRecord)

            let activeRuleIDs = Set(delegationRules.map(\.id))
            delegationActionStatusByRuleID = delegationActionStatusByRuleID.filter { activeRuleIDs.contains($0.key) }
            delegationRevokeInFlightRuleIDs = Set(delegationRevokeInFlightRuleIDs.filter { activeRuleIDs.contains($0) })
            delegationStatusMessage = delegationRules.isEmpty
                ? "No delegation rules returned for workspace \(response.workspaceID)."
                : "Workspace \(response.workspaceID) • \(delegationRules.count) delegation rule(s) loaded."
            connectionStatus = .connected
        } catch {
            delegationRules = []
            delegationStatusMessage = daemonErrorMessage(
                error,
                fallbackContext: "Delegation list query failed",
                updateConnectionStatus: false
            )
        }
    }

    private func fetchChatPersonaPolicy() async {
        await fetchChatPersonaPolicy(authToken: nil)
    }

    private func fetchChatPersonaPolicy(authToken providedAuthToken: String?) async {
        if isChatPersonaPolicyRequestInFlight {
            return
        }
        isChatPersonaPolicyRequestInFlight = true
        isChatPersonaPolicyLoading = true
        defer {
            isChatPersonaPolicyRequestInFlight = false
            isChatPersonaPolicyLoading = false
        }

        guard let authToken = providedAuthToken ?? resolvedAuthToken() else {
            chatPersonaHasLoadedPolicy = false
            chatPersonaPolicyItem = nil
            chatPersonaStylePromptDraft = ""
            chatPersonaGuardrailsDraft = ""
            chatPersonaLoadedStylePrompt = ""
            chatPersonaLoadedGuardrails = []
            chatPersonaPolicyStatusMessage = "Set Assistant Access Token to load chat persona policy."
            return
        }

        let principalActorID = resolvedChatPersonaPrincipalActorID()
        let channelID = resolvedChatPersonaChannelID()
        chatPersonaPolicyStatusMessage = "Loading chat persona policy for \(chatPersonaScopeSummary(principalActorID: principalActorID, channelID: channelID))…"

        do {
            let response = try await daemonClient.chat.chatPersonaPolicyGet(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                principalActorID: principalActorID,
                channelID: channelID
            )
            updateWorkspaceContext(from: response.workspaceID)
            let policy = mapChatPersonaPolicy(response)
            applyChatPersonaPolicy(
                policy,
                statusMessage: "Loaded chat persona policy (\(policy.source)) for \(chatPersonaScopeSummary(principalActorID: policy.principalActorID, channelID: policy.channelID))."
            )
            connectionStatus = .connected
        } catch {
            chatPersonaPolicyStatusMessage = userFacingChatPersonaPolicyErrorMessage(
                error,
                operation: "loading"
            )
        }
    }

    private func performChatPersonaPolicySave(_ input: ChatPersonaPolicyMutationInput) async {
        guard !isChatPersonaPolicySaveRequestInFlight else {
            return
        }

        let stylePrompt = input.stylePrompt.trimmingCharacters(in: .whitespacesAndNewlines)
        if stylePrompt.isEmpty {
            chatPersonaPolicyStatusMessage = "Style prompt is required before saving persona policy."
            return
        }

        guard let authToken = resolvedAuthToken() else {
            chatPersonaPolicyStatusMessage = "Set Assistant Access Token before saving persona policy."
            return
        }

        isChatPersonaPolicySaveRequestInFlight = true
        isChatPersonaPolicySaveInFlight = true
        defer {
            isChatPersonaPolicySaveRequestInFlight = false
            isChatPersonaPolicySaveInFlight = false
        }

        let guardrails = normalizedChatPersonaGuardrails(input.guardrailsText)
        let principalActorID = normalizedIdentityID(input.principalActorID)
        let channelID = normalizedChatPersonaChannelID(input.channelID)
        chatPersonaPolicyStatusMessage = "Saving chat persona policy for \(chatPersonaScopeSummary(principalActorID: principalActorID, channelID: channelID))…"

        do {
            let response = try await daemonClient.chat.chatPersonaPolicySet(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                principalActorID: principalActorID,
                channelID: channelID,
                stylePrompt: stylePrompt,
                guardrails: guardrails
            )
            updateWorkspaceContext(from: response.workspaceID)
            let policy = mapChatPersonaPolicy(response)
            applyChatPersonaPolicy(
                policy,
                statusMessage: "Saved chat persona policy for \(chatPersonaScopeSummary(principalActorID: policy.principalActorID, channelID: policy.channelID))."
            )
            connectionStatus = .connected
        } catch {
            chatPersonaPolicyStatusMessage = userFacingChatPersonaPolicyErrorMessage(
                error,
                operation: "saving"
            )
        }
    }

    private func applyChatPersonaPolicy(
        _ policy: ChatPersonaPolicyItem,
        statusMessage: String
    ) {
        chatPersonaPolicyItem = policy
        chatPersonaStylePromptDraft = policy.stylePrompt
        chatPersonaGuardrailsDraft = policy.guardrails.joined(separator: "\n")
        chatPersonaLoadedStylePrompt = policy.stylePrompt
        chatPersonaLoadedGuardrails = policy.guardrails
        chatPersonaHasLoadedPolicy = true
        chatPersonaPolicyStatusMessage = statusMessage
    }

    private func mapChatPersonaPolicy(_ response: DaemonChatPersonaPolicyResponse) -> ChatPersonaPolicyItem {
        let normalizedPrincipal = normalizedIdentityID(response.principalActorID)
        let normalizedChannel = normalizedChatPersonaChannelID(response.channelID)
        let normalizedStylePrompt = response.stylePrompt.trimmingCharacters(in: .whitespacesAndNewlines)
        let normalizedGuardrails = normalizedChatPersonaGuardrails(response.guardrails.joined(separator: "\n"))
        let updatedAtRaw = nonEmpty(response.updatedAt)
        return ChatPersonaPolicyItem(
            workspaceID: response.workspaceID,
            principalActorID: normalizedPrincipal,
            channelID: normalizedChannel,
            stylePrompt: normalizedStylePrompt,
            guardrails: normalizedGuardrails,
            source: nonEmpty(response.source) ?? "default",
            updatedAtRaw: updatedAtRaw,
            updatedAtLabel: updatedAtRaw.map(formattedWorkflowTimestamp)
        )
    }

    private func resolvedChatPersonaPrincipalActorID() -> String? {
        switch chatPersonaScopeType {
        case .workspace, .channel:
            return nil
        case .principal, .principalChannel:
            return normalizedIdentityID(chatPersonaScopePrincipalActorID) ?? "default"
        }
    }

    private func resolvedChatPersonaChannelID() -> String? {
        switch chatPersonaScopeType {
        case .workspace, .principal:
            return nil
        case .channel, .principalChannel:
            return normalizedChatPersonaChannelID(chatPersonaScopeChannelID) ?? "app"
        }
    }

    private func normalizedChatPersonaChannelID(_ raw: String?) -> String? {
        guard let normalized = nonEmpty(raw)?.lowercased() else {
            return nil
        }
        guard Self.chatPersonaChannelOptions.contains(normalized) else {
            return nil
        }
        return normalized
    }

    private func normalizedChatPersonaGuardrails(_ raw: String) -> [String] {
        raw
            .split(whereSeparator: \.isNewline)
            .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
            .filter { !$0.isEmpty }
            .reduce(into: [String]()) { acc, value in
                if !acc.contains(where: { $0.caseInsensitiveCompare(value) == .orderedSame }) {
                    acc.append(value)
                }
            }
    }

    private func chatPersonaScopeSummary(
        principalActorID: String?,
        channelID: String?
    ) -> String {
        let principalLabel = principalActorID.map { principalOptionDisplayName(for: $0) } ?? "Workspace Default"
        let channelLabel = channelID.map { $0.capitalized } ?? "All Channels"
        if principalActorID == nil, channelID == nil {
            return "Workspace Default"
        }
        if principalActorID == nil {
            return "\(channelLabel) Channel"
        }
        if channelID == nil {
            return "Principal \(principalLabel)"
        }
        return "Principal \(principalLabel) • \(channelLabel) Channel"
    }

    private func userFacingChatPersonaPolicyErrorMessage(
        _ error: Error,
        operation: String
    ) -> String {
        if let daemonError = error as? DaemonAPIError {
            if daemonError.isUnauthorized {
                connectionStatus = .degraded
                return "Authentication failed while \(operation) persona policy. Save a valid Assistant Access Token and retry."
            }
            if daemonError.isConnectivityIssue {
                connectionStatus = .disconnected
                return "Could not reach daemon while \(operation) persona policy. Start or repair daemon, then retry."
            }
            if daemonError.serverCode == "service_not_configured" {
                connectionStatus = .degraded
                return "Persona policy service is not configured in daemon yet. Update or repair daemon, then retry."
            }
            if daemonError.serverCode == "resource_not_found" {
                connectionStatus = .degraded
                return "Persona policy API is unavailable in this daemon build. Update app/daemon and retry."
            }
            if case .decoding = daemonError {
                connectionStatus = .degraded
                return "Received an unexpected persona policy response. Refresh and try again."
            }
            connectionStatus = .degraded
            return daemonError.errorDescription ?? "Persona policy request failed."
        }
        connectionStatus = .degraded
        return "Persona policy request failed. Refresh and try again."
    }

    private func responseShapingProfileID(for channelID: String?) -> String {
        switch nonEmpty(channelID)?.lowercased() {
        case "message":
            return "message.compact"
        case "voice":
            return "voice.spoken"
        default:
            return "app.default"
        }
    }

    private struct CapabilityGrantMutationPayload {
        let grantID: String?
        let actorID: String?
        let capabilityKey: String?
        let scopeJSON: String?
        let status: String
        let expiresAt: String?
    }

    private struct CapabilityGrantValidationError: Error {
        let message: String
    }

    private func fetchCapabilityGrantAndTrustReceiptPanels() async {
        await fetchCapabilityGrantAndTrustReceiptPanels(authToken: nil)
    }

    private func fetchCapabilityGrantAndTrustReceiptPanels(authToken providedAuthToken: String?) async {
        async let grantsRefresh: Void = fetchCapabilityGrantInventory(authToken: providedAuthToken)
        async let webhookRefresh: Void = fetchWebhookTrustReceipts(authToken: providedAuthToken)
        async let ingestRefresh: Void = fetchIngestTrustReceipts(authToken: providedAuthToken)
        _ = await (grantsRefresh, webhookRefresh, ingestRefresh)
    }

    private func fetchCapabilityGrantInventory() async {
        await fetchCapabilityGrantInventory(authToken: nil)
    }

    private func fetchCapabilityGrantInventory(authToken providedAuthToken: String?) async {
        if isCapabilityGrantInventoryRequestInFlight {
            return
        }
        isCapabilityGrantInventoryRequestInFlight = true
        isCapabilityGrantInventoryLoading = true
        defer {
            isCapabilityGrantInventoryRequestInFlight = false
            isCapabilityGrantInventoryLoading = false
        }

        guard let authToken = providedAuthToken ?? resolvedAuthToken() else {
            capabilityGrantItems = []
            capabilityGrantInventoryHasMore = false
            capabilityGrantStatusMessage = "Set Assistant Access Token to query capability grants."
            return
        }

        let actorID = nonEmpty(capabilityGrantActorFilter) ?? nonEmpty(selectedPrincipal)
        if let actorID {
            capabilityGrantActorFilter = actorID
        }
        let capabilityKey = nonEmpty(capabilityGrantKeyFilter)
        let status = normalizedCapabilityGrantStatusFilter(capabilityGrantStatusFilter)
        let requestLimit = clampedGovernanceQueryLimit(capabilityGrantLimit)
        if capabilityGrantLimit != requestLimit {
            capabilityGrantLimit = requestLimit
        }

        do {
            let response = try await daemonClient.identity.capabilityGrantList(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                actorID: actorID,
                capabilityKey: capabilityKey,
                status: status,
                cursorCreatedAt: nil,
                cursorID: nil,
                limit: requestLimit
            )
            updateWorkspaceContext(from: response.workspaceID)
            let mapped = response.items
                .map(mapCapabilityGrantRecord)
                .sorted { lhs, rhs in
                    if lhs.sortTimestamp == rhs.sortTimestamp {
                        return lhs.id > rhs.id
                    }
                    return lhs.sortTimestamp > rhs.sortTimestamp
                }
            capabilityGrantItems = mapped
            capabilityGrantInventoryHasMore = response.hasMore
            capabilityGrantStatusMessage = capabilityGrantInventorySummaryMessage(
                itemCount: mapped.count,
                hasMore: response.hasMore,
                actorID: actorID,
                capabilityKey: capabilityKey,
                status: status
            )
            let activeIDs = Set(mapped.map(\.id))
            capabilityGrantActionStatusByID = capabilityGrantActionStatusByID.filter { activeIDs.contains($0.key) }
            capabilityGrantRevokeInFlightIDs = Set(capabilityGrantRevokeInFlightIDs.filter { activeIDs.contains($0) })
            connectionStatus = .connected
        } catch {
            capabilityGrantItems = []
            capabilityGrantInventoryHasMore = false
            capabilityGrantStatusMessage = daemonErrorMessage(
                error,
                fallbackContext: "Capability grant list query failed",
                updateConnectionStatus: false
            )
        }
    }

    private func performCapabilityGrantUpsert(_ input: CapabilityGrantMutationInput) async {
        let payload: CapabilityGrantMutationPayload
        do {
            payload = try resolvedCapabilityGrantMutationPayload(from: input)
        } catch let validationError as CapabilityGrantValidationError {
            capabilityGrantMutationStatusMessage = validationError.message
            return
        } catch {
            capabilityGrantMutationStatusMessage = "Capability grant input validation failed."
            return
        }

        guard !isCapabilityGrantMutationRequestInFlight else {
            return
        }
        guard let authToken = resolvedAuthToken() else {
            capabilityGrantMutationStatusMessage = "Set Assistant Access Token before mutating capability grants."
            return
        }

        isCapabilityGrantMutationRequestInFlight = true
        isCapabilityGrantMutationInFlight = true
        defer {
            isCapabilityGrantMutationRequestInFlight = false
            isCapabilityGrantMutationInFlight = false
        }

        capabilityGrantMutationStatusMessage = "Saving capability grant…"

        do {
            let response = try await daemonClient.identity.capabilityGrantUpsert(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                grantID: payload.grantID,
                actorID: payload.actorID,
                capabilityKey: payload.capabilityKey,
                scopeJSON: payload.scopeJSON,
                status: payload.status,
                expiresAt: payload.expiresAt
            )
            updateWorkspaceContext(from: response.workspaceID)
            let resolvedID = nonEmpty(response.grantID) ?? nonEmpty(payload.grantID) ?? UUID().uuidString.lowercased()
            capabilityGrantActionStatusByID[resolvedID] = "Capability grant saved."
            capabilityGrantMutationStatusMessage = "Saved capability grant \(resolvedID) (\(response.status.uppercased()))."
            connectionStatus = .connected
            await fetchCapabilityGrantInventory(authToken: authToken)
        } catch {
            capabilityGrantMutationStatusMessage = daemonErrorMessage(
                error,
                fallbackContext: "Capability grant mutation failed",
                updateConnectionStatus: false
            )
        }
    }

    private func performCapabilityGrantRevoke(grantID: String) async {
        let normalizedGrantID = grantID.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !normalizedGrantID.isEmpty else {
            return
        }
        guard let authToken = resolvedAuthToken() else {
            let message = "Set Assistant Access Token before revoking capability grants."
            capabilityGrantActionStatusByID[normalizedGrantID] = message
            capabilityGrantMutationStatusMessage = message
            return
        }
        guard !capabilityGrantRevokeInFlightIDs.contains(normalizedGrantID) else {
            return
        }

        capabilityGrantRevokeInFlightIDs.insert(normalizedGrantID)
        defer {
            capabilityGrantRevokeInFlightIDs.remove(normalizedGrantID)
        }

        do {
            let response = try await daemonClient.identity.capabilityGrantUpsert(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                grantID: normalizedGrantID,
                actorID: nil,
                capabilityKey: nil,
                scopeJSON: nil,
                status: "REVOKED",
                expiresAt: nil
            )
            updateWorkspaceContext(from: response.workspaceID)
            capabilityGrantActionStatusByID[normalizedGrantID] = "Capability grant \(normalizedGrantID) revoked."
            capabilityGrantMutationStatusMessage = capabilityGrantActionStatusByID[normalizedGrantID]
            connectionStatus = .connected
            await fetchCapabilityGrantInventory(authToken: authToken)
        } catch {
            let message = daemonErrorMessage(
                error,
                fallbackContext: "Capability grant revoke failed",
                updateConnectionStatus: false
            )
            capabilityGrantActionStatusByID[normalizedGrantID] = message
            capabilityGrantMutationStatusMessage = message
        }
    }

    private func resolvedCapabilityGrantMutationPayload(
        from input: CapabilityGrantMutationInput
    ) throws -> CapabilityGrantMutationPayload {
        let grantID = nonEmpty(input.grantID)
        let actorID = nonEmpty(input.actorID)
        let capabilityKey = nonEmpty(input.capabilityKey)?.lowercased()
        if grantID == nil, actorID == nil {
            throw CapabilityGrantValidationError(message: "Actor ID is required when creating a capability grant.")
        }
        if grantID == nil, capabilityKey == nil {
            throw CapabilityGrantValidationError(message: "Capability key is required when creating a capability grant.")
        }

        let status = input.status.trimmingCharacters(in: .whitespacesAndNewlines).uppercased()
        guard status == "ACTIVE" || status == "DISABLED" || status == "REVOKED" else {
            throw CapabilityGrantValidationError(message: "Status must be ACTIVE, DISABLED, or REVOKED.")
        }

        let scopeJSON = nonEmpty(input.scopeJSON)
        if let scopeJSON {
            guard let data = scopeJSON.data(using: .utf8),
                  (try? JSONDecoder().decode(DaemonJSONValue.self, from: data)) != nil else {
                throw CapabilityGrantValidationError(message: "Scope JSON must be valid JSON.")
            }
        }

        let expiresAtRaw = nonEmpty(input.expiresAt)
        if let expiresAtRaw,
           let parsed = parseDaemonTimestamp(expiresAtRaw),
           status == "ACTIVE",
           parsed <= Date() {
            throw CapabilityGrantValidationError(message: "Expires At must be in the future for ACTIVE grants.")
        } else if let expiresAtRaw, parseDaemonTimestamp(expiresAtRaw) == nil {
            throw CapabilityGrantValidationError(message: "Expires At must be a valid RFC3339 timestamp.")
        }

        return CapabilityGrantMutationPayload(
            grantID: grantID,
            actorID: actorID,
            capabilityKey: capabilityKey,
            scopeJSON: scopeJSON,
            status: status,
            expiresAt: expiresAtRaw
        )
    }

    private func fetchWebhookTrustReceipts() async {
        await fetchWebhookTrustReceipts(authToken: nil)
    }

    private func fetchWebhookTrustReceipts(authToken providedAuthToken: String?) async {
        if isWebhookReceiptsRequestInFlight {
            return
        }
        isWebhookReceiptsRequestInFlight = true
        isWebhookReceiptsLoading = true
        defer {
            isWebhookReceiptsRequestInFlight = false
            isWebhookReceiptsLoading = false
        }

        guard let authToken = providedAuthToken ?? resolvedAuthToken() else {
            webhookReceiptItems = []
            webhookReceiptsHasMore = false
            webhookReceiptsStatusMessage = "Set Assistant Access Token to query webhook trust receipts."
            return
        }

        let provider = nonEmpty(webhookReceiptProviderFilter)?.lowercased()
        let providerEventID = nonEmpty(webhookReceiptProviderEventIDFilter)
        let providerEventQuery = nonEmpty(webhookReceiptProviderEventQueryFilter)
        let eventID = nonEmpty(webhookReceiptEventIDFilter)
        let requestLimit = clampedGovernanceQueryLimit(webhookReceiptLimit)
        if requestLimit != webhookReceiptLimit {
            webhookReceiptLimit = requestLimit
        }

        do {
            let response = try await daemonClient.communications.commWebhookReceiptList(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                provider: provider,
                providerEventID: providerEventID,
                providerEventQuery: providerEventQuery,
                eventID: eventID,
                cursorCreatedAt: nil,
                cursorID: nil,
                limit: requestLimit
            )
            updateWorkspaceContext(from: response.workspaceID)
            let mapped = response.items
                .map(mapWebhookTrustReceiptItem)
                .sorted { lhs, rhs in
                    if lhs.sortTimestamp == rhs.sortTimestamp {
                        return lhs.id > rhs.id
                    }
                    return lhs.sortTimestamp > rhs.sortTimestamp
                }
            webhookReceiptItems = mapped
            webhookReceiptsHasMore = response.hasMore
            webhookReceiptsStatusMessage = webhookReceiptSummaryMessage(
                itemCount: mapped.count,
                hasMore: response.hasMore,
                provider: provider,
                providerEventID: providerEventID,
                providerEventQuery: providerEventQuery,
                eventID: eventID
            )
            connectionStatus = .connected
        } catch {
            webhookReceiptItems = []
            webhookReceiptsHasMore = false
            webhookReceiptsStatusMessage = daemonErrorMessage(
                error,
                fallbackContext: "Webhook trust receipt query failed",
                updateConnectionStatus: false
            )
        }
    }

    private func fetchIngestTrustReceipts() async {
        await fetchIngestTrustReceipts(authToken: nil)
    }

    private func fetchIngestTrustReceipts(authToken providedAuthToken: String?) async {
        if isIngestReceiptsRequestInFlight {
            return
        }
        isIngestReceiptsRequestInFlight = true
        isIngestReceiptsLoading = true
        defer {
            isIngestReceiptsRequestInFlight = false
            isIngestReceiptsLoading = false
        }

        guard let authToken = providedAuthToken ?? resolvedAuthToken() else {
            ingestReceiptItems = []
            ingestReceiptsHasMore = false
            ingestReceiptsStatusMessage = "Set Assistant Access Token to query ingest trust receipts."
            return
        }

        let source = nonEmpty(ingestReceiptSourceFilter)?.lowercased()
        let sourceScope = nonEmpty(ingestReceiptSourceScopeFilter)
        let sourceEventID = nonEmpty(ingestReceiptSourceEventIDFilter)
        let sourceEventQuery = nonEmpty(ingestReceiptSourceEventQueryFilter)
        let trustState = normalizedTrustStateFilter(ingestReceiptTrustStateFilter)
        let eventID = nonEmpty(ingestReceiptEventIDFilter)
        let requestLimit = clampedGovernanceQueryLimit(ingestReceiptLimit)
        if requestLimit != ingestReceiptLimit {
            ingestReceiptLimit = requestLimit
        }

        do {
            let response = try await daemonClient.communications.commIngestReceiptList(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                source: source,
                sourceScope: sourceScope,
                sourceEventID: sourceEventID,
                sourceEventQuery: sourceEventQuery,
                trustState: trustState,
                eventID: eventID,
                cursorCreatedAt: nil,
                cursorID: nil,
                limit: requestLimit
            )
            updateWorkspaceContext(from: response.workspaceID)
            let mapped = response.items
                .map(mapIngestTrustReceiptItem)
                .sorted { lhs, rhs in
                    if lhs.sortTimestamp == rhs.sortTimestamp {
                        return lhs.id > rhs.id
                    }
                    return lhs.sortTimestamp > rhs.sortTimestamp
                }
            ingestReceiptItems = mapped
            ingestReceiptsHasMore = response.hasMore
            ingestReceiptsStatusMessage = ingestReceiptSummaryMessage(
                itemCount: mapped.count,
                hasMore: response.hasMore,
                source: source,
                sourceScope: sourceScope,
                sourceEventID: sourceEventID,
                sourceEventQuery: sourceEventQuery,
                trustState: trustState,
                eventID: eventID
            )
            connectionStatus = .connected
        } catch {
            ingestReceiptItems = []
            ingestReceiptsHasMore = false
            ingestReceiptsStatusMessage = daemonErrorMessage(
                error,
                fallbackContext: "Ingest trust receipt query failed",
                updateConnectionStatus: false
            )
        }
    }

    private func performDelegationGrant(_ input: DelegationGrantInput) async {
        let payload: DelegationGrantPayload
        do {
            payload = try resolvedDelegationGrantPayload(from: input)
        } catch let validationError as DelegationGrantValidationError {
            delegationStatusMessage = validationError.message
            return
        } catch {
            delegationStatusMessage = "Delegation grant input validation failed."
            return
        }

        guard let authToken = resolvedAuthToken() else {
            delegationStatusMessage = "Set Assistant Access Token before granting delegation rules."
            return
        }

        isDelegationGrantInFlight = true
        defer {
            isDelegationGrantInFlight = false
        }

        do {
            let created = try await daemonClient.identity.delegationGrant(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                fromActorID: payload.fromActorID,
                toActorID: payload.toActorID,
                scopeType: payload.scopeType,
                scopeKey: payload.scopeKey,
                expiresAt: payload.expiresAt
            )
            delegationActionStatusByRuleID[created.id] = "Delegation rule granted."
            let scopeSummary = delegationScopeSummary(scopeType: created.scopeType, scopeKey: created.scopeKey)
            delegationStatusMessage = "Granted delegation \(created.fromActorID) -> \(created.toActorID) (\(scopeSummary))."
            connectionStatus = .connected
            await fetchDelegationRules(authToken: authToken)
        } catch {
            delegationStatusMessage = daemonErrorMessage(
                error,
                fallbackContext: "Delegation grant failed",
                updateConnectionStatus: false
            )
        }
    }

    private func performDelegationRevoke(ruleID: String) async {
        let trimmedRuleID = ruleID.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmedRuleID.isEmpty else {
            return
        }

        guard let authToken = resolvedAuthToken() else {
            delegationActionStatusByRuleID[trimmedRuleID] = "Set Assistant Access Token before revoking delegation rules."
            delegationStatusMessage = delegationActionStatusByRuleID[trimmedRuleID]
            return
        }

        delegationRevokeInFlightRuleIDs.insert(trimmedRuleID)
        defer {
            delegationRevokeInFlightRuleIDs.remove(trimmedRuleID)
        }

        do {
            let response = try await daemonClient.identity.delegationRevoke(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                ruleID: trimmedRuleID
            )
            let normalizedStatus = response.status.trimmingCharacters(in: .whitespacesAndNewlines)
            let statusLabel = normalizedStatus.isEmpty ? "revoked" : normalizedStatus.lowercased()
            delegationActionStatusByRuleID[trimmedRuleID] = "Delegation rule \(trimmedRuleID) \(statusLabel)."
            delegationStatusMessage = delegationActionStatusByRuleID[trimmedRuleID]
            connectionStatus = .connected
            await fetchDelegationRules(authToken: authToken)
        } catch {
            let message = daemonErrorMessage(
                error,
                fallbackContext: "Delegation revoke failed",
                updateConnectionStatus: false
            )
            delegationActionStatusByRuleID[trimmedRuleID] = message
            delegationStatusMessage = message
        }
    }

    private func resolvedDelegationGrantPayload(
        from input: DelegationGrantInput
    ) throws -> DelegationGrantPayload {
        let fromActorID = input.fromActorID.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !fromActorID.isEmpty else {
            throw DelegationGrantValidationError(message: "From actor is required.")
        }

        let toActorID = input.toActorID.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !toActorID.isEmpty else {
            throw DelegationGrantValidationError(message: "To actor is required.")
        }

        guard fromActorID != toActorID else {
            throw DelegationGrantValidationError(message: "Delegation denied: self delegation is not allowed.")
        }

        let scopeType = input.scopeType.trimmingCharacters(in: .whitespacesAndNewlines).uppercased()
        guard delegationScopeOptions.contains(scopeType) else {
            throw DelegationGrantValidationError(message: "Scope type must be EXECUTION, APPROVAL, or ALL.")
        }

        let trimmedScopeKey = input.scopeKey?.trimmingCharacters(in: .whitespacesAndNewlines)
        let scopeKey = (trimmedScopeKey?.isEmpty ?? true) ? nil : trimmedScopeKey
        if scopeType == "ALL", scopeKey != nil {
            throw DelegationGrantValidationError(message: "scope_key is not allowed when scope_type=ALL.")
        }

        let trimmedExpiresAt = input.expiresAt?.trimmingCharacters(in: .whitespacesAndNewlines)
        let expiresAt = (trimmedExpiresAt?.isEmpty ?? true) ? nil : trimmedExpiresAt
        if let expiresAt {
            guard let parsed = parseDaemonTimestamp(expiresAt) else {
                throw DelegationGrantValidationError(message: "Expires At must be a valid RFC3339 timestamp.")
            }
            if parsed <= Date() {
                throw DelegationGrantValidationError(message: "Delegation denied: expires_at must be in the future.")
            }
        }

        return DelegationGrantPayload(
            fromActorID: fromActorID,
            toActorID: toActorID,
            scopeType: scopeType,
            scopeKey: scopeKey,
            expiresAt: expiresAt
        )
    }

    private func performRetentionPurge() async {
        guard let authToken = resolvedAuthToken() else {
            retentionStatusMessage = "Set Assistant Access Token before running retention purge."
            return
        }

        isRetentionActionInFlight = true
        defer {
            isRetentionActionInFlight = false
        }

        do {
            let response = try await daemonClient.context.retentionPurge(
                baseURL: daemonBaseURL,
                authToken: authToken,
                traceDays: retentionTraceDays,
                transcriptDays: retentionTranscriptDays,
                memoryDays: retentionMemoryDays
            )
            retentionStatusMessage = "Retention purge completed: \(truncateText(response.displayText, limit: 240))"
            connectionStatus = .connected
        } catch {
            retentionStatusMessage = daemonErrorMessage(
                error,
                fallbackContext: "Retention purge failed",
                updateConnectionStatus: false
            )
        }
    }

    private func performRetentionCompactMemory() async {
        guard let authToken = resolvedAuthToken() else {
            retentionStatusMessage = "Set Assistant Access Token before running memory compaction."
            return
        }

        let owner = nonEmpty(selectedPrincipal) ?? "default"
        isRetentionActionInFlight = true
        defer {
            isRetentionActionInFlight = false
        }

        do {
            let response = try await daemonClient.context.retentionCompactMemory(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                ownerActor: owner,
                tokenThreshold: retentionTokenThreshold,
                staleAfterHours: retentionStaleAfterHours,
                limit: retentionCompactionLimit,
                apply: retentionCompactionApply
            )
            let modeLabel = retentionCompactionApply ? "Applied" : "Preview"
            retentionStatusMessage = "\(modeLabel) memory compaction for \(owner): \(truncateText(response.displayText, limit: 240))"
            connectionStatus = .connected
        } catch {
            retentionStatusMessage = daemonErrorMessage(
                error,
                fallbackContext: "Memory compaction failed",
                updateConnectionStatus: false
            )
        }
    }

    private func performContextSamplesQuery() async {
        guard let authToken = resolvedAuthToken() else {
            contextStatusMessage = "Set Assistant Access Token before querying context samples."
            return
        }

        let taskClass = nonEmpty(contextTaskClass) ?? "chat"
        contextTaskClass = taskClass
        isContextActionInFlight = true
        defer {
            isContextActionInFlight = false
        }

        do {
            let response = try await daemonClient.context.contextSamples(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                taskClass: taskClass,
                limit: contextSamplesLimit
            )
            let sampleCount = jsonArrayCount(value: response, key: "samples") ?? 0
            let summary = truncateText(response.displayText, limit: 220)
            contextStatusMessage = sampleCount > 0
                ? "Loaded \(sampleCount) sample(s) for \(taskClass): \(summary)"
                : "No context samples returned for \(taskClass): \(summary)"
            connectionStatus = .connected
        } catch {
            contextStatusMessage = daemonErrorMessage(
                error,
                fallbackContext: "Context samples query failed",
                updateConnectionStatus: false
            )
        }
    }

    private func performContextTune() async {
        guard let authToken = resolvedAuthToken() else {
            contextStatusMessage = "Set Assistant Access Token before tuning context profile."
            return
        }

        let taskClass = nonEmpty(contextTaskClass) ?? "chat"
        contextTaskClass = taskClass
        isContextActionInFlight = true
        defer {
            isContextActionInFlight = false
        }

        do {
            let response = try await daemonClient.context.contextTune(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                taskClass: taskClass
            )
            contextStatusMessage = "Context tune completed for \(taskClass): \(truncateText(response.displayText, limit: 220))"
            connectionStatus = .connected
        } catch {
            contextStatusMessage = daemonErrorMessage(
                error,
                fallbackContext: "Context tune failed",
                updateConnectionStatus: false
            )
        }
    }

    private func fetchContextMemoryAndRetrievalPanels() async {
        async let inventoryRefresh: Void = fetchContextMemoryInventory()
        async let candidatesRefresh: Void = fetchContextMemoryCandidates()
        async let retrievalRefresh: Void = fetchContextRetrievalDocumentsAndChunks()
        _ = await (inventoryRefresh, candidatesRefresh, retrievalRefresh)
    }

    private func fetchContextRetrievalDocumentsAndChunks() async {
        await fetchContextRetrievalDocuments()
        await fetchContextRetrievalChunks()
    }

    private func fetchContextMemoryInventory() async {
        if isContextMemoryInventoryRequestInFlight {
            return
        }
        isContextMemoryInventoryRequestInFlight = true
        isContextMemoryInventoryLoading = true
        defer {
            isContextMemoryInventoryRequestInFlight = false
            isContextMemoryInventoryLoading = false
        }

        guard let authToken = resolvedAuthToken() else {
            contextMemoryInventoryItems = []
            contextMemoryInventoryHasMore = false
            contextMemoryInventoryStatusMessage = "Set Assistant Access Token to query context memory inventory."
            return
        }

        let ownerActorID = nonEmpty(contextMemoryOwnerActorFilter) ?? nonEmpty(selectedPrincipal)
        if let ownerActorID {
            contextMemoryOwnerActorFilter = ownerActorID
        }
        let scopeType = nonEmpty(contextMemoryScopeTypeFilter)
        let status = normalizedContextFilter(contextMemoryStatusFilter)
        let sourceType = nonEmpty(contextMemorySourceTypeFilter)
        let sourceRefQuery = nonEmpty(contextMemorySourceRefQuery)
        let requestLimit = clampedContextQueryLimit(contextMemoryLimit)
        if requestLimit != contextMemoryLimit {
            contextMemoryLimit = requestLimit
        }

        do {
            let response = try await daemonClient.context.contextMemoryInventory(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                ownerActorID: ownerActorID,
                scopeType: scopeType,
                status: status,
                sourceType: sourceType,
                sourceRefQuery: sourceRefQuery,
                cursorUpdatedAt: nil,
                cursorID: nil,
                limit: requestLimit
            )
            let mappedItems = response.items
                .map(mapContextMemoryInventoryItem)
                .sorted { lhs, rhs in
                    if lhs.sortTimestamp == rhs.sortTimestamp {
                        return lhs.id > rhs.id
                    }
                    return lhs.sortTimestamp > rhs.sortTimestamp
                }
            contextMemoryInventoryItems = mappedItems
            contextMemoryInventoryHasMore = response.hasMore
            contextMemoryInventoryStatusMessage = contextMemoryInventorySummaryMessage(
                itemCount: mappedItems.count,
                hasMore: response.hasMore,
                ownerActorID: ownerActorID,
                scopeType: scopeType,
                status: status,
                sourceType: sourceType,
                sourceRefQuery: sourceRefQuery
            )
            connectionStatus = .connected
        } catch {
            contextMemoryInventoryItems = []
            contextMemoryInventoryHasMore = false
            contextMemoryInventoryStatusMessage = daemonErrorMessage(
                error,
                fallbackContext: "Context memory inventory query failed",
                updateConnectionStatus: false
            )
        }
    }

    private func fetchContextMemoryCandidates() async {
        if isContextMemoryCandidatesRequestInFlight {
            return
        }
        isContextMemoryCandidatesRequestInFlight = true
        isContextMemoryCandidatesLoading = true
        defer {
            isContextMemoryCandidatesRequestInFlight = false
            isContextMemoryCandidatesLoading = false
        }

        guard let authToken = resolvedAuthToken() else {
            contextMemoryCandidateItems = []
            contextMemoryCandidatesHasMore = false
            contextMemoryCandidatesStatusMessage = "Set Assistant Access Token to query memory compaction candidates."
            return
        }

        let ownerActorID = nonEmpty(contextMemoryCandidatesOwnerActorFilter) ?? nonEmpty(selectedPrincipal)
        if let ownerActorID {
            contextMemoryCandidatesOwnerActorFilter = ownerActorID
        }
        let status = normalizedContextFilter(contextMemoryCandidatesStatusFilter)
        let requestLimit = clampedContextQueryLimit(contextMemoryCandidatesLimit)
        if requestLimit != contextMemoryCandidatesLimit {
            contextMemoryCandidatesLimit = requestLimit
        }

        do {
            let response = try await daemonClient.context.contextMemoryCompactionCandidates(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                ownerActorID: ownerActorID,
                status: status,
                cursorCreatedAt: nil,
                cursorID: nil,
                limit: requestLimit
            )
            let mappedItems = response.items
                .map(mapContextMemoryCandidateItem)
                .sorted { lhs, rhs in
                    if lhs.sortTimestamp == rhs.sortTimestamp {
                        return lhs.id > rhs.id
                    }
                    return lhs.sortTimestamp > rhs.sortTimestamp
                }
            contextMemoryCandidateItems = mappedItems
            contextMemoryCandidatesHasMore = response.hasMore
            contextMemoryCandidatesStatusMessage = contextMemoryCandidatesSummaryMessage(
                itemCount: mappedItems.count,
                hasMore: response.hasMore,
                ownerActorID: ownerActorID,
                status: status
            )
            connectionStatus = .connected
        } catch {
            contextMemoryCandidateItems = []
            contextMemoryCandidatesHasMore = false
            contextMemoryCandidatesStatusMessage = daemonErrorMessage(
                error,
                fallbackContext: "Context memory candidates query failed",
                updateConnectionStatus: false
            )
        }
    }

    private func fetchContextRetrievalDocuments() async {
        if isContextRetrievalDocumentsRequestInFlight {
            return
        }
        isContextRetrievalDocumentsRequestInFlight = true
        isContextRetrievalDocumentsLoading = true
        defer {
            isContextRetrievalDocumentsRequestInFlight = false
            isContextRetrievalDocumentsLoading = false
        }

        guard let authToken = resolvedAuthToken() else {
            contextRetrievalDocumentItems = []
            contextRetrievalDocumentsHasMore = false
            selectedContextRetrievalDocumentID = ""
            contextRetrievalDocumentsStatusMessage = "Set Assistant Access Token to query retrieval documents."
            contextRetrievalChunkItems = []
            contextRetrievalChunksHasMore = false
            contextRetrievalChunksStatusMessage = "Set Assistant Access Token to query retrieval chunks."
            return
        }

        let ownerActorID = nonEmpty(contextRetrievalOwnerActorFilter) ?? nonEmpty(selectedPrincipal)
        if let ownerActorID {
            contextRetrievalOwnerActorFilter = ownerActorID
        }
        let sourceURIQuery = nonEmpty(contextRetrievalSourceURIQuery)
        let requestLimit = clampedContextQueryLimit(contextRetrievalDocumentsLimit)
        if requestLimit != contextRetrievalDocumentsLimit {
            contextRetrievalDocumentsLimit = requestLimit
        }

        do {
            let response = try await daemonClient.context.contextRetrievalDocuments(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                ownerActorID: ownerActorID,
                sourceURIQuery: sourceURIQuery,
                cursorCreatedAt: nil,
                cursorID: nil,
                limit: requestLimit
            )
            let mappedItems = response.items
                .map(mapContextRetrievalDocumentItem)
                .sorted { lhs, rhs in
                    if lhs.sortTimestamp == rhs.sortTimestamp {
                        return lhs.id > rhs.id
                    }
                    return lhs.sortTimestamp > rhs.sortTimestamp
                }
            contextRetrievalDocumentItems = mappedItems
            contextRetrievalDocumentsHasMore = response.hasMore
            contextRetrievalDocumentsStatusMessage = contextRetrievalDocumentsSummaryMessage(
                itemCount: mappedItems.count,
                hasMore: response.hasMore,
                ownerActorID: ownerActorID,
                sourceURIQuery: sourceURIQuery
            )
            if let selectedID = nonEmpty(selectedContextRetrievalDocumentID),
               mappedItems.contains(where: { $0.id == selectedID }) {
                selectedContextRetrievalDocumentID = selectedID
            } else {
                selectedContextRetrievalDocumentID = mappedItems.first?.id ?? ""
            }
            if mappedItems.isEmpty {
                contextRetrievalChunkItems = []
                contextRetrievalChunksHasMore = false
                contextRetrievalChunksStatusMessage = "Select a retrieval document to inspect retrieval chunks."
            }
            connectionStatus = .connected
        } catch {
            contextRetrievalDocumentItems = []
            contextRetrievalDocumentsHasMore = false
            selectedContextRetrievalDocumentID = ""
            contextRetrievalDocumentsStatusMessage = daemonErrorMessage(
                error,
                fallbackContext: "Context retrieval documents query failed",
                updateConnectionStatus: false
            )
            contextRetrievalChunkItems = []
            contextRetrievalChunksHasMore = false
            contextRetrievalChunksStatusMessage = "Select a retrieval document to inspect retrieval chunks."
        }
    }

    private func fetchContextRetrievalChunks() async {
        if isContextRetrievalChunksRequestInFlight {
            return
        }
        guard let selectedDocumentID = nonEmpty(selectedContextRetrievalDocumentID) else {
            contextRetrievalChunkItems = []
            contextRetrievalChunksHasMore = false
            contextRetrievalChunksStatusMessage = "Select a retrieval document to inspect retrieval chunks."
            return
        }
        isContextRetrievalChunksRequestInFlight = true
        isContextRetrievalChunksLoading = true
        defer {
            isContextRetrievalChunksRequestInFlight = false
            isContextRetrievalChunksLoading = false
        }

        guard let authToken = resolvedAuthToken() else {
            contextRetrievalChunkItems = []
            contextRetrievalChunksHasMore = false
            contextRetrievalChunksStatusMessage = "Set Assistant Access Token to query retrieval chunks."
            return
        }

        let ownerActorID = nonEmpty(contextRetrievalOwnerActorFilter) ?? nonEmpty(selectedPrincipal)
        let sourceURIQuery = nonEmpty(contextRetrievalSourceURIQuery)
        let chunkTextQuery = nonEmpty(contextRetrievalChunkTextQuery)
        let requestLimit = clampedContextQueryLimit(contextRetrievalChunksLimit)
        if requestLimit != contextRetrievalChunksLimit {
            contextRetrievalChunksLimit = requestLimit
        }

        do {
            let response = try await daemonClient.context.contextRetrievalChunks(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                documentID: selectedDocumentID,
                ownerActorID: ownerActorID,
                sourceURIQuery: sourceURIQuery,
                chunkTextQuery: chunkTextQuery,
                cursorCreatedAt: nil,
                cursorID: nil,
                limit: requestLimit
            )
            let mappedItems = response.items
                .map(mapContextRetrievalChunkItem)
                .sorted { lhs, rhs in
                    if lhs.sortTimestamp == rhs.sortTimestamp {
                        if lhs.chunkIndex == rhs.chunkIndex {
                            return lhs.id < rhs.id
                        }
                        return lhs.chunkIndex < rhs.chunkIndex
                    }
                    return lhs.sortTimestamp > rhs.sortTimestamp
                }
            contextRetrievalChunkItems = mappedItems
            contextRetrievalChunksHasMore = response.hasMore
            contextRetrievalChunksStatusMessage = contextRetrievalChunksSummaryMessage(
                itemCount: mappedItems.count,
                hasMore: response.hasMore,
                documentID: selectedDocumentID,
                chunkTextQuery: chunkTextQuery
            )
            connectionStatus = .connected
        } catch {
            contextRetrievalChunkItems = []
            contextRetrievalChunksHasMore = false
            contextRetrievalChunksStatusMessage = daemonErrorMessage(
                error,
                fallbackContext: "Context retrieval chunks query failed",
                updateConnectionStatus: false
            )
        }
    }

    private func performConnectorPermissionRequest(id: String) async {
        guard !connectorPermissionRequestInFlightIDs.contains(id) else {
            return
        }
        guard let index = connectorCards.firstIndex(where: { $0.id == id }) else {
            return
        }
        guard connectorCards[index].permissionState != .granted else {
            connectorPermissionActionStatusByID[id] = "Permission already granted."
            return
        }
        guard let authToken = resolvedAuthToken() else {
            connectorPermissionActionStatusByID[id] = "Set Assistant Access Token before requesting connector permission."
            return
        }

        connectorPermissionRequestInFlightIDs.insert(id)
        connectorPermissionActionStatusByID[id] = "Requesting daemon-managed permission prompt…"
        defer {
            connectorPermissionRequestInFlightIDs.remove(id)
        }

        do {
            let response = try await daemonClient.connectors.connectorPermissionRequest(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                connectorID: id
            )
            applyConnectorPermissionState(
                connectorPermissionState(
                    fromDaemonState: response.permissionState,
                    fallback: connectorPermissionStatesByID[id] ?? .unknown
                ),
                connectorID: id
            )
            connectorPermissionRefreshPendingIDs.remove(id)
            connectorPermissionActionStatusByID[id] = response.message ?? "Connector permission request submitted via daemon."
            connectorsStatusMessage = "Connector permission flow routed through Personal Agent Daemon for \(id)."
            connectionStatus = .connected
        } catch {
            connectorPermissionActionStatusByID[id] = daemonErrorMessage(
                error,
                fallbackContext: "Connector permission request failed",
                updateConnectionStatus: false,
                panelContext: .connectors
            )
        }

        await fetchConnectorCards()
    }

    private func performChannelConfigurationSave(channelID: String) async {
        let normalizedChannelID = channelID.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        guard !normalizedChannelID.isEmpty else {
            return
        }
        guard !channelConfigSaveInFlightIDs.contains(normalizedChannelID) else {
            return
        }
        guard let authToken = resolvedAuthToken() else {
            channelConfigActionStatusByID[normalizedChannelID] = "Set Assistant Access Token before saving channel configuration."
            return
        }

        let draft = channelConfigDraftByID[normalizedChannelID] ?? [:]
        let kindMap = channelConfigKindsByID[normalizedChannelID] ?? [:]
        if let missingRequiredMessage = connectionConfigStore.missingRequiredConfigurationFieldMessage(
            draft: draft,
            descriptors: channelConfigFieldDescriptors(channelID: normalizedChannelID),
            subject: "Channel"
        ) {
            channelConfigActionStatusByID[normalizedChannelID] = missingRequiredMessage
            return
        }
        let payload: [String: DaemonConfigMutationValue]
        do {
            payload = try connectionConfigStore.configurationMutationPayload(
                draft: draft,
                kindMap: kindMap
            )
        } catch let error as AppConnectionConfigStore.ConfigurationMutationValidationError {
            channelConfigActionStatusByID[normalizedChannelID] = error.message
            return
        } catch {
            channelConfigActionStatusByID[normalizedChannelID] = "Channel configuration payload is invalid."
            return
        }

        channelConfigSaveInFlightIDs.insert(normalizedChannelID)
        channelConfigActionStatusByID[normalizedChannelID] = "Saving channel configuration…"
        defer {
            channelConfigSaveInFlightIDs.remove(normalizedChannelID)
        }

        var shouldRefresh = false
        do {
            let response = try await daemonClient.channels.channelConfigUpsert(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                channelID: normalizedChannelID,
                configuration: payload,
                merge: true
            )
            let mapped = mapConfigurationFields(
                response.configuration,
                descriptors: channelConfigFieldDescriptors(channelID: normalizedChannelID)
            )
            connectionConfigStore.applyChannelConfigurationFromDaemon(
                channelID: normalizedChannelID,
                editable: mapped.editable,
                editableKinds: mapped.editableKinds
            )
            channelConfigActionStatusByID[normalizedChannelID] =
                "Saved channel configuration at \(formattedWorkflowTimestamp(response.updatedAt))."
            channelsStatusMessage = "Saved channel configuration for \(normalizedChannelID)."
            connectionStatus = .connected
            shouldRefresh = true
        } catch {
            channelConfigActionStatusByID[normalizedChannelID] = daemonErrorMessage(
                error,
                fallbackContext: "Channel configuration save failed",
                updateConnectionStatus: false,
                panelContext: .channels
            )
        }

        if shouldRefresh {
            await fetchChannelCards()
        }
    }

    private func performConnectorConfigurationSave(connectorID: String) async {
        let normalizedConnectorID = connectorID.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        guard !normalizedConnectorID.isEmpty else {
            return
        }
        guard !connectorConfigSaveInFlightIDs.contains(normalizedConnectorID) else {
            return
        }
        guard let authToken = resolvedAuthToken() else {
            connectorConfigActionStatusByID[normalizedConnectorID] = "Set Assistant Access Token before saving connector configuration."
            return
        }

        let draft = connectorConfigDraftByID[normalizedConnectorID] ?? [:]
        let kindMap = connectorConfigKindsByID[normalizedConnectorID] ?? [:]
        if let missingRequiredMessage = connectionConfigStore.missingRequiredConfigurationFieldMessage(
            draft: draft,
            descriptors: connectorConfigFieldDescriptors(connectorID: normalizedConnectorID),
            subject: "Connector"
        ) {
            connectorConfigActionStatusByID[normalizedConnectorID] = missingRequiredMessage
            return
        }
        let payload: [String: DaemonConfigMutationValue]
        do {
            payload = try connectionConfigStore.configurationMutationPayload(
                draft: draft,
                kindMap: kindMap
            )
        } catch let error as AppConnectionConfigStore.ConfigurationMutationValidationError {
            connectorConfigActionStatusByID[normalizedConnectorID] = error.message
            return
        } catch {
            connectorConfigActionStatusByID[normalizedConnectorID] = "Connector configuration payload is invalid."
            return
        }

        connectorConfigSaveInFlightIDs.insert(normalizedConnectorID)
        connectorConfigActionStatusByID[normalizedConnectorID] = "Saving connector configuration…"
        defer {
            connectorConfigSaveInFlightIDs.remove(normalizedConnectorID)
        }

        var shouldRefresh = false
        do {
            let response = try await daemonClient.connectors.connectorConfigUpsert(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                connectorID: normalizedConnectorID,
                configuration: payload,
                merge: true
            )
            let mapped = mapConfigurationFields(
                response.configuration,
                descriptors: connectorConfigFieldDescriptors(connectorID: normalizedConnectorID)
            )
            connectionConfigStore.applyConnectorConfigurationFromDaemon(
                connectorID: normalizedConnectorID,
                editable: mapped.editable,
                editableKinds: mapped.editableKinds
            )
            connectorConfigActionStatusByID[normalizedConnectorID] =
                "Saved connector configuration at \(formattedWorkflowTimestamp(response.updatedAt))."
            connectorsStatusMessage = "Saved connector configuration for \(normalizedConnectorID)."
            connectionStatus = .connected
            shouldRefresh = true
        } catch {
            connectorConfigActionStatusByID[normalizedConnectorID] = daemonErrorMessage(
                error,
                fallbackContext: "Connector configuration save failed",
                updateConnectionStatus: false,
                panelContext: .connectors
            )
        }

        if shouldRefresh {
            await fetchConnectorCards()
        }
    }

    private func performChannelHealthCheck(channelID: String) async {
        let normalizedChannelID = channelID.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        guard !normalizedChannelID.isEmpty else {
            return
        }
        guard !channelTestInFlightIDs.contains(normalizedChannelID) else {
            return
        }
        guard let authToken = resolvedAuthToken() else {
            channelConfigActionStatusByID[normalizedChannelID] = "Set Assistant Access Token before running channel health check."
            return
        }

        channelTestInFlightIDs.insert(normalizedChannelID)
        channelConfigActionStatusByID[normalizedChannelID] = "Running channel health check…"
        defer {
            channelTestInFlightIDs.remove(normalizedChannelID)
        }

        var shouldRefresh = false
        do {
            let response = try await daemonClient.channels.channelTestOperation(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                channelID: normalizedChannelID,
                operation: "health"
            )
            let result = connectionConfigStore.mapConfigurationTestResult(
                from: response,
                formattedWorkflowTimestamp: formattedWorkflowTimestamp(_:)
            )
            channelLastTestResultByID[normalizedChannelID] = result
            channelConfigActionStatusByID[normalizedChannelID] = "Health check (\(result.status)): \(result.summary)"
            channelsStatusMessage = "Channel health check completed for \(normalizedChannelID)."
            connectionStatus = .connected
            shouldRefresh = true
        } catch {
            channelConfigActionStatusByID[normalizedChannelID] = daemonErrorMessage(
                error,
                fallbackContext: "Channel health check failed",
                updateConnectionStatus: false,
                panelContext: .channels
            )
        }

        if shouldRefresh {
            await fetchChannelCards()
        }
    }

    private func performChannelDeliveryPolicySave(channelID: String) async {
        let normalizedChannelID = normalizedChannelDeliveryPolicyChannelID(channelID)
        guard !normalizedChannelID.isEmpty else {
            return
        }
        guard !channelDeliveryPolicySaveInFlightIDs.contains(normalizedChannelID) else {
            return
        }
        guard let authToken = resolvedAuthToken() else {
            channelDeliveryPolicyActionStatusByID[normalizedChannelID] = "Set Assistant Access Token before saving delivery policy."
            return
        }

        var draft = channelDeliveryPolicyDraft(channelID: normalizedChannelID)
        let primaryChannel = normalizedChannelDeliveryPolicyChannelID(
            nonEmpty(draft.primaryChannel) ?? normalizedChannelID
        )
        guard !primaryChannel.isEmpty else {
            channelDeliveryPolicyActionStatusByID[normalizedChannelID] = "Primary delivery channel is required."
            return
        }

        draft.primaryChannel = primaryChannel
        draft.retryCount = max(draft.retryCount, 0)
        channelDeliveryPolicyDraftByID[normalizedChannelID] = draft
        let fallbackChannels = Self.canonicalLogicalChannelIDs(
            parseChannelFallbackChannels(draft.fallbackChannelsText)
        )
            .filter { $0 != primaryChannel }

        channelDeliveryPolicySaveInFlightIDs.insert(normalizedChannelID)
        channelDeliveryPolicyActionStatusByID[normalizedChannelID] = "Saving delivery policy…"
        defer {
            channelDeliveryPolicySaveInFlightIDs.remove(normalizedChannelID)
        }

        var shouldRefresh = false
        do {
            let response = try await daemonClient.communications.commPolicySet(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                policyID: nonEmpty(draft.policyID),
                sourceChannel: normalizedChannelID,
                endpointPattern: nonEmpty(draft.endpointPattern),
                primaryChannel: primaryChannel,
                retryCount: max(draft.retryCount, 0),
                fallbackChannels: fallbackChannels,
                isDefault: draft.isDefault
            )
            applyChannelDeliveryPolicyRecord(response)
            channelDeliveryPolicyActionStatusByID[normalizedChannelID] =
                "Saved delivery policy at \(formattedWorkflowTimestamp(response.updatedAt))."
            channelsStatusMessage = "Saved delivery policy for \(normalizedChannelID)."
            connectionStatus = .connected
            shouldRefresh = true
        } catch {
            channelDeliveryPolicyActionStatusByID[normalizedChannelID] = daemonErrorMessage(
                error,
                fallbackContext: "Delivery policy save failed",
                updateConnectionStatus: false,
                panelContext: .channels
            )
        }

        if shouldRefresh {
            await fetchChannelCards()
        }
    }

    private func performConnectorHealthCheck(connectorID: String) async {
        let normalizedConnectorID = connectorID.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        guard !normalizedConnectorID.isEmpty else {
            return
        }
        guard !connectorTestInFlightIDs.contains(normalizedConnectorID) else {
            return
        }
        guard let authToken = resolvedAuthToken() else {
            connectorConfigActionStatusByID[normalizedConnectorID] = "Set Assistant Access Token before running connector health check."
            return
        }

        connectorTestInFlightIDs.insert(normalizedConnectorID)
        connectorConfigActionStatusByID[normalizedConnectorID] = "Running connector health check…"
        defer {
            connectorTestInFlightIDs.remove(normalizedConnectorID)
        }

        var shouldRefresh = false
        do {
            let response = try await daemonClient.connectors.connectorTestOperation(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                connectorID: normalizedConnectorID,
                operation: "health"
            )
            let result = connectionConfigStore.mapConfigurationTestResult(
                from: response,
                formattedWorkflowTimestamp: formattedWorkflowTimestamp(_:)
            )
            connectorLastTestResultByID[normalizedConnectorID] = result
            connectorConfigActionStatusByID[normalizedConnectorID] = "Health check (\(result.status)): \(result.summary)"
            connectorsStatusMessage = "Connector health check completed for \(normalizedConnectorID)."
            connectionStatus = .connected
            shouldRefresh = true
        } catch {
            connectorConfigActionStatusByID[normalizedConnectorID] = daemonErrorMessage(
                error,
                fallbackContext: "Connector health check failed",
                updateConnectionStatus: false,
                panelContext: .connectors
            )
        }

        if shouldRefresh {
            await fetchConnectorCards()
        }
    }

    private func performProviderSetupSave(providerID: String) async {
        let normalizedProvider = normalizedProviderID(providerID)
        guard Self.canonicalProviderOrder.contains(normalizedProvider) else {
            return
        }
        guard !providerSetupInFlightIDs.contains(normalizedProvider) else {
            return
        }
        guard let authToken = resolvedAuthToken() else {
            providerSetupStatusByID[normalizedProvider] = "Set Assistant Access Token before saving provider setup."
            return
        }

        let endpoint = nonEmpty(providerEndpointDraftByID[normalizedProvider])
            ?? Self.providerDefaultEndpoints[normalizedProvider]
        guard let endpoint else {
            providerSetupStatusByID[normalizedProvider] = "Endpoint is required for \(providerDisplayName(normalizedProvider))."
            return
        }

        let requiresAPIKey = providerRequiresAPIKey(normalizedProvider)
        let defaultSecretName = defaultProviderSecretName(for: normalizedProvider)
        let secretName = nonEmpty(providerAPIKeySecretNameDraftByID[normalizedProvider])
            ?? (requiresAPIKey ? defaultSecretName : nil)
        let secretValue = nonEmpty(providerAPIKeySecretValueDraftByID[normalizedProvider])

        if requiresAPIKey && secretName == nil {
            providerSetupStatusByID[normalizedProvider] = "API key secret name is required for \(providerDisplayName(normalizedProvider))."
            return
        }

        providerSetupInFlightIDs.insert(normalizedProvider)
        defer {
            providerSetupInFlightIDs.remove(normalizedProvider)
        }

        if let secretValue, let secretName {
            let secretReference = providerSecretReferenceMetadata(secretName: secretName)
            providerSetupStatusByID[normalizedProvider] = "Saving API key in local keychain…"
            do {
                try LocalSecretStore.upsertSecret(
                    value: secretValue,
                    service: secretReference.service,
                    account: secretReference.account
                )
            } catch {
                providerSetupStatusByID[normalizedProvider] = "Failed to save API key locally: \(error.localizedDescription)"
                return
            }

            providerSetupStatusByID[normalizedProvider] = "Registering secret reference with daemon…"
            do {
                _ = try await daemonClient.models.secretReferenceUpsert(
                    baseURL: daemonBaseURL,
                    authToken: authToken,
                    workspaceID: secretReference.workspaceID,
                    name: secretReference.name,
                    backend: secretReference.backend,
                    service: secretReference.service,
                    account: secretReference.account
                )
            } catch {
                providerSetupStatusByID[normalizedProvider] = daemonErrorMessage(
                    error,
                    fallbackContext: "Secret reference registration failed",
                    updateConnectionStatus: false,
                    panelContext: .models
                )
                return
            }
        }

        providerSetupStatusByID[normalizedProvider] = "Saving provider configuration…"
        do {
            let response = try await daemonClient.models.providerSet(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                provider: normalizedProvider,
                endpoint: endpoint,
                apiKeySecretName: secretName,
                clearAPIKey: false
            )
            providerEndpointDraftByID[normalizedProvider] = response.endpoint
            providerEndpointSourceByID[normalizedProvider] = response.endpoint
            if providerRequiresAPIKey(normalizedProvider) {
                let resolvedSecretName = nonEmpty(response.apiKeySecretName)
                    ?? defaultProviderSecretName(for: normalizedProvider)
                providerAPIKeySecretNameDraftByID[normalizedProvider] = resolvedSecretName
                providerSecretNameSourceByID[normalizedProvider] = resolvedSecretName
            } else {
                let resolvedSecretName = nonEmpty(response.apiKeySecretName) ?? ""
                providerAPIKeySecretNameDraftByID[normalizedProvider] = resolvedSecretName
                providerSecretNameSourceByID[normalizedProvider] = resolvedSecretName
            }
            providerAPIKeySecretValueDraftByID[normalizedProvider] = ""
            providerSetupStatusByID[normalizedProvider] = "Saved \(providerDisplayName(normalizedProvider)) provider settings."
            connectionStatus = .connected
            await fetchProviderAndModelStatus(runChecks: false)
        } catch {
            providerSetupStatusByID[normalizedProvider] = daemonErrorMessage(
                error,
                fallbackContext: "Provider setup save failed",
                updateConnectionStatus: false,
                panelContext: .models
            )
        }
    }

    private func performProviderConnectivityCheck(providerID: String) async {
        let normalizedProvider = normalizedProviderID(providerID)
        guard Self.canonicalProviderOrder.contains(normalizedProvider) else {
            return
        }
        guard !providerCheckInFlightIDs.contains(normalizedProvider) else {
            return
        }
        guard let authToken = resolvedAuthToken() else {
            providerSetupStatusByID[normalizedProvider] = "Set Assistant Access Token before running provider checks."
            return
        }

        providerCheckInFlightIDs.insert(normalizedProvider)
        defer {
            providerCheckInFlightIDs.remove(normalizedProvider)
        }

        do {
            let response = try await daemonClient.models.providerCheck(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                provider: normalizedProvider
            )
            updateWorkspaceContext(from: response.workspaceID)

            guard let checkItem = response.results.first(where: {
                $0.provider.caseInsensitiveCompare(normalizedProvider) == .orderedSame
            }) else {
                providerSetupStatusByID[normalizedProvider] = "No check result returned for \(providerDisplayName(normalizedProvider))."
                return
            }

            applyProviderCheckResult(providerID: normalizedProvider, checkItem: checkItem)
            providerStatusMessage = checkItem.success
                ? "\(providerDisplayName(normalizedProvider)) check passed."
                : "\(providerDisplayName(normalizedProvider)) check failed."
            providerSetupStatusByID[normalizedProvider] = checkItem.message
            connectionStatus = .connected
        } catch {
            providerSetupStatusByID[normalizedProvider] = daemonErrorMessage(
                error,
                fallbackContext: "Provider check failed",
                updateConnectionStatus: false,
                panelContext: .models
            )
        }
    }

    private func applyProviderCheckResult(providerID: String, checkItem: DaemonProviderCheckItem) {
        guard let index = providerReadinessItems.firstIndex(where: {
            $0.provider.caseInsensitiveCompare(providerID) == .orderedSame
        }) else {
            return
        }

        let current = providerReadinessItems[index]
        let updatedStatus: ProviderReadinessStatus = checkItem.success ? .healthy : .checkFailed
        let updatedDetail = nonEmpty(checkItem.message)
            ?? (checkItem.success ? "Provider check succeeded." : "Provider check failed.")

        providerReadinessItems[index] = ProviderReadinessItem(
            id: current.id,
            provider: current.provider,
            endpoint: current.endpoint,
            status: updatedStatus,
            detail: updatedDetail,
            updatedAtLabel: current.updatedAtLabel
        )
    }

    private func performModelDiscover(providerID: String) async {
        let normalizedProvider = normalizedProviderID(providerID)
        guard !normalizedProvider.isEmpty else {
            return
        }
        guard !modelCatalogDiscoverInFlightProviderIDs.contains(normalizedProvider) else {
            return
        }
        guard let authToken = resolvedAuthToken() else {
            modelCatalogManagementStatusByProviderID[normalizedProvider] = "Set Assistant Access Token before discovering models."
            return
        }

        modelCatalogDiscoverInFlightProviderIDs.insert(normalizedProvider)
        modelCatalogManagementStatusByProviderID[normalizedProvider] = "Discovering provider-backed models…"
        defer {
            modelCatalogDiscoverInFlightProviderIDs.remove(normalizedProvider)
        }

        do {
            let response = try await daemonClient.models.modelDiscover(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                provider: normalizedProvider
            )
            updateWorkspaceContext(from: response.workspaceID)

            let result = response.results.first {
                $0.provider.caseInsensitiveCompare(normalizedProvider) == .orderedSame
            }

            guard let result else {
                discoveredModelsByProviderID[normalizedProvider] = []
                modelCatalogManagementStatusByProviderID[normalizedProvider] = "No discover result returned for \(providerDisplayName(normalizedProvider))."
                connectionStatus = .connected
                return
            }

            discoveredModelsByProviderID[normalizedProvider] = result.models
                .map(mapDiscoveredModelRecord)
                .sorted { lhs, rhs in
                    lhs.modelKey.localizedCaseInsensitiveCompare(rhs.modelKey) == .orderedAscending
                }
            syncDiscoveredModelCatalogFlags(providerID: normalizedProvider)
            let details = nonEmpty(result.message)
            let readinessLabel = result.providerReady ? "ready" : "not ready"
            modelCatalogManagementStatusByProviderID[normalizedProvider] = result.success
                ? "Discovered \(result.models.count) model(s) for \(providerDisplayName(normalizedProvider)) (\(readinessLabel))."
                : "Discovery failed for \(providerDisplayName(normalizedProvider)): \(details ?? "provider unavailable")"
            connectionStatus = .connected
        } catch {
            modelCatalogManagementStatusByProviderID[normalizedProvider] = daemonErrorMessage(
                error,
                fallbackContext: "Model discovery failed",
                updateConnectionStatus: false,
                panelContext: .models
            )
        }
    }

    private func performModelCatalogAdd(
        providerID: String,
        modelKey: String,
        enabled: Bool
    ) async {
        let normalizedProvider = normalizedProviderID(providerID)
        let normalizedModelKey = modelKey.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !normalizedProvider.isEmpty else {
            return
        }
        guard !normalizedModelKey.isEmpty else {
            modelCatalogManagementStatusByProviderID[normalizedProvider] = "Model key is required."
            return
        }
        guard !modelCatalogManageInFlightProviderIDs.contains(normalizedProvider) else {
            return
        }
        guard let authToken = resolvedAuthToken() else {
            modelCatalogManagementStatusByProviderID[normalizedProvider] = "Set Assistant Access Token before adding models."
            return
        }

        modelCatalogManageInFlightProviderIDs.insert(normalizedProvider)
        defer {
            modelCatalogManageInFlightProviderIDs.remove(normalizedProvider)
        }

        do {
            let response = try await daemonClient.models.modelAdd(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                provider: normalizedProvider,
                modelKey: normalizedModelKey,
                enabled: enabled
            )
            updateWorkspaceContext(from: response.workspaceID)
            modelManualAddDraftByProviderID[normalizedProvider] = ""
            modelCatalogManagementStatusByProviderID[normalizedProvider] = "Added \(normalizedModelKey) to \(providerDisplayName(normalizedProvider))."
            connectionStatus = .connected
            await fetchModelCatalogAndPolicies(authToken: authToken)
            await refreshModelRouteSummary(authToken: authToken)
        } catch {
            modelCatalogManagementStatusByProviderID[normalizedProvider] = daemonErrorMessage(
                error,
                fallbackContext: "Model add failed",
                updateConnectionStatus: false,
                panelContext: .models
            )
        }
    }

    private func performModelCatalogRemove(providerID: String, modelKey: String) async {
        let normalizedProvider = normalizedProviderID(providerID)
        let normalizedModelKey = modelKey.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !normalizedProvider.isEmpty else {
            return
        }
        guard !normalizedModelKey.isEmpty else {
            modelCatalogManagementStatusByProviderID[normalizedProvider] = "Model key is required."
            return
        }
        guard !modelCatalogManageInFlightProviderIDs.contains(normalizedProvider) else {
            return
        }
        guard let authToken = resolvedAuthToken() else {
            modelCatalogManagementStatusByProviderID[normalizedProvider] = "Set Assistant Access Token before removing models."
            return
        }

        modelCatalogManageInFlightProviderIDs.insert(normalizedProvider)
        defer {
            modelCatalogManageInFlightProviderIDs.remove(normalizedProvider)
        }

        do {
            let response = try await daemonClient.models.modelRemove(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                provider: normalizedProvider,
                modelKey: normalizedModelKey
            )
            updateWorkspaceContext(from: response.workspaceID)
            modelCatalogManagementStatusByProviderID[normalizedProvider] = response.removed
                ? "Removed \(normalizedModelKey) from \(providerDisplayName(normalizedProvider))."
                : "\(normalizedModelKey) was already removed from \(providerDisplayName(normalizedProvider))."
            connectionStatus = .connected
            await fetchModelCatalogAndPolicies(authToken: authToken)
            await refreshModelRouteSummary(authToken: authToken)
        } catch {
            modelCatalogManagementStatusByProviderID[normalizedProvider] = daemonErrorMessage(
                error,
                fallbackContext: "Model remove failed",
                updateConnectionStatus: false,
                panelContext: .models
            )
        }
    }

    private func performModelEnabledMutation(
        providerID: String,
        modelKey: String,
        enabled: Bool
    ) async {
        let normalizedProvider = normalizedProviderID(providerID)
        let normalizedModelKey = modelKey.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !normalizedModelKey.isEmpty else {
            return
        }

        let modelID = modelCatalogIdentifier(providerID: normalizedProvider, modelKey: normalizedModelKey)
        guard !modelMutationInFlightIDs.contains(modelID) else {
            return
        }
        guard let authToken = resolvedAuthToken() else {
            modelMutationStatusByID[modelID] = "Set Assistant Access Token before changing model enablement."
            return
        }

        modelMutationInFlightIDs.insert(modelID)
        defer {
            modelMutationInFlightIDs.remove(modelID)
        }

        do {
            let response: DaemonModelCatalogEntryRecord
            if enabled {
                response = try await daemonClient.models.modelEnable(
                    baseURL: daemonBaseURL,
                    authToken: authToken,
                    workspaceID: workspaceID,
                    provider: normalizedProvider,
                    modelKey: normalizedModelKey
                )
            } else {
                response = try await daemonClient.models.modelDisable(
                    baseURL: daemonBaseURL,
                    authToken: authToken,
                    workspaceID: workspaceID,
                    provider: normalizedProvider,
                    modelKey: normalizedModelKey
                )
            }

            updateWorkspaceContext(from: response.workspaceID)

            if let index = modelCatalogItems.firstIndex(where: { $0.id == modelID }) {
                var item = modelCatalogItems[index]
                item = ModelCatalogEntryItem(
                    id: item.id,
                    provider: item.provider,
                    modelKey: item.modelKey,
                    enabled: response.enabled,
                    providerReady: item.providerReady,
                    providerEndpoint: item.providerEndpoint
                )
                modelCatalogItems[index] = item
            } else {
                await fetchModelCatalogAndPolicies(authToken: authToken)
            }

            modelMutationStatusByID[modelID] = response.enabled
                ? "\(normalizedModelKey) enabled."
                : "\(normalizedModelKey) disabled."
            syncDiscoveredModelCatalogFlags(providerID: normalizedProvider)
            modelCatalogStatusMessage = "Model catalog updated for \(providerDisplayName(normalizedProvider))."
            connectionStatus = .connected
            await refreshModelRouteSummary(authToken: authToken)
        } catch {
            modelMutationStatusByID[modelID] = daemonErrorMessage(
                error,
                fallbackContext: "Model toggle failed",
                updateConnectionStatus: false,
                panelContext: .models
            )
        }
    }

    private func performSetModelAsChatRoute(providerID: String, modelKey: String) async {
        let normalizedProvider = normalizedProviderID(providerID)
        let normalizedModelKey = modelKey.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !normalizedProvider.isEmpty else {
            modelRoutePolicySaveStatusMessage = "Provider is required."
            return
        }
        guard !normalizedModelKey.isEmpty else {
            modelRoutePolicySaveStatusMessage = "Model key is required."
            return
        }
        guard let authToken = resolvedAuthToken() else {
            modelRoutePolicySaveStatusMessage = "Set Assistant Access Token before setting chat route."
            return
        }

        if let entry = modelCatalogItems.first(where: {
            $0.provider.caseInsensitiveCompare(normalizedProvider) == .orderedSame &&
                $0.modelKey.caseInsensitiveCompare(normalizedModelKey) == .orderedSame
        }), !entry.enabled {
            await performModelEnabledMutation(
                providerID: normalizedProvider,
                modelKey: normalizedModelKey,
                enabled: true
            )
        }

        modelRoutePolicySaveStatusMessage =
            "Setting chat route to \(providerDisplayName(normalizedProvider)) • \(normalizedModelKey)…"
        await performModelRoutePolicySave(
            taskClass: "chat",
            providerID: normalizedProvider,
            modelKey: normalizedModelKey
        )
        await refreshModelRouteSummary(authToken: authToken)
    }

    private func performModelRoutePolicySave(
        taskClass: String,
        providerID: String,
        modelKey: String
    ) async {
        let normalizedTaskClass = taskClass.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        let normalizedProvider = normalizedProviderID(providerID)
        let normalizedModelKey = modelKey.trimmingCharacters(in: .whitespacesAndNewlines)

        guard !normalizedTaskClass.isEmpty else {
            modelRoutePolicySaveStatusMessage = "Task class is required."
            return
        }
        guard !normalizedProvider.isEmpty else {
            modelRoutePolicySaveStatusMessage = "Provider is required."
            return
        }
        guard !normalizedModelKey.isEmpty else {
            modelRoutePolicySaveStatusMessage = "Model key is required."
            return
        }
        guard !isModelRoutePolicySaveInFlight else {
            return
        }
        guard let authToken = resolvedAuthToken() else {
            modelRoutePolicySaveStatusMessage = "Set Assistant Access Token before saving route policy."
            return
        }

        isModelRoutePolicySaveInFlight = true
        defer {
            isModelRoutePolicySaveInFlight = false
        }

        do {
            let response = try await daemonClient.models.modelSelect(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                taskClass: normalizedTaskClass,
                provider: normalizedProvider,
                modelKey: normalizedModelKey
            )
            updateWorkspaceContext(from: response.workspaceID)

            let policyItem = mapModelPolicyRecord(response)
            if let index = modelPolicyItems.firstIndex(where: {
                $0.taskClass.caseInsensitiveCompare(normalizedTaskClass) == .orderedSame
            }) {
                modelPolicyItems[index] = policyItem
            } else {
                modelPolicyItems.append(policyItem)
            }
            modelPolicyItems.sort { lhs, rhs in
                lhs.taskClass.localizedCaseInsensitiveCompare(rhs.taskClass) == .orderedAscending
            }

            modelRoutePolicySaveStatusMessage = "Saved \(normalizedTaskClass) route to \(providerDisplayName(normalizedProvider)) • \(normalizedModelKey)."
            connectionStatus = .connected
            await refreshModelRouteSummary(authToken: authToken)
        } catch {
            modelRoutePolicySaveStatusMessage = daemonErrorMessage(
                error,
                fallbackContext: "Model route policy save failed",
                updateConnectionStatus: false,
                panelContext: .models
            )
        }
    }

    private func performModelRouteSimulation() async {
        guard !isModelRouteSimulationInFlight else {
            return
        }

        let normalizedTaskClass = modelRouteSimulationTaskClass
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .lowercased()
        guard !normalizedTaskClass.isEmpty else {
            modelRouteSimulationResult = nil
            modelRouteSimulationStatusMessage = "Task class is required for route simulation."
            return
        }
        if modelRouteSimulationTaskClass.caseInsensitiveCompare(normalizedTaskClass) != .orderedSame {
            modelRouteSimulationTaskClass = normalizedTaskClass
        }

        guard let authToken = resolvedAuthToken() else {
            modelRouteSimulationResult = nil
            modelRouteSimulationStatusMessage = "Set Assistant Access Token before running route simulation."
            return
        }

        isModelRouteSimulationInFlight = true
        modelRouteSimulationStatusMessage = "Running route simulation…"
        defer {
            isModelRouteSimulationInFlight = false
        }

        do {
            let response = try await daemonClient.models.modelRouteSimulate(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                taskClass: normalizedTaskClass,
                principalActorID: nonEmpty(modelRouteSimulationPrincipalActorID)
            )
            updateWorkspaceContext(from: response.workspaceID)
            let mapped = mapModelRouteSimulationResponse(response)
            modelRouteSimulationResult = mapped
            modelRouteSimulationStatusMessage = modelRouteSimulationSummaryMessage(mapped)
            connectionStatus = .connected
        } catch {
            modelRouteSimulationResult = nil
            modelRouteSimulationStatusMessage = daemonErrorMessage(
                error,
                fallbackContext: "Model route simulation failed",
                updateConnectionStatus: false,
                panelContext: .models
            )
        }
    }

    private func performModelRouteExplainability() async {
        guard !isModelRouteExplainInFlight else {
            return
        }

        let normalizedTaskClass = modelRouteSimulationTaskClass
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .lowercased()
        guard !normalizedTaskClass.isEmpty else {
            modelRouteExplainResult = nil
            modelRouteExplainStatusMessage = "Task class is required for route explainability."
            return
        }
        if modelRouteSimulationTaskClass.caseInsensitiveCompare(normalizedTaskClass) != .orderedSame {
            modelRouteSimulationTaskClass = normalizedTaskClass
        }

        guard let authToken = resolvedAuthToken() else {
            modelRouteExplainResult = nil
            modelRouteExplainStatusMessage = "Set Assistant Access Token before running route explainability."
            return
        }

        isModelRouteExplainInFlight = true
        modelRouteExplainStatusMessage = "Loading route explainability trace…"
        defer {
            isModelRouteExplainInFlight = false
        }

        do {
            let response = try await daemonClient.models.modelRouteExplain(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                taskClass: normalizedTaskClass,
                principalActorID: nonEmpty(modelRouteSimulationPrincipalActorID)
            )
            updateWorkspaceContext(from: response.workspaceID)
            let mapped = mapModelRouteExplainResponse(response)
            modelRouteExplainResult = mapped
            modelRouteExplainStatusMessage = modelRouteExplainSummaryMessage(mapped)
            connectionStatus = .connected
        } catch {
            modelRouteExplainResult = nil
            modelRouteExplainStatusMessage = daemonErrorMessage(
                error,
                fallbackContext: "Model route explainability failed",
                updateConnectionStatus: false,
                panelContext: .models
            )
        }
    }

    private func syncProviderSetupDrafts(from configuredByProvider: [String: DaemonProviderConfigRecord]) {
        modelsRouteStore.syncProviderSetupDrafts(
            configuredByProvider: configuredByProvider,
            canonicalProviderOrder: Self.canonicalProviderOrder,
            providerDefaultEndpoints: Self.providerDefaultEndpoints,
            providerRequiresAPIKey: providerRequiresAPIKey(_:),
            defaultProviderSecretName: defaultProviderSecretName(for:)
        )
    }

    private func resetProviderSetupDraftsToDefaults() {
        modelsRouteStore.resetProviderSetupDraftsToDefaults(
            defaultProviderEndpointDrafts: Self.defaultProviderEndpointDrafts(),
            defaultProviderSecretNameDrafts: Self.defaultProviderSecretNameDrafts()
        )
    }

    private func resetModelCatalogManagementState() {
        modelsRouteStore.resetModelCatalogManagementState()
    }

    private func fetchProviderAndModelStatus(runChecks: Bool) async {
        isProviderStatusLoading = true
        clearPanelProblemSignal(for: .models)
        defer {
            isProviderStatusLoading = false
            hasLoadedProviderStatus = true
            updateOnboardingCompletionState()
        }

        guard let authToken = resolvedAuthToken() else {
            providerStatusMessage = "Set Assistant Access Token to load provider inventory."
            providerReadinessItems = Self.defaultProviderReadinessItems()
            resetProviderSetupDraftsToDefaults()
            modelCatalogDiscoverInFlightProviderIDs.removeAll()
            modelCatalogManageInFlightProviderIDs.removeAll()
            discoveredModelsByProviderID.removeAll()
            modelManualAddDraftByProviderID.removeAll()
            modelRouteSummary = nil
            modelRouteStatusMessage = "Set Assistant Access Token to resolve model route."
            modelCatalogItems = []
            modelPolicyItems = []
            modelCatalogStatusMessage = "Set Assistant Access Token to load model catalog."
            modelRouteSimulationResult = nil
            modelRouteExplainResult = nil
            modelRouteSimulationStatusMessage = "Set Assistant Access Token before running route simulation."
            modelRouteExplainStatusMessage = "Set Assistant Access Token before running route explainability."
            return
        }

        do {
            let listResponse = try await daemonClient.models.providerList(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID
            )
            updateWorkspaceContext(from: listResponse.workspaceID)

            let configuredByProvider = Dictionary(
                uniqueKeysWithValues: listResponse.providers.map { ($0.provider.lowercased(), $0) }
            )
            syncProviderSetupDrafts(from: configuredByProvider)

            var checkByProvider: [String: DaemonProviderCheckItem] = [:]
            if runChecks {
                do {
                    let checkResponse = try await daemonClient.models.providerCheck(
                        baseURL: daemonBaseURL,
                        authToken: authToken,
                        workspaceID: workspaceID
                    )
                    updateWorkspaceContext(from: checkResponse.workspaceID)
                    checkByProvider = Dictionary(
                        uniqueKeysWithValues: checkResponse.results.map { ($0.provider.lowercased(), $0) }
                    )
                    providerStatusMessage = checkResponse.success
                        ? "All configured provider checks passed."
                        : "Provider checks completed with one or more failures."
                    connectionStatus = .connected
                } catch {
                    providerStatusMessage = daemonErrorMessage(
                        error,
                        fallbackContext: "Provider checks failed",
                        updateConnectionStatus: false,
                        panelContext: .models
                    )
                }
            } else {
                providerStatusMessage = "Workspace \(listResponse.workspaceID) • Provider inventory refreshed."
            }

            providerReadinessItems = Self.canonicalProviderOrder.map { providerID in
                mapProviderReadinessItem(
                    providerID: providerID,
                    config: configuredByProvider[providerID],
                    check: checkByProvider[providerID]
                )
            }

            await refreshModelRouteSummary(authToken: authToken)
            await fetchModelCatalogAndPolicies(authToken: authToken)
        } catch {
            providerStatusMessage = daemonErrorMessage(
                error,
                fallbackContext: "Provider inventory query failed",
                updateConnectionStatus: false,
                panelContext: .models
            )
            providerReadinessItems = Self.defaultProviderReadinessItems()
            resetProviderSetupDraftsToDefaults()
            resetModelCatalogManagementState()
            modelRouteSummary = nil
            modelRouteStatusMessage = "Route unavailable until provider inventory is available."
            modelCatalogItems = []
            modelPolicyItems = []
            modelCatalogStatusMessage = "Model catalog unavailable until provider inventory is available."
        }
    }

    private func fetchAutomationPanelData() async {
        clearPanelProblemSignal(for: .automation)
        defer {
            hasLoadedAutomationPanelData = true
        }
        async let triggerRefresh: Void = fetchAutomationTriggers()
        async let fireHistoryRefresh: Void = fetchAutomationFireHistory()
        _ = await (triggerRefresh, fireHistoryRefresh)
    }

    private func fetchAutomationTriggers() async {
        isAutomationLoading = true
        defer {
            isAutomationLoading = false
        }

        guard let authToken = resolvedAuthToken() else {
            automationStatusMessage = "Set Assistant Access Token to query automation triggers."
            return
        }

        do {
            let response = try await daemonClient.automation.automationList(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                includeDisabled: true
            )
            updateWorkspaceContext(from: response.workspaceID)
            automationTriggers = response.triggers.map(mapAutomationTriggerRecord)
            let activeTriggerIDs = Set(automationTriggers.map(\.id))
            workflowQueueStore.pruneAutomationActionStatus(validTriggerIDs: activeTriggerIDs)
            automationStatusMessage = automationTriggers.isEmpty
                ? "No automation triggers returned for workspace \(response.workspaceID)."
                : "Workspace \(response.workspaceID) • \(automationTriggers.count) trigger(s) loaded."
            connectionStatus = .connected
        } catch {
            automationStatusMessage = daemonErrorMessage(
                error,
                fallbackContext: "Automation list query failed",
                updateConnectionStatus: false,
                panelContext: .automation
            )
        }
    }

    private func fetchAutomationFireHistory() async {
        isAutomationFireHistoryLoading = true
        defer {
            isAutomationFireHistoryLoading = false
        }

        guard let authToken = resolvedAuthToken() else {
            automationFireHistoryItems = []
            automationFireHistoryStatusMessage = "Set Assistant Access Token to query trigger fire history."
            return
        }

        do {
            let response = try await daemonClient.automation.automationFireHistory(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                limit: 60
            )
            updateWorkspaceContext(from: response.workspaceID)
            automationFireHistoryItems = response.fires
                .map(mapAutomationFireHistoryRecord)
                .sorted { lhs, rhs in
                    if lhs.sortTimestamp == rhs.sortTimestamp {
                        return lhs.id > rhs.id
                    }
                    return lhs.sortTimestamp > rhs.sortTimestamp
                }
            automationFireHistoryStatusMessage = automationFireHistoryItems.isEmpty
                ? "No trigger fire history returned for workspace \(response.workspaceID)."
                : "Workspace \(response.workspaceID) • \(automationFireHistoryItems.count) fire record(s) loaded."
            connectionStatus = .connected
        } catch {
            automationFireHistoryItems = []
            automationFireHistoryStatusMessage = daemonErrorMessage(
                error,
                fallbackContext: "Automation fire-history query failed",
                updateConnectionStatus: false,
                panelContext: .automation
            )
        }
    }

    private struct AutomationMutationPayload {
        let triggerType: String
        let subjectActorID: String
        let title: String
        let instruction: String
        let intervalSeconds: Int?
        let filterJSON: String?
        let cooldownSeconds: Int
        let enabled: Bool
    }

    private struct AutomationMutationValidationError: Error {
        let message: String
    }

    private func performAutomationCreate(_ input: AutomationTriggerMutationInput) async {
        let payload: AutomationMutationPayload
        do {
            payload = try resolvedAutomationMutationPayload(from: input)
        } catch let validationError as AutomationMutationValidationError {
            automationManagementStatusMessage = validationError.message
            return
        } catch {
            automationManagementStatusMessage = "Automation input validation failed."
            return
        }

        guard let authToken = resolvedAuthToken() else {
            automationManagementStatusMessage = "Set Assistant Access Token before creating automation triggers."
            return
        }

        guard workflowQueueStore.beginAutomationCreate() else {
            return
        }
        defer {
            workflowQueueStore.endAutomationCreate()
        }

        do {
            let created = try await daemonClient.automation.automationCreate(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                subjectActorID: payload.subjectActorID,
                triggerType: payload.triggerType,
                title: payload.title,
                instruction: payload.instruction,
                intervalSeconds: payload.intervalSeconds,
                filterJSON: payload.filterJSON,
                cooldownSeconds: payload.cooldownSeconds,
                enabled: payload.enabled
            )
            automationManagementStatusMessage = "Created \(automationTriggerTypeLabel(payload.triggerType)) trigger \(created.triggerID)."
            workflowQueueStore.setAutomationActionStatus(triggerID: created.triggerID, message: "Created from app.")
            connectionStatus = .connected
            await fetchAutomationPanelData()
        } catch {
            automationManagementStatusMessage = daemonErrorMessage(
                error,
                fallbackContext: "Automation create failed",
                updateConnectionStatus: false,
                panelContext: .automation
            )
        }
    }

    private func performAutomationUpdate(triggerID: String, input: AutomationTriggerMutationInput) async {
        let trimmedTriggerID = triggerID.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmedTriggerID.isEmpty else {
            return
        }

        let payload: AutomationMutationPayload
        do {
            payload = try resolvedAutomationMutationPayload(from: input)
        } catch let validationError as AutomationMutationValidationError {
            workflowQueueStore.setAutomationActionStatus(triggerID: trimmedTriggerID, message: validationError.message)
            automationManagementStatusMessage = validationError.message
            return
        } catch {
            workflowQueueStore.setAutomationActionStatus(
                triggerID: trimmedTriggerID,
                message: "Automation input validation failed."
            )
            automationManagementStatusMessage = "Automation input validation failed."
            return
        }

        guard let authToken = resolvedAuthToken() else {
            let message = "Set Assistant Access Token before updating automation triggers."
            workflowQueueStore.setAutomationActionStatus(triggerID: trimmedTriggerID, message: message)
            automationManagementStatusMessage = message
            return
        }

        guard let queuedTriggerID = workflowQueueStore.beginAutomationUpdate(triggerID: trimmedTriggerID) else {
            return
        }
        defer {
            workflowQueueStore.endAutomationUpdate(triggerID: queuedTriggerID)
        }

        do {
            let response = try await daemonClient.automation.automationUpdate(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                triggerID: queuedTriggerID,
                subjectActorID: payload.subjectActorID,
                title: payload.title,
                instruction: payload.instruction,
                intervalSeconds: payload.intervalSeconds,
                filterJSON: payload.filterJSON,
                cooldownSeconds: payload.cooldownSeconds,
                enabled: payload.enabled
            )
            let message = response.updated
                ? "Trigger updated and persisted."
                : "No changes detected; daemon reported an idempotent update."
            workflowQueueStore.setAutomationActionStatus(triggerID: queuedTriggerID, message: message)
            automationManagementStatusMessage = "\(message) (\(response.trigger.directiveTitle))"
            connectionStatus = .connected
            await fetchAutomationPanelData()
        } catch {
            let message = daemonErrorMessage(
                error,
                fallbackContext: "Automation update failed",
                updateConnectionStatus: false,
                panelContext: .automation
            )
            workflowQueueStore.setAutomationActionStatus(triggerID: queuedTriggerID, message: message)
            automationManagementStatusMessage = message
        }
    }

    private func performAutomationDelete(triggerID: String) async {
        let trimmedTriggerID = triggerID.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmedTriggerID.isEmpty else {
            return
        }

        guard let authToken = resolvedAuthToken() else {
            let message = "Set Assistant Access Token before deleting automation triggers."
            workflowQueueStore.setAutomationActionStatus(triggerID: trimmedTriggerID, message: message)
            automationManagementStatusMessage = message
            return
        }

        guard let queuedTriggerID = workflowQueueStore.beginAutomationDelete(triggerID: trimmedTriggerID) else {
            return
        }
        defer {
            workflowQueueStore.endAutomationDelete(triggerID: queuedTriggerID)
        }

        do {
            let response = try await daemonClient.automation.automationDelete(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                triggerID: queuedTriggerID
            )
            let message = response.deleted
                ? "Trigger deleted from daemon."
                : "Trigger already deleted (idempotent)."
            workflowQueueStore.setAutomationActionStatus(triggerID: queuedTriggerID, message: message)
            automationManagementStatusMessage = message
            connectionStatus = .connected
            await fetchAutomationPanelData()
        } catch {
            let message = daemonErrorMessage(
                error,
                fallbackContext: "Automation delete failed",
                updateConnectionStatus: false,
                panelContext: .automation
            )
            workflowQueueStore.setAutomationActionStatus(triggerID: queuedTriggerID, message: message)
            automationManagementStatusMessage = message
        }
    }

    private func resolvedAutomationMutationPayload(
        from input: AutomationTriggerMutationInput
    ) throws -> AutomationMutationPayload {
        let triggerType = input.triggerType.trimmingCharacters(in: .whitespacesAndNewlines).uppercased()
        guard triggerType == "SCHEDULE" || triggerType == "ON_COMM_EVENT" else {
            throw AutomationMutationValidationError(message: "Trigger type must be `SCHEDULE` or `ON_COMM_EVENT`.")
        }

        let subjectActor = input.subjectActorID.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !subjectActor.isEmpty else {
            throw AutomationMutationValidationError(message: "Subject actor is required.")
        }

        let title = input.title.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !title.isEmpty else {
            throw AutomationMutationValidationError(message: "Directive title is required.")
        }

        let instruction = input.instruction.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !instruction.isEmpty else {
            throw AutomationMutationValidationError(message: "Directive instruction is required.")
        }

        guard input.cooldownSeconds >= 0 else {
            throw AutomationMutationValidationError(message: "Cooldown seconds cannot be negative.")
        }

        if triggerType == "SCHEDULE" {
            guard let interval = input.scheduleIntervalSeconds, interval > 0 else {
                throw AutomationMutationValidationError(message: "Schedule interval must be greater than zero.")
            }
            return AutomationMutationPayload(
                triggerType: triggerType,
                subjectActorID: subjectActor,
                title: title,
                instruction: instruction,
                intervalSeconds: interval,
                filterJSON: nil,
                cooldownSeconds: input.cooldownSeconds,
                enabled: input.enabled
            )
        }

        let filterJSON = (input.commEventFilterJSON ?? "").trimmingCharacters(in: .whitespacesAndNewlines)
        let resolvedFilter = filterJSON.isEmpty ? "{}" : filterJSON
        guard isValidJSONText(resolvedFilter) else {
            throw AutomationMutationValidationError(message: "Comm-event filter JSON must be valid JSON.")
        }
        let canonicalizedFilter = canonicalizedAutomationCommEventFilterJSON(resolvedFilter)
        return AutomationMutationPayload(
            triggerType: triggerType,
            subjectActorID: subjectActor,
            title: title,
            instruction: instruction,
            intervalSeconds: nil,
            filterJSON: canonicalizedFilter,
            cooldownSeconds: input.cooldownSeconds,
            enabled: input.enabled
        )
    }

    func canonicalizedAutomationCommEventFilterJSON(_ raw: String) -> String {
        guard let data = raw.data(using: .utf8),
              var object = try? JSONSerialization.jsonObject(with: data) as? [String: Any] else {
            return raw
        }

        if let channelValue = object["channels"] {
            let rawChannels: [String]
            if let channels = channelValue as? [String] {
                rawChannels = channels
            } else if let channel = channelValue as? String {
                rawChannels = [channel]
            } else if let anyValues = channelValue as? [Any] {
                rawChannels = anyValues.compactMap { value in
                    if let text = value as? String {
                        return text
                    }
                    return nil
                }
            } else {
                rawChannels = []
            }
            object["channels"] = Self.canonicalLogicalChannelIDs(rawChannels)
        }

        guard let normalizedData = try? JSONSerialization.data(withJSONObject: object, options: [.sortedKeys]),
              let normalized = String(data: normalizedData, encoding: .utf8) else {
            return raw
        }
        return normalized
    }

    private func isValidJSONText(_ raw: String) -> Bool {
        guard let data = raw.data(using: .utf8) else {
            return false
        }
        return (try? JSONDecoder().decode(DaemonJSONValue.self, from: data)) != nil
    }

    private func automationTriggerTypeLabel(_ raw: String) -> String {
        switch raw.trimmingCharacters(in: .whitespacesAndNewlines).uppercased() {
        case "SCHEDULE":
            return "schedule"
        case "ON_COMM_EVENT":
            return "comm-event"
        default:
            return "automation"
        }
    }

    private func fetchApprovalsInbox() async {
        isApprovalsLoading = true
        clearPanelProblemSignal(for: .approvals)
        defer {
            isApprovalsLoading = false
            hasLoadedApprovalsInbox = true
        }

        guard let authToken = resolvedAuthToken() else {
            approvalsStatusMessage = "Set Assistant Access Token to query approval inbox."
            return
        }

        do {
            let response = try await daemonClient.approvals.approvalInbox(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                includeFinal: true,
                limit: 120
            )
            updateWorkspaceContext(from: response.workspaceID)
            approvalInboxItems = response.approvals.map(mapApprovalInboxRecord)
            let activeApprovalIDs = Set(approvalInboxItems.map(\.id))
            approvalEvidenceByID = approvalEvidenceByID.filter { activeApprovalIDs.contains($0.key) }
            approvalEvidenceStatusByID = approvalEvidenceStatusByID.filter { activeApprovalIDs.contains($0.key) }
            approvalEvidenceInFlightIDs = Set(approvalEvidenceInFlightIDs.filter { activeApprovalIDs.contains($0) })
            let pendingCount = approvalInboxItems.filter { $0.decisionState == .pending }.count
            let finalCount = approvalInboxItems.count - pendingCount
            approvalsStatusMessage = approvalInboxItems.isEmpty
                ? "No approvals returned for workspace \(response.workspaceID)."
                : "Workspace \(response.workspaceID) • Pending: \(pendingCount) • Finalized: \(finalCount)"
            connectionStatus = .connected
        } catch {
            approvalsStatusMessage = daemonErrorMessage(
                error,
                fallbackContext: "Approval inbox query failed",
                updateConnectionStatus: false,
                panelContext: .approvals
            )
        }
    }

    private func fetchApprovalEvidence(for item: ApprovalInboxItem, runID: String) async {
        let approvalID = item.id.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !approvalID.isEmpty else {
            return
        }

        guard let authToken = resolvedAuthToken() else {
            approvalEvidenceStatusByID[approvalID] = "Set Assistant Access Token to query approval evidence."
            return
        }

        approvalEvidenceInFlightIDs.insert(approvalID)
        defer {
            approvalEvidenceInFlightIDs.remove(approvalID)
        }

        do {
            let response = try await daemonClient.inspect.inspectRun(
                baseURL: daemonBaseURL,
                authToken: authToken,
                runID: runID
            )
            approvalEvidenceByID[approvalID] = mapApprovalEvidence(
                approvalID: approvalID,
                focusStepID: nonEmpty(item.stepID),
                response: response
            )
            approvalEvidenceStatusByID[approvalID] = "Evidence loaded from run \(runID)."
            connectionStatus = .connected
        } catch {
            approvalEvidenceStatusByID[approvalID] = daemonErrorMessage(
                error,
                fallbackContext: "Approval evidence query failed",
                updateConnectionStatus: false,
                panelContext: .approvals
            )
        }
    }

    private func mapApprovalEvidence(
        approvalID: String,
        focusStepID: String?,
        response: DaemonInspectRunResponse
    ) -> ApprovalEvidenceItem {
        let sortedSteps = response.steps.sorted { lhs, rhs in
            lhs.stepIndex < rhs.stepIndex
        }
        let focusedStep = focusStepID.flatMap { id in
            sortedSteps.first { $0.stepID == id }
        } ?? sortedSteps.first

        let sortedAudits = response.auditEntries.sorted { lhs, rhs in
            (parseDaemonTimestamp(lhs.createdAt) ?? .distantPast) > (parseDaemonTimestamp(rhs.createdAt) ?? .distantPast)
        }
        let scopedAudits: [DaemonInspectRunAuditEntry] = {
            guard let focusedStep else {
                return sortedAudits
            }
            let matched = sortedAudits.filter { nonEmpty($0.stepID) == focusedStep.stepID }
            return matched.isEmpty ? sortedAudits : matched
        }()

        let summaries = Self.approvalEvidenceInputOutputSummaries(from: scopedAudits)
        let stepInputSummary = summaries.input ?? "No step input payload was recorded for this step."
        let stepOutputSummary: String = summaries.output ?? {
            if let focusedStep, let lastError = nonEmpty(focusedStep.lastError) {
                return "Step error: \(lastError)"
            }
            return "No step output payload was recorded for this step."
        }()

        let artifacts = response.artifacts
            .sorted { lhs, rhs in
                (parseDaemonTimestamp(lhs.createdAt) ?? .distantPast) > (parseDaemonTimestamp(rhs.createdAt) ?? .distantPast)
            }
            .filter { artifact in
                guard let focusedStep else {
                    return true
                }
                let artifactStepID = nonEmpty(artifact.stepID)
                return artifactStepID == nil || artifactStepID == focusedStep.stepID
            }
            .map { artifact in
                ApprovalEvidenceArtifactItem(
                    id: artifact.artifactID,
                    type: nonEmpty(artifact.artifactType) ?? "unknown",
                    stepID: nonEmpty(artifact.stepID),
                    uri: nonEmpty(artifact.uri),
                    contentHash: nonEmpty(artifact.contentHash),
                    createdAtLabel: formattedWorkflowTimestamp(artifact.createdAt)
                )
            }

        let auditEntries = scopedAudits.prefix(8).map { entry in
            ApprovalEvidenceAuditItem(
                id: entry.auditID,
                eventType: nonEmpty(entry.eventType) ?? "unknown",
                createdAtLabel: formattedWorkflowTimestamp(entry.createdAt),
                payloadSummary: Self.approvalEvidencePayloadSummary(from: entry.payloadJSON)
            )
        }

        let step = focusedStep.map { step in
            ApprovalEvidenceStepItem(
                stepID: step.stepID,
                name: nonEmpty(step.name) ?? "Unnamed step",
                statusLabel: normalizedWorkflowStateLabel(step.status) ?? "Unknown",
                capability: nonEmpty(step.capabilityKey),
                interactionLevel: nonEmpty(step.interactionLevel),
                updatedAtLabel: formattedWorkflowTimestamp(step.updatedAt),
                inputSummary: stepInputSummary,
                outputSummary: stepOutputSummary,
                lastError: nonEmpty(step.lastError)
            )
        }

        return ApprovalEvidenceItem(
            id: approvalID,
            runID: response.run.runID,
            taskID: response.task.taskID,
            title: nonEmpty(response.task.title) ?? "Untitled task",
            updatedAtLabel: formattedWorkflowTimestamp(response.run.updatedAt),
            step: step,
            artifacts: artifacts,
            auditEntries: Array(auditEntries)
        )
    }

    private func fetchTaskRunList() async {
        isTasksLoading = true
        clearPanelProblemSignal(for: .tasks)
        defer {
            isTasksLoading = false
            hasLoadedTaskRunList = true
        }

        guard let authToken = resolvedAuthToken() else {
            tasksStatusMessage = "Set Assistant Access Token to query task/runs."
            return
        }

        do {
            let response = try await daemonClient.tasks.taskRunList(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                limit: 120
            )
            updateWorkspaceContext(from: response.workspaceID)
            taskRunItems = response.items
                .map(mapTaskRunListRecord)
                .sorted { lhs, rhs in
                    if lhs.sortTimestamp == rhs.sortTimestamp {
                        return lhs.id > rhs.id
                    }
                    return lhs.sortTimestamp > rhs.sortTimestamp
                }
            let validRunIDs = Set(taskRunItems.compactMap(\.runID))
            workflowQueueStore.pruneTaskRunControlState(validRunIDs: validRunIDs)

            tasksStatusMessage = taskRunItems.isEmpty
                ? "No task/runs returned for workspace \(response.workspaceID)."
                : "Workspace \(response.workspaceID) • \(taskRunItems.count) task/run row(s) loaded."
            connectionStatus = .connected
        } catch {
            tasksStatusMessage = daemonErrorMessage(
                error,
                fallbackContext: "Task/run list query failed",
                updateConnectionStatus: false,
                panelContext: .tasks
            )
        }
    }

    private func performTaskSubmit(
        title: String,
        description: String?,
        taskClass: String?,
        requestedByActorID: String,
        subjectPrincipalActorID: String
    ) async {
        let trimmedTitle = title.trimmingCharacters(in: .whitespacesAndNewlines)
        let trimmedDescription = nonEmpty(description)
        let trimmedTaskClass = nonEmpty(taskClass)?.lowercased()
        let trimmedRequestedBy = requestedByActorID.trimmingCharacters(in: .whitespacesAndNewlines)
        let trimmedSubjectPrincipal = subjectPrincipalActorID.trimmingCharacters(in: .whitespacesAndNewlines)

        guard !trimmedTitle.isEmpty else {
            taskSubmitStatusMessage = "Goal is required."
            return
        }
        guard !trimmedRequestedBy.isEmpty else {
            taskSubmitStatusMessage = "Requested By actor is required."
            return
        }
        guard !trimmedSubjectPrincipal.isEmpty else {
            taskSubmitStatusMessage = "Subject principal actor is required."
            return
        }

        let validPrincipalIDs = Set(taskSubmissionPrincipalOptions)
        guard validPrincipalIDs.contains(trimmedRequestedBy) else {
            taskSubmitStatusMessage = "Requested By actor `\(trimmedRequestedBy)` is not in the active workspace directory."
            return
        }
        guard validPrincipalIDs.contains(trimmedSubjectPrincipal) else {
            taskSubmitStatusMessage = "Subject principal actor `\(trimmedSubjectPrincipal)` is not in the active workspace directory."
            return
        }

        guard let authToken = resolvedAuthToken() else {
            taskSubmitStatusMessage = "Set Assistant Access Token before submitting tasks."
            return
        }

        guard workflowQueueStore.beginTaskSubmit() else {
            return
        }
        defer {
            workflowQueueStore.endTaskSubmit()
        }

        do {
            let response = try await daemonClient.tasks.taskSubmit(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                requestedByActorID: trimmedRequestedBy,
                subjectPrincipalActorID: trimmedSubjectPrincipal,
                title: trimmedTitle,
                description: trimmedDescription,
                taskClass: trimmedTaskClass
            )
            guard !response.taskID.isEmpty, !response.runID.isEmpty else {
                taskSubmitStatusMessage = "Daemon task submit response was missing task/run identifiers."
                return
            }
            latestTaskSubmissionReceipt = TaskSubmissionReceiptItem(
                taskID: response.taskID,
                runID: response.runID,
                state: response.state,
                correlationID: nonEmpty(response.correlationID),
                submittedAt: Date.now
            )
            taskSubmitStatusMessage = "Submitted task \(response.taskID) with run \(response.runID) (state \(response.state))."
            tasksStatusMessage = "Task submit accepted. Refreshing task list for workspace \(workspaceID)…"
            connectionStatus = .connected
            appendNotification(
                source: "tasks",
                action: "task_submit",
                message: "Submitted task \(response.taskID) • run \(response.runID).",
                level: .success
            )
            markHomeFirstSessionStepComplete(
                .createTask,
                source: "task_submit",
                completedAt: latestTaskSubmissionReceipt?.submittedAt ?? .now
            )
            await fetchTaskRunList()
        } catch {
            let message = daemonErrorMessage(
                error,
                fallbackContext: "Task submit failed",
                updateConnectionStatus: false,
                panelContext: .tasks
            )
            taskSubmitStatusMessage = message
            appendNotification(
                source: "tasks",
                action: "task_submit",
                message: message,
                level: .error
            )
        }
    }

    private func performTaskRunControl(
        _ action: TaskRunControlAction,
        taskID: String,
        runID: String
    ) async {
        guard workflowQueueStore.canStartTaskRunControl(runID: runID) else {
            return
        }
        guard let authToken = resolvedAuthToken() else {
            let message = "Set Assistant Access Token before running task controls."
            workflowQueueStore.setTaskRunControlStatus(runID: runID, message: message)
            tasksStatusMessage = message
            return
        }

        let inFlightMessage: String = {
            switch action {
            case .cancel:
                return isAdvancedInformationDensityEnabled
                    ? "Cancelling run \(runID)…"
                    : "Cancelling task…"
            case .retry:
                return isAdvancedInformationDensityEnabled
                    ? "Retrying run \(runID)…"
                    : "Retrying task…"
            case .requeue:
                return isAdvancedInformationDensityEnabled
                    ? "Requeueing run \(runID)…"
                    : "Requeueing task…"
            }
        }()
        workflowQueueStore.beginTaskRunControl(runID: runID, inFlightMessage: inFlightMessage)
        defer {
            workflowQueueStore.finishTaskRunControl(runID: runID)
        }

        do {
            let reason = "Triggered from Tasks panel (\(action.rawValue))."
            let statusMessage: String
            var updatedRunID: String = runID
            var updatedActions: TaskRunActionAvailabilityItem = .unavailable

            switch action {
            case .cancel:
                let response = try await daemonClient.tasks.taskCancel(
                    baseURL: daemonBaseURL,
                    authToken: authToken,
                    workspaceID: workspaceID,
                    taskID: taskID,
                    runID: runID,
                    reason: reason
                )
                updatedRunID = nonEmpty(response.runID) ?? runID
                if response.alreadyTerminal {
                    statusMessage = isAdvancedInformationDensityEnabled
                        ? "Run \(updatedRunID) is already terminal (\(response.runState))."
                        : "Task is already finished (\(response.runState))."
                } else if response.cancelled {
                    statusMessage = isAdvancedInformationDensityEnabled
                        ? "Cancelled run \(updatedRunID)."
                        : "Cancelled task."
                } else {
                    statusMessage = isAdvancedInformationDensityEnabled
                        ? "Cancel request processed for run \(updatedRunID)."
                        : "Cancel request processed."
                }
                updatedActions = taskRunActionAvailability(from: nil)

            case .retry:
                let response = try await daemonClient.tasks.taskRetry(
                    baseURL: daemonBaseURL,
                    authToken: authToken,
                    workspaceID: workspaceID,
                    taskID: taskID,
                    runID: runID,
                    reason: reason
                )
                updatedRunID = nonEmpty(response.runID) ?? runID
                statusMessage = response.retried
                    ? (
                        isAdvancedInformationDensityEnabled
                            ? "Retried run \(nonEmpty(response.previousRunID) ?? runID) -> queued run \(updatedRunID)."
                            : "Retry started. A new attempt is queued."
                    )
                    : (
                        isAdvancedInformationDensityEnabled
                            ? "Retry request returned without creating a new run."
                            : "Retry request completed without creating a new attempt."
                    )
                updatedActions = taskRunActionAvailability(from: response.actions)

            case .requeue:
                let response = try await daemonClient.tasks.taskRequeue(
                    baseURL: daemonBaseURL,
                    authToken: authToken,
                    workspaceID: workspaceID,
                    taskID: taskID,
                    runID: runID,
                    reason: reason
                )
                updatedRunID = nonEmpty(response.runID) ?? runID
                statusMessage = response.requeued
                    ? (
                        isAdvancedInformationDensityEnabled
                            ? "Requeued run \(nonEmpty(response.previousRunID) ?? runID) -> queued run \(updatedRunID)."
                            : "Requeue started. The task is queued again."
                    )
                    : (
                        isAdvancedInformationDensityEnabled
                            ? "Requeue request returned without creating a new run."
                            : "Requeue request completed without creating a new attempt."
                    )
                updatedActions = taskRunActionAvailability(from: response.actions)
            }

            workflowQueueStore.setTaskRunControlStatus(runID: runID, updatedRunID: updatedRunID, message: statusMessage)
            tasksStatusMessage = statusMessage
            connectionStatus = .connected
            appendNotification(
                source: "tasks",
                action: action.statusActionID,
                message: statusMessage,
                level: .success
            )
            await fetchTaskRunList()

            if action == .cancel {
                if selectedTaskRunDetail?.runID == runID {
                    await fetchTaskRunDetail(runID: runID, actions: updatedActions)
                }
            } else if selectedTaskRunDetail?.runID == runID {
                await fetchTaskRunDetail(runID: updatedRunID, actions: updatedActions)
            }
        } catch {
            let message = daemonErrorMessage(
                error,
                fallbackContext: "\(action.title) failed",
                updateConnectionStatus: false,
                panelContext: .tasks
            )
            workflowQueueStore.setTaskRunControlStatus(runID: runID, message: message)
            tasksStatusMessage = message
            appendNotification(
                source: "tasks",
                action: action.statusActionID,
                message: message,
                level: .error
            )
        }
    }

    private func performApprovalDecision(
        approvalID: String,
        decisionPhrase: String,
        decisionByActorID: String,
        rationale: String?
    ) async {
        let trimmedApprovalID = approvalID.trimmingCharacters(in: .whitespacesAndNewlines)
        let trimmedPhrase = decisionPhrase.trimmingCharacters(in: .whitespacesAndNewlines)
        let trimmedActorID = decisionByActorID.trimmingCharacters(in: .whitespacesAndNewlines)
        let trimmedRationale = nonEmpty(rationale)

        guard !trimmedApprovalID.isEmpty else {
            return
        }
        guard !trimmedPhrase.isEmpty else {
            workflowQueueStore.setApprovalActionStatus(
                approvalID: trimmedApprovalID,
                message: "Decision phrase is required. For approval, use the exact requested phrase."
            )
            return
        }
        guard !trimmedActorID.isEmpty else {
            workflowQueueStore.setApprovalActionStatus(
                approvalID: trimmedApprovalID,
                message: "Decision actor is required. Select `Decision By` and retry."
            )
            return
        }

        guard let authToken = resolvedAuthToken() else {
            workflowQueueStore.setApprovalActionStatus(
                approvalID: trimmedApprovalID,
                message: "Set Assistant Access Token before submitting approval decisions."
            )
            return
        }

        guard let queuedApprovalID = workflowQueueStore.beginApprovalDecision(approvalID: trimmedApprovalID) else {
            return
        }
        defer {
            workflowQueueStore.finishApprovalDecision(approvalID: queuedApprovalID)
        }

        do {
            let response = try await daemonClient.approvals.approvalDecision(
                baseURL: daemonBaseURL,
                authToken: authToken,
                approvalID: queuedApprovalID,
                decisionPhrase: trimmedPhrase,
                decisionByActorID: trimmedActorID,
                rationale: trimmedRationale
            )
            let normalizedDecision = response.decision
                .trimmingCharacters(in: .whitespacesAndNewlines)
                .lowercased()
            let statusMessage: String
            if response.accepted {
                statusMessage = "Decision submitted: Approved. Run resumed."
            } else if normalizedDecision == "reject" || normalizedDecision == "rejected" {
                statusMessage = "Decision submitted: Rejected. Open Tasks or Inspect for follow-up."
            } else {
                statusMessage = "Decision submitted as \(response.decision). Open Tasks or Inspect for next steps."
            }
            workflowQueueStore.setApprovalActionStatus(approvalID: queuedApprovalID, message: statusMessage)
            appendNotification(
                source: "approvals",
                action: "decision_submit",
                message: statusMessage,
                level: .success
            )
            markHomeFirstSessionStepComplete(.reviewApprovals, source: "approvals_decision")
            await fetchApprovalsInbox()
        } catch {
            let statusMessage = daemonErrorMessage(
                error,
                fallbackContext: "Approval decision submit failed",
                updateConnectionStatus: false,
                panelContext: .approvals
            )
            workflowQueueStore.setApprovalActionStatus(approvalID: queuedApprovalID, message: statusMessage)
            appendNotification(
                source: "approvals",
                action: "decision_submit",
                message: statusMessage,
                level: .error
            )
        }
    }

    private func runAutomationScheduleSimulation() async {
        guard let authToken = resolvedAuthToken() else {
            automationSimulationStatusMessage = "Set Assistant Access Token before running schedule simulation."
            return
        }

        isAutomationSimulationInFlight = true
        defer {
            isAutomationSimulationInFlight = false
        }

        do {
            let response = try await daemonClient.automation.automationRunSchedule(
                baseURL: daemonBaseURL,
                authToken: authToken
            )
            let summary = truncateText(response.result.displayText, limit: 220)
            automationSimulationStatusMessage = "Schedule simulation at \(response.at): \(summary)"
            connectionStatus = .connected
            await fetchAutomationPanelData()
        } catch {
            automationSimulationStatusMessage = daemonErrorMessage(
                error,
                fallbackContext: "Schedule simulation failed",
                updateConnectionStatus: false,
                panelContext: .automation
            )
        }
    }

    private func fetchTaskRunDetail(
        runID: String,
        actions: TaskRunActionAvailabilityItem = .unavailable
    ) async {
        isTaskRunDetailLoading = true
        defer {
            isTaskRunDetailLoading = false
        }

        guard let authToken = resolvedAuthToken() else {
            taskRunDetailStatusMessage = "Set Assistant Access Token to query run detail."
            return
        }

        do {
            let response = try await daemonClient.inspect.inspectRun(
                baseURL: daemonBaseURL,
                authToken: authToken,
                runID: runID
            )
            selectedTaskRunDetail = mapTaskRunDetail(response, actions: actions)
            taskRunDetailStatusMessage = "Loaded run detail for \(runID)."
            connectionStatus = .connected
        } catch {
            taskRunDetailStatusMessage = daemonErrorMessage(
                error,
                fallbackContext: "Task run detail query failed",
                updateConnectionStatus: false,
                panelContext: .tasks
            )
        }
    }

    private func runAutomationCommEventSimulation() async {
        guard let authToken = resolvedAuthToken() else {
            automationSimulationStatusMessage = "Set Assistant Access Token before running comm-event simulation."
            return
        }

        isAutomationSimulationInFlight = true
        defer {
            isAutomationSimulationInFlight = false
        }

        let eventID = "ui-sim-\(UUID().uuidString.lowercased())"
        let body = "UI simulation event \(Date.now.formatted(date: .omitted, time: .standard))"
        do {
            let response = try await daemonClient.automation.automationRunCommEvent(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                eventID: eventID,
                channel: "app",
                body: body,
                sender: "ui-simulator"
            )
            let summary = truncateText(response.result.displayText, limit: 220)
            let seededLabel = response.seededEvent ? "seeded" : "existing"
            automationSimulationStatusMessage = "Comm-event simulation (\(seededLabel), \(response.eventID)): \(summary)"
            connectionStatus = .connected
            await fetchAutomationPanelData()
        } catch {
            automationSimulationStatusMessage = daemonErrorMessage(
                error,
                fallbackContext: "Comm-event simulation failed",
                updateConnectionStatus: false,
                panelContext: .automation
            )
        }
    }

    private func mapAutomationTriggerRecord(_ record: DaemonAutomationTriggerRecord) -> AutomationTriggerItem {
        let updatedAtLabel: String
        if let parsed = parseDaemonTimestamp(record.updatedAt) {
            updatedAtLabel = parsed.formatted(date: .abbreviated, time: .shortened)
        } else {
            updatedAtLabel = nonEmpty(record.updatedAt) ?? "n/a"
        }

        return AutomationTriggerItem(
            id: record.triggerID,
            triggerType: nonEmpty(record.triggerType) ?? "unknown",
            enabled: record.enabled,
            directiveTitle: nonEmpty(record.directiveTitle) ?? "Untitled directive",
            directiveInstruction: nonEmpty(record.directiveInstruction) ?? "No instruction summary.",
            directiveStatus: nonEmpty(record.directiveStatus) ?? "unknown",
            subjectPrincipalActor: nonEmpty(record.subjectPrincipalActor) ?? "default",
            cooldownSeconds: record.cooldownSeconds,
            filterSummary: nonEmpty(record.filterJSON) ?? "No filter constraints.",
            updatedAtLabel: updatedAtLabel
        )
    }

    private func mapAutomationFireHistoryRecord(_ record: DaemonAutomationFireHistoryRecord) -> AutomationFireHistoryItem {
        let firedAtRaw = nonEmpty(record.firedAt) ?? ""
        let firedTimestamp = parseDaemonTimestamp(firedAtRaw) ?? .distantPast
        let firedAtLabel = firedAtRaw.isEmpty
            ? "n/a"
            : formattedWorkflowTimestamp(firedAtRaw)

        return AutomationFireHistoryItem(
            id: nonEmpty(record.fireID) ?? UUID().uuidString.lowercased(),
            triggerID: nonEmpty(record.triggerID) ?? "unknown",
            triggerType: nonEmpty(record.triggerType) ?? "unknown",
            status: automationFireHistoryStatus(from: record.status, outcome: record.outcome),
            outcome: nonEmpty(record.outcome) ?? "n/a",
            idempotencySignal: nonEmpty(record.idempotencySignal) ?? "n/a",
            idempotencyKey: nonEmpty(record.idempotencyKey) ?? "n/a",
            firedAtLabel: firedAtLabel,
            taskID: nonEmpty(record.taskID),
            runID: nonEmpty(record.runID),
            sortTimestamp: firedTimestamp,
            route: mapWorkflowRoute(record.route)
        )
    }

    private func automationFireHistoryStatus(from rawStatus: String, outcome rawOutcome: String?) -> AutomationFireHistoryStatus {
        let normalizedStatus = rawStatus.trimmingCharacters(in: .whitespacesAndNewlines)
            .replacingOccurrences(of: "-", with: "_")
            .uppercased()
        if normalizedStatus == "CREATED_TASK" {
            return .createdTask
        }
        if normalizedStatus == "PENDING" {
            return .pending
        }
        if normalizedStatus == "FAILED" {
            return .failed
        }

        let normalizedOutcome = rawOutcome?
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .replacingOccurrences(of: "-", with: "_")
            .uppercased()

        if normalizedOutcome == "CREATED_TASK" {
            return .createdTask
        }
        if normalizedOutcome == "PENDING" {
            return .pending
        }
        if normalizedOutcome == "FAILED" {
            return .failed
        }
        return .other(nonEmpty(rawStatus) ?? nonEmpty(rawOutcome) ?? "unknown")
    }

    private func mapApprovalInboxRecord(_ record: DaemonApprovalInboxRecord) -> ApprovalInboxItem {
        let state = approvalDecisionState(from: record.state)
        let decision = approvalDecisionOutcome(from: record.decision, state: state)
        let taskState = normalizedWorkflowStateLabel(record.taskState)
        let runState = normalizedWorkflowStateLabel(record.runState) ?? taskState
        let stepName = nonEmpty(record.stepName)
            ?? nonEmpty(record.stepCapabilityKey)
            ?? "Approval step"

        return ApprovalInboxItem(
            id: record.approvalRequestID,
            taskTitle: nonEmpty(record.taskTitle) ?? "Untitled task request",
            decisionState: state,
            decisionOutcome: decision,
            riskLevel: approvalRiskLevel(from: record.riskLevel),
            riskRationale: nonEmpty(record.riskRationale) ?? "Risk rationale not provided.",
            requestedAtLabel: formattedWorkflowTimestamp(record.requestedAt),
            decidedAtLabel: nonEmpty(record.decidedAt).map(formattedWorkflowTimestamp),
            decisionByActorID: nonEmpty(record.decisionByActorID),
            decisionRationale: nonEmpty(record.decisionRationale),
            requestedPhrase: nonEmpty(record.requestedPhrase),
            taskState: taskState ?? "Unknown",
            runState: runState ?? "Unknown",
            stepName: stepName,
            stepCapabilityKey: nonEmpty(record.stepCapabilityKey),
            requestedByActorID: nonEmpty(record.requestedByActorID) ?? "unknown",
            subjectPrincipalActorID: nonEmpty(record.subjectPrincipalActorID) ?? "unknown",
            actingAsActorID: nonEmpty(record.actingAsActorID) ?? "unknown",
            taskID: nonEmpty(record.taskID),
            runID: nonEmpty(record.runID),
            stepID: nonEmpty(record.stepID),
            route: mapWorkflowRoute(record.route)
        )
    }

    private func taskRunActionAvailability(
        from value: DaemonTaskRunActionAvailability?
    ) -> TaskRunActionAvailabilityItem {
        guard let value else {
            return .unavailable
        }
        return TaskRunActionAvailabilityItem(
            canCancel: value.canCancel,
            canRetry: value.canRetry,
            canRequeue: value.canRequeue
        )
    }

    private func mapTaskRunListRecord(_ record: DaemonTaskRunListRecord) -> TaskRunListRowItem {
        let taskState = normalizedWorkflowStateLabel(record.taskState) ?? "Unknown"
        let runState = normalizedWorkflowStateLabel(record.runState) ?? taskState
        let effectiveState = taskRunWorkflowState(from: nonEmpty(record.runState) ?? record.taskState)
        let taskUpdatedAtLabel = formattedWorkflowTimestamp(record.taskUpdatedAt)
        let runUpdatedAtLabel = nonEmpty(record.runUpdatedAt).map(formattedWorkflowTimestamp)
        let sortReference = nonEmpty(record.runUpdatedAt)
            ?? nonEmpty(record.taskUpdatedAt)
            ?? nonEmpty(record.startedAt)
            ?? nonEmpty(record.taskCreatedAt)
        let route = mapWorkflowRoute(record.route)
        let actions = taskRunActionAvailability(from: record.actions)

        return TaskRunListRowItem(
            id: "\(record.taskID)::\(nonEmpty(record.runID) ?? "no-run")",
            title: nonEmpty(record.title) ?? "Untitled task",
            taskID: record.taskID,
            runID: nonEmpty(record.runID),
            taskState: taskState,
            runState: runState,
            effectiveState: effectiveState,
            priority: record.priority,
            priorityLabel: workflowPriorityLabel(record.priority),
            requestedByActorID: nonEmpty(record.requestedByActorID) ?? "unknown",
            subjectPrincipalActorID: nonEmpty(record.subjectPrincipalActorID) ?? "unknown",
            actingAsActorID: nonEmpty(record.actingAsActorID)
                ?? nonEmpty(record.subjectPrincipalActorID)
                ?? "unknown",
            taskCreatedAtLabel: formattedWorkflowTimestamp(record.taskCreatedAt),
            taskUpdatedAtLabel: taskUpdatedAtLabel,
            runCreatedAtLabel: nonEmpty(record.runCreatedAt).map(formattedWorkflowTimestamp),
            runUpdatedAtLabel: runUpdatedAtLabel,
            startedAtLabel: nonEmpty(record.startedAt).map(formattedWorkflowTimestamp),
            finishedAtLabel: nonEmpty(record.finishedAt).map(formattedWorkflowTimestamp),
            lastError: nonEmpty(record.lastError),
            actions: actions,
            sortTimestamp: sortReference.flatMap(parseDaemonTimestamp) ?? .distantPast,
            route: route
        )
    }

    private func mapTaskRunDetail(
        _ response: DaemonInspectRunResponse,
        actions: TaskRunActionAvailabilityItem
    ) -> TaskRunDetailItem {
        let steps = response.steps
            .map { step in
                TaskRunDetailStepItem(
                    id: step.stepID,
                    index: step.stepIndex,
                    name: nonEmpty(step.name) ?? "Unnamed step",
                    statusLabel: normalizedWorkflowStateLabel(step.status) ?? "Unknown",
                    capability: nonEmpty(step.capabilityKey),
                    interactionLevel: nonEmpty(step.interactionLevel),
                    retryLabel: "\(step.retryCount)/\(step.retryMax)",
                    timeoutLabel: step.timeoutSeconds.map { "\($0)s" } ?? "n/a",
                    updatedAtLabel: formattedWorkflowTimestamp(step.updatedAt),
                    lastError: nonEmpty(step.lastError)
                )
            }
            .sorted { lhs, rhs in
                lhs.index < rhs.index
            }

        let artifacts = response.artifacts
            .sorted { lhs, rhs in
                (parseDaemonTimestamp(lhs.createdAt) ?? .distantPast) > (parseDaemonTimestamp(rhs.createdAt) ?? .distantPast)
            }
            .map { artifact in
                TaskRunDetailArtifactItem(
                    id: artifact.artifactID,
                    type: nonEmpty(artifact.artifactType) ?? "unknown",
                    stepID: nonEmpty(artifact.stepID),
                    uri: nonEmpty(artifact.uri),
                    contentHash: nonEmpty(artifact.contentHash),
                    createdAtLabel: formattedWorkflowTimestamp(artifact.createdAt)
                )
            }

        let auditEntries = response.auditEntries
            .sorted { lhs, rhs in
                (parseDaemonTimestamp(lhs.createdAt) ?? .distantPast) > (parseDaemonTimestamp(rhs.createdAt) ?? .distantPast)
            }
            .map { entry in
                TaskRunDetailAuditItem(
                    id: entry.auditID,
                    eventType: nonEmpty(entry.eventType) ?? "unknown",
                    actorID: nonEmpty(entry.actorID),
                    actingAsActorID: nonEmpty(entry.actingAsActorID),
                    correlationID: nonEmpty(entry.correlationID),
                    payloadSummary: nonEmpty(entry.payloadJSON).map { truncateText($0, limit: 220) },
                    createdAtLabel: formattedWorkflowTimestamp(entry.createdAt)
                )
            }

        return TaskRunDetailItem(
            id: response.run.runID,
            taskID: response.task.taskID,
            runID: response.run.runID,
            title: nonEmpty(response.task.title) ?? "Untitled task",
            taskState: normalizedWorkflowStateLabel(response.task.state) ?? "Unknown",
            runState: normalizedWorkflowStateLabel(response.run.state) ?? "Unknown",
            priorityLabel: workflowPriorityLabel(response.task.priority),
            requestedByActorID: nonEmpty(response.task.requestedByActorID) ?? "unknown",
            subjectPrincipalActorID: nonEmpty(response.task.subjectPrincipalActorID) ?? "unknown",
            actingAsActorID: nonEmpty(response.run.actingAsActorID) ?? "unknown",
            startedAtLabel: nonEmpty(response.run.startedAt).map(formattedWorkflowTimestamp),
            finishedAtLabel: nonEmpty(response.run.finishedAt).map(formattedWorkflowTimestamp),
            updatedAtLabel: formattedWorkflowTimestamp(response.run.updatedAt),
            lastError: nonEmpty(response.run.lastError),
            actions: actions,
            route: mapWorkflowRoute(response.route),
            steps: steps,
            artifacts: artifacts,
            auditEntries: auditEntries
        )
    }

    private func mapCommunicationThreadRecord(_ record: DaemonCommThreadListRecord) -> CommunicationThreadItem {
        let threadID = nonEmpty(record.threadID) ?? UUID().uuidString.lowercased()
        let sortReference = nonEmpty(record.updatedAt)
            ?? nonEmpty(record.lastOccurredAt)
            ?? nonEmpty(record.createdAt)
        let logicalChannelID = logicalCommunicationChannelID(rawChannelID: record.channel)
        return CommunicationThreadItem(
            id: threadID,
            workspaceID: nonEmpty(record.workspaceID) ?? workspaceID,
            channel: logicalChannelID,
            connectorID: nonEmpty(record.connectorID),
            title: nonEmpty(record.title) ?? threadID,
            externalRef: nonEmpty(record.externalRef),
            lastEventID: nonEmpty(record.lastEventID),
            lastEventType: nonEmpty(record.lastEventType),
            lastDirection: nonEmpty(record.lastDirection),
            lastOccurredAtLabel: nonEmpty(record.lastOccurredAt).map(formattedWorkflowTimestamp),
            lastBodyPreview: nonEmpty(record.lastBodyPreview),
            participantAddresses: record.participantAddresses
                .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
                .filter { !$0.isEmpty },
            eventCount: max(record.eventCount, 0),
            createdAtLabel: formattedWorkflowTimestamp(record.createdAt),
            updatedAtLabel: formattedWorkflowTimestamp(record.updatedAt),
            sortTimestamp: sortReference.flatMap(parseDaemonTimestamp) ?? .distantPast
        )
    }

    private func mapCommunicationEventRecord(_ record: DaemonCommEventTimelineRecord) -> CommunicationEventItem {
        let eventID = nonEmpty(record.eventID) ?? UUID().uuidString.lowercased()
        let sortReference = nonEmpty(record.occurredAt) ?? nonEmpty(record.createdAt)
        let logicalChannelID = logicalCommunicationChannelID(rawChannelID: record.channel)
        let addresses = record.addresses
            .sorted { lhs, rhs in
                if lhs.position == rhs.position {
                    if lhs.role == rhs.role {
                        return lhs.value.localizedCaseInsensitiveCompare(rhs.value) == .orderedAscending
                    }
                    return lhs.role.localizedCaseInsensitiveCompare(rhs.role) == .orderedAscending
                }
                return lhs.position < rhs.position
            }
            .enumerated()
            .map { index, address in
                CommunicationEventAddressItem(
                    id: "\(eventID)::addr::\(index)",
                    role: nonEmpty(address.role) ?? "unknown",
                    value: nonEmpty(address.value) ?? "unknown",
                    display: nonEmpty(address.display),
                    position: address.position
                )
            }

        return CommunicationEventItem(
            id: eventID,
            workspaceID: nonEmpty(record.workspaceID) ?? workspaceID,
            threadID: nonEmpty(record.threadID) ?? "unknown",
            channel: logicalChannelID,
            connectorID: nonEmpty(record.connectorID),
            eventType: nonEmpty(record.eventType) ?? "unknown",
            direction: nonEmpty(record.direction) ?? "unknown",
            assistantEmitted: record.assistantEmitted,
            bodyText: nonEmpty(record.bodyText),
            occurredAtLabel: formattedWorkflowTimestamp(record.occurredAt),
            createdAtLabel: formattedWorkflowTimestamp(record.createdAt),
            addresses: addresses,
            sortTimestamp: sortReference.flatMap(parseDaemonTimestamp) ?? .distantPast
        )
    }

    private func mapCommunicationCallSessionRecord(_ record: DaemonCommCallSessionListRecord) -> CommunicationCallSessionItem {
        let sessionID = nonEmpty(record.sessionID) ?? UUID().uuidString.lowercased()
        let sortReference = nonEmpty(record.updatedAt)
            ?? nonEmpty(record.endedAt)
            ?? nonEmpty(record.startedAt)
        return CommunicationCallSessionItem(
            id: sessionID,
            workspaceID: nonEmpty(record.workspaceID) ?? workspaceID,
            provider: nonEmpty(record.provider) ?? "unknown",
            connectorID: nonEmpty(record.connectorID),
            providerCallID: nonEmpty(record.providerCallID),
            threadID: nonEmpty(record.threadID),
            direction: nonEmpty(record.direction) ?? "unknown",
            fromAddress: nonEmpty(record.fromAddress),
            toAddress: nonEmpty(record.toAddress),
            status: nonEmpty(record.status) ?? "unknown",
            startedAtLabel: nonEmpty(record.startedAt).map(formattedWorkflowTimestamp),
            endedAtLabel: nonEmpty(record.endedAt).map(formattedWorkflowTimestamp),
            updatedAtLabel: formattedWorkflowTimestamp(record.updatedAt),
            sortTimestamp: sortReference.flatMap(parseDaemonTimestamp) ?? .distantPast
        )
    }

    private func mapCommunicationDeliveryAttemptRecord(_ record: DaemonCommAttemptRecord) -> CommunicationDeliveryAttemptItem {
        let attemptID = nonEmpty(record.attemptID) ?? UUID().uuidString.lowercased()
        let normalizedRoutePhase = nonEmpty(record.routePhase) ?? "primary"
        let logicalChannelID = logicalCommunicationChannelID(rawChannelID: record.channel)
        return CommunicationDeliveryAttemptItem(
            id: attemptID,
            workspaceID: nonEmpty(record.workspaceID) ?? workspaceID,
            operationID: nonEmpty(record.operationID),
            taskID: nonEmpty(record.taskID),
            runID: nonEmpty(record.runID),
            stepID: nonEmpty(record.stepID),
            eventID: nonEmpty(record.eventID),
            threadID: nonEmpty(record.threadID),
            destinationEndpoint: nonEmpty(record.destinationEndpoint) ?? "unknown",
            idempotencyKey: nonEmpty(record.idempotencyKey) ?? "unknown",
            channel: logicalChannelID,
            routeIndex: record.routeIndex,
            routePhase: normalizedRoutePhase,
            retryOrdinal: max(record.retryOrdinal, 0),
            fallbackFromChannel: nonEmpty(record.fallbackFromChannel),
            status: nonEmpty(record.status) ?? "unknown",
            providerReceipt: nonEmpty(record.providerReceipt),
            error: nonEmpty(record.error),
            attemptedAtLabel: formattedWorkflowTimestamp(record.attemptedAt),
            sortTimestamp: parseDaemonTimestamp(record.attemptedAt) ?? .distantPast
        )
    }

    private func nonNegativeInteger(_ value: DaemonJSONValue?) -> Int? {
        switch value {
        case .number(let number):
            let rounded = Int(number.rounded())
            guard number == number.rounded(), rounded >= 0 else {
                return nil
            }
            return rounded
        case .string(let raw):
            guard let parsed = Int(raw.trimmingCharacters(in: .whitespacesAndNewlines)),
                  parsed >= 0 else {
                return nil
            }
            return parsed
        default:
            return nil
        }
    }

    private func logicalCommunicationChannelID(rawChannelID: String) -> String {
        let normalizedChannelID = rawChannelID
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .lowercased()
            .replacingOccurrences(of: "-", with: "_")
            .replacingOccurrences(of: " ", with: "_")
        switch normalizedChannelID {
        case "app", "app_chat":
            return "app"
        case "voice", "twilio_voice":
            return "voice"
        case "message", "sms", "imessage", "imessage_sms", "imessage_sms_bridge", "imessage_bridge", "twilio_sms", "twilio_sms_bridge":
            return "message"
        case "":
            return "unknown"
        default:
            return normalizedChannelID
        }
    }

    private func mapChannelDeliveryPolicyRecord(_ record: DaemonCommPolicyRecord) -> ChannelDeliveryPolicyItem {
        let policyID = nonEmpty(record.id) ?? UUID().uuidString.lowercased()
        let sourceChannel = normalizedChannelDeliveryPolicyChannelID(record.sourceChannel)
        let policy = record.policy
        let sortReference = nonEmpty(record.updatedAt) ?? nonEmpty(record.createdAt)
        return ChannelDeliveryPolicyItem(
            id: policyID,
            workspaceID: nonEmpty(record.workspaceID) ?? workspaceID,
            sourceChannel: sourceChannel,
            endpointPattern: nonEmpty(record.endpointPattern),
            isDefault: record.isDefault,
            primaryChannel: normalizedChannelDeliveryPolicyChannelID(policy.primaryChannel),
            retryCount: max(policy.retryCount, 0),
            fallbackChannels: Self.canonicalLogicalChannelIDs(policy.fallbackChannels),
            createdAtLabel: formattedWorkflowTimestamp(record.createdAt),
            updatedAtLabel: formattedWorkflowTimestamp(record.updatedAt),
            sortTimestamp: sortReference.flatMap(parseDaemonTimestamp) ?? .distantPast
        )
    }

    private func mapChannelDeliveryPoliciesBySource(
        _ records: [DaemonCommPolicyRecord]
    ) -> [String: [ChannelDeliveryPolicyItem]] {
        var grouped: [String: [ChannelDeliveryPolicyItem]] = [:]
        for record in records {
            let item = mapChannelDeliveryPolicyRecord(record)
            guard !item.sourceChannel.isEmpty else {
                continue
            }
            grouped[item.sourceChannel, default: []].append(item)
        }
        return grouped.mapValues(sortedChannelDeliveryPolicies)
    }

    private func sortedChannelDeliveryPolicies(_ policies: [ChannelDeliveryPolicyItem]) -> [ChannelDeliveryPolicyItem] {
        policies.sorted { lhs, rhs in
            if lhs.isDefault != rhs.isDefault {
                return lhs.isDefault && !rhs.isDefault
            }
            if lhs.sortTimestamp == rhs.sortTimestamp {
                return lhs.id.localizedCaseInsensitiveCompare(rhs.id) == .orderedAscending
            }
            return lhs.sortTimestamp > rhs.sortTimestamp
        }
    }

    private func channelDeliveryDraft(from policy: ChannelDeliveryPolicyItem) -> ChannelDeliveryPolicyDraft {
        ChannelDeliveryPolicyDraft(
            policyID: policy.id,
            endpointPattern: policy.endpointPattern ?? "",
            primaryChannel: policy.primaryChannel,
            retryCount: max(policy.retryCount, 0),
            fallbackChannelsText: policy.fallbackChannels.joined(separator: ", "),
            isDefault: policy.isDefault
        )
    }

    private func defaultChannelDeliveryPolicyDraft(
        channelID: String,
        policies: [ChannelDeliveryPolicyItem]
    ) -> ChannelDeliveryPolicyDraft {
        let normalizedChannelID = normalizedChannelDeliveryPolicyChannelID(channelID)
        let preferredPolicy = sortedChannelDeliveryPolicies(policies).first
        if let preferredPolicy {
            return channelDeliveryDraft(from: preferredPolicy)
        }
        return ChannelDeliveryPolicyDraft(
            policyID: nil,
            endpointPattern: "",
            primaryChannel: normalizedChannelID,
            retryCount: 0,
            fallbackChannelsText: "",
            isDefault: true
        )
    }

    private func parseChannelFallbackChannels(_ raw: String) -> [String] {
        var seen = Set<String>()
        var values: [String] = []
        for component in raw.split(whereSeparator: { $0 == "," || $0 == "\n" || $0 == ";" }) {
            let normalized = normalizedChannelDeliveryPolicyChannelID(String(component))
            guard !normalized.isEmpty else {
                continue
            }
            if seen.insert(normalized).inserted {
                values.append(normalized)
            }
        }
        return values
    }

    private func applyChannelDeliveryPolicyRecord(_ record: DaemonCommPolicyRecord) {
        let mapped = mapChannelDeliveryPolicyRecord(record)
        guard !mapped.sourceChannel.isEmpty else {
            return
        }
        var policies = channelDeliveryPoliciesByChannelID[mapped.sourceChannel] ?? []
        if let index = policies.firstIndex(where: { $0.id == mapped.id }) {
            policies[index] = mapped
        } else {
            policies.append(mapped)
        }
        let sortedPolicies = sortedChannelDeliveryPolicies(policies)
        channelDeliveryPoliciesByChannelID[mapped.sourceChannel] = sortedPolicies
        channelDeliveryPolicyDraftByID[mapped.sourceChannel] = defaultChannelDeliveryPolicyDraft(
            channelID: mapped.sourceChannel,
            policies: sortedPolicies
        )
    }

    private func synchronizeChannelDeliveryPolicyDrafts(
        with cards: [ChannelCardItem],
        policiesByChannelID: [String: [ChannelDeliveryPolicyItem]]
    ) {
        var validIDs = Set(cards.map { normalizedChannelDeliveryPolicyChannelID($0.id) })
        validIDs.formUnion(policiesByChannelID.keys.map(normalizedChannelDeliveryPolicyChannelID))
        validIDs.formUnion(channelDeliveryPolicyDraftByID.keys.map(normalizedChannelDeliveryPolicyChannelID))
        validIDs = Set(validIDs.filter { !$0.isEmpty })

        var collapsedPolicies: [String: [ChannelDeliveryPolicyItem]] = [:]
        for (rawChannelID, policies) in policiesByChannelID {
            let normalizedChannelID = normalizedChannelDeliveryPolicyChannelID(rawChannelID)
            guard !normalizedChannelID.isEmpty else {
                continue
            }
            collapsedPolicies[normalizedChannelID, default: []].append(contentsOf: policies.map { policy in
                ChannelDeliveryPolicyItem(
                    id: policy.id,
                    workspaceID: policy.workspaceID,
                    sourceChannel: normalizedChannelID,
                    endpointPattern: policy.endpointPattern,
                    isDefault: policy.isDefault,
                    primaryChannel: normalizedChannelDeliveryPolicyChannelID(policy.primaryChannel),
                    retryCount: max(policy.retryCount, 0),
                    fallbackChannels: Self.canonicalLogicalChannelIDs(policy.fallbackChannels),
                    createdAtLabel: policy.createdAtLabel,
                    updatedAtLabel: policy.updatedAtLabel,
                    sortTimestamp: policy.sortTimestamp
                )
            })
        }

        var normalizedExistingDrafts: [String: ChannelDeliveryPolicyDraft] = [:]
        for (rawChannelID, draft) in channelDeliveryPolicyDraftByID {
            let normalizedChannelID = normalizedChannelDeliveryPolicyChannelID(rawChannelID)
            guard !normalizedChannelID.isEmpty else {
                continue
            }
            var normalizedDraft = draft
            let normalizedPrimary = normalizedChannelDeliveryPolicyChannelID(draft.primaryChannel)
            normalizedDraft.primaryChannel = normalizedPrimary.isEmpty ? normalizedChannelID : normalizedPrimary
            normalizedDraft.retryCount = max(draft.retryCount, 0)
            normalizedExistingDrafts[normalizedChannelID] = normalizedDraft
        }

        let normalizedStatusByChannelID = channelDeliveryPolicyActionStatusByID.reduce(into: [String: String]()) { partialResult, entry in
            let normalizedChannelID = normalizedChannelDeliveryPolicyChannelID(entry.key)
            guard !normalizedChannelID.isEmpty else {
                return
            }
            partialResult[normalizedChannelID] = entry.value
        }

        let normalizedInFlightIDs = Set(
            channelDeliveryPolicySaveInFlightIDs
                .map(normalizedChannelDeliveryPolicyChannelID)
                .filter { !$0.isEmpty }
        )

        var normalizedPolicies: [String: [ChannelDeliveryPolicyItem]] = [:]
        var nextDrafts: [String: ChannelDeliveryPolicyDraft] = [:]

        for channelID in validIDs {
            let policies = sortedChannelDeliveryPolicies(collapsedPolicies[channelID] ?? [])
            normalizedPolicies[channelID] = policies

            let fallbackDraft = defaultChannelDeliveryPolicyDraft(
                channelID: channelID,
                policies: policies
            )
            guard var existingDraft = normalizedExistingDrafts[channelID] else {
                nextDrafts[channelID] = fallbackDraft
                continue
            }

            if let policyID = nonEmpty(existingDraft.policyID),
               !policies.contains(where: { $0.id == policyID }) {
                nextDrafts[channelID] = fallbackDraft
                continue
            }

            if nonEmpty(existingDraft.primaryChannel) == nil {
                existingDraft.primaryChannel = channelID
            }
            existingDraft.retryCount = max(existingDraft.retryCount, 0)
            nextDrafts[channelID] = existingDraft
        }

        channelDeliveryPoliciesByChannelID = normalizedPolicies
        channelDeliveryPolicyDraftByID = nextDrafts
        channelDeliveryPolicyActionStatusByID = normalizedStatusByChannelID.filter { validIDs.contains($0.key) }
        channelDeliveryPolicySaveInFlightIDs = normalizedInFlightIDs.intersection(validIDs)
    }

    private func normalizedChannelDeliveryPolicyChannelID(_ raw: String) -> String {
        Self.canonicalLogicalChannelID(from: raw)
    }

    private func normalizedChannelConnectorMappingChannelID(_ raw: String) -> String {
        Self.canonicalLogicalChannelID(from: raw)
    }

    private func normalizedChannelConnectorMappingConnectorID(_ raw: String) -> String {
        Self.canonicalConnectorID(from: raw)
    }

    private func discoveredLogicalChannelIDsForMappings() -> [String] {
        var channelIDs: Set<String> = []
        channelIDs.formUnion(
            channelConnectorMappingsByChannelID.keys.map(normalizedChannelConnectorMappingChannelID)
        )
        channelIDs.formUnion(
            channelConnectorMappingDraftByChannelID.keys.map(normalizedChannelConnectorMappingChannelID)
        )
        channelIDs.formUnion(
            channelCards.map { normalizedChannelConnectorMappingChannelID($0.logicalChannelID) }
        )
        channelIDs.formUnion(
            channelCards.map { normalizedChannelConnectorMappingChannelID($0.id) }
        )
        channelIDs.formUnion(
            logicalChannelCards.map { normalizedChannelConnectorMappingChannelID($0.channelID) }
        )
        if channelIDs.isEmpty {
            channelIDs.formUnion(Self.canonicalLogicalChannelSortOrder.keys)
        }
        let normalized = channelIDs.filter { !$0.isEmpty }
        return normalized.sorted(by: logicalChannelSortComparator)
    }

    private func parseConnectorIDsFromDraftText(_ raw: String?) -> [String] {
        guard let raw = nonEmpty(raw) else {
            return []
        }
        return normalizedConnectorIDList(fromRawText: raw)
    }

    private func normalizedConnectorIDList(from value: DaemonJSONValue?) -> [String] {
        guard let value else {
            return []
        }
        return normalizedConnectorIDList(from: value)
    }

    private func normalizedConnectorIDList(from value: DaemonJSONValue) -> [String] {
        switch value {
        case .array(let values):
            var merged: [String] = []
            for nested in values {
                merged.append(contentsOf: normalizedConnectorIDList(from: nested))
            }
            return Self.deduplicatedPreservingOrder(merged)
        case .string(let rawValue):
            return normalizedConnectorIDList(fromRawText: rawValue)
        case .object(let object):
            let preferredKeys = [
                "connector_ids",
                "mapped_connector_ids",
                "enabled_connector_ids",
                "connectors",
                "ids"
            ]
            for key in preferredKeys {
                if let nested = object[key] {
                    let parsed = normalizedConnectorIDList(from: nested)
                    if !parsed.isEmpty {
                        return parsed
                    }
                }
            }
            return []
        case .number, .bool, .null:
            return []
        }
    }

    private func normalizedConnectorIDList(fromRawText raw: String) -> [String] {
        var normalizedIDs: [String] = []
        var seenIDs: Set<String> = []
        for token in Self.splitListTokens(raw) {
            var candidate = token
            if let equalsIndex = candidate.lastIndex(of: "=") {
                let suffix = candidate[candidate.index(after: equalsIndex)...]
                    .trimmingCharacters(in: .whitespacesAndNewlines)
                if !suffix.isEmpty {
                    candidate = suffix
                }
            }
            candidate = candidate.trimmingCharacters(in: CharacterSet(charactersIn: "\"'`"))
            let canonicalID = normalizedChannelConnectorMappingConnectorID(candidate)
            guard !canonicalID.isEmpty else {
                continue
            }
            if seenIDs.insert(canonicalID).inserted {
                normalizedIDs.append(canonicalID)
            }
        }
        return normalizedIDs
    }

    private func normalizedCapabilityList(_ capabilities: [String]) -> [String] {
        var seenCapabilities: Set<String> = []
        var normalizedCapabilities: [String] = []
        for rawCapability in capabilities {
            for token in Self.splitListTokens(rawCapability) {
                let normalizedKey = token.lowercased()
                if seenCapabilities.insert(normalizedKey).inserted {
                    normalizedCapabilities.append(token)
                }
            }
        }
        return normalizedCapabilities
    }

    private func sortedChannelConnectorMappings(_ mappings: [ChannelConnectorMappingItem]) -> [ChannelConnectorMappingItem] {
        let sorted = mappings.sorted { lhs, rhs in
            if lhs.priority == rhs.priority {
                return lhs.connectorID.localizedCaseInsensitiveCompare(rhs.connectorID) == .orderedAscending
            }
            return lhs.priority < rhs.priority
        }
        return sorted.enumerated().map { index, item in
            var normalizedItem = item
            normalizedItem.priority = index + 1
            return normalizedItem
        }
    }

    private func rebalancedChannelConnectorMappingsPreservingCurrentOrder(
        _ mappings: [ChannelConnectorMappingItem]
    ) -> [ChannelConnectorMappingItem] {
        mappings.enumerated().map { index, item in
            var normalizedItem = item
            normalizedItem.priority = index + 1
            return normalizedItem
        }
    }

    private func mergedChannelConnectorMappings(
        observed: [ChannelConnectorMappingItem],
        inferred: [ChannelConnectorMappingItem],
        channelID: String
    ) -> [ChannelConnectorMappingItem] {
        var byConnectorID: [String: ChannelConnectorMappingItem] = [:]
        for mapping in inferred {
            if let normalized = normalizedChannelConnectorMapping(
                mapping,
                fallbackChannelID: channelID
            ) {
                byConnectorID[normalized.connectorID] = normalized
            }
        }
        for mapping in observed {
            if let normalized = normalizedChannelConnectorMapping(
                mapping,
                fallbackChannelID: channelID
            ) {
                byConnectorID[normalized.connectorID] = normalized
            }
        }
        return sortedChannelConnectorMappings(Array(byConnectorID.values))
    }

    private func inferredChannelConnectorMappingsByLogicalChannelID(
        from cards: [ChannelCardItem]
    ) -> [String: [ChannelConnectorMappingItem]] {
        var grouped: [String: [ChannelConnectorMappingItem]] = [:]
        for card in cards {
            let logicalChannelID = normalizedChannelConnectorMappingChannelID(card.logicalChannelID)
            guard !logicalChannelID.isEmpty else {
                continue
            }

            var mappedConnectorIDs = card.mappedConnectorIDs
                .map(normalizedChannelConnectorMappingConnectorID)
                .filter { !$0.isEmpty }
            if mappedConnectorIDs.isEmpty {
                mappedConnectorIDs = parseConnectorIDsFromDraftText(
                    card.readOnlyConfiguration["mapped_connector_ids"]
                        ?? card.editableConfiguration["mapped_connector_ids"]
                        ?? card.details["Mapped Connector IDs"]
                )
            }

            var enabledConnectorIDs = card.enabledConnectorIDs
                .map(normalizedChannelConnectorMappingConnectorID)
                .filter { !$0.isEmpty }
            if enabledConnectorIDs.isEmpty {
                enabledConnectorIDs = parseConnectorIDsFromDraftText(
                    card.readOnlyConfiguration["enabled_connector_ids"]
                        ?? card.editableConfiguration["enabled_connector_ids"]
                        ?? card.details["Enabled Connector IDs"]
                )
            }

            let primaryConnectorID = nonEmpty(
                normalizedChannelConnectorMappingConnectorID(
                    card.primaryConnectorID
                        ?? card.readOnlyConfiguration["primary_connector_id"]
                        ?? card.editableConfiguration["primary_connector_id"]
                        ?? card.details["Primary Connector ID"]
                        ?? ""
                )
            )

            var connectorIDs = mappedConnectorIDs
            if connectorIDs.isEmpty {
                connectorIDs = enabledConnectorIDs
            }
            if let primaryConnectorID, !connectorIDs.contains(primaryConnectorID) {
                connectorIDs.insert(primaryConnectorID, at: 0)
            }
            guard !connectorIDs.isEmpty else {
                continue
            }

            let enabledConnectorSet = Set(enabledConnectorIDs)
            let defaultEnabled = enabledConnectorSet.isEmpty

            for (index, connectorID) in connectorIDs.enumerated() {
                grouped[logicalChannelID, default: []].append(
                    ChannelConnectorMappingItem(
                        channelID: logicalChannelID,
                        connectorID: connectorID,
                        enabled: defaultEnabled ? true : enabledConnectorSet.contains(connectorID),
                        priority: index + 1,
                        capabilities: card.declaredCapabilities,
                        createdAtLabel: nil,
                        updatedAtLabel: nil
                    )
                )
            }
        }

        return grouped.mapValues { mappings in
            sortedChannelConnectorMappings(mappings)
        }
    }

    private func mapChannelConnectorMappingsByChannel(
        _ records: [DaemonChannelConnectorMappingRecord]
    ) -> [String: [ChannelConnectorMappingItem]] {
        var grouped: [String: [ChannelConnectorMappingItem]] = [:]
        for record in records {
            let channelID = normalizedChannelConnectorMappingChannelID(record.channelID)
            let connectorID = normalizedChannelConnectorMappingConnectorID(record.connectorID)
            guard !channelID.isEmpty, !connectorID.isEmpty else {
                continue
            }
            let capabilities = record.capabilities
                .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
                .filter { !$0.isEmpty }
            grouped[channelID, default: []].append(
                ChannelConnectorMappingItem(
                    channelID: channelID,
                    connectorID: connectorID,
                    enabled: record.enabled,
                    priority: max(record.priority, 1),
                    capabilities: capabilities,
                    createdAtLabel: nonEmpty(record.createdAt).map(formattedWorkflowTimestamp),
                    updatedAtLabel: nonEmpty(record.updatedAt).map(formattedWorkflowTimestamp)
                )
            )
        }

        return grouped.mapValues(sortedChannelConnectorMappings)
    }

    private func performChannelConnectorMappingSave(channelID: String) async {
        let normalizedChannelID = normalizedChannelConnectorMappingChannelID(channelID)
        guard !normalizedChannelID.isEmpty else {
            return
        }
        guard !channelConnectorMappingSaveInFlightChannelIDs.contains(normalizedChannelID) else {
            return
        }
        guard let authToken = resolvedAuthToken() else {
            channelConnectorMappingActionStatusByChannelID[normalizedChannelID] =
                "Set Assistant Access Token before saving connector mappings."
            return
        }

        let source = sortedChannelConnectorMappings(channelConnectorMappingsByChannelID[normalizedChannelID] ?? [])
        let draft = sortedChannelConnectorMappings(
            channelConnectorMappingDraftByChannelID[normalizedChannelID] ?? source
        )
        guard !draft.isEmpty else {
            channelConnectorMappingActionStatusByChannelID[normalizedChannelID] =
                "No connector mappings are available to save for \(normalizedChannelID)."
            return
        }
        guard source != draft else {
            channelConnectorMappingActionStatusByChannelID[normalizedChannelID] =
                "No connector mapping changes to save for \(normalizedChannelID)."
            return
        }

        channelConnectorMappingSaveInFlightChannelIDs.insert(normalizedChannelID)
        channelConnectorMappingActionStatusByChannelID[normalizedChannelID] =
            "Saving connector mappings for \(normalizedChannelID)…"
        defer {
            channelConnectorMappingSaveInFlightChannelIDs.remove(normalizedChannelID)
        }

        do {
            var lastResponse: DaemonChannelConnectorMappingUpsertResponse?
            for mapping in draft {
                lastResponse = try await daemonClient.channels.channelConnectorMappingUpsert(
                    baseURL: daemonBaseURL,
                    authToken: authToken,
                    workspaceID: workspaceID,
                    channelID: normalizedChannelID,
                    connectorID: mapping.connectorID,
                    enabled: mapping.enabled,
                    priority: mapping.priority,
                    fallbackPolicy: channelConnectorMappingFallbackPolicy
                )
            }

            if let lastResponse {
                updateWorkspaceContext(from: lastResponse.workspaceID)
                channelConnectorMappingFallbackPolicy =
                    nonEmpty(lastResponse.fallbackPolicy) ?? channelConnectorMappingFallbackPolicy
                let remapped = mapChannelConnectorMappingsByChannel(lastResponse.bindings)
                let nextSource = mergedChannelConnectorMappings(
                    observed: remapped[normalizedChannelID] ?? [],
                    inferred: inferredChannelConnectorMappingsByLogicalChannelID(from: channelCards)[normalizedChannelID] ?? [],
                    channelID: normalizedChannelID
                )
                channelConnectorMappingsByChannelID[normalizedChannelID] = nextSource
                channelConnectorMappingDraftByChannelID[normalizedChannelID] = nextSource
            } else {
                let nextSource = mergedChannelConnectorMappings(
                    observed: draft,
                    inferred: inferredChannelConnectorMappingsByLogicalChannelID(from: channelCards)[normalizedChannelID] ?? [],
                    channelID: normalizedChannelID
                )
                channelConnectorMappingsByChannelID[normalizedChannelID] = nextSource
                channelConnectorMappingDraftByChannelID[normalizedChannelID] = nextSource
            }

            let enabledCount = channelConnectorMappings(channelID: normalizedChannelID).filter(\.enabled).count
            channelConnectorMappingActionStatusByChannelID[normalizedChannelID] =
                "Saved \(draft.count) mapping(s) for \(normalizedChannelID); \(enabledCount) enabled."
            channelsStatusMessage = "Updated connector mappings for \(normalizedChannelID)."
            connectionStatus = .connected
        } catch {
            channelConnectorMappingActionStatusByChannelID[normalizedChannelID] = daemonErrorMessage(
                error,
                fallbackContext: "Channel connector mapping update failed",
                updateConnectionStatus: false,
                panelContext: .channels
            )
        }
        updateOnboardingCompletionState()
    }

    private func refreshModelRouteSummary(authToken: String) async {
        do {
            let route = try await daemonClient.models.modelResolve(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID,
                taskClass: "chat"
            )
            modelRouteSummary = ModelRouteSummary(
                provider: route.provider,
                modelKey: route.modelKey,
                source: route.source,
                notes: nonEmpty(route.notes)
            )
            modelRouteStatusMessage = "Resolved chat route from \(route.source)."
            connectionStatus = .connected
        } catch {
            modelRouteSummary = nil
            modelRouteStatusMessage = daemonErrorMessage(
                error,
                fallbackContext: "Model route resolve failed",
                updateConnectionStatus: false,
                panelContext: .models
            )
        }
        updateOnboardingCompletionState()
    }

    private func fetchModelCatalogAndPolicies(authToken: String) async {
        do {
            let listResponse = try await daemonClient.models.modelList(
                baseURL: daemonBaseURL,
                authToken: authToken,
                workspaceID: workspaceID
            )
            updateWorkspaceContext(from: listResponse.workspaceID)
            modelCatalogItems = listResponse.models
                .map(mapModelCatalogRecord)
                .sorted { lhs, rhs in
                    if lhs.provider == rhs.provider {
                        return lhs.modelKey < rhs.modelKey
                    }
                    return lhs.provider < rhs.provider
                }
            let modelIDs = Set(modelCatalogItems.map(\.id))
            modelMutationStatusByID = modelMutationStatusByID.filter { modelIDs.contains($0.key) }
            modelMutationInFlightIDs = Set(modelMutationInFlightIDs.filter { modelIDs.contains($0) })
            syncDiscoveredModelCatalogFlags()

            do {
                let policyResponse = try await daemonClient.models.modelPolicy(
                    baseURL: daemonBaseURL,
                    authToken: authToken,
                    workspaceID: workspaceID
                )
                updateWorkspaceContext(from: policyResponse.workspaceID)
                let policies = policyResponse.policies
                    ?? (policyResponse.policy.map { [$0] } ?? [])
                modelPolicyItems = policies
                    .map(mapModelPolicyRecord)
                    .sorted { lhs, rhs in
                        lhs.taskClass < rhs.taskClass
                    }
                modelCatalogStatusMessage = "Model catalog loaded (\(modelCatalogItems.count)) • Policies: \(modelPolicyItems.count)"
                connectionStatus = .connected
            } catch {
                modelPolicyItems = []
                modelCatalogStatusMessage = daemonErrorMessage(
                    error,
                    fallbackContext: "Model policy query failed",
                    updateConnectionStatus: false,
                    panelContext: .models
                )
            }
        } catch {
            modelCatalogItems = []
            modelPolicyItems = []
            modelCatalogStatusMessage = daemonErrorMessage(
                error,
                fallbackContext: "Model catalog query failed",
                updateConnectionStatus: false,
                panelContext: .models
            )
        }
    }

    private func modelCatalogIdentifier(providerID: String, modelKey: String) -> String {
        modelsRouteStore.modelCatalogIdentifier(
            providerID: providerID,
            modelKey: modelKey,
            normalizedProviderID: normalizedProviderID(_:)
        )
    }

    private func mapModelCatalogRecord(_ record: DaemonModelCatalogRecord) -> ModelCatalogEntryItem {
        modelsRouteStore.mapModelCatalogRecord(record)
    }

    private func mapDiscoveredModelRecord(_ record: DaemonModelDiscoverItem) -> DiscoveredModelEntryItem {
        modelsRouteStore.mapDiscoveredModelRecord(
            record,
            modelCatalogIdentifier: { [weak self] providerID, modelKey in
                guard let self else {
                    return "\(providerID)::\(modelKey)"
                }
                return self.modelCatalogIdentifier(providerID: providerID, modelKey: modelKey)
            }
        )
    }

    private func syncDiscoveredModelCatalogFlags(providerID: String? = nil) {
        modelsRouteStore.syncDiscoveredModelCatalogFlags(
            providerID: providerID,
            normalizedProviderID: normalizedProviderID(_:),
            modelCatalogIdentifier: { [weak self] providerID, modelKey in
                guard let self else {
                    return "\(providerID)::\(modelKey)"
                }
                return self.modelCatalogIdentifier(providerID: providerID, modelKey: modelKey)
            }
        )
    }

    private func mapModelPolicyRecord(_ record: DaemonModelRoutingPolicyRecord) -> ModelPolicyItem {
        modelsRouteStore.mapModelPolicyRecord(
            record,
            parseDaemonTimestamp: parseDaemonTimestamp(_:)
        )
    }

    private func mapModelRouteSimulationResponse(
        _ response: DaemonModelRouteSimulationResponse
    ) -> ModelRouteSimulationResultItem {
        modelsRouteStore.mapModelRouteSimulationResponse(
            response,
            workspaceID: workspaceID
        )
    }

    private func mapModelRouteExplainResponse(
        _ response: DaemonModelRouteExplainResponse
    ) -> ModelRouteExplainResultItem {
        modelsRouteStore.mapModelRouteExplainResponse(
            response,
            workspaceID: workspaceID
        )
    }

    private func modelRouteSimulationSummaryMessage(_ item: ModelRouteSimulationResultItem) -> String {
        modelsRouteStore.modelRouteSimulationSummaryMessage(
            item,
            providerDisplayName: providerDisplayName(_:)
        )
    }

    private func modelRouteExplainSummaryMessage(_ item: ModelRouteExplainResultItem) -> String {
        modelsRouteStore.modelRouteExplainSummaryMessage(item)
    }

    private func mapProviderReadinessItem(
        providerID: String,
        config: DaemonProviderConfigRecord?,
        check: DaemonProviderCheckItem?
    ) -> ProviderReadinessItem {
        let endpoint = nonEmpty(config?.endpoint)
            ?? Self.providerDefaultEndpoints[providerID]
            ?? "Not configured"

        let status: ProviderReadinessStatus
        let detail: String
        if let check {
            status = check.success ? .healthy : .checkFailed
            detail = nonEmpty(check.message) ?? "No check detail returned."
        } else if let config {
            let apiKeyReady = !providerRequiresAPIKey(providerID) || config.apiKeyConfigured
            if apiKeyReady {
                status = .configured
                detail = "Configured. Run checks to validate connectivity."
            } else {
                status = .missingSetup
                detail = "API key secret is required for this provider."
            }
        } else {
            status = .missingSetup
            detail = "Provider is not configured for this workspace."
        }

        let updatedAtLabel: String
        if let config {
            updatedAtLabel = formattedProviderTimestamp(config.updatedAt)
        } else {
            updatedAtLabel = "n/a"
        }

        return ProviderReadinessItem(
            id: providerID,
            provider: providerID,
            endpoint: endpoint,
            status: status,
            detail: detail,
            updatedAtLabel: updatedAtLabel
        )
    }

    private func mapWorkflowRoute(_ route: DaemonWorkflowRouteMetadata?) -> WorkflowRouteContext {
        guard let route else {
            return WorkflowRouteContext()
        }
        let taskClass = nonEmpty(route.taskClass)
        let provider = nonEmpty(route.provider)
        let modelKey = nonEmpty(route.modelKey)
        let taskClassSource = nonEmpty(route.taskClassSource)
        let routeSource = nonEmpty(route.routeSource)
        let notes = nonEmpty(route.notes)
        return WorkflowRouteContext(
            available: route.available || taskClass != nil || provider != nil || modelKey != nil,
            taskClass: taskClass,
            provider: provider,
            modelKey: modelKey,
            taskClassSource: taskClassSource,
            routeSource: routeSource,
            notes: notes
        )
    }

    private func automationRouteSummary(_ route: WorkflowRouteContext) -> String? {
        let provider = nonEmpty(route.provider)
        let modelKey = nonEmpty(route.modelKey)
        let taskClass = nonEmpty(route.taskClass)

        if let provider, let modelKey {
            return "\(provider)/\(modelKey)"
        }
        if let provider {
            return provider
        }
        if let modelKey {
            return modelKey
        }
        if let taskClass {
            return taskClass
        }
        return nil
    }

    private enum ChannelDiagnosticsExecution {
        case refreshStatus
        case openSection(section: AppSection, message: String)
        case openSystemSettings(channelID: String, destination: String?)
        case channelSetup
        case daemonLifecycleControl(action: String)
    }

    private enum ConnectorDiagnosticsExecution {
        case refreshStatus
        case openSection(section: AppSection, message: String)
        case requestPermission(connectorID: String)
        case openSystemSettings(connectorID: String, destination: String?)
        case daemonLifecycleControl(action: String)
    }

    private func resolveChannelDiagnosticsExecution(
        channelID: String,
        action: DiagnosticsActionItem
    ) -> ChannelDiagnosticsExecution? {
        let intent = normalizedDiagnosticsActionIntent(
            intent: action.intent,
            identifier: action.id,
            destination: action.destination
        )
        switch intent {
        case "refresh_status":
            return .refreshStatus
        case "open_system_settings":
            return .openSystemSettings(
                channelID: normalizedChannelIdentifier(channelID, parameters: action.parameters),
                destination: action.destination
            )
        case "navigate":
            if let components = diagnosticsDestinationComponents(action.destination),
               components.host?.caseInsensitiveCompare("system-settings") == .orderedSame {
                return .openSystemSettings(
                    channelID: normalizedChannelIdentifier(channelID, parameters: action.parameters),
                    destination: action.destination
                )
            }
            if let section = diagnosticsDestinationSection(action.destination) {
                return .openSection(
                    section: section,
                    message: channelDiagnosticsNavigationMessage(
                        section: section,
                        channelID: channelID,
                        parameters: action.parameters
                    )
                )
            }
            switch action.id {
            case "open_channel_logs":
                return .openSection(
                    section: .inspect,
                    message: "Opened Inspect for channel diagnostics."
                )
            case "open_channel_setup", "configure_twilio_channel":
                return .channelSetup
            default:
                return nil
            }
        case "daemon_lifecycle_control":
            guard let daemonAction = normalizedDaemonLifecycleAction(action.parameters["action"]) else {
                return nil
            }
            return .daemonLifecycleControl(action: daemonAction)
        default:
            return nil
        }
    }

    private func resolveConnectorDiagnosticsExecution(
        connectorID: String,
        action: DiagnosticsActionItem
    ) -> ConnectorDiagnosticsExecution? {
        let intent = normalizedDiagnosticsActionIntent(
            intent: action.intent,
            identifier: action.id,
            destination: action.destination
        )
        switch intent {
        case "refresh_status":
            return .refreshStatus
        case "request_permission":
            return .requestPermission(
                connectorID: normalizedConnectorIdentifier(connectorID, parameters: action.parameters)
            )
        case "open_system_settings":
            return .openSystemSettings(
                connectorID: normalizedConnectorIdentifier(connectorID, parameters: action.parameters),
                destination: action.destination
            )
        case "navigate":
            if let components = diagnosticsDestinationComponents(action.destination),
               components.host?.caseInsensitiveCompare("system-settings") == .orderedSame {
                return .openSystemSettings(
                    connectorID: normalizedConnectorIdentifier(connectorID, parameters: action.parameters),
                    destination: action.destination
                )
            }
            if let section = diagnosticsDestinationSection(action.destination) {
                return .openSection(
                    section: section,
                    message: connectorDiagnosticsNavigationMessage(section: section)
                )
            }
            switch action.id {
            case "open_connector_logs":
                return .openSection(
                    section: .inspect,
                    message: "Opened Inspect for connector diagnostics."
                )
            case "request_connector_permission":
                return .requestPermission(connectorID: connectorID)
            case "open_connector_system_settings":
                return .openSystemSettings(connectorID: connectorID, destination: action.destination)
            default:
                return nil
            }
        case "daemon_lifecycle_control":
            guard let daemonAction = normalizedDaemonLifecycleAction(action.parameters["action"]) else {
                return nil
            }
            return .daemonLifecycleControl(action: daemonAction)
        default:
            return nil
        }
    }

    private func channelDiagnosticsNavigationMessage(
        section: AppSection,
        channelID: String,
        parameters: [String: String]
    ) -> String {
        if section == .inspect {
            return "Opened Inspect for channel diagnostics."
        }
        if section == .configuration,
           parameters["channel_family"]?.caseInsensitiveCompare("twilio") == .orderedSame {
            return "Opened Configuration for Twilio channel setup."
        }
        if section == .configuration {
            return "Opened Configuration for channel setup actions."
        }
        let resolvedChannelID = normalizedChannelIdentifier(channelID, parameters: parameters)
        return "Opened \(section.title) for channel \(resolvedChannelID) diagnostics."
    }

    private func connectorDiagnosticsNavigationMessage(section: AppSection) -> String {
        if section == .inspect {
            return "Opened Inspect for connector diagnostics."
        }
        return "Opened \(section.title) for connector diagnostics."
    }

    private func diagnosticsDestinationComponents(_ destination: String?) -> URLComponents? {
        guard let destination = nonEmpty(destination),
              destination.lowercased().hasPrefix("ui://"),
              let components = URLComponents(string: destination) else {
            return nil
        }
        return components
    }

    private func diagnosticsDestinationSection(_ destination: String?) -> AppSection? {
        guard let host = diagnosticsDestinationComponents(destination)?.host?.lowercased() else {
            return nil
        }
        switch host {
        case "configuration":
            return .configuration
        case "chat":
            return .chat
        case "communications":
            return .communications
        case "automation":
            return .automation
        case "approvals":
            return .approvals
        case "tasks":
            return .tasks
        case "inspect":
            return .inspect
        case "channels":
            return .channels
        case "connectors":
            return .connectors
        case "models":
            return .models
        default:
            return nil
        }
    }

    private func normalizedDiagnosticsActionIntent(
        intent: String,
        identifier: String,
        destination: String?
    ) -> String {
        let trimmed = intent.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        if !trimmed.isEmpty {
            return trimmed
        }

        switch identifier {
        case "refresh_channel_status", "refresh_connector_status":
            return "refresh_status"
        case "open_channel_setup", "configure_twilio_channel", "open_channel_logs", "open_connector_logs":
            return "navigate"
        case "repair_daemon_runtime":
            return "daemon_lifecycle_control"
        case "request_connector_permission":
            return "request_permission"
        case "open_connector_system_settings":
            return "open_system_settings"
        default:
            break
        }

        guard let destination = nonEmpty(destination) else {
            return "unknown"
        }
        if destination.hasPrefix("/v1/daemon/lifecycle/control") {
            return "daemon_lifecycle_control"
        }
        if destination.hasPrefix("/v1/") {
            return "refresh_status"
        }
        if destination.lowercased().hasPrefix("ui://system-settings") {
            return "open_system_settings"
        }
        if destination.lowercased().hasPrefix("ui://connectors/request-permission/") {
            return "request_permission"
        }
        if destination.lowercased().hasPrefix("ui://") {
            return "navigate"
        }
        return "unknown"
    }

    private func normalizedDaemonLifecycleAction(_ value: String?) -> String? {
        guard let value = nonEmpty(value) else {
            return "repair"
        }
        switch value.lowercased() {
        case "start", "stop", "restart", "install", "uninstall", "repair":
            return value.lowercased()
        default:
            return nil
        }
    }

    private func normalizedChannelIdentifier(_ fallbackChannelID: String, parameters: [String: String]) -> String {
        let resolved = nonEmpty(parameters["channel_id"]) ?? fallbackChannelID
        return Self.canonicalLogicalChannelID(from: resolved)
    }

    private func normalizedConnectorIdentifier(_ fallbackConnectorID: String, parameters: [String: String]) -> String {
        nonEmpty(parameters["connector_id"]) ?? fallbackConnectorID
    }

    private func defaultConnectorIDForChannelSystemSettings(channelID: String) -> String {
        let normalizedChannelID = Self.canonicalLogicalChannelID(from: channelID)
        let mappings = channelConnectorMappings(channelID: normalizedChannelID)
            .sorted { lhs, rhs in
                if lhs.priority == rhs.priority {
                    return lhs.connectorID.localizedCaseInsensitiveCompare(rhs.connectorID) == .orderedAscending
                }
                return lhs.priority < rhs.priority
            }
        if let enabled = mappings.first(where: \.enabled)?.connectorID {
            return normalizedChannelConnectorMappingConnectorID(enabled)
        }
        if let firstMapped = mappings.first?.connectorID {
            return normalizedChannelConnectorMappingConnectorID(firstMapped)
        }
        if let logicalCard = logicalChannelCards.first(where: {
            normalizedChannelConnectorMappingChannelID($0.channelID) == normalizedChannelID
        }) {
            if let primaryConnectorID = logicalCard.mappedConnectorRollups.first?.connectorID {
                return normalizedChannelConnectorMappingConnectorID(primaryConnectorID)
            }
        }
        if let channelCard = channelCards.first(where: {
            normalizedChannelConnectorMappingChannelID($0.logicalChannelID) == normalizedChannelID
        }) {
            if let primaryConnectorID = nonEmpty(channelCard.primaryConnectorID) {
                return normalizedChannelConnectorMappingConnectorID(primaryConnectorID)
            }
            if let fallbackConnectorID = (channelCard.enabledConnectorIDs + channelCard.mappedConnectorIDs).first {
                return normalizedChannelConnectorMappingConnectorID(fallbackConnectorID)
            }
        }
        switch normalizedChannelID {
        case "message":
            return "imessage"
        case "app":
            return "builtin.app"
        case "voice":
            return "twilio"
        default:
            return normalizedChannelConnectorMappingConnectorID(normalizedChannelID)
        }
    }

    private func mapDiagnosticsAction(_ action: DaemonDiagnosticsRemediationAction) -> DiagnosticsActionItem {
        let normalizedParameters = action.parameters.reduce(into: [String: String]()) { partialResult, entry in
            let key = entry.key.trimmingCharacters(in: .whitespacesAndNewlines)
            let value = entry.value.trimmingCharacters(in: .whitespacesAndNewlines)
            guard !key.isEmpty else {
                return
            }
            partialResult[key] = value
        }
        return DiagnosticsActionItem(
            id: action.identifier,
            title: nonEmpty(action.label) ?? action.identifier,
            intent: normalizedDiagnosticsActionIntent(
                intent: action.intent,
                identifier: action.identifier,
                destination: action.destination
            ),
            destination: nonEmpty(action.destination),
            parameters: normalizedParameters,
            enabled: action.enabled,
            recommended: action.recommended,
            reason: nonEmpty(action.reason)
        )
    }

    private struct ConfigurationFieldMapping {
        let editable: [String: String]
        let editableKinds: [String: ConfigurationDraftValueKind]
        let descriptors: [ConfigurationFieldDescriptorItem]
        let readOnly: [String: String]
    }

    private func mapConfigurationFields(
        _ configuration: [String: DaemonJSONValue]?,
        descriptors: [ConfigurationFieldDescriptorItem] = []
    ) -> ConfigurationFieldMapping {
        let configuration = configuration ?? [:]
        var editable: [String: String] = [:]
        var editableKinds: [String: ConfigurationDraftValueKind] = [:]
        var readOnly: [String: String] = [:]
        let descriptorByKey = Dictionary(uniqueKeysWithValues: descriptors.map { ($0.key, $0) })

        for descriptor in descriptors {
            if descriptor.editable {
                editable[descriptor.key] = configuration[descriptor.key]?.displayText ?? ""
                editableKinds[descriptor.key] = descriptor.draftKind
            } else {
                readOnly[descriptor.key] = configuration[descriptor.key]?.displayText
                    ?? (descriptor.writeOnly ? "Hidden by daemon (write-only)." : "Daemon-managed value.")
            }
        }

        for key in configuration.keys.sorted() {
            guard descriptorByKey[key] == nil, let value = configuration[key] else {
                continue
            }
            // When daemon descriptors are present, treat unknown keys as daemon-managed
            // read-only fields to avoid draft churn from runtime status metadata.
            if !descriptors.isEmpty {
                readOnly[key] = value.displayText
                continue
            }
            let kind = configurationDraftKind(from: value)
            if kind.supportsInlineEditing {
                editable[key] = value.displayText
                editableKinds[key] = kind
            } else {
                readOnly[key] = value.displayText
            }
        }

        return ConfigurationFieldMapping(
            editable: editable,
            editableKinds: editableKinds,
            descriptors: descriptors,
            readOnly: readOnly
        )
    }

    private func mapConfigurationFieldDescriptors(
        _ descriptors: [DaemonConfigFieldDescriptor]
    ) -> [ConfigurationFieldDescriptorItem] {
        descriptors.compactMap { descriptor in
            let key = descriptor.key.trimmingCharacters(in: .whitespacesAndNewlines)
            guard !key.isEmpty else {
                return nil
            }
            let normalizedOptions = descriptor.enumOptions
                .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
                .filter { !$0.isEmpty }
            let label = nonEmpty(descriptor.label) ?? key
            return ConfigurationFieldDescriptorItem(
                key: key,
                label: label,
                required: descriptor.required,
                enumOptions: normalizedOptions,
                editable: descriptor.editable,
                secret: descriptor.secret,
                writeOnly: descriptor.writeOnly,
                helpText: nonEmpty(descriptor.helpText),
                draftKind: configurationDraftKind(
                    fromDescriptorType: descriptor.type,
                    enumOptions: normalizedOptions
                )
            )
        }
        .sorted(by: configurationFieldDescriptorSortOrder)
    }

    private func configurationFieldDescriptorSortOrder(
        _ lhs: ConfigurationFieldDescriptorItem,
        _ rhs: ConfigurationFieldDescriptorItem
    ) -> Bool {
        let lhsTuple = (
            lhs.required ? 0 : 1,
            lhs.label.lowercased(),
            lhs.key.lowercased()
        )
        let rhsTuple = (
            rhs.required ? 0 : 1,
            rhs.label.lowercased(),
            rhs.key.lowercased()
        )
        return lhsTuple < rhsTuple
    }

    private func synthesizedConfigurationFieldDescriptors(
        editableConfiguration: [String: String],
        editableConfigurationKinds: [String: ConfigurationDraftValueKind]
    ) -> [ConfigurationFieldDescriptorItem] {
        editableConfiguration.keys.compactMap { rawKey in
            let key = rawKey.trimmingCharacters(in: .whitespacesAndNewlines)
            guard !key.isEmpty else {
                return nil
            }
            return ConfigurationFieldDescriptorItem(
                key: key,
                label: synthesizedConfigurationFieldLabel(key),
                required: false,
                enumOptions: [],
                editable: true,
                secret: false,
                writeOnly: false,
                helpText: "Guided field synthesized from daemon editable configuration metadata.",
                draftKind: editableConfigurationKinds[key]
                    ?? connectionConfigStore.inferConfigurationDraftKind(from: editableConfiguration[key] ?? "")
            )
        }
        .sorted(by: configurationFieldDescriptorSortOrder)
    }

    private func synthesizedConfigurationFieldLabel(_ key: String) -> String {
        let normalized = key
            .replacingOccurrences(of: ".", with: " ")
            .replacingOccurrences(of: "_", with: " ")
            .replacingOccurrences(of: "-", with: " ")
        let collapsed = normalized
            .split(whereSeparator: { $0.isWhitespace })
            .map(String.init)
            .joined(separator: " ")
        let display = nonEmpty(collapsed) ?? key
        return display.capitalized
    }

    private func configurationDraftKind(
        fromDescriptorType rawType: String,
        enumOptions: [String]
    ) -> ConfigurationDraftValueKind {
        if !enumOptions.isEmpty {
            return .string
        }

        switch rawType.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
        case "bool", "boolean":
            return .bool
        case "int", "int32", "int64", "integer", "float", "double", "number":
            return .number
        case "null":
            return .null
        case "array", "list":
            return .array
        case "object", "map":
            return .object
        default:
            return .string
        }
    }

    private func configurationDraftKind(from value: DaemonJSONValue) -> ConfigurationDraftValueKind {
        switch value {
        case .string:
            return .string
        case .number:
            return .number
        case .bool:
            return .bool
        case .null:
            return .null
        case .object:
            return .object
        case .array:
            return .array
        }
    }

    private func mapChannelCard(
        _ card: DaemonChannelStatusCard,
        diagnosticsActions: [DiagnosticsActionItem]
    ) -> ChannelCardItem {
        let descriptors = mapConfigurationFieldDescriptors(card.configFieldDescriptors)
        let mappedConfiguration = mapConfigurationFields(card.configuration?.allValues, descriptors: descriptors)
        let declaredCapabilities = normalizedCapabilityList(card.capabilities ?? [])
        let logicalChannelID = normalizedChannelConnectorMappingChannelID(card.channelID)
        let mappedConnectorIDs = normalizedConnectorIDList(
            from: card.configuration?["mapped_connector_ids"]
        )
        let enabledConnectorIDs = normalizedConnectorIDList(
            from: card.configuration?["enabled_connector_ids"]
        )
        let primaryConnectorID = nonEmpty(
            normalizedChannelConnectorMappingConnectorID(
                card.configuration?["primary_connector_id"]?.displayText ?? ""
            )
        )
        var details: [String: String] = [
            "Channel ID": card.channelID,
            "Category": card.category,
            "Enabled": card.enabled ? "true" : "false",
            "Configured": card.configured ? "true" : "false"
        ]
        let actionReadiness = effectiveChannelActionReadiness(card)
        details["Action Readiness"] = actionReadinessLabel(actionReadiness)
        if let actionBlockerSummary = actionReadinessBlockerSummary(card.actionBlockers) {
            details["Action Blockers"] = actionBlockerSummary
        }
        if !declaredCapabilities.isEmpty {
            details["Capabilities"] = declaredCapabilities.joined(separator: ", ")
        }
        if !mappedConnectorIDs.isEmpty {
            details["Mapped Connector IDs"] = mappedConnectorIDs.joined(separator: ", ")
        }
        if !enabledConnectorIDs.isEmpty {
            details["Enabled Connector IDs"] = enabledConnectorIDs.joined(separator: ", ")
        }
        if let primaryConnectorID {
            details["Primary Connector ID"] = primaryConnectorID
        }
        if let worker = card.worker {
            details["Worker Plugin"] = worker.pluginID
            details["Worker State"] = worker.state
            details["Worker PID"] = worker.processID > 0 ? "\(worker.processID)" : "n/a"
            details["Worker Restarts"] = "\(worker.restartCount)"
            if let lastError = nonEmpty(worker.lastError) {
                details["Worker Last Error"] = lastError
            }
        }

        let unavailableActionReason = diagnosticsActions.isEmpty
            ? "No daemon remediation actions are currently available for this channel."
            : "Use daemon-declared remediation actions for setup and health troubleshooting."

        return ChannelCardItem(
            id: card.channelID,
            name: card.displayName,
            logicalChannelID: logicalChannelID,
            mappedConnectorIDs: mappedConnectorIDs,
            enabledConnectorIDs: enabledConnectorIDs,
            primaryConnectorID: primaryConnectorID,
            declaredCapabilities: declaredCapabilities,
            status: channelCardStatus(from: card.status, actionReadiness: card.actionReadiness),
            summary: nonEmpty(card.summary) ?? "No channel summary reported.",
            details: details,
            editableConfiguration: mappedConfiguration.editable,
            editableConfigurationKinds: mappedConfiguration.editableKinds,
            configurationFieldDescriptors: mappedConfiguration.descriptors,
            readOnlyConfiguration: mappedConfiguration.readOnly,
            actions: diagnosticsActions,
            unavailableActionReason: unavailableActionReason
        )
    }

    private func mapConnectorCard(
        _ card: DaemonConnectorStatusCard,
        diagnosticsActions: [DiagnosticsActionItem],
        permissionState: ConnectorPermissionState,
        isExpanded: Bool
    ) -> ConnectorCardItem {
        let descriptors = mapConfigurationFieldDescriptors(card.configFieldDescriptors)
        let mappedConfiguration = mapConfigurationFields(card.configuration?.allValues, descriptors: descriptors)
        let declaredCapabilities = normalizedCapabilityList(card.capabilities ?? [])
        let logicalConnectorID = nonEmpty(
            normalizedChannelConnectorMappingConnectorID(
                connectorConfigurationValue(card, key: "logical_connector_id") ?? card.connectorID
            )
        ) ?? normalizedChannelConnectorMappingConnectorID(card.connectorID)
        let requiresPermission = connectorRequiresPermission(
            card,
            diagnosticsActions: diagnosticsActions,
            declaredCapabilities: declaredCapabilities
        )
        var details: [String: String] = [
            "Connector ID": card.connectorID,
            "Plugin ID": card.pluginID,
            "Enabled": card.enabled ? "true" : "false",
            "Configured": card.configured ? "true" : "false"
        ]
        let connectorHealth = connectorHealthStatus(card, permissionState: permissionState)
        let actionReadiness = effectiveConnectorActionReadiness(card, connectorHealth: connectorHealth)
        details["Action Readiness"] = actionReadinessLabel(actionReadiness)
        if let actionBlockerSummary = actionReadinessBlockerSummary(card.actionBlockers) {
            details["Action Blockers"] = actionBlockerSummary
        }
        if !declaredCapabilities.isEmpty {
            details["Capabilities"] = declaredCapabilities.joined(separator: ", ")
        }
        if let worker = card.worker {
            details["Worker State"] = worker.state
            details["Worker PID"] = worker.processID > 0 ? "\(worker.processID)" : "n/a"
            details["Worker Restarts"] = "\(worker.restartCount)"
            if let lastError = nonEmpty(worker.lastError) {
                details["Worker Last Error"] = lastError
            }
        }

        let unavailableActionReason = diagnosticsActions.isEmpty
            ? "No daemon remediation actions are currently available for this connector."
            : "Use daemon-declared remediation actions for setup and runtime troubleshooting."

        return ConnectorCardItem(
            id: card.connectorID,
            name: card.displayName,
            logicalConnectorID: logicalConnectorID,
            declaredCapabilities: declaredCapabilities,
            requiresPermission: requiresPermission,
            health: connectorHealth,
            permissionState: permissionState,
            permissionScope: connectorPermissionScope(
                for: card,
                declaredCapabilities: declaredCapabilities,
                diagnosticsActions: diagnosticsActions,
                requiresPermission: requiresPermission
            ),
            statusReason: nonEmpty(connectorStatusReason(for: card)),
            summary: nonEmpty(card.summary) ?? "No connector summary reported.",
            details: details,
            editableConfiguration: mappedConfiguration.editable,
            editableConfigurationKinds: mappedConfiguration.editableKinds,
            configurationFieldDescriptors: mappedConfiguration.descriptors,
            readOnlyConfiguration: mappedConfiguration.readOnly,
            actions: diagnosticsActions,
            unavailableActionReason: unavailableActionReason,
            isExpanded: isExpanded
        )
    }

    private func approvalDecisionState(from raw: String) -> ApprovalInboxDecisionState {
        switch raw.lowercased() {
        case "pending":
            return .pending
        default:
            return .final
        }
    }

    private func approvalDecisionOutcome(
        from rawDecision: String?,
        state: ApprovalInboxDecisionState
    ) -> ApprovalInboxDecisionOutcome? {
        guard state == .final else {
            return nil
        }
        let value = nonEmpty(rawDecision)?.lowercased() ?? ""
        switch value {
        case "approved", "accepted", "accept":
            return .approved
        case "rejected", "denied", "reject":
            return .rejected
        case "":
            return .other("finalized")
        default:
            return .other(value)
        }
    }

    private func approvalRiskLevel(from raw: String) -> ApprovalInboxRiskLevel {
        switch raw.lowercased() {
        case "destructive":
            return .destructive
        case "policy":
            return .policy
        default:
            return .other(raw)
        }
    }

    private func formattedWorkflowTimestamp(_ value: String) -> String {
        guard let parsed = parseDaemonTimestamp(value) else {
            return nonEmpty(value) ?? "n/a"
        }
        return parsed.formatted(date: .abbreviated, time: .shortened)
    }

    private func normalizedWorkflowStateLabel(_ raw: String?) -> String? {
        guard let raw = nonEmpty(raw) else {
            return nil
        }
        return raw
            .replacingOccurrences(of: "_", with: " ")
            .capitalized
    }

    private func taskRunWorkflowState(from rawState: String) -> TaskRunWorkflowState {
        switch rawState.lowercased() {
        case "queued":
            return .queued
        case "planning":
            return .planning
        case "awaiting_approval":
            return .awaitingApproval
        case "running":
            return .running
        case "blocked":
            return .blocked
        case "completed":
            return .completed
        case "failed":
            return .failed
        case "cancelled", "canceled":
            return .cancelled
        default:
            return .unknown(rawState)
        }
    }

    private func workflowPriorityLabel(_ priority: Int) -> String {
        switch priority {
        case ..<2:
            return "Priority Low"
        case 2:
            return "Priority Medium"
        default:
            return "Priority High"
        }
    }

    private func resolvedAuthToken() -> String? {
        guard let daemonAuthToken else {
            return nil
        }
        let trimmed = daemonAuthToken.trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? nil : trimmed
    }

    private func recordStatusNotification(
        source: String,
        oldValue: String?,
        newValue: String?
    ) {
        notificationStore.recordStatusNotification(
            workspaceID: nonEmpty(workspaceID) ?? Self.defaultWorkspaceID,
            source: source,
            oldValue: oldValue,
            newValue: newValue
        )
    }

    private func appendNotification(
        source: String,
        action: String,
        message: String,
        level: AppNotificationLevel
    ) {
        notificationStore.postNotification(
            workspaceID: nonEmpty(workspaceID) ?? Self.defaultWorkspaceID,
            source: source,
            action: action,
            message: message,
            level: level
        )
    }

    private func loadPersistedNotifications() {
        notificationStore.loadPersistedNotifications()
    }

    private func currentPanelFilterWorkspaceID() -> String {
        contextRetentionStore.currentWorkspaceID(for: workspaceID)
    }

    private func setCurrentWorkspaceInformationDensityMode(_ mode: AppInformationDensityMode) {
        contextRetentionStore.setInformationDensityMode(mode, for: workspaceID)
    }

    private func applyWorkspaceScopedInformationDensityMode(for workspaceID: String?) {
        let mode = contextRetentionStore.informationDensityMode(for: workspaceID)
        if informationDensityMode != mode {
            informationDensityMode = mode
        }
    }

    private func currentWorkspacePanelFilterContext() -> WorkspacePanelFilterContext {
        contextRetentionStore.panelFilterContext(for: workspaceID)
    }

    private func updateCurrentWorkspacePanelFilterContext(
        _ mutate: (inout WorkspacePanelFilterContext) -> Void
    ) {
        contextRetentionStore.updatePanelFilterContext(for: workspaceID, mutate)
    }

    private func currentWorkspaceContinuityContext() -> WorkspaceContinuityContext {
        contextRetentionStore.workspaceContinuityContext(for: workspaceID)
    }

    private func updateCurrentWorkspaceContinuityContext(
        _ mutate: (inout WorkspaceContinuityContext) -> Void
    ) {
        contextRetentionStore.updateWorkspaceContinuityContext(for: workspaceID, mutate)
    }

    private func persistPanelFilterContexts() {
        contextRetentionStore.persistPanelFilterContexts()
    }

    private func loadPersistedPanelFilterContexts() {
        contextRetentionStore.loadPersistedPanelFilterContexts()
    }

    private func persistCommunicationsTriageContexts() {
        contextRetentionStore.persistCommunicationsTriageContexts()
    }

    private func loadPersistedCommunicationsTriageContexts() {
        contextRetentionStore.loadPersistedCommunicationsTriageContexts()
    }

    private func persistWorkspaceContinuityContexts() {
        contextRetentionStore.persistWorkspaceContinuityContexts()
    }

    private func loadPersistedWorkspaceContinuityContexts() {
        contextRetentionStore.loadPersistedWorkspaceContinuityContexts()
    }

    private func persistHomeFirstSessionProgress() {
        contextRetentionStore.persistHomeFirstSessionProgress()
    }

    private func loadPersistedHomeFirstSessionProgress() {
        contextRetentionStore.loadPersistedHomeFirstSessionProgress()
    }

    private func persistInformationDensityModes() {
        contextRetentionStore.persistInformationDensityModes()
    }

    private func loadPersistedInformationDensityModes() {
        contextRetentionStore.loadPersistedInformationDensityModes()
    }

    private func recordAppCommandUsage(_ actionID: AppCommandActionID) {
        commandHistoryStore.recordUsage(actionID)
    }

    private func loadPersistedRecentAppCommands() {
        commandHistoryStore.loadPersistedUsage()
    }

    private var missingReadyChatModelRouteGuidance: String {
        "No enabled chat model is ready. Open Models, configure a provider, enable a model, and save a chat route policy."
    }

    enum UserFacingPanelErrorContext: Hashable {
        case chat
        case models
        case channels
        case connectors
        case automation
        case approvals
        case tasks

        var sectionTitle: String {
            switch self {
            case .chat:
                return "Chat"
            case .models:
                return "Models"
            case .channels:
                return "Channels"
            case .connectors:
                return "Connectors"
            case .automation:
                return "Automation"
            case .approvals:
                return "Approvals"
            case .tasks:
                return "Tasks"
            }
        }

        var setupDestinationTitle: String {
            switch self {
            case .chat:
                return "Models"
            case .models:
                return "Configuration"
            case .channels:
                return "Channels"
            case .connectors:
                return "Connectors"
            case .automation, .approvals, .tasks:
                return "Configuration"
            }
        }

        var section: AppSection {
            switch self {
            case .chat:
                return .chat
            case .models:
                return .models
            case .channels:
                return .channels
            case .connectors:
                return .connectors
            case .automation:
                return .automation
            case .approvals:
                return .approvals
            case .tasks:
                return .tasks
            }
        }
    }

    private func selectedActorIDForChatSubmission() -> String? {
        let actorID = nonEmpty(selectedPrincipal)
        if actorID == "default" {
            return nil
        }
        return actorID
    }

    private func chatRealtimeTransportConnectedForActiveTurn() -> Bool {
        chatOrchestrationStore.realtimeConnectedForActiveTurn
            && chatOrchestrationStore.realtimeFallbackReason == nil
    }

    private func applyChatRealtimeFallbackContext(_ context: ChatRealtimeFallbackContext) {
        chatOrchestrationStore.realtimeFallbackReason = context.reason
        chatOrchestrationStore.realtimeFallbackDetail = context.remediationDetail
        connectionStatus = .degraded
        chatStatusMessage = context.statusMessage
        chatProgressDetail = context.progressDetail
    }

    private func chatRealtimeFallbackContext(
        from error: Error,
        defaultReason: ChatRealtimeFallbackReason = .unavailable
    ) -> ChatRealtimeFallbackContext {
        if let daemonError = error as? DaemonAPIError {
            if daemonError.isUnauthorized || daemonError.serverStatusCode == 401 {
                return ChatRealtimeFallbackContext(
                    reason: .unauthorized,
                    statusMessage: "Realtime auth was rejected. Continuing in fallback mode.",
                    progressDetail: "Realtime auth failed; waiting for one-shot daemon response.",
                    remediationDetail: "Realtime auth failed. Verify Assistant Access Token scope, then retry realtime.",
                    notificationSummary: "Realtime auth failed; fallback mode is active."
                )
            }
            if daemonError.serverStatusCode == 429 || daemonError.serverCode == "rate_limit_exceeded" {
                return ChatRealtimeFallbackContext(
                    reason: .capacityRejected,
                    statusMessage: "Realtime capacity reached. Continuing in fallback mode.",
                    progressDetail: "Realtime session was rejected due to capacity; waiting for one-shot daemon response.",
                    remediationDetail: "Realtime session capacity is currently full. Wait for active streams to clear, then retry realtime.",
                    notificationSummary: "Realtime capacity reached; fallback mode is active."
                )
            }
            let normalizedMessage = nonEmpty(daemonError.errorDescription)?.lowercased() ?? ""
            if normalizedMessage.contains("realtime_stale_session")
                || normalizedMessage.contains("heartbeat")
                || normalizedMessage.contains("pong")
                || normalizedMessage.contains("timed out")
            {
                return ChatRealtimeFallbackContext(
                    reason: .staleSession,
                    statusMessage: "Realtime session expired. Continuing in fallback mode.",
                    progressDetail: "Realtime session became stale; waiting for one-shot daemon response.",
                    remediationDetail: "Realtime session heartbeat timed out. Retry realtime to restore streaming.",
                    notificationSummary: "Realtime session expired; fallback mode is active."
                )
            }
            if daemonError.isConnectivityIssue || normalizedMessage.contains("disconnected") || normalizedMessage.contains("not connected") {
                return ChatRealtimeFallbackContext(
                    reason: .disconnected,
                    statusMessage: "Realtime stream disconnected. Continuing in fallback mode.",
                    progressDetail: "Realtime connection dropped; waiting for one-shot daemon response.",
                    remediationDetail: "Realtime connection dropped. Retry realtime after confirming runtime connectivity.",
                    notificationSummary: "Realtime stream disconnected; fallback mode is active."
                )
            }
        }

        switch defaultReason {
        case .unauthorized:
            return ChatRealtimeFallbackContext(
                reason: .unauthorized,
                statusMessage: "Realtime auth was rejected. Continuing in fallback mode.",
                progressDetail: "Realtime auth failed; waiting for one-shot daemon response.",
                remediationDetail: "Realtime auth failed. Verify Assistant Access Token scope, then retry realtime.",
                notificationSummary: "Realtime auth failed; fallback mode is active."
            )
        case .capacityRejected:
            return ChatRealtimeFallbackContext(
                reason: .capacityRejected,
                statusMessage: "Realtime capacity reached. Continuing in fallback mode.",
                progressDetail: "Realtime session was rejected due to capacity; waiting for one-shot daemon response.",
                remediationDetail: "Realtime session capacity is currently full. Wait for active streams to clear, then retry realtime.",
                notificationSummary: "Realtime capacity reached; fallback mode is active."
            )
        case .staleSession:
            return ChatRealtimeFallbackContext(
                reason: .staleSession,
                statusMessage: "Realtime session expired. Continuing in fallback mode.",
                progressDetail: "Realtime session became stale; waiting for one-shot daemon response.",
                remediationDetail: "Realtime session heartbeat timed out. Retry realtime to restore streaming.",
                notificationSummary: "Realtime session expired; fallback mode is active."
            )
        case .disconnected:
            return ChatRealtimeFallbackContext(
                reason: .disconnected,
                statusMessage: "Realtime stream disconnected. Continuing in fallback mode.",
                progressDetail: "Realtime connection dropped; waiting for one-shot daemon response.",
                remediationDetail: "Realtime connection dropped. Retry realtime after confirming runtime connectivity.",
                notificationSummary: "Realtime stream disconnected; fallback mode is active."
            )
        case .unavailable:
            return ChatRealtimeFallbackContext(
                reason: .unavailable,
                statusMessage: "Realtime unavailable. Continuing in fallback mode.",
                progressDetail: "Realtime unavailable; waiting for one-shot daemon response.",
                remediationDetail: "Realtime stream is unavailable. Retry realtime or refresh daemon status.",
                notificationSummary: "Realtime unavailable; fallback mode is active."
            )
        }
    }

    private func chatRealtimeFallbackContext(
        fromRealtimeEventCode errorCode: String?,
        message: String?,
        defaultReason: ChatRealtimeFallbackReason
    ) -> ChatRealtimeFallbackContext {
        let normalizedCode = nonEmpty(errorCode)?.lowercased() ?? ""
        let normalizedMessage = nonEmpty(message)?.lowercased() ?? ""
        if normalizedCode == "rate_limit_exceeded"
            || normalizedMessage.contains("capacity")
            || normalizedMessage.contains("rate limit")
        {
            return chatRealtimeFallbackContext(
                from: DaemonAPIError.server(statusCode: 429, message: message ?? "Realtime capacity exceeded."),
                defaultReason: .capacityRejected
            )
        }
        if normalizedCode == "auth_scope"
            || normalizedCode == "unauthorized"
            || normalizedMessage.contains("auth")
            || normalizedMessage.contains("unauthorized")
        {
            return chatRealtimeFallbackContext(
                from: DaemonAPIError.server(statusCode: 401, message: message ?? "Realtime auth failed."),
                defaultReason: .unauthorized
            )
        }
        if normalizedMessage.contains("stale")
            || normalizedMessage.contains("heartbeat")
            || normalizedMessage.contains("pong")
        {
            return chatRealtimeFallbackContext(
                from: DaemonAPIError.transport("realtime_stale_session: \(message ?? "heartbeat timeout")"),
                defaultReason: .staleSession
            )
        }
        return chatRealtimeFallbackContext(
            from: DaemonAPIError.transport(message ?? "Realtime stream error."),
            defaultReason: defaultReason
        )
    }

    private func chatRealtimeErrorSummaryForDisplay(
        errorCode: String?,
        message: String?
    ) -> String {
        let normalizedCode = nonEmpty(errorCode)?
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .lowercased() ?? ""
        let normalizedMessage = nonEmpty(message)?
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .lowercased() ?? ""

        if normalizedCode == "auth_scope"
            || normalizedCode == "unauthorized"
            || normalizedMessage.contains("unauthorized")
            || normalizedMessage.contains("auth")
            || normalizedMessage.contains("token scope")
        {
            return "Realtime auth failed before daemon receipt completed this turn."
        }

        if normalizedCode == "rate_limit_exceeded"
            || normalizedMessage.contains("capacity")
            || normalizedMessage.contains("rate limit")
        {
            return "Realtime capacity was reached before daemon receipt completed this turn."
        }

        if normalizedMessage.contains("context canceled")
            || normalizedMessage.contains("context cancelled")
            || normalizedMessage.contains("request canceled")
            || normalizedMessage.contains("request cancelled")
            || normalizedMessage.contains("begin tx")
            || normalizedMessage.contains("transaction")
            || normalizedMessage.contains("database is locked")
        {
            return "Realtime stream was interrupted before daemon receipt completed this turn."
        }

        if normalizedMessage.contains("disconnected")
            || normalizedMessage.contains("not connected")
            || normalizedMessage.contains("connection refused")
            || normalizedMessage.contains("connection reset")
            || normalizedMessage.contains("broken pipe")
            || normalizedMessage.contains("timed out")
            || normalizedMessage.contains("deadline exceeded")
            || normalizedMessage == "eof"
        {
            return "Realtime stream disconnected before daemon receipt completed this turn."
        }

        return "Realtime stream reported an error before daemon receipt completed this turn."
    }

    private func chatTurnCompletionStatusMessage(
        response: DaemonChatTurnResponse,
        realtimeConnected: Bool
    ) -> String {
        let signals = chatTurnContextStore.chatTurnSignals(from: response.items)
        let transportLabel = realtimeConnected ? "Realtime" : "Fallback"
        if signals.clarificationRequired {
            let prompt = nonEmpty(signals.clarificationPrompt) ?? "additional details are required"
            return "Action needs clarification: \(prompt)"
        }
        if signals.approvalRequired {
            if let approvalRequestID = nonEmpty(signals.approvalRequestID) {
                return "Action awaiting approval (\(approvalRequestID)) • \(transportLabel)"
            }
            return "Action awaiting approval • \(transportLabel)"
        }

        let hasToolCalls = response.items.contains {
            $0.type.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() == "tool_call"
        }
        let hasToolFailure = response.items.contains { item in
            item.type.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() == "tool_result"
                && ChatTimelineEventStore.normalizedTimelineItemState(item.status) == .failed
        }
        if hasToolFailure {
            return "Tool execution failed • \(transportLabel)"
        }
        if hasToolCalls {
            let runState = nonEmpty(response.taskRunCorrelation.runState)?
                .replacingOccurrences(of: "_", with: " ")
                .capitalized ?? "Completed"
            return "Action run state: \(runState) • \(transportLabel)"
        }
        return "Response complete • \(transportLabel)"
    }

    private func chatTurnNotificationSummary(
        response: DaemonChatTurnResponse,
        realtimeConnected: Bool
    ) -> String {
        let signals = chatTurnContextStore.chatTurnSignals(from: response.items)
        if signals.approvalRequired {
            if let approvalRequestID = nonEmpty(signals.approvalRequestID) {
                return "Action awaiting approval (\(approvalRequestID)) (\(realtimeConnected ? "realtime" : "fallback"))."
            }
            return "Action awaiting approval (\(realtimeConnected ? "realtime" : "fallback"))."
        }
        let runID = nonEmpty(response.taskRunCorrelation.runID)
        if let runID {
            return "Completed action run \(runID) via \(response.provider) • \(response.modelKey) (\(realtimeConnected ? "realtime" : "fallback"))."
        }
        return "Sent chat turn via \(response.provider) • \(response.modelKey) (\(realtimeConnected ? "realtime" : "fallback"))."
    }

    private func effectiveChatRouteContext() -> WorkflowRouteContext {
        let hasLiveTraceability = chatLatestTurnTraceability != nil
        let taskClass = nonEmpty(chatLatestTurnTraceability?.taskClass)
            ?? nonEmpty(chatLatestTurnExplainability?.taskClass)
            ?? ((hasLiveTraceability || modelRouteSummary != nil || chatActiveCorrelationID != nil || isChatStreaming) ? "chat" : nil)
        let provider = nonEmpty(chatLatestTurnTraceability?.provider)
            ?? nonEmpty(chatLatestTurnExplainability?.selectedProvider)
            ?? nonEmpty(modelRouteSummary?.provider)
        let modelKey = nonEmpty(chatLatestTurnTraceability?.modelKey)
            ?? nonEmpty(chatLatestTurnExplainability?.selectedModelKey)
            ?? nonEmpty(modelRouteSummary?.modelKey)
        let routeSource = nonEmpty(chatLatestTurnTraceability?.routeSource)
            ?? nonEmpty(chatLatestTurnExplainability?.selectedSource)
            ?? nonEmpty(modelRouteSummary?.source)
        let hasContext = taskClass != nil || provider != nil || modelKey != nil || routeSource != nil
        return WorkflowRouteContext(
            available: hasContext,
            taskClass: taskClass,
            provider: provider,
            modelKey: modelKey,
            taskClassSource: taskClass == nil ? nil : "chat.turn",
            routeSource: routeSource,
            notes: nonEmpty(chatLatestTurnExplainability?.routeSummary) ?? nonEmpty(modelRouteSummary?.notes)
        )
    }

    private func isMissingReadyChatModelRoute(_ error: Error) -> Bool {
        guard let daemonError = error as? DaemonAPIError else {
            return false
        }
        return daemonError.isMissingReadyChatModelRoute
    }

    private func isPanelProblemRetryInFlight(for section: AppSection) -> Bool {
        switch section {
        case .chat:
            return isChatStreaming || isChatInterruptInFlight
        case .models:
            return isProviderStatusLoading
        case .channels:
            return isChannelsLoading || isChannelConnectorMappingsLoading
        case .connectors:
            return isConnectorsLoading
        case .automation:
            return isAutomationLoading || isAutomationFireHistoryLoading
        case .approvals:
            return isApprovalsLoading
        case .tasks:
            return isTasksLoading || isTaskRunDetailLoading
        default:
            return false
        }
    }

    private func clearPanelProblemSignal(for section: AppSection) {
        panelProblemStore.clearSignal(for: section)
    }

    private func typedPanelProblemRemediationMessage(
        daemonError: DaemonAPIError,
        panelContext: UserFacingPanelErrorContext
    ) -> String? {
        panelProblemStore.typedRemediationMessage(
            daemonError: daemonError,
            section: panelContext.section,
            sectionTitle: panelContext.sectionTitle
        )
    }

    private func chatSendFailureGuidance(_ error: Error) -> String {
        guard let daemonError = error as? DaemonAPIError else {
            clearPanelProblemSignal(for: .chat)
            return "Chat request failed. Refresh daemon status and retry."
        }
        if let message = typedPanelProblemRemediationMessage(
            daemonError: daemonError,
            panelContext: .chat
        ) {
            return message
        }
        clearPanelProblemSignal(for: .chat)
        if daemonError.isUnauthorized {
            return "Chat request failed authentication. Open Configuration, save a valid token, then retry."
        }
        if daemonError.isConnectivityIssue {
            return "Chat request could not reach daemon. Start or refresh daemon status, then retry."
        }
        if daemonError.serverCode == "service_not_configured" {
            return "Chat service setup is incomplete. Open Models, finish provider/model route setup, then retry."
        }
        if daemonError.serverCode == "resource_not_found",
           let message = daemonError.serverMessage?.lowercased(),
           message.contains("unknown control route") {
            return "Chat route endpoint is unavailable in this daemon build. Update app/daemon and retry."
        }
        return "Chat request failed. Review runtime status and retry."
    }

    private func daemonErrorMessage(
        _ error: Error,
        fallbackContext: String,
        updateConnectionStatus: Bool = true,
        panelContext: UserFacingPanelErrorContext? = nil
    ) -> String {
        if let panelContext {
            return userFacingPanelErrorMessage(
                error,
                panelContext: panelContext,
                fallbackContext: fallbackContext,
                updateConnectionStatus: updateConnectionStatus
            )
        }

        if let daemonError = error as? DaemonAPIError {
            if daemonError.isUnauthorized {
                if updateConnectionStatus {
                    connectionStatus = .degraded
                }
                return "Daemon auth failed. Check Assistant Access Token."
            }
            if daemonError.isConnectivityIssue {
                if updateConnectionStatus {
                    connectionStatus = .disconnected
                }
                return "Daemon is unreachable at \(daemonBaseURL.absoluteString)."
            }
            if daemonError.isMissingReadyChatModelRoute {
                if updateConnectionStatus {
                    connectionStatus = .connected
                }
                return missingReadyChatModelRouteGuidance
            }
            if updateConnectionStatus {
                connectionStatus = .degraded
            }
            return daemonError.errorDescription ?? fallbackContext
        }
        if updateConnectionStatus {
            connectionStatus = .degraded
        }
        return "\(fallbackContext): \(error.localizedDescription)"
    }

    func panelErrorMessageForTesting(
        _ error: Error,
        panelContext: UserFacingPanelErrorContext
    ) -> String {
        userFacingPanelErrorMessage(
            error,
            panelContext: panelContext,
            fallbackContext: "\(panelContext.sectionTitle) request failed",
            updateConnectionStatus: false
        )
    }

    private func userFacingPanelErrorMessage(
        _ error: Error,
        panelContext: UserFacingPanelErrorContext,
        fallbackContext: String,
        updateConnectionStatus: Bool
    ) -> String {
        if let daemonError = error as? DaemonAPIError {
            if let typedMessage = typedPanelProblemRemediationMessage(
                daemonError: daemonError,
                panelContext: panelContext
            ) {
                if updateConnectionStatus {
                    connectionStatus = .degraded
                }
                return typedMessage
            }
            clearPanelProblemSignal(for: panelContext.section)
            if daemonError.isUnauthorized {
                if updateConnectionStatus {
                    connectionStatus = .degraded
                }
                return "Authentication failed for \(panelContext.sectionTitle). Open Configuration and save a valid Assistant Access Token."
            }
            if daemonError.isConnectivityIssue {
                if updateConnectionStatus {
                    connectionStatus = .disconnected
                }
                return "Could not reach daemon while loading \(panelContext.sectionTitle). Start or repair daemon, then refresh."
            }
            if panelContext == .chat && daemonError.isMissingReadyChatModelRoute {
                if updateConnectionStatus {
                    connectionStatus = .connected
                }
                return missingReadyChatModelRouteGuidance
            }
            if let message = serviceNotConfiguredPanelErrorMessage(
                daemonError,
                panelContext: panelContext
            ) {
                if updateConnectionStatus {
                    connectionStatus = .degraded
                }
                return message
            }
            if let message = unknownRoutePanelErrorMessage(
                daemonError,
                panelContext: panelContext
            ) {
                if updateConnectionStatus {
                    connectionStatus = .degraded
                }
                return message
            }
            if case .decoding = daemonError {
                if updateConnectionStatus {
                    connectionStatus = .degraded
                }
                return "Received an unexpected daemon response while loading \(panelContext.sectionTitle). Refresh and try again."
            }
            if let statusCode = daemonError.serverStatusCode {
                if updateConnectionStatus {
                    connectionStatus = .degraded
                }
                switch statusCode {
                case 400:
                    return "Daemon rejected the \(panelContext.sectionTitle.lowercased()) request. Refresh and retry."
                case 404:
                    return "\(panelContext.sectionTitle) is unavailable because this app and daemon API versions are out of sync. Update and retry."
                case 500...:
                    return "Daemon failed while loading \(panelContext.sectionTitle.lowercased()). Retry, or restart daemon from Configuration."
                default:
                    return "\(panelContext.sectionTitle) is currently unavailable. Refresh and try again."
                }
            }

            if updateConnectionStatus {
                connectionStatus = .degraded
            }
            return "\(panelContext.sectionTitle) is currently unavailable. Refresh and try again."
        }

        clearPanelProblemSignal(for: panelContext.section)
        if updateConnectionStatus {
            connectionStatus = .degraded
        }
        return "\(fallbackContext). Refresh and try again."
    }

    private func serviceNotConfiguredPanelErrorMessage(
        _ daemonError: DaemonAPIError,
        panelContext: UserFacingPanelErrorContext
    ) -> String? {
        guard daemonError.serverCode == "service_not_configured" else {
            return nil
        }
        let serviceLabel = nonEmpty(daemonError.serverDetails?.service?.label)
            ?? nonEmpty(daemonError.serverDetails?.service?.id)?
                .replacingOccurrences(of: "_", with: " ")
                .capitalized
            ?? "Required daemon service"
        let remediationLabel = nonEmpty(daemonError.serverDetails?.remediation?.label)
            ?? "complete setup"

        switch panelContext {
        case .chat:
            return "\(serviceLabel) is not configured yet. Open Models, finish provider/model route setup, then retry chat."
        case .models:
            return "\(serviceLabel) is not configured yet. Open Configuration, \(remediationLabel.lowercased()), then refresh Models."
        case .channels, .connectors, .automation, .approvals, .tasks:
            return "\(serviceLabel) is not configured yet. Open \(panelContext.setupDestinationTitle), \(remediationLabel.lowercased()), then refresh \(panelContext.sectionTitle)."
        }
    }

    private func unknownRoutePanelErrorMessage(
        _ daemonError: DaemonAPIError,
        panelContext: UserFacingPanelErrorContext
    ) -> String? {
        guard daemonError.serverCode == "resource_not_found" else {
            return nil
        }
        guard let serverMessage = daemonError.serverMessage?.lowercased(),
              serverMessage.contains("unknown control route")
        else {
            return nil
        }
        return "\(panelContext.sectionTitle) is unavailable because this app build is behind daemon API changes. Update app/daemon and refresh."
    }

    private func channelCardStatus(from rawStatus: String, actionReadiness: String) -> ChannelCardStatus {
        switch normalizedActionReadiness(actionReadiness) {
        case "ready":
            return .active
        case "blocked":
            return .setupRequired
        case "degraded":
            return .degraded
        default:
            switch rawStatus.lowercased() {
            case "ready":
                return .active
            case "starting", "degraded", "stopped":
                return .degraded
            default:
                return .setupRequired
            }
        }
    }

    private func normalizedActionReadiness(_ raw: String?) -> String {
        switch raw?.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
        case "ready":
            return "ready"
        case "blocked":
            return "blocked"
        case "degraded":
            return "degraded"
        default:
            return ""
        }
    }

    private func effectiveChannelActionReadiness(_ card: DaemonChannelStatusCard) -> String {
        let normalized = normalizedActionReadiness(card.actionReadiness)
        if !normalized.isEmpty {
            return normalized
        }
        switch card.status.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
        case "ready":
            return "ready"
        case "starting", "degraded", "stopped":
            return "degraded"
        default:
            return "blocked"
        }
    }

    private func effectiveConnectorActionReadiness(
        _ card: DaemonConnectorStatusCard,
        connectorHealth: ConnectorHealthStatus
    ) -> String {
        let normalized = normalizedActionReadiness(card.actionReadiness)
        if !normalized.isEmpty {
            return normalized
        }
        switch connectorHealth {
        case .ready:
            return "ready"
        case .needsPermission:
            return "blocked"
        case .unavailable:
            return "degraded"
        }
    }

    private func actionReadinessLabel(_ readiness: String) -> String {
        switch readiness {
        case "ready":
            return "Ready"
        case "degraded":
            return "Degraded"
        default:
            return "Blocked"
        }
    }

    private func actionReadinessBlockerSummary(_ blockers: [DaemonActionReadinessBlocker]) -> String? {
        let summaries = blockers.compactMap { blocker -> String? in
            let message = blocker.message.trimmingCharacters(in: .whitespacesAndNewlines)
            if !message.isEmpty {
                return message
            }
            let code = blocker.code.trimmingCharacters(in: .whitespacesAndNewlines)
            return code.isEmpty ? nil : code.replacingOccurrences(of: "_", with: " ")
        }
        guard !summaries.isEmpty else {
            return nil
        }
        return summaries.joined(separator: " • ")
    }

    private func channelSetupDestination(channelID: String) -> (
        section: AppSection,
        actionTitle: String,
        message: String
    ) {
        switch Self.canonicalLogicalChannelID(from: channelID) {
        case "app":
            return (
                section: .chat,
                actionTitle: "Open App Channel",
                message: "Opened Chat for app channel checks."
            )
        case "message":
            return (
                section: .connectors,
                actionTitle: "Open Message Setup",
                message: "Opened Connectors for message channel setup and permissions."
            )
        case "voice":
            return (
                section: .configuration,
                actionTitle: "Open Voice Setup",
                message: "Opened Configuration for voice channel setup."
            )
        default:
            return (
                section: .configuration,
                actionTitle: "Open Channel Setup",
                message: "Opened Configuration for channel setup actions."
            )
        }
    }

    func connectorPermissionState(for card: DaemonConnectorStatusCard) -> ConnectorPermissionState {
        let fallback = connectorPermissionStatesByID[card.connectorID] ?? .unknown
        let statusReason = connectorStatusReason(for: card)
        return resolveConnectorPermissionState(
            daemonPermissionStateRaw: connectorConfigurationValue(card, key: "permission_state"),
            statusReason: statusReason,
            connectorStatus: card.status,
            fallback: fallback
        )
    }

    func resolveConnectorPermissionState(
        daemonPermissionStateRaw: String?,
        statusReason: String,
        connectorStatus: String,
        fallback: ConnectorPermissionState
    ) -> ConnectorPermissionState {
        if statusReason == "permission_missing" {
            return .missing
        }
        let daemonPermissionState = connectorPermissionState(
            fromDaemonState: daemonPermissionStateRaw,
            fallback: .unknown
        )
        if daemonPermissionState != .unknown {
            return daemonPermissionState
        }
        if fallback == .missing {
            return .missing
        }
        return .unknown
    }

    private func connectorPermissionState(fromDaemonState raw: String?, fallback: ConnectorPermissionState) -> ConnectorPermissionState {
        switch raw?.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() {
        case "granted":
            return .granted
        case "missing":
            return .missing
        case "unknown":
            return .unknown
        default:
            return fallback
        }
    }

    private func applyConnectorPermissionState(_ state: ConnectorPermissionState, connectorID: String) {
        connectorPermissionStatesByID[connectorID] = state
        guard let cardIndex = connectorCards.firstIndex(where: { $0.id == connectorID }) else {
            return
        }
        connectorCards[cardIndex].permissionState = state
        if state == .missing {
            connectorCards[cardIndex].health = .needsPermission
        }
    }

    private func connectorHealthStatus(
        _ card: DaemonConnectorStatusCard,
        permissionState: ConnectorPermissionState
    ) -> ConnectorHealthStatus {
        let readiness = normalizedActionReadiness(card.actionReadiness)
        if readiness == "ready" {
            return .ready
        }
        if readiness == "blocked" {
            return .needsPermission
        }
        if readiness == "degraded" {
            return .unavailable
        }
        if permissionState == .missing {
            return .needsPermission
        }
        if permissionState == .unknown && connectorRequiresPermission(card) {
            return .needsPermission
        }
        let statusReason = connectorStatusReason(for: card)
        if statusReason == "permission_missing" {
            return .needsPermission
        }
        let normalizedStatus = card.status.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        if normalizedStatus == "ready" || statusReason == "ready" {
            return .ready
        }
        return .unavailable
    }

    private func connectorStatusReason(for card: DaemonConnectorStatusCard) -> String {
        connectorConfigurationValue(card, key: "status_reason")?
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .lowercased() ?? ""
    }

    private func connectorConfigurationValue(
        _ card: DaemonConnectorStatusCard,
        key: String
    ) -> String? {
        guard let value = card.configuration?[key] else {
            return nil
        }
        if let stringValue = nonEmpty(value.stringValue) {
            return stringValue
        }
        let fallback = nonEmpty(value.displayText)
        return fallback == "null" ? nil : fallback
    }

    private func connectorPermissionScope(
        for card: DaemonConnectorStatusCard,
        declaredCapabilities: [String],
        diagnosticsActions: [DiagnosticsActionItem],
        requiresPermission: Bool
    ) -> String {
        if let configuredScope = nonEmpty(
            connectorConfigurationValue(card, key: "permission_scope")
        ) {
            return configuredScope
        }
        if let configuredDetail = nonEmpty(
            connectorConfigurationValue(card, key: "permission_detail")
        ) {
            return configuredDetail
        }

        var fragments: [String] = []
        let capabilityText = declaredCapabilities.joined(separator: " ").lowercased()
        let connectorID = card.connectorID.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()

        if capabilityText.contains("message") || capabilityText.contains("imessage") || connectorID == "imessage" {
            fragments.append("Messages automation + Full Disk Access")
        }
        if capabilityText.contains("mail") || connectorID == "mail" {
            fragments.append("Mail automation")
        }
        if capabilityText.contains("calendar") || connectorID == "calendar" {
            fragments.append("Calendar automation")
        }
        if capabilityText.contains("browser") || capabilityText.contains("safari") || connectorID == "browser" {
            fragments.append("Safari automation")
        }
        if capabilityText.contains("finder") || connectorID == "finder" {
            fragments.append("Finder automation")
        }

        fragments = Self.deduplicatedPreservingOrder(fragments)
        if !fragments.isEmpty {
            return fragments.joined(separator: " • ")
        }
        if requiresPermission {
            return "System permission required"
        }
        if diagnosticsActions.contains(where: { action in
            let normalizedIntent = action.intent.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
            return normalizedIntent == "request_permission" || normalizedIntent == "open_system_settings"
        }) {
            return "System permission checks available"
        }
        return "No explicit system permission requirement"
    }

    private func connectorRequiresPermission(_ card: DaemonConnectorStatusCard) -> Bool {
        connectorRequiresPermission(
            card,
            diagnosticsActions: card.remediationActions?.map(mapDiagnosticsAction) ?? [],
            declaredCapabilities: normalizedCapabilityList(card.capabilities ?? [])
        )
    }

    private func connectorRequiresPermission(
        _ card: DaemonConnectorStatusCard,
        diagnosticsActions: [DiagnosticsActionItem],
        declaredCapabilities: [String]
    ) -> Bool {
        if connectorPermissionState(fromDaemonState: connectorConfigurationValue(card, key: "permission_state"), fallback: .unknown) == .missing {
            return true
        }
        if let requiredFlag = normalizedBoolean(
            card.configuration?["permission_required"]
        ) {
            return requiredFlag
        }
        let statusReason = connectorStatusReason(for: card)
        if statusReason == "permission_missing" {
            return true
        }
        if diagnosticsActions.contains(where: { action in
            let normalizedIntent = action.intent.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
            let normalizedID = action.id.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
            return normalizedIntent == "request_permission"
                || normalizedIntent == "open_system_settings"
                || normalizedID == "request_connector_permission"
                || normalizedID == "open_connector_system_settings"
        }) {
            return true
        }
        let capabilityText = declaredCapabilities.joined(separator: " ").lowercased()
        return capabilityText.contains("mail")
            || capabilityText.contains("calendar")
            || capabilityText.contains("finder")
            || capabilityText.contains("browser")
            || capabilityText.contains("message")
            || capabilityText.contains("imessage")
    }

    private func ensurePrincipalFallbackOptions() -> [String] {
        identityContextStore.ensurePrincipalFallbackOptions()
    }

    private func principalOptionsForPrincipalSelection(including actorID: String?) -> [String] {
        identityContextStore.principalOptionsForPrincipalSelection(including: actorID)
    }

    private func normalizedIdentityID(_ value: String?) -> String? {
        let trimmed = value?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        guard !trimmed.isEmpty else {
            return nil
        }
        return trimmed
    }

    private func applyIdentityActiveContext(
        _ context: DaemonIdentityActiveContext?,
        fallbackWorkspaceID: String,
        fallbackPrincipalActorID: String?,
        contextUpdateIntent: WorkspaceContextUpdateIntent
    ) {
        let resolvedWorkspaceID = Self.canonicalWorkspaceID(
            nonEmpty(context?.workspaceID) ?? fallbackWorkspaceID
        ) ?? Self.defaultWorkspaceID
        updateWorkspaceContext(from: resolvedWorkspaceID, intent: contextUpdateIntent)

        let resolvedPrincipalID = nonEmpty(context?.principalActorID)
            ?? fallbackPrincipalActorID
            ?? "default"
        if !resolvedPrincipalID.isEmpty {
            selectedPrincipal = resolvedPrincipalID
        }

        let workspaceSource = nonEmpty(context?.workspaceSource) ?? "unknown"
        let principalSource = nonEmpty(context?.principalSource) ?? "unknown"
        let lastUpdatedLabel = nonEmpty(context?.lastUpdatedAt).map(formattedWorkflowTimestamp)
        let workspaceResolved = context?.workspaceResolved ?? false

        identityActiveContext = IdentityActiveContextItem(
            workspaceID: resolvedWorkspaceID,
            principalActorID: resolvedPrincipalID,
            workspaceSource: workspaceSource,
            principalSource: principalSource,
            lastUpdatedLabel: lastUpdatedLabel,
            workspaceResolved: workspaceResolved
        )
    }

    private func mapIdentityDirectoryRecords(
        workspaceResponse: DaemonIdentityWorkspacesResponse,
        principalsResponse: DaemonIdentityPrincipalsResponse
    ) {
        var mappedWorkspaceItemsByID: [String: IdentityWorkspaceItem] = [:]
        var updatedAtByWorkspaceID: [String: Date] = [:]

        for workspace in workspaceResponse.workspaces {
            let rawWorkspaceID = nonEmpty(workspace.workspaceID)
            let normalizedWorkspaceID = Self.canonicalWorkspaceID(rawWorkspaceID, fallbackToDefault: false) ?? workspaceID
            let updatedAt = parseDaemonTimestamp(workspace.updatedAt) ?? .distantPast

            let mappedItem = IdentityWorkspaceItem(
                id: normalizedWorkspaceID,
                name: nonEmpty(workspace.name) ?? normalizedWorkspaceID,
                status: nonEmpty(workspace.status) ?? "UNKNOWN",
                principalCount: workspace.principalCount,
                actorCount: workspace.actorCount,
                handleCount: workspace.handleCount,
                updatedAtLabel: nonEmpty(workspace.updatedAt).map(formattedWorkflowTimestamp) ?? "n/a",
                isActive: workspace.isActive
            )

            if mappedWorkspaceItemsByID[normalizedWorkspaceID] != nil {
                if let existingUpdatedAt = updatedAtByWorkspaceID[normalizedWorkspaceID],
                   existingUpdatedAt > updatedAt {
                    continue
                }
            }

            mappedWorkspaceItemsByID[normalizedWorkspaceID] = mappedItem
            updatedAtByWorkspaceID[normalizedWorkspaceID] = updatedAt
        }

        identityWorkspaceItems = mappedWorkspaceItemsByID.values.sorted { lhs, rhs in
            if lhs.isActive != rhs.isActive {
                return lhs.isActive
            }
            let nameCompare = lhs.name.localizedCaseInsensitiveCompare(rhs.name)
            if nameCompare != .orderedSame {
                return nameCompare == .orderedAscending
            }
            return lhs.id.localizedCaseInsensitiveCompare(rhs.id) == .orderedAscending
        }

        identityPrincipalItems = principalsResponse.principals.map { principal in
            let handles: [IdentityPrincipalHandleItem] = principal.handles.enumerated().map { index, handle in
                let actorID = nonEmpty(principal.actorID) ?? "unknown"
                let channel = nonEmpty(handle.channel) ?? "unknown"
                let handleValue = nonEmpty(handle.handleValue) ?? "unknown"
                return IdentityPrincipalHandleItem(
                    id: "\(actorID)::\(channel)::\(handleValue)::\(index)",
                    channel: channel,
                    handleValue: handleValue,
                    isPrimary: handle.isPrimary,
                    updatedAtLabel: nonEmpty(handle.updatedAt).map(formattedWorkflowTimestamp) ?? "n/a"
                )
            }

            return IdentityPrincipalItem(
                id: nonEmpty(principal.actorID) ?? "unknown",
                displayName: nonEmpty(principal.displayName) ?? (nonEmpty(principal.actorID) ?? "Unknown"),
                actorType: nonEmpty(principal.actorType) ?? "unknown",
                actorStatus: nonEmpty(principal.actorStatus) ?? "UNKNOWN",
                principalStatus: nonEmpty(principal.principalStatus) ?? "UNKNOWN",
                isActive: principal.isActive,
                handles: handles
            )
        }
    }

    private func mapDelegationRuleRecord(_ record: DaemonDelegationRuleRecord) -> DelegationRuleItem {
        let scopeType = nonEmpty(record.scopeType)?.uppercased() ?? "EXECUTION"
        return DelegationRuleItem(
            id: nonEmpty(record.id) ?? UUID().uuidString.lowercased(),
            workspaceID: nonEmpty(record.workspaceID) ?? workspaceID,
            fromActorID: nonEmpty(record.fromActorID) ?? "unknown",
            toActorID: nonEmpty(record.toActorID) ?? "unknown",
            scopeType: scopeType,
            scopeKey: nonEmpty(record.scopeKey),
            status: nonEmpty(record.status) ?? "ACTIVE",
            createdAtLabel: nonEmpty(record.createdAt).map(formattedWorkflowTimestamp) ?? "n/a",
            expiresAtLabel: nonEmpty(record.expiresAt).map(formattedWorkflowTimestamp)
        )
    }

    private func delegationScopeSummary(scopeType: String, scopeKey: String?) -> String {
        let normalizedScopeType = scopeType.trimmingCharacters(in: .whitespacesAndNewlines).uppercased()
        if let scopeKey = nonEmpty(scopeKey) {
            return "\(normalizedScopeType):\(scopeKey)"
        }
        return normalizedScopeType
    }

    private func updateWorkspaceContext(
        from daemonWorkspaceID: String?,
        intent: WorkspaceContextUpdateIntent = .passiveResponse,
        persistSelection: Bool = true
    ) {
        identityContextStore.updateWorkspaceContext(
            from: daemonWorkspaceID,
            intent: intent,
            persistSelection: persistSelection,
            canonicalWorkspaceID: Self.canonicalWorkspaceID,
            applyWorkspaceScopedInformationDensityMode: { resolvedWorkspaceID in
                self.applyWorkspaceScopedInformationDensityMode(for: resolvedWorkspaceID)
            },
            persistWorkspaceSelection: { resolvedWorkspaceID in
                self.persistWorkspaceSelection(resolvedWorkspaceID)
            }
        )
    }

    func _test_applyWorkspaceContextPassive(_ workspaceID: String?) {
        updateWorkspaceContext(from: workspaceID, intent: .passiveResponse)
    }

    func _test_applyWorkspaceContextIdentitySync(_ workspaceID: String?) {
        updateWorkspaceContext(from: workspaceID, intent: .identityDirectorySync)
    }

    func _test_applyWorkspaceContextExplicitSelection(_ workspaceID: String?) {
        updateWorkspaceContext(from: workspaceID, intent: .explicitSelection, persistSelection: false)
    }

    func _test_applyWorkspaceContextExplicitSelectionPersisted(_ workspaceID: String?) {
        updateWorkspaceContext(from: workspaceID, intent: .explicitSelection, persistSelection: true)
    }

    func _test_mapIdentityDirectoryRecords(
        workspaceResponse: DaemonIdentityWorkspacesResponse,
        principalsResponse: DaemonIdentityPrincipalsResponse
    ) {
        mapIdentityDirectoryRecords(
            workspaceResponse: workspaceResponse,
            principalsResponse: principalsResponse
        )
    }

    func _test_storedWorkspaceSelection() -> String? {
        Self.canonicalWorkspaceID(
            Self.userDefaultsStore.string(forKey: Self.workspaceDefaultsKey),
            fallbackToDefault: false
        )
    }

    func _test_resetHomeFirstSessionProgress() {
        contextRetentionStore.resetHomeFirstSessionProgress()
    }

    func _test_markHomeFirstSessionStepComplete(
        _ stepID: HomeFirstSessionStepID,
        source: String = "unknown",
        completedAt: Date = .now
    ) {
        markHomeFirstSessionStepComplete(
            stepID,
            source: source,
            completedAt: completedAt
        )
    }

    func _test_panelLatencyBudget(
        section: AppSection,
        category: UIPanelLatencyCategory
    ) -> Int {
        panelLatencyStore.panelLatencyBudgetMS(for: section, category: category)
    }

    func _test_panelLatencyCategoryForTransition(section: AppSection) -> UIPanelLatencyCategory {
        panelLatencyStore.panelLatencyCategory(
            for: section,
            trigger: .transition,
            hasLoadedPanelData: { [self] target in
                hasLoadedPanelData(for: target)
            }
        )
    }

    func _test_panelLatencyCategoryForBootstrap(section: AppSection) -> UIPanelLatencyCategory {
        panelLatencyStore.panelLatencyCategory(
            for: section,
            trigger: .bootstrap,
            hasLoadedPanelData: { [self] target in
                hasLoadedPanelData(for: target)
            }
        )
    }

    func _test_panelLatencyCategoryForRefresh(section: AppSection) -> UIPanelLatencyCategory {
        panelLatencyStore.panelLatencyCategory(
            for: section,
            trigger: .refresh,
            hasLoadedPanelData: { [self] target in
                hasLoadedPanelData(for: target)
            }
        )
    }

    func _test_recordPanelLatencySample(
        section: AppSection,
        category: UIPanelLatencyCategory,
        durationMS: Int,
        capturedAt: Date = .now
    ) {
        panelLatencyStore.recordPanelLatencySample(
            section: section,
            category: category,
            durationMS: durationMS,
            capturedAt: capturedAt
        )
    }

    func _test_chatRealtimeFallbackContext(
        error: Error,
        defaultReason: ChatRealtimeFallbackReason = .unavailable
    ) -> (
        reason: ChatRealtimeFallbackReason,
        statusMessage: String,
        progressDetail: String,
        remediationDetail: String
    ) {
        let context = chatRealtimeFallbackContext(
            from: error,
            defaultReason: defaultReason
        )
        return (
            reason: context.reason,
            statusMessage: context.statusMessage,
            progressDetail: context.progressDetail,
            remediationDetail: context.remediationDetail
        )
    }

    func _test_chatRealtimeFallbackContext(
        eventErrorCode: String?,
        message: String?,
        defaultReason: ChatRealtimeFallbackReason = .unavailable
    ) -> (
        reason: ChatRealtimeFallbackReason,
        statusMessage: String,
        progressDetail: String,
        remediationDetail: String
    ) {
        let context = chatRealtimeFallbackContext(
            fromRealtimeEventCode: eventErrorCode,
            message: message,
            defaultReason: defaultReason
        )
        return (
            reason: context.reason,
            statusMessage: context.statusMessage,
            progressDetail: context.progressDetail,
            remediationDetail: context.remediationDetail
        )
    }

    func _test_setChatRealtimeTracking(
        connected: Bool,
        reason: ChatRealtimeFallbackReason?,
        detail: String? = nil
    ) {
        chatOrchestrationStore.realtimeConnectedForActiveTurn = connected
        chatOrchestrationStore.realtimeFallbackReason = reason
        chatOrchestrationStore.realtimeFallbackDetail = detail
    }

    func _test_chatRealtimeTransportConnectedForActiveTurn() -> Bool {
        chatRealtimeTransportConnectedForActiveTurn()
    }

    func _test_recoveredChatTurnSnapshot(
        history: DaemonChatTurnHistoryResponse,
        correlationID: String
    ) -> (
        correlationID: String,
        taskClass: String?,
        channelID: String?,
        itemTypes: [String]
    )? {
        guard let snapshot = ChatTurnExecutionStore.recoveredChatTurnSnapshot(
            from: history,
            correlationID: correlationID,
            parseDaemonTimestamp: { [weak self] value in
                guard let self else {
                    return nil
                }
                return self.parseDaemonTimestamp(value)
            }
        ) else {
            return nil
        }
        return (
            correlationID: snapshot.correlationID,
            taskClass: snapshot.taskClass,
            channelID: snapshot.channelID,
            itemTypes: snapshot.items.map(\.type)
        )
    }

    func _test_chatRealtimeErrorSummaryForDisplay(
        errorCode: String?,
        message: String?
    ) -> String {
        chatRealtimeErrorSummaryForDisplay(
            errorCode: errorCode,
            message: message
        )
    }

    func _test_performDaemonLifecycleControl(action: String) async {
        await performDaemonLifecycleControl(action: action)
    }

    private func persistWorkspaceSelection(_ workspaceID: String) {
        guard let normalizedWorkspaceID = Self.canonicalWorkspaceID(
            workspaceID,
            fallbackToDefault: false
        ) else {
            return
        }
        Self.userDefaultsStore.set(normalizedWorkspaceID, forKey: Self.workspaceDefaultsKey)
    }

    private func updateOnboardingCompletionState() {
        if onboardingReadinessMet {
            if !hasCompletedFirstRunOnboarding {
                hasCompletedFirstRunOnboarding = true
                Self.userDefaultsStore.set(true, forKey: Self.onboardingCompleteDefaultsKey)
            }
            return
        }

        if hasCompletedFirstRunOnboarding {
            hasCompletedFirstRunOnboarding = false
            Self.userDefaultsStore.set(false, forKey: Self.onboardingCompleteDefaultsKey)
        }
    }

    private func jsonArrayCount(value: DaemonJSONValue, key: String) -> Int? {
        guard case .object(let object) = value else {
            return nil
        }
        guard case .array(let values) = object[key] else {
            return nil
        }
        return values.count
    }

    static func approvalEvidencePayloadSummary(from payloadJSON: String?) -> String? {
        guard let payloadJSON else {
            return nil
        }
        let trimmed = payloadJSON.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            return nil
        }

        guard let payload = approvalEvidencePayloadValue(from: trimmed) else {
            return approvalEvidenceTruncate(trimmed, limit: 220)
        }

        guard case .object(let objectPayload) = payload else {
            return approvalEvidenceSummarizeValue(payload, limit: 220)
                ?? approvalEvidenceTruncate(trimmed, limit: 220)
        }

        let orderedKeys = ["summary", "output", "input", "status", "evidence", "requested_phrase", "capability_key", "rationale"]
        var parts: [String] = []
        for key in orderedKeys {
            guard let value = objectPayload[key],
                  let valueSummary = approvalEvidenceSummarizeValue(value, limit: 120) else {
                continue
            }
            let label = key.replacingOccurrences(of: "_", with: " ")
            parts.append("\(label): \(valueSummary)")
            if parts.count >= 3 {
                break
            }
        }

        if !parts.isEmpty {
            return approvalEvidenceTruncate(parts.joined(separator: " • "), limit: 220)
        }

        if let serialized = approvalEvidenceSummarizeValue(.object(objectPayload), limit: 220) {
            return serialized
        }
        return approvalEvidenceTruncate(trimmed, limit: 220)
    }

    static func approvalEvidenceInputOutputSummaries(
        from auditEntries: [DaemonInspectRunAuditEntry]
    ) -> (input: String?, output: String?) {
        var inputSummary: String?
        var outputSummary: String?

        for entry in auditEntries {
            guard let payload = entry.payloadObject else {
                continue
            }
            if inputSummary == nil {
                inputSummary = approvalEvidenceFieldSummary(
                    payload,
                    keys: ["input", "inputs", "arguments", "request", "prompt"],
                    limit: 220
                )
            }
            if outputSummary == nil {
                outputSummary = approvalEvidenceFieldSummary(
                    payload,
                    keys: ["summary", "output", "result", "response", "evidence", "status"],
                    limit: 220
                )
            }
            if inputSummary != nil && outputSummary != nil {
                break
            }
        }

        return (inputSummary, outputSummary)
    }

    private static func approvalEvidencePayloadValue(from payloadJSON: String) -> DaemonJSONValue? {
        guard let data = payloadJSON.data(using: .utf8),
              let payload = try? JSONDecoder().decode(DaemonJSONValue.self, from: data) else {
            return nil
        }
        return payload
    }

    private static func approvalEvidenceFieldSummary(
        _ payload: [String: DaemonJSONValue],
        keys: [String],
        limit: Int
    ) -> String? {
        for key in keys {
            guard let rawValue = payload[key],
                  let summary = approvalEvidenceSummarizeValue(rawValue, limit: limit) else {
                continue
            }
            return summary
        }
        return nil
    }

    private static func approvalEvidenceSummarizeValue(_ value: DaemonJSONValue, limit: Int) -> String? {
        switch value {
        case .null:
            return nil
        case .string(let text):
            let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
            guard !trimmed.isEmpty else {
                return nil
            }
            return approvalEvidenceTruncate(trimmed, limit: limit)
        case .number(let number):
            if number.rounded() == number {
                return approvalEvidenceTruncate(String(Int(number)), limit: limit)
            }
            return approvalEvidenceTruncate(String(number), limit: limit)
        case .bool(let flag):
            return approvalEvidenceTruncate(flag ? "true" : "false", limit: limit)
        case .array(let values):
            if values.isEmpty {
                return "[]"
            }
            let parts = values.compactMap { entry in
                approvalEvidenceSummarizeValue(entry, limit: max(limit / 2, 32))
            }
            guard !parts.isEmpty else {
                return "[]"
            }
            return approvalEvidenceTruncate("[\(parts.joined(separator: ", "))]", limit: limit)
        case .object(let values):
            if values.isEmpty {
                return "{}"
            }
            let keys = values.keys.sorted {
                $0.localizedCaseInsensitiveCompare($1) == .orderedAscending
            }
            let parts = keys.compactMap { key -> String? in
                guard let summary = values[key].flatMap({
                    approvalEvidenceSummarizeValue($0, limit: max(limit / 2, 48))
                }) else {
                    return nil
                }
                return "\(key): \(summary)"
            }
            if parts.isEmpty {
                return "{}"
            }
            return approvalEvidenceTruncate(parts.joined(separator: " • "), limit: limit)
        }
    }

    private static func approvalEvidenceTruncate(_ value: String, limit: Int) -> String {
        guard value.count > limit else {
            return value
        }
        let endIndex = value.index(value.startIndex, offsetBy: limit)
        return "\(value[..<endIndex])…"
    }

    private struct MappedLogicalChannelConnectorContext {
        let rollup: LogicalChannelConnectorRollupItem
        let reasonLines: [String]
        let actionTitles: [String]
    }

    private func buildLogicalChannelCards(
        channelCards: [ChannelCardItem],
        connectorCards: [ConnectorCardItem],
        channelConnectorMappingsByChannelID: [String: [ChannelConnectorMappingItem]]
    ) -> [LogicalChannelCardItem] {
        var groupedChannels: [String: [ChannelCardItem]] = [:]
        for card in channelCards {
            let logicalChannelID = normalizedChannelConnectorMappingChannelID(card.logicalChannelID)
            guard !logicalChannelID.isEmpty else {
                continue
            }
            groupedChannels[logicalChannelID, default: []].append(card)
        }

        let connectorMembersByCanonicalID = groupedConnectorCardsByCanonicalID(connectorCards)
        let normalizedMappingsByChannelID = normalizedChannelConnectorMappingsByLogicalChannelID(
            channelConnectorMappingsByChannelID
        )
        let inferredMappingsByChannelID = inferredChannelConnectorMappingsByLogicalChannelID(from: channelCards)

        let orderedChannelIDs = groupedChannels.keys.sorted(by: logicalChannelSortComparator)
        return orderedChannelIDs.compactMap { channelID in
            let members = groupedChannels[channelID] ?? []
            guard !members.isEmpty else {
                return nil
            }
            let orderedMembers = members.sorted {
                $0.name.localizedCaseInsensitiveCompare($1.name) == .orderedAscending
            }
            let mappings = mergedChannelConnectorMappings(
                observed: normalizedMappingsByChannelID[channelID] ?? [],
                inferred: inferredMappingsByChannelID[channelID] ?? [],
                channelID: channelID
            )
            let primaryID = primaryChannelCardID(
                for: channelID,
                members: orderedMembers,
                mappings: mappings
            )
            guard let primaryCard = orderedMembers.first(where: { $0.id == primaryID }) ?? orderedMembers.first else {
                return nil
            }

            let connectorContexts = mappedConnectorContexts(
                for: channelID,
                mappings: mappings,
                connectorMembersByCanonicalID: connectorMembersByCanonicalID
            )
            let connectorRollups = connectorContexts.map(\.rollup)
            let connectorActionTitles = Self.deduplicatedPreservingOrder(
                connectorContexts.flatMap(\.actionTitles)
            )
            let connectorReasonLines = Self.deduplicatedPreservingOrder(
                connectorContexts.flatMap(\.reasonLines)
            )
            let connectorReasonSummary = connectorReasonLines.joined(separator: " • ")

            var details = primaryCard.details
            let logicalChannelTitle = logicalChannelDisplayName(
                for: channelID,
                memberDisplayNames: orderedMembers.map(\.name)
            )
            details["Logical Channel"] = logicalChannelTitle
            details["Mapped Implementations"] = orderedMembers.map(\.name).joined(separator: ", ")
            details["Mapped Channel IDs"] = orderedMembers.map(\.id).joined(separator: ", ")
            if !connectorRollups.isEmpty {
                details["Mapped Connectors"] = connectorRollups.map(\.connectorName).joined(separator: ", ")
                details["Connector Health"] = connectorRollups
                    .map { item in "\(item.connectorName): \(item.health.label)" }
                    .joined(separator: " • ")
            }
            if !connectorReasonSummary.isEmpty {
                details["Connector Reasons"] = connectorReasonSummary
            }
            if !connectorActionTitles.isEmpty {
                details["Connector Actions"] = connectorActionTitles.joined(separator: ", ")
            }

            return LogicalChannelCardItem(
                channelID: channelID,
                displayName: logicalChannelTitle,
                status: logicalChannelStatusRollup(for: orderedMembers),
                summary: logicalChannelSummary(
                    members: orderedMembers,
                    connectorCount: connectorRollups.count
                ),
                details: details,
                primaryChannelCardID: primaryCard.id,
                channelCardIDs: orderedMembers.map(\.id),
                actions: Self.deduplicatedDiagnosticsActions(orderedMembers.flatMap(\.actions)),
                unavailableActionReason: orderedMembers.first(where: {
                    !$0.unavailableActionReason.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
                })?.unavailableActionReason
                    ?? "No channel remediation actions are currently available. Open Connectors for connector-level remediation.",
                mappedConnectorRollups: connectorRollups,
                connectorActionTitles: connectorActionTitles,
                connectorReasonSummary: connectorReasonSummary.isEmpty ? nil : connectorReasonSummary
            )
        }
    }

    private func logicalChannelSortComparator(_ lhs: String, _ rhs: String) -> Bool {
        let lhsOrder = Self.canonicalLogicalChannelSortOrder[Self.canonicalLogicalChannelID(from: lhs)] ?? Int.max
        let rhsOrder = Self.canonicalLogicalChannelSortOrder[Self.canonicalLogicalChannelID(from: rhs)] ?? Int.max
        if lhsOrder != rhsOrder {
            return lhsOrder < rhsOrder
        }
        return logicalChannelDisplayName(for: lhs, memberDisplayNames: [])
            .localizedCaseInsensitiveCompare(logicalChannelDisplayName(for: rhs, memberDisplayNames: [])) == .orderedAscending
    }

    private func logicalChannelDisplayName(
        for channelID: String,
        memberDisplayNames: [String]
    ) -> String {
        let canonicalChannelID = Self.canonicalLogicalChannelID(from: channelID)
        if canonicalChannelID == "app" || canonicalChannelID == "message" || canonicalChannelID == "voice" {
            return Self.logicalChannelDisplayName(for: canonicalChannelID)
        }
        if let memberName = memberDisplayNames
            .map({ $0.trimmingCharacters(in: .whitespacesAndNewlines) })
            .first(where: { !$0.isEmpty }) {
            return memberName
        }
        return Self.logicalChannelDisplayName(for: channelID)
    }

    private func groupedConnectorCardsByCanonicalID(
        _ connectorCards: [ConnectorCardItem]
    ) -> [String: [ConnectorCardItem]] {
        var grouped: [String: [ConnectorCardItem]] = [:]
        for connector in connectorCards {
            let canonicalID = normalizedChannelConnectorMappingConnectorID(connector.logicalConnectorID)
            guard !canonicalID.isEmpty else {
                continue
            }
            grouped[canonicalID, default: []].append(connector)
        }
        return grouped.mapValues { connectors in
            connectors.sorted { lhs, rhs in
                lhs.name.localizedCaseInsensitiveCompare(rhs.name) == .orderedAscending
            }
        }
    }

    private func normalizedChannelConnectorMappingsByLogicalChannelID(
        _ mappingsByChannelID: [String: [ChannelConnectorMappingItem]]
    ) -> [String: [ChannelConnectorMappingItem]] {
        var grouped: [String: [ChannelConnectorMappingItem]] = [:]
        for (rawChannelID, mappings) in mappingsByChannelID {
            let fallbackChannelID = normalizedChannelConnectorMappingChannelID(rawChannelID)
            for mapping in mappings {
                guard let normalized = normalizedChannelConnectorMapping(
                    mapping,
                    fallbackChannelID: fallbackChannelID
                ) else {
                    continue
                }
                grouped[normalized.channelID, default: []].append(normalized)
            }
        }
        return grouped.mapValues { mappings in
            let ordered = mappings.sorted { lhs, rhs in
                if lhs.priority == rhs.priority {
                    return lhs.connectorID.localizedCaseInsensitiveCompare(rhs.connectorID) == .orderedAscending
                }
                return lhs.priority < rhs.priority
            }

            var seenConnectorIDs: Set<String> = []
            var deduplicated: [ChannelConnectorMappingItem] = []
            for mapping in ordered {
                if seenConnectorIDs.insert(mapping.connectorID).inserted {
                    deduplicated.append(mapping)
                }
            }
            return deduplicated
        }
    }

    private func normalizedChannelConnectorMapping(
        _ mapping: ChannelConnectorMappingItem,
        fallbackChannelID: String
    ) -> ChannelConnectorMappingItem? {
        let resolvedRawChannelID = nonEmpty(mapping.channelID) ?? fallbackChannelID
        let normalizedChannelID = normalizedChannelConnectorMappingChannelID(resolvedRawChannelID)
        let normalizedConnectorID = normalizedChannelConnectorMappingConnectorID(mapping.connectorID)
        guard !normalizedChannelID.isEmpty, !normalizedConnectorID.isEmpty else {
            return nil
        }
        return ChannelConnectorMappingItem(
            channelID: normalizedChannelID,
            connectorID: normalizedConnectorID,
            enabled: mapping.enabled,
            priority: max(1, mapping.priority),
            capabilities: mapping.capabilities,
            createdAtLabel: mapping.createdAtLabel,
            updatedAtLabel: mapping.updatedAtLabel
        )
    }

    private func mappedConnectorContexts(
        for channelID: String,
        mappings: [ChannelConnectorMappingItem],
        connectorMembersByCanonicalID: [String: [ConnectorCardItem]]
    ) -> [MappedLogicalChannelConnectorContext] {
        let orderedMappings = mappings.sorted { lhs, rhs in
            if lhs.priority == rhs.priority {
                return lhs.connectorID.localizedCaseInsensitiveCompare(rhs.connectorID) == .orderedAscending
            }
            return lhs.priority < rhs.priority
        }
        let enabledMappings = orderedMappings.filter(\.enabled)
        let effectiveMappings = enabledMappings.isEmpty ? orderedMappings : enabledMappings

        return effectiveMappings.compactMap { mapping in
            guard normalizedChannelConnectorMappingChannelID(mapping.channelID) == channelID else {
                return nil
            }
            let canonicalConnectorID = normalizedChannelConnectorMappingConnectorID(mapping.connectorID)
            guard !canonicalConnectorID.isEmpty else {
                return nil
            }

            let members = connectorMembersByCanonicalID[canonicalConnectorID] ?? []
            let connectorName = logicalChannelConnectorDisplayName(
                connectorID: canonicalConnectorID,
                fallback: members.first?.name
            )
            let connectorHealth: ConnectorHealthStatus = members.isEmpty
                ? .unavailable
                : Self.logicalConnectorHealthRollup(for: members)
            let reasonLines = Self.deduplicatedPreservingOrder(
                members.compactMap { member in
                    let trimmedReason = member.statusReason?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
                    guard !trimmedReason.isEmpty else {
                        return nil
                    }
                    return "\(member.name): \(trimmedReason)"
                }
            )
            let actionTitles = Self.deduplicatedPreservingOrder(
                members.flatMap { member in
                    member.actions.map(\.title)
                }
            )
            let reason = reasonLines.first
                ?? (members.isEmpty ? "\(connectorName): Connector status unavailable." : nil)

            return MappedLogicalChannelConnectorContext(
                rollup: LogicalChannelConnectorRollupItem(
                    connectorID: canonicalConnectorID,
                    connectorName: connectorName,
                    health: connectorHealth,
                    reason: reason
                ),
                reasonLines: reasonLines,
                actionTitles: actionTitles
            )
        }
    }

    private func logicalChannelConnectorDisplayName(connectorID: String, fallback: String?) -> String {
        let normalizedConnectorID = normalizedChannelConnectorMappingConnectorID(connectorID)
        if let fallback = nonEmpty(fallback) {
            return fallback
        }
        return channelConnectorDisplayName(connectorID: normalizedConnectorID)
    }

    private func primaryChannelCardID(
        for channelID: String,
        members: [ChannelCardItem],
        mappings: [ChannelConnectorMappingItem]
    ) -> String {
        let normalizedChannelID = normalizedChannelConnectorMappingChannelID(channelID)
        if let primaryConnectorID = logicalChannelPrimaryConnectorID(members: members, mappings: mappings),
           let preferredMember = members.first(where: { member in
               Self.channelMemberID(member.id, matchesConnectorID: primaryConnectorID)
           }) {
            return preferredMember.id
        }

        if let canonicalMember = members.first(where: {
            normalizedChannelConnectorMappingChannelID($0.logicalChannelID) == normalizedChannelID
                || normalizedChannelConnectorMappingChannelID($0.id) == normalizedChannelID
        }) {
            return canonicalMember.id
        }

        let orderedMappings = mappings.sorted { lhs, rhs in
            if lhs.priority == rhs.priority {
                return lhs.connectorID.localizedCaseInsensitiveCompare(rhs.connectorID) == .orderedAscending
            }
            return lhs.priority < rhs.priority
        }
        let enabledMappings = orderedMappings.filter(\.enabled)
        let effectiveMappings = enabledMappings.isEmpty ? orderedMappings : enabledMappings

        for mapping in effectiveMappings {
            let connectorID = normalizedChannelConnectorMappingConnectorID(mapping.connectorID)
            if let member = members.first(where: { member in
                Self.channelMemberID(member.id, matchesConnectorID: connectorID)
            }) {
                return member.id
            }
        }
        return members[0].id
    }

    private func logicalChannelPrimaryConnectorID(
        members: [ChannelCardItem],
        mappings: [ChannelConnectorMappingItem]
    ) -> String? {
        for member in members {
            if let primaryConnectorID = nonEmpty(
                normalizedChannelConnectorMappingConnectorID(
                    member.primaryConnectorID
                        ?? member.readOnlyConfiguration["primary_connector_id"]
                        ?? member.editableConfiguration["primary_connector_id"]
                        ?? member.details["Primary Connector ID"]
                        ?? ""
                )
            ) {
                return primaryConnectorID
            }
        }
        let orderedMappings = mappings.sorted { lhs, rhs in
            if lhs.priority == rhs.priority {
                return lhs.connectorID.localizedCaseInsensitiveCompare(rhs.connectorID) == .orderedAscending
            }
            return lhs.priority < rhs.priority
        }
        if let enabledConnectorID = orderedMappings.first(where: \.enabled)?.connectorID {
            return normalizedChannelConnectorMappingConnectorID(enabledConnectorID)
        }
        if let mappedConnectorID = orderedMappings.first?.connectorID {
            return normalizedChannelConnectorMappingConnectorID(mappedConnectorID)
        }
        return nil
    }

    private nonisolated static func channelMemberID(_ rawMemberID: String, matchesConnectorID connectorID: String) -> Bool {
        let normalizedMemberID = normalizedLogicalToken(rawMemberID)
        let normalizedConnectorID = normalizedLogicalToken(connectorID)
        if normalizedMemberID == normalizedConnectorID {
            return true
        }
        let connectorTokens = normalizedConnectorID
            .replacingOccurrences(of: ".", with: "_")
            .split(separator: "_")
            .map(String.init)
            .filter { !$0.isEmpty }
        return connectorTokens.contains { token in
            normalizedMemberID.contains(token)
        }
    }

    private func logicalChannelStatusRollup(for members: [ChannelCardItem]) -> ChannelCardStatus {
        if members.contains(where: { $0.status == .setupRequired }) {
            return .setupRequired
        }
        if members.contains(where: { $0.status == .degraded }) {
            return .degraded
        }
        return .active
    }

    private func logicalChannelSummary(
        members: [ChannelCardItem],
        connectorCount: Int
    ) -> String {
        if members.count == 1 {
            if connectorCount > 0 {
                return "\(members[0].summary) • \(connectorCount) mapped connector(s)"
            }
            return members[0].summary
        }

        let setupRequiredCount = members.filter { $0.status == .setupRequired }.count
        let degradedCount = members.filter { $0.status == .degraded }.count
        let activeCount = max(0, members.count - setupRequiredCount - degradedCount)

        var summaryParts: [String] = [
            "\(members.count) mapped implementations",
            "\(activeCount) active"
        ]
        if degradedCount > 0 {
            summaryParts.append("\(degradedCount) degraded")
        }
        if setupRequiredCount > 0 {
            summaryParts.append("\(setupRequiredCount) setup required")
        }
        if connectorCount > 0 {
            summaryParts.append("\(connectorCount) mapped connector(s)")
        }
        return summaryParts.joined(separator: " • ")
    }

    private static func deduplicatedDiagnosticsActions(_ actions: [DiagnosticsActionItem]) -> [DiagnosticsActionItem] {
        var seenActionIDs: Set<String> = []
        var deduplicated: [DiagnosticsActionItem] = []
        for action in actions {
            if seenActionIDs.insert(action.id).inserted {
                deduplicated.append(action)
            }
        }
        return deduplicated
    }

    private static func deduplicatedPreservingOrder(_ values: [String]) -> [String] {
        var seenValues: Set<String> = []
        var deduplicated: [String] = []
        for value in values {
            let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines)
            guard !trimmed.isEmpty else {
                continue
            }
            if seenValues.insert(trimmed).inserted {
                deduplicated.append(trimmed)
            }
        }
        return deduplicated
    }

    private nonisolated static func splitListTokens(_ raw: String) -> [String] {
        let cleaned = raw
            .replacingOccurrences(of: "[", with: " ")
            .replacingOccurrences(of: "]", with: " ")
            .replacingOccurrences(of: "\"", with: " ")
        let separators = CharacterSet(charactersIn: ",;\n\t|")
        return cleaned
            .components(separatedBy: separators)
            .map {
                $0.trimmingCharacters(in: CharacterSet.whitespacesAndNewlines.union(CharacterSet(charactersIn: "\"'`")))
            }
            .filter { !$0.isEmpty }
    }

    private nonisolated static func humanizedIdentifierTitle(_ raw: String) -> String {
        let normalized = normalizedLogicalToken(raw)
        guard !normalized.isEmpty else {
            return "Unknown"
        }
        let tokens = normalized
            .replacingOccurrences(of: ".", with: "_")
            .split(separator: "_")
            .map(String.init)
            .filter { !$0.isEmpty }
        let mappedTokens = tokens.map { token -> String in
            switch token {
            case "id":
                return "ID"
            case "ui":
                return "UI"
            case "url":
                return "URL"
            case "api":
                return "API"
            case "sms":
                return "SMS"
            case "mms":
                return "MMS"
            case "tcc":
                return "TCC"
            case "imessage":
                return "iMessage"
            case "builtin":
                return "Built-in"
            default:
                return token.capitalized
            }
        }
        if mappedTokens.isEmpty {
            return normalized
        }
        return mappedTokens.joined(separator: " ")
    }

    private nonisolated static func logicalChannelDisplayName(for channelID: String) -> String {
        let canonicalID = canonicalLogicalChannelID(from: channelID)
        switch canonicalID {
        case "app":
            return "App"
        case "message":
            return "Message"
        case "voice":
            return "Voice"
        default:
            return humanizedIdentifierTitle(canonicalID)
        }
    }

    private nonisolated static func capabilityFamilyLabel(from capability: String) -> String? {
        let normalized = capability.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        guard !normalized.isEmpty else {
            return nil
        }
        if normalized.contains("mail") {
            return "Mail"
        }
        if normalized.contains("calendar") {
            return "Calendar"
        }
        if normalized.contains("message")
            || normalized.contains("imessage")
            || normalized.contains("sms")
            || normalized.contains("mms") {
            return "Messaging"
        }
        if normalized.contains("voice")
            || normalized.contains("call")
            || normalized.contains("telephony")
            || normalized.contains("phone") {
            return "Voice"
        }
        if normalized.contains("browser")
            || normalized.contains("web")
            || normalized.contains("url")
            || normalized.contains("http")
            || normalized.contains("safari") {
            return "Browser"
        }
        if normalized.contains("finder")
            || normalized.contains("file")
            || normalized.contains("filesystem")
            || normalized.contains("path") {
            return "Files"
        }
        if normalized.contains("map")
            || normalized.contains("geo")
            || normalized.contains("location") {
            return "Maps"
        }
        if normalized.contains("reminder")
            || normalized.contains("todo")
            || normalized.contains("task") {
            return "Tasks"
        }
        if normalized.contains("note") {
            return "Notes"
        }
        if normalized.contains("chat") || normalized.contains("assistant") {
            return "Assistant"
        }
        return nil
    }

    private nonisolated static func normalizedLogicalToken(_ value: String) -> String {
        value
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .lowercased()
            .replacingOccurrences(of: "-", with: "_")
            .replacingOccurrences(of: " ", with: "_")
    }

    nonisolated static func canonicalLogicalChannelID(from raw: String) -> String {
        let normalized = normalizedLogicalToken(raw)
        switch normalized {
        case "app", "app_chat":
            return "app"
        case "message", "messages", "sms", "imessage", "imessage_sms", "imessage_sms_bridge", "imessage_bridge", "twilio_sms", "twilio_sms_bridge":
            return "message"
        case "voice", "twilio_voice", "twilio_voice_bridge":
            return "voice"
        default:
            return normalized
        }
    }

    nonisolated static func canonicalLogicalChannelIDs(_ values: [String]) -> [String] {
        var seenChannelIDs: Set<String> = []
        var canonicalIDs: [String] = []
        for value in values {
            let canonicalID = canonicalLogicalChannelID(from: value)
            guard !canonicalID.isEmpty else {
                continue
            }
            if seenChannelIDs.insert(canonicalID).inserted {
                canonicalIDs.append(canonicalID)
            }
        }
        return canonicalIDs
    }

    private nonisolated static func canonicalConnectorID(from raw: String) -> String {
        let normalized = normalizedLogicalToken(raw)
        switch normalized {
        case "app", "app_chat", "builtin_app", "builtinapp", "builtin.app":
            return "builtin.app"
        case "imessage", "messages", "imessage_sms", "imessage_bridge", "imessage_sms_bridge":
            return "imessage"
        case "twilio", "twilio_sms", "twilio_voice", "twilio_sms_bridge", "twilio_voice_bridge":
            return "twilio"
        default:
            return normalized
        }
    }

    static func buildLogicalConnectorCards(
        connectorCards: [ConnectorCardItem]
    ) -> [LogicalConnectorCardItem] {
        var groupedByLogicalID: [String: [ConnectorCardItem]] = [:]
        var orderedLogicalIDs: [String] = []
        var titlesByLogicalID: [String: String] = [:]

        for connector in connectorCards {
            let logicalID = logicalConnectorID(for: connector)
            if groupedByLogicalID[logicalID] == nil {
                orderedLogicalIDs.append(logicalID)
                titlesByLogicalID[logicalID] = logicalConnectorTitle(
                    for: logicalID,
                    fallbackName: connector.name
                )
            }
            groupedByLogicalID[logicalID, default: []].append(connector)
        }

        return orderedLogicalIDs.compactMap { logicalID in
            guard let members = groupedByLogicalID[logicalID], !members.isEmpty else {
                return nil
            }

            let normalizedLogicalID = normalizedLogicalToken(logicalID)
            let orderedMembers = members.sorted { lhs, rhs in
                let lhsCanonical = normalizedLogicalToken(lhs.id) == normalizedLogicalID
                    || normalizedLogicalToken(lhs.logicalConnectorID) == normalizedLogicalID
                let rhsCanonical = normalizedLogicalToken(rhs.id) == normalizedLogicalID
                    || normalizedLogicalToken(rhs.logicalConnectorID) == normalizedLogicalID
                if lhsCanonical != rhsCanonical {
                    return lhsCanonical && !rhsCanonical
                }
                let nameOrder = lhs.name.localizedCaseInsensitiveCompare(rhs.name)
                if nameOrder != .orderedSame {
                    return nameOrder == .orderedAscending
                }
                return lhs.id.localizedCaseInsensitiveCompare(rhs.id) == .orderedAscending
            }
            let primaryConnectorID = primaryConnectorCardID(
                for: logicalID,
                members: orderedMembers
            )
            guard let primaryCard = orderedMembers.first(where: { $0.id == primaryConnectorID }) ?? orderedMembers.first else {
                return nil
            }

            let title = titlesByLogicalID[logicalID] ?? primaryCard.name
            let mappedConnectorIDs = orderedMembers.map(\.id)
            var details = primaryCard.details
            details["Canonical Connector ID"] = logicalID
            details["Logical Connector"] = title
            details["Mapped Connectors"] = orderedMembers.map(\.name).joined(separator: ", ")
            details["Mapped Connector IDs"] = mappedConnectorIDs.joined(separator: ", ")

            let capabilities = combinedCapabilityValues(from: orderedMembers)
            if !capabilities.isEmpty {
                details["Capabilities"] = capabilities.joined(separator: ", ")
            }

            let statusReasonLines = deduplicatedPreservingOrder(
                orderedMembers.compactMap { member in
                    let trimmedReason = member.statusReason?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
                    guard !trimmedReason.isEmpty else {
                        return nil
                    }
                    return "\(member.name): \(trimmedReason)"
                }
            )
            if !statusReasonLines.isEmpty {
                details["Status Reasons"] = statusReasonLines.joined(separator: " • ")
            }

            let actions = logicalConnectorActions(for: orderedMembers)
            let actionTitles = deduplicatedPreservingOrder(actions.map(\.title))
            if !actionTitles.isEmpty {
                details["Remediation Actions"] = actionTitles.joined(separator: ", ")
            }

            let permissionScopes = deduplicatedPreservingOrder(orderedMembers.map(\.permissionScope))
            let permissionScope = permissionScopes.isEmpty
                ? primaryCard.permissionScope
                : permissionScopes.joined(separator: " • ")

            return LogicalConnectorCardItem(
                id: logicalID,
                title: title,
                health: logicalConnectorHealthRollup(for: orderedMembers),
                permissionState: logicalConnectorPermissionRollup(for: orderedMembers),
                permissionScope: permissionScope,
                statusReason: primaryCard.statusReason,
                summary: logicalConnectorSummary(for: orderedMembers),
                details: details,
                primaryConnectorCardID: primaryCard.id,
                connectorCardIDs: mappedConnectorIDs,
                actions: actions,
                unavailableActionReason: orderedMembers.first(where: {
                    !$0.unavailableActionReason.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
                })?.unavailableActionReason
                    ?? "No daemon remediation actions are currently available for this connector."
            )
        }
    }

    private static func logicalConnectorID(for connector: ConnectorCardItem) -> String {
        let candidates = [connector.logicalConnectorID, connector.id, connector.name]
        for candidate in candidates {
            let canonicalID = canonicalConnectorID(from: candidate)
            if !canonicalID.isEmpty {
                return canonicalID
            }
        }
        let fallback = normalizedLogicalToken(connector.id)
        return fallback.isEmpty ? "unknown" : fallback
    }

    private static func logicalConnectorTitle(for logicalID: String, fallbackName: String) -> String {
        if canonicalConnectorID(from: logicalID) == "builtin.app" {
            return "App"
        }
        return humanizedIdentifierTitle(logicalID)
    }

    private static func primaryConnectorCardID(
        for logicalID: String,
        members: [ConnectorCardItem]
    ) -> String {
        let normalizedLogicalID = normalizedLogicalToken(logicalID)
        if let canonicalMember = members.first(where: { member in
            normalizedLogicalToken(member.id) == normalizedLogicalID
                || normalizedLogicalToken(member.logicalConnectorID) == normalizedLogicalID
        }) {
            return canonicalMember.id
        }
        return members[0].id
    }

    private static func logicalConnectorHealthRollup(for members: [ConnectorCardItem]) -> ConnectorHealthStatus {
        if members.contains(where: { $0.health == .needsPermission }) {
            return .needsPermission
        }
        if members.contains(where: { $0.health == .unavailable }) {
            return .unavailable
        }
        return .ready
    }

    private static func logicalConnectorPermissionRollup(for members: [ConnectorCardItem]) -> ConnectorPermissionState {
        if members.contains(where: { $0.permissionState == .missing }) {
            return .missing
        }
        if members.contains(where: { $0.permissionState == .unknown }) {
            return .unknown
        }
        return .granted
    }

    private static func logicalConnectorSummary(for members: [ConnectorCardItem]) -> String {
        if members.count == 1 {
            return members[0].summary
        }

        let readyCount = members.filter { $0.health == .ready }.count
        let unavailableCount = members.filter { $0.health == .unavailable }.count
        let needsPermissionCount = members.filter { $0.health == .needsPermission }.count

        var summaryParts: [String] = [
            "\(members.count) mapped connectors",
            "\(readyCount) ready"
        ]
        if unavailableCount > 0 {
            summaryParts.append("\(unavailableCount) unavailable")
        }
        if needsPermissionCount > 0 {
            summaryParts.append("\(needsPermissionCount) permission required")
        }
        return summaryParts.joined(separator: " • ")
    }

    private static func logicalConnectorActions(for members: [ConnectorCardItem]) -> [DiagnosticsActionItem] {
        var actionsByKey: [String: DiagnosticsActionItem] = [:]
        var actionOrder: [String] = []

        for member in members {
            for action in member.actions {
                var parameters = action.parameters
                let existingConnectorID = parameters["connector_id"]?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
                if existingConnectorID.isEmpty {
                    parameters["connector_id"] = member.id
                }

                let normalizedAction = DiagnosticsActionItem(
                    id: action.id,
                    title: action.title,
                    intent: action.intent,
                    destination: action.destination,
                    parameters: parameters,
                    enabled: action.enabled,
                    recommended: action.recommended,
                    reason: action.reason
                )
                let key = "\(normalizedAction.id)|\(normalizedAction.intent)|\(normalizedAction.destination ?? "")|\(normalizedAction.title)"
                if actionsByKey[key] == nil {
                    actionsByKey[key] = normalizedAction
                    actionOrder.append(key)
                    continue
                }
                if normalizedAction.recommended, actionsByKey[key]?.recommended == false {
                    actionsByKey[key] = normalizedAction
                }
            }
        }

        return actionOrder.compactMap { actionsByKey[$0] }
    }

    private static func combinedCapabilityValues(from members: [ConnectorCardItem]) -> [String] {
        var seenCapabilityKeys: Set<String> = []
        var capabilityValues: [String] = []
        for member in members {
            let rawCapabilities: [String] = member.declaredCapabilities.isEmpty
                ? [member.details["Capabilities"] ?? ""]
                : member.declaredCapabilities
            for rawCapability in rawCapabilities {
                for capability in splitListTokens(rawCapability) {
                    let key = capability.lowercased()
                    if seenCapabilityKeys.insert(key).inserted {
                        capabilityValues.append(capability)
                    }
                }
            }
        }
        return capabilityValues
    }

    private func providerRequiresAPIKey(_ providerID: String) -> Bool {
        switch providerID.lowercased() {
        case "openai", "anthropic", "google":
            return true
        default:
            return false
        }
    }

    private func normalizedProviderID(_ raw: String) -> String {
        raw.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
    }

    private func providerDisplayName(_ providerID: String) -> String {
        switch normalizedProviderID(providerID) {
        case "openai":
            return "OpenAI"
        case "anthropic":
            return "Anthropic"
        case "google":
            return "Google"
        case "ollama":
            return "Ollama"
        default:
            return providerID.capitalized
        }
    }

    private func defaultProviderSecretName(for providerID: String) -> String {
        switch normalizedProviderID(providerID) {
        case "openai":
            return "OPENAI_API_KEY"
        case "anthropic":
            return "ANTHROPIC_API_KEY"
        case "google":
            return "GOOGLE_API_KEY"
        default:
            return "\(providerID.uppercased())_API_KEY"
        }
    }

    private func providerSecretReferenceMetadata(secretName: String) -> (
        workspaceID: String,
        name: String,
        backend: String,
        service: String,
        account: String
    ) {
        let normalizedWorkspace = nonEmpty(workspaceID) ?? Self.defaultWorkspaceID
        return (
            workspaceID: normalizedWorkspace,
            name: secretName,
            backend: "keyring",
            service: "personal-agent.\(normalizedWorkspace)",
            account: secretName
        )
    }

    private func formattedProviderTimestamp(_ value: String?) -> String {
        guard let value = value,
              let parsed = parseDaemonTimestamp(value) else {
            return "n/a"
        }
        return parsed.formatted(date: .abbreviated, time: .shortened)
    }

    private static func defaultProviderReadinessItems() -> [ProviderReadinessItem] {
        canonicalProviderOrder.map { providerID in
            ProviderReadinessItem(
                id: providerID,
                provider: providerID,
                endpoint: providerDefaultEndpoints[providerID] ?? "Not configured",
                status: .missingSetup,
                detail: "Provider is not configured for this workspace.",
                updatedAtLabel: "n/a"
            )
        }
    }

    private static func defaultProviderEndpointDrafts() -> [String: String] {
        canonicalProviderOrder.reduce(into: [String: String]()) { partialResult, providerID in
            partialResult[providerID] = providerDefaultEndpoints[providerID] ?? ""
        }
    }

    private static func defaultProviderSecretNameDrafts() -> [String: String] {
        canonicalProviderOrder.reduce(into: [String: String]()) { partialResult, providerID in
            switch providerID {
            case "openai":
                partialResult[providerID] = "OPENAI_API_KEY"
            case "anthropic":
                partialResult[providerID] = "ANTHROPIC_API_KEY"
            case "google":
                partialResult[providerID] = "GOOGLE_API_KEY"
            default:
                partialResult[providerID] = ""
            }
        }
    }

    private func normalizedContextFilter(_ raw: String) -> String? {
        let trimmed = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            return nil
        }
        if trimmed.lowercased() == "all" {
            return nil
        }
        return trimmed
    }

    private func clampedContextQueryLimit(_ raw: Int) -> Int {
        switch raw {
        case ..<1:
            return 25
        case 1...200:
            return raw
        default:
            return 200
        }
    }

    private func normalizedCapabilityGrantStatusFilter(_ raw: String) -> String? {
        let trimmed = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty, trimmed.lowercased() != "all" else {
            return nil
        }
        return trimmed.uppercased()
    }

    private func normalizedTrustStateFilter(_ raw: String) -> String? {
        let trimmed = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty, trimmed.lowercased() != "all" else {
            return nil
        }
        return trimmed.lowercased()
    }

    private func clampedGovernanceQueryLimit(_ raw: Int) -> Int {
        switch raw {
        case ..<1:
            return 25
        case 1...200:
            return raw
        default:
            return 200
        }
    }

    private func mapCapabilityGrantRecord(_ record: DaemonCapabilityGrantRecord) -> CapabilityGrantItem {
        let grantID = nonEmpty(record.grantID) ?? UUID().uuidString.lowercased()
        let normalizedStatus = nonEmpty(record.status)?.uppercased() ?? "ACTIVE"
        let scopeJSON = nonEmpty(record.scopeJSON) ?? "{}"
        let scopeSummary = truncateText(scopeJSON, limit: 180)
        let createdAtDate = parseDaemonTimestamp(record.createdAt) ?? .distantPast
        return CapabilityGrantItem(
            id: grantID,
            workspaceID: nonEmpty(record.workspaceID) ?? workspaceID,
            actorID: nonEmpty(record.actorID) ?? "unknown",
            capabilityKey: nonEmpty(record.capabilityKey) ?? "unknown",
            scopeJSON: scopeJSON,
            scopeSummary: scopeSummary,
            status: normalizedStatus,
            createdAtLabel: nonEmpty(record.createdAt).map(formattedWorkflowTimestamp) ?? "n/a",
            createdAtRaw: nonEmpty(record.createdAt) ?? "",
            expiresAtRaw: nonEmpty(record.expiresAt),
            expiresAtLabel: nonEmpty(record.expiresAt).map(formattedWorkflowTimestamp),
            sortTimestamp: createdAtDate
        )
    }

    private func mapTrustReceiptAuditLink(_ record: DaemonReceiptAuditLinkRecord) -> TrustReceiptAuditLinkItem {
        TrustReceiptAuditLinkItem(
            id: nonEmpty(record.auditID) ?? UUID().uuidString.lowercased(),
            eventType: nonEmpty(record.eventType) ?? "unknown",
            createdAtLabel: nonEmpty(record.createdAt).map(formattedWorkflowTimestamp) ?? "n/a"
        )
    }

    private func mapWebhookTrustReceiptItem(_ record: DaemonCommWebhookReceiptItem) -> WebhookTrustReceiptItem {
        WebhookTrustReceiptItem(
            id: nonEmpty(record.receiptID) ?? UUID().uuidString.lowercased(),
            workspaceID: nonEmpty(record.workspaceID) ?? workspaceID,
            provider: nonEmpty(record.provider) ?? "unknown",
            providerEventID: nonEmpty(record.providerEventID) ?? "unknown",
            trustState: nonEmpty(record.trustState)?.lowercased() ?? "unknown",
            signatureValid: record.signatureValid,
            signatureValuePresent: record.signatureValuePresent,
            payloadHash: nonEmpty(record.payloadHash) ?? "n/a",
            eventID: nonEmpty(record.eventID),
            threadID: nonEmpty(record.threadID),
            receivedAtLabel: nonEmpty(record.receivedAt).map(formattedWorkflowTimestamp),
            createdAtLabel: nonEmpty(record.createdAt).map(formattedWorkflowTimestamp) ?? "n/a",
            auditLinks: record.auditLinks.map(mapTrustReceiptAuditLink),
            sortTimestamp: parseDaemonTimestamp(record.createdAt) ?? .distantPast
        )
    }

    private func mapIngestTrustReceiptItem(_ record: DaemonCommIngestReceiptItem) -> IngestTrustReceiptItem {
        IngestTrustReceiptItem(
            id: nonEmpty(record.receiptID) ?? UUID().uuidString.lowercased(),
            workspaceID: nonEmpty(record.workspaceID) ?? workspaceID,
            source: nonEmpty(record.source) ?? "unknown",
            sourceScope: nonEmpty(record.sourceScope) ?? "unknown",
            sourceEventID: nonEmpty(record.sourceEventID) ?? "unknown",
            sourceCursor: nonEmpty(record.sourceCursor),
            trustState: nonEmpty(record.trustState)?.lowercased() ?? "unknown",
            payloadHash: nonEmpty(record.payloadHash) ?? "n/a",
            eventID: nonEmpty(record.eventID),
            threadID: nonEmpty(record.threadID),
            receivedAtLabel: nonEmpty(record.receivedAt).map(formattedWorkflowTimestamp),
            createdAtLabel: nonEmpty(record.createdAt).map(formattedWorkflowTimestamp) ?? "n/a",
            auditLinks: record.auditLinks.map(mapTrustReceiptAuditLink),
            sortTimestamp: parseDaemonTimestamp(record.createdAt) ?? .distantPast
        )
    }

    private func mapContextMemorySourceItem(_ record: DaemonContextMemorySourceRecord) -> MemorySourceItem {
        let fallbackID = UUID().uuidString.lowercased()
        let sourceID = nonEmpty(record.sourceID) ?? fallbackID
        return MemorySourceItem(
            id: sourceID,
            sourceType: nonEmpty(record.sourceType) ?? "unknown",
            sourceRef: nonEmpty(record.sourceRef) ?? "n/a",
            createdAtLabel: nonEmpty(record.createdAt).map(formattedWorkflowTimestamp) ?? "n/a"
        )
    }

    private func mapContextMemoryInventoryItem(_ record: DaemonContextMemoryInventoryRecord) -> MemoryInventoryItem {
        let memoryID = nonEmpty(record.memoryID) ?? UUID().uuidString.lowercased()
        let updatedAt = parseDaemonTimestamp(record.updatedAt)
            ?? parseDaemonTimestamp(record.createdAt)
            ?? .distantPast
        let sources = record.sources.map(mapContextMemorySourceItem)
        return MemoryInventoryItem(
            id: memoryID,
            workspaceID: nonEmpty(record.workspaceID) ?? workspaceID,
            ownerActorID: nonEmpty(record.ownerActorID) ?? "unknown",
            scopeType: nonEmpty(record.scopeType) ?? "unknown",
            key: nonEmpty(record.key) ?? "n/a",
            status: nonEmpty(record.status) ?? "unknown",
            kind: nonEmpty(record.kind) ?? "unknown",
            isCanonical: record.isCanonical,
            tokenEstimate: max(0, record.tokenEstimate),
            sourceSummary: nonEmpty(record.sourceSummary) ?? "No source summary.",
            sourceCount: max(record.sourceCount, sources.count),
            createdAtLabel: nonEmpty(record.createdAt).map(formattedWorkflowTimestamp) ?? "n/a",
            updatedAtLabel: nonEmpty(record.updatedAt).map(formattedWorkflowTimestamp) ?? "n/a",
            valueSummary: truncateText(nonEmpty(record.valueJSON) ?? "{}", limit: 180),
            sources: sources,
            sortTimestamp: updatedAt
        )
    }

    private func mapContextMemoryCandidateItem(_ record: DaemonContextMemoryCandidateRecord) -> MemoryCompactionCandidateItem {
        let candidateID = nonEmpty(record.candidateID) ?? UUID().uuidString.lowercased()
        let createdAt = parseDaemonTimestamp(record.createdAt) ?? .distantPast
        return MemoryCompactionCandidateItem(
            id: candidateID,
            workspaceID: nonEmpty(record.workspaceID) ?? workspaceID,
            ownerActorID: nonEmpty(record.ownerActorID) ?? "unknown",
            status: nonEmpty(record.status) ?? "unknown",
            score: record.score,
            candidateKind: nonEmpty(record.candidateKind) ?? "unknown",
            tokenEstimate: max(0, record.tokenEstimate),
            sourceIDs: record.sourceIDs.compactMap { nonEmpty($0) },
            sourceRefs: record.sourceRefs.compactMap { nonEmpty($0) },
            candidateSummary: truncateText(nonEmpty(record.candidateJSON) ?? "{}", limit: 180),
            createdAtLabel: nonEmpty(record.createdAt).map(formattedWorkflowTimestamp) ?? "n/a",
            sortTimestamp: createdAt
        )
    }

    private func mapContextRetrievalDocumentItem(_ record: DaemonContextRetrievalDocumentRecord) -> RetrievalDocumentItem {
        let documentID = nonEmpty(record.documentID) ?? UUID().uuidString.lowercased()
        return RetrievalDocumentItem(
            id: documentID,
            workspaceID: nonEmpty(record.workspaceID) ?? workspaceID,
            ownerActorID: nonEmpty(record.ownerActorID) ?? "unknown",
            sourceURI: nonEmpty(record.sourceURI) ?? "n/a",
            checksum: nonEmpty(record.checksum) ?? "n/a",
            chunkCount: max(0, record.chunkCount),
            createdAtLabel: nonEmpty(record.createdAt).map(formattedWorkflowTimestamp) ?? "n/a",
            sortTimestamp: parseDaemonTimestamp(record.createdAt) ?? .distantPast
        )
    }

    private func mapContextRetrievalChunkItem(_ record: DaemonContextRetrievalChunkRecord) -> RetrievalChunkItem {
        let chunkID = nonEmpty(record.chunkID) ?? UUID().uuidString.lowercased()
        return RetrievalChunkItem(
            id: chunkID,
            workspaceID: nonEmpty(record.workspaceID) ?? workspaceID,
            documentID: nonEmpty(record.documentID) ?? "unknown",
            ownerActorID: nonEmpty(record.ownerActorID) ?? "unknown",
            sourceURI: nonEmpty(record.sourceURI) ?? "n/a",
            chunkIndex: max(0, record.chunkIndex),
            tokenCount: max(0, record.tokenCount),
            textPreview: truncateText(nonEmpty(record.textBody) ?? "No text body.", limit: 220),
            createdAtLabel: nonEmpty(record.createdAt).map(formattedWorkflowTimestamp) ?? "n/a",
            sortTimestamp: parseDaemonTimestamp(record.createdAt) ?? .distantPast
        )
    }

    private func contextMemoryInventorySummaryMessage(
        itemCount: Int,
        hasMore: Bool,
        ownerActorID: String?,
        scopeType: String?,
        status: String?,
        sourceType: String?,
        sourceRefQuery: String?
    ) -> String {
        var parts: [String] = ["Memory inventory • \(itemCount) item(s)"]
        if let ownerActorID {
            parts.append("owner=\(ownerActorID)")
        }
        if let scopeType {
            parts.append("scope=\(scopeType)")
        }
        if let status {
            parts.append("status=\(status)")
        }
        if let sourceType {
            parts.append("source_type=\(sourceType)")
        }
        if let sourceRefQuery {
            parts.append("source_ref~\(sourceRefQuery)")
        }
        if hasMore {
            parts.append("more available")
        }
        return parts.joined(separator: " • ")
    }

    private func contextMemoryCandidatesSummaryMessage(
        itemCount: Int,
        hasMore: Bool,
        ownerActorID: String?,
        status: String?
    ) -> String {
        var parts: [String] = ["Compaction candidates • \(itemCount) item(s)"]
        if let ownerActorID {
            parts.append("owner=\(ownerActorID)")
        }
        if let status {
            parts.append("status=\(status)")
        }
        if hasMore {
            parts.append("more available")
        }
        return parts.joined(separator: " • ")
    }

    private func contextRetrievalDocumentsSummaryMessage(
        itemCount: Int,
        hasMore: Bool,
        ownerActorID: String?,
        sourceURIQuery: String?
    ) -> String {
        var parts: [String] = ["Retrieval documents • \(itemCount) item(s)"]
        if let ownerActorID {
            parts.append("owner=\(ownerActorID)")
        }
        if let sourceURIQuery {
            parts.append("source_uri~\(sourceURIQuery)")
        }
        if hasMore {
            parts.append("more available")
        }
        return parts.joined(separator: " • ")
    }

    private func contextRetrievalChunksSummaryMessage(
        itemCount: Int,
        hasMore: Bool,
        documentID: String,
        chunkTextQuery: String?
    ) -> String {
        var parts: [String] = ["Retrieval chunks • \(itemCount) item(s)"]
        parts.append("document=\(documentID)")
        if let chunkTextQuery {
            parts.append("text~\(chunkTextQuery)")
        }
        if hasMore {
            parts.append("more available")
        }
        return parts.joined(separator: " • ")
    }

    private func capabilityGrantInventorySummaryMessage(
        itemCount: Int,
        hasMore: Bool,
        actorID: String?,
        capabilityKey: String?,
        status: String?
    ) -> String {
        var parts: [String] = ["Capability grants • \(itemCount) item(s)"]
        if let actorID {
            parts.append("actor=\(actorID)")
        }
        if let capabilityKey {
            parts.append("capability=\(capabilityKey)")
        }
        if let status {
            parts.append("status=\(status)")
        }
        if hasMore {
            parts.append("more available")
        }
        return parts.joined(separator: " • ")
    }

    private func webhookReceiptSummaryMessage(
        itemCount: Int,
        hasMore: Bool,
        provider: String?,
        providerEventID: String?,
        providerEventQuery: String?,
        eventID: String?
    ) -> String {
        var parts: [String] = ["Webhook trust receipts • \(itemCount) item(s)"]
        if let provider {
            parts.append("provider=\(provider)")
        }
        if let providerEventID {
            parts.append("provider_event_id=\(providerEventID)")
        }
        if let providerEventQuery {
            parts.append("provider_event~\(providerEventQuery)")
        }
        if let eventID {
            parts.append("event_id=\(eventID)")
        }
        if hasMore {
            parts.append("more available")
        }
        return parts.joined(separator: " • ")
    }

    private func ingestReceiptSummaryMessage(
        itemCount: Int,
        hasMore: Bool,
        source: String?,
        sourceScope: String?,
        sourceEventID: String?,
        sourceEventQuery: String?,
        trustState: String?,
        eventID: String?
    ) -> String {
        var parts: [String] = ["Ingest trust receipts • \(itemCount) item(s)"]
        if let source {
            parts.append("source=\(source)")
        }
        if let sourceScope {
            parts.append("source_scope=\(sourceScope)")
        }
        if let sourceEventID {
            parts.append("source_event_id=\(sourceEventID)")
        }
        if let sourceEventQuery {
            parts.append("source_event~\(sourceEventQuery)")
        }
        if let trustState {
            parts.append("trust=\(trustState)")
        }
        if let eventID {
            parts.append("event_id=\(eventID)")
        }
        if hasMore {
            parts.append("more available")
        }
        return parts.joined(separator: " • ")
    }

    private struct LocalDevAuthBootstrapTransportTarget {
        let mode: String
        let address: String
    }

    private struct LocalDevAuthBootstrapCLIResponse: Decodable {
        let operation: String
        let tokenFile: String
        let tokenCreated: Bool
        let tokenRotated: Bool
        let activeProfile: String
        let nextStepReminder: String?

        enum CodingKeys: String, CodingKey {
            case operation
            case tokenFile = "token_file"
            case tokenCreated = "token_created"
            case tokenRotated = "token_rotated"
            case activeProfile = "active_profile"
            case nextStepReminder = "next_step_reminder"
        }
    }

    private enum LocalDevAuthBootstrapUIError: Error {
        case emptyCommandOutput
        case invalidCommandOutput
        case missingTokenFilePath
        case emptyTokenFile
    }

    private func localDevAuthBootstrapArguments() -> [String] {
        let transport = localDevAuthBootstrapTransportTarget()
        return [
            "auth",
            "bootstrap-local-dev",
            "--profile",
            "local-daemon",
            "--mode",
            transport.mode,
            "--address",
            transport.address,
            "--workspace",
            nonEmpty(workspaceID) ?? Self.defaultWorkspaceID
        ]
    }

    private func localDevAuthBootstrapTransportTarget() -> LocalDevAuthBootstrapTransportTarget {
        let fallback = LocalDevAuthBootstrapTransportTarget(mode: "tcp", address: "127.0.0.1:7071")
        let scheme = daemonBaseURL.scheme?.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() ?? ""
        switch scheme {
        case "http", "https":
            if let hostPort = daemonEndpointHostPort() {
                return LocalDevAuthBootstrapTransportTarget(mode: "tcp", address: hostPort)
            }
            return fallback
        case "unix":
            if let path = nonEmpty(daemonBaseURL.path) {
                return LocalDevAuthBootstrapTransportTarget(mode: "unix", address: path)
            }
            return fallback
        case "named_pipe", "named-pipe":
            if let host = nonEmpty(daemonBaseURL.host) {
                let suffixPath = daemonBaseURL.path.trimmingCharacters(in: .whitespacesAndNewlines)
                let resolved = nonEmpty("\(host)\(suffixPath)") ?? host
                return LocalDevAuthBootstrapTransportTarget(mode: "named_pipe", address: resolved)
            }
            if let path = nonEmpty(daemonBaseURL.path) {
                return LocalDevAuthBootstrapTransportTarget(mode: "named_pipe", address: path)
            }
            return fallback
        default:
            let endpoint = daemonEndpointLabel.trimmingCharacters(in: .whitespacesAndNewlines)
            if !endpoint.isEmpty, !endpoint.contains("://") {
                return LocalDevAuthBootstrapTransportTarget(mode: "tcp", address: endpoint)
            }
            return fallback
        }
    }

    private func daemonEndpointHostPort() -> String? {
        guard let host = nonEmpty(daemonBaseURL.host) else {
            return nil
        }
        let port = daemonBaseURL.port
            ?? ((daemonBaseURL.scheme?.lowercased() == "https") ? 443 : 80)
        return "\(host):\(port)"
    }

    private func decodeLocalDevAuthBootstrapCLIResponse(
        stdout: String
    ) throws -> LocalDevAuthBootstrapCLIResponse {
        let trimmed = stdout.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            throw LocalDevAuthBootstrapUIError.emptyCommandOutput
        }
        guard let data = trimmed.data(using: .utf8) else {
            throw LocalDevAuthBootstrapUIError.invalidCommandOutput
        }
        let response = try JSONDecoder().decode(LocalDevAuthBootstrapCLIResponse.self, from: data)
        guard response.operation == "bootstrap_local_dev" else {
            throw LocalDevAuthBootstrapUIError.invalidCommandOutput
        }
        return response
    }

    private func loadLocalDevAuthTokenFromFile(_ rawTokenFilePath: String) throws -> String {
        let tokenFilePath = rawTokenFilePath.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !tokenFilePath.isEmpty else {
            throw LocalDevAuthBootstrapUIError.missingTokenFilePath
        }
        let tokenRaw = try String(contentsOfFile: tokenFilePath, encoding: .utf8)
        let token = tokenRaw.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !token.isEmpty else {
            throw LocalDevAuthBootstrapUIError.emptyTokenFile
        }
        return token
    }

    private func localDevAuthBootstrapFailureMessage(stderr: String, exitCode: Int32) -> String {
        let trimmed = stderr.trimmingCharacters(in: .whitespacesAndNewlines)
        let normalized = trimmed.lowercased()
        if normalized.contains("command not found")
            || normalized.contains("no such file")
            || normalized.contains("could not find") {
            return "Bootstrap command failed because `personal-agent` is not available on PATH. Build or install CLI, then retry."
        }
        if let firstLine = nonEmpty(trimmed.components(separatedBy: .newlines).first) {
            return "Bootstrap command failed (\(exitCode)): \(truncateText(firstLine, limit: 220))"
        }
        return "Bootstrap command failed (\(exitCode)). Retry or copy command to run manually."
    }

    private static func executeLocalDevAuthBootstrapShellCommand(
        arguments: [String]
    ) async -> LocalDevAuthBootstrapCommandExecution {
        await withCheckedContinuation { continuation in
            DispatchQueue.global(qos: .userInitiated).async {
                let process = Process()
                process.executableURL = URL(fileURLWithPath: "/usr/bin/env")
                process.arguments = arguments

                let stdoutPipe = Pipe()
                let stderrPipe = Pipe()
                process.standardOutput = stdoutPipe
                process.standardError = stderrPipe

                do {
                    try process.run()
                    process.waitUntilExit()
                    let stdoutData = stdoutPipe.fileHandleForReading.readDataToEndOfFile()
                    let stderrData = stderrPipe.fileHandleForReading.readDataToEndOfFile()
                    let stdout = String(data: stdoutData, encoding: .utf8) ?? ""
                    let stderr = String(data: stderrData, encoding: .utf8) ?? ""
                    continuation.resume(returning: (process.terminationStatus, stdout, stderr))
                } catch {
                    continuation.resume(returning: (-1, "", error.localizedDescription))
                }
            }
        }
    }

    private static func shellEscapedCommandArgument(_ value: String) -> String {
        if value.range(of: #"^[A-Za-z0-9._:/@+-]+$"#, options: .regularExpression) != nil {
            return value
        }
        let escaped = value.replacingOccurrences(of: "'", with: "'\"'\"'")
        return "'\(escaped)'"
    }

    private func parseDaemonTimestamp(_ value: String) -> Date? {
        let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            return nil
        }
        return daemonTimestampParser().date(from: trimmed)
            ?? ISO8601DateFormatter().date(from: trimmed)
    }

    private func truncateText(_ value: String, limit: Int) -> String {
        guard value.count > limit else {
            return value
        }
        let endIndex = value.index(value.startIndex, offsetBy: limit)
        return "\(value[..<endIndex])…"
    }

    private func assignIfChanged<T: Equatable>(
        _ keyPath: ReferenceWritableKeyPath<AppShellState, T>,
        _ value: T
    ) {
        guard self[keyPath: keyPath] != value else {
            return
        }
        self[keyPath: keyPath] = value
    }

    private func normalizedDrillInContext(_ context: DrillInNavigationContext) -> DrillInNavigationContext {
        var normalizedChips: [String] = []
        for chip in context.chips {
            guard let normalized = nonEmpty(chip) else {
                continue
            }
            let bounded = normalized.count > 36 ? "\(normalized.prefix(36))…" : normalized
            guard !normalizedChips.contains(bounded) else {
                continue
            }
            normalizedChips.append(bounded)
            if normalizedChips.count >= 3 {
                break
            }
        }
        return DrillInNavigationContext(
            sourceSection: context.sourceSection,
            destinationSection: context.destinationSection,
            chips: normalizedChips
        )
    }

    private func makeDrillInChip(label: String, value: String?) -> String? {
        guard let value = nonEmpty(value) else {
            return nil
        }
        let bounded = value.count > 24 ? "\(value.prefix(24))…" : value
        return "\(label): \(bounded)"
    }

    private func nonEmpty(_ value: String?) -> String? {
        guard let trimmed = value?.trimmingCharacters(in: .whitespacesAndNewlines), !trimmed.isEmpty else {
            return nil
        }
        return trimmed
    }

    static func _test_setLocalDevAuthBootstrapCommandRunner(
        _ runner: @escaping LocalDevAuthBootstrapCommandRunner
    ) {
        localDevAuthBootstrapCommandRunner = runner
    }

    static func _test_setLocalDevAuthBootstrapRefreshHandler(
        _ handler: @escaping LocalDevAuthBootstrapRefreshHandler
    ) {
        localDevAuthBootstrapRefreshHandler = handler
    }

    static func _test_setDaemonLocalServiceInstallRunner(
        _ runner: @escaping DaemonLocalServiceInstallRunner
    ) {
        daemonLocalServiceInstallRunner = runner
    }

    static func _test_setDaemonLifecycleControlRunner(
        _ runner: @escaping DaemonLifecycleControlRunner
    ) {
        daemonLifecycleControlRunner = runner
    }

    static func _test_setLocalDevTokenSecretReference(service: String, account: String) {
        localDevTokenSecretReferenceOverride = LocalDevTokenSecretReference(
            service: service,
            account: account
        )
    }

    static func _test_readPersistedLocalDevToken() -> String? {
        let secretReference = localDevTokenSecretReference()
        return try? LocalSecretStore.readSecret(
            service: secretReference.service,
            account: secretReference.account
        )
    }

    static func _test_loadPersistedLocalDevToken() -> String? {
        loadPersistedLocalDevToken()
    }

    static func _test_clearPersistedLocalDevToken() {
        _ = try? clearPersistedLocalDevToken()
    }

    static func _test_resetLocalDevAuthBootstrapHooks() {
        localDevAuthBootstrapCommandRunner = defaultLocalDevAuthBootstrapCommandRunner
        localDevAuthBootstrapRefreshHandler = defaultLocalDevAuthBootstrapRefreshHandler
        localDevTokenSecretReferenceOverride = nil
    }

    static func _test_resetDaemonLocalServiceInstallHooks() {
        daemonLocalServiceInstallRunner = defaultDaemonLocalServiceInstallRunner
    }

    static func _test_resetDaemonLifecycleControlHooks() {
        daemonLifecycleControlRunner = defaultDaemonLifecycleControlRunner
    }

    static func _test_resetLocalDevTokenPersistenceHooks() {
        localDevTokenSecretReferenceOverride = nil
    }
}
