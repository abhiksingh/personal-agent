import Foundation

public enum ReplaySource: String, CaseIterable, Identifiable, Hashable {
    case app
    case iMessage
    case whatsapp
    case telegram
    case email
    case voice

    public var id: String { rawValue }

    var label: String {
        switch self {
        case .app:
            return "App Chat"
        case .iMessage:
            return "iMessage"
        case .whatsapp:
            return "WhatsApp"
        case .telegram:
            return "Telegram"
        case .email:
            return "Email"
        case .voice:
            return "Voice"
        }
    }

    var systemImage: String {
        switch self {
        case .app:
            return "bubble.left.and.bubble.right"
        case .iMessage:
            return "message"
        case .whatsapp:
            return "phone.bubble"
        case .telegram:
            return "paperplane"
        case .email:
            return "envelope"
        case .voice:
            return "waveform"
        }
    }
}

public enum ReplayEventStatus: String {
    case completed
    case awaitingApproval
    case failed
    case running

    var label: String {
        switch self {
        case .completed:
            return "Completed"
        case .awaitingApproval:
            return "Needs Approval"
        case .failed:
            return "Failed"
        case .running:
            return "Running"
        }
    }

    var systemImage: String {
        switch self {
        case .completed:
            return "checkmark.circle"
        case .awaitingApproval:
            return "hand.raised"
        case .failed:
            return "exclamationmark.triangle"
        case .running:
            return "clock"
        }
    }
}

public enum ReplayStatusFilter: String, CaseIterable, Identifiable {
    case all
    case waiting
    case needsApproval
    case completed
    case failed
    case running

    public var id: String { rawValue }

    var label: String {
        switch self {
        case .all:
            return "All"
        case .waiting:
            return "Needs Attention"
        case .needsApproval:
            return "Needs Approval"
        case .completed:
            return "Completed"
        case .failed:
            return "Failed"
        case .running:
            return "Running"
        }
    }

    func matches(_ status: ReplayEventStatus) -> Bool {
        switch self {
        case .all:
            return true
        case .waiting:
            return status == .awaitingApproval || status == .running
        case .needsApproval:
            return status == .awaitingApproval
        case .completed:
            return status == .completed
        case .failed:
            return status == .failed
        case .running:
            return status == .running
        }
    }
}

public enum ReplayRiskLevel: String {
    case low
    case medium
    case high

    var label: String {
        rawValue.capitalized
    }
}

public enum ApprovalDecisionAction: String, CaseIterable, Identifiable {
    case approve
    case reject

    public var id: String { rawValue }

    var title: String {
        switch self {
        case .approve:
            return "Approve and Execute"
        case .reject:
            return "Reject"
        }
    }
}

public enum ReplayDecisionStageStatus: String {
    case completed
    case pending
    case blocked

    var label: String {
        switch self {
        case .completed:
            return "Completed"
        case .pending:
            return "Pending"
        case .blocked:
            return "Blocked"
        }
    }

    var systemImage: String {
        switch self {
        case .completed:
            return "checkmark.circle.fill"
        case .pending:
            return "clock"
        case .blocked:
            return "exclamationmark.triangle.fill"
        }
    }
}

public struct ReplayDecisionStage: Identifiable, Hashable {
    public let id: UUID
    public var title: String
    public var detail: String
    public var status: ReplayDecisionStageStatus

    public init(id: UUID = UUID(), title: String, detail: String, status: ReplayDecisionStageStatus) {
        self.id = id
        self.title = title
        self.detail = detail
        self.status = status
    }
}

public struct ApprovalAuditEntry: Identifiable, Hashable {
    public let id: UUID
    public var decidedAt: Date
    public var action: ApprovalDecisionAction
    public var actor: String
    public var note: String

    public init(
        id: UUID = UUID(),
        decidedAt: Date,
        action: ApprovalDecisionAction,
        actor: String,
        note: String
    ) {
        self.id = id
        self.decidedAt = decidedAt
        self.action = action
        self.actor = actor
        self.note = note
    }
}

public struct ReplaySourceContextField: Identifiable, Hashable {
    public let id: String
    public var label: String
    public var value: String

    public init(label: String, value: String) {
        self.id = label
        self.label = label
        self.value = value
    }
}

