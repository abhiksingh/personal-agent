import Combine
import Foundation

@MainActor
public final class AppShellV2Store: ObservableObject {
    @Published public var selectedSection: AssistantWorkspaceSection = .replayAndAsk {
        didSet {
            guard selectedSection != oldValue else { return }
            sessionConfigStore.persistSelectedSection(selectedSection)
        }
    }
    @Published public var selectedSources: Set<ReplaySource> = [] {
        didSet { handleReplayFilterChange() }
    }
    @Published public var statusFilter: ReplayStatusFilter = .needsApproval {
        didSet { handleReplayFilterChange() }
    }
    @Published public var searchQuery: String = "" {
        didSet { handleReplayFilterChange() }
    }
    @Published public var selectedEventID: ReplayEvent.ID?
    @Published public var askDraft: String = ""
    @Published public var connectors: [ConnectorState] {
        didSet { refreshPanelLifecycleStates() }
    }
    @Published public var models: [ModelOption] {
        didSet { refreshPanelLifecycleStates() }
    }
    @Published public var activeModelID: ModelOption.ID?
    @Published public var replayEvents: [ReplayEvent] {
        didSet { refreshPanelLifecycleStates() }
    }
    @Published public var replayDetailEvidenceByReplayKey: [String: V2ReplayDetailEvidenceState] = [:]
    @Published public var replayFeedQueryState = V2ReplayFeedQueryState() {
        didSet { refreshPanelLifecycleStates() }
    }
    @Published public var replayRealtimeState = V2ReplayRealtimeState()
    @Published public var hasLoadedConnectorInventory = false {
        didSet { refreshPanelLifecycleStates() }
    }
    @Published public var isConnectorInventoryRefreshInFlight = false {
        didSet { refreshPanelLifecycleStates() }
    }
    @Published public var connectorActionStatusByID: [ConnectorState.ID: String] = [:]
    @Published public var connectorActionInFlightByID: [ConnectorState.ID: V2ConnectorRowAction] = [:] {
        didSet { refreshPanelLifecycleStates() }
    }
    @Published public var hasLoadedModelInventory = false {
        didSet { refreshPanelLifecycleStates() }
    }
    @Published public var isModelInventoryRefreshInFlight = false {
        didSet { refreshPanelLifecycleStates() }
    }
    @Published public var modelActionStatusByID: [ModelOption.ID: String] = [:]
    @Published public var modelActionInFlightByID: [ModelOption.ID: V2ModelRowAction] = [:] {
        didSet { refreshPanelLifecycleStates() }
    }
    @Published public var modelRouteTaskClass: String = "chat"
    @Published public var modelRouteResolution: V2DaemonModelResolveResponse?
    @Published public var modelRouteSimulation: V2DaemonModelRouteSimulationResponse?
    @Published public var modelRouteExplainability: V2DaemonModelRouteExplainResponse?
    @Published public var modelRouteStatusMessage: String?
    @Published public var modelRouteSimulationStatusMessage: String?
    @Published public var modelRouteExplainStatusMessage: String?
    @Published public private(set) var lastFeedback: String?
    @Published public private(set) var panelProblemByContext: [V2ProblemContext: V2PanelProblemState] = [:] {
        didSet { refreshPanelLifecycleStates() }
    }
    @Published public var accessTokenInput: String = ""
    @Published public var isDaemonProbeInFlight = false {
        didSet { refreshPanelLifecycleStates() }
    }
    @Published public var getStartedReadinessSnapshot = V2GetStartedReadinessSnapshot() {
        didSet { refreshPanelLifecycleStates() }
    }
    @Published public private(set) var setupActionStatusByStep: [V2SetupChecklistStepID: String] = [:]
    @Published public var isReadinessRefreshInFlight = false {
        didSet { refreshPanelLifecycleStates() }
    }
    @Published public private(set) var panelStateByWorkflow: [V2WorkflowPanel: V2PanelLifecycleState] = [:]
    @Published public private(set) var mutationStateByAction: [V2MutationActionID: V2MutationLifecycleState] = [:]

