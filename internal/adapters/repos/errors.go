package repos

import "github.com/ARUMANDESU/ucms/pkg/errorx"

var (
	ErrInvalidInput   = errorx.ErrInvalidInput
	ErrNotFound       = errorx.ErrNotFound
	ErrAlreadyExists  = errorx.NewAlreadyProcessed()
	ErrNoRowsAffected = errorx.ErrNotFound
)
