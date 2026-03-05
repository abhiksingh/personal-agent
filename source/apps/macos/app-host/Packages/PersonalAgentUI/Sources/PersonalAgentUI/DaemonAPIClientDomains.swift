import Foundation

extension DaemonAPIClient {
    var lifecycle: DaemonLifecycleAPI { DaemonLifecycleAPI(client: self) }
    var models: DaemonModelsAPI { DaemonModelsAPI(client: self) }
    var inspect: DaemonInspectAPI { DaemonInspectAPI(client: self) }
    var communications: DaemonCommunicationsAPI { DaemonCommunicationsAPI(client: self) }
    var channels: DaemonChannelsAPI { DaemonChannelsAPI(client: self) }
    var connectors: DaemonConnectorsAPI { DaemonConnectorsAPI(client: self) }
    var identity: DaemonIdentityAPI { DaemonIdentityAPI(client: self) }
    var approvals: DaemonApprovalsAPI { DaemonApprovalsAPI(client: self) }
    var tasks: DaemonTasksAPI { DaemonTasksAPI(client: self) }
    var automation: DaemonAutomationAPI { DaemonAutomationAPI(client: self) }
    var context: DaemonContextAPI { DaemonContextAPI(client: self) }
    var chat: DaemonChatAPI { DaemonChatAPI(client: self) }
}

struct DaemonLifecycleAPI {
    private let client: DaemonAPIClient

    fileprivate init(client: DaemonAPIClient) {
        self.client = client
    }

    func daemonLifecycleStatus(baseURL: URL, authToken: String) async throws -> DaemonLifecycleStatusResponse {
        try await client.daemonLifecycleStatus(baseURL: baseURL, authToken: authToken)
    }

    func daemonLifecycleControl(
        baseURL: URL,
        authToken: String,
        action: String,
        reason: String
    ) async throws -> DaemonLifecycleControlResponse {
        try await client.daemonLifecycleControl(
            baseURL: baseURL,
            authToken: authToken,
            action: action,
            reason: reason
        )
    }

    func daemonPluginLifecycleHistory(
        baseURL: URL,
        authToken: String,
        workspaceID: String? = nil,
        pluginID: String? = nil,
        kind: String? = nil,
        state: String? = nil,
        eventType: String? = nil,
        cursorCreatedAt: String? = nil,
        cursorID: String? = nil,
        limit: Int? = nil
    ) async throws -> DaemonPluginLifecycleHistoryResponse {
        try await client.daemonPluginLifecycleHistory(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            pluginID: pluginID,
            kind: kind,
            state: state,
            eventType: eventType,
            cursorCreatedAt: cursorCreatedAt,
            cursorID: cursorID,
            limit: limit
        )
    }
}

struct DaemonModelsAPI {
    private let client: DaemonAPIClient

    fileprivate init(client: DaemonAPIClient) {
        self.client = client
    }

    func secretReferenceUpsert(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        name: String,
        backend: String,
        service: String,
        account: String
    ) async throws -> DaemonSecretReferenceResponse {
        try await client.secretReferenceUpsert(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            name: name,
            backend: backend,
            service: service,
            account: account
        )
    }

    func providerList(
        baseURL: URL,
        authToken: String,
        workspaceID: String
    ) async throws -> DaemonProviderListResponse {
        try await client.providerList(baseURL: baseURL, authToken: authToken, workspaceID: workspaceID)
    }

    func providerSet(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        provider: String,
        endpoint: String? = nil,
        apiKeySecretName: String? = nil,
        clearAPIKey: Bool = false
    ) async throws -> DaemonProviderConfigRecord {
        try await client.providerSet(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            provider: provider,
            endpoint: endpoint,
            apiKeySecretName: apiKeySecretName,
            clearAPIKey: clearAPIKey
        )
    }

    func providerCheck(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        provider: String? = nil
    ) async throws -> DaemonProviderCheckResponse {
        try await client.providerCheck(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            provider: provider
        )
    }

    func modelResolve(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        taskClass: String = "chat"
    ) async throws -> DaemonModelResolveResponse {
        try await client.modelResolve(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            taskClass: taskClass
        )
    }

