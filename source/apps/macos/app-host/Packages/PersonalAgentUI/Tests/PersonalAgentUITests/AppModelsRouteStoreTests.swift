import XCTest
@testable import PersonalAgentUI

@MainActor
final class AppModelsRouteStoreTests: XCTestCase {
    func testProviderDraftHelpersTrackDirtyStateAndReset() {
        let store = AppModelsRouteStore()
        store.providerEndpointSourceByID = ["openai": "https://api.openai.com/v1"]
        store.providerSecretNameSourceByID = ["openai": "OPENAI_API_KEY"]
        store.providerEndpointDraftByID = ["openai": "https://api.openai.com/v1"]
        store.providerAPIKeySecretNameDraftByID = ["openai": "OPENAI_API_KEY"]

        XCTAssertFalse(
            store.providerSetupHasDraftChanges(
                providerID: "openai",
                normalizedProviderID: normalizeProviderID(_:),
                providerDefaultEndpoints: ["openai": "https://api.openai.com/v1"],
                defaultProviderSecretName: defaultProviderSecretName(for:)
            )
        )

        store.setProviderSecretValueDraft(
            "sk-test",
            providerID: "openai",
            normalizedProviderID: normalizeProviderID(_:)
        )
        XCTAssertTrue(
            store.providerSetupHasDraftChanges(
                providerID: "openai",
                normalizedProviderID: normalizeProviderID(_:),
                providerDefaultEndpoints: ["openai": "https://api.openai.com/v1"],
                defaultProviderSecretName: defaultProviderSecretName(for:)
            )
        )

        store.resetProviderSetupDraft(
            providerID: "openai",
            normalizedProviderID: normalizeProviderID(_:),
            providerDefaultEndpoints: ["openai": "https://api.openai.com/v1"],
            defaultProviderSecretName: defaultProviderSecretName(for:)
        )

        XCTAssertEqual(store.providerSecretValueDraft(for: "openai", normalizedProviderID: normalizeProviderID(_:)), "")
        XCTAssertEqual(store.providerEndpointDraft(for: "openai", normalizedProviderID: normalizeProviderID(_:), providerDefaultEndpoints: ["openai": "https://api.openai.com/v1"]), "https://api.openai.com/v1")
    }

    func testSyncProviderSetupDraftsMergesConfiguredAndPrunesStaleState() throws {
        let store = AppModelsRouteStore()
        store.providerSetupStatusByID = ["openai": "Saved", "stale": "remove"]
        store.providerSetupInFlightIDs = ["openai", "stale"]

        let configuredOpenAI = try decode(
            DaemonProviderConfigRecord.self,
            from: """
            {
              "workspace_id": "ws1",
              "provider": "openai",
              "endpoint": "https://proxy.example/v1",
              "api_key_secret_name": "OPENAI_PROXY_KEY",
              "api_key_configured": true,
              "updated_at": "2026-03-04T12:00:00Z"
            }
            """
        )

        store.syncProviderSetupDrafts(
            configuredByProvider: ["openai": configuredOpenAI],
            canonicalProviderOrder: ["openai", "ollama"],
            providerDefaultEndpoints: [
                "openai": "https://api.openai.com/v1",
                "ollama": "http://127.0.0.1:11434",
            ],
            providerRequiresAPIKey: { providerID in
                providerID == "openai"
            },
            defaultProviderSecretName: defaultProviderSecretName(for:)
        )

        XCTAssertEqual(store.providerEndpointDraftByID["openai"], "https://proxy.example/v1")
        XCTAssertEqual(store.providerAPIKeySecretNameDraftByID["openai"], "OPENAI_PROXY_KEY")
        XCTAssertEqual(store.providerEndpointDraftByID["ollama"], "http://127.0.0.1:11434")
        XCTAssertEqual(store.providerAPIKeySecretNameDraftByID["ollama"], "")
        XCTAssertNil(store.providerSetupStatusByID["stale"])
        XCTAssertFalse(store.providerSetupInFlightIDs.contains("stale"))
    }

    func testModelRouteSimulationTaskClassOptionsDeduplicatesPolicyEntries() {
        let store = AppModelsRouteStore()
        store.modelPolicyItems = [
            ModelPolicyItem(
                id: "chat::openai::gpt-4.1",
                taskClass: "Chat",
                provider: "openai",
                modelKey: "gpt-4.1",
                updatedAtLabel: "now"
            ),
            ModelPolicyItem(
                id: "automation::openai::gpt-4.1",
                taskClass: "automation",
                provider: "openai",
                modelKey: "gpt-4.1",
                updatedAtLabel: "now"
            )
        ]

        let options = store.modelRouteSimulationTaskClassOptions(
            contextTaskClassOptions: ["chat", "agent"]
        )

        XCTAssertEqual(options, ["chat", "agent", "automation"])
    }

