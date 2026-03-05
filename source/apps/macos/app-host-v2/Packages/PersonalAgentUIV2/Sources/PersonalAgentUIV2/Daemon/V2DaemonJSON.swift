import Foundation

public enum V2DaemonJSONValue: Codable, Sendable, Equatable {
    case string(String)
    case number(Double)
    case bool(Bool)
    case object([String: V2DaemonJSONValue])
    case array([V2DaemonJSONValue])
    case null

    public init(from decoder: Decoder) throws {
        let container = try decoder.singleValueContainer()
        if container.decodeNil() {
            self = .null
            return
        }
        if let boolValue = try? container.decode(Bool.self) {
            self = .bool(boolValue)
            return
        }
        if let intValue = try? container.decode(Int.self) {
            self = .number(Double(intValue))
            return
        }
        if let doubleValue = try? container.decode(Double.self) {
            self = .number(doubleValue)
            return
        }
        if let stringValue = try? container.decode(String.self) {
            self = .string(stringValue)
            return
        }
        if let objectValue = try? container.decode([String: V2DaemonJSONValue].self) {
            self = .object(objectValue)
            return
        }
        if let arrayValue = try? container.decode([V2DaemonJSONValue].self) {
            self = .array(arrayValue)
            return
        }
        throw DecodingError.typeMismatch(
            V2DaemonJSONValue.self,
            DecodingError.Context(codingPath: decoder.codingPath, debugDescription: "Unsupported daemon JSON payload value.")
        )
    }

    public func encode(to encoder: Encoder) throws {
        var container = encoder.singleValueContainer()
        switch self {
        case .string(let value):
            try container.encode(value)
        case .number(let value):
            try container.encode(value)
        case .bool(let value):
            try container.encode(value)
        case .object(let value):
            try container.encode(value)
        case .array(let value):
            try container.encode(value)
        case .null:
            try container.encodeNil()
        }
    }

    public var stringValue: String? {
        if case .string(let value) = self {
            return value
        }
        return nil
    }

    public var boolValue: Bool? {
        if case .bool(let value) = self {
            return value
        }
        return nil
    }

    public var objectValue: [String: V2DaemonJSONValue]? {
        if case .object(let value) = self {
            return value
        }
        return nil
    }

    public var displayText: String {
        switch self {
        case .string(let value):
            return value
        case .number(let value):
            if value.rounded() == value {
                return String(Int(value))
            }
            return String(value)
        case .bool(let value):
            return value ? "true" : "false"
        case .object(let value):
            return value
                .keys
                .sorted()
                .map { key in "\(key)=\(value[key]?.displayText ?? "")" }
                .joined(separator: ", ")
        case .array(let values):
            return values.map(\.displayText).joined(separator: ", ")
        case .null:
            return "null"
        }
    }
}

extension KeyedDecodingContainer {
    internal func decodeLossyString(forKey key: K) throws -> String? {
        if let value = try decodeIfPresent(String.self, forKey: key)?.trimmingCharacters(in: .whitespacesAndNewlines), !value.isEmpty {
            return value
        }
        if let intValue = try? decodeIfPresent(Int.self, forKey: key) {
            return String(intValue)
        }
        if let doubleValue = try? decodeIfPresent(Double.self, forKey: key) {
            return String(doubleValue)
        }
        if let boolValue = try? decodeIfPresent(Bool.self, forKey: key) {
            return boolValue ? "true" : "false"
        }
        return nil
    }

    internal func decodeLossyBool(forKey key: K) throws -> Bool? {
        if let value = try? decodeIfPresent(Bool.self, forKey: key) {
            return value
        }
        if let value = try? decodeIfPresent(String.self, forKey: key) {
            switch value.lowercased().trimmingCharacters(in: .whitespacesAndNewlines) {
            case "true", "1", "yes", "enabled":
                return true
            case "false", "0", "no", "disabled":
                return false
            default:
                return nil
            }
        }
        if let intValue = try? decodeIfPresent(Int.self, forKey: key) {
            return intValue != 0
        }
        return nil
    }

    internal func decodeLossyInt(forKey key: K) throws -> Int? {
        if let value = try? decodeIfPresent(Int.self, forKey: key) {
            return value
        }
        if let value = try? decodeIfPresent(Double.self, forKey: key) {
            return Int(value)
        }
        if let value = try? decodeIfPresent(String.self, forKey: key) {
            return Int(value.trimmingCharacters(in: .whitespacesAndNewlines))
        }
        return nil
    }

    internal func decodeLossyStringArray(forKey key: K) throws -> [String]? {
        if let values = try? decodeIfPresent([String].self, forKey: key) {
            let normalized = values
                .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
                .filter { !$0.isEmpty }
            return normalized
        }
        if let values = try? decodeIfPresent([Int].self, forKey: key) {
            return values.map(String.init)
        }
        if let raw = try? decodeIfPresent(String.self, forKey: key) {
            let normalized = raw
                .split(separator: ",")
                .map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }
                .filter { !$0.isEmpty }
            return normalized
        }
        return nil
    }
}

func v2DecodeDaemonJSONObject(from decoder: Decoder) -> [String: V2DaemonJSONValue] {
    guard let value = try? V2DaemonJSONValue(from: decoder),
          case .object(let object) = value else {
        return [:]
    }
    return object
}
