import PersonalAgentUIV2
import SwiftUI

@main
struct PersonalAgentV2HostApp: App {
    var body: some Scene {
        Window("Personal Agent", id: "main") {
            AppShellV2View()
        }
        .defaultSize(width: 1320, height: 860)
    }
}