public struct AppReplaySourceContext: Hashable {
    public var workspace: String
    public var sessionID: String
    public var messageID: String

    public init(workspace: String, sessionID: String, messageID: String) {
        self.workspace = workspace
        self.sessionID = sessionID
        self.messageID = messageID
    }
}

public struct IMessageReplaySourceContext: Hashable {
    public var contactName: String
    public var contactPhoneSuffix: String
    public var threadID: String

    public init(contactName: String, contactPhoneSuffix: String, threadID: String) {
        self.contactName = contactName
        self.contactPhoneSuffix = contactPhoneSuffix
        self.threadID = threadID
    }
}

public struct WhatsAppReplaySourceContext: Hashable {
    public var contactName: String
    public var chatID: String
    public var phoneSuffix: String

    public init(contactName: String, chatID: String, phoneSuffix: String) {
        self.contactName = contactName
        self.chatID = chatID
        self.phoneSuffix = phoneSuffix
    }
}

public struct TelegramReplaySourceContext: Hashable {
    public var handle: String
    public var chatID: String
    public var botID: String

    public init(handle: String, chatID: String, botID: String) {
        self.handle = handle
        self.chatID = chatID
        self.botID = botID
    }
}

public struct EmailReplaySourceContext: Hashable {
    public var sender: String
    public var subject: String
    public var mailbox: String

    public init(sender: String, subject: String, mailbox: String) {
        self.sender = sender
        self.subject = subject
        self.mailbox = mailbox
    }
}

public struct VoiceReplaySourceContext: Hashable {
    public var deviceName: String
    public var transcriptConfidence: Int
    public var utteranceDurationSeconds: Int

    public init(deviceName: String, transcriptConfidence: Int, utteranceDurationSeconds: Int) {
        self.deviceName = deviceName
        self.transcriptConfidence = transcriptConfidence
        self.utteranceDurationSeconds = utteranceDurationSeconds
    }
}

public enum ReplaySourceContext: Hashable {
    case app(AppReplaySourceContext)
    case iMessage(IMessageReplaySourceContext)
    case whatsapp(WhatsAppReplaySourceContext)
    case telegram(TelegramReplaySourceContext)
    case email(EmailReplaySourceContext)
    case voice(VoiceReplaySourceContext)

    var fields: [ReplaySourceContextField] {
        switch self {
        case .app(let context):
            return [
                ReplaySourceContextField(label: "Workspace", value: context.workspace),
                ReplaySourceContextField(label: "Session ID", value: context.sessionID),
                ReplaySourceContextField(label: "Message ID", value: context.messageID)
            ]
        case .iMessage(let context):
            return [
                ReplaySourceContextField(label: "Contact", value: context.contactName),
                ReplaySourceContextField(label: "Phone", value: context.contactPhoneSuffix),
                ReplaySourceContextField(label: "Thread ID", value: context.threadID)
            ]
        case .whatsapp(let context):
            return [
                ReplaySourceContextField(label: "Contact", value: context.contactName),
                ReplaySourceContextField(label: "Chat ID", value: context.chatID),
                ReplaySourceContextField(label: "Phone", value: context.phoneSuffix)
            ]
        case .telegram(let context):
            return [
                ReplaySourceContextField(label: "Handle", value: context.handle),
                ReplaySourceContextField(label: "Chat ID", value: context.chatID),
                ReplaySourceContextField(label: "Bot", value: context.botID)
            ]
        case .email(let context):
            return [
                ReplaySourceContextField(label: "Sender", value: context.sender),
                ReplaySourceContextField(label: "Subject", value: context.subject),
                ReplaySourceContextField(label: "Mailbox", value: context.mailbox)
            ]
        case .voice(let context):
            return [
                ReplaySourceContextField(label: "Device", value: context.deviceName),
                ReplaySourceContextField(label: "Transcript Confidence", value: "\(context.transcriptConfidence)%"),
                ReplaySourceContextField(label: "Duration", value: "\(context.utteranceDurationSeconds)s")
            ]
        }
    }

    var searchableText: String {
        fields.map(\.value).joined(separator: " ")
    }

