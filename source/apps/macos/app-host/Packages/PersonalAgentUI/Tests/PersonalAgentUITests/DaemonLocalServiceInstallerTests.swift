import Foundation
import XCTest
@testable import PersonalAgentUI

final class DaemonLocalServiceInstallerTests: XCTestCase {
    func testInstallFailsWhenAppOutsideApplications() {
        let runtime = DaemonLocalServiceInstallerRuntime(
            appBundleURL: URL(fileURLWithPath: "/tmp/PersonalAgent.app"),
            homeDirectoryURL: URL(fileURLWithPath: "/tmp"),
            pathEnvironment: "/usr/bin:/bin",
            userID: 501,
            commandRunner: { _, _ in
                XCTFail("command runner should not be invoked when app path is invalid")
                return DaemonLocalServiceInstallCommandResult(exitCode: 0, stdout: "", stderr: "")
            }
        )

        XCTAssertThrowsError(
            try DaemonLocalServiceInstaller.installOrRepair(
                action: "install",
                authToken: "token",
                runtime: runtime
            )
        ) { error in
            guard case .appNotInApplications(let appPath) = error as? DaemonLocalServiceInstallError else {
                return XCTFail("expected appNotInApplications error, got \(error)")
            }
            XCTAssertEqual(appPath, "/tmp/PersonalAgent.app")
        }
    }

    func testInstallCopiesHelperWritesLaunchAgentAndRunsLaunchctl() throws {
        let tempRoot = try makeTempDirectory("installer-success")
        defer { try? FileManager.default.removeItem(at: tempRoot) }

        let appBundleURL = tempRoot.appendingPathComponent("PersonalAgent.app")
        try createEmbeddedDaemonBundle(in: appBundleURL, marker: "new")
        let homeDirectoryURL = tempRoot.appendingPathComponent("home")
        try FileManager.default.createDirectory(at: homeDirectoryURL, withIntermediateDirectories: true)

        var launchctlInvocations: [[String]] = []
        let runtime = DaemonLocalServiceInstallerRuntime(
            appBundleURL: appBundleURL,
            homeDirectoryURL: homeDirectoryURL,
            pathEnvironment: "/usr/bin:/bin",
            userID: 501,
            commandRunner: { _, arguments in
                launchctlInvocations.append(arguments)
                return DaemonLocalServiceInstallCommandResult(exitCode: 0, stdout: "", stderr: "")
            }
        )

        let result = try DaemonLocalServiceInstaller.installOrRepair(
            action: "install",
            authToken: "token-123",
            runtime: runtime,
            requireSystemApplicationsInstall: false
        )

        XCTAssertTrue(FileManager.default.fileExists(atPath: result.daemonExecutablePath))
        XCTAssertTrue(FileManager.default.fileExists(atPath: result.launchAgentPath))
        XCTAssertTrue(FileManager.default.fileExists(atPath: result.authTokenFilePath))
        XCTAssertTrue(result.summaryText.contains("Daemon helper installed"))

        let markerPath = URL(fileURLWithPath: result.daemonAppPath)
            .appendingPathComponent("Contents")
            .appendingPathComponent("Resources")
            .appendingPathComponent("build-id.txt")
            .path
        let markerValue = try String(contentsOfFile: markerPath, encoding: .utf8)
        XCTAssertEqual(markerValue, "new")

        let tokenValue = try String(contentsOfFile: result.authTokenFilePath, encoding: .utf8)
        XCTAssertEqual(tokenValue, "token-123\n")

        XCTAssertEqual(launchctlInvocations.count, 4)
        XCTAssertEqual(launchctlInvocations[0].first, "bootout")
        XCTAssertEqual(launchctlInvocations[1].first, "bootstrap")
        XCTAssertEqual(launchctlInvocations[2].first, "enable")
        XCTAssertEqual(launchctlInvocations[3].first, "kickstart")
    }