    public let sessionConfigStore: V2SessionConfigStore

    nonisolated let daemonClient: V2DaemonAPIClient
    private var sessionConfigCancellable: AnyCancellable?
    private var replayEventIDCache: [String: ReplayEvent.ID] = [:]
    var replayDetailFetchInFlightKeys: Set<String> = []
    var replayRealtimeTask: Task<Void, Never>?
    var replayRealtimeReconnectTask: Task<Void, Never>?
    var replayRealtimeRefreshTask: Task<Void, Never>?
    var replayRealtimeSession: V2DaemonRealtimeSession?
    var replayRealtimeConnectionID: UUID?

    public init(
        connectors: [ConnectorState] = AppShellV2Store.defaultConnectors,
        models: [ModelOption] = AppShellV2Store.defaultModels,
        replayEvents: [ReplayEvent] = AppShellV2Store.defaultReplayEvents,
        daemonClient: V2DaemonAPIClient = V2DaemonAPIClient(),
        sessionConfigStore: V2SessionConfigStore = V2SessionConfigStore()
    ) {
        self.sessionConfigStore = sessionConfigStore
        self.daemonClient = daemonClient
        self.connectors = connectors
        self.models = models
        self.activeModelID = models.first(where: { $0.enabled })?.id

        let sortedEvents = replayEvents.sorted(by: { $0.receivedAt > $1.receivedAt })
        self.replayEvents = sortedEvents
        self.replayFeedQueryState = V2ReplayFeedQueryState(hasLoadedOnce: !sortedEvents.isEmpty)
        self.selectedEventID = sortedEvents.first?.id
        self.selectedSection = sessionConfigStore.selectedSection

        sessionConfigCancellable = sessionConfigStore.objectWillChange.sink { [weak self] _ in
            self?.objectWillChange.send()
            self?.refreshPanelLifecycleStates()
        }
        refreshPanelLifecycleStates()
    }

    public var filteredEvents: [ReplayEvent] {
        let sourceFiltered = selectedSources.isEmpty
            ? replayEvents
            : replayEvents.filter { selectedSources.contains($0.source) }

        let statusFiltered = sourceFiltered.filter { statusFilter.matches($0.status) }

        let query = searchQuery.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !query.isEmpty else {
            return statusFiltered
        }

        return statusFiltered.filter { $0.searchableText.localizedCaseInsensitiveContains(query) }
    }

    public var selectedEvent: ReplayEvent? {
        guard let selectedEventID else {
            return filteredEvents.first
        }
        return filteredEvents.first(where: { $0.id == selectedEventID })
    }

    public var setupProgress: Double {
        let steps = setupChecklist
        guard !steps.isEmpty else { return 0 }
        let completeCount = steps.filter(\.isDone).count
        return Double(completeCount) / Double(steps.count)
    }

