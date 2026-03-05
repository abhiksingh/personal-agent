import Foundation
import Darwin

struct DaemonLocalServiceInstallCommandResult: Sendable {
    let exitCode: Int32
    let stdout: String
    let stderr: String
}

struct DaemonLocalServiceInstallResult: Sendable {
    let daemonAppPath: String
    let daemonExecutablePath: String
    let launchAgentPath: String
    let authTokenFilePath: String
    let helperUpdated: Bool

    var summaryText: String {
        if helperUpdated {
            return "Daemon helper installed and launch agent updated."
        }
        return "Daemon launch agent refreshed."
    }
}

enum DaemonLocalServiceInstallError: LocalizedError, Equatable {
    case unsupportedAction(String)
    case missingAuthToken
    case appNotInApplications(String)
    case missingEmbeddedDaemon(String)
    case missingEmbeddedDaemonExecutable(String)
    case filesystem(String)
    case launchctlBootstrap(String)
    case launchctlEnable(String)
    case launchctlKickstart(String)

    var errorDescription: String? {
        switch self {
        case .unsupportedAction(let action):
            return "Unsupported daemon setup action: \(action)."
        case .missingAuthToken:
            return "Set Assistant Access Token before installing daemon setup."
        case .appNotInApplications:
            return "Move PersonalAgent.app to /Applications before running daemon install or repair."
        case .missingEmbeddedDaemon:
            return "Packaged daemon helper is missing from this app build. Rebuild release assets and retry."
        case .missingEmbeddedDaemonExecutable:
            return "Packaged daemon helper executable is missing. Rebuild release assets and retry."
        case .filesystem(let detail):
            return "Daemon setup failed while preparing local files: \(detail)"
        case .launchctlBootstrap(let detail):
            return "Daemon setup failed while loading launchctl service: \(detail)"
        case .launchctlEnable(let detail):
            return "Daemon setup failed while enabling launchctl service: \(detail)"
        case .launchctlKickstart(let detail):
            return "Daemon setup failed while starting launchctl service: \(detail)"
        }
    }
}

struct DaemonLocalServiceInstallerRuntime {
    typealias CommandRunner = (_ executablePath: String, _ arguments: [String]) throws -> DaemonLocalServiceInstallCommandResult

    let appBundleURL: URL
    let homeDirectoryURL: URL
    let pathEnvironment: String
    let userID: UInt32
    let commandRunner: CommandRunner

    static func live() -> DaemonLocalServiceInstallerRuntime {
        DaemonLocalServiceInstallerRuntime(
            appBundleURL: Bundle.main.bundleURL,
            homeDirectoryURL: FileManager.default.homeDirectoryForCurrentUser,
            pathEnvironment: ProcessInfo.processInfo.environment["PATH"] ?? "",
            userID: getuid(),
            commandRunner: DaemonLocalServiceInstallerRuntime.defaultCommandRunner
        )
    }

    private static func defaultCommandRunner(
        executablePath: String,
        arguments: [String]
    ) throws -> DaemonLocalServiceInstallCommandResult {
        let process = Process()
        process.executableURL = URL(fileURLWithPath: executablePath)
        process.arguments = arguments

        let stdoutPipe = Pipe()
        let stderrPipe = Pipe()
        process.standardOutput = stdoutPipe
        process.standardError = stderrPipe

        try process.run()
        process.waitUntilExit()

        let stdoutData = stdoutPipe.fileHandleForReading.readDataToEndOfFile()
        let stderrData = stderrPipe.fileHandleForReading.readDataToEndOfFile()
        let stdout = String(data: stdoutData, encoding: .utf8) ?? ""
        let stderr = String(data: stderrData, encoding: .utf8) ?? ""
        return DaemonLocalServiceInstallCommandResult(
            exitCode: process.terminationStatus,
            stdout: stdout,
            stderr: stderr
        )
    }
}

enum DaemonLocalServiceInstaller {
    private static let launchAgentLabel = "com.personalagent.daemon"
    private static let daemonAppName = "Personal Agent Daemon.app"
    private static let daemonExecutableName = "personal-agent-daemon"
    private static let daemonListenMode = "tcp"
    private static let daemonListenAddress = "127.0.0.1:7071"

