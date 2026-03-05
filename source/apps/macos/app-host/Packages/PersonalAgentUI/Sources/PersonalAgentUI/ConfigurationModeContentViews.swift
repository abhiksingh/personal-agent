import SwiftUI

struct ConfigurationSetupModeContent<Setup: View, Runtime: View, Token: View>: View {
    private let setup: Setup
    private let token: Token
    private let runtime: Runtime

    init(
        @ViewBuilder setup: () -> Setup,
        @ViewBuilder token: () -> Token,
        @ViewBuilder runtime: () -> Runtime
    ) {
        self.setup = setup()
        self.token = token()
        self.runtime = runtime()
    }

    var body: some View {
        setup
        token
        runtime
    }
}

struct ConfigurationWorkspaceModeContent<Identity: View, Persona: View, Devices: View, Delegation: View>: View {
    private let identity: Identity
    private let persona: Persona
    private let delegation: Delegation
    private let devices: Devices

    init(
        @ViewBuilder identity: () -> Identity,
        @ViewBuilder persona: () -> Persona,
        @ViewBuilder delegation: () -> Delegation,
        @ViewBuilder devices: () -> Devices
    ) {
        self.identity = identity()
        self.persona = persona()
        self.delegation = delegation()
        self.devices = devices()
    }

    var body: some View {
        identity
        persona
        delegation
        devices
    }
}

struct ConfigurationIntegrationsModeContent<Capability: View, Trust: View>: View {
    private let capability: Capability
    private let trust: Trust

    init(
        @ViewBuilder capability: () -> Capability,
        @ViewBuilder trust: () -> Trust
    ) {
        self.capability = capability()
        self.trust = trust()
    }

    var body: some View {
        capability
        trust
    }
}

struct ConfigurationDataModeContent<Retention: View, Context: View, Memory: View, Retrieval: View>: View {
    private let retention: Retention
    private let context: Context
    private let memory: Memory
    private let retrieval: Retrieval

    init(
        @ViewBuilder retention: () -> Retention,
        @ViewBuilder context: () -> Context,
        @ViewBuilder memory: () -> Memory,
        @ViewBuilder retrieval: () -> Retrieval
    ) {
        self.retention = retention()
        self.context = context()
        self.memory = memory()
        self.retrieval = retrieval()
    }

    var body: some View {
        retention
        context
        memory
        retrieval
    }
}

struct ConfigurationAdvancedModeContent<Timeline: View, Performance: View, Lifecycle: View>: View {
    private let lifecycle: Lifecycle
    private let timeline: Timeline
    private let performance: Performance

    init(
        @ViewBuilder lifecycle: () -> Lifecycle,
        @ViewBuilder timeline: () -> Timeline,
        @ViewBuilder performance: () -> Performance
    ) {
        self.lifecycle = lifecycle()
        self.timeline = timeline()
        self.performance = performance()
    }

    var body: some View {
        lifecycle
        timeline
        performance
    }
}
