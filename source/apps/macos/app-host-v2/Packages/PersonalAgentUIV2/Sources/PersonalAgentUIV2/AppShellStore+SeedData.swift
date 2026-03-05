import Foundation

@MainActor
extension AppShellV2Store {
    public static let defaultConnectors: [ConnectorState] = [
        ConnectorState(
            id: "imessage",
            name: "iMessage",
            status: .connected,
            summary: "Connected and healthy.",
            enabled: true,
            configured: true,
            actionReadiness: "ready",
            configurationBaseline: ["enabled": "true"],
            lastCheckAt: Date().addingTimeInterval(-900),
            lastCheckSummary: "Connection healthy.",
            lastCheckSucceeded: true
        ),
        ConnectorState(
            id: "whatsapp",
            name: "WhatsApp",
            status: .connected,
            summary: "Connected and healthy.",
            enabled: true,
            configured: true,
            actionReadiness: "ready",
            configurationBaseline: ["enabled": "true"],
            lastCheckAt: Date().addingTimeInterval(-1200),
            lastCheckSummary: "Connection healthy.",
            lastCheckSucceeded: true
        ),
        ConnectorState(
            id: "telegram",
            name: "Telegram",
            status: .needsAttention,
            summary: "Permission is missing.",
            enabled: false,
            configured: false,
            actionReadiness: "blocked",
            actionBlockers: [
                V2DaemonActionReadinessBlocker(
                    code: "permission_missing",
                    message: "System permission is required before this connector can run.",
                    remediationAction: "request_permission"
                )
            ],
            configurationBaseline: ["enabled": "false"],
            lastCheckAt: Date().addingTimeInterval(-7200),
            lastCheckSummary: "Permission check failed.",
            lastCheckSucceeded: false
        ),
        ConnectorState(
            id: "email",
            name: "Email",
            status: .connected,
            summary: "Connected and healthy.",
            enabled: true,
            configured: true,
            actionReadiness: "ready",
            configurationBaseline: ["enabled": "true"],
            lastCheckAt: Date().addingTimeInterval(-1500),
            lastCheckSummary: "Connection healthy.",
            lastCheckSucceeded: true
        )
    ]

    public static let defaultModels: [ModelOption] = [
        ModelOption(
            providerID: "built_in",
            providerName: "Built-In",
            modelKey: "personalagent_default",
            enabled: true,
            providerReady: true,
            providerEndpoint: "local"
        ),
        ModelOption(
            providerID: "openai",
            providerName: "OpenAI",
            modelKey: "gpt-4.1",
            enabled: true,
            providerReady: true,
            providerEndpoint: "https://api.openai.com/v1"
        ),
        ModelOption(
            providerID: "anthropic",
            providerName: "Anthropic",
            modelKey: "claude-sonnet",
            enabled: false,
            providerReady: false,
            providerEndpoint: "https://api.anthropic.com/v1"
        )
    ]