    static func installOrRepair(
        action: String,
        authToken: String,
        runtime: DaemonLocalServiceInstallerRuntime = .live(),
        requireSystemApplicationsInstall: Bool = true
    ) throws -> DaemonLocalServiceInstallResult {
        let normalizedAction = action.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        guard normalizedAction == "install" || normalizedAction == "repair" else {
            throw DaemonLocalServiceInstallError.unsupportedAction(action)
        }

        let trimmedToken = authToken.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmedToken.isEmpty else {
            throw DaemonLocalServiceInstallError.missingAuthToken
        }

        let appBundleURL = runtime.appBundleURL.standardizedFileURL
        if requireSystemApplicationsInstall {
            let appPath = appBundleURL.path
            guard appPath == "/Applications/PersonalAgent.app" || appPath.hasPrefix("/Applications/") else {
                throw DaemonLocalServiceInstallError.appNotInApplications(appPath)
            }
        }

        let embeddedDaemonAppURL = appBundleURL
            .appendingPathComponent("Contents")
            .appendingPathComponent("Resources")
            .appendingPathComponent("Daemon")
            .appendingPathComponent(daemonAppName)
        guard isDirectory(embeddedDaemonAppURL) else {
            throw DaemonLocalServiceInstallError.missingEmbeddedDaemon(embeddedDaemonAppURL.path)
        }

        let embeddedDaemonExecutableURL = embeddedDaemonAppURL
            .appendingPathComponent("Contents")
            .appendingPathComponent("MacOS")
            .appendingPathComponent(daemonExecutableName)
        guard isExecutableFile(embeddedDaemonExecutableURL) else {
            throw DaemonLocalServiceInstallError.missingEmbeddedDaemonExecutable(embeddedDaemonExecutableURL.path)
        }

        let appSupportRootURL = runtime.homeDirectoryURL
            .appendingPathComponent("Library")
            .appendingPathComponent("Application Support")
            .appendingPathComponent("personal-agent")
        let helperInstallRootURL = appSupportRootURL.appendingPathComponent("daemon")
        let helperInstallAppURL = helperInstallRootURL.appendingPathComponent(daemonAppName)

        let fileManager = FileManager.default
        do {
            try createDirectory(helperInstallRootURL, mode: 0o700)
            let helperPreviouslyExisted = fileManager.fileExists(atPath: helperInstallAppURL.path)
            if helperPreviouslyExisted {
                try fileManager.removeItem(at: helperInstallAppURL)
            }
            try fileManager.copyItem(at: embeddedDaemonAppURL, to: helperInstallAppURL)

            let helperExecutableURL = helperInstallAppURL
                .appendingPathComponent("Contents")
                .appendingPathComponent("MacOS")
                .appendingPathComponent(daemonExecutableName)
            guard isExecutableFile(helperExecutableURL) else {
                throw DaemonLocalServiceInstallError.missingEmbeddedDaemonExecutable(helperExecutableURL.path)
            }

            let controlDirURL = appSupportRootURL.appendingPathComponent("control")
            try createDirectory(controlDirURL, mode: 0o700)
            let authTokenFileURL = controlDirURL.appendingPathComponent("local-dev.control.token")
            try writeToken(trimmedToken, to: authTokenFileURL)

            let dbPath = appSupportRootURL.appendingPathComponent("runtime.db").path
            let launchAgentsDirURL = runtime.homeDirectoryURL
                .appendingPathComponent("Library")
                .appendingPathComponent("LaunchAgents")
            let launchAgentURL = launchAgentsDirURL.appendingPathComponent("\(launchAgentLabel).plist")
            try createDirectory(launchAgentsDirURL, mode: 0o755)

            let logsDirURL = runtime.homeDirectoryURL
                .appendingPathComponent("Library")
                .appendingPathComponent("Logs")
                .appendingPathComponent("personal-agent")
            try createDirectory(logsDirURL, mode: 0o755)

            let launchAgentPlist = renderLaunchAgentPlist(
                executablePath: helperExecutableURL.path,
                authTokenFilePath: authTokenFileURL.path,
                dbPath: dbPath,
                homePath: runtime.homeDirectoryURL.path,
                pathEnvironment: runtime.pathEnvironment,
                stdoutLogPath: logsDirURL.appendingPathComponent("daemon-service-macos.out.log").path,
                stderrLogPath: logsDirURL.appendingPathComponent("daemon-service-macos.err.log").path
            )
            try launchAgentPlist.write(to: launchAgentURL, atomically: true, encoding: .utf8)

            let domain = "gui/\(runtime.userID)"
            _ = try? runtime.commandRunner("/bin/launchctl", ["bootout", domain, launchAgentURL.path])

            let bootstrapResult = try runtime.commandRunner("/bin/launchctl", ["bootstrap", domain, launchAgentURL.path])
            guard bootstrapResult.exitCode == 0 else {
                throw DaemonLocalServiceInstallError.launchctlBootstrap(commandFailureDetail(bootstrapResult))
            }

            let enableTarget = "\(domain)/\(launchAgentLabel)"
            let enableResult = try runtime.commandRunner("/bin/launchctl", ["enable", enableTarget])
            guard enableResult.exitCode == 0 else {
                throw DaemonLocalServiceInstallError.launchctlEnable(commandFailureDetail(enableResult))
            }

            let kickstartResult = try runtime.commandRunner("/bin/launchctl", ["kickstart", "-k", enableTarget])
            guard kickstartResult.exitCode == 0 else {
                throw DaemonLocalServiceInstallError.launchctlKickstart(commandFailureDetail(kickstartResult))
            }

            return DaemonLocalServiceInstallResult(
                daemonAppPath: helperInstallAppURL.path,
                daemonExecutablePath: helperExecutableURL.path,
                launchAgentPath: launchAgentURL.path,
                authTokenFilePath: authTokenFileURL.path,
                helperUpdated: true
            )
        } catch let installError as DaemonLocalServiceInstallError {
            throw installError
        } catch {
            throw DaemonLocalServiceInstallError.filesystem(error.localizedDescription)
        }
    }