    func testSyncDiscoveredModelCatalogFlagsAlignsWithCatalogState() {
        let store = AppModelsRouteStore()
        store.modelCatalogItems = [
            ModelCatalogEntryItem(
                id: "openai::gpt-4.1",
                provider: "openai",
                modelKey: "gpt-4.1",
                enabled: true,
                providerReady: true,
                providerEndpoint: "https://api.openai.com/v1"
            )
        ]
        store.discoveredModelsByProviderID = [
            "openai": [
                DiscoveredModelEntryItem(
                    id: "openai::gpt-4.1",
                    provider: "openai",
                    modelKey: "gpt-4.1",
                    displayName: "GPT-4.1",
                    source: "provider_discovery",
                    inCatalog: false,
                    enabled: false
                ),
                DiscoveredModelEntryItem(
                    id: "openai::gpt-oss:20b",
                    provider: "openai",
                    modelKey: "gpt-oss:20b",
                    displayName: "gpt-oss:20b",
                    source: "provider_discovery",
                    inCatalog: true,
                    enabled: true
                )
            ]
        ]

        store.syncDiscoveredModelCatalogFlags(
            normalizedProviderID: normalizeProviderID(_:),
            modelCatalogIdentifier: { providerID, modelKey in
                "\(normalizeProviderID(providerID))::\(modelKey)"
            }
        )

        let discovered = store.discoveredModelsByProviderID["openai"] ?? []
        XCTAssertEqual(discovered.count, 2)
        XCTAssertEqual(discovered.first?.inCatalog, true)
        XCTAssertEqual(discovered.first?.enabled, true)
        XCTAssertEqual(discovered.last?.inCatalog, false)
        XCTAssertEqual(discovered.last?.enabled, false)
    }

    func testMapModelRouteResponsesNormalizesTraceOutput() throws {
        let store = AppModelsRouteStore()
        let simulation = try decode(
            DaemonModelRouteSimulationResponse.self,
            from: """
            {
              "workspace_id": "ws1",
              "task_class": "chat",
              "principal_actor_id": "actor.default",
              "selected_provider": "openai",
              "selected_model_key": "gpt-4.1",
              "selected_source": "workspace_policy",
              "notes": "routing note",
              "reason_codes": ["provider_ready"],
              "decisions": [
                {
                  "step": "candidate",
                  "decision": "selected",
                  "reason_code": "provider_ready",
                  "provider": "openai",
                  "model_key": "gpt-4.1",
                  "note": "best latency"
                }
              ],
              "fallback_chain": [
                {
                  "rank": 2,
                  "provider": "openai",
                  "model_key": "gpt-4o-mini",
                  "selected": false,
                  "reason_code": "fallback"
                },
                {
                  "rank": 1,
                  "provider": "openai",
                  "model_key": "gpt-4.1",
                  "selected": true,
                  "reason_code": "primary"
                }
              ]
            }
            """
        )

        let mappedSimulation = store.mapModelRouteSimulationResponse(simulation, workspaceID: "ws-fallback")
        XCTAssertEqual(mappedSimulation.workspaceID, "ws1")
        XCTAssertEqual(mappedSimulation.selectedProvider, "openai")
        XCTAssertEqual(mappedSimulation.fallbackChain.map(\.rank), [1, 2])
        XCTAssertEqual(
            store.modelRouteSimulationSummaryMessage(mappedSimulation, providerDisplayName: { _ in "OpenAI" }),
            "Simulated chat route for actor.default: OpenAI • gpt-4.1."
        )

        let explain = try decode(
            DaemonModelRouteExplainResponse.self,
            from: """
            {
              "workspace_id": "ws1",
              "task_class": "chat",
              "principal_actor_id": "actor.default",
              "selected_provider": "openai",
              "selected_model_key": "gpt-4.1",
              "selected_source": "workspace_policy",
              "summary": "Matched workspace route",
              "explanations": ["provider configured"],
              "reason_codes": ["provider_ready"],
              "decisions": [],
              "fallback_chain": []
            }
            """
        )

        let mappedExplain = store.mapModelRouteExplainResponse(explain, workspaceID: "ws-fallback")
        XCTAssertEqual(mappedExplain.summary, "Matched workspace route")
        XCTAssertEqual(
            store.modelRouteExplainSummaryMessage(mappedExplain),
            "Loaded explainability trace for chat (actor.default)."
        )
    }

    private func decode<T: Decodable>(_ type: T.Type, from json: String) throws -> T {
        try JSONDecoder().decode(T.self, from: Data(json.utf8))
    }

    private func normalizeProviderID(_ raw: String) -> String {
        raw.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
    }

    private func defaultProviderSecretName(for providerID: String) -> String {
        switch normalizeProviderID(providerID) {
        case "openai":
            return "OPENAI_API_KEY"
        case "anthropic":
            return "ANTHROPIC_API_KEY"
        case "google":
            return "GOOGLE_API_KEY"
        default:
            return "\(providerID.uppercased())_API_KEY"
        }
    }
}