    public var setupChecklist: [SetupChecklistItem] {
        let lifecycleDetail = lifecycleChecklistDetail
        let routeDetail = routeChecklistDetail
        let connectorDetail = connectorChecklistDetail
        let replayDetail = replayChecklistDetail

        return [
            SetupChecklistItem(
                id: .sessionConfiguration,
                title: "Session configuration is complete",
                detail: "Set daemon URL, access token, workspace, and principal to unlock live actions.",
                isDone: sessionConfigStore.readiness.isReadyForDaemonMutations,
                actionTitle: "Configure Session",
                panelActionID: .openGetStarted,
                action: { [weak self] in
                    self?.performSetupChecklistAction(.sessionConfiguration)
                }
            ),
            SetupChecklistItem(
                id: .lifecycleOperational,
                title: "Daemon lifecycle is operational",
                detail: lifecycleDetail,
                isDone: getStartedReadinessSnapshot.lifecycleIsOperational,
                actionTitle: "Verify Daemon",
                panelActionID: .retry,
                action: { [weak self] in
                    self?.performSetupChecklistAction(.lifecycleOperational)
                }
            ),
            SetupChecklistItem(
                id: .defaultRouteResolved,
                title: "Default model route is resolved",
                detail: routeDetail,
                isDone: getStartedReadinessSnapshot.hasRouteResolution,
                actionTitle: "Open Connectors & Models",
                panelActionID: .openConnectorsAndModels,
                action: { [weak self] in
                    self?.performSetupChecklistAction(.defaultRouteResolved)
                }
            ),
            SetupChecklistItem(
                id: .connectorHealthy,
                title: "At least one external connector is healthy",
                detail: connectorDetail,
                isDone: getStartedReadinessSnapshot.connectedConnectorCount > 0,
                actionTitle: "Open Connectors & Models",
                panelActionID: .openConnectorsAndModels,
                action: { [weak self] in
                    self?.performSetupChecklistAction(.connectorHealthy)
                }
            ),
            SetupChecklistItem(
                id: .replayActivityAvailable,
                title: "Replay has live assistant activity",
                detail: replayDetail,
                isDone: getStartedReadinessSnapshot.hasReplayActivity,
                actionTitle: "Open Replay & Ask",
                panelActionID: .openReplayAndAsk,
                action: { [weak self] in
                    self?.performSetupChecklistAction(.replayActivityAvailable)
                }
            )
        ]
    }

    public var currentSetupBlocker: SetupChecklistItem? {
        setupChecklist.first(where: { !$0.isDone })
    }

    public var shouldShowSetupBlockerRibbon: Bool {
        selectedSection != .getStarted && currentSetupBlocker != nil
    }

    public func setupActionStatus(for stepID: V2SetupChecklistStepID) -> String? {
        setupActionStatusByStep[stepID]
    }

    public func fixNextSetupBlocker() {
        guard let blocker = currentSetupBlocker else {
            return
        }
        blocker.action()
    }

    public func performSetupChecklistAction(_ stepID: V2SetupChecklistStepID) {
        switch stepID {
        case .sessionConfiguration:
            selectedSection = .getStarted
            setSetupActionStatus(stepID, message: "Update session settings, then verify daemon.")
        case .lifecycleOperational:
            selectedSection = .getStarted
            setSetupActionStatus(stepID, message: "Verifying daemon lifecycle…")
            Task { [weak self] in
                await self?.probeDaemonConnection()
            }
        case .defaultRouteResolved:
            selectedSection = .connectorsAndModels
            setSetupActionStatus(stepID, message: "Configure or select a default chat route in Models.")
        case .connectorHealthy:
            selectedSection = .connectorsAndModels
            setSetupActionStatus(stepID, message: "Connect and verify at least one external connector.")
        case .replayActivityAvailable:
            selectedSection = .replayAndAsk
            setSetupActionStatus(stepID, message: "Run one real instruction so Replay has live evidence.")
        }
    }

    public var metrics: ReplayMetrics {
        ReplayMetrics(
            total: replayEvents.count,
            completed: replayEvents.filter { $0.status == .completed }.count,
            needsApproval: replayEvents.filter { $0.status == .awaitingApproval }.count,
            running: replayEvents.filter { $0.status == .running }.count,
            failed: replayEvents.filter { $0.status == .failed }.count
        )
    }

    public var hasActiveReplayFilters: Bool {
        !selectedSources.isEmpty || statusFilter != .needsApproval || !searchQuery.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }

    public var replayFilterSummary: String {
        var clauses: [String] = []

        clauses.append("Status: \(statusFilter.label)")

        if !selectedSources.isEmpty {
            let sourceNames = selectedSources
                .sorted(by: { $0.label < $1.label })
                .map(\.label)
                .joined(separator: ", ")
            clauses.append("Sources: \(sourceNames)")
        }

        let query = searchQuery.trimmingCharacters(in: .whitespacesAndNewlines)
        if !query.isEmpty {
            clauses.append("Search: \"\(query)\"")
        }

        return clauses.joined(separator: "  •  ")
    }

    public var activeModel: ModelOption? {
        guard let activeModelID else { return nil }
        return models.first(where: { $0.id == activeModelID })
    }