    private static func commandFailureDetail(_ result: DaemonLocalServiceInstallCommandResult) -> String {
        let stderr = result.stderr.trimmingCharacters(in: .whitespacesAndNewlines)
        if !stderr.isEmpty {
            return stderr
        }
        let stdout = result.stdout.trimmingCharacters(in: .whitespacesAndNewlines)
        if !stdout.isEmpty {
            return stdout
        }
        return "launchctl exited with status \(result.exitCode)"
    }

    private static func createDirectory(_ url: URL, mode: Int) throws {
        let fileManager = FileManager.default
        try fileManager.createDirectory(at: url, withIntermediateDirectories: true)
        try fileManager.setAttributes([.posixPermissions: mode], ofItemAtPath: url.path)
    }

    private static func writeToken(_ token: String, to url: URL) throws {
        let contents = token + "\n"
        try contents.write(to: url, atomically: true, encoding: .utf8)
        try FileManager.default.setAttributes([.posixPermissions: 0o600], ofItemAtPath: url.path)
    }

    private static func isDirectory(_ url: URL) -> Bool {
        var isDirectory = ObjCBool(false)
        let exists = FileManager.default.fileExists(atPath: url.path, isDirectory: &isDirectory)
        return exists && isDirectory.boolValue
    }

    private static func isExecutableFile(_ url: URL) -> Bool {
        guard let attributes = try? FileManager.default.attributesOfItem(atPath: url.path),
              let posix = attributes[.posixPermissions] as? NSNumber else {
            return false
        }
        return (posix.uint16Value & 0o111) != 0
    }

    private static func renderLaunchAgentPlist(
        executablePath: String,
        authTokenFilePath: String,
        dbPath: String,
        homePath: String,
        pathEnvironment: String,
        stdoutLogPath: String,
        stderrLogPath: String
    ) -> String {
        let escapedExecutable = xmlEscaped(executablePath)
        let escapedToken = xmlEscaped(authTokenFilePath)
        let escapedDBPath = xmlEscaped(dbPath)
        let escapedHome = xmlEscaped(homePath)
        let escapedPATH = xmlEscaped(pathEnvironment)
        let escapedStdout = xmlEscaped(stdoutLogPath)
        let escapedStderr = xmlEscaped(stderrLogPath)

        return """
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>\(launchAgentLabel)</string>
  <key>ProgramArguments</key>
  <array>
    <string>\(escapedExecutable)</string>
    <string>--listen-mode</string>
    <string>\(daemonListenMode)</string>
    <string>--listen-address</string>
    <string>\(daemonListenAddress)</string>
    <string>--auth-token-file</string>
    <string>\(escapedToken)</string>
    <string>--db</string>
    <string>\(escapedDBPath)</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
  <key>ProcessType</key>
  <string>Background</string>
  <key>EnvironmentVariables</key>
  <dict>
    <key>HOME</key>
    <string>\(escapedHome)</string>
    <key>PATH</key>
    <string>\(escapedPATH)</string>
  </dict>
  <key>StandardOutPath</key>
  <string>\(escapedStdout)</string>
  <key>StandardErrorPath</key>
  <string>\(escapedStderr)</string>
</dict>
</plist>
"""
    }

    private static func xmlEscaped(_ value: String) -> String {
        value
            .replacingOccurrences(of: "&", with: "&amp;")
            .replacingOccurrences(of: "<", with: "&lt;")
            .replacingOccurrences(of: ">", with: "&gt;")
            .replacingOccurrences(of: "\"", with: "&quot;")
            .replacingOccurrences(of: "'", with: "&apos;")
    }
}
