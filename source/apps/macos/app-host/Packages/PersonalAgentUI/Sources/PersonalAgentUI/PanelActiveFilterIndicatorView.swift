import SwiftUI

struct PanelActiveFilterIndicatorView: View {
    let summaryParts: [String]
    let clearButtonTitle: String
    let clearAction: () -> Void

    private var countLabel: String {
        let count = summaryParts.count
        return "\(count) active filter\(count == 1 ? "" : "s")"
    }

    private var summaryText: String {
        summaryParts.joined(separator: " • ")
    }

    var body: some View {
        HStack(spacing: 8) {
            Label(countLabel, systemImage: "line.3.horizontal.decrease.circle.fill")
                .font(.caption.weight(.semibold))
                .foregroundStyle(.secondary)

            Text(summaryText)
                .font(.caption)
                .foregroundStyle(.secondary)
                .lineLimit(1)
                .truncationMode(.tail)

            Spacer(minLength: 0)

            Button(clearButtonTitle, action: clearAction)
                .buttonStyle(.bordered)
                .controlSize(.small)
        }
        .padding(10)
        .cardSurface(.subtle)
        .accessibilityElement(children: .combine)
        .accessibilityLabel("Active filter summary")
        .accessibilityValue(summaryText)
    }
}
