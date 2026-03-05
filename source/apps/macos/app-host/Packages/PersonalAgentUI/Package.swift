// swift-tools-version: 6.0
import PackageDescription

let package = Package(
    name: "PersonalAgentUI",
    platforms: [
        .macOS(.v14)
    ],
    products: [
        .library(
            name: "PersonalAgentUI",
            targets: ["PersonalAgentUI"]
        )
    ],
    targets: [
        .target(
            name: "PersonalAgentUI",
            exclude: [
                "api/README.md",
                "panels/README.md",
                "shared/README.md",
                "shell/README.md",
                "stores/README.md"
            ]
        ),
        .testTarget(
            name: "PersonalAgentUITests",
            dependencies: ["PersonalAgentUI"],
            exclude: [
                "api/README.md",
                "panels/README.md",
                "shell/README.md",
                "stores/README.md"
            ]
        )
    ]
)
