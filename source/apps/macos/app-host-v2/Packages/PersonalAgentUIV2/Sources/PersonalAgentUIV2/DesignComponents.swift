import SwiftUI

enum PACardTone {
    case neutral
    case cool
    case warm
    case emerald
}

struct PAAtmosphereBackground: View {
    var body: some View {
        ZStack {
            LinearGradient(
                colors: [.paCanvasTop, .paCanvasBottom],
                startPoint: .topLeading,
                endPoint: .bottomTrailing
            )

            RadialGradient(
                colors: [.paPromo.opacity(0.14), .clear],
                center: .topTrailing,
                startRadius: 12,
                endRadius: 420
            )

            RadialGradient(
                colors: [.paInfo.opacity(0.1), .clear],
                center: .bottomLeading,
                startRadius: 12,
                endRadius: 420
            )
        }
        .ignoresSafeArea()
    }
}

struct PASectionHeader: View {
    let title: String
    let subtitle: String

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(title)
                .font(.paTitle)
                .foregroundStyle(Color.paTextPrimary)
            Text(subtitle)
                .font(.paSubtitle)
                .foregroundStyle(Color.paTextSecondary)
                .lineLimit(2)
        }
    }
}

struct PASurfaceCard<Content: View>: View {
    let title: String?
    let tone: PACardTone
    @ViewBuilder var content: Content

    init(_ title: String? = nil, tone: PACardTone = .neutral, @ViewBuilder content: () -> Content) {
        self.title = title
        self.tone = tone
        self.content = content()
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            if let title {
                Text(title)
                    .font(.paSectionTitle)
                    .foregroundStyle(Color.paTextPrimary)
            }
            content
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(8)
        .background(
            RoundedRectangle(cornerRadius: PATokens.radiusLG, style: .continuous)
                .fill(.ultraThinMaterial)
                .overlay(
                    RoundedRectangle(cornerRadius: PATokens.radiusLG, style: .continuous)
                        .fill(tintOverlay)
                )
        )
        .overlay(
            RoundedRectangle(cornerRadius: PATokens.radiusLG, style: .continuous)
                .stroke(stroke, lineWidth: 1)
        )
        .overlay(
            RoundedRectangle(cornerRadius: PATokens.radiusLG, style: .continuous)
                .stroke(
                    LinearGradient(
                        colors: [.white.opacity(0.26), .clear],
                        startPoint: .top,
                        endPoint: .bottom
                    ),
                    lineWidth: 1
                )
                .opacity(0.6)
        )
        .shadow(color: shadowColor, radius: 8, x: 0, y: 4)
    }

    private var tintOverlay: Color {
        switch tone {
        case .neutral:
            return Color.paSurfaceStrong.opacity(0.26)
        case .cool:
            return Color.paInfo.opacity(0.11)
        case .warm:
            return Color.paDanger.opacity(0.12)
        case .emerald:
            return Color.paSuccess.opacity(0.12)
        }
    }

    private var stroke: Color {
        switch tone {
        case .neutral:
            return .paStrokeSoft
        case .cool:
            return .paInfo.opacity(0.25)
        case .warm:
            return .paDanger.opacity(0.25)
        case .emerald:
            return .paSuccess.opacity(0.25)
        }
    }

    private var shadowColor: Color {
        switch tone {
        case .neutral:
            return .black.opacity(0.24)
        case .cool:
            return .paInfo.opacity(0.1)
        case .warm:
            return .paDanger.opacity(0.1)
        case .emerald:
            return .paSuccess.opacity(0.1)
        }
    }
}

struct PAStatusChip: View {
    let label: String
    var systemImage: String? = nil
    var tone: PAStatusTone = .neutral

    var body: some View {
        HStack(spacing: 4) {
            if let systemImage {
                Image(systemName: systemImage)
                    .font(.system(size: 8, weight: .semibold))
            }
            Text(label)
                .font(.system(size: 9, weight: .semibold, design: .rounded))
        }
        .foregroundStyle(tone.foreground)
        .padding(.horizontal, 6)
        .padding(.vertical, 1)
        .background(tone.background, in: Capsule())
        .overlay(Capsule().stroke(tone.stroke, lineWidth: 1))
    }
}

enum PABannerTone {
    case success
    case info
    case warning
}

struct PAInlineBanner: View {
    let text: String
    let tone: PABannerTone

    var body: some View {
        HStack(spacing: 8) {
            Image(systemName: iconName)
                .foregroundStyle(iconColor)
            Text(text)
                .font(.paBody)
                .foregroundStyle(Color.paTextPrimary)
            Spacer(minLength: 8)
        }
        .padding(7)
        .background(
            RoundedRectangle(cornerRadius: PATokens.radiusMD, style: .continuous)
                .fill(fillColor)
        )
        .overlay(
            RoundedRectangle(cornerRadius: PATokens.radiusMD, style: .continuous)
                .stroke(strokeColor, lineWidth: 1)
        )
    }

