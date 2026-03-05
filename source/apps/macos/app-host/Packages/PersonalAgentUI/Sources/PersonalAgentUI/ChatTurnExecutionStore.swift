import Foundation

@MainActor
final class ChatTurnExecutionStore {
    struct RecoveredChatTurnSnapshot {
        let workspaceID: String?
        let correlationID: String
        let taskClass: String?
        let channelID: String?
        let items: [DaemonChatTurnItem]
        let taskRunCorrelation: DaemonChatTurnTaskRunCorrelation
    }

    struct Bindings {
        let daemonClient: DaemonAPIClient
        let daemonBaseURL: URL
        let workspaceID: String
        let parseDaemonTimestamp: (String) -> Date?
        let beginTurnExecution: (_ correlationID: String) -> Void
        let daemonMessagesForSubmission: () -> [(role: String, content: String)]
        let selectedActorIDForSubmission: () -> String?
        let handleRealtimeConnected: (_ session: DaemonRealtimeSession, _ correlationID: String) -> Void
        let handleRealtimeConnectionFailure: (_ error: Error) -> Void
        let handleTurnResponse: (
            _ response: DaemonChatTurnResponse,
            _ correlationID: String,
            _ authToken: String,
            _ selectedActorID: String?
        ) async -> Void
        let handleTurnCancellation: (_ correlationID: String) -> Void
        let handleRecoveredSnapshot: (
            _ snapshot: RecoveredChatTurnSnapshot,
            _ authToken: String,
            _ requestedByActorID: String?,
            _ subjectActorID: String?,
            _ actingAsActorID: String?
        ) async -> Void
        let handleTurnFailure: (_ error: Error, _ correlationID: String, _ submittedDraft: String) -> Void
        let finishTurnExecution: () async -> Void
    }

    func executeTurn(
        authToken: String,
        submittedDraft: String,
        bindings: Bindings
    ) async {
        let correlationID = UUID().uuidString.lowercased()
        bindings.beginTurnExecution(correlationID)

        let daemonMessages = bindings.daemonMessagesForSubmission()
        let selectedActorID = bindings.selectedActorIDForSubmission()

        do {
            let realtimeSession = try bindings.daemonClient.chat.connectRealtime(
                baseURL: bindings.daemonBaseURL,
                authToken: authToken,
                correlationID: correlationID
            )
            bindings.handleRealtimeConnected(realtimeSession, correlationID)
        } catch {
            bindings.handleRealtimeConnectionFailure(error)
        }

        do {
            let response = try await bindings.daemonClient.chat.chatTurn(
                baseURL: bindings.daemonBaseURL,
                authToken: authToken,
                workspaceID: bindings.workspaceID,
                messages: daemonMessages,
                requestedByActorID: selectedActorID,
                subjectActorID: selectedActorID,
                actingAsActorID: selectedActorID,
                correlationID: correlationID
            )
            await bindings.handleTurnResponse(response, correlationID, authToken, selectedActorID)
        } catch is CancellationError {
            bindings.handleTurnCancellation(correlationID)
        } catch {
            if let snapshot = await recoverTurnFromHistoryAfterTransportBlip(
                error: error,
                authToken: authToken,
                correlationID: correlationID,
                bindings: bindings
            ) {
                await bindings.handleRecoveredSnapshot(
                    snapshot,
                    authToken,
                    selectedActorID,
                    selectedActorID,
                    selectedActorID
                )
            } else {
                bindings.handleTurnFailure(error, correlationID, submittedDraft)
            }
        }

        await bindings.finishTurnExecution()
    }

    private func recoverTurnFromHistoryAfterTransportBlip(
        error: Error,
        authToken: String,
        correlationID: String,
        bindings: Bindings
    ) async -> RecoveredChatTurnSnapshot? {
        guard let daemonError = error as? DaemonAPIError,
              daemonError.isConnectivityIssue else {
            return nil
        }
        do {
            let history = try await bindings.daemonClient.chat.chatTurnHistory(
                baseURL: bindings.daemonBaseURL,
                authToken: authToken,
                workspaceID: bindings.workspaceID,
                correlationID: correlationID,
                limit: 160
            )
            return Self.recoveredChatTurnSnapshot(
                from: history,
                correlationID: correlationID,
                parseDaemonTimestamp: bindings.parseDaemonTimestamp
            )
        } catch {
            return nil
        }
    }

    nonisolated static func recoveredChatTurnSnapshot(
        from history: DaemonChatTurnHistoryResponse,
        correlationID: String,
        parseDaemonTimestamp: (String) -> Date?
    ) -> RecoveredChatTurnSnapshot? {
        let normalizedCorrelationID = correlationID
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .lowercased()
        guard !normalizedCorrelationID.isEmpty else {
            return nil
        }

        let matchingRecords = history.items.filter { record in
            ChatTextNormalization.normalizedNonEmpty(record.correlationID)?
                .trimmingCharacters(in: .whitespacesAndNewlines)
                .lowercased() == normalizedCorrelationID
        }
        guard !matchingRecords.isEmpty else {
            return nil
        }

        let recordsByTurnID = Dictionary(grouping: matchingRecords) { record in
            ChatTextNormalization.normalizedNonEmpty(record.turnID) ?? "unknown-turn"
        }
        let selectedTurnRecords = recordsByTurnID.values.max { lhs, rhs in
            let lhsNewest = lhs.compactMap { parseDaemonTimestamp($0.createdAt) }.max() ?? .distantPast
            let rhsNewest = rhs.compactMap { parseDaemonTimestamp($0.createdAt) }.max() ?? .distantPast
            if lhsNewest == rhsNewest {
                return lhs.count < rhs.count
            }
            return lhsNewest < rhsNewest
        } ?? matchingRecords

        let orderedRecords = selectedTurnRecords.sorted { lhs, rhs in
            if lhs.itemIndex == rhs.itemIndex {
                let lhsCreatedAt = parseDaemonTimestamp(lhs.createdAt) ?? .distantPast
                let rhsCreatedAt = parseDaemonTimestamp(rhs.createdAt) ?? .distantPast
                if lhsCreatedAt == rhsCreatedAt {
                    return lhs.recordID < rhs.recordID
                }
                return lhsCreatedAt < rhsCreatedAt
            }
            return lhs.itemIndex < rhs.itemIndex
        }
        let items = orderedRecords.map(\.item)
        guard !items.isEmpty else {
            return nil
        }
        return RecoveredChatTurnSnapshot(
            workspaceID: ChatTextNormalization.normalizedNonEmpty(history.workspaceID),
            correlationID: normalizedCorrelationID,
            taskClass: ChatTextNormalization.normalizedNonEmpty(orderedRecords.first?.taskClass),
            channelID: ChatTextNormalization.normalizedNonEmpty(orderedRecords.first?.channelID),
            items: items,
            taskRunCorrelation: orderedRecords.first?.taskRunReference ?? DaemonChatTurnTaskRunCorrelation()
        )
    }
}
