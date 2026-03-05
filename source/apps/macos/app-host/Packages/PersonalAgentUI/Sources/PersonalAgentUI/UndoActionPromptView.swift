import SwiftUI

struct UndoActionPromptView: View {
    @ObservedObject private var state: AppShellState
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @Environment(\.colorSchemeContrast) private var colorSchemeContrast

    init(state: AppShellState) {
        self.state = state
    }

    private var promptTransition: AnyTransition {
        if reduceMotion {
            return .identity
        }
        return .move(edge: .bottom).combined(with: .opacity)
    }

    var body: some View {
        if let prompt = state.activeUndoActionPrompt {
            HStack(spacing: 10) {
                VStack(alignment: .leading, spacing: 3) {
                    Text(prompt.title)
                        .font(.caption.weight(.semibold))
                    Text(prompt.message)
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }

                Spacer(minLength: 0)

                Button(prompt.actionTitle) {
                    state.performActiveUndoAction()
                }
                .buttonStyle(.borderedProminent)
                .controlSize(.small)
                .accessibilityHint("Performs the pending undo action.")

                Button {
                    state.dismissActiveUndoActionPrompt()
                } label: {
                    Image(systemName: "xmark")
                        .font(.caption2.weight(.semibold))
                }
                .buttonStyle(.plain)
                .foregroundStyle(.secondary)
                .accessibilityLabel("Dismiss undo prompt")
                .accessibilityHint("Closes the undo prompt without performing the undo action.")
            }
            .padding(10)
            .frame(maxWidth: 360, alignment: .leading)
            .background(.regularMaterial, in: RoundedRectangle(cornerRadius: 12, style: .continuous))
            .overlay(
                RoundedRectangle(cornerRadius: 12, style: .continuous)
                    .stroke(
                        Color.secondary.opacity(
                            UIAccessibilityStylePolicy.overlayBorderOpacity(contrast: colorSchemeContrast)
                        ),
                        lineWidth: colorSchemeContrast == .increased ? 1.2 : 1
                    )
            )
            .shadow(
                color: .black.opacity(
                    UIAccessibilityStylePolicy.overlayShadowOpacity(contrast: colorSchemeContrast)
                ),
                radius: 8,
                x: 0,
                y: 4
            )
            .transition(promptTransition)
            .animation(
                reduceMotion ? nil : .snappy(duration: 0.18),
                value: state.activeUndoActionPrompt?.id
            )
        }
    }
}
