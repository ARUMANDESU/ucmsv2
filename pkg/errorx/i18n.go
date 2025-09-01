package errorx

import (
	"errors"
	"fmt"
	"maps"
	"net/http"
	"strings"

	"github.com/nicksnyder/go-i18n/v2/i18n"

	"github.com/ARUMANDESU/ucms/pkg/i18nx"
)

type I18nErrors []*I18nError

func (errs I18nErrors) Error() string {
	messages := make([]string, len(errs))
	for i, err := range errs {
		messages[i] = err.Error()
	}
	return strings.Join(messages, "; ")
}

func (errs I18nErrors) Unwrap() []error {
	if len(errs) == 0 {
		return nil
	}
	var unwrappedErrs []error
	for _, err := range errs {
		unwrappedErrs = append(unwrappedErrs, err)
	}

	return unwrappedErrs
}

func (errs I18nErrors) Is(target error) bool {
	if target == nil {
		return len(errs) == 0
	}
	for _, err := range errs {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}

func (errs I18nErrors) Localize(localizer *i18n.Localizer) string {
	message := strings.Builder{}
	for _, err := range errs {
		if message.Len() > 0 {
			message.WriteString("; ")
		}
		message.WriteString(err.Localize(localizer))
	}

	return message.String()
}

func (errs I18nErrors) Code() Code {
	if len(errs) == 0 {
		return CodeInternal
	}

	return errs[0].Code
}

func (errs I18nErrors) HTTPStatusCode() int {
	if len(errs) == 0 {
		return http.StatusOK
	}

	var maxHTTPCode int
	for _, err := range errs {
		code := err.HTTPStatusCode()
		if code > maxHTTPCode {
			maxHTTPCode = code
		}
	}

	return maxHTTPCode
}

type I18nError struct {
	cause              error
	MessageKey         string
	MessageArgs        map[string]any
	MessagePluralCount any
	HTTPCode           int
	Code               Code
	Details            string
}

func (e *I18nError) Error() string {
	if e.cause == nil {
		return fmt.Sprintf("[%s] %s", e.Code, e.MessageKey)
	}

	return fmt.Sprintf("[%s] %s: %s", e.Code, e.MessageKey, e.cause)
}

func (e *I18nError) Unwrap() error {
	return e.cause
}

func (e *I18nError) Is(target error) bool {
	if target == nil {
		return e == nil
	}
	if targetErr, ok := target.(*I18nError); ok {
		return e != nil && e.Code == targetErr.Code
	}
	return false
}

func (e *I18nError) Localize(localizer *i18n.Localizer) string {
	for key, value := range e.MessageArgs {
		if !strings.HasPrefix(key, i18nx.ArgLocalePrefix) {
			continue
		}
		if str, ok := value.(string); ok {
			e.MessageArgs[key] = localizer.MustLocalize(&i18n.LocalizeConfig{
				MessageID: str,
			})
		}
	}
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

func (e *I18nError) WithKey(key string) *I18nError {
	e.MessageKey = key
	return e
}

func (e *I18nError) WithDetails(details string) *I18nError {
	e.Details = details
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

// Client Errors (4xx)
func NewInvalidRequest() *I18nError {
	return &I18nError{
		MessageKey: i18nx.KeyInvalid,
		Code:       CodeInvalid,
		HTTPCode:   http.StatusBadRequest,
	}
}

func NewValidationFieldFailed(field string) *I18nError {
	return &I18nError{
		MessageKey:  i18nx.KeyValidationFailedField,
		MessageArgs: map[string]any{i18nx.ArgField: field},
		Code:        CodeValidationFailed,
		HTTPCode:    http.StatusBadRequest,
	}
}

func NewMalformedJSON() *I18nError {
	return &I18nError{
		MessageKey: i18nx.KeyInvalid,
		Code:       CodeMalformedJSON,
		HTTPCode:   http.StatusBadRequest,
	}
}

func NewUnauthorized() *I18nError {
	return &I18nError{
		MessageKey: i18nx.KeyUnauthorized,
		Code:       CodeUnauthorized,
		HTTPCode:   http.StatusUnauthorized,
	}
}

func NewInvalidCredentials() *I18nError {
	return &I18nError{
		MessageKey: i18nx.KeyInvalidCredentials,
		Code:       CodeInvalidCredentials,
		HTTPCode:   http.StatusUnauthorized,
	}
}

func NewTokenExpired() *I18nError {
	return &I18nError{
		MessageKey: i18nx.KeyTokenExpired,
		Code:       CodeTokenExpired,
		HTTPCode:   http.StatusUnauthorized,
	}
}

func NewForbidden() *I18nError {
	return &I18nError{
		MessageKey: i18nx.KeyForbidden,
		Code:       CodeForbidden,
		HTTPCode:   http.StatusForbidden,
	}
}

func NewNotFound() *I18nError {
	return &I18nError{
		MessageKey: i18nx.KeyNotFound,
		Code:       CodeNotFound,
		HTTPCode:   http.StatusNotFound,
	}
}

func NewResourceNotFound(resourceType string) *I18nError {
	return &I18nError{
		MessageKey:  i18nx.KeyNotFoundWithType,
		MessageArgs: map[string]any{i18nx.ArgLocaleResourceType: resourceType},
		Code:        CodeNotFound,
		HTTPCode:    http.StatusNotFound,
	}
}

func NewConflict() *I18nError {
	return &I18nError{
		MessageKey: i18nx.KeyConflict,
		Code:       CodeConflict,
		HTTPCode:   http.StatusConflict,
	}
}

func NewDuplicateEntry() *I18nError {
	return &I18nError{
		MessageKey: i18nx.KeyDuplicateEntry,
		Code:       CodeDuplicateEntry,
		HTTPCode:   http.StatusConflict,
	}
}

func NewDuplicateEntryWithField(resourceType, field string) *I18nError {
	return &I18nError{
		MessageKey: i18nx.KeyDuplicateEntryWithField,
		MessageArgs: map[string]any{
			i18nx.ArgResourceType: resourceType,
			i18nx.ArgField:        field,
		},
		Code:     CodeDuplicateEntry,
		HTTPCode: http.StatusConflict,
	}
}

func NewRateLimitExceeded() *I18nError {
	return &I18nError{
		MessageKey: i18nx.KeyRateLimitExceeded,
		Code:       CodeRateLimitExceeded,
		HTTPCode:   http.StatusTooManyRequests,
	}
}

func NewRateLimitExceededWithRetry(retryAfter int) *I18nError {
	return &I18nError{
		MessageKey:  i18nx.KeyRateLimitExceededWithTime,
		MessageArgs: map[string]any{i18nx.ArgRetryAfter: retryAfter},
		Code:        CodeRateLimitExceeded,
		HTTPCode:    http.StatusTooManyRequests,
	}
}

// Idempotency Errors
func NewIdempotencyKeyMissing() *I18nError {
	return &I18nError{
		MessageKey: i18nx.KeyIdempotencyKeyMissing,
		Code:       CodeIdempotencyKeyMissing,
		HTTPCode:   http.StatusBadRequest,
	}
}

func NewIdempotencyKeyMismatch() *I18nError {
	return &I18nError{
		MessageKey: i18nx.KeyIdempotencyKeyMismatch,
		Code:       CodeIdempotencyKeyMismatch,
		HTTPCode:   http.StatusUnprocessableEntity,
	}
}

func NewIdempotencyKeyInProgress() *I18nError {
	return &I18nError{
		MessageKey: i18nx.KeyIdempotencyKeyInProgress,
		Code:       CodeIdempotencyKeyInProgress,
		HTTPCode:   http.StatusConflict,
	}
}

// Business Logic Errors
func NewAlreadyProcessed() *I18nError {
	return &I18nError{
		MessageKey: i18nx.KeyAlreadyProcessed,
		Code:       CodeAlreadyProcessed,
		HTTPCode:   http.StatusConflict,
	}
}

func NewBusinessRuleViolation() *I18nError {
	return &I18nError{
		MessageKey: i18nx.KeyBusinessRuleViolation,
		Code:       CodeBusinessRuleViolation,
		HTTPCode:   http.StatusUnprocessableEntity,
	}
}

func NewInsufficientPermissions() *I18nError {
	return &I18nError{
		MessageKey: i18nx.KeyInsufficientPermissions,
		Code:       CodeInsufficientPermissions,
		HTTPCode:   http.StatusForbidden,
	}
}

// Server Errors (5xx)
func NewInternalError() *I18nError {
	return &I18nError{
		MessageKey: i18nx.KeyInternalError,
		Code:       CodeInternal,
		HTTPCode:   http.StatusInternalServerError,
	}
}

func NewServiceUnavailable() *I18nError {
	return &I18nError{
		MessageKey: i18nx.KeyServiceUnavailable,
		Code:       CodeServiceUnavailable,
		HTTPCode:   http.StatusServiceUnavailable,
	}
}
