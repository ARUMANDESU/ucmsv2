package registration

import (
	"fmt"
	"net/http"

	"github.com/ARUMANDESU/ucms/pkg/apperr"
)

var (
	ErrInvalidEmail            = apperr.NewInvalid("invalid email address")
	ErrEmailAlreadyExists      = apperr.NewConflict("email already exists")
	ErrEmailExceedsMaxLength   = apperr.NewInvalid(fmt.Sprintf("email exceeds maximum length of %d characters", MaxEmailLength))
	ErrEmailParseFailed        = apperr.NewInvalid("failed to parse email address")
	ErrInvalidEmailFormat      = apperr.NewInvalid("invalid email format")
	ErrEmptyEmail              = apperr.NewInvalid("email cannot be empty")
	ErrEmailDomainNotAllowed   = apperr.NewInvalid("email domain is not allowed")
	ErrInvalidVerificationCode = apperr.New(apperr.CodeInvalid, "invalid verification code", http.StatusUnprocessableEntity)
	ErrInvalidStatus           = apperr.NewInvalid("invalid registration status")
	ErrCodeExpired             = apperr.NewInvalid("verification code has expired")
	ErrWaitUntilResend         = apperr.NewInvalid("wait until resend timeout expires before requesting a new code")
	ErrTooManyAttempts         = apperr.New(apperr.CodeInvalid, "too many verification code attempts", http.StatusTooManyRequests)
)
