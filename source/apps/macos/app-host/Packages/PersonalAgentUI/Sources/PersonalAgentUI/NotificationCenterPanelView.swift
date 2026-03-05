import SwiftUI

struct NotificationCenterPanelView: View {
    @ObservedObject private var state: AppShellState
    @Environment(\.dismiss) private var dismiss

    init(state: AppShellState) {
        self.state = state
    }

    var body: some View {
        NavigationStack {
            VStack(alignment: .leading, spacing: UIStyle.standardSpacing) {
                filterBar
                notificationList
            }
            .padding(UIStyle.panelPadding)
            .navigationTitle("Notification Center")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Done") {
                        state.dismissNotificationCenter()
                        dismiss()
                    }
                }
                ToolbarItemGroup(placement: .automatic) {
                    Button("Mark All Read") {
                        state.markAllNotificationsRead()
                    }
                    .disabled(state.unreadNotificationCount == 0)

                    Button("Clear Read") {
                        state.clearReadNotifications()
                    }
                    .disabled(!state.notificationItems.contains(where: \.isRead))

                    Button("Clear All", role: .destructive) {
                        state.clearAllNotifications()
                    }
                    .disabled(state.notificationItems.isEmpty)
                }
            }
        }
    }

    private var filterBar: some View {
        HStack(spacing: UIStyle.standardSpacing) {
            TextField("Search inbox items", text: $state.notificationCenterSearchQuery)
                .textFieldStyle(.roundedBorder)

            Picker("Source", selection: $state.notificationCenterSourceFilter) {
                ForEach(state.notificationSourceOptions, id: \.self) { source in
                    Text(notificationSourceLabel(source)).tag(source)
                }
            }
            .pickerStyle(.menu)
            .frame(width: 180)
        }
    }

    @ViewBuilder
    private var notificationList: some View {
        if state.groupedFilteredNotificationSections.isEmpty {
            ContentUnavailableView(
                "No Inbox Items",
                systemImage: "bell.slash",
                description: Text("Workflow outcomes appear here with direct next steps when available.")
            )
        } else {
            List {
                ForEach(state.groupedFilteredNotificationSections) { section in
                    Section {
                        ForEach(section.items) { notification in
                            notificationRow(notification)
                        }
                    } header: {
                        HStack(spacing: 8) {
                            Label(section.intent.title, systemImage: section.intent.symbolName)
                                .font(.caption.weight(.semibold))
                                .foregroundStyle(section.intent.tint)
                            if section.unreadCount > 0 {
                                Text("\(section.unreadCount) unread")
                                    .font(.caption2.weight(.semibold))
                                    .foregroundStyle(.secondary)
                            }
                        }
                    }
                }
            }
            .listStyle(.inset)
        }
    }

    private func notificationRow(_ notification: AppNotificationItem) -> some View {
        let inboxActions = state.notificationInboxActions(for: notification)
        return VStack(alignment: .leading, spacing: 6) {
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                Image(systemName: notification.level.symbolName)
                    .foregroundStyle(notification.level.tint)
                Text(notification.message)
                    .font(.callout.weight(.semibold))
                    .foregroundStyle(.primary)
                Spacer(minLength: 0)
                Text(notification.createdAt.formatted(date: .omitted, time: .shortened))
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }

            Text("\(notificationSourceLabel(notification.source)) • \(notification.action) • \(notification.workspaceID)")
                .font(.caption)
                .foregroundStyle(.secondary)

            HStack(spacing: 8) {
                if let inboxAction = inboxActions.first {
                    Button {
                        state.performNotificationInboxAction(
                            inboxAction,
                            notificationID: notification.id
                        )
                    } label: {
                        Label(inboxAction.title, systemImage: inboxAction.symbolName)
                    }
                    .buttonStyle(.borderedProminent)
                    .controlSize(.small)
                }

                if notification.isRead {
                    Label("Read", systemImage: "checkmark.circle")
                        .font(.caption2.weight(.semibold))
                        .foregroundStyle(.secondary)
                } else {
                    Button("Mark Read") {
                        state.markNotificationRead(notificationID: notification.id)
                    }
                    .buttonStyle(.bordered)
                }

                Button("Remove", role: .destructive) {
                    state.clearNotification(notificationID: notification.id)
                }
                .buttonStyle(.bordered)
                .controlSize(.small)
            }
        }
        .padding(.vertical, 4)
    }

    private func notificationSourceLabel(_ source: String) -> String {
        let normalized = source.trimmingCharacters(in: .whitespacesAndNewlines)
        if normalized == "all" {
            return "All Sources"
        }
        if normalized.isEmpty {
            return "Unknown"
        }
        return normalized
            .split(separator: "_")
            .map { $0.capitalized }
            .joined(separator: " ")
    }
}

struct NotificationToastStackView: View {
    @ObservedObject private var state: AppShellState
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    init(state: AppShellState) {
        self.state = state
    }

    private var visibleToastItems: [AppNotificationItem] {
        Array(state.notificationToastItems.prefix(3))
    }

    private var visibleToastIDs: [String] {
        visibleToastItems.map(\.id)
    }

    private var toastTransition: AnyTransition {
        if reduceMotion {
            return .identity
        }
        return .move(edge: .trailing).combined(with: .opacity)
    }

    var body: some View {
        if visibleToastItems.isEmpty {
            EmptyView()
        } else {
            VStack(alignment: .trailing, spacing: 8) {
                ForEach(visibleToastItems) { notification in
                    NotificationToastCardView(
                        notification: notification,
                        onDismiss: {
                            state.dismissNotificationToast(notificationID: notification.id)
                        }
                    )
                    .transition(toastTransition)
                }
            }
            .frame(maxWidth: 360)
            .allowsHitTesting(true)
            .animation(reduceMotion ? nil : .snappy(duration: 0.18), value: visibleToastIDs)
        }
    }
}

private struct NotificationToastCardView: View {
    let notification: AppNotificationItem
    let onDismiss: () -> Void
    @Environment(\.colorSchemeContrast) private var colorSchemeContrast
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    var body: some View {
        HStack(alignment: .top, spacing: 10) {
            toastIcon

            VStack(alignment: .leading, spacing: 3) {
                Text(notification.message)
                    .font(.caption.weight(.semibold))
                Text("\(notification.source) • \(notification.action)")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }

            Spacer(minLength: 0)

            Button(action: onDismiss) {
                Image(systemName: "xmark")
                    .font(.caption2.weight(.semibold))
            }
            .buttonStyle(.plain)
            .foregroundStyle(.secondary)
            .accessibilityLabel("Dismiss notification toast")
            .accessibilityHint("Removes this notification from the toast stack and marks it read.")
        }
        .padding(10)
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
    }

    @ViewBuilder
    private var toastIcon: some View {
        if reduceMotion {
            Image(systemName: notification.level.symbolName)
                .foregroundStyle(notification.level.tint)
                .padding(.top, 2)
        } else {
            Image(systemName: notification.level.symbolName)
                .foregroundStyle(notification.level.tint)
                .padding(.top, 2)
                .symbolEffect(
                    .bounce,
                    options: .speed(1.15),
                    value: notification.id
                )
        }
    }
}
