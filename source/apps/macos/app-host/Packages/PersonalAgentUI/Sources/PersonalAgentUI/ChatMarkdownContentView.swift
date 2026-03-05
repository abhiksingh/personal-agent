import AppKit
import SwiftUI

enum ChatMarkdownSegment: Equatable {
    case text(AttributedString)
    case code(language: String?, content: String)
    case image(url: URL, altText: String?)
    case table(headers: [AttributedString], rows: [[AttributedString]])

    var prefersExpandedTimelineWidth: Bool {
        switch self {
        case .code, .table:
            return true
        case .text, .image:
            return false
        }
    }
}

enum ChatMarkdownParser {
    static func prefersExpandedTimelineWidth(_ text: String) -> Bool {
        parse(text).contains { $0.prefersExpandedTimelineWidth }
    }

    static func parse(_ text: String) -> [ChatMarkdownSegment] {
        let normalized = ChatTextNormalization.normalizedNewlines(text)
        guard ChatTextNormalization.normalizedNonEmpty(normalized) != nil else {
            return []
        }

        guard let attributed = fullMarkdownAttributedText(from: normalized) else {
            return fallbackTextSegment(from: normalized)
        }

        var segments: [ChatMarkdownSegment] = []
        var textBuffer = AttributedString()
        var previousTextBlock: MarkdownBlockContext? = nil
        var listPrefixAppliedForBlockID: Int? = nil

        func flushTextBuffer() {
            guard ChatTextNormalization.nonEmptyPreservingWhitespace(String(textBuffer.characters)) != nil else {
                textBuffer = AttributedString()
                previousTextBlock = nil
                listPrefixAppliedForBlockID = nil
                return
            }
            segments.append(.text(textBuffer))
            textBuffer = AttributedString()
            previousTextBlock = nil
            listPrefixAppliedForBlockID = nil
        }

        var index = attributed.runs.startIndex
        while index < attributed.runs.endIndex {
            let run = attributed.runs[index]
            let runSlice = attributed[run.range]
            let runText = String(runSlice.characters)

            if tableCellContext(from: run.presentationIntent) != nil {
                flushTextBuffer()
                let tableResult = extractTableSegment(
                    from: attributed,
                    startIndex: index
                )
                if let tableSegment = tableResult.segment {
                    segments.append(tableSegment)
                }
                index = tableResult.nextIndex
                continue
            }

            if let codeContext = codeBlockContext(from: run.presentationIntent) {
                flushTextBuffer()

                var codeContent = runText
                var nextIndex = attributed.runs.index(after: index)
                while nextIndex < attributed.runs.endIndex {
                    let nextRun = attributed.runs[nextIndex]
                    guard codeBlockContext(from: nextRun.presentationIntent)?.identity == codeContext.identity else {
                        break
                    }
                    codeContent += String(attributed[nextRun.range].characters)
                    nextIndex = attributed.runs.index(after: nextIndex)
                }

                if let normalizedCode = normalizedCodeContent(codeContent) {
                    segments.append(.code(language: codeContext.language, content: normalizedCode))
                }
                index = nextIndex
                continue
            }

            if let imageURL = run.imageURL {
                flushTextBuffer()
                segments.append(
                    .image(
                        url: imageURL,
                        altText: ChatTextNormalization.normalizedNonEmpty(runText)
                    )
                )
                index = attributed.runs.index(after: index)
                continue
            }

            let currentTextBlock = markdownBlockContext(from: run.presentationIntent)
            if let previousTextBlock,
               let currentTextBlock,
               previousTextBlock != currentTextBlock {
                textBuffer += AttributedString(
                    separatorBetweenMarkdownBlocks(
                        previous: previousTextBlock,
                        current: currentTextBlock
                    )
                )
                listPrefixAppliedForBlockID = nil
            }

            if let currentTextBlock,
               case .listItem = currentTextBlock.kind,
               listPrefixAppliedForBlockID != currentTextBlock.identity,
               let listPrefix = listItemPrefix(from: run.presentationIntent) {
                textBuffer += AttributedString(listPrefix)
                listPrefixAppliedForBlockID = currentTextBlock.identity
            }

            textBuffer += normalizeSoftBreakWhitespace(
                in: AttributedString(runSlice),
                rawText: runText,
                inlineIntent: run.inlinePresentationIntent
            )
            previousTextBlock = currentTextBlock
            index = attributed.runs.index(after: index)
        }

        flushTextBuffer()
        if segments.isEmpty {
            return fallbackTextSegment(from: normalized)
        }
        return segments
    }

