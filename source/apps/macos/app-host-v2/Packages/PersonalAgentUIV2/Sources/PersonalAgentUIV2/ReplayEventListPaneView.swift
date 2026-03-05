import SwiftUI

struct ReplayEventListPaneView: View {
    let events: [ReplayEvent]
    @Binding var selectedEventID: ReplayEvent.ID?
    let canLoadMore: Bool
    let isLoadingMore: Bool
    let onLoadMore: () -> Void

    var body: some View {
        ScrollView {
            LazyVStack(spacing: V2WorkflowLayout.listRowSpacing) {
                ForEach(events) { event in
                    Button {
                        selectedEventID = event.id
                    } label: {
                        ReplayEventRowView(
                            event: event,
                            isSelected: selectedEventID == event.id
                        )
                    }
                    .buttonStyle(.plain)
                    .accessibilityLabel(replayRowAccessibilityLabel(event))
                    .accessibilityHint("Opens replay detail.")
                }

                if canLoadMore || isLoadingMore {
                    HStack {
                        Spacer(minLength: 0)
                        Button(isLoadingMore ? "Loading…" : "Load More") {
                            onLoadMore()
                        }
                        .buttonStyle(.bordered)
                        .disabled(isLoadingMore)
                        Spacer(minLength: 0)
                    }
                    .padding(.vertical, 4)
                }
            }
        }
        .frame(minWidth: 300, idealWidth: 430)
        .accessibilityLabel("Replay Activity List")
    }

    private func replayRowAccessibilityLabel(_ event: ReplayEvent) -> String {
        "\(event.source.label). \(event.status.label). \(event.instruction)"
    }
}

private struct ReplayEventRowView: View {
    let event: ReplayEvent
    let isSelected: Bool

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(spacing: 6) {
                Label(event.source.label, systemImage: event.source.systemImage)
                    .font(.system(size: 12, weight: .medium, design: .rounded))
                    .foregroundStyle(Color.paTextSecondary)
                Spacer(minLength: 8)
                Text(event.receivedAt, format: .dateTime.hour().minute())
                    .font(.paCaption)
                    .foregroundStyle(Color.paTextTertiary)
            }

            Text(event.instruction)
                .font(.system(size: 14, weight: .semibold, design: .rounded))
                .lineLimit(2)
                .foregroundStyle(Color.paTextPrimary)

            HStack(spacing: 6) {
                PAStatusChip(label: event.status.label, systemImage: event.status.systemImage, tone: event.status.statusTone)
                PAStatusChip(label: "Risk \(event.risk.label)", tone: event.risk.statusTone)
                Text("Confidence: \(event.confidenceScore)%")
                    .font(.paCaption)
            }
            .foregroundStyle(Color.paTextSecondary)
        }
        .padding(10)
        .paSelectableSurface(isSelected: isSelected, tone: .info)
        .padding(.vertical, 2)
        .accessibilityElement(children: .combine)
    }
}
