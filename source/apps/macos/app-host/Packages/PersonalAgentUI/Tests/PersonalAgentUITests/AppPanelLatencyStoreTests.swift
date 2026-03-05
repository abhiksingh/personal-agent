import XCTest
@testable import PersonalAgentUI

@MainActor
final class AppPanelLatencyStoreTests: XCTestCase {
    func testPanelLatencyCategoryUsesPanelLoadStateClosure() {
        let store = AppPanelLatencyStore()

        XCTAssertEqual(
            store.panelLatencyCategory(
                for: .communications,
                trigger: .transition,
                hasLoadedPanelData: { _ in false }
            ),
            .initialRender
        )
        XCTAssertEqual(
            store.panelLatencyCategory(
                for: .communications,
                trigger: .bootstrap,
                hasLoadedPanelData: { _ in false }
            ),
            .initialRender
        )
        XCTAssertEqual(
            store.panelLatencyCategory(
                for: .communications,
                trigger: .refresh,
                hasLoadedPanelData: { _ in false }
            ),
            .refresh
        )
        XCTAssertEqual(
            store.panelLatencyCategory(
                for: .communications,
                trigger: .transition,
                hasLoadedPanelData: { _ in true }
            ),
            .transition
        )
        XCTAssertEqual(
            store.panelLatencyCategory(
                for: .communications,
                trigger: .bootstrap,
                hasLoadedPanelData: { _ in true }
            ),
            .refresh
        )
    }

    func testPanelLatencyBudgetMappingRemainsDeterministic() {
        let store = AppPanelLatencyStore()

        XCTAssertEqual(store.panelLatencyBudgetMS(for: .chat, category: .refresh), 500)
        XCTAssertEqual(store.panelLatencyBudgetMS(for: .configuration, category: .initialRender), 2200)
        XCTAssertEqual(store.panelLatencyBudgetMS(for: .inspect, category: .transition), 900)
    }

    func testPanelLatencyRegressionStatusTracksLatestSamplePerSection() {
        let store = AppPanelLatencyStore()
        let chatBudget = store.panelLatencyBudgetMS(for: .chat, category: .refresh)
        let channelsBudget = store.panelLatencyBudgetMS(for: .channels, category: .refresh)

        store.recordPanelLatencySample(
            section: .chat,
            category: .refresh,
            durationMS: chatBudget - 25
        )

        XCTAssertEqual(store.panelLatencyRegressionSamples.count, 0)
        XCTAssertEqual(store.panelLatencyStatusMessage, "Panel latency within budget across 1 section(s).")

        store.recordPanelLatencySample(
            section: .channels,
            category: .refresh,
            durationMS: channelsBudget + 40
        )

        XCTAssertEqual(store.panelLatencyRegressionSamples.count, 1)
        XCTAssertEqual(store.panelLatencyStatusMessage, "1 section exceeded panel latency budget.")

        store.recordPanelLatencySample(
            section: .channels,
            category: .refresh,
            durationMS: channelsBudget - 10
        )

        XCTAssertEqual(store.panelLatencyRegressionSamples.count, 0)
        XCTAssertEqual(store.panelLatencyStatusMessage, "Panel latency within budget across 2 section(s).")
    }

    func testPanelLatencySampleHistoryIsBounded() {
        let store = AppPanelLatencyStore(maxSampleCount: 120)

        for index in 0..<130 {
            store.recordPanelLatencySample(
                section: .tasks,
                category: .refresh,
                durationMS: index,
                capturedAt: Date(timeIntervalSince1970: TimeInterval(index))
            )
        }

        XCTAssertEqual(store.panelLatencySampleCount, 120)
        XCTAssertEqual(store.panelLatencySamples.first?.durationMS, 10)
        XCTAssertEqual(store.panelLatencySamples.last?.durationMS, 129)
    }

    func testClearPanelLatencySamplesResetsStatus() {
        let store = AppPanelLatencyStore()
        store.recordPanelLatencySample(
            section: .automation,
            category: .refresh,
            durationMS: 500
        )

        XCTAssertFalse(store.panelLatencySamples.isEmpty)
        XCTAssertFalse(store.panelLatencyLatestBySectionID.isEmpty)

        store.clearPanelLatencySamples()

        XCTAssertTrue(store.panelLatencySamples.isEmpty)
        XCTAssertTrue(store.panelLatencyLatestBySectionID.isEmpty)
        XCTAssertEqual(store.panelLatencyStatusMessage, "No panel latency samples captured yet.")
    }
}