    private static func fallbackTextSegment(from markdown: String) -> [ChatMarkdownSegment] {
        guard ChatTextNormalization.nonEmptyPreservingWhitespace(markdown) != nil else {
            return []
        }
        if let attributed = inlineMarkdownAttributedText(from: markdown) {
            return [.text(attributed)]
        }
        return [.text(AttributedString(markdown))]
    }

    private static func fullMarkdownAttributedText(from markdown: String) -> AttributedString? {
        do {
            return try AttributedString(
                markdown: markdown,
                options: .init(
                    interpretedSyntax: .full,
                    failurePolicy: .returnPartiallyParsedIfPossible
                )
            )
        } catch {
            return nil
        }
    }

    private static func inlineMarkdownAttributedText(from markdown: String) -> AttributedString? {
        do {
            return try AttributedString(
                markdown: markdown,
                options: .init(
                    interpretedSyntax: .inlineOnlyPreservingWhitespace,
                    failurePolicy: .returnPartiallyParsedIfPossible
                )
            )
        } catch {
            return nil
        }
    }

    private static func codeBlockContext(from intent: PresentationIntent?) -> MarkdownCodeBlockContext? {
        guard let intent else {
            return nil
        }
        for component in intent.components {
            if case .codeBlock(let languageHint) = component.kind {
                return MarkdownCodeBlockContext(
                    identity: component.identity,
                    language: ChatTextNormalization.normalizedNonEmpty(languageHint)
                )
            }
        }
        return nil
    }

    private static func tableCellContext(from intent: PresentationIntent?) -> MarkdownTableCellContext? {
        guard let intent else {
            return nil
        }

        var tableIdentity: Int? = nil
        var rowIdentity: Int? = nil
        var rowKind: MarkdownTableRowKind? = nil
        var columnIndex: Int? = nil

        for component in intent.components {
            switch component.kind {
            case .table:
                tableIdentity = component.identity
            case .tableHeaderRow:
                rowIdentity = component.identity
                rowKind = .header
            case .tableRow:
                rowIdentity = component.identity
                rowKind = .body
            case .tableCell(let column):
                columnIndex = column
            default:
                continue
            }
        }

        guard let tableIdentity,
              let rowIdentity,
              let rowKind else {
            return nil
        }

        return MarkdownTableCellContext(
            tableIdentity: tableIdentity,
            rowIdentity: rowIdentity,
            rowKind: rowKind,
            columnIndex: columnIndex ?? 0
        )
    }

    private static func extractTableSegment(
        from attributed: AttributedString,
        startIndex: AttributedString.Runs.Index
    ) -> (segment: ChatMarkdownSegment?, nextIndex: AttributedString.Runs.Index) {
        guard let startContext = tableCellContext(
            from: attributed.runs[startIndex].presentationIntent
        ) else {
            return (nil, attributed.runs.index(after: startIndex))
        }

        let tableIdentity = startContext.tableIdentity
        var headerRows: [MarkdownTableRow] = []
        var bodyRows: [MarkdownTableRow] = []
        var currentRowIdentity: Int? = nil
        var currentRowKind: MarkdownTableRowKind = .body
        var rowCellsByColumn: [Int: AttributedString] = [:]
        var sawAnyCell = false

        func finalizeCurrentRow() {
            guard currentRowIdentity != nil,
                  !rowCellsByColumn.isEmpty else {
                return
            }

            let highestColumn = rowCellsByColumn.keys.max() ?? -1
            guard highestColumn >= 0 else {
                return
            }

            let cells = (0...highestColumn).map { column -> AttributedString in
                rowCellsByColumn[column] ?? AttributedString("")
            }
            let row = MarkdownTableRow(kind: currentRowKind, cells: cells)
            switch row.kind {
            case .header:
                headerRows.append(row)
            case .body:
                bodyRows.append(row)
            }
        }

        var index = startIndex
        while index < attributed.runs.endIndex {
            let run = attributed.runs[index]
            guard let context = tableCellContext(from: run.presentationIntent),
                  context.tableIdentity == tableIdentity else {
                break
            }

            sawAnyCell = true
            if currentRowIdentity != context.rowIdentity || currentRowKind != context.rowKind {
                finalizeCurrentRow()
                currentRowIdentity = context.rowIdentity
                currentRowKind = context.rowKind
                rowCellsByColumn = [:]
            }

            let runSlice = attributed[run.range]
            let runText = String(runSlice.characters)
            let normalizedRun = normalizeSoftBreakWhitespace(
                in: AttributedString(runSlice),
                rawText: runText,
                inlineIntent: run.inlinePresentationIntent
            )

            var cellText = rowCellsByColumn[context.columnIndex] ?? AttributedString("")
            cellText += normalizedRun
            rowCellsByColumn[context.columnIndex] = cellText

            index = attributed.runs.index(after: index)
        }

        finalizeCurrentRow()
        guard sawAnyCell else {
            return (nil, index)
        }

        let headers = headerRows.first?.cells ?? []
        var rows = bodyRows.map(\.cells)
        if headerRows.count > 1 {
            rows = headerRows.dropFirst().map(\.cells) + rows
        }

        guard !headers.isEmpty || !rows.isEmpty else {
            return (nil, index)
        }

        return (
            .table(headers: headers, rows: rows),
            index
        )
    }

