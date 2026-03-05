import XCTest
@testable import PersonalAgentUI

final class ChatMarkdownParserTests: XCTestCase {
    func testParsePlainMarkdownReturnsSingleTextSegment() {
        let segments = ChatMarkdownParser.parse("Hello **world**")

        XCTAssertEqual(segments.count, 1)
        guard case .text(let attributed) = segments[0] else {
            XCTFail("expected text segment")
            return
        }

        XCTAssertEqual(String(attributed.characters), "Hello world")
    }

    func testParseFencedCodeExtractsLanguageAndContent() {
        let text = """
        Here is JSON:

        ```json
        {"ok":true}
        ```

        Done.
        """

        let segments = ChatMarkdownParser.parse(text)

        XCTAssertEqual(segments.count, 3)

        guard case .text(let prefixText) = segments[0] else {
            XCTFail("expected leading text segment")
            return
        }
        XCTAssertEqual(String(prefixText.characters), "Here is JSON:")

        guard case .code(let language, let content) = segments[1] else {
            XCTFail("expected code segment")
            return
        }
        XCTAssertEqual(language, "json")
        XCTAssertEqual(content, "{\"ok\":true}")

        guard case .text(let suffixText) = segments[2] else {
            XCTFail("expected trailing text segment")
            return
        }
        XCTAssertEqual(String(suffixText.characters), "Done.")
    }

    func testParseMarkdownImageCreatesImageSegment() {
        let segments = ChatMarkdownParser.parse("Diagram: ![System Diagram](https://example.com/diagram.png)")

        XCTAssertEqual(segments.count, 2)
        guard segments.count == 2 else {
            XCTFail("expected text + image segments")
            return
        }

        guard case .text(let textSegment) = segments[0] else {
            XCTFail("expected text segment")
            return
        }
        XCTAssertEqual(String(textSegment.characters), "Diagram: ")

        guard case .image(let url, let altText) = segments[1] else {
            XCTFail("expected image segment")
            return
        }
        XCTAssertEqual(url.absoluteString, "https://example.com/diagram.png")
        XCTAssertEqual(altText, "System Diagram")
    }

    func testParseUnclosedFenceStillReturnsCodeSegment() {
        let text = """
        ```yaml
        key: value
        another: item
        """

        let segments = ChatMarkdownParser.parse(text)

        XCTAssertEqual(segments.count, 1)
        guard case .code(let language, let content) = segments[0] else {
            XCTFail("expected code segment")
            return
        }

        XCTAssertEqual(language, "yaml")
        XCTAssertEqual(content, "key: value\nanother: item")
    }

    func testParseMarkdownListPreservesLineBreaksAndMarkers() {
        let text = """
        I can help with a wide range of tasks, such as:
        - Answering questions
        - Explaining concepts
        - Drafting text
        """

        let segments = ChatMarkdownParser.parse(text)
        XCTAssertEqual(segments.count, 1)

        guard case .text(let attributed) = segments[0] else {
            XCTFail("expected text segment")
            return
        }

        let rendered = String(attributed.characters)
        XCTAssertEqual(
            rendered,
            """
            I can help with a wide range of tasks, such as:
            - Answering questions
            - Explaining concepts
            - Drafting text
            """
        )
    }

    func testParseMarkdownOrderedListPreservesNumericMarkers() {
        let text = """
        I can help with:
        1. First step
        2. Second step
        """

        let segments = ChatMarkdownParser.parse(text)
        XCTAssertEqual(segments.count, 1)

        guard case .text(let attributed) = segments[0] else {
            XCTFail("expected text segment")
            return
        }

        let rendered = String(attributed.characters)
        XCTAssertEqual(
            rendered,
            """
            I can help with:
            1. First step
            2. Second step
            """
        )
    }

    func testParseBulletCharacterLinesPreserveNewlines() {
        let text = """
        I can:
        • Answer questions.
        • Browse web pages.
        • Manage calendar.
        """

        let segments = ChatMarkdownParser.parse(text)
        XCTAssertEqual(segments.count, 1)

        guard case .text(let attributed) = segments[0] else {
            XCTFail("expected text segment")
            return
        }

        XCTAssertEqual(
            String(attributed.characters),
            """
            I can:
            • Answer questions.
            • Browse web pages.
            • Manage calendar.
            """
        )
    }

    func testParseMarkdownPreservesBlankLineSeparation() {
        let text = """
        First paragraph.

        Second paragraph.
        """

        let segments = ChatMarkdownParser.parse(text)
        XCTAssertEqual(segments.count, 1)

        guard case .text(let attributed) = segments[0] else {
            XCTFail("expected text segment")
            return
        }

        XCTAssertEqual(
            String(attributed.characters),
            """
            First paragraph.

            Second paragraph.
            """
        )
    }

    func testParseMarkdownTableCreatesTableSegment() {
        let text = """
        | What you want | How to ask | Example |
        | --- | --- | --- |
        | Get information | “Explain how photosynthesis works.” | “What causes the aurora borealis?” |
        | Write something | “Draft a polite email asking for a meeting.” | “Write a short story about a time-traveling cat.” |
        """

        let segments = ChatMarkdownParser.parse(text)
        XCTAssertEqual(segments.count, 1)

        guard case .table(let headers, let rows) = segments[0] else {
            XCTFail("expected table segment")
            return
        }

        XCTAssertEqual(
            headers.map { String($0.characters) },
            ["What you want", "How to ask", "Example"]
        )
        XCTAssertEqual(rows.count, 2)
        XCTAssertEqual(
            rows[0].map { String($0.characters) },
            [
                "Get information",
                "“Explain how photosynthesis works.”",
                "“What causes the aurora borealis?”"
            ]
        )
        XCTAssertEqual(
            rows[1].map { String($0.characters) },
            [
                "Write something",
                "“Draft a polite email asking for a meeting.”",
                "“Write a short story about a time-traveling cat.”"
            ]
        )
    }

    func testParseMarkdownTablePreservesLongCellTextWithoutTruncation() {
        let text = """
        | Goal | Prompt |
        | --- | --- |
        | Plan something | Create a study schedule for 10 days with daily checkpoints and review milestones. |
        """

        let segments = ChatMarkdownParser.parse(text)
        XCTAssertEqual(segments.count, 1)

        guard case .table(let headers, let rows) = segments[0] else {
            XCTFail("expected table segment")
            return
        }

        XCTAssertEqual(headers.map { String($0.characters) }, ["Goal", "Prompt"])
        XCTAssertEqual(rows.count, 1)
        XCTAssertEqual(
            String(rows[0][1].characters),
            "Create a study schedule for 10 days with daily checkpoints and review milestones."
        )
    }

    func testPrefersExpandedTimelineWidthForTables() {
        let text = """
        | Name | Role |
        | --- | --- |
        | Alex | Operator |
        """

        XCTAssertTrue(ChatMarkdownParser.prefersExpandedTimelineWidth(text))
    }

    func testPrefersExpandedTimelineWidthForPlainTextIsFalse() {
        XCTAssertFalse(
            ChatMarkdownParser.prefersExpandedTimelineWidth(
                "I can help with planning and writing."
            )
        )
    }
}
