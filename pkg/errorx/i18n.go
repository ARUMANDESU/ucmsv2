package errorx

import (
	"fmt"
	"maps"
	"net/http"

	"github.com/nicksnyder/go-i18n/v2/i18n"
)

var (
	ErrNotFound         = NewNotFound()
	ErrInvalidInput     = NewInvalidRequest()
	ErrInternal         = NewInternalError()
	ErrConflict         = NewConflict()
	ErrUnauthorized     = NewUnauthorized()
	ErrForbidden        = NewForbidden()
	ErrAlreadyProcessed = NewAlreadyProcessed()
)

type I18nError struct {
	cause              error
	MessageKey         string
	MessageArgs        map[string]any
	MessagePluralCount any
	HTTPCode           int
	Code               Code
}

func (e *I18nError) Error() string {
	if e.cause == nil {
		return fmt.Sprintf("[%s] %s", e.Code, e.MessageKey)
	}

	return fmt.Sprintf("[%s] %s: %s", e.Code, e.MessageKey, e.cause)
}

func (e *I18nError) Localize(localizer *i18n.Localizer) string {
	return localizer.MustLocalize(&i18n.LocalizeConfig{
		MessageID:    e.MessageKey,
		TemplateData: e.MessageArgs,
		PluralCount:  e.MessagePluralCount,
	})
}

func (e *I18nError) HTTPStatusCode() int {
	if e.HTTPCode != 0 {
		return e.HTTPCode
	}

	return HTTPStatusCode(e.Code)
}

func (e *I18nError) WithHTTPCode(code int) *I18nError {
	e.HTTPCode = code
	return e
}

func (e *I18nError) WithArgs(args map[string]any) *I18nError {
	if e.MessageArgs == nil {
		e.MessageArgs = make(map[string]any)
	}

	maps.Copy(e.MessageArgs, args)

	return e
}

func (e *I18nError) WithCause(cause error) *I18nError {
	e.cause = cause
	return e
}

func New(messageKey string) *I18nError {
	return &I18nError{
		MessageKey:  messageKey,
		MessageArgs: make(map[string]any),
		HTTPCode:    500,
		Code:        CodeInternal,
	}
}