    private static func markdownBlockContext(from intent: PresentationIntent?) -> MarkdownBlockContext? {
        guard let intent else {
            return nil
        }

        var paragraphID: Int? = nil
        var listItemID: Int? = nil
        var listID: Int? = nil
        var orderedList = false
        var headerContext: (level: Int, identity: Int)? = nil

        for component in intent.components {
            switch component.kind {
            case .paragraph:
                paragraphID = component.identity
            case .listItem:
                listItemID = component.identity
            case .orderedList:
                listID = component.identity
                orderedList = true
            case .unorderedList:
                listID = component.identity
                orderedList = false
            case .header(let level):
                headerContext = (level: level, identity: component.identity)
            default:
                continue
            }
        }

        if let listItemID {
            return MarkdownBlockContext(
                kind: .listItem(
                    listIdentity: listID ?? -1,
                    ordered: orderedList
                ),
                identity: listItemID
            )
        }
        if let paragraphID {
            return MarkdownBlockContext(kind: .paragraph, identity: paragraphID)
        }
        if let headerContext {
            return MarkdownBlockContext(
                kind: .header(level: headerContext.level),
                identity: headerContext.identity
            )
        }
        if let firstComponent = intent.components.first {
            return MarkdownBlockContext(kind: .other, identity: firstComponent.identity)
        }
        return nil
    }

    private static func listItemPrefix(from intent: PresentationIntent?) -> String? {
        guard let intent else {
            return nil
        }

        var listDepth = 0
        var listItemOrdinal: Int? = nil
        var orderedList = false

        for component in intent.components {
            switch component.kind {
            case .orderedList:
                listDepth += 1
                orderedList = true
            case .unorderedList:
                listDepth += 1
                orderedList = false
            case .listItem(let ordinal):
                listItemOrdinal = ordinal
            default:
                continue
            }
        }

        guard let listItemOrdinal else {
            return nil
        }
        let indent = String(repeating: "  ", count: max(0, listDepth - 1))
        let marker = orderedList ? "\(listItemOrdinal)." : "-"
        return "\(indent)\(marker) "
    }

    private static func separatorBetweenMarkdownBlocks(
        previous: MarkdownBlockContext,
        current: MarkdownBlockContext
    ) -> String {
        switch (previous.kind, current.kind) {
        case let (
            .listItem(previousListIdentity, _),
            .listItem(currentListIdentity, _)
        ) where previousListIdentity == currentListIdentity:
            return "\n"
        case (.listItem, _), (_, .listItem):
            return "\n"
        default:
            return "\n\n"
        }
    }

    private static func normalizedCodeContent(_ rawCode: String) -> String? {
        var normalized = ChatTextNormalization.normalizedNewlines(rawCode)
        while normalized.hasSuffix("\n") {
            normalized.removeLast()
        }
        return ChatTextNormalization.nonEmptyPreservingWhitespace(normalized)
    }

    private static func normalizeSoftBreakWhitespace(
        in attributedText: AttributedString,
        rawText: String,
        inlineIntent: InlinePresentationIntent?
    ) -> AttributedString {
        guard let inlineIntent,
              inlineIntent.intersection([.softBreak, .lineBreak]).isEmpty == false,
              rawText.unicodeScalars.allSatisfy({ CharacterSet.whitespacesAndNewlines.contains($0) }) else {
            return attributedText
        }

        let newlineCount = max(1, rawText.count)
        return AttributedString(String(repeating: "\n", count: newlineCount))
    }
}

private struct MarkdownCodeBlockContext {
    let identity: Int
    let language: String?
}

private enum MarkdownTableRowKind {
    case header
    case body
}

private struct MarkdownTableCellContext {
    let tableIdentity: Int
    let rowIdentity: Int
    let rowKind: MarkdownTableRowKind
    let columnIndex: Int
}

