import XCTest
@testable import PersonalAgentUI

@MainActor
final class AppShellStatePanelLatencyTests: XCTestCase {
    func testPanelLatencyCategoryUsesPanelLoadState() {
        let state = AppShellState()
        state.hasLoadedCommunicationsInbox = false

        XCTAssertEqual(
            state._test_panelLatencyCategoryForTransition(section: .communications),
            .initialRender
        )
        XCTAssertEqual(
            state._test_panelLatencyCategoryForBootstrap(section: .communications),
            .initialRender
        )
        XCTAssertEqual(
            state._test_panelLatencyCategoryForRefresh(section: .communications),
            .refresh
        )

        state.hasLoadedCommunicationsInbox = true
        XCTAssertEqual(
            state._test_panelLatencyCategoryForTransition(section: .communications),
            .transition
        )
        XCTAssertEqual(
            state._test_panelLatencyCategoryForBootstrap(section: .communications),
            .refresh
        )
    }

    func testPanelLatencyRegressionStatusTracksLatestSamplePerSection() {
        let state = AppShellState()
        let chatBudget = state._test_panelLatencyBudget(section: .chat, category: .refresh)
        let channelsBudget = state._test_panelLatencyBudget(section: .channels, category: .refresh)

        state._test_recordPanelLatencySample(
            section: .chat,
            category: .refresh,
            durationMS: chatBudget - 25
        )

        XCTAssertEqual(state.panelLatencyRegressionSamples.count, 0)
        XCTAssertEqual(state.panelLatencyStatusMessage, "Panel latency within budget across 1 section(s).")

        state._test_recordPanelLatencySample(
            section: .channels,
            category: .refresh,
            durationMS: channelsBudget + 40
        )

        XCTAssertEqual(state.panelLatencyRegressionSamples.count, 1)
        XCTAssertEqual(state.panelLatencyStatusMessage, "1 section exceeded panel latency budget.")

        state._test_recordPanelLatencySample(
            section: .channels,
            category: .refresh,
            durationMS: channelsBudget - 10
        )

        XCTAssertEqual(state.panelLatencyRegressionSamples.count, 0)
        XCTAssertEqual(state.panelLatencyStatusMessage, "Panel latency within budget across 2 section(s).")
    }

    func testPanelLatencySampleHistoryIsBounded() {
        let state = AppShellState()

        for index in 0..<130 {
            state._test_recordPanelLatencySample(
                section: .tasks,
                category: .refresh,
                durationMS: index,
                capturedAt: Date(timeIntervalSince1970: TimeInterval(index))
            )
        }

        XCTAssertEqual(state.panelLatencySampleCount, 120)
        XCTAssertEqual(state.panelLatencySamples.first?.durationMS, 10)
        XCTAssertEqual(state.panelLatencySamples.last?.durationMS, 129)
    }

    func testClearPanelLatencySamplesResetsStatus() {
        let state = AppShellState()
        state._test_recordPanelLatencySample(
            section: .automation,
            category: .refresh,
            durationMS: 500
        )

        XCTAssertFalse(state.panelLatencySamples.isEmpty)
        XCTAssertFalse(state.panelLatencyLatestBySectionID.isEmpty)

        state.clearPanelLatencySamples()

        XCTAssertTrue(state.panelLatencySamples.isEmpty)
        XCTAssertTrue(state.panelLatencyLatestBySectionID.isEmpty)
        XCTAssertEqual(state.panelLatencyStatusMessage, "No panel latency samples captured yet.")
    }
}
