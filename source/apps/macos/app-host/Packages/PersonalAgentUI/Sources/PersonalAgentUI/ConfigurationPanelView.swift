import SwiftUI

struct ConfigurationPanelView: View {
    @ObservedObject var state: AppShellState
    @State var selectedMode: ConfigurationMode = .setup
    @State var isSetupMatrixExpanded = false
    @State var isSetupRuntimeDetailsExpanded = false
    @State var isWorkspaceIdentityOperationsExpanded = false
    @State var isWorkspaceDelegationExpanded = false
    @State var isRuntimeSupervisorTimelineExpanded = false
    @State var isAdvancedLifecycleControlsExpanded = false
    @State var isPanelLatencyDiagnosticsExpanded = false
    @State var isCapabilityGrantsExpanded = false
    @State var isTrustReceiptsExpanded = false
    @State var isMemoryBrowserExpanded = false
    @State var isRetrievalInspectorExpanded = false
    @State var isChatPersonaAdvancedExpanded = false
    @StateObject var configurationDraftStore = ConfigurationDraftStore()

    init(state: AppShellState) {
        self.state = state
    }

    enum TrustReceiptInventoryKind: String, CaseIterable, Identifiable {
        case webhook
        case ingest

        var id: String { rawValue }

        var label: String {
            switch self {
            case .webhook:
                return "Webhook"
            case .ingest:
                return "Ingest"
            }
        }
    }

    enum ConfigurationMode: String, CaseIterable, Identifiable {
        case setup
        case workspace
        case integrations
        case data
        case advanced

        var id: String { rawValue }

        var title: String {
            switch self {
            case .setup:
                return "Setup"
            case .workspace:
                return "Workspace"
            case .integrations:
                return "Integrations"
            case .data:
                return "Data"
            case .advanced:
                return "Advanced"
            }
        }

        var subtitle: String {
            switch self {
            case .setup:
                return "Get the daemon, token, and chat model ready."
            case .workspace:
                return "Manage identity, persona policy, sessions, and delegation."
            case .integrations:
                return "Review capability and communication trust settings."
            case .data:
                return "Tune retention and inspect memory context."
            case .advanced:
                return "Runtime diagnostics and maintenance."
            }
        }
    }

    enum SetupMatrixStatus {
        case ready
        case attention
        case loading

        var symbolName: String {
            switch self {
            case .ready:
                return "checkmark.circle.fill"
            case .attention:
                return "exclamationmark.triangle.fill"
            case .loading:
                return "clock.arrow.circlepath"
            }
        }

        var tint: Color {
            switch self {
            case .ready:
                return .green
            case .attention:
                return .orange
            case .loading:
                return .secondary
            }
        }

        var label: String {
            switch self {
            case .ready:
                return "Ready"
            case .attention:
                return "Needs Attention"
            case .loading:
                return "Checking"
            }
        }
    }

    enum SetupMatrixRemediation: String, Identifiable {
        case startDaemon
        case installDaemon
        case repairDaemon
        case openChannels
        case openModels
        case openOnboarding

        var id: String { rawValue }

        var title: String {
            switch self {
            case .startDaemon:
                return "Start Daemon"
            case .installDaemon:
                return "Install Daemon"
            case .repairDaemon:
                return "Repair Daemon"
            case .openChannels:
                return "Open Channels"
            case .openModels:
                return "Open Models"
            case .openOnboarding:
                return "Open Onboarding"
            }
        }
    }

    struct SetupMatrixItem: Identifiable {
        let id: String
        let title: String
        let detail: String
        let status: SetupMatrixStatus
        let remediations: [SetupMatrixRemediation]
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: UIStyle.sectionSpacing) {
                header
                modeSelector
                if selectedMode != .setup, let runtimeBannerMessage {
                    RuntimeStateBanner(message: runtimeBannerMessage)
                }
                modeSections
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(UIStyle.panelPadding)
        }
        .background(UIStyle.panelGradient)
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    var runtimeBannerMessage: RuntimeStateBannerMessage? {
        RuntimeStateBannerMessage.resolve(
            daemonStatus: state.daemonStatus,
            connectionStatus: state.connectionStatus,
            detail: state.daemonStatusDetail,
            hasLoadedDaemonStatus: state.hasLoadedDaemonStatus && !state.isDaemonLifecycleLoading
        )
    }

    var header: some View {
        TahoeSectionHeader(
            title: "Configuration",
            subtitle: selectedMode.subtitle
        ) {
            EmptyView()
        }
    }

    var modeSelector: some View {
        VStack(alignment: .leading, spacing: 8) {
            Picker("Configuration Mode", selection: $selectedMode) {
                ForEach(ConfigurationMode.allCases) { mode in
                    Text(mode.title).tag(mode)
                }
            }
            .pickerStyle(.segmented)
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .cardSurface(.standard)
    }

    @ViewBuilder
    var modeSections: some View {
        switch selectedMode {
        case .setup:
            setupModeContent
        case .workspace:
            workspaceModeContent
        case .integrations:
            integrationsModeContent
        case .data:
            dataModeContent
        case .advanced:
            advancedModeContent
        }
    }

    func operatorDisclosure<Content: View>(
        title: String,
        isExpanded: Binding<Bool>,
        @ViewBuilder content: @escaping () -> Content
    ) -> some View {
        DisclosureGroup(isExpanded: isExpanded) {
            content()
                .padding(.top, 6)
        } label: {
            sectionTitle(title)
        }
        .padding(.vertical, 4)
    }

}