    static func placeholder(for source: ReplaySource) -> ReplaySourceContext {
        switch source {
        case .app:
            return .app(AppReplaySourceContext(workspace: "Default", sessionID: "session-local", messageID: "msg-local"))
        case .iMessage:
            return .iMessage(IMessageReplaySourceContext(contactName: "Unknown", contactPhoneSuffix: "••••", threadID: "imessage-thread"))
        case .whatsapp:
            return .whatsapp(WhatsAppReplaySourceContext(contactName: "Unknown", chatID: "wa-chat", phoneSuffix: "••••"))
        case .telegram:
            return .telegram(TelegramReplaySourceContext(handle: "@unknown", chatID: "tg-chat", botID: "tg-bot"))
        case .email:
            return .email(EmailReplaySourceContext(sender: "unknown@example.com", subject: "Unknown", mailbox: "Inbox"))
        case .voice:
            return .voice(VoiceReplaySourceContext(deviceName: "Unknown Device", transcriptConfidence: 70, utteranceDurationSeconds: 3))
        }
    }
}

public struct ReplayEvent: Identifiable, Hashable {
    public let id: UUID
    public var replayKey: String
    public var source: ReplaySource
    public var sourceContext: ReplaySourceContext
    public var receivedAt: Date
    public var instruction: String
    public var interpretedIntent: String
    public var actionSummary: String
    public var status: ReplayEventStatus
    public var risk: ReplayRiskLevel
    public var approvalReason: String?
    public var channelsTouched: [String]
    public var decisionTrace: [ReplayDecisionStage]
    public var confidenceScore: Int
    public var failureRecoveryHint: String?
    public var approvalAudit: [ApprovalAuditEntry]
    public var daemonLocator: ReplayEventDaemonLocator?

    public init(
        id: UUID = UUID(),
        replayKey: String? = nil,
        source: ReplaySource,
        sourceContext: ReplaySourceContext? = nil,
        receivedAt: Date,
        instruction: String,
        interpretedIntent: String,
        actionSummary: String,
        status: ReplayEventStatus,
        risk: ReplayRiskLevel,
        approvalReason: String? = nil,
        channelsTouched: [String],
        decisionTrace: [ReplayDecisionStage] = [],
        confidenceScore: Int = 80,
        failureRecoveryHint: String? = nil,
        approvalAudit: [ApprovalAuditEntry] = [],
        daemonLocator: ReplayEventDaemonLocator? = nil
    ) {
        self.id = id
        self.replayKey = replayKey ?? id.uuidString.lowercased()
        self.source = source
        self.sourceContext = sourceContext ?? ReplaySourceContext.placeholder(for: source)
        self.receivedAt = receivedAt
        self.instruction = instruction
        self.interpretedIntent = interpretedIntent
        self.actionSummary = actionSummary
        self.status = status
        self.risk = risk
        self.approvalReason = approvalReason
        self.channelsTouched = channelsTouched
        self.decisionTrace = decisionTrace
        self.confidenceScore = max(0, min(100, confidenceScore))
        self.failureRecoveryHint = failureRecoveryHint
        self.approvalAudit = approvalAudit.sorted(by: { $0.decidedAt > $1.decidedAt })
        self.daemonLocator = daemonLocator
    }

    var canInlineApprove: Bool {
        status == .awaitingApproval && risk == .low
    }

    var requiresManualApproval: Bool {
        status == .awaitingApproval
    }

    var inlineApprovalDisabledReason: String? {
        guard requiresManualApproval else {
            return nil
        }
        guard !canInlineApprove else {
            return nil
        }
        return "High-impact request: verify intent and impact before approving."
    }

    var searchableText: String {
        [
            source.label,
            instruction,
            interpretedIntent,
            actionSummary,
            channelsTouched.joined(separator: " "),
            approvalReason ?? "",
            sourceContext.searchableText
        ].joined(separator: " ")
    }
}

public struct ReplayEventDaemonLocator: Hashable, Sendable {
    public var correlationID: String?
    public var turnID: String?
    public var historyRecordIDs: [String]
    public var approvalRequestID: String?
    public var taskID: String?
    public var runID: String?
    public var channelID: String?