    func modelRouteSimulate(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        taskClass: String = "chat",
        principalActorID: String? = nil
    ) async throws -> DaemonModelRouteSimulationResponse {
        try await client.modelRouteSimulate(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            taskClass: taskClass,
            principalActorID: principalActorID
        )
    }

    func modelRouteExplain(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        taskClass: String = "chat",
        principalActorID: String? = nil
    ) async throws -> DaemonModelRouteExplainResponse {
        try await client.modelRouteExplain(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            taskClass: taskClass,
            principalActorID: principalActorID
        )
    }

    func modelList(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        provider: String? = nil
    ) async throws -> DaemonModelListResponse {
        try await client.modelList(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            provider: provider
        )
    }

    func modelDiscover(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        provider: String? = nil
    ) async throws -> DaemonModelDiscoverResponse {
        try await client.modelDiscover(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            provider: provider
        )
    }

    func modelAdd(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        provider: String,
        modelKey: String,
        enabled: Bool = false
    ) async throws -> DaemonModelCatalogEntryRecord {
        try await client.modelAdd(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            provider: provider,
            modelKey: modelKey,
            enabled: enabled
        )
    }

    func modelRemove(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        provider: String,
        modelKey: String
    ) async throws -> DaemonModelCatalogRemoveResponse {
        try await client.modelRemove(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            provider: provider,
            modelKey: modelKey
        )
    }

    func modelEnable(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        provider: String,
        modelKey: String
    ) async throws -> DaemonModelCatalogEntryRecord {
        try await client.modelEnable(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            provider: provider,
            modelKey: modelKey
        )
    }

    func modelDisable(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        provider: String,
        modelKey: String
    ) async throws -> DaemonModelCatalogEntryRecord {
        try await client.modelDisable(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            provider: provider,
            modelKey: modelKey
        )
    }

    func modelSelect(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        taskClass: String = "chat",
        provider: String,
        modelKey: String
    ) async throws -> DaemonModelRoutingPolicyRecord {
        try await client.modelSelect(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            taskClass: taskClass,
            provider: provider,
            modelKey: modelKey
        )
    }

    func modelPolicy(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        taskClass: String? = nil
    ) async throws -> DaemonModelPolicyResponse {
        try await client.modelPolicy(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            taskClass: taskClass
        )
    }
}

struct DaemonInspectAPI {
    private let client: DaemonAPIClient

    fileprivate init(client: DaemonAPIClient) {
        self.client = client
    }

    func inspectLogsQuery(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        runID: String? = nil,
        eventType: String? = nil,
        beforeCreatedAt: String? = nil,
        beforeID: String? = nil,
        limit: Int = 80
    ) async throws -> DaemonInspectLogQueryResponse {
        try await client.inspectLogsQuery(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            runID: runID,
            eventType: eventType,
            beforeCreatedAt: beforeCreatedAt,
            beforeID: beforeID,
            limit: limit
        )
    }

    func inspectLogsStream(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        runID: String? = nil,
        cursorCreatedAt: String?,
        cursorID: String?,
        limit: Int = 80,
        timeoutMS: Int64 = 1500,
        pollIntervalMS: Int64 = 200
    ) async throws -> DaemonInspectLogStreamResponse {
        try await client.inspectLogsStream(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            runID: runID,
            cursorCreatedAt: cursorCreatedAt,
            cursorID: cursorID,
            limit: limit,
            timeoutMS: timeoutMS,
            pollIntervalMS: pollIntervalMS
        )
    }

    func inspectRun(
        baseURL: URL,
        authToken: String,
        runID: String
    ) async throws -> DaemonInspectRunResponse {
        try await client.inspectRun(baseURL: baseURL, authToken: authToken, runID: runID)
    }
}

struct DaemonCommunicationsAPI {
    private let client: DaemonAPIClient

    fileprivate init(client: DaemonAPIClient) {
        self.client = client
    }

