import Foundation

public enum UIPanelLatencyCategory: String, Codable, CaseIterable, Sendable, Equatable {
    case initialRender = "initial_render"
    case refresh
    case transition

    public var title: String {
        switch self {
        case .initialRender:
            return "Initial Render"
        case .refresh:
            return "Refresh"
        case .transition:
            return "Transition"
        }
    }
}

public struct UIPanelLatencySample: Identifiable, Codable, Sendable, Equatable {
    public let id: String
    public let sectionID: String
    public let category: UIPanelLatencyCategory
    public let durationMS: Int
    public let budgetMS: Int
    public let isOverBudget: Bool
    public let capturedAt: String

    public init(
        id: String = UUID().uuidString,
        sectionID: String,
        category: UIPanelLatencyCategory,
        durationMS: Int,
        budgetMS: Int,
        isOverBudget: Bool,
        capturedAt: String
    ) {
        self.id = id
        self.sectionID = sectionID
        self.category = category
        self.durationMS = durationMS
        self.budgetMS = budgetMS
        self.isOverBudget = isOverBudget
        self.capturedAt = capturedAt
    }

    public var section: AppSection? {
        AppSection(rawValue: sectionID)
    }

    public var sectionTitle: String {
        section?.title ?? sectionID
    }
}
