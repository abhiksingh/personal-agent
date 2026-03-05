import Foundation

@MainActor
extension AppShellV2Store {
    public func refreshConnectorsInventoryIfNeeded(force: Bool = false) async {
        if isConnectorInventoryRefreshInFlight {
            return
        }
        if !force,
           hasLoadedConnectorInventory,
           selectedSection != .connectorsAndModels {
            return
        }
        await refreshConnectorsInventory(force: force)
    }

    public func refreshConnectorsInventory(force: Bool = false) async {
        guard let context = connectorMutationContext(redirectToSetup: false) else {
            hasLoadedConnectorInventory = false
            return
        }

        isConnectorInventoryRefreshInFlight = true
        defer { isConnectorInventoryRefreshInFlight = false }

        do {
            let status = try await daemonClient.connectors.connectorStatus(
                baseURL: context.baseURL,
                authToken: context.authToken,
                workspaceID: workspaceID
            )
            applyConnectorStatus(status)
            clearPanelProblem(for: .connectors)
            if force {
                setFeedback("Connector inventory refreshed.")
            }
        } catch {
            setPanelProblem(error, context: .connectors)
            if force {
                setFeedback(V2DaemonProblemMapper.map(error: error, context: .connectors).summary)
            }
        }
    }

    public func connectFirstAvailableConnector() {
        guard let connector = connectors.first(where: { !$0.enabled }) ?? connectors.first(where: { $0.status != .connected }) else {
            setFeedback("All connectors are already connected.")
            return
        }
        toggleConnector(connector.id)
    }

    public func toggleConnector(_ connectorID: ConnectorState.ID) {
        let lifecycle = mutationLifecycle(for: .connectorToggle)
        if lifecycle.isDisabled {
            setFeedback(lifecycle.message ?? "Connector actions are unavailable.")
            selectedSection = .getStarted
            return
        }
        guard !lifecycle.isInFlight else {
            return
        }
        guard let connector = connectors.first(where: { $0.id == connectorID }) else {
            return
        }
        if let disabledReason = connectorActionDisabledReason(for: connectorID, action: .toggle) {
            setConnectorActionStatus(disabledReason, for: connectorID)
            setFeedback(disabledReason)
            return
        }
        guard let context = connectorMutationContext() else {
            failMutation(.connectorToggle, message: "Connector actions are unavailable.")
            return
        }

        let targetEnabled = !connector.enabled
        startMutation(.connectorToggle, message: targetEnabled ? "Connecting \(connector.name)…" : "Disconnecting \(connector.name)…")
        setConnectorActionInFlight(.toggle, for: connectorID)

        Task { [weak self] in
            guard let self else {
                return
            }

            do {
                _ = try await self.daemonClient.connectors.connectorConfigUpsert(
                    baseURL: context.baseURL,
                    authToken: context.authToken,
                    workspaceID: self.workspaceID,
                    connectorID: connectorID,
                    configuration: ["enabled": .bool(targetEnabled)],
                    merge: true
                )
                self.setConnectorActionStatus(
                    targetEnabled ? "\(connector.name) connected." : "\(connector.name) disconnected.",
                    for: connectorID
                )
                self.completeMutation(
                    .connectorToggle,
                    message: targetEnabled ? "\(connector.name) connected." : "\(connector.name) disconnected."
                )
                self.clearPanelProblem(for: .connectors)
                await self.refreshConnectorsInventory(force: true)
            } catch {
                let summary = V2DaemonProblemMapper.map(error: error, context: .connectors).summary
                self.failMutation(.connectorToggle, message: summary)
                self.setPanelProblem(error, context: .connectors)
                self.setConnectorActionStatus(summary, for: connectorID)
                self.setFeedback(summary)
            }

            self.clearConnectorActionInFlight(for: connectorID)
        }
    }

    public func runConnectorCheck(_ connectorID: ConnectorState.ID) {
        let lifecycle = mutationLifecycle(for: .connectorCheck)
        if lifecycle.isDisabled {
            setFeedback(lifecycle.message ?? "Connector checks are unavailable.")
            selectedSection = .getStarted
            return
        }
        guard !lifecycle.isInFlight else {
            return
        }
        guard let connector = connectors.first(where: { $0.id == connectorID }) else {
            return
        }
        if let disabledReason = connectorActionDisabledReason(for: connectorID, action: .check) {
            setConnectorActionStatus(disabledReason, for: connectorID)
            setFeedback(disabledReason)
            return
        }

        startMutation(.connectorCheck, message: "Running health check for \(connector.name)…")
        setConnectorActionInFlight(.check, for: connectorID)

        guard let context = connectorMutationContext() else {
            clearConnectorActionInFlight(for: connectorID)
            failMutation(.connectorCheck, message: "Connector checks are unavailable.")
            return
        }

        Task { [weak self] in
            guard let self else {
                return
            }

            do {
                let response = try await self.daemonClient.connectors.connectorTestOperation(
                    baseURL: context.baseURL,
                    authToken: context.authToken,
                    workspaceID: self.workspaceID,
                    connectorID: connectorID,
                    operation: "health_check"
                )

                self.updateConnector(connectorID) { draft in
                    draft.lastCheckAt = self.parseConnectorDate(response.checkedAt)
                    draft.lastCheckSummary = response.summary
                    draft.lastCheckSucceeded = response.success
                }
                self.setConnectorActionStatus(response.summary, for: connectorID)
                self.completeMutation(.connectorCheck, message: response.summary)
                self.clearPanelProblem(for: .connectors)
                await self.refreshConnectorsInventory(force: true)
            } catch {
                let summary = V2DaemonProblemMapper.map(error: error, context: .connectors).summary
                self.failMutation(.connectorCheck, message: summary)
                self.setPanelProblem(error, context: .connectors)
                self.setConnectorActionStatus(summary, for: connectorID)
                self.setFeedback(summary)
            }

            self.clearConnectorActionInFlight(for: connectorID)
        }
    }

