package repos

import "errors"

var (
	ErrInvalidInput   = errors.New("invalid input")
	ErrNotFound       = errors.New("not found")
	ErrAlreadyExists  = errors.New("already exists")
	ErrNoRowsAffected = errors.New("no rows affected")
)