    func testRepairReplacesStaleHelperBundle() throws {
        let tempRoot = try makeTempDirectory("installer-repair")
        defer { try? FileManager.default.removeItem(at: tempRoot) }

        let appBundleURL = tempRoot.appendingPathComponent("PersonalAgent.app")
        try createEmbeddedDaemonBundle(in: appBundleURL, marker: "fresh")

        let homeDirectoryURL = tempRoot.appendingPathComponent("home")
        let staleHelperMarkerURL = homeDirectoryURL
            .appendingPathComponent("Library")
            .appendingPathComponent("Application Support")
            .appendingPathComponent("personal-agent")
            .appendingPathComponent("daemon")
            .appendingPathComponent("Personal Agent Daemon.app")
            .appendingPathComponent("Contents")
            .appendingPathComponent("Resources")
            .appendingPathComponent("build-id.txt")
        try FileManager.default.createDirectory(
            at: staleHelperMarkerURL.deletingLastPathComponent(),
            withIntermediateDirectories: true
        )
        try "stale".write(to: staleHelperMarkerURL, atomically: true, encoding: .utf8)

        let runtime = DaemonLocalServiceInstallerRuntime(
            appBundleURL: appBundleURL,
            homeDirectoryURL: homeDirectoryURL,
            pathEnvironment: "/usr/bin:/bin",
            userID: 501,
            commandRunner: { _, _ in
                DaemonLocalServiceInstallCommandResult(exitCode: 0, stdout: "", stderr: "")
            }
        )

        let result = try DaemonLocalServiceInstaller.installOrRepair(
            action: "repair",
            authToken: "token",
            runtime: runtime,
            requireSystemApplicationsInstall: false
        )

        let markerPath = URL(fileURLWithPath: result.daemonAppPath)
            .appendingPathComponent("Contents")
            .appendingPathComponent("Resources")
            .appendingPathComponent("build-id.txt")
            .path
        let markerValue = try String(contentsOfFile: markerPath, encoding: .utf8)
        XCTAssertEqual(markerValue, "fresh")
    }

    func testInstallSurfacesLaunchctlBootstrapFailure() throws {
        let tempRoot = try makeTempDirectory("installer-bootstrap-failure")
        defer { try? FileManager.default.removeItem(at: tempRoot) }

        let appBundleURL = tempRoot.appendingPathComponent("PersonalAgent.app")
        try createEmbeddedDaemonBundle(in: appBundleURL, marker: "new")

        let runtime = DaemonLocalServiceInstallerRuntime(
            appBundleURL: appBundleURL,
            homeDirectoryURL: tempRoot.appendingPathComponent("home"),
            pathEnvironment: "/usr/bin:/bin",
            userID: 501,
            commandRunner: { _, arguments in
                if arguments.first == "bootstrap" {
                    return DaemonLocalServiceInstallCommandResult(
                        exitCode: 5,
                        stdout: "",
                        stderr: "bootstrap failed"
                    )
                }
                return DaemonLocalServiceInstallCommandResult(exitCode: 0, stdout: "", stderr: "")
            }
        )

        XCTAssertThrowsError(
            try DaemonLocalServiceInstaller.installOrRepair(
                action: "install",
                authToken: "token",
                runtime: runtime,
                requireSystemApplicationsInstall: false
            )
        ) { error in
            guard case .launchctlBootstrap(let detail) = error as? DaemonLocalServiceInstallError else {
                return XCTFail("expected launchctlBootstrap error, got \(error)")
            }
            XCTAssertTrue(detail.contains("bootstrap failed"))
        }
    }

    private func makeTempDirectory(_ suffix: String) throws -> URL {
        let url = FileManager.default.temporaryDirectory
            .appendingPathComponent("pa-ui-\(suffix)-\(UUID().uuidString)")
        try FileManager.default.createDirectory(at: url, withIntermediateDirectories: true)
        return url
    }

    private func createEmbeddedDaemonBundle(in appBundleURL: URL, marker: String) throws {
        let executableURL = appBundleURL
            .appendingPathComponent("Contents")
            .appendingPathComponent("Resources")
            .appendingPathComponent("Daemon")
            .appendingPathComponent("Personal Agent Daemon.app")
            .appendingPathComponent("Contents")
            .appendingPathComponent("MacOS")
            .appendingPathComponent("personal-agent-daemon")
        try FileManager.default.createDirectory(
            at: executableURL.deletingLastPathComponent(),
            withIntermediateDirectories: true
        )
        try "#!/bin/sh\necho daemon\n".write(to: executableURL, atomically: true, encoding: .utf8)
        try FileManager.default.setAttributes([.posixPermissions: 0o755], ofItemAtPath: executableURL.path)

        let markerURL = appBundleURL
            .appendingPathComponent("Contents")
            .appendingPathComponent("Resources")
            .appendingPathComponent("Daemon")
            .appendingPathComponent("Personal Agent Daemon.app")
            .appendingPathComponent("Contents")
            .appendingPathComponent("Resources")
            .appendingPathComponent("build-id.txt")
        try FileManager.default.createDirectory(
            at: markerURL.deletingLastPathComponent(),
            withIntermediateDirectories: true
        )
        try marker.write(to: markerURL, atomically: true, encoding: .utf8)
    }
}