    func commThreadList(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        channel: String? = nil,
        connectorID: String? = nil,
        query: String? = nil,
        cursor: String? = nil,
        limit: Int = 80
    ) async throws -> DaemonCommThreadListResponse {
        try await client.commThreadList(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            channel: channel,
            connectorID: connectorID,
            query: query,
            cursor: cursor,
            limit: limit
        )
    }

    func commEventTimeline(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        threadID: String? = nil,
        channel: String? = nil,
        connectorID: String? = nil,
        eventType: String? = nil,
        direction: String? = nil,
        query: String? = nil,
        cursor: String? = nil,
        limit: Int = 120
    ) async throws -> DaemonCommEventTimelineResponse {
        try await client.commEventTimeline(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            threadID: threadID,
            channel: channel,
            connectorID: connectorID,
            eventType: eventType,
            direction: direction,
            query: query,
            cursor: cursor,
            limit: limit
        )
    }

    func commCallSessionList(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        threadID: String? = nil,
        provider: String? = nil,
        connectorID: String? = nil,
        direction: String? = nil,
        status: String? = nil,
        providerCallID: String? = nil,
        query: String? = nil,
        cursor: String? = nil,
        limit: Int = 80
    ) async throws -> DaemonCommCallSessionListResponse {
        try await client.commCallSessionList(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            threadID: threadID,
            provider: provider,
            connectorID: connectorID,
            direction: direction,
            status: status,
            providerCallID: providerCallID,
            query: query,
            cursor: cursor,
            limit: limit
        )
    }

    func commAttempts(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        operationID: String? = nil,
        threadID: String? = nil,
        taskID: String? = nil,
        runID: String? = nil,
        stepID: String? = nil,
        channel: String? = nil,
        status: String? = nil,
        cursor: String? = nil,
        limit: Int = 120
    ) async throws -> DaemonCommAttemptsResponse {
        try await client.commAttempts(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            operationID: operationID,
            threadID: threadID,
            taskID: taskID,
            runID: runID,
            stepID: stepID,
            channel: channel,
            status: status,
            cursor: cursor,
            limit: limit
        )
    }

    func commSend(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        operationID: String,
        sourceChannel: String,
        threadID: String? = nil,
        connectorID: String? = nil,
        destination: String? = nil,
        message: String,
        stepID: String? = nil,
        eventID: String? = nil,
        iMessageFailures: Int? = nil,
        smsFailures: Int? = nil
    ) async throws -> DaemonCommSendResponse {
        try await client.commSend(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            operationID: operationID,
            sourceChannel: sourceChannel,
            threadID: threadID,
            connectorID: connectorID,
            destination: destination,
            message: message,
            stepID: stepID,
            eventID: eventID,
            iMessageFailures: iMessageFailures,
            smsFailures: smsFailures
        )
    }

    func commWebhookReceiptList(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        provider: String? = nil,
        providerEventID: String? = nil,
        providerEventQuery: String? = nil,
        eventID: String? = nil,
        cursorCreatedAt: String? = nil,
        cursorID: String? = nil,
        limit: Int = 80
    ) async throws -> DaemonCommWebhookReceiptListResponse {
        try await client.commWebhookReceiptList(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            provider: provider,
            providerEventID: providerEventID,
            providerEventQuery: providerEventQuery,
            eventID: eventID,
            cursorCreatedAt: cursorCreatedAt,
            cursorID: cursorID,
            limit: limit
        )
    }

    func commIngestReceiptList(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        source: String? = nil,
        sourceScope: String? = nil,
        sourceEventID: String? = nil,
        sourceEventQuery: String? = nil,
        trustState: String? = nil,
        eventID: String? = nil,
        cursorCreatedAt: String? = nil,
        cursorID: String? = nil,
        limit: Int = 80
    ) async throws -> DaemonCommIngestReceiptListResponse {
        try await client.commIngestReceiptList(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            source: source,
            sourceScope: sourceScope,
            sourceEventID: sourceEventID,
            sourceEventQuery: sourceEventQuery,
            trustState: trustState,
            eventID: eventID,
            cursorCreatedAt: cursorCreatedAt,
            cursorID: cursorID,
            limit: limit
        )
    }

