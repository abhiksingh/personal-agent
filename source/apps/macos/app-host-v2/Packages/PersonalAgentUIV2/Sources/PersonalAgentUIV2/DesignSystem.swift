import SwiftUI

enum PATokens {
    static let radiusSM: CGFloat = 10
    static let radiusMD: CGFloat = 12
    static let radiusLG: CGFloat = 16
    static let radiusXL: CGFloat = 20
    static let radiusPill: CGFloat = 999
}

extension Font {
    static let paTitle: Font = .system(size: 22, weight: .bold, design: .rounded)
    static let paSubtitle: Font = .system(size: 12, weight: .medium, design: .rounded)
    static let paSectionTitle: Font = .system(size: 14, weight: .semibold, design: .rounded)
    static let paBody: Font = .system(size: 12, weight: .regular, design: .rounded)
    static let paCaption: Font = .system(size: 10, weight: .medium, design: .rounded)
}

extension Color {
    static let paCanvasTop = Color(hex: 0x030014)
    static let paCanvasBottom = Color(hex: 0x010A19)
    static let paSurface = Color(hex: 0x18181B)
    static let paSurfaceStrong = Color(hex: 0x111118)
    static let paSurfaceElevated = Color(hex: 0x25262A)
    static let paSurfaceMuted = Color(hex: 0x2F3238)

    static let paTextPrimary = Color.white
    static let paTextSecondary = Color.white.opacity(0.74)
    static let paTextTertiary = Color.white.opacity(0.55)

    static let paInfo = Color(hex: 0x099CED)
    static let paInfoDeep = Color(hex: 0x00168D)
    static let paPromo = Color(hex: 0x5500DC)
    static let paSuccess = Color(hex: 0x08A85B)
    static let paWarning = Color(hex: 0xF7BE00)
    static let paDanger = Color(hex: 0xE68670)
    static let paNeutral = Color(hex: 0x9CA3AF)

    static let paStrokeSoft = Color.white.opacity(0.11)
    static let paStrokeStrong = Color.white.opacity(0.18)
}

enum PAStatusTone {
    case neutral
    case info
    case success
    case warning
    case danger
}

extension PAStatusTone {
    var foreground: Color {
        switch self {
        case .neutral:
            return .paNeutral
        case .info:
            return .paInfo
        case .success:
            return .paSuccess
        case .warning:
            return .paWarning
        case .danger:
            return .paDanger
        }
    }

    var background: Color {
        switch self {
        case .neutral:
            return .paNeutral.opacity(0.18)
        case .info:
            return .paInfo.opacity(0.18)
        case .success:
            return .paSuccess.opacity(0.18)
        case .warning:
            return .paWarning.opacity(0.2)
        case .danger:
            return .paDanger.opacity(0.2)
        }
    }

    var stroke: Color {
        foreground.opacity(0.35)
    }
}

extension ReplayEventStatus {
    var statusTone: PAStatusTone {
        switch self {
        case .completed:
            return .success
        case .awaitingApproval:
            return .warning
        case .failed:
            return .danger
        case .running:
            return .info
        }
    }
}

extension ReplayDecisionStageStatus {
    var statusTone: PAStatusTone {
        switch self {
        case .completed:
            return .success
        case .pending:
            return .warning
        case .blocked:
            return .danger
        }
    }
}

extension ReplayRiskLevel {
    var statusTone: PAStatusTone {
        switch self {
        case .low:
            return .success
        case .medium:
            return .warning
        case .high:
            return .danger
        }
    }
}

extension ConnectorStatus {
    var statusTone: PAStatusTone {
        switch self {
        case .connected:
            return .success
        case .notConnected:
            return .neutral
        case .needsAttention:
            return .warning
        }
    }
}

private extension Color {
    init(hex: UInt32, opacity: Double = 1) {
        self.init(
            .sRGB,
            red: Double((hex >> 16) & 0xFF) / 255.0,
            green: Double((hex >> 8) & 0xFF) / 255.0,
            blue: Double(hex & 0xFF) / 255.0,
            opacity: opacity
        )
    }
}
