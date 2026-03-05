import AppKit
import PersonalAgentUI
import SwiftUI

@main
struct PersonalAgentHostApp: App {
    @Environment(\.openWindow) private var openWindow
    @Environment(\.scenePhase) private var scenePhase
    @StateObject private var appState = AppShellState()

    init() {
        guard Self.isAppHostTestMode else {
            return
        }

        NSApplication.shared.setActivationPolicy(.regular)
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.2) {
            Self.focusMainWindowIfPresent()
        }
    }

    var body: some Scene {
        Window("Personal Agent", id: "main") {
            AppShellView(state: appState)
                .onAppear {
                    setMainWindowVisibility(true)
                    if Self.isAppHostTestMode {
                        Self.focusMainWindowIfPresent()
                    }
                }
                .onDisappear {
                    setMainWindowVisibility(false)
                }
                .onChange(of: scenePhase) { _, newPhase in
                    if newPhase == .active {
                        appState.handleAppDidBecomeActive()
                    }
                }
        }
        .defaultSize(width: 1320, height: 860)
        .commands {
            CommandMenu("Personal Agent") {
                Button("Do…") {
                    appState.presentDoEntryPoint()
                }
                .keyboardShortcut("d", modifiers: [.command, .shift])

                Button("Show Command Palette…") {
                    appState.presentCommandPalette()
                }
                .keyboardShortcut("p", modifiers: [.command, .shift])

                Button("Refresh Current Panel") {
                    appState.performAppCommand(.refreshCurrentSection)
                }
                .keyboardShortcut("r", modifiers: [.command])
                .disabled(!appState.isAppCommandEnabled(.refreshCurrentSection))

                Button("Open Notification Center") {
                    appState.performAppCommand(.openNotificationCenter)
                }
                .keyboardShortcut("n", modifiers: [.command, .shift])
                .disabled(!appState.isAppCommandEnabled(.openNotificationCenter))

                Button("Run Fix Next Setup Step") {
                    appState.performAppCommand(.performOnboardingFixNextStep)
                }
                .keyboardShortcut("f", modifiers: [.command, .option])
                .disabled(!appState.isAppCommandEnabled(.performOnboardingFixNextStep))
            }

            CommandMenu("Navigate") {
                Button("Configuration") { appState.performAppCommand(.openConfiguration) }
                    .keyboardShortcut("1", modifiers: [.command])
                Button("Chat") { appState.performAppCommand(.openChat) }
                    .keyboardShortcut("2", modifiers: [.command])
                Button("Communications") { appState.performAppCommand(.openCommunications) }
                    .keyboardShortcut("3", modifiers: [.command])
                Button("Automation") { appState.performAppCommand(.openAutomation) }
                    .keyboardShortcut("4", modifiers: [.command])
                Button("Approvals") { appState.performAppCommand(.openApprovals) }
                    .keyboardShortcut("5", modifiers: [.command])
                Button("Tasks") { appState.performAppCommand(.openTasks) }
                    .keyboardShortcut("6", modifiers: [.command])
                Button("Inspect") { appState.performAppCommand(.openInspect) }
                    .keyboardShortcut("7", modifiers: [.command])
                Button("Channels") { appState.performAppCommand(.openChannels) }
                    .keyboardShortcut("8", modifiers: [.command])
                Button("Connectors") { appState.performAppCommand(.openConnectors) }
                    .keyboardShortcut("9", modifiers: [.command])
                Button("Models") { appState.performAppCommand(.openModels) }
                    .keyboardShortcut("0", modifiers: [.command])
            }

            CommandMenu("Diagnostics") {
                Button("Open Inspect") { appState.performAppCommand(.openInspect) }
                    .keyboardShortcut("i", modifiers: [.command, .option])
                Button("Open Channels") { appState.performAppCommand(.openChannels) }
                    .keyboardShortcut("c", modifiers: [.command, .option])
                Button("Open Connectors") { appState.performAppCommand(.openConnectors) }
                    .keyboardShortcut("k", modifiers: [.command, .option])
            }

            CommandMenu("Runtime") {
                Button("Start Daemon") { appState.performAppCommand(.startDaemon) }
                    .keyboardShortcut("s", modifiers: [.command, .option])
                    .disabled(!appState.isAppCommandEnabled(.startDaemon))
                Button("Stop Daemon") { appState.performAppCommand(.stopDaemon) }
                    .keyboardShortcut(".", modifiers: [.command, .option])
                    .disabled(!appState.isAppCommandEnabled(.stopDaemon))
                Button("Restart Daemon") { appState.performAppCommand(.restartDaemon) }
                    .keyboardShortcut("r", modifiers: [.command, .option])
                    .disabled(!appState.isAppCommandEnabled(.restartDaemon))
            }
        }

        MenuBarExtra {
            TaskbarMenuView(
                state: appState,
                openMainWindow: {
                    openMainWindow()
                },
                closeMainWindow: {
                    closeMainWindow()
                },
                quitApp: {
                    quitFromTaskbar()
                }
            )
            .frame(width: 320)
        } label: {
            Image("MenuBarIcon")
                .renderingMode(.template)
        }
        .menuBarExtraStyle(.window)
    }

    private func openMainWindow() {
        setMainWindowVisibility(true)
        openWindow(id: "main")
        Self.focusMainWindowIfPresent()
    }

    private func closeMainWindow() {
        for window in mainWindows() {
            window.performClose(nil)
        }
        setMainWindowVisibility(false)
    }

    private func quitFromTaskbar() {
        closeMainWindow()
        Task { @MainActor in
            await appState.stopDaemonForTermination()
            NSApplication.shared.terminate(nil)
        }
    }

    private func setMainWindowVisibility(_ isVisible: Bool) {
        appState.mainWindowVisible = isVisible
        Self.updateActivationPolicyForMainWindowVisibility(
            isVisible,
            isAppHostTestMode: Self.isAppHostTestMode
        )
    }

    private func mainWindows() -> [NSWindow] {
        NSApplication.shared.windows.filter { $0.title == "Personal Agent" }
    }

    private static func updateActivationPolicyForMainWindowVisibility(
        _ isVisible: Bool,
        isAppHostTestMode: Bool
    ) {
        if isAppHostTestMode {
            NSApplication.shared.setActivationPolicy(.regular)
            return
        }

        let desiredPolicy: NSApplication.ActivationPolicy = isVisible ? .regular : .accessory
        if NSApplication.shared.activationPolicy() != desiredPolicy {
            NSApplication.shared.setActivationPolicy(desiredPolicy)
        }

        if isVisible {
            NSApplication.shared.activate(ignoringOtherApps: true)
        }
    }

    private static func focusMainWindowIfPresent() {
        NSApplication.shared.activate(ignoringOtherApps: true)
        guard let window = NSApplication.shared.windows.first(where: { $0.title == "Personal Agent" }) else {
            return
        }

        if isAppHostVisualMode {
            let fixedContentSize = NSSize(width: 1320, height: 860)
            window.minSize = fixedContentSize
            window.maxSize = fixedContentSize
            window.setContentSize(fixedContentSize)
        }

        window.makeKeyAndOrderFront(nil)
    }

    private static var isAppHostTestMode: Bool {
        isAppHostSmokeMode || isAppHostVisualMode
    }

    private static var isAppHostSmokeMode: Bool {
        let env = ProcessInfo.processInfo.environment
        if env["PA_UI_SMOKE_FIXTURE_SCENARIO"] != nil {
            return true
        }
        return env["PA_UI_APP_HOST_SMOKE_MODE"] == "1"
    }

    private static var isAppHostVisualMode: Bool {
        ProcessInfo.processInfo.environment["PA_UI_APP_HOST_VISUAL_MODE"] == "1"
    }
}
