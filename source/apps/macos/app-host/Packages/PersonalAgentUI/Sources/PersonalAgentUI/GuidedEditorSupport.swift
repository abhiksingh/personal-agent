import Foundation

struct GuidedEditorScopeEntry: Identifiable, Equatable, Sendable {
    let id: UUID
    var key: String
    var value: String

    init(id: UUID = UUID(), key: String, value: String) {
        self.id = id
        self.key = key
        self.value = value
    }
}

struct GuidedEditorSupport {
    struct NormalizedTokenEntries: Equatable, Sendable {
        let values: [String]
        let duplicateCount: Int
    }

    static func normalizeTokenEntries(fromCommaSeparated raw: String) -> NormalizedTokenEntries {
        let trimmed = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            return NormalizedTokenEntries(values: [], duplicateCount: 0)
        }

        let parts = raw.split(separator: ",", omittingEmptySubsequences: false)
        var values: [String] = []
        var seen = Set<String>()
        var duplicateCount = 0

        for part in parts {
            let candidate = part.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
            if candidate.isEmpty {
                continue
            }
            if seen.contains(candidate) {
                duplicateCount += 1
                continue
            }
            seen.insert(candidate)
            values.append(candidate)
        }

        return NormalizedTokenEntries(values: values, duplicateCount: duplicateCount)
    }

    static func normalizeChannelEntries(fromCommaSeparated raw: String) -> NormalizedTokenEntries {
        let normalized = normalizeTokenEntries(fromCommaSeparated: raw)
        var values: [String] = []
        var seen = Set<String>()
        var duplicateCount = normalized.duplicateCount

        for value in normalized.values {
            let canonical = AppShellState.canonicalLogicalChannelID(from: value)
            guard !canonical.isEmpty else {
                continue
            }
            if seen.insert(canonical).inserted {
                values.append(canonical)
            } else {
                duplicateCount += 1
            }
        }

        return NormalizedTokenEntries(values: values, duplicateCount: duplicateCount)
    }

    static func normalizedScopeEntries(_ entries: [GuidedEditorScopeEntry]) -> [GuidedEditorScopeEntry] {
        var normalized: [GuidedEditorScopeEntry] = []
        var seenKeys = Set<String>()

        for entry in entries {
            let key = entry.key.trimmingCharacters(in: .whitespacesAndNewlines)
            let value = entry.value.trimmingCharacters(in: .whitespacesAndNewlines)
            guard !key.isEmpty, !value.isEmpty else {
                continue
            }
            let dedupeKey = key.lowercased()
            guard seenKeys.insert(dedupeKey).inserted else {
                continue
            }
            normalized.append(GuidedEditorScopeEntry(key: key, value: value))
        }
        return normalized
    }

    static func scopeJSON(from entries: [GuidedEditorScopeEntry]) -> String {
        let normalized = normalizedScopeEntries(entries)
        guard !normalized.isEmpty else {
            return "{}"
        }
        let dictionary = Dictionary(
            uniqueKeysWithValues: normalized.map { ($0.key, DaemonJSONValue.string($0.value)) }
        )
        return encodeJSONObject(dictionary) ?? "{}"
    }

    static func scopeEntries(from rawJSON: String) -> [GuidedEditorScopeEntry]? {
        guard let object = decodeJSONObject(rawJSON) else {
            return nil
        }
        return object
            .map { key, value in
                GuidedEditorScopeEntry(
                    key: key,
                    value: stringValue(fromJSONValue: value) ?? ""
                )
            }
            .filter { !$0.value.isEmpty }
            .sorted { $0.key.localizedCaseInsensitiveCompare($1.key) == .orderedAscending }
    }

    static func isValidRawJSONObject(_ rawJSON: String) -> Bool {
        decodeJSONObject(rawJSON) != nil
    }

    static func normalizedRawJSONObject(_ rawJSON: String) -> String {
        let trimmed = rawJSON.trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? "{}" : trimmed
    }

    private static func decodeJSONObject(_ rawJSON: String) -> [String: DaemonJSONValue]? {
        let normalized = normalizedRawJSONObject(rawJSON)
        guard let data = normalized.data(using: .utf8),
              let value = try? JSONDecoder().decode(DaemonJSONValue.self, from: data),
              case .object(let object) = value else {
            return nil
        }
        return object
    }

    private static func encodeJSONObject(_ object: [String: DaemonJSONValue]) -> String? {
        let encoder = JSONEncoder()
        encoder.outputFormatting = [.sortedKeys]
        guard let data = try? encoder.encode(DaemonJSONValue.object(object)),
              let encoded = String(data: data, encoding: .utf8) else {
            return nil
        }
        return encoded
    }

    private static func stringValue(fromJSONValue value: DaemonJSONValue) -> String? {
        switch value {
        case .string(let stringValue):
            return stringValue
        case .bool(let boolValue):
            return boolValue ? "true" : "false"
        case .number(let numberValue):
            if numberValue.rounded() == numberValue {
                return String(Int(numberValue))
            }
            return String(numberValue)
        default:
            return nil
        }
    }
}
