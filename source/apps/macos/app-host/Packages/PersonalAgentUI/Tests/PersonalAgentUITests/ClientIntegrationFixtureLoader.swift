import Foundation

enum ClientIntegrationFixtureLoader {
    private static let fixtureRelativePath = "packages/contracts/control/fixtures/client-integration"

    static func data(named fileName: String) throws -> Data {
        let fixtureURL = try fixtureURL(named: fileName)
        return try Data(contentsOf: fixtureURL)
    }

    private static func fixtureURL(named fileName: String) throws -> URL {
        var cursor = URL(fileURLWithPath: #filePath).deletingLastPathComponent()
        let fileManager = FileManager.default

        while cursor.path != "/" {
            let fixtureDirectory = cursor.appendingPathComponent(fixtureRelativePath, isDirectory: true)
            let candidate = fixtureDirectory.appendingPathComponent(fileName, isDirectory: false)
            if fileManager.fileExists(atPath: candidate.path) {
                return candidate
            }
            cursor.deleteLastPathComponent()
        }

        throw NSError(
            domain: "ClientIntegrationFixtureLoader",
            code: 1,
            userInfo: [
                NSLocalizedDescriptionKey: "Unable to locate fixture \(fileName) under \(fixtureRelativePath)."
            ]
        )
    }
}