    func commPolicySet(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        policyID: String? = nil,
        sourceChannel: String,
        endpointPattern: String? = nil,
        primaryChannel: String,
        retryCount: Int,
        fallbackChannels: [String],
        isDefault: Bool
    ) async throws -> DaemonCommPolicyRecord {
        try await client.commPolicySet(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            policyID: policyID,
            sourceChannel: sourceChannel,
            endpointPattern: endpointPattern,
            primaryChannel: primaryChannel,
            retryCount: retryCount,
            fallbackChannels: fallbackChannels,
            isDefault: isDefault
        )
    }

    func commPolicyList(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        sourceChannel: String? = nil
    ) async throws -> DaemonCommPolicyListResponse {
        try await client.commPolicyList(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            sourceChannel: sourceChannel
        )
    }
}

struct DaemonChannelsAPI {
    private let client: DaemonAPIClient

    fileprivate init(client: DaemonAPIClient) {
        self.client = client
    }

    func channelStatus(
        baseURL: URL,
        authToken: String,
        workspaceID: String
    ) async throws -> DaemonChannelStatusResponse {
        try await client.channelStatus(baseURL: baseURL, authToken: authToken, workspaceID: workspaceID)
    }

    func channelConnectorMappingsList(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        channelID: String? = nil
    ) async throws -> DaemonChannelConnectorMappingListResponse {
        try await client.channelConnectorMappingsList(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            channelID: channelID
        )
    }

    func channelConnectorMappingUpsert(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        channelID: String,
        connectorID: String,
        enabled: Bool,
        priority: Int? = nil,
        fallbackPolicy: String? = nil
    ) async throws -> DaemonChannelConnectorMappingUpsertResponse {
        try await client.channelConnectorMappingUpsert(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            channelID: channelID,
            connectorID: connectorID,
            enabled: enabled,
            priority: priority,
            fallbackPolicy: fallbackPolicy
        )
    }

    func channelDiagnostics(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        channelID: String? = nil
    ) async throws -> DaemonChannelDiagnosticsResponse {
        try await client.channelDiagnostics(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            channelID: channelID
        )
    }

    func channelConfigUpsert(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        channelID: String,
        configuration: [String: DaemonConfigMutationValue],
        merge: Bool = true
    ) async throws -> DaemonChannelConfigUpsertResponse {
        try await client.channelConfigUpsert(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            channelID: channelID,
            configuration: configuration,
            merge: merge
        )
    }

    func channelTestOperation(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        channelID: String,
        operation: String = "health"
    ) async throws -> DaemonChannelTestOperationResponse {
        try await client.channelTestOperation(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            channelID: channelID,
            operation: operation
        )
    }
}

struct DaemonConnectorsAPI {
    private let client: DaemonAPIClient

    fileprivate init(client: DaemonAPIClient) {
        self.client = client
    }

    func connectorStatus(
        baseURL: URL,
        authToken: String,
        workspaceID: String
    ) async throws -> DaemonConnectorStatusResponse {
        try await client.connectorStatus(baseURL: baseURL, authToken: authToken, workspaceID: workspaceID)
    }

    func connectorDiagnostics(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        connectorID: String? = nil
    ) async throws -> DaemonConnectorDiagnosticsResponse {
        try await client.connectorDiagnostics(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            connectorID: connectorID
        )
    }

    func connectorConfigUpsert(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        connectorID: String,
        configuration: [String: DaemonConfigMutationValue],
        merge: Bool = true
    ) async throws -> DaemonConnectorConfigUpsertResponse {
        try await client.connectorConfigUpsert(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            connectorID: connectorID,
            configuration: configuration,
            merge: merge
        )
    }

    func connectorTestOperation(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        connectorID: String,
        operation: String = "health"
    ) async throws -> DaemonConnectorTestOperationResponse {
        try await client.connectorTestOperation(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            connectorID: connectorID,
            operation: operation
        )
    }

    func connectorPermissionRequest(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        connectorID: String
    ) async throws -> DaemonConnectorPermissionResponse {
        try await client.connectorPermissionRequest(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            connectorID: connectorID
        )
    }
}

struct DaemonIdentityAPI {
    private let client: DaemonAPIClient