    public var sessionReadiness: V2SessionReadiness {
        sessionConfigStore.readiness
    }

    public var highImpactActionDisabledReason: String? {
        sessionReadiness.highImpactActionDisabledReason
    }

    public var canRunHighImpactActions: Bool {
        highImpactActionDisabledReason == nil
    }

    public var daemonBaseURL: String {
        get { sessionConfigStore.daemonBaseURL }
        set { sessionConfigStore.daemonBaseURL = newValue }
    }

    public var workspaceID: String {
        get { sessionConfigStore.workspaceID }
        set { sessionConfigStore.workspaceID = newValue }
    }

    public var principalActorID: String {
        get { sessionConfigStore.principalActorID }
        set { sessionConfigStore.principalActorID = newValue }
    }

    public var informationDensityMode: V2InformationDensityMode {
        get { sessionConfigStore.informationDensityMode }
        set { sessionConfigStore.informationDensityMode = newValue }
    }

    public var hasStoredAccessToken: Bool {
        sessionConfigStore.hasStoredAccessToken
    }

    public var tokenReferenceLabel: String {
        sessionConfigStore.tokenReferenceLabel
    }

    public var connectedLiveConnectorCount: Int {
        getStartedReadinessSnapshot.connectedConnectorCount
    }

    public var liveReplayInstructionCount: Int {
        getStartedReadinessSnapshot.replayAvailability?.total ?? 0
    }

    public func panelProblem(for context: V2ProblemContext) -> V2PanelProblemState? {
        panelProblemByContext[context]
    }

    public func panelLifecycleState(for workflow: V2WorkflowPanel) -> V2PanelLifecycleState {
        panelStateByWorkflow[workflow] ?? .loading("Checking current status…")
    }

    public func mutationLifecycle(for actionID: V2MutationActionID) -> V2MutationLifecycleState {
        if let disabledReason = mutationDisabledReason(for: actionID) {
            return .disabled(disabledReason)
        }
        return mutationStateByAction[actionID] ?? .idle
    }

    public func clearPanelProblem(for context: V2ProblemContext) {
        panelProblemByContext.removeValue(forKey: context)
    }

    public func setPanelProblem(_ error: Error, context: V2ProblemContext) {
        panelProblemByContext[context] = V2DaemonProblemMapper.map(error: error, context: context)
    }

    public func handlePanelProblemAction(_ actionID: V2ProblemActionID) {
        switch actionID {
        case .retry, .retryRealtime:
            return
        case .clearReplayFilters:
            clearFilters()
            selectedSection = .replayAndAsk
        case .openGetStarted:
            selectedSection = .getStarted
        case .openConnectorsAndModels:
            selectedSection = .connectorsAndModels
        case .openReplayAndAsk:
            selectedSection = .replayAndAsk
        case .reauthenticate:
            selectedSection = .getStarted
            setFeedback("Update access token in Get Started, then retry.")
        }
    }

    public func performPanelStateAction(_ actionID: V2ProblemActionID, workflow: V2WorkflowPanel) {
        switch actionID {
        case .retry:
            switch workflow {
            case .getStarted:
                Task { await probeDaemonConnection() }
            case .replayAndAsk:
                Task {
                    await refreshReplayFeed(resetPagination: true)
                }
            case .connectorsAndModels:
                clearPanelProblem(for: .connectors)
                clearPanelProblem(for: .models)
                Task {
                    await refreshConnectorsInventory(force: true)
                    await refreshModelsInventory(force: true)
                }
            }
        case .retryRealtime:
            retryReplayRealtimeStream()
        default:
            handlePanelProblemAction(actionID)
        }
    }

    public func saveAccessTokenFromInput() {
        startMutation(.tokenSave, message: "Saving assistant access token…")
        do {
            try sessionConfigStore.saveAccessToken(accessTokenInput)
            accessTokenInput = ""
            setFeedback("Assistant access token saved to Keychain.")
            completeMutation(.tokenSave, message: "Assistant access token saved.")
            Task { [weak self] in
                await self?.refreshGetStartedReadinessIfNeeded(force: true)
            }
        } catch {
            setFeedback(error.localizedDescription)
            failMutation(.tokenSave, message: error.localizedDescription)
        }
    }

