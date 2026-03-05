package transport

import "net/http"

const (
	taskControlErrorCategoryValidation        = "task_control_validation"
	taskControlErrorCategoryLookup            = "task_control_lookup"
	taskControlErrorCategoryReferenceMismatch = "task_control_reference_mismatch"
	taskControlErrorCategoryStateConflict     = "task_control_state_conflict"
	taskControlErrorCategoryAuthorization     = "task_control_authorization"
	taskControlErrorCategoryBackend           = "task_control_backend"
)

func NewTaskControlMissingReferenceError(message string) error {
	return NewTransportDomainError(http.StatusBadRequest, "missing_required_field", message, map[string]any{
		"category": taskControlErrorCategoryValidation,
	})
}

func NewTaskControlNotFoundError(message string) error {
	return NewTransportDomainError(http.StatusNotFound, "resource_not_found", message, map[string]any{
		"category": taskControlErrorCategoryLookup,
	})
}

func NewTaskControlReferenceMismatchError(message string) error {
	return NewTransportDomainError(http.StatusBadRequest, "invalid_request", message, map[string]any{
		"category": taskControlErrorCategoryReferenceMismatch,
	})
}

func NewTaskControlStateConflictError(message string) error {
	return NewTransportDomainError(http.StatusConflict, "resource_conflict", message, map[string]any{
		"category": taskControlErrorCategoryStateConflict,
	})
}

func NewTaskControlForbiddenError(message string) error {
	return NewTransportDomainError(http.StatusForbidden, "auth_forbidden", message, map[string]any{
		"category": taskControlErrorCategoryAuthorization,
	})
}

func NewTaskControlBackendError(message string) error {
	return NewTransportDomainError(http.StatusInternalServerError, "internal_error", message, map[string]any{
		"category": taskControlErrorCategoryBackend,
	})
}