private struct MarkdownTableRow {
    let kind: MarkdownTableRowKind
    let cells: [AttributedString]
}

private struct MarkdownBlockContext: Equatable {
    enum Kind: Equatable {
        case paragraph
        case listItem(listIdentity: Int, ordered: Bool)
        case header(level: Int)
        case other
    }

    let kind: Kind
    let identity: Int
}

struct ChatMarkdownContentView: View {
    enum Style {
        case body
        case caption

        var markdownFont: Font {
            switch self {
            case .body:
                return .body
            case .caption:
                return .caption
            }
        }

        var codeFont: Font {
            switch self {
            case .body:
                return .system(.body, design: .monospaced)
            case .caption:
                return .system(.caption, design: .monospaced)
            }
        }

        var codeLanguageFont: Font {
            switch self {
            case .body:
                return .caption2.weight(.semibold)
            case .caption:
                return .caption2
            }
        }

        var imageCaptionFont: Font {
            switch self {
            case .body:
                return .caption
            case .caption:
                return .caption2
            }
        }

        var imageStatusFont: Font {
            switch self {
            case .body:
                return .caption
            case .caption:
                return .caption2
            }
        }

        var imageMaxWidth: CGFloat {
            switch self {
            case .body:
                return 560
            case .caption:
                return 420
            }
        }

        var imageMinimumHeight: CGFloat {
            switch self {
            case .body:
                return 120
            case .caption:
                return 96
            }
        }

        var blockSpacing: CGFloat {
            switch self {
            case .body:
                return 8
            case .caption:
                return 6
            }
        }

        var lineSpacing: CGFloat {
            switch self {
            case .body:
                return 2
            case .caption:
                return 1
            }
        }

        var tableMinimumColumnWidth: CGFloat {
            switch self {
            case .body:
                return 140
            case .caption:
                return 108
            }
        }
    }

    let text: String
    let style: Style

    private var segments: [ChatMarkdownSegment] {
        ChatMarkdownParser.parse(text)
    }

    init(text: String, style: Style = .body) {
        self.text = text
        self.style = style
    }

    var body: some View {
        VStack(alignment: .leading, spacing: style.blockSpacing) {
            ForEach(Array(segments.enumerated()), id: \.offset) { _, segment in
                switch segment {
                case .text(let attributed):
                    textSegment(attributed)
                case .code(let language, let content):
                    codeBlock(language: language, content: content)
                case .image(let url, let altText):
                    imageBlock(url: url, altText: altText)
                case .table(let headers, let rows):
                    tableBlock(headers: headers, rows: rows)
                }
            }
        }
    }

    private func textSegment(_ attributed: AttributedString) -> some View {
        Text(attributed)
            .font(style.markdownFont)
            .lineSpacing(style.lineSpacing)
            .lineLimit(nil)
            .fixedSize(horizontal: false, vertical: true)
            .frame(maxWidth: .infinity, alignment: .leading)
            .textSelection(.enabled)
    }

    private func codeBlock(language: String?, content: String) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            if let language = ChatTextNormalization.normalizedNonEmpty(language) {
                Text(language.uppercased())
                    .font(style.codeLanguageFont)
                    .foregroundStyle(.secondary)
                    .textSelection(.enabled)
            }

