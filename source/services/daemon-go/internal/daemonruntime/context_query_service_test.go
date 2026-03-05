package daemonruntime

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"personalagent/runtime/internal/transport"
)

func TestContextQueryMemoryInventoryAndCandidatesSupportsFiltersAndPaging(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedContextQueryFixtures(t, container.DB)

	service, err := NewAutomationInspectRetentionContextService(container)
	if err != nil {
		t.Fatalf("new automation inspect service: %v", err)
	}

	page1, err := service.QueryContextMemoryInventory(context.Background(), transport.ContextMemoryInventoryRequest{
		WorkspaceID: "ws1",
		SourceType:  "comm_event",
		Limit:       1,
	})
	if err != nil {
		t.Fatalf("query context memory inventory page 1: %v", err)
	}
	if len(page1.Items) != 1 || page1.Items[0].MemoryID != "mem-2" || !page1.HasMore {
		t.Fatalf("unexpected memory inventory page 1 payload: %+v", page1)
	}
	if page1.Items[0].TokenEstimate <= 0 || len(page1.Items[0].Sources) == 0 {
		t.Fatalf("expected parsed token estimate and sources, got %+v", page1.Items[0])
	}

	page2, err := service.QueryContextMemoryInventory(context.Background(), transport.ContextMemoryInventoryRequest{
		WorkspaceID:     "ws1",
		SourceType:      "comm_event",
		CursorUpdatedAt: page1.NextCursorUpdatedAt,
		CursorID:        page1.NextCursorID,
		Limit:           1,
	})
	if err != nil {
		t.Fatalf("query context memory inventory page 2: %v", err)
	}
	if len(page2.Items) != 1 || page2.Items[0].MemoryID != "mem-1" {
		t.Fatalf("unexpected memory inventory page 2 payload: %+v", page2)
	}

	filtered, err := service.QueryContextMemoryInventory(context.Background(), transport.ContextMemoryInventoryRequest{
		WorkspaceID:    "ws1",
		OwnerActorID:   "actor.a",
		Status:         "ACTIVE",
		SourceRefQuery: "event://1",
		Limit:          10,
	})
	if err != nil {
		t.Fatalf("query filtered context memory inventory: %v", err)
	}
	if len(filtered.Items) != 1 || filtered.Items[0].MemoryID != "mem-1" {
		t.Fatalf("unexpected filtered memory inventory payload: %+v", filtered)
	}

	candidatePage1, err := service.QueryContextMemoryCandidates(context.Background(), transport.ContextMemoryCandidatesRequest{
		WorkspaceID:  "ws1",
		OwnerActorID: "actor.a",
		Status:       "pending",
		Limit:        1,
	})
	if err != nil {
		t.Fatalf("query context memory candidates page 1: %v", err)
	}
	if len(candidatePage1.Items) != 1 || candidatePage1.Items[0].CandidateID != "cand-3" || !candidatePage1.HasMore {
		t.Fatalf("unexpected memory candidates page 1 payload: %+v", candidatePage1)
	}
	if candidatePage1.Items[0].CandidateKind != "summary" || candidatePage1.Items[0].TokenEstimate != 18 {
		t.Fatalf("expected parsed candidate metadata, got %+v", candidatePage1.Items[0])
	}
	if len(candidatePage1.Items[0].SourceIDs) != 2 || len(candidatePage1.Items[0].SourceRefs) != 2 {
		t.Fatalf("expected source ids/refs parsing, got %+v", candidatePage1.Items[0])
	}

	candidatePage2, err := service.QueryContextMemoryCandidates(context.Background(), transport.ContextMemoryCandidatesRequest{
		WorkspaceID:     "ws1",
		OwnerActorID:    "actor.a",
		Status:          "PENDING",
		CursorCreatedAt: candidatePage1.NextCursorCreatedAt,
		CursorID:        candidatePage1.NextCursorID,
		Limit:           1,
	})
	if err != nil {
		t.Fatalf("query context memory candidates page 2: %v", err)
	}
	if len(candidatePage2.Items) != 1 || candidatePage2.Items[0].CandidateID != "cand-1" {
		t.Fatalf("unexpected memory candidates page 2 payload: %+v", candidatePage2)
	}
}

