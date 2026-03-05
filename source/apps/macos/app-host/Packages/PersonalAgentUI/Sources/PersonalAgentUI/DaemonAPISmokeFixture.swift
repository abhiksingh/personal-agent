import Foundation

enum DaemonAPISmokeFixtureScenario: String, Sendable {
    case ready
    case onboarding
    case degraded

    static let environmentKey = "PA_UI_SMOKE_FIXTURE_SCENARIO"

    static func fromEnvironment(
        _ environment: [String: String] = ProcessInfo.processInfo.environment
    ) -> DaemonAPISmokeFixtureScenario? {
        guard let rawValue = environment[environmentKey]?
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .lowercased(),
              !rawValue.isEmpty else {
            return nil
        }
        return DaemonAPISmokeFixtureScenario(rawValue: rawValue)
    }
}

enum DaemonAPISmokeFixture {
    private static let workspaceID = "ws1"
    private static let now = "2026-02-26T20:00:00Z"
    private static let oneMinuteAgo = "2026-02-26T19:59:00Z"
    private static let twoMinutesAgo = "2026-02-26T19:58:00Z"
    private static let fiveMinutesAgo = "2026-02-26T19:55:00Z"

    static func responseData(
        method: String,
        path: String,
        body: Data?,
        scenario: DaemonAPISmokeFixtureScenario
    ) throws -> Data {
        let payloadObject = try payloadObject(
            method: method,
            path: path,
            body: body,
            scenario: scenario
        )
        return try JSONSerialization.data(withJSONObject: payloadObject, options: [])
    }

    private static func payloadObject(
        method: String,
        path: String,
        body: Data?,
        scenario: DaemonAPISmokeFixtureScenario
    ) throws -> [String: Any] {
        let normalizedMethod = method.trimmingCharacters(in: .whitespacesAndNewlines).uppercased()
        let normalizedPath = path.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()

        switch scenario {
        case .ready:
            return try readyPayload(
                method: normalizedMethod,
                path: normalizedPath,
                body: body
            )
        case .onboarding:
            return try onboardingPayload(
                method: normalizedMethod,
                path: normalizedPath,
                body: body
            )
        case .degraded:
            return try degradedPayload(
                method: normalizedMethod,
                path: normalizedPath,
                body: body
            )
        }
    }

    private static func degradedPayload(
        method: String,
        path: String,
        body: Data?
    ) throws -> [String: Any] {
        if method == "GET", path == "/v1/daemon/lifecycle/status" {
            return lifecycleDegradedPayload()
        }
        if method == "POST", path == "/v1/connectors/status" {
            return connectorStatusDegradedPayload()
        }
        if method == "POST", path == "/v1/connectors/diagnostics" {
            return connectorDiagnosticsDegradedPayload()
        }
        return try readyPayload(
            method: method,
            path: path,
            body: body
        )
    }

    private static func readyPayload(
        method: String,
        path: String,
        body: Data?
    ) throws -> [String: Any] {
        if method == "POST", path == "/v1/tasks" {
            return taskSubmitReadyPayload(body: body)
        }
        if method == "POST",
           path.hasPrefix("/v1/approvals/"),
           path != "/v1/approvals/list" {
            return approvalDecisionReadyPayload(path: path, body: body)
        }

        switch (method, path) {
        case ("GET", "/v1/daemon/lifecycle/status"):
            return lifecycleReadyPayload()
        case ("POST", "/v1/identity/context"):
            return identityContextPayload()
        case ("POST", "/v1/identity/workspaces"):
            return identityWorkspacesPayload()
        case ("POST", "/v1/identity/principals"):
            return identityPrincipalsPayload()
        case ("POST", "/v1/providers/list"):
            return providersListReadyPayload()
        case ("POST", "/v1/models/resolve"):
            return modelResolveReadyPayload()
        case ("POST", "/v1/models/list"):
            return modelListReadyPayload()
        case ("POST", "/v1/models/policy"):
            return modelPolicyReadyPayload()
        case ("POST", "/v1/channels/mappings/list"):
            return channelMappingsReadyPayload()
        case ("POST", "/v1/channels/status"):
            return channelStatusReadyPayload()
        case ("POST", "/v1/channels/diagnostics"):
            return channelDiagnosticsReadyPayload()
        case ("POST", "/v1/comm/policy/list"):
            return ["workspace_id": workspaceID, "policies": []]
        case ("POST", "/v1/connectors/status"):
            return connectorStatusReadyPayload()
        case ("POST", "/v1/connectors/diagnostics"):
            return connectorDiagnosticsReadyPayload()
        case ("POST", "/v1/tasks/list"):
            return taskListReadyPayload()
        case ("POST", "/v1/approvals/list"):
            return approvalsReadyPayload()
        case ("POST", "/v1/inspect/run"):
            let runID = requestStringValue(key: "run_id", from: body) ?? "run-1"
            return inspectRunReadyPayload(runID: runID)
        case ("POST", "/v1/inspect/logs/query"):
            return inspectLogsQueryReadyPayload()
        case ("POST", "/v1/inspect/logs/stream"):
            return inspectLogsStreamReadyPayload()
        case ("POST", "/v1/chat/turn"):
            let correlationID = requestHeaderCorrelationFallback(from: body) ?? "corr-fixture-chat"
            return chatTurnReadyPayload(
                correlationID: correlationID,
                latestUserMessage: requestLatestUserMessage(from: body)
            )
        case ("POST", "/v1/chat/turn/explain"):
            return chatTurnExplainReadyPayload(body: body)
        case ("POST", "/v1/chat/history"):
            return chatTurnHistoryReadyPayload()
        default:
            throw DaemonAPIError.server(
                statusCode: 404,
                message: "Fixture missing endpoint for \(method) \(path)"
            )
        }
    }

