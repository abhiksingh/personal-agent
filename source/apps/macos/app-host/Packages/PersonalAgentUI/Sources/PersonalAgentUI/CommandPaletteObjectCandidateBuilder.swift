import Foundation

enum CommandPaletteObjectCandidateBuilder {
    static func build(
        taskRunItems: [TaskRunListRowItem],
        approvalInboxItems: [ApprovalInboxItem],
        communicationThreads: [CommunicationThreadItem],
        logicalConnectorCards: [LogicalConnectorCardItem],
        modelCatalogItems: [ModelCatalogEntryItem],
        principalDisplayName: (String?) -> String?,
        providerDisplayName: (String) -> String
    ) -> [CommandPaletteObjectSearchCandidate] {
        var candidates: [CommandPaletteObjectSearchCandidate] = []

        for item in taskRunItems {
            let searchSeed = nonEmpty(item.runID) ?? item.taskID
            let result = CommandPaletteObjectSearchItem(
                id: "task:\(item.id)",
                kind: .taskRun,
                title: item.title,
                subtitle: "Task \(searchSeed) • \(item.effectiveState.label)",
                target: .taskRun(rowID: item.id)
            )
            let searchableValues = [
                item.title,
                item.taskID,
                item.runID,
                item.taskState,
                item.runState,
                item.priorityLabel,
                item.route.taskClass,
                item.route.provider,
                item.route.modelKey,
                principalDisplayName(item.requestedByActorID),
                principalDisplayName(item.subjectPrincipalActorID),
                principalDisplayName(item.actingAsActorID)
            ].compactMap { $0 }
            candidates.append(
                CommandPaletteObjectSearchCandidate(
                    item: result,
                    searchableValues: searchableValues
                )
            )
        }

        for item in approvalInboxItems {
            let result = CommandPaletteObjectSearchItem(
                id: "approval:\(item.id)",
                kind: .approval,
                title: item.taskTitle,
                subtitle: "\(item.decisionState.label) • \(item.stepName)",
                target: .approval(approvalID: item.id)
            )
            let searchableValues = [
                item.id,
                item.taskTitle,
                item.stepName,
                item.taskID,
                item.runID,
                item.stepID,
                item.riskRationale,
                item.route.taskClass,
                item.route.provider,
                item.route.modelKey,
                principalDisplayName(item.requestedByActorID),
                principalDisplayName(item.subjectPrincipalActorID),
                principalDisplayName(item.actingAsActorID)
            ].compactMap { $0 }
            candidates.append(
                CommandPaletteObjectSearchCandidate(
                    item: result,
                    searchableValues: searchableValues
                )
            )
        }

        for item in communicationThreads {
            let result = CommandPaletteObjectSearchItem(
                id: "thread:\(item.id)",
                kind: .thread,
                title: item.title,
                subtitle: "\(item.channel) • \(item.id)",
                target: .thread(threadID: item.id)
            )
            var searchableValues: [String] = [
                item.id,
                item.title,
                item.channel
            ]
            if let connectorID = item.connectorID {
                searchableValues.append(connectorID)
            }
            if let externalRef = item.externalRef {
                searchableValues.append(externalRef)
            }
            if let lastEventID = item.lastEventID {
                searchableValues.append(lastEventID)
            }
            if let lastEventType = item.lastEventType {
                searchableValues.append(lastEventType)
            }
            if let lastDirection = item.lastDirection {
                searchableValues.append(lastDirection)
            }
            if let lastBodyPreview = item.lastBodyPreview {
                searchableValues.append(lastBodyPreview)
            }
            searchableValues.append(contentsOf: item.participantAddresses)
            candidates.append(
                CommandPaletteObjectSearchCandidate(
                    item: result,
                    searchableValues: searchableValues
                )
            )
        }

        for item in logicalConnectorCards {
            let detailValues = item.details.flatMap { [String($0.key), $0.value] }
            let result = CommandPaletteObjectSearchItem(
                id: "connector:\(item.id)",
                kind: .connector,
                title: item.title,
                subtitle: item.summary,
                target: .connector(connectorID: item.id)
            )
            let searchableValues = [
                item.id,
                item.title,
                item.summary,
                item.statusReason,
                item.permissionScope
            ].compactMap { $0 } + detailValues
            candidates.append(
                CommandPaletteObjectSearchCandidate(
                    item: result,
                    searchableValues: searchableValues
                )
            )
        }

        for item in modelCatalogItems {
            let status = item.enabled ? "Enabled" : "Disabled"
            let readiness = item.providerReady ? "Provider Ready" : "Provider Setup Required"
            let result = CommandPaletteObjectSearchItem(
                id: "model:\(item.provider)::\(item.modelKey)",
                kind: .model,
                title: "\(providerDisplayName(item.provider)) • \(item.modelKey)",
                subtitle: "\(status) • \(readiness)",
                target: .model(providerID: item.provider, modelKey: item.modelKey)
            )
            let searchableValues = [
                item.id,
                item.provider,
                providerDisplayName(item.provider),
                item.modelKey,
                item.providerEndpoint,
                status,
                readiness
            ]
            candidates.append(
                CommandPaletteObjectSearchCandidate(
                    item: result,
                    searchableValues: searchableValues
                )
            )
        }

        return candidates
    }

    private static func nonEmpty(_ value: String?) -> String? {
        guard let trimmed = value?.trimmingCharacters(in: .whitespacesAndNewlines),
              !trimmed.isEmpty else {
            return nil
        }
        return trimmed
    }
}
