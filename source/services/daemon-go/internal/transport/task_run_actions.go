package transport

import "strings"

// ResolveTaskRunActionAvailability determines deterministic task-run action flags
// from lifecycle state metadata for list/detail control surfaces.
func ResolveTaskRunActionAvailability(taskState string, runState string) TaskRunActionAvailability {
	effective := normalizeTaskRunActionState(runState)
	if effective == "" {
		effective = normalizeTaskRunActionState(taskState)
	}

	switch effective {
	case "queued", "planning", "awaiting_approval", "blocked":
		return TaskRunActionAvailability{
			CanCancel:  true,
			CanRetry:   false,
			CanRequeue: true,
		}
	case "running":
		return TaskRunActionAvailability{
			CanCancel:  true,
			CanRetry:   false,
			CanRequeue: false,
		}
	case "failed", "cancelled":
		return TaskRunActionAvailability{
			CanCancel:  false,
			CanRetry:   true,
			CanRequeue: false,
		}
	default:
		return TaskRunActionAvailability{}
	}
}

func normalizeTaskRunActionState(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}