    private static func onboardingPayload(
        method: String,
        path: String,
        body: Data?
    ) throws -> [String: Any] {
        if method == "POST", path == "/v1/tasks" {
            return taskSubmitReadyPayload(body: body)
        }
        if method == "POST",
           path.hasPrefix("/v1/approvals/"),
           path != "/v1/approvals/list" {
            return approvalDecisionReadyPayload(path: path, body: body)
        }

        switch (method, path) {
        case ("GET", "/v1/daemon/lifecycle/status"):
            return lifecycleReadyPayload()
        case ("POST", "/v1/identity/context"):
            return identityContextPayload()
        case ("POST", "/v1/identity/workspaces"):
            return identityWorkspacesPayload()
        case ("POST", "/v1/identity/principals"):
            return identityPrincipalsPayload()
        case ("POST", "/v1/providers/list"):
            return providersListReadyPayload()
        case ("POST", "/v1/models/resolve"):
            throw DaemonAPIError.server(
                statusCode: 400,
                message: "no enabled models with ready provider configuration for workspace \"ws1\""
            )
        case ("POST", "/v1/models/list"):
            return modelListReadyPayload()
        case ("POST", "/v1/models/policy"):
            return modelPolicyReadyPayload()
        case ("POST", "/v1/channels/mappings/list"):
            return channelMappingsReadyPayload()
        case ("POST", "/v1/channels/status"):
            return channelStatusReadyPayload()
        case ("POST", "/v1/channels/diagnostics"):
            return channelDiagnosticsReadyPayload()
        case ("POST", "/v1/comm/policy/list"):
            return ["workspace_id": workspaceID, "policies": []]
        case ("POST", "/v1/connectors/status"):
            return connectorStatusReadyPayload()
        case ("POST", "/v1/connectors/diagnostics"):
            return connectorDiagnosticsReadyPayload()
        case ("POST", "/v1/tasks/list"):
            return taskListReadyPayload()
        case ("POST", "/v1/approvals/list"):
            return approvalsReadyPayload()
        case ("POST", "/v1/inspect/run"):
            let runID = requestStringValue(key: "run_id", from: body) ?? "run-1"
            return inspectRunReadyPayload(runID: runID)
        case ("POST", "/v1/inspect/logs/query"):
            return inspectLogsQueryReadyPayload()
        case ("POST", "/v1/inspect/logs/stream"):
            return inspectLogsStreamReadyPayload()
        case ("POST", "/v1/chat/turn"):
            let correlationID = requestHeaderCorrelationFallback(from: body) ?? "corr-fixture-chat"
            return chatTurnReadyPayload(
                correlationID: correlationID,
                latestUserMessage: requestLatestUserMessage(from: body)
            )
        case ("POST", "/v1/chat/turn/explain"):
            return chatTurnExplainReadyPayload(body: body)
        case ("POST", "/v1/chat/history"):
            return chatTurnHistoryReadyPayload()
        default:
            throw DaemonAPIError.server(
                statusCode: 404,
                message: "Fixture missing endpoint for \(method) \(path)"
            )
        }
    }

    private static func lifecycleReadyPayload() -> [String: Any] {
        [
            "lifecycle_state": "running",
            "process_id": 7311,
            "started_at": fiveMinutesAgo,
            "last_transition_at": oneMinuteAgo,
            "runtime_mode": "tcp",
            "configured_address": "127.0.0.1:7071",
            "bound_address": "127.0.0.1:7071",
            "setup_state": "ready",
            "install_state": "installed",
            "needs_install": false,
            "needs_repair": false,
            "health_classification": [
                "overall_state": "ready",
                "core_runtime_state": "ready",
                "plugin_runtime_state": "healthy",
                "blocking": false
            ],
            "database_ready": true,
            "control_auth": [
                "state": "configured",
                "source": "fixture",
                "remediation_hints": []
            ],
            "worker_summary": [
                "total": 9,
                "registered": 9,
                "starting": 0,
                "running": 9,
                "restarting": 0,
                "stopped": 0,
                "failed": 0
            ],
            "controls": [
                "start": false,
                "stop": true,
                "restart": true,
                "install": false,
                "uninstall": false,
                "repair": false
            ],
            "control_operation": [
                "action": "",
                "state": "idle"
            ]
        ]
    }

    private static func lifecycleDegradedPayload() -> [String: Any] {
        [
            "lifecycle_state": "running",
            "process_id": 7311,
            "started_at": fiveMinutesAgo,
            "last_transition_at": oneMinuteAgo,
            "runtime_mode": "tcp",
            "configured_address": "127.0.0.1:7071",
            "bound_address": "127.0.0.1:7071",
            "setup_state": "ready",
            "install_state": "installed",
            "needs_install": false,
            "needs_repair": false,
            "health_classification": [
                "overall_state": "degraded",
                "core_runtime_state": "ready",
                "plugin_runtime_state": "degraded",
                "blocking": false
            ],
            "database_ready": true,
            "control_auth": [
                "state": "configured",
                "source": "fixture",
                "remediation_hints": []
            ],
            "worker_summary": [
                "total": 9,
                "registered": 9,
                "starting": 0,
                "running": 8,
                "restarting": 0,
                "stopped": 0,
                "failed": 1
            ],
            "controls": [
                "start": false,
                "stop": true,
                "restart": true,
                "install": false,
                "uninstall": false,
                "repair": false
            ],
            "control_operation": [
                "action": "",
                "state": "idle"
            ]
        ]
    }

    private static func identityContextPayload() -> [String: Any] {
        [
            "active_context": [
                "workspace_id": workspaceID,
                "principal_actor_id": "default",
                "workspace_source": "selected",
                "principal_source": "selected",
                "last_updated_at": now,
                "workspace_resolved": true
            ]
        ]
    }

