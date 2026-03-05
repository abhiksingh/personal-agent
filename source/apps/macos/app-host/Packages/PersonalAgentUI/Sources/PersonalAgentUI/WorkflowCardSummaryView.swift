import SwiftUI

struct WorkflowCardSummaryView: View {
    let summary: WorkflowCardSummary

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            summaryRow(title: "What happened", value: summary.whatHappened)
            summaryRow(title: "What needs action", value: summary.whatNeedsAction)
            summaryRow(title: "What next", value: summary.whatNext)
        }
        .padding(10)
        .cardSurface(.subtle)
    }

    @ViewBuilder
    private func summaryRow(title: String, value: String) -> some View {
        VStack(alignment: .leading, spacing: 2) {
            Text(title)
                .font(.caption.weight(.semibold))
                .foregroundStyle(.secondary)
            Text(value)
                .font(.callout)
                .frame(maxWidth: .infinity, alignment: .leading)
                .textSelection(.enabled)
        }
    }
}
