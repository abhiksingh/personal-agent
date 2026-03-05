# PersonalAgent macOS App Host

This workspace contains the primary SwiftUI macOS app host for PersonalAgent.

## Structure

- `AppHost/`: app target sources and resources
- `Packages/PersonalAgentUI/`: reusable UI package
- `project.yml`: XcodeGen project definition
- `PersonalAgent.xcodeproj`: generated project committed for deterministic local builds

## Generate the Project

```bash
cd source/apps/macos/app-host
xcodegen generate
```

## Build

```bash
cd source/apps/macos/app-host
xcodebuild \
  -project PersonalAgent.xcodeproj \
  -scheme PersonalAgent \
  -configuration Debug \
  -derivedDataPath ../../../../out/build/xcode-derived-data \
  CODE_SIGNING_ALLOWED=NO \
  build
```

## Package Tests

```bash
swift test \
  --package-path source/apps/macos/app-host/Packages/PersonalAgentUI \
  --scratch-path out/build/swiftpm/personal-agent-ui
```

## Responsibilities

- menu-bar and app-shell experience
- chat, approvals, tasks, inspect, channels, connectors, and configuration surfaces
- daemon-facing app transport integration
