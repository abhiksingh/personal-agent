import SwiftUI

struct InspectComponentGalleryView: View {
    private let emptyStateActions: [EmptyStateRemediationAction] = [
        EmptyStateRemediationAction(
            actionID: .refreshInspect,
            title: "Refresh Logs",
            symbolName: "arrow.clockwise",
            isProminent: true
        ),
        EmptyStateRemediationAction(
            actionID: .openTasks,
            title: "Open Tasks",
            symbolName: "checklist"
        ),
        EmptyStateRemediationAction(
            actionID: .openConfiguration,
            title: "Open Configuration",
            symbolName: "gearshape"
        )
    ]

    var body: some View {
        ScrollView {
            LazyVStack(alignment: .leading, spacing: UIStyle.sectionSpacing) {
                InspectComponentGallerySection(
                    title: "Status Badges",
                    subtitle: "Canonical status treatments used across workflow rows and cards."
                ) {
                    HStack(spacing: 8) {
                        TahoeStatusBadge(
                            text: "Ready",
                            symbolName: "checkmark.circle.fill",
                            tint: .green
                        )
                        TahoeStatusBadge(
                            text: "Attention",
                            symbolName: "exclamationmark.triangle.fill",
                            tint: .orange
                        )
                        TahoeStatusBadge(
                            text: "Blocked",
                            symbolName: "xmark.octagon.fill",
                            tint: .red
                        )
                    }
                }

                InspectComponentGallerySection(
                    title: "Action Hierarchy",
                    subtitle: "Primary, secondary, and destructive controls should remain visually distinct."
                ) {
                    HStack(spacing: 8) {
                        Button("Primary Action") {}
                            .panelActionStyle(.primary)
                            .controlSize(.small)
                        Button("Secondary Action") {}
                            .panelActionStyle(.secondary)
                            .controlSize(.small)
                        Button("Destructive Action") {}
                            .panelActionStyle(.destructive)
                            .controlSize(.small)
                    }
                    HStack(spacing: 8) {
                        Button("Blocked Primary") {}
                            .panelActionStyle(.primary)
                            .controlSize(.small)
                            .disabled(true)
                        Button("Blocked Secondary") {}
                            .panelActionStyle(.secondary)
                            .controlSize(.small)
                            .disabled(true)
                    }
                }

                InspectComponentGallerySection(
                    title: "Card Surfaces",
                    subtitle: "Shared card emphasis levels keep panel hierarchy consistent."
                ) {
                    VStack(alignment: .leading, spacing: 8) {
                        galleryCard(label: "Subtle", emphasis: .subtle)
                        galleryCard(label: "Standard", emphasis: .standard)
                        galleryCard(label: "Elevated", emphasis: .elevated)
                    }
                }

                InspectComponentGallerySection(
                    title: "Runtime Banners",
                    subtitle: "Runtime notices should use deterministic iconography and remediation-focused copy."
                ) {
                    VStack(spacing: 8) {
                        RuntimeStateBanner(
                            message: RuntimeStateBannerMessage(
                                title: "Daemon connection degraded",
                                detail: "Some status and action updates may be delayed or incomplete.",
                                symbolName: "exclamationmark.triangle.fill",
                                tint: .orange
                            )
                        )
                        RuntimeStateBanner(
                            message: RuntimeStateBannerMessage(
                                title: "Daemon is stopped",
                                detail: "Start the daemon to resume live updates and action execution.",
                                symbolName: "pause.circle.fill",
                                tint: .secondary
                            )
                        )
                    }
                }

                InspectComponentGallerySection(
                    title: "Empty-State Remediation",
                    subtitle: "Empty and no-data flows should include direct remediation controls."
                ) {
                    PanelRemediationEmptyStateView(
                        title: "No Activity Yet",
                        systemImage: "doc.text.magnifyingglass",
                        description: "Workflow activity updates will appear here when available.",
                        statusMessage: "Sample status message for regression preview.",
                        actions: emptyStateActions,
                        onAction: { _ in }
                    )
                    .frame(maxWidth: .infinity, minHeight: 220)
                    .cardSurface(.subtle)
                }

                InspectComponentGallerySection(
                    title: "Filter-Bar Card",
                    subtitle: "Shared filter-bar card styling for major workflow panels."
                ) {
                    PanelFilterBarCard(summaryText: "Showing 12 of 24 rows in this preview.") {
                        HStack(spacing: 8) {
                            HStack(spacing: 6) {
                                Image(systemName: "magnifyingglass")
                                    .foregroundStyle(.secondary)
                                Text("Search field styling sample")
                                    .font(.callout)
                                    .foregroundStyle(.secondary)
                            }
                            .padding(.horizontal, 10)
                            .padding(.vertical, 8)
                            .background(
                                RoundedRectangle(cornerRadius: UIStyle.controlCornerRadius, style: .continuous)
                                    .fill(Color(nsColor: .textBackgroundColor).opacity(0.82))
                            )
                            Spacer(minLength: 0)
                            Button("Clear Filters") {}
                                .panelActionStyle(.secondary)
                                .controlSize(.small)
                        }
                    }
                }
            }
            .padding(UIStyle.panelPadding)
        }
        .background(UIStyle.panelGradient)
        .accessibilityIdentifier("inspect-component-gallery")
    }

    private func galleryCard(
        label: String,
        emphasis: TahoeCardEmphasis
    ) -> some View {
        HStack(spacing: 8) {
            TahoeStatusBadge(
                text: label,
                symbolName: "rectangle.on.rectangle",
                tint: .blue
            )
            Text("Sample card content")
                .font(.callout)
                .foregroundStyle(.secondary)
            Spacer(minLength: 0)
        }
        .padding(10)
        .cardSurface(emphasis)
    }
}

private struct InspectComponentGallerySection<Content: View>: View {
    let title: String
    let subtitle: String
    @ViewBuilder let content: Content

    init(
        title: String,
        subtitle: String,
        @ViewBuilder content: () -> Content
    ) {
        self.title = title
        self.subtitle = subtitle
        self.content = content()
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(title)
                .font(.headline)
            Text(subtitle)
                .font(.caption)
                .foregroundStyle(.secondary)
            content
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(12)
        .cardSurface(.standard)
    }
}
