import XCTest
@testable import PersonalAgentUI

final class ChatTextNormalizationTests: XCTestCase {
    func testNormalizedNonEmptyTrimsAndDropsWhitespaceOnly() {
        XCTAssertNil(ChatTextNormalization.normalizedNonEmpty(nil))
        XCTAssertNil(ChatTextNormalization.normalizedNonEmpty("   \n\t  "))
        XCTAssertEqual(ChatTextNormalization.normalizedNonEmpty("  hello \n"), "hello")
    }

    func testNonEmptyPreservingWhitespaceKeepsOriginalContent() {
        XCTAssertNil(ChatTextNormalization.nonEmptyPreservingWhitespace(nil))
        XCTAssertNil(ChatTextNormalization.nonEmptyPreservingWhitespace(" \n\t "))
        XCTAssertEqual(
            ChatTextNormalization.nonEmptyPreservingWhitespace("  hello \n"),
            "  hello \n"
        )
    }

    func testNormalizedNewlinesCanonicalizesCarriageReturns() {
        XCTAssertEqual(
            ChatTextNormalization.normalizedNewlines("line1\r\nline2\rline3"),
            "line1\nline2\nline3"
        )
    }
}
