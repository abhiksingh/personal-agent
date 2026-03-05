import SwiftUI

struct SetupBlockerBannerView: View {
    let blocker: SetupChecklistItem
    let onFixNext: () -> Void
    let onOpenGetStarted: () -> Void

    var body: some View {
        PASurfaceCard(tone: .warm) {
            HStack(alignment: .top, spacing: 8) {
                Image(systemName: "exclamationmark.triangle.fill")
                    .foregroundStyle(Color.paWarning)

                VStack(alignment: .leading, spacing: 6) {
                    HStack(spacing: 6) {
                        Text("Current Blocker")
                            .font(.paSectionTitle)
                        PAStatusChip(label: blocker.title, tone: .warning)
                    }

                    Text(blocker.detail)
                        .font(.paBody)
                        .foregroundStyle(Color.paTextSecondary)

                    HStack(spacing: 6) {
                        Button("Fix Next") {
                            onFixNext()
                        }
                        .buttonStyle(.borderedProminent)
                        .tint(.paInfo)

                        Button("Open Get Started") {
                            onOpenGetStarted()
                        }
                        .buttonStyle(.bordered)
                        .tint(.paNeutral)
                    }
                    .controlSize(.small)
                }
                .frame(maxWidth: .infinity, alignment: .leading)
            }
        }
    }
}
