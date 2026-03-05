package transport

import "context"

type IdentityDirectoryService interface {
	ListWorkspaces(ctx context.Context, request IdentityWorkspacesRequest) (IdentityWorkspacesResponse, error)
	ListPrincipals(ctx context.Context, request IdentityPrincipalsRequest) (IdentityPrincipalsResponse, error)
	GetActiveContext(ctx context.Context, request IdentityActiveContextRequest) (IdentityActiveContextResponse, error)
	SelectWorkspace(ctx context.Context, request IdentityWorkspaceSelectRequest) (IdentityActiveContextResponse, error)
	Bootstrap(ctx context.Context, request IdentityBootstrapRequest) (IdentityBootstrapResponse, error)
	ListDevices(ctx context.Context, request IdentityDeviceListRequest) (IdentityDeviceListResponse, error)
	ListSessions(ctx context.Context, request IdentitySessionListRequest) (IdentitySessionListResponse, error)
	RevokeSession(ctx context.Context, request IdentitySessionRevokeRequest) (IdentitySessionRevokeResponse, error)
}
