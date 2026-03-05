# PersonalAgent

PersonalAgent is a macOS-first autonomous assistant that executes end-to-end workflows across app, message, and voice channels with deterministic auditability.

## Status

This project is active and pre-release. The codebase is substantial, but interfaces and workflows are still evolving. Expect change, and treat `main` as the only supported branch.

## What Is Here

- A Go daemon runtime for orchestration, transport, policy, persistence, and connector execution.
- A Go CLI for local control-plane and workflow operations.
- SwiftUI macOS app hosts for the primary app shell and a v2 design track.
- Shared contracts, specs, validation runners, and packaging/operations documentation.

## Repository Layout

- `source/services/daemon-go`: daemon runtime module
- `source/clients/cli-go`: CLI module
- `source/apps/macos/app-host`: primary macOS app host
- `source/apps/macos/app-host-v2`: v2 macOS app host/design track
- `source/packages/contracts`: shared transport contracts
- `docs/spec`: product and data-model specs
- `tools/scripts`: harness, test runners, packaging, and local dev scripts

## Prerequisites

- macOS for the app-host workflows
- Go `1.24`
- Xcode and Swift toolchain for macOS app work
- `xcodegen` for generating the committed Xcode project from `project.yml`

## Quickstart

Run the repository checks:

```bash
tools/scripts/check_harness.sh
```

Run the full automated suite:

```bash
tools/scripts/run_tests_all.sh
```

Build the primary macOS app host:

```bash
cd source/apps/macos/app-host
xcodegen generate
xcodebuild \
  -project PersonalAgent.xcodeproj \
  -scheme PersonalAgent \
  -configuration Debug \
  -derivedDataPath ../../../../out/build/xcode-derived-data \
  CODE_SIGNING_ALLOWED=NO \
  build
```

## Core Docs

- Product spec: [`docs/spec/spec.md`](./docs/spec/spec.md)
- Data model: [`docs/spec/data-model.md`](./docs/spec/data-model.md)
- Bootstrap overview: [`docs/spec/bootstrap.md`](./docs/spec/bootstrap.md)
- Runtime architecture notes: [`docs/harness/ARCHITECTURE.md`](./docs/harness/ARCHITECTURE.md)
- Security baseline: [`docs/harness/SECURITY.md`](./docs/harness/SECURITY.md)
- Packaging guide: [`docs/ops/macos-daemon-packaging.md`](./docs/ops/macos-daemon-packaging.md)

## Contributing

Start with [CONTRIBUTING.md](./CONTRIBUTING.md). Use [SUPPORT.md](./SUPPORT.md) for general questions and [SECURITY.md](./SECURITY.md) for private vulnerability reporting.

## License

Licensed under the [Apache License 2.0](./LICENSE).