    public func clearStoredAccessToken() {
        startMutation(.tokenClear, message: "Clearing assistant access token…")
        do {
            try sessionConfigStore.clearAccessToken()
            accessTokenInput = ""
            setFeedback("Assistant access token cleared.")
            completeMutation(.tokenClear, message: "Assistant access token cleared.")
            getStartedReadinessSnapshot = V2GetStartedReadinessSnapshot(lastUpdatedAt: Date())
            clearPanelProblem(for: .models)
            clearPanelProblem(for: .connectors)
            clearPanelProblem(for: .replay)
            clearPanelProblem(for: .realtime)
            hasLoadedConnectorInventory = false
            connectorActionInFlightByID.removeAll()
            connectorActionStatusByID.removeAll()
            hasLoadedModelInventory = false
            modelActionInFlightByID.removeAll()
            modelActionStatusByID.removeAll()
            modelRouteResolution = nil
            modelRouteSimulation = nil
            modelRouteExplainability = nil
            modelRouteStatusMessage = nil
            modelRouteSimulationStatusMessage = nil
            modelRouteExplainStatusMessage = nil
            Task { [weak self] in
                await self?.stopReplayRealtimeStream()
            }
        } catch {
            setFeedback(error.localizedDescription)
            failMutation(.tokenClear, message: error.localizedDescription)
        }
    }

    func replaceReplayEvents(_ events: [ReplayEvent]) {
        replayEvents = events.sorted(by: { $0.receivedAt > $1.receivedAt })
        let knownKeys = Set(replayEvents.map(\.replayKey))
        replayDetailEvidenceByReplayKey = replayDetailEvidenceByReplayKey.filter { knownKeys.contains($0.key) }
        ensureSelectionIsVisible()
    }

    public func replayDetailEvidence(for event: ReplayEvent?) -> V2ReplayDetailEvidenceState? {
        guard let event else {
            return nil
        }
        return replayDetailEvidenceByReplayKey[event.replayKey]
    }

    func stableReplayEventID(for replayKey: String) -> ReplayEvent.ID {
        let normalized = replayKey.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        let cacheKey = normalized.isEmpty ? UUID().uuidString.lowercased() : normalized
        if let existing = replayEventIDCache[cacheKey] {
            return existing
        }
        let created = UUID()
        replayEventIDCache[cacheKey] = created
        return created
    }

    func updateEvent(_ eventID: ReplayEvent.ID, mutate: (inout ReplayEvent) -> Void) {
        guard let index = replayEvents.firstIndex(where: { $0.id == eventID }) else {
            return
        }

        var updated = replayEvents
        mutate(&updated[index])
        replaceReplayEvents(updated)
    }

    func setFeedback(_ message: String) {
        lastFeedback = message
    }

    func clearFeedback() {
        lastFeedback = nil
    }

    func ensureSelectionIsVisible() {
        if let selectedEvent, !filteredEvents.contains(selectedEvent) {
            selectedEventID = filteredEvents.first?.id
        }
        if selectedEventID == nil {
            selectedEventID = filteredEvents.first?.id
        }
    }

    private func handleReplayFilterChange() {
        ensureSelectionIsVisible()
        refreshPanelLifecycleStates()
    }

