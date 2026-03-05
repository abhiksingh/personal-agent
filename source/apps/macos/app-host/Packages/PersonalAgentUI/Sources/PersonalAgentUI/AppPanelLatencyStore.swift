import Foundation

@MainActor
final class AppPanelLatencyStore: ObservableObject {
    enum Trigger {
        case bootstrap
        case transition
        case refresh
    }

    @Published private(set) var panelLatencySamples: [UIPanelLatencySample] = []
    @Published private(set) var panelLatencyLatestBySectionID: [String: UIPanelLatencySample] = [:]
    @Published private(set) var panelLatencyStatusMessage = "No panel latency samples captured yet."

    private let maxSampleCount: Int

    init(maxSampleCount: Int = 120) {
        self.maxSampleCount = max(1, maxSampleCount)
    }

    var panelLatencyLatestSamplesSorted: [UIPanelLatencySample] {
        AppSection.allCases.compactMap { panelLatencyLatestBySectionID[$0.rawValue] }
    }

    var panelLatencyRegressionSamples: [UIPanelLatencySample] {
        panelLatencyLatestSamplesSorted.filter(\.isOverBudget)
    }

    var panelLatencySampleCount: Int {
        panelLatencySamples.count
    }

    func clearPanelLatencySamples() {
        panelLatencyLatestBySectionID = [:]
        panelLatencySamples = []
        updatePanelLatencyStatusMessage()
    }

    func panelLatencyCategory(
        for section: AppSection,
        trigger: Trigger,
        hasLoadedPanelData: (AppSection) -> Bool
    ) -> UIPanelLatencyCategory {
        switch trigger {
        case .refresh:
            return .refresh
        case .transition:
            return hasLoadedPanelData(section) ? .transition : .initialRender
        case .bootstrap:
            return hasLoadedPanelData(section) ? .refresh : .initialRender
        }
    }

    func panelLatencyBudgetMS(
        for section: AppSection,
        category: UIPanelLatencyCategory
    ) -> Int {
        switch category {
        case .initialRender:
            switch section {
            case .configuration:
                return 2200
            case .communications, .automation, .channels, .connectors, .models:
                return 1600
            case .approvals, .tasks, .inspect:
                return 1400
            case .home:
                return 900
            case .chat:
                return 600
            }
        case .refresh:
            switch section {
            case .configuration:
                return 1800
            case .communications, .automation, .channels, .connectors, .models:
                return 1400
            case .approvals, .tasks, .inspect:
                return 1200
            case .home:
                return 700
            case .chat:
                return 500
            }
        case .transition:
            switch section {
            case .configuration:
                return 1400
            case .communications, .automation, .approvals, .tasks, .inspect, .channels, .connectors, .models:
                return 900
            case .home:
                return 600
            case .chat:
                return 400
            }
        }
    }

    func recordPanelLatencySample(
        section: AppSection,
        category: UIPanelLatencyCategory,
        durationMS: Int,
        capturedAt: Date = .now
    ) {
        let normalizedDurationMS = max(0, durationMS)
        let budgetMS = panelLatencyBudgetMS(for: section, category: category)
        let sample = UIPanelLatencySample(
            sectionID: section.rawValue,
            category: category,
            durationMS: normalizedDurationMS,
            budgetMS: budgetMS,
            isOverBudget: normalizedDurationMS > budgetMS,
            capturedAt: timestampString(capturedAt)
        )
        panelLatencyLatestBySectionID[section.rawValue] = sample
        panelLatencySamples.append(sample)
        if panelLatencySamples.count > maxSampleCount {
            let overflow = panelLatencySamples.count - maxSampleCount
            panelLatencySamples.removeFirst(overflow)
        }
        updatePanelLatencyStatusMessage()
    }

    private func updatePanelLatencyStatusMessage() {
        let regressionCount = panelLatencyRegressionSamples.count
        if regressionCount == 0 {
            if panelLatencyLatestBySectionID.isEmpty {
                panelLatencyStatusMessage = "No panel latency samples captured yet."
            } else {
                panelLatencyStatusMessage = "Panel latency within budget across \(panelLatencyLatestBySectionID.count) section(s)."
            }
        } else if regressionCount == 1 {
            panelLatencyStatusMessage = "1 section exceeded panel latency budget."
        } else {
            panelLatencyStatusMessage = "\(regressionCount) sections exceeded panel latency budget."
        }
    }

    private func timestampString(_ date: Date) -> String {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return formatter.string(from: date)
    }
}
