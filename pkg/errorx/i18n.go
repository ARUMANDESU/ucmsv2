package errorx

import (
	"errors"
	"fmt"
	"maps"
	"net/http"

	"github.com/nicksnyder/go-i18n/v2/i18n"
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

func IsCode(err error, code Code) bool {
	if err == nil {
		return false
	}

	var i18nErr *I18nError
	if errors.As(err, &i18nErr) {
		return i18nErr.Code == code || i18nErr.HTTPCode == HTTPStatusCode(code)
	}

	return false
}

func IsNotFound(err error) bool {
	return IsCode(err, CodeNotFound)
}

func IsConflict(err error) bool {
	return IsCode(err, CodeConflict)
}

func IsDuplicateEntry(err error) bool {
	return IsCode(err, CodeDuplicateEntry)
}

// Client Errors (4xx)
func NewInvalidRequest() *I18nError {
	return &I18nError{
		MessageKey: "invalid",
		Code:       CodeInvalid,
		HTTPCode:   http.StatusBadRequest,
	}
}

func NewValidationFailed() *I18nError {
	return &I18nError{
		MessageKey: "validation_failed",
		Code:       CodeValidationFailed,
		HTTPCode:   http.StatusBadRequest,
	}
}

func NewValidationFieldFailed(field string) *I18nError {
	return &I18nError{
		MessageKey:  "validation_failed_field",
		MessageArgs: map[string]any{"Field": field},
		Code:        CodeValidationFailed,
		HTTPCode:    http.StatusBadRequest,
	}
}

func NewMalformedJSON() *I18nError {
	return &I18nError{
		MessageKey: "malformed_json",
		Code:       CodeMalformedJSON,
		HTTPCode:   http.StatusBadRequest,
	}
}

func NewUnauthorized() *I18nError {
	return &I18nError{
		MessageKey: "unauthorized",
		Code:       CodeUnauthorized,
		HTTPCode:   http.StatusUnauthorized,
	}
}

func NewInvalidCredentials() *I18nError {
	return &I18nError{
		MessageKey: "invalid_credentials",
		Code:       CodeInvalidCredentials,
		HTTPCode:   http.StatusUnauthorized,
	}
}

func NewTokenExpired() *I18nError {
	return &I18nError{
		MessageKey: "token_expired",
		Code:       CodeTokenExpired,
		HTTPCode:   http.StatusUnauthorized,
	}
}

func NewForbidden() *I18nError {
	return &I18nError{
		MessageKey: "forbidden",
		Code:       CodeForbidden,
		HTTPCode:   http.StatusForbidden,
	}
}

func NewAccessDenied() *I18nError {
	return &I18nError{
		MessageKey: "access_denied",
		Code:       CodeAccessDenied,
		HTTPCode:   http.StatusForbidden,
	}
}

func NewNotFound() *I18nError {
	return &I18nError{
		MessageKey: "not_found",
		Code:       CodeNotFound,
		HTTPCode:   http.StatusNotFound,
	}
}

func NewResourceNotFound(resourceType string) *I18nError {
	return &I18nError{
		MessageKey:  "not_found_with_type",
		MessageArgs: map[string]any{"ResourceType": resourceType},
		Code:        CodeNotFound,
		HTTPCode:    http.StatusNotFound,
	}
}

func NewMethodNotAllowed() *I18nError {
	return &I18nError{
		MessageKey: "method_not_allowed",
		Code:       CodeMethodNotAllowed,
		HTTPCode:   http.StatusMethodNotAllowed,
	}
}

func NewConflict() *I18nError {
	return &I18nError{
		MessageKey: "conflict",
		Code:       CodeConflict,
		HTTPCode:   http.StatusConflict,
	}
}

func NewDuplicateEntry() *I18nError {
	return &I18nError{
		MessageKey: "duplicate_entry",
		Code:       CodeDuplicateEntry,
		HTTPCode:   http.StatusConflict,
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
		HTTPCode: http.StatusConflict,
	}
}

func NewRateLimitExceeded() *I18nError {
	return &I18nError{
		MessageKey: "rate_limit_exceeded",
		Code:       CodeRateLimitExceeded,
		HTTPCode:   http.StatusTooManyRequests,
	}
}

func NewRateLimitExceededWithRetry(retryAfter int) *I18nError {
	return &I18nError{
		MessageKey:  "rate_limit_exceeded_with_time",
		MessageArgs: map[string]any{"RetryAfter": retryAfter},
		Code:        CodeRateLimitExceeded,
		HTTPCode:    http.StatusTooManyRequests,
	}
}

// Idempotency Errors
func NewIdempotencyKeyMissing() *I18nError {
	return &I18nError{
		MessageKey: "idempotency_key_missing",
		Code:       CodeIdempotencyKeyMissing,
		HTTPCode:   http.StatusBadRequest,
	}
}

func NewIdempotencyKeyMismatch() *I18nError {
	return &I18nError{
		MessageKey: "idempotency_key_payload_mismatch",
		Code:       CodeIdempotencyKeyMismatch,
		HTTPCode:   http.StatusUnprocessableEntity,
	}
}

func NewIdempotencyKeyInProgress() *I18nError {
	return &I18nError{
		MessageKey: "idempotency_key_in_progress",
		Code:       CodeIdempotencyKeyInProgress,
		HTTPCode:   http.StatusConflict,
	}
}

// Password Validation Errors
func NewPasswordTooWeak() *I18nError {
	return &I18nError{
		MessageKey: "password_too_weak",
		Code:       CodePasswordTooWeak,
		HTTPCode:   http.StatusBadRequest,
	}
}

func NewPasswordFormatInvalid() *I18nError {
	return &I18nError{
		MessageKey: "password_format_invalid",
		Code:       CodePasswordFormatInvalid,
		HTTPCode:   http.StatusBadRequest,
	}
}

// Business Logic Errors
func NewAlreadyProcessed() *I18nError {
	return &I18nError{
		MessageKey: "already_processed",
		Code:       CodeAlreadyProcessed,
		HTTPCode:   http.StatusConflict,
	}
}

func NewBusinessRuleViolation() *I18nError {
	return &I18nError{
		MessageKey: "business_rule_violation",
		Code:       CodeBusinessRuleViolation,
		HTTPCode:   http.StatusUnprocessableEntity,
	}
}

func NewInsufficientPermissions() *I18nError {
	return &I18nError{
		MessageKey: "insufficient_permissions",
		Code:       CodeInsufficientPermissions,
		HTTPCode:   http.StatusForbidden,
	}
}

// Server Errors (5xx)
func NewInternalError() *I18nError {
	return &I18nError{
		MessageKey: "internal_error",
		Code:       CodeInternal,
		HTTPCode:   http.StatusInternalServerError,
	}
}

func NewServiceUnavailable() *I18nError {
	return &I18nError{
		MessageKey: "service_unavailable",
		Code:       CodeServiceUnavailable,
		HTTPCode:   http.StatusServiceUnavailable,
	}
}

func NewUpstreamServiceError() *I18nError {
	return &I18nError{
		MessageKey: "upstream_service_error",
		Code:       CodeUpstreamError,
		HTTPCode:   http.StatusBadGateway,
	}
}

func NewUpstreamTimeout() *I18nError {
	return &I18nError{
		MessageKey: "upstream_timeout",
		Code:       CodeUpstreamTimeout,
		HTTPCode:   http.StatusGatewayTimeout,
	}
}

func NewMaintenanceMode() *I18nError {
	return &I18nError{
		MessageKey: "maintenance_mode",
		Code:       CodeMaintenanceMode,
		HTTPCode:   http.StatusServiceUnavailable,
	}
}

// DB
func NewNoRowsAffected() *I18nError {
	return &I18nError{
		MessageKey: "not_found",
		Code:       CodeNotFound,
		HTTPCode:   http.StatusNotFound,
	}
}
