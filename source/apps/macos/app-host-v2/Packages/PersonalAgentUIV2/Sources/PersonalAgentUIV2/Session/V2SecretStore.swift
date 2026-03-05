import Foundation
import Security

public struct V2SessionTokenReference: Codable, Sendable, Equatable {
    public var service: String
    public var account: String

    public init(service: String, account: String) {
        self.service = service
        self.account = account
    }

    public static let defaultReference = V2SessionTokenReference(
        service: "personalagent.ui.v2.access_token.v1",
        account: "daemon_auth_token"
    )
}

public enum V2SecretStoreError: LocalizedError {
    case invalidValue
    case keychainStatus(OSStatus)

    public var errorDescription: String? {
        switch self {
        case .invalidValue:
            return "Secret value must be valid UTF-8 text."
        case .keychainStatus(let status):
            if let message = SecCopyErrorMessageString(status, nil) as String? {
                return message
            }
            return "Keychain operation failed (\(status))."
        }
    }
}

public protocol V2SecretStoring {
    func upsertSecret(value: String, reference: V2SessionTokenReference) throws
    func readSecret(reference: V2SessionTokenReference) throws -> String?
    func deleteSecret(reference: V2SessionTokenReference) throws
}

public struct V2KeychainSecretStore: V2SecretStoring {
    public init() {}

    public func upsertSecret(value: String, reference: V2SessionTokenReference) throws {
        guard let data = value.data(using: .utf8) else {
            throw V2SecretStoreError.invalidValue
        }

        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: reference.service,
            kSecAttrAccount as String: reference.account
        ]
        let attributes: [String: Any] = [
            kSecValueData as String: data
        ]

        let addStatus = SecItemAdd(query.merging(attributes) { _, new in new } as CFDictionary, nil)
        if addStatus == errSecSuccess {
            return
        }
        if addStatus == errSecDuplicateItem {
            let updateStatus = SecItemUpdate(query as CFDictionary, attributes as CFDictionary)
            guard updateStatus == errSecSuccess else {
                throw V2SecretStoreError.keychainStatus(updateStatus)
            }
            return
        }
        throw V2SecretStoreError.keychainStatus(addStatus)
    }

    public func readSecret(reference: V2SessionTokenReference) throws -> String? {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: reference.service,
            kSecAttrAccount as String: reference.account,
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitOne
        ]

        var result: CFTypeRef?
        let status = SecItemCopyMatching(query as CFDictionary, &result)
        switch status {
        case errSecSuccess:
            break
        case errSecItemNotFound:
            return nil
        default:
            throw V2SecretStoreError.keychainStatus(status)
        }

        guard let data = result as? Data else {
            throw V2SecretStoreError.invalidValue
        }
        guard let value = String(data: data, encoding: .utf8) else {
            throw V2SecretStoreError.invalidValue
        }
        return value
    }

    public func deleteSecret(reference: V2SessionTokenReference) throws {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: reference.service,
            kSecAttrAccount as String: reference.account
        ]

        let status = SecItemDelete(query as CFDictionary)
        switch status {
        case errSecSuccess, errSecItemNotFound:
            return
        default:
            throw V2SecretStoreError.keychainStatus(status)
        }
    }
}

public final class V2InMemorySecretStore: V2SecretStoring {
    private var values: [String: String]
    private let lock = NSLock()

    public init(values: [String: String] = [:]) {
        self.values = values
    }

    public func upsertSecret(value: String, reference: V2SessionTokenReference) throws {
        let trimmed = value.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            throw V2SecretStoreError.invalidValue
        }
        lock.lock()
        values[key(for: reference)] = trimmed
        lock.unlock()
    }

    public func readSecret(reference: V2SessionTokenReference) throws -> String? {
        lock.lock()
        let value = values[key(for: reference)]
        lock.unlock()
        return value
    }

    public func deleteSecret(reference: V2SessionTokenReference) throws {
        lock.lock()
        values.removeValue(forKey: key(for: reference))
        lock.unlock()
    }

    private func key(for reference: V2SessionTokenReference) -> String {
        "\(reference.service)|\(reference.account)"
    }
}
