import Foundation

enum ChatRealtimeFallbackReason: String, Sendable, Equatable {
    case unavailable
    case disconnected
    case staleSession
    case capacityRejected
    case unauthorized
}

@MainActor
final class ChatOrchestrationStore {
    var turnTask: Task<Void, Never>?
    var realtimeTask: Task<Void, Never>?
    var realtimeSession: DaemonRealtimeSession?
    var realtimeConnectedForActiveTurn = false
    var realtimeFallbackReason: ChatRealtimeFallbackReason?
    var realtimeFallbackDetail: String?

    func resetRealtimeTrackingForNewTurn() {
        realtimeConnectedForActiveTurn = false
        realtimeFallbackReason = nil
        realtimeFallbackDetail = nil
    }

    func cancelActiveTasks() {
        turnTask?.cancel()
        turnTask = nil
        realtimeTask?.cancel()
        realtimeTask = nil
        resetRealtimeTrackingForNewTurn()
    }
}
