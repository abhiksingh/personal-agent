import Foundation
import Security

enum LocalSecretStoreError: LocalizedError {
    case invalidValue
    case keychainStatus(OSStatus)

    var errorDescription: String? {
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

enum LocalSecretStore {
    static func upsertSecret(value: String, service: String, account: String) throws {
        guard let data = value.data(using: .utf8) else {
            throw LocalSecretStoreError.invalidValue
        }

        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account
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
                throw LocalSecretStoreError.keychainStatus(updateStatus)
            }
            return
        }
        throw LocalSecretStoreError.keychainStatus(addStatus)
    }

    static func readSecret(service: String, account: String) throws -> String? {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
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
            throw LocalSecretStoreError.keychainStatus(status)
        }

        guard let data = result as? Data else {
            throw LocalSecretStoreError.invalidValue
        }
        guard let value = String(data: data, encoding: .utf8) else {
            throw LocalSecretStoreError.invalidValue
        }
        return value
    }

    static func deleteSecret(service: String, account: String) throws {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account
        ]

        let status = SecItemDelete(query as CFDictionary)
        switch status {
        case errSecSuccess, errSecItemNotFound:
            return
        default:
            throw LocalSecretStoreError.keychainStatus(status)
        }
    }
}
