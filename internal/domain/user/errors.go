package user

import "errors"

var (
	ErrMissingID         = errors.New("id is required")
	ErrMissingEmail      = errors.New("email is required")
	ErrMissingPassHash   = errors.New("password hash is required")
	ErrMissingFirstName  = errors.New("first name is required")
	ErrFirstNameTooLong  = errors.New("first name is too long")
	ErrFirstNameTooShort = errors.New("first name is too short")
	ErrMissingLastName   = errors.New("last name is required")
	ErrLastNameTooLong   = errors.New("last name is too long")
	ErrLastNameTooShort  = errors.New("last name is too short")
	ErrMissingGroupID    = errors.New("group ID is required")
	ErrInvalidGroupID    = errors.New("group ID is invalid")
	ErrInvalidEmail      = errors.New("email is invalid")
)
