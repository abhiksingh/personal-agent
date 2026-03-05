import Foundation
import SwiftUI

@MainActor
final class AppWorkflowMutationStore: ObservableObject {
    @Published var pendingHighImpactActionConfirmation: HighImpactActionConfirmation? = nil
    @Published var activeUndoActionPrompt: UndoActionPrompt? = nil

    private var pendingHighImpactActionHandler: (() -> Void)?
    private var activeUndoActionHandler: (() -> Void)?
    private var undoPromptDismissTask: Task<Void, Never>?

    func presentHighImpactActionConfirmation(
        title: String,
        message: String,
        confirmButtonTitle: String,
        isDestructive: Bool,
        irreversibleNote: String? = nil,
        onConfirm: @escaping () -> Void
    ) {
        pendingHighImpactActionConfirmation = HighImpactActionConfirmation(
            title: title,
            message: message,
            confirmButtonTitle: confirmButtonTitle,
            isDestructive: isDestructive,
            irreversibleNote: irreversibleNote
        )
        pendingHighImpactActionHandler = onConfirm
    }

    func confirmPendingHighImpactAction() {
        guard let handler = pendingHighImpactActionHandler else {
            pendingHighImpactActionConfirmation = nil
            return
        }
        pendingHighImpactActionConfirmation = nil
        pendingHighImpactActionHandler = nil
        handler()
    }

    func cancelPendingHighImpactAction() {
        pendingHighImpactActionConfirmation = nil
        pendingHighImpactActionHandler = nil
    }

    func presentUndoActionPrompt(
        title: String,
        message: String,
        actionTitle: String = "Undo",
        visibleForSeconds: TimeInterval = 8,
        onUndo: @escaping () -> Void
    ) {
        undoPromptDismissTask?.cancel()
        activeUndoActionPrompt = UndoActionPrompt(
            title: title,
            message: message,
            actionTitle: actionTitle
        )
        activeUndoActionHandler = onUndo
        undoPromptDismissTask = Task { [weak self] in
            guard visibleForSeconds > 0 else {
                return
            }
            let duration = UInt64(visibleForSeconds * 1_000_000_000)
            try? await Task.sleep(nanoseconds: duration)
            guard !Task.isCancelled else {
                return
            }
            await MainActor.run {
                self?.dismissActiveUndoActionPrompt()
            }
        }
    }

    func performActiveUndoAction() {
        guard let handler = activeUndoActionHandler else {
            dismissActiveUndoActionPrompt()
            return
        }
        dismissActiveUndoActionPrompt()
        handler()
    }

    func dismissActiveUndoActionPrompt() {
        undoPromptDismissTask?.cancel()
        undoPromptDismissTask = nil
        activeUndoActionPrompt = nil
        activeUndoActionHandler = nil
    }
}