    private static func identityWorkspacesPayload() -> [String: Any] {
        [
            "active_context": [
                "workspace_id": workspaceID,
                "principal_actor_id": "default",
                "workspace_source": "selected",
                "principal_source": "selected",
                "last_updated_at": now,
                "workspace_resolved": true
            ],
            "workspaces": [
                [
                    "workspace_id": workspaceID,
                    "name": "Default Workspace",
                    "status": "ACTIVE",
                    "principal_count": 2,
                    "actor_count": 2,
                    "handle_count": 1,
                    "updated_at": now,
                    "is_active": true
                ]
            ]
        ]
    }

    private static func identityPrincipalsPayload() -> [String: Any] {
        [
            "workspace_id": workspaceID,
            "active_context": [
                "workspace_id": workspaceID,
                "principal_actor_id": "default",
                "workspace_source": "selected",
                "principal_source": "selected",
                "last_updated_at": now,
                "workspace_resolved": true
            ],
            "principals": [
                [
                    "actor_id": "default",
                    "display_name": "Default Principal",
                    "actor_type": "human",
                    "actor_status": "ACTIVE",
                    "principal_status": "ACTIVE",
                    "is_active": true,
                    "handles": []
                ],
                [
                    "actor_id": "owner",
                    "display_name": "Workspace Owner",
                    "actor_type": "human",
                    "actor_status": "ACTIVE",
                    "principal_status": "ACTIVE",
                    "is_active": true,
                    "handles": []
                ]
            ]
        ]
    }

    private static func providersListReadyPayload() -> [String: Any] {
        [
            "workspace_id": workspaceID,
            "providers": [
                [
                    "workspace_id": workspaceID,
                    "provider": "openai",
                    "endpoint": "https://api.openai.com/v1",
                    "api_key_secret_name": "OPENAI_API_KEY",
                    "api_key_configured": true,
                    "updated_at": now
                ]
            ]
        ]
    }

    private static func modelResolveReadyPayload() -> [String: Any] {
        [
            "workspace_id": workspaceID,
            "task_class": "chat",
            "provider": "openai",
            "model_key": "gpt-4.1",
            "source": "policy",
            "notes": "Fixture route selected"
        ]
    }

    private static func modelListReadyPayload() -> [String: Any] {
        [
            "workspace_id": workspaceID,
            "models": [
                [
                    "workspace_id": workspaceID,
                    "provider": "openai",
                    "model_key": "gpt-4.1",
                    "enabled": true,
                    "provider_ready": true,
                    "provider_endpoint": "https://api.openai.com/v1"
                ],
                [
                    "workspace_id": workspaceID,
                    "provider": "openai",
                    "model_key": "gpt-4o-mini",
                    "enabled": false,
                    "provider_ready": true,
                    "provider_endpoint": "https://api.openai.com/v1"
                ]
            ]
        ]
    }

    private static func modelPolicyReadyPayload() -> [String: Any] {
        [
            "workspace_id": workspaceID,
            "policies": [
                [
                    "workspace_id": workspaceID,
                    "task_class": "chat",
                    "provider": "openai",
                    "model_key": "gpt-4.1",
                    "updated_at": now
                ]
            ]
        ]
    }

    private static func channelMappingsReadyPayload() -> [String: Any] {
        [
            "workspace_id": workspaceID,
            "fallback_policy": "priority_order",
            "bindings": [
                [
                    "channel_id": "app",
                    "connector_id": "builtin.app",
                    "enabled": true,
                    "priority": 1,
                    "capabilities": ["channel.app.send"],
                    "created_at": twoMinutesAgo,
                    "updated_at": oneMinuteAgo
                ],
                [
                    "channel_id": "message",
                    "connector_id": "imessage",
                    "enabled": true,
                    "priority": 1,
                    "capabilities": ["channel.message.send"],
                    "created_at": twoMinutesAgo,
                    "updated_at": oneMinuteAgo
                ],
                [
                    "channel_id": "message",
                    "connector_id": "twilio",
                    "enabled": true,
                    "priority": 2,
                    "capabilities": ["channel.message.send"],
                    "created_at": twoMinutesAgo,
                    "updated_at": oneMinuteAgo
                ],
                [
                    "channel_id": "voice",
                    "connector_id": "twilio",
                    "enabled": true,
                    "priority": 1,
                    "capabilities": ["channel.voice.call"],
                    "created_at": twoMinutesAgo,
                    "updated_at": oneMinuteAgo
                ]
            ]
        ]
    }

    private static func channelStatusReadyPayload() -> [String: Any] {
        [
            "workspace_id": workspaceID,
            "channels": [
                [
                    "channel_id": "app",
                    "display_name": "App",
                    "category": "chat",
                    "enabled": true,
                    "configured": true,
                    "status": "ready",
                    "summary": "App channel is ready."
                ],
                [
                    "channel_id": "message",
                    "display_name": "Message",
                    "category": "message",
                    "enabled": true,
                    "configured": true,
                    "status": "ready",
                    "summary": "Message channel is ready."
                ],
                [
                    "channel_id": "voice",
                    "display_name": "Voice",
                    "category": "voice",
                    "enabled": true,
                    "configured": true,
                    "status": "ready",
                    "summary": "Voice channel is ready."
                ]
            ]
        ]
    }

