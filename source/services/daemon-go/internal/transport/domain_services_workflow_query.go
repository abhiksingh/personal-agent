package transport

import "context"

type WorkflowQueryService interface {
	ListApprovalInbox(ctx context.Context, request ApprovalInboxRequest) (ApprovalInboxResponse, error)
	ListTaskRuns(ctx context.Context, request TaskRunListRequest) (TaskRunListResponse, error)
	ListCommThreads(ctx context.Context, request CommThreadListRequest) (CommThreadListResponse, error)
	ListCommEvents(ctx context.Context, request CommEventTimelineRequest) (CommEventTimelineResponse, error)
	ListCommCallSessions(ctx context.Context, request CommCallSessionListRequest) (CommCallSessionListResponse, error)
}