    public func saveConnectorConfiguration(_ connectorID: ConnectorState.ID) {
        let lifecycle = mutationLifecycle(for: .connectorSaveConfig)
        if lifecycle.isDisabled {
            setFeedback(lifecycle.message ?? "Connector configuration is unavailable.")
            selectedSection = .getStarted
            return
        }
        guard !lifecycle.isInFlight else {
            return
        }
        guard let connector = connectors.first(where: { $0.id == connectorID }) else {
            return
        }

        startMutation(.connectorSaveConfig, message: "Saving config for \(connector.name)…")
        setConnectorActionInFlight(.saveConfig, for: connectorID)

        guard let context = connectorMutationContext() else {
            clearConnectorActionInFlight(for: connectorID)
            failMutation(.connectorSaveConfig, message: "Connector configuration is unavailable.")
            return
        }

        let configuration = connector.configurationDraft.reduce(into: [String: V2DaemonJSONValue]()) { partial, pair in
            partial[pair.key] = parseConnectorConfigValue(pair.value)
        }

        Task { [weak self] in
            guard let self else {
                return
            }

            do {
                let response = try await self.daemonClient.connectors.connectorConfigUpsert(
                    baseURL: context.baseURL,
                    authToken: context.authToken,
                    workspaceID: self.workspaceID,
                    connectorID: connectorID,
                    configuration: configuration,
                    merge: true
                )

                let baseline = self.toConnectorConfigDraftValues(response.configuration)
                self.updateConnector(connectorID) { draft in
                    draft.configurationBaseline = baseline
                    draft.configurationDraft = baseline
                    draft.lastCheckAt = Date()
                    draft.lastCheckSummary = "Configuration saved."
                    draft.lastCheckSucceeded = true
                }

                self.setConnectorActionStatus("Configuration saved.", for: connectorID)
                self.completeMutation(.connectorSaveConfig, message: "Configuration saved for \(connector.name).")
                self.clearPanelProblem(for: .connectors)
                await self.refreshConnectorsInventory(force: true)
            } catch {
                let summary = V2DaemonProblemMapper.map(error: error, context: .connectors).summary
                self.failMutation(.connectorSaveConfig, message: summary)
                self.setPanelProblem(error, context: .connectors)
                self.setConnectorActionStatus(summary, for: connectorID)
                self.setFeedback(summary)
            }

            self.clearConnectorActionInFlight(for: connectorID)
        }
    }

    public func resetConnectorConfigurationDraft(_ connectorID: ConnectorState.ID) {
        updateConnector(connectorID) { draft in
            draft.configurationDraft = draft.configurationBaseline
        }
        setConnectorActionStatus("Draft reset.", for: connectorID)
    }

    public func connectorConfigurationDraftKeys(_ connectorID: ConnectorState.ID) -> [String] {
        guard let connector = connectors.first(where: { $0.id == connectorID }) else {
            return []
        }
        return connector.configurationDraft.keys.sorted()
    }

    public func connectorConfigurationDraftValue(connectorID: ConnectorState.ID, key: String) -> String {
        connectors.first(where: { $0.id == connectorID })?.configurationDraft[key] ?? ""
    }

    public func setConnectorConfigurationDraftValue(connectorID: ConnectorState.ID, key: String, value: String) {
        updateConnector(connectorID) { draft in
            let trimmedKey = key.trimmingCharacters(in: .whitespacesAndNewlines)
            guard !trimmedKey.isEmpty else {
                return
            }
            draft.configurationDraft[trimmedKey] = value
        }
    }

    public func connectorActionStatus(for connectorID: ConnectorState.ID) -> String? {
        connectorActionStatusByID[connectorID]
    }

    public func isConnectorActionInFlight(_ connectorID: ConnectorState.ID, action: V2ConnectorRowAction? = nil) -> Bool {
        guard let inFlight = connectorActionInFlightByID[connectorID] else {
            return false
        }
        if let action {
            return inFlight == action
        }
        return true
    }

    public func connectorActionDisabledReason(for connectorID: ConnectorState.ID, action: V2ConnectorRowAction) -> String? {
        guard let connector = connectors.first(where: { $0.id == connectorID }) else {
            return "Connector not found."
        }

        let readiness = connector.actionReadiness.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        if !["ready", "ok", "allowed"].contains(readiness),
           let blocker = connector.actionBlockers.first(where: { !$0.message.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty }) {
            switch action {
            case .toggle, .saveConfig, .check:
                return blocker.message
            case .requestPermission:
                break
            case .remediation:
                break
            }
        }

        if action == .requestPermission,
           let permission = connector.permissionState?.trimmingCharacters(in: .whitespacesAndNewlines).lowercased(),
           permission == "granted" {
            return "Permission already granted."
        }

        return nil
    }

