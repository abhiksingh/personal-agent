import SwiftUI

struct V2PanelStateBannerView: View {
    let state: V2PanelLifecycleState
    let onAction: (V2ProblemActionID) -> Void

    var body: some View {
        guard state.kind != .ready else {
            return AnyView(EmptyView())
        }

        return AnyView(
            PASurfaceCard(tone: tone) {
                HStack(alignment: .top, spacing: 8) {
                    if state.kind == .loading {
                        ProgressView()
                            .controlSize(.small)
                    } else {
                        Image(systemName: symbolName)
                            .foregroundStyle(iconColor)
                    }

                    VStack(alignment: .leading, spacing: 5) {
                        Text(title)
                            .font(.paSectionTitle)
                            .foregroundStyle(Color.paTextPrimary)
                        Text(state.summary)
                            .font(.paBody)
                            .foregroundStyle(Color.paTextSecondary)

                        if !state.actions.isEmpty {
                            HStack(spacing: 6) {
                                ForEach(state.actions) { action in
                                    if action.isPrimary {
                                        Button(action.label) {
                                            onAction(action.actionID)
                                        }
                                        .buttonStyle(.borderedProminent)
                                        .tint(.paInfo)
                                    } else {
                                        Button(action.label) {
                                            onAction(action.actionID)
                                        }
                                        .buttonStyle(.bordered)
                                        .tint(.paNeutral)
                                    }
                                }
                            }
                            .controlSize(.small)
                        }
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                }
            }
        )
    }

    private var tone: PACardTone {
        switch state.kind {
        case .loading:
            return .cool
        case .ready:
            return .neutral
        case .empty:
            return .neutral
        case .degraded:
            return .warm
        case .error:
            return .warm
        }
    }

    private var title: String {
        switch state.kind {
        case .loading:
            return "Loading"
        case .ready:
            return "Ready"
        case .empty:
            return "Nothing Here Yet"
        case .degraded:
            return "Needs Attention"
        case .error:
            return "Action Needed"
        }
    }

    private var symbolName: String {
        switch state.kind {
        case .loading:
            return "hourglass"
        case .ready:
            return "checkmark.circle.fill"
        case .empty:
            return "tray"
        case .degraded:
            return "exclamationmark.triangle.fill"
        case .error:
            return "xmark.octagon.fill"
        }
    }

    private var iconColor: Color {
        switch state.kind {
        case .loading:
            return .paInfo
        case .ready:
            return .paSuccess
        case .empty:
            return .paNeutral
        case .degraded:
            return .paWarning
        case .error:
            return .paDanger
        }
    }
}
