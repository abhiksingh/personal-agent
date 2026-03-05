package daemonruntime

import (
	"database/sql"
	"fmt"
	"sync"

	"personalagent/runtime/internal/transport"
)

const (
	identitySourceDefault  = "default"
	identitySourceDerived  = "derived"
	identitySourceRequest  = "request"
	identitySourceSelected = "selected"

	defaultIdentityDirectoryListLimit = 50
	maxIdentityDirectoryListLimit     = 200

	identitySessionHealthActive  = "active"
	identitySessionHealthExpired = "expired"
	identitySessionHealthRevoked = "revoked"

	identityBootstrapDefaultStatus    = "ACTIVE"
	identityBootstrapDefaultActorType = "human"
	identityBootstrapAuditEventType   = "identity_bootstrap_upsert"

	identityMutationSourceDaemon = "daemon"
	identityMutationSourceCLI    = "cli"
	identityMutationSourceApp    = "app"

	identityMutationReasonExplicitSelect = "explicit_select_workspace"
	identityMutationReasonRequest        = "request_override"
	identityMutationReasonDerived        = "derived_resolution"
	identityMutationReasonDefault        = "default_resolution"
)

var reservedSystemWorkspaceIDs = map[string]struct{}{
	daemonPluginAuditWorkspaceID: {},
}

type IdentityDirectoryService struct {
	db *sql.DB

	mu       sync.RWMutex
	selected identitySelectionState
}

type identitySelectionState struct {
	WorkspaceID      string
	PrincipalActorID string
	WorkspaceSource  string
	PrincipalSource  string
	UpdatedAt        string
	MutationSource   string
	MutationReason   string
	Version          int64
}

type identityWorkspaceRow struct {
	WorkspaceID    string
	Name           string
	Status         string
	PrincipalCount int
	ActorCount     int
	HandleCount    int
	UpdatedAt      string
}

type identityPrincipalRow struct {
	ActorID         string
	DisplayName     string
	ActorType       string
	ActorStatus     string
	PrincipalStatus string
}

type identityHandleRow struct {
	ActorID     string
	Channel     string
	HandleValue string
	IsPrimary   bool
	UpdatedAt   string
}

type identityDeviceRow struct {
	DeviceID               string
	WorkspaceID            string
	UserID                 string
	DeviceType             string
	Platform               string
	Label                  string
	LastSeenAt             string
	CreatedAt              string
	SessionTotal           int
	SessionActiveCount     int
	SessionExpiredCount    int
	SessionRevokedCount    int
	SessionLatestStartedAt string
}

type identitySessionRow struct {
	SessionID        string
	WorkspaceID      string
	DeviceID         string
	UserID           string
	DeviceType       string
	Platform         string
	DeviceLabel      string
	DeviceLastSeenAt string
	StartedAt        string
	ExpiresAt        string
	RevokedAt        string
}

var _ transport.IdentityDirectoryService = (*IdentityDirectoryService)(nil)

func NewIdentityDirectoryService(container *ServiceContainer) (*IdentityDirectoryService, error) {
	if container == nil || container.DB == nil {
		return nil, fmt.Errorf("database is not configured")
	}
	return &IdentityDirectoryService{db: container.DB}, nil
}
