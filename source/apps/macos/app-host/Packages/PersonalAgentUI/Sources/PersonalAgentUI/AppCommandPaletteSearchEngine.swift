import Foundation

struct CommandPaletteObjectSearchCandidate: Sendable {
    let item: CommandPaletteObjectSearchItem
    let searchableValues: [String]
}

enum AppCommandPaletteSearchEngine {
    static func actionItems(
        onboardingFixNextStepDetail: String?,
        recentUsage: [AppCommandActionID],
        isEnabled: (AppCommandActionID) -> Bool,
        disabledReason: (AppCommandActionID) -> String?
    ) -> [AppCommandActionItem] {
        let baseItems: [AppCommandActionItem] = [
            appCommandActionItem(
                .doSendMessage,
                category: .workflow,
                title: "Do: Send a Message",
                subtitle: "Open Chat with a send-message starter prompt.",
                symbolName: "paperplane.fill",
                shortcutHint: nil,
                isEnabled: isEnabled,
                disabledReason: disabledReason
            ),
            appCommandActionItem(
                .doSendEmail,
                category: .workflow,
                title: "Do: Send an Email",
                subtitle: "Open Chat with an email draft-and-send starter prompt.",
                symbolName: "envelope.fill",
                shortcutHint: nil,
                isEnabled: isEnabled,
                disabledReason: disabledReason
            ),
            appCommandActionItem(
                .doCreateTask,
                category: .workflow,
                title: "Do: Create a Task",
                subtitle: "Open Tasks and start a new task draft.",
                symbolName: "plus.rectangle.on.rectangle",
                shortcutHint: nil,
                isEnabled: isEnabled,
                disabledReason: disabledReason
            ),
            appCommandActionItem(
                .doReviewApprovals,
                category: .workflow,
                title: "Do: Review Approvals",
                subtitle: "Open Approvals and continue pending decisions.",
                symbolName: "checkmark.shield.fill",
                shortcutHint: nil,
                isEnabled: isEnabled,
                disabledReason: disabledReason
            ),
            appCommandActionItem(
                .doInspectIssue,
                category: .workflow,
                title: "Do: Inspect an Issue",
                subtitle: "Open Inspect and review recent runtime activity.",
                symbolName: "waveform.path.ecg.rectangle",
                shortcutHint: nil,
                isEnabled: isEnabled,
                disabledReason: disabledReason
            ),
            appCommandActionItem(
                .openConfiguration,
                category: .navigation,
                title: "Open Configuration",
                subtitle: "Workspace, identity, runtime, and setup controls.",
                symbolName: AppSection.configuration.symbolName,
                shortcutHint: "⌘1",
                isEnabled: isEnabled,
                disabledReason: disabledReason
            ),
            appCommandActionItem(
                .openChat,
                category: .navigation,
                title: "Open Chat",
                subtitle: "Conversation surface and prompt composer.",
                symbolName: AppSection.chat.symbolName,
                shortcutHint: "⌘2",
                isEnabled: isEnabled,
                disabledReason: disabledReason
            ),
            appCommandActionItem(
                .openCommunications,
                category: .navigation,
                title: "Open Communications",
                subtitle: "Threads, events, and delivery attempt timeline.",
                symbolName: AppSection.communications.symbolName,
                shortcutHint: "⌘3",
                isEnabled: isEnabled,
                disabledReason: disabledReason
            ),
            appCommandActionItem(
                .openAutomation,
                category: .navigation,
                title: "Open Automation",
                subtitle: "Trigger inventory, simulations, and fire history.",
                symbolName: AppSection.automation.symbolName,
                shortcutHint: "⌘4",
                isEnabled: isEnabled,
                disabledReason: disabledReason
            ),
            appCommandActionItem(
                .openApprovals,
                category: .navigation,
                title: "Open Approvals",
                subtitle: "Pending/final decisions with evidence drill-ins.",
                symbolName: AppSection.approvals.symbolName,
                shortcutHint: "⌘5",
                isEnabled: isEnabled,
                disabledReason: disabledReason
            ),
            appCommandActionItem(
                .openTasks,
                category: .navigation,
                title: "Open Tasks",
                subtitle: "Task/run inventory, details, and workflow links.",
                symbolName: AppSection.tasks.symbolName,
                shortcutHint: "⌘6",
                isEnabled: isEnabled,
                disabledReason: disabledReason
            ),
            appCommandActionItem(
                .openModels,
                category: .navigation,
                title: "Open Models",
                subtitle: "Provider setup, catalog, and route policy.",
                symbolName: AppSection.models.symbolName,
                shortcutHint: "⌘0",
                isEnabled: isEnabled,
                disabledReason: disabledReason
            ),
            appCommandActionItem(
                .openInspect,
                category: .diagnostics,
                title: "Open Inspect",
                subtitle: "LIFO operational logs and trace diagnostics.",
                symbolName: AppSection.inspect.symbolName,
                shortcutHint: "⌘7 / ⌥⌘I",
                isEnabled: isEnabled,
                disabledReason: disabledReason
            ),
            appCommandActionItem(
                .openChannels,
                category: .diagnostics,
                title: "Open Channels",
                subtitle: "Channel mappings, delivery policy, and health.",
                symbolName: AppSection.channels.symbolName,
                shortcutHint: "⌘8 / ⌥⌘C",
                isEnabled: isEnabled,
                disabledReason: disabledReason
            ),
            appCommandActionItem(
                .openConnectors,
                category: .diagnostics,
                title: "Open Connectors",
                subtitle: "Connector status, permissions, and remediation.",
                symbolName: AppSection.connectors.symbolName,
                shortcutHint: "⌘9 / ⌥⌘K",
                isEnabled: isEnabled,
                disabledReason: disabledReason
            ),
            appCommandActionItem(
                .refreshCurrentSection,
                category: .workflow,
                title: "Refresh Current Panel",
                subtitle: "Re-run data queries for the active section.",
                symbolName: "arrow.clockwise",
                shortcutHint: "⌘R",
                isEnabled: isEnabled,
                disabledReason: disabledReason
            ),
            appCommandActionItem(
                .openNotificationCenter,
                category: .workflow,
                title: "Open Notification Center",
                subtitle: "View cross-panel action outcomes and activity log.",
                symbolName: "bell",
                shortcutHint: "⇧⌘N",
                isEnabled: isEnabled,
                disabledReason: disabledReason
            ),
            appCommandActionItem(
                .setSimpleDensityMode,
                category: .workflow,
                title: "Set Density: Simple",
                subtitle: "Hide low-level metadata and internal diagnostics labels.",
                symbolName: AppInformationDensityMode.simple.symbolName,
                shortcutHint: nil,
                isEnabled: isEnabled,
                disabledReason: disabledReason
            ),
            appCommandActionItem(
                .setAdvancedDensityMode,
                category: .workflow,
                title: "Set Density: Advanced",
                subtitle: "Show full operator metadata and trace internals.",
                symbolName: AppInformationDensityMode.advanced.symbolName,
                shortcutHint: nil,
                isEnabled: isEnabled,
                disabledReason: disabledReason
            ),
            appCommandActionItem(
                .performOnboardingFixNextStep,
                category: .workflow,
                title: "Run Fix Next Setup Step",
                subtitle: onboardingFixNextStepDetail ?? "Resolve the top setup blocker from onboarding checks.",
                symbolName: "wrench.and.screwdriver",
                shortcutHint: "⌥⌘F",
                isEnabled: isEnabled,
                disabledReason: disabledReason
            ),
            appCommandActionItem(
                .startDaemon,
                category: .runtime,
                title: "Start Daemon",
                subtitle: "Start runtime workers and control plane.",
                symbolName: "play.fill",
                shortcutHint: "⌥⌘S",
                isEnabled: isEnabled,
                disabledReason: disabledReason
            ),
            appCommandActionItem(
                .stopDaemon,
                category: .runtime,
                title: "Stop Daemon",
                subtitle: "Stop runtime workers and control plane.",
                symbolName: "stop.fill",
                shortcutHint: "⌥⌘.",
                isEnabled: isEnabled,
                disabledReason: disabledReason
            ),
            appCommandActionItem(
                .restartDaemon,
                category: .runtime,
                title: "Restart Daemon",
                subtitle: "Restart daemon process and refresh lifecycle status.",
                symbolName: "arrow.clockwise.circle",
                shortcutHint: "⌥⌘R",
                isEnabled: isEnabled,
                disabledReason: disabledReason
            )
        ]

        let recentRankByID = Dictionary(
            uniqueKeysWithValues: recentUsage.enumerated().map { index, actionID in
                (actionID, index)
            }
        )
        let categoryRankByID = Dictionary(
            uniqueKeysWithValues: AppCommandActionCategory.allCases.enumerated().map { index, category in
                (category, index)
            }
        )

        return baseItems.sorted { lhs, rhs in
            let lhsRecentRank = recentRankByID[lhs.actionID]
            let rhsRecentRank = recentRankByID[rhs.actionID]
            switch (lhsRecentRank, rhsRecentRank) {
            case let (lhsRank?, rhsRank?):
                if lhsRank != rhsRank {
                    return lhsRank < rhsRank
                }
            case (.some, .none):
                return true
            case (.none, .some):
                return false
            case (.none, .none):
                break
            }

            let lhsCategoryRank = categoryRankByID[lhs.category] ?? Int.max
            let rhsCategoryRank = categoryRankByID[rhs.category] ?? Int.max
            if lhsCategoryRank != rhsCategoryRank {
                return lhsCategoryRank < rhsCategoryRank
            }
            return lhs.title.localizedCaseInsensitiveCompare(rhs.title) == .orderedAscending
        }
    }

