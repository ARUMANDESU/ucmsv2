package repos

import "errors"

var (
	ErrNotFound       = errors.New("not found")
	ErrAlreadyExists  = errors.New("already exists")
	ErrNoRowsAffected = errors.New("no rows affected")
)