            ScrollView(.horizontal) {
                Text(content)
                    .font(style.codeFont)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .textSelection(.enabled)
            }
            .accessibilityIdentifier("chat-markdown-code-block")
        }
        .padding(10)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(
            RoundedRectangle(cornerRadius: UIStyle.controlCornerRadius, style: .continuous)
                .fill(Color(nsColor: .textBackgroundColor).opacity(0.8))
        )
        .overlay(
            RoundedRectangle(cornerRadius: UIStyle.controlCornerRadius, style: .continuous)
                .stroke(Color(nsColor: .separatorColor).opacity(0.3), lineWidth: 0.5)
        )
    }

    @ViewBuilder
    private func tableBlock(headers: [AttributedString], rows: [[AttributedString]]) -> some View {
        let columnCount = max(headers.count, rows.map(\.count).max() ?? 0)
        if columnCount > 0 {
            Grid(alignment: .leading, horizontalSpacing: 0, verticalSpacing: 0) {
                if !headers.isEmpty {
                    GridRow {
                        ForEach(0..<columnCount, id: \.self) { columnIndex in
                            tableCell(
                                text: tableCellText(
                                    at: columnIndex,
                                    in: headers
                                ),
                                isHeader: true
                            )
                        }
                    }
                }

                ForEach(Array(rows.enumerated()), id: \.offset) { _, row in
                    GridRow {
                        ForEach(0..<columnCount, id: \.self) { columnIndex in
                            tableCell(
                                text: tableCellText(
                                    at: columnIndex,
                                    in: row
                                ),
                                isHeader: false
                            )
                        }
                    }
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(10)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(
                RoundedRectangle(cornerRadius: UIStyle.controlCornerRadius, style: .continuous)
                    .fill(Color(nsColor: .textBackgroundColor).opacity(0.8))
            )
            .overlay(
                RoundedRectangle(cornerRadius: UIStyle.controlCornerRadius, style: .continuous)
                    .stroke(Color(nsColor: .separatorColor).opacity(0.3), lineWidth: 0.5)
            )
            .accessibilityIdentifier("chat-markdown-table")
        }
    }

    private func tableCellText(at index: Int, in row: [AttributedString]) -> AttributedString {
        guard index < row.count else {
            return AttributedString("")
        }
        return row[index]
    }

    private func tableCell(text: AttributedString, isHeader: Bool) -> some View {
        Text(text)
            .font(
                isHeader
                    ? style.markdownFont.weight(.semibold)
                    : style.markdownFont
            )
            .lineSpacing(style.lineSpacing)
            .lineLimit(nil)
            .fixedSize(horizontal: false, vertical: true)
            .textSelection(.enabled)
            .frame(
                minWidth: 0,
                maxWidth: .infinity,
                alignment: .leading
            )
            .padding(.vertical, 6)
            .padding(.horizontal, 8)
            .background(
                isHeader
                    ? Color(nsColor: .windowBackgroundColor).opacity(0.45)
                    : Color.clear
            )
            .overlay(
                Rectangle()
                    .stroke(Color(nsColor: .separatorColor).opacity(0.35), lineWidth: 0.5)
            )
    }

    @ViewBuilder
    private func imageBlock(url: URL, altText: String?) -> some View {
        let resolvedURL = resolvedImageURL(url)

        VStack(alignment: .leading, spacing: 6) {
            if resolvedURL.isFileURL {
                if let image = NSImage(contentsOf: resolvedURL) {
                    renderedImage(Image(nsImage: image))
                } else {
                    imageFailurePlaceholder
                }
            } else {
                AsyncImage(url: resolvedURL) { phase in
                    switch phase {
                    case .empty:
                        imageLoadingPlaceholder
                    case .success(let image):
                        renderedImage(image)
                    case .failure:
                        imageFailurePlaceholder
                    @unknown default:
                        imageFailurePlaceholder
                    }
                }
            }

            if let altText = ChatTextNormalization.normalizedNonEmpty(altText) {
                Text(altText)
                    .font(style.imageCaptionFont)
                    .foregroundStyle(.secondary)
                    .textSelection(.enabled)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .accessibilityIdentifier("chat-markdown-image")
    }

    private func renderedImage(_ image: Image) -> some View {
        image
            .resizable()
            .scaledToFit()
            .frame(maxWidth: style.imageMaxWidth, alignment: .leading)
            .frame(maxWidth: .infinity, alignment: .leading)
            .clipShape(
                RoundedRectangle(
                    cornerRadius: UIStyle.controlCornerRadius,
                    style: .continuous
                )
            )
    }

    private var imageLoadingPlaceholder: some View {
        HStack(spacing: 8) {
            ProgressView()
                .controlSize(.small)
            Text("Loading image...")
                .font(style.imageStatusFont)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity, minHeight: style.imageMinimumHeight)
        .padding(8)
        .background(imagePlaceholderBackground)
    }

    private var imageFailurePlaceholder: some View {
        HStack(spacing: 8) {
            Image(systemName: "photo")
                .foregroundStyle(.secondary)
            Text("Image could not be loaded.")
                .font(style.imageStatusFont)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity, minHeight: style.imageMinimumHeight, alignment: .leading)
        .padding(8)
        .background(imagePlaceholderBackground)
    }

    private var imagePlaceholderBackground: some View {
        RoundedRectangle(cornerRadius: UIStyle.controlCornerRadius, style: .continuous)
            .fill(Color(nsColor: .textBackgroundColor).opacity(0.8))
            .overlay(
                RoundedRectangle(cornerRadius: UIStyle.controlCornerRadius, style: .continuous)
                    .stroke(Color(nsColor: .separatorColor).opacity(0.3), lineWidth: 0.5)
            )
    }

    private func resolvedImageURL(_ url: URL) -> URL {
        if url.scheme == nil, url.path.hasPrefix("/") {
            return URL(fileURLWithPath: url.path)
        }
        return url
    }
}