    static func rankedActionItems(
        for query: String,
        from baseItems: [AppCommandActionItem]
    ) -> [AppCommandActionItem] {
        let normalizedQuery = query
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .lowercased()
        guard !normalizedQuery.isEmpty else {
            return baseItems
        }

        let queryTokens = normalizedSearchTokens(normalizedQuery)
        let baseOrderByActionID = Dictionary(
            uniqueKeysWithValues: baseItems.enumerated().map { index, item in
                (item.actionID, index)
            }
        )

        let scoredItems: [(item: AppCommandActionItem, score: Int)] = baseItems.compactMap { item in
            let score = appCommandIntentMatchScore(
                item: item,
                normalizedQuery: normalizedQuery,
                queryTokens: queryTokens
            )
            guard score > 0 else {
                return nil
            }
            return (item, score)
        }

        return scoredItems
            .sorted { lhs, rhs in
                if lhs.score != rhs.score {
                    return lhs.score > rhs.score
                }
                if lhs.item.isEnabled != rhs.item.isEnabled {
                    return lhs.item.isEnabled && !rhs.item.isEnabled
                }
                let lhsBaseOrder = baseOrderByActionID[lhs.item.actionID] ?? Int.max
                let rhsBaseOrder = baseOrderByActionID[rhs.item.actionID] ?? Int.max
                if lhsBaseOrder != rhsBaseOrder {
                    return lhsBaseOrder < rhsBaseOrder
                }
                return lhs.item.title.localizedCaseInsensitiveCompare(rhs.item.title) == .orderedAscending
            }
            .map(\.item)
    }

