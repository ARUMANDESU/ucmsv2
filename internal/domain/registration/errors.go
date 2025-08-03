package registration

import (
	"fmt"

	"github.com/ARUMANDESU/ucms/pkg/apperr"
)

var (
	ErrInvalidEmail          = apperr.NewInvalid("invalid email address")
	ErrEmailAlreadyExists    = apperr.NewConflict("email already exists")
	ErrEmailExceedsMaxLength = apperr.NewInvalid(fmt.Sprintf("email exceeds maximum length of %d characters", MaxEmailLength))
	ErrEmailParseFailed      = apperr.NewInvalid("failed to parse email address")
	ErrInvalidEmailFormat    = apperr.NewInvalid("invalid email format")
	ErrEmptyEmail            = apperr.NewInvalid("email cannot be empty")
	ErrEmailDomainNotAllowed = apperr.NewInvalid("email domain is not allowed")
)