    private var lifecycleChecklistDetail: String {
        if !sessionReadiness.isReadyForDaemonMutations {
            return "Complete session setup first, then verify daemon lifecycle."
        }

        if let lifecycleError = getStartedReadinessSnapshot.lifecycleError {
            return lifecycleError
        }

        guard let lifecycleStatus = getStartedReadinessSnapshot.lifecycleStatus else {
            return "Verify daemon to load lifecycle and worker health."
        }

        if lifecycleStatus.needsInstall {
            return "Daemon install is incomplete. Run lifecycle install/repair and verify again."
        }
        if lifecycleStatus.needsRepair {
            return lifecycleStatus.repairHint ?? "Daemon requires repair before trust workflows are reliable."
        }

        if lifecycleStatus.workerSummary.failed > 0 {
            return "\(lifecycleStatus.workerSummary.failed) plugin worker(s) failed. Open Connectors & Models to remediate."
        }

        let lifecycleState = lifecycleStatus.lifecycleState.trimmingCharacters(in: .whitespacesAndNewlines)
        let healthState = lifecycleStatus.healthClassification.overallState.trimmingCharacters(in: .whitespacesAndNewlines)
        let authState = lifecycleStatus.controlAuth.state.trimmingCharacters(in: .whitespacesAndNewlines)

        if getStartedReadinessSnapshot.lifecycleIsOperational {
            return "Daemon \(lifecycleState) • health \(healthState) • auth \(authState)."
        }
        return "Daemon \(lifecycleState) • health \(healthState) • auth \(authState). Resolve this before trusting automation."
    }

    private var routeChecklistDetail: String {
        if !sessionReadiness.isReadyForDaemonMutations {
            return "Complete session setup first, then resolve the default model route."
        }

        if let routeError = getStartedReadinessSnapshot.modelRouteError {
            return routeError
        }

        guard let route = getStartedReadinessSnapshot.modelRoute else {
            return "No route snapshot yet. Verify daemon to load route readiness."
        }

        let provider = route.provider.trimmingCharacters(in: .whitespacesAndNewlines)
        let modelKey = route.modelKey.trimmingCharacters(in: .whitespacesAndNewlines)
        if getStartedReadinessSnapshot.hasRouteResolution {
            return "Chat route resolves to \(provider) / \(modelKey) from \(route.source)."
        }
        return "Route is unresolved. Configure provider/model routing in Connectors & Models."
    }

    private var connectorChecklistDetail: String {
        if !sessionReadiness.isReadyForDaemonMutations {
            return "Complete session setup first, then check connector health."
        }

        if let connectorError = getStartedReadinessSnapshot.connectorError {
            return connectorError
        }

        if getStartedReadinessSnapshot.connectorCards.isEmpty {
            return "No connector inventory loaded yet. Verify daemon to refresh connector readiness."
        }

        let connected = getStartedReadinessSnapshot.connectedConnectorCount
        let attention = getStartedReadinessSnapshot.connectorsNeedingAttentionCount
        if connected > 0 {
            if attention > 0 {
                return "\(connected) connector(s) healthy, \(attention) need attention."
            }
            return "\(connected) connector(s) healthy."
        }
        return "No healthy connectors yet. Connect at least one external channel."
    }

    private var replayChecklistDetail: String {
        if !sessionReadiness.isReadyForDaemonMutations {
            return "Complete session setup first, then verify replay activity."
        }

        if let replayError = getStartedReadinessSnapshot.replayError {
            return replayError
        }

        guard let availability = getStartedReadinessSnapshot.replayAvailability else {
            return "Replay availability has not been checked yet."
        }

        if availability.total > 0 {
            return "Found \(availability.total) live replay signals across approvals, tasks, and chat history."
        }
        return "No replay activity yet. Send one instruction via App Chat or an external channel."
    }

    func setSetupActionStatus(_ stepID: V2SetupChecklistStepID, message: String) {
        setupActionStatusByStep[stepID] = message
        setFeedback(message)
    }

    private func pruneCompletedSetupActionStatus() {
        let incomplete = Set(setupChecklist.filter { !$0.isDone }.map(\.id))
        if incomplete.isEmpty {
            setupActionStatusByStep.removeAll()
            return
        }
        setupActionStatusByStep = setupActionStatusByStep.filter { incomplete.contains($0.key) }
    }

    func startMutation(_ actionID: V2MutationActionID, message: String? = nil) {
        mutationStateByAction[actionID] = .inFlight(message)
        refreshPanelLifecycleStates()
    }

