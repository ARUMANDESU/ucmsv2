package repos

import (
	"net/http"

	"github.com/ARUMANDESU/ucms/pkg/apperr"
)

var (
	ErrInvalidInput   = apperr.ErrInvalidInput
	ErrNotFound       = apperr.ErrNotFound
	ErrAlreadyExists  = apperr.New(apperr.CodeAlreadyProcessed, "resource already exists", http.StatusConflict)
	ErrNoRowsAffected = apperr.New(apperr.CodeNotFound, "no rows affected", http.StatusNotFound)
)