    private var fillColor: Color {
        switch tone {
        case .success:
            return Color.paSuccess.opacity(0.18)
        case .info:
            return Color.paInfo.opacity(0.18)
        case .warning:
            return Color.paWarning.opacity(0.16)
        }
    }

    private var strokeColor: Color {
        switch tone {
        case .success:
            return Color.paSuccess.opacity(0.35)
        case .info:
            return Color.paInfo.opacity(0.35)
        case .warning:
            return Color.paWarning.opacity(0.35)
        }
    }

    private var iconColor: Color {
        switch tone {
        case .success:
            return .paSuccess
        case .info:
            return .paInfo
        case .warning:
            return .paWarning
        }
    }

    private var iconName: String {
        switch tone {
        case .success:
            return "checkmark.seal.fill"
        case .info:
            return "info.circle.fill"
        case .warning:
            return "exclamationmark.triangle.fill"
        }
    }
}

private struct PAInputSurfaceModifier: ViewModifier {
    func body(content: Content) -> some View {
        content
            .padding(.horizontal, 8)
            .padding(.vertical, 6)
            .background(
                RoundedRectangle(cornerRadius: PATokens.radiusSM, style: .continuous)
                    .fill(.thinMaterial)
                    .overlay(
                        RoundedRectangle(cornerRadius: PATokens.radiusSM, style: .continuous)
                            .fill(Color.paSurfaceStrong.opacity(0.28))
                    )
            )
            .overlay(
                RoundedRectangle(cornerRadius: PATokens.radiusSM, style: .continuous)
                    .stroke(Color.paStrokeStrong, lineWidth: 1)
            )
    }
}

private struct PASubsurfaceModifier: ViewModifier {
    let tone: PACardTone

    func body(content: Content) -> some View {
        content
            .padding(7)
            .background(
                RoundedRectangle(cornerRadius: PATokens.radiusMD, style: .continuous)
                    .fill(.thinMaterial)
                    .overlay(
                        RoundedRectangle(cornerRadius: PATokens.radiusMD, style: .continuous)
                            .fill(fillColor)
                    )
            )
            .overlay(
                RoundedRectangle(cornerRadius: PATokens.radiusMD, style: .continuous)
                    .stroke(Color.paStrokeSoft, lineWidth: 1)
            )
    }

    private var fillColor: Color {
        switch tone {
        case .neutral:
            return Color.paSurfaceElevated.opacity(0.76)
        case .cool:
            return Color.paInfoDeep.opacity(0.32)
        case .warm:
            return Color.paDanger.opacity(0.24)
        case .emerald:
            return Color.paSuccess.opacity(0.22)
        }
    }
}

enum PASelectionTone {
    case info
    case warning
}

private extension PASelectionTone {
    var color: Color {
        switch self {
        case .info:
            return .paInfo
        case .warning:
            return .paWarning
        }
    }
}

private struct PASelectableSurfaceModifier: ViewModifier {
    let isSelected: Bool
    let tone: PASelectionTone
    let cornerRadius: CGFloat

    func body(content: Content) -> some View {
        content
            .background(
                RoundedRectangle(cornerRadius: cornerRadius, style: .continuous)
                    .fill(.thinMaterial)
                    .overlay(
                        RoundedRectangle(cornerRadius: cornerRadius, style: .continuous)
                            .fill(tone.color.opacity(isSelected ? 0.18 : 0.04))
                    )
            )
            .overlay(
                RoundedRectangle(cornerRadius: cornerRadius, style: .continuous)
                    .stroke(isSelected ? tone.color.opacity(0.52) : Color.paStrokeSoft, lineWidth: 1)
            )
            .overlay(
                RoundedRectangle(cornerRadius: cornerRadius, style: .continuous)
                    .stroke(
                        LinearGradient(
                            colors: [.white.opacity(0.22), .clear],
                            startPoint: .top,
                            endPoint: .bottom
                        ),
                        lineWidth: 1
                    )
                    .opacity(0.55)
            )
            .shadow(
                color: isSelected ? tone.color.opacity(0.16) : .black.opacity(0.12),
                radius: isSelected ? 7 : 4,
                x: 0,
                y: isSelected ? 4 : 2
            )
    }
}

extension View {
    func paInputSurface() -> some View {
        modifier(PAInputSurfaceModifier())
    }

    func paSubsurface(_ tone: PACardTone = .neutral) -> some View {
        modifier(PASubsurfaceModifier(tone: tone))
    }

    func paSelectableSurface(
        isSelected: Bool,
        tone: PASelectionTone = .info,
        cornerRadius: CGFloat = PATokens.radiusMD
    ) -> some View {
        modifier(PASelectableSurfaceModifier(isSelected: isSelected, tone: tone, cornerRadius: cornerRadius))
    }
}
