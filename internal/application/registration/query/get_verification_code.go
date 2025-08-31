package query

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ARUMANDESU/ucms/pkg/errorx"
)

type GetVerificationCodeHandler struct {
	pool *pgxpool.Pool
}

func NewGetVerificationCodeHandler(pool *pgxpool.Pool) *GetVerificationCodeHandler {
	return &GetVerificationCodeHandler{
		pool: pool,
	}
}

func (h *GetVerificationCodeHandler) Handle(ctx context.Context, email string) (string, error) {
	var code string
	err := h.pool.QueryRow(ctx, `
        SELECT verification_code
        FROM registrations
        WHERE email = $1
        ORDER BY created_at DESC
        LIMIT 1
    `, email).Scan(&code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", errorx.NewNotFound().WithCause(err)
		}
		return "", err
	}
	return code, nil
}