    func completeMutation(_ actionID: V2MutationActionID, message: String? = nil) {
        mutationStateByAction[actionID] = .succeeded(message)
        refreshPanelLifecycleStates()
    }

    func failMutation(_ actionID: V2MutationActionID, message: String? = nil) {
        mutationStateByAction[actionID] = .failed(message)
        refreshPanelLifecycleStates()
    }

    private func mutationDisabledReason(for actionID: V2MutationActionID) -> String? {
        switch actionID {
        case .askSend,
                .replayApprove,
                .replayReject,
                .replayRetry,
                .replayComplete,
                .connectorToggle,
                .connectorCheck,
                .connectorSaveConfig,
                .connectorPermission,
                .modelToggle,
                .modelSetPrimary,
                .modelRouteSimulate,
                .modelRouteExplain:
            return highImpactActionDisabledReason
        case .daemonProbe:
            if !sessionReadiness.hasValidDaemonBaseURL {
                return "Set a valid daemon URL in Get Started before verifying daemon connection."
            }
            return nil
        case .tokenSave:
            return accessTokenInput.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
                ? "Paste an assistant access token before saving."
                : nil
        case .tokenClear:
            return hasStoredAccessToken ? nil : "No stored access token to clear."
        }
    }

    private func refreshPanelLifecycleStates() {
        pruneCompletedSetupActionStatus()
        panelStateByWorkflow[.getStarted] = resolveGetStartedPanelState()
        panelStateByWorkflow[.replayAndAsk] = resolveReplayPanelState()
        panelStateByWorkflow[.connectorsAndModels] = resolveConnectorsModelsPanelState()
    }

    private func resolveGetStartedPanelState() -> V2PanelLifecycleState {
        if mutationLifecycle(for: .daemonProbe).isInFlight || isDaemonProbeInFlight || isReadinessRefreshInFlight {
            return .loading("Verifying daemon connection.")
        }
        if let setupProblem = panelProblem(for: .setup) {
            return lifecycleState(from: setupProblem)
        }
        if !sessionReadiness.isReadyForDaemonMutations {
            return .degraded(
                sessionReadiness.setupSummary,
                actions: [
                    V2ProblemAction(actionID: .openGetStarted, label: "Configure Session", isPrimary: true)
                ]
            )
        }

        if let nextStep = setupChecklist.first(where: { !$0.isDone }) {
            return .degraded(
                nextStep.detail,
                actions: [
                    V2ProblemAction(actionID: nextStep.panelActionID, label: nextStep.actionTitle, isPrimary: true)
                ]
            )
        }

        return .ready("Setup is complete and trust checks are healthy.")
    }

    private func resolveReplayPanelState() -> V2PanelLifecycleState {
        if mutationLifecycle(for: .replayApprove).isInFlight ||
            mutationLifecycle(for: .replayReject).isInFlight ||
            mutationLifecycle(for: .replayRetry).isInFlight ||
            mutationLifecycle(for: .replayComplete).isInFlight {
            return .loading("Applying replay mutation…")
        }

        if replayFeedQueryState.isRefreshing && replayEvents.isEmpty {
            return .loading("Loading live replay activity…")
        }

        if let replayProblem = panelProblem(for: .replay) {
            return lifecycleState(from: replayProblem)
        }
        if let realtimeProblem = panelProblem(for: .realtime) {
            return .degraded(realtimeProblem.summary, actions: realtimeProblem.actions)
        }

        if !replayFeedQueryState.hasLoadedOnce && sessionReadiness.isReadyForDaemonMutations {
            return .loading("Loading live replay activity…")
        }

        if replayEvents.isEmpty {
            return .empty(
                "No instructions have been captured yet.",
                actions: [
                    V2ProblemAction(actionID: .openGetStarted, label: "Open Get Started", isPrimary: true)
                ]
            )
        }

        if filteredEvents.isEmpty {
            return .empty(
                "No replay items match your current filters.",
                actions: [
                    V2ProblemAction(actionID: .clearReplayFilters, label: "Clear Filters", isPrimary: true)
                ]
            )
        }

        if let disabledReason = highImpactActionDisabledReason {
            return .degraded(
                disabledReason,
                actions: [
                    V2ProblemAction(actionID: .openGetStarted, label: "Open Get Started", isPrimary: true)
                ]
            )
        }
        return .ready()
    }

