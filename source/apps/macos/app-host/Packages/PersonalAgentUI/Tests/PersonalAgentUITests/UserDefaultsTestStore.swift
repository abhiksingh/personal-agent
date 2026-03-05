import Foundation

private let uiDefaultsSuiteEnvKey = "PA_UI_DEFAULTS_SUITE"

func appShellStateTestUserDefaults() -> UserDefaults {
    let suiteName = ProcessInfo.processInfo.environment[uiDefaultsSuiteEnvKey]?
        .trimmingCharacters(in: .whitespacesAndNewlines)
    if let suiteName, !suiteName.isEmpty, let suiteDefaults = UserDefaults(suiteName: suiteName) {
        return suiteDefaults
    }
    return .standard
}
