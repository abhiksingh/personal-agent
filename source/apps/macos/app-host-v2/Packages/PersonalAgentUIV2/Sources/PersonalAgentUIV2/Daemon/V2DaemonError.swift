import Foundation

public struct V2DaemonProblemService: Decodable, Sendable, Equatable {
    public let id: String?
    public let label: String?
    public let configField: String?

    enum CodingKeys: String, CodingKey {
        case id
        case label
        case configField = "config_field"
    }
}

public struct V2DaemonProblemRemediation: Decodable, Sendable, Equatable {
    public let action: String?
    public let label: String?
    public let hint: String?
}

public struct V2DaemonProblemDetails: Decodable, Sendable, Equatable {
    public let category: String?
    public let domain: String?
    public let service: V2DaemonProblemService?
    public let remediation: V2DaemonProblemRemediation?
}

public enum V2DaemonAPIError: LocalizedError, Sendable, Equatable {
    case missingAuthToken
    case invalidBaseURL
    case invalidResponse
    case transport(String)
    case server(statusCode: Int, message: String)
    case serverProblem(
        statusCode: Int,
        message: String,
        code: String,
        details: V2DaemonProblemDetails?,
        correlationID: String?
    )
    case decoding(String)

    private var serverTuple: (
        statusCode: Int,
        message: String,
        code: String?,
        details: V2DaemonProblemDetails?,
        correlationID: String?
    )? {
        switch self {
        case .server(let statusCode, let message):
            return (statusCode, message, nil, nil, nil)
        case .serverProblem(
            let statusCode,
            let message,
            let code,
            let details,
            let correlationID
        ):
            return (statusCode, message, code, details, correlationID)
        default:
            return nil
        }
    }

    public var errorDescription: String? {
        switch self {
        case .missingAuthToken:
            return "Assistant access token is not configured."
        case .invalidBaseURL:
            return "Daemon URL is invalid."
        case .invalidResponse:
            return "Daemon returned an invalid response."
        case .transport(let message):
            return message
        case .server(let statusCode, let message):
            return "Daemon request failed (\(statusCode)): \(message)"
        case .serverProblem(let statusCode, let message, _, _, _):
            return "Daemon request failed (\(statusCode)): \(message)"
        case .decoding(let message):
            return "Failed to decode daemon payload: \(message)"
        }
    }

    public var serverStatusCode: Int? { serverTuple?.statusCode }
    public var serverMessage: String? { serverTuple?.message }
    public var serverCode: String? { serverTuple?.code }
    public var serverDetails: V2DaemonProblemDetails? { serverTuple?.details }
    public var serverCorrelationID: String? { serverTuple?.correlationID }

    public var isUnauthorized: Bool {
        serverStatusCode == 401
    }

    public var isConnectivityIssue: Bool {
        if case .transport = self {
            return true
        }
        return false
    }

    public var isRateLimited: Bool {
        if serverStatusCode == 429 {
            return true
        }
        return serverCode == "rate_limit_exceeded"
    }

    public var isAuthScopeDenied: Bool {
        guard let code = serverCode else {
            return false
        }
        return code == "auth_scope_denied" || code == "insufficient_scope"
    }
}

struct V2DaemonErrorResponse: Decodable {
    private struct ErrorObject: Decodable {
        let code: String?
        let message: String?
        let details: V2DaemonProblemDetails?
    }

    let message: String
    let code: String?
    let details: V2DaemonProblemDetails?
    let correlationID: String?
    let title: String?
    let type: String?

    enum CodingKeys: String, CodingKey {
        case error
        case correlationID = "correlation_id"
        case title
        case detail
        case type
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)

        let errorObject = try? container.decode(ErrorObject.self, forKey: .error)
        let messageFromObject = Self.trimmed(errorObject?.message)
        let messageFromDetail = Self.trimmed(try container.decodeIfPresent(String.self, forKey: .detail))
        let messageFromTitle = Self.trimmed(try container.decodeIfPresent(String.self, forKey: .title))

        message = messageFromObject ?? messageFromDetail ?? messageFromTitle ?? "Request failed"

        let codeFromObject = Self.trimmed(errorObject?.code)
        let type = Self.trimmed(try container.decodeIfPresent(String.self, forKey: .type))
        let codeFromType = Self.problemCode(fromTypeURI: type)
        code = codeFromObject ?? codeFromType

        details = errorObject?.details
        correlationID = Self.trimmed(try container.decodeIfPresent(String.self, forKey: .correlationID))
        title = messageFromTitle
        self.type = type
    }

    private static func trimmed(_ value: String?) -> String? {
        guard let value else {
            return nil
        }
        let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? nil : trimmed
    }

    private static func problemCode(fromTypeURI typeURI: String?) -> String? {
        guard let trimmedType = trimmed(typeURI),
              let slashIndex = trimmedType.lastIndex(of: "/") else {
            return nil
        }
        let tail = String(trimmedType[trimmedType.index(after: slashIndex)...])
        return trimmed(tail)
    }
}
