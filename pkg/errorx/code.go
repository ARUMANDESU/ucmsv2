package errorx

import (
	"errors"
	"net/http"
)

type Code string

func (c Code) String() string {
	return string(c)
}

const (
	// Client errors (4xx)
	CodeInvalid            Code = "INVALID"
	CodeValidationFailed   Code = "VALIDATION_FAILED"
	CodeMalformedJSON      Code = "MALFORMED_JSON"
	CodeUnauthorized       Code = "UNAUTHORIZED"
	CodeInvalidCredentials Code = "INVALID_CREDENTIALS"
	CodeTokenExpired       Code = "TOKEN_EXPIRED"
	CodeForbidden          Code = "FORBIDDEN"
	CodeNotFound           Code = "NOT_FOUND"
	CodeConflict           Code = "CONFLICT"
	CodeDuplicateEntry     Code = "DUPLICATE_ENTRY"
	CodeRateLimitExceeded  Code = "RATE_LIMIT_EXCEEDED"

	// Idempotency codes
	CodeIdempotencyKeyMissing    Code = "IDEMPOTENCY_KEY_MISSING"
	CodeIdempotencyKeyMismatch   Code = "IDEMPOTENCY_KEY_PAYLOAD_MISMATCH"
	CodeIdempotencyKeyInProgress Code = "IDEMPOTENCY_KEY_IN_PROGRESS"

	// Business logic
	CodeAlreadyProcessed        Code = "ALREADY_PROCESSED"
	CodeBusinessRuleViolation   Code = "BUSINESS_RULE_VIOLATION"
	CodeInsufficientPermissions Code = "INSUFFICIENT_PERMISSIONS"

	// Server errors (5xx)
	CodeInternal           Code = "INTERNAL_ERROR"
	CodeServiceUnavailable Code = "SERVICE_UNAVAILABLE"
)

func HTTPStatusCode(code Code) int {
	switch code {
	case CodeInvalid, CodeValidationFailed, CodeMalformedJSON, CodeIdempotencyKeyMissing:
		return http.StatusBadRequest
	case CodeUnauthorized, CodeInvalidCredentials, CodeTokenExpired:
		return http.StatusUnauthorized
	case CodeForbidden, CodeInsufficientPermissions:
		return http.StatusForbidden
	case CodeNotFound:
		return http.StatusNotFound
	case CodeConflict, CodeAlreadyProcessed, CodeIdempotencyKeyInProgress:
		return http.StatusConflict
	case CodeDuplicateEntry:
		return http.StatusConflict
	case CodeBusinessRuleViolation, CodeIdempotencyKeyMismatch:
		return http.StatusUnprocessableEntity
	case CodeRateLimitExceeded:
		return http.StatusTooManyRequests
	case CodeServiceUnavailable:
		return http.StatusServiceUnavailable
	case CodeInternal:
		return http.StatusInternalServerError
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
		return i18nErr.Code == code
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