    public func requestConnectorPermission(_ connectorID: ConnectorState.ID) {
        let lifecycle = mutationLifecycle(for: .connectorPermission)
        if lifecycle.isDisabled {
            setFeedback(lifecycle.message ?? "Permission request is unavailable.")
            selectedSection = .getStarted
            return
        }
        guard !lifecycle.isInFlight else {
            return
        }
        guard let connector = connectors.first(where: { $0.id == connectorID }) else {
            return
        }
        if let disabledReason = connectorActionDisabledReason(for: connectorID, action: .requestPermission) {
            setConnectorActionStatus(disabledReason, for: connectorID)
            setFeedback(disabledReason)
            return
        }

        startMutation(.connectorPermission, message: "Requesting permission for \(connector.name)…")
        setConnectorActionInFlight(.requestPermission, for: connectorID)

        guard let context = connectorMutationContext() else {
            clearConnectorActionInFlight(for: connectorID)
            failMutation(.connectorPermission, message: "Permission request is unavailable.")
            return
        }

        Task { [weak self] in
            guard let self else {
                return
            }

            do {
                let response = try await self.daemonClient.connectors.connectorPermissionRequest(
                    baseURL: context.baseURL,
                    authToken: context.authToken,
                    workspaceID: self.workspaceID,
                    connectorID: connectorID
                )
                let message = response.message?.trimmingCharacters(in: .whitespacesAndNewlines)
                self.setConnectorActionStatus(message?.isEmpty == false ? message! : "Permission request submitted.", for: connectorID)
                self.completeMutation(.connectorPermission, message: "Permission request submitted for \(connector.name).")
                self.clearPanelProblem(for: .connectors)
                await self.refreshConnectorsInventory(force: true)
            } catch {
                let summary = V2DaemonProblemMapper.map(error: error, context: .connectors).summary
                self.failMutation(.connectorPermission, message: summary)
                self.setPanelProblem(error, context: .connectors)
                self.setConnectorActionStatus(summary, for: connectorID)
                self.setFeedback(summary)
            }

            self.clearConnectorActionInFlight(for: connectorID)
        }
    }

