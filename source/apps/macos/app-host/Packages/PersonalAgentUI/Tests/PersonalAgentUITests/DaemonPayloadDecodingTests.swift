import XCTest
@testable import PersonalAgentUI

final class DaemonPayloadDecodingTests: XCTestCase {
    private let decoder = JSONDecoder()

    func testChannelStatusResponseDecodesSnakeCasePayload() throws {
        let payload = """
        {
          "workspace_id": "ws-alpha",
          "channels": [
            {
              "channel_id": "app_chat",
              "display_name": "App Chat",
              "category": "chat",
              "enabled": true,
              "configured": true,
              "status": "ready",
              "summary": "Healthy",
              "configuration": {
                "mode": "daemon"
              },
              "config_field_descriptors": [
                {
                  "key": "mode",
                  "label": "Mode",
                  "type": "string",
                  "required": true,
                  "enum_options": ["daemon", "local"],
                  "editable": true,
                  "secret": false,
                  "write_only": false,
                  "help_text": "Channel dispatch mode."
                }
              ],
              "capabilities": ["send", "status"],
              "action_readiness": "blocked",
              "action_blockers": [
                {
                  "code": "config_incomplete",
                  "message": "Channel setup is incomplete.",
                  "remediation_action": "configure_channel"
                }
              ],
              "worker": {
                "plugin_id": "channel-app-chat",
                "kind": "channel",
                "state": "running",
                "process_id": 9123,
                "restart_count": 2,
                "last_error": "",
                "last_heartbeat": "2026-02-24T18:00:00Z",
                "last_transition": "2026-02-24T17:59:00Z"
              }
            }
          ]
        }
        """

        let decoded = try decoder.decode(
            DaemonChannelStatusResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws-alpha")
        XCTAssertEqual(decoded.channels.count, 1)
        XCTAssertEqual(decoded.channels.first?.channelID, "app_chat")
        XCTAssertEqual(decoded.channels.first?.displayName, "App Chat")
        XCTAssertEqual(decoded.channels.first?.configFieldDescriptors.count, 1)
        XCTAssertEqual(decoded.channels.first?.configFieldDescriptors.first?.key, "mode")
        XCTAssertEqual(decoded.channels.first?.configFieldDescriptors.first?.enumOptions, ["daemon", "local"])
        XCTAssertEqual(decoded.channels.first?.actionReadiness, "blocked")
        XCTAssertEqual(decoded.channels.first?.actionBlockers.first?.code, "config_incomplete")
        XCTAssertEqual(decoded.channels.first?.actionBlockers.first?.remediationAction, "configure_channel")
        XCTAssertEqual(decoded.channels.first?.worker?.pluginID, "channel-app-chat")
        XCTAssertEqual(decoded.channels.first?.worker?.processID, 9123)
    }

    func testConnectorStatusResponseDecodesSnakeCasePayload() throws {
        let payload = """
        {
          "workspace_id": "ws-beta",
          "connectors": [
            {
              "connector_id": "mail",
              "plugin_id": "connector-mail",
              "display_name": "Mail",
              "enabled": true,
              "configured": false,
              "status": "degraded",
              "summary": "Permission missing",
              "configuration": {
                "account": "primary",
                "status_reason": "permission_missing",
                "permission_state": "missing"
              },
              "config_field_descriptors": [
                {
                  "key": "auth_token",
                  "label": "Auth Token",
                  "type": "string",
                  "required": false,
                  "editable": true,
                  "secret": true,
                  "write_only": true
                }
              ],
              "capabilities": ["draft", "send"],
              "action_readiness": "blocked",
              "action_blockers": [
                {
                  "code": "permission_missing",
                  "message": "Connector permission is missing.",
                  "remediation_action": "open_connector_system_settings"
                }
              ],
              "remediation_actions": [
                {
                  "identifier": "open_connector_system_settings",
                  "label": "Open System Settings",
                  "intent": "open_system_settings",
                  "destination": "ui://system-settings/privacy/automation",
                  "parameters": {
                    "connector_id": "mail"
                  },
                  "enabled": true,
                  "recommended": true
                }
              ],
              "worker": {
                "plugin_id": "connector-mail",
                "kind": "connector",
                "state": "running",
                "process_id": 0,
                "restart_count": 0,
                "last_error": null,
                "last_heartbeat": null,
                "last_transition": null
              }
            }
          ]
        }
        """

        let decoded = try decoder.decode(
            DaemonConnectorStatusResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws-beta")
        XCTAssertEqual(decoded.connectors.count, 1)
        XCTAssertEqual(decoded.connectors.first?.connectorID, "mail")
        XCTAssertEqual(decoded.connectors.first?.pluginID, "connector-mail")
        XCTAssertEqual(decoded.connectors.first?.status, "degraded")
        XCTAssertEqual(decoded.connectors.first?.actionReadiness, "blocked")
        XCTAssertEqual(decoded.connectors.first?.actionBlockers.first?.code, "permission_missing")
        XCTAssertEqual(decoded.connectors.first?.configFieldDescriptors.count, 1)
        XCTAssertEqual(decoded.connectors.first?.configFieldDescriptors.first?.key, "auth_token")
        XCTAssertTrue(decoded.connectors.first?.configFieldDescriptors.first?.secret ?? false)
        XCTAssertTrue(decoded.connectors.first?.configFieldDescriptors.first?.writeOnly ?? false)
        XCTAssertEqual(
            decoded.connectors.first?.configuration?["status_reason"]?.stringValue,
            "permission_missing"
        )
        XCTAssertEqual(
            decoded.connectors.first?.configuration?["permission_state"]?.stringValue,
            "missing"
        )
        XCTAssertEqual(
            decoded.connectors.first?.remediationActions?.first?.identifier,
            "open_connector_system_settings"
        )
        XCTAssertEqual(
            decoded.connectors.first?.remediationActions?.first?.intent,
            "open_system_settings"
        )
    }

    func testTaskRunListResponseDecodesActionAvailabilityMetadata() throws {
        let payload = """
        {
          "workspace_id": "ws-task",
          "items": [
            {
              "task_id": "task-1",
              "run_id": "run-1",
              "workspace_id": "ws-task",
              "title": "Refresh inbox",
              "task_state": "running",
              "run_state": "running",
              "priority": 2,
              "requested_by_actor_id": "owner",
              "subject_principal_actor_id": "default",
              "acting_as_actor_id": "default",
              "last_error": "",
              "task_created_at": "2026-02-26T10:00:00Z",
              "task_updated_at": "2026-02-26T10:01:00Z",
              "run_created_at": "2026-02-26T10:00:15Z",
              "run_updated_at": "2026-02-26T10:01:00Z",
              "started_at": "2026-02-26T10:00:20Z",
              "finished_at": null,
              "actions": {
                "can_cancel": true
              }
            },
            {
              "task_id": "task-2",
              "run_id": "run-2",
              "workspace_id": "ws-task",
              "title": "Draft summary",
              "task_state": "completed",
              "run_state": "completed",
              "priority": 1,
              "requested_by_actor_id": "owner",
              "subject_principal_actor_id": "default",
              "acting_as_actor_id": "default",
              "last_error": null,
              "task_created_at": "2026-02-26T09:00:00Z",
              "task_updated_at": "2026-02-26T09:03:00Z",
              "run_created_at": "2026-02-26T09:00:05Z",
              "run_updated_at": "2026-02-26T09:03:00Z",
              "started_at": "2026-02-26T09:00:10Z",
              "finished_at": "2026-02-26T09:03:00Z"
            }
          ]
        }
        """

        let decoded = try decoder.decode(
            DaemonTaskRunListResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws-task")
        XCTAssertEqual(decoded.items.count, 2)
        XCTAssertEqual(decoded.items.first?.actions?.canCancel, true)
        XCTAssertEqual(decoded.items.first?.actions?.canRetry, false)
        XCTAssertEqual(decoded.items.first?.actions?.canRequeue, false)
        XCTAssertNil(decoded.items.last?.actions)
    }

    func testTaskRetryResponseMissingActionsDecodesDeterministicDefaults() throws {
        let payload = """
        {
          "workspace_id": "ws-task",
          "task_id": "task-1",
          "previous_run_id": "run-old",
          "run_id": "run-new",
          "previous_task_state": "failed",
          "previous_run_state": "failed",
          "task_state": "queued",
          "run_state": "queued",
          "retried": true
        }
        """

        let decoded = try decoder.decode(
            DaemonTaskRetryResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws-task")
        XCTAssertEqual(decoded.taskID, "task-1")
        XCTAssertEqual(decoded.previousRunID, "run-old")
        XCTAssertEqual(decoded.runID, "run-new")
        XCTAssertEqual(decoded.actions.canCancel, false)
        XCTAssertEqual(decoded.actions.canRetry, false)
        XCTAssertEqual(decoded.actions.canRequeue, false)
    }

    func testChannelDiagnosticsResponseDecodesRemediationActions() throws {
        let payload = """
        {
          "workspace_id": "ws-diag",
          "diagnostics": [
            {
              "channel_id": "app_chat",
              "display_name": "App Chat",
              "category": "chat",
              "configured": true,
              "status": "ready",
              "summary": "Healthy",
              "worker_health": {
                "registered": true,
                "worker": {
                  "plugin_id": "channel-app-chat",
                  "kind": "channel",
                  "state": "running",
                  "process_id": 1001,
                  "restart_count": 0
                }
              },
              "remediation_actions": [
                {
                  "action_id": "refresh_channel_status",
                  "title": "Refresh Channel Status",
                  "target": "/v1/channels/status",
                  "enabled": true,
                  "recommended": false
                }
              ]
            }
          ]
        }
        """

        let decoded = try decoder.decode(
            DaemonChannelDiagnosticsResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws-diag")
        XCTAssertEqual(decoded.diagnostics.count, 1)
        XCTAssertEqual(decoded.diagnostics.first?.channelID, "app_chat")
        XCTAssertEqual(decoded.diagnostics.first?.remediationActions.first?.identifier, "refresh_channel_status")
        XCTAssertEqual(decoded.diagnostics.first?.remediationActions.first?.label, "Refresh Channel Status")
        XCTAssertEqual(decoded.diagnostics.first?.remediationActions.first?.intent, "refresh_status")
        XCTAssertEqual(decoded.diagnostics.first?.remediationActions.first?.destination, "/v1/channels/status")
    }

    func testConnectorDiagnosticsResponseDecodesRemediationActions() throws {
        let payload = """
        {
          "workspace_id": "ws-diag",
          "diagnostics": [
            {
              "connector_id": "mail",
              "plugin_id": "mail.daemon",
              "display_name": "Mail",
              "configured": true,
              "status": "ready",
              "summary": "Healthy",
              "worker_health": {
                "registered": true,
                "worker": {
                  "plugin_id": "mail.daemon",
                  "kind": "connector",
                  "state": "running",
                  "process_id": 1002,
                  "restart_count": 1
                }
              },
              "remediation_actions": [
                {
                  "action_id": "open_connector_logs",
                  "title": "Open Inspect Logs",
                  "target": "ui://inspect/logs?scope=connector:mail",
                  "enabled": true,
                  "recommended": true
                }
              ]
            }
          ]
        }
        """

        let decoded = try decoder.decode(
            DaemonConnectorDiagnosticsResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws-diag")
        XCTAssertEqual(decoded.diagnostics.count, 1)
        XCTAssertEqual(decoded.diagnostics.first?.connectorID, "mail")
        XCTAssertEqual(decoded.diagnostics.first?.remediationActions.first?.identifier, "open_connector_logs")
        XCTAssertEqual(decoded.diagnostics.first?.remediationActions.first?.label, "Open Inspect Logs")
        XCTAssertEqual(decoded.diagnostics.first?.remediationActions.first?.intent, "navigate")
        XCTAssertEqual(
            decoded.diagnostics.first?.remediationActions.first?.destination,
            "ui://inspect/logs?scope=connector:mail"
        )
    }

    func testConnectorPermissionResponseDecodesSnakeCasePayload() throws {
        let payload = """
        {
          "workspace_id": "ws-permissions",
          "connector_id": "mail",
          "permission_state": "missing",
          "message": "Mail automation access is not granted."
        }
        """

        let decoded = try decoder.decode(
            DaemonConnectorPermissionResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws-permissions")
        XCTAssertEqual(decoded.connectorID, "mail")
        XCTAssertEqual(decoded.permissionState, "missing")
        XCTAssertEqual(decoded.message, "Mail automation access is not granted.")
    }

    func testChannelConfigUpsertResponseDecodesSnakeCasePayload() throws {
        let payload = """
        {
          "workspace_id": "ws-config",
          "channel_id": "app_chat",
          "configuration": {
            "enabled": true,
            "transport": "daemon_realtime",
            "retry_count": 2
          },
          "updated_at": "2026-02-25T10:30:00Z"
        }
        """

        let decoded = try decoder.decode(
            DaemonChannelConfigUpsertResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws-config")
        XCTAssertEqual(decoded.channelID, "app_chat")
        XCTAssertEqual(decoded.configuration["enabled"]?.displayText, "true")
        XCTAssertEqual(decoded.configuration["transport"]?.displayText, "daemon_realtime")
        XCTAssertEqual(decoded.configuration["retry_count"]?.displayText, "2")
        XCTAssertEqual(decoded.updatedAt, "2026-02-25T10:30:00Z")
    }

    func testConnectorConfigUpsertResponseDecodesSnakeCasePayload() throws {
        let payload = """
        {
          "workspace_id": "ws-config",
          "connector_id": "mail",
          "configuration": {
            "scope": "inbox",
            "enabled": false
          },
          "updated_at": "2026-02-25T10:31:00Z"
        }
        """

        let decoded = try decoder.decode(
            DaemonConnectorConfigUpsertResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws-config")
        XCTAssertEqual(decoded.connectorID, "mail")
        XCTAssertEqual(decoded.configuration["scope"]?.displayText, "inbox")
        XCTAssertEqual(decoded.configuration["enabled"]?.displayText, "false")
        XCTAssertEqual(decoded.updatedAt, "2026-02-25T10:31:00Z")
    }

    func testChannelTestOperationResponseDecodesSnakeCasePayload() throws {
        let payload = """
        {
          "workspace_id": "ws-config",
          "channel_id": "app_chat",
          "operation": "health",
          "success": true,
          "status": "ok",
          "summary": "app_chat channel worker is healthy.",
          "checked_at": "2026-02-25T10:32:00Z",
          "details": {
            "plugin_id": "app_chat.daemon",
            "worker_registered": true
          }
        }
        """

        let decoded = try decoder.decode(
            DaemonChannelTestOperationResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws-config")
        XCTAssertEqual(decoded.channelID, "app_chat")
        XCTAssertEqual(decoded.operation, "health")
        XCTAssertTrue(decoded.success)
        XCTAssertEqual(decoded.status, "ok")
        XCTAssertEqual(decoded.details?.pluginID, "app_chat.daemon")
        XCTAssertEqual(decoded.details?.workerRegistered, true)
    }

    func testConnectorTestOperationResponseDecodesSnakeCasePayload() throws {
        let payload = """
        {
          "workspace_id": "ws-config",
          "connector_id": "mail",
          "operation": "health",
          "success": false,
          "status": "degraded",
          "summary": "mail connector worker is starting.",
          "checked_at": "2026-02-25T10:33:00Z",
          "details": {
            "plugin_id": "mail.daemon",
            "worker_state": "starting"
          }
        }
        """

        let decoded = try decoder.decode(
            DaemonConnectorTestOperationResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws-config")
        XCTAssertEqual(decoded.connectorID, "mail")
        XCTAssertEqual(decoded.operation, "health")
        XCTAssertFalse(decoded.success)
        XCTAssertEqual(decoded.status, "degraded")
        XCTAssertEqual(decoded.details?.pluginID, "mail.daemon")
        XCTAssertEqual(decoded.details?.workerState, "starting")
    }

    func testConnectorStatusResponseCloudflaredActionContractDecodesWithoutPermissionRequest() throws {
        let payload = """
        {
          "workspace_id": "ws-cloudflared",
          "connectors": [
            {
              "connector_id": "cloudflared",
              "plugin_id": "cloudflared.daemon",
              "display_name": "Cloudflared Connector",
              "enabled": true,
              "configured": false,
              "status": "degraded",
              "summary": "Cloudflared binary is unavailable.",
              "configuration": {
                "status_reason": "cloudflared_binary_missing"
              },
              "remediation_actions": [
                {
                  "identifier": "install_cloudflared_connector",
                  "label": "Open Cloudflared Setup",
                  "intent": "navigate",
                  "destination": "ui://configuration/connectors/cloudflared",
                  "enabled": true,
                  "recommended": true
                },
                {
                  "identifier": "open_connector_system_settings",
                  "label": "Open System Settings",
                  "intent": "open_system_settings",
                  "destination": "ui://system-settings/privacy",
                  "enabled": true,
                  "recommended": false
                }
              ]
            }
          ]
        }
        """

        let decoded = try decoder.decode(
            DaemonConnectorStatusResponse.self,
            from: Data(payload.utf8)
        )

        let card = try XCTUnwrap(decoded.connectors.first)
        let actionIDs = card.remediationActions?.map(\.identifier) ?? []
        XCTAssertEqual(card.configuration?.statusReason, "cloudflared_binary_missing")
        XCTAssertTrue(actionIDs.contains("install_cloudflared_connector"))
        XCTAssertFalse(actionIDs.contains("request_connector_permission"))
    }

    func testChannelDiagnosticsResponseDecodesTypedRemediationActionFields() throws {
        let payload = """
        {
          "workspace_id": "ws-diag",
          "diagnostics": [
            {
              "channel_id": "twilio_sms",
              "display_name": "Twilio SMS",
              "category": "sms",
              "configured": false,
              "status": "not_configured",
              "summary": "Missing Twilio credentials",
              "worker_health": {
                "registered": false
              },
              "remediation_actions": [
                {
                  "identifier": "configure_twilio_channel",
                  "label": "Configure Twilio Credentials",
                  "intent": "navigate",
                  "destination": "ui://configuration/channels/twilio",
                  "parameters": {
                    "channel_family": "twilio"
                  },
                  "enabled": true,
                  "recommended": true
                }
              ]
            }
          ]
        }
        """

        let decoded = try decoder.decode(
            DaemonChannelDiagnosticsResponse.self,
            from: Data(payload.utf8)
        )

        let action = try XCTUnwrap(decoded.diagnostics.first?.remediationActions.first)
        XCTAssertEqual(action.identifier, "configure_twilio_channel")
        XCTAssertEqual(action.label, "Configure Twilio Credentials")
        XCTAssertEqual(action.intent, "navigate")
        XCTAssertEqual(action.destination, "ui://configuration/channels/twilio")
        XCTAssertEqual(action.parameters["channel_family"], "twilio")
        XCTAssertTrue(action.enabled)
        XCTAssertTrue(action.recommended)
    }

    func testInspectLogQueryDecodesSnakeCasePayload() throws {
        let payload = """
        {
          "workspace_id": "ws-gamma",
          "logs": [
            {
              "log_id": "log-1",
              "workspace_id": "ws-gamma",
              "run_id": "run-1",
              "step_id": "step-1",
              "event_type": "connector.execute",
              "status": "success",
              "input_summary": "in",
              "output_summary": "out",
              "correlation_id": "corr-1",
              "actor_id": "actor-1",
              "acting_as_actor_id": "actor-2",
              "created_at": "2026-02-24T18:00:00Z",
              "metadata": {
                "duration_ms": 42
              },
              "route": {
                "available": true,
                "task_class": "chat",
                "provider": "openai",
                "model_key": "gpt-5-codex",
                "task_class_source": "policy",
                "route_source": "explicit"
              }
            }
          ],
          "next_cursor_created_at": "2026-02-24T18:00:00Z",
          "next_cursor_id": "log-1"
        }
        """

        let decoded = try decoder.decode(
            DaemonInspectLogQueryResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws-gamma")
        XCTAssertEqual(decoded.logs.count, 1)
        XCTAssertEqual(decoded.logs.first?.logID, "log-1")
        XCTAssertEqual(decoded.logs.first?.eventType, "connector.execute")
        XCTAssertEqual(decoded.logs.first?.route.available, true)
        XCTAssertEqual(decoded.logs.first?.route.taskClass, "chat")
        XCTAssertEqual(decoded.logs.first?.route.provider, "openai")
        XCTAssertEqual(decoded.logs.first?.route.modelKey, "gpt-5-codex")
        XCTAssertEqual(decoded.nextCursorID, "log-1")
    }

    func testInspectLogQueryDecodesWhenRouteMetadataIsOmitted() throws {
        let payload = """
        {
          "workspace_id": "ws-legacy",
          "logs": [
            {
              "log_id": "log-legacy",
              "workspace_id": "ws-legacy",
              "event_type": "task.step",
              "status": "running",
              "input_summary": "in",
              "output_summary": "out",
              "created_at": "2026-02-24T18:00:00Z"
            }
          ]
        }
        """

        let decoded = try decoder.decode(
            DaemonInspectLogQueryResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.logs.count, 1)
        XCTAssertEqual(decoded.logs.first?.route.available, false)
        XCTAssertEqual(decoded.logs.first?.route.taskClass, "")
    }

    func testTaskRunListResponseDecodesRouteMetadata() throws {
        let payload = """
        {
          "workspace_id": "ws-tasks",
          "items": [
            {
              "task_id": "task-1",
              "run_id": "run-1",
              "workspace_id": "ws-tasks",
              "title": "Follow up",
              "task_state": "running",
              "run_state": "running",
              "priority": 2,
              "requested_by_actor_id": "owner",
              "subject_principal_actor_id": "default",
              "acting_as_actor_id": "default",
              "task_created_at": "2026-02-25T10:00:00Z",
              "task_updated_at": "2026-02-25T10:05:00Z",
              "run_created_at": "2026-02-25T10:01:00Z",
              "run_updated_at": "2026-02-25T10:05:00Z",
              "route": {
                "available": true,
                "task_class": "chat",
                "provider": "openai",
                "model_key": "gpt-5-codex",
                "task_class_source": "policy",
                "route_source": "explicit"
              }
            }
          ]
        }
        """

        let decoded = try decoder.decode(
            DaemonTaskRunListResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws-tasks")
        XCTAssertEqual(decoded.items.count, 1)
        XCTAssertEqual(decoded.items.first?.taskID, "task-1")
        XCTAssertEqual(decoded.items.first?.route?.available, true)
        XCTAssertEqual(decoded.items.first?.route?.provider, "openai")
        XCTAssertEqual(decoded.items.first?.route?.modelKey, "gpt-5-codex")
    }

    func testInspectRunResponseDecodesRouteMetadata() throws {
        let payload = """
        {
          "task": {
            "task_id": "task-1",
            "workspace_id": "ws1",
            "requested_by_actor_id": "owner",
            "subject_principal_actor_id": "default",
            "title": "Sample",
            "state": "running",
            "priority": 2,
            "created_at": "2026-02-25T10:00:00Z",
            "updated_at": "2026-02-25T10:05:00Z"
          },
          "run": {
            "run_id": "run-1",
            "workspace_id": "ws1",
            "task_id": "task-1",
            "acting_as_actor_id": "default",
            "state": "running",
            "created_at": "2026-02-25T10:01:00Z",
            "updated_at": "2026-02-25T10:05:00Z"
          },
          "steps": [],
          "artifacts": [],
          "audit_entries": [],
          "route": {
            "available": true,
            "task_class": "chat",
            "provider": "openai",
            "model_key": "gpt-5-codex",
            "task_class_source": "policy",
            "route_source": "explicit"
          }
        }
        """

        let decoded = try decoder.decode(
            DaemonInspectRunResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.run.runID, "run-1")
        XCTAssertEqual(decoded.route?.available, true)
        XCTAssertEqual(decoded.route?.taskClass, "chat")
        XCTAssertEqual(decoded.route?.provider, "openai")
    }

    func testApprovalInboxResponseDecodesRouteMetadata() throws {
        let payload = """
        {
          "workspace_id": "ws-approvals",
          "approvals": [
            {
              "approval_request_id": "approval-1",
              "workspace_id": "ws-approvals",
              "state": "pending",
              "risk_level": "destructive",
              "risk_rationale": "needs approval",
              "requested_at": "2026-02-25T10:00:00Z",
              "task_id": "task-1",
              "run_id": "run-1",
              "requested_by_actor_id": "owner",
              "subject_principal_actor_id": "default",
              "acting_as_actor_id": "default",
              "route": {
                "available": true,
                "task_class": "finder",
                "provider": "ollama",
                "model_key": "llama3.2",
                "task_class_source": "step_capability",
                "route_source": "fallback_enabled"
              }
            }
          ]
        }
        """

        let decoded = try decoder.decode(
            DaemonApprovalInboxResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws-approvals")
        XCTAssertEqual(decoded.approvals.count, 1)
        XCTAssertEqual(decoded.approvals.first?.approvalRequestID, "approval-1")
        XCTAssertEqual(decoded.approvals.first?.route?.available, true)
        XCTAssertEqual(decoded.approvals.first?.route?.provider, "ollama")
        XCTAssertEqual(decoded.approvals.first?.route?.modelKey, "llama3.2")
    }

    func testModelListResponseDecodesWhenProviderEndpointIsOmitted() throws {
        let payload = """
        {
          "workspace_id": "ws-models",
          "models": [
            {
              "workspace_id": "ws-models",
              "provider": "openai",
              "model_key": "gpt-5-codex",
              "enabled": true,
              "provider_ready": true
            }
          ]
        }
        """

        let decoded = try decoder.decode(
            DaemonModelListResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws-models")
        XCTAssertEqual(decoded.models.count, 1)
        XCTAssertEqual(decoded.models.first?.provider, "openai")
        XCTAssertEqual(decoded.models.first?.modelKey, "gpt-5-codex")
        XCTAssertNil(decoded.models.first?.providerEndpoint)
    }

    func testModelDiscoverResponseDecodesSnakeCasePayload() throws {
        let payload = """
        {
          "workspace_id": "ws-models",
          "results": [
            {
              "provider": "ollama",
              "provider_ready": true,
              "provider_endpoint": "http://127.0.0.1:11434",
              "success": true,
              "message": "discovered 2 model(s)",
              "models": [
                {
                  "provider": "ollama",
                  "model_key": "llama3.2",
                  "display_name": "Llama 3.2",
                  "source": "provider_discovery",
                  "in_catalog": false,
                  "enabled": false
                }
              ]
            }
          ]
        }
        """

        let decoded = try decoder.decode(
            DaemonModelDiscoverResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws-models")
        XCTAssertEqual(decoded.results.count, 1)
        XCTAssertEqual(decoded.results.first?.provider, "ollama")
        XCTAssertEqual(decoded.results.first?.providerReady, true)
        XCTAssertEqual(decoded.results.first?.models.first?.modelKey, "llama3.2")
        XCTAssertEqual(decoded.results.first?.models.first?.displayName, "Llama 3.2")
        XCTAssertEqual(decoded.results.first?.models.first?.inCatalog, false)
    }

    func testModelCatalogRemoveResponseDecodesSnakeCasePayload() throws {
        let payload = """
        {
          "workspace_id": "ws-models",
          "provider": "ollama",
          "model_key": "llama3.2",
          "removed": true,
          "removed_at": "2026-02-25T11:22:00Z"
        }
        """

        let decoded = try decoder.decode(
            DaemonModelCatalogRemoveResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws-models")
        XCTAssertEqual(decoded.provider, "ollama")
        XCTAssertEqual(decoded.modelKey, "llama3.2")
        XCTAssertEqual(decoded.removed, true)
        XCTAssertEqual(decoded.removedAt, "2026-02-25T11:22:00Z")
    }

    func testSecretReferenceResponseDecodesSnakeCasePayload() throws {
        let payload = """
        {
          "reference": {
            "workspace_id": "ws-models",
            "name": "OPENAI_API_KEY",
            "backend": "keyring",
            "service": "personal-agent.ws-models",
            "account": "OPENAI_API_KEY"
          },
          "correlation_id": "corr-secret-1"
        }
        """

        let decoded = try decoder.decode(
            DaemonSecretReferenceResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.reference.workspaceID, "ws-models")
        XCTAssertEqual(decoded.reference.name, "OPENAI_API_KEY")
        XCTAssertEqual(decoded.reference.backend, "keyring")
        XCTAssertEqual(decoded.reference.service, "personal-agent.ws-models")
        XCTAssertEqual(decoded.reference.account, "OPENAI_API_KEY")
        XCTAssertEqual(decoded.correlationID, "corr-secret-1")
    }

    func testModelCatalogEntryRecordDecodesSnakeCasePayload() throws {
        let payload = """
        {
          "workspace_id": "ws-models",
          "provider": "openai",
          "model_key": "gpt-5-codex",
          "enabled": false,
          "updated_at": "2026-02-24T19:24:00Z"
        }
        """

        let decoded = try decoder.decode(
            DaemonModelCatalogEntryRecord.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws-models")
        XCTAssertEqual(decoded.provider, "openai")
        XCTAssertEqual(decoded.modelKey, "gpt-5-codex")
        XCTAssertEqual(decoded.enabled, false)
        XCTAssertEqual(decoded.updatedAt, "2026-02-24T19:24:00Z")
    }

    func testModelRoutingPolicyRecordDecodesSnakeCasePayload() throws {
        let payload = """
        {
          "workspace_id": "ws-models",
          "task_class": "chat",
          "provider": "openai",
          "model_key": "gpt-5-codex",
          "updated_at": "2026-02-24T19:24:00Z"
        }
        """

        let decoded = try decoder.decode(
            DaemonModelRoutingPolicyRecord.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws-models")
        XCTAssertEqual(decoded.taskClass, "chat")
        XCTAssertEqual(decoded.provider, "openai")
        XCTAssertEqual(decoded.modelKey, "gpt-5-codex")
        XCTAssertEqual(decoded.updatedAt, "2026-02-24T19:24:00Z")
    }

    func testModelRouteSimulationResponseDecodesDecisionAndFallbackTrace() throws {
        let payload = """
        {
          "workspace_id": "ws-models",
          "task_class": "automation",
          "principal_actor_id": "alex",
          "selected_provider": "openai",
          "selected_model_key": "gpt-4.1",
          "selected_source": "task_class_policy",
          "notes": "policy override matched",
          "reason_codes": ["policy_match", "provider_ready"],
          "decisions": [
            {
              "step": "task_class_policy",
              "decision": "selected",
              "reason_code": "policy_match",
              "provider": "openai",
              "model_key": "gpt-4.1",
              "note": "explicit route policy"
            }
          ],
          "fallback_chain": [
            {
              "rank": 0,
              "provider": "openai",
              "model_key": "gpt-4.1",
              "selected": true,
              "reason_code": "policy_match"
            },
            {
              "rank": 1,
              "provider": "ollama",
              "model_key": "llama3.2",
              "selected": false,
              "reason_code": "fallback_candidate"
            }
          ]
        }
        """

        let decoded = try decoder.decode(
            DaemonModelRouteSimulationResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws-models")
        XCTAssertEqual(decoded.taskClass, "automation")
        XCTAssertEqual(decoded.principalActorID, "alex")
        XCTAssertEqual(decoded.selectedProvider, "openai")
        XCTAssertEqual(decoded.selectedModelKey, "gpt-4.1")
        XCTAssertEqual(decoded.selectedSource, "task_class_policy")
        XCTAssertEqual(decoded.reasonCodes, ["policy_match", "provider_ready"])
        XCTAssertEqual(decoded.decisions.count, 1)
        XCTAssertEqual(decoded.decisions.first?.step, "task_class_policy")
        XCTAssertEqual(decoded.decisions.first?.reasonCode, "policy_match")
        XCTAssertEqual(decoded.fallbackChain.count, 2)
        XCTAssertEqual(decoded.fallbackChain.first?.selected, true)
        XCTAssertEqual(decoded.fallbackChain.last?.provider, "ollama")
    }

    func testModelRouteExplainResponseDecodesSummaryAndExplanations() throws {
        let payload = """
        {
          "workspace_id": "ws-models",
          "task_class": "chat",
          "principal_actor_id": "default",
          "selected_provider": "openai",
          "selected_model_key": "gpt-4.1",
          "selected_source": "task_class_policy",
          "summary": "Route selected from explicit chat policy.",
          "explanations": [
            "Task class policy matched openai/gpt-4.1.",
            "Provider check reported healthy endpoint."
          ],
          "reason_codes": ["policy_match", "provider_ready"],
          "decisions": [
            {
              "step": "policy_lookup",
              "decision": "selected",
              "reason_code": "policy_match",
              "provider": "openai",
              "model_key": "gpt-4.1"
            }
          ],
          "fallback_chain": []
        }
        """

        let decoded = try decoder.decode(
            DaemonModelRouteExplainResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws-models")
        XCTAssertEqual(decoded.taskClass, "chat")
        XCTAssertEqual(decoded.principalActorID, "default")
        XCTAssertEqual(decoded.selectedProvider, "openai")
        XCTAssertEqual(decoded.selectedModelKey, "gpt-4.1")
        XCTAssertEqual(decoded.selectedSource, "task_class_policy")
        XCTAssertEqual(decoded.summary, "Route selected from explicit chat policy.")
        XCTAssertEqual(decoded.explanations.count, 2)
        XCTAssertEqual(decoded.reasonCodes, ["policy_match", "provider_ready"])
        XCTAssertEqual(decoded.decisions.first?.step, "policy_lookup")
        XCTAssertEqual(decoded.fallbackChain.count, 0)
    }

    func testChatTurnResponseDecodesTaskRunCorrelationPayload() throws {
        let payload = """
        {
          "workspace_id": "ws-chat",
          "task_class": "chat",
          "provider": "openai",
          "model_key": "gpt-5-codex",
          "items": [
            {
              "item_id": "item-assistant",
              "type": "assistant_message",
              "role": "assistant",
              "status": "completed",
              "content": "Done."
            }
          ],
          "correlation_id": "corr-chat-1",
          "task_run_correlation": {
            "available": true,
            "source": "audit",
            "task_id": "task-1",
            "run_id": "run-1",
            "task_state": "completed",
            "run_state": "completed"
          }
        }
        """

        let decoded = try decoder.decode(
            DaemonChatTurnResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws-chat")
        XCTAssertEqual(decoded.taskClass, "chat")
        XCTAssertEqual(decoded.provider, "openai")
        XCTAssertEqual(decoded.modelKey, "gpt-5-codex")
        XCTAssertEqual(decoded.correlationID, "corr-chat-1")
        XCTAssertEqual(decoded.items.count, 1)
        XCTAssertEqual(decoded.items.first?.type, "assistant_message")
        XCTAssertEqual(decoded.taskRunCorrelation.available, true)
        XCTAssertEqual(decoded.taskRunCorrelation.source, "audit")
        XCTAssertEqual(decoded.taskRunCorrelation.taskID, "task-1")
        XCTAssertEqual(decoded.taskRunCorrelation.runID, "run-1")
        XCTAssertEqual(decoded.taskRunCorrelation.taskState, "completed")
        XCTAssertEqual(decoded.taskRunCorrelation.runState, "completed")
        XCTAssertNil(decoded.contractVersion)
        XCTAssertNil(decoded.turnItemSchemaVersion)
        XCTAssertNil(decoded.realtimeEventContractVersion)
    }

    func testChatTurnResponseDecodesContractV2Metadata() throws {
        let payload = """
        {
          "workspace_id": "ws-chat",
          "task_class": "chat",
          "provider": "openai",
          "model_key": "gpt-4.1",
          "correlation_id": "corr-chat-contract",
          "contract_version": "chat_turn.v2",
          "turn_item_schema_version": "chat_turn_item.v2",
          "realtime_event_contract_version": "chat_realtime_lifecycle.v2",
          "items": [],
          "task_run_correlation": {
            "available": false,
            "source": "none"
          }
        }
        """

        let decoded = try decoder.decode(
            DaemonChatTurnResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.contractVersion, "chat_turn.v2")
        XCTAssertEqual(decoded.turnItemSchemaVersion, "chat_turn_item.v2")
        XCTAssertEqual(decoded.realtimeEventContractVersion, "chat_realtime_lifecycle.v2")
    }

    func testChatTurnResponseMissingTaskRunCorrelationDecodesDeterministicDefaults() throws {
        let payload = """
        {
          "workspace_id": "ws-chat",
          "provider": "openai",
          "model_key": "gpt-5-codex",
          "items": [
            {
              "item_id": "item-assistant",
              "type": "assistant_message",
              "role": "assistant",
              "status": "completed",
              "content": "Done."
            }
          ],
          "correlation_id": "corr-chat-2"
        }
        """

        let decoded = try decoder.decode(
            DaemonChatTurnResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.taskClass, "chat")
        XCTAssertEqual(decoded.items.count, 1)
        XCTAssertEqual(decoded.items.first?.content, "Done.")
        XCTAssertEqual(decoded.taskRunCorrelation.available, false)
        XCTAssertEqual(decoded.taskRunCorrelation.source, "none")
        XCTAssertNil(decoded.taskRunCorrelation.taskID)
        XCTAssertNil(decoded.taskRunCorrelation.runID)
        XCTAssertNil(decoded.taskRunCorrelation.taskState)
        XCTAssertNil(decoded.taskRunCorrelation.runState)
    }

    func testChatTurnResponseDecodesCanonicalTurnItems() throws {
        let payload = """
        {
          "workspace_id": "ws-chat",
          "task_class": "chat",
          "provider": "openai",
          "model_key": "gpt-5-codex",
          "correlation_id": "corr-chat-3",
          "task_run_correlation": {
            "available": true,
            "source": "agent_run",
            "task_id": "task-1",
            "run_id": "run-1",
            "task_state": "awaiting_approval",
            "run_state": "awaiting_approval"
          },
          "items": [
            {
              "item_id": "item-tool-call",
              "type": "tool_call",
              "status": "started",
              "tool_name": "send_email",
              "tool_call_id": "tool-send-email"
            },
            {
              "item_id": "item-tool-result",
              "type": "tool_result",
              "status": "awaiting_approval",
              "tool_name": "send_email",
              "tool_call_id": "tool-send-email",
              "approval_request_id": "approval-1",
              "output": {
                "task_id": "task-1",
                "run_id": "run-1"
              }
            },
            {
              "item_id": "item-approval",
              "type": "approval_request",
              "status": "awaiting_approval",
              "approval_request_id": "approval-1",
              "content": "Approval is required before execution can continue."
            },
            {
              "item_id": "item-assistant",
              "type": "assistant_message",
              "role": "assistant",
              "status": "completed",
              "content": "send_email is waiting for approval (request approval-1)."
            }
          ]
        }
        """

        let decoded = try decoder.decode(
            DaemonChatTurnResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.items.count, 4)
        XCTAssertEqual(decoded.items.first?.type, "tool_call")
        XCTAssertEqual(decoded.items.dropFirst().first?.type, "tool_result")
        XCTAssertEqual(decoded.items.dropFirst().first?.approvalRequestID, "approval-1")
        XCTAssertEqual(decoded.items.last?.type, "assistant_message")
    }

    func testChatTurnExplainResponseDecodesSelectedRouteToolCatalogAndPolicyDecisions() throws {
        let payload = """
        {
          "workspace_id": "ws-chat",
          "task_class": "chat",
          "requested_by_actor_id": "default",
          "subject_actor_id": "default",
          "acting_as_actor_id": "default",
          "contract_version": "chat_turn_explain.v1",
          "selected_route": {
            "workspace_id": "ws-chat",
            "task_class": "chat",
            "selected_provider": "openai",
            "selected_model_key": "gpt-4.1",
            "selected_source": "task_class_policy",
            "summary": "Policy selected route.",
            "explanations": ["Policy matched."],
            "reason_codes": ["policy_match"],
            "decisions": [],
            "fallback_chain": []
          },
          "tool_catalog": [
            {
              "name": "send_email",
              "description": "Send mail.",
              "capability_keys": ["connector.mail.send"],
              "input_schema": {
                "type": "object"
              }
            }
          ],
          "policy_decisions": [
            {
              "tool_name": "send_email",
              "capability_key": "connector.mail.send",
              "decision": "allow",
              "reason": "Connector ready."
            }
          ]
        }
        """

        let decoded = try decoder.decode(
            DaemonChatTurnExplainResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws-chat")
        XCTAssertEqual(decoded.contractVersion, "chat_turn_explain.v1")
        XCTAssertEqual(decoded.selectedRoute?.selectedProvider, "openai")
        XCTAssertEqual(decoded.selectedRoute?.selectedModelKey, "gpt-4.1")
        XCTAssertEqual(decoded.toolCatalog.count, 1)
        XCTAssertEqual(decoded.toolCatalog.first?.name, "send_email")
        XCTAssertEqual(decoded.toolCatalog.first?.capabilityKeys, ["connector.mail.send"])
        XCTAssertEqual(decoded.policyDecisions.count, 1)
        XCTAssertEqual(decoded.policyDecisions.first?.toolName, "send_email")
        XCTAssertEqual(decoded.policyDecisions.first?.decision, "allow")
    }

    func testRealtimeEventEnvelopeDecodesLifecycleContractMetadata() throws {
        let payload = """
        {
          "event_id": "evt-chat-1",
          "sequence": 42,
          "event_type": "turn_item_started",
          "occurred_at": "2026-03-03T16:22:00Z",
          "correlation_id": "corr-chat-1",
          "contract_version": "chat_realtime_lifecycle.v2",
          "lifecycle_schema_version": "chat_lifecycle_item.v2",
          "payload": {
            "item_type": "tool_call",
            "status": "started"
          }
        }
        """

        let decoded = try decoder.decode(
            DaemonRealtimeEventEnvelope.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.eventID, "evt-chat-1")
        XCTAssertEqual(decoded.contractVersion, "chat_realtime_lifecycle.v2")
        XCTAssertEqual(decoded.lifecycleSchemaVersion, "chat_lifecycle_item.v2")
        XCTAssertEqual(decoded.payload.itemType, "tool_call")
    }

    func testChatPersonaPolicyResponseDecodesScopeAndGuardrails() throws {
        let payload = """
        {
          "workspace_id": "ws-chat",
          "principal_actor_id": "owner",
          "channel_id": "message",
          "style_prompt": "Keep responses concise and action-oriented.",
          "guardrails": [
            "Confirm recipients before sending.",
            "Never expose token values."
          ],
          "source": "persisted",
          "updated_at": "2026-02-28T12:30:00Z"
        }
        """

        let decoded = try decoder.decode(
            DaemonChatPersonaPolicyResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws-chat")
        XCTAssertEqual(decoded.principalActorID, "owner")
        XCTAssertEqual(decoded.channelID, "message")
        XCTAssertEqual(decoded.stylePrompt, "Keep responses concise and action-oriented.")
        XCTAssertEqual(decoded.guardrails.count, 2)
        XCTAssertEqual(decoded.guardrails.first, "Confirm recipients before sending.")
        XCTAssertEqual(decoded.source, "persisted")
        XCTAssertEqual(decoded.updatedAt, "2026-02-28T12:30:00Z")
    }

    func testChatPersonaPolicyResponseMissingOptionalFieldsUsesDeterministicDefaults() throws {
        let payload = """
        {
          "workspace_id": "ws-chat",
          "style_prompt": "Default persona prompt."
        }
        """

        let decoded = try decoder.decode(
            DaemonChatPersonaPolicyResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws-chat")
        XCTAssertNil(decoded.principalActorID)
        XCTAssertNil(decoded.channelID)
        XCTAssertEqual(decoded.stylePrompt, "Default persona prompt.")
        XCTAssertEqual(decoded.guardrails, [])
        XCTAssertEqual(decoded.source, "default")
        XCTAssertNil(decoded.updatedAt)
    }

    func testDaemonLifecycleStatusDecodesSetupControlsAndControlOperation() throws {
        let payload = """
        {
          "lifecycle_state": "running",
          "process_id": 9921,
          "started_at": "2026-02-25T10:00:00Z",
          "last_transition_at": "2026-02-25T10:01:00Z",
          "runtime_mode": "tcp",
          "configured_address": "127.0.0.1:7071",
          "bound_address": "127.0.0.1:7071",
          "setup_state": "ready",
          "install_state": "installed",
          "needs_install": false,
          "needs_repair": false,
          "health_classification": {
            "overall_state": "ready",
            "core_runtime_state": "ready",
            "plugin_runtime_state": "healthy",
            "blocking": false
          },
          "database_ready": true,
          "control_auth": {
            "state": "configured",
            "source": "auth_token_flag",
            "remediation_hints": []
          },
          "worker_summary": {
            "total": 6,
            "registered": 6,
            "starting": 0,
            "running": 6,
            "restarting": 0,
            "stopped": 0,
            "failed": 0
          },
          "controls": {
            "start": true,
            "stop": true,
            "restart": true,
            "install": true,
            "uninstall": true,
            "repair": true
          },
          "control_operation": {
            "action": "install",
            "state": "in_progress",
            "message": "daemon install operation started",
            "requested_at": "2026-02-25T10:01:02Z"
          }
        }
        """

        let decoded = try decoder.decode(
            DaemonLifecycleStatusResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.lifecycleState, "running")
        XCTAssertEqual(decoded.controls.install, true)
        XCTAssertEqual(decoded.controls.uninstall, true)
        XCTAssertEqual(decoded.controls.repair, true)
        XCTAssertEqual(decoded.controlOperation.action, "install")
        XCTAssertEqual(decoded.controlOperation.state, "in_progress")
        XCTAssertEqual(decoded.controlOperation.requestedAt, "2026-02-25T10:01:02Z")
        XCTAssertEqual(decoded.healthClassification.overallState, "ready")
        XCTAssertEqual(decoded.healthClassification.coreRuntimeState, "ready")
        XCTAssertEqual(decoded.healthClassification.pluginRuntimeState, "healthy")
        XCTAssertEqual(decoded.healthClassification.blocking, false)
        XCTAssertEqual(decoded.controlAuth.state, "configured")
        XCTAssertEqual(decoded.controlAuth.source, "auth_token_flag")
        XCTAssertEqual(decoded.controlAuth.remediationHints, [])
    }

    func testDaemonLifecycleControlResponseDefaultsOperationStateWhenMissing() throws {
        let payload = """
        {
          "action": "restart",
          "accepted": true,
          "idempotent": true,
          "lifecycle_state": "running",
          "message": "daemon already running"
        }
        """

        let decoded = try decoder.decode(
            DaemonLifecycleControlResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.action, "restart")
        XCTAssertEqual(decoded.accepted, true)
        XCTAssertEqual(decoded.operationState, "succeeded")
        XCTAssertNil(decoded.requestedAt)
        XCTAssertNil(decoded.completedAt)
        XCTAssertNil(decoded.error)
    }

    func testDaemonLifecycleStatusDefaultsHealthClassificationWhenMissing() throws {
        let payload = """
        {
          "lifecycle_state": "running",
          "process_id": 221,
          "started_at": "2026-02-25T11:00:00Z",
          "last_transition_at": "2026-02-25T11:00:01Z",
          "setup_state": "ready",
          "install_state": "installed",
          "needs_install": false,
          "needs_repair": false,
          "database_ready": true
        }
        """

        let decoded = try decoder.decode(
            DaemonLifecycleStatusResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.healthClassification.overallState, "unknown")
        XCTAssertEqual(decoded.healthClassification.coreRuntimeState, "unknown")
        XCTAssertEqual(decoded.healthClassification.pluginRuntimeState, "unknown")
        XCTAssertEqual(decoded.healthClassification.blocking, false)
        XCTAssertEqual(decoded.controlAuth.state, "unknown")
        XCTAssertEqual(decoded.controlAuth.source, "unknown")
        XCTAssertEqual(decoded.controlAuth.remediationHints, [])
    }

    func testDaemonPluginLifecycleHistoryResponseDecodesSnakeCasePayload() throws {
        let payload = """
        {
          "workspace_id": "daemon",
          "items": [
            {
              "audit_id": "audit-1",
              "workspace_id": "daemon",
              "plugin_id": "messages.daemon",
              "kind": "channel",
              "state": "running",
              "event_type": "PLUGIN_HANDSHAKE_ACCEPTED",
              "process_id": 9221,
              "restart_count": 2,
              "reason": "worker_recovered",
              "restart_event": false,
              "failure_event": false,
              "recovery_event": true,
              "occurred_at": "2026-02-25T12:01:00Z"
            }
          ],
          "has_more": true,
          "next_cursor_created_at": "2026-02-25T12:01:00Z",
          "next_cursor_id": "audit-1"
        }
        """

        let decoded = try decoder.decode(
            DaemonPluginLifecycleHistoryResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "daemon")
        XCTAssertEqual(decoded.items.count, 1)
        XCTAssertEqual(decoded.items.first?.pluginID, "messages.daemon")
        XCTAssertEqual(decoded.items.first?.eventType, "PLUGIN_HANDSHAKE_ACCEPTED")
        XCTAssertEqual(decoded.items.first?.restartCount, 2)
        XCTAssertEqual(decoded.items.first?.recoveryEvent, true)
        XCTAssertTrue(decoded.hasMore)
        XCTAssertEqual(decoded.nextCursorCreatedAt, "2026-02-25T12:01:00Z")
        XCTAssertEqual(decoded.nextCursorID, "audit-1")
    }

    func testContextMemoryInventoryResponseDecodesSnakeCasePayload() throws {
        let payload = """
        {
          "workspace_id": "ws1",
          "items": [
            {
              "memory_id": "mem-1",
              "workspace_id": "ws1",
              "owner_actor_id": "actor.context.a",
              "scope_type": "conversation",
              "key": "summary-1",
              "status": "ACTIVE",
              "kind": "summary",
              "is_canonical": true,
              "token_estimate": 21,
              "source_summary": "event://manual-1",
              "source_count": 1,
              "created_at": "2026-02-25T12:10:00Z",
              "updated_at": "2026-02-25T12:11:00Z",
              "value_json": "{\\"kind\\":\\"summary\\"}",
              "sources": [
                {
                  "source_id": "src-1",
                  "source_type": "comm_event",
                  "source_ref": "event://manual-1",
                  "created_at": "2026-02-25T12:10:00Z"
                }
              ]
            }
          ],
          "has_more": true,
          "next_cursor_updated_at": "2026-02-25T12:11:00Z",
          "next_cursor_id": "mem-1"
        }
        """

        let decoded = try decoder.decode(
            DaemonContextMemoryInventoryResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws1")
        XCTAssertEqual(decoded.items.count, 1)
        XCTAssertEqual(decoded.items.first?.memoryID, "mem-1")
        XCTAssertEqual(decoded.items.first?.ownerActorID, "actor.context.a")
        XCTAssertEqual(decoded.items.first?.sources.count, 1)
        XCTAssertEqual(decoded.items.first?.sources.first?.sourceType, "comm_event")
        XCTAssertTrue(decoded.hasMore)
        XCTAssertEqual(decoded.nextCursorUpdatedAt, "2026-02-25T12:11:00Z")
        XCTAssertEqual(decoded.nextCursorID, "mem-1")
    }

    func testContextMemoryCandidatesResponseDecodesSnakeCasePayload() throws {
        let payload = """
        {
          "workspace_id": "ws1",
          "items": [
            {
              "candidate_id": "cand-1",
              "workspace_id": "ws1",
              "owner_actor_id": "actor.context.a",
              "status": "PENDING",
              "score": 0.93,
              "candidate_json": "{\\"kind\\":\\"summary\\"}",
              "candidate_kind": "summary",
              "token_estimate": 55,
              "source_ids": ["mem-1","mem-2"],
              "source_refs": ["event://manual-1","event://manual-2"],
              "created_at": "2026-02-25T12:12:00Z"
            }
          ],
          "has_more": false
        }
        """

        let decoded = try decoder.decode(
            DaemonContextMemoryCandidatesResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws1")
        XCTAssertEqual(decoded.items.count, 1)
        XCTAssertEqual(decoded.items.first?.candidateID, "cand-1")
        XCTAssertEqual(decoded.items.first?.candidateKind, "summary")
        XCTAssertEqual(decoded.items.first?.sourceIDs.count, 2)
        XCTAssertFalse(decoded.hasMore)
    }

    func testContextRetrievalDocumentsResponseDecodesSnakeCasePayload() throws {
        let payload = """
        {
          "workspace_id": "ws1",
          "items": [
            {
              "document_id": "doc-1",
              "workspace_id": "ws1",
              "owner_actor_id": "actor.context.a",
              "source_uri": "memory://manual/doc-1",
              "checksum": "abc123",
              "chunk_count": 2,
              "created_at": "2026-02-25T12:13:00Z"
            }
          ],
          "has_more": false
        }
        """

        let decoded = try decoder.decode(
            DaemonContextRetrievalDocumentsResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws1")
        XCTAssertEqual(decoded.items.count, 1)
        XCTAssertEqual(decoded.items.first?.documentID, "doc-1")
        XCTAssertEqual(decoded.items.first?.chunkCount, 2)
    }

    func testContextRetrievalChunksResponseDecodesSnakeCasePayload() throws {
        let payload = """
        {
          "workspace_id": "ws1",
          "document_id": "doc-1",
          "items": [
            {
              "chunk_id": "chunk-1",
              "workspace_id": "ws1",
              "document_id": "doc-1",
              "owner_actor_id": "actor.context.a",
              "source_uri": "memory://manual/doc-1",
              "chunk_index": 0,
              "text_body": "manual retrieval chunk",
              "token_count": 7,
              "created_at": "2026-02-25T12:14:00Z"
            }
          ],
          "has_more": true,
          "next_cursor_created_at": "2026-02-25T12:14:00Z",
          "next_cursor_id": "chunk-1"
        }
        """

        let decoded = try decoder.decode(
            DaemonContextRetrievalChunksResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws1")
        XCTAssertEqual(decoded.documentID, "doc-1")
        XCTAssertEqual(decoded.items.count, 1)
        XCTAssertEqual(decoded.items.first?.chunkID, "chunk-1")
        XCTAssertEqual(decoded.items.first?.textBody, "manual retrieval chunk")
        XCTAssertTrue(decoded.hasMore)
        XCTAssertEqual(decoded.nextCursorID, "chunk-1")
    }

    func testAutomationListResponseDecodesCanonicalSubjectActorKeyAndMissingCooldown() throws {
        let payload = """
        {
          "workspace_id": "ws-automation",
          "triggers": [
            {
              "trigger_id": "trigger-1",
              "workspace_id": "ws-automation",
              "directive_id": "directive-1",
              "trigger_type": "SCHEDULE",
              "enabled": true,
              "filter_json": "{\\"interval_seconds\\":300}",
              "subject_principal_actor": "actor.default",
              "directive_title": "Check inbox",
              "directive_instruction": "Review unread mail",
              "directive_status": "ACTIVE",
              "created_at": "2026-02-25T12:00:00Z",
              "updated_at": "2026-02-25T12:00:00Z"
            }
          ]
        }
        """

        let decoded = try decoder.decode(
            DaemonAutomationListResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws-automation")
        XCTAssertEqual(decoded.triggers.count, 1)
        XCTAssertEqual(decoded.triggers.first?.triggerID, "trigger-1")
        XCTAssertEqual(decoded.triggers.first?.subjectPrincipalActor, "actor.default")
        XCTAssertEqual(decoded.triggers.first?.cooldownSeconds, 0)
    }

    func testAutomationFireHistoryResponseDecodesSnakeCasePayload() throws {
        let payload = """
        {
          "workspace_id": "ws-automation",
          "fires": [
            {
              "fire_id": "fire-1",
              "workspace_id": "ws-automation",
              "trigger_id": "trigger-1",
              "trigger_type": "SCHEDULE",
              "directive_id": "directive-1",
              "status": "CREATED_TASK",
              "outcome": "CREATED_TASK",
              "idempotency_key": "ws-automation::trigger-1::seed",
              "idempotency_signal": "seed",
              "fired_at": "2026-02-25T12:00:00Z",
              "task_id": "task-1",
              "run_id": "run-1",
              "route": {
                "available": true,
                "task_class": "chat",
                "provider": "openai",
                "model_key": "gpt-5-codex",
                "task_class_source": "policy",
                "route_source": "explicit"
              }
            }
          ]
        }
        """

        let decoded = try decoder.decode(
            DaemonAutomationFireHistoryResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws-automation")
        XCTAssertEqual(decoded.fires.count, 1)
        XCTAssertEqual(decoded.fires.first?.fireID, "fire-1")
        XCTAssertEqual(decoded.fires.first?.triggerType, "SCHEDULE")
        XCTAssertEqual(decoded.fires.first?.status, "CREATED_TASK")
        XCTAssertEqual(decoded.fires.first?.idempotencySignal, "seed")
        XCTAssertEqual(decoded.fires.first?.taskID, "task-1")
        XCTAssertEqual(decoded.fires.first?.runID, "run-1")
        XCTAssertEqual(decoded.fires.first?.route?.available, true)
        XCTAssertEqual(decoded.fires.first?.route?.provider, "openai")
        XCTAssertEqual(decoded.fires.first?.route?.modelKey, "gpt-5-codex")
    }

    func testCommThreadListResponseDecodesSnakeCasePayload() throws {
        let payload = """
        {
          "workspace_id": "ws-comm",
          "items": [
            {
              "thread_id": "thread-1",
              "workspace_id": "ws-comm",
              "channel": "message",
              "connector_id": "imessage",
              "external_ref": "chatdb:42",
              "title": "Family",
              "last_event_id": "event-2",
              "last_event_type": "MESSAGE",
              "last_direction": "inbound",
              "last_occurred_at": "2026-02-25T13:00:00Z",
              "last_body_preview": "Dinner at 7?",
              "participant_addresses": ["+15550000001", "+15550000002"],
              "event_count": 2,
              "created_at": "2026-02-25T12:00:00Z",
              "updated_at": "2026-02-25T13:00:00Z"
            }
          ],
          "has_more": true,
          "next_cursor": "2026-02-25T12:00:00Z|thread-1"
        }
        """

        let decoded = try decoder.decode(
            DaemonCommThreadListResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws-comm")
        XCTAssertEqual(decoded.items.count, 1)
        XCTAssertEqual(decoded.items.first?.threadID, "thread-1")
        XCTAssertEqual(decoded.items.first?.channel, "message")
        XCTAssertEqual(decoded.items.first?.connectorID, "imessage")
        XCTAssertEqual(decoded.items.first?.participantAddresses.count, 2)
        XCTAssertEqual(decoded.items.first?.eventCount, 2)
        XCTAssertTrue(decoded.hasMore)
        XCTAssertEqual(decoded.nextCursor, "2026-02-25T12:00:00Z|thread-1")
    }

    func testCommEventTimelineResponseDecodesAddressesAndCursor() throws {
        let payload = """
        {
          "workspace_id": "ws-comm",
          "thread_id": "thread-1",
          "items": [
            {
              "event_id": "event-9",
              "workspace_id": "ws-comm",
              "thread_id": "thread-1",
              "channel": "message",
              "connector_id": "imessage",
              "event_type": "MESSAGE",
              "direction": "outbound",
              "assistant_emitted": true,
              "body_text": "On it.",
              "occurred_at": "2026-02-25T13:01:00Z",
              "created_at": "2026-02-25T13:01:01Z",
              "addresses": [
                {
                  "role": "FROM",
                  "value": "+15550000001",
                  "display": "Requester",
                  "position": 0
                },
                {
                  "role": "TO",
                  "value": "+15550000002",
                  "position": 1
                }
              ]
            }
          ],
          "has_more": false
        }
        """

        let decoded = try decoder.decode(
            DaemonCommEventTimelineResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws-comm")
        XCTAssertEqual(decoded.threadID, "thread-1")
        XCTAssertEqual(decoded.items.count, 1)
        XCTAssertEqual(decoded.items.first?.eventID, "event-9")
        XCTAssertEqual(decoded.items.first?.channel, "message")
        XCTAssertEqual(decoded.items.first?.connectorID, "imessage")
        XCTAssertEqual(decoded.items.first?.assistantEmitted, true)
        XCTAssertEqual(decoded.items.first?.addresses.count, 2)
        XCTAssertEqual(decoded.items.first?.addresses.first?.role, "FROM")
        XCTAssertEqual(decoded.items.first?.addresses.first?.display, "Requester")
        XCTAssertFalse(decoded.hasMore)
        XCTAssertNil(decoded.nextCursor)
    }

    func testCommCallSessionListResponseDecodesSnakeCasePayload() throws {
        let payload = """
        {
          "workspace_id": "ws-comm",
          "items": [
            {
              "session_id": "call-session-1",
              "workspace_id": "ws-comm",
              "provider": "twilio_voice",
              "connector_id": "twilio",
              "provider_call_id": "CA123",
              "thread_id": "thread-voice",
              "direction": "inbound",
              "from_address": "+15550000003",
              "to_address": "+15550000004",
              "status": "in_progress",
              "started_at": "2026-02-25T13:02:00Z",
              "updated_at": "2026-02-25T13:03:00Z"
            }
          ],
          "has_more": true,
          "next_cursor": "2026-02-25T13:03:00Z|call-session-1"
        }
        """

        let decoded = try decoder.decode(
            DaemonCommCallSessionListResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws-comm")
        XCTAssertEqual(decoded.items.count, 1)
        XCTAssertEqual(decoded.items.first?.sessionID, "call-session-1")
        XCTAssertEqual(decoded.items.first?.provider, "twilio_voice")
        XCTAssertEqual(decoded.items.first?.connectorID, "twilio")
        XCTAssertEqual(decoded.items.first?.providerCallID, "CA123")
        XCTAssertEqual(decoded.items.first?.status, "in_progress")
        XCTAssertTrue(decoded.hasMore)
        XCTAssertEqual(decoded.nextCursor, "2026-02-25T13:03:00Z|call-session-1")
    }

    func testCommAttemptsResponseDecodesContextAndFallbackMetadata() throws {
        let payload = """
        {
          "workspace_id": "ws-comm",
          "thread_id": "thread-1",
          "task_id": "task-1",
          "run_id": "run-1",
          "has_more": true,
          "next_cursor": "cursor-1",
          "attempts": [
            {
              "attempt_id": "attempt-1",
              "workspace_id": "ws-comm",
              "operation_id": "op-1",
              "task_id": "task-1",
              "run_id": "run-1",
              "step_id": "step-1",
              "event_id": "event-1",
              "thread_id": "thread-1",
              "destination_endpoint": "+15550000002",
              "idempotency_key": "idem-1",
              "channel": "sms",
              "route_index": 1,
              "route_phase": "fallback",
              "retry_ordinal": 2,
              "fallback_from_channel": "imessage",
              "status": "delivered",
              "provider_receipt": "receipt-1",
              "attempted_at": "2026-02-25T13:04:00Z"
            }
          ]
        }
        """

        let decoded = try decoder.decode(
            DaemonCommAttemptsResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws-comm")
        XCTAssertEqual(decoded.threadID, "thread-1")
        XCTAssertEqual(decoded.taskID, "task-1")
        XCTAssertEqual(decoded.runID, "run-1")
        XCTAssertTrue(decoded.hasMore)
        XCTAssertEqual(decoded.nextCursor, "cursor-1")
        XCTAssertEqual(decoded.attempts.count, 1)
        XCTAssertEqual(decoded.attempts.first?.attemptID, "attempt-1")
        XCTAssertEqual(decoded.attempts.first?.routePhase, "fallback")
        XCTAssertEqual(decoded.attempts.first?.retryOrdinal, 2)
        XCTAssertEqual(decoded.attempts.first?.fallbackFromChannel, "imessage")
        XCTAssertEqual(decoded.attempts.first?.status, "delivered")
    }

    func testCommSendResponseDecodesThreadAwareRoutingMetadata() throws {
        let payload = """
        {
          "workspace_id": "ws-comm",
          "operation_id": "op-comm-send-1",
          "thread_id": "thread-reply-1",
          "resolved_source_channel": "imessage_sms",
          "resolved_connector_id": "imessage",
          "resolved_destination": "+15550000002",
          "success": true,
          "result": {
            "Delivered": true,
            "Channel": "imessage",
            "IdempotentReplay": false
          }
        }
        """

        let decoded = try decoder.decode(
            DaemonCommSendResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws-comm")
        XCTAssertEqual(decoded.operationID, "op-comm-send-1")
        XCTAssertEqual(decoded.threadID, "thread-reply-1")
        XCTAssertEqual(decoded.resolvedSourceChannel, "imessage_sms")
        XCTAssertEqual(decoded.resolvedConnectorID, "imessage")
        XCTAssertEqual(decoded.resolvedDestination, "+15550000002")
        XCTAssertTrue(decoded.success)
        XCTAssertNotNil(decoded.result)
    }

    func testCommPolicyRecordDecodesSnakeCasePayload() throws {
        let payload = """
        {
          "id": "policy-1",
          "workspace_id": "ws-comm",
          "source_channel": "imessage",
          "endpoint_pattern": "+1555*",
          "is_default": true,
          "policy": {
            "primary_channel": "imessage",
            "retry_count": 1,
            "fallback_channels": ["sms"]
          },
          "created_at": "2026-02-25T13:00:00Z",
          "updated_at": "2026-02-25T13:05:00Z"
        }
        """

        let decoded = try decoder.decode(
            DaemonCommPolicyRecord.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.id, "policy-1")
        XCTAssertEqual(decoded.workspaceID, "ws-comm")
        XCTAssertEqual(decoded.sourceChannel, "imessage")
        XCTAssertEqual(decoded.endpointPattern, "+1555*")
        XCTAssertTrue(decoded.isDefault)
        XCTAssertEqual(decoded.policy.primaryChannel, "imessage")
        XCTAssertEqual(decoded.policy.retryCount, 1)
        XCTAssertEqual(decoded.policy.fallbackChannels, ["sms"])
        XCTAssertEqual(decoded.createdAt, "2026-02-25T13:00:00Z")
        XCTAssertEqual(decoded.updatedAt, "2026-02-25T13:05:00Z")
    }

    func testCommPolicyListResponseDecodesPolicyArray() throws {
        let payload = """
        {
          "workspace_id": "ws-comm",
          "policies": [
            {
              "id": "policy-1",
              "workspace_id": "ws-comm",
              "source_channel": "imessage",
              "is_default": true,
              "policy": {
                "primary_channel": "imessage",
                "retry_count": 1,
                "fallback_channels": ["sms"]
              },
              "created_at": "2026-02-25T13:00:00Z",
              "updated_at": "2026-02-25T13:05:00Z"
            }
          ]
        }
        """

        let decoded = try decoder.decode(
            DaemonCommPolicyListResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws-comm")
        XCTAssertEqual(decoded.policies.count, 1)
        XCTAssertEqual(decoded.policies.first?.id, "policy-1")
        XCTAssertEqual(decoded.policies.first?.sourceChannel, "imessage")
        XCTAssertEqual(decoded.policies.first?.policy.retryCount, 1)
        XCTAssertEqual(decoded.policies.first?.policy.fallbackChannels, ["sms"])
    }

    func testIdentityWorkspacesResponseDecodesDirectoryContext() throws {
        let payload = """
        {
          "active_context": {
            "workspace_id": "ws1",
            "principal_actor_id": "actor.requester",
            "workspace_source": "selected",
            "principal_source": "derived",
            "last_updated_at": "2026-02-25T00:00:00Z",
            "workspace_resolved": true
          },
          "workspaces": [
            {
              "workspace_id": "ws1",
              "name": "Workspace One",
              "status": "ACTIVE",
              "principal_count": 2,
              "actor_count": 3,
              "handle_count": 4,
              "updated_at": "2026-02-25T00:00:00Z",
              "is_active": true
            }
          ]
        }
        """

        let decoded = try decoder.decode(
            DaemonIdentityWorkspacesResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.activeContext?.workspaceID, "ws1")
        XCTAssertEqual(decoded.activeContext?.principalActorID, "actor.requester")
        XCTAssertEqual(decoded.activeContext?.workspaceResolved, true)
        XCTAssertEqual(decoded.workspaces.count, 1)
        XCTAssertEqual(decoded.workspaces.first?.workspaceID, "ws1")
        XCTAssertEqual(decoded.workspaces.first?.principalCount, 2)
        XCTAssertEqual(decoded.workspaces.first?.handleCount, 4)
        XCTAssertEqual(decoded.workspaces.first?.isActive, true)
    }

    func testIdentityPrincipalsResponseDecodesHandleMappings() throws {
        let payload = """
        {
          "workspace_id": "ws1",
          "active_context": {
            "workspace_id": "ws1",
            "principal_actor_id": "actor.requester",
            "workspace_source": "selected",
            "principal_source": "selected",
            "workspace_resolved": true
          },
          "principals": [
            {
              "actor_id": "actor.requester",
              "display_name": "Requester",
              "actor_type": "human",
              "actor_status": "ACTIVE",
              "principal_status": "ACTIVE",
              "is_active": true,
              "handles": [
                {
                  "channel": "imessage",
                  "handle_value": "+15550000001",
                  "is_primary": true,
                  "updated_at": "2026-02-25T00:00:00Z"
                }
              ]
            }
          ]
        }
        """

        let decoded = try decoder.decode(
            DaemonIdentityPrincipalsResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws1")
        XCTAssertEqual(decoded.activeContext?.principalActorID, "actor.requester")
        XCTAssertEqual(decoded.principals.count, 1)
        XCTAssertEqual(decoded.principals.first?.actorID, "actor.requester")
        XCTAssertEqual(decoded.principals.first?.displayName, "Requester")
        XCTAssertEqual(decoded.principals.first?.handles.count, 1)
        XCTAssertEqual(decoded.principals.first?.handles.first?.channel, "imessage")
        XCTAssertEqual(decoded.principals.first?.handles.first?.handleValue, "+15550000001")
        XCTAssertEqual(decoded.principals.first?.handles.first?.isPrimary, true)
    }

    func testIdentityDeviceListResponseDecodesSessionMetadata() throws {
        let payload = """
        {
          "workspace_id": "ws1",
          "user_id": "user.primary",
          "device_type": "desktop",
          "platform": "macos",
          "has_more": true,
          "next_cursor_created_at": "2026-02-25T12:00:00Z",
          "next_cursor_id": "device-1",
          "items": [
            {
              "device_id": "device-1",
              "workspace_id": "ws1",
              "user_id": "user.primary",
              "device_type": "desktop",
              "platform": "macos",
              "label": "Abhishek MacBook",
              "last_seen_at": "2026-02-25T12:05:00Z",
              "created_at": "2026-02-24T09:30:00Z",
              "session_total": 4,
              "session_active_count": 2,
              "session_expired_count": 1,
              "session_revoked_count": 1,
              "session_latest_started_at": "2026-02-25T11:55:00Z"
            }
          ]
        }
        """

        let decoded = try decoder.decode(
            DaemonIdentityDeviceListResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws1")
        XCTAssertEqual(decoded.userID, "user.primary")
        XCTAssertEqual(decoded.deviceType, "desktop")
        XCTAssertEqual(decoded.platform, "macos")
        XCTAssertEqual(decoded.items.count, 1)
        XCTAssertEqual(decoded.items.first?.deviceID, "device-1")
        XCTAssertEqual(decoded.items.first?.sessionActiveCount, 2)
        XCTAssertEqual(decoded.items.first?.sessionRevokedCount, 1)
        XCTAssertTrue(decoded.hasMore)
        XCTAssertEqual(decoded.nextCursorID, "device-1")
    }

    func testIdentitySessionListResponseDecodesSessionHealthMetadata() throws {
        let payload = """
        {
          "workspace_id": "ws1",
          "device_id": "device-1",
          "session_health": "active",
          "has_more": false,
          "items": [
            {
              "session_id": "session-1",
              "workspace_id": "ws1",
              "device_id": "device-1",
              "user_id": "user.primary",
              "device_type": "desktop",
              "platform": "macos",
              "device_label": "Abhishek MacBook",
              "device_last_seen_at": "2026-02-25T12:05:00Z",
              "started_at": "2026-02-25T11:00:00Z",
              "expires_at": "2026-02-26T11:00:00Z",
              "revoked_at": "",
              "session_health": "active"
            }
          ]
        }
        """

        let decoded = try decoder.decode(
            DaemonIdentitySessionListResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws1")
        XCTAssertEqual(decoded.deviceID, "device-1")
        XCTAssertEqual(decoded.sessionHealth, "active")
        XCTAssertEqual(decoded.items.count, 1)
        XCTAssertEqual(decoded.items.first?.sessionID, "session-1")
        XCTAssertEqual(decoded.items.first?.deviceLabel, "Abhishek MacBook")
        XCTAssertEqual(decoded.items.first?.sessionHealth, "active")
        XCTAssertFalse(decoded.hasMore)
    }

    func testIdentitySessionRevokeResponseDecodesIdempotentResult() throws {
        let payload = """
        {
          "workspace_id": "ws1",
          "session_id": "session-1",
          "device_id": "device-1",
          "started_at": "2026-02-25T11:00:00Z",
          "expires_at": "2026-02-26T11:00:00Z",
          "revoked_at": "2026-02-25T12:06:00Z",
          "device_last_seen_at": "2026-02-25T12:05:00Z",
          "session_health": "revoked",
          "idempotent": true
        }
        """

        let decoded = try decoder.decode(
            DaemonIdentitySessionRevokeResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws1")
        XCTAssertEqual(decoded.sessionID, "session-1")
        XCTAssertEqual(decoded.deviceID, "device-1")
        XCTAssertEqual(decoded.sessionHealth, "revoked")
        XCTAssertTrue(decoded.idempotent)
    }

    func testDelegationListResponseDecodesRules() throws {
        let payload = """
        {
          "workspace_id": "ws1",
          "rules": [
            {
              "id": "rule-1",
              "workspace_id": "ws1",
              "from_actor_id": "actor.requester",
              "to_actor_id": "actor.delegate",
              "scope_type": "EXECUTION",
              "scope_key": "chat.send",
              "status": "ACTIVE",
              "created_at": "2026-02-25T00:00:00Z",
              "expires_at": "2026-03-01T00:00:00Z"
            }
          ]
        }
        """

        let decoded = try decoder.decode(
            DaemonDelegationListResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws1")
        XCTAssertEqual(decoded.rules.count, 1)
        XCTAssertEqual(decoded.rules.first?.id, "rule-1")
        XCTAssertEqual(decoded.rules.first?.fromActorID, "actor.requester")
        XCTAssertEqual(decoded.rules.first?.toActorID, "actor.delegate")
        XCTAssertEqual(decoded.rules.first?.scopeType, "EXECUTION")
        XCTAssertEqual(decoded.rules.first?.scopeKey, "chat.send")
    }

    func testDelegationRevokeResponseDefaultsStatusWhenMissing() throws {
        let payload = """
        {
          "workspace_id": "ws1",
          "rule_id": "rule-1"
        }
        """

        let decoded = try decoder.decode(
            DaemonDelegationRevokeResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws1")
        XCTAssertEqual(decoded.ruleID, "rule-1")
        XCTAssertEqual(decoded.status, "revoked")
    }

    func testCapabilityGrantListResponseDecodesSnakeCasePayload() throws {
        let payload = """
        {
          "workspace_id": "ws1",
          "has_more": true,
          "next_cursor_created_at": "2026-02-25T01:02:03Z",
          "next_cursor_id": "grant-1",
          "items": [
            {
              "grant_id": "grant-1",
              "workspace_id": "ws1",
              "actor_id": "actor.requester",
              "capability_key": "messages_send_sms",
              "scope_json": "{\\"channel\\":\\"sms\\"}",
              "status": "ACTIVE",
              "created_at": "2026-02-25T01:02:03Z",
              "expires_at": "2026-03-01T00:00:00Z"
            }
          ]
        }
        """

        let decoded = try decoder.decode(
            DaemonCapabilityGrantListResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws1")
        XCTAssertEqual(decoded.items.count, 1)
        XCTAssertEqual(decoded.items.first?.grantID, "grant-1")
        XCTAssertEqual(decoded.items.first?.actorID, "actor.requester")
        XCTAssertEqual(decoded.items.first?.capabilityKey, "messages_send_sms")
        XCTAssertEqual(decoded.items.first?.status, "ACTIVE")
        XCTAssertTrue(decoded.hasMore)
        XCTAssertEqual(decoded.nextCursorID, "grant-1")
    }

    func testCommWebhookReceiptListResponseDecodesAuditLinksAndTrustState() throws {
        let payload = """
        {
          "workspace_id": "ws1",
          "provider": "twilio",
          "has_more": false,
          "items": [
            {
              "receipt_id": "wr-1",
              "workspace_id": "ws1",
              "provider": "twilio",
              "provider_event_id": "SM123",
              "trust_state": "accepted",
              "signature_valid": true,
              "signature_value_present": true,
              "payload_hash": "hash-wr-1",
              "event_id": "event-1",
              "thread_id": "thread-1",
              "received_at": "2026-02-25T00:00:01Z",
              "created_at": "2026-02-25T00:00:01Z",
              "audit_links": [
                {
                  "audit_id": "audit-1",
                  "event_type": "twilio_webhook_received",
                  "created_at": "2026-02-25T00:00:01Z"
                }
              ]
            }
          ]
        }
        """

        let decoded = try decoder.decode(
            DaemonCommWebhookReceiptListResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws1")
        XCTAssertEqual(decoded.provider, "twilio")
        XCTAssertEqual(decoded.items.count, 1)
        XCTAssertEqual(decoded.items.first?.receiptID, "wr-1")
        XCTAssertEqual(decoded.items.first?.trustState, "accepted")
        XCTAssertEqual(decoded.items.first?.signatureValid, true)
        XCTAssertEqual(decoded.items.first?.auditLinks.count, 1)
        XCTAssertEqual(decoded.items.first?.auditLinks.first?.auditID, "audit-1")
    }

    func testCommIngestReceiptListResponseDecodesAuditLinksAndTrustState() throws {
        let payload = """
        {
          "workspace_id": "ws1",
          "source": "apple_mail_rule",
          "source_scope": "mail-default",
          "has_more": true,
          "next_cursor_created_at": "2026-02-25T00:00:01Z",
          "next_cursor_id": "ir-1",
          "items": [
            {
              "receipt_id": "ir-1",
              "workspace_id": "ws1",
              "source": "apple_mail_rule",
              "source_scope": "mail-default",
              "source_event_id": "mail-event-1",
              "source_cursor": "101",
              "trust_state": "rejected",
              "payload_hash": "hash-ir-1",
              "event_id": null,
              "thread_id": null,
              "received_at": "2026-02-25T00:00:01Z",
              "created_at": "2026-02-25T00:00:01Z",
              "audit_links": [
                {
                  "audit_id": "audit-ir-1",
                  "event_type": "comm_ingest_rejected",
                  "created_at": "2026-02-25T00:00:01Z"
                }
              ]
            }
          ]
        }
        """

        let decoded = try decoder.decode(
            DaemonCommIngestReceiptListResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws1")
        XCTAssertEqual(decoded.source, "apple_mail_rule")
        XCTAssertEqual(decoded.sourceScope, "mail-default")
        XCTAssertEqual(decoded.items.count, 1)
        XCTAssertEqual(decoded.items.first?.receiptID, "ir-1")
        XCTAssertEqual(decoded.items.first?.trustState, "rejected")
        XCTAssertEqual(decoded.items.first?.sourceCursor, "101")
        XCTAssertEqual(decoded.items.first?.auditLinks.count, 1)
        XCTAssertEqual(decoded.items.first?.auditLinks.first?.eventType, "comm_ingest_rejected")
        XCTAssertTrue(decoded.hasMore)
        XCTAssertEqual(decoded.nextCursorID, "ir-1")
    }

    func testChannelConnectorMappingListResponseDecodesCanonicalBindings() throws {
        let payload = """
        {
          "workspace_id": "ws-mappings",
          "fallback_policy": "priority_order",
          "bindings": [
            {
              "channel_id": "message",
              "connector_id": "imessage",
              "enabled": true,
              "priority": 1,
              "capabilities": ["channel.messages.send", "channel.messages.status"],
              "created_at": "2026-02-26T09:00:00Z",
              "updated_at": "2026-02-26T09:01:00Z"
            },
            {
              "channel_id": "message",
              "connector_id": "twilio",
              "enabled": false,
              "priority": 2,
              "capabilities": ["channel.twilio.sms.send"],
              "created_at": "2026-02-26T09:00:00Z",
              "updated_at": "2026-02-26T09:01:00Z"
            }
          ]
        }
        """

        let decoded = try decoder.decode(
            DaemonChannelConnectorMappingListResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "ws-mappings")
        XCTAssertEqual(decoded.fallbackPolicy, "priority_order")
        XCTAssertEqual(decoded.bindings.count, 2)
        XCTAssertEqual(decoded.bindings.first?.channelID, "message")
        XCTAssertEqual(decoded.bindings.first?.connectorID, "imessage")
        XCTAssertEqual(decoded.bindings.first?.enabled, true)
        XCTAssertEqual(decoded.bindings.first?.priority, 1)
        XCTAssertEqual(decoded.bindings.first?.capabilities, ["channel.messages.send", "channel.messages.status"])
    }

    func testChannelConnectorMappingUpsertResponseDefaultsWhenPayloadIsSparse() throws {
        let payload = """
        {
          "bindings": [
            {
              "channel_id": "voice",
              "connector_id": "twilio"
            }
          ]
        }
        """

        let decoded = try decoder.decode(
            DaemonChannelConnectorMappingUpsertResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "")
        XCTAssertEqual(decoded.channelID, "")
        XCTAssertEqual(decoded.connectorID, "")
        XCTAssertEqual(decoded.enabled, false)
        XCTAssertEqual(decoded.priority, 0)
        XCTAssertEqual(decoded.fallbackPolicy, "priority_order")
        XCTAssertEqual(decoded.bindings.count, 1)
        XCTAssertEqual(decoded.bindings.first?.channelID, "voice")
        XCTAssertEqual(decoded.bindings.first?.connectorID, "twilio")
        XCTAssertEqual(decoded.bindings.first?.capabilities, [])
    }

    func testChannelStatusResponseDefaultsWhenPayloadIsSparse() throws {
        let payload = """
        {
          "channels": [
            {
              "channel_id": "message"
            }
          ]
        }
        """

        let decoded = try decoder.decode(
            DaemonChannelStatusResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "")
        XCTAssertEqual(decoded.channels.count, 1)
        XCTAssertEqual(decoded.channels.first?.channelID, "message")
        XCTAssertEqual(decoded.channels.first?.displayName, "message")
        XCTAssertEqual(decoded.channels.first?.status, "unknown")
        XCTAssertNil(decoded.channels.first?.worker)
    }

    func testConnectorStatusResponseDefaultsWhenPayloadIsSparse() throws {
        let payload = """
        {
          "connectors": [
            {
              "connector_id": "twilio"
            }
          ]
        }
        """

        let decoded = try decoder.decode(
            DaemonConnectorStatusResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "")
        XCTAssertEqual(decoded.connectors.count, 1)
        XCTAssertEqual(decoded.connectors.first?.connectorID, "twilio")
        XCTAssertEqual(decoded.connectors.first?.pluginID, "twilio")
        XCTAssertEqual(decoded.connectors.first?.status, "unknown")
        XCTAssertNil(decoded.connectors.first?.worker)
    }

    func testChannelDiagnosticsResponseDefaultsWhenPayloadIsSparse() throws {
        let payload = """
        {
          "diagnostics": [
            {
              "channel_id": "voice"
            }
          ]
        }
        """

        let decoded = try decoder.decode(
            DaemonChannelDiagnosticsResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "")
        XCTAssertEqual(decoded.diagnostics.count, 1)
        XCTAssertEqual(decoded.diagnostics.first?.channelID, "voice")
        XCTAssertEqual(decoded.diagnostics.first?.status, "unknown")
        XCTAssertEqual(decoded.diagnostics.first?.workerHealth.registered, false)
        XCTAssertEqual(decoded.diagnostics.first?.remediationActions.count, 0)
    }

    func testConnectorDiagnosticsResponseDefaultsWhenPayloadIsSparse() throws {
        let payload = """
        {
          "diagnostics": [
            {
              "connector_id": "imessage"
            }
          ]
        }
        """

        let decoded = try decoder.decode(
            DaemonConnectorDiagnosticsResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "")
        XCTAssertEqual(decoded.diagnostics.count, 1)
        XCTAssertEqual(decoded.diagnostics.first?.connectorID, "imessage")
        XCTAssertEqual(decoded.diagnostics.first?.pluginID, "imessage")
        XCTAssertEqual(decoded.diagnostics.first?.status, "unknown")
        XCTAssertEqual(decoded.diagnostics.first?.workerHealth.registered, false)
        XCTAssertEqual(decoded.diagnostics.first?.remediationActions.count, 0)
    }

    func testInspectLogQueryResponseDefaultsWhenEnvelopeFieldsMissing() throws {
        let payload = """
        {}
        """

        let decoded = try decoder.decode(
            DaemonInspectLogQueryResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "")
        XCTAssertEqual(decoded.logs.count, 0)
        XCTAssertNil(decoded.nextCursorCreatedAt)
        XCTAssertNil(decoded.nextCursorID)
    }

    func testInspectLogStreamResponseDefaultsWhenEnvelopeFieldsMissing() throws {
        let payload = """
        {}
        """

        let decoded = try decoder.decode(
            DaemonInspectLogStreamResponse.self,
            from: Data(payload.utf8)
        )

        XCTAssertEqual(decoded.workspaceID, "")
        XCTAssertEqual(decoded.logs.count, 0)
        XCTAssertNil(decoded.cursorCreatedAt)
        XCTAssertNil(decoded.cursorID)
        XCTAssertFalse(decoded.timedOut)
    }
}
