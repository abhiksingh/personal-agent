# PersonalAgent macOS App Host v2

This workspace contains the v2 SwiftUI app host used to explore the assistant trust and audit experience.

## Structure

- `AppHost/`: app entrypoint and resources
- `Packages/PersonalAgentUIV2/`: reusable v2 UI package
- `project.yml`: XcodeGen definition
- `PersonalAgent.xcodeproj`: generated project
- `docs/design-system.md`: visual and interaction system notes

## Generate the Project

```bash
cd source/apps/macos/app-host-v2
xcodegen generate
```

## Build

```bash
cd source/apps/macos/app-host-v2
xcodebuild \
  -project PersonalAgent.xcodeproj \
  -scheme PersonalAgent \
  -configuration Debug \
  -derivedDataPath ../../../../out/build/xcode-derived-data-v2 \
  CODE_SIGNING_ALLOWED=NO \
  build
```

## Package Tests

```bash
swift test \
  --package-path source/apps/macos/app-host-v2/Packages/PersonalAgentUIV2 \
  --scratch-path out/build/swiftpm/personal-agent-ui-v2
```
