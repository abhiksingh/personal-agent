import SwiftUI
import XCTest
@testable import PersonalAgentUI

final class UIAccessibilityStylePolicyTests: XCTestCase {
    func testCardStrokeOpacityIncreasedContrastIsHigher() {
        let standardOpacity = UIAccessibilityStylePolicy.cardStrokeOpacity(
            emphasis: .standard,
            contrast: .standard
        )
        let increasedOpacity = UIAccessibilityStylePolicy.cardStrokeOpacity(
            emphasis: .standard,
            contrast: .increased
        )

        XCTAssertGreaterThan(increasedOpacity, standardOpacity)
    }

    func testCardShadowOpacityIncreasedContrastIsHigher() {
        let standardOpacity = UIAccessibilityStylePolicy.cardShadowOpacity(
            emphasis: .elevated,
            contrast: .standard
        )
        let increasedOpacity = UIAccessibilityStylePolicy.cardShadowOpacity(
            emphasis: .elevated,
            contrast: .increased
        )

        XCTAssertGreaterThan(increasedOpacity, standardOpacity)
    }

    func testCardStrokeLineWidthIncreasedContrastIsThicker() {
        let standardLineWidth = UIAccessibilityStylePolicy.cardStrokeLineWidth(contrast: .standard)
        let increasedLineWidth = UIAccessibilityStylePolicy.cardStrokeLineWidth(contrast: .increased)

        XCTAssertGreaterThan(increasedLineWidth, standardLineWidth)
    }

    func testStatusBadgeIncreasedContrastUsesHigherChrome() {
        let standardBackgroundOpacity = UIAccessibilityStylePolicy.statusBadgeBackgroundOpacity(contrast: .standard)
        let increasedBackgroundOpacity = UIAccessibilityStylePolicy.statusBadgeBackgroundOpacity(contrast: .increased)
        let standardBorderOpacity = UIAccessibilityStylePolicy.statusBadgeBorderOpacity(contrast: .standard)
        let increasedBorderOpacity = UIAccessibilityStylePolicy.statusBadgeBorderOpacity(contrast: .increased)

        XCTAssertGreaterThan(increasedBackgroundOpacity, standardBackgroundOpacity)
        XCTAssertGreaterThan(increasedBorderOpacity, standardBorderOpacity)
    }

    func testOverlayChromeIncreasedContrastUsesHigherOpacity() {
        let standardBorderOpacity = UIAccessibilityStylePolicy.overlayBorderOpacity(contrast: .standard)
        let increasedBorderOpacity = UIAccessibilityStylePolicy.overlayBorderOpacity(contrast: .increased)
        let standardShadowOpacity = UIAccessibilityStylePolicy.overlayShadowOpacity(contrast: .standard)
        let increasedShadowOpacity = UIAccessibilityStylePolicy.overlayShadowOpacity(contrast: .increased)

        XCTAssertGreaterThan(increasedBorderOpacity, standardBorderOpacity)
        XCTAssertGreaterThan(increasedShadowOpacity, standardShadowOpacity)
    }
}
