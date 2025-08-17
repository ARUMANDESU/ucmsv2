package registration

import (
	"net/http"

	"github.com/ARUMANDESU/ucms/pkg/errorx"
)

var (
	ErrInvalidEmail                       = errorx.NewValidationFieldFailed("email")
	ErrEmailAlreadyExists                 = errorx.NewDuplicateEntryWithField("registration", "email")
	ErrEmailExceedsMaxLength              = errorx.NewInvalidRequest().WithKey("email_max_len")
	ErrEmptyEmail                         = errorx.NewInvalidRequest().WithKey("empty_email")
	ErrEmailParseFailed                   = errorx.NewValidationFieldFailed("email")
	ErrInvalidEmailFormat                 = errorx.NewInvalidRequest().WithKey("invalid_email_format")
	ErrInvalidVerificationCode            = errorx.NewValidationFieldFailed("verification_code").WithHTTPCode(http.StatusUnprocessableEntity)
	ErrCodeExpired                        = errorx.NewValidationFieldFailed("verification_code")
	ErrInvalidStatus                      = errorx.NewValidationFieldFailed("status").WithHTTPCode(http.StatusUnprocessableEntity)
	ErrRegistrationCompleted              = errorx.NewAlreadyProcessed()
	ErrWaitUntilResend                    = errorx.NewRateLimitExceeded()
	ErrPersistentTooManyAttempts          = errorx.NewPersistable(errorx.NewRateLimitExceeded())
	ErrPersistentVerificationCodeMismatch = errorx.NewPersistable(
		errorx.NewValidationFieldFailed("verification_code").WithHTTPCode(http.StatusUnprocessableEntity),
	)
)
