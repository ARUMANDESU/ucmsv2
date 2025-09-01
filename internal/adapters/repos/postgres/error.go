package postgres

import (
	"errors"
)

var (
	ErrNoRowsAffected = errors.New("no rows affected")
	ErrNilFunc        = errors.New("update function cannot be nil")
)