    private static func channelDiagnosticsReadyPayload() -> [String: Any] {
        [
            "workspace_id": workspaceID,
            "diagnostics": [
                [
                    "channel_id": "app",
                    "display_name": "App",
                    "category": "chat",
                    "configured": true,
                    "status": "ready",
                    "summary": "App channel healthy.",
                    "worker_health": [
                        "registered": true,
                        "worker": [
                            "plugin_id": "channel-app",
                            "kind": "channel",
                            "state": "running",
                            "process_id": 7001,
                            "restart_count": 0
                        ]
                    ],
                    "remediation_actions": []
                ],
                [
                    "channel_id": "message",
                    "display_name": "Message",
                    "category": "message",
                    "configured": true,
                    "status": "ready",
                    "summary": "Message channel healthy.",
                    "worker_health": [
                        "registered": true,
                        "worker": [
                            "plugin_id": "channel-message",
                            "kind": "channel",
                            "state": "running",
                            "process_id": 7002,
                            "restart_count": 0
                        ]
                    ],
                    "remediation_actions": []
                ],
                [
                    "channel_id": "voice",
                    "display_name": "Voice",
                    "category": "voice",
                    "configured": true,
                    "status": "ready",
                    "summary": "Voice channel healthy.",
                    "worker_health": [
                        "registered": true,
                        "worker": [
                            "plugin_id": "channel-voice",
                            "kind": "channel",
                            "state": "running",
                            "process_id": 7003,
                            "restart_count": 0
                        ]
                    ],
                    "remediation_actions": []
                ]
            ]
        ]
    }

    private static func connectorStatusReadyPayload() -> [String: Any] {
        [
            "workspace_id": workspaceID,
            "connectors": [
                [
                    "connector_id": "builtin.app",
                    "plugin_id": "connector-builtin-app",
                    "display_name": "App",
                    "enabled": true,
                    "configured": true,
                    "status": "ready",
                    "summary": "In-app transport ready.",
                    "configuration": [
                        "permission_state": "granted"
                    ],
                    "remediation_actions": []
                ],
                [
                    "connector_id": "imessage",
                    "plugin_id": "connector-imessage",
                    "display_name": "iMessage",
                    "enabled": true,
                    "configured": true,
                    "status": "ready",
                    "summary": "iMessage connector ready.",
                    "configuration": [
                        "permission_state": "granted"
                    ],
                    "remediation_actions": [
                        [
                            "identifier": "open_connector_system_settings",
                            "label": "Open System Settings",
                            "intent": "open_system_settings",
                            "destination": "ui://system-settings/privacy/full-disk-access",
                            "enabled": true,
                            "recommended": false
                        ]
                    ]
                ],
                [
                    "connector_id": "twilio",
                    "plugin_id": "connector-twilio",
                    "display_name": "Twilio",
                    "enabled": true,
                    "configured": true,
                    "status": "ready",
                    "summary": "Twilio connector ready.",
                    "configuration": [
                        "permission_state": "granted"
                    ],
                    "remediation_actions": []
                ]
            ]
        ]
    }

    private static func connectorDiagnosticsReadyPayload() -> [String: Any] {
        [
            "workspace_id": workspaceID,
            "diagnostics": [
                [
                    "connector_id": "builtin.app",
                    "plugin_id": "connector-builtin-app",
                    "display_name": "App",
                    "configured": true,
                    "status": "ready",
                    "summary": "Connector healthy.",
                    "worker_health": [
                        "registered": true,
                        "worker": [
                            "plugin_id": "connector-builtin-app",
                            "kind": "connector",
                            "state": "running",
                            "process_id": 7101,
                            "restart_count": 0
                        ]
                    ],
                    "remediation_actions": []
                ],
                [
                    "connector_id": "imessage",
                    "plugin_id": "connector-imessage",
                    "display_name": "iMessage",
                    "configured": true,
                    "status": "ready",
                    "summary": "Connector healthy.",
                    "worker_health": [
                        "registered": true,
                        "worker": [
                            "plugin_id": "connector-imessage",
                            "kind": "connector",
                            "state": "running",
                            "process_id": 7102,
                            "restart_count": 0
                        ]
                    ],
                    "remediation_actions": []
                ],
                [
                    "connector_id": "twilio",
                    "plugin_id": "connector-twilio",
                    "display_name": "Twilio",
                    "configured": true,
                    "status": "ready",
                    "summary": "Connector healthy.",
                    "worker_health": [
                        "registered": true,
                        "worker": [
                            "plugin_id": "connector-twilio",
                            "kind": "connector",
                            "state": "running",
                            "process_id": 7103,
                            "restart_count": 0
                        ]
                    ],
                    "remediation_actions": []
                ]
            ]
        ]
    }

    private static func connectorStatusDegradedPayload() -> [String: Any] {
        [
            "workspace_id": workspaceID,
            "connectors": [
                [
                    "connector_id": "builtin.app",
                    "plugin_id": "connector-builtin-app",
                    "display_name": "App",
                    "enabled": true,
                    "configured": true,
                    "status": "degraded",
                    "summary": "App connector degraded in fixture.",
                    "action_readiness": "degraded",
                    "configuration": [
                        "permission_state": "granted",
                        "status_reason": "degraded"
                    ],
                    "remediation_actions": [
                        [
                            "identifier": "refresh_connector_status",
                            "label": "Refresh Connector Status",
                            "intent": "refresh_status",
                            "enabled": true,
                            "recommended": true
                        ]
                    ]
                ],
                [
                    "connector_id": "imessage",
                    "plugin_id": "connector-imessage",
                    "display_name": "iMessage",
                    "enabled": true,
                    "configured": true,
                    "status": "ready",
                    "summary": "iMessage connector ready.",
                    "action_readiness": "ready",
                    "configuration": [
                        "permission_state": "granted"
                    ],
                    "remediation_actions": [
                        [
                            "identifier": "open_connector_system_settings",
                            "label": "Open System Settings",
                            "intent": "open_system_settings",
                            "destination": "ui://system-settings/privacy/full-disk-access",
                            "enabled": true,
                            "recommended": false
                        ]
                    ]
                ],
                [
                    "connector_id": "twilio",
                    "plugin_id": "connector-twilio",
                    "display_name": "Twilio",
                    "enabled": true,
                    "configured": true,
                    "status": "ready",
                    "summary": "Twilio connector ready.",
                    "action_readiness": "ready",
                    "configuration": [
                        "permission_state": "granted"
                    ],
                    "remediation_actions": []
                ]
            ]
        ]
    }

