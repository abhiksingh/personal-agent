import AppKit
import SwiftUI

struct IdentityValueInlineView: View {
    let displayText: String
    let rawID: String?
    var valueFont: Font = .caption
    var rawIDLabel: String = "Raw ID"

    @State private var rawIDRevealed = false

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            HStack(alignment: .firstTextBaseline, spacing: 6) {
                Text(displayText)
                    .font(valueFont)
                if rawID != nil {
                    Menu {
                        Button(rawIDRevealed ? "Hide ID" : "Reveal ID") {
                            rawIDRevealed.toggle()
                        }
                        if let rawID {
                            Button("Copy ID") {
                                copyToPasteboard(rawID)
                            }
                        }
                    } label: {
                        Image(systemName: "ellipsis.circle")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                            .accessibilityLabel("Identity actions")
                    }
                    .menuStyle(.borderlessButton)
                    .controlSize(.mini)
                }
            }

            if rawIDRevealed, let rawID {
                HStack(alignment: .firstTextBaseline, spacing: 6) {
                    Text("\(rawIDLabel):")
                        .font(.caption2.weight(.semibold))
                        .foregroundStyle(.secondary)
                    Text(rawID)
                        .font(.caption2.monospaced())
                        .foregroundStyle(.secondary)
                        .textSelection(.enabled)
                    Button("Copy") {
                        copyToPasteboard(rawID)
                    }
                    .buttonStyle(.borderless)
                    .font(.caption2)
                }
            }
        }
    }

    private func copyToPasteboard(_ value: String) {
        NSPasteboard.general.clearContents()
        NSPasteboard.general.setString(value, forType: .string)
    }
}

struct IdentityDetailRowView: View {
    let label: String
    let displayText: String
    let rawID: String?
    var labelWidth: CGFloat = 92
    var valueFont: Font = .caption

    var body: some View {
        HStack(alignment: .top, spacing: 10) {
            Text(label)
                .font(.caption.weight(.semibold))
                .foregroundStyle(.secondary)
                .frame(width: labelWidth, alignment: .leading)
            IdentityValueInlineView(
                displayText: displayText,
                rawID: rawID,
                valueFont: valueFont
            )
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }
}
