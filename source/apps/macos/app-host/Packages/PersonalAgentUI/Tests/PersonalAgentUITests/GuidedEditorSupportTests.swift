import XCTest
@testable import PersonalAgentUI

final class GuidedEditorSupportTests: XCTestCase {
    func testNormalizeTokenEntriesDeduplicatesAndLowercases() {
        let normalized = GuidedEditorSupport.normalizeTokenEntries(
            fromCommaSeparated: "  Alpha, beta,ALPHA,, beta , gamma "
        )

        XCTAssertEqual(normalized.values, ["alpha", "beta", "gamma"])
        XCTAssertEqual(normalized.duplicateCount, 2)
    }

    func testNormalizeChannelEntriesCanonicalizesAliases() {
        let normalized = GuidedEditorSupport.normalizeChannelEntries(
            fromCommaSeparated: "imessage_sms_bridge, twilio_sms, app_chat, VOICE, message"
        )

        XCTAssertEqual(normalized.values, ["message", "app", "voice"])
        XCTAssertEqual(normalized.duplicateCount, 2)
    }

    func testScopeJSONNormalizesEntriesAndDeduplicatesKeys() {
        let entries = [
            GuidedEditorScopeEntry(key: " channel ", value: "message"),
            GuidedEditorScopeEntry(key: "CHANNEL", value: "app"),
            GuidedEditorScopeEntry(key: "tenant", value: "ws1"),
            GuidedEditorScopeEntry(key: "", value: "ignored"),
            GuidedEditorScopeEntry(key: "empty", value: "")
        ]

        let json = GuidedEditorSupport.scopeJSON(from: entries)

        XCTAssertEqual(json, #"{"channel":"message","tenant":"ws1"}"#)
    }

    func testScopeEntriesParsesScalarObjectValues() {
        let entries = GuidedEditorSupport.scopeEntries(
            from: #"{"tenant":"ws1","max_attempts":3,"enabled":true}"#
        )

        XCTAssertEqual(entries?.count, 3)
        XCTAssertEqual(entries?.map(\.key), ["enabled", "max_attempts", "tenant"])
        XCTAssertEqual(entries?.map(\.value), ["true", "3", "ws1"])
    }

    func testIsValidRawJSONObjectRejectsArraysAndInvalidJSON() {
        XCTAssertTrue(GuidedEditorSupport.isValidRawJSONObject(#"{"k":"v"}"#))
        XCTAssertFalse(GuidedEditorSupport.isValidRawJSONObject(#"[1,2,3]"#))
        XCTAssertFalse(GuidedEditorSupport.isValidRawJSONObject("{invalid"))
    }
}