    public init(
        correlationID: String? = nil,
        turnID: String? = nil,
        historyRecordIDs: [String] = [],
        approvalRequestID: String? = nil,
        taskID: String? = nil,
        runID: String? = nil,
        channelID: String? = nil
    ) {
        self.correlationID = correlationID
        self.turnID = turnID
        self.historyRecordIDs = historyRecordIDs
        self.approvalRequestID = approvalRequestID
        self.taskID = taskID
        self.runID = runID
        self.channelID = channelID
    }
}

public struct V2ReplayFeedQueryState: Sendable, Equatable {
    public var pageSize: Int
    public var requestedPage: Int
    public var hasLoadedOnce: Bool
    public var isRefreshing: Bool
    public var isLoadingMore: Bool
    public var canLoadMore: Bool
    public var lastLoadedAt: Date?
    public var lastErrorSummary: String?

    public init(
        pageSize: Int = 24,
        requestedPage: Int = 1,
        hasLoadedOnce: Bool = false,
        isRefreshing: Bool = false,
        isLoadingMore: Bool = false,
        canLoadMore: Bool = false,
        lastLoadedAt: Date? = nil,
        lastErrorSummary: String? = nil
    ) {
        self.pageSize = max(12, pageSize)
        self.requestedPage = max(1, requestedPage)
        self.hasLoadedOnce = hasLoadedOnce
        self.isRefreshing = isRefreshing
        self.isLoadingMore = isLoadingMore
        self.canLoadMore = canLoadMore
        self.lastLoadedAt = lastLoadedAt
        self.lastErrorSummary = lastErrorSummary
    }

    public var approvalLimit: Int {
        pageSize * requestedPage
    }

    public var taskLimit: Int {
        pageSize * requestedPage
    }

    public var historyLimit: Int {
        pageSize * requestedPage
    }
}

public enum V2ReplayRealtimePhase: String, Sendable, Equatable {
    case idle
    case connecting
    case connected
    case reconnecting
    case disconnected
}

public struct V2ReplayRealtimeState: Sendable, Equatable {
    public var phase: V2ReplayRealtimePhase
    public var lastEventAt: Date?
    public var lastErrorSummary: String?
    public var reconnectAttempt: Int

    public init(
        phase: V2ReplayRealtimePhase = .idle,
        lastEventAt: Date? = nil,
        lastErrorSummary: String? = nil,
        reconnectAttempt: Int = 0
    ) {
        self.phase = phase
        self.lastEventAt = lastEventAt
        self.lastErrorSummary = lastErrorSummary
        self.reconnectAttempt = max(0, reconnectAttempt)
    }
}

public enum V2ReplayDetailEvidencePhase: String, Equatable {
    case loading
    case ready
    case empty
    case failed
}

public struct V2ReplayDetailEvidenceState: Equatable {
    public var replayKey: String
    public var phase: V2ReplayDetailEvidencePhase
    public var summary: String?
    public var whatCameIn: String?
    public var whatAssistantUnderstood: String?
    public var whatHappened: String?
    public var approvalContext: String?
    public var sourceContextFields: [ReplaySourceContextField]?
    public var decisionTrace: [ReplayDecisionStage]?
    public var channelsTouched: [String]?
    public var confidenceScore: Int?
    public var failureHint: String?
    public var lastUpdatedAt: Date?

    public init(
        replayKey: String,
        phase: V2ReplayDetailEvidencePhase,
        summary: String? = nil,
        whatCameIn: String? = nil,
        whatAssistantUnderstood: String? = nil,
        whatHappened: String? = nil,
        approvalContext: String? = nil,
        sourceContextFields: [ReplaySourceContextField]? = nil,
        decisionTrace: [ReplayDecisionStage]? = nil,
        channelsTouched: [String]? = nil,
        confidenceScore: Int? = nil,
        failureHint: String? = nil,
        lastUpdatedAt: Date? = nil
    ) {
        self.replayKey = replayKey
        self.phase = phase
        self.summary = summary
        self.whatCameIn = whatCameIn
        self.whatAssistantUnderstood = whatAssistantUnderstood
        self.whatHappened = whatHappened
        self.approvalContext = approvalContext
        self.sourceContextFields = sourceContextFields
        self.decisionTrace = decisionTrace
        self.channelsTouched = channelsTouched
        self.confidenceScore = confidenceScore
        self.failureHint = failureHint
        self.lastUpdatedAt = lastUpdatedAt
    }
}
