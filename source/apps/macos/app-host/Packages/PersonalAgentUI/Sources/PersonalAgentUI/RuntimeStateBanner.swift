import SwiftUI

struct RuntimeStateBannerMessage {
    let title: String
    let detail: String
    let symbolName: String
    let tint: Color

    static func resolve(
        daemonStatus: DaemonStatus,
        connectionStatus: ConnectionStatus,
        detail: String,
        hasLoadedDaemonStatus: Bool = true
    ) -> RuntimeStateBannerMessage? {
        if !hasLoadedDaemonStatus {
            return nil
        }
        switch connectionStatus {
        case .disconnected:
            return RuntimeStateBannerMessage(
                title: "Daemon connection unavailable",
                detail: "Verify daemon runtime and local auth token. \(detail)",
                symbolName: "bolt.slash.circle",
                tint: .orange
            )
        case .degraded:
            return RuntimeStateBannerMessage(
                title: "Daemon connection degraded",
                detail: "Some status and action updates may be delayed or incomplete. \(detail)",
                symbolName: "exclamationmark.triangle.fill",
                tint: .orange
            )
        case .connected:
            switch daemonStatus {
            case .missing:
                return RuntimeStateBannerMessage(
                    title: "Daemon not installed",
                    detail: "Install or repair daemon setup from Configuration > Advanced.",
                    symbolName: "wrench.and.screwdriver.fill",
                    tint: .orange
                )
            case .broken:
                return RuntimeStateBannerMessage(
                    title: "Daemon setup needs repair",
                    detail: "Runtime controls may fail until the daemon installation is repaired.",
                    symbolName: "exclamationmark.triangle.fill",
                    tint: .orange
                )
            case .stopped:
                return RuntimeStateBannerMessage(
                    title: "Daemon is stopped",
                    detail: "Start the daemon to resume live updates and action execution.",
                    symbolName: "pause.circle.fill",
                    tint: .secondary
                )
            case .unknown:
                return RuntimeStateBannerMessage(
                    title: "Daemon status unknown",
                    detail: "Refresh status to confirm runtime readiness.",
                    symbolName: "questionmark.circle.fill",
                    tint: .secondary
                )
            case .running:
                return nil
            }
        }
    }
}

struct RuntimeStateBanner: View {
    let message: RuntimeStateBannerMessage

    var body: some View {
        HStack(alignment: .top, spacing: 10) {
            Image(systemName: message.symbolName)
                .font(.callout.weight(.semibold))
                .foregroundStyle(message.tint)
                .padding(.top, 1)

            VStack(alignment: .leading, spacing: 4) {
                Text(message.title)
                    .font(.subheadline.weight(.semibold))
                Text(message.detail)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }

            Spacer(minLength: 0)
        }
        .padding(11)
        .frame(maxWidth: .infinity, alignment: .leading)
        .cardSurface(.subtle)
        .accessibilityElement(children: .combine)
        .accessibilityLabel("Runtime status notice")
        .accessibilityValue("\(message.title). \(message.detail)")
    }
}
