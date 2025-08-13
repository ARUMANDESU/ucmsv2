package registration

import (
	"net/http"

	"github.com/ARUMANDESU/ucms/pkg/errorx"
)

var (
	ErrInvalidEmail                       = errorx.NewValidationFieldFailed("email")
	ErrEmailAlreadyExists                 = errorx.NewDuplicateEntryWithField("registration", "email")
	ErrEmailExceedsMaxLength              = errorx.NewValidationFieldFailed("email")
	ErrEmailParseFailed                   = errorx.NewValidationFieldFailed("email")
	ErrInvalidEmailFormat                 = errorx.NewValidationFieldFailed("email")
	ErrEmptyEmail                         = errorx.NewValidationFieldFailed("email")
	ErrEmailDomainNotAllowed              = errorx.NewValidationFieldFailed("email")
	ErrInvalidVerificationCode            = errorx.NewValidationFieldFailed("verification_code").WithHTTPCode(http.StatusUnprocessableEntity)
	ErrCodeExpired                        = errorx.NewValidationFieldFailed("verification_code")
	ErrInvalidStatus                      = errorx.NewValidationFieldFailed("status").WithHTTPCode(http.StatusUnprocessableEntity)
	ErrWaitUntilResend                    = errorx.NewRateLimitExceeded()
	ErrPersistentTooManyAttempts          = errorx.NewPersistable(errorx.NewRateLimitExceeded())
	ErrPersistentVerificationCodeMismatch = errorx.NewPersistable(
		errorx.NewValidationFieldFailed("verification_code").WithHTTPCode(http.StatusUnprocessableEntity),
	)
)