    public func performConnectorRemediation(connectorID: ConnectorState.ID, actionID: String) {
        guard let connector = connectors.first(where: { $0.id == connectorID }) else {
            return
        }
        guard let remediation = connector.remediationActions.first(where: { $0.identifier == actionID }) else {
            return
        }

        let intent = remediation.intent.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        switch intent {
        case "request_permission":
            requestConnectorPermission(connectorID)
        case "refresh_status", "refresh", "retry", "retry_status":
            Task {
                await refreshConnectorsInventory(force: true)
            }
        case "navigate":
            routeConnectorRemediationDestination(remediation.destination)
            setConnectorActionStatus("Opened remediation destination.", for: connectorID)
        default:
            if let destination = remediation.destination,
               !destination.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                routeConnectorRemediationDestination(destination)
                setConnectorActionStatus("Opened remediation destination.", for: connectorID)
            } else {
                let fallback = remediation.reason?.trimmingCharacters(in: .whitespacesAndNewlines)
                let message = (fallback?.isEmpty == false ? fallback! : "Remediation action is unavailable in this build.")
                setConnectorActionStatus(message, for: connectorID)
                setFeedback(message)
            }
        }
    }

    public func refreshModelsInventoryIfNeeded(force: Bool = false) async {
        if isModelInventoryRefreshInFlight {
            return
        }
        if !force,
           hasLoadedModelInventory,
           selectedSection != .connectorsAndModels {
            return
        }
        await refreshModelsInventory(force: force)
    }

    public func refreshModelsInventory(force: Bool = false) async {
        guard let context = connectorMutationContext(redirectToSetup: false) else {
            hasLoadedModelInventory = false
            return
        }

        if isModelInventoryRefreshInFlight {
            return
        }

        isModelInventoryRefreshInFlight = true
        defer { isModelInventoryRefreshInFlight = false }

        async let modelListResult: Result<V2DaemonModelListResponse, Error> = capturedModel { [self] in
            try await self.daemonClient.models.modelList(
                baseURL: context.baseURL,
                authToken: context.authToken,
                workspaceID: self.workspaceID
            )
        }

        async let routeResult: Result<V2DaemonModelResolveResponse, Error> = capturedModel { [self] in
            try await self.daemonClient.models.modelResolve(
                baseURL: context.baseURL,
                authToken: context.authToken,
                workspaceID: self.workspaceID,
                taskClass: self.normalizedModelRouteTaskClass()
            )
        }

        let resolvedModelList = await modelListResult
        let resolvedRoute = await routeResult

        var firstError: Error?

        switch resolvedModelList {
        case .success(let list):
            applyModelCatalog(list.models)
        case .failure(let error):
            firstError = firstError ?? error
            hasLoadedModelInventory = false
        }

        switch resolvedRoute {
        case .success(let route):
            applyRouteResolution(route)
        case .failure(let error):
            firstError = firstError ?? error
            modelRouteResolution = nil
            modelRouteStatusMessage = V2DaemonProblemMapper.map(error: error, context: .models).summary
            var snapshot = getStartedReadinessSnapshot
            snapshot.modelRoute = nil
            snapshot.modelRouteError = modelRouteStatusMessage
            if snapshot.lastUpdatedAt != nil {
                snapshot.lastUpdatedAt = Date()
            }
            getStartedReadinessSnapshot = snapshot
        }

        if let firstError {
            setPanelProblem(firstError, context: .models)
            if force {
                setFeedback(V2DaemonProblemMapper.map(error: firstError, context: .models).summary)
            }
            return
        }

        clearPanelProblem(for: .models)
        if force {
            setFeedback("Model routing refreshed.")
        }
    }

    public func canSetActiveModel(_ modelID: ModelOption.ID) -> Bool {
        guard let model = models.first(where: { $0.id == modelID }) else {
            return false
        }
        return modelActionDisabledReason(for: modelID, action: .setPrimary) == nil
    }

    public func isModelActive(_ modelID: ModelOption.ID) -> Bool {
        activeModelID == modelID
    }

    public func modelActionStatus(for modelID: ModelOption.ID) -> String? {
        modelActionStatusByID[modelID]
    }

    public func isModelActionInFlight(_ modelID: ModelOption.ID, action: V2ModelRowAction? = nil) -> Bool {
        guard let inFlight = modelActionInFlightByID[modelID] else {
            return false
        }
        if let action {
            return inFlight == action
        }
        return true
    }

    public func disableModelReason(for modelID: ModelOption.ID) -> String? {
        guard let model = models.first(where: { $0.id == modelID }) else {
            return "Model not found."
        }

        if model.enabled && models.filter({ $0.enabled }).count == 1 {
            return "At least one model must stay enabled."
        }

        return nil
    }

    public func modelActionDisabledReason(for modelID: ModelOption.ID, action: V2ModelRowAction) -> String? {
        guard let model = models.first(where: { $0.id == modelID }) else {
            return "Model not found."
        }

        switch action {
        case .toggle:
            if model.enabled, let reason = disableModelReason(for: modelID) {
                return reason
            }
            if !model.enabled, !model.providerReady {
                return "Provider setup is incomplete for this model."
            }
        case .setPrimary:
            if !model.enabled {
                return "Enable this model before setting it as primary."
            }
            if !model.providerReady {
                return "Provider setup is incomplete for this model."
            }
            if activeModelID == modelID {
                return "This model is already primary."
            }
        case .simulateRoute, .explainRoute:
            break
        }

        return nil
    }

    public func setActiveModel(_ modelID: ModelOption.ID) {
        let lifecycle = mutationLifecycle(for: .modelSetPrimary)
        if lifecycle.isDisabled {
            setFeedback(lifecycle.message ?? "Model route actions are unavailable.")
            selectedSection = .getStarted
            return
        }
        guard !lifecycle.isInFlight else {
            return
        }
        guard let model = models.first(where: { $0.id == modelID }) else {
            return
        }
        if let disabledReason = modelActionDisabledReason(for: modelID, action: .setPrimary) {
            setModelActionStatus(disabledReason, for: modelID)
            setFeedback(disabledReason)
            return
        }
        guard let context = connectorMutationContext() else {
            failMutation(.modelSetPrimary, message: "Model route actions are unavailable.")
            return
        }

        startMutation(.modelSetPrimary, message: "Setting primary model route…")
        setModelActionInFlight(.setPrimary, for: modelID)

        Task { [weak self] in
            guard let self else {
                return
            }

            do {
                let taskClass = self.normalizedModelRouteTaskClass()
                _ = try await self.daemonClient.models.modelSelect(
                    baseURL: context.baseURL,
                    authToken: context.authToken,
                    workspaceID: self.workspaceID,
                    taskClass: taskClass,
                    provider: model.providerID,
                    modelKey: model.modelKey
                )
                self.modelRouteSimulation = nil
                self.modelRouteExplainability = nil
                self.modelRouteSimulationStatusMessage = "Primary route changed. Run simulation for updated decision details."
                self.modelRouteExplainStatusMessage = "Primary route changed. Run explainability for updated rationale."
                self.setModelActionStatus("Primary route set to \(model.routeLabel).", for: modelID)
                await self.refreshModelsInventory(force: false)
                self.completeMutation(.modelSetPrimary, message: "Primary route updated.")
                self.setFeedback("Primary route set to \(model.routeLabel).")
            } catch {
                let summary = V2DaemonProblemMapper.map(error: error, context: .models).summary
                self.failMutation(.modelSetPrimary, message: summary)
                self.setPanelProblem(error, context: .models)
                self.setModelActionStatus(summary, for: modelID)
                self.setFeedback(summary)
            }

            self.clearModelActionInFlight(for: modelID)
        }
    }

    public func toggleModelEnabled(_ modelID: ModelOption.ID) {
        let lifecycle = mutationLifecycle(for: .modelToggle)
        if lifecycle.isDisabled {
            setFeedback(lifecycle.message ?? "Model toggle is unavailable.")
            selectedSection = .getStarted
            return
        }
        guard !lifecycle.isInFlight else {
            return
        }
        guard let model = models.first(where: { $0.id == modelID }) else {
            return
        }
        if let disabledReason = modelActionDisabledReason(for: modelID, action: .toggle) {
            setModelActionStatus(disabledReason, for: modelID)
            setFeedback(disabledReason)
            return
        }
        guard let context = connectorMutationContext() else {
            failMutation(.modelToggle, message: "Model toggle is unavailable.")
            return
        }

        let targetEnabled = !model.enabled
        startMutation(.modelToggle, message: targetEnabled ? "Enabling \(model.routeLabel)…" : "Disabling \(model.routeLabel)…")
        setModelActionInFlight(.toggle, for: modelID)

        Task { [weak self] in
            guard let self else {
                return
            }

            do {
                if targetEnabled {
                    _ = try await self.daemonClient.models.modelEnable(
                        baseURL: context.baseURL,
                        authToken: context.authToken,
                        workspaceID: self.workspaceID,
                        provider: model.providerID,
                        modelKey: model.modelKey
                    )
                } else {
                    _ = try await self.daemonClient.models.modelDisable(
                        baseURL: context.baseURL,
                        authToken: context.authToken,
                        workspaceID: self.workspaceID,
                        provider: model.providerID,
                        modelKey: model.modelKey
                    )
                }

                let statusMessage = targetEnabled
                    ? "Enabled \(model.routeLabel)."
                    : "Disabled \(model.routeLabel)."
                self.setModelActionStatus(statusMessage, for: modelID)
                await self.refreshModelsInventory(force: false)
                self.completeMutation(.modelToggle, message: statusMessage)
                self.setFeedback(statusMessage)
            } catch {
                let summary = V2DaemonProblemMapper.map(error: error, context: .models).summary
                self.failMutation(.modelToggle, message: summary)
                self.setPanelProblem(error, context: .models)
                self.setModelActionStatus(summary, for: modelID)
                self.setFeedback(summary)
            }

            self.clearModelActionInFlight(for: modelID)
        }
    }

    public func simulateModelRoute() {
        let lifecycle = mutationLifecycle(for: .modelRouteSimulate)
        if lifecycle.isDisabled {
            setFeedback(lifecycle.message ?? "Route simulation is unavailable.")
            selectedSection = .getStarted
            return
        }
        guard !lifecycle.isInFlight else {
            return
        }
        guard let context = connectorMutationContext() else {
            failMutation(.modelRouteSimulate, message: "Route simulation is unavailable.")
            return
        }

        let taskClass = normalizedModelRouteTaskClass()
        startMutation(.modelRouteSimulate, message: "Running route simulation…")
        modelRouteSimulationStatusMessage = "Running route simulation…"

        Task { [weak self] in
            guard let self else {
                return
            }

            do {
                let response = try await self.daemonClient.models.modelRouteSimulate(
                    baseURL: context.baseURL,
                    authToken: context.authToken,
                    workspaceID: self.workspaceID,
                    taskClass: taskClass,
                    principalActorID: self.normalizedPrincipalActorID()
                )
                self.modelRouteSimulation = response
                self.modelRouteSimulationStatusMessage = self.routeSimulationSummary(response)
                self.completeMutation(.modelRouteSimulate, message: "Route simulation loaded.")
                self.clearPanelProblem(for: .models)
                self.setFeedback(self.modelRouteSimulationStatusMessage ?? "Route simulation loaded.")
            } catch {
                let summary = V2DaemonProblemMapper.map(error: error, context: .models).summary
                self.modelRouteSimulation = nil
                self.modelRouteSimulationStatusMessage = summary
                self.failMutation(.modelRouteSimulate, message: summary)
                self.setPanelProblem(error, context: .models)
                self.setFeedback(summary)
            }
        }
    }

    public func explainModelRoute() {
        let lifecycle = mutationLifecycle(for: .modelRouteExplain)
        if lifecycle.isDisabled {
            setFeedback(lifecycle.message ?? "Route explainability is unavailable.")
            selectedSection = .getStarted
            return
        }
        guard !lifecycle.isInFlight else {
            return
        }
        guard let context = connectorMutationContext() else {
            failMutation(.modelRouteExplain, message: "Route explainability is unavailable.")
            return
        }

        let taskClass = normalizedModelRouteTaskClass()
        startMutation(.modelRouteExplain, message: "Loading route explainability…")
        modelRouteExplainStatusMessage = "Loading route explainability…"

        Task { [weak self] in
            guard let self else {
                return
            }

            do {
                let response = try await self.daemonClient.models.modelRouteExplain(
                    baseURL: context.baseURL,
                    authToken: context.authToken,
                    workspaceID: self.workspaceID,
                    taskClass: taskClass,
                    principalActorID: self.normalizedPrincipalActorID()
                )
                self.modelRouteExplainability = response
                self.modelRouteExplainStatusMessage = self.routeExplainabilitySummary(response)
                self.completeMutation(.modelRouteExplain, message: "Route explainability loaded.")
                self.clearPanelProblem(for: .models)
                self.setFeedback(self.modelRouteExplainStatusMessage ?? "Route explainability loaded.")
            } catch {
                let summary = V2DaemonProblemMapper.map(error: error, context: .models).summary
                self.modelRouteExplainability = nil
                self.modelRouteExplainStatusMessage = summary
                self.failMutation(.modelRouteExplain, message: summary)
                self.setPanelProblem(error, context: .models)
                self.setFeedback(summary)
            }
        }
    }

    private func connectorMutationContext(redirectToSetup: Bool = true) -> (baseURL: URL, authToken: String)? {
        guard sessionReadiness.isReadyForDaemonMutations else {
            if redirectToSetup {
                selectedSection = .getStarted
            }
            return nil
        }

        let trimmedBaseURL = daemonBaseURL.trimmingCharacters(in: .whitespacesAndNewlines)
        guard let baseURL = URL(string: trimmedBaseURL),
              !trimmedBaseURL.isEmpty,
              let components = URLComponents(url: baseURL, resolvingAgainstBaseURL: false),
              let scheme = components.scheme?.lowercased(),
              ["http", "https", "ws", "wss"].contains(scheme),
              components.host != nil else {
            if redirectToSetup {
                selectedSection = .getStarted
            }
            return nil
        }

        guard let authToken = sessionConfigStore.resolvedAccessToken()?.trimmingCharacters(in: .whitespacesAndNewlines),
              !authToken.isEmpty else {
            if redirectToSetup {
                selectedSection = .getStarted
            }
            return nil
        }

        return (baseURL, authToken)
    }

    private func applyConnectorStatus(_ response: V2DaemonConnectorStatusResponse) {
        let existingByID = Dictionary(uniqueKeysWithValues: connectors.map { ($0.id, $0) })
        var projected: [ConnectorState] = response.connectors.map { card in
            projectConnectorState(card, existing: existingByID[normalizedConnectorIdentifier(card.connectorID, fallback: card.pluginID)])
        }
        projected.sort { $0.name.localizedCaseInsensitiveCompare($1.name) == .orderedAscending }

        let visibleIDs = Set(projected.map(\.id))
        connectorActionStatusByID = connectorActionStatusByID.filter { visibleIDs.contains($0.key) }
        connectorActionInFlightByID = connectorActionInFlightByID.filter { visibleIDs.contains($0.key) }

        connectors = projected
        hasLoadedConnectorInventory = true

        getStartedReadinessSnapshot.connectorCards = response.connectors
        getStartedReadinessSnapshot.connectorError = nil
        if getStartedReadinessSnapshot.lastUpdatedAt != nil {
            getStartedReadinessSnapshot.lastUpdatedAt = Date()
        }
    }

    private func applyModelCatalog(_ records: [V2DaemonModelCatalogRecord]) {
        let existingByID = Dictionary(uniqueKeysWithValues: models.map { ($0.id, $0) })
        var projected = records.map { record in
            let modelID = modelIdentifier(providerID: record.provider, modelKey: record.modelKey)
            return projectModelOption(record, existing: existingByID[modelID])
        }
        projected.sort { lhs, rhs in
            let providerOrder = lhs.providerName.localizedCaseInsensitiveCompare(rhs.providerName)
            if providerOrder == .orderedSame {
                return lhs.modelKey.localizedCaseInsensitiveCompare(rhs.modelKey) == .orderedAscending
            }
            return providerOrder == .orderedAscending
        }

        let visibleIDs = Set(projected.map(\.id))
        modelActionStatusByID = modelActionStatusByID.filter { visibleIDs.contains($0.key) }
        modelActionInFlightByID = modelActionInFlightByID.filter { visibleIDs.contains($0.key) }

        models = projected
        hasLoadedModelInventory = true

        reconcileActiveModel(route: modelRouteResolution, projectedModels: projected)
    }

    private func applyRouteResolution(_ route: V2DaemonModelResolveResponse) {
        modelRouteResolution = route
        modelRouteStatusMessage = routeResolutionSummary(route)

        var snapshot = getStartedReadinessSnapshot
        snapshot.modelRoute = route
        snapshot.modelRouteError = nil
        if snapshot.lastUpdatedAt != nil {
            snapshot.lastUpdatedAt = Date()
        }
        getStartedReadinessSnapshot = snapshot

        reconcileActiveModel(route: route, projectedModels: models)
    }

    private func projectModelOption(_ record: V2DaemonModelCatalogRecord, existing: ModelOption?) -> ModelOption {
        let providerID = normalizedProviderIdentifier(record.provider)
        let modelKey = record.modelKey.trimmingCharacters(in: .whitespacesAndNewlines)
        let endpoint = record.providerEndpoint?.trimmingCharacters(in: .whitespacesAndNewlines)
        return ModelOption(
            id: modelIdentifier(providerID: providerID, modelKey: modelKey),
            providerID: providerID,
            providerName: existing?.providerName ?? providerDisplayName(providerID),
            modelKey: modelKey,
            enabled: record.enabled,
            providerReady: record.providerReady,
            providerEndpoint: endpoint?.isEmpty == true ? nil : endpoint
        )
    }

    private func reconcileActiveModel(route: V2DaemonModelResolveResponse?, projectedModels: [ModelOption]) {
        if let route {
            let resolvedModelID = modelIdentifier(providerID: route.provider, modelKey: route.modelKey)
            if projectedModels.contains(where: { $0.id == resolvedModelID }) {
                activeModelID = resolvedModelID
                return
            }
        }

        if let activeModelID,
           projectedModels.contains(where: { $0.id == activeModelID && $0.enabled }) {
            return
        }
        activeModelID = projectedModels.first(where: { $0.enabled })?.id
    }

    private func routeResolutionSummary(_ route: V2DaemonModelResolveResponse) -> String {
        let provider = providerDisplayName(route.provider)
        let modelKey = route.modelKey.trimmingCharacters(in: .whitespacesAndNewlines)
        let source = route.source.trimmingCharacters(in: .whitespacesAndNewlines)
        if source.isEmpty {
            return "Route resolves to \(provider) / \(modelKey)."
        }
        return "Route resolves to \(provider) / \(modelKey) from \(source)."
    }

    private func routeSimulationSummary(_ response: V2DaemonModelRouteSimulationResponse) -> String {
        let provider = providerDisplayName(response.selectedProvider)
        let modelKey = response.selectedModelKey.trimmingCharacters(in: .whitespacesAndNewlines)
        return "Simulated \(response.taskClass) route: \(provider) / \(modelKey)."
    }

    private func routeExplainabilitySummary(_ response: V2DaemonModelRouteExplainResponse) -> String {
        let summary = response.summary.trimmingCharacters(in: .whitespacesAndNewlines)
        if !summary.isEmpty {
            return summary
        }
        let provider = providerDisplayName(response.selectedProvider)
        let modelKey = response.selectedModelKey.trimmingCharacters(in: .whitespacesAndNewlines)
        return "Loaded explainability for \(response.taskClass): \(provider) / \(modelKey)."
    }

    private func setModelActionStatus(_ message: String, for modelID: ModelOption.ID) {
        modelActionStatusByID[modelID] = message
    }

    private func setModelActionInFlight(_ action: V2ModelRowAction, for modelID: ModelOption.ID) {
        modelActionInFlightByID[modelID] = action
    }

    private func clearModelActionInFlight(for modelID: ModelOption.ID) {
        modelActionInFlightByID.removeValue(forKey: modelID)
    }

    private func modelIdentifier(providerID: String, modelKey: String) -> ModelOption.ID {
        let normalizedProvider = normalizedProviderIdentifier(providerID)
        let normalizedModel = modelKey.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        return "\(normalizedProvider)::\(normalizedModel)"
    }

    private func normalizedProviderIdentifier(_ providerID: String) -> String {
        providerID.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
    }

    private func providerDisplayName(_ providerID: String) -> String {
        switch normalizedProviderIdentifier(providerID) {
        case "openai":
            return "OpenAI"
        case "anthropic":
            return "Anthropic"
        case "built_in", "builtin", "personalagent":
            return "Built-In"
        default:
            let raw = providerID.trimmingCharacters(in: .whitespacesAndNewlines)
            if raw.isEmpty {
                return "Unknown"
            }
            return raw
                .replacingOccurrences(of: "_", with: " ")
                .replacingOccurrences(of: "-", with: " ")
                .split(separator: " ")
                .map { segment in
                    segment.prefix(1).uppercased() + segment.dropFirst().lowercased()
                }
                .joined(separator: " ")
        }
    }

    private func normalizedModelRouteTaskClass() -> String {
        let normalized = modelRouteTaskClass.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        if normalized.isEmpty {
            modelRouteTaskClass = "chat"
            return "chat"
        }
        modelRouteTaskClass = normalized
        return normalized
    }

    private func normalizedPrincipalActorID() -> String? {
        let normalized = principalActorID.trimmingCharacters(in: .whitespacesAndNewlines)
        return normalized.isEmpty ? nil : normalized
    }

    private func capturedModel<T>(_ operation: @escaping () async throws -> T) async -> Result<T, Error> {
        do {
            return .success(try await operation())
        } catch {
            return .failure(error)
        }
    }

    private func projectConnectorState(_ card: V2DaemonConnectorStatusCard, existing: ConnectorState?) -> ConnectorState {
        let identifier = normalizedConnectorIdentifier(card.connectorID, fallback: card.pluginID)
        let baseline = toConnectorConfigDraftValues(connectorConfigurationPayload(from: card))
        let draft: [String: String]
        if let existing, existing.hasConfigDraftChanges {
            draft = existing.configurationDraft
        } else {
            draft = baseline
        }

        return ConnectorState(
            id: identifier,
            pluginID: card.pluginID,
            name: card.displayName,
            status: connectorStatus(for: card),
            summary: connectorSummary(for: card),
            enabled: card.enabled,
            configured: card.configured,
            actionReadiness: card.actionReadiness,
            actionBlockers: card.actionBlockers,
            remediationActions: card.remediationActions,
            permissionState: card.configuration?.permissionState,
            configurationBaseline: baseline,
            configurationDraft: draft,
            lastCheckAt: existing?.lastCheckAt,
            lastCheckSummary: existing?.lastCheckSummary,
            lastCheckSucceeded: existing?.lastCheckSucceeded
        )
    }

    private func connectorSummary(for card: V2DaemonConnectorStatusCard) -> String {
        if let summary = card.summary?.trimmingCharacters(in: .whitespacesAndNewlines), !summary.isEmpty {
            return summary
        }
        if !card.enabled {
            return "Connector is disabled."
        }
        if !card.configured {
            return "Connector is not configured yet."
        }
        return "Connector status is \(card.status)."
    }

    private func connectorStatus(for card: V2DaemonConnectorStatusCard) -> ConnectorStatus {
        let normalizedStatus = card.status.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        let readiness = card.actionReadiness.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()

        let healthyStates = ["healthy", "ready", "connected", "ok", "running", "active"]
        if card.enabled && card.configured && healthyStates.contains(normalizedStatus) {
            return .connected
        }
        if !card.enabled || normalizedStatus == "disabled" || normalizedStatus == "not_connected" {
            return .notConnected
        }
        if readiness != "ready" || !card.actionBlockers.isEmpty {
            return .needsAttention
        }
        return .needsAttention
    }

    private func connectorConfigurationPayload(from card: V2DaemonConnectorStatusCard) -> [String: V2DaemonJSONValue] {
        var payload: [String: V2DaemonJSONValue] = ["enabled": .bool(card.enabled)]
        if let mode = card.configuration?.mode?.trimmingCharacters(in: .whitespacesAndNewlines), !mode.isEmpty {
            payload["mode"] = .string(mode)
        }
        if let transport = card.configuration?.transport?.trimmingCharacters(in: .whitespacesAndNewlines), !transport.isEmpty {
            payload["transport"] = .string(transport)
        }
        if let additional = card.configuration?.additional {
            for (key, value) in additional {
                payload[key] = value
            }
        }
        return payload
    }

    private func toConnectorConfigDraftValues(_ values: [String: V2DaemonJSONValue]) -> [String: String] {
        var draft: [String: String] = [:]
        for (key, value) in values {
            draft[key] = connectorConfigDraftValue(from: value)
        }
        return draft
    }

    private func connectorConfigDraftValue(from value: V2DaemonJSONValue) -> String {
        switch value {
        case .object, .array:
            guard let data = try? JSONEncoder().encode(value),
                  let encoded = String(data: data, encoding: .utf8) else {
                return value.displayText
            }
            return encoded
        default:
            return value.displayText
        }
    }

    private func parseConnectorConfigValue(_ raw: String) -> V2DaemonJSONValue {
        let trimmed = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmed.isEmpty {
            return .null
        }

        let lowered = trimmed.lowercased()
        if lowered == "true" {
            return .bool(true)
        }
        if lowered == "false" {
            return .bool(false)
        }
        if lowered == "null" {
            return .null
        }
        if let integer = Int(trimmed) {
            return .number(Double(integer))
        }
        if let floating = Double(trimmed) {
            return .number(floating)
        }

        if ((trimmed.hasPrefix("{") && trimmed.hasSuffix("}")) || (trimmed.hasPrefix("[") && trimmed.hasSuffix("]"))),
           let jsonData = trimmed.data(using: .utf8),
           let parsed = try? JSONDecoder().decode(V2DaemonJSONValue.self, from: jsonData) {
            return parsed
        }

        return .string(trimmed)
    }

    private func updateConnector(_ connectorID: ConnectorState.ID, mutate: (inout ConnectorState) -> Void) {
        guard let index = connectors.firstIndex(where: { $0.id == connectorID }) else {
            return
        }

        var updated = connectors
        mutate(&updated[index])
        connectors = updated
    }

    private func setConnectorActionStatus(_ message: String, for connectorID: ConnectorState.ID) {
        connectorActionStatusByID[connectorID] = message
    }

    private func setConnectorActionInFlight(_ action: V2ConnectorRowAction, for connectorID: ConnectorState.ID) {
        connectorActionInFlightByID[connectorID] = action
    }

    private func clearConnectorActionInFlight(for connectorID: ConnectorState.ID) {
        connectorActionInFlightByID.removeValue(forKey: connectorID)
    }

    private func normalizedConnectorIdentifier(_ connectorID: String, fallback: String) -> String {
        let primary = connectorID.trimmingCharacters(in: .whitespacesAndNewlines)
        if !primary.isEmpty {
            return primary
        }
        let fallbackTrimmed = fallback.trimmingCharacters(in: .whitespacesAndNewlines)
        return fallbackTrimmed.isEmpty ? UUID().uuidString.lowercased() : fallbackTrimmed
    }

    private func parseConnectorDate(_ rawValue: String) -> Date {
        let trimmed = rawValue.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            return Date()
        }

        let withFractional = ISO8601DateFormatter()
        withFractional.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        if let parsed = withFractional.date(from: trimmed) {
            return parsed
        }

        let withoutFractional = ISO8601DateFormatter()
        withoutFractional.formatOptions = [.withInternetDateTime]
        return withoutFractional.date(from: trimmed) ?? Date()
    }

    private func routeConnectorRemediationDestination(_ destination: String?) {
        let normalized = destination?.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() ?? ""
        guard !normalized.isEmpty else {
            selectedSection = .connectorsAndModels
            return
        }

        if normalized.contains("setup") || normalized.contains("token") || normalized.contains("configuration") {
            selectedSection = .getStarted
            return
        }

        if normalized.contains("replay") || normalized.contains("approval") {
            selectedSection = .replayAndAsk
            return
        }

        selectedSection = .connectorsAndModels
    }
}