    fileprivate init(client: DaemonAPIClient) {
        self.client = client
    }

    func identityWorkspaces(
        baseURL: URL,
        authToken: String,
        includeInactive: Bool = true
    ) async throws -> DaemonIdentityWorkspacesResponse {
        try await client.identityWorkspaces(
            baseURL: baseURL,
            authToken: authToken,
            includeInactive: includeInactive
        )
    }

    func identityPrincipals(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        includeInactive: Bool = true
    ) async throws -> DaemonIdentityPrincipalsResponse {
        try await client.identityPrincipals(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            includeInactive: includeInactive
        )
    }

    func identityActiveContext(
        baseURL: URL,
        authToken: String,
        workspaceID: String? = nil
    ) async throws -> DaemonIdentityActiveContextResponse {
        try await client.identityActiveContext(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID
        )
    }

    func identitySelectWorkspace(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        principalActorID: String? = nil,
        source: String? = "ui.configuration.identity_hub"
    ) async throws -> DaemonIdentityActiveContextResponse {
        try await client.identitySelectWorkspace(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            principalActorID: principalActorID,
            source: source
        )
    }

    func identityDevicesList(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        userID: String? = nil,
        deviceType: String? = nil,
        platform: String? = nil,
        cursorCreatedAt: String? = nil,
        cursorID: String? = nil,
        limit: Int = 25
    ) async throws -> DaemonIdentityDeviceListResponse {
        try await client.identityDevicesList(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            userID: userID,
            deviceType: deviceType,
            platform: platform,
            cursorCreatedAt: cursorCreatedAt,
            cursorID: cursorID,
            limit: limit
        )
    }

    func identitySessionsList(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        deviceID: String? = nil,
        userID: String? = nil,
        sessionHealth: String? = nil,
        cursorStartedAt: String? = nil,
        cursorID: String? = nil,
        limit: Int = 25
    ) async throws -> DaemonIdentitySessionListResponse {
        try await client.identitySessionsList(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            deviceID: deviceID,
            userID: userID,
            sessionHealth: sessionHealth,
            cursorStartedAt: cursorStartedAt,
            cursorID: cursorID,
            limit: limit
        )
    }

    func identitySessionRevoke(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        sessionID: String
    ) async throws -> DaemonIdentitySessionRevokeResponse {
        try await client.identitySessionRevoke(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            sessionID: sessionID
        )
    }

    func delegationList(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        fromActorID: String? = nil,
        toActorID: String? = nil
    ) async throws -> DaemonDelegationListResponse {
        try await client.delegationList(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            fromActorID: fromActorID,
            toActorID: toActorID
        )
    }

    func delegationGrant(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        fromActorID: String,
        toActorID: String,
        scopeType: String,
        scopeKey: String? = nil,
        expiresAt: String? = nil
    ) async throws -> DaemonDelegationRuleRecord {
        try await client.delegationGrant(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            fromActorID: fromActorID,
            toActorID: toActorID,
            scopeType: scopeType,
            scopeKey: scopeKey,
            expiresAt: expiresAt
        )
    }

    func delegationRevoke(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        ruleID: String
    ) async throws -> DaemonDelegationRevokeResponse {
        try await client.delegationRevoke(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            ruleID: ruleID
        )
    }

    func capabilityGrantUpsert(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        grantID: String? = nil,
        actorID: String? = nil,
        capabilityKey: String? = nil,
        scopeJSON: String? = nil,
        status: String? = nil,
        expiresAt: String? = nil
    ) async throws -> DaemonCapabilityGrantRecord {
        try await client.capabilityGrantUpsert(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            grantID: grantID,
            actorID: actorID,
            capabilityKey: capabilityKey,
            scopeJSON: scopeJSON,
            status: status,
            expiresAt: expiresAt
        )
    }

    func capabilityGrantList(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        actorID: String? = nil,
        capabilityKey: String? = nil,
        status: String? = nil,
        cursorCreatedAt: String? = nil,
        cursorID: String? = nil,
        limit: Int = 50
    ) async throws -> DaemonCapabilityGrantListResponse {
        try await client.capabilityGrantList(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            actorID: actorID,
            capabilityKey: capabilityKey,
            status: status,
            cursorCreatedAt: cursorCreatedAt,
            cursorID: cursorID,
            limit: limit
        )
    }
}