    static func firstEnabledAction(
        for query: String,
        from baseItems: [AppCommandActionItem]
    ) -> AppCommandActionItem? {
        rankedActionItems(for: query, from: baseItems).first(where: \.isEnabled)
    }

    static func rankedObjectItems(
        for query: String,
        from candidates: [CommandPaletteObjectSearchCandidate]
    ) -> [CommandPaletteObjectSearchItem] {
        let normalizedQuery = query
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .lowercased()
        guard !normalizedQuery.isEmpty else {
            return []
        }

        let queryTokens = normalizedSearchTokens(normalizedQuery)
        let scoredItems: [(item: CommandPaletteObjectSearchItem, score: Int)] = candidates.compactMap { candidate in
            let score = commandPaletteObjectMatchScore(
                item: candidate.item,
                searchableValues: candidate.searchableValues,
                normalizedQuery: normalizedQuery,
                queryTokens: queryTokens
            )
            guard score > 0 else {
                return nil
            }
            return (candidate.item, score)
        }

        return scoredItems
            .sorted { lhs, rhs in
                if lhs.score != rhs.score {
                    return lhs.score > rhs.score
                }
                if lhs.item.kind.rankingOrder != rhs.item.kind.rankingOrder {
                    return lhs.item.kind.rankingOrder < rhs.item.kind.rankingOrder
                }
                let titleComparison = lhs.item.title.localizedCaseInsensitiveCompare(rhs.item.title)
                if titleComparison != .orderedSame {
                    return titleComparison == .orderedAscending
                }
                return lhs.item.id.localizedCaseInsensitiveCompare(rhs.item.id) == .orderedAscending
            }
            .map(\.item)
    }

