import SwiftUI

extension ConfigurationPanelView {
    var dataModeContent: some View {
        ConfigurationDataModeContent {
            retentionSection
        } context: {
            contextSection
        } memory: {
            operatorDisclosure(
                title: "Memory Browser",
                isExpanded: $isMemoryBrowserExpanded
            ) {
                memoryBrowserSection
            }
        } retrieval: {
            operatorDisclosure(
                title: "Retrieval Context Inspector",
                isExpanded: $isRetrievalInspectorExpanded
            ) {
                retrievalContextInspectorSection
            }
        }
    }

    var retentionSection: some View {
        VStack(alignment: .leading, spacing: 10) {
            sectionTitle("Retention Controls")

            Stepper(value: $state.retentionTraceDays, in: 1...365) {
                Text("Trace Days: \(state.retentionTraceDays)")
                    .font(.callout)
            }
            Stepper(value: $state.retentionTranscriptDays, in: 1...365) {
                Text("Transcript Days: \(state.retentionTranscriptDays)")
                    .font(.callout)
            }
            Stepper(value: $state.retentionMemoryDays, in: 1...365) {
                Text("Memory Days: \(state.retentionMemoryDays)")
                    .font(.callout)
            }

            Divider()

            Stepper(value: $state.retentionTokenThreshold, in: 200...20000, step: 100) {
                Text("Compaction Token Threshold: \(state.retentionTokenThreshold)")
                    .font(.callout)
            }
            Stepper(value: $state.retentionStaleAfterHours, in: 24...720, step: 24) {
                Text("Compaction Stale Age (hours): \(state.retentionStaleAfterHours)")
                    .font(.callout)
            }
            Stepper(value: $state.retentionCompactionLimit, in: 10...2000, step: 10) {
                Text("Compaction Scan Limit: \(state.retentionCompactionLimit)")
                    .font(.callout)
            }
            Toggle("Apply memory compaction writes", isOn: $state.retentionCompactionApply)

            HStack(spacing: 8) {
                Button("Run Retention Purge") {
                    state.requestRunRetentionPurge()
                }
                .buttonStyle(.bordered)
                .disabled(state.isRetentionActionInFlight)

                Button(state.retentionCompactionApply ? "Run Memory Compaction" : "Preview Memory Compaction") {
                    state.requestRunRetentionCompactMemory()
                }
                .buttonStyle(.bordered)
                .disabled(state.isRetentionActionInFlight)

                if state.isRetentionActionInFlight {
                    ProgressView()
                        .controlSize(.small)
                }
            }

            if let message = state.retentionStatusMessage {
                Text(message)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .cardSurface(.standard)
    }

    var contextSection: some View {
        VStack(alignment: .leading, spacing: 10) {
            sectionTitle("Context Budget Controls")

            Picker("Task Class", selection: $state.contextTaskClass) {
                ForEach(state.contextTaskClassOptions, id: \.self) { taskClass in
                    Text(taskClass).tag(taskClass)
                }
            }
            .pickerStyle(.menu)

            Stepper(value: $state.contextSamplesLimit, in: 5...200, step: 5) {
                Text("Sample Limit: \(state.contextSamplesLimit)")
                    .font(.callout)
            }

            HStack(spacing: 8) {
                Button("Load Context Samples") {
                    state.loadContextSamples()
                }
                .buttonStyle(.bordered)
                .disabled(state.isContextActionInFlight)

                Button("Tune Context Profile") {
                    state.runContextTune()
                }
                .buttonStyle(.bordered)
                .disabled(state.isContextActionInFlight)

                if state.isContextActionInFlight {
                    ProgressView()
                        .controlSize(.small)
                }
            }

            if let message = state.contextStatusMessage {
                Text(message)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .cardSurface(.standard)
    }

    var memoryBrowserSection: some View {
        VStack(alignment: .leading, spacing: 10) {
            GroupBox("Inventory Filters") {
                VStack(alignment: .leading, spacing: 8) {
                    TextField("Owner Actor ID (optional)", text: $state.contextMemoryOwnerActorFilter)
                        .textFieldStyle(.roundedBorder)

                    TextField("Scope Type (optional)", text: $state.contextMemoryScopeTypeFilter)
                        .textFieldStyle(.roundedBorder)

                    Picker("Status", selection: $state.contextMemoryStatusFilter) {
                        ForEach(state.contextMemoryStatusFilterOptions, id: \.self) { option in
                            Text(contextFilterOptionLabel(option)).tag(option)
                        }
                    }
                    .pickerStyle(.menu)

                    TextField("Source Type (optional)", text: $state.contextMemorySourceTypeFilter)
                        .textFieldStyle(.roundedBorder)

                    TextField("Source Ref Query (optional)", text: $state.contextMemorySourceRefQuery)
                        .textFieldStyle(.roundedBorder)

                    Stepper(
                        "Limit: \(state.contextMemoryLimit)",
                        value: $state.contextMemoryLimit,
                        in: 5...200,
                        step: 5
                    )

                    HStack(spacing: 8) {
                        Button("Refresh Inventory") {
                            state.refreshContextMemoryInventory()
                        }
                        .buttonStyle(.bordered)
                        .disabled(state.isContextMemoryInventoryLoading)

                        Button("Reset Filters") {
                            state.resetContextMemoryInventoryFilters()
                            state.refreshContextMemoryInventory()
                        }
                        .buttonStyle(.bordered)
                        .disabled(state.isContextMemoryInventoryLoading)

                        if state.isContextMemoryInventoryLoading {
                            ProgressView()
                                .controlSize(.small)
                        }
                    }
                }
            }

            configurationSecondaryStatusMessage(state.contextMemoryInventoryStatusMessage)

            configurationInventoryGroup(
                title: "Memory Inventory (\(state.contextMemoryInventoryItems.count))",
                items: state.contextMemoryInventoryItems,
                isLoading: state.isContextMemoryInventoryLoading,
                loadingMessage: "Loading context memory inventory…",
                emptyMessage: "No memory inventory records matched the current filters.",
                hasMore: state.contextMemoryInventoryHasMore,
                hasMoreMessage: "More memory inventory items are available. Narrow filters or reduce noise to inspect quickly."
            ) { item in
                memoryInventoryRow(item)
            }

            GroupBox("Compaction Candidate Filters") {
                VStack(alignment: .leading, spacing: 8) {
                    TextField("Owner Actor ID (optional)", text: $state.contextMemoryCandidatesOwnerActorFilter)
                        .textFieldStyle(.roundedBorder)

                    Picker("Status", selection: $state.contextMemoryCandidatesStatusFilter) {
                        ForEach(state.contextMemoryCandidateStatusFilterOptions, id: \.self) { option in
                            Text(contextFilterOptionLabel(option)).tag(option)
                        }
                    }
                    .pickerStyle(.menu)

                    Stepper(
                        "Limit: \(state.contextMemoryCandidatesLimit)",
                        value: $state.contextMemoryCandidatesLimit,
                        in: 5...200,
                        step: 5
                    )

                    HStack(spacing: 8) {
                        Button("Refresh Candidates") {
                            state.refreshContextMemoryCandidates()
                        }
                        .buttonStyle(.bordered)
                        .disabled(state.isContextMemoryCandidatesLoading)

                        Button("Reset Candidate Filters") {
                            state.resetContextMemoryCandidatesFilters()
                            state.refreshContextMemoryCandidates()
                        }
                        .buttonStyle(.bordered)
                        .disabled(state.isContextMemoryCandidatesLoading)

                        if state.isContextMemoryCandidatesLoading {
                            ProgressView()
                                .controlSize(.small)
                        }
                    }
                }
            }

            configurationSecondaryStatusMessage(state.contextMemoryCandidatesStatusMessage)

            configurationInventoryGroup(
                title: "Compaction Candidates (\(state.contextMemoryCandidateItems.count))",
                items: state.contextMemoryCandidateItems,
                isLoading: state.isContextMemoryCandidatesLoading,
                loadingMessage: "Loading memory compaction candidates…",
                emptyMessage: "No memory compaction candidates matched the current filters.",
                hasMore: state.contextMemoryCandidatesHasMore,
                hasMoreMessage: "More compaction candidates are available. Refine filters to inspect a narrower set."
            ) { item in
                memoryCompactionCandidateRow(item)
            }
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .cardSurface(.standard)
    }

    var retrievalContextInspectorSection: some View {
        VStack(alignment: .leading, spacing: 10) {
            GroupBox("Document Filters") {
                VStack(alignment: .leading, spacing: 8) {
                    TextField("Owner Actor ID (optional)", text: $state.contextRetrievalOwnerActorFilter)
                        .textFieldStyle(.roundedBorder)

                    TextField("Source URI Query (optional)", text: $state.contextRetrievalSourceURIQuery)
                        .textFieldStyle(.roundedBorder)

                    Stepper(
                        "Limit: \(state.contextRetrievalDocumentsLimit)",
                        value: $state.contextRetrievalDocumentsLimit,
                        in: 5...200,
                        step: 5
                    )

                    HStack(spacing: 8) {
                        Button("Refresh Documents") {
                            state.refreshContextRetrievalDocuments()
                        }
                        .buttonStyle(.bordered)
                        .disabled(state.isContextRetrievalDocumentsLoading)

                        Button("Reset Document Filters") {
                            state.resetContextRetrievalDocumentFilters()
                            state.refreshContextRetrievalDocuments()
                        }
                        .buttonStyle(.bordered)
                        .disabled(state.isContextRetrievalDocumentsLoading)

                        if state.isContextRetrievalDocumentsLoading {
                            ProgressView()
                                .controlSize(.small)
                        }
                    }
                }
            }

            configurationSecondaryStatusMessage(state.contextRetrievalDocumentsStatusMessage)

            configurationInventoryGroup(
                title: "Documents (\(state.contextRetrievalDocumentItems.count))",
                items: state.contextRetrievalDocumentItems,
                isLoading: state.isContextRetrievalDocumentsLoading,
                loadingMessage: "Loading retrieval documents…",
                emptyMessage: "No retrieval documents matched the current filters.",
                hasMore: state.contextRetrievalDocumentsHasMore,
                hasMoreMessage: "More retrieval documents are available. Narrow the source query to inspect a smaller set."
            ) { item in
                retrievalDocumentRow(item)
            }

            GroupBox("Chunk Filters") {
                VStack(alignment: .leading, spacing: 8) {
                    Text(state.selectedContextRetrievalDocumentID.isEmpty
                        ? "Selected Document: none"
                        : "Selected Document: \(state.selectedContextRetrievalDocumentID)")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                        .textSelection(.enabled)

                    TextField("Chunk Text Query (optional)", text: $state.contextRetrievalChunkTextQuery)
                        .textFieldStyle(.roundedBorder)

                    Stepper(
                        "Limit: \(state.contextRetrievalChunksLimit)",
                        value: $state.contextRetrievalChunksLimit,
                        in: 5...200,
                        step: 5
                    )

                    HStack(spacing: 8) {
                        Button("Load Chunks") {
                            state.refreshContextRetrievalChunks()
                        }
                        .buttonStyle(.bordered)
                        .disabled(state.selectedContextRetrievalDocumentID.isEmpty || state.isContextRetrievalChunksLoading)

                        if state.isContextRetrievalChunksLoading {
                            ProgressView()
                                .controlSize(.small)
                        }
                    }
                }
            }

            configurationSecondaryStatusMessage(state.contextRetrievalChunksStatusMessage)

            configurationInventoryGroup(
                title: "Chunks (\(state.contextRetrievalChunkItems.count))",
                items: state.contextRetrievalChunkItems,
                isLoading: state.isContextRetrievalChunksLoading,
                loadingMessage: "Loading retrieval chunks…",
                emptyMessage: "No retrieval chunks matched the current document/query filters.",
                hasMore: state.contextRetrievalChunksHasMore,
                hasMoreMessage: "More retrieval chunks are available. Narrow the chunk query for targeted inspection."
            ) { item in
                retrievalChunkRow(item)
            }
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .cardSurface(.standard)
    }

    func memoryInventoryRow(_ item: MemoryInventoryItem) -> some View {
        configurationRecordCard {
            HStack(spacing: 8) {
                Text(item.key)
                    .font(.caption.weight(.semibold))
                    .textSelection(.enabled)
                TahoeStatusBadge(
                    text: contextFilterOptionLabel(item.status),
                    symbolName: item.status.uppercased() == "ACTIVE" ? "checkmark.circle.fill" : "circle",
                    tint: item.status.uppercased() == "ACTIVE" ? .green : .secondary
                )
                TahoeStatusBadge(
                    text: contextFilterOptionLabel(item.scopeType),
                    symbolName: "square.stack.3d.up",
                    tint: .secondary
                )
                if item.isCanonical {
                    TahoeStatusBadge(
                        text: "Canonical",
                        symbolName: "star.circle.fill",
                        tint: .blue
                    )
                }
                Spacer(minLength: 0)
                Text(item.updatedAtLabel)
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }

            Text("Owner \(item.ownerActorID) • Kind \(item.kind) • Tokens \(item.tokenEstimate) • Sources \(item.sourceCount)")
                .font(.caption2)
                .foregroundStyle(.secondary)

            Text(item.sourceSummary)
                .font(.caption2)
                .foregroundStyle(.secondary)

            Text("Value: \(item.valueSummary)")
                .font(.caption2)
                .foregroundStyle(.secondary)
                .textSelection(.enabled)

            if !item.sources.isEmpty {
                ForEach(item.sources) { source in
                    HStack(alignment: .firstTextBaseline, spacing: 6) {
                        Text(source.sourceType)
                            .font(.caption2.weight(.semibold))
                        Text(source.sourceRef)
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                            .textSelection(.enabled)
                        Spacer(minLength: 0)
                        Text(source.createdAtLabel)
                            .font(.caption2)
                            .foregroundStyle(.secondary)
                    }
                }
            }
        }
    }

    func memoryCompactionCandidateRow(_ item: MemoryCompactionCandidateItem) -> some View {
        configurationRecordCard {
            HStack(spacing: 8) {
                Text(item.id)
                    .font(.caption.weight(.semibold))
                    .textSelection(.enabled)
                TahoeStatusBadge(
                    text: contextFilterOptionLabel(item.status),
                    symbolName: item.status.uppercased() == "PENDING" ? "clock.badge.exclamationmark" : "checkmark.circle.fill",
                    tint: item.status.uppercased() == "PENDING" ? .orange : .secondary
                )
                Spacer(minLength: 0)
                Text(item.createdAtLabel)
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }

            Text("Owner \(item.ownerActorID) • Kind \(item.candidateKind) • Score \(String(format: "%.2f", item.score)) • Tokens \(item.tokenEstimate)")
                .font(.caption2)
                .foregroundStyle(.secondary)

            if !item.sourceIDs.isEmpty {
                Text("Source IDs: \(item.sourceIDs.joined(separator: ", "))")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
                    .textSelection(.enabled)
            }
            if !item.sourceRefs.isEmpty {
                Text("Source Refs: \(item.sourceRefs.joined(separator: ", "))")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
                    .textSelection(.enabled)
            }

            Text("Candidate: \(item.candidateSummary)")
                .font(.caption2)
                .foregroundStyle(.secondary)
                .textSelection(.enabled)
        }
    }

    func retrievalDocumentRow(_ item: RetrievalDocumentItem) -> some View {
        let isSelected = state.selectedContextRetrievalDocumentID == item.id
        return configurationRecordCard {
            HStack(spacing: 8) {
                Text(item.id)
                    .font(.caption.weight(.semibold))
                    .textSelection(.enabled)
                if isSelected {
                    TahoeStatusBadge(
                        text: "Selected",
                        symbolName: "checkmark.circle.fill",
                        tint: .green
                    )
                }
                Spacer(minLength: 0)
                Text(item.createdAtLabel)
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }

            Text("Owner \(item.ownerActorID) • Chunks \(item.chunkCount)")
                .font(.caption2)
                .foregroundStyle(.secondary)

            Text("Source URI: \(item.sourceURI)")
                .font(.caption2)
                .foregroundStyle(.secondary)
                .textSelection(.enabled)

            Text("Checksum: \(item.checksum)")
                .font(.caption2)
                .foregroundStyle(.secondary)
                .textSelection(.enabled)

            HStack(spacing: 8) {
                if isSelected {
                    Button("Chunks Loaded") {
                        state.selectContextRetrievalDocument(item.id)
                    }
                    .buttonStyle(.borderedProminent)
                    .controlSize(.small)
                    .disabled(state.isContextRetrievalChunksLoading)
                } else {
                    Button("Inspect Chunks") {
                        state.selectContextRetrievalDocument(item.id)
                    }
                    .buttonStyle(.bordered)
                    .controlSize(.small)
                }
            }
        }
    }

    func retrievalChunkRow(_ item: RetrievalChunkItem) -> some View {
        configurationRecordCard {
            HStack(spacing: 8) {
                Text("\(item.chunkIndex)")
                    .font(.caption.weight(.semibold))
                Text(item.id)
                    .font(.caption2)
                    .foregroundStyle(.secondary)
                    .textSelection(.enabled)
                Spacer(minLength: 0)
                Text(item.createdAtLabel)
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }

            Text("Tokens \(item.tokenCount) • Owner \(item.ownerActorID)")
                .font(.caption2)
                .foregroundStyle(.secondary)

            Text(item.textPreview)
                .font(.caption2)
                .foregroundStyle(.primary)
                .textSelection(.enabled)
        }
    }
}