    private static func connectorDiagnosticsDegradedPayload() -> [String: Any] {
        [
            "workspace_id": workspaceID,
            "diagnostics": [
                [
                    "connector_id": "builtin.app",
                    "plugin_id": "connector-builtin-app",
                    "display_name": "App",
                    "configured": true,
                    "status": "degraded",
                    "summary": "Connector worker failed health check in fixture.",
                    "worker_health": [
                        "registered": true,
                        "worker": [
                            "plugin_id": "connector-builtin-app",
                            "kind": "connector",
                            "state": "degraded",
                            "process_id": 7101,
                            "restart_count": 2,
                            "last_error": "fixture worker timeout"
                        ]
                    ],
                    "remediation_actions": [
                        [
                            "identifier": "refresh_connector_status",
                            "label": "Refresh Connector Status",
                            "intent": "refresh_status",
                            "enabled": true,
                            "recommended": true
                        ],
                        [
                            "identifier": "open_inspect_logs",
                            "label": "Open Inspect Logs",
                            "intent": "open_logs",
                            "enabled": true,
                            "recommended": false
                        ]
                    ]
                ],
                [
                    "connector_id": "imessage",
                    "plugin_id": "connector-imessage",
                    "display_name": "iMessage",
                    "configured": true,
                    "status": "ready",
                    "summary": "Connector healthy.",
                    "worker_health": [
                        "registered": true,
                        "worker": [
                            "plugin_id": "connector-imessage",
                            "kind": "connector",
                            "state": "running",
                            "process_id": 7102,
                            "restart_count": 0
                        ]
                    ],
                    "remediation_actions": []
                ],
                [
                    "connector_id": "twilio",
                    "plugin_id": "connector-twilio",
                    "display_name": "Twilio",
                    "configured": true,
                    "status": "ready",
                    "summary": "Connector healthy.",
                    "worker_health": [
                        "registered": true,
                        "worker": [
                            "plugin_id": "connector-twilio",
                            "kind": "connector",
                            "state": "running",
                            "process_id": 7103,
                            "restart_count": 0
                        ]
                    ],
                    "remediation_actions": []
                ]
            ]
        ]
    }

    private static func taskListReadyPayload() -> [String: Any] {
        [
            "workspace_id": workspaceID,
            "items": [
                [
                    "task_id": "task-1",
                    "run_id": "run-1",
                    "workspace_id": workspaceID,
                    "title": "Summarize customer updates",
                    "task_state": "running",
                    "run_state": "running",
                    "priority": 2,
                    "requested_by_actor_id": "owner",
                    "subject_principal_actor_id": "default",
                    "acting_as_actor_id": "default",
                    "last_error": NSNull(),
                    "task_created_at": fiveMinutesAgo,
                    "task_updated_at": oneMinuteAgo,
                    "run_created_at": fiveMinutesAgo,
                    "run_updated_at": oneMinuteAgo,
                    "started_at": fiveMinutesAgo,
                    "finished_at": NSNull(),
                    "actions": [
                        "can_cancel": true,
                        "can_retry": false,
                        "can_requeue": false
                    ],
                    "route": [
                        "available": true,
                        "task_class": "chat",
                        "provider": "openai",
                        "model_key": "gpt-4.1",
                        "task_class_source": "task",
                        "route_source": "policy"
                    ]
                ],
                [
                    "task_id": "task-2",
                    "run_id": "run-2",
                    "workspace_id": workspaceID,
                    "title": "Draft follow-up reply",
                    "task_state": "completed",
                    "run_state": "completed",
                    "priority": 1,
                    "requested_by_actor_id": "owner",
                    "subject_principal_actor_id": "default",
                    "acting_as_actor_id": "default",
                    "last_error": NSNull(),
                    "task_created_at": "2026-02-26T19:30:00Z",
                    "task_updated_at": "2026-02-26T19:45:00Z",
                    "run_created_at": "2026-02-26T19:31:00Z",
                    "run_updated_at": "2026-02-26T19:45:00Z",
                    "started_at": "2026-02-26T19:31:00Z",
                    "finished_at": "2026-02-26T19:45:00Z",
                    "actions": [
                        "can_cancel": false,
                        "can_retry": true,
                        "can_requeue": true
                    ],
                    "route": [
                        "available": true,
                        "task_class": "chat",
                        "provider": "openai",
                        "model_key": "gpt-4.1",
                        "task_class_source": "task",
                        "route_source": "policy"
                    ]
                ]
            ]
        ]
    }

    private static func approvalsReadyPayload() -> [String: Any] {
        [
            "workspace_id": workspaceID,
            "approvals": [
                [
                    "approval_request_id": "approval-1",
                    "workspace_id": workspaceID,
                    "state": "pending",
                    "risk_level": "destructive",
                    "risk_rationale": "Deleting records requires explicit confirmation.",
                    "requested_at": oneMinuteAgo,
                    "task_id": "task-1",
                    "task_title": "Summarize customer updates",
                    "task_state": "running",
                    "run_id": "run-1",
                    "run_state": "awaiting_approval",
                    "step_id": "step-approval-1",
                    "step_name": "Confirm destructive operation",
                    "requested_by_actor_id": "owner",
                    "subject_principal_actor_id": "default",
                    "acting_as_actor_id": "default",
                    "route": [
                        "available": true,
                        "task_class": "chat",
                        "provider": "openai",
                        "model_key": "gpt-4.1",
                        "task_class_source": "task",
                        "route_source": "policy"
                    ]
                ]
            ]
        ]
    }

