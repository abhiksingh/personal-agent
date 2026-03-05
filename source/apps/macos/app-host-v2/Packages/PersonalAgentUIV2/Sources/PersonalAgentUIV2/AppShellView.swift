import SwiftUI

public struct AppShellV2View: View {
    @StateObject private var store = AppShellV2Store()
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    public init() {}

    public var body: some View {
        ZStack {
            PAAtmosphereBackground()

            NavigationSplitView {
                List(AssistantWorkspaceSection.allCases, selection: $store.selectedSection) { section in
                    VStack(alignment: .leading, spacing: 2) {
                        Label(section.title, systemImage: section.systemImage)
                            .font(.system(size: 12, weight: .semibold, design: .rounded))
                            .foregroundStyle(Color.paTextPrimary)
                        Text(section.subtitle)
                            .font(.system(size: 10, weight: .medium, design: .rounded))
                            .foregroundStyle(Color.paTextSecondary)
                            .lineLimit(2)
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(.vertical, 4)
                    .padding(.horizontal, 2)
                    .tag(section)
                }
                .navigationTitle("Personal Agent")
                .listStyle(.sidebar)
                .scrollContentBackground(.hidden)
                .background(Color.clear)
                .padding(4)
                .navigationSplitViewColumnWidth(min: 210, ideal: 220, max: 240)
                .accessibilityLabel("Primary Navigation")
                .accessibilityHint("Select a workflow section.")
            } detail: {
                Group {
                    switch store.selectedSection {
                    case .replayAndAsk:
                        ReplayAndAskWorkflowView(store: store)
                    case .getStarted:
                        GetStartedWorkflowView(store: store)
                    case .connectorsAndModels:
                        ConnectorsAndModelsWorkflowView(store: store)
                    }
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
                .padding(10)
            }
        }
        .navigationSplitViewStyle(.balanced)
        .tint(.paInfo)
        .foregroundStyle(Color.paTextPrimary)
        .preferredColorScheme(.dark)
        .transaction { transaction in
            if reduceMotion {
                transaction.disablesAnimations = true
                transaction.animation = nil
            }
        }
    }
}

#Preview {
    AppShellV2View()
}