func HTTPStatusCode(code Code) int {
	switch code {
	case CodeInternal:
		return http.StatusInternalServerError
	case CodeNotFound:
		return http.StatusNotFound
	case CodeInvalid:
		return http.StatusBadRequest
	case CodeConflict:
		return http.StatusConflict
	case CodeUnauthorized:
		return http.StatusUnauthorized
	case CodeForbidden:
		return http.StatusForbidden
	case CodeAlreadyProcessed:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

// Success constructors (though these might not be errors)
func NewSuccess() *I18nError {
	return &I18nError{
		MessageKey: "success",
		Code:       CodeSuccess,
		HTTPCode:   200,
	}
}

func NewResourceCreated() *I18nError {
	return &I18nError{
		MessageKey: "resource_created",
		Code:       CodeCreated,
		HTTPCode:   201,
	}
}

func NewResourceDeleted() *I18nError {
	return &I18nError{
		MessageKey: "resource_deleted",
		Code:       CodeDeleted,
		HTTPCode:   200,
	}
}

// Client Errors (4xx)
func NewInvalidRequest() *I18nError {
	return &I18nError{
		MessageKey: "invalid",
		Code:       CodeInvalid,
		HTTPCode:   400,
	}
}

func NewValidationFailed() *I18nError {
	return &I18nError{
		MessageKey: "validation_failed",
		Code:       CodeValidationFailed,
		HTTPCode:   400,
	}
}

func NewValidationFieldFailed(field string) *I18nError {
	return &I18nError{
		MessageKey:  "validation_failed_field",
		MessageArgs: map[string]any{"Field": field},
		Code:        CodeValidationFailed,
		HTTPCode:    400,
	}
}

func NewMalformedJSON() *I18nError {
	return &I18nError{
		MessageKey: "malformed_json",
		Code:       CodeMalformedJSON,
		HTTPCode:   400,
	}
}

func NewUnauthorized() *I18nError {
	return &I18nError{
		MessageKey: "unauthorized",
		Code:       CodeUnauthorized,
		HTTPCode:   401,
	}
}

func NewInvalidCredentials() *I18nError {
	return &I18nError{
		MessageKey: "invalid_credentials",
		Code:       CodeInvalidCredentials,
		HTTPCode:   401,
	}
}

func NewTokenExpired() *I18nError {
	return &I18nError{
		MessageKey: "token_expired",
		Code:       CodeTokenExpired,
		HTTPCode:   401,
	}
}

func NewForbidden() *I18nError {
	return &I18nError{
		MessageKey: "forbidden",
		Code:       CodeForbidden,
		HTTPCode:   403,
	}
}

func NewAccessDenied() *I18nError {
	return &I18nError{
		MessageKey: "access_denied",
		Code:       CodeAccessDenied,
		HTTPCode:   403,
	}
}

func NewNotFound() *I18nError {
	return &I18nError{
		MessageKey: "not_found",
		Code:       CodeNotFound,
		HTTPCode:   404,
	}
}

func NewResourceNotFound(resourceType string) *I18nError {
	return &I18nError{
		MessageKey:  "not_found_with_type",
		MessageArgs: map[string]any{"ResourceType": resourceType},
		Code:        CodeNotFound,
		HTTPCode:    404,
	}
}

func NewMethodNotAllowed() *I18nError {
	return &I18nError{
		MessageKey: "method_not_allowed",
		Code:       CodeMethodNotAllowed,
		HTTPCode:   405,
	}
}

func NewConflict() *I18nError {
	return &I18nError{
		MessageKey: "conflict",
		Code:       CodeConflict,
		HTTPCode:   409,
	}
}

func NewDuplicateEntry() *I18nError {
	return &I18nError{
		MessageKey: "duplicate_entry",
		Code:       CodeDuplicateEntry,
		HTTPCode:   409,
	}
}

func NewDuplicateEntryWithField(resourceType, field string) *I18nError {
	return &I18nError{
		MessageKey: "duplicate_entry_with_field",
		MessageArgs: map[string]any{
			"ResourceType": resourceType,
			"Field":        field,
		},
		Code:     CodeDuplicateEntry,
		HTTPCode: 409,
	}
}

func NewRateLimitExceeded() *I18nError {
	return &I18nError{
		MessageKey: "rate_limit_exceeded",
		Code:       CodeRateLimitExceeded,
		HTTPCode:   429,
	}
}

func NewRateLimitExceededWithRetry(retryAfter int) *I18nError {
	return &I18nError{
		MessageKey:  "rate_limit_exceeded_with_time",
		MessageArgs: map[string]any{"RetryAfter": retryAfter},
		Code:        CodeRateLimitExceeded,
		HTTPCode:    429,
	}
}

// Idempotency Errors
func NewIdempotencyKeyMissing() *I18nError {
	return &I18nError{
		MessageKey: "idempotency_key_missing",
		Code:       CodeIdempotencyKeyMissing,
		HTTPCode:   400,
	}
}

func NewIdempotencyKeyMismatch() *I18nError {
	return &I18nError{
		MessageKey: "idempotency_key_payload_mismatch",
		Code:       CodeIdempotencyKeyMismatch,
		HTTPCode:   422,
	}
}

func NewIdempotencyKeyInProgress() *I18nError {
	return &I18nError{
		MessageKey: "idempotency_key_in_progress",
		Code:       CodeIdempotencyKeyInProgress,
		HTTPCode:   409,
	}
}

// Password Validation Errors
func NewPasswordTooWeak() *I18nError {
	return &I18nError{
		MessageKey: "password_too_weak",
		Code:       CodePasswordTooWeak,
		HTTPCode:   400,
	}
}

func NewPasswordFormatInvalid() *I18nError {
	return &I18nError{
		MessageKey: "password_format_invalid",
		Code:       CodePasswordFormatInvalid,
		HTTPCode:   400,
	}
}

// Business Logic Errors
func NewAlreadyProcessed() *I18nError {
	return &I18nError{
		MessageKey: "already_processed",
		Code:       CodeAlreadyProcessed,
		HTTPCode:   409,
	}
}

func NewBusinessRuleViolation() *I18nError {
	return &I18nError{
		MessageKey: "business_rule_violation",
		Code:       CodeBusinessRuleViolation,
		HTTPCode:   422,
	}
}

func NewInsufficientPermissions() *I18nError {
	return &I18nError{
		MessageKey: "insufficient_permissions",
		Code:       CodeInsufficientPermissions,
		HTTPCode:   403,
	}
}

// Server Errors (5xx)
func NewInternalError() *I18nError {
	return &I18nError{
		MessageKey: "internal_error",
		Code:       CodeInternal,
		HTTPCode:   500,
	}
}

func NewServiceUnavailable() *I18nError {
	return &I18nError{
		MessageKey: "service_unavailable",
		Code:       CodeServiceUnavailable,
		HTTPCode:   503,
	}
}

func NewUpstreamServiceError() *I18nError {
	return &I18nError{
		MessageKey: "upstream_service_error",
		Code:       CodeUpstreamError,
		HTTPCode:   502,
	}
}

func NewUpstreamTimeout() *I18nError {
	return &I18nError{
		MessageKey: "upstream_timeout",
		Code:       CodeUpstreamTimeout,
		HTTPCode:   504,
	}
}

func NewMaintenanceMode() *I18nError {
	return &I18nError{
		MessageKey: "maintenance_mode",
		Code:       CodeMaintenanceMode,
		HTTPCode:   503,
	}
}