    private static func inspectRunReadyPayload(runID: String) -> [String: Any] {
        [
            "task": [
                "task_id": "task-1",
                "workspace_id": workspaceID,
                "requested_by_actor_id": "owner",
                "subject_principal_actor_id": "default",
                "title": "Summarize customer updates",
                "description": "Fixture task detail",
                "state": "running",
                "priority": 2,
                "deadline_at": NSNull(),
                "channel": "app",
                "created_at": fiveMinutesAgo,
                "updated_at": oneMinuteAgo
            ],
            "run": [
                "run_id": runID,
                "workspace_id": workspaceID,
                "task_id": "task-1",
                "acting_as_actor_id": "default",
                "state": "running",
                "started_at": fiveMinutesAgo,
                "finished_at": NSNull(),
                "last_error": NSNull(),
                "created_at": fiveMinutesAgo,
                "updated_at": oneMinuteAgo
            ],
            "steps": [
                [
                    "step_id": "step-1",
                    "run_id": runID,
                    "step_index": 1,
                    "name": "Gather context",
                    "status": "completed",
                    "interaction_level": "low",
                    "capability_key": "context.read",
                    "timeout_seconds": 30,
                    "retry_max": 1,
                    "retry_count": 0,
                    "last_error": NSNull(),
                    "created_at": fiveMinutesAgo,
                    "updated_at": twoMinutesAgo
                ],
                [
                    "step_id": "step-2",
                    "run_id": runID,
                    "step_index": 2,
                    "name": "Generate summary",
                    "status": "running",
                    "interaction_level": "low",
                    "capability_key": "chat.turn",
                    "timeout_seconds": 60,
                    "retry_max": 2,
                    "retry_count": 0,
                    "last_error": NSNull(),
                    "created_at": twoMinutesAgo,
                    "updated_at": oneMinuteAgo
                ]
            ],
            "artifacts": [
                [
                    "artifact_id": "artifact-1",
                    "run_id": runID,
                    "step_id": "step-1",
                    "artifact_type": "summary",
                    "uri": "fixture://artifact/summary-1",
                    "content_hash": "sha256:fixture",
                    "created_at": twoMinutesAgo
                ]
            ],
            "audit_entries": [
                [
                    "audit_id": "audit-1",
                    "workspace_id": workspaceID,
                    "run_id": runID,
                    "step_id": "step-2",
                    "event_type": "task_step_progress",
                    "actor_id": "assistant",
                    "acting_as_actor_id": "default",
                    "correlation_id": "corr-fixture-chat",
                    "payload_json": "{\"summary\":\"Generating response\"}",
                    "created_at": oneMinuteAgo
                ]
            ],
            "route": [
                "available": true,
                "task_class": "chat",
                "provider": "openai",
                "model_key": "gpt-4.1",
                "task_class_source": "task",
                "route_source": "policy"
            ]
        ]
    }

    private static func inspectLogsQueryReadyPayload() -> [String: Any] {
        [
            "workspace_id": workspaceID,
            "logs": [
                [
                    "log_id": "log-1",
                    "workspace_id": workspaceID,
                    "run_id": "run-1",
                    "step_id": "step-2",
                    "event_type": "task_step_progress",
                    "status": "running",
                    "input_summary": "Generating response",
                    "output_summary": "In progress",
                    "correlation_id": "corr-fixture-chat",
                    "actor_id": "assistant",
                    "acting_as_actor_id": "default",
                    "created_at": oneMinuteAgo,
                    "metadata": [
                        "task_id": "task-1"
                    ],
                    "route": [
                        "available": true,
                        "task_class": "chat",
                        "provider": "openai",
                        "model_key": "gpt-4.1",
                        "task_class_source": "task",
                        "route_source": "policy"
                    ]
                ]
            ],
            "next_cursor_created_at": oneMinuteAgo,
            "next_cursor_id": "log-1"
        ]
    }

    private static func inspectLogsStreamReadyPayload() -> [String: Any] {
        [
            "workspace_id": workspaceID,
            "logs": [],
            "cursor_created_at": oneMinuteAgo,
            "cursor_id": "log-1",
            "timed_out": true
        ]
    }