func TestContextQueryRetrievalDocumentsAndChunksSupportsFiltersAndPaging(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedContextQueryFixtures(t, container.DB)

	service, err := NewAutomationInspectRetentionContextService(container)
	if err != nil {
		t.Fatalf("new automation inspect service: %v", err)
	}

	docPage1, err := service.QueryContextRetrievalDocuments(context.Background(), transport.ContextRetrievalDocumentsRequest{
		WorkspaceID:    "ws1",
		OwnerActorID:   "actor.a",
		SourceURIQuery: "memory://",
		Limit:          1,
	})
	if err != nil {
		t.Fatalf("query retrieval documents page 1: %v", err)
	}
	if len(docPage1.Items) != 1 || docPage1.Items[0].DocumentID != "doc-2" || !docPage1.HasMore {
		t.Fatalf("unexpected retrieval documents page 1 payload: %+v", docPage1)
	}
	if docPage1.Items[0].ChunkCount <= 0 {
		t.Fatalf("expected chunk count metadata on retrieval documents page 1: %+v", docPage1.Items[0])
	}

	docPage2, err := service.QueryContextRetrievalDocuments(context.Background(), transport.ContextRetrievalDocumentsRequest{
		WorkspaceID:     "ws1",
		OwnerActorID:    "actor.a",
		SourceURIQuery:  "memory://",
		CursorCreatedAt: docPage1.NextCursorCreatedAt,
		CursorID:        docPage1.NextCursorID,
		Limit:           1,
	})
	if err != nil {
		t.Fatalf("query retrieval documents page 2: %v", err)
	}
	if len(docPage2.Items) != 1 || docPage2.Items[0].DocumentID != "doc-1" {
		t.Fatalf("unexpected retrieval documents page 2 payload: %+v", docPage2)
	}

	chunkPage1, err := service.QueryContextRetrievalChunks(context.Background(), transport.ContextRetrievalChunksRequest{
		WorkspaceID:    "ws1",
		OwnerActorID:   "actor.a",
		ChunkTextQuery: "note",
		Limit:          1,
	})
	if err != nil {
		t.Fatalf("query retrieval chunks page 1: %v", err)
	}
	if len(chunkPage1.Items) != 1 || chunkPage1.Items[0].ChunkID != "chunk-2" || !chunkPage1.HasMore {
		t.Fatalf("unexpected retrieval chunks page 1 payload: %+v", chunkPage1)
	}

	chunkPage2, err := service.QueryContextRetrievalChunks(context.Background(), transport.ContextRetrievalChunksRequest{
		WorkspaceID:     "ws1",
		OwnerActorID:    "actor.a",
		ChunkTextQuery:  "note",
		CursorCreatedAt: chunkPage1.NextCursorCreatedAt,
		CursorID:        chunkPage1.NextCursorID,
		Limit:           1,
	})
	if err != nil {
		t.Fatalf("query retrieval chunks page 2: %v", err)
	}
	if len(chunkPage2.Items) != 1 || chunkPage2.Items[0].ChunkID != "chunk-1" {
		t.Fatalf("unexpected retrieval chunks page 2 payload: %+v", chunkPage2)
	}

	documentScoped, err := service.QueryContextRetrievalChunks(context.Background(), transport.ContextRetrievalChunksRequest{
		WorkspaceID: "ws1",
		DocumentID:  "doc-1",
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("query retrieval chunks with document filter: %v", err)
	}
	if len(documentScoped.Items) != 1 || documentScoped.Items[0].DocumentID != "doc-1" {
		t.Fatalf("unexpected retrieval chunks document-filter payload: %+v", documentScoped)
	}
}

func TestContextQueryValidationErrors(t *testing.T) {
	container := newLifecycleTestContainer(t, nil)
	seedContextQueryFixtures(t, container.DB)

	service, err := NewAutomationInspectRetentionContextService(container)
	if err != nil {
		t.Fatalf("new automation inspect service: %v", err)
	}

	_, err = service.QueryContextMemoryInventory(context.Background(), transport.ContextMemoryInventoryRequest{
		WorkspaceID: "ws1",
		CursorID:    "mem-2",
		Limit:       10,
	})
	if err == nil || !strings.Contains(err.Error(), "cursor_updated_at is required") {
		t.Fatalf("expected cursor_updated_at validation error, got %v", err)
	}

	_, err = service.QueryContextMemoryCandidates(context.Background(), transport.ContextMemoryCandidatesRequest{
		WorkspaceID:     "ws1",
		CursorCreatedAt: "not-a-timestamp",
		Limit:           10,
	})
	if err == nil || !strings.Contains(err.Error(), "cursor_created_at must be RFC3339 timestamp") {
		t.Fatalf("expected cursor_created_at validation error, got %v", err)
	}

	_, err = service.QueryContextRetrievalDocuments(context.Background(), transport.ContextRetrievalDocumentsRequest{
		WorkspaceID: "ws1",
		CursorID:    "doc-2",
		Limit:       10,
	})
	if err == nil || !strings.Contains(err.Error(), "cursor_created_at is required") {
		t.Fatalf("expected retrieval documents cursor validation error, got %v", err)
	}

	_, err = service.QueryContextRetrievalChunks(context.Background(), transport.ContextRetrievalChunksRequest{
		WorkspaceID:     "ws1",
		CursorCreatedAt: "bad-time",
		Limit:           10,
	})
	if err == nil || !strings.Contains(err.Error(), "cursor_created_at must be RFC3339 timestamp") {
		t.Fatalf("expected retrieval chunks cursor validation error, got %v", err)
	}
}

func seedContextQueryFixtures(t *testing.T, db *sql.DB) {
	t.Helper()
	statements := []string{
		`INSERT INTO workspaces(id, name, status, created_at, updated_at) VALUES ('ws1', 'WS 1', 'ACTIVE', '2026-02-25T00:00:00Z', '2026-02-25T00:00:00Z')`,
		`INSERT INTO actors(id, workspace_id, actor_type, display_name, status, created_at, updated_at) VALUES ('actor.a', 'ws1', 'human', 'Actor A', 'ACTIVE', '2026-02-25T00:00:00Z', '2026-02-25T00:00:00Z')`,
		`INSERT INTO actors(id, workspace_id, actor_type, display_name, status, created_at, updated_at) VALUES ('actor.b', 'ws1', 'human', 'Actor B', 'ACTIVE', '2026-02-25T00:00:00Z', '2026-02-25T00:00:00Z')`,
		`INSERT INTO memory_items(id, workspace_id, owner_principal_actor_id, scope_type, key, value_json, status, source_summary, created_at, updated_at)
		 VALUES ('mem-1', 'ws1', 'actor.a', 'conversation', 'mem-key-1', '{"kind":"summary","is_canonical":false,"token_estimate":12,"content":"alpha"}', 'ACTIVE', 'event://1', '2026-02-25T00:00:01Z', '2026-02-25T00:00:01Z')`,
		`INSERT INTO memory_items(id, workspace_id, owner_principal_actor_id, scope_type, key, value_json, status, source_summary, created_at, updated_at)
		 VALUES ('mem-2', 'ws1', 'actor.a', 'conversation', 'mem-key-2', '{"kind":"fact","is_canonical":true,"token_estimate":24,"content":"beta"}', 'DISABLED', 'event://2', '2026-02-25T00:00:02Z', '2026-02-25T00:00:03Z')`,
		`INSERT INTO memory_items(id, workspace_id, owner_principal_actor_id, scope_type, key, value_json, status, source_summary, created_at, updated_at)
		 VALUES ('mem-3', 'ws1', 'actor.b', 'conversation', 'mem-key-3', '{"kind":"summary","is_canonical":false,"token_estimate":18,"content":"gamma"}', 'ACTIVE', 'mail://1', '2026-02-25T00:00:04Z', '2026-02-25T00:00:05Z')`,
		`INSERT INTO memory_sources(id, memory_item_id, source_type, source_ref, created_at) VALUES ('src-1', 'mem-1', 'comm_event', 'event://1', '2026-02-25T00:00:01Z')`,
		`INSERT INTO memory_sources(id, memory_item_id, source_type, source_ref, created_at) VALUES ('src-2', 'mem-2', 'comm_event', 'event://2', '2026-02-25T00:00:03Z')`,
		`INSERT INTO memory_sources(id, memory_item_id, source_type, source_ref, created_at) VALUES ('src-3', 'mem-3', 'mail', 'mail://1', '2026-02-25T00:00:05Z')`,
		`INSERT INTO memory_candidates(id, workspace_id, owner_principal_actor_id, candidate_json, score, status, created_at)
		 VALUES ('cand-1', 'ws1', 'actor.a', '{"kind":"summary","token_estimate":10,"source_ids":["mem-1"],"source_refs":["event://1"]}', 0.91, 'PENDING', '2026-02-25T00:00:01Z')`,
		`INSERT INTO memory_candidates(id, workspace_id, owner_principal_actor_id, candidate_json, score, status, created_at)
		 VALUES ('cand-2', 'ws1', 'actor.b', '{"kind":"summary","token_estimate":12,"source_ids":["mem-3"],"source_refs":["mail://1"]}', 0.67, 'APPLIED', '2026-02-25T00:00:02Z')`,
		`INSERT INTO memory_candidates(id, workspace_id, owner_principal_actor_id, candidate_json, score, status, created_at)
		 VALUES ('cand-3', 'ws1', 'actor.a', '{"kind":"summary","token_estimate":18,"source_ids":["mem-1","mem-2"],"source_refs":["event://1","event://2"]}', 0.99, 'PENDING', '2026-02-25T00:00:03Z')`,
		`INSERT INTO context_documents(id, workspace_id, owner_principal_actor_id, source_uri, checksum, created_at)
		 VALUES ('doc-1', 'ws1', 'actor.a', 'memory://personal/doc-1', 'checksum-1', '2026-02-25T00:00:01Z')`,
		`INSERT INTO context_documents(id, workspace_id, owner_principal_actor_id, source_uri, checksum, created_at)
		 VALUES ('doc-2', 'ws1', 'actor.a', 'memory://project/doc-2', 'checksum-2', '2026-02-25T00:00:02Z')`,
		`INSERT INTO context_documents(id, workspace_id, owner_principal_actor_id, source_uri, checksum, created_at)
		 VALUES ('doc-3', 'ws1', 'actor.b', 'mail://thread/doc-3', 'checksum-3', '2026-02-25T00:00:03Z')`,
		`INSERT INTO context_chunks(id, document_id, chunk_index, text_body, token_count, created_at)
		 VALUES ('chunk-1', 'doc-1', 0, 'personal note chunk', 9, '2026-02-25T00:00:01Z')`,
		`INSERT INTO context_chunks(id, document_id, chunk_index, text_body, token_count, created_at)
		 VALUES ('chunk-2', 'doc-2', 0, 'project note chunk', 10, '2026-02-25T00:00:02Z')`,
		`INSERT INTO context_chunks(id, document_id, chunk_index, text_body, token_count, created_at)
		 VALUES ('chunk-3', 'doc-3', 0, 'mail watcher chunk', 11, '2026-02-25T00:00:03Z')`,
	}

	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("seed context query fixture failed: %v\nstatement: %s", err, statement)
		}
	}
}