    private func resolveConnectorsModelsPanelState() -> V2PanelLifecycleState {
        if isConnectorInventoryRefreshInFlight || isModelInventoryRefreshInFlight {
            return .loading("Loading live connector/model inventory…")
        }

        if mutationLifecycle(for: .connectorToggle).isInFlight ||
            mutationLifecycle(for: .connectorCheck).isInFlight ||
            mutationLifecycle(for: .connectorSaveConfig).isInFlight ||
            mutationLifecycle(for: .connectorPermission).isInFlight ||
            mutationLifecycle(for: .modelToggle).isInFlight ||
            mutationLifecycle(for: .modelSetPrimary).isInFlight ||
            mutationLifecycle(for: .modelRouteSimulate).isInFlight ||
            mutationLifecycle(for: .modelRouteExplain).isInFlight {
            return .loading("Applying maintenance change…")
        }

        if let connectorsProblem = panelProblem(for: .connectors) {
            return lifecycleState(from: connectorsProblem)
        }
        if let modelsProblem = panelProblem(for: .models) {
            return lifecycleState(from: modelsProblem)
        }

        if sessionReadiness.isReadyForDaemonMutations &&
            (!hasLoadedConnectorInventory || !hasLoadedModelInventory) {
            return .loading("Loading live connector/model inventory…")
        }

        if connectors.isEmpty && models.isEmpty {
            return .empty(
                "No connectors or models are available yet.",
                actions: [
                    V2ProblemAction(actionID: .openGetStarted, label: "Open Get Started", isPrimary: true)
                ]
            )
        }

        if let disabledReason = highImpactActionDisabledReason {
            return .degraded(
                disabledReason,
                actions: [
                    V2ProblemAction(actionID: .openGetStarted, label: "Open Get Started", isPrimary: true)
                ]
            )
        }

        return .ready()
    }

    private func lifecycleState(from problem: V2PanelProblemState) -> V2PanelLifecycleState {
        switch problem.kind {
        case .decoding, .validation, .unknown:
            return .error(problem.summary, actions: problem.actions)
        case .missingAuth, .authScope, .rateLimited, .daemonUnavailable, .connectivity:
            return .degraded(problem.summary, actions: problem.actions)
        }
    }
}

public enum V2SetupChecklistStepID: String, Sendable, Hashable {
    case sessionConfiguration
    case lifecycleOperational
    case defaultRouteResolved
    case connectorHealthy
    case replayActivityAvailable
}

public struct SetupChecklistItem {
    public var id: V2SetupChecklistStepID
    public var title: String
    public var detail: String
    public var isDone: Bool
    public var actionTitle: String
    public var panelActionID: V2ProblemActionID
    public var action: () -> Void

    public init(
        id: V2SetupChecklistStepID,
        title: String,
        detail: String,
        isDone: Bool,
        actionTitle: String,
        panelActionID: V2ProblemActionID,
        action: @escaping () -> Void
    ) {
        self.id = id
        self.title = title
        self.detail = detail
        self.isDone = isDone
        self.actionTitle = actionTitle
        self.panelActionID = panelActionID
        self.action = action
    }
}

public struct ReplayMetrics {
    public var total: Int
    public var completed: Int
    public var needsApproval: Int
    public var running: Int
    public var failed: Int

    public var waiting: Int {
        needsApproval + running
    }

    public var atRisk: Int {
        failed
    }

    public var automatedSafely: Int {
        completed
    }

    public init(total: Int, completed: Int, needsApproval: Int, running: Int = 0, failed: Int) {
        self.total = total
        self.completed = completed
        self.needsApproval = needsApproval
        self.running = running
        self.failed = failed
    }
}