struct DaemonApprovalsAPI {
    private let client: DaemonAPIClient

    fileprivate init(client: DaemonAPIClient) {
        self.client = client
    }

    func approvalInbox(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        includeFinal: Bool = true,
        limit: Int = 80,
        state: String? = nil
    ) async throws -> DaemonApprovalInboxResponse {
        try await client.approvalInbox(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            includeFinal: includeFinal,
            limit: limit,
            state: state
        )
    }

    func approvalDecision(
        baseURL: URL,
        authToken: String,
        approvalID: String,
        decisionPhrase: String,
        decisionByActorID: String,
        rationale: String? = nil
    ) async throws -> DaemonApprovalDecisionResponse {
        try await client.approvalDecision(
            baseURL: baseURL,
            authToken: authToken,
            approvalID: approvalID,
            decisionPhrase: decisionPhrase,
            decisionByActorID: decisionByActorID,
            rationale: rationale
        )
    }
}

struct DaemonTasksAPI {
    private let client: DaemonAPIClient

    fileprivate init(client: DaemonAPIClient) {
        self.client = client
    }

    func taskRunList(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        state: String? = nil,
        limit: Int = 80
    ) async throws -> DaemonTaskRunListResponse {
        try await client.taskRunList(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            state: state,
            limit: limit
        )
    }

    func taskCancel(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        taskID: String? = nil,
        runID: String? = nil,
        reason: String? = nil
    ) async throws -> DaemonTaskCancelResponse {
        try await client.taskCancel(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            taskID: taskID,
            runID: runID,
            reason: reason
        )
    }

    func taskRetry(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        taskID: String? = nil,
        runID: String? = nil,
        reason: String? = nil
    ) async throws -> DaemonTaskRetryResponse {
        try await client.taskRetry(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            taskID: taskID,
            runID: runID,
            reason: reason
        )
    }

    func taskRequeue(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        taskID: String? = nil,
        runID: String? = nil,
        reason: String? = nil
    ) async throws -> DaemonTaskRequeueResponse {
        try await client.taskRequeue(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            taskID: taskID,
            runID: runID,
            reason: reason
        )
    }

    func taskSubmit(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        requestedByActorID: String,
        subjectPrincipalActorID: String,
        title: String,
        description: String? = nil,
        taskClass: String? = nil
    ) async throws -> DaemonTaskSubmitResponse {
        try await client.taskSubmit(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            requestedByActorID: requestedByActorID,
            subjectPrincipalActorID: subjectPrincipalActorID,
            title: title,
            description: description,
            taskClass: taskClass
        )
    }
}

struct DaemonAutomationAPI {
    private let client: DaemonAPIClient

    fileprivate init(client: DaemonAPIClient) {
        self.client = client
    }

    func automationList(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        triggerType: String? = nil,
        includeDisabled: Bool = true
    ) async throws -> DaemonAutomationListResponse {
        try await client.automationList(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            triggerType: triggerType,
            includeDisabled: includeDisabled
        )
    }

    func automationCreate(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        subjectActorID: String?,
        triggerType: String,
        title: String,
        instruction: String,
        intervalSeconds: Int? = nil,
        filterJSON: String? = nil,
        cooldownSeconds: Int? = nil,
        enabled: Bool
    ) async throws -> DaemonAutomationTriggerRecord {
        try await client.automationCreate(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            subjectActorID: subjectActorID,
            triggerType: triggerType,
            title: title,
            instruction: instruction,
            intervalSeconds: intervalSeconds,
            filterJSON: filterJSON,
            cooldownSeconds: cooldownSeconds,
            enabled: enabled
        )
    }

    func automationFireHistory(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        triggerID: String? = nil,
        status: String? = nil,
        limit: Int = 50
    ) async throws -> DaemonAutomationFireHistoryResponse {
        try await client.automationFireHistory(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            triggerID: triggerID,
            status: status,
            limit: limit
        )
    }

