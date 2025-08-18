package registration

import (
	"net/http"

	"github.com/ARUMANDESU/ucms/pkg/errorx"
)

var (
	ErrInvalidVerificationCode = errorx.NewInvalidRequest().
					WithKey("business_error_invalid_verification_code").
					WithHTTPCode(http.StatusUnprocessableEntity)
	ErrCodeExpired = errorx.NewInvalidRequest().
			WithKey("business_error_code_expired").
			WithHTTPCode(http.StatusUnprocessableEntity)
	ErrInvalidStatus                      = errorx.NewValidationFieldFailed("status").WithHTTPCode(http.StatusUnprocessableEntity)
	ErrRegistrationCompleted              = errorx.NewAlreadyProcessed()
	ErrWaitUntilResend                    = errorx.NewRateLimitExceeded()
	ErrPersistentTooManyAttempts          = errorx.NewPersistable(errorx.NewRateLimitExceeded())
	ErrPersistentVerificationCodeMismatch = errorx.NewPersistable(
		errorx.NewValidationFieldFailed("verification_code").WithHTTPCode(http.StatusUnprocessableEntity),
	)
	ErrVerifyFirst = errorx.NewInvalidRequest().WithKey("business_error_verify_first")
)
