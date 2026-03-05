// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "PersonalAgentUIV2",
    platforms: [
        .macOS(.v14)
    ],
    products: [
        .library(
            name: "PersonalAgentUIV2",
            targets: ["PersonalAgentUIV2"]
        )
    ],
    targets: [
        .target(
            name: "PersonalAgentUIV2"
        ),
        .testTarget(
            name: "PersonalAgentUIV2Tests",
            dependencies: ["PersonalAgentUIV2"]
        )
    ]
)
