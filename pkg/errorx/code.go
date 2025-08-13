package errorx

type Code string

func (c Code) String() string {
	return string(c)
}

const (
	// Success codes
	CodeSuccess Code = "SUCCESS"
	CodeCreated Code = "RESOURCE_CREATED"
	CodeDeleted Code = "RESOURCE_DELETED"

	// Client errors (4xx)
	CodeInvalid            Code = "INVALID"
	CodeValidationFailed   Code = "VALIDATION_FAILED"
	CodeMalformedJSON      Code = "MALFORMED_JSON"
	CodeUnauthorized       Code = "UNAUTHORIZED"
	CodeInvalidCredentials Code = "INVALID_CREDENTIALS"
	CodeTokenExpired       Code = "TOKEN_EXPIRED"
	CodeForbidden          Code = "FORBIDDEN"
	CodeAccessDenied       Code = "ACCESS_DENIED"
	CodeNotFound           Code = "NOT_FOUND"
	CodeMethodNotAllowed   Code = "METHOD_NOT_ALLOWED"
	CodeConflict           Code = "CONFLICT"
	CodeDuplicateEntry     Code = "DUPLICATE_ENTRY"
	CodeRateLimitExceeded  Code = "RATE_LIMIT_EXCEEDED"

	// Idempotency codes
	CodeIdempotencyKeyMissing    Code = "IDEMPOTENCY_KEY_MISSING"
	CodeIdempotencyKeyMismatch   Code = "IDEMPOTENCY_KEY_PAYLOAD_MISMATCH"
	CodeIdempotencyKeyInProgress Code = "IDEMPOTENCY_KEY_IN_PROGRESS"

	// Password validation
	CodePasswordTooWeak       Code = "PASSWORD_TOO_WEAK"
	CodePasswordFormatInvalid Code = "PASSWORD_FORMAT_INVALID"

	// Business logic
	CodeAlreadyProcessed        Code = "ALREADY_PROCESSED"
	CodeBusinessRuleViolation   Code = "BUSINESS_RULE_VIOLATION"
	CodeInsufficientPermissions Code = "INSUFFICIENT_PERMISSIONS"

	// Server errors (5xx)
	CodeInternal           Code = "INTERNAL_ERROR"
	CodeServiceUnavailable Code = "SERVICE_UNAVAILABLE"
	CodeUpstreamError      Code = "UPSTREAM_SERVICE_ERROR"
	CodeUpstreamTimeout    Code = "UPSTREAM_TIMEOUT"
	CodeMaintenanceMode    Code = "MAINTENANCE_MODE"
)