    static func firstObjectMatch(
        for query: String,
        from candidates: [CommandPaletteObjectSearchCandidate]
    ) -> CommandPaletteObjectSearchItem? {
        rankedObjectItems(for: query, from: candidates).first
    }

    private static func appCommandActionItem(
        _ actionID: AppCommandActionID,
        category: AppCommandActionCategory,
        title: String,
        subtitle: String?,
        symbolName: String,
        shortcutHint: String?,
        isEnabled: (AppCommandActionID) -> Bool,
        disabledReason: (AppCommandActionID) -> String?
    ) -> AppCommandActionItem {
        AppCommandActionItem(
            actionID: actionID,
            category: category,
            title: title,
            subtitle: subtitle,
            symbolName: symbolName,
            shortcutHint: shortcutHint,
            isEnabled: isEnabled(actionID),
            disabledReason: disabledReason(actionID)
        )
    }

    private static func appCommandIntentMatchScore(
        item: AppCommandActionItem,
        normalizedQuery: String,
        queryTokens: [String]
    ) -> Int {
        var score = 0
        let normalizedTitle = item.title.lowercased()
        let normalizedSubtitle = (item.subtitle ?? "").lowercased()
        let normalizedCategory = item.category.title.lowercased()
        let normalizedShortcut = (item.shortcutHint ?? "").lowercased()
        let normalizedDisabledReason = (item.disabledReason ?? "").lowercased()
        let intentKeywords = appCommandIntentKeywords(for: item.actionID)

        if normalizedTitle == normalizedQuery {
            score += 520
        } else if normalizedTitle.hasPrefix(normalizedQuery) {
            score += 420
        } else if normalizedTitle.contains(normalizedQuery) {
            score += 320
        }

        if normalizedCategory.contains(normalizedQuery) {
            score += 180
        }
        if normalizedSubtitle.contains(normalizedQuery) {
            score += 220
        }
        if normalizedShortcut.contains(normalizedQuery) {
            score += 80
        }
        if normalizedDisabledReason.contains(normalizedQuery) {
            score += 120
        }

        for keyword in intentKeywords {
            if keyword == normalizedQuery {
                score += 560
            } else if keyword.hasPrefix(normalizedQuery) {
                score += 420
            } else if keyword.contains(normalizedQuery) {
                score += 300
            }
        }

        if !queryTokens.isEmpty {
            let searchableTokens = Set(
                normalizedSearchTokens(
                    [
                        normalizedTitle,
                        normalizedSubtitle,
                        normalizedCategory,
                        normalizedShortcut,
                        normalizedDisabledReason,
                        intentKeywords.joined(separator: " ")
                    ]
                    .joined(separator: " ")
                )
            )

            let matchedTokenCount = queryTokens.filter { searchableTokens.contains($0) }.count
            score += matchedTokenCount * 70
            if matchedTokenCount == queryTokens.count {
                score += 120
            }
        }

        return score
    }

