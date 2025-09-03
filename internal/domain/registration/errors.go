package registration

import (
	"net/http"

	"github.com/ARUMANDESU/ucms/pkg/errorx"
	"github.com/ARUMANDESU/ucms/pkg/i18nx"
)

var (
	ErrInvalidVerificationCode = errorx.NewInvalidRequest().
					WithKey(i18nx.KeyInvalidVerificationCode).
					WithHTTPCode(http.StatusUnprocessableEntity)
	ErrCodeExpired                        = errorx.NewInvalidRequest().WithKey(i18nx.KeyCodeExpired).WithHTTPCode(http.StatusUnprocessableEntity)
	ErrInvalidStatus                      = errorx.NewValidationFieldFailed(i18nx.FieldStatus).WithHTTPCode(http.StatusUnprocessableEntity)
	ErrRegistrationCompleted              = errorx.NewAlreadyProcessed()
	ErrWaitUntilResend                    = errorx.NewRateLimitExceeded()
	ErrPersistentTooManyAttempts          = errorx.NewPersistable(errorx.NewRateLimitExceeded())
	ErrPersistentVerificationCodeMismatch = errorx.NewPersistable(
		errorx.NewValidationFieldFailed(i18nx.FieldVerificationCode).WithHTTPCode(http.StatusUnprocessableEntity),
	)
	ErrVerifyFirst = errorx.NewInvalidRequest().WithKey(i18nx.KeyVerifyFirst)
)