    private static func chatTurnReadyPayload(
        correlationID: String,
        latestUserMessage: String?
    ) -> [String: Any] {
        let prompt = latestUserMessage?
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .lowercased() ?? ""

        let actionKeywords = ["email", "text", "sms", "message", "file", "find", "browse", "web", "website"]
        let isActionPrompt = actionKeywords.contains(where: { prompt.contains($0) })
        guard isActionPrompt else {
            return [
                "workspace_id": workspaceID,
                "task_class": "chat",
                "provider": "openai",
                "model_key": "gpt-4.1",
                "correlation_id": correlationID,
                "contract_version": "chat_turn.v2",
                "turn_item_schema_version": "chat_turn_item.v2",
                "realtime_event_contract_version": "chat_realtime_lifecycle.v2",
                "items": [
                    [
                        "item_id": "item-assistant-fixture",
                        "type": "assistant_message",
                        "role": "assistant",
                        "status": "completed",
                        "content": "Fixture response: smoke chat completed."
                    ]
                ],
                "task_run_correlation": [
                    "available": true,
                    "source": "task_run",
                    "task_id": "task-1",
                    "run_id": "run-1",
                    "task_state": "running",
                    "run_state": "running"
                ]
            ]
        }

        let workflow: String
        let taskState: String
        let runState: String
        let assistantMessage: String
        let approvalRequired: Bool
        let approvalRequestID: String?

        switch true {
        case prompt.contains("email"):
            workflow = "send_email"
            taskState = "awaiting_approval"
            runState = "awaiting_approval"
            assistantMessage = "send_email is waiting for approval (request approval-fixture-email)."
            approvalRequired = true
            approvalRequestID = "approval-fixture-email"
        case prompt.contains("text"), prompt.contains("sms"), prompt.contains("message"):
            workflow = "send_message"
            taskState = "completed"
            runState = "completed"
            assistantMessage = "send_message completed successfully."
            approvalRequired = false
            approvalRequestID = nil
        case prompt.contains("file"), prompt.contains("find"):
            workflow = "find_files"
            taskState = "blocked"
            runState = "blocked"
            assistantMessage = "find_files is blocked: connector permission is missing."
            approvalRequired = false
            approvalRequestID = nil
        case prompt.contains("browse"), prompt.contains("web"), prompt.contains("website"):
            workflow = "browse_web"
            taskState = "completed"
            runState = "completed"
            assistantMessage = "browse_web completed successfully."
            approvalRequired = false
            approvalRequestID = nil
        default:
            workflow = "chat_action"
            taskState = "running"
            runState = "running"
            assistantMessage = "Action request submitted."
            approvalRequired = false
            approvalRequestID = nil
        }

        let taskID = "task-\(workflow)"
        let runID = "run-\(workflow)"

        var items: [[String: Any]] = [
            [
                "item_id": "item-tool-call-\(workflow)",
                "type": "tool_call",
                "status": "started",
                "tool_name": workflow,
                "tool_call_id": "tool-\(workflow)",
                "arguments": [
                    "prompt": latestUserMessage ?? ""
                ]
            ],
            [
                "item_id": "item-tool-result-\(workflow)",
                "type": "tool_result",
                "status": approvalRequired ? "awaiting_approval" : runState,
                "tool_name": workflow,
                "tool_call_id": "tool-\(workflow)",
                "output": [
                    "workflow": workflow,
                    "task_id": taskID,
                    "run_id": runID,
                    "task_state": taskState,
                    "run_state": runState,
                    "approval_required": approvalRequired
                ],
                "approval_request_id": approvalRequestID as Any
            ]
        ]

        if approvalRequired {
            items.append(
                [
                    "item_id": "item-approval-\(workflow)",
                    "type": "approval_request",
                    "status": "awaiting_approval",
                    "approval_request_id": approvalRequestID as Any,
                    "content": "Approval is required before execution can continue."
                ]
            )
        }

        items.append(
            [
                "item_id": "item-assistant-\(workflow)",
                "type": "assistant_message",
                "role": "assistant",
                "status": "completed",
                "content": assistantMessage
            ]
        )

        return [
            "workspace_id": workspaceID,
            "task_class": "chat",
            "provider": "openai",
            "model_key": "gpt-4.1",
            "correlation_id": correlationID,
            "contract_version": "chat_turn.v2",
            "turn_item_schema_version": "chat_turn_item.v2",
            "realtime_event_contract_version": "chat_realtime_lifecycle.v2",
            "items": items,
            "task_run_correlation": [
                "available": true,
                "source": "agent_run",
                "task_id": taskID,
                "run_id": runID,
                "task_state": taskState,
                "run_state": runState
            ]
        ]
    }

    private static func chatTurnExplainReadyPayload(body: Data?) -> [String: Any] {
        let requestedByActorID = requestStringValue(key: "requested_by_actor_id", from: body)
        let subjectActorID = requestStringValue(key: "subject_actor_id", from: body)
        let actingAsActorID = requestStringValue(key: "acting_as_actor_id", from: body)

        var payload: [String: Any] = [
            "workspace_id": workspaceID,
            "task_class": "chat",
            "channel": [
                "channel_id": "app"
            ],
            "contract_version": "chat_turn_explain.v1",
            "selected_route": [
                "workspace_id": workspaceID,
                "task_class": "chat",
                "selected_provider": "openai",
                "selected_model_key": "gpt-4.1",
                "selected_source": "task_class_policy",
                "summary": "Selected route uses workspace chat policy with ready provider health.",
                "explanations": [
                    "Workspace policy explicitly sets the chat route.",
                    "Provider and model readiness checks passed."
                ],
                "reason_codes": [
                    "policy_match",
                    "provider_ready"
                ],
                "decisions": [
                    [
                        "step": "policy_lookup",
                        "decision": "selected",
                        "reason_code": "policy_match",
                        "provider": "openai",
                        "model_key": "gpt-4.1",
                        "note": "chat task class policy matched."
                    ]
                ],
                "fallback_chain": []
            ],
            "tool_catalog": [
                [
                    "name": "send_email",
                    "description": "Draft and send mail through configured connectors.",
                    "capability_keys": ["connector.mail.send"],
                    "input_schema": [
                        "type": "object",
                        "required": ["to", "subject", "body"]
                    ]
                ],
                [
                    "name": "find_files",
                    "description": "Search local files using connector-provided capabilities.",
                    "capability_keys": ["connector.files.search"],
                    "input_schema": [
                        "type": "object",
                        "required": ["query"]
                    ]
                ]
            ],
            "policy_decisions": [
                [
                    "tool_name": "send_email",
                    "capability_key": "connector.mail.send",
                    "decision": "allow",
                    "reason": "Mail connector is enabled and permission-ready."
                ],
                [
                    "tool_name": "find_files",
                    "capability_key": "connector.files.search",
                    "decision": "allow_with_approval",
                    "reason": "File access is allowed but requires user approval for protected locations."
                ]
            ]
        ]
        if let requestedByActorID {
            payload["requested_by_actor_id"] = requestedByActorID
        }
        if let subjectActorID {
            payload["subject_actor_id"] = subjectActorID
        }
        if let actingAsActorID {
            payload["acting_as_actor_id"] = actingAsActorID
        }
        return payload
    }

