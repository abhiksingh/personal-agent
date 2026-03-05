import SwiftUI

struct PanelProblemRemediationCardView: View {
    let context: PanelProblemRemediationContext
    let onAction: (PanelProblemRemediationActionID) -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack(spacing: 8) {
                Label(context.kind.title, systemImage: context.kind.symbolName)
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(context.kind.tint)
                Spacer(minLength: 0)
                TahoeStatusBadge(
                    text: context.section.title,
                    symbolName: context.section.symbolName,
                    tint: .secondary
                )
                .controlSize(.small)
            }

            Text(context.detail)
                .font(.caption)
                .foregroundStyle(.secondary)

            HStack(spacing: 8) {
                ForEach(context.actions) { action in
                    Button {
                        onAction(action.actionID)
                    } label: {
                        Label(action.title, systemImage: action.symbolName)
                    }
                    .panelActionStyle(action.role)
                    .disabled(!action.isEnabled)
                }
            }

            if let disabledReason = context.actions.first(where: { !$0.isEnabled })?.disabledReason,
               !disabledReason.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                Text(disabledReason)
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .cardSurface(.subtle)
        .accessibilityIdentifier("panel-problem-remediation-\(context.section.rawValue)")
    }
}
