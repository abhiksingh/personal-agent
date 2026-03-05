package transport

import "context"

type DaemonLifecycleService interface {
	DaemonLifecycleStatus(ctx context.Context) (DaemonLifecycleStatusResponse, error)
	DaemonLifecycleControl(ctx context.Context, request DaemonLifecycleControlRequest) (DaemonLifecycleControlResponse, error)
	DaemonPluginLifecycleHistory(ctx context.Context, request DaemonPluginLifecycleHistoryRequest) (DaemonPluginLifecycleHistoryResponse, error)
}