    private static func chatTurnHistoryReadyPayload() -> [String: Any] {
        [
            "workspace_id": workspaceID,
            "items": [
                [
                    "record_id": "history-voice-1",
                    "turn_id": "turn-voice-1",
                    "workspace_id": workspaceID,
                    "task_class": "chat",
                    "correlation_id": "corr-cont-voice",
                    "channel_id": "voice",
                    "connector_id": "twilio",
                    "thread_id": "thread-voice-1",
                    "item_index": 0,
                    "item": [
                        "item_id": "item-tool-result-voice",
                        "type": "tool_result",
                        "status": "failed",
                        "tool_name": "start_call",
                        "tool_call_id": "tool-start-call",
                        "error_message": "Voice connector is offline."
                    ],
                    "task_run_reference": [
                        "available": true,
                        "source": "task_run",
                        "task_id": "task-voice-1",
                        "run_id": "run-voice-1",
                        "task_state": "failed",
                        "run_state": "failed"
                    ],
                    "created_at": now
                ],
                [
                    "record_id": "history-message-1",
                    "turn_id": "turn-message-1",
                    "workspace_id": workspaceID,
                    "task_class": "chat",
                    "correlation_id": "corr-cont-message",
                    "channel_id": "message",
                    "connector_id": "twilio",
                    "thread_id": "thread-message-1",
                    "item_index": 0,
                    "item": [
                        "item_id": "item-user-message",
                        "type": "user_message",
                        "role": "user",
                        "status": "completed",
                        "content": "Text Alex that the release is complete."
                    ],
                    "task_run_reference": [
                        "available": true,
                        "source": "task_run",
                        "task_id": "task-message-1",
                        "run_id": "run-message-1",
                        "task_state": "completed",
                        "run_state": "completed"
                    ],
                    "created_at": oneMinuteAgo
                ],
                [
                    "record_id": "history-message-2",
                    "turn_id": "turn-message-1",
                    "workspace_id": workspaceID,
                    "task_class": "chat",
                    "correlation_id": "corr-cont-message",
                    "channel_id": "message",
                    "connector_id": "twilio",
                    "thread_id": "thread-message-1",
                    "item_index": 1,
                    "item": [
                        "item_id": "item-assistant-message",
                        "type": "assistant_message",
                        "role": "assistant",
                        "status": "completed",
                        "content": "Sent a text update to Alex."
                    ],
                    "task_run_reference": [
                        "available": true,
                        "source": "task_run",
                        "task_id": "task-message-1",
                        "run_id": "run-message-1",
                        "task_state": "completed",
                        "run_state": "completed"
                    ],
                    "created_at": oneMinuteAgo
                ],
                [
                    "record_id": "history-app-1",
                    "turn_id": "turn-app-1",
                    "workspace_id": workspaceID,
                    "task_class": "chat",
                    "correlation_id": "corr-cont-app",
                    "channel_id": "app",
                    "connector_id": "builtin.app",
                    "thread_id": "thread-app-1",
                    "item_index": 0,
                    "item": [
                        "item_id": "item-approval-request",
                        "type": "approval_request",
                        "status": "awaiting_approval",
                        "approval_request_id": "approval-cont-1",
                        "content": "Approval is required before execution can continue."
                    ],
                    "task_run_reference": [
                        "available": true,
                        "source": "task_run",
                        "task_id": "task-app-1",
                        "run_id": "run-app-1",
                        "task_state": "awaiting_approval",
                        "run_state": "awaiting_approval"
                    ],
                    "created_at": twoMinutesAgo
                ]
            ],
            "has_more": false,
            "next_cursor_created_at": "",
            "next_cursor_item_id": ""
        ]
    }

    private static func taskSubmitReadyPayload(body: Data?) -> [String: Any] {
        let requestedTaskClass = requestStringValue(key: "task_class", from: body) ?? "chat"
        let correlationID = requestHeaderCorrelationFallback(from: body) ?? "corr-fixture-task-submit"

        return [
            "task_id": "task-fixture-submit",
            "run_id": "run-fixture-submit",
            "state": "queued",
            "task_class": requestedTaskClass,
            "correlation_id": correlationID
        ]
    }

    private static func approvalDecisionReadyPayload(path: String, body: Data?) -> [String: Any] {
        let approvalID = path
            .replacingOccurrences(of: "/v1/approvals/", with: "")
            .trimmingCharacters(in: .whitespacesAndNewlines)
        let normalizedApprovalID = approvalID.isEmpty ? "approval-1" : approvalID
        let phrase = requestStringValue(key: "decision_phrase", from: body) ?? "REJECT"
        let normalizedPhrase = phrase.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        let accepted = normalizedPhrase == "go ahead"
        let decision: String = accepted ? "approved" : "rejected"
        let correlationID = requestHeaderCorrelationFallback(from: body) ?? "corr-fixture-approval-decision"

        return [
            "approval_id": normalizedApprovalID,
            "decision": decision,
            "accepted": accepted,
            "correlation_id": correlationID
        ]
    }

    private static func requestStringValue(key: String, from body: Data?) -> String? {
        guard let body,
              let payload = try? JSONSerialization.jsonObject(with: body) as? [String: Any],
              let value = payload[key] as? String else {
            return nil
        }
        let trimmedValue = value.trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmedValue.isEmpty ? nil : trimmedValue
    }

    private static func requestHeaderCorrelationFallback(from body: Data?) -> String? {
        requestStringValue(key: "correlation_id", from: body)
    }

    private static func requestLatestUserMessage(from body: Data?) -> String? {
        guard let body,
              let payload = try? JSONSerialization.jsonObject(with: body) as? [String: Any],
              let items = payload["items"] as? [[String: Any]] else {
            return nil
        }

        for item in items.reversed() {
            guard let itemType = item["type"] as? String,
                  itemType.trimmingCharacters(in: .whitespacesAndNewlines).lowercased() == "user_message",
                  let content = item["content"] as? String else {
                continue
            }
            let trimmed = content.trimmingCharacters(in: .whitespacesAndNewlines)
            if !trimmed.isEmpty {
                return trimmed
            }
        }
        return nil
    }
}
