import SwiftUI

struct PanelLoadingSkeletonView: View {
    let title: String
    let subtitle: String
    let rowCount: Int

    init(
        title: String,
        subtitle: String,
        rowCount: Int = 3
    ) {
        self.title = title
        self.subtitle = subtitle
        self.rowCount = max(1, rowCount)
    }

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: UIStyle.standardSpacing) {
                VStack(alignment: .leading, spacing: 4) {
                    Text(title)
                        .font(.headline)
                    Text(subtitle)
                        .font(.callout)
                        .foregroundStyle(.secondary)
                }
                .frame(maxWidth: .infinity, alignment: .leading)
                .redacted(reason: .placeholder)

                ForEach(0..<rowCount, id: \.self) { index in
                    skeletonCard(index: index)
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(UIStyle.panelPadding)
        }
        .background(UIStyle.panelGradient)
    }

    private func skeletonCard(index: Int) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("Loading row \(index)")
                .font(.subheadline.weight(.semibold))
            Text("Placeholder detail one for layout continuity.")
                .font(.caption)
            Text("Placeholder detail two for stable loading geometry.")
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(12)
        .cardSurface(.standard)
        .redacted(reason: .placeholder)
    }
}