    func automationUpdate(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        triggerID: String,
        subjectActorID: String?,
        title: String?,
        instruction: String?,
        intervalSeconds: Int? = nil,
        filterJSON: String? = nil,
        cooldownSeconds: Int? = nil,
        enabled: Bool? = nil
    ) async throws -> DaemonAutomationUpdateResponse {
        try await client.automationUpdate(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            triggerID: triggerID,
            subjectActorID: subjectActorID,
            title: title,
            instruction: instruction,
            intervalSeconds: intervalSeconds,
            filterJSON: filterJSON,
            cooldownSeconds: cooldownSeconds,
            enabled: enabled
        )
    }

    func automationDelete(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        triggerID: String
    ) async throws -> DaemonAutomationDeleteResponse {
        try await client.automationDelete(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            triggerID: triggerID
        )
    }

    func automationRunSchedule(
        baseURL: URL,
        authToken: String,
        at: String? = nil
    ) async throws -> DaemonAutomationRunScheduleResponse {
        try await client.automationRunSchedule(baseURL: baseURL, authToken: authToken, at: at)
    }

    func automationRunCommEvent(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        eventID: String,
        channel: String,
        body: String,
        sender: String
    ) async throws -> DaemonAutomationRunCommEventResponse {
        try await client.automationRunCommEvent(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            eventID: eventID,
            channel: channel,
            body: body,
            sender: sender
        )
    }
}

struct DaemonContextAPI {
    private let client: DaemonAPIClient

    fileprivate init(client: DaemonAPIClient) {
        self.client = client
    }

    func retentionPurge(
        baseURL: URL,
        authToken: String,
        traceDays: Int,
        transcriptDays: Int,
        memoryDays: Int
    ) async throws -> DaemonJSONValue {
        try await client.retentionPurge(
            baseURL: baseURL,
            authToken: authToken,
            traceDays: traceDays,
            transcriptDays: transcriptDays,
            memoryDays: memoryDays
        )
    }

    func retentionCompactMemory(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        ownerActor: String,
        tokenThreshold: Int,
        staleAfterHours: Int,
        limit: Int,
        apply: Bool
    ) async throws -> DaemonJSONValue {
        try await client.retentionCompactMemory(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            ownerActor: ownerActor,
            tokenThreshold: tokenThreshold,
            staleAfterHours: staleAfterHours,
            limit: limit,
            apply: apply
        )
    }

    func contextSamples(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        taskClass: String,
        limit: Int
    ) async throws -> DaemonJSONValue {
        try await client.contextSamples(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            taskClass: taskClass,
            limit: limit
        )
    }

    func contextTune(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        taskClass: String
    ) async throws -> DaemonJSONValue {
        try await client.contextTune(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            taskClass: taskClass
        )
    }

    func contextMemoryInventory(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        ownerActorID: String? = nil,
        scopeType: String? = nil,
        status: String? = nil,
        sourceType: String? = nil,
        sourceRefQuery: String? = nil,
        cursorUpdatedAt: String? = nil,
        cursorID: String? = nil,
        limit: Int = 25
    ) async throws -> DaemonContextMemoryInventoryResponse {
        try await client.contextMemoryInventory(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            ownerActorID: ownerActorID,
            scopeType: scopeType,
            status: status,
            sourceType: sourceType,
            sourceRefQuery: sourceRefQuery,
            cursorUpdatedAt: cursorUpdatedAt,
            cursorID: cursorID,
            limit: limit
        )
    }

    func contextMemoryCompactionCandidates(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        ownerActorID: String? = nil,
        status: String? = nil,
        cursorCreatedAt: String? = nil,
        cursorID: String? = nil,
        limit: Int = 25
    ) async throws -> DaemonContextMemoryCandidatesResponse {
        try await client.contextMemoryCompactionCandidates(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            ownerActorID: ownerActorID,
            status: status,
            cursorCreatedAt: cursorCreatedAt,
            cursorID: cursorID,
            limit: limit
        )
    }

