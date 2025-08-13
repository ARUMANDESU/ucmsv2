package user

import "github.com/ARUMANDESU/ucms/pkg/errorx"

var (
	ErrMissingID               = errorx.NewValidationFieldFailed("id")
	ErrMissingEmail            = errorx.NewValidationFieldFailed("email")
	ErrMissingPassHash         = errorx.NewValidationFieldFailed("password_hash")
	ErrMissingFirstName        = errorx.NewValidationFieldFailed("first_name")
	ErrFirstNameTooLong        = errorx.NewValidationFieldFailed("first_name")
	ErrFirstNameTooShort       = errorx.NewValidationFieldFailed("first_name")
	ErrMissingLastName         = errorx.NewValidationFieldFailed("last_name")
	ErrLastNameTooLong         = errorx.NewValidationFieldFailed("last_name")
	ErrLastNameTooShort        = errorx.NewValidationFieldFailed("last_name")
	ErrMissingGroupID          = errorx.NewValidationFieldFailed("group_id")
	ErrInvalidGroupID          = errorx.NewValidationFieldFailed("group_id")
	ErrInvalidEmail            = errorx.NewValidationFieldFailed("email")
	ErrPasswordNotStrongEnough = errorx.NewPasswordFormatInvalid()
)
