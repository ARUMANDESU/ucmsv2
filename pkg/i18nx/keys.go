package i18nx

// Error message keys
const (
	// Client errors
	KeyInvalid                   = "invalid"
	KeyMalformedJSON             = "malformed_json"
	KeyValidationFailed          = "validation_failed"
	KeyValidationFailedField     = "validation_failed_field"
	KeyUnauthorized              = "unauthorized"
	KeyInvalidCredentials        = "invalid_credentials"
	KeyTokenExpired              = "token_expired"
	KeyForbidden                 = "forbidden"
	KeyAccessDenied              = "access_denied"
	KeyNotFound                  = "not_found"
	KeyNotFoundWithType          = "not_found_with_type"
	KeyNotFoundOrDeleted         = "not_found_or_deleted"
	KeyMethodNotAllowed          = "method_not_allowed"
	KeyConflict                  = "conflict"
	KeyDuplicateEntry            = "duplicate_entry"
	KeyDuplicateEntryWithField   = "duplicate_entry_with_field"
	KeyRateLimitExceeded         = "rate_limit_exceeded"
	KeyRateLimitExceededWithTime = "rate_limit_exceeded_with_time"

	// Idempotency errors
	KeyIdempotencyKeyMissing    = "idempotency_key_missing"
	KeyIdempotencyKeyMismatch   = "idempotency_key_payload_mismatch"
	KeyIdempotencyKeyInProgress = "idempotency_key_in_progress"

	// Password validation
	KeyPasswordTooWeak       = "password_too_weak"
	KeyPasswordFormatInvalid = "password_format_invalid"

	// Business logic errors
	KeyAlreadyProcessed        = "already_processed"
	KeyBusinessRuleViolation   = "business_rule_violation"
	KeyInsufficientPermissions = "insufficient_permissions"

	// Server errors
	KeyInternalError        = "internal_error"
	KeyServiceUnavailable   = "service_unavailable"
	KeyUpstreamServiceError = "upstream_service_error"
	KeyUpstreamTimeout      = "upstream_timeout"
	KeyMaintenanceMode      = "maintenance_mode"

	// Authentication specific
	KeyWrongEmailBarcodePassword = "wrong_email_or_barcode_or_password"
	KeyWrongEmailBarcodeFormat   = "wrong_email_or_barcode_format"
	KeyInvalidRefreshTokenClaims = "invalid_refresh_token_claims"
	KeyInvalidRefreshTokenExp    = "invalid_refresh_token_exp"
	KeyRefreshTokenExpired       = "refresh_token_expired"

	// Registration specific
	KeyEmailMaxLen          = "email_max_len"
	KeyEmptyEmail           = "empty_email"
	KeyInvalidEmailFormat   = "invalid_email_format"
	KeyEmailNotAvailable    = "error_email_not_available"
	KeyBarcodeNotAvailable  = "error_barcode_not_available"
	KeyUsernameNotAvailable = "error_username_not_available"

	// Staff invitation specific
	KeyInvalidInvitation       = "invalid_invitation"
	KeyTimestampInPast         = "timestamp_in_past"
	KeyAtLeastOneEmail         = "at_least_one_email"
	KeyEmailAlreadyExistsField = "email_already_exists_field"
	KeyMaxEmailsExceededField  = "max_emails_exceeded_field"

	// Business errors
	KeyCodeExpired             = "business_error_code_expired"
	KeyVerifyFirst             = "business_error_verify_first"
	KeyInvalidVerificationCode = "business_error_invalid_verification_code"
)

// Validation message keys (project-specific validation errors)
//
//	Custom validation rules
const (
	ValidationIsEmail             = "validation_is_email"
	ValidationIsPassword          = "validation_is_password"
	ValidationIsName              = "validation_is_name"
	ValidationIsUsername          = "validation_is_username"
	ValidationNoDuplicate         = "validation_no_duplicate"
	ValidationTimeInPast          = "validation_time_in_past"
	ValidationTimeBeforeThreshold = "validation_time_before_threshold"
)

// Validation messages (English defaults)
const (
	MsgValidationIsEmailOther             = "must be a valid email address"
	MsgValidationIsPasswordOther          = "must contain at least 8 characters with uppercase, lowercase, number, and special character"
	MsgValidationIsNameOther              = "must contain only letters, spaces, and common name characters"
	MsgValidationIsUsernameOther          = "must be between 3 and 30 characters long, start with a letter, and contain only lowercase letters, digits, periods, and underscores. Cannot contain consecutive periods or underscores, or period followed by underscore or vice versa"
	MsgValidationNoDuplicateOther         = "duplicate values are not allowed"
	MsgValidationTimeInPastOther          = "time cannot be in the past"
	MsgValidationTimeBeforeThresholdOther = "time must be after {{.threshold}}"
)

// Field name keys
const (
	FieldEmailBarcode     = "email_barcode"
	FieldPassword         = "password"
	FieldEmail            = "email"
	FieldVerificationCode = "verification_code"
	FieldBarcode          = "barcode"
	FieldFirstName        = "first_name"
	FieldLastName         = "last_name"
	FieldGroupID          = "group_id"
	FieldGroup            = "group"
	FieldUsername         = "username"
	FieldStatus           = "status"
)

// Template argument keys (snake_case naming)
const (
	ArgLocalePrefix       = "locale_"
	ArgField              = "field"
	ArgResourceType       = "resource_type"
	ArgLocaleResourceType = "locale_resource_type"
	ArgRetryAfter         = "retry_after"
	ArgMaxEmails          = "max_emails"
	ArgThreshold          = "threshold"
)
