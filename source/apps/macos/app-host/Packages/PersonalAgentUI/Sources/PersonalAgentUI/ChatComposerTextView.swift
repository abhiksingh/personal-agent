import AppKit
import SwiftUI

struct ChatComposerTextView: NSViewRepresentable {
    @Binding var text: String
    let onSubmit: () -> Void

    func makeCoordinator() -> Coordinator {
        Coordinator(parent: self)
    }

    func makeNSView(context: Context) -> NSScrollView {
        let textView = SubmitAwareTextView()
        textView.delegate = context.coordinator
        textView.font = .preferredFont(forTextStyle: .body)
        textView.isRichText = false
        textView.isAutomaticQuoteSubstitutionEnabled = false
        textView.isAutomaticTextReplacementEnabled = false
        textView.isAutomaticSpellingCorrectionEnabled = true
        textView.drawsBackground = false
        textView.backgroundColor = .clear
        textView.onSubmit = onSubmit
        textView.textContainerInset = NSSize(width: 8, height: 10)
        textView.string = text
        textView.textContainer?.widthTracksTextView = true
        textView.textContainer?.heightTracksTextView = false
        textView.textContainer?.containerSize = NSSize(width: 0, height: CGFloat.greatestFiniteMagnitude)
        textView.minSize = NSSize(width: 0, height: 0)
        textView.maxSize = NSSize(
            width: CGFloat.greatestFiniteMagnitude,
            height: CGFloat.greatestFiniteMagnitude
        )
        textView.isVerticallyResizable = true
        textView.isHorizontallyResizable = false
        textView.setAccessibilityLabel("Chat message input")
        textView.setAccessibilityHelp("Press Enter to send. Press Shift and Enter for a new line.")
        textView.setAccessibilityIdentifier("chat-composer-native-input")

        let scrollView = NSScrollView()
        scrollView.hasVerticalScroller = true
        scrollView.borderType = .noBorder
        scrollView.autohidesScrollers = true
        scrollView.drawsBackground = false
        scrollView.documentView = textView
        scrollView.setAccessibilityIdentifier("chat-composer-scroll-view")

        context.coordinator.textView = textView
        return scrollView
    }

    func updateNSView(_ nsView: NSScrollView, context: Context) {
        guard let textView = context.coordinator.textView else {
            return
        }
        if textView.string != text {
            textView.string = text
        }
        textView.onSubmit = onSubmit
    }

    final class Coordinator: NSObject, NSTextViewDelegate {
        var parent: ChatComposerTextView
        fileprivate weak var textView: SubmitAwareTextView?

        init(parent: ChatComposerTextView) {
            self.parent = parent
        }

        func textDidChange(_ notification: Notification) {
            guard let textView else {
                return
            }
            textView.scrollRangeToVisible(textView.selectedRange())
            parent.text = textView.string
        }
    }
}

fileprivate final class SubmitAwareTextView: NSTextView {
    var onSubmit: (() -> Void)?

    override func keyDown(with event: NSEvent) {
        let isReturn = event.keyCode == 36 || event.keyCode == 76
        let modifiers = event.modifierFlags.intersection(.deviceIndependentFlagsMask)
        let hasShift = modifiers.contains(.shift)

        if isReturn && !hasShift {
            onSubmit?()
            return
        }
        super.keyDown(with: event)
    }
}