    func contextRetrievalDocuments(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        ownerActorID: String? = nil,
        sourceURIQuery: String? = nil,
        cursorCreatedAt: String? = nil,
        cursorID: String? = nil,
        limit: Int = 25
    ) async throws -> DaemonContextRetrievalDocumentsResponse {
        try await client.contextRetrievalDocuments(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            ownerActorID: ownerActorID,
            sourceURIQuery: sourceURIQuery,
            cursorCreatedAt: cursorCreatedAt,
            cursorID: cursorID,
            limit: limit
        )
    }

    func contextRetrievalChunks(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        documentID: String,
        ownerActorID: String? = nil,
        sourceURIQuery: String? = nil,
        chunkTextQuery: String? = nil,
        cursorCreatedAt: String? = nil,
        cursorID: String? = nil,
        limit: Int = 25
    ) async throws -> DaemonContextRetrievalChunksResponse {
        try await client.contextRetrievalChunks(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            documentID: documentID,
            ownerActorID: ownerActorID,
            sourceURIQuery: sourceURIQuery,
            chunkTextQuery: chunkTextQuery,
            cursorCreatedAt: cursorCreatedAt,
            cursorID: cursorID,
            limit: limit
        )
    }
}

struct DaemonChatAPI {
    private let client: DaemonAPIClient

    fileprivate init(client: DaemonAPIClient) {
        self.client = client
    }

    func connectRealtime(
        baseURL: URL,
        authToken: String,
        correlationID: String? = nil
    ) throws -> DaemonRealtimeSession {
        try client.connectRealtime(baseURL: baseURL, authToken: authToken, correlationID: correlationID)
    }

    func chatTurn(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        messages: [(role: String, content: String)],
        systemPrompt: String? = nil,
        requestedByActorID: String? = nil,
        subjectActorID: String? = nil,
        actingAsActorID: String? = nil,
        correlationID: String? = nil
    ) async throws -> DaemonChatTurnResponse {
        try await client.chatTurn(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            messages: messages,
            systemPrompt: systemPrompt,
            requestedByActorID: requestedByActorID,
            subjectActorID: subjectActorID,
            actingAsActorID: actingAsActorID,
            correlationID: correlationID
        )
    }

    func chatTurnExplain(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        taskClass: String = "chat",
        requestedByActorID: String? = nil,
        subjectActorID: String? = nil,
        actingAsActorID: String? = nil
    ) async throws -> DaemonChatTurnExplainResponse {
        try await client.chatTurnExplain(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            taskClass: taskClass,
            requestedByActorID: requestedByActorID,
            subjectActorID: subjectActorID,
            actingAsActorID: actingAsActorID
        )
    }

    func chatTurnHistory(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        channelID: String? = nil,
        connectorID: String? = nil,
        threadID: String? = nil,
        correlationID: String? = nil,
        beforeCreatedAt: String? = nil,
        beforeItemID: String? = nil,
        limit: Int = 120
    ) async throws -> DaemonChatTurnHistoryResponse {
        try await client.chatTurnHistory(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            channelID: channelID,
            connectorID: connectorID,
            threadID: threadID,
            correlationID: correlationID,
            beforeCreatedAt: beforeCreatedAt,
            beforeItemID: beforeItemID,
            limit: limit
        )
    }

    func chatPersonaPolicyGet(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        principalActorID: String? = nil,
        channelID: String? = nil
    ) async throws -> DaemonChatPersonaPolicyResponse {
        try await client.chatPersonaPolicyGet(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            principalActorID: principalActorID,
            channelID: channelID
        )
    }

    func chatPersonaPolicySet(
        baseURL: URL,
        authToken: String,
        workspaceID: String,
        principalActorID: String? = nil,
        channelID: String? = nil,
        stylePrompt: String,
        guardrails: [String]
    ) async throws -> DaemonChatPersonaPolicyResponse {
        try await client.chatPersonaPolicySet(
            baseURL: baseURL,
            authToken: authToken,
            workspaceID: workspaceID,
            principalActorID: principalActorID,
            channelID: channelID,
            stylePrompt: stylePrompt,
            guardrails: guardrails
        )
    }
}