    public static let defaultReplayEvents: [ReplayEvent] = [
        ReplayEvent(
            source: .iMessage,
            sourceContext: .iMessage(
                IMessageReplaySourceContext(
                    contactName: "Alex Rivera",
                    contactPhoneSuffix: "+1 •••• 4721",
                    threadID: "imsg-thread-284"
                )
            ),
            receivedAt: Date().addingTimeInterval(-320),
            instruction: "Move Friday standup to 10:30 and tell the team in Slack.",
            interpretedIntent: "Reschedule an event and broadcast a team update.",
            actionSummary: "Calendar hold moved and Slack draft prepared.",
            status: .awaitingApproval,
            risk: .low,
            approvalReason: "This changes a shared meeting time.",
            channelsTouched: ["calendar", "slack"],
            decisionTrace: trace(
                received: "Instruction captured from iMessage.",
                intent: "Mapped to calendar update + team notification workflow.",
                planning: "Prepared calendar mutation and Slack draft for confirmation.",
                execution: "Waiting for user approval before broadcasting changes.",
                executionStatus: .pending
            ),
            confidenceScore: 91
        ),
        ReplayEvent(
            source: .email,
            sourceContext: .email(
                EmailReplaySourceContext(
                    sender: "ceo@company.com",
                    subject: "Send revised contract to vendor today",
                    mailbox: "VIP"
                )
            ),
            receivedAt: Date().addingTimeInterval(-730),
            instruction: "Send revised contract to Acme and authorize legal wording changes.",
            interpretedIntent: "Dispatch a legal contract draft and approve wording updates.",
            actionSummary: "Draft prepared. Waiting for high-risk approval.",
            status: .awaitingApproval,
            risk: .high,
            approvalReason: "Legal and financial exposure requires explicit approval note.",
            channelsTouched: ["email", "contracts"],
            decisionTrace: trace(
                received: "Request parsed from executive email.",
                intent: "Mapped to legal dispatch workflow.",
                planning: "Prepared contract package and compliance checks.",
                execution: "Blocked until high-risk approval decision is recorded.",
                executionStatus: .pending
            ),
            confidenceScore: 88
        ),
        ReplayEvent(
            source: .whatsapp,
            sourceContext: .whatsapp(
                WhatsAppReplaySourceContext(
                    contactName: "Jordan Lee",
                    chatID: "wa-338199",
                    phoneSuffix: "+1 •••• 9088"
                )
            ),
            receivedAt: Date().addingTimeInterval(-1120),
            instruction: "Tell client we received their files and I'll respond by 5 PM.",
            interpretedIntent: "Acknowledge receipt and commit to response timeline.",
            actionSummary: "Response generation running.",
            status: .running,
            risk: .low,
            channelsTouched: ["whatsapp"],
            decisionTrace: trace(
                received: "Message ingested from WhatsApp.",
                intent: "Classified as client acknowledgment.",
                planning: "Built response with SLA wording.",
                execution: "Sending response now.",
                executionStatus: .pending
            ),
            confidenceScore: 93
        ),
        ReplayEvent(
            source: .email,
            sourceContext: .email(
                EmailReplaySourceContext(
                    sender: "support@company.com",
                    subject: "Summarize today's customer escalation thread",
                    mailbox: "Important"
                )
            ),
            receivedAt: Date().addingTimeInterval(-1800),
            instruction: "Summarize today's customer escalation thread.",
            interpretedIntent: "Generate concise executive digest for active escalation.",
            actionSummary: "Digest delivered to Inbox/Important folder.",
            status: .completed,
            risk: .low,
            channelsTouched: ["email"],
            decisionTrace: trace(
                received: "Thread request captured from email.",
                intent: "Classified as executive summary generation.",
                planning: "Extracted escalation milestones and key risks.",
                execution: "Sent summary response back to your inbox.",
                executionStatus: .completed
            ),
            confidenceScore: 95
        ),
        ReplayEvent(
            source: .voice,
            sourceContext: .voice(
                VoiceReplaySourceContext(
                    deviceName: "iPhone 17 Pro",
                    transcriptConfidence: 86,
                    utteranceDurationSeconds: 9
                )
            ),
            receivedAt: Date().addingTimeInterval(-4600),
            instruction: "Book travel to NYC next Tuesday morning.",
            interpretedIntent: "Assemble flight options respecting travel policy constraints.",
            actionSummary: "Policy conflict on fare cap blocked automatic booking.",
            status: .failed,
            risk: .medium,
            approvalReason: "Travel budget cap exceeded by preferred options.",
            channelsTouched: ["travel", "policy"],
            decisionTrace: trace(
                received: "Voice request transcribed and validated.",
                intent: "Mapped to travel-booking workflow with policy checks.",
                planning: "Generated compliant options and compared price caps.",
                execution: "Stopped booking due to policy cap mismatch.",
                executionStatus: .blocked
            ),
            confidenceScore: 84,
            failureRecoveryHint: "Adjust travel policy cap or confirm manual exception approval, then retry."
        ),
        ReplayEvent(
            source: .app,
            sourceContext: .app(
                AppReplaySourceContext(
                    workspace: "Personal",
                    sessionID: "session-a17c4f",
                    messageID: "msg-daily-plan"
                )
            ),
            receivedAt: Date().addingTimeInterval(-9200),
            instruction: "Draft a concise daily plan from my task list.",
            interpretedIntent: "Prioritize tasks and generate a plan for today.",
            actionSummary: "Plan posted in app chat.",
            status: .completed,
            risk: .low,
            channelsTouched: ["tasks", "app_chat"],
            decisionTrace: trace(
                received: "Instruction captured from app chat.",
                intent: "Classified as planning and prioritization.",
                planning: "Ranked tasks by urgency and meeting schedule.",
                execution: "Posted concise plan with top priorities.",
                executionStatus: .completed
            ),
            confidenceScore: 96
        )
    ]

    static func trace(
        received: String,
        intent: String,
        planning: String,
        execution: String,
        executionStatus: ReplayDecisionStageStatus
    ) -> [ReplayDecisionStage] {
        [
            ReplayDecisionStage(title: "Instruction Received", detail: received, status: .completed),
            ReplayDecisionStage(title: "Intent Interpreted", detail: intent, status: .completed),
            ReplayDecisionStage(title: "Action Plan Built", detail: planning, status: .completed),
            ReplayDecisionStage(title: "Execution", detail: execution, status: executionStatus)
        ]
    }

    static func markTraceAsCompleted(_ trace: [ReplayDecisionStage]) -> [ReplayDecisionStage] {
        trace.map { stage in
            var updated = stage
            updated.status = .completed
            return updated
        }
    }

    static func markTraceExecutionAsBlocked(_ trace: [ReplayDecisionStage]) -> [ReplayDecisionStage] {
        trace.map { stage in
            guard stage.title == "Execution" else { return stage }
            var updated = stage
            updated.status = .blocked
            return updated
        }
    }

    static func resetBlockedTraceToPending(_ trace: [ReplayDecisionStage]) -> [ReplayDecisionStage] {
        trace.map { stage in
            var updated = stage
            if updated.status == .blocked {
                updated.status = .pending
            }
            return updated
        }
    }
}