    private static func appCommandIntentKeywords(for actionID: AppCommandActionID) -> [String] {
        switch actionID {
        case .doSendMessage:
            return ["do", "send message", "text someone", "message contact", "send text", "send sms"]
        case .doSendEmail:
            return ["do", "send email", "email someone", "write email", "draft email", "compose email"]
        case .doCreateTask:
            return ["do", "create task", "new task", "add task", "track work", "make todo"]
        case .doReviewApprovals:
            return ["do", "review approvals", "approve request", "pending approvals", "decision queue"]
        case .doInspectIssue:
            return ["do", "inspect issue", "debug issue", "check logs", "investigate failure", "inspect activity"]
        case .openConfiguration:
            return ["settings", "configuration", "config", "preferences", "setup"]
        case .openChat:
            return ["chat", "assistant", "conversation", "message assistant", "ask"]
        case .openCommunications:
            return ["communications", "inbox", "messages", "calls", "delivery attempts"]
        case .openAutomation:
            return ["automation", "triggers", "schedule", "on comm event", "fire history"]
        case .openApprovals:
            return ["approvals", "decision", "risk review", "go ahead", "pending approvals"]
        case .openTasks:
            return ["tasks", "task queue", "runs", "workflow runs", "new task"]
        case .openModels:
            return ["models", "providers", "model catalog", "routing policy", "chat route"]
        case .openInspect:
            return ["inspect", "activity", "trace", "logs", "diagnostics"]
        case .openChannels:
            return ["channels", "channel mapping", "delivery policy", "message channel", "voice channel"]
        case .openConnectors:
            return ["connectors", "integrations", "permissions", "mail", "calendar", "browser", "finder", "twilio", "imessage"]
        case .refreshCurrentSection:
            return ["refresh", "reload", "recheck", "sync", "update panel"]
        case .openNotificationCenter:
            return ["notifications", "notification center", "alerts", "activity inbox", "status history"]
        case .setSimpleDensityMode:
            return ["simple mode", "simple density", "compact details", "less detail"]
        case .setAdvancedDensityMode:
            return ["advanced mode", "advanced density", "more detail", "full metadata"]
        case .performOnboardingFixNextStep:
            return ["fix next", "resolve blocker", "setup recovery", "repair setup"]
        case .startDaemon:
            return ["start daemon", "start service", "bring daemon online", "run daemon"]
        case .stopDaemon:
            return ["stop daemon", "stop service", "halt daemon", "shutdown daemon"]
        case .restartDaemon:
            return ["restart daemon", "restart service", "reboot daemon", "cycle daemon"]
        }
    }

    private static func commandPaletteObjectMatchScore(
        item: CommandPaletteObjectSearchItem,
        searchableValues: [String],
        normalizedQuery: String,
        queryTokens: [String]
    ) -> Int {
        var score = 0
        let normalizedTitle = item.title.lowercased()
        let normalizedSubtitle = (item.subtitle ?? "").lowercased()
        let normalizedKind = item.kind.title.lowercased()
        let normalizedSearchableValues = searchableValues.map { $0.lowercased() }

        if normalizedTitle == normalizedQuery {
            score += 680
        } else if normalizedTitle.hasPrefix(normalizedQuery) {
            score += 560
        } else if normalizedTitle.contains(normalizedQuery) {
            score += 420
        }

        if normalizedSubtitle.contains(normalizedQuery) {
            score += 220
        }
        if normalizedKind.contains(normalizedQuery) {
            score += 180
        }

        for value in normalizedSearchableValues {
            if value == normalizedQuery {
                score += 520
            } else if value.hasPrefix(normalizedQuery) {
                score += 340
            } else if value.contains(normalizedQuery) {
                score += 220
            }
        }

        if !queryTokens.isEmpty {
            let searchableTokens = Set(
                normalizedSearchTokens(
                    ([normalizedTitle, normalizedSubtitle, normalizedKind] + normalizedSearchableValues)
                        .joined(separator: " ")
                )
            )
            let matchedTokenCount = queryTokens.filter { searchableTokens.contains($0) }.count
            score += matchedTokenCount * 90
            if matchedTokenCount == queryTokens.count {
                score += 180
            }
        }

        return score
    }

    private static func normalizedSearchTokens(_ value: String) -> [String] {
        let separators = CharacterSet.alphanumerics.inverted
        return value
            .lowercased()
            .components(separatedBy: separators)
            .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
            .filter { !$0.isEmpty }
    }
}
