import SwiftUI

enum UIStyle {
    static let cardCornerRadius: CGFloat = 14
    static let chipCornerRadius: CGFloat = 10
    static let controlCornerRadius: CGFloat = 10

    static let panelPadding: CGFloat = 12
    static let compactSpacing: CGFloat = 6
    static let standardSpacing: CGFloat = 10
    static let sectionSpacing: CGFloat = 14

    static let panelGradient = LinearGradient(
        colors: [
            Color(nsColor: .textBackgroundColor).opacity(0.95),
            Color(nsColor: .windowBackgroundColor)
        ],
        startPoint: .top,
        endPoint: .bottom
    )
}

enum TahoeCardEmphasis {
    case subtle
    case standard
    case elevated
}

enum PanelActionRole {
    case primary
    case secondary
    case destructive
}

enum UIAccessibilityStylePolicy {
    static func cardStrokeOpacity(
        emphasis: TahoeCardEmphasis,
        contrast: ColorSchemeContrast
    ) -> Double {
        let baseValue: Double
        switch emphasis {
        case .subtle:
            baseValue = 0.22
        case .standard:
            baseValue = 0.28
        case .elevated:
            baseValue = 0.3
        }
        guard contrast == .increased else {
            return baseValue
        }
        return min(baseValue + 0.16, 0.6)
    }

    static func cardShadowOpacity(
        emphasis: TahoeCardEmphasis,
        contrast: ColorSchemeContrast
    ) -> Double {
        let baseValue: Double
        switch emphasis {
        case .subtle:
            baseValue = 0.03
        case .standard:
            baseValue = 0.05
        case .elevated:
            baseValue = 0.07
        }
        guard contrast == .increased else {
            return baseValue
        }
        return min(baseValue + 0.04, 0.14)
    }

    static func cardStrokeLineWidth(contrast: ColorSchemeContrast) -> CGFloat {
        contrast == .increased ? 1.0 : 0.7
    }

    static func cardStrokeColor(contrast: ColorSchemeContrast) -> Color {
        contrast == .increased ? .primary : .white
    }

    static func statusBadgeBackgroundOpacity(contrast: ColorSchemeContrast) -> Double {
        contrast == .increased ? 0.9 : 0.75
    }

    static func statusBadgeBorderOpacity(contrast: ColorSchemeContrast) -> Double {
        contrast == .increased ? 0.26 : 0.0
    }

    static func overlayBorderOpacity(contrast: ColorSchemeContrast) -> Double {
        contrast == .increased ? 0.28 : 0.15
    }

    static func overlayShadowOpacity(contrast: ColorSchemeContrast) -> Double {
        contrast == .increased ? 0.12 : 0.08
    }
}

private struct TahoeCardSurfaceModifier: ViewModifier {
    let emphasis: TahoeCardEmphasis
    @Environment(\.colorSchemeContrast) private var colorSchemeContrast

    private var materialStyle: AnyShapeStyle {
        switch emphasis {
        case .subtle:
            return AnyShapeStyle(.ultraThinMaterial)
        case .standard:
            return AnyShapeStyle(.thinMaterial)
        case .elevated:
            return AnyShapeStyle(.regularMaterial)
        }
    }

    func body(content: Content) -> some View {
        content
            .background(
                RoundedRectangle(cornerRadius: UIStyle.cardCornerRadius, style: .continuous)
                    .fill(materialStyle)
            )
            .overlay(
                RoundedRectangle(cornerRadius: UIStyle.cardCornerRadius, style: .continuous)
                    .strokeBorder(
                        UIAccessibilityStylePolicy.cardStrokeColor(contrast: colorSchemeContrast)
                            .opacity(
                                UIAccessibilityStylePolicy.cardStrokeOpacity(
                                    emphasis: emphasis,
                                    contrast: colorSchemeContrast
                                )
                            ),
                        lineWidth: UIAccessibilityStylePolicy.cardStrokeLineWidth(contrast: colorSchemeContrast)
                    )
            )
            .shadow(
                color: Color.black.opacity(
                    UIAccessibilityStylePolicy.cardShadowOpacity(
                        emphasis: emphasis,
                        contrast: colorSchemeContrast
                    )
                ),
                radius: 5,
                x: 0,
                y: 1
            )
    }
}

struct TahoeStatusBadge: View {
    let text: String
    let symbolName: String
    let tint: Color
    @Environment(\.colorSchemeContrast) private var colorSchemeContrast

    var body: some View {
        Label(text, systemImage: symbolName)
            .font(.caption.weight(.medium))
            .foregroundStyle(tint)
            .lineLimit(1)
            .padding(.horizontal, 10)
            .padding(.vertical, 5)
            .background(
                Capsule(style: .continuous)
                    .fill(
                        Color(nsColor: .controlBackgroundColor)
                            .opacity(
                                UIAccessibilityStylePolicy.statusBadgeBackgroundOpacity(
                                    contrast: colorSchemeContrast
                                )
                            )
                    )
            )
            .overlay {
                Capsule(style: .continuous)
                    .strokeBorder(
                        Color.primary.opacity(
                            UIAccessibilityStylePolicy.statusBadgeBorderOpacity(
                                contrast: colorSchemeContrast
                            )
                        ),
                        lineWidth: colorSchemeContrast == .increased ? 0.9 : 0
                    )
            }
    }
}

struct TahoeSectionHeader<Accessory: View>: View {
    let title: String
    let subtitle: String?
    @ViewBuilder let accessory: Accessory

    init(
        title: String,
        subtitle: String? = nil,
        @ViewBuilder accessory: () -> Accessory
    ) {
        self.title = title
        self.subtitle = subtitle
        self.accessory = accessory()
    }

    var body: some View {
        HStack(alignment: .top, spacing: UIStyle.standardSpacing) {
            VStack(alignment: .leading, spacing: 4) {
                Text(title)
                    .font(.title3.weight(.semibold))
                if let subtitle, !subtitle.isEmpty {
                    Text(subtitle)
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                }
            }
            Spacer(minLength: UIStyle.standardSpacing)
            accessory
        }
    }
}

extension View {
    func cardSurface(_ emphasis: TahoeCardEmphasis = .standard) -> some View {
        modifier(TahoeCardSurfaceModifier(emphasis: emphasis))
    }

    @ViewBuilder
    func panelActionStyle(_ role: PanelActionRole) -> some View {
        switch role {
        case .primary:
            self.buttonStyle(.borderedProminent)
        case .secondary:
            self.buttonStyle(.bordered)
        case .destructive:
            self
                .buttonStyle(.bordered)
                .tint(.red)
        }
    }

    // Prefer built-in macOS button treatments to keep control behavior native.
    func quietButtonChrome() -> some View {
        controlSize(.regular)
            .buttonStyle(.bordered)
    }

    @ViewBuilder
    func successSymbolEffect<Value: Equatable>(
        _ value: Value,
        reduceMotion: Bool
    ) -> some View {
        if reduceMotion {
            self
        } else {
            self.symbolEffect(.bounce, options: .speed(1.2), value: value)
        }
    }
}
